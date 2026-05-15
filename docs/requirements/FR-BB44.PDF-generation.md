# FR-BB44 — PDF Generation

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB44 |
| Phase | 4 — Results & Certificates |
| Priority | 1 |
| Status | implemented |
| Depends On | none (requires only that `Certificate` and `TemplateSnapshot` types exist in the `certificates` package) |

## Description
Implements the server-side PDF certificate generator as a pure Go library function. This requirement replaces the existing stub in `internal/certificates/pdf.go` with the full A4-landscape implementation. Given a `Certificate` record and a `TemplateSnapshot` (tenant branding captured at issuance time), it produces an A4-landscape PDF with the company branding, employee details, a QR code pointing to the public verification URL, and a signatory section. The function has no file system or network dependencies at generation time; all inputs come from in-memory data structures.

## Scope

| Layer | Items |
|-------|-------|
| Backend | `internal/certificates/pdf.go` (replaces existing stub); `go.mod` (two new dependencies: `github.com/go-pdf/fpdf` and `github.com/skip2/go-qrcode`) |
| Frontend | N/A |
| Database | N/A |
| i18n | N/A |

## Acceptance Criteria
- [ ] AC-1: `GeneratePDF` accepts `*Certificate` and `TemplateSnapshot` (value, not pointer) and returns `([]byte, error)`; it has no side effects on the database or file system.
- [ ] AC-2: The generated PDF is A4 landscape (297 mm × 210 mm); all content fits within 10 mm margins on all sides.
- [ ] AC-3: The company logo (decoded from `TemplateSnapshot.LogoBase64`) is rendered top-left; if `LogoBase64` is empty, the company name text is rendered in its place.
- [ ] AC-4: The certificate title "Certificate of Completion" is centred in a font size ≥ 28pt; the employee's full name is centred below it in a font size ≥ 22pt.
- [ ] AC-5: The exam title, score percentage, and date issued are displayed below the employee name.
- [ ] AC-6: A QR code is rendered in the bottom-right quadrant pointing to `{base_url}/verify/{verification_code}`; the QR code is at least 30 mm × 30 mm.
- [ ] AC-7: The certificate's `verification_code` is printed as text in the footer.
- [ ] AC-8: The signatory name and title (from `TemplateSnapshot.SignatoryName` and `TemplateSnapshot.SignatoryTitle`) are printed with a horizontal rule above them in the bottom-left quadrant.
- [ ] AC-9: `GeneratePDF` returns an error (not a panic) if `LogoBase64` contains invalid base64 data; the caller handles the error and returns HTTP 500.
- [ ] AC-10: The function is covered by at least one unit test that calls `GeneratePDF` with mock data and asserts the returned bytes are a valid PDF (starts with `%PDF-`).

## Technical Specification

### Dependencies

```
github.com/go-pdf/fpdf/v2 v2.x        // PDF layout engine (community successor to jung-kurt/gofpdf, API-compatible)
github.com/skip2/go-qrcode v0.0.0     // inline QR code as PNG bytes
```

Library selection rationale: `go-pdf/fpdf` is chosen over `unipdf` (avoids commercial licence) and `wkhtmltopdf` (avoids system binary dependency). It is the actively maintained community fork of the archived `jung-kurt/gofpdf` with an identical API.

Add to `backend/go.mod` via:
```bash
go get github.com/go-pdf/fpdf/v2@latest
go get github.com/skip2/go-qrcode@latest
```

### Function Signature

```go
// Package certificates — internal/certificates/pdf.go

// GeneratePDF produces an A4-landscape PDF certificate.
// cert must not be nil. snap is the TemplateSnapshot captured at issuance time.
// Returns raw PDF bytes or an error.
func GeneratePDF(cert *Certificate, snap TemplateSnapshot, verifyBaseURL string) ([]byte, error)
```

This signature matches the existing call site in `internal/certificates/handler.go`:

```go
// handler.go — streamPDF (already present, must not be changed)
pdfBytes, err := GeneratePDF(cert, snap, h.baseURL)
```

### PDF Layout (A4 Landscape: 297 mm × 210 mm)

