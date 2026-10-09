package email

import (
	"strings"
	"testing"
)

// TestRenderTemplate_AllTemplates verifies that all five email templates render
// without error and produce non-empty output in every locale (AC-1, AC-7).
func TestRenderTemplate_AllTemplates(t *testing.T) {
	t.Parallel()

	templates := []struct {
		name string
		data map[string]any
	}{
		{
			name: "new_exam_assigned",
			data: map[string]any{
				"ExamTitle":  "Go Fundamentals",
				"Deadline":   "2026-12-31 08:00 UTC",
				"PortalLink": "http://localhost/portal/exams/test-id",
			},
		},
		{
			name: "exam_passed",
			data: map[string]any{
				"ExamTitle":        "Go Fundamentals",
				"ScorePct":         "90.0",
				"CertDownloadLink": "http://localhost/portal/sessions/test-id/certificate",
			},
		},
		{
			name: "exam_failed",
			data: map[string]any{
				"ExamTitle":         "Go Fundamentals",
				"ScorePct":          "40.0",
				"PassingScorePct":   "60.0",
				"AttemptsRemaining": 2,
			},
		},
		{
			name: "deadline_reminder",
			data: map[string]any{
				"ExamTitle": "Go Fundamentals",
				"Deadline":  "2026-12-31 08:00 UTC",
			},
		},
		{
			name: "password_reset",
			data: map[string]any{
				"TempPassword": "Tmp!Abc123",
			},
		},
		{
			name: "password_reset_link",
			data: map[string]any{
				"ResetLink": "https://app.example.com/reset-password?token=abc_DEF-123",
			},
		},
	}

	localeList := []string{"en", "kk", "ru"}

	for _, tt := range templates {
		for _, locale := range localeList {
			tt := tt
			locale := locale
			t.Run(tt.name+"/"+locale, func(t *testing.T) {
				t.Parallel()

				rendered, err := renderTemplate(tt.name, tt.data, locale)
				if err != nil {
					t.Fatalf("renderTemplate(%q, %q): %v", tt.name, locale, err)
				}
				if rendered.Subject == "" {
					t.Errorf("locale=%q template=%q: Subject is empty", locale, tt.name)
				}
				if rendered.TextBody == "" {
					t.Errorf("locale=%q template=%q: TextBody is empty", locale, tt.name)
				}
				if rendered.HTMLBody == "" {
					t.Errorf("locale=%q template=%q: HTMLBody is empty", locale, tt.name)
				}
				// Ensure template variables were substituted (no raw {{  }}  left)
				for _, part := range []string{rendered.Subject, rendered.TextBody, rendered.HTMLBody} {
					if strings.Contains(part, "{{") {
						t.Errorf("locale=%q template=%q: unresolved template directive in output: %q",
							locale, tt.name, part)
					}
				}
			})
		}
	}
}

// TestResolveLocale verifies the three-level fallback (AC-7).
func TestResolveLocale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		userLocale   string
		tenantLocale string
		want         string
	}{
		// Level 1: user preference matches a known locale
		{userLocale: "kk", tenantLocale: "ru", want: "kk"},
		{userLocale: "ru", tenantLocale: "en", want: "ru"},
		{userLocale: "en", tenantLocale: "kk", want: "en"},
		// Level 2: user preference unknown, fall back to tenant default
		{userLocale: "", tenantLocale: "ru", want: "ru"},
		{userLocale: "fr", tenantLocale: "kk", want: "kk"},
		// Level 3: both unknown, fall back to "en"
		{userLocale: "", tenantLocale: "", want: "en"},
		{userLocale: "de", tenantLocale: "fr", want: "en"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.userLocale+"|"+tt.tenantLocale, func(t *testing.T) {
			t.Parallel()
			got := resolveLocale(tt.userLocale, tt.tenantLocale)
			if got != tt.want {
				t.Errorf("resolveLocale(%q, %q) = %q; want %q",
					tt.userLocale, tt.tenantLocale, got, tt.want)
			}
		})
	}
}

// FR-BB115 AC-3: the reset link appears in both the plain-text and HTML parts for every locale.
func TestPasswordResetLinkTemplate_ContainsLinkInAllLocales(t *testing.T) {
	const link = "https://app.example.com/reset-password?token=abc_DEF-123"
	for _, locale := range []string{"en", "ru", "kk"} {
		rendered, err := renderTemplate("password_reset_link", map[string]any{"ResetLink": link}, locale)
		if err != nil {
			t.Fatalf("%s: %v", locale, err)
		}
		if !strings.Contains(rendered.TextBody, link) {
			t.Errorf("%s: text body missing link", locale)
		}
		if !strings.Contains(rendered.HTMLBody, `href="`+link+`"`) {
			t.Errorf("%s: html body missing href link: %s", locale, rendered.HTMLBody)
		}
		if strings.Contains(rendered.Subject, "{{") {
			t.Errorf("%s: unrendered subject", locale)
		}
	}
}
