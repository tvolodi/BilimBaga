package exams

// #566: real-Postgres test of the mode-only update. The service, the repository and the audit writer run against the
// production migrations, so the row, the audit metadata and the refusal are the production ones. Skipped unless
// TEST_DATABASE_URL is set.
//
//	cd backend && TEST_DATABASE_URL='postgres://USER:PASS@HOST:PORT/DB?sslmode=disable' \
//	    go test ./internal/exams -run UpdateAIGradingMode_RealPostgres -v

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	dbpkg "github.com/bilimbaga/bilimbaga/internal/db"
	"github.com/bilimbaga/bilimbaga/internal/testpg"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedPublishedExam inserts an exam in the given status with the database defaults, and returns its id and its
// creator's id.
func seedPublishedExam(t *testing.T, db *sqlx.DB, status string) (examID, userID string) {
	t.Helper()
	var roleID string
	require.NoError(t, db.Get(&roleID, `INSERT INTO roles (name) VALUES ('exam-admin') ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id::text`))
	require.NoError(t, db.Get(&userID, `INSERT INTO users (email, password_hash, full_name, role_id) VALUES ('mode566@test.local', 'x', 'Admin', $1) RETURNING id::text`, roleID))
	require.NoError(t, db.Get(&examID, `
INSERT INTO exams (title, time_limit_minutes, passing_score_pct, created_by, status)
VALUES ('Fire safety', 30, 70, $1, $2::exam_status) RETURNING id::text`, userID, status))
	return examID, userID
}

// putExam sends a PUT /exams/{id} through the handler, as the authenticated user, and returns the response.
func putExam(t *testing.T, h *Handler, examID, userID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/exams/"+examID, strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", examID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, ctxkeys.CtxTenantID, "public")
	ctx = context.WithValue(ctx, ctxkeys.CtxUserID, userID)
	w := httptest.NewRecorder()
	h.Update(w, req.WithContext(ctx))
	return w
}

// modeBody is the body of a request that repeats the seeded exam (database defaults) and sets the mode.
func modeBody(title, mode string) string {
	return fmt.Sprintf(`{"title":%q,"time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1,"show_answers":"never","on_tab_switch":"log","ai_grading_mode":%q}`, title, mode)
}

func newExamsPGHandler(t *testing.T) (*Handler, *sqlx.DB) {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set: skipping the real-Postgres exam mode update test")
	}
	db := testpg.NewSchema(t, base, "bb_566")
	require.NoError(t, dbpkg.RunMigrations(db, "../../migrations"))
	svc := NewService(NewRepository(db))
	return NewHandler(svc, audit.NewWriter(db, slog.Default())), db
}

func storedMode(t *testing.T, db *sqlx.DB, examID string) (title, mode string) {
	t.Helper()
	require.NoError(t, db.QueryRow(`SELECT title, ai_grading_mode FROM exams WHERE id = $1`, examID).Scan(&title, &mode))
	return title, mode
}

func TestUpdateAIGradingMode_RealPostgres_PublishedModeChangeIsAuditedFromAndTo(t *testing.T) {
	h, db := newExamsPGHandler(t)
	examID, userID := seedPublishedExam(t, db, "active")

	w := putExam(t, h, examID, userID, modeBody("Fire safety", "off"))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	title, mode := storedMode(t, db, examID)
	assert.Equal(t, "off", mode)
	assert.Equal(t, "Fire safety", title, "only the mode changed")

	var from, to, auditTitle string
	require.NoError(t, db.QueryRow(`
SELECT metadata->>'ai_grading_mode_from', metadata->>'ai_grading_mode_to', metadata->>'title'
FROM audit_log WHERE action = 'exam.update' AND entity_id = $1`, examID).Scan(&from, &to, &auditTitle))
	assert.Equal(t, "suggest", from, "the seeded exam had the database default")
	assert.Equal(t, "off", to)
	assert.Equal(t, "Fire safety", auditTitle)
}

