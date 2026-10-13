package email

import (
	"errors"
	"fmt"
	"strings"
)

// errPublicAppURLMissing is returned when a mail needs a link but PUBLIC_APP_URL is not configured. Every
// employee-facing link opens the SPA (FR-BB68 AC-1); there is no fallback to the API host.
var errPublicAppURLMissing = errors.New("email: PUBLIC_APP_URL is not configured; links in mails must open the SPA")

// portalPath is the SPA route listing the employee's exam cards (FR-BB313); target of the assignment and reminder mails.
const portalPath = "/portal"

// appLink builds an absolute SPA link from Config.PublicAppURL (trailing slash trimmed) and a path that starts
// with "/". It is the only place that builds a link for an outgoing mail (FR-BB68 AC-1).
func (s *EmailService) appLink(path string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(s.cfg.PublicAppURL), "/")
	if base == "" {
		return "", errPublicAppURLMissing
	}
	return base + path, nil
}

// sessionResultLink is the SPA result page of a session (FR-BB322 AC-10). It is an SPA route, so the employee
// signs in there, and the certificate is downloaded from that page with a token.
func (s *EmailService) sessionResultLink(sessionID string) (string, error) {
	return s.appLink("/portal/sessions/" + sessionID + "/result")
}

// refuseSend records a mail that was not sent because its link could not be built: an email_log row with status
// failed and an error log, no SMTP call (FR-BB68 AC-1).
func (s *EmailService) refuseSend(to, tmpl string, err error) error {
	s.logAttempt(to, tmpl, err)
	s.logger.Error("email refused", "template", tmpl, "to", to, "error", err)
	return fmt.Errorf("email: %s refused: %w", tmpl, err)
}
