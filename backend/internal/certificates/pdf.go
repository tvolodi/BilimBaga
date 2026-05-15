package certificates

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	fpdf "github.com/go-pdf/fpdf"
	"github.com/skip2/go-qrcode"
)

// GeneratePDF produces an A4-landscape PDF certificate.
// cert must not be nil. snap is the TemplateSnapshot captured at issuance time.
// Returns raw PDF bytes or an error.
func GeneratePDF(cert *Certificate, snap TemplateSnapshot, verifyBaseURL string) ([]byte, error) {
	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "L",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	pdf.SetMargins(10, 10, 10)
	pdf.AddPage()

	// --- Header strip (h=20mm) ---
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

	// App name top-right
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetXY(200, 8)
	pdf.CellFormat(87, 10, "BilimBaga", "", 0, "R", false, 0, "")

	// --- Certificate body ---
	pdf.SetFont("Helvetica", "B", 28)
	pdf.SetXY(10, 30)
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

	// --- Footer: signatory (bottom-left quadrant) ---
	pdf.Line(10, 168, 100, 168)
	pdf.SetXY(10, 170)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(90, 6, snap.SignatoryName, "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(90, 5, snap.SignatoryTitle, "", 0, "L", false, 0, "")

	// --- Footer: QR code (bottom-right quadrant, ≥30mm×30mm) ---
	verifyURL := fmt.Sprintf("%s/verify/%s", verifyBaseURL, cert.VerificationCode)
	qrBytes, err := qrcode.Encode(verifyURL, qrcode.Medium, 128)
	if err != nil {
		return nil, fmt.Errorf("certificates: qr generation failed: %w", err)
	}
	pdf.RegisterImageOptionsReader("qr", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(qrBytes))
	pdf.ImageOptions("qr", 255, 162, 32, 32, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	// Verify URL and Certificate ID text (footer right, below QR)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetXY(150, 186)
	pdf.CellFormat(137, 4, "Verify: "+verifyURL, "", 1, "L", false, 0, "")
	pdf.SetX(150)
	pdf.CellFormat(137, 4, "Certificate ID: "+cert.VerificationCode, "", 0, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("certificates: pdf output failed: %w", err)
	}
	return buf.Bytes(), nil
}
