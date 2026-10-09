package api

import "testing"

func TestCSVSafe(t *testing.T) {
	cases := map[string]string{
		`=HYPERLINK("http://evil","x")`: `'=HYPERLINK("http://evil","x")`,
		"+1":                            "'+1",
		"-2+3":                          "'-2+3",
		"@SUM(A1)":                      "'@SUM(A1)",
		"\t=1":                          "'\t=1",
		"\r=1":                          "'\r=1",
		"Aibek Nurlan":                  "Aibek Nurlan",
		"a=b":                           "a=b",
		"":                              "",
		"Жанар Әлиева":                  "Жанар Әлиева",
		"user@example.com":              "user@example.com",
		"'=1":                           "''=1",
		"'x":                            "'x",
		" =1+1":                         " =1+1",
	}
	for in, want := range cases {
		if got := CSVSafe(in); got != want {
			t.Errorf("CSVSafe(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCSVSafeRecord(t *testing.T) {
	in := []string{"=1", "ok", ""}
	got := CSVSafeRecord(in)
	if got[0] != "'=1" || got[1] != "ok" || got[2] != "" {
		t.Errorf("unexpected %v", got)
	}
	if in[0] != "=1" {
		t.Error("input mutated")
	}
}

func TestCSVUnsafeRoundTrip(t *testing.T) {
	for _, s := range []string{"=1+1", "-5 degrees", "@x", "+1", "\t=1", "\r=1", "plain", "", "'quoted", "'"} {
		if got := CSVUnsafe(CSVSafe(s)); got != s {
			t.Errorf("round trip %q -> %q", s, got)
		}
	}
}