```
┌──────────────────────────────────────────────────────────────────────┐
│  [Logo / Company Name]              [App name: BilimBaga]            │  ← header strip (h=20mm)
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│                      Certificate of Completion                       │  ← 28pt bold, centred
│                                                                      │
│                           Aibek Seitkali                             │  ← 22pt, centred
│                                                                      │
│             for successfully completing                               │  ← 14pt regular
│                                                                      │
│                    Fire Safety Fundamentals                          │  ← 18pt, centred
│                                                                      │
│             Score: 84.50%            Issued: 14 May 2026             │  ← 12pt, centred
│                                                                      │
├─────────────────────────────────┬────────────────────────────────────┤
│  ─────────────────────────      │      [QR code 32mm×32mm]           │  ← footer (h=35mm)
│  Aigerim Bekova                 │                                    │
│  Head of Human Resources        │  Verify: {base_url}/verify/{code}  │
│                                 │  Certificate ID: {uuid}            │
└─────────────────────────────────┴────────────────────────────────────┘
```

### Implementation Notes

```go
func GeneratePDF(cert *Certificate, snap TemplateSnapshot, verifyBaseURL string) ([]byte, error) {
    pdf := fpdf.NewCustom(&fpdf.InitType{
        OrientationStr: "L",
        UnitStr:        "mm",
        SizeStr:        "A4",
    })
    pdf.SetMargins(10, 10, 10)
    pdf.AddPage()

    // --- Header ---
    if snap.LogoBase64 != "" {
        // Strip any data URI prefix (e.g. "data:image/png;base64,") before decoding.
        logoData := snap.LogoBase64
        if idx := strings.LastIndex(logoData, ","); idx >= 0 {
            logoData = logoData[idx+1:]
        }
        imgBytes, err := base64.StdEncoding.DecodeString(logoData)
        if err != nil {
            return nil, fmt.Errorf("certificates: invalid logo base64: %w", err)
        }
        imgOpt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
        pdf.RegisterImageOptionsReader("logo", imgOpt, bytes.NewReader(imgBytes))
        pdf.ImageOptions("logo", 10, 5, 40, 0, false, imgOpt, 0, "")
    } else {
        pdf.SetFont("Helvetica", "B", 14)
        pdf.SetXY(10, 8)
        pdf.CellFormat(60, 10, snap.CompanyName, "", 0, "L", false, 0, "")
    }

    // --- QR Code ---
    verifyURL := fmt.Sprintf("%s/verify/%s", verifyBaseURL, cert.VerificationCode)
    qrBytes, err := qrcode.Encode(verifyURL, qrcode.Medium, 128)
    if err != nil {
        return nil, fmt.Errorf("certificates: qr generation failed: %w", err)
    }
    pdf.RegisterImageOptionsReader("qr", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(qrBytes))
    pdf.ImageOptions("qr", 255, 162, 32, 32, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")

    // --- Signatory ---
    pdf.SetXY(10, 170)
    pdf.Line(10, 168, 100, 168)
    pdf.SetFont("Helvetica", "B", 12)
    pdf.CellFormat(90, 6, snap.SignatoryName, "", 1, "L", false, 0, "")
    pdf.SetFont("Helvetica", "", 10)
    pdf.CellFormat(90, 5, snap.SignatoryTitle, "", 0, "L", false, 0, "")

    // --- Certificate Body ---
    pdf.SetFont("Helvetica", "B", 28)
    pdf.SetXY(10, 40)
    pdf.CellFormat(277, 12, "Certificate of Completion", "", 1, "C", false, 0, "")

    pdf.SetFont("Helvetica", "", 22)
    pdf.SetX(10)
    pdf.CellFormat(277, 10, cert.EmployeeName, "", 1, "C", false, 0, "")

    pdf.SetFont("Helvetica", "", 14)
    pdf.SetX(10)
    pdf.CellFormat(277, 8, "for successfully completing", "", 1, "C", false, 0, "")

    pdf.SetFont("Helvetica", "B", 18)
    pdf.SetX(10)
    pdf.CellFormat(277, 9, cert.ExamTitle, "", 1, "C", false, 0, "")

    pdf.SetFont("Helvetica", "", 12)
    pdf.SetX(10)
    scoreDate := fmt.Sprintf("Score: %.2f%%          Issued: %s",
        cert.ScorePct, cert.IssuedAt.UTC().Format("02 Jan 2006"))
    pdf.CellFormat(277, 8, scoreDate, "", 1, "C", false, 0, "")

    var buf bytes.Buffer
    if err := pdf.Output(&buf); err != nil {
        return nil, fmt.Errorf("certificates: pdf output failed: %w", err)
    }
    return buf.Bytes(), nil
}
```

