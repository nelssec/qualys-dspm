package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// StringArray is an alias for pq.StringArray to handle PostgreSQL arrays
type StringArray = pq.StringArray

type Provider string

const (
	ProviderAWS   Provider = "AWS"
	ProviderAzure Provider = "AZURE"
	ProviderGCP   Provider = "GCP"
)

type Sensitivity string

const (
	SensitivityCritical Sensitivity = "CRITICAL"
	SensitivityHigh     Sensitivity = "HIGH"
	SensitivityMedium   Sensitivity = "MEDIUM"
	SensitivityLow      Sensitivity = "LOW"
	SensitivityUnknown  Sensitivity = "UNKNOWN"
)

type Category string

const (
	CategoryPII     Category = "PII"
	CategoryPHI     Category = "PHI"
	CategoryPCI     Category = "PCI"
	CategorySecrets Category = "SECRETS"
	CategoryCustom  Category = "CUSTOM"
)

type ResourceType string

const (
	ResourceTypeS3Bucket      ResourceType = "s3_bucket"
	ResourceTypeAzureBlob     ResourceType = "azure_blob_container"
	ResourceTypeGCSBucket     ResourceType = "gcs_bucket"
	ResourceTypeLambda        ResourceType = "lambda_function"
	ResourceTypeAzureFunction ResourceType = "azure_function"
	ResourceTypeCloudFunction ResourceType = "cloud_function"
	ResourceTypeRDS           ResourceType = "rds_instance"
	ResourceTypeDynamoDB      ResourceType = "dynamodb_table"
	ResourceTypeAzureSQL      ResourceType = "azure_sql_database"
	ResourceTypeCloudSQL      ResourceType = "cloud_sql_instance"
	ResourceTypeBigQuery      ResourceType = "bigquery_dataset"
)

type EncryptionStatus string

const (
	EncryptionNone   EncryptionStatus = "NONE"
	EncryptionSSE    EncryptionStatus = "SSE"
	EncryptionSSEKMS EncryptionStatus = "SSE_KMS"
	EncryptionCMK    EncryptionStatus = "CMK"
)

type PermissionLevel string

const (
	PermissionRead  PermissionLevel = "READ"
	PermissionWrite PermissionLevel = "WRITE"
	PermissionAdmin PermissionLevel = "ADMIN"
	PermissionFull  PermissionLevel = "FULL"
)

type FindingSeverity string

const (
	SeverityCritical FindingSeverity = "CRITICAL"
	SeverityHigh     FindingSeverity = "HIGH"
	SeverityMedium   FindingSeverity = "MEDIUM"
	SeverityLow      FindingSeverity = "LOW"
	SeverityInfo     FindingSeverity = "INFO"
)

type FindingStatus string

const (
	FindingStatusOpen          FindingStatus = "open"
	FindingStatusInProgress    FindingStatus = "in_progress"
	FindingStatusResolved      FindingStatus = "resolved"
	FindingStatusSuppressed    FindingStatus = "suppressed"
	FindingStatusFalsePositive FindingStatus = "false_positive"
)

type ScanStatus string

const (
	ScanStatusPending   ScanStatus = "pending"
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusFailed    ScanStatus = "failed"
	ScanStatusCancelled ScanStatus = "cancelled"
)

type ScanType string

const (
	ScanTypeFull           ScanType = "FULL"
	ScanTypeIncremental    ScanType = "INCREMENTAL"
	ScanTypeAssetDiscovery ScanType = "ASSET_DISCOVERY"
	ScanTypeClassification ScanType = "CLASSIFICATION"
	ScanTypeAccessAnalysis ScanType = "ACCESS_ANALYSIS"
)

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

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

