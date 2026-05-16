package auth

import (
	"testing"
)

// TestHashTokenExported_Deterministic verifies the exported HashToken wrapper
// produces consistent output (complements the unexported hashToken tests in handler_test.go).
func TestHashTokenExported_Deterministic(t *testing.T) {
	h1 := HashToken("my-secret-token")
	h2 := HashToken("my-secret-token")
	if h1 != h2 {
		t.Errorf("HashToken is not deterministic: %q != %q", h1, h2)
	}
}

// TestHashTokenExported_Different verifies different inputs produce different hashes.
func TestHashTokenExported_Different(t *testing.T) {
	h1 := HashToken("token-one")
	h2 := HashToken("token-two")
	if h1 == h2 {
		t.Error("different inputs produced the same hash")
	}
}

// TestHashTokenExported_Length verifies SHA-256 output is 64 hex characters.
func TestHashTokenExported_Length(t *testing.T) {
	h := HashToken("any-token-value")
	if len(h) != 64 {
		t.Errorf("expected 64 hex chars, got %d: %q", len(h), h)
	}
}
