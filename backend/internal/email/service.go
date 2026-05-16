// Package email provides fire-and-forget transactional email delivery for BilimBaga.
// It must not import any other internal domain package to avoid circular imports.
package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"math/rand"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Config holds SMTP delivery settings.
type Config struct {
	Host       string
	Port       int
	User       string
	Pass       string
	TLS        bool
	From       string // may include display name: "BilimBaga <noreply@example.com>"
	APIBaseURL string
}

// EmailService coordinates template rendering, SMTP delivery, and audit logging.
type EmailService struct {
	cfg    Config
	repo   Repository
	logger *slog.Logger
}

// NewEmailService constructs an EmailService backed by a postgres repository.
// db may be nil only in tests (repo will be replaced separately).
func NewEmailService(cfg Config, db *sqlx.DB, logger *slog.Logger) *EmailService {
	if logger == nil {
		logger = slog.Default()
	}
	svc := &EmailService{cfg: cfg, logger: logger}
	if db != nil {
		svc.repo = newPostgresRepository(db)
	}
	return svc
}

// ── Public trigger methods (all fire-and-forget) ──────────────────────────────

// TriggerSessionResult looks up the session result and sends exam_passed or
// exam_failed to the session owner.  Called after auto-grading or manual grading.
func (s *EmailService) TriggerSessionResult(sessionID string) {
	go func() {
		ctx := context.Background()
		if s.repo == nil {
			return
		}
		data, err := s.repo.GetSessionEmailData(ctx, sessionID)
		if err != nil {
			s.logger.Error("email: get session data", "session_id", sessionID, "error", err)
			return
		}
		tmplName := "exam_failed"
		if data.Passed {
			tmplName = "exam_passed"
		}

		var tmplData map[string]any
		if data.Passed {
			certLink := s.cfg.APIBaseURL + "/portal/sessions/" + sessionID + "/certificate"
			tmplData = map[string]any{
				"ExamTitle":        data.ExamTitle,
				"ScorePct":         fmt.Sprintf("%.1f", data.ScorePct),
				"CertDownloadLink": certLink,
			}
		} else {
			attemptsRemaining := 0
			if data.MaxAttempts > 0 {
				attemptsRemaining = data.MaxAttempts - data.AttemptNumber
				if attemptsRemaining < 0 {
					attemptsRemaining = 0
				}
			}
			tmplData = map[string]any{
				"ExamTitle":         data.ExamTitle,
				"ScorePct":          fmt.Sprintf("%.1f", data.ScorePct),
				"PassingScorePct":   fmt.Sprintf("%.1f", data.PassingScorePct),
				"AttemptsRemaining": attemptsRemaining,
			}
		}

		sendErr := s.send(data.UserEmail, tmplName, tmplData, data.UserLocale)
		s.logAttempt(data.UserEmail, tmplName, sendErr)
		if sendErr != nil {
			s.logger.Error("email send failed", "template", tmplName, "to", data.UserEmail, "error", sendErr)
		}
	}()
}

