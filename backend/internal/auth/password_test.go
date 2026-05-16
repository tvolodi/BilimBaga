package auth

import (
	"errors"
	"testing"
)

func TestValidateComplexity_Pass(t *testing.T) {
	passwords := []string{
		"Abcdef1g",
		"Password1",
		"MyStr0ngPwd",
		"Hello123World",
		"A1bcdefg",
	}
	for _, p := range passwords {
		if err := ValidateComplexity(p); err != nil {
			t.Errorf("expected nil for %q, got %v", p, err)
		}
	}
}

func TestValidateComplexity_TooShort(t *testing.T) {
	err := ValidateComplexity("Ab1cdef") // 7 chars
	if !errors.Is(err, ErrWeakPassword) {
		t.Errorf("expected ErrWeakPassword for too-short password, got %v", err)
	}
}

func TestValidateComplexity_NoUpper(t *testing.T) {
	err := ValidateComplexity("abcdef1g")
	if !errors.Is(err, ErrWeakPassword) {
		t.Errorf("expected ErrWeakPassword for no uppercase, got %v", err)
	}
}

func TestValidateComplexity_NoLower(t *testing.T) {
	err := ValidateComplexity("ABCDEF1G")
	if !errors.Is(err, ErrWeakPassword) {
		t.Errorf("expected ErrWeakPassword for no lowercase, got %v", err)
	}
}

func TestValidateComplexity_NoDigit(t *testing.T) {
	err := ValidateComplexity("Abcdefgh")
	if !errors.Is(err, ErrWeakPassword) {
		t.Errorf("expected ErrWeakPassword for no digit, got %v", err)
	}
}
