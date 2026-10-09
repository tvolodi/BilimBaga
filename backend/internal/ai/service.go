package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"log/slog"
	"strings"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/deptscope"
	"golang.org/x/sync/singleflight"
)

const (
	featureQuestionGeneration = "question_generation"
	featureExamInsights       = "exam_insights"
	rateLimit                 = 20
	insightCacheTTL           = 24 * time.Hour

	// DefaultInsightsDailyLimit is the default per-user cap on paid insight calls per 24 h.
	DefaultInsightsDailyLimit = 50
)

// validDifficulties lists the accepted values for the difficulty field.
var validDifficulties = map[string]bool{
	"easy":   true,
	"medium": true,
	"hard":   true,
}

// Service defines the business logic interface for the AI domain.
type Service interface {
	// GenerateQuestions generates draft questions using the Anthropic API.
	// It validates inputs, enforces rate limits, calls the AI, logs usage, and returns drafts.
	GenerateQuestions(ctx context.Context, userID string, req GenerateQuestionsRequest) ([]DraftQuestion, error)

	// GetInsights returns AI-generated performance insights for the given exam.
	// Results are cached for 24 hours; forceRefresh bypasses the cache.
	GetInsights(ctx context.Context, examID, tenantID, userID string, forceRefresh bool) (*InsightResult, error)

	// GetLoyaltyNarrative generates a 2–3 sentence values profile narrative for
	// the given session's Likert responses. adminRole must be the JWT role of the
	// requesting admin; it is used to skip the department check for super_admin.
	GetLoyaltyNarrative(ctx context.Context, sessionID, adminUserID, adminRole string) (*LoyaltyNarrativeResult, error)
}

type aiService struct {
	repo   Repository
	client AnthropicClient
	model  string
	logger *slog.Logger
	// scoped holds insights for department-scoped callers, keyed by exam id +
	// scope hash (in-process; durable scope_key cache deferred, ISS-232 item 3).
	scoped *scopedInsightCache
	// flight coalesces concurrent cold insight requests (ISS-232).
	flight singleflight.Group
	// insightsDailyLimit caps paid insight calls per user per 24 h; <= 0 disables.
	insightsDailyLimit int
}

// Option customises an aiService.
type Option func(*aiService)

// WithInsightsDailyLimit sets the per-user daily cap on paid insight calls
// (AI_INSIGHTS_DAILY_LIMIT); n <= 0 disables the cap.
func WithInsightsDailyLimit(n int) Option {
	return func(s *aiService) { s.insightsDailyLimit = n }
}