type CloudAccount struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	Provider        Provider   `json:"provider" db:"provider"`
	ExternalID      string     `json:"external_id" db:"external_id"`
	DisplayName     string     `json:"display_name" db:"display_name"`
	ConnectorConfig JSONB      `json:"connector_config" db:"connector_config"`
	Status          string     `json:"status" db:"status"`
	StatusMessage   string     `json:"status_message,omitempty" db:"status_message"`
	LastScanAt      *time.Time `json:"last_scan_at,omitempty" db:"last_scan_at"`
	LastScanStatus  string     `json:"last_scan_status,omitempty" db:"last_scan_status"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

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
	DataCategories      StringArray         `json:"data_categories" db:"data_categories"`
	ClassificationCount int              `json:"classification_count" db:"classification_count"`
	LastScannedAt       *time.Time       `json:"last_scanned_at,omitempty" db:"last_scanned_at"`
	LastAccessedAt      *time.Time       `json:"last_accessed_at,omitempty" db:"last_accessed_at"`
	ScanStatus          string           `json:"scan_status" db:"scan_status"`
	ScanError           string           `json:"scan_error,omitempty" db:"scan_error"`
	CreatedAt           time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at" db:"updated_at"`
}

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
	ComplianceFrameworks StringArray     `json:"compliance_frameworks" db:"compliance_frameworks"`
	Evidence             JSONB           `json:"evidence,omitempty" db:"evidence"`
	ResourceSnapshot     JSONB           `json:"resource_snapshot,omitempty" db:"resource_snapshot"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`
	ResolvedAt           *time.Time      `json:"resolved_at,omitempty" db:"resolved_at"`
	FirstSeenAt          time.Time       `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt           time.Time       `json:"last_seen_at" db:"last_seen_at"`
}

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

type AccountStatus string

const (
	AccountStatusActive   AccountStatus = "active"
	AccountStatusInactive AccountStatus = "inactive"
	AccountStatusError    AccountStatus = "error"
)

type AssetType = ResourceType

type ClassificationSummary struct {
	MaxSensitivity Sensitivity    `json:"max_sensitivity"`
	Categories     []Category     `json:"categories"`
	TotalFindings  int            `json:"total_findings"`
	BySensitivity  map[string]int `json:"by_sensitivity,omitempty"`
	ByCategory     map[string]int `json:"by_category,omitempty"`
}

// =====================================================
// Phase 2: Data Lineage Types
// =====================================================

type FlowType string

const (
	FlowReadsFrom    FlowType = "READS_FROM"
	FlowWritesTo     FlowType = "WRITES_TO"
	FlowExportsTo    FlowType = "EXPORTS_TO"
	FlowReplicatesTo FlowType = "REPLICATES_TO"
)

type InferenceSource string

const (
	InferIAMPolicy   InferenceSource = "IAM_POLICY"
	InferEnvVariable InferenceSource = "ENV_VARIABLE"
	InferEventSource InferenceSource = "EVENT_SOURCE"
	InferCloudTrail  InferenceSource = "CLOUDTRAIL"
)

type LineageEvent struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	AccountID          uuid.UUID       `json:"account_id" db:"account_id"`
	SourceResourceARN  string          `json:"source_resource_arn" db:"source_resource_arn"`
	SourceResourceType string          `json:"source_resource_type" db:"source_resource_type"`
	SourceResourceName string          `json:"source_resource_name" db:"source_resource_name"`
	TargetResourceARN  string          `json:"target_resource_arn" db:"target_resource_arn"`
	TargetResourceType string          `json:"target_resource_type" db:"target_resource_type"`
	TargetResourceName string          `json:"target_resource_name" db:"target_resource_name"`
	FlowType           FlowType        `json:"flow_type" db:"flow_type"`
	AccessMethod       string          `json:"access_method" db:"access_method"`
	Frequency          string          `json:"frequency" db:"frequency"`
	DataVolumeBytes    int64           `json:"data_volume_bytes" db:"data_volume_bytes"`
	InferredFrom       InferenceSource `json:"inferred_from" db:"inferred_from"`
	ConfidenceScore    float64         `json:"confidence_score" db:"confidence_score"`
	Evidence           JSONB           `json:"evidence" db:"evidence"`
	FirstObservedAt    time.Time       `json:"first_observed_at" db:"first_observed_at"`
	LastObservedAt     time.Time       `json:"last_observed_at" db:"last_observed_at"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
}

type LineagePath struct {
	ID                    uuid.UUID   `json:"id" db:"id"`
	AccountID             uuid.UUID   `json:"account_id" db:"account_id"`
	OriginARN             string      `json:"origin_arn" db:"origin_arn"`
	OriginType            string      `json:"origin_type" db:"origin_type"`
	DestinationARN        string      `json:"destination_arn" db:"destination_arn"`
	DestinationType       string      `json:"destination_type" db:"destination_type"`
	PathHops              int         `json:"path_hops" db:"path_hops"`
	PathARNs              []string    `json:"path_arns" db:"path_arns"`
	FlowTypes             []string    `json:"flow_types" db:"flow_types"`
	ContainsSensitiveData bool        `json:"contains_sensitive_data" db:"contains_sensitive_data"`
	SensitivityLevel      Sensitivity `json:"sensitivity_level" db:"sensitivity_level"`
	DataCategories        []string    `json:"data_categories" db:"data_categories"`
	RiskScore             int         `json:"risk_score" db:"risk_score"`
	ComputedAt            time.Time   `json:"computed_at" db:"computed_at"`
}

