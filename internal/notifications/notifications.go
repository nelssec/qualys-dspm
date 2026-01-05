package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/qualys/dspm/internal/models"
)

// NotificationType defines the type of notification
type NotificationType string

const (
	NotifyNewFinding      NotificationType = "new_finding"
	NotifyCriticalFinding NotificationType = "critical_finding"
	NotifyScanComplete    NotificationType = "scan_complete"
	NotifyScanFailed      NotificationType = "scan_failed"
	NotifyDailyDigest     NotificationType = "daily_digest"
	NotifyWeeklyReport    NotificationType = "weekly_report"
)

// Channel defines notification channels
type Channel string

const (
	ChannelSlack Channel = "slack"
	ChannelEmail Channel = "email"
)

// Notification represents a notification to be sent
type Notification struct {
	Type      NotificationType
	Title     string
	Message   string
	Severity  models.Sensitivity
	Data      map[string]interface{}
	Timestamp time.Time
}

// Config holds notification configuration
type Config struct {
	Slack SlackConfig
	Email EmailConfig
}

// SlackConfig holds Slack configuration
type SlackConfig struct {
	WebhookURL     string
	Channel        string
	Username       string
	IconEmoji      string
	Enabled        bool
	MinSeverity    models.Sensitivity // Minimum severity to notify
}

// EmailConfig holds email configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	Username     string
	Password     string
	From         string
	To           []string
	Enabled      bool
	MinSeverity  models.Sensitivity
}

// Service handles notifications
type Service struct {
	config  Config
	logger  *slog.Logger
	client  *http.Client
}

