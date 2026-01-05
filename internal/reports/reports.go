package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/qualys/dspm/internal/models"
)

// ReportType defines the type of report
type ReportType string

const (
	ReportTypeFindings       ReportType = "findings"
	ReportTypeAssets         ReportType = "assets"
	ReportTypeCompliance     ReportType = "compliance"
	ReportTypeExecutive      ReportType = "executive"
	ReportTypeClassification ReportType = "classification"
)

// ReportFormat defines the output format
type ReportFormat string

const (
	FormatCSV  ReportFormat = "csv"
	FormatPDF  ReportFormat = "pdf"
	FormatJSON ReportFormat = "json"
)

// ReportRequest contains report generation parameters
type ReportRequest struct {
	Type       ReportType
	Format     ReportFormat
	Title      string
	AccountIDs []string
	DateFrom   *time.Time
	DateTo     *time.Time
	Severities []models.Sensitivity
	Categories []models.Category
	Statuses   []models.FindingStatus
}

// Report represents a generated report
type Report struct {
	ID          string
	Type        ReportType
	Format      ReportFormat
	Title       string
	GeneratedAt time.Time
	GeneratedBy string
	Data        []byte
	Filename    string
	MimeType    string
}

// DataProvider interface for fetching report data
type DataProvider interface {
	GetFindings(ctx context.Context, filters FindingsFilter) ([]*models.Finding, error)
	GetAssets(ctx context.Context, filters AssetsFilter) ([]*models.DataAsset, error)
	GetClassifications(ctx context.Context, assetID string) ([]*models.Classification, error)
	GetAccounts(ctx context.Context) ([]*models.CloudAccount, error)
	GetStats(ctx context.Context) (*Stats, error)
}

// FindingsFilter contains filter parameters for findings
type FindingsFilter struct {
	AccountIDs []string
	Severities []models.Sensitivity
	Categories []models.Category
	Statuses   []models.FindingStatus
	DateFrom   *time.Time
	DateTo     *time.Time
}

// AssetsFilter contains filter parameters for assets
type AssetsFilter struct {
	AccountIDs   []string
	AssetTypes   []models.AssetType
	Sensitivities []models.Sensitivity
}

// Stats holds summary statistics
type Stats struct {
	TotalAccounts        int
	TotalAssets          int
	TotalFindings        int
	CriticalFindings     int
	HighFindings         int
	MediumFindings       int
	LowFindings          int
	OpenFindings         int
	ResolvedFindings     int
	ClassificationCounts map[models.Category]int
	SensitivityCounts    map[models.Sensitivity]int
}

// Generator handles report generation
type Generator struct {
	provider DataProvider
}

// NewGenerator creates a new report generator
func NewGenerator(provider DataProvider) *Generator {
	return &Generator{provider: provider}
}

