package email

import (
	"bufio"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeSMTP is a tiny in-process SMTP server bound to a loopback ephemeral port.
// It exists only so smtpSend can be exercised deterministically; nothing leaves
// the machine.
type fakeSMTP struct {
	ln       net.Listener
	Host     string
	Port     int
	failMail bool
	failRcpt bool
	failAuth bool
	failData bool

	mu       sync.Mutex
	messages []string // raw DATA bodies
	rcpts    []string
	authSeen bool
}

func startFakeSMTP(t *testing.T, mutate func(*fakeSMTP)) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("loopback listen unavailable: %v", err)
	}
	s := &fakeSMTP{ln: ln}
	addr := ln.Addr().(*net.TCPAddr)
	s.Host, s.Port = "127.0.0.1", addr.Port
	if mutate != nil {
		mutate(s)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go s.serve(c)
		}
	}()
	t.Cleanup(func() { _ = ln.Close() })
	return s
}

func (s *fakeSMTP) serve(c net.Conn) {
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	r := bufio.NewReader(c)
	w := func(line string) { _, _ = c.Write([]byte(line + "\r\n")) }
	w("220 fake ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			w("250-fake")
			w("250 AUTH PLAIN")
		case strings.HasPrefix(cmd, "AUTH"):
			s.mu.Lock()
			s.authSeen = true
			s.mu.Unlock()
			if s.failAuth {
				w("535 auth failed")
			} else {
				w("235 ok")
			}
		case strings.HasPrefix(cmd, "MAIL"):
			if s.failMail {
				w("550 sender rejected")
			} else {
				w("250 ok")
			}
		case strings.HasPrefix(cmd, "RCPT"):
			if s.failRcpt {
				w("550 rcpt rejected")
				continue
			}
			s.mu.Lock()
			s.rcpts = append(s.rcpts, strings.TrimSpace(line))
			s.mu.Unlock()
			w("250 ok")
		case cmd == "DATA":
			if s.failData {
				w("554 no data")
				continue
			}
			w("354 go")
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
			s.mu.Lock()
			s.messages = append(s.messages, sb.String())
			s.mu.Unlock()
			w("250 queued")
		case cmd == "RSET", cmd == "NOOP":
			w("250 ok")
		case cmd == "QUIT":
			w("221 bye")
			return
		default:
			w("500 unknown")
		}
	}
}

func (s *fakeSMTP) snapshot() (msgs, rcpts []string, auth bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.messages...), append([]string(nil), s.rcpts...), s.authSeen
}
