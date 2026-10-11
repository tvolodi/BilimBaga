package questions

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// #514: parseIntParam reads a positive integer query parameter. Anything missing, non-numeric or below 1 gives the
// default, so a client cannot turn a page size or limit into 0, a negative number or a malformed value.
func TestParseIntParam_PositiveIntegerOrDefault(t *testing.T) {
	cases := []struct {
		query string
		want  int
	}{
		{"", 25},
		{"page=7", 25},
		{"limit=7", 7},
		{"limit=1", 1},
		{"limit=0", 25},
		{"limit=-3", 25},
		{"limit=abc", 25},
		{"limit=3.5", 25},
		{"limit=%204", 25},
	}
	for _, tc := range cases {
		r := httptest.NewRequest("GET", "/api/v1/questions?"+tc.query, nil)
		assert.Equal(t, tc.want, parseIntParam(r, "limit", 25), "query %q", tc.query)
	}
}
