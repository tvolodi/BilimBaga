package email

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FR-BB322 AC-10: result mails link to the SPA result page, not to the certificate API path. The certificate is
// downloaded from that page, with a token.
func TestSessionResultMails_LinkToTheResultPage(t *testing.T) {
	svc := svcFor(newStubRepo(), nil)
	svc.cfg.PublicAppURL = "https://app.example.com/"
	link := "https://app.example.com/portal/sessions/s-1/result"
	got, err := svc.sessionResultLink("s-1")
	require.NoError(t, err)
	require.Equal(t, link, got, "trailing slash of PUBLIC_APP_URL is trimmed")

	data := map[string]any{
		"ExamTitle":         "Safety",
		"ScorePct":          "90.0",
		"PassingScorePct":   "60.0",
		"AttemptsRemaining": 0,
		"ResultLink":        link,
		"SubmittedAt":       "2026-10-11 09:30 UTC",
	}
	for _, locale := range []string{"en", "kk", "ru"} {
		for _, tmpl := range []string{"exam_passed", "exam_failed", "exam_submitted", "exam_result_corrected"} {
			r, err := renderTemplate(tmpl, data, locale)
			require.NoError(t, err, tmpl+" "+locale)
			if tmpl != "exam_submitted" {
				assert.Contains(t, r.TextBody, link, tmpl+" text "+locale)
			}
			assert.Contains(t, r.HTMLBody, link, tmpl+" html "+locale)
			assert.NotContains(t, r.TextBody, "/certificate", tmpl+" must not link the certificate API path "+locale)
		}
	}
}

// AC-9: exam_submitted is sent once per call, to the session owner, with the submit time.
func TestTriggerExamSubmitted_SendsOneMailToTheOwner(t *testing.T) {
	srv := startFakeSMTP(t, nil)
	repo := newStubRepo()
	submitted := time.Date(2026, 10, 11, 9, 30, 0, 0, time.UTC)
	repo.session = &SessionEmailData{UserEmail: "u@x.io", UserLocale: "ru", ExamTitle: "Safety", SubmittedAt: &submitted}
	svcFor(repo, srv).TriggerExamSubmitted("s-1")

	logs := repo.waitLogs(t, 1)
	assert.Equal(t, "exam_submitted", logs[0].tmpl)
	assert.Equal(t, "u@x.io", logs[0].to)
	assert.NoError(t, logs[0].err)
	msgs, _, _ := srv.snapshot()
	assert.Len(t, msgs, 1)
}

// AC-9: a send failure is recorded and does not panic or block the caller.
func TestTriggerExamSubmitted_SendFailureIsLogged(t *testing.T) {
	repo := newStubRepo()
	repo.session = &SessionEmailData{UserEmail: "u@x.io", ExamTitle: "E"}
	svcFor(repo, nil).TriggerExamSubmitted("s-1") // SMTP refused
	logs := repo.waitLogs(t, 1)
	assert.Error(t, logs[0].err)
}
