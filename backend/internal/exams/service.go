package exams

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/email"
)

// uuidPattern matches a UUID v4 string (case-insensitive).
var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// isValidUUID returns true if s is a valid UUID string.
func isValidUUID(s string) bool {
	return uuidPattern.MatchString(s)
}

// Service defines the business logic for the exams domain.
type Service interface {
	CreateExam(ctx context.Context, input CreateExamInput) (*Exam, error)
	GetExam(ctx context.Context, id string) (*ExamDetail, error)
	ListExams(ctx context.Context, filter ExamFilter) ([]*ExamListItem, int, error)
	UpdateExam(ctx context.Context, id string, input UpdateExamInput) (*Exam, error)
	TransitionStatus(ctx context.Context, id, newStatus string) (*Exam, error)
	Publish(ctx context.Context, id string) (*Exam, error)
	Archive(ctx context.Context, id string) (*Exam, error)
	DeleteExam(ctx context.Context, id string) error

	// UnpublishExam reverts an active exam back to draft status (FR-BB318).
	UnpublishExam(ctx context.Context, id string) (*Exam, error)

	CreateSection(ctx context.Context, examID string, input SectionInput) (*ExamSection, error)
	UpdateSection(ctx context.Context, examID, sectionID string, input SectionInput) (*ExamSection, error)
	DeleteSection(ctx context.Context, examID, sectionID string) error

	CreateRule(ctx context.Context, examID string, input QuestionRuleInput) (*ExamQuestionRule, error)
	UpdateRule(ctx context.Context, examID, ruleID string, input QuestionRuleInput) (*ExamQuestionRule, error)
	DeleteRule(ctx context.Context, examID, ruleID string) error
	SetManualQuestions(ctx context.Context, ruleID string, questions []ManualQuestionInput) error

	// Assignments (FR-BB33)
	CreateAssignment(ctx context.Context, input CreateAssignmentInput) (*ExamAssignment, error)
	DeleteAssignment(ctx context.Context, examID, assignmentID string, callerRole, callerDeptID string) error
	ListAssignments(ctx context.Context, examID string) ([]*AssignmentDetail, error)

	// GetEligibleCounts returns eligible question counts per rule for an exam (FR-BB315).
	GetEligibleCounts(ctx context.Context, examID string) ([]RuleEligibleCount, error)
}

type service struct {
	repo     Repository
	emailSvc *email.EmailService
}

// NewService returns a Service backed by the given Repository.
// An optional EmailService may be passed as the second argument to enable
// transactional email delivery; pass nil or omit to disable.
func NewService(repo Repository, emailSvc ...*email.EmailService) Service {
	s := &service{repo: repo}
	if len(emailSvc) > 0 {
		s.emailSvc = emailSvc[0]
	}
	return s
}

// validTransitions defines the allowed exam status state machine.
var validTransitions = map[string]string{
	"draft":  "active",
	"active": "archived",
}

func (s *service) CreateExam(ctx context.Context, input CreateExamInput) (*Exam, error) {
	// Validate availability window.
	if input.AvailableFrom != nil && input.AvailableUntil != nil &&
		!input.AvailableFrom.Before(*input.AvailableUntil) {
		return nil, fmt.Errorf("%w: available_from must be before available_until", ErrInvalidInput)
	}
	e := &Exam{
		Title:              input.Title,
		Description:        input.Description,
		Status:             "draft",
		TimeLimitMinutes:   input.TimeLimitMinutes,
		PassingScorePct:    input.PassingScorePct,
		MaxAttempts:        input.MaxAttempts,
		AvailableFrom:      input.AvailableFrom,
		AvailableUntil:     input.AvailableUntil,
		ShuffleQuestions:   input.ShuffleQuestions,
		ShuffleOptions:     input.ShuffleOptions,
		ShowAnswers:        input.ShowAnswers,
		OnTabSwitch:        input.OnTabSwitch,
		CertificateEnabled: input.CertificateEnabled,
		Adaptive:           input.Adaptive,
		CreatedBy:          input.CreatedBy,
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, fmt.Errorf("exams: CreateExam: %w", err)
	}
	return e, nil
}

func (s *service) GetExam(ctx context.Context, id string) (*ExamDetail, error) {
	detail, err := s.repo.GetWithDetails(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("exams: GetExam: %w", err)
	}
	return detail, nil
}

func (s *service) ListExams(ctx context.Context, filter ExamFilter) ([]*ExamListItem, int, error) {
	items, total, err := s.repo.ListFiltered(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("exams: ListExams: %w", err)
	}
	return items, total, nil
}

