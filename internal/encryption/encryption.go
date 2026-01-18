package encryption

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/qualys/dspm/internal/models"
)

// Service provides encryption visibility and compliance scoring functionality
type Service struct {
	store  Store
	scorer *ComplianceScorer
}

// Store defines the interface for encryption data persistence
type Store interface {
	// Encryption Keys
	CreateEncryptionKey(ctx context.Context, key *models.EncryptionKey) error
	UpdateEncryptionKey(ctx context.Context, key *models.EncryptionKey) error
	GetEncryptionKey(ctx context.Context, id uuid.UUID) (*models.EncryptionKey, error)
	GetEncryptionKeyByARN(ctx context.Context, arn string) (*models.EncryptionKey, error)
	ListEncryptionKeys(ctx context.Context, accountID uuid.UUID) ([]*models.EncryptionKey, error)
	DeleteEncryptionKey(ctx context.Context, id uuid.UUID) error

	// Key Usage
	CreateKeyUsage(ctx context.Context, usage *models.EncryptionKeyUsage) error
	UpdateKeyUsage(ctx context.Context, usage *models.EncryptionKeyUsage) error
	GetKeyUsageByAsset(ctx context.Context, assetID uuid.UUID) ([]*models.EncryptionKeyUsage, error)
	GetKeyUsageByKey(ctx context.Context, keyID uuid.UUID) ([]*models.EncryptionKeyUsage, error)
	ListKeyUsage(ctx context.Context, accountID uuid.UUID) ([]*models.EncryptionKeyUsage, error)

	// Transit Encryption
	CreateTransitEncryption(ctx context.Context, transit *models.TransitEncryption) error
	UpdateTransitEncryption(ctx context.Context, transit *models.TransitEncryption) error
	GetTransitEncryption(ctx context.Context, assetID uuid.UUID) (*models.TransitEncryption, error)
	ListTransitEncryption(ctx context.Context, accountID uuid.UUID) ([]*models.TransitEncryption, error)

	// Compliance
	CreateEncryptionCompliance(ctx context.Context, compliance *models.EncryptionCompliance) error
	UpdateEncryptionCompliance(ctx context.Context, compliance *models.EncryptionCompliance) error
	GetEncryptionCompliance(ctx context.Context, assetID uuid.UUID) (*models.EncryptionCompliance, error)
	ListEncryptionCompliance(ctx context.Context, accountID uuid.UUID) ([]*models.EncryptionCompliance, error)

	// Assets (for lookups)
	GetDataAsset(ctx context.Context, id uuid.UUID) (*models.DataAsset, error)
	ListDataAssets(ctx context.Context, accountID uuid.UUID) ([]*models.DataAsset, error)
}

// NewService creates a new encryption service
func NewService(store Store) *Service {
	return &Service{
		store:  store,
		scorer: NewComplianceScorer(),
	}
}

// NewServiceWithScorer creates a new encryption service with a custom scorer
func NewServiceWithScorer(store Store, scorer *ComplianceScorer) *Service {
	return &Service{
		store:  store,
		scorer: scorer,
	}
}

// GetEncryptionOverview returns an overview of encryption status for an account
func (s *Service) GetEncryptionOverview(ctx context.Context, accountID uuid.UUID) (*EncryptionOverview, error) {
	assets, err := s.store.ListDataAssets(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing assets: %w", err)
	}

	keys, err := s.store.ListEncryptionKeys(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing keys: %w", err)
	}

	compliance, err := s.store.ListEncryptionCompliance(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing compliance: %w", err)
	}

	overview := &EncryptionOverview{
		AccountID:         accountID,
		TotalAssets:       len(assets),
		EncryptionByType:  make(map[string]int),
		ComplianceByGrade: make(map[string]int),
		LastEvaluatedAt:   time.Now(),
	}

	// Count encrypted vs unencrypted assets
	for _, asset := range assets {
		if asset.EncryptionStatus == models.EncryptionNone {
			overview.UnencryptedAssets++
		} else {
			overview.EncryptedAssets++
		}
		overview.EncryptionByType[string(asset.EncryptionStatus)]++
	}

	// Count keys with rotation
	overview.TotalKeys = len(keys)
	for _, key := range keys {
		if key.RotationEnabled {
			overview.KeysWithRotation++
		}
	}

	// Calculate average compliance and grade distribution
	var totalScore int
	for _, c := range compliance {
		totalScore += c.ComplianceScore
		overview.ComplianceByGrade[c.Grade]++
		overview.CriticalFindings += c.CriticalFindings
	}
	if len(compliance) > 0 {
		overview.AverageCompliance = float64(totalScore) / float64(len(compliance))
	}

	return overview, nil
}

