package remediation

import (
	"time"

	"github.com/google/uuid"
)

// ActionType represents the type of remediation action
type ActionType string

const (
	ActionEnableBucketEncryption ActionType = "ENABLE_BUCKET_ENCRYPTION"
	ActionBlockPublicAccess      ActionType = "BLOCK_PUBLIC_ACCESS"
	ActionEnableKMSRotation      ActionType = "ENABLE_KMS_ROTATION"
	ActionUpgradeTLS             ActionType = "UPGRADE_TLS"
	ActionRevokePublicACL        ActionType = "REVOKE_PUBLIC_ACL"
	ActionEnableVersioning       ActionType = "ENABLE_VERSIONING"
	ActionEnableLogging          ActionType = "ENABLE_LOGGING"
	ActionRestrictIAMPolicy      ActionType = "RESTRICT_IAM_POLICY"
)

// ActionStatus represents the status of a remediation action
type ActionStatus string

const (
	StatusPending    ActionStatus = "PENDING"
	StatusApproved   ActionStatus = "APPROVED"
	StatusExecuting  ActionStatus = "EXECUTING"
	StatusCompleted  ActionStatus = "COMPLETED"
	StatusFailed     ActionStatus = "FAILED"
	StatusRolledBack ActionStatus = "ROLLED_BACK"
	StatusRejected   ActionStatus = "REJECTED"
)

// RiskLevel indicates the risk level of a remediation action
type RiskLevel string

const (
	RiskLow    RiskLevel = "LOW"
	RiskMedium RiskLevel = "MEDIUM"
	RiskHigh   RiskLevel = "HIGH"
)

// Action represents a remediation action
type Action struct {
	ID                uuid.UUID              `json:"id"`
	AccountID         uuid.UUID              `json:"account_id"`
	AssetID           uuid.UUID              `json:"asset_id"`
	FindingID         *uuid.UUID             `json:"finding_id,omitempty"`
	ActionType        ActionType             `json:"action_type"`
	Status            ActionStatus           `json:"status"`
	RiskLevel         RiskLevel              `json:"risk_level"`
	Description       string                 `json:"description"`
	Parameters        map[string]interface{} `json:"parameters"`
	PreviousState     map[string]interface{} `json:"previous_state,omitempty"`
	NewState          map[string]interface{} `json:"new_state,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	ApprovedAt        *time.Time             `json:"approved_at,omitempty"`
	ApprovedBy        string                 `json:"approved_by,omitempty"`
	ExecutedAt        *time.Time             `json:"executed_at,omitempty"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage      string                 `json:"error_message,omitempty"`
	RollbackAvailable bool                   `json:"rollback_available"`
}

// CreateActionRequest represents a request to create a remediation action
type CreateActionRequest struct {
	AccountID   uuid.UUID              `json:"account_id"`
	AssetID     uuid.UUID              `json:"asset_id"`
	FindingID   *uuid.UUID             `json:"finding_id,omitempty"`
	ActionType  ActionType             `json:"action_type"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// ApproveActionRequest represents a request to approve a remediation action
type ApproveActionRequest struct {
	ApprovedBy string `json:"approved_by"`
	Comment    string `json:"comment,omitempty"`
}

// ExecuteResult represents the result of executing a remediation action
type ExecuteResult struct {
	Success       bool                   `json:"success"`
	PreviousState map[string]interface{} `json:"previous_state,omitempty"`
	NewState      map[string]interface{} `json:"new_state,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
}

// RollbackResult represents the result of rolling back a remediation action
type RollbackResult struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// Playbook represents a predefined remediation playbook
type Playbook struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Actions     []ActionType `json:"actions"`
	RiskLevel   RiskLevel    `json:"risk_level"`
	AutoApprove bool         `json:"auto_approve"`
}

// ActionSummary provides a summary of remediation actions
type ActionSummary struct {
	TotalActions     int            `json:"total_actions"`
	PendingCount     int            `json:"pending_count"`
	ApprovedCount    int            `json:"approved_count"`
	CompletedCount   int            `json:"completed_count"`
	FailedCount      int            `json:"failed_count"`
	ByActionType     map[string]int `json:"by_action_type"`
	ByRiskLevel      map[string]int `json:"by_risk_level"`
	RecentlyExecuted []Action       `json:"recently_executed"`
}

