package api

import "strings"

// NormalizeEmail returns the canonical stored form of an email address: surrounding
// whitespace trimmed and lowercased. Every code path that stores or looks up a user by
// email must pass the address through this function (ISS-164) so that "John.Doe@Corp.com"
// and "john.doe@corp.com" always resolve to the same account.
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
