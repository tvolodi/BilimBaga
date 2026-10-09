package ai

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRepo_CountAIUsageLastHour(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"count"}, [][]driver.Value{{int64(7)}})
	n, err := NewRepository(db).CountAIUsageLastHour(context.Background(), "u1", "exam_insights")
	if err != nil || n != 7 {
		t.Fatalf("got (%d,%v)", n, err)
	}
	if !strings.Contains(f.queries[0], "ai_usage_log") {
		t.Fatalf("query = %s", f.queries[0])
	}
	if f.args[0][0] != "u1" || f.args[0][1] != "exam_insights" {
		t.Fatalf("args = %v", f.args[0])
	}
	// Window start must be ~1h ago.
	since, ok := f.args[0][2].(time.Time)
	if !ok || time.Since(since) < 59*time.Minute || time.Since(since) > 61*time.Minute {
		t.Fatalf("window arg = %v", f.args[0][2])
	}
}

func TestRepo_CountAIUsageLastHour_ErrorWrapped(t *testing.T) {
	db, f := newFakeDB(t)
	f.qErr = errors.New("conn reset")
	_, err := NewRepository(db).CountAIUsageLastHour(context.Background(), "u1", "x")
	if err == nil || !errors.Is(err, f.qErr) || !strings.Contains(err.Error(), "CountAIUsageLastHour") {
		t.Fatalf("got %v", err)
	}
}

func TestRepo_LogUsage(t *testing.T) {
	db, f := newFakeDB(t)
	err := NewRepository(db).LogUsage(context.Background(), UsageLog{UserID: "u1", Feature: "f", TokensUsed: 42, Model: "m"})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !strings.Contains(f.queries[0], "INSERT INTO ai_usage_log") {
		t.Fatalf("query = %s", f.queries[0])
	}
	a := f.args[0]
	if a[0] != "u1" || a[1] != "f" || a[2] != int64(42) || a[3] != "m" {
		t.Fatalf("args = %v", a)
	}
}

func TestRepo_LogUsage_ErrorWrapped(t *testing.T) {
	db, f := newFakeDB(t)
	f.xErr = errors.New("insert failed")
	err := NewRepository(db).LogUsage(context.Background(), UsageLog{})
	if !errors.Is(err, f.xErr) || !strings.Contains(err.Error(), "LogUsage") {
		t.Fatalf("got %v", err)
	}
}

func TestRepo_GetCategoryName(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"name"}, [][]driver.Value{{"Fire Safety"}})
	name, err := NewRepository(db).GetCategoryName(context.Background(), "cat-1")
	if err != nil || name != "Fire Safety" {
		t.Fatalf("got (%q,%v)", name, err)
	}
	if f.args[0][0] != "cat-1" {
		t.Fatalf("args = %v", f.args[0])
	}
}

func TestRepo_GetCategoryName_NotFoundWrapped(t *testing.T) {
	db, _ := newFakeDB(t)
	_, err := NewRepository(db).GetCategoryName(context.Background(), "missing")
	if err == nil || !strings.Contains(err.Error(), "GetCategoryName") {
		t.Fatalf("got %v", err)
	}
}

func TestRepo_GetInsightCache_Hit(t *testing.T) {
	db, f := newFakeDB(t)
	ts := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	f.queue([]string{"insights", "generated_at"}, [][]driver.Value{{[]byte(`["x","y","z"]`), ts}})
	res, err := NewRepository(db).GetInsightCache(context.Background(), "exam-1")
	if err != nil || res == nil {
		t.Fatalf("got (%v,%v)", res, err)
	}
	if !res.Cached || len(res.Insights) != 3 || res.Insights[1] != "y" || !res.GeneratedAt.Equal(ts) {
		t.Fatalf("result = %+v", res)
	}
}

func TestRepo_GetInsightCache_MissIsNilNil(t *testing.T) {
	db, _ := newFakeDB(t)
	res, err := NewRepository(db).GetInsightCache(context.Background(), "exam-1")
	if res != nil || err != nil {
		t.Fatalf("cache miss must be (nil,nil), got (%v,%v)", res, err)
	}
}

