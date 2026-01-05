package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Provider represents a cloud provider
type Provider string

const (
	ProviderAWS   Provider = "AWS"
	ProviderAzure Provider = "AZURE"
	ProviderGCP   Provider = "GCP"
)

// Sensitivity levels for data classification
type Sensitivity string

const (
	SensitivityCritical Sensitivity = "CRITICAL"
	SensitivityHigh     Sensitivity = "HIGH"
	SensitivityMedium   Sensitivity = "MEDIUM"
	SensitivityLow      Sensitivity = "LOW"
	SensitivityUnknown  Sensitivity = "UNKNOWN"
)

// Category represents data classification categories
type Category string

const (
	CategoryPII     Category = "PII"
	CategoryPHI     Category = "PHI"
	CategoryPCI     Category = "PCI"
	CategorySecrets Category = "SECRETS"
	CategoryCustom  Category = "CUSTOM"
)

// ResourceType represents types of cloud resources
type ResourceType string

const (
	ResourceTypeS3Bucket        ResourceType = "s3_bucket"
	ResourceTypeAzureBlob       ResourceType = "azure_blob_container"
	ResourceTypeGCSBucket       ResourceType = "gcs_bucket"
	ResourceTypeLambda          ResourceType = "lambda_function"
	ResourceTypeAzureFunction   ResourceType = "azure_function"
	ResourceTypeCloudFunction   ResourceType = "cloud_function"
	ResourceTypeRDS             ResourceType = "rds_instance"
	ResourceTypeDynamoDB        ResourceType = "dynamodb_table"
	ResourceTypeAzureSQL        ResourceType = "azure_sql_database"
	ResourceTypeCloudSQL        ResourceType = "cloud_sql_instance"
	ResourceTypeBigQuery        ResourceType = "bigquery_dataset"
)

// EncryptionStatus represents the encryption state of a resource
type EncryptionStatus string

const (
	EncryptionNone   EncryptionStatus = "NONE"
	EncryptionSSE    EncryptionStatus = "SSE"      // Server-side encryption (provider managed)
	EncryptionSSEKMS EncryptionStatus = "SSE_KMS"  // Server-side with KMS
	EncryptionCMK    EncryptionStatus = "CMK"      // Customer managed key
)

// PermissionLevel represents access permission levels
type PermissionLevel string

const (
	PermissionRead  PermissionLevel = "READ"
	PermissionWrite PermissionLevel = "WRITE"
	PermissionAdmin PermissionLevel = "ADMIN"
	PermissionFull  PermissionLevel = "FULL"
)

// FindingSeverity represents the severity of a security finding
type FindingSeverity string

const (
	SeverityCritical FindingSeverity = "CRITICAL"
	SeverityHigh     FindingSeverity = "HIGH"
	SeverityMedium   FindingSeverity = "MEDIUM"
	SeverityLow      FindingSeverity = "LOW"
	SeverityInfo     FindingSeverity = "INFO"
)

// FindingStatus represents the status of a finding
type FindingStatus string

const (
	FindingStatusOpen          FindingStatus = "open"
	FindingStatusInProgress    FindingStatus = "in_progress"
	FindingStatusResolved      FindingStatus = "resolved"
	FindingStatusSuppressed    FindingStatus = "suppressed"
	FindingStatusFalsePositive FindingStatus = "false_positive"
)

// ScanStatus represents the status of a scan job
type ScanStatus string

const (
	ScanStatusPending   ScanStatus = "pending"
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusFailed    ScanStatus = "failed"
	ScanStatusCancelled ScanStatus = "cancelled"
)

// ScanType represents types of scans
type ScanType string

const (
	ScanTypeFull           ScanType = "FULL"
	ScanTypeIncremental    ScanType = "INCREMENTAL"
	ScanTypeAssetDiscovery ScanType = "ASSET_DISCOVERY"
	ScanTypeClassification ScanType = "CLASSIFICATION"
	ScanTypeAccessAnalysis ScanType = "ACCESS_ANALYSIS"
)

// JSONB is a wrapper type for JSONB columns
type JSONB map[string]interface{}

// Value implements driver.Valuer
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

