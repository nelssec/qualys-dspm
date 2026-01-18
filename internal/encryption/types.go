package encryption

import (
	"time"

	"github.com/google/uuid"
	"github.com/qualys/dspm/internal/models"
)

// AssetEncryptionProfile contains all encryption-related information for an asset
type AssetEncryptionProfile struct {
	AssetID            uuid.UUID
	AssetARN           string
	AssetType          string
	EncryptionStatus   models.EncryptionStatus
	EncryptionKeyARN   string
	Key                *models.EncryptionKey
	KeyRotationEnabled bool
	TransitEncryption  *models.TransitEncryption
	Findings           []EncryptionFinding
}

// EncryptionFinding represents a security finding related to encryption
type EncryptionFinding struct {
	Type        string
	Severity    models.FindingSeverity
	Title       string
	Description string
	Remediation string
}

// KeyDiscoveryResult contains discovered KMS keys and their usage
type KeyDiscoveryResult struct {
	Keys      []*models.EncryptionKey
	KeyUsages []*models.EncryptionKeyUsage
	Errors    []error
}

// ComplianceResult contains the results of encryption compliance evaluation
type ComplianceResult struct {
	Score           int
	Grade           string
	AtRestScore     int
	InTransitScore  int
	KeyMgmtScore    int
	Findings        []EncryptionFinding
	Recommendations []string
}

// EncryptionOverview provides a summary of encryption status across an account
type EncryptionOverview struct {
	AccountID           uuid.UUID                 `json:"account_id"`
	TotalAssets         int                       `json:"total_assets"`
	EncryptedAssets     int                       `json:"encrypted_assets"`
	UnencryptedAssets   int                       `json:"unencrypted_assets"`
	EncryptionByType    map[string]int            `json:"encryption_by_type"`
	TotalKeys           int                       `json:"total_keys"`
	KeysWithRotation    int                       `json:"keys_with_rotation"`
	AverageCompliance   float64                   `json:"average_compliance"`
	ComplianceByGrade   map[string]int            `json:"compliance_by_grade"`
	CriticalFindings    int                       `json:"critical_findings"`
	LastEvaluatedAt     time.Time                 `json:"last_evaluated_at"`
}

// KeyUsageSummary summarizes how a key is being used
type KeyUsageSummary struct {
	Key         *models.EncryptionKey      `json:"key"`
	UsageCount  int                        `json:"usage_count"`
	AssetTypes  map[string]int             `json:"asset_types"`
	UsageTypes  map[string]int             `json:"usage_types"`
	Assets      []*models.EncryptionKeyUsage `json:"assets,omitempty"`
}

// TransitEncryptionCheck represents the result of checking in-transit encryption
type TransitEncryptionCheck struct {
	AssetID               uuid.UUID `json:"asset_id"`
	EndpointType          string    `json:"endpoint_type"`
	TLSEnabled            bool      `json:"tls_enabled"`
	TLSVersion            string    `json:"tls_version"`
	MeetsMinimumStandards bool      `json:"meets_minimum_standards"`
	Issues                []string  `json:"issues"`
}

// ScoringWeights defines the weights for compliance scoring
type ScoringWeights struct {
	AtRest        float64 `json:"at_rest"`
	InTransit     float64 `json:"in_transit"`
	KeyManagement float64 `json:"key_management"`
}

// DefaultScoringWeights returns the default scoring weights
func DefaultScoringWeights() ScoringWeights {
	return ScoringWeights{
		AtRest:        0.40,
		InTransit:     0.30,
		KeyManagement: 0.30,
	}
}