func TestRepo_GetInsightCache_BadJSONAndQueryError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"insights", "generated_at"}, [][]driver.Value{{[]byte(`{not json`), time.Now()}})
	_, err := NewRepository(db).GetInsightCache(context.Background(), "e")
	if err == nil || !strings.Contains(err.Error(), "unmarshal insights") {
		t.Fatalf("got %v", err)
	}

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("db down")
	_, err = NewRepository(db2).GetInsightCache(context.Background(), "e")
	if !errors.Is(err, f2.qErr) || !strings.Contains(err.Error(), "GetInsightCache") {
		t.Fatalf("got %v", err)
	}
}

func TestRepo_UpsertInsightCache(t *testing.T) {
	db, f := newFakeDB(t)
	err := NewRepository(db).UpsertInsightCache(context.Background(), "exam-1", "user-1", []string{"a", "b"})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !strings.Contains(f.queries[0], "ON CONFLICT (exam_id)") {
		t.Fatalf("query = %s", f.queries[0])
	}
	a := f.args[0]
	if a[0] != "exam-1" || a[2] != "user-1" {
		t.Fatalf("args = %v", a)
	}
	if b, ok := a[1].([]byte); !ok || string(b) != `["a","b"]` {
		t.Fatalf("insights arg = %#v", a[1])
	}
}

func TestRepo_UpsertInsightCache_ErrorWrapped(t *testing.T) {
	db, f := newFakeDB(t)
	f.xErr = errors.New("upsert failed")
	err := NewRepository(db).UpsertInsightCache(context.Background(), "e", "u", nil)
	if !errors.Is(err, f.xErr) || !strings.Contains(err.Error(), "UpsertInsightCache") {
		t.Fatalf("got %v", err)
	}
}

func TestRepo_GetExamInsightData_FullMapping(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"title", "passing_score_pct"}, [][]driver.Value{{"Safety", 69.6}})
	f.queue([]string{"total_attempts", "pass_rate", "avg_score_pct", "avg_completion_secs"},
		[][]driver.Value{{int64(10), 0.6, 0.75, 900.0}})
	f.queue([]string{"order_num", "stem", "correct_rate", "avg_time_secs"},
		[][]driver.Value{
			{int64(1), "Q1", 0.9, 12.0},
			{int64(2), "Q2", nil, nil}, // NULL stats must map to zero values
		})
	data, err := NewRepository(db).GetExamInsightData(context.Background(), "exam-1", "public")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if data.PassingScorePct != 70 || data.PassRate != 0.6 || data.AvgScorePct != 0.75 || data.AvgCompletionSecs != 900 {
		t.Fatalf("summary mapping wrong: %+v", data)
	}
	if len(data.QuestionStats) != 2 {
		t.Fatalf("question stats = %+v", data.QuestionStats)
	}
	q1, q2 := data.QuestionStats[0], data.QuestionStats[1]
	if q1.OrderNum != 1 || q1.Stem != "Q1" || q1.CorrectRate != 0.9 || q1.AvgTimeSecs != 12 {
		t.Fatalf("q1 = %+v", q1)
	}
	if q2.CorrectRate != 0 || q2.AvgTimeSecs != 0 {
		t.Fatalf("q2 NULLs must map to zero: %+v", q2)
	}
}

func TestRepo_GetExamInsightData_HeaderQueryError(t *testing.T) {
	db, f := newFakeDB(t)
	f.qErr = errors.New("db down")
	_, err := NewRepository(db).GetExamInsightData(context.Background(), "e", "")
	if errors.Is(err, ErrExamNotFound) || !errors.Is(err, f.qErr) || !strings.Contains(err.Error(), "fetch exam") {
		t.Fatalf("real DB errors must not be reported as not-found: %v", err)
	}
}

func TestRepo_GetExamInsightData_SummaryScanError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"title", "passing_score_pct"}, [][]driver.Value{{"T", 50.0}})
	// no summary rows queued -> sql.ErrNoRows from the aggregate query
	_, err := NewRepository(db).GetExamInsightData(context.Background(), "e", "")
	if err == nil || !strings.Contains(err.Error(), "aggregate sessions") {
		t.Fatalf("got %v", err)
	}
}

