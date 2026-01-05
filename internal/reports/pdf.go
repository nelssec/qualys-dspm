package reports

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type PDFReport struct {
	pdf   *gofpdf.Fpdf
	title string
}

func NewPDFReport(title string) *PDFReport {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 20)

	r := &PDFReport{
		pdf:   pdf,
		title: title,
	}

	r.addHeader()
	return r
}

func (r *PDFReport) addHeader() {
	r.pdf.AddPage()

	r.pdf.SetFont("Arial", "B", 20)
	r.pdf.SetTextColor(33, 37, 41)
	r.pdf.CellFormat(0, 15, r.title, "", 1, "C", false, 0, "")

	r.pdf.SetFont("Arial", "", 10)
	r.pdf.SetTextColor(108, 117, 125)
	r.pdf.CellFormat(0, 8, fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006 3:04 PM")), "", 1, "C", false, 0, "")

	r.pdf.Ln(10)
}

func (r *PDFReport) AddSection(title string) {
	r.pdf.SetFont("Arial", "B", 14)
	r.pdf.SetTextColor(33, 37, 41)
	r.pdf.SetFillColor(240, 240, 240)
	r.pdf.CellFormat(0, 10, title, "", 1, "L", true, 0, "")
	r.pdf.Ln(5)
}

func (r *PDFReport) AddParagraph(text string) {
	r.pdf.SetFont("Arial", "", 10)
	r.pdf.SetTextColor(33, 37, 41)
	r.pdf.MultiCell(0, 6, text, "", "L", false)
	r.pdf.Ln(5)
}

func (r *PDFReport) AddTable(headers []string, rows [][]string) {
	pageWidth := 180.0 // A4 width minus margins
	colWidth := pageWidth / float64(len(headers))

	r.pdf.SetFont("Arial", "B", 9)
	r.pdf.SetFillColor(52, 58, 64)
	r.pdf.SetTextColor(255, 255, 255)
	for _, h := range headers {
		r.pdf.CellFormat(colWidth, 8, h, "1", 0, "C", true, 0, "")
	}
	r.pdf.Ln(-1)

	r.pdf.SetFont("Arial", "", 9)
	r.pdf.SetTextColor(33, 37, 41)
	fill := false
	for _, row := range rows {
		if fill {
			r.pdf.SetFillColor(248, 249, 250)
		} else {
			r.pdf.SetFillColor(255, 255, 255)
		}
		for _, cell := range row {
			if len(cell) > 25 {
				cell = cell[:22] + "..."
			}
			r.pdf.CellFormat(colWidth, 7, cell, "1", 0, "L", true, 0, "")
		}
		r.pdf.Ln(-1)
		fill = !fill
	}

	r.pdf.Ln(5)
}

func (r *PDFReport) AddSummaryTable(data map[string]int) {
	r.pdf.SetFont("Arial", "", 10)

	for key, value := range data {
		r.pdf.SetTextColor(108, 117, 125)
		r.pdf.CellFormat(60, 7, key+":", "", 0, "L", false, 0, "")

		r.pdf.SetFont("Arial", "B", 10)
		r.pdf.SetTextColor(33, 37, 41)
		r.pdf.CellFormat(0, 7, fmt.Sprintf("%d", value), "", 1, "L", false, 0, "")
		r.pdf.SetFont("Arial", "", 10)
	}

	r.pdf.Ln(5)
}

func (r *PDFReport) AddChart(title string, data map[string]int) {
	r.pdf.SetFont("Arial", "B", 11)
	r.pdf.SetTextColor(33, 37, 41)
	r.pdf.CellFormat(0, 8, title, "", 1, "L", false, 0, "")

	max := 0
	for _, v := range data {
		if v > max {
			max = v
		}
	}

	if max == 0 {
		max = 1
	}

	barMaxWidth := 100.0

	for label, value := range data {
		r.pdf.SetFont("Arial", "", 9)
		r.pdf.SetTextColor(108, 117, 125)
		r.pdf.CellFormat(40, 6, label, "", 0, "L", false, 0, "")

		barWidth := float64(value) / float64(max) * barMaxWidth
		r.pdf.SetFillColor(66, 133, 244) // Blue
		r.pdf.CellFormat(barWidth, 6, "", "", 0, "L", true, 0, "")

		r.pdf.SetTextColor(33, 37, 41)
		r.pdf.CellFormat(30, 6, fmt.Sprintf(" %d", value), "", 1, "L", false, 0, "")
	}

	r.pdf.Ln(5)
}

func (r *PDFReport) AddSeverityIndicator(severity string) {
	var red, green, blue int

	switch severity {
	case "critical":
		red, green, blue = 220, 53, 69
	case "high":
		red, green, blue = 253, 126, 20
	case "medium":
		red, green, blue = 255, 193, 7
	case "low":
		red, green, blue = 40, 167, 69
	default:
		red, green, blue = 108, 117, 125
	}

	r.pdf.SetFillColor(red, green, blue)
	r.pdf.CellFormat(15, 6, "", "1", 0, "C", true, 0, "")
	r.pdf.SetFont("Arial", "B", 9)
	r.pdf.SetTextColor(255, 255, 255)
	r.pdf.CellFormat(15, 6, severity, "", 0, "C", true, 0, "")
}

func (r *PDFReport) AddPageBreak() {
	r.pdf.AddPage()
}

func (r *PDFReport) AddFooter() {
	r.pdf.SetFooterFunc(func() {
		r.pdf.SetY(-15)
		r.pdf.SetFont("Arial", "I", 8)
		r.pdf.SetTextColor(128, 128, 128)
		r.pdf.CellFormat(0, 10, fmt.Sprintf("Page %d", r.pdf.PageNo()), "", 0, "C", false, 0, "")
	})
}

func (r *PDFReport) Output() ([]byte, error) {
	r.AddFooter()

	var buf bytes.Buffer
	err := r.pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func (r *PDFReport) OutputToFile(filename string) error {
	r.AddFooter()
	return r.pdf.OutputFileAndClose(filename)
}

func ComplianceReportPDF(title string, frameworks map[string]*ComplianceStatus) ([]byte, error) {
	pdf := NewPDFReport(title)

	pdf.AddSection("Compliance Overview")
	pdf.AddParagraph("This report provides an overview of compliance status across multiple regulatory frameworks.")

	for _, status := range frameworks {
		compliancePct := float64(status.PassedChecks) / float64(status.TotalChecks) * 100
		pdf.pdf.SetFont("Arial", "B", 11)
		pdf.pdf.SetTextColor(33, 37, 41)
		pdf.pdf.CellFormat(50, 8, status.Framework, "", 0, "L", false, 0, "")

		barWidth := compliancePct * 0.8 // Scale to max 80mm
		if compliancePct >= 80 {
			pdf.pdf.SetFillColor(40, 167, 69) // Green
		} else if compliancePct >= 50 {
			pdf.pdf.SetFillColor(255, 193, 7) // Yellow
		} else {
			pdf.pdf.SetFillColor(220, 53, 69) // Red
		}
		pdf.pdf.CellFormat(barWidth, 8, "", "", 0, "L", true, 0, "")
		pdf.pdf.CellFormat(0, 8, fmt.Sprintf(" %.0f%%", compliancePct), "", 1, "L", false, 0, "")
	}

	pdf.pdf.Ln(10)

	for _, status := range frameworks {
		if len(status.Findings) > 0 {
			pdf.AddSection(fmt.Sprintf("%s - %d Issues", status.Framework, len(status.Findings)))

			headers := []string{"Finding", "Severity", "Status", "Asset"}
			rows := make([][]string, 0, len(status.Findings))
			for _, f := range status.Findings {
				rows = append(rows, []string{
					truncate(f.Title, 35),
					string(f.Severity),
					string(f.Status),
					truncate(f.AssetID, 15),
				})
			}
			pdf.AddTable(headers, rows)
		}
	}

	return pdf.Output()
}

func ExecutiveSummaryPDF(title string, stats *Stats) ([]byte, error) {
	pdf := NewPDFReport(title)

	pdf.AddSection("Security Posture Summary")

	metrics := []struct {
		label string
		value int
		color []int
	}{
		{"Total Assets", stats.TotalAssets, []int{66, 133, 244}},
		{"Total Findings", stats.TotalFindings, []int{108, 117, 125}},
		{"Critical", stats.CriticalFindings, []int{220, 53, 69}},
		{"High", stats.HighFindings, []int{253, 126, 20}},
	}

	boxWidth := 42.0
	for i, m := range metrics {
		x := 15 + float64(i)*boxWidth + float64(i)*5
		pdf.pdf.SetFillColor(m.color[0], m.color[1], m.color[2])
		pdf.pdf.Rect(x, pdf.pdf.GetY(), boxWidth, 25, "F")

		pdf.pdf.SetXY(x, pdf.pdf.GetY()+3)
		pdf.pdf.SetFont("Arial", "B", 18)
		pdf.pdf.SetTextColor(255, 255, 255)
		pdf.pdf.CellFormat(boxWidth, 10, fmt.Sprintf("%d", m.value), "", 0, "C", false, 0, "")

		pdf.pdf.SetXY(x, pdf.pdf.GetY()+12)
		pdf.pdf.SetFont("Arial", "", 9)
		pdf.pdf.CellFormat(boxWidth, 8, m.label, "", 0, "C", false, 0, "")
	}

	pdf.pdf.Ln(35)

	pdf.AddSection("Findings by Severity")
	pdf.AddChart("", map[string]int{
		"Critical": stats.CriticalFindings,
		"High":     stats.HighFindings,
		"Medium":   stats.MediumFindings,
		"Low":      stats.LowFindings,
	})

	pdf.AddSection("Findings by Status")
	pdf.AddSummaryTable(map[string]int{
		"Open":     stats.OpenFindings,
		"Resolved": stats.ResolvedFindings,
	})

	return pdf.Output()
}
