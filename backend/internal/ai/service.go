package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
)

const (
	featureQuestionGeneration = "question_generation"
	rateLimit                 = 20
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
