package upload

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// countingReader records how many bytes were consumed.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

func multipartBody(t *testing.T, size int) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "x.csv")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte(strings.Repeat("a", size)))
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}

func TestParseImportMultipart_OverCapRejectedWithoutReadingAll(t *testing.T) {
	body, ct := multipartBody(t, 20<<20)
	total := int64(body.Len())
	cr := &countingReader{r: body}
	req := httptest.NewRequest(http.MethodPost, "/", cr)
	req.Header.Set("Content-Type", ct)
	err := ParseImportMultipart(httptest.NewRecorder(), req)
	if err != ErrFileTooLarge {
		t.Fatalf("err = %v, want ErrFileTooLarge", err)
	}
	if cr.n > MaxImportBodyBytes+(64<<10) || cr.n >= total {
		t.Fatalf("read %d of %d bytes; cap %d not enforced", cr.n, total, MaxImportBodyBytes)
	}
}

func TestParseImportMultipart_WithinCapOK(t *testing.T) {
	body, ct := multipartBody(t, 1<<20)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", ct)
	if err := ParseImportMultipart(httptest.NewRecorder(), req); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, _, err := req.FormFile("file"); err != nil {
		t.Fatal(err)
	}
}

func TestParseImportMultipart_NotMultipart(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("x"))
	err := ParseImportMultipart(httptest.NewRecorder(), req)
	if err == nil || err == ErrFileTooLarge {
		t.Fatalf("err = %v, want non-size parse error", err)
	}
}
