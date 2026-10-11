package ai

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/config"
)

// #519: the server's WriteTimeout (15 s) would cut an AI answer that waits for the provider. The WriteDeadline middleware
// gives every response on the AI routes its own write deadline through http.ResponseController.

// deadlineWriter records the write deadline the middleware asks for.
type deadlineWriter struct {
	*httptest.ResponseRecorder
	deadline time.Time
	set      bool
}

func (d *deadlineWriter) SetWriteDeadline(t time.Time) error {
	d.deadline, d.set = t, true
	return nil
}

func TestWriteDeadline_SetsADeadlineCoveringTheBudget(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	w := &deadlineWriter{ResponseRecorder: httptest.NewRecorder()}
	before := time.Now()
	WriteDeadline(AIWriteBudget)(next).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/ai/insights/e1", nil))

	if !w.set {
		t.Fatal("the AI route middleware must set a write deadline")
	}
	if got := w.deadline.Sub(before); got < AIWriteBudget-5*time.Second || got > AIWriteBudget+5*time.Second {
		t.Fatalf("the write deadline is %v from now, want about %v", got, AIWriteBudget)
	}
}

func TestAIWriteBudget_CoversTheLargestAITimeout(t *testing.T) {
	need := time.Duration(config.MaxAITimeoutSeconds+30) * time.Second
	if AIWriteBudget < need {
		t.Fatalf("AIWriteBudget %v must cover AI_TIMEOUT_SECONDS max plus 30 s (%v)", AIWriteBudget, need)
	}
}
