package auth

import (
	"errors"
	"regexp"
)

// ErrWeakPassword is returned when a password does not meet the complexity requirements.
var ErrWeakPassword = errors.New("WEAK_PASSWORD")

var (
	hasUpper = regexp.MustCompile(`[A-Z]`)
	hasLower = regexp.MustCompile(`[a-z]`)
	hasDigit = regexp.MustCompile(`[0-9]`)
)

// ValidateComplexity enforces the password complexity policy:
// minimum 8 characters, at least one uppercase letter, one lowercase letter, and one digit.
// Returns ErrWeakPassword if the password does not meet the criteria.
func ValidateComplexity(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	if !hasUpper.MatchString(password) {
		return ErrWeakPassword
	}
	if !hasLower.MatchString(password) {
		return ErrWeakPassword
	}
	if !hasDigit.MatchString(password) {
		return ErrWeakPassword
	}
	return nil
}
