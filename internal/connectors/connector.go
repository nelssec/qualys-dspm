package connectors

import (
	"context"
	"io"
	"time"

	"github.com/qualys/dspm/internal/models"
)

type Connector interface {
	Provider() models.Provider

	Validate(ctx context.Context) error

	Close() error
}

type StorageConnector interface {
	Connector

	ListBuckets(ctx context.Context) ([]BucketInfo, error)

	GetBucketMetadata(ctx context.Context, bucketName string) (*BucketMetadata, error)

	ListObjects(ctx context.Context, bucketName, prefix string, maxKeys int) ([]ObjectInfo, error)

	GetObject(ctx context.Context, bucketName, objectKey string, byteRange *ByteRange) (io.ReadCloser, error)

	GetBucketPolicy(ctx context.Context, bucketName string) (*BucketPolicy, error)

	GetBucketACL(ctx context.Context, bucketName string) (*BucketACL, error)
}

type IAMConnector interface {
	Connector

	ListUsers(ctx context.Context) ([]Principal, error)

	ListRoles(ctx context.Context) ([]Principal, error)

	ListPolicies(ctx context.Context) ([]PolicyInfo, error)

	GetPolicy(ctx context.Context, policyARN string) (*PolicyDocument, error)

	ListAttachedPolicies(ctx context.Context, principalARN string) ([]PolicyInfo, error)

	GetServiceAccounts(ctx context.Context) ([]Principal, error)
}

type ServerlessConnector interface {
	Connector

	ListFunctions(ctx context.Context) ([]FunctionInfo, error)

	GetFunctionConfig(ctx context.Context, functionName string) (*FunctionConfig, error)

	GetFunctionPolicy(ctx context.Context, functionName string) (*PolicyDocument, error)
}

type DatabaseConnector interface {
	Connector

	ListDatabases(ctx context.Context) ([]DatabaseInfo, error)

	GetDatabaseMetadata(ctx context.Context, databaseID string) (*DatabaseMetadata, error)
}

type KMSConnector interface {
	Connector

	ListKeys(ctx context.Context) ([]KeyInfo, error)

	GetKeyMetadata(ctx context.Context, keyID string) (*KeyMetadata, error)

	GetKeyPolicy(ctx context.Context, keyID string) (*PolicyDocument, error)
}

type ByteRange struct {
	Start int64
	End   int64
}

type BucketInfo struct {
	Name      string
	Region    string
	CreatedAt string
	ARN       string
}

type BucketMetadata struct {
	Name              string
	Region            string
	ARN               string
	CreatedAt         string
	Encryption        EncryptionConfig
	Versioning        bool
	Logging           LoggingConfig
	PublicAccessBlock PublicAccessBlockConfig
	Tags              map[string]string
}

type EncryptionConfig struct {
	Enabled   bool
	Type      models.EncryptionStatus
	KeyARN    string
	Algorithm string
}

type LoggingConfig struct {
	Enabled      bool
	TargetBucket string
	TargetPrefix string
}

type PublicAccessBlockConfig struct {
	BlockPublicAcls       bool
	IgnorePublicAcls      bool
	BlockPublicPolicy     bool
	RestrictPublicBuckets bool
}

type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified string
	StorageClass string
	ETag         string
}

type BucketPolicy struct {
	Policy         string
	IsPublic       bool
	PublicActions  []string
	PolicyDocument *PolicyDocument
}

type BucketACL struct {
	Owner  string
	Grants []ACLGrant
}

type ACLGrant struct {
	Grantee     string
	GranteeType string
	Permission  string
	IsPublic    bool
}

type Principal struct {
	ARN         string
	Name        string
	Type        string // USER, ROLE, GROUP, SERVICE
	CreatedAt   string
	Description string
	Tags        map[string]string
}

type PolicyInfo struct {
	ARN         string
	Name        string
	Type        string // MANAGED, INLINE, RESOURCE_BASED
	Description string
	IsAttached  bool
	AttachCount int
}

