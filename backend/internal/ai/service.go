package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/deptscope"
)

const (
	featureQuestionGeneration = "question_generation"
	featureExamInsights       = "exam_insights"
	rateLimit                 = 20
	insightCacheTTL           = 24 * time.Hour
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
}

// NewService returns a Service backed by the given Repository and AnthropicClient.
func NewService(repo Repository, client AnthropicClient, model string, logger *slog.Logger) Service {
	return &aiService{
		repo:   repo,
		client: client,
		model:  model,
		logger: logger,
	}
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
func (s *aiService) GetInsights(ctx context.Context, examID, tenantID, userID string, forceRefresh bool) (*InsightResult, error) {
	// ISS-165: a department_admin's insights are computed over its own
	// department subtree only, so they must neither read nor populate the
	// exam-wide shared cache.
	scoped := deptscope.FromContext(ctx).Restricted

	// AC-3: Check cache first when not forcing refresh.
	if !forceRefresh && !scoped {
		cached, err := s.repo.GetInsightCache(ctx, examID)
		if err != nil {
			s.logger.Warn("insight cache read failed", "exam_id", examID, "error", err)
			// Non-fatal: fall through to generate fresh.
		}
		if cached != nil && time.Since(cached.GeneratedAt) < insightCacheTTL {
			return cached, nil
		}
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

	// Upsert cache (never for department-scoped results; see above).
	if scoped {
		return &InsightResult{
			Insights:    insights,
			GeneratedAt: time.Now().UTC(),
			Cached:      false,
		}, nil
	}
	if cacheErr := s.repo.UpsertInsightCache(ctx, examID, userID, insights); cacheErr != nil {
		s.logger.Error("failed to upsert insight cache", "exam_id", examID, "error", cacheErr)
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