// TriggerAssignment sends new_exam_assigned to the relevant users when an exam
// is assigned.
func (s *EmailService) TriggerAssignment(examID, examTitle string, deadline *time.Time, assigneeType string, assigneeID *string) {
	go func() {
		ctx := context.Background()
		if s.repo == nil {
			return
		}
		var recipients []UserEmailData
		var err error

		switch assigneeType {
		case "user":
			if assigneeID == nil {
				return
			}
			u, uerr := s.repo.GetUserForEmail(ctx, *assigneeID)
			if uerr != nil {
				s.logger.Error("email: get user for assignment", "assignee", *assigneeID, "error", uerr)
				return
			}
			recipients = []UserEmailData{u}
		case "department":
			if assigneeID == nil {
				return
			}
			recipients, err = s.repo.GetDepartmentUsersForEmail(ctx, *assigneeID)
		case "all":
			recipients, err = s.repo.GetAllActiveUsersForEmail(ctx)
		default:
			return
		}

		if err != nil {
			s.logger.Error("email: get assignment recipients", "type", assigneeType, "error", err)
			return
		}

		deadlineStr := "–"
		if deadline != nil {
			deadlineStr = deadline.UTC().Format("2006-01-02 15:04 UTC")
		}
		portalLink := s.cfg.APIBaseURL + "/portal/exams/" + examID

		for _, u := range recipients {
			tmplData := map[string]any{
				"ExamTitle":  examTitle,
				"Deadline":   deadlineStr,
				"PortalLink": portalLink,
			}
			sendErr := s.send(u.Email, "new_exam_assigned", tmplData, u.Locale)
			s.logAttempt(u.Email, "new_exam_assigned", sendErr)
			if sendErr != nil {
				s.logger.Error("email send failed", "template", "new_exam_assigned", "to", u.Email, "error", sendErr)
			}
		}
	}()
}

// TriggerPasswordReset sends a password_reset email with the temporary password.
func (s *EmailService) TriggerPasswordReset(userID, tempPassword string) {
	go func() {
		ctx := context.Background()
		if s.repo == nil {
			return
		}
		u, err := s.repo.GetUserForEmail(ctx, userID)
		if err != nil {
			s.logger.Error("email: get user for password reset", "user_id", userID, "error", err)
			return
		}
		tmplData := map[string]any{
			"TempPassword": tempPassword,
		}
		sendErr := s.send(u.Email, "password_reset", tmplData, u.Locale)
		s.logAttempt(u.Email, "password_reset", sendErr)
		if sendErr != nil {
			s.logger.Error("email send failed", "template", "password_reset", "to", u.Email, "error", sendErr)
		}
	}()
}

// GetUserEmail looks up a user's email address for use in the test notification handler.
func (s *EmailService) GetUserEmail(ctx context.Context, userID string) (string, error) {
	if s.repo == nil {
		return "", fmt.Errorf("email: repository not initialised")
	}
	u, err := s.repo.GetUserForEmail(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("email: get user email: %w", err)
	}
	return u.Email, nil
}

// TestSend opens a real SMTP connection and sends a test message; returns any error.
// Unlike Send, this is synchronous so the handler can report failure immediately.
func (s *EmailService) TestSend(to string) error {
	fromAddr, err := extractEmailAddress(s.cfg.From)
	if err != nil {
		fromAddr = s.cfg.From
	}
	subject := "BilimBaga — Test Notification"
	textBody := "This is a test email from BilimBaga. If you received this, email delivery is working correctly."
	htmlBody := "<!DOCTYPE html><html><body style=\"font-family:Arial,sans-serif\"><p>This is a test email from BilimBaga. If you received this, email delivery is working correctly.</p></body></html>"
	msg := buildMIMEMessage(s.cfg.From, sanitiseHeader(to), sanitiseHeader(subject), textBody, htmlBody)
	return s.smtpSend(fromAddr, to, msg)
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// send renders the template, builds the MIME message, and sends via SMTP.
func (s *EmailService) send(to, tmplName string, data map[string]any, userLocale string) error {
	tenantLocale := "en"
	if s.repo != nil {
		tl, err := s.repo.GetTenantDefaultLocale(context.Background())
		if err == nil && tl != "" {
			tenantLocale = tl
		}
	}

	rendered, err := renderTemplate(tmplName, data, resolveLocale(userLocale, tenantLocale))
	if err != nil {
		return fmt.Errorf("email: render %s: %w", tmplName, err)
	}

	fromAddr, err := extractEmailAddress(s.cfg.From)
	if err != nil {
		fromAddr = s.cfg.From
	}

	msg := buildMIMEMessage(s.cfg.From, sanitiseHeader(to), sanitiseHeader(rendered.Subject), rendered.TextBody, rendered.HTMLBody)
	return s.smtpSend(fromAddr, to, msg)
}

// logAttempt writes an audit record to email_log; errors are swallowed.
func (s *EmailService) logAttempt(to, tmpl string, sendErr error) {
	if s.repo == nil {
		return
	}
	ctx := context.Background()
	if logErr := s.repo.LogAttempt(ctx, to, tmpl, sendErr); logErr != nil {
		s.logger.Error("email: log attempt failed", "to", to, "template", tmpl, "error", logErr)
	}
}

// sanitiseHeader removes CR and LF characters to prevent header injection (CWE-93).
func sanitiseHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// extractEmailAddress parses an RFC 5322 address and returns the bare addr-spec.
func extractEmailAddress(from string) (string, error) {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		return "", fmt.Errorf("email: parse from address %q: %w", from, err)
	}
	return addr.Address, nil
}

