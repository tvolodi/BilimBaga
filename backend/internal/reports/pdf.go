package reports

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	fpdf "github.com/go-pdf/fpdf"
)

// GenerateDashboardPDF renders a dashboard summary PDF from the provided data.
// It returns raw PDF bytes or an error.
// AC-9: uses the same gofpdf library as FR-BB44 (certificates package).
func GenerateDashboardPDF(data *DashboardReportData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	pageW := 180.0 // 210 - 15*2

	// ── Header: Logo + Title ─────────────────────────────────────────────────
	if data.LogoBase64 != "" {
		logoData := data.LogoBase64
		if idx := strings.LastIndex(logoData, ","); idx >= 0 {
			logoData = logoData[idx+1:]
		}
		imgBytes, err := base64.StdEncoding.DecodeString(logoData)
		if err == nil && len(imgBytes) > 0 {
			imgOpt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
			pdf.RegisterImageOptionsReader("logo", imgOpt, bytes.NewReader(imgBytes))
			pdf.ImageOptions("logo", 15, 10, 30, 0, false, imgOpt, 0, "")
		}
	}

	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(50, 12)
	title := data.CompanyName
	if title == "" {
		title = "BilimBaga"
	}
	pdf.CellFormat(pageW-35, 8, title+" — Analytics Report", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 11)
	pdf.SetX(50)
	pdf.CellFormat(pageW-35, 6, "Date Range: "+data.From+" – "+data.To, "", 1, "L", false, 0, "")

	pdf.Ln(4)

	// ── Completion Rates Table ───────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(pageW, 8, "Completion Rates by Exam", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	// Table header
	colWidths := []float64{70, 30, 30, 30, 20}
	headers := []string{"Exam", "Assigned", "Completed", "Passed", "Pass%"}
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetFillColor(255, 255, 255)
	for _, cr := range data.CompletionRates {
		passRate := ""
		if cr.AssignedCount > 0 {
			pct := float64(cr.PassedCount) / float64(cr.AssignedCount) * 100
			passRate = fmt.Sprintf("%.1f%%", pct)
		}
		pdf.CellFormat(colWidths[0], 6, cr.Title, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[1], 6, fmt.Sprintf("%d", cr.AssignedCount), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[2], 6, fmt.Sprintf("%d", cr.CompletedCount), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[3], 6, fmt.Sprintf("%d", cr.PassedCount), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[4], 6, passRate, "1", 0, "C", false, 0, "")
		pdf.Ln(-1)
	}
	if len(data.CompletionRates) == 0 {
		pdf.CellFormat(pageW, 6, "No data in range.", "1", 1, "L", false, 0, "")
	}

	pdf.Ln(6)

	// ── Top 5 Questions ──────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(pageW, 8, "Top 5 Questions (Highest Correct Rate)", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	pdf.SetFont("Helvetica", "", 10)
	if len(data.TopQuestions) == 0 {
		pdf.CellFormat(pageW, 6, "No data in range.", "", 1, "L", false, 0, "")
	}
	for i, q := range data.TopQuestions {
		rateStr := "N/A"
		if q.CorrectRate != nil {
			rateStr = fmt.Sprintf("%.0f%%", *q.CorrectRate*100)
		}
		line := fmt.Sprintf("%d. \"%s\" — %s", i+1, q.StemPreview, rateStr)
		pdf.MultiCell(pageW, 5, line, "", "L", false)
	}

	pdf.Ln(4)

	// ── Bottom 5 Questions ───────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(pageW, 8, "Bottom 5 Questions (Lowest Correct Rate)", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	pdf.SetFont("Helvetica", "", 10)
	if len(data.BottomQuestions) == 0 {
		pdf.CellFormat(pageW, 6, "No data in range.", "", 1, "L", false, 0, "")
	}
	for i, q := range data.BottomQuestions {
		rateStr := "N/A"
		if q.CorrectRate != nil {
			rateStr = fmt.Sprintf("%.0f%%", *q.CorrectRate*100)
		}
		line := fmt.Sprintf("%d. \"%s\" — %s", i+1, q.StemPreview, rateStr)
		pdf.MultiCell(pageW, 5, line, "", "L", false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("reports: GenerateDashboardPDF: output: %w", err)
	}
	return buf.Bytes(), nil
}