// =====================================================
// Phase 2: ML Classification Types
// =====================================================

type MLModelType string

const (
	MLModelNER                MLModelType = "NER"
	MLModelDocumentClassifier MLModelType = "DOCUMENT_CLASSIFIER"
	MLModelConfidenceScorer   MLModelType = "CONFIDENCE_SCORER"
)

type MLModelStatus string

const (
	MLModelStatusActive     MLModelStatus = "active"
	MLModelStatusTraining   MLModelStatus = "training"
	MLModelStatusDeprecated MLModelStatus = "deprecated"
)

type ReviewStatus string

const (
	ReviewStatusPending  ReviewStatus = "pending"
	ReviewStatusApproved ReviewStatus = "approved"
	ReviewStatusRejected ReviewStatus = "rejected"
)

type ReviewQueueStatus string

const (
	ReviewQueueStatusPending  ReviewQueueStatus = "pending"
	ReviewQueueStatusInReview ReviewQueueStatus = "in_review"
	ReviewQueueStatusResolved ReviewQueueStatus = "resolved"
	ReviewQueueStatusSkipped  ReviewQueueStatus = "skipped"
)

type MLModel struct {
	ID                  uuid.UUID     `json:"id" db:"id"`
	Name                string        `json:"name" db:"name"`
	ModelType           MLModelType   `json:"model_type" db:"model_type"`
	Version             string        `json:"version" db:"version"`
	Description         string        `json:"description" db:"description"`
	Framework           string        `json:"framework" db:"framework"`
	ModelPath           string        `json:"model_path" db:"model_path"`
	Config              JSONB         `json:"config" db:"config"`
	Accuracy            float64       `json:"accuracy" db:"accuracy"`
	PrecisionScore      float64       `json:"precision_score" db:"precision_score"`
	RecallScore         float64       `json:"recall_score" db:"recall_score"`
	F1Score             float64       `json:"f1_score" db:"f1_score"`
	Status              MLModelStatus `json:"status" db:"status"`
	IsDefault           bool          `json:"is_default" db:"is_default"`
	TrainedOnSamples    int           `json:"trained_on_samples" db:"trained_on_samples"`
	TrainingDataVersion string        `json:"training_data_version" db:"training_data_version"`
	CreatedAt           time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at" db:"updated_at"`
}

type MLPrediction struct {
	ID                uuid.UUID    `json:"id" db:"id"`
	ClassificationID  uuid.UUID    `json:"classification_id" db:"classification_id"`
	ModelID           uuid.UUID    `json:"model_id" db:"model_id"`
	PredictionType    string       `json:"prediction_type" db:"prediction_type"`
	PredictedLabel    string       `json:"predicted_label" db:"predicted_label"`
	ConfidenceScore   float64      `json:"confidence_score" db:"confidence_score"`
	EntityText        string       `json:"entity_text" db:"entity_text"`
	EntityStartOffset int          `json:"entity_start_offset" db:"entity_start_offset"`
	EntityEndOffset   int          `json:"entity_end_offset" db:"entity_end_offset"`
	RawOutput         JSONB        `json:"raw_output" db:"raw_output"`
	ReviewStatus      ReviewStatus `json:"review_status" db:"review_status"`
	ReviewedBy        *uuid.UUID   `json:"reviewed_by" db:"reviewed_by"`
	ReviewedAt        *time.Time   `json:"reviewed_at" db:"reviewed_at"`
	ReviewNotes       string       `json:"review_notes" db:"review_notes"`
	CreatedAt         time.Time    `json:"created_at" db:"created_at"`
}

type ClassificationReviewQueue struct {
	ID                 uuid.UUID         `json:"id" db:"id"`
	ClassificationID   uuid.UUID         `json:"classification_id" db:"classification_id"`
	PredictionID       *uuid.UUID        `json:"prediction_id" db:"prediction_id"`
	Priority           int               `json:"priority" db:"priority"`
	Reason             string            `json:"reason" db:"reason"`
	OriginalConfidence float64           `json:"original_confidence" db:"original_confidence"`
	AssignedTo         *uuid.UUID        `json:"assigned_to" db:"assigned_to"`
	AssignedAt         *time.Time        `json:"assigned_at" db:"assigned_at"`
	DueBy              *time.Time        `json:"due_by" db:"due_by"`
	Status             ReviewQueueStatus `json:"status" db:"status"`
	ResolvedAt         *time.Time        `json:"resolved_at" db:"resolved_at"`
	Resolution         string            `json:"resolution" db:"resolution"`
	FinalLabel         string            `json:"final_label" db:"final_label"`
	FinalConfidence    float64           `json:"final_confidence" db:"final_confidence"`
	CreatedAt          time.Time         `json:"created_at" db:"created_at"`
}