// smtpSend delivers msg using either STARTTLS or plain SMTP depending on cfg.TLS.
func (s *EmailService) smtpSend(fromAddr, to string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	if s.cfg.TLS {
		c, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("email: smtp dial %s: %w", addr, err)
		}
		defer c.Close() //nolint:errcheck

		if ok, _ := c.Extension("STARTTLS"); ok {
			tlsCfg := &tls.Config{
				ServerName: s.cfg.Host,
				MinVersion: tls.VersionTLS12,
			}
			if err := c.StartTLS(tlsCfg); err != nil {
				return fmt.Errorf("email: smtp starttls: %w", err)
			}
		}

		if s.cfg.User != "" {
			auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Pass, s.cfg.Host)
			if err := c.Auth(auth); err != nil {
				return fmt.Errorf("email: smtp auth: %w", err)
			}
		}

		if err := c.Mail(fromAddr); err != nil {
			return fmt.Errorf("email: smtp MAIL FROM: %w", err)
		}
		if err := c.Rcpt(to); err != nil {
			return fmt.Errorf("email: smtp RCPT TO: %w", err)
		}
		w, err := c.Data()
		if err != nil {
			return fmt.Errorf("email: smtp DATA: %w", err)
		}
		if _, err := w.Write(msg); err != nil {
			return fmt.Errorf("email: smtp write body: %w", err)
		}
		return w.Close()
	}

	// Plain / no TLS (e.g. Mailhog on port 1025 in dev)
	var auth smtp.Auth
	if s.cfg.User != "" {
		auth = smtp.PlainAuth("", s.cfg.User, s.cfg.Pass, s.cfg.Host)
	}
	if err := smtp.SendMail(addr, auth, fromAddr, []string{to}, msg); err != nil {
		return fmt.Errorf("email: smtp sendmail: %w", err)
	}
	return nil
}

// buildMIMEMessage constructs a multipart/alternative MIME email.
func buildMIMEMessage(from, to, subject, textBody, htmlBody string) []byte {
	boundary := fmt.Sprintf("===============%d==", rand.Int63()) //nolint:gosec

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=%q\r\n", boundary)
	fmt.Fprintf(&buf, "\r\n")

	// text/plain part
	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
	fmt.Fprintf(&buf, "Content-Transfer-Encoding: quoted-printable\r\n")
	fmt.Fprintf(&buf, "\r\n")
	qpW := quotedprintable.NewWriter(&buf)
	_, _ = qpW.Write([]byte(textBody))
	_ = qpW.Close()
	fmt.Fprintf(&buf, "\r\n")

	// text/html part
	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Type: text/html; charset=UTF-8\r\n")
	fmt.Fprintf(&buf, "Content-Transfer-Encoding: quoted-printable\r\n")
	fmt.Fprintf(&buf, "\r\n")
	qpW2 := quotedprintable.NewWriter(&buf)
	_, _ = qpW2.Write([]byte(htmlBody))
	_ = qpW2.Close()
	fmt.Fprintf(&buf, "\r\n")

	fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	return buf.Bytes()
}