func TestUpdateAIGradingMode_RealPostgres_PublishedTitleChangeIsRefusedAndWritesNothing(t *testing.T) {
	h, db := newExamsPGHandler(t)
	examID, userID := seedPublishedExam(t, db, "active")

	w := putExam(t, h, examID, userID, modeBody("Renamed", "off"))

	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "EXAM_NOT_DRAFT")
	title, mode := storedMode(t, db, examID)
	assert.Equal(t, "Fire safety", title)
	assert.Equal(t, "suggest", mode, "the mixed request changed neither the title nor the mode")
	var audits int
	require.NoError(t, db.Get(&audits, `SELECT COUNT(*) FROM audit_log WHERE action = 'exam.update' AND entity_id = $1`, examID))
	assert.Zero(t, audits, "a refused update writes no audit row")
}

func TestUpdateAIGradingMode_RealPostgres_ArchivedModeChangeIsAccepted(t *testing.T) {
	h, db := newExamsPGHandler(t)
	examID, userID := seedPublishedExam(t, db, "archived")

	w := putExam(t, h, examID, userID, modeBody("Fire safety", "auto"))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	_, mode := storedMode(t, db, examID)
	assert.Equal(t, "auto", mode)
}

// descBody repeats the seeded exam with a description, the way the wizard sends it.
func descBody(description, mode string) string {
	return fmt.Sprintf(`{"title":"Fire safety","description":%q,"time_limit_minutes":30,"passing_score_pct":70,"max_attempts":1,"show_answers":"never","on_tab_switch":"log","ai_grading_mode":%q}`, description, mode)
}

// #570: a request that repeats a published exam changes nothing and writes no audit row.
func TestUpdateAIGradingMode_RealPostgres_NoChangeWritesNoAuditRow(t *testing.T) {
	h, db := newExamsPGHandler(t)
	examID, userID := seedPublishedExam(t, db, "active")

	w := putExam(t, h, examID, userID, modeBody("Fire safety", "suggest"))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var audits int
	require.NoError(t, db.Get(&audits, `SELECT COUNT(*) FROM audit_log WHERE action = 'exam.update' AND entity_id = $1`, examID))
	assert.Zero(t, audits, "a request that changes nothing on a published exam writes no audit row")
}

// #570: a changed description sent with a mode change is refused, and nothing is written; the mode is not applied.
func TestUpdateAIGradingMode_RealPostgres_ChangedDescriptionIsRefusedAndWritesNothing(t *testing.T) {
	h, db := newExamsPGHandler(t)
	examID, userID := seedPublishedExam(t, db, "active")
	_, err := db.Exec(`UPDATE exams SET description = 'Old text' WHERE id = $1`, examID)
	require.NoError(t, err)

	w := putExam(t, h, examID, userID, descBody("New text", "off"))

	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "EXAM_NOT_DRAFT")
	var description string
	require.NoError(t, db.Get(&description, `SELECT description FROM exams WHERE id = $1`, examID))
	assert.Equal(t, "Old text", description)
	_, mode := storedMode(t, db, examID)
	assert.Equal(t, "suggest", mode, "the refused request does not change the mode either")
	var audits int
	require.NoError(t, db.Get(&audits, `SELECT COUNT(*) FROM audit_log WHERE action = 'exam.update' AND entity_id = $1`, examID))
	assert.Zero(t, audits)
}

// #570: the same description with a mode change is not a change to the description, so the mode change goes through.
func TestUpdateAIGradingMode_RealPostgres_SameDescriptionWithModeChangeIsAccepted(t *testing.T) {
	h, db := newExamsPGHandler(t)
	examID, userID := seedPublishedExam(t, db, "active")
	_, err := db.Exec(`UPDATE exams SET description = 'Old text' WHERE id = $1`, examID)
	require.NoError(t, err)

	w := putExam(t, h, examID, userID, descBody("Old text", "off"))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	_, mode := storedMode(t, db, examID)
	assert.Equal(t, "off", mode)
}

