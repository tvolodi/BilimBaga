package router_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"database/sql"

	"github.com/bilimbaga/bilimbaga/internal/questions"
	"github.com/bilimbaga/bilimbaga/internal/rbac"
	"github.com/bilimbaga/bilimbaga/internal/router"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #176 (BA decision, PR #401): a size-cap failure on a CSV import is 413 with the size code,
// never 400. These tests send real multipart uploads through the full router (the users
// import route, the same harness as the authz suite).

const (
	mib           = 1 << 20
	csvContentCap = 10 * mib // the content cap (upload.MaxCSVBytes)
	importBodyCap = 11 * mib // the body cap (upload.MaxImportBodyBytes)
)

// multipartCSVUpload builds a multipart/form-data body whose "file" part is size bytes of
// plain text (magic-byte detection reports text/plain, so only the size decides).
func multipartCSVUpload(t *testing.T, size int) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("file", "import.csv")
	require.NoError(t, err)
	_, err = fw.Write(bytes.Repeat([]byte("a"), size))
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	return body, mw.FormDataContentType()
}

func postUpload(h http.Handler, path, tok string, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var env struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), rec.Body.String())
	require.NotNil(t, env.Error, rec.Body.String())
	return env.Error.Code
}

// File content one byte over the 10 MiB cap, inside the body cap: the content check rejects it.
func TestUsersImport_ContentOverCap_Returns413(t *testing.T) {
	h, _ := newUsersRouter(t)
	tok := supToken(t)
	body, ct := multipartCSVUpload(t, csvContentCap+1)

	rec := postUpload(h, "/api/v1/users/import", tok, body, ct)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
	assert.Equal(t, "FILE_TOO_LARGE", errorCode(t, rec))
}

// Whole body over the 11 MiB body cap: rejected before parsing, and must be 413, not 400.
func TestUsersImport_BodyOverCap_Returns413(t *testing.T) {
	h, _ := newUsersRouter(t)
	tok := supToken(t)
	body, ct := multipartCSVUpload(t, importBodyCap+mib)

	rec := postUpload(h, "/api/v1/users/import", tok, body, ct)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
	assert.Equal(t, "FILE_TOO_LARGE", errorCode(t, rec))
}

// Exactly 10 MiB is accepted (the boundary), so it must not be a size error.
func TestUsersImport_ExactlyAtCap_IsNotSizeRejected(t *testing.T) {
	h, _ := newUsersRouter(t)
	tok := supToken(t)
	body, ct := multipartCSVUpload(t, csvContentCap)

	rec := postUpload(h, "/api/v1/users/import", tok, body, ct)

	assert.NotEqual(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
}

// newQuestionsImportRouter builds the full router with a questions handler (no service: the
// oversize paths answer before any service call) and a super_admin cache that grants questions:write.
func newQuestionsImportRouter(t *testing.T) http.Handler {
	t.Helper()
	repo := newAuthzRepo()
	db := sqlx.NewDb(sql.OpenDB(stateConnector{repo}), "postgres")
	t.Cleanup(func() { _ = db.Close() })
	cache := rbac.NewCache()
	cache.LoadFromMap(map[string]rbac.PermissionSet{"super_admin": {"questions:write": true, "questions:read": true}})
	qh := questions.NewHandler(nil, nil)
	return router.New(nil, nil, nil, nil, nil, nil, nil, qh, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"test-secret", cache, db, "test", zerolog.Nop())
}

// The questions import answers 413 ERR_FILE_TOO_LARGE for an oversize body (BA rule, #176).
func TestQuestionsImport_BodyOverCap_Returns413(t *testing.T) {
	h := newQuestionsImportRouter(t)
	body, ct := multipartCSVUpload(t, importBodyCap+mib)

	rec := postUpload(h, "/api/v1/questions/import", supToken(t), body, ct)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
	assert.Equal(t, "ERR_FILE_TOO_LARGE", errorCode(t, rec))
}

// File content over the 10 MiB cap inside the body cap: 413 ERR_FILE_TOO_LARGE, never 400.
func TestQuestionsImport_ContentOverCap_Returns413(t *testing.T) {
	h := newQuestionsImportRouter(t)
	body, ct := multipartCSVUpload(t, csvContentCap+1)

	rec := postUpload(h, "/api/v1/questions/import", supToken(t), body, ct)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
	assert.Equal(t, "ERR_FILE_TOO_LARGE", errorCode(t, rec))
}

// rawCSVUpload is a non-multipart body (Content-Type text/csv) of size bytes of plain text.
// The import routes must still apply the size cap to it (#176, BA rule: size before the 400).
func rawCSVUpload(size int) *bytes.Buffer {
	return bytes.NewBuffer(bytes.Repeat([]byte("a"), size))
}

func postRaw(h http.Handler, path, tok string, body *bytes.Buffer) *httptest.ResponseRecorder {
	return postUpload(h, path, tok, body, "text/csv")
}

// A raw text/csv body over the body cap is 413 with the size code, not 400 (#176).
func TestUsersImport_RawCSVOverCap_Returns413(t *testing.T) {
	h, _ := newUsersRouter(t)
	rec := postRaw(h, "/api/v1/users/import", supToken(t), rawCSVUpload(importBodyCap+mib/2))

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
	assert.Equal(t, "FILE_TOO_LARGE", errorCode(t, rec))
}

func TestQuestionsImport_RawCSVOverCap_Returns413(t *testing.T) {
	h := newQuestionsImportRouter(t)
	rec := postRaw(h, "/api/v1/questions/import", supToken(t), rawCSVUpload(importBodyCap+mib))

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
	assert.Equal(t, "ERR_FILE_TOO_LARGE", errorCode(t, rec))
}

// A small raw body (under the cap) that is not multipart stays 400: the body shape is wrong, not its size.
func TestUsersImport_RawCSVUnderCap_Returns400(t *testing.T) {
	h, _ := newUsersRouter(t)
	rec := postRaw(h, "/api/v1/users/import", supToken(t), rawCSVUpload(1024))

	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Equal(t, "INVALID_BODY", errorCode(t, rec))
}

func TestQuestionsImport_RawCSVUnderCap_Returns400(t *testing.T) {
	h := newQuestionsImportRouter(t)
	rec := postRaw(h, "/api/v1/questions/import", supToken(t), rawCSVUpload(1024))

	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Equal(t, "ERR_INVALID_BODY", errorCode(t, rec))
}