func (s *service) UpdateExam(ctx context.Context, id string, input UpdateExamInput) (*Exam, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("exams: UpdateExam: %w", err)
	}
	if e.Status != "draft" {
		return nil, ErrNotDraft
	}
	// AC-1: adaptive can only be changed in draft status (already enforced by ErrNotDraft above).
	// If the exam is not draft and the caller tries to change adaptive, it is rejected.
	updated, err := s.repo.Update(ctx, id, input)
	if err != nil {
		return nil, fmt.Errorf("exams: UpdateExam: %w", err)
	}
	return updated, nil
}

func (s *service) TransitionStatus(ctx context.Context, id, newStatus string) (*Exam, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("exams: TransitionStatus: %w", err)
	}
	allowed, ok := validTransitions[e.Status]
	if !ok || allowed != newStatus {
		return nil, fmt.Errorf("%w: cannot transition from %q to %q", ErrInvalidTransition, e.Status, newStatus)
	}
	if err := s.repo.UpdateStatus(ctx, id, newStatus); err != nil {
		return nil, fmt.Errorf("exams: TransitionStatus: %w", err)
	}
	e.Status = newStatus
	return e, nil
}

// minAdaptiveQuestionsPerDifficulty is the minimum number of questions required at each
// difficulty level for each rule when publishing an adaptive exam (FR-BB72 AC-2).
const minAdaptiveQuestionsPerDifficulty = 5

func (s *service) Publish(ctx context.Context, id string) (*Exam, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("exams: Publish: %w", err)
	}
	if e.Status != "draft" {
		return nil, ErrNotDraft
	}

	rules, err := s.repo.ListRulesForExam(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("exams: Publish: list rules: %w", err)
	}

	// AC-2 (FR-BB72): if adaptive, validate ≥5 questions per difficulty per rule.
	if e.Adaptive {
		for _, rule := range rules {
			if rule.Mode == "manual" {
				continue
			}
			for _, diff := range []string{"easy", "medium", "hard"} {
				count, err := s.repo.CountQuestionsPerDifficulty(ctx, rule, diff)
				if err != nil {
					return nil, fmt.Errorf("exams: Publish: adaptive count rule %s diff %s: %w", rule.ID, diff, err)
				}
				if count < minAdaptiveQuestionsPerDifficulty {
					return nil, &AdaptivePublishValidationError{
						RuleID:     rule.ID,
						Difficulty: diff,
						Required:   minAdaptiveQuestionsPerDifficulty,
						Available:  count,
					}
				}
			}
		}
	}

	var unsatisfied []RuleUnsatisfiedDetail
	for _, rule := range rules {
		var available int
		if rule.Mode == "manual" {
			available, err = s.repo.CountAvailableForManualRule(ctx, rule.ID)
			if err != nil {
				return nil, fmt.Errorf("exams: Publish: count manual rule %s: %w", rule.ID, err)
			}
			if available < rule.Count {
				unsatisfied = append(unsatisfied, RuleUnsatisfiedDetail{
					RuleID:    rule.ID,
					Required:  rule.Count,
					Available: available,
					Filter:    map[string]any{"mode": "manual"},
				})
			}
			continue
		}
		available, err = s.repo.CountAvailableForRule(ctx, rule)
		if err != nil {
			return nil, fmt.Errorf("exams: Publish: count for rule %s: %w", rule.ID, err)
		}
		if available < rule.Count {
			filter := map[string]any{}
			if rule.CategoryID != nil {
				filter["category_id"] = *rule.CategoryID
			}
			if rule.Difficulty != nil {
				filter["difficulty"] = *rule.Difficulty
			}
			var tagIDs []string
			if len(rule.TagIDs) > 0 {
				_ = json.Unmarshal(rule.TagIDs, &tagIDs)
			}
			if len(tagIDs) > 0 {
				filter["tag_ids"] = tagIDs
			}
			unsatisfied = append(unsatisfied, RuleUnsatisfiedDetail{
				RuleID:    rule.ID,
				Required:  rule.Count,
				Available: available,
				Filter:    filter,
			})
		}
	}
	if len(unsatisfied) > 0 {
		return nil, &PublishValidationError{Details: unsatisfied}
	}

	if err := s.repo.UpdateStatus(ctx, id, "active"); err != nil {
		return nil, fmt.Errorf("exams: Publish: %w", err)
	}
	e.Status = "active"
	return e, nil
}

func (s *service) Archive(ctx context.Context, id string) (*Exam, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("exams: Archive: %w", err)
	}
	if e.Status == "archived" {
		return nil, fmt.Errorf("%w: already archived", ErrInvalidTransition)
	}
	if err := s.repo.UpdateStatus(ctx, id, "archived"); err != nil {
		return nil, fmt.Errorf("exams: Archive: %w", err)
	}
	e.Status = "archived"
	return e, nil
}

