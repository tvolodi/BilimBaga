package auth

// HashToken returns the hex-encoded SHA-256 hash of a token string.
// It is an exported wrapper for use in tests and other packages that need
// to verify refresh token hashing behaviour.
// The internal hashToken function in service.go is the canonical implementation;
// this wrapper delegates to it so there is a single code path.
func HashToken(plaintext string) string {
	return hashToken(plaintext)
}
