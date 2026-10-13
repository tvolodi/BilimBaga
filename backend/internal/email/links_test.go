package email

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAppURL = "https://app.example.com"

// spaRoutes mirrors the route table of frontend/src/App.tsx for the paths a mail may link to (FR-BB68 AC-6).
var spaRoutes = []string{
	"/portal",
	"/portal/sessions/:sessionId/result",
	"/reset-password",
}

var urlInMail = regexp.MustCompile(`https?://[^\s"'<>]+`)

func matchesSPARoute(path string) bool {
	for _, route := range spaRoutes {
		re := "^" + regexp.MustCompile(`:[A-Za-z]+`).ReplaceAllString(route, `[^/]+`) + "$"
		if regexp.MustCompile(re).MatchString(path) {
			return true
		}
	}
	return false
}

// assertSPALinks checks every URL of a rendered mail part against FR-BB68 AC-6.
func assertSPALinks(t *testing.T, part, label string) {
	t.Helper()
	urls := urlInMail.FindAllString(part, -1)
	require.NotEmpty(t, urls, label+": mail carries no link")
	for _, u := range urls {
		assert.True(t, strings.HasPrefix(u, testAppURL+"/"), label+": "+u+" must start with PublicAppURL")
		assert.NotContains(t, u, "/api/", label)
		path := strings.TrimPrefix(u, testAppURL)
		if i := strings.IndexAny(path, "?#"); i >= 0 {
			path = path[:i]
		}
		assert.True(t, matchesSPARoute(path), label+": "+path+" is not an SPA route")
	}
}

func TestAppLink(t *testing.T) {
	cases := []struct{ base, want string }{
		{"https://app.example.com", "https://app.example.com/portal"},
		{"https://app.example.com/", "https://app.example.com/portal"},
		{"https://app.example.com///", "https://app.example.com/portal"},
	}
	for _, c := range cases {
		svc := &EmailService{cfg: Config{PublicAppURL: c.base}}
		got, err := svc.appLink("/portal")
		require.NoError(t, err)
		assert.Equal(t, c.want, got)
	}
}

func TestAppLink_EmptyBaseIsRefusedEvenWithAPIBaseURL(t *testing.T) {
	for _, base := range []string{"", "  ", "/"} {
		svc := &EmailService{cfg: Config{PublicAppURL: base, APIBaseURL: "https://api.example.com"}}
		_, err := svc.appLink("/portal")
		assert.ErrorIs(t, err, errPublicAppURLMissing, "base %q", base)
	}
}

// AC-6: every template, every locale, with the links the senders build.
func TestMailLinks_AreSPARoutesUnderPublicAppURL(t *testing.T) {
	svc := &EmailService{cfg: Config{PublicAppURL: testAppURL + "/"}}
	portal, err := svc.appLink(portalPath)
	require.NoError(t, err)
	result, err := svc.sessionResultLink("s-1")
	require.NoError(t, err)
	reset, err := svc.appLink("/reset-password?token=abc")
	require.NoError(t, err)

	base := map[string]any{"ExamTitle": "Safety", "Deadline": "2026-01-02 08:00 UTC", "ScorePct": "90.0",
		"PassingScorePct": "60.0", "AttemptsRemaining": 0, "SubmittedAt": "2026-01-02 08:00 UTC", "Passed": true}
	with := func(k, v string) map[string]any {
		m := map[string]any{}
		for kk, vv := range base {
			m[kk] = vv
		}
		m[k] = v
		return m
	}
	templates := map[string]map[string]any{
		"new_exam_assigned":     with("PortalLink", portal),
		"deadline_reminder":     with("PortalLink", portal),
		"overdue_reminder":      with("PortalLink", portal),
		"exam_passed":           with("ResultLink", result),
		"exam_failed":           with("ResultLink", result),
		"exam_submitted":        with("ResultLink", result),
		"exam_result_corrected": with("ResultLink", result),
		"password_reset_link":   with("ResetLink", reset),
	}
	for tmpl, data := range templates {
		for _, locale := range []string{"en", "ru", "kk"} {
			r, err := renderTemplate(tmpl, data, locale)
			require.NoError(t, err, tmpl+" "+locale)
			assertSPALinks(t, r.TextBody, tmpl+" text "+locale)
			assertSPALinks(t, r.HTMLBody, tmpl+" html "+locale)
		}
	}
}