type PolicyDocument struct {
	Version    string
	Statements []PolicyStatement
	Raw        string
}

type PolicyStatement struct {
	SID        string
	Effect     string
	Principals []string
	Actions    []string
	Resources  []string
	Conditions map[string]interface{}
}

type FunctionInfo struct {
	ARN          string
	Name         string
	Runtime      string
	Handler      string
	Region       string
	MemorySize   int
	Timeout      int
	LastModified string
	Tags         map[string]string
}

type FunctionConfig struct {
	FunctionInfo
	Role              string
	VPCConfig         *VPCConfig
	Environment       map[string]string
	KMSKeyARN         string
	Layers            []string
	FileSystemConfigs []FileSystemConfig
}

type VPCConfig struct {
	SubnetIDs        []string
	SecurityGroupIDs []string
	VPCID            string
}

type FileSystemConfig struct {
	ARN            string
	LocalMountPath string
}

type DatabaseInfo struct {
	ID            string
	ARN           string
	Name          string
	Engine        string
	EngineVersion string
	Status        string
	Region        string
}

type DatabaseMetadata struct {
	DatabaseInfo
	Endpoint           string
	Port               int
	StorageEncrypted   bool
	KMSKeyID           string
	PubliclyAccessible bool
	VPCSecurityGroups  []string
	SubnetGroup        string
	MultiAZ            bool
	Tags               map[string]string
}

type KeyInfo struct {
	ID          string
	ARN         string
	Alias       string
	Description string
	Enabled     bool
	KeyManager  string // AWS, CUSTOMER
}

type KeyMetadata struct {
	KeyInfo
	CreatedAt       string
	KeyState        string
	KeyUsage        string
	Origin          string
	RotationEnabled bool
	DeletionDate    string
	Tags            map[string]string
}

// =====================================================
// Phase 2: Lineage Connector Interface
// =====================================================

// LineageConnector provides data lineage discovery capabilities
type LineageConnector interface {
	Connector

	// GetFunctionEventSources returns event source mappings for a function
	GetFunctionEventSources(ctx context.Context, functionName string) ([]EventSourceMapping, error)

	// GetReplicationConfigurations returns cross-region/cross-account replications
	GetReplicationConfigurations(ctx context.Context, bucketName string) ([]ReplicationConfig, error)

	// GetDataExportConfigs returns data export configurations (analytics, inventory)
	GetDataExportConfigs(ctx context.Context, bucketName string) ([]DataExportConfig, error)
}

// EventSourceMapping represents a Lambda event source mapping
type EventSourceMapping struct {
	UUID             string
	EventSourceARN   string
	FunctionARN      string
	BatchSize        int
	StartingPosition string
	State            string
	StateReason      string
	LastModified     string
}

// ReplicationConfig represents S3 cross-region/cross-account replication
type ReplicationConfig struct {
	ID                 string
	Status             string
	Priority           int
	SourceBucket       string
	DestinationBucket  string
	DestinationAccount string
	DestinationRegion  string
	ReplicaKMSKeyID    string
	StorageClass       string
	FilterPrefix       string
	FilterTags         map[string]string
}

// DataExportConfig represents data export configuration
type DataExportConfig struct {
	ID              string
	ExportType      string // INVENTORY, ANALYTICS
	Status          string
	DestinationARN  string
	DestinationType string
	Frequency       string // DAILY, WEEKLY
	Format          string // CSV, PARQUET, ORC
	Prefix          string
	Encryption      bool
}

// =====================================================
// Phase 2: AI Connector Interface
// =====================================================

