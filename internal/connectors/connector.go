package connectors

import (
	"context"
	"io"

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