// ActionDefinition describes a remediation action type
type ActionDefinition struct {
	ActionType        ActionType `json:"action_type"`
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	RiskLevel         RiskLevel  `json:"risk_level"`
	RequiredParams    []string   `json:"required_params"`
	OptionalParams    []string   `json:"optional_params,omitempty"`
	RollbackAvailable bool       `json:"rollback_available"`
	ApplicableAssets  []string   `json:"applicable_assets"`
}

// GetActionDefinitions returns all available remediation action definitions
func GetActionDefinitions() []ActionDefinition {
	return []ActionDefinition{
		{
			ActionType:        ActionEnableBucketEncryption,
			Name:              "Enable Bucket Encryption",
			Description:       "Enable server-side encryption on S3 bucket using AES-256 or KMS",
			RiskLevel:         RiskLow,
			RequiredParams:    []string{"bucket_name"},
			OptionalParams:    []string{"kms_key_id", "algorithm"},
			RollbackAvailable: false,
			ApplicableAssets:  []string{"S3_BUCKET"},
		},
		{
			ActionType:        ActionBlockPublicAccess,
			Name:              "Block Public Access",
			Description:       "Enable S3 Block Public Access settings to prevent public exposure",
			RiskLevel:         RiskMedium,
			RequiredParams:    []string{"bucket_name"},
			OptionalParams:    []string{"block_public_acls", "ignore_public_acls", "block_public_policy", "restrict_public_buckets"},
			RollbackAvailable: true,
			ApplicableAssets:  []string{"S3_BUCKET"},
		},
		{
			ActionType:        ActionEnableKMSRotation,
			Name:              "Enable KMS Key Rotation",
			Description:       "Enable automatic annual rotation for a KMS customer managed key",
			RiskLevel:         RiskLow,
			RequiredParams:    []string{"key_id"},
			RollbackAvailable: true,
			ApplicableAssets:  []string{"KMS_KEY"},
		},
		{
			ActionType:        ActionUpgradeTLS,
			Name:              "Upgrade TLS Version",
			Description:       "Update minimum TLS version requirement to TLS 1.2 or higher",
			RiskLevel:         RiskMedium,
			RequiredParams:    []string{"resource_arn", "min_tls_version"},
			RollbackAvailable: true,
			ApplicableAssets:  []string{"S3_BUCKET", "CLOUDFRONT_DISTRIBUTION", "API_GATEWAY"},
		},
		{
			ActionType:        ActionRevokePublicACL,
			Name:              "Revoke Public ACL",
			Description:       "Remove public grants from S3 bucket ACL",
			RiskLevel:         RiskMedium,
			RequiredParams:    []string{"bucket_name"},
			RollbackAvailable: true,
			ApplicableAssets:  []string{"S3_BUCKET"},
		},
		{
			ActionType:        ActionEnableVersioning,
			Name:              "Enable Versioning",
			Description:       "Enable versioning on S3 bucket for data protection",
			RiskLevel:         RiskLow,
			RequiredParams:    []string{"bucket_name"},
			RollbackAvailable: false,
			ApplicableAssets:  []string{"S3_BUCKET"},
		},
		{
			ActionType:        ActionEnableLogging,
			Name:              "Enable Logging",
			Description:       "Enable access logging for S3 bucket",
			RiskLevel:         RiskLow,
			RequiredParams:    []string{"bucket_name", "target_bucket"},
			OptionalParams:    []string{"target_prefix"},
			RollbackAvailable: true,
			ApplicableAssets:  []string{"S3_BUCKET"},
		},
		{
			ActionType:        ActionRestrictIAMPolicy,
			Name:              "Restrict IAM Policy",
			Description:       "Modify IAM policy to remove overly permissive statements",
			RiskLevel:         RiskHigh,
			RequiredParams:    []string{"policy_arn", "new_policy_document"},
			RollbackAvailable: true,
			ApplicableAssets:  []string{"IAM_POLICY"},
		},
	}
}
