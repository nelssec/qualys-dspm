package connectors

import (
	"context"
	"io"

	"github.com/qualys/dspm/internal/models"
)

// Connector defines the interface for cloud provider connectors
type Connector interface {
	// Provider returns the cloud provider type
	Provider() models.Provider

	// Validate tests the connection and permissions
	Validate(ctx context.Context) error

	// Close releases any resources held by the connector
	Close() error
}

// StorageConnector provides storage-specific operations
type StorageConnector interface {
	Connector

	// ListBuckets returns all storage buckets/containers
	ListBuckets(ctx context.Context) ([]BucketInfo, error)

	// GetBucketMetadata returns detailed metadata for a bucket
	GetBucketMetadata(ctx context.Context, bucketName string) (*BucketMetadata, error)

	// ListObjects lists objects in a bucket with optional prefix
	ListObjects(ctx context.Context, bucketName, prefix string, maxKeys int) ([]ObjectInfo, error)

	// GetObject retrieves an object's content (or a sample)
	GetObject(ctx context.Context, bucketName, objectKey string, byteRange *ByteRange) (io.ReadCloser, error)

	// GetBucketPolicy returns the bucket policy if set
	GetBucketPolicy(ctx context.Context, bucketName string) (*BucketPolicy, error)

	// GetBucketACL returns the bucket ACL
	GetBucketACL(ctx context.Context, bucketName string) (*BucketACL, error)
}

// IAMConnector provides IAM-specific operations
type IAMConnector interface {
	Connector

	// ListUsers returns all IAM users
	ListUsers(ctx context.Context) ([]Principal, error)

	// ListRoles returns all IAM roles
	ListRoles(ctx context.Context) ([]Principal, error)

	// ListPolicies returns all IAM policies
	ListPolicies(ctx context.Context) ([]PolicyInfo, error)

	// GetPolicy returns a specific policy document
	GetPolicy(ctx context.Context, policyARN string) (*PolicyDocument, error)

	// ListAttachedPolicies returns policies attached to a principal
	ListAttachedPolicies(ctx context.Context, principalARN string) ([]PolicyInfo, error)

	// GetServiceAccounts returns service accounts (GCP) or service-linked roles
	GetServiceAccounts(ctx context.Context) ([]Principal, error)
}

// ServerlessConnector provides serverless function operations
type ServerlessConnector interface {
	Connector

	// ListFunctions returns all serverless functions
	ListFunctions(ctx context.Context) ([]FunctionInfo, error)

	// GetFunctionConfig returns configuration for a function
	GetFunctionConfig(ctx context.Context, functionName string) (*FunctionConfig, error)

	// GetFunctionPolicy returns the resource policy for a function
	GetFunctionPolicy(ctx context.Context, functionName string) (*PolicyDocument, error)
}

// DatabaseConnector provides database discovery operations
type DatabaseConnector interface {
	Connector

	// ListDatabases returns all database instances
	ListDatabases(ctx context.Context) ([]DatabaseInfo, error)

	// GetDatabaseMetadata returns detailed metadata for a database
	GetDatabaseMetadata(ctx context.Context, databaseID string) (*DatabaseMetadata, error)
}

// KMSConnector provides encryption key operations
type KMSConnector interface {
	Connector

	// ListKeys returns all encryption keys
	ListKeys(ctx context.Context) ([]KeyInfo, error)

	// GetKeyMetadata returns metadata for a key
	GetKeyMetadata(ctx context.Context, keyID string) (*KeyMetadata, error)

	// GetKeyPolicy returns the key policy
	GetKeyPolicy(ctx context.Context, keyID string) (*PolicyDocument, error)
}

// ByteRange specifies a byte range for partial object retrieval
type ByteRange struct {
	Start int64
	End   int64
}

// BucketInfo contains basic bucket information
type BucketInfo struct {
	Name      string
	Region    string
	CreatedAt string
	ARN       string
}

// BucketMetadata contains detailed bucket metadata
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

// EncryptionConfig describes bucket encryption
type EncryptionConfig struct {
	Enabled   bool
	Type      models.EncryptionStatus
	KeyARN    string
	Algorithm string
}

// LoggingConfig describes bucket logging
type LoggingConfig struct {
	Enabled      bool
	TargetBucket string
	TargetPrefix string
}

// PublicAccessBlockConfig describes public access settings
type PublicAccessBlockConfig struct {
	BlockPublicAcls       bool
	IgnorePublicAcls      bool
	BlockPublicPolicy     bool
	RestrictPublicBuckets bool
}

// ObjectInfo contains basic object information
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified string
	StorageClass string
	ETag         string
}

// BucketPolicy represents a bucket policy
type BucketPolicy struct {
	Policy         string
	IsPublic       bool
	PublicActions  []string
	PolicyDocument *PolicyDocument
}

// BucketACL represents bucket ACL settings
type BucketACL struct {
	Owner  string
	Grants []ACLGrant
}

// ACLGrant represents an ACL grant
type ACLGrant struct {
	Grantee    string
	GranteeType string
	Permission string
	IsPublic   bool
}

// Principal represents an IAM principal (user, role, service account)
type Principal struct {
	ARN         string
	Name        string
	Type        string // USER, ROLE, GROUP, SERVICE
	CreatedAt   string
	Description string
	Tags        map[string]string
}

// PolicyInfo contains basic policy information
type PolicyInfo struct {
	ARN         string
	Name        string
	Type        string // MANAGED, INLINE, RESOURCE_BASED
	Description string
	IsAttached  bool
	AttachCount int
}

// PolicyDocument represents a parsed IAM policy
type PolicyDocument struct {
	Version    string
	Statements []PolicyStatement
	Raw        string
}

// PolicyStatement represents a single policy statement
type PolicyStatement struct {
	SID        string
	Effect     string
	Principals []string
	Actions    []string
	Resources  []string
	Conditions map[string]interface{}
}

// FunctionInfo contains basic function information
type FunctionInfo struct {
	ARN         string
	Name        string
	Runtime     string
	Handler     string
	Region      string
	MemorySize  int
	Timeout     int
	LastModified string
	Tags        map[string]string
}

// FunctionConfig contains detailed function configuration
type FunctionConfig struct {
	FunctionInfo
	Role           string
	VPCConfig      *VPCConfig
	Environment    map[string]string
	KMSKeyARN      string
	Layers         []string
	FileSystemConfigs []FileSystemConfig
}

// VPCConfig describes VPC configuration
type VPCConfig struct {
	SubnetIDs        []string
	SecurityGroupIDs []string
	VPCID            string
}

// FileSystemConfig describes EFS configuration
type FileSystemConfig struct {
	ARN            string
	LocalMountPath string
}

// DatabaseInfo contains basic database information
type DatabaseInfo struct {
	ID           string
	ARN          string
	Name         string
	Engine       string
	EngineVersion string
	Status       string
	Region       string
}

// DatabaseMetadata contains detailed database metadata
type DatabaseMetadata struct {
	DatabaseInfo
	Endpoint          string
	Port              int
	StorageEncrypted  bool
	KMSKeyID          string
	PubliclyAccessible bool
	VPCSecurityGroups []string
	SubnetGroup       string
	MultiAZ           bool
	Tags              map[string]string
}

// KeyInfo contains basic KMS key information
type KeyInfo struct {
	ID          string
	ARN         string
	Alias       string
	Description string
	Enabled     bool
	KeyManager  string // AWS, CUSTOMER
}

// KeyMetadata contains detailed key metadata
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