// staleReadRepo reads the exam the way the service does, and then sets the mode the way a concurrent request would, before
// the service writes (#570 review). The service's read is therefore stale by the time it writes.
type staleReadRepo struct {
	Repository
	db    *sqlx.DB
	raced string
}

func (r *staleReadRepo) GetByID(ctx context.Context, id string) (*Exam, error) {
	e, err := r.Repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE exams SET ai_grading_mode = $2 WHERE id = $1`, id, r.raced); err != nil {
		return nil, err
	}
	return e, nil
}

// #570 review: the service read auto, a concurrent request set suggest, and this request asks for auto. The audit row must
// say suggest to auto: the exam changed, so the row is written, and its from is the mode the write replaced.
func TestUpdateAIGradingMode_RealPostgres_RaceAfterReadWritesTheTrueFromAndTo(t *testing.T) {
	h, db := newExamsPGHandler(t)
	examID, userID := seedPublishedExam(t, db, "active")
	_, err := db.Exec(`UPDATE exams SET ai_grading_mode = 'auto' WHERE id = $1`, examID)
	require.NoError(t, err)
	racing := NewHandler(NewService(&staleReadRepo{Repository: NewRepository(db), db: db, raced: "suggest"}), h.writer)

	w := putExam(t, racing, examID, userID, modeBody("Fire safety", "auto"))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	_, mode := storedMode(t, db, examID)
	assert.Equal(t, "auto", mode)
	var audits int
	require.NoError(t, db.Get(&audits, `SELECT COUNT(*) FROM audit_log WHERE action = 'exam.update' AND entity_id = $1`, examID))
	require.Equal(t, 1, audits, "the change is audited exactly once, although the service's read was stale")
	var from, to string
	require.NoError(t, db.QueryRow(`
SELECT metadata->>'ai_grading_mode_from', metadata->>'ai_grading_mode_to'
FROM audit_log WHERE action = 'exam.update' AND entity_id = $1`, examID).Scan(&from, &to))
	assert.Equal(t, "suggest", from, "from is the mode the write replaced, not the one the service read")
	assert.Equal(t, "auto", to)
}

// #570 review: the write takes the row lock before it reads the previous mode. A write that finds the row held waits for
// the holder, and reports the mode the holder committed.
func TestUpdateAIGradingMode_RealPostgres_WaitsForTheLockAndReportsTheCommittedMode(t *testing.T) {
	_, db := newExamsPGHandler(t)
	examID, _ := seedPublishedExam(t, db, "active")
	repo := NewRepository(db)

	holder, err := db.Beginx()
	require.NoError(t, err)
	defer holder.Rollback() //nolint:errcheck
	var held string
	require.NoError(t, holder.Get(&held, `SELECT ai_grading_mode FROM exams WHERE id = $1 FOR UPDATE`, examID))

	type outcome struct {
		updated  *Exam
		previous string
		err      error
	}
	done := make(chan outcome, 1)
	go func() {
		updated, previous, err := repo.UpdateAIGradingMode(context.Background(), examID, "auto")
		done <- outcome{updated, previous, err}
	}()
	select {
	case <-done:
		t.Fatal("the write must wait while another transaction holds the row")
	case <-time.After(300 * time.Millisecond):
	}

	_, err = holder.Exec(`UPDATE exams SET ai_grading_mode = 'off' WHERE id = $1`, examID)
	require.NoError(t, err)
	require.NoError(t, holder.Commit())

	select {
	case res := <-done:
		require.NoError(t, res.err)
		assert.Equal(t, "off", res.previous, "the previous mode is the one the holder committed, not the one before it")
		assert.Equal(t, "auto", res.updated.AIGradingMode)
	case <-time.After(10 * time.Second):
		t.Fatal("the write did not finish after the holder committed")
	}
}
