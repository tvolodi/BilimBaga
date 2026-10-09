package exams

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake-DB coverage (follow-up #327) for the exams section, rule and assignment
// write paths left at 0% after #315. Each path gets success, not-found, duplicate
// or wrapped-error coverage where the fake can express it. The fake always
// reports one affected row, so the not-found branches of the Delete* functions
// are not reachable here.

var sectionCols = []string{"id", "exam_id", "title", "sort_order"}
var ruleCols = []string{"id", "exam_id", "section_id", "mode", "category_id", "tag_ids", "difficulty", "count", "sort_order"}

func TestCreateSection_ReturnsRowAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(sectionCols, [][]driver.Value{{"sec-1", "exam-1", "Safety", int64(1)}})
	s, err := NewRepository(db).CreateSection(context.Background(), "exam-1", SectionInput{Title: strp("Safety"), SortOrder: 1})
	require.NoError(t, err)
	assert.Equal(t, "sec-1", s.ID)
	require.NotNil(t, s.Title)
	assert.Equal(t, "Safety", *s.Title)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2).CreateSection(context.Background(), "exam-1", SectionInput{Title: strp("Safety")})
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "CreateSection")
}

func TestUpdateSection_UpdatesNotFoundAndError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(sectionCols, [][]driver.Value{{"sec-1", "exam-1", "Fire", int64(2)}})
	s, err := NewRepository(db).UpdateSection(context.Background(), "exam-1", "sec-1", SectionInput{Title: strp("Fire"), SortOrder: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, s.SortOrder)

	db2, f2 := newFakeDB(t)
	f2.queue(sectionCols, nil)
	_, err = NewRepository(db2).UpdateSection(context.Background(), "exam-1", "missing", SectionInput{})
	assert.ErrorIs(t, err, ErrNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).UpdateSection(context.Background(), "exam-1", "sec-1", SectionInput{})
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "UpdateSection")
}

func TestDeleteSection_DeletesAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db).DeleteSection(context.Background(), "exam-1", "sec-1"))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "DELETE FROM exam_sections")

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err := NewRepository(db2).DeleteSection(context.Background(), "exam-1", "sec-1")
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "DeleteSection")
}

func TestCreateRule_ReturnsRowAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(ruleCols, [][]driver.Value{{"rule-1", "exam-1", nil, "random", nil, []byte("[]"), nil, int64(3), int64(1)}})
	rule, err := NewRepository(db).CreateRule(context.Background(), "exam-1", QuestionRuleInput{Mode: "random", Count: 3, SortOrder: 1})
	require.NoError(t, err)
	assert.Equal(t, "rule-1", rule.ID)
	assert.Equal(t, 3, rule.Count)
	require.Len(t, f.args, 1)
	assert.Equal(t, []byte("[]"), f.args[0][4], "nil tag list is stored as an empty JSON array")

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2).CreateRule(context.Background(), "exam-1", QuestionRuleInput{Mode: "random", Count: 1})
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "CreateRule")
}

func TestUpdateRule_ReturnsRowAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue(ruleCols, [][]driver.Value{{"rule-1", "exam-1", nil, "manual", nil, []byte("[]"), nil, int64(2), int64(1)}})
	rule, err := NewRepository(db).UpdateRule(context.Background(), "rule-1", QuestionRuleInput{Mode: "manual", Count: 2})
	require.NoError(t, err)
	assert.Equal(t, "manual", rule.Mode)

	db2, f2 := newFakeDB(t)
	f2.qErr = errors.New("boom")
	_, err = NewRepository(db2).UpdateRule(context.Background(), "rule-1", QuestionRuleInput{Mode: "manual"})
	assert.ErrorIs(t, err, f2.qErr)
	assert.ErrorContains(t, err, "UpdateRule")
}

func TestDeleteRule_DeletesAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db).DeleteRule(context.Background(), "rule-1"))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "DELETE FROM exam_question_rules")

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err := NewRepository(db2).DeleteRule(context.Background(), "rule-1")
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "DeleteRule")
}

func TestCreateAssignment_SetsIDAndMapsDuplicate(t *testing.T) {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	db, f := newFakeDB(t)
	f.queue([]string{"id", "assigned_at"}, [][]driver.Value{{"asg-1", at}})
	a := &ExamAssignment{ExamID: "exam-1", AssigneeType: "all", AssignedBy: "user-1"}
	require.NoError(t, NewRepository(db).CreateAssignment(context.Background(), a))
	assert.Equal(t, "asg-1", a.ID)
	assert.True(t, a.AssignedAt.Equal(at))

	db2, f2 := newFakeDB(t)
	f2.qErr = &pq.Error{Code: "23505"}
	err := NewRepository(db2).CreateAssignment(context.Background(), &ExamAssignment{ExamID: "exam-1", AssigneeType: "all"})
	assert.ErrorIs(t, err, ErrAssignmentExists)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	err = NewRepository(db3).CreateAssignment(context.Background(), &ExamAssignment{ExamID: "exam-1", AssigneeType: "all"})
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "CreateAssignment")
}

func TestGetAssignmentByID_FoundNotFoundAndError(t *testing.T) {
	db, f := newFakeDB(t)
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	f.queue([]string{"id", "exam_id", "assignee_type", "assignee_id", "deadline", "assigned_by", "assigned_at"},
		[][]driver.Value{{"asg-1", "exam-1", "all", nil, nil, "user-1", at}})
	a, err := NewRepository(db).GetAssignmentByID(context.Background(), "asg-1")
	require.NoError(t, err)
	assert.Equal(t, "all", a.AssigneeType)

	db2, f2 := newFakeDB(t)
	f2.queue([]string{"id"}, nil)
	_, err = NewRepository(db2).GetAssignmentByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrAssignmentNotFound)

	db3, f3 := newFakeDB(t)
	f3.qErr = errors.New("boom")
	_, err = NewRepository(db3).GetAssignmentByID(context.Background(), "asg-1")
	assert.ErrorIs(t, err, f3.qErr)
	assert.ErrorContains(t, err, "GetAssignmentByID")
}

func TestDeleteAssignment_DeletesAndWrapsError(t *testing.T) {
	db, f := newFakeDB(t)
	require.NoError(t, NewRepository(db).DeleteAssignment(context.Background(), "asg-1"))
	require.Len(t, f.queries, 1)
	assert.Contains(t, f.queries[0], "DELETE FROM exam_assignments")

	db2, f2 := newFakeDB(t)
	f2.xErr = errors.New("boom")
	err := NewRepository(db2).DeleteAssignment(context.Background(), "asg-1")
	assert.ErrorIs(t, err, f2.xErr)
	assert.ErrorContains(t, err, "DeleteAssignment")
}

func strp(s string) *string { return &s }