func (s *service) UnpublishExam(ctx context.Context, id string) (*Exam, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("exams: UnpublishExam: %w", err)
	}
	if e.Status != "active" {
		return nil, fmt.Errorf("%w: exam must be active to unpublish", ErrNotActive)
	}
	count, err := s.repo.CountActiveSessionsForExam(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("exams: UnpublishExam: %w", err)
	}
	if count > 0 {
		return nil, ErrActiveSessionsExist
	}
	if err := s.repo.UpdateStatus(ctx, id, "draft"); err != nil {
		return nil, fmt.Errorf("exams: UnpublishExam: %w", err)
	}
	e.Status = "draft"
	return e, nil
}

func (s *service) DeleteExam(ctx context.Context, id string) error {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("exams: DeleteExam: %w", err)
	}
	if e.Status != "draft" {
		return ErrNotDraft
	}
	if err := s.repo.DeleteByID(ctx, id); err != nil {
		return fmt.Errorf("exams: DeleteExam: %w", err)
	}
	return nil
}

func (s *service) CreateSection(ctx context.Context, examID string, input SectionInput) (*ExamSection, error) {
	if _, err := s.repo.GetByID(ctx, examID); err != nil {
		return nil, fmt.Errorf("exams: CreateSection: %w", err)
	}
	section, err := s.repo.CreateSection(ctx, examID, input)
	if err != nil {
		return nil, fmt.Errorf("exams: CreateSection: %w", err)
	}
	return section, nil
}

func (s *service) UpdateSection(ctx context.Context, examID, sectionID string, input SectionInput) (*ExamSection, error) {
	if _, err := s.repo.GetByID(ctx, examID); err != nil {
		return nil, fmt.Errorf("exams: UpdateSection: %w", err)
	}
	// AC-7: repo enforces exam_id = examID so sections from other exams return ErrNotFound.
	section, err := s.repo.UpdateSection(ctx, examID, sectionID, input)
	if err != nil {
		return nil, fmt.Errorf("exams: UpdateSection: %w", err)
	}
	return section, nil
}

func (s *service) DeleteSection(ctx context.Context, examID, sectionID string) error {
	if _, err := s.repo.GetByID(ctx, examID); err != nil {
		return fmt.Errorf("exams: DeleteSection: %w", err)
	}
	// AC-7: repo enforces exam_id = examID so sections from other exams return ErrNotFound.
	if err := s.repo.DeleteSection(ctx, examID, sectionID); err != nil {
		return fmt.Errorf("exams: DeleteSection: %w", err)
	}
	return nil
}

func (s *service) CreateRule(ctx context.Context, examID string, input QuestionRuleInput) (*ExamQuestionRule, error) {
	if _, err := s.repo.GetByID(ctx, examID); err != nil {
		return nil, fmt.Errorf("exams: CreateRule: %w", err)
	}
	// Validate that every tag_id is a valid UUID string.
	for _, tagID := range input.TagIDs {
		if !isValidUUID(tagID) {
			return nil, fmt.Errorf("%w: tag_id %q is not a valid UUID", ErrInvalidInput, tagID)
		}
	}
	rule, err := s.repo.CreateRule(ctx, examID, input)
	if err != nil {
		return nil, fmt.Errorf("exams: CreateRule: %w", err)
	}
	// Persist manual questions if provided alongside the rule.
	if input.Mode == "manual" && len(input.Questions) > 0 {
		if err := s.repo.SetManualQuestions(ctx, rule.ID, input.Questions); err != nil {
			return nil, fmt.Errorf("exams: CreateRule: set manual questions: %w", err)
		}
	}
	return rule, nil
}

func (s *service) UpdateRule(ctx context.Context, examID, ruleID string, input QuestionRuleInput) (*ExamQuestionRule, error) {
	if _, err := s.repo.GetByID(ctx, examID); err != nil {
		return nil, fmt.Errorf("exams: UpdateRule: %w", err)
	}
	// Validate that every tag_id is a valid UUID string.
	for _, tagID := range input.TagIDs {
		if !isValidUUID(tagID) {
			return nil, fmt.Errorf("%w: tag_id %q is not a valid UUID", ErrInvalidInput, tagID)
		}
	}
	rule, err := s.repo.UpdateRule(ctx, ruleID, input)
	if err != nil {
		return nil, fmt.Errorf("exams: UpdateRule: %w", err)
	}
	return rule, nil
}

func (s *service) DeleteRule(ctx context.Context, examID, ruleID string) error {
	if _, err := s.repo.GetByID(ctx, examID); err != nil {
		return fmt.Errorf("exams: DeleteRule: %w", err)
	}
	if err := s.repo.DeleteRule(ctx, ruleID); err != nil {
		return fmt.Errorf("exams: DeleteRule: %w", err)
	}
	return nil
}