// NewService returns a Service backed by the given Repository and AnthropicClient.
func NewService(repo Repository, client AnthropicClient, model string, logger *slog.Logger, opts ...Option) Service {
	s := &aiService{
		repo:               repo,
		client:             client,
		model:              model,
		logger:             logger,
		insightsDailyLimit: DefaultInsightsDailyLimit,
		scoped:             newScopedInsightCache(maxScopedInsightEntries, insightCacheTTL),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// GenerateQuestions implements Service.
func (s *aiService) GenerateQuestions(ctx context.Context, userID string, req GenerateQuestionsRequest) ([]DraftQuestion, error) {
	// AC-6: Validate inputs.
	if err := s.validateRequest(req); err != nil {
		return nil, err
	}

	// Resolve category name from DB.
	catName, err := s.repo.GetCategoryName(ctx, req.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("ai: GenerateQuestions: resolve category: %w", err)
	}
	req.CategoryName = catName

	// AC-5: Enforce per-user rate limit before calling Anthropic.
	if err := s.checkRateLimit(ctx, userID); err != nil {
		return nil, err
	}

	// Build prompt.
	prompt, err := BuildQuestionsPrompt(req)
	if err != nil {
		return nil, fmt.Errorf("ai: GenerateQuestions: build prompt: %w", err)
	}

	// AC-3: Call Anthropic. Any error → ErrAIUnavailable.
	text, tokensUsed, err := s.client.GenerateText(ctx, prompt, s.model)
	if err != nil {
		if errors.Is(err, ErrAIUnavailable) {
			return nil, ErrAIUnavailable
		}
		s.logger.Warn("anthropic call failed", "error", err)
		return nil, ErrAIUnavailable
	}

	// Parse the JSON response from the model.
	var result GeneratedQuestionsResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		s.logger.Warn("failed to parse AI response", "raw", text, "error", err)
		return nil, ErrAIUnavailable
	}

	// AC-4: Log usage on every successful call.
	if logErr := s.repo.LogUsage(ctx, UsageLog{
		UserID:     userID,
		Feature:    featureQuestionGeneration,
		TokensUsed: tokensUsed,
		Model:      s.model,
	}); logErr != nil {
		// Non-fatal: log but don't fail the request.
		s.logger.Error("failed to log AI usage", "error", logErr)
	}

	return result.Questions, nil
}

// checkRateLimit returns ErrAIRateLimited if userID has made >= rateLimit calls in the last hour.
func (s *aiService) checkRateLimit(ctx context.Context, userID string) error {
	count, err := s.repo.CountAIUsageLastHour(ctx, userID, featureQuestionGeneration)
	if err != nil {
		return fmt.Errorf("ai: checkRateLimit: %w", err)
	}
	if count >= rateLimit {
		return ErrAIRateLimited
	}
	return nil
}

// validateRequest checks all AC-6 constraints.
func (s *aiService) validateRequest(req GenerateQuestionsRequest) error {
	if req.Count < 1 || req.Count > 10 {
		return fmt.Errorf("%w: count must be between 1 and 10", ErrValidation)
	}
	if len(req.ContextText) > 2000 {
		return fmt.Errorf("%w: context_text must be at most 2000 characters", ErrValidation)
	}
	if !validDifficulties[req.Difficulty] {
		return fmt.Errorf("%w: difficulty must be one of easy, medium, hard", ErrValidation)
	}
	if req.CategoryID == "" {
		return fmt.Errorf("%w: category_id is required", ErrValidation)
	}
	return nil
}

// GetInsights implements Service.
// AC-3: Cache hit within 24 h → return cached, no Anthropic call.
// AC-4: forceRefresh=true → call Anthropic regardless, upsert cache.
// AC-6: Anthropic error → ErrAIUnavailable (no stale-cache fallback).
// AC-8: Every Anthropic call is logged in ai_usage_log with feature="exam_insights".
// ISS-232: results are cached per (exam, scope key) (unrestricted in the DB,
// scoped in-process; durable scope_key migration deferred); concurrent cold
// requests for the same exam + scope share one paid call (singleflight); each
// paid call counts against the caller's daily cap (ErrAIRateLimited).
func (s *aiService) GetInsights(ctx context.Context, examID, tenantID, userID string, forceRefresh bool) (*InsightResult, error) {
	// ISS-218: every role except super_admin gets insights computed over its
	// own department subtree. Those results are cached under a scope key (hash
	// of the visible department-id set, deptscope.ScopeKey) so callers with
	// equal sets share an entry and different sets never do; unrestricted
	// callers use ScopeKeyAll.
	scope := deptscope.FromContext(ctx)
	scopeKey := deptscope.ScopeKeyAll
	cacheable := true
	if scope.Restricted {
		ids, idErr := s.repo.GetScopeDepartmentIDs(ctx, scope)
		if idErr != nil {
			// Fail safe: no scope key means no cache read, write or request
			// coalescing; the data query below is still scoped in SQL.
			s.logger.Warn("insight scope lookup failed; skipping cache", "exam_id", examID, "error", idErr)
			cacheable = false
		} else {
			scopeKey = deptscope.ScopeKey(scope, ids)
		}
	}

	// AC-3: Check cache first when not forcing refresh.
	if !forceRefresh && cacheable {
		if hit := s.readInsightCache(ctx, examID, scopeKey); hit != nil {
			return hit, nil
		}
	}

	if !cacheable {
		return s.generateInsights(ctx, examID, tenantID, userID, scopeKey, false, forceRefresh)
	}

	// Coalesce concurrent cold requests for the same tenant + exam + scope so
	// only one pays. The shared call must not die with the leader's request
	// context, so it runs detached from cancellation (values are kept).
	key := tenantID + "|" + examID + "|" + scopeKey + "|" + strconv.FormatBool(forceRefresh)
	shared := context.WithoutCancel(ctx)
	ch := s.flight.DoChan(key, func() (any, error) {
		return s.generateInsights(shared, examID, tenantID, userID, scopeKey, true, forceRefresh)
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		r := res.Val.(*InsightResult)
		// Each caller gets its own copy (the flight result is shared).
		return &InsightResult{Insights: append([]string(nil), r.Insights...), GeneratedAt: r.GeneratedAt, Cached: r.Cached}, nil
	}
}

// readInsightCache returns a fresh cached entry or nil (read errors are non-fatal).
func (s *aiService) readInsightCache(ctx context.Context, examID, scopeKey string) *InsightResult {
	if scopeKey != deptscope.ScopeKeyAll {
		return s.scoped.get(scopedInsightKey{examID: examID, scope: scopeKey})
	}
	cached, err := s.repo.GetInsightCache(ctx, examID)
	if err != nil {
		s.logger.Warn("insight cache read failed", "exam_id", examID, "error", err)
		return nil
	}
	if cached != nil && time.Since(cached.GeneratedAt) < insightCacheTTL {
		return cached
	}
	return nil
}

// checkInsightsLimit returns ErrAIRateLimited when userID has already made
// >= the daily cap of paid insight calls in the last 24 hours. A cap <= 0
// disables the limit.
func (s *aiService) checkInsightsLimit(ctx context.Context, userID string) error {
	if s.insightsDailyLimit <= 0 {
		return nil
	}
	count, err := s.repo.CountAIUsageLastDay(ctx, userID, featureExamInsights)
	if err != nil {
		return fmt.Errorf("ai: checkInsightsLimit: %w", err)
	}
	if count >= s.insightsDailyLimit {
		return ErrAIRateLimited
	}
	return nil
}

// generateInsights performs the paid path: (re-)check cache, enforce the daily
// cap, gather data, call Anthropic, log usage and persist the cache entry.
func (s *aiService) generateInsights(ctx context.Context, examID, tenantID, userID, scopeKey string, cacheable, forceRefresh bool) (*InsightResult, error) {
	// A flight that just finished may have filled the cache after our miss.
	if !forceRefresh && cacheable {
		if hit := s.readInsightCache(ctx, examID, scopeKey); hit != nil {
			return hit, nil
		}
	}

	if err := s.checkInsightsLimit(ctx, userID); err != nil {
		return nil, err
	}

	// Gather anonymised analytics data (AC-5: no PII).
	data, err := s.repo.GetExamInsightData(ctx, examID, tenantID)
	if err != nil {
		if errors.Is(err, ErrExamNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, fmt.Errorf("ai: GetInsights: gather data: %w", err)
	}

	// Build prompt.
	prompt, err := BuildInsightPrompt(*data)
	if err != nil {
		return nil, fmt.Errorf("ai: GetInsights: build prompt: %w", err)
	}

	// Call Anthropic.
	text, tokensUsed, err := s.client.GenerateText(ctx, prompt, s.model)
	if err != nil {
		s.logger.Warn("anthropic call failed for insights", "exam_id", examID, "error", err)
		return nil, ErrAIUnavailable
	}

	// Parse the JSON array of insight strings.
	var insights []string
	if parseErr := json.Unmarshal([]byte(text), &insights); parseErr != nil {
		s.logger.Warn("failed to parse insight response", "raw", text, "error", parseErr)
		return nil, ErrAIUnavailable
	}
	if len(insights) < 3 || len(insights) > 5 {
		s.logger.Warn("insight count out of expected range", "count", len(insights), "exam_id", examID)
		// Partial results are acceptable per spec — do not reject.
	}

	// AC-8: Log usage before writing cache.
	if logErr := s.repo.LogUsage(ctx, UsageLog{
		UserID:     userID,
		Feature:    featureExamInsights,
		TokensUsed: tokensUsed,
		Model:      s.model,
	}); logErr != nil {
		s.logger.Error("failed to log insight usage", "error", logErr)
	}

	if cacheable {
		if scopeKey != deptscope.ScopeKeyAll {
			s.scoped.put(scopedInsightKey{examID: examID, scope: scopeKey}, insights, time.Now().UTC())
		} else if cacheErr := s.repo.UpsertInsightCache(ctx, examID, userID, insights); cacheErr != nil {
			s.logger.Error("failed to upsert insight cache", "exam_id", examID, "error", cacheErr)
		}
	}

	return &InsightResult{
		Insights:    insights,
		GeneratedAt: time.Now().UTC(),
		Cached:      false,
	}, nil
}

// GetLoyaltyNarrative implements Service.
// AC-1: Validates that the session belongs to a loyalty-track exam.
// AC-2: Verifies admin has department access to the employee (or is super_admin).
// AC-3: Prompt contains only anonymised Likert weights — no PII.
// AC-5: Logs usage with admin user_id, NOT employee user_id.
// AC-6: Anthropic error or empty response → ErrAIUnavailable.
func (s *aiService) GetLoyaltyNarrative(ctx context.Context, sessionID, adminUserID, adminRole string) (*LoyaltyNarrativeResult, error) {
	// Step 1: Load session track and employee user ID.
	track, employeeUserID, err := s.repo.GetSessionCategoryTrack(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrLoyaltySessionNotFound) {
			return nil, ErrLoyaltySessionNotFound
		}
		return nil, fmt.Errorf("ai: GetLoyaltyNarrative: get session track: %w", err)
	}
	if track != "loyalty" {
		return nil, ErrNotLoyaltySession
	}

	// Step 2: Authorization — super_admin bypasses department check.
	if adminRole != "super_admin" {
		ok, depErr := s.repo.IsEmployeeInAdminDepartment(ctx, adminUserID, employeeUserID)
		if depErr != nil {
			return nil, fmt.Errorf("ai: GetLoyaltyNarrative: check department access: %w", depErr)
		}
		if !ok {
			return nil, ErrForbidden
		}
	}

	// Step 3: Collect anonymised Likert responses.
	responses, err := s.repo.CollectLikertResponses(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("ai: GetLoyaltyNarrative: collect responses: %w", err)
	}

	// Step 4: Build prompt.
	prompt, err := BuildLoyaltyPrompt(LoyaltyPromptData{Responses: responses})
	if err != nil {
		return nil, fmt.Errorf("ai: GetLoyaltyNarrative: build prompt: %w", err)
	}

	// Step 5: Call Anthropic.
	text, tokensUsed, err := s.client.GenerateText(ctx, prompt, s.model)
	if err != nil {
		s.logger.Warn("anthropic call failed for loyalty narrative", "session_id", sessionID, "error", err)
		return nil, ErrAIUnavailable
	}

	// Step 6: Validate response.
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrAIUnavailable
	}

	// Step 7: Log usage with admin user_id only (AC-5: employee ID must NOT be logged).
	if logErr := s.repo.LogUsage(ctx, UsageLog{
		UserID:     adminUserID,
		Feature:    featureLoyaltyNarrative,
		TokensUsed: tokensUsed,
		Model:      s.model,
	}); logErr != nil {
		s.logger.Error("failed to log loyalty narrative usage", "error", logErr)
	}

	return &LoyaltyNarrativeResult{
		Narrative:   text,
		GeneratedAt: time.Now().UTC(),
	}, nil
}
