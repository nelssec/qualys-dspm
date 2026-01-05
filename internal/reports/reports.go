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

type ReportType string

const (
	ReportTypeFindings       ReportType = "findings"
	ReportTypeAssets         ReportType = "assets"
	ReportTypeCompliance     ReportType = "compliance"
	ReportTypeExecutive      ReportType = "executive"
	ReportTypeClassification ReportType = "classification"
)

type ReportFormat string

const (
	FormatCSV  ReportFormat = "csv"
	FormatPDF  ReportFormat = "pdf"
	FormatJSON ReportFormat = "json"
)

type ReportRequest struct {
	Type       ReportType
	Format     ReportFormat
	Title      string
	AccountIDs []string
	DateFrom   *time.Time
	DateTo     *time.Time
	Severities []string
	Categories []string
	Statuses   []string
}

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

type ReportFinding struct {
	ID          string
	Title       string
	Description string
	Severity    string
	Category    string
	Status      string
	AssetID     string
	Remediation string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ReportAsset struct {
	ID            string
	Name          string
	AssetType     string
	Provider      string
	AccountID     string
	Region        string
	Sensitivity   string
	LastScannedAt *time.Time
	CreatedAt     time.Time
}

type ReportAccount struct {
	ID       string
	Name     string
	Provider string
	Status   string
}

type DataProvider interface {
	GetFindings(ctx context.Context, filters FindingsFilter) ([]*ReportFinding, error)
	GetAssets(ctx context.Context, filters AssetsFilter) ([]*ReportAsset, error)
	GetAccounts(ctx context.Context) ([]*ReportAccount, error)
	GetStats(ctx context.Context) (*Stats, error)
}

type FindingsFilter struct {
	AccountIDs []string
	Severities []string
	Categories []string
	Statuses   []string
	DateFrom   *time.Time
	DateTo     *time.Time
}

type AssetsFilter struct {
	AccountIDs    []string
	AssetTypes    []string
	Sensitivities []string
}

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
	ClassificationCounts map[string]int
	SensitivityCounts    map[string]int
}

type Generator struct {
	provider DataProvider
}

func NewGenerator(provider DataProvider) *Generator {
	return &Generator{provider: provider}
}

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

