package encryption

import (
	"github.com/qualys/dspm/internal/models"
)

// ComplianceScorer calculates encryption compliance scores
type ComplianceScorer struct {
	weights ScoringWeights
}

// NewComplianceScorer creates a new compliance scorer with default weights
func NewComplianceScorer() *ComplianceScorer {
	return &ComplianceScorer{
		weights: DefaultScoringWeights(),
	}
}

// NewComplianceScorerWithWeights creates a compliance scorer with custom weights
func NewComplianceScorerWithWeights(weights ScoringWeights) *ComplianceScorer {
	return &ComplianceScorer{
		weights: weights,
	}
}

// CalculateComplianceScore computes the overall encryption compliance score for an asset
func (cs *ComplianceScorer) CalculateComplianceScore(profile *AssetEncryptionProfile) *ComplianceResult {
	result := &ComplianceResult{
		Findings:        []EncryptionFinding{},
		Recommendations: []string{},
	}

	// Calculate individual scores
	result.AtRestScore = cs.calculateAtRestScore(profile, result)
	result.InTransitScore = cs.calculateInTransitScore(profile, result)
	result.KeyMgmtScore = cs.calculateKeyManagementScore(profile, result)

	// Calculate weighted total
	result.Score = int(
		float64(result.AtRestScore)*cs.weights.AtRest +
			float64(result.InTransitScore)*cs.weights.InTransit +
			float64(result.KeyMgmtScore)*cs.weights.KeyManagement,
	)

	// Assign grade
	result.Grade = cs.calculateGrade(result.Score)

	return result
}

// calculateAtRestScore calculates the at-rest encryption score (max 100)
func (cs *ComplianceScorer) calculateAtRestScore(profile *AssetEncryptionProfile, result *ComplianceResult) int {
	score := 0

	// Check if encryption is enabled at all
	if profile.EncryptionStatus == models.EncryptionNone {
		result.Findings = append(result.Findings, EncryptionFinding{
			Type:        "UNENCRYPTED_STORAGE",
			Severity:    models.SeverityHigh,
			Title:       "Storage is not encrypted at rest",
			Description: "The data asset does not have encryption enabled, leaving data vulnerable to unauthorized access.",
			Remediation: "Enable server-side encryption using SSE-S3, SSE-KMS, or customer-managed keys.",
		})
		result.Recommendations = append(result.Recommendations, "Enable encryption at rest for this asset")
		return 0
	}

	// Base score for having encryption enabled: +40 points
	score += 40

	// KMS encryption (SSE-KMS or CMK): +30 points
	if profile.EncryptionStatus == models.EncryptionSSEKMS || profile.EncryptionStatus == models.EncryptionCMK {
		score += 30
	} else {
		result.Recommendations = append(result.Recommendations, "Consider upgrading to KMS-managed encryption for better key control")
	}

	// Customer-managed key (CMK): +20 additional points
	if profile.EncryptionStatus == models.EncryptionCMK {
		score += 20
	}

	// Key rotation enabled: +10 points
	if profile.KeyRotationEnabled {
		score += 10
	} else if profile.EncryptionStatus == models.EncryptionSSEKMS || profile.EncryptionStatus == models.EncryptionCMK {
		result.Findings = append(result.Findings, EncryptionFinding{
			Type:        "KEY_ROTATION_DISABLED",
			Severity:    models.SeverityMedium,
			Title:       "Encryption key rotation is not enabled",
			Description: "The KMS key used for encryption does not have automatic rotation enabled.",
			Remediation: "Enable automatic key rotation for the KMS key.",
		})
		result.Recommendations = append(result.Recommendations, "Enable automatic key rotation")
	}

	return min(score, 100)
}