func TestRepo_GetExamInsightData_QuestionScanError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"title", "passing_score_pct"}, [][]driver.Value{{"T", 50.0}})
	f.queue([]string{"total_attempts", "pass_rate", "avg_score_pct", "avg_completion_secs"},
		[][]driver.Value{{int64(1), nil, nil, nil}})
	f.queue([]string{"order_num", "stem", "correct_rate", "avg_time_secs"},
		[][]driver.Value{{"not-an-int", "Q", 0.1, 1.0}})
	_, err := NewRepository(db).GetExamInsightData(context.Background(), "e", "")
	if err == nil || !strings.Contains(err.Error(), "scan question row") {
		t.Fatalf("got %v", err)
	}
}

func TestRepo_GetSessionCategoryTrack_NullTrackAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"track", "user_id"}, [][]driver.Value{{nil, "user-3"}})
	track, uid, err := NewRepository(db).GetSessionCategoryTrack(context.Background(), "s1")
	if err != nil || track != "" || uid != "user-3" {
		t.Fatalf("NULL track must map to empty: (%q,%q,%v)", track, uid, err)
	}

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("db down")
	_, _, err = NewRepository(db2).GetSessionCategoryTrack(context.Background(), "s1")
	if errors.Is(err, ErrLoyaltySessionNotFound) || !errors.Is(err, f2.qErr) {
		t.Fatalf("got %v", err)
	}
}

func TestRepo_IsEmployeeInAdminDepartment(t *testing.T) {
	for _, want := range []bool{true, false} {
		db, f := newFakeDB(t)
		f.queue([]string{"exists"}, [][]driver.Value{{want}})
		got, err := NewRepository(db).IsEmployeeInAdminDepartment(context.Background(), "admin-1", "emp-1")
		if err != nil || got != want {
			t.Fatalf("want %v got (%v,%v)", want, got, err)
		}
		// Argument order: employee first, admin second.
		if f.args[0][0] != "emp-1" || f.args[0][1] != "admin-1" {
			t.Fatalf("args = %v", f.args[0])
		}
	}

	db, f := newFakeDB(t)
	f.qErr = errors.New("db down")
	ok, err := NewRepository(db).IsEmployeeInAdminDepartment(context.Background(), "a", "e")
	if ok || !errors.Is(err, f.qErr) || !strings.Contains(err.Error(), "IsEmployeeInAdminDepartment") {
		t.Fatalf("got (%v,%v)", ok, err)
	}
}

func TestRepo_CollectLikertResponses_PolarityInversion(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"dimension_label", "raw_weight", "likert_polarity"}, [][]driver.Value{
		{"Loyalty", int64(5), "positive"},
		{"Loyalty", int64(5), "negative"}, // 6-5 = 1
		{"", int64(2), "negative"},        // 6-2 = 4
	})
	res, err := NewRepository(db).CollectLikertResponses(context.Background(), "s1")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	want := []LikertResponseData{{"Loyalty", 5}, {"Loyalty", 1}, {"", 4}}
	if len(res) != len(want) {
		t.Fatalf("res = %+v", res)
	}
	for i := range want {
		if res[i] != want[i] {
			t.Fatalf("res[%d] = %+v, want %+v", i, res[i], want[i])
		}
	}
}

func TestRepo_CollectLikertResponses_EmptyAndErrors(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"dimension_label", "raw_weight", "likert_polarity"}, nil)
	res, err := NewRepository(db).CollectLikertResponses(context.Background(), "s1")
	if err != nil || len(res) != 0 {
		t.Fatalf("got (%v,%v)", res, err)
	}

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("db down")
	_, err = NewRepository(db2).CollectLikertResponses(context.Background(), "s1")
	if !errors.Is(err, f2.qErr) || !strings.Contains(err.Error(), "CollectLikertResponses") {
		t.Fatalf("got %v", err)
	}

	db3, f3 := newFakeDB(t)
	f3.queue([]string{"dimension_label", "raw_weight", "likert_polarity"},
		[][]driver.Value{{"L", "bad", "positive"}})
	_, err = NewRepository(db3).CollectLikertResponses(context.Background(), "s1")
	if err == nil || !strings.Contains(err.Error(), "scan") {
		t.Fatalf("got %v", err)
	}
}
