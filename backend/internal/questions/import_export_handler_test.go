package questions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/bilimbaga/bilimbaga/internal/rbac"
	"github.com/golang-jwt/jwt/v5"
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

// ── Tests with exact names required by spec ───────────────────────────────────

// TestImportHandler_DryRun_200 is the spec-required alias for TestImportHandler_DryRun_Returns200.
func TestImportHandler_DryRun_200(t *testing.T) {
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
}

// TestImportHandler_Commit_201 is the spec-required alias for TestImportHandler_Commit_Returns201.
func TestImportHandler_Commit_201(t *testing.T) {
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

// TestImportHandler_BatchTooLarge_413 is the spec-required alias for TestImportHandler_TooManyRows_Returns413.
func TestImportHandler_BatchTooLarge_413(t *testing.T) {
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
	var env struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	require.NotNil(t, env.Error)
	assert.Equal(t, "ERR_BATCH_TOO_LARGE", env.Error.Code)
}

// TestImportHandler_ValidationFailure_422 verifies that commit mode with invalid
// rows returns 422 ERR_IMPORT_VALIDATION with error details.
func TestImportHandler_ValidationFailure_422(t *testing.T) {
	svc := &mockQService{
		validateAndImportFn: func(_ context.Context, _ []ImportRow, _ bool, _ string) (*DryRunReport, *CommitResult, error) {
			return nil, nil, &importValidationError{
				validCount:  0,
				errorRows:   []ImportRowError{{Row: 1, Errors: []string{"type: invalid value 'checkbox'"}}},
				warningRows: []ImportRowWarning{},
			}
		},
	}
	h := NewHandler(svc, nil)

	body, ct := buildQCSVMultipart(t, validCSV, "questions.csv")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import", body)
	req.Header.Set("Content-Type", ct)
	req = withQImportAuthCtx(req)
	w := httptest.NewRecorder()
	h.Import(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var env struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	require.NotNil(t, env.Error)
	assert.Equal(t, "ERR_IMPORT_VALIDATION", env.Error.Code)
}

// ── makeImportJWT is a helper that signs a JWT with the given role ──────────

const importTestSecret = "test-import-jwt-secret"

func makeImportJWT(t *testing.T, role string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":  "user-1",
		"role": role,
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(importTestSecret))
	require.NoError(t, err)
	return signed
}

// TestImportHandler_Unauthorized_401 verifies that requests without a JWT are
// rejected with 401. The auth middleware runs before the handler in the real
// router; we simulate that chain here.
func TestImportHandler_Unauthorized_401(t *testing.T) {
	cache := rbac.NewCache()
	cache.LoadFromMap(map[string]rbac.PermissionSet{
		"examiner": {"questions:write": true},
	})

	h := NewHandler(&mockQService{}, nil)
	// Wrap handler with auth.Authenticate + rbac.RequirePermission to test the middleware stack.
	chain := auth.Authenticate(importTestSecret)(rbac.RequirePermission(cache, "questions", "write")(http.HandlerFunc(h.Import)))

	body, ct := buildQCSVMultipart(t, validCSV, "questions.csv")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import", body)
	req.Header.Set("Content-Type", ct)
	// No Authorization header.
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestImportHandler_Forbidden_403 verifies that a role without questions:write
// permission receives 403.
func TestImportHandler_Forbidden_403(t *testing.T) {
	cache := rbac.NewCache()
	cache.LoadFromMap(map[string]rbac.PermissionSet{
		"examiner": {"questions:write": true},
		"employee": {},
	})

	h := NewHandler(&mockQService{}, nil)
	chain := auth.Authenticate(importTestSecret)(rbac.RequirePermission(cache, "questions", "write")(http.HandlerFunc(h.Import)))

	body, ct := buildQCSVMultipart(t, validCSV, "questions.csv")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+makeImportJWT(t, "employee"))
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestExportHandler_CSV_200 is the spec-required alias for TestExportHandler_CSV_Returns200.
func TestExportHandler_CSV_200(t *testing.T) {
	svc := &mockQService{
		streamExportFn: func(_ context.Context, _ ExportFilter, fn func(*ExportRow) error) error {
			return fn(&ExportRow{
				Type: "single", Difficulty: "easy", CategoryPath: "Science",
				DefaultLocale: "kk",
				Translations:  map[string]TranslationDetail{"kk": {Stem: "Q?"}},
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
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
}

// TestExportHandler_JSON_200 is the spec-required alias for TestExportHandler_JSON_Returns200.
func TestExportHandler_JSON_200(t *testing.T) {
	svc := &mockQService{
		streamExportFn: func(_ context.Context, _ ExportFilter, fn func(*ExportRow) error) error {
			return fn(&ExportRow{
				Type: "single", Difficulty: "easy", CategoryPath: "Science",
				DefaultLocale: "kk",
				Translations:  map[string]TranslationDetail{"kk": {Stem: "Q?"}},
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
	// Verify valid JSON array.
	var rows []json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rows))
	assert.Len(t, rows, 1)
}

// ── Compile-time check: errors package must be used ──────────────────────────
var _ = errors.New

// ── AC-3 (FR-BB64): upload content validation ────────────────────────────────

func TestImportHandler_OversizeFile_Returns413(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)

	big := validCSV + strings.Repeat("a", 10<<20+1)
	body, ct := buildQCSVMultipart(t, big, "questions.csv")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import", body)
	req.Header.Set("Content-Type", ct)
	req = withQImportAuthCtx(req)
	w := httptest.NewRecorder()
	h.Import(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	_, apiErr := decodeQEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "ERR_FILE_TOO_LARGE", apiErr.Code)
}

func TestImportHandler_BinaryContent_Returns415(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)

	// PNG magic bytes disguised as a .csv upload.
	png := string([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52})
	body, ct := buildQCSVMultipart(t, png, "questions.csv")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import", body)
	req.Header.Set("Content-Type", ct)
	req = withQImportAuthCtx(req)
	w := httptest.NewRecorder()
	h.Import(w, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	_, apiErr := decodeQEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "ERR_INVALID_FILE_TYPE", apiErr.Code)
}

// ISS-141: export id filters must be UUIDs; malformed -> 422 before streaming.
func TestExportHandler_MalformedUUIDFilters_Return422(t *testing.T) {
	const ok = "3f2b8c1e-9d4a-4b6e-8a1f-0c7d5e9a1b22"
	for _, q := range []string{"ids=" + ok + ",bad", "category_id=cat", "tag_ids=bad"} {
		h := NewHandler(&mockQService{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/export?"+q, nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		h.Export(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code, q)
		assert.Contains(t, w.Body.String(), "VALIDATION_ERROR", q)
	}
}

func TestImportHandler_BodyOverCap_Returns413BeforeParse(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)

	big := strings.Repeat("a", 13<<20)
	body, ct := buildQCSVMultipart(t, big, "questions.csv")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import", body)
	req.Header.Set("Content-Type", ct)
	req = withQImportAuthCtx(req)
	w := httptest.NewRecorder()
	h.Import(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	_, apiErr := decodeQEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "ERR_FILE_TOO_LARGE", apiErr.Code)
}

// ISS-176 round 2: over-cap body over a real HTTP server -> 413 JSON envelope, clean connection;
// non-multipart body stays 400.
func TestImportHandler_OverCapOverHTTP_Returns413(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.Import(w, withQImportAuthCtx(r))
	}))
	defer srv.Close()

	for _, mb := range []int{11, 12, 30} {
		body, ct := buildQCSVMultipart(t, validCSV+strings.Repeat("a", mb<<20), "questions.csv")
		resp, err := http.Post(srv.URL, ct, body)
		require.NoError(t, err, "%d MiB", mb)
		var env struct {
			Data  any `json:"data"`
			Error *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
		resp.Body.Close()
		assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode, "%d MiB", mb)
		require.NotNil(t, env.Error)
		assert.Equal(t, "ERR_FILE_TOO_LARGE", env.Error.Code)
		assert.Nil(t, env.Data)
	}
}

func TestImportHandler_NonMultipartBody_Returns400(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import", strings.NewReader(`{"a":1}`))
	req.Header.Set("Content-Type", "application/json")
	req = withQImportAuthCtx(req)
	w := httptest.NewRecorder()
	h.Import(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	_, apiErr := decodeQEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "ERR_INVALID_BODY", apiErr.Code)
}
