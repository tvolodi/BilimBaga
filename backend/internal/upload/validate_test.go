package upload

import (
	"bytes"
	"errors"
	"testing"
)

// minimalPNG is a valid 1×1 pixel PNG image for testing.
var minimalPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk length + type
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1×1 px
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // bit depth, color type, ...
	0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
	0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
	0x00, 0x00, 0x02, 0x00, 0x01, 0xE2, 0x21, 0xBC,
	0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, // IEND chunk
	0x44, 0xAE, 0x42, 0x60, 0x82,
}

// minimalJPEG is the SOI (start of image) marker for JPEG.
var minimalJPEG = []byte{
	0xFF, 0xD8, 0xFF, 0xE0, // JPEG SOI + APP0 marker
	0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, // JFIF header
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, // aspect ratio
}

func TestValidateLogoFile_PNG(t *testing.T) {
	if err := ValidateLogoFile(minimalPNG); err != nil {
		t.Errorf("expected nil for valid PNG, got %v", err)
	}
}

func TestValidateLogoFile_JPEG(t *testing.T) {
	if err := ValidateLogoFile(minimalJPEG); err != nil {
		t.Errorf("expected nil for valid JPEG, got %v", err)
	}
}

func TestValidateLogoFile_TooLarge(t *testing.T) {
	// Create a PNG-like file that exceeds MaxLogoBytes.
	large := make([]byte, MaxLogoBytes+1)
	copy(large, magicPNG)
	err := ValidateLogoFile(large)
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestValidateLogoFile_WrongMIME(t *testing.T) {
	// Use a PDF magic byte header.
	pdfHeader := []byte{0x25, 0x50, 0x44, 0x46, 0x2D} // %PDF-
	err := ValidateLogoFile(pdfHeader)
	if !errors.Is(err, ErrInvalidMIME) {
		t.Errorf("expected ErrInvalidMIME for PDF bytes, got %v", err)
	}
}

func TestValidateCSVFile_Valid(t *testing.T) {
	csv := []byte("name,email,department\nAlice,alice@example.com,Engineering\n")
	if err := ValidateCSVFile(csv); err != nil {
		t.Errorf("expected nil for valid CSV text, got %v", err)
	}
}

func TestValidateCSVFile_TooLarge(t *testing.T) {
	large := bytes.Repeat([]byte("a,b,c\n"), MaxCSVBytes/6+1)
	err := ValidateCSVFile(large)
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got %v", err)
	}
}
