package anomaly

import (
	"time"

	"github.com/google/uuid"
)

// AnomalyType represents the type of detected anomaly
type AnomalyType string

const (
	AnomalyVolumeSpike     AnomalyType = "VOLUME_SPIKE"
	AnomalyFrequencySpike  AnomalyType = "FREQUENCY_SPIKE"
	AnomalyNewDestination  AnomalyType = "NEW_DESTINATION"
	AnomalyOffHoursAccess  AnomalyType = "OFF_HOURS_ACCESS"
	AnomalyBulkDownload    AnomalyType = "BULK_DOWNLOAD"
	AnomalyUnusualPattern  AnomalyType = "UNUSUAL_PATTERN"
	AnomalyGeoAnomaly      AnomalyType = "GEO_ANOMALY"
	AnomalyPrivilegeEscal  AnomalyType = "PRIVILEGE_ESCALATION"
)

// AnomalyStatus represents the status of an anomaly
type AnomalyStatus string

const (
	StatusNew          AnomalyStatus = "NEW"
	StatusInvestigating AnomalyStatus = "INVESTIGATING"
	StatusConfirmed    AnomalyStatus = "CONFIRMED"
	StatusFalsePositive AnomalyStatus = "FALSE_POSITIVE"
	StatusResolved     AnomalyStatus = "RESOLVED"
)

// SeverityLevel represents the severity of an anomaly
type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "LOW"
	SeverityMedium   SeverityLevel = "MEDIUM"
	SeverityHigh     SeverityLevel = "HIGH"
	SeverityCritical SeverityLevel = "CRITICAL"
)

// Anomaly represents a detected anomaly
type Anomaly struct {
	ID              uuid.UUID              `json:"id"`
	AccountID       uuid.UUID              `json:"account_id"`
	AssetID         *uuid.UUID             `json:"asset_id,omitempty"`
	PrincipalID     string                 `json:"principal_id,omitempty"`
	PrincipalType   string                 `json:"principal_type,omitempty"`
	PrincipalName   string                 `json:"principal_name,omitempty"`
	AnomalyType     AnomalyType            `json:"anomaly_type"`
	Status          AnomalyStatus          `json:"status"`
	Severity        SeverityLevel          `json:"severity"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Details         map[string]interface{} `json:"details"`
	BaselineValue   float64                `json:"baseline_value"`
	ObservedValue   float64                `json:"observed_value"`
	DeviationFactor float64                `json:"deviation_factor"`
	DetectedAt      time.Time              `json:"detected_at"`
	ResolvedAt      *time.Time             `json:"resolved_at,omitempty"`
	ResolvedBy      string                 `json:"resolved_by,omitempty"`
	Resolution      string                 `json:"resolution,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// AccessBaseline represents normal access patterns for a principal
type AccessBaseline struct {
	ID                uuid.UUID         `json:"id"`
	AccountID         uuid.UUID         `json:"account_id"`
	PrincipalID       string            `json:"principal_id"`
	PrincipalType     string            `json:"principal_type"`
	PrincipalName     string            `json:"principal_name"`
	TimeWindow        string            `json:"time_window"` // "HOURLY", "DAILY", "WEEKLY"
	BaselinePeriodStart time.Time       `json:"baseline_period_start"`
	BaselinePeriodEnd   time.Time       `json:"baseline_period_end"`

	// Access patterns
	AvgDailyAccessCount    float64   `json:"avg_daily_access_count"`
	StdDevAccessCount      float64   `json:"std_dev_access_count"`
	AvgDataVolumeBytes     float64   `json:"avg_data_volume_bytes"`
	StdDevDataVolume       float64   `json:"std_dev_data_volume"`
	NormalAccessHours      []int     `json:"normal_access_hours"`
	NormalAccessDays       []int     `json:"normal_access_days"`
	CommonAssets           []string  `json:"common_assets"`
	CommonOperations       []string  `json:"common_operations"`
	CommonSourceIPs        []string  `json:"common_source_ips"`
	CommonGeoLocations     []string  `json:"common_geo_locations"`

	// Thresholds
	AccessCountThreshold   float64   `json:"access_count_threshold"`
	DataVolumeThreshold    float64   `json:"data_volume_threshold"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AccessEvent represents a data access event for analysis
type AccessEvent struct {
	EventID        string    `json:"event_id"`
	AccountID      uuid.UUID `json:"account_id"`
	PrincipalID    string    `json:"principal_id"`
	PrincipalType  string    `json:"principal_type"`
	PrincipalName  string    `json:"principal_name"`
	AssetID        uuid.UUID `json:"asset_id"`
	AssetName      string    `json:"asset_name"`
	Operation      string    `json:"operation"`
	DataVolumeBytes int64    `json:"data_volume_bytes"`
	SourceIP       string    `json:"source_ip"`
	GeoLocation    string    `json:"geo_location"`
	UserAgent      string    `json:"user_agent"`
	Timestamp      time.Time `json:"timestamp"`
	Success        bool      `json:"success"`
	ErrorCode      string    `json:"error_code,omitempty"`
}

// ThreatScore represents an insider threat score for a principal
type ThreatScore struct {
	ID              uuid.UUID                `json:"id"`
	AccountID       uuid.UUID                `json:"account_id"`
	PrincipalID     string                   `json:"principal_id"`
	PrincipalType   string                   `json:"principal_type"`
	PrincipalName   string                   `json:"principal_name"`
	Score           float64                  `json:"score"`            // 0-100
	RiskLevel       SeverityLevel            `json:"risk_level"`
	Factors         []ThreatFactor           `json:"factors"`
	RecentAnomalies int                      `json:"recent_anomalies"`
	TrendDirection  string                   `json:"trend_direction"`  // "UP", "DOWN", "STABLE"
	LastUpdated     time.Time                `json:"last_updated"`
	Details         map[string]interface{}   `json:"details"`
}

// ThreatFactor represents a factor contributing to threat score
type ThreatFactor struct {
	Factor      string  `json:"factor"`
	Weight      float64 `json:"weight"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
}

// DetectionRule represents a rule for anomaly detection
type DetectionRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	AnomalyType AnomalyType            `json:"anomaly_type"`
	Enabled     bool                   `json:"enabled"`
	Severity    SeverityLevel          `json:"severity"`
	Conditions  map[string]interface{} `json:"conditions"`
	Threshold   float64                `json:"threshold"`
}

