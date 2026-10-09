package ai

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
)

// ISS-82: exams has no tenant_id column, so GetExamInsightData must not select
// or compare it, and session_questions orders by sort_order (not order_num).

func TestGetExamInsightData_NotFound(t *testing.T) {
	db, f := newFakeDB(t)
	repo := NewRepository(db)

	_, err := repo.GetExamInsightData(context.Background(), "exam-1", "public")
	if !errors.Is(err, ErrExamNotFound) {
		t.Fatalf("want ErrExamNotFound, got %v", err)
	}
	if len(f.queries) != 1 {
		t.Fatalf("expected only the header query, got %d", len(f.queries))
	}
	if strings.Contains(f.queries[0], "tenant_id") {
		t.Fatalf("header query references nonexistent exams.tenant_id: %s", f.queries[0])
	}
}

func TestGetExamInsightData_IgnoresTenantAndUsesSortOrder(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"title", "passing_score_pct"}, [][]driver.Value{{"Safety 101", int64(70)}})
	f.queue([]string{"total_attempts", "pass_rate", "avg_score_pct", "avg_completion_secs"},
		[][]driver.Value{{int64(3), 0.6667, 0.8, 600.0}})
	f.queue([]string{"order_num", "stem", "correct_rate", "avg_time_secs"},
		[][]driver.Value{{int64(1), "Q1", 0.5, 30.0}})
	repo := NewRepository(db)

	// A tenant string that differs from anything stored must not matter.
	data, err := repo.GetExamInsightData(context.Background(), "exam-1", "some-other-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data == nil || data.ExamTitle != "Safety 101" || data.TotalAttempts != 3 || len(data.QuestionStats) != 1 {
		t.Fatalf("unexpected data: %+v", data)
	}
	for _, q := range f.queries {
		if strings.Contains(q, "tenant_id") {
			t.Fatalf("query references nonexistent tenant_id: %s", q)
		}
		if strings.Contains(q, "order_num") && strings.Contains(q, "sq.order_num") {
			t.Fatalf("query references nonexistent session_questions.order_num: %s", q)
		}
	}
	if !strings.Contains(f.queries[2], "sq.sort_order") {
		t.Fatalf("per-question query must order by session_questions.sort_order: %s", f.queries[2])
	}
}

func TestGetSessionCategoryTrack_NotFoundAndNoExamCategoryColumn(t *testing.T) {
	db, f := newFakeDB(t)
	repo := NewRepository(db)

	_, _, err := repo.GetSessionCategoryTrack(context.Background(), "sess-1")
	if !errors.Is(err, ErrLoyaltySessionNotFound) {
		t.Fatalf("want ErrLoyaltySessionNotFound, got %v", err)
	}
	if strings.Contains(f.queries[0], "e.category_id") {
		t.Fatalf("query references nonexistent exams.category_id: %s", f.queries[0])
	}
	if !strings.Contains(f.queries[0], "exam_question_rules") {
		t.Fatalf("track must be derived via exam_question_rules.category_id: %s", f.queries[0])
	}
}

func TestGetSessionCategoryTrack_Found(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"track", "user_id"}, [][]driver.Value{{"loyalty", "user-9"}})
	repo := NewRepository(db)

	track, uid, err := repo.GetSessionCategoryTrack(context.Background(), "sess-1")
	if err != nil || track != "loyalty" || uid != "user-9" {
		t.Fatalf("got (%q,%q,%v)", track, uid, err)
	}
}

// ISS-093: exams.passing_score_pct is numeric(5,2); pgx hands it over as the
// string "70.00", which must not break the scan into the insight data.
func TestGetExamInsightData_NumericPassingScore(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"title", "passing_score_pct"}, [][]driver.Value{{"Safety 101", "70.00"}})
	f.queue([]string{"total_attempts", "pass_rate", "avg_score_pct", "avg_completion_secs"},
		[][]driver.Value{{int64(0), nil, nil, nil}})
	f.queue([]string{"order_num", "stem", "correct_rate", "avg_time_secs"}, nil)
	repo := NewRepository(db)

	data, err := repo.GetExamInsightData(context.Background(), "exam-1", "public")
	if err != nil {
		t.Fatalf("numeric passing_score_pct must scan: %v", err)
	}
	if data.PassingScorePct != 70 {
		t.Fatalf("PassingScorePct = %d, want 70", data.PassingScorePct)
	}
}