// calculateInTransitScore calculates the in-transit encryption score (max 100)
func (cs *ComplianceScorer) calculateInTransitScore(profile *AssetEncryptionProfile, result *ComplianceResult) int {
	// If no transit encryption info available, assume compliant (cloud defaults)
	if profile.TransitEncryption == nil {
		return 80 // Default score when not explicitly checked
	}

	transit := profile.TransitEncryption
	score := 0

	// TLS enabled: +50 points
	if transit.TLSEnabled {
		score += 50
	} else {
		result.Findings = append(result.Findings, EncryptionFinding{
			Type:        "TRANSIT_ENCRYPTION_DISABLED",
			Severity:    models.SeverityCritical,
			Title:       "In-transit encryption is disabled",
			Description: "Data transmitted to/from this asset is not encrypted, exposing it to interception.",
			Remediation: "Enable TLS/SSL encryption for all data in transit.",
		})
		result.Recommendations = append(result.Recommendations, "Enable TLS encryption for data in transit")
		return 0
	}

	// Modern TLS version (1.2+): +25 points
	if transit.TLSVersion == "TLSv1.2" || transit.TLSVersion == "TLSv1.3" {
		score += 25
	} else if transit.TLSVersion != "" {
		result.Findings = append(result.Findings, EncryptionFinding{
			Type:        "TLS_OUTDATED",
			Severity:    models.SeverityMedium,
			Title:       "Outdated TLS version in use",
			Description: "The TLS version in use is outdated and may have known vulnerabilities.",
			Remediation: "Upgrade to TLS 1.2 or TLS 1.3.",
		})
		result.Recommendations = append(result.Recommendations, "Upgrade to TLS 1.2 or higher")
	}

	// TLS 1.3: +10 bonus points
	if transit.TLSVersion == "TLSv1.3" {
		score += 10
	}

	// Perfect forward secrecy: +10 points
	if transit.SupportsPerfectForwardSecrecy {
		score += 10
	}

	// Valid certificate: +5 points
	if transit.CertificateARN != "" {
		score += 5
	}

	return min(score, 100)
}

// calculateKeyManagementScore calculates the key management score (max 100)
func (cs *ComplianceScorer) calculateKeyManagementScore(profile *AssetEncryptionProfile, result *ComplianceResult) int {
	// If no encryption, key management is N/A - return 0
	if profile.EncryptionStatus == models.EncryptionNone {
		return 0
	}

	// If using SSE (AWS-managed), limited control - base score
	if profile.EncryptionStatus == models.EncryptionSSE {
		return 50
	}

	// If no key info available for KMS encryption, return moderate score
	if profile.Key == nil {
		return 60
	}

	key := profile.Key
	score := 0

	// Key is enabled: +30 points
	if key.Enabled {
		score += 30
	} else {
		result.Findings = append(result.Findings, EncryptionFinding{
			Type:        "KEY_DISABLED",
			Severity:    models.SeverityHigh,
			Title:       "Encryption key is disabled",
			Description: "The KMS key used for encryption is disabled.",
			Remediation: "Enable the KMS key or rotate to a new key.",
		})
		return 0
	}

	// Key rotation enabled: +25 points
	if key.RotationEnabled {
		score += 25
	}

	// Customer managed (not AWS managed): +20 points
	if key.KeyManager == "CUSTOMER" {
		score += 20
	}

	// Key policy doesn't allow public access: +15 points
	if !key.AllowsPublicAccess {
		score += 15
	} else {
		result.Findings = append(result.Findings, EncryptionFinding{
			Type:        "KEY_POLICY_OVERPERMISSIVE",
			Severity:    models.SeverityCritical,
			Title:       "KMS key policy allows public access",
			Description: "The KMS key policy is overly permissive and may allow unauthorized access.",
			Remediation: "Review and restrict the KMS key policy to only necessary principals.",
		})
	}

	// Key not pending deletion: +10 points
	if key.KeyState != models.KeyStatePendingDeletion {
		score += 10
	} else {
		result.Findings = append(result.Findings, EncryptionFinding{
			Type:        "KEY_PENDING_DELETION",
			Severity:    models.SeverityHigh,
			Title:       "Encryption key is pending deletion",
			Description: "The KMS key is scheduled for deletion. Data encrypted with this key will become inaccessible.",
			Remediation: "Cancel key deletion if data is still needed, or migrate to a new key.",
		})
	}

	return min(score, 100)
}

// calculateGrade assigns a letter grade based on the score
func (cs *ComplianceScorer) calculateGrade(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
