// Package upload provides server-side file validation for uploaded files.
// It validates magic bytes (not the client-supplied Content-Type header) and
// enforces maximum file sizes.
package upload

import (
	"errors"
	"net/http"
)

const (
	// MaxLogoBytes is the maximum allowed size for a logo file (2 MB).
	MaxLogoBytes = 2 << 20 // 2 MiB

	// MaxCSVBytes is the maximum allowed size for a CSV import file (10 MB).
	MaxCSVBytes = 10 << 20 // 10 MiB
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