// AIConnector provides AI/ML service discovery and tracking
type AIConnector interface {
	Connector

	// SageMaker operations
	ListSageMakerModels(ctx context.Context) ([]SageMakerModel, error)
	GetSageMakerModelDetails(ctx context.Context, modelName string) (*SageMakerModelDetails, error)
	ListTrainingJobs(ctx context.Context) ([]TrainingJob, error)
	GetTrainingJobDetails(ctx context.Context, jobName string) (*TrainingJobDetails, error)
	ListEndpoints(ctx context.Context) ([]SageMakerEndpoint, error)

	// Bedrock operations (AWS)
	ListBedrockModels(ctx context.Context) ([]BedrockModel, error)
	ListModelInvocations(ctx context.Context, modelID string, since string) ([]ModelInvocation, error)
}

// SageMakerModel represents a SageMaker model
type SageMakerModel struct {
	ModelARN         string
	ModelName        string
	CreationTime     string
	ExecutionRoleARN string
	PrimaryContainer ContainerDefinition
	Tags             map[string]string
}

// SageMakerModelDetails contains detailed model information
type SageMakerModelDetails struct {
	SageMakerModel
	VPCConfig            *VPCConfig
	EnableNetworkIsolation bool
	Containers           []ContainerDefinition
}

// ContainerDefinition represents a model container
type ContainerDefinition struct {
	Image            string
	Mode             string
	ModelDataURL     string
	Environment      map[string]string
	ContainerHostname string
}

// TrainingJob represents a SageMaker training job
type TrainingJob struct {
	TrainingJobARN    string
	TrainingJobName   string
	TrainingJobStatus string
	CreationTime      string
	LastModifiedTime  string
	ModelArtifacts    string
	RoleARN           string
}

// TrainingJobDetails contains detailed training job information
type TrainingJobDetails struct {
	TrainingJob
	TrainingStartTime    string
	TrainingEndTime      string
	InputDataConfig      []DataChannelConfig
	OutputDataConfig     OutputDataConfig
	ResourceConfig       ResourceConfig
	StoppingCondition    StoppingCondition
	HyperParameters      map[string]string
	BillableTimeSeconds  int
}

// DataChannelConfig represents training data channel configuration
type DataChannelConfig struct {
	ChannelName     string
	DataSource      string
	S3DataSource    *S3DataSourceConfig
	ContentType     string
	CompressionType string
	RecordWrapper   string
	InputMode       string
}

// S3DataSourceConfig represents S3 data source configuration
type S3DataSourceConfig struct {
	S3URI            string
	S3DataType       string
	S3DataDistribution string
}

// OutputDataConfig represents training output configuration
type OutputDataConfig struct {
	S3OutputPath string
	KMSKeyID     string
}

// ResourceConfig represents training resource configuration
type ResourceConfig struct {
	InstanceType  string
	InstanceCount int
	VolumeSizeGB  int
	VolumeKmsKeyID string
}

// StoppingCondition represents training stopping condition
type StoppingCondition struct {
	MaxRuntimeSeconds      int
	MaxWaitTimeSeconds     int
}

// SageMakerEndpoint represents a SageMaker endpoint
type SageMakerEndpoint struct {
	EndpointARN        string
	EndpointName       string
	EndpointStatus     string
	EndpointConfigName string
	CreationTime       string
	LastModifiedTime   string
	Tags               map[string]string
}

// BedrockModel represents a Bedrock foundation model
type BedrockModel struct {
	ModelID          string
	ModelARN         string
	ModelName        string
	ProviderName     string
	ModelStatus      string
	InputModalities  []string
	OutputModalities []string
	CustomizationsSupported []string
	InferenceTypesSupported []string
}

// ModelInvocation represents a model invocation event
type ModelInvocation struct {
	InvocationID     string
	ModelID          string
	Timestamp        string
	InputTokenCount  int
	OutputTokenCount int
	Status           string
	ErrorCode        string
}

// =====================================================
// Phase 2: CloudTrail Data Access Events
// =====================================================

