package api

import "testing"

func TestNormalizeEmail(t *testing.T) {
	cases := map[string]string{
		"John.Doe@Corp.com":     "john.doe@corp.com",
		"  a@b.co \t":           "a@b.co",
		"\nUPPER@EXAMPLE.ORG  ": "upper@example.org",
		"":                      "",
	}
	for in, want := range cases {
		if got := NormalizeEmail(in); got != want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", in, got, want)
		}
	}
}