type TrainingFeedback struct {
	ID                     uuid.UUID  `json:"id" db:"id"`
	ModelID                *uuid.UUID `json:"model_id" db:"model_id"`
	PredictionID           *uuid.UUID `json:"prediction_id" db:"prediction_id"`
	OriginalPrediction     string     `json:"original_prediction" db:"original_prediction"`
	CorrectedLabel         string     `json:"corrected_label" db:"corrected_label"`
	FeedbackType           string     `json:"feedback_type" db:"feedback_type"`
	SampleContent          string     `json:"sample_content" db:"sample_content"`
	SampleHash             string     `json:"sample_hash" db:"sample_hash"`
	ContextWindow          string     `json:"context_window" db:"context_window"`
	IncorporatedInTraining bool       `json:"incorporated_in_training" db:"incorporated_in_training"`
	TrainingRunID          string     `json:"training_run_id" db:"training_run_id"`
	SubmittedBy            *uuid.UUID `json:"submitted_by" db:"submitted_by"`
	SubmittedAt            time.Time  `json:"submitted_at" db:"submitted_at"`
}

// =====================================================
// Phase 2: AI Source Tracking Types
// =====================================================

type AIServiceType string

const (
	AIServiceSageMaker AIServiceType = "SAGEMAKER"
	AIServiceBedrock   AIServiceType = "BEDROCK"
	AIServiceVertexAI  AIServiceType = "VERTEX_AI"
	AIServiceAzureML   AIServiceType = "AZURE_ML"
)

type AIModelType string

const (
	AIModelTypeTraining   AIModelType = "TRAINING"
	AIModelTypeInference  AIModelType = "INFERENCE"
	AIModelTypeFoundation AIModelType = "FOUNDATION"
)

type AIEventType string

const (
	AIEventTrainingJob      AIEventType = "TRAINING_JOB"
	AIEventInferenceRequest AIEventType = "INFERENCE_REQUEST"
	AIEventDataFetch        AIEventType = "DATA_FETCH"
	AIEventBatchTransform   AIEventType = "BATCH_TRANSFORM"
)

type AIService struct {
	ID            uuid.UUID     `json:"id" db:"id"`
	AccountID     uuid.UUID     `json:"account_id" db:"account_id"`
	ServiceType   AIServiceType `json:"service_type" db:"service_type"`
	ServiceARN    string        `json:"service_arn" db:"service_arn"`
	ServiceName   string        `json:"service_name" db:"service_name"`
	Region        string        `json:"region" db:"region"`
	ServiceConfig JSONB         `json:"service_config" db:"service_config"`
	Status        string        `json:"status" db:"status"`
	CreatedAt     time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at" db:"updated_at"`
}

