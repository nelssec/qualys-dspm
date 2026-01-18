package aitracking

import (
	"time"

	"github.com/google/uuid"
	"github.com/qualys/dspm/internal/models"
)

// AIRiskReport provides a comprehensive risk assessment for AI/ML usage
type AIRiskReport struct {
	AccountID              uuid.UUID           `json:"account_id"`
	TotalAIServices        int                 `json:"total_ai_services"`
	TotalAIModels          int                 `json:"total_ai_models"`
	ModelsAccessingSensitive int              `json:"models_accessing_sensitive"`
	SensitiveTrainingJobs  int                 `json:"sensitive_training_jobs"`
	HighRiskEvents         int                 `json:"high_risk_events"`
	RiskByCategory         map[string]int      `json:"risk_by_category"`
	TopRiskyModels         []ModelRiskSummary  `json:"top_risky_models"`
	Recommendations        []string            `json:"recommendations"`
	GeneratedAt            time.Time           `json:"generated_at"`
}

// ModelRiskSummary provides a risk summary for an AI model
type ModelRiskSummary struct {
	Model            *models.AIModel   `json:"model"`
	RiskScore        int               `json:"risk_score"`
	RiskFactors      []string          `json:"risk_factors"`
	SensitiveDataSources int           `json:"sensitive_data_sources"`
	LastAccessEvent  *time.Time        `json:"last_access_event,omitempty"`
}

// AIServiceOverview provides an overview of AI services in an account
type AIServiceOverview struct {
	AccountID        uuid.UUID          `json:"account_id"`
	ServicesByType   map[string]int     `json:"services_by_type"`
	ModelsByType     map[string]int     `json:"models_by_type"`
	EventsByType     map[string]int     `json:"events_by_type"`
	RecentEvents     []*models.AIProcessingEvent `json:"recent_events"`
	LastUpdated      time.Time          `json:"last_updated"`
}

// TrainingDataAnalysis provides analysis of training data usage
type TrainingDataAnalysis struct {
	ModelID              uuid.UUID   `json:"model_id"`
	TotalDataSources     int         `json:"total_data_sources"`
	SensitiveDataSources int         `json:"sensitive_data_sources"`
	DataCategories       []string    `json:"data_categories"`
	HighestSensitivity   models.Sensitivity `json:"highest_sensitivity"`
	TotalDataSizeBytes   int64       `json:"total_data_size_bytes"`
	DataSources          []*models.AITrainingData `json:"data_sources"`
}

// SensitiveDataAccessSummary summarizes AI access to sensitive data
type SensitiveDataAccessSummary struct {
	TotalEvents        int                 `json:"total_events"`
	EventsBySensitivity map[string]int     `json:"events_by_sensitivity"`
	EventsByCategory   map[string]int      `json:"events_by_category"`
	EventsByModel      map[string]int      `json:"events_by_model"`
	MostAccessedAssets []AssetAccessCount  `json:"most_accessed_assets"`
}

// AssetAccessCount tracks AI access to a specific asset
type AssetAccessCount struct {
	AssetID          uuid.UUID          `json:"asset_id"`
	AssetARN         string             `json:"asset_arn"`
	AssetName        string             `json:"asset_name"`
	SensitivityLevel models.Sensitivity `json:"sensitivity_level"`
	AccessCount      int                `json:"access_count"`
	LastAccessed     time.Time          `json:"last_accessed"`
}

// SageMakerModelInfo contains SageMaker-specific model information
type SageMakerModelInfo struct {
	ModelARN           string
	ModelName          string
	CreationTime       time.Time
	ModelStatus        string
	PrimaryContainer   ContainerInfo
	ExecutionRoleARN   string
	VPCConfig          *VPCConfig
	Tags               map[string]string
}

// ContainerInfo contains model container information
type ContainerInfo struct {
	Image         string
	ModelDataURL  string
	Environment   map[string]string
}

// VPCConfig contains VPC configuration
type VPCConfig struct {
	Subnets        []string
	SecurityGroups []string
}

// TrainingJobInfo contains training job information
type TrainingJobInfo struct {
	TrainingJobARN      string
	TrainingJobName     string
	TrainingJobStatus   string
	CreationTime        time.Time
	TrainingStartTime   *time.Time
	TrainingEndTime     *time.Time
	ModelArtifacts      string
	InputDataConfig     []DataChannelConfig
	OutputDataConfig    OutputDataConfig
	ResourceConfig      ResourceConfig
	RoleARN             string
}

// DataChannelConfig contains data channel configuration for training
type DataChannelConfig struct {
	ChannelName    string
	DataSource     string
	S3DataSource   *S3DataSource
	ContentType    string
	CompressionType string
}

// S3DataSource contains S3 data source configuration
type S3DataSource struct {
	S3URI      string
	S3DataType string
}

// OutputDataConfig contains output data configuration
type OutputDataConfig struct {
	S3OutputPath string
	KMSKeyID     string
}

// ResourceConfig contains training resource configuration
type ResourceConfig struct {
	InstanceType  string
	InstanceCount int
	VolumeSizeGB  int
}

// BedrockModelInfo contains Bedrock-specific model information
type BedrockModelInfo struct {
	ModelID          string
	ModelARN         string
	ModelName        string
	ProviderName     string
	ModelStatus      string
	InputModalities  []string
	OutputModalities []string
}

// RiskFactorWeights defines weights for risk calculation
type RiskFactorWeights struct {
	SensitiveDataAccess   float64
	CriticalDataTraining  float64
	UnencryptedData       float64
	CrossAccountAccess    float64
	HighVolumeAccess      float64
}

// DefaultRiskFactorWeights returns default risk weights
func DefaultRiskFactorWeights() RiskFactorWeights {
	return RiskFactorWeights{
		SensitiveDataAccess:  0.30,
		CriticalDataTraining: 0.25,
		UnencryptedData:      0.20,
		CrossAccountAccess:   0.15,
		HighVolumeAccess:     0.10,
	}
}