### Unit Test Skeleton

```go
// internal/certificates/pdf_test.go
package certificates

import (
    "testing"
    "time"

    "github.com/stretchr/testify/require"
)

func TestGeneratePDF_ValidOutput(t *testing.T) {
    cert := &Certificate{
        VerificationCode: "550e8400-e29b-41d4-a716-446655440000",
        EmployeeName:     "Test User",
        ExamTitle:        "Test Exam",
        ScorePct:         90.0,
        IssuedAt:         time.Now(),
    }
    snap := TemplateSnapshot{
        CompanyName:    "Test Corp",
        PrimaryColor:   "#336699",
        SignatoryName:  "HR Manager",
        SignatoryTitle: "Head of HR",
    }
    b, err := GeneratePDF(cert, snap, "https://app.bilimbaga.kz")
    require.NoError(t, err)
    require.GreaterOrEqual(t, len(b), 5, "output too short to be a PDF")
    require.Equal(t, "%PDF-", string(b[:5]), "output must start with PDF magic bytes")
    require.Greater(t, len(b), 1024)
}

func TestGeneratePDF_InvalidLogoBase64(t *testing.T) {
    cert := &Certificate{
        VerificationCode: "550e8400-e29b-41d4-a716-446655440000",
        EmployeeName:     "Test User",
        ExamTitle:        "Test Exam",
        ScorePct:         90.0,
        IssuedAt:         time.Now(),
    }
    snap := TemplateSnapshot{
        CompanyName: "Test Corp",
        LogoBase64:  "!!!not-valid-base64!!!",
    }
    _, err := GeneratePDF(cert, snap, "https://app.bilimbaga.kz")
    require.Error(t, err, "invalid base64 logo must return an error")
}

func TestGeneratePDF_EmptyLogoFallsBackToCompanyName(t *testing.T) {
    cert := &Certificate{
        VerificationCode: "550e8400-e29b-41d4-a716-446655440000",
        EmployeeName:     "Test User",
        ExamTitle:        "Test Exam",
        ScorePct:         90.0,
        IssuedAt:         time.Now(),
    }
    snap := TemplateSnapshot{
        CompanyName:    "Fallback Corp",
        LogoBase64:     "", // empty — should render company name text
        SignatoryName:  "HR Manager",
        SignatoryTitle: "Head of HR",
    }
    b, err := GeneratePDF(cert, snap, "https://app.bilimbaga.kz")
    require.NoError(t, err)
    require.Equal(t, "%PDF-", string(b[:5]))
}
```

## Out of Scope

- Storing PDF bytes to disk or object storage — the caller streams bytes directly to the HTTP response.
- Multi-font Cyrillic/Kazakh support beyond the library's native capability — a follow-up requirement may embed a DejaVu or Noto font for full Unicode coverage.
- Email delivery of the certificate PDF.

## Test Strategy

- **Unit tests** (`internal/certificates/pdf_test.go`):
  1. Valid inputs produce bytes where `string(b[:5]) == "%PDF-"` (AC-10).
  2. `LogoBase64` containing invalid base64 returns a non-nil error without panicking (AC-9).
  3. Empty `LogoBase64` produces a valid PDF (company name text fallback path, AC-3).
- No integration tests are required for this requirement; `GeneratePDF` is a pure in-memory function with no external dependencies.

## Notes
- `verifyBaseURL` is read from `Config.APIBaseURL` set at startup; never hardcoded.
- If the PDF library is replaced in the future, the function signature `GeneratePDF(cert *Certificate, snap TemplateSnapshot, verifyBaseURL string) ([]byte, error)` must remain unchanged to avoid modifying callers.
