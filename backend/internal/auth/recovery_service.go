package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// resetTokenBytes is the entropy of a password reset token (FR-BB115 AC-2).
	resetTokenBytes = 32
	// resetTokenTTL is how long a reset token stays valid.
	resetTokenTTL = 60 * time.Minute
	// resetMaxPerHour throttles forgot-password requests per user (AC-9).
	resetMaxPerHour = 3
	// resetPurgeAge is how long an expired token row is kept (must exceed the throttle window).
	resetPurgeAge = 24 * time.Hour
)

// ResetMailer delivers the password reset link. It is implemented by *email.EmailService.
type ResetMailer interface {
	TriggerPasswordResetLink(userID, token string)
}

// RecoveryService is the password recovery part of the auth Service (FR-BB115).
type RecoveryService interface {
	// ForgotPassword issues a reset token for an active user. It returns the user ID when a
	// token was issued and "" otherwise; callers must respond identically in both cases.
	ForgotPassword(ctx context.Context, req *ForgotPasswordRequest, ipAddr string) (string, error)
	// ResetPassword consumes a reset token and sets a new password. Returns the user ID.
	ResetPassword(ctx context.Context, req *ResetPasswordRequest, ipAddr string) (string, error)
}

// ForgotPassword never reveals whether the email exists: unknown, inactive and throttled
// requests all return ("", nil). Only a malformed email is an error (VALIDATION_ERROR).
func (s *service) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest, ipAddr string) (string, error) {
	email := strings.TrimSpace(req.Email)
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return "", &ServiceError{Code: "VALIDATION_ERROR", Message: "a valid email is required", HTTPStatus: http.StatusBadRequest}
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Same cheap work as the found path to even out latency.
			if raw, genErr := generateResetToken(); genErr == nil {
				_ = hashToken(raw)
			}
			return "", nil
		}
		return "", fmt.Errorf("auth.service.ForgotPassword: look up user: %w", err)
	}
	if user.Status != "active" {
		return "", nil
	}

	// Bound table growth (only for real accounts, so unauthenticated unknown-email calls stay write-free): drop long-expired rows (kept past the throttle window).
	if err := s.repo.PurgeExpiredResetTokens(ctx, s.now().Add(-resetPurgeAge)); err != nil {
		s.logger.Error("auth.ForgotPassword: purge expired reset tokens", "error", err)
	}

	recent, err := s.repo.CountRecentResetTokens(ctx, user.ID, s.now().Add(-time.Hour))
	if err != nil {
		return "", fmt.Errorf("auth.service.ForgotPassword: count recent tokens: %w", err)
	}
	if recent >= resetMaxPerHour {
		return "", nil // silently throttled (AC-9)
	}

	raw, err := generateResetToken()
	if err != nil {
		return "", fmt.Errorf("auth.service.ForgotPassword: %w", err)
	}
	if err := s.repo.CreateResetToken(ctx, user.ID, hashToken(raw), s.now().Add(resetTokenTTL)); err != nil {
		return "", fmt.Errorf("auth.service.ForgotPassword: store token: %w", err)
	}
	if s.mailer != nil {
		s.mailer.TriggerPasswordResetLink(user.ID, raw)
	}
	return user.ID, nil
}

// ResetPassword consumes the token. A policy-violating password is rejected before the
// token is touched, so it is not consumed.
func (s *service) ResetPassword(ctx context.Context, req *ResetPasswordRequest, ipAddr string) (string, error) {
	invalidToken := &ServiceError{Code: "INVALID_TOKEN", Message: "reset link is invalid or has expired", HTTPStatus: http.StatusBadRequest}

	if req.Token == "" {
		return "", invalidToken
	}
	if err := ValidateComplexity(req.NewPassword); err != nil {
		return "", &ServiceError{
			Code:       "VALIDATION_ERROR",
			Message:    "password must be at least 8 characters and contain uppercase, lowercase, and a digit",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.cfg.BcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth.service.ResetPassword: hash password: %w", err)
	}

	userID, err := s.repo.CompleteReset(ctx, hashToken(req.Token), string(hash), s.now())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", invalidToken
		}
		return "", fmt.Errorf("auth.service.ResetPassword: complete reset: %w", err)
	}
	return userID, nil
}

// generateResetToken returns a URL-safe random token with 256 bits of entropy.
func generateResetToken() (string, error) {
	b := make([]byte, resetTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
