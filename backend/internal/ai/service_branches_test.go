package ai

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGenerateQuestions_CategoryLookupError_Wrapped(t *testing.T) {
	boom := errors.New("no category")
	svc := NewService(&mockRepository{categoryNameErr: boom}, &mockClient{}, "m", newLogger())
	_, err := svc.GenerateQuestions(context.Background(), "u", validRequest())
	if !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateQuestions_RateLimitLookupError_Wrapped(t *testing.T) {
	boom := errors.New("count failed")
	svc := NewService(&mockRepository{countErr: boom}, &mockClient{}, "m", newLogger())
	_, err := svc.GenerateQuestions(context.Background(), "u", validRequest())
	if !errors.Is(err, boom) || errors.Is(err, ErrAIRateLimited) {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateQuestions_ClientReturnsErrAIUnavailable(t *testing.T) {
	svc := NewService(&mockRepository{}, &mockClient{err: ErrAIUnavailable}, "m", newLogger())
	_, err := svc.GenerateQuestions(context.Background(), "u", validRequest())
	if err != ErrAIUnavailable {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateQuestions_LogUsageFailureIsNonFatal(t *testing.T) {
	repo := &mockRepository{logErr: errors.New("log failed")}
	svc := NewService(repo, &mockClient{text: validJSONResponse, tokens: 5}, "m", newLogger())
	qs, err := svc.GenerateQuestions(context.Background(), "u", validRequest())
	if err != nil || len(qs) != 1 || !repo.logCalled {
		t.Fatalf("got (%v,%v) logCalled=%v", qs, err, repo.logCalled)
	}
}

func TestGetInsights_StaleCache_Regenerates(t *testing.T) {
	repo := &mockRepository{
		insightCache:    &InsightResult{Insights: []string{"old"}, GeneratedAt: time.Now().Add(-25 * time.Hour), Cached: true},
		examInsightData: &ExamInsightData{ExamTitle: "T"},
	}
	svc := NewService(repo, &mockClient{text: `["a","b","c"]`, tokens: 9}, "m", newLogger())
	res, err := svc.GetInsights(scopedCtx("super_admin"), "e", "t", "u", false)
	if err != nil || res.Cached || len(res.Insights) != 3 || !repo.upsertCalled {
		t.Fatalf("got (%+v,%v) upsert=%v", res, err, repo.upsertCalled)
	}
	if repo.lastLog.Feature != featureExamInsights || repo.lastLog.TokensUsed != 9 || repo.lastLog.UserID != "u" {
		t.Fatalf("usage log = %+v", repo.lastLog)
	}
}

func TestGetInsights_CacheReadErrorIsNonFatal(t *testing.T) {
	repo := &mockRepository{
		insightCacheErr: errors.New("cache down"),
		examInsightData: &ExamInsightData{},
	}
	svc := NewService(repo, &mockClient{text: `["a"]`}, "m", newLogger())
	res, err := svc.GetInsights(scopedCtx("super_admin"), "e", "t", "u", false)
	if err != nil || len(res.Insights) != 1 { // partial (<3) results accepted
		t.Fatalf("got (%+v,%v)", res, err)
	}
}

func TestGetInsights_DataGatherErrorWrapped(t *testing.T) {
	boom := errors.New("sql broke")
	repo := &mockRepository{examInsightErr: boom}
	svc := NewService(repo, &mockClient{}, "m", newLogger())
	_, err := svc.GetInsights(scopedCtx("super_admin"), "e", "t", "u", true)
	if !errors.Is(err, boom) || errors.Is(err, ErrExamNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestGetInsights_MalformedJSON_ReturnsUnavailable(t *testing.T) {
	repo := &mockRepository{examInsightData: &ExamInsightData{}}
	svc := NewService(repo, &mockClient{text: "not json"}, "m", newLogger())
	_, err := svc.GetInsights(scopedCtx("super_admin"), "e", "t", "u", true)
	if err != ErrAIUnavailable || repo.upsertCalled || repo.logCalled {
		t.Fatalf("got %v upsert=%v log=%v", err, repo.upsertCalled, repo.logCalled)
	}
}

func TestGetInsights_LogAndUpsertFailuresNonFatal(t *testing.T) {
	repo := &mockRepository{
		examInsightData: &ExamInsightData{},
		logErr:          errors.New("log"),
		upsertErr:       errors.New("upsert"),
	}
	svc := NewService(repo, &mockClient{text: `["a","b","c","d"]`}, "m", newLogger())
	res, err := svc.GetInsights(scopedCtx("super_admin"), "e", "t", "u", true)
	if err != nil || len(res.Insights) != 4 {
		t.Fatalf("got (%+v,%v)", res, err)
	}
}

func TestGetLoyaltyNarrative_RepoErrorsWrapped(t *testing.T) {
	boom := errors.New("db")
	cases := map[string]*mockRepository{
		"track":      {sessionTrackErr: boom},
		"department": {sessionTrack: "loyalty", inDepartmentErr: boom},
		"likert":     {sessionTrack: "loyalty", inDepartment: true, likertResponsesErr: boom},
	}
	for name, repo := range cases {
		t.Run(name, func(t *testing.T) {
			svc := NewService(repo, &mockClient{text: "x"}, "m", newLogger())
			_, err := svc.GetLoyaltyNarrative(context.Background(), "s", "a", "department_admin")
			if !errors.Is(err, boom) || errors.Is(err, ErrForbidden) || errors.Is(err, ErrLoyaltySessionNotFound) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestGetLoyaltyNarrative_TrimsNarrativeAndLogUsageNonFatal(t *testing.T) {
	repo := &mockRepository{sessionTrack: "loyalty", inDepartment: true, logErr: errors.New("log")}
	svc := NewService(repo, &mockClient{text: "  Profile text.\n", tokens: 3}, "m", newLogger())
	res, err := svc.GetLoyaltyNarrative(context.Background(), "s", "a", "department_admin")
	if err != nil || res.Narrative != "Profile text." {
		t.Fatalf("got (%+v,%v)", res, err)
	}
}
