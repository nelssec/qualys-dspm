package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/qualys/dspm/internal/models"
)

type Store struct {
	db *sqlx.DB
}

type Config struct {
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
}

func New(cfg Config) (*Store, error) {
	db, err := sqlx.Connect("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) DB() *sqlx.DB {
	return s.db
}

func (s *Store) CreateAccount(ctx context.Context, account *models.CloudAccount) error {
	query := `
		INSERT INTO cloud_accounts (id, provider, external_id, display_name, connector_config, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	account.ID = uuid.New()
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()
	if account.Status == "" {
		account.Status = "active"
	}

	_, err := s.db.ExecContext(ctx, query,
		account.ID,
		account.Provider,
		account.ExternalID,
		account.DisplayName,
		account.ConnectorConfig,
		account.Status,
		account.CreatedAt,
		account.UpdatedAt,
	)
	return err
}

func (s *Store) GetAccount(ctx context.Context, id uuid.UUID) (*models.CloudAccount, error) {
	var account models.CloudAccount
	query := `SELECT * FROM cloud_accounts WHERE id = $1`
	err := s.db.GetContext(ctx, &account, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &account, err
}

func (s *Store) GetAccountByExternalID(ctx context.Context, provider models.Provider, externalID string) (*models.CloudAccount, error) {
	var account models.CloudAccount
	query := `SELECT * FROM cloud_accounts WHERE provider = $1 AND external_id = $2`
	err := s.db.GetContext(ctx, &account, query, provider, externalID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &account, err
}

func (s *Store) ListAccounts(ctx context.Context, provider *models.Provider, status *string) ([]models.CloudAccount, error) {
	query := `SELECT * FROM cloud_accounts WHERE 1=1`
	args := make([]interface{}, 0)
	argIdx := 1

	if provider != nil {
		query += fmt.Sprintf(" AND provider = $%d", argIdx)
		args = append(args, *provider)
		argIdx++
	}
	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC"

	var accounts []models.CloudAccount
	err := s.db.SelectContext(ctx, &accounts, query, args...)
	return accounts, err
}

func (s *Store) UpdateAccountStatus(ctx context.Context, id uuid.UUID, status, message string) error {
	query := `UPDATE cloud_accounts SET status = $1, status_message = $2, updated_at = $3 WHERE id = $4`
	_, err := s.db.ExecContext(ctx, query, status, message, time.Now(), id)
	return err
}

func (s *Store) UpdateAccountLastScan(ctx context.Context, id uuid.UUID, scanStatus string) error {
	query := `UPDATE cloud_accounts SET last_scan_at = $1, last_scan_status = $2, updated_at = $1 WHERE id = $3`
	_, err := s.db.ExecContext(ctx, query, time.Now(), scanStatus, id)
	return err
}

func (s *Store) DeleteAccount(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM cloud_accounts WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *Store) UpsertAsset(ctx context.Context, asset *models.DataAsset) error {
	query := `
		INSERT INTO data_assets (
			id, account_id, resource_type, resource_arn, region, name,
			encryption_status, encryption_key_arn, public_access, public_access_details,
			versioning_enabled, logging_enabled, size_bytes, object_count,
			tags, raw_metadata, sensitivity_level, data_categories, classification_count,
			last_scanned_at, scan_status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)
		ON CONFLICT (resource_arn) DO UPDATE SET
			encryption_status = EXCLUDED.encryption_status,
			encryption_key_arn = EXCLUDED.encryption_key_arn,
			public_access = EXCLUDED.public_access,
			public_access_details = EXCLUDED.public_access_details,
			versioning_enabled = EXCLUDED.versioning_enabled,
			logging_enabled = EXCLUDED.logging_enabled,
			size_bytes = EXCLUDED.size_bytes,
			object_count = EXCLUDED.object_count,
			tags = EXCLUDED.tags,
			raw_metadata = EXCLUDED.raw_metadata,
			sensitivity_level = EXCLUDED.sensitivity_level,
			data_categories = EXCLUDED.data_categories,
			classification_count = EXCLUDED.classification_count,
			last_scanned_at = EXCLUDED.last_scanned_at,
			scan_status = EXCLUDED.scan_status,
			updated_at = EXCLUDED.updated_at
	`

	if asset.ID == uuid.Nil {
		asset.ID = uuid.New()
	}
	now := time.Now()
	asset.CreatedAt = now
	asset.UpdatedAt = now
	asset.LastScannedAt = &now

	_, err := s.db.ExecContext(ctx, query,
		asset.ID, asset.AccountID, asset.ResourceType, asset.ResourceARN, asset.Region, asset.Name,
		asset.EncryptionStatus, asset.EncryptionKeyARN, asset.PublicAccess, asset.PublicAccessDetails,
		asset.VersioningEnabled, asset.LoggingEnabled, asset.SizeBytes, asset.ObjectCount,
		asset.Tags, asset.RawMetadata, asset.SensitivityLevel, asset.DataCategories, asset.ClassificationCount,
		asset.LastScannedAt, asset.ScanStatus, asset.CreatedAt, asset.UpdatedAt,
	)
	return err
}

func (s *Store) GetAsset(ctx context.Context, id uuid.UUID) (*models.DataAsset, error) {
	var asset models.DataAsset
	query := `SELECT * FROM data_assets WHERE id = $1`
	err := s.db.GetContext(ctx, &asset, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &asset, err
}

func (s *Store) GetAssetByARN(ctx context.Context, arn string) (*models.DataAsset, error) {
	var asset models.DataAsset
	query := `SELECT * FROM data_assets WHERE resource_arn = $1`
	err := s.db.GetContext(ctx, &asset, query, arn)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &asset, err
}

type ListAssetFilters struct {
	AccountID        *uuid.UUID
	ResourceType     *models.ResourceType
	SensitivityLevel *models.Sensitivity
	PublicOnly       bool
	Limit            int
	Offset           int
}

func (s *Store) ListAssets(ctx context.Context, filters ListAssetFilters) ([]models.DataAsset, int, error) {
	baseQuery := `FROM data_assets WHERE 1=1`
	args := make([]interface{}, 0)
	argIdx := 1

	if filters.AccountID != nil {
		baseQuery += fmt.Sprintf(" AND account_id = $%d", argIdx)
		args = append(args, *filters.AccountID)
		argIdx++
	}
	if filters.ResourceType != nil {
		baseQuery += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, *filters.ResourceType)
		argIdx++
	}
	if filters.SensitivityLevel != nil {
		baseQuery += fmt.Sprintf(" AND sensitivity_level = $%d", argIdx)
		args = append(args, *filters.SensitivityLevel)
		argIdx++
	}
	if filters.PublicOnly {
		baseQuery += " AND public_access = true"
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := s.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	selectQuery := "SELECT * " + baseQuery + " ORDER BY sensitivity_level DESC, updated_at DESC"
	if filters.Limit > 0 {
		selectQuery += fmt.Sprintf(" LIMIT %d", filters.Limit)
	}
	if filters.Offset > 0 {
		selectQuery += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	var assets []models.DataAsset
	if err := s.db.SelectContext(ctx, &assets, selectQuery, args...); err != nil {
		return nil, 0, err
	}

	return assets, total, nil
}

func (s *Store) UpdateAssetClassification(ctx context.Context, assetID uuid.UUID, sensitivity models.Sensitivity, categories []string, count int) error {
	query := `
		UPDATE data_assets
		SET sensitivity_level = $1, data_categories = $2, classification_count = $3, updated_at = $4
		WHERE id = $5
	`
	_, err := s.db.ExecContext(ctx, query, sensitivity, categories, count, time.Now(), assetID)
	return err
}

func (s *Store) CreateClassification(ctx context.Context, classification *models.Classification) error {
	query := `
		INSERT INTO classifications (
			id, asset_id, object_path, object_size, rule_name, category, sensitivity,
			finding_count, sample_matches, match_locations, confidence_score, validated, discovered_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (asset_id, object_path, rule_name) DO UPDATE SET
			finding_count = EXCLUDED.finding_count,
			sample_matches = EXCLUDED.sample_matches,
			match_locations = EXCLUDED.match_locations,
			confidence_score = EXCLUDED.confidence_score
	`

	if classification.ID == uuid.Nil {
		classification.ID = uuid.New()
	}
	classification.DiscoveredAt = time.Now()

	_, err := s.db.ExecContext(ctx, query,
		classification.ID, classification.AssetID, classification.ObjectPath, classification.ObjectSize,
		classification.RuleName, classification.Category, classification.Sensitivity,
		classification.FindingCount, classification.SampleMatches, classification.MatchLocations,
		classification.ConfidenceScore, classification.Validated, classification.DiscoveredAt,
	)
	return err
}

func (s *Store) ListClassificationsByAsset(ctx context.Context, assetID uuid.UUID) ([]models.Classification, error) {
	var classifications []models.Classification
	query := `SELECT * FROM classifications WHERE asset_id = $1 ORDER BY sensitivity DESC, discovered_at DESC`
	err := s.db.SelectContext(ctx, &classifications, query, assetID)
	return classifications, err
}

func (s *Store) GetClassificationStats(ctx context.Context, accountID *uuid.UUID) (map[string]int, error) {
	query := `
		SELECT category, SUM(finding_count) as count
		FROM classifications c
		JOIN data_assets a ON c.asset_id = a.id
		WHERE 1=1
	`
	args := make([]interface{}, 0)
	if accountID != nil {
		query += " AND a.account_id = $1"
		args = append(args, *accountID)
	}
	query += " GROUP BY category"

	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, err
		}
		stats[category] = count
	}
	return stats, nil
}

func (s *Store) CreateFinding(ctx context.Context, finding *models.Finding) error {
	query := `
		INSERT INTO findings (
			id, account_id, asset_id, finding_type, severity, title, description, remediation,
			status, compliance_frameworks, evidence, resource_snapshot,
			created_at, updated_at, first_seen_at, last_seen_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	if finding.ID == uuid.Nil {
		finding.ID = uuid.New()
	}
	now := time.Now()
	finding.CreatedAt = now
	finding.UpdatedAt = now
	finding.FirstSeenAt = now
	finding.LastSeenAt = now

	_, err := s.db.ExecContext(ctx, query,
		finding.ID, finding.AccountID, finding.AssetID, finding.FindingType, finding.Severity,
		finding.Title, finding.Description, finding.Remediation, finding.Status,
		finding.ComplianceFrameworks, finding.Evidence, finding.ResourceSnapshot,
		finding.CreatedAt, finding.UpdatedAt, finding.FirstSeenAt, finding.LastSeenAt,
	)
	return err
}

func (s *Store) GetFinding(ctx context.Context, id uuid.UUID) (*models.Finding, error) {
	var finding models.Finding
	query := `SELECT * FROM findings WHERE id = $1`
	err := s.db.GetContext(ctx, &finding, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &finding, err
}

type ListFindingFilters struct {
	AccountID   *uuid.UUID
	AssetID     *uuid.UUID
	Severity    *models.FindingSeverity
	Status      *models.FindingStatus
	FindingType *string
	Limit       int
	Offset      int
}

func (s *Store) ListFindings(ctx context.Context, filters ListFindingFilters) ([]models.Finding, int, error) {
	baseQuery := `FROM findings WHERE 1=1`
	args := make([]interface{}, 0)
	argIdx := 1

	if filters.AccountID != nil {
		baseQuery += fmt.Sprintf(" AND account_id = $%d", argIdx)
		args = append(args, *filters.AccountID)
		argIdx++
	}
	if filters.AssetID != nil {
		baseQuery += fmt.Sprintf(" AND asset_id = $%d", argIdx)
		args = append(args, *filters.AssetID)
		argIdx++
	}
	if filters.Severity != nil {
		baseQuery += fmt.Sprintf(" AND severity = $%d", argIdx)
		args = append(args, *filters.Severity)
		argIdx++
	}
	if filters.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.FindingType != nil {
		baseQuery += fmt.Sprintf(" AND finding_type = $%d", argIdx)
		args = append(args, *filters.FindingType)
		_ = argIdx
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := s.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	selectQuery := "SELECT * " + baseQuery + " ORDER BY severity DESC, created_at DESC"
	if filters.Limit > 0 {
		selectQuery += fmt.Sprintf(" LIMIT %d", filters.Limit)
	}
	if filters.Offset > 0 {
		selectQuery += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	var findings []models.Finding
	if err := s.db.SelectContext(ctx, &findings, selectQuery, args...); err != nil {
		return nil, 0, err
	}

	return findings, total, nil
}

func (s *Store) UpdateFindingStatus(ctx context.Context, id uuid.UUID, status models.FindingStatus, reason string) error {
	query := `UPDATE findings SET status = $1, status_reason = $2, updated_at = $3`
	args := []interface{}{status, reason, time.Now()}

	if status == models.FindingStatusResolved {
		query += ", resolved_at = $4 WHERE id = $5"
		now := time.Now()
		args = append(args, now, id)
	} else {
		query += " WHERE id = $4"
		args = append(args, id)
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *Store) GetFindingStats(ctx context.Context, accountID *uuid.UUID) (map[string]map[string]int, error) {
	query := `
		SELECT severity, status, COUNT(*) as count
		FROM findings
		WHERE 1=1
	`
	args := make([]interface{}, 0)
	if accountID != nil {
		query += " AND account_id = $1"
		args = append(args, *accountID)
	}
	query += " GROUP BY severity, status"

	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]map[string]int)
	for rows.Next() {
		var severity, status string
		var count int
		if err := rows.Scan(&severity, &status, &count); err != nil {
			return nil, err
		}
		if stats[severity] == nil {
			stats[severity] = make(map[string]int)
		}
		stats[severity][status] = count
	}
	return stats, nil
}

func (s *Store) CreateScanJob(ctx context.Context, job *models.ScanJob) error {
	query := `
		INSERT INTO scan_jobs (
			id, account_id, scan_type, scan_scope, status, triggered_by, scheduled_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	job.Status = models.ScanStatusPending

	_, err := s.db.ExecContext(ctx, query,
		job.ID, job.AccountID, job.ScanType, job.ScanScope, job.Status, job.TriggeredBy, job.ScheduledAt,
	)
	return err
}

func (s *Store) GetScanJob(ctx context.Context, id uuid.UUID) (*models.ScanJob, error) {
	var job models.ScanJob
	query := `SELECT * FROM scan_jobs WHERE id = $1`
	err := s.db.GetContext(ctx, &job, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &job, err
}

func (s *Store) UpdateScanJobStatus(ctx context.Context, id uuid.UUID, status models.ScanStatus, workerID string) error {
	query := `UPDATE scan_jobs SET status = $1, worker_id = $2`
	args := []interface{}{status, workerID}

	switch status {
	case models.ScanStatusRunning:
		query += ", started_at = $3 WHERE id = $4"
		args = append(args, time.Now(), id)
	case models.ScanStatusCompleted, models.ScanStatusFailed:
		query += ", completed_at = $3 WHERE id = $4"
		args = append(args, time.Now(), id)
	default:
		query += " WHERE id = $3"
		args = append(args, id)
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *Store) UpdateScanJobProgress(ctx context.Context, id uuid.UUID, scanned, findings, classifications int) error {
	query := `
		UPDATE scan_jobs
		SET scanned_assets = $1, findings_count = $2, classifications_count = $3
		WHERE id = $4
	`
	_, err := s.db.ExecContext(ctx, query, scanned, findings, classifications, id)
	return err
}

func (s *Store) ListPendingScanJobs(ctx context.Context, limit int) ([]models.ScanJob, error) {
	var jobs []models.ScanJob
	query := `
		SELECT * FROM scan_jobs
		WHERE status = $1
		ORDER BY scheduled_at ASC NULLS LAST, created_at ASC
		LIMIT $2
	`
	err := s.db.SelectContext(ctx, &jobs, query, models.ScanStatusPending, limit)
	return jobs, err
}

type DashboardCounts struct {
	TotalAccounts    int `db:"total_accounts"`
	ActiveAccounts   int `db:"active_accounts"`
	TotalAssets      int `db:"total_assets"`
	PublicAssets     int `db:"public_assets"`
	CriticalAssets   int `db:"critical_assets"`
	TotalFindings    int `db:"total_findings"`
	OpenFindings     int `db:"open_findings"`
	CriticalFindings int `db:"critical_findings"`
}

func (s *Store) GetDashboardCounts(ctx context.Context) (*DashboardCounts, error) {
	counts := &DashboardCounts{}

	query := `
		SELECT
			(SELECT COUNT(*) FROM cloud_accounts) AS total_accounts,
			(SELECT COUNT(*) FROM cloud_accounts WHERE status = 'active') AS active_accounts,
			(SELECT COUNT(*) FROM data_assets) AS total_assets,
			(SELECT COUNT(*) FROM data_assets WHERE public_access = true) AS public_assets,
			(SELECT COUNT(*) FROM data_assets WHERE sensitivity_level = 'CRITICAL') AS critical_assets,
			(SELECT COUNT(*) FROM findings) AS total_findings,
			(SELECT COUNT(*) FROM findings WHERE status = 'open') AS open_findings,
			(SELECT COUNT(*) FROM findings WHERE severity = 'CRITICAL') AS critical_findings
	`

	err := s.db.GetContext(ctx, counts, query)
	if err != nil {
		return nil, fmt.Errorf("getting dashboard counts: %w", err)
	}

	return counts, nil
}