type AIModel struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	ServiceID   *uuid.UUID  `json:"service_id" db:"service_id"`
	AccountID   uuid.UUID   `json:"account_id" db:"account_id"`
	ModelARN    string      `json:"model_arn" db:"model_arn"`
	ModelName   string      `json:"model_name" db:"model_name"`
	ModelType   AIModelType `json:"model_type" db:"model_type"`
	Framework   string      `json:"framework" db:"framework"`
	Version     string      `json:"version" db:"version"`
	Description string      `json:"description" db:"description"`
	Status      string      `json:"status" db:"status"`
	EndpointARN string      `json:"endpoint_arn" db:"endpoint_arn"`
	Tags        JSONB       `json:"tags" db:"tags"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

type AITrainingData struct {
	ID                    uuid.UUID   `json:"id" db:"id"`
	ModelID               uuid.UUID   `json:"model_id" db:"model_id"`
	DataSourceARN         string      `json:"data_source_arn" db:"data_source_arn"`
	DataSourceType        string      `json:"data_source_type" db:"data_source_type"`
	DataPath              string      `json:"data_path" db:"data_path"`
	DataFormat            string      `json:"data_format" db:"data_format"`
	SampleCount           int64       `json:"sample_count" db:"sample_count"`
	DataSizeBytes         int64       `json:"data_size_bytes" db:"data_size_bytes"`
	ContainsSensitiveData bool        `json:"contains_sensitive_data" db:"contains_sensitive_data"`
	SensitivityCategories []string    `json:"sensitivity_categories" db:"sensitivity_categories"`
	SensitivityLevel      Sensitivity `json:"sensitivity_level" db:"sensitivity_level"`
	UsedAt                time.Time   `json:"used_at" db:"used_at"`
	DiscoveredAt          time.Time   `json:"discovered_at" db:"discovered_at"`
}

type AIProcessingEvent struct {
	ID                      uuid.UUID   `json:"id" db:"id"`
	AccountID               uuid.UUID   `json:"account_id" db:"account_id"`
	ServiceID               *uuid.UUID  `json:"service_id" db:"service_id"`
	ModelID                 *uuid.UUID  `json:"model_id" db:"model_id"`
	EventType               AIEventType `json:"event_type" db:"event_type"`
	EventTime               time.Time   `json:"event_time" db:"event_time"`
	DataSourceARN           string      `json:"data_source_arn" db:"data_source_arn"`
	DataAssetID             *uuid.UUID  `json:"data_asset_id" db:"data_asset_id"`
	AccessedSensitivityLevel Sensitivity `json:"accessed_sensitivity_level" db:"accessed_sensitivity_level"`
	AccessedCategories      []string    `json:"accessed_categories" db:"accessed_categories"`
	DataVolumeBytes         int64       `json:"data_volume_bytes" db:"data_volume_bytes"`
	RecordCount             int         `json:"record_count" db:"record_count"`
	PrincipalARN            string      `json:"principal_arn" db:"principal_arn"`
	PrincipalType           string      `json:"principal_type" db:"principal_type"`
	EventDetails            JSONB       `json:"event_details" db:"event_details"`
	RiskScore               int         `json:"risk_score" db:"risk_score"`
	RiskFactors             []string    `json:"risk_factors" db:"risk_factors"`
	CreatedAt               time.Time   `json:"created_at" db:"created_at"`
}

// =====================================================
// Phase 2: Enhanced Encryption Visibility Types
// =====================================================

type KeyType string

const (
	KeyTypeSymmetric  KeyType = "SYMMETRIC"
	KeyTypeAsymmetric KeyType = "ASYMMETRIC"
)

type KeyUsageType string

const (
	KeyUsageEncryptDecrypt KeyUsageType = "ENCRYPT_DECRYPT"
	KeyUsageSignVerify     KeyUsageType = "SIGN_VERIFY"
)

type KeyState string

const (
	KeyStateEnabled         KeyState = "Enabled"
	KeyStateDisabled        KeyState = "Disabled"
	KeyStatePendingDeletion KeyState = "PendingDeletion"
	KeyStatePendingImport   KeyState = "PendingImport"
	KeyStateUnavailable     KeyState = "Unavailable"
)

type EncryptionUsageType string

const (
	EncryptionUsageBucket EncryptionUsageType = "BUCKET_ENCRYPTION"
	EncryptionUsageEBS    EncryptionUsageType = "EBS_ENCRYPTION"
	EncryptionUsageRDS    EncryptionUsageType = "RDS_ENCRYPTION"
	EncryptionUsageLambda EncryptionUsageType = "LAMBDA_ENV"
)

type EncryptionKey struct {
	ID                     uuid.UUID    `json:"id" db:"id"`
	AccountID              uuid.UUID    `json:"account_id" db:"account_id"`
	KeyID                  string       `json:"key_id" db:"key_id"`
	KeyARN                 string       `json:"key_arn" db:"key_arn"`
	Alias                  string       `json:"alias" db:"alias"`
	Description            string       `json:"description" db:"description"`
	KeyType                KeyType      `json:"key_type" db:"key_type"`
	KeyUsage               KeyUsageType `json:"key_usage" db:"key_usage"`
	KeySpec                string       `json:"key_spec" db:"key_spec"`
	KeyManager             string       `json:"key_manager" db:"key_manager"`
	Origin                 string       `json:"origin" db:"origin"`
	KeyState               KeyState     `json:"key_state" db:"key_state"`
	Enabled                bool         `json:"enabled" db:"enabled"`
	RotationEnabled        bool         `json:"rotation_enabled" db:"rotation_enabled"`
	LastRotatedAt          *time.Time   `json:"last_rotated_at" db:"last_rotated_at"`
	NextRotationAt         *time.Time   `json:"next_rotation_at" db:"next_rotation_at"`
	RotationPeriodDays     int          `json:"rotation_period_days" db:"rotation_period_days"`
	DeletionDate           *time.Time   `json:"deletion_date" db:"deletion_date"`
	PendingDeletionDays    int          `json:"pending_deletion_days" db:"pending_deletion_days"`
	KeyPolicy              JSONB        `json:"key_policy" db:"key_policy"`
	AllowsPublicAccess     bool         `json:"allows_public_access" db:"allows_public_access"`
	AllowsCrossAccount     bool         `json:"allows_cross_account" db:"allows_cross_account"`
	CrossAccountPrincipals []string     `json:"cross_account_principals" db:"cross_account_principals"`
	Tags                   JSONB        `json:"tags" db:"tags"`
	Region                 string       `json:"region" db:"region"`
	CreatedAt              time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time    `json:"updated_at" db:"updated_at"`
	DiscoveredAt           time.Time    `json:"discovered_at" db:"discovered_at"`
}

type EncryptionKeyUsage struct {
	ID                uuid.UUID           `json:"id" db:"id"`
	KeyID             uuid.UUID           `json:"key_id" db:"key_id"`
	AssetID           *uuid.UUID          `json:"asset_id" db:"asset_id"`
	AssetARN          string              `json:"asset_arn" db:"asset_arn"`
	AssetType         string              `json:"asset_type" db:"asset_type"`
	UsageType         EncryptionUsageType `json:"usage_type" db:"usage_type"`
	EncryptionContext JSONB               `json:"encryption_context" db:"encryption_context"`
	FirstSeenAt       time.Time           `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt        time.Time           `json:"last_seen_at" db:"last_seen_at"`
}

