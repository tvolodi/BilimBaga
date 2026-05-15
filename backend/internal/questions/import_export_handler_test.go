package questions

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func buildQCSVMultipart(t *testing.T, content, filename string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	require.NoError(t, err)
	fw.Write([]byte(content))
	mw.Close()
	return &buf, mw.FormDataContentType()
}

func withQImportAuthCtx(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), ctxkeys.CtxUserID, "actor-1")
	return r.WithContext(ctx)
}

// ── Import (CSV) ──────────────────────────────────────────────────────────────

const validCSV = `type,difficulty,category_path,default_locale,stem,option_1,option_2,correct,tags
single,easy,Science,kk,What is H2O?,Water,Fire,1,chem
`

func TestImportHandler_DryRun_Returns200(t *testing.T) {
	svc := &mockQService{
		validateAndImportFn: func(_ context.Context, _ []ImportRow, dryRun bool, _ string) (*DryRunReport, *CommitResult, error) {
			return &DryRunReport{DryRun: true, ValidCount: 1, ErrorRows: []ImportRowError{}, WarningRows: []ImportRowWarning{}}, nil, nil
		},
	}
	h := NewHandler(svc, nil)

	body, ct := buildQCSVMultipart(t, validCSV, "questions.csv")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import?dry_run=true", body)
	req.Header.Set("Content-Type", ct)
	req = withQImportAuthCtx(req)
	w := httptest.NewRecorder()
	h.Import(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data, apiErr := decodeQEnvelope(t, w)
	assert.Nil(t, apiErr)
	assert.NotEmpty(t, data)
}

func TestImportHandler_Commit_Returns201(t *testing.T) {
	svc := &mockQService{
		validateAndImportFn: func(_ context.Context, _ []ImportRow, _ bool, _ string) (*DryRunReport, *CommitResult, error) {
			return nil, &CommitResult{ImportedCount: 1, QuestionIDs: []string{"q-1"}}, nil
		},
	}
	h := NewHandler(svc, nil)

	body, ct := buildQCSVMultipart(t, validCSV, "questions.csv")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import", body)
	req.Header.Set("Content-Type", ct)
	req = withQImportAuthCtx(req)
	w := httptest.NewRecorder()
	h.Import(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestImportHandler_MissingFile_Returns400(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req = withQImportAuthCtx(req)
	w := httptest.NewRecorder()
	h.Import(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_, apiErr := decodeQEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "ERR_INVALID_BODY", apiErr.Code)
}

func TestImportHandler_TooManyRows_Returns413(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)

	var sb strings.Builder
	sb.WriteString("type,difficulty,category_path,default_locale,stem,option_1,option_2,correct,tags\n")
	for i := 0; i < 501; i++ {
		sb.WriteString("single,easy,Science,kk,What?,A,B,1,\n")
	}
	body, ct := buildQCSVMultipart(t, sb.String(), "questions.csv")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import", body)
	req.Header.Set("Content-Type", ct)
	req = withQImportAuthCtx(req)
	w := httptest.NewRecorder()
	h.Import(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestImportHandler_InvalidCSVBody_Returns400(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)

	// Send a file with an invalid CSV header (missing required columns).
	body, ct := buildQCSVMultipart(t, "garbage,data\nwithout,proper,header\n", "bad.csv")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import", body)
	req.Header.Set("Content-Type", ct)
	req = withQImportAuthCtx(req)
	w := httptest.NewRecorder()
	h.Import(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── Export (JSON) ─────────────────────────────────────────────────────────────

func TestExportHandler_JSON_Returns200(t *testing.T) {
	svc := &mockQService{
		streamExportFn: func(_ context.Context, _ ExportFilter, fn func(*ExportRow) error) error {
			return fn(&ExportRow{
				Type:          "single",
				Difficulty:    "easy",
				CategoryPath:  "Science",
				DefaultLocale: "kk",
				Translations: map[string]TranslationDetail{
					"kk": {Stem: "What is H2O?"},
				},
				AnswerOptions: []AnswerOptionDetail{},
				Tags:          []string{},
			})
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/export", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.Export(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
}

func TestExportHandler_CSV_Returns200(t *testing.T) {
	svc := &mockQService{
		streamExportFn: func(_ context.Context, _ ExportFilter, fn func(*ExportRow) error) error {
			return fn(&ExportRow{
				Type:          "single",
				Difficulty:    "easy",
				CategoryPath:  "Science",
				DefaultLocale: "kk",
				Translations: map[string]TranslationDetail{
					"kk": {Stem: "What is H2O?"},
				},
				AnswerOptions: []AnswerOptionDetail{},
				Tags:          []string{},
			})
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/export", nil)
	req.Header.Set("Accept", "text/csv")
	w := httptest.NewRecorder()
	h.Export(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
}

func TestExportHandler_TooManyIDs_Returns400(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)

	ids := make([]string, 101)
	for i := range ids {
		ids[i] = "q-" + string(rune('0'+i%10))
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/export?ids="+strings.Join(ids, ","), nil)
	w := httptest.NewRecorder()
	h.Export(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