// CloudAccount represents a connected cloud account
type CloudAccount struct {
	ID              uuid.UUID `json:"id" db:"id"`
	Provider        Provider  `json:"provider" db:"provider"`
	ExternalID      string    `json:"external_id" db:"external_id"`
	DisplayName     string    `json:"display_name" db:"display_name"`
	ConnectorConfig JSONB     `json:"connector_config" db:"connector_config"`
	Status          string    `json:"status" db:"status"`
	StatusMessage   string    `json:"status_message,omitempty" db:"status_message"`
	LastScanAt      *time.Time `json:"last_scan_at,omitempty" db:"last_scan_at"`
	LastScanStatus  string    `json:"last_scan_status,omitempty" db:"last_scan_status"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// DataAsset represents a discovered data asset (bucket, database, etc.)
type DataAsset struct {
	ID                  uuid.UUID        `json:"id" db:"id"`
	AccountID           uuid.UUID        `json:"account_id" db:"account_id"`
	ResourceType        ResourceType     `json:"resource_type" db:"resource_type"`
	ResourceARN         string           `json:"resource_arn" db:"resource_arn"`
	Region              string           `json:"region" db:"region"`
	Name                string           `json:"name" db:"name"`
	EncryptionStatus    EncryptionStatus `json:"encryption_status" db:"encryption_status"`
	EncryptionKeyARN    string           `json:"encryption_key_arn,omitempty" db:"encryption_key_arn"`
	PublicAccess        bool             `json:"public_access" db:"public_access"`
	PublicAccessDetails JSONB            `json:"public_access_details,omitempty" db:"public_access_details"`
	VersioningEnabled   bool             `json:"versioning_enabled" db:"versioning_enabled"`
	LoggingEnabled      bool             `json:"logging_enabled" db:"logging_enabled"`
	SizeBytes           int64            `json:"size_bytes" db:"size_bytes"`
	ObjectCount         int              `json:"object_count" db:"object_count"`
	Tags                JSONB            `json:"tags" db:"tags"`
	RawMetadata         JSONB            `json:"raw_metadata" db:"raw_metadata"`
	SensitivityLevel    Sensitivity      `json:"sensitivity_level" db:"sensitivity_level"`
	DataCategories      []string         `json:"data_categories" db:"data_categories"`
	ClassificationCount int              `json:"classification_count" db:"classification_count"`
	LastScannedAt       *time.Time       `json:"last_scanned_at,omitempty" db:"last_scanned_at"`
	ScanStatus          string           `json:"scan_status" db:"scan_status"`
	ScanError           string           `json:"scan_error,omitempty" db:"scan_error"`
	CreatedAt           time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at" db:"updated_at"`
}

// Classification represents a data classification finding
type Classification struct {
	ID              uuid.UUID   `json:"id" db:"id"`
	AssetID         uuid.UUID   `json:"asset_id" db:"asset_id"`
	ObjectPath      string      `json:"object_path" db:"object_path"`
	ObjectSize      int64       `json:"object_size" db:"object_size"`
	RuleName        string      `json:"rule_name" db:"rule_name"`
	Category        Category    `json:"category" db:"category"`
	Sensitivity     Sensitivity `json:"sensitivity" db:"sensitivity"`
	FindingCount    int         `json:"finding_count" db:"finding_count"`
	SampleMatches   JSONB       `json:"sample_matches,omitempty" db:"sample_matches"`
	MatchLocations  JSONB       `json:"match_locations,omitempty" db:"match_locations"`
	ConfidenceScore float64     `json:"confidence_score" db:"confidence_score"`
	Validated       bool        `json:"validated" db:"validated"`
	DiscoveredAt    time.Time   `json:"discovered_at" db:"discovered_at"`
}