// Generate generates a report
func (g *Generator) Generate(ctx context.Context, req *ReportRequest) (*Report, error) {
	switch req.Type {
	case ReportTypeFindings:
		return g.generateFindingsReport(ctx, req)
	case ReportTypeAssets:
		return g.generateAssetsReport(ctx, req)
	case ReportTypeClassification:
		return g.generateClassificationReport(ctx, req)
	case ReportTypeExecutive:
		return g.generateExecutiveReport(ctx, req)
	case ReportTypeCompliance:
		return g.generateComplianceReport(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported report type: %s", req.Type)
	}
}

// generateFindingsReport generates a findings report
func (g *Generator) generateFindingsReport(ctx context.Context, req *ReportRequest) (*Report, error) {
	findings, err := g.provider.GetFindings(ctx, FindingsFilter{
		AccountIDs: req.AccountIDs,
		Severities: req.Severities,
		Categories: req.Categories,
		Statuses:   req.Statuses,
		DateFrom:   req.DateFrom,
		DateTo:     req.DateTo,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch findings: %w", err)
	}

	var data []byte
	var filename string
	var mimeType string

	switch req.Format {
	case FormatCSV:
		data, err = g.findingsToCSV(findings)
		filename = fmt.Sprintf("findings_%s.csv", time.Now().Format("20060102_150405"))
		mimeType = "text/csv"
	case FormatPDF:
		data, err = g.findingsToPDF(findings, req.Title)
		filename = fmt.Sprintf("findings_%s.pdf", time.Now().Format("20060102_150405"))
		mimeType = "application/pdf"
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}

	if err != nil {
		return nil, err
	}

	return &Report{
		Type:        req.Type,
		Format:      req.Format,
		Title:       req.Title,
		GeneratedAt: time.Now(),
		Data:        data,
		Filename:    filename,
		MimeType:    mimeType,
	}, nil
}

// findingsToCSV converts findings to CSV
func (g *Generator) findingsToCSV(findings []*models.Finding) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Header
	header := []string{
		"ID", "Title", "Description", "Severity", "Category", "Status",
		"Asset ID", "Remediation", "Created At", "Updated At",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}

	// Data rows
	for _, f := range findings {
		row := []string{
			f.ID,
			f.Title,
			f.Description,
			string(f.Severity),
			string(f.Category),
			string(f.Status),
			f.AssetID,
			f.Remediation,
			f.CreatedAt.Format(time.RFC3339),
			f.UpdatedAt.Format(time.RFC3339),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

// findingsToPDF generates a PDF report for findings
func (g *Generator) findingsToPDF(findings []*models.Finding, title string) ([]byte, error) {
	pdf := NewPDFReport(title)

	// Summary section
	pdf.AddSection("Summary")
	summary := map[string]int{
		"Critical": 0, "High": 0, "Medium": 0, "Low": 0,
	}
	for _, f := range findings {
		summary[string(f.Severity)]++
	}
	pdf.AddSummaryTable(summary)

	// Findings table
	pdf.AddSection("Findings Detail")
	headers := []string{"ID", "Title", "Severity", "Category", "Status"}
	rows := make([][]string, len(findings))
	for i, f := range findings {
		rows[i] = []string{
			f.ID[:8] + "...",
			truncate(f.Title, 40),
			string(f.Severity),
			string(f.Category),
			string(f.Status),
		}
	}
	pdf.AddTable(headers, rows)

	return pdf.Output()
}

// generateAssetsReport generates an assets report
func (g *Generator) generateAssetsReport(ctx context.Context, req *ReportRequest) (*Report, error) {
	assets, err := g.provider.GetAssets(ctx, AssetsFilter{
		AccountIDs: req.AccountIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assets: %w", err)
	}

	var data []byte
	var filename string
	var mimeType string

	switch req.Format {
	case FormatCSV:
		data, err = g.assetsToCSV(assets)
		filename = fmt.Sprintf("assets_%s.csv", time.Now().Format("20060102_150405"))
		mimeType = "text/csv"
	case FormatPDF:
		data, err = g.assetsToPDF(assets, req.Title)
		filename = fmt.Sprintf("assets_%s.pdf", time.Now().Format("20060102_150405"))
		mimeType = "application/pdf"
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}

	if err != nil {
		return nil, err
	}

	return &Report{
		Type:        req.Type,
		Format:      req.Format,
		Title:       req.Title,
		GeneratedAt: time.Now(),
		Data:        data,
		Filename:    filename,
		MimeType:    mimeType,
	}, nil
}

// assetsToCSV converts assets to CSV
func (g *Generator) assetsToCSV(assets []*models.DataAsset) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	header := []string{
		"ID", "Name", "Type", "Provider", "Account ID", "Region",
		"Max Sensitivity", "Last Scanned", "Created At",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}

	for _, a := range assets {
		lastScanned := ""
		if a.LastScanned != nil {
			lastScanned = a.LastScanned.Format(time.RFC3339)
		}
		row := []string{
			a.ID,
			a.Name,
			string(a.AssetType),
			string(a.Provider),
			a.AccountID,
			a.Region,
			string(a.Classification.MaxSensitivity),
			lastScanned,
			a.CreatedAt.Format(time.RFC3339),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

// assetsToPDF generates a PDF report for assets
func (g *Generator) assetsToPDF(assets []*models.DataAsset, title string) ([]byte, error) {
	pdf := NewPDFReport(title)

	pdf.AddSection("Asset Inventory")

	headers := []string{"Name", "Type", "Provider", "Region", "Sensitivity"}
	rows := make([][]string, len(assets))
	for i, a := range assets {
		rows[i] = []string{
			truncate(a.Name, 30),
			string(a.AssetType),
			string(a.Provider),
			a.Region,
			string(a.Classification.MaxSensitivity),
		}
	}
	pdf.AddTable(headers, rows)

	return pdf.Output()
}

// generateClassificationReport generates a classification report
func (g *Generator) generateClassificationReport(ctx context.Context, req *ReportRequest) (*Report, error) {
	stats, err := g.provider.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}

	var data []byte
	var filename string
	var mimeType string

	switch req.Format {
	case FormatCSV:
		data, err = g.classificationToCSV(stats)
		filename = fmt.Sprintf("classification_%s.csv", time.Now().Format("20060102_150405"))
		mimeType = "text/csv"
	case FormatPDF:
		data, err = g.classificationToPDF(stats, req.Title)
		filename = fmt.Sprintf("classification_%s.pdf", time.Now().Format("20060102_150405"))
		mimeType = "application/pdf"
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}

	if err != nil {
		return nil, err
	}

	return &Report{
		Type:        req.Type,
		Format:      req.Format,
		Title:       req.Title,
		GeneratedAt: time.Now(),
		Data:        data,
		Filename:    filename,
		MimeType:    mimeType,
	}, nil
}

// classificationToCSV converts classification stats to CSV
func (g *Generator) classificationToCSV(stats *Stats) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write([]string{"Classification Summary"}); err != nil {
		return nil, err
	}
	if err := w.Write([]string{""}); err != nil {
		return nil, err
	}

	if err := w.Write([]string{"Category", "Count"}); err != nil {
		return nil, err
	}
	for cat, count := range stats.ClassificationCounts {
		if err := w.Write([]string{string(cat), fmt.Sprintf("%d", count)}); err != nil {
			return nil, err
		}
	}

	if err := w.Write([]string{""}); err != nil {
		return nil, err
	}
	if err := w.Write([]string{"Sensitivity", "Count"}); err != nil {
		return nil, err
	}
	for sens, count := range stats.SensitivityCounts {
		if err := w.Write([]string{string(sens), fmt.Sprintf("%d", count)}); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

// classificationToPDF generates a PDF classification report
func (g *Generator) classificationToPDF(stats *Stats, title string) ([]byte, error) {
	pdf := NewPDFReport(title)

	pdf.AddSection("Classification by Category")
	catHeaders := []string{"Category", "Count"}
	catRows := make([][]string, 0, len(stats.ClassificationCounts))
	for cat, count := range stats.ClassificationCounts {
		catRows = append(catRows, []string{string(cat), fmt.Sprintf("%d", count)})
	}
	pdf.AddTable(catHeaders, catRows)

	pdf.AddSection("Classification by Sensitivity")
	sensHeaders := []string{"Sensitivity", "Count"}
	sensRows := make([][]string, 0, len(stats.SensitivityCounts))
	for sens, count := range stats.SensitivityCounts {
		sensRows = append(sensRows, []string{string(sens), fmt.Sprintf("%d", count)})
	}
	pdf.AddTable(sensHeaders, sensRows)

	return pdf.Output()
}

// generateExecutiveReport generates an executive summary report
func (g *Generator) generateExecutiveReport(ctx context.Context, req *ReportRequest) (*Report, error) {
	stats, err := g.provider.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}

	accounts, err := g.provider.GetAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	var data []byte
	var filename string
	var mimeType string

	switch req.Format {
	case FormatCSV:
		data, err = g.executiveToCSV(stats, accounts)
		filename = fmt.Sprintf("executive_%s.csv", time.Now().Format("20060102_150405"))
		mimeType = "text/csv"
	case FormatPDF:
		data, err = g.executiveToPDF(stats, accounts, req.Title)
		filename = fmt.Sprintf("executive_%s.pdf", time.Now().Format("20060102_150405"))
		mimeType = "application/pdf"
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}

	if err != nil {
		return nil, err
	}

	return &Report{
		Type:        req.Type,
		Format:      req.Format,
		Title:       req.Title,
		GeneratedAt: time.Now(),
		Data:        data,
		Filename:    filename,
		MimeType:    mimeType,
	}, nil
}

// executiveToCSV generates executive summary CSV
func (g *Generator) executiveToCSV(stats *Stats, accounts []*models.CloudAccount) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	w.Write([]string{"Executive Summary Report"})
	w.Write([]string{"Generated", time.Now().Format(time.RFC1123)})
	w.Write([]string{""})

	w.Write([]string{"Metric", "Value"})
	w.Write([]string{"Total Accounts", fmt.Sprintf("%d", stats.TotalAccounts)})
	w.Write([]string{"Total Assets", fmt.Sprintf("%d", stats.TotalAssets)})
	w.Write([]string{"Total Findings", fmt.Sprintf("%d", stats.TotalFindings)})
	w.Write([]string{"Critical Findings", fmt.Sprintf("%d", stats.CriticalFindings)})
	w.Write([]string{"High Findings", fmt.Sprintf("%d", stats.HighFindings)})
	w.Write([]string{"Open Findings", fmt.Sprintf("%d", stats.OpenFindings)})
	w.Write([]string{"Resolved Findings", fmt.Sprintf("%d", stats.ResolvedFindings)})

	w.Flush()
	return buf.Bytes(), w.Error()
}

// executiveToPDF generates executive summary PDF
func (g *Generator) executiveToPDF(stats *Stats, accounts []*models.CloudAccount, title string) ([]byte, error) {
	pdf := NewPDFReport(title)

	pdf.AddSection("Executive Summary")
	pdf.AddParagraph(fmt.Sprintf("Report generated on %s", time.Now().Format(time.RFC1123)))

	// Key metrics
	pdf.AddSection("Key Metrics")
	pdf.AddSummaryTable(map[string]int{
		"Total Accounts":    stats.TotalAccounts,
		"Total Assets":      stats.TotalAssets,
		"Total Findings":    stats.TotalFindings,
		"Critical Findings": stats.CriticalFindings,
		"High Findings":     stats.HighFindings,
		"Open Findings":     stats.OpenFindings,
	})

	// Findings by severity
	pdf.AddSection("Findings by Severity")
	pdf.AddSummaryTable(map[string]int{
		"Critical": stats.CriticalFindings,
		"High":     stats.HighFindings,
		"Medium":   stats.MediumFindings,
		"Low":      stats.LowFindings,
	})

	// Cloud accounts
	pdf.AddSection("Cloud Accounts")
	accHeaders := []string{"Name", "Provider", "Status"}
	accRows := make([][]string, len(accounts))
	for i, a := range accounts {
		accRows[i] = []string{a.Name, string(a.Provider), string(a.Status)}
	}
	pdf.AddTable(accHeaders, accRows)

	return pdf.Output()
}

// generateComplianceReport generates a compliance report
func (g *Generator) generateComplianceReport(ctx context.Context, req *ReportRequest) (*Report, error) {
	findings, err := g.provider.GetFindings(ctx, FindingsFilter{
		AccountIDs: req.AccountIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch findings: %w", err)
	}

	// Group findings by compliance framework
	complianceMap := g.mapFindingsToCompliance(findings)

	var data []byte
	var filename string
	var mimeType string

	switch req.Format {
	case FormatCSV:
		data, err = g.complianceToCSV(complianceMap)
		filename = fmt.Sprintf("compliance_%s.csv", time.Now().Format("20060102_150405"))
		mimeType = "text/csv"
	case FormatPDF:
		data, err = g.complianceToPDF(complianceMap, req.Title)
		filename = fmt.Sprintf("compliance_%s.pdf", time.Now().Format("20060102_150405"))
		mimeType = "application/pdf"
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}

	if err != nil {
		return nil, err
	}

	return &Report{
		Type:        req.Type,
		Format:      req.Format,
		Title:       req.Title,
		GeneratedAt: time.Now(),
		Data:        data,
		Filename:    filename,
		MimeType:    mimeType,
	}, nil
}

// ComplianceStatus represents compliance status for a framework
type ComplianceStatus struct {
	Framework    string
	TotalChecks  int
	PassedChecks int
	FailedChecks int
	Findings     []*models.Finding
}

// mapFindingsToCompliance maps findings to compliance frameworks
func (g *Generator) mapFindingsToCompliance(findings []*models.Finding) map[string]*ComplianceStatus {
	frameworks := map[string]*ComplianceStatus{
		"GDPR":     {Framework: "GDPR", TotalChecks: 10},
		"HIPAA":    {Framework: "HIPAA", TotalChecks: 8},
		"PCI-DSS":  {Framework: "PCI-DSS", TotalChecks: 12},
		"SOC2":     {Framework: "SOC2", TotalChecks: 15},
	}

	for _, f := range findings {
		switch f.Category {
		case models.CategoryPII:
			frameworks["GDPR"].Findings = append(frameworks["GDPR"].Findings, f)
			frameworks["GDPR"].FailedChecks++
		case models.CategoryPHI:
			frameworks["HIPAA"].Findings = append(frameworks["HIPAA"].Findings, f)
			frameworks["HIPAA"].FailedChecks++
		case models.CategoryPCI:
			frameworks["PCI-DSS"].Findings = append(frameworks["PCI-DSS"].Findings, f)
			frameworks["PCI-DSS"].FailedChecks++
		case models.CategorySecrets:
			frameworks["SOC2"].Findings = append(frameworks["SOC2"].Findings, f)
			frameworks["SOC2"].FailedChecks++
		}
	}

	// Calculate passed checks
	for _, status := range frameworks {
		status.PassedChecks = status.TotalChecks - status.FailedChecks
		if status.PassedChecks < 0 {
			status.PassedChecks = 0
		}
	}

	return frameworks
}

// complianceToCSV generates compliance CSV
func (g *Generator) complianceToCSV(complianceMap map[string]*ComplianceStatus) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	w.Write([]string{"Compliance Report"})
	w.Write([]string{""})
	w.Write([]string{"Framework", "Total Checks", "Passed", "Failed", "Compliance %"})

	for _, status := range complianceMap {
		compliancePct := float64(status.PassedChecks) / float64(status.TotalChecks) * 100
		w.Write([]string{
			status.Framework,
			fmt.Sprintf("%d", status.TotalChecks),
			fmt.Sprintf("%d", status.PassedChecks),
			fmt.Sprintf("%d", status.FailedChecks),
			fmt.Sprintf("%.1f%%", compliancePct),
		})
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

// complianceToPDF generates compliance PDF
func (g *Generator) complianceToPDF(complianceMap map[string]*ComplianceStatus, title string) ([]byte, error) {
	pdf := NewPDFReport(title)

	pdf.AddSection("Compliance Overview")

	headers := []string{"Framework", "Passed", "Failed", "Compliance"}
	rows := make([][]string, 0, len(complianceMap))
	for _, status := range complianceMap {
		compliancePct := float64(status.PassedChecks) / float64(status.TotalChecks) * 100
		rows = append(rows, []string{
			status.Framework,
			fmt.Sprintf("%d/%d", status.PassedChecks, status.TotalChecks),
			fmt.Sprintf("%d", status.FailedChecks),
			fmt.Sprintf("%.0f%%", compliancePct),
		})
	}
	pdf.AddTable(headers, rows)

	// Details per framework
	for _, status := range complianceMap {
		if len(status.Findings) > 0 {
			pdf.AddSection(fmt.Sprintf("%s Findings", status.Framework))
			findHeaders := []string{"Title", "Severity", "Status"}
			findRows := make([][]string, len(status.Findings))
			for i, f := range status.Findings {
				findRows[i] = []string{truncate(f.Title, 40), string(f.Severity), string(f.Status)}
			}
			pdf.AddTable(findHeaders, findRows)
		}
	}

	return pdf.Output()
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// StreamCSV streams CSV data to a writer
func (g *Generator) StreamCSV(ctx context.Context, w io.Writer, req *ReportRequest) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	switch req.Type {
	case ReportTypeFindings:
		findings, err := g.provider.GetFindings(ctx, FindingsFilter{
			AccountIDs: req.AccountIDs,
			Severities: req.Severities,
			Categories: req.Categories,
			Statuses:   req.Statuses,
			DateFrom:   req.DateFrom,
			DateTo:     req.DateTo,
		})
		if err != nil {
			return err
		}

		header := []string{"ID", "Title", "Severity", "Category", "Status", "Asset ID", "Created At"}
		if err := csvWriter.Write(header); err != nil {
			return err
		}

		for _, f := range findings {
			row := []string{
				f.ID, f.Title, string(f.Severity), string(f.Category),
				string(f.Status), f.AssetID, f.CreatedAt.Format(time.RFC3339),
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}

	case ReportTypeAssets:
		assets, err := g.provider.GetAssets(ctx, AssetsFilter{AccountIDs: req.AccountIDs})
		if err != nil {
			return err
		}

		header := []string{"ID", "Name", "Type", "Provider", "Region", "Sensitivity"}
		if err := csvWriter.Write(header); err != nil {
			return err
		}

		for _, a := range assets {
			row := []string{
				a.ID, a.Name, string(a.AssetType), string(a.Provider),
				a.Region, string(a.Classification.MaxSensitivity),
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}