// EvaluateAssetCompliance evaluates and stores encryption compliance for an asset
func (s *Service) EvaluateAssetCompliance(ctx context.Context, assetID uuid.UUID) (*models.EncryptionCompliance, error) {
	// Get asset details
	asset, err := s.store.GetDataAsset(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("getting asset: %w", err)
	}

	// Build encryption profile
	profile := &AssetEncryptionProfile{
		AssetID:          asset.ID,
		AssetARN:         asset.ResourceARN,
		AssetType:        string(asset.ResourceType),
		EncryptionStatus: asset.EncryptionStatus,
		EncryptionKeyARN: asset.EncryptionKeyARN,
	}

	// Get key details if using KMS encryption
	if asset.EncryptionKeyARN != "" {
		key, err := s.store.GetEncryptionKeyByARN(ctx, asset.EncryptionKeyARN)
		if err == nil && key != nil {
			profile.Key = key
			profile.KeyRotationEnabled = key.RotationEnabled
		}
	}

	// Get transit encryption details
	transit, err := s.store.GetTransitEncryption(ctx, assetID)
	if err == nil && transit != nil {
		profile.TransitEncryption = transit
	}

	// Calculate compliance score
	result := s.scorer.CalculateComplianceScore(profile)

	// Create compliance record
	compliance := &models.EncryptionCompliance{
		ID:                 uuid.New(),
		AccountID:          &asset.AccountID,
		AssetID:            &asset.ID,
		ComplianceScore:    result.Score,
		Grade:              result.Grade,
		AtRestScore:        result.AtRestScore,
		InTransitScore:     result.InTransitScore,
		KeyManagementScore: result.KeyMgmtScore,
		FindingsCount:      len(result.Findings),
		Recommendations:    result.Recommendations,
		EvaluatedAt:        time.Now(),
	}

	// Count critical findings
	for _, finding := range result.Findings {
		if finding.Severity == models.SeverityCritical {
			compliance.CriticalFindings++
		}
	}

	// Store compliance record
	err = s.store.CreateEncryptionCompliance(ctx, compliance)
	if err != nil {
		// Try update if already exists
		err = s.store.UpdateEncryptionCompliance(ctx, compliance)
		if err != nil {
			return nil, fmt.Errorf("storing compliance: %w", err)
		}
	}

	return compliance, nil
}

// GetKeyUsageSummary returns a summary of how a key is being used
func (s *Service) GetKeyUsageSummary(ctx context.Context, keyID uuid.UUID) (*KeyUsageSummary, error) {
	key, err := s.store.GetEncryptionKey(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("getting key: %w", err)
	}

	usages, err := s.store.GetKeyUsageByKey(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("getting key usage: %w", err)
	}

	summary := &KeyUsageSummary{
		Key:        key,
		UsageCount: len(usages),
		AssetTypes: make(map[string]int),
		UsageTypes: make(map[string]int),
		Assets:     usages,
	}

	for _, usage := range usages {
		summary.AssetTypes[usage.AssetType]++
		summary.UsageTypes[string(usage.UsageType)]++
	}

	return summary, nil
}

// RecordKeyUsage records that an asset is using a specific encryption key
func (s *Service) RecordKeyUsage(ctx context.Context, keyARN string, assetID uuid.UUID, assetARN string, assetType string, usageType models.EncryptionUsageType) error {
	// Get key by ARN
	key, err := s.store.GetEncryptionKeyByARN(ctx, keyARN)
	if err != nil {
		return fmt.Errorf("getting key by ARN: %w", err)
	}

	usage := &models.EncryptionKeyUsage{
		ID:          uuid.New(),
		KeyID:       key.ID,
		AssetID:     &assetID,
		AssetARN:    assetARN,
		AssetType:   assetType,
		UsageType:   usageType,
		FirstSeenAt: time.Now(),
		LastSeenAt:  time.Now(),
	}

	err = s.store.CreateKeyUsage(ctx, usage)
	if err != nil {
		// Update last seen if already exists
		usage.LastSeenAt = time.Now()
		return s.store.UpdateKeyUsage(ctx, usage)
	}

	return nil
}

// ListEncryptionKeys returns all encryption keys for an account
func (s *Service) ListEncryptionKeys(ctx context.Context, accountID uuid.UUID) ([]*models.EncryptionKey, error) {
	return s.store.ListEncryptionKeys(ctx, accountID)
}

// GetEncryptionKey returns a specific encryption key
func (s *Service) GetEncryptionKey(ctx context.Context, keyID uuid.UUID) (*models.EncryptionKey, error) {
	return s.store.GetEncryptionKey(ctx, keyID)
}

// ListTransitEncryption returns all transit encryption settings for an account
func (s *Service) ListTransitEncryption(ctx context.Context, accountID uuid.UUID) ([]*models.TransitEncryption, error) {
	return s.store.ListTransitEncryption(ctx, accountID)
}

// GetComplianceSummary returns compliance scores grouped by grade for an account
func (s *Service) GetComplianceSummary(ctx context.Context, accountID uuid.UUID) (map[string]int, error) {
	compliance, err := s.store.ListEncryptionCompliance(ctx, accountID)
	if err != nil {
		return nil, err
	}

	summary := make(map[string]int)
	for _, c := range compliance {
		summary[c.Grade]++
	}

	return summary, nil
}

// GetAssetComplianceScore returns the compliance score for a specific asset
func (s *Service) GetAssetComplianceScore(ctx context.Context, assetID uuid.UUID) (*models.EncryptionCompliance, error) {
	return s.store.GetEncryptionCompliance(ctx, assetID)
}