// NewService creates a new notification service
func NewService(config Config, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		config: config,
		logger: logger,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send sends a notification to all enabled channels
func (s *Service) Send(ctx context.Context, notif *Notification) error {
	var errs []error

	if s.config.Slack.Enabled && s.shouldNotify(notif.Severity, s.config.Slack.MinSeverity) {
		if err := s.sendSlack(ctx, notif); err != nil {
			errs = append(errs, fmt.Errorf("slack: %w", err))
		}
	}

	if s.config.Email.Enabled && s.shouldNotify(notif.Severity, s.config.Email.MinSeverity) {
		if err := s.sendEmail(ctx, notif); err != nil {
			errs = append(errs, fmt.Errorf("email: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %v", errs)
	}

	return nil
}

// shouldNotify checks if notification should be sent based on severity
func (s *Service) shouldNotify(actual, minimum models.Sensitivity) bool {
	severityOrder := map[models.Sensitivity]int{
		models.SensitivityLow:      1,
		models.SensitivityMedium:   2,
		models.SensitivityHigh:     3,
		models.SensitivityCritical: 4,
	}

	return severityOrder[actual] >= severityOrder[minimum]
}

// SlackMessage represents a Slack message payload
type SlackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack attachment
type SlackAttachment struct {
	Color      string       `json:"color,omitempty"`
	Title      string       `json:"title,omitempty"`
	TitleLink  string       `json:"title_link,omitempty"`
	Text       string       `json:"text,omitempty"`
	Fallback   string       `json:"fallback,omitempty"`
	Fields     []SlackField `json:"fields,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
}

// SlackField represents a field in a Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// sendSlack sends a notification to Slack
func (s *Service) sendSlack(ctx context.Context, notif *Notification) error {
	color := s.severityToColor(notif.Severity)

	fields := []SlackField{}
	if notif.Data != nil {
		if accountID, ok := notif.Data["account_id"].(string); ok {
			fields = append(fields, SlackField{
				Title: "Account",
				Value: accountID,
				Short: true,
			})
		}
		if assetType, ok := notif.Data["asset_type"].(string); ok {
			fields = append(fields, SlackField{
				Title: "Asset Type",
				Value: assetType,
				Short: true,
			})
		}
		if count, ok := notif.Data["finding_count"].(int); ok {
			fields = append(fields, SlackField{
				Title: "Findings",
				Value: fmt.Sprintf("%d", count),
				Short: true,
			})
		}
		if severity, ok := notif.Data["severity"].(string); ok {
			fields = append(fields, SlackField{
				Title: "Severity",
				Value: severity,
				Short: true,
			})
		}
	}

	msg := SlackMessage{
		Channel:   s.config.Slack.Channel,
		Username:  s.config.Slack.Username,
		IconEmoji: s.config.Slack.IconEmoji,
		Attachments: []SlackAttachment{
			{
				Color:     color,
				Title:     notif.Title,
				Text:      notif.Message,
				Fallback:  fmt.Sprintf("%s: %s", notif.Title, notif.Message),
				Fields:    fields,
				Footer:    "DSPM Alert System",
				Timestamp: notif.Timestamp.Unix(),
			},
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.Slack.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	s.logger.Info("slack notification sent",
		"type", notif.Type,
		"title", notif.Title)

	return nil
}

// severityToColor converts severity to Slack color
func (s *Service) severityToColor(severity models.Sensitivity) string {
	switch severity {
	case models.SensitivityCritical:
		return "#FF0000" // Red
	case models.SensitivityHigh:
		return "#FFA500" // Orange
	case models.SensitivityMedium:
		return "#FFFF00" // Yellow
	default:
		return "#36A64F" // Green
	}
}

// sendEmail sends a notification via email
func (s *Service) sendEmail(ctx context.Context, notif *Notification) error {
	subject := fmt.Sprintf("[DSPM Alert] %s", notif.Title)
	body, err := s.formatEmailBody(notif)
	if err != nil {
		return err
	}

	msg := s.buildEmailMessage(subject, body)

	auth := smtp.PlainAuth("", s.config.Email.Username, s.config.Email.Password, s.config.Email.SMTPHost)
	addr := fmt.Sprintf("%s:%d", s.config.Email.SMTPHost, s.config.Email.SMTPPort)

	err = smtp.SendMail(addr, auth, s.config.Email.From, s.config.Email.To, []byte(msg))
	if err != nil {
		return err
	}

	s.logger.Info("email notification sent",
		"type", notif.Type,
		"title", notif.Title,
		"recipients", len(s.config.Email.To))

	return nil
}

// buildEmailMessage builds an email message
func (s *Service) buildEmailMessage(subject, body string) string {
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", s.config.Email.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(s.config.Email.To, ",")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)
	return msg.String()
}

// formatEmailBody formats the email body
func (s *Service) formatEmailBody(notif *Notification) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { padding: 20px; background: {{.HeaderColor}}; color: white; border-radius: 8px 8px 0 0; }
        .content { padding: 20px; }
        .severity { display: inline-block; padding: 4px 8px; border-radius: 4px; font-weight: bold; background: {{.SeverityColor}}; color: white; }
        .data-table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        .data-table td { padding: 8px; border-bottom: 1px solid #eee; }
        .data-table td:first-child { font-weight: bold; width: 30%; }
        .footer { padding: 15px 20px; background: #f9f9f9; border-radius: 0 0 8px 8px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2 style="margin:0;">{{.Title}}</h2>
        </div>
        <div class="content">
            <p>{{.Message}}</p>
            <p>Severity: <span class="severity">{{.Severity}}</span></p>
            {{if .HasData}}
            <table class="data-table">
                {{range $key, $value := .Data}}
                <tr>
                    <td>{{$key}}</td>
                    <td>{{$value}}</td>
                </tr>
                {{end}}
            </table>
            {{end}}
        </div>
        <div class="footer">
            <p>This is an automated alert from the DSPM system.</p>
            <p>Generated at: {{.Timestamp}}</p>
        </div>
    </div>
</body>
</html>
`
	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}

	headerColor := "#2196F3" // Default blue
	severityColor := s.severityToColor(notif.Severity)

	switch notif.Severity {
	case models.SensitivityCritical:
		headerColor = "#F44336"
	case models.SensitivityHigh:
		headerColor = "#FF9800"
	case models.SensitivityMedium:
		headerColor = "#FFC107"
	}

	data := map[string]interface{}{
		"Title":         notif.Title,
		"Message":       notif.Message,
		"Severity":      string(notif.Severity),
		"HeaderColor":   headerColor,
		"SeverityColor": severityColor,
		"Data":          notif.Data,
		"HasData":       len(notif.Data) > 0,
		"Timestamp":     notif.Timestamp.Format(time.RFC1123),
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// NotifyNewFinding sends a notification for a new finding
func (s *Service) NotifyNewFinding(ctx context.Context, finding *models.Finding, asset *models.DataAsset) error {
	notif := &Notification{
		Type:     NotifyNewFinding,
		Title:    fmt.Sprintf("New %s Finding Detected", finding.Severity),
		Message:  finding.Title,
		Severity: finding.Severity,
		Data: map[string]interface{}{
			"finding_id":  finding.ID,
			"asset_id":    asset.ID,
			"asset_name":  asset.Name,
			"asset_type":  string(asset.AssetType),
			"category":    string(finding.Category),
			"severity":    string(finding.Severity),
		},
		Timestamp: time.Now(),
	}

	return s.Send(ctx, notif)
}

// NotifyCriticalFinding sends an immediate notification for critical findings
func (s *Service) NotifyCriticalFinding(ctx context.Context, finding *models.Finding, asset *models.DataAsset) error {
	notif := &Notification{
		Type:     NotifyCriticalFinding,
		Title:    "CRITICAL Security Finding",
		Message:  fmt.Sprintf("Critical finding detected: %s on %s", finding.Title, asset.Name),
		Severity: models.SensitivityCritical,
		Data: map[string]interface{}{
			"finding_id":   finding.ID,
			"asset_id":     asset.ID,
			"asset_name":   asset.Name,
			"description":  finding.Description,
			"remediation":  finding.Remediation,
		},
		Timestamp: time.Now(),
	}

	return s.Send(ctx, notif)
}

// NotifyScanComplete sends a notification when a scan completes
func (s *Service) NotifyScanComplete(ctx context.Context, accountID string, stats ScanStats) error {
	notif := &Notification{
		Type:     NotifyScanComplete,
		Title:    "Scan Completed",
		Message:  fmt.Sprintf("Scan completed for account %s", accountID),
		Severity: s.statsToSeverity(stats),
		Data: map[string]interface{}{
			"account_id":       accountID,
			"assets_scanned":   stats.AssetsScanned,
			"findings_total":   stats.TotalFindings,
			"findings_critical": stats.CriticalFindings,
			"findings_high":    stats.HighFindings,
			"duration":         stats.Duration.String(),
		},
		Timestamp: time.Now(),
	}

	return s.Send(ctx, notif)
}

// ScanStats holds scan statistics
type ScanStats struct {
	AssetsScanned    int
	TotalFindings    int
	CriticalFindings int
	HighFindings     int
	MediumFindings   int
	LowFindings      int
	Duration         time.Duration
}

// statsToSeverity determines notification severity from scan stats
func (s *Service) statsToSeverity(stats ScanStats) models.Sensitivity {
	if stats.CriticalFindings > 0 {
		return models.SensitivityCritical
	}
	if stats.HighFindings > 0 {
		return models.SensitivityHigh
	}
	if stats.MediumFindings > 0 {
		return models.SensitivityMedium
	}
	return models.SensitivityLow
}

// NotifyScanFailed sends a notification when a scan fails
func (s *Service) NotifyScanFailed(ctx context.Context, accountID string, err error) error {
	notif := &Notification{
		Type:     NotifyScanFailed,
		Title:    "Scan Failed",
		Message:  fmt.Sprintf("Scan failed for account %s: %s", accountID, err.Error()),
		Severity: models.SensitivityHigh,
		Data: map[string]interface{}{
			"account_id": accountID,
			"error":      err.Error(),
		},
		Timestamp: time.Now(),
	}

	return s.Send(ctx, notif)
}

// DigestStats holds daily/weekly digest statistics
type DigestStats struct {
	Period           string
	NewFindings      int
	ResolvedFindings int
	CriticalFindings int
	HighFindings     int
	AccountsScanned  int
	AssetsScanned    int
	TopCategories    map[string]int
}

// NotifyDailyDigest sends a daily digest notification
func (s *Service) NotifyDailyDigest(ctx context.Context, stats DigestStats) error {
	notif := &Notification{
		Type:     NotifyDailyDigest,
		Title:    "Daily Security Digest",
		Message:  fmt.Sprintf("Summary: %d new findings, %d resolved", stats.NewFindings, stats.ResolvedFindings),
		Severity: s.digestToSeverity(stats),
		Data: map[string]interface{}{
			"period":            stats.Period,
			"new_findings":      stats.NewFindings,
			"resolved_findings": stats.ResolvedFindings,
			"critical_findings": stats.CriticalFindings,
			"high_findings":     stats.HighFindings,
			"accounts_scanned":  stats.AccountsScanned,
			"assets_scanned":    stats.AssetsScanned,
		},
		Timestamp: time.Now(),
	}

	return s.Send(ctx, notif)
}

// digestToSeverity determines notification severity from digest stats
func (s *Service) digestToSeverity(stats DigestStats) models.Sensitivity {
	if stats.CriticalFindings > 0 {
		return models.SensitivityCritical
	}
	if stats.HighFindings > 5 {
		return models.SensitivityHigh
	}
	if stats.NewFindings > 10 {
		return models.SensitivityMedium
	}
	return models.SensitivityLow
}