// CloudTrailConnector provides CloudTrail event access
type CloudTrailConnector interface {
	Connector

	// LookupEvents searches for CloudTrail events
	LookupEvents(ctx context.Context, startTime, endTime time.Time, eventName string) ([]CloudTrailEvent, error)

	// GetEventsByResource retrieves events for a specific resource
	GetEventsByResource(ctx context.Context, resourceARN string, since time.Time) ([]CloudTrailEvent, error)

	// GetS3DataAccessEvents retrieves S3 data access events
	GetS3DataAccessEvents(ctx context.Context, bucketName string, since time.Time) ([]CloudTrailEvent, error)
}

// CloudTrailEvent represents a CloudTrail event
type CloudTrailEvent struct {
	EventID        string
	EventName      string
	EventSource    string
	EventTime      time.Time
	Username       string
	AccessKeyID    string
	ReadOnly       string
	CloudTrailJSON string
	Resources      []CloudTrailResource
}

// CloudTrailResource represents a resource in a CloudTrail event
type CloudTrailResource struct {
	ResourceType string
	ResourceName string
}

// =====================================================
// Phase 2: Enhanced KMS Types
// =====================================================

// KeyGrant represents a KMS key grant
type KeyGrant struct {
	GrantID           string
	KeyID             string
	GranteePrincipal  string
	RetiringPrincipal string
	IssuingAccount    string
	Name              string
	Operations        []string
	Constraints       map[string]string
	CreationDate      string
}

// KeyRotationInfo represents KMS key rotation information
type KeyRotationInfo struct {
	KeyID              string
	RotationEnabled    bool
	RotationPeriodDays int
	LastRotatedDate    string
	NextRotationDate   string
	KeyManager         string
	KeyState           string
	CreationDate       string
}

// KeyAlias represents a KMS key alias
type KeyAlias struct {
	AliasName       string
	AliasARN        string
	TargetKeyID     string
	CreationDate    string
	LastUpdatedDate string
}

// =====================================================
// Phase 2: Enhanced SageMaker Types
// =====================================================

// ProcessingJob represents a SageMaker processing job
type ProcessingJob struct {
	ProcessingJobARN    string
	ProcessingJobName   string
	ProcessingJobStatus string
	CreationTime        string
	ProcessingEndTime   string
	LastModifiedTime    string
}

// ProcessingJobDetails contains detailed processing job information
type ProcessingJobDetails struct {
	ProcessingJob
	ProcessingStartTime string
	RoleARN             string
	ProcessingInputs    []ProcessingInput
	OutputConfig        ProcessingOutputConfig
	ResourceConfig      ProcessingResourceConfig
	ExitMessage         string
	FailureReason       string
}

// ProcessingInput represents a processing job input
type ProcessingInput struct {
	InputName   string
	S3URI       string
	S3DataType  string
	S3InputMode string
	LocalPath   string
}

// ProcessingOutputConfig represents processing job output configuration
type ProcessingOutputConfig struct {
	KMSKeyID string
	Outputs  []ProcessingOutput
}

// ProcessingOutput represents a processing job output
type ProcessingOutput struct {
	OutputName   string
	S3URI        string
	S3UploadMode string
	LocalPath    string
}

// ProcessingResourceConfig represents processing job resource configuration
type ProcessingResourceConfig struct {
	InstanceType   string
	InstanceCount  int
	VolumeSizeGB   int
	VolumeKmsKeyID string
}

// NotebookInstance represents a SageMaker notebook instance
type NotebookInstance struct {
	NotebookInstanceARN    string
	NotebookInstanceName   string
	NotebookInstanceStatus string
	InstanceType           string
	CreationTime           string
	LastModifiedTime       string
	URL                    string
}

// NotebookInstanceDetails contains detailed notebook instance information
type NotebookInstanceDetails struct {
	NotebookInstance
	RoleARN                    string
	KMSKeyID                   string
	NetworkInterfaceID         string
	SubnetID                   string
	VolumeSizeGB               int
	DirectInternetAccess       string
	RootAccess                 string
	SecurityGroups             []string
	AcceleratorTypes           []string
	DefaultCodeRepository      string
	AdditionalCodeRepositories []string
	PlatformIdentifier         string
}
