package ai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// #519, architect fix list item 4: a real server with WriteTimeout 1 s, the real request-logger middleware, and a stub
// that waits 2 s, as a provider call does. The client must receive the 200 body. Without the AI write deadline the
// server cuts the response at 1 s.
func TestWriteDeadline_RealServerDeliversAnAnswerPastTheWriteTimeout(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.RequestLogger(zerolog.Nop()))
	r.Group(func(gr chi.Router) {
		gr.Use(WriteDeadline(AIWriteBudget))
		gr.Get("/api/v1/admin/ai/insights/{examId}", func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(2 * time.Second)
			_, _ = io.WriteString(w, `{"data":"answer"}`)
		})
	})

	srv := httptest.NewUnstartedServer(r)
	srv.Config.WriteTimeout = time.Second
	srv.Start()
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL + "/api/v1/admin/ai/insights/e1")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading the body failed; the server cut the answer: %v", err)
	}
	if resp.StatusCode != http.StatusOK || string(body) != `{"data":"answer"}` {
		t.Fatalf("want 200 with the answer, got %d %q", resp.StatusCode, body)
	}
}
