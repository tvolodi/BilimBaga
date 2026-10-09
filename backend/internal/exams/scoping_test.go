package exams

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// FR-BB117: a custom role holding exams:assign must be department-limited (default-deny).
func TestCreateAssignment_CustomRole_DepartmentLimited(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	_, err := svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID: "exam-1", AssigneeType: "all", AssignedBy: "u", CallerRole: "qa_lead", CallerDeptID: "dept-1",
	})
	assert.ErrorIs(t, err, ErrForbidden)
	other := "dept-99"
	_, err = svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID: "exam-1", AssigneeType: "department", AssigneeID: &other, AssignedBy: "u", CallerRole: "qa_lead", CallerDeptID: "dept-1",
	})
	assert.ErrorIs(t, err, ErrForbidden)
	own := "dept-1"
	_, err = svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID: "exam-1", AssigneeType: "department", AssigneeID: &own, AssignedBy: "u", CallerRole: "qa_lead", CallerDeptID: "dept-1",
	})
	assert.NoError(t, err)
}

func TestCreateAssignment_Examiner_StaysOrgWide(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	_, err := NewService(repo).CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID: "exam-1", AssigneeType: "all", AssignedBy: "u", CallerRole: "examiner",
	})
	assert.NoError(t, err)
}
