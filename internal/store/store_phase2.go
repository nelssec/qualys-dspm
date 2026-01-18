package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/qualys/dspm/internal/models"
)

// =====================================================
// Encryption Keys Store Methods
// =====================================================

func (s *Store) CreateEncryptionKey(ctx context.Context, key *models.EncryptionKey) error {
	query := `
		INSERT INTO encryption_keys (
			id, account_id, key_id, key_arn, alias, description,
			key_type, key_usage, key_spec, key_manager, origin,
			key_state, enabled, rotation_enabled, last_rotated_at, next_rotation_at,
			rotation_period_days, deletion_date, pending_deletion_days,
			key_policy, allows_public_access, allows_cross_account, cross_account_principals,
			tags, region, created_at, updated_at, discovered_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28
		)
		ON CONFLICT (key_arn) DO UPDATE SET
			key_state = EXCLUDED.key_state,
			enabled = EXCLUDED.enabled,
			rotation_enabled = EXCLUDED.rotation_enabled,
			last_rotated_at = EXCLUDED.last_rotated_at,
			next_rotation_at = EXCLUDED.next_rotation_at,
			key_policy = EXCLUDED.key_policy,
			allows_public_access = EXCLUDED.allows_public_access,
			allows_cross_account = EXCLUDED.allows_cross_account,
			cross_account_principals = EXCLUDED.cross_account_principals,
			updated_at = EXCLUDED.updated_at
	`

	if key.ID == uuid.Nil {
		key.ID = uuid.New()
	}
	now := time.Now()
	if key.CreatedAt.IsZero() {
		key.CreatedAt = now
	}
	key.UpdatedAt = now
	if key.DiscoveredAt.IsZero() {
		key.DiscoveredAt = now
	}

	_, err := s.db.ExecContext(ctx, query,
		key.ID, key.AccountID, key.KeyID, key.KeyARN, key.Alias, key.Description,
		key.KeyType, key.KeyUsage, key.KeySpec, key.KeyManager, key.Origin,
		key.KeyState, key.Enabled, key.RotationEnabled, key.LastRotatedAt, key.NextRotationAt,
		key.RotationPeriodDays, key.DeletionDate, key.PendingDeletionDays,
		key.KeyPolicy, key.AllowsPublicAccess, key.AllowsCrossAccount, pq.Array(key.CrossAccountPrincipals),
		key.Tags, key.Region, key.CreatedAt, key.UpdatedAt, key.DiscoveredAt,
	)
	return err
}

func (s *Store) UpdateEncryptionKey(ctx context.Context, key *models.EncryptionKey) error {
	query := `
		UPDATE encryption_keys SET
			key_state = $1, enabled = $2, rotation_enabled = $3,
			last_rotated_at = $4, next_rotation_at = $5, key_policy = $6,
			allows_public_access = $7, allows_cross_account = $8,
			cross_account_principals = $9, updated_at = $10
		WHERE id = $11
	`
	key.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, query,
		key.KeyState, key.Enabled, key.RotationEnabled,
		key.LastRotatedAt, key.NextRotationAt, key.KeyPolicy,
		key.AllowsPublicAccess, key.AllowsCrossAccount,
		pq.Array(key.CrossAccountPrincipals), key.UpdatedAt, key.ID,
	)
	return err
}