// AccessPolicy represents an IAM policy
type AccessPolicy struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	AccountID          uuid.UUID `json:"account_id" db:"account_id"`
	PolicyARN          string    `json:"policy_arn" db:"policy_arn"`
	PolicyName         string    `json:"policy_name" db:"policy_name"`
	PolicyType         string    `json:"policy_type" db:"policy_type"`
	PolicyDocument     JSONB     `json:"policy_document" db:"policy_document"`
	PolicyVersion      string    `json:"policy_version" db:"policy_version"`
	AttachedTo         JSONB     `json:"attached_to" db:"attached_to"`
	AllowsPublicAccess bool      `json:"allows_public_access" db:"allows_public_access"`
	OverlyPermissive   bool      `json:"overly_permissive" db:"overly_permissive"`
	AnalysisNotes      string    `json:"analysis_notes,omitempty" db:"analysis_notes"`
	DiscoveredAt       time.Time `json:"discovered_at" db:"discovered_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// AccessEdge represents an access relationship between a principal and an asset
type AccessEdge struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	SourceType      string          `json:"source_type" db:"source_type"`
	SourceARN       string          `json:"source_arn" db:"source_arn"`
	SourceName      string          `json:"source_name" db:"source_name"`
	TargetAssetID   uuid.UUID       `json:"target_asset_id" db:"target_asset_id"`
	TargetARN       string          `json:"target_arn" db:"target_arn"`
	PermissionLevel PermissionLevel `json:"permission_level" db:"permission_level"`
	Permissions     []string        `json:"permissions" db:"permissions"`
	PolicyID        *uuid.UUID      `json:"policy_id,omitempty" db:"policy_id"`
	GrantType       string          `json:"grant_type" db:"grant_type"`
	IsDirect        bool            `json:"is_direct" db:"is_direct"`
	IsPublic        bool            `json:"is_public" db:"is_public"`
	IsCrossAccount  bool            `json:"is_cross_account" db:"is_cross_account"`
	Conditions      JSONB           `json:"conditions,omitempty" db:"conditions"`
	DiscoveredAt    time.Time       `json:"discovered_at" db:"discovered_at"`
}

// Finding represents a security finding or risk
type Finding struct {
	ID                   uuid.UUID       `json:"id" db:"id"`
	AccountID            uuid.UUID       `json:"account_id" db:"account_id"`
	AssetID              *uuid.UUID      `json:"asset_id,omitempty" db:"asset_id"`
	FindingType          string          `json:"finding_type" db:"finding_type"`
	Severity             FindingSeverity `json:"severity" db:"severity"`
	Title                string          `json:"title" db:"title"`
	Description          string          `json:"description" db:"description"`
	Remediation          string          `json:"remediation" db:"remediation"`
	Status               FindingStatus   `json:"status" db:"status"`
	StatusReason         string          `json:"status_reason,omitempty" db:"status_reason"`
	AssignedTo           string          `json:"assigned_to,omitempty" db:"assigned_to"`
	ComplianceFrameworks []string        `json:"compliance_frameworks" db:"compliance_frameworks"`
	Evidence             JSONB           `json:"evidence,omitempty" db:"evidence"`
	ResourceSnapshot     JSONB           `json:"resource_snapshot,omitempty" db:"resource_snapshot"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`
	ResolvedAt           *time.Time      `json:"resolved_at,omitempty" db:"resolved_at"`
	FirstSeenAt          time.Time       `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt           time.Time       `json:"last_seen_at" db:"last_seen_at"`
}

// ScanJob represents a scan job
type ScanJob struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	AccountID            uuid.UUID  `json:"account_id" db:"account_id"`
	ScanType             ScanType   `json:"scan_type" db:"scan_type"`
	ScanScope            JSONB      `json:"scan_scope,omitempty" db:"scan_scope"`
	Status               ScanStatus `json:"status" db:"status"`
	TotalAssets          int        `json:"total_assets" db:"total_assets"`
	ScannedAssets        int        `json:"scanned_assets" db:"scanned_assets"`
	FindingsCount        int        `json:"findings_count" db:"findings_count"`
	ClassificationsCount int        `json:"classifications_count" db:"classifications_count"`
	Errors               JSONB      `json:"errors,omitempty" db:"errors"`
	ScheduledAt          *time.Time `json:"scheduled_at,omitempty" db:"scheduled_at"`
	StartedAt            *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt          *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	TriggeredBy          string     `json:"triggered_by" db:"triggered_by"`
	WorkerID             string     `json:"worker_id,omitempty" db:"worker_id"`
}
