package lineage

import (
	"time"

	"github.com/google/uuid"
	"github.com/qualys/dspm/internal/models"
)

// LineageGraph represents a complete data lineage graph for visualization
type LineageGraph struct {
	Nodes []LineageNode `json:"nodes"`
	Edges []LineageEdge `json:"edges"`
}

// LineageNode represents a node in the lineage graph
type LineageNode struct {
	ID               string            `json:"id"`
	ARN              string            `json:"arn"`
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	SensitivityLevel models.Sensitivity `json:"sensitivity_level,omitempty"`
	DataCategories   []string          `json:"data_categories,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// LineageEdge represents an edge in the lineage graph
type LineageEdge struct {
	ID              string          `json:"id"`
	Source          string          `json:"source"`
	Target          string          `json:"target"`
	FlowType        models.FlowType `json:"flow_type"`
	InferredFrom    string          `json:"inferred_from"`
	ConfidenceScore float64         `json:"confidence_score"`
	AccessMethod    string          `json:"access_method,omitempty"`
}

// InferredFlow represents a data flow inferred from configuration
type InferredFlow struct {
	SourceARN       string
	SourceType      string
	SourceName      string
	TargetARN       string
	TargetType      string
	TargetName      string
	FlowType        models.FlowType
	InferredFrom    models.InferenceSource
	ConfidenceScore float64
	Evidence        map[string]interface{}
}

// FunctionConfig contains Lambda/serverless function configuration for lineage inference
type FunctionConfig struct {
	FunctionARN      string
	FunctionName     string
	Runtime          string
	Handler          string
	Role             string
	Environment      map[string]string
	EventSources     []EventSource
	VPCConfig        *VPCConfig
	KMSKeyARN        string
	Layers           []string
}

// EventSource represents an event source mapping for a function
type EventSource struct {
	EventSourceARN string
	Type           string // S3, DynamoDB, SQS, Kinesis, etc.
	BatchSize      int
	State          string
}

// VPCConfig contains VPC configuration for a function
type VPCConfig struct {
	SubnetIDs        []string
	SecurityGroupIDs []string
	VPCID            string
}

// PolicyStatement represents an IAM policy statement
type PolicyStatement struct {
	Effect    string
	Actions   []string
	Resources []string
	Principal interface{}
}

// LineageOverview provides a summary of data lineage for an account
type LineageOverview struct {
	AccountID            uuid.UUID          `json:"account_id"`
	TotalFlows           int                `json:"total_flows"`
	FlowsByType          map[string]int     `json:"flows_by_type"`
	SensitiveDataFlows   int                `json:"sensitive_data_flows"`
	CrossAccountFlows    int                `json:"cross_account_flows"`
	TopDataSources       []ResourceSummary  `json:"top_data_sources"`
	TopDataDestinations  []ResourceSummary  `json:"top_data_destinations"`
	LastUpdated          time.Time          `json:"last_updated"`
}

// ResourceSummary provides a summary of a resource in lineage context
type ResourceSummary struct {
	ARN              string `json:"arn"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	FlowCount        int    `json:"flow_count"`
	SensitiveFlows   int    `json:"sensitive_flows"`
}

// SensitiveDataFlow represents a data flow involving sensitive data
type SensitiveDataFlow struct {
	Flow             *models.LineageEvent `json:"flow"`
	SensitivityLevel models.Sensitivity   `json:"sensitivity_level"`
	DataCategories   []string             `json:"data_categories"`
	RiskScore        int                  `json:"risk_score"`
	RiskFactors      []string             `json:"risk_factors"`
}

// LineagePathRequest represents a request to find lineage paths
type LineagePathRequest struct {
	SourceARN      string `json:"source_arn,omitempty"`
	DestinationARN string `json:"destination_arn,omitempty"`
	MaxHops        int    `json:"max_hops,omitempty"`
	SensitiveOnly  bool   `json:"sensitive_only,omitempty"`
}

// EnvironmentVariablePattern represents patterns for inferring data sources from env vars
type EnvironmentVariablePattern struct {
	NamePattern   string
	ValuePattern  string
	ResourceType  string
	FlowType      models.FlowType
}

// DefaultEnvVarPatterns returns common patterns for inferring data sources
func DefaultEnvVarPatterns() []EnvironmentVariablePattern {
	return []EnvironmentVariablePattern{
		// S3 buckets
		{NamePattern: "(?i)bucket|s3", ValuePattern: `^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`, ResourceType: "s3_bucket", FlowType: models.FlowReadsFrom},
		{NamePattern: "(?i)output.*bucket|destination.*bucket", ValuePattern: `^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`, ResourceType: "s3_bucket", FlowType: models.FlowWritesTo},

		// DynamoDB tables
		{NamePattern: "(?i)table|dynamodb", ValuePattern: `^[a-zA-Z0-9._-]+$`, ResourceType: "dynamodb_table", FlowType: models.FlowReadsFrom},

		// RDS/Database endpoints
		{NamePattern: "(?i)db.*host|database.*host|rds", ValuePattern: `.*\.rds\.amazonaws\.com$`, ResourceType: "rds_instance", FlowType: models.FlowReadsFrom},

		// SQS queues
		{NamePattern: "(?i)queue|sqs", ValuePattern: `^https://sqs\..*\.amazonaws\.com/`, ResourceType: "sqs_queue", FlowType: models.FlowWritesTo},

		// SNS topics
		{NamePattern: "(?i)topic|sns", ValuePattern: `^arn:aws:sns:`, ResourceType: "sns_topic", FlowType: models.FlowWritesTo},
	}
}