func (s *service) SetManualQuestions(ctx context.Context, ruleID string, questions []ManualQuestionInput) error {
	rule, err := s.repo.GetRuleByID(ctx, ruleID)
	if err != nil {
		return fmt.Errorf("exams: SetManualQuestions: %w", err)
	}
	if rule.Mode != "manual" {
		return ErrRulesModeConflict
	}
	if err := s.repo.SetManualQuestions(ctx, ruleID, questions); err != nil {
		return fmt.Errorf("exams: SetManualQuestions: %w", err)
	}
	return nil
}

// ── Assignment methods (FR-BB33) ─────────────────────────────────────────────

func (s *service) CreateAssignment(ctx context.Context, input CreateAssignmentInput) (*ExamAssignment, error) {
	exam, err := s.repo.GetByID(ctx, input.ExamID)
	if err != nil {
		return nil, fmt.Errorf("exams: CreateAssignment: %w", err)
	}
	if exam.Status != "active" && exam.Status != "draft" {
		return nil, ErrNotActive
	}

	// AC-6: department admins may only assign within their own department.
	if input.CallerRole == "department_admin" {
		if input.AssigneeType == "all" {
			return nil, ErrForbidden
		}
		if input.AssigneeType == "department" && (input.AssigneeID == nil || *input.AssigneeID != input.CallerDeptID) {
			return nil, ErrForbidden
		}
	}

	// AC-8: deadline must be in the future if provided.
	if input.Deadline != nil && !input.Deadline.After(now()) {
		return nil, ErrDeadlineInPast
	}

	a := &ExamAssignment{
		ExamID:       input.ExamID,
		AssigneeType: input.AssigneeType,
		AssigneeID:   input.AssigneeID,
		Deadline:     input.Deadline,
		AssignedBy:   input.AssignedBy,
	}
	if err := s.repo.CreateAssignment(ctx, a); err != nil {
		return nil, fmt.Errorf("exams: CreateAssignment: %w", err)
	}

	if s.emailSvc != nil {
		s.emailSvc.TriggerAssignment(input.ExamID, exam.Title, input.Deadline, input.AssigneeType, input.AssigneeID)
	}

	return a, nil
}

func (s *service) DeleteAssignment(ctx context.Context, examID, assignmentID string, callerRole, callerDeptID string) error {
	if _, err := s.repo.GetByID(ctx, examID); err != nil {
		return fmt.Errorf("exams: DeleteAssignment: %w", err)
	}
	a, err := s.repo.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		return fmt.Errorf("exams: DeleteAssignment: %w", err)
	}
	// Verify the assignment belongs to this exam.
	if a.ExamID != examID {
		return ErrAssignmentNotFound
	}
	// AC-6: department admins may only remove their own dept assignments.
	if callerRole == "department_admin" {
		if a.AssigneeType == "all" {
			return ErrForbidden
		}
		if a.AssigneeType == "department" && (a.AssigneeID == nil || *a.AssigneeID != callerDeptID) {
			return ErrForbidden
		}
	}
	if err := s.repo.DeleteAssignment(ctx, assignmentID); err != nil {
		return fmt.Errorf("exams: DeleteAssignment: %w", err)
	}
	return nil
}

func (s *service) ListAssignments(ctx context.Context, examID string) ([]*AssignmentDetail, error) {
	if _, err := s.repo.GetByID(ctx, examID); err != nil {
		return nil, fmt.Errorf("exams: ListAssignments: %w", err)
	}
	details, err := s.repo.ListAssignmentsWithStats(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("exams: ListAssignments: %w", err)
	}
	return details, nil
}

// now is a variable so tests can override it.
var now = func() time.Time { return time.Now().UTC() }

// GetEligibleCounts returns the number of eligible (active, matching) questions
// for each rule defined on the given exam (FR-BB315).
func (s *service) GetEligibleCounts(ctx context.Context, examID string) ([]RuleEligibleCount, error) {
	// Verify exam exists.
	if _, err := s.repo.GetByID(ctx, examID); err != nil {
		return nil, fmt.Errorf("exams: GetEligibleCounts: %w", err)
	}

	rules, err := s.repo.ListRulesForExam(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("exams: GetEligibleCounts: list rules: %w", err)
	}

	counts := make([]RuleEligibleCount, 0, len(rules))
	for _, rule := range rules {
		var eligible int
		switch rule.Mode {
		case "manual":
			eligible, err = s.repo.CountAvailableForManualRule(ctx, rule.ID)
		default:
			eligible, err = s.repo.CountAvailableForRule(ctx, rule)
		}
		if err != nil {
			return nil, fmt.Errorf("exams: GetEligibleCounts: rule %s: %w", rule.ID, err)
		}
		counts = append(counts, RuleEligibleCount{RuleID: rule.ID, Eligible: eligible})
	}
	return counts, nil
}
