// Package upload provides server-side file validation for uploaded files.
// It validates magic bytes (not the client-supplied Content-Type header) and
// enforces maximum file sizes.
package upload

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
)

const (
	// MaxLogoBytes is the maximum allowed size for a logo file (2 MB).
	MaxLogoBytes = 2 << 20 // 2 MiB

	// MaxCSVBytes is the maximum allowed size for a CSV import file (10 MB).
	MaxCSVBytes = 10 << 20 // 10 MiB

	// multipartOverhead is slack added to MaxCSVBytes for multipart boundaries and headers.
	multipartOverhead = 1 << 20 // 1 MiB

	// MaxImportBodyBytes caps the whole request body of a CSV/JSON import upload.
	MaxImportBodyBytes = MaxCSVBytes + multipartOverhead
)

// magic byte prefixes for supported image types.
var (
	magicPNG  = []byte{0x89, 0x50, 0x4E, 0x47}
	magicJPEG = []byte{0xFF, 0xD8, 0xFF}
)

// Sentinel errors returned by the validation functions.
var (
	// ErrFileTooLarge is returned when an uploaded file exceeds the allowed size.
	ErrFileTooLarge = errors.New("file too large")
	// ErrInvalidMIME is returned when the detected MIME type does not match the expected type.
	ErrInvalidMIME = errors.New("invalid file type")
)

// DetectMIME returns the MIME type inferred from the file's magic bytes.
// It reads up to 512 bytes for detection (same as http.DetectContentType).
func DetectMIME(data []byte) string {
	if len(data) > 512 {
		return http.DetectContentType(data[:512])
	}
	return http.DetectContentType(data)
}

// hasMagicBytes returns true if data starts with the given prefix.
func hasMagicBytes(data, prefix []byte) bool {
	if len(data) < len(prefix) {
		return false
	}
	for i, b := range prefix {
		if data[i] != b {
			return false
		}
	}
	return true
}

// ValidateLogoFile checks that data is a valid PNG or JPEG image and does not
// exceed MaxLogoBytes. Returns ErrFileTooLarge or ErrInvalidMIME on failure.
func ValidateLogoFile(data []byte) error {
	if len(data) > MaxLogoBytes {
		return ErrFileTooLarge
	}
	if hasMagicBytes(data, magicPNG) || hasMagicBytes(data, magicJPEG) {
		return nil
	}
	return ErrInvalidMIME
}

// ValidateCSVFile checks that data looks like a plain-text file and does not
// exceed MaxCSVBytes. CSV files have no standard magic bytes, so we accept any
// content for which http.DetectContentType returns a text/* MIME type.
// Returns ErrFileTooLarge or ErrInvalidMIME on failure.
func ValidateCSVFile(data []byte) error {
	if len(data) > MaxCSVBytes {
		return ErrFileTooLarge
	}
	mime := DetectMIME(data)
	// http.DetectContentType returns "text/plain; charset=utf-8" for UTF-8 text,
	// which is the expected result for a well-formed CSV file.
	if len(mime) >= 5 && mime[:5] == "text/" {
		return nil
	}
	return ErrInvalidMIME
}

// ParseImportMultipart parses a multipart import upload after capping the request body at
// MaxImportBodyBytes (http.MaxBytesReader) and limiting in-memory parsing to MaxCSVBytes, so an
// oversize upload is rejected without being parsed. It returns ErrFileTooLarge when the body
// exceeds the cap; any other error means the body is not valid multipart/form-data.
func ParseImportMultipart(w http.ResponseWriter, r *http.Request) error {
	// A declared length over the cap is an oversize upload whatever its shape (#176).
	if r.ContentLength > MaxImportBodyBytes {
		return ErrFileTooLarge
	}
	capped := &errRecordingReader{r: http.MaxBytesReader(w, r.Body, MaxImportBodyBytes)}
	r.Body = capped
	err := r.ParseMultipartForm(MaxCSVBytes)
	if err == nil {
		return nil
	}
	// The multipart parser may wrap or replace the underlying read error, so also
	// consult the error the body reader itself reported.
	if isTooLarge(err) || isTooLarge(capped.err) {
		return ErrFileTooLarge
	}
	// The parser can refuse a body before reading any of it (not multipart/form-data, missing
	// boundary). Measure the rest of the body: an oversize body is 413 whatever its shape, while
	// a small body keeps the parser's error (400).
	if drainErr := drain(capped); isTooLarge(drainErr) || isTooLarge(capped.err) {
		return ErrFileTooLarge
	}
	return err
}

// drain reads and discards the rest of the body and returns the first error other than EOF.
func drain(r io.Reader) error {
	_, err := io.Copy(io.Discard, r)
	return err
}

func isTooLarge(err error) bool {
	if err == nil {
		return false
	}
	var mbe *http.MaxBytesError
	return errors.As(err, &mbe) || errors.Is(err, multipart.ErrMessageTooLarge)
}

// errRecordingReader remembers the first non-EOF error returned by the wrapped reader.
type errRecordingReader struct {
	r   io.ReadCloser
	err error
}

func (e *errRecordingReader) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if err != nil && err != io.EOF && e.err == nil {
		e.err = err
	}
	return n, err
}

func (e *errRecordingReader) Close() error { return e.r.Close() }