// AC-2 / AC-3: the links the senders produce in a real send.
func TestSenders_LinkToSPAWhenPublicAppURLIsSet(t *testing.T) {
	srv := startFakeSMTP(t, nil)
	repo := newStubRepo()
	uid := "u-1"
	repo.allUsers = []UserEmailData{{Email: "a@x.io", Locale: "en"}}
	repo.targets = []ReminderRow{{Email: "b@x.io", Title: "Safety", Deadline: time.Now().Add(30 * time.Hour)}}
	repo.session = &SessionEmailData{UserEmail: "c@x.io", ExamTitle: "Safety", Passed: true, ScorePct: 90}
	svc := svcFor(repo, srv)
	svc.cfg.PublicAppURL = testAppURL

	svc.TriggerAssignment("e-1", "Safety", nil, "all", &uid)
	repo.waitLogs(t, 1)
	runDeadlineReminders(context.Background(), svc)
	repo.waitLogs(t, 1)
	svc.TriggerSessionResult("s-1")
	repo.waitLogs(t, 1)

	msgs, _, _ := srv.snapshot()
	require.Len(t, msgs, 3)
	want := []string{testAppURL + "/portal", testAppURL + "/portal", testAppURL + "/portal/sessions/s-1/result"}
	for i, raw := range msgs {
		text, html := decodeParts(t, raw)
		assert.Contains(t, text, want[i])
		assert.Contains(t, html, `href="`+want[i]+`"`)
		assertSPALinks(t, text, "sent text")
		assertSPALinks(t, html, "sent html")
	}
}

// AC-1: no PublicAppURL -> no SMTP call, a failed email_log row, an error to the caller.
func TestSenders_RefuseWithoutPublicAppURL(t *testing.T) {
	srv := startFakeSMTP(t, nil)
	repo := newStubRepo()
	uid := "u-1"
	repo.allUsers = []UserEmailData{{Email: "a@x.io"}}
	repo.targets = []ReminderRow{{Email: "b@x.io", Title: "Safety", Deadline: time.Now()}}
	repo.session = &SessionEmailData{UserEmail: "c@x.io", ExamTitle: "Safety", Passed: true}
	svc := svcFor(repo, srv)
	svc.cfg.PublicAppURL = ""
	svc.cfg.APIBaseURL = "https://api.example.com"

	svc.TriggerAssignment("e-1", "Safety", nil, "all", &uid)
	runDeadlineReminders(context.Background(), svc)
	svc.TriggerSessionResult("s-1")
	svc.TriggerExamSubmitted("s-1")
	svc.TriggerResultCorrected("s-1")
	logs := repo.waitLogs(t, 5)
	tmpls := map[string]bool{}
	for _, l := range logs {
		assert.ErrorIs(t, l.err, errPublicAppURLMissing)
		tmpls[l.tmpl] = true
	}
	assert.Len(t, tmpls, 5, "one refused row per template: %v", tmpls)

	repo.session = nil
	repo2 := &reminderRepo{stubRepo: newStubRepo(), data: &OverdueReminderData{Email: "e@x.io", ExamTitle: "T"}}
	svc2 := svcFor(repo2, srv)
	svc2.cfg.PublicAppURL = ""
	_, err := svc2.SendOverdueReminder(context.Background(), "u", "e")
	require.ErrorIs(t, err, errPublicAppURLMissing)
	l2 := repo2.waitLogs(t, 1)
	assert.Equal(t, "overdue_reminder", l2[0].tmpl)
	assert.Error(t, l2[0].err)

	msgs, _, _ := srv.snapshot()
	assert.Empty(t, msgs, "no SMTP call")
}