// AnomalySummary provides a summary of anomalies
type AnomalySummary struct {
	TotalAnomalies    int                  `json:"total_anomalies"`
	NewCount          int                  `json:"new_count"`
	InvestigatingCount int                 `json:"investigating_count"`
	ConfirmedCount    int                  `json:"confirmed_count"`
	BySeverity        map[string]int       `json:"by_severity"`
	ByType            map[string]int       `json:"by_type"`
	RecentAnomalies   []Anomaly            `json:"recent_anomalies"`
	TopPrincipals     []PrincipalAnomaly   `json:"top_principals"`
	TrendData         []TrendDataPoint     `json:"trend_data"`
}

// PrincipalAnomaly represents anomaly stats for a principal
type PrincipalAnomaly struct {
	PrincipalID   string        `json:"principal_id"`
	PrincipalName string        `json:"principal_name"`
	AnomalyCount  int           `json:"anomaly_count"`
	ThreatScore   float64       `json:"threat_score"`
	RiskLevel     SeverityLevel `json:"risk_level"`
}

// TrendDataPoint represents a data point in anomaly trends
type TrendDataPoint struct {
	Date         string `json:"date"`
	AnomalyCount int    `json:"anomaly_count"`
	AvgSeverity  float64 `json:"avg_severity"`
}

// BaselineRequest represents a request to build access baselines
type BaselineRequest struct {
	AccountID     uuid.UUID `json:"account_id"`
	PrincipalID   string    `json:"principal_id,omitempty"`
	DaysToAnalyze int       `json:"days_to_analyze"`
}

// UpdateAnomalyStatusRequest represents a request to update anomaly status
type UpdateAnomalyStatusRequest struct {
	Status     AnomalyStatus `json:"status"`
	Resolution string        `json:"resolution,omitempty"`
	ResolvedBy string        `json:"resolved_by,omitempty"`
}

// AnomalyReport represents a detailed anomaly report
type AnomalyReport struct {
	GeneratedAt       time.Time            `json:"generated_at"`
	ReportPeriod      string               `json:"report_period"`
	AccountID         uuid.UUID            `json:"account_id"`
	Summary           AnomalySummary       `json:"summary"`
	ThreatScores      []ThreatScore        `json:"threat_scores"`
	HighRiskPrincipals []PrincipalAnomaly  `json:"high_risk_principals"`
	CriticalAnomalies  []Anomaly           `json:"critical_anomalies"`
	Recommendations   []string             `json:"recommendations"`
}

// GetDefaultDetectionRules returns the default anomaly detection rules
func GetDefaultDetectionRules() []DetectionRule {
	return []DetectionRule{
		{
			ID:          "volume-spike",
			Name:        "Data Volume Spike",
			Description: "Detects unusual data access volume compared to baseline",
			AnomalyType: AnomalyVolumeSpike,
			Enabled:     true,
			Severity:    SeverityHigh,
			Threshold:   3.0, // 3 standard deviations
			Conditions: map[string]interface{}{
				"min_volume_bytes": 10485760, // 10MB
			},
		},
		{
			ID:          "frequency-spike",
			Name:        "Access Frequency Spike",
			Description: "Detects unusual access frequency compared to baseline",
			AnomalyType: AnomalyFrequencySpike,
			Enabled:     true,
			Severity:    SeverityMedium,
			Threshold:   3.0,
			Conditions: map[string]interface{}{
				"min_access_count": 10,
			},
		},
		{
			ID:          "new-destination",
			Name:        "New Data Destination",
			Description: "Detects data flow to a new or unusual destination",
			AnomalyType: AnomalyNewDestination,
			Enabled:     true,
			Severity:    SeverityMedium,
			Conditions:  map[string]interface{}{},
		},
		{
			ID:          "off-hours-access",
			Name:        "Off-Hours Data Access",
			Description: "Detects data access outside normal working hours",
			AnomalyType: AnomalyOffHoursAccess,
			Enabled:     true,
			Severity:    SeverityLow,
			Conditions: map[string]interface{}{
				"off_hours_start": 22, // 10 PM
				"off_hours_end":   6,  // 6 AM
			},
		},
		{
			ID:          "bulk-download",
			Name:        "Bulk Data Download",
			Description: "Detects large-scale data extraction events",
			AnomalyType: AnomalyBulkDownload,
			Enabled:     true,
			Severity:    SeverityCritical,
			Threshold:   5.0,
			Conditions: map[string]interface{}{
				"min_volume_bytes":     104857600, // 100MB
				"time_window_minutes": 60,
			},
		},
		{
			ID:          "geo-anomaly",
			Name:        "Geographic Anomaly",
			Description: "Detects access from unusual geographic locations",
			AnomalyType: AnomalyGeoAnomaly,
			Enabled:     true,
			Severity:    SeverityHigh,
			Conditions:  map[string]interface{}{},
		},
	}
}