type TransitEncryption struct {
	ID                           uuid.UUID  `json:"id" db:"id"`
	AssetID                      uuid.UUID  `json:"asset_id" db:"asset_id"`
	EndpointType                 string     `json:"endpoint_type" db:"endpoint_type"`
	EndpointURL                  string     `json:"endpoint_url" db:"endpoint_url"`
	TLSEnabled                   bool       `json:"tls_enabled" db:"tls_enabled"`
	TLSVersion                   string     `json:"tls_version" db:"tls_version"`
	MinTLSVersion                string     `json:"min_tls_version" db:"min_tls_version"`
	CertificateARN               string     `json:"certificate_arn" db:"certificate_arn"`
	CertificateExpiry            *time.Time `json:"certificate_expiry" db:"certificate_expiry"`
	CipherSuites                 []string   `json:"cipher_suites" db:"cipher_suites"`
	SupportsPerfectForwardSecrecy bool       `json:"supports_perfect_forward_secrecy" db:"supports_perfect_forward_secrecy"`
	MeetsMinimumStandards        bool       `json:"meets_minimum_standards" db:"meets_minimum_standards"`
	ComplianceIssues             []string   `json:"compliance_issues" db:"compliance_issues"`
	LastCheckedAt                time.Time  `json:"last_checked_at" db:"last_checked_at"`
}

type EncryptionCompliance struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	AccountID          *uuid.UUID `json:"account_id" db:"account_id"`
	AssetID            *uuid.UUID `json:"asset_id" db:"asset_id"`
	ComplianceScore    int        `json:"compliance_score" db:"compliance_score"`
	Grade              string     `json:"grade" db:"grade"`
	AtRestScore        int        `json:"at_rest_score" db:"at_rest_score"`
	InTransitScore     int        `json:"in_transit_score" db:"in_transit_score"`
	KeyManagementScore int        `json:"key_management_score" db:"key_management_score"`
	FindingsCount      int        `json:"findings_count" db:"findings_count"`
	CriticalFindings   int        `json:"critical_findings" db:"critical_findings"`
	ComplianceDetails  JSONB      `json:"compliance_details" db:"compliance_details"`
	Recommendations    []string   `json:"recommendations" db:"recommendations"`
	EvaluatedAt        time.Time  `json:"evaluated_at" db:"evaluated_at"`
}

// ConfidenceThresholds defines thresholds for ML classification review
type ConfidenceThresholds struct {
	AutoApprove   float64 `json:"auto_approve"`
	RequireReview float64 `json:"require_review"`
	AutoReject    float64 `json:"auto_reject"`
}

// DefaultConfidenceThresholds returns the default thresholds
func DefaultConfidenceThresholds() ConfidenceThresholds {
	return ConfidenceThresholds{
		AutoApprove:   0.85,
		RequireReview: 0.50,
		AutoReject:    0.20,
	}
}
