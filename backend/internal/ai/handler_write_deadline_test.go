package ai

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/config"
)

// #519: the server's WriteTimeout (15 s) would cut an insights answer that waits for the provider. The insights handler
// gives its own response a write deadline through http.ResponseController, which reaches the connection through the
// request logger's wrapper (chi's WrapResponseWriter implements Unwrap).

// deadlineWriter records the write deadline the handler asks for.
type deadlineWriter struct {
	*httptest.ResponseRecorder
	deadline time.Time
	set      bool
}

func (d *deadlineWriter) SetWriteDeadline(t time.Time) error {
	d.deadline, d.set = t, true
	return nil
}

func TestHandler_GetInsights_SetsAWriteDeadlineCoveringTheAITimeout(t *testing.T) {
	svc := &fnService{insights: func(string, string, string, bool) (*InsightResult, error) {
		return &InsightResult{Insights: []string{"a"}, GeneratedAt: time.Now().UTC()}, nil
	}}
	w := &deadlineWriter{ResponseRecorder: httptest.NewRecorder()}
	before := time.Now()
	NewHandler(svc).HandleGetInsights(w, paramRequest("/x", "examId", "exam-1", "u1"))

	if !w.set {
		t.Fatal("the insights handler must set a write deadline")
	}
	if got := w.deadline.Sub(before); got < InsightsWriteBudget-5*time.Second || got > InsightsWriteBudget+5*time.Second {
		t.Fatalf("the write deadline is %v from now, want about %v", got, InsightsWriteBudget)
	}
}

func TestInsightsWriteBudget_CoversTheLargestAITimeout(t *testing.T) {
	need := time.Duration(config.MaxAITimeoutSeconds+30) * time.Second
	if InsightsWriteBudget < need {
		t.Fatalf("InsightsWriteBudget %v must cover AI_TIMEOUT_SECONDS max plus 30 s (%v)", InsightsWriteBudget, need)
	}
}
