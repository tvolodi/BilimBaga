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
