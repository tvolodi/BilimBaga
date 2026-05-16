package sessions

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/ai"
)

// mockAIClient satisfies ai.AnthropicClient for testing.
type mockAIClient struct {
	response string
	tokens   int
	err      error
}

func (m *mockAIClient) GenerateText(_ context.Context, _, _ string) (string, int, error) {
	return m.response, m.tokens, m.err
}

var _ ai.AnthropicClient = (*mockAIClient)(nil)

func strPtr(s string) *string { return &s }

func newTestAIEngine(client *mockAIClient) *AIGradingEngine {
	return &AIGradingEngine{
		aiClient: client,
		model:    "test-model",
		db:       nil, // logAIUsage is a no-op when db is nil
		logger:   slog.Default(),
	}
}

func TestGradeShortText_AutoGradeFalse_ReturnsPendingManual(t *testing.T) {
	e := newTestAIEngine(&mockAIClient{})
	score, max, status, reasoning := e.gradeShortText("s1", "q1", false, strPtr("model"), strPtr("answer"))
	if status != GradingStatusPendingManual {
		t.Fatalf("expected pending_manual, got %s", status)
	}
	if score != 0 || max != 1 || reasoning != nil {
		t.Fatalf("expected zero score and nil reasoning, got score=%v max=%v reasoning=%v", score, max, reasoning)
	}
}

func TestGradeShortText_ModelAnswerNil_ReturnsPendingManual(t *testing.T) {
	e := newTestAIEngine(&mockAIClient{})
	_, _, status, _ := e.gradeShortText("s1", "q1", true, nil, strPtr("answer"))
	if status != GradingStatusPendingManual {
		t.Fatalf("expected pending_manual, got %s", status)
	}
}

func TestGradeShortText_ModelAnswerEmpty_ReturnsPendingManual(t *testing.T) {
	e := newTestAIEngine(&mockAIClient{})
	_, _, status, _ := e.gradeShortText("s1", "q1", true, strPtr("   "), strPtr("answer"))
	if status != GradingStatusPendingManual {
		t.Fatalf("expected pending_manual for blank model answer, got %s", status)
	}
}

func TestGradeShortText_TextAnswerNil_ReturnsPendingManual(t *testing.T) {
	e := newTestAIEngine(&mockAIClient{})
	_, _, status, _ := e.gradeShortText("s1", "q1", true, strPtr("model"), nil)
	if status != GradingStatusPendingManual {
		t.Fatalf("expected pending_manual, got %s", status)
	}
}

func TestGradeShortText_AIError_FallsBackToPendingManual(t *testing.T) {
	e := newTestAIEngine(&mockAIClient{err: errors.New("network timeout")})
	_, _, status, _ := e.gradeShortText("s1", "q1", true, strPtr("model"), strPtr("answer"))
	if status != GradingStatusPendingManual {
		t.Fatalf("expected pending_manual on AI error, got %s", status)
	}
}

func TestGradeShortText_MalformedJSON_FallsBackToPendingManual(t *testing.T) {
	e := newTestAIEngine(&mockAIClient{response: "no json here", tokens: 5})
	_, _, status, _ := e.gradeShortText("s1", "q1", true, strPtr("model"), strPtr("answer"))
	if status != GradingStatusPendingManual {
		t.Fatalf("expected pending_manual on malformed JSON, got %s", status)
	}
}

func TestGradeShortText_ScoreTooHigh_FallsBackToPendingManual(t *testing.T) {
	e := newTestAIEngine(&mockAIClient{response: `{"score_pct": 150, "reasoning": "too high"}`, tokens: 5})
	_, _, status, _ := e.gradeShortText("s1", "q1", true, strPtr("model"), strPtr("answer"))
	if status != GradingStatusPendingManual {
		t.Fatalf("expected pending_manual for score>100, got %s", status)
	}
}

func TestGradeShortText_ScoreNegative_FallsBackToPendingManual(t *testing.T) {
	e := newTestAIEngine(&mockAIClient{response: `{"score_pct": -10, "reasoning": "negative"}`, tokens: 5})
	_, _, status, _ := e.gradeShortText("s1", "q1", true, strPtr("model"), strPtr("answer"))
	if status != GradingStatusPendingManual {
		t.Fatalf("expected pending_manual for score<0, got %s", status)
	}
}

func TestGradeShortText_HappyPath_ReturnsAIGraded(t *testing.T) {
	e := newTestAIEngine(&mockAIClient{
		response: `{"score_pct": 80, "reasoning": "mostly correct"}`,
		tokens:   12,
	})
	score, max, status, reasoning := e.gradeShortText("s1", "q1", true, strPtr("Paris"), strPtr("Paris, France"))
	if status != GradingStatusAIGraded {
		t.Fatalf("expected ai_graded, got %s", status)
	}
	if score != 0.8 {
		t.Errorf("expected score=0.8, got %v", score)
	}
	if max != 1.0 {
		t.Errorf("expected max=1.0, got %v", max)
	}
	if reasoning == nil || *reasoning != "mostly correct" {
		t.Errorf("unexpected reasoning: %v", reasoning)
	}
}

func TestGradeShortText_ZeroScore_ReturnsAIGraded(t *testing.T) {
	e := newTestAIEngine(&mockAIClient{
		response: `{"score_pct": 0, "reasoning": "completely wrong"}`,
		tokens:   8,
	})
	score, _, status, _ := e.gradeShortText("s1", "q1", true, strPtr("Paris"), strPtr("Berlin"))
	if status != GradingStatusAIGraded {
		t.Fatalf("expected ai_graded for score=0, got %s", status)
	}
	if score != 0.0 {
		t.Errorf("expected score=0.0, got %v", score)
	}
}

func TestGradeShortText_JSONWithSurroundingText_ParsesCorrectly(t *testing.T) {
	e := newTestAIEngine(&mockAIClient{
		response: `Here is my assessment: {"score_pct": 60, "reasoning": "partial"} Thank you.`,
		tokens:   15,
	})
	_, _, status, reasoning := e.gradeShortText("s1", "q1", true, strPtr("model"), strPtr("answer"))
	if status != GradingStatusAIGraded {
		t.Fatalf("expected ai_graded when JSON is embedded in text, got %s", status)
	}
	if reasoning == nil || *reasoning != "partial" {
		t.Errorf("unexpected reasoning: %v", reasoning)
	}
}
