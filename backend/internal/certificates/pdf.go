package certificates

import (
	"bytes"
	"fmt"
	"strings"
)

// GeneratePDF creates a simple single-page PDF certificate.
// Uses only built-in PDF Type1 fonts (Helvetica / Helvetica-Bold) — no external font files required.
// Non-ASCII characters (e.g. Cyrillic) render as Latin-1 equivalents; for Phase 4 this is acceptable.
func GeneratePDF(cert *Certificate, snapshot TemplateSnapshot, verifyBaseURL string) ([]byte, error) {
	verifyURL := verifyBaseURL + "/verify/" + cert.VerificationCode

	companyName := snapshot.CompanyName
	if companyName == "" {
		companyName = "BilimBaga"
	}

	score := fmt.Sprintf("Score: %.1f%%", cert.ScorePct)
	date := "Date: " + cert.IssuedAt.UTC().Format("January 2, 2006")
	code := "Verification Code: " + cert.VerificationCode
	verify := "Verify at: " + verifyURL

	// Build the page content stream using PDF drawing operators.
	var cs strings.Builder
	text := func(font string, size int, x, y float64, s string) {
		cs.WriteString(fmt.Sprintf("BT /%s %d Tf %.1f %.1f Td (%s) Tj ET\n",
			font, size, x, y, pdfEscape(s)))
	}

	text("F1", 18, 50, 740, companyName)
	text("F1", 24, 50, 700, "Certificate of Completion")
	text("F2", 12, 50, 645, "This certifies that")
	text("F1", 16, 50, 620, cert.EmployeeName)
	text("F2", 12, 50, 588, "has successfully completed")
	text("F1", 14, 50, 562, cert.ExamTitle)
	text("F2", 12, 50, 520, score)
	text("F2", 12, 50, 500, date)

	if snapshot.SignatoryName != "" {
		text("F1", 12, 50, 440, snapshot.SignatoryName)
		if snapshot.SignatoryTitle != "" {
			text("F2", 10, 50, 425, snapshot.SignatoryTitle)
		}
	}

	text("F2", 9, 50, 375, code)
	text("F2", 9, 50, 360, verify)

	content := cs.String()

	// Assemble the PDF object bodies.
	catalog := "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n"
	pages := "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n"
	page := "3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792]\n" +
		"   /Contents 4 0 R\n" +
		"   /Resources << /Font << /F1 5 0 R /F2 6 0 R >> >> >>\nendobj\n"
	stream := fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%sendstream\nendobj\n",
		len(content), content)
	fontBold := "5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>\nendobj\n"
	fontPlain := "6 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n"

	bodies := []string{catalog, pages, page, stream, fontBold, fontPlain}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(bodies))
	for i, body := range bodies {
		offsets[i] = buf.Len()
		buf.WriteString(body)
	}

	// Cross-reference table. Each entry is exactly 20 bytes (10+1+5+1+1+1+1).
	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(bodies)+1)
	fmt.Fprintf(&buf, "0000000000 65535 f \n") // free object 0
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}

	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\n", len(bodies)+1)
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	return buf.Bytes(), nil
}

// pdfEscape escapes characters that are special inside PDF string literals.
func pdfEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