func (g *Generator) findingsToCSV(findings []*ReportFinding) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	header := []string{
		"ID", "Title", "Description", "Severity", "Category", "Status",
		"Asset ID", "Remediation", "Created At", "Updated At",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}

	for _, f := range findings {
		row := []string{
			f.ID,
			f.Title,
			f.Description,
			f.Severity,
			f.Category,
			f.Status,
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

func (g *Generator) findingsToPDF(findings []*ReportFinding, title string) ([]byte, error) {
	pdf := NewPDFReport(title)

	pdf.AddSection("Summary")
	summary := map[string]int{
		"Critical": 0, "High": 0, "Medium": 0, "Low": 0,
	}
	for _, f := range findings {
		summary[f.Severity]++
	}
	pdf.AddSummaryTable(summary)

	pdf.AddSection("Findings Detail")
	headers := []string{"ID", "Title", "Severity", "Category", "Status"}
	rows := make([][]string, len(findings))
	for i, f := range findings {
		idShort := f.ID
		if len(idShort) > 8 {
			idShort = idShort[:8] + "..."
		}
		rows[i] = []string{
			idShort,
			truncate(f.Title, 40),
			f.Severity,
			f.Category,
			f.Status,
		}
	}
	pdf.AddTable(headers, rows)

	return pdf.Output()
}

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

func (g *Generator) assetsToCSV(assets []*ReportAsset) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	header := []string{
		"ID", "Name", "Type", "Provider", "Account ID", "Region",
		"Sensitivity", "Last Scanned", "Created At",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}

	for _, a := range assets {
		lastScanned := ""
		if a.LastScannedAt != nil {
			lastScanned = a.LastScannedAt.Format(time.RFC3339)
		}
		row := []string{
			a.ID,
			a.Name,
			a.AssetType,
			a.Provider,
			a.AccountID,
			a.Region,
			a.Sensitivity,
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

func (g *Generator) assetsToPDF(assets []*ReportAsset, title string) ([]byte, error) {
	pdf := NewPDFReport(title)

	pdf.AddSection("Asset Inventory")

	headers := []string{"Name", "Type", "Provider", "Region", "Sensitivity"}
	rows := make([][]string, len(assets))
	for i, a := range assets {
		rows[i] = []string{
			truncate(a.Name, 30),
			a.AssetType,
			a.Provider,
			a.Region,
			a.Sensitivity,
		}
	}
	pdf.AddTable(headers, rows)

	return pdf.Output()
}

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
		if err := w.Write([]string{cat, fmt.Sprintf("%d", count)}); err != nil {
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
		if err := w.Write([]string{sens, fmt.Sprintf("%d", count)}); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

func (g *Generator) classificationToPDF(stats *Stats, title string) ([]byte, error) {
	pdf := NewPDFReport(title)

	pdf.AddSection("Classification by Category")
	catHeaders := []string{"Category", "Count"}
	catRows := make([][]string, 0, len(stats.ClassificationCounts))
	for cat, count := range stats.ClassificationCounts {
		catRows = append(catRows, []string{cat, fmt.Sprintf("%d", count)})
	}
	pdf.AddTable(catHeaders, catRows)

	pdf.AddSection("Classification by Sensitivity")
	sensHeaders := []string{"Sensitivity", "Count"}
	sensRows := make([][]string, 0, len(stats.SensitivityCounts))
	for sens, count := range stats.SensitivityCounts {
		sensRows = append(sensRows, []string{sens, fmt.Sprintf("%d", count)})
	}
	pdf.AddTable(sensHeaders, sensRows)

	return pdf.Output()
}

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

func (g *Generator) executiveToCSV(stats *Stats, accounts []*ReportAccount) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	_ = w.Write([]string{"Executive Summary Report"})
	_ = w.Write([]string{"Generated", time.Now().Format(time.RFC1123)})
	_ = w.Write([]string{""})

	_ = w.Write([]string{"Metric", "Value"})
	_ = w.Write([]string{"Total Accounts", fmt.Sprintf("%d", stats.TotalAccounts)})
	_ = w.Write([]string{"Total Assets", fmt.Sprintf("%d", stats.TotalAssets)})
	_ = w.Write([]string{"Total Findings", fmt.Sprintf("%d", stats.TotalFindings)})
	_ = w.Write([]string{"Critical Findings", fmt.Sprintf("%d", stats.CriticalFindings)})
	_ = w.Write([]string{"High Findings", fmt.Sprintf("%d", stats.HighFindings)})
	_ = w.Write([]string{"Open Findings", fmt.Sprintf("%d", stats.OpenFindings)})
	_ = w.Write([]string{"Resolved Findings", fmt.Sprintf("%d", stats.ResolvedFindings)})

	w.Flush()
	return buf.Bytes(), w.Error()
}

func (g *Generator) executiveToPDF(stats *Stats, accounts []*ReportAccount, title string) ([]byte, error) {
	pdf := NewPDFReport(title)

	pdf.AddSection("Executive Summary")
	pdf.AddParagraph(fmt.Sprintf("Report generated on %s", time.Now().Format(time.RFC1123)))

	pdf.AddSection("Key Metrics")
	pdf.AddSummaryTable(map[string]int{
		"Total Accounts":    stats.TotalAccounts,
		"Total Assets":      stats.TotalAssets,
		"Total Findings":    stats.TotalFindings,
		"Critical Findings": stats.CriticalFindings,
		"High Findings":     stats.HighFindings,
		"Open Findings":     stats.OpenFindings,
	})

	pdf.AddSection("Findings by Severity")
	pdf.AddSummaryTable(map[string]int{
		"Critical": stats.CriticalFindings,
		"High":     stats.HighFindings,
		"Medium":   stats.MediumFindings,
		"Low":      stats.LowFindings,
	})

	pdf.AddSection("Cloud Accounts")
	accHeaders := []string{"Name", "Provider", "Status"}
	accRows := make([][]string, len(accounts))
	for i, a := range accounts {
		accRows[i] = []string{a.Name, a.Provider, a.Status}
	}
	pdf.AddTable(accHeaders, accRows)

	return pdf.Output()
}

func (g *Generator) generateComplianceReport(ctx context.Context, req *ReportRequest) (*Report, error) {
	findings, err := g.provider.GetFindings(ctx, FindingsFilter{
		AccountIDs: req.AccountIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch findings: %w", err)
	}

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

type ComplianceStatus struct {
	Framework    string
	TotalChecks  int
	PassedChecks int
	FailedChecks int
	Findings     []*ReportFinding
}

func (g *Generator) mapFindingsToCompliance(findings []*ReportFinding) map[string]*ComplianceStatus {
	frameworks := map[string]*ComplianceStatus{
		"GDPR":    {Framework: "GDPR", TotalChecks: 10},
		"HIPAA":   {Framework: "HIPAA", TotalChecks: 8},
		"PCI-DSS": {Framework: "PCI-DSS", TotalChecks: 12},
		"SOC2":    {Framework: "SOC2", TotalChecks: 15},
	}

	for _, f := range findings {
		switch f.Category {
		case string(models.CategoryPII):
			frameworks["GDPR"].Findings = append(frameworks["GDPR"].Findings, f)
			frameworks["GDPR"].FailedChecks++
		case string(models.CategoryPHI):
			frameworks["HIPAA"].Findings = append(frameworks["HIPAA"].Findings, f)
			frameworks["HIPAA"].FailedChecks++
		case string(models.CategoryPCI):
			frameworks["PCI-DSS"].Findings = append(frameworks["PCI-DSS"].Findings, f)
			frameworks["PCI-DSS"].FailedChecks++
		case string(models.CategorySecrets):
			frameworks["SOC2"].Findings = append(frameworks["SOC2"].Findings, f)
			frameworks["SOC2"].FailedChecks++
		}
	}

	for _, status := range frameworks {
		status.PassedChecks = status.TotalChecks - status.FailedChecks
		if status.PassedChecks < 0 {
			status.PassedChecks = 0
		}
	}

	return frameworks
}

func (g *Generator) complianceToCSV(complianceMap map[string]*ComplianceStatus) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	_ = w.Write([]string{"Compliance Report"})
	_ = w.Write([]string{""})
	_ = w.Write([]string{"Framework", "Total Checks", "Passed", "Failed", "Compliance %"})

	for _, status := range complianceMap {
		compliancePct := float64(status.PassedChecks) / float64(status.TotalChecks) * 100
		_ = w.Write([]string{
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

	for _, status := range complianceMap {
		if len(status.Findings) > 0 {
			pdf.AddSection(fmt.Sprintf("%s Findings", status.Framework))
			findHeaders := []string{"Title", "Severity", "Status"}
			findRows := make([][]string, len(status.Findings))
			for i, f := range status.Findings {
				findRows[i] = []string{truncate(f.Title, 40), f.Severity, f.Status}
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
				f.ID, f.Title, f.Severity, f.Category,
				f.Status, f.AssetID, f.CreatedAt.Format(time.RFC3339),
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
				a.ID, a.Name, a.AssetType, a.Provider,
				a.Region, a.Sensitivity,
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}
