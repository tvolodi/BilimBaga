package email

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer is a goroutine-safe log sink.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// startStubSMTP starts a minimal SMTP server and delivers each received message body on the channel.
func startStubSMTP(t *testing.T) (port int, msgs <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	ch := make(chan string, 4)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close() //nolint:errcheck
				r := bufio.NewReader(c)
				fmt.Fprint(c, "220 stub ESMTP\r\n")
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					cmd := strings.ToUpper(strings.TrimSpace(line))
					switch {
					case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
						fmt.Fprint(c, "250 stub\r\n")
					case strings.HasPrefix(cmd, "DATA"):
						fmt.Fprint(c, "354 go ahead\r\n")
						var sb strings.Builder
						for {
							l, err := r.ReadString('\n')
							if err != nil {
								return
							}
							if l == ".\r\n" {
								break
							}
							sb.WriteString(l)
						}
						ch <- sb.String()
						fmt.Fprint(c, "250 queued\r\n")
					case strings.HasPrefix(cmd, "QUIT"):
						fmt.Fprint(c, "221 bye\r\n")
						return
					default:
						fmt.Fprint(c, "250 ok\r\n")
					}
				}
			}(conn)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, ch
}

// decodeParts returns the decoded text/plain and text/html bodies of a MIME message.
func decodeParts(t *testing.T, raw string) (text, html string) {
	t.Helper()
	m, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("parse message: %v", err)
	}
	_, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("content-type: %v", err)
	}
	mr := multipart.NewReader(m.Body, params["boundary"])
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		b, _ := io.ReadAll(quotedprintable.NewReader(p))
		if strings.HasPrefix(p.Header.Get("Content-Type"), "text/plain") {
			text = string(b)
		} else {
			html = string(b)
		}
	}
	return text, html
}

// FR-BB115 AC-3: link built from PublicAppURL, in both parts, locale rendered, token never logged.
func TestTriggerPasswordResetLink_SendsLinkAndNeverLogsToken(t *testing.T) {
	port, msgs := startStubSMTP(t)
	logs := &syncBuffer{}
	repo := &mockRepo{}
	svc := &EmailService{
		cfg:    Config{Host: "127.0.0.1", Port: port, From: "noreply@test.local", PublicAppURL: "https://app.example.com/"},
		repo:   repo,
		logger: slog.New(slog.NewTextHandler(logs, nil)),
	}
	const token = "TOKEN_abc-123_SECRETSECRETSECRETSECRETSECRETSECRET"

	svc.TriggerPasswordResetLink("user-1", token)

	select {
	case raw := <-msgs:
		text, html := decodeParts(t, raw)
		want := "https://app.example.com/reset-password?token=" + token
		if !strings.Contains(text, want) {
			t.Errorf("text part missing link %q:\n%s", want, text)
		}
		if !strings.Contains(html, `href="`+want+`"`) {
			t.Errorf("html part missing link %q:\n%s", want, html)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no message received")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !repo.logAttemptCalled {
		time.Sleep(10 * time.Millisecond)
	}
	if repo.loggedTmpl != "password_reset_link" || repo.loggedErr != nil {
		t.Errorf("email_log attempt = (%q, %v); want (password_reset_link, nil)", repo.loggedTmpl, repo.loggedErr)
	}
	if strings.Contains(logs.String(), token) {
		t.Errorf("token leaked into logs: %s", logs.String())
	}
}

func TestTriggerPasswordResetLink_SendFailureLoggedWithoutToken(t *testing.T) {
	logs := &syncBuffer{}
	repo := &mockRepo{}
	svc := &EmailService{
		cfg:    Config{Host: "127.0.0.1", Port: 1, From: "noreply@test.local", PublicAppURL: "https://app.example.com"},
		repo:   repo,
		logger: slog.New(slog.NewTextHandler(logs, nil)),
	}
	const token = "TOKEN_FAILPATH_SECRETSECRETSECRETSECRETSECRETSECRET"

	svc.TriggerPasswordResetLink("user-1", token)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !strings.Contains(logs.String(), "email send failed") {
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(logs.String(), "email send failed") {
		t.Fatalf("expected send failure to be logged, got: %q", logs.String())
	}
	if strings.Contains(logs.String(), token) {
		t.Errorf("token leaked into failure log: %s", logs.String())
	}
}
