# FR-BB44 — PDF Generation

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB44 |
| Phase | 4 — Results & Certificates |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB43 |

## Description
Implements the server-side PDF certificate generator as a pure Go library function. Given a `Certificate` record and tenant configuration, it produces an A4-landscape PDF with the company branding, employee details, a QR code pointing to the public verification URL, and a signatory section. The function has no file system or network dependencies at generation time; all inputs come from in-memory data structures.

## Acceptance Criteria
- [ ] AC-1: `GeneratePDF` accepts `*Certificate` and `*TenantConfig` and returns `([]byte, error)`; it has no side effects on the database or file system.
- [ ] AC-2: The generated PDF is A4 landscape (297 mm × 210 mm); all content fits within 10 mm margins on all sides.
- [ ] AC-3: The company logo (decoded from `template_snapshot.logo_base64`) is rendered top-left; if `logo_base64` is empty, the company name text is rendered in its place.
- [ ] AC-4: The certificate title "Certificate of Completion" is centred in a font size ≥ 28pt; the employee's full name is centred below it in a font size ≥ 22pt.
- [ ] AC-5: The exam title, score percentage, and date issued are displayed below the employee name.
- [ ] AC-6: A QR code is rendered in the bottom-right quadrant pointing to `{base_url}/verify/{verification_code}`; the QR code is at least 30 mm × 30 mm.
- [ ] AC-7: The certificate's `verification_code` UUID is printed as text in the footer.
- [ ] AC-8: The signatory name and title are printed with a horizontal rule above them in the bottom-left quadrant.
- [ ] AC-9: `GeneratePDF` returns an error (not a panic) if `logo_base64` contains invalid base64 data; the caller handles the error and returns HTTP 500.
- [ ] AC-10: The function is covered by at least one unit test that calls `GeneratePDF` with mock data and asserts the returned bytes are a valid PDF (starts with `%PDF-`).

## Technical Specification

### Dependencies

```
github.com/jung-kurt/gofpdf v1.16.2   // PDF layout engine
github.com/skip2/go-qrcode v0.0.0     // inline QR code as PNG bytes
```

Add to `backend/go.mod` via:
```bash
go get github.com/jung-kurt/gofpdf@v1.16.2
go get github.com/skip2/go-qrcode@latest
```

### Function Signature

```go
// Package certificates — internal/certificates/pdf.go

// GeneratePDF produces an A4-landscape PDF certificate.
// cert must not be nil. tenantCfg must not be nil.
// Returns raw PDF bytes or an error.
func GeneratePDF(cert *Certificate, tenantCfg *TenantConfig, verifyBaseURL string) ([]byte, error)
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
func GeneratePDF(cert *Certificate, tenantCfg *TenantConfig, verifyBaseURL string) ([]byte, error) {
    pdf := gofpdf.NewCustom(&gofpdf.InitType{
        OrientationStr: "L",
        UnitStr:        "mm",
        SizeStr:        "A4",
    })
    pdf.SetMargins(10, 10, 10)
    pdf.AddPage()

    // --- Header ---
    if tenantCfg.LogoBase64 != "" {
        imgBytes, err := base64.StdEncoding.DecodeString(stripDataURI(tenantCfg.LogoBase64))
        if err != nil {
            return nil, fmt.Errorf("certificates: invalid logo base64: %w", err)
        }
        imgOpt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
        pdf.RegisterImageOptionsReader("logo", imgOpt, bytes.NewReader(imgBytes))
        pdf.ImageOptions("logo", 10, 5, 40, 0, false, imgOpt, 0, "")
    } else {
        pdf.SetFont("Helvetica", "B", 14)
        pdf.SetXY(10, 8)
        pdf.CellFormat(60, 10, tenantCfg.CompanyName, "", 0, "L", false, 0, "")
    }

    // --- QR Code ---
    verifyURL := fmt.Sprintf("%s/verify/%s", verifyBaseURL, cert.VerificationCode)
    qrBytes, err := qrcode.Encode(verifyURL, qrcode.Medium, 128)
    if err != nil {
        return nil, fmt.Errorf("certificates: qr generation failed: %w", err)
    }
    pdf.RegisterImageOptionsReader("qr", gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(qrBytes))
    pdf.ImageOptions("qr", 255, 162, 32, 32, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

    // ... (title, name, exam title, score, signatory sections) ...

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
func TestGeneratePDF_ValidOutput(t *testing.T) {
    cert := &Certificate{
        VerificationCode: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
        EmployeeName:     "Test User",
        ExamTitle:        "Test Exam",
        ScorePct:         90.0,
        IssuedAt:         time.Now(),
    }
    cfg := &TenantConfig{
        CompanyName:    "Test Corp",
        PrimaryColor:   "#336699",
        SignatoryName:  "HR Manager",
        SignatoryTitle: "Head of HR",
    }
    b, err := GeneratePDF(cert, cfg, "https://app.bilimbaga.kz")
    require.NoError(t, err)
    require.True(t, bytes.HasPrefix(b, []byte("%PDF-")), "output must be valid PDF")
    require.Greater(t, len(b), 1024)
}
```

## Notes
- `gofpdf` supports Unicode via UTF-8 fonts; ensure a UTF-8 capable font (e.g. DejaVu) is embedded for Cyrillic and Kazakh character support in employee names.
- If `gofpdf` is replaced in the future (e.g. by `unipdf`), the function signature must remain unchanged to avoid modifying callers.
- `verifyBaseURL` is read from `Config.BaseURL` set at startup; never hardcoded.