func (s *Store) GetEncryptionKey(ctx context.Context, id uuid.UUID) (*models.EncryptionKey, error) {
	var key models.EncryptionKey
	query := `SELECT * FROM encryption_keys WHERE id = $1`
	err := s.db.GetContext(ctx, &key, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &key, err
}

func (s *Store) GetEncryptionKeyByARN(ctx context.Context, arn string) (*models.EncryptionKey, error) {
	var key models.EncryptionKey
	query := `SELECT * FROM encryption_keys WHERE key_arn = $1`
	err := s.db.GetContext(ctx, &key, query, arn)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &key, err
}

func (s *Store) ListEncryptionKeys(ctx context.Context, accountID uuid.UUID) ([]*models.EncryptionKey, error) {
	var keys []*models.EncryptionKey
	query := `SELECT * FROM encryption_keys WHERE account_id = $1 ORDER BY created_at DESC`
	err := s.db.SelectContext(ctx, &keys, query, accountID)
	return keys, err
}

func (s *Store) DeleteEncryptionKey(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM encryption_keys WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// =====================================================
// Encryption Key Usage Store Methods
// =====================================================

func (s *Store) CreateKeyUsage(ctx context.Context, usage *models.EncryptionKeyUsage) error {
	query := `
		INSERT INTO encryption_key_usage (
			id, key_id, asset_id, asset_arn, asset_type, usage_type,
			encryption_context, first_seen_at, last_seen_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (key_id, asset_arn) DO UPDATE SET
			last_seen_at = EXCLUDED.last_seen_at
	`

	if usage.ID == uuid.Nil {
		usage.ID = uuid.New()
	}
	now := time.Now()
	if usage.FirstSeenAt.IsZero() {
		usage.FirstSeenAt = now
	}
	usage.LastSeenAt = now

	_, err := s.db.ExecContext(ctx, query,
		usage.ID, usage.KeyID, usage.AssetID, usage.AssetARN, usage.AssetType,
		usage.UsageType, usage.EncryptionContext, usage.FirstSeenAt, usage.LastSeenAt,
	)
	return err
}

func (s *Store) UpdateKeyUsage(ctx context.Context, usage *models.EncryptionKeyUsage) error {
	query := `UPDATE encryption_key_usage SET last_seen_at = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, time.Now(), usage.ID)
	return err
}

func (s *Store) GetKeyUsageByAsset(ctx context.Context, assetID uuid.UUID) ([]*models.EncryptionKeyUsage, error) {
	var usages []*models.EncryptionKeyUsage
	query := `SELECT * FROM encryption_key_usage WHERE asset_id = $1`
	err := s.db.SelectContext(ctx, &usages, query, assetID)
	return usages, err
}

func (s *Store) GetKeyUsageByKey(ctx context.Context, keyID uuid.UUID) ([]*models.EncryptionKeyUsage, error) {
	var usages []*models.EncryptionKeyUsage
	query := `SELECT * FROM encryption_key_usage WHERE key_id = $1`
	err := s.db.SelectContext(ctx, &usages, query, keyID)
	return usages, err
}

func (s *Store) ListKeyUsage(ctx context.Context, accountID uuid.UUID) ([]*models.EncryptionKeyUsage, error) {
	var usages []*models.EncryptionKeyUsage
	query := `
		SELECT ku.* FROM encryption_key_usage ku
		JOIN encryption_keys k ON ku.key_id = k.id
		WHERE k.account_id = $1
	`
	err := s.db.SelectContext(ctx, &usages, query, accountID)
	return usages, err
}

// =====================================================
// Transit Encryption Store Methods
// =====================================================

func (s *Store) CreateTransitEncryption(ctx context.Context, transit *models.TransitEncryption) error {
	query := `
		INSERT INTO transit_encryption (
			id, asset_id, endpoint_type, endpoint_url, tls_enabled, tls_version,
			min_tls_version, certificate_arn, certificate_expiry, cipher_suites,
			supports_perfect_forward_secrecy, meets_minimum_standards, compliance_issues,
			last_checked_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (asset_id) DO UPDATE SET
			tls_enabled = EXCLUDED.tls_enabled,
			tls_version = EXCLUDED.tls_version,
			min_tls_version = EXCLUDED.min_tls_version,
			certificate_expiry = EXCLUDED.certificate_expiry,
			meets_minimum_standards = EXCLUDED.meets_minimum_standards,
			compliance_issues = EXCLUDED.compliance_issues,
			last_checked_at = EXCLUDED.last_checked_at
	`

	if transit.ID == uuid.Nil {
		transit.ID = uuid.New()
	}
	transit.LastCheckedAt = time.Now()

	_, err := s.db.ExecContext(ctx, query,
		transit.ID, transit.AssetID, transit.EndpointType, transit.EndpointURL,
		transit.TLSEnabled, transit.TLSVersion, transit.MinTLSVersion,
		transit.CertificateARN, transit.CertificateExpiry, pq.Array(transit.CipherSuites),
		transit.SupportsPerfectForwardSecrecy, transit.MeetsMinimumStandards,
		pq.Array(transit.ComplianceIssues), transit.LastCheckedAt,
	)
	return err
}

func (s *Store) UpdateTransitEncryption(ctx context.Context, transit *models.TransitEncryption) error {
	query := `
		UPDATE transit_encryption SET
			tls_enabled = $1, tls_version = $2, min_tls_version = $3,
			meets_minimum_standards = $4, compliance_issues = $5, last_checked_at = $6
		WHERE id = $7
	`
	_, err := s.db.ExecContext(ctx, query,
		transit.TLSEnabled, transit.TLSVersion, transit.MinTLSVersion,
		transit.MeetsMinimumStandards, pq.Array(transit.ComplianceIssues),
		time.Now(), transit.ID,
	)
	return err
}

func (s *Store) GetTransitEncryption(ctx context.Context, assetID uuid.UUID) (*models.TransitEncryption, error) {
	var transit models.TransitEncryption
	query := `SELECT * FROM transit_encryption WHERE asset_id = $1`
	err := s.db.GetContext(ctx, &transit, query, assetID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &transit, err
}

func (s *Store) ListTransitEncryption(ctx context.Context, accountID uuid.UUID) ([]*models.TransitEncryption, error) {
	var transits []*models.TransitEncryption
	query := `
		SELECT t.* FROM transit_encryption t
		JOIN data_assets a ON t.asset_id = a.id
		WHERE a.account_id = $1
	`
	err := s.db.SelectContext(ctx, &transits, query, accountID)
	return transits, err
}

// =====================================================
// Encryption Compliance Store Methods
// =====================================================

func (s *Store) CreateEncryptionCompliance(ctx context.Context, compliance *models.EncryptionCompliance) error {
	query := `
		INSERT INTO encryption_compliance (
			id, account_id, asset_id, compliance_score, grade,
			at_rest_score, in_transit_score, key_management_score,
			findings_count, critical_findings, compliance_details,
			recommendations, evaluated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (asset_id) DO UPDATE SET
			compliance_score = EXCLUDED.compliance_score,
			grade = EXCLUDED.grade,
			at_rest_score = EXCLUDED.at_rest_score,
			in_transit_score = EXCLUDED.in_transit_score,
			key_management_score = EXCLUDED.key_management_score,
			findings_count = EXCLUDED.findings_count,
			critical_findings = EXCLUDED.critical_findings,
			recommendations = EXCLUDED.recommendations,
			evaluated_at = EXCLUDED.evaluated_at
	`

	if compliance.ID == uuid.Nil {
		compliance.ID = uuid.New()
	}
	compliance.EvaluatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, query,
		compliance.ID, compliance.AccountID, compliance.AssetID,
		compliance.ComplianceScore, compliance.Grade,
		compliance.AtRestScore, compliance.InTransitScore, compliance.KeyManagementScore,
		compliance.FindingsCount, compliance.CriticalFindings, compliance.ComplianceDetails,
		pq.Array(compliance.Recommendations), compliance.EvaluatedAt,
	)
	return err
}

func (s *Store) UpdateEncryptionCompliance(ctx context.Context, compliance *models.EncryptionCompliance) error {
	query := `
		UPDATE encryption_compliance SET
			compliance_score = $1, grade = $2, at_rest_score = $3,
			in_transit_score = $4, key_management_score = $5,
			findings_count = $6, critical_findings = $7,
			recommendations = $8, evaluated_at = $9
		WHERE asset_id = $10
	`
	_, err := s.db.ExecContext(ctx, query,
		compliance.ComplianceScore, compliance.Grade, compliance.AtRestScore,
		compliance.InTransitScore, compliance.KeyManagementScore,
		compliance.FindingsCount, compliance.CriticalFindings,
		pq.Array(compliance.Recommendations), time.Now(), compliance.AssetID,
	)
	return err
}

func (s *Store) GetEncryptionCompliance(ctx context.Context, assetID uuid.UUID) (*models.EncryptionCompliance, error) {
	var compliance models.EncryptionCompliance
	query := `SELECT * FROM encryption_compliance WHERE asset_id = $1`
	err := s.db.GetContext(ctx, &compliance, query, assetID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &compliance, err
}

func (s *Store) ListEncryptionCompliance(ctx context.Context, accountID uuid.UUID) ([]*models.EncryptionCompliance, error) {
	var compliances []*models.EncryptionCompliance
	query := `SELECT * FROM encryption_compliance WHERE account_id = $1 ORDER BY compliance_score ASC`
	err := s.db.SelectContext(ctx, &compliances, query, accountID)
	return compliances, err
}

// =====================================================
// Lineage Events Store Methods
// =====================================================

func (s *Store) CreateLineageEvent(ctx context.Context, event *models.LineageEvent) error {
	query := `
		INSERT INTO lineage_events (
			id, account_id, source_resource_arn, source_resource_type, source_resource_name,
			target_resource_arn, target_resource_type, target_resource_name,
			flow_type, access_method, frequency, data_volume_bytes,
			inferred_from, confidence_score, evidence,
			first_observed_at, last_observed_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (source_resource_arn, target_resource_arn, flow_type) DO UPDATE SET
			last_observed_at = EXCLUDED.last_observed_at,
			confidence_score = EXCLUDED.confidence_score,
			evidence = EXCLUDED.evidence
	`

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	now := time.Now()
	if event.FirstObservedAt.IsZero() {
		event.FirstObservedAt = now
	}
	event.LastObservedAt = now
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}

	_, err := s.db.ExecContext(ctx, query,
		event.ID, event.AccountID, event.SourceResourceARN, event.SourceResourceType,
		event.SourceResourceName, event.TargetResourceARN, event.TargetResourceType,
		event.TargetResourceName, event.FlowType, event.AccessMethod, event.Frequency,
		event.DataVolumeBytes, event.InferredFrom, event.ConfidenceScore, event.Evidence,
		event.FirstObservedAt, event.LastObservedAt, event.CreatedAt,
	)
	return err
}

func (s *Store) UpdateLineageEvent(ctx context.Context, event *models.LineageEvent) error {
	query := `
		UPDATE lineage_events SET
			last_observed_at = $1, confidence_score = $2, evidence = $3
		WHERE source_resource_arn = $4 AND target_resource_arn = $5 AND flow_type = $6
	`
	_, err := s.db.ExecContext(ctx, query,
		time.Now(), event.ConfidenceScore, event.Evidence,
		event.SourceResourceARN, event.TargetResourceARN, event.FlowType,
	)
	return err
}

func (s *Store) GetLineageEvent(ctx context.Context, id uuid.UUID) (*models.LineageEvent, error) {
	var event models.LineageEvent
	query := `SELECT * FROM lineage_events WHERE id = $1`
	err := s.db.GetContext(ctx, &event, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &event, err
}

func (s *Store) ListLineageEvents(ctx context.Context, accountID uuid.UUID) ([]*models.LineageEvent, error) {
	var events []*models.LineageEvent
	query := `SELECT * FROM lineage_events WHERE account_id = $1 ORDER BY last_observed_at DESC`
	err := s.db.SelectContext(ctx, &events, query, accountID)
	return events, err
}

func (s *Store) GetLineageEventsBySource(ctx context.Context, sourceARN string) ([]*models.LineageEvent, error) {
	var events []*models.LineageEvent
	query := `SELECT * FROM lineage_events WHERE source_resource_arn = $1`
	err := s.db.SelectContext(ctx, &events, query, sourceARN)
	return events, err
}

func (s *Store) GetLineageEventsByTarget(ctx context.Context, targetARN string) ([]*models.LineageEvent, error) {
	var events []*models.LineageEvent
	query := `SELECT * FROM lineage_events WHERE target_resource_arn = $1`
	err := s.db.SelectContext(ctx, &events, query, targetARN)
	return events, err
}

func (s *Store) DeleteLineageEvent(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM lineage_events WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// =====================================================
// Lineage Paths Store Methods
// =====================================================

func (s *Store) CreateLineagePath(ctx context.Context, path *models.LineagePath) error {
	query := `
		INSERT INTO lineage_paths (
			id, account_id, origin_arn, origin_type, destination_arn, destination_type,
			path_hops, path_arns, flow_types, contains_sensitive_data,
			sensitivity_level, data_categories, risk_score, computed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	if path.ID == uuid.Nil {
		path.ID = uuid.New()
	}
	path.ComputedAt = time.Now()

	_, err := s.db.ExecContext(ctx, query,
		path.ID, path.AccountID, path.OriginARN, path.OriginType,
		path.DestinationARN, path.DestinationType, path.PathHops,
		pq.Array(path.PathARNs), pq.Array(path.FlowTypes),
		path.ContainsSensitiveData, path.SensitivityLevel,
		pq.Array(path.DataCategories), path.RiskScore, path.ComputedAt,
	)
	return err
}

func (s *Store) ListLineagePaths(ctx context.Context, accountID uuid.UUID) ([]*models.LineagePath, error) {
	var paths []*models.LineagePath
	query := `SELECT * FROM lineage_paths WHERE account_id = $1 ORDER BY risk_score DESC`
	err := s.db.SelectContext(ctx, &paths, query, accountID)
	return paths, err
}

func (s *Store) GetLineagePathsByOrigin(ctx context.Context, originARN string) ([]*models.LineagePath, error) {
	var paths []*models.LineagePath
	query := `SELECT * FROM lineage_paths WHERE origin_arn = $1`
	err := s.db.SelectContext(ctx, &paths, query, originARN)
	return paths, err
}

func (s *Store) GetLineagePathsByDestination(ctx context.Context, destARN string) ([]*models.LineagePath, error) {
	var paths []*models.LineagePath
	query := `SELECT * FROM lineage_paths WHERE destination_arn = $1`
	err := s.db.SelectContext(ctx, &paths, query, destARN)
	return paths, err
}

func (s *Store) GetSensitiveDataPaths(ctx context.Context, accountID uuid.UUID) ([]*models.LineagePath, error) {
	var paths []*models.LineagePath
	query := `SELECT * FROM lineage_paths WHERE account_id = $1 AND contains_sensitive_data = true ORDER BY risk_score DESC`
	err := s.db.SelectContext(ctx, &paths, query, accountID)
	return paths, err
}

func (s *Store) DeleteLineagePaths(ctx context.Context, accountID uuid.UUID) error {
	query := `DELETE FROM lineage_paths WHERE account_id = $1`
	_, err := s.db.ExecContext(ctx, query, accountID)
	return err
}

// =====================================================
// AI Services Store Methods
// =====================================================

func (s *Store) CreateAIService(ctx context.Context, service *models.AIService) error {
	query := `
		INSERT INTO ai_services (
			id, account_id, service_type, service_arn, service_name,
			region, service_config, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (service_arn) DO UPDATE SET
			status = EXCLUDED.status,
			service_config = EXCLUDED.service_config,
			updated_at = EXCLUDED.updated_at
	`

	if service.ID == uuid.Nil {
		service.ID = uuid.New()
	}
	now := time.Now()
	if service.CreatedAt.IsZero() {
		service.CreatedAt = now
	}
	service.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, query,
		service.ID, service.AccountID, service.ServiceType, service.ServiceARN,
		service.ServiceName, service.Region, service.ServiceConfig,
		service.Status, service.CreatedAt, service.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateAIService(ctx context.Context, service *models.AIService) error {
	query := `UPDATE ai_services SET status = $1, service_config = $2, updated_at = $3 WHERE id = $4`
	_, err := s.db.ExecContext(ctx, query, service.Status, service.ServiceConfig, time.Now(), service.ID)
	return err
}

func (s *Store) GetAIService(ctx context.Context, id uuid.UUID) (*models.AIService, error) {
	var service models.AIService
	query := `SELECT * FROM ai_services WHERE id = $1`
	err := s.db.GetContext(ctx, &service, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &service, err
}

func (s *Store) GetAIServiceByARN(ctx context.Context, arn string) (*models.AIService, error) {
	var service models.AIService
	query := `SELECT * FROM ai_services WHERE service_arn = $1`
	err := s.db.GetContext(ctx, &service, query, arn)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &service, err
}

func (s *Store) ListAIServices(ctx context.Context, accountID uuid.UUID) ([]*models.AIService, error) {
	var services []*models.AIService
	query := `SELECT * FROM ai_services WHERE account_id = $1 ORDER BY created_at DESC`
	err := s.db.SelectContext(ctx, &services, query, accountID)
	return services, err
}

func (s *Store) DeleteAIService(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ai_services WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// =====================================================
// AI Models Store Methods
// =====================================================

func (s *Store) CreateAIModel(ctx context.Context, model *models.AIModel) error {
	query := `
		INSERT INTO ai_models (
			id, service_id, account_id, model_arn, model_name, model_type,
			framework, version, description, status, endpoint_arn, tags,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (model_arn) DO UPDATE SET
			status = EXCLUDED.status,
			endpoint_arn = EXCLUDED.endpoint_arn,
			updated_at = EXCLUDED.updated_at
	`

	if model.ID == uuid.Nil {
		model.ID = uuid.New()
	}
	now := time.Now()
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
	}
	model.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, query,
		model.ID, model.ServiceID, model.AccountID, model.ModelARN, model.ModelName,
		model.ModelType, model.Framework, model.Version, model.Description,
		model.Status, model.EndpointARN, model.Tags, model.CreatedAt, model.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateAIModel(ctx context.Context, model *models.AIModel) error {
	query := `UPDATE ai_models SET status = $1, endpoint_arn = $2, updated_at = $3 WHERE id = $4`
	_, err := s.db.ExecContext(ctx, query, model.Status, model.EndpointARN, time.Now(), model.ID)
	return err
}

func (s *Store) GetAIModel(ctx context.Context, id uuid.UUID) (*models.AIModel, error) {
	var model models.AIModel
	query := `SELECT * FROM ai_models WHERE id = $1`
	err := s.db.GetContext(ctx, &model, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &model, err
}

func (s *Store) GetAIModelByARN(ctx context.Context, arn string) (*models.AIModel, error) {
	var model models.AIModel
	query := `SELECT * FROM ai_models WHERE model_arn = $1`
	err := s.db.GetContext(ctx, &model, query, arn)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &model, err
}

func (s *Store) ListAIModels(ctx context.Context, accountID uuid.UUID) ([]*models.AIModel, error) {
	var models []*models.AIModel
	query := `SELECT * FROM ai_models WHERE account_id = $1 ORDER BY created_at DESC`
	err := s.db.SelectContext(ctx, &models, query, accountID)
	return models, err
}

func (s *Store) ListAIModelsByService(ctx context.Context, serviceID uuid.UUID) ([]*models.AIModel, error) {
	var models []*models.AIModel
	query := `SELECT * FROM ai_models WHERE service_id = $1 ORDER BY created_at DESC`
	err := s.db.SelectContext(ctx, &models, query, serviceID)
	return models, err
}

func (s *Store) DeleteAIModel(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ai_models WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// =====================================================
// AI Training Data Store Methods
// =====================================================

func (s *Store) CreateAITrainingData(ctx context.Context, data *models.AITrainingData) error {
	query := `
		INSERT INTO ai_training_data (
			id, model_id, data_source_arn, data_source_type, data_path, data_format,
			sample_count, data_size_bytes, contains_sensitive_data,
			sensitivity_categories, sensitivity_level, used_at, discovered_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	if data.ID == uuid.Nil {
		data.ID = uuid.New()
	}
	now := time.Now()
	if data.UsedAt.IsZero() {
		data.UsedAt = now
	}
	if data.DiscoveredAt.IsZero() {
		data.DiscoveredAt = now
	}

	_, err := s.db.ExecContext(ctx, query,
		data.ID, data.ModelID, data.DataSourceARN, data.DataSourceType,
		data.DataPath, data.DataFormat, data.SampleCount, data.DataSizeBytes,
		data.ContainsSensitiveData, pq.Array(data.SensitivityCategories),
		data.SensitivityLevel, data.UsedAt, data.DiscoveredAt,
	)
	return err
}

func (s *Store) UpdateAITrainingData(ctx context.Context, data *models.AITrainingData) error {
	query := `
		UPDATE ai_training_data SET
			contains_sensitive_data = $1, sensitivity_categories = $2, sensitivity_level = $3
		WHERE id = $4
	`
	_, err := s.db.ExecContext(ctx, query,
		data.ContainsSensitiveData, pq.Array(data.SensitivityCategories),
		data.SensitivityLevel, data.ID,
	)
	return err
}

func (s *Store) GetAITrainingDataByModel(ctx context.Context, modelID uuid.UUID) ([]*models.AITrainingData, error) {
	var data []*models.AITrainingData
	query := `SELECT * FROM ai_training_data WHERE model_id = $1`
	err := s.db.SelectContext(ctx, &data, query, modelID)
	return data, err
}

func (s *Store) ListSensitiveTrainingData(ctx context.Context, accountID uuid.UUID) ([]*models.AITrainingData, error) {
	var data []*models.AITrainingData
	query := `
		SELECT td.* FROM ai_training_data td
		JOIN ai_models m ON td.model_id = m.id
		WHERE m.account_id = $1 AND td.contains_sensitive_data = true
	`
	err := s.db.SelectContext(ctx, &data, query, accountID)
	return data, err
}

// =====================================================
// AI Processing Events Store Methods
// =====================================================

func (s *Store) CreateAIProcessingEvent(ctx context.Context, event *models.AIProcessingEvent) error {
	query := `
		INSERT INTO ai_processing_events (
			id, account_id, service_id, model_id, event_type, event_time,
			data_source_arn, data_asset_id, accessed_sensitivity_level,
			accessed_categories, data_volume_bytes, record_count,
			principal_arn, principal_type, event_details, risk_score, risk_factors,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	_, err := s.db.ExecContext(ctx, query,
		event.ID, event.AccountID, event.ServiceID, event.ModelID,
		event.EventType, event.EventTime, event.DataSourceARN, event.DataAssetID,
		event.AccessedSensitivityLevel, pq.Array(event.AccessedCategories),
		event.DataVolumeBytes, event.RecordCount, event.PrincipalARN, event.PrincipalType,
		event.EventDetails, event.RiskScore, pq.Array(event.RiskFactors), event.CreatedAt,
	)
	return err
}

func (s *Store) GetAIProcessingEvent(ctx context.Context, id uuid.UUID) (*models.AIProcessingEvent, error) {
	var event models.AIProcessingEvent
	query := `SELECT * FROM ai_processing_events WHERE id = $1`
	err := s.db.GetContext(ctx, &event, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &event, err
}

func (s *Store) ListAIProcessingEvents(ctx context.Context, accountID uuid.UUID, limit int) ([]*models.AIProcessingEvent, error) {
	var events []*models.AIProcessingEvent
	query := `SELECT * FROM ai_processing_events WHERE account_id = $1 ORDER BY event_time DESC LIMIT $2`
	err := s.db.SelectContext(ctx, &events, query, accountID, limit)
	return events, err
}

func (s *Store) ListAIProcessingEventsByModel(ctx context.Context, modelID uuid.UUID) ([]*models.AIProcessingEvent, error) {
	var events []*models.AIProcessingEvent
	query := `SELECT * FROM ai_processing_events WHERE model_id = $1 ORDER BY event_time DESC`
	err := s.db.SelectContext(ctx, &events, query, modelID)
	return events, err
}

func (s *Store) ListSensitiveDataAccessEvents(ctx context.Context, accountID uuid.UUID) ([]*models.AIProcessingEvent, error) {
	var events []*models.AIProcessingEvent
	query := `
		SELECT * FROM ai_processing_events
		WHERE account_id = $1 AND accessed_sensitivity_level IN ('CRITICAL', 'HIGH', 'MEDIUM')
		ORDER BY event_time DESC
	`
	err := s.db.SelectContext(ctx, &events, query, accountID)
	return events, err
}

// =====================================================
// Data Asset Wrapper Methods for Phase 2 Service Interfaces
// =====================================================

// GetDataAsset returns a data asset by ID (wraps GetAsset for interface compatibility)
func (s *Store) GetDataAsset(ctx context.Context, id uuid.UUID) (*models.DataAsset, error) {
	return s.GetAsset(ctx, id)
}

// GetDataAssetByARN returns a data asset by ARN (wraps GetAssetByARN for interface compatibility)
func (s *Store) GetDataAssetByARN(ctx context.Context, arn string) (*models.DataAsset, error) {
	return s.GetAssetByARN(ctx, arn)
}

// ListDataAssets returns all data assets for an account (simplified interface)
func (s *Store) ListDataAssets(ctx context.Context, accountID uuid.UUID) ([]*models.DataAsset, error) {
	var assets []*models.DataAsset
	query := `SELECT * FROM data_assets WHERE account_id = $1 ORDER BY created_at DESC`
	err := s.db.SelectContext(ctx, &assets, query, accountID)
	return assets, err
}
