package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

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

	if provider != nil {
		args = append(args, *provider)
		query += fmt.Sprintf(" AND provider = $%d", len(args))
	}
	if status != nil {
		args = append(args, *status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
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
			sensitivity_level = COALESCE(NULLIF(EXCLUDED.sensitivity_level, ''), data_assets.sensitivity_level),
			data_categories = COALESCE(NULLIF(EXCLUDED.data_categories, '{}'), data_assets.data_categories),
			classification_count = data_assets.classification_count,
			last_scanned_at = EXCLUDED.last_scanned_at,
			scan_status = EXCLUDED.scan_status,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`

	if asset.ID == uuid.Nil {
		asset.ID = uuid.New()
	}
	now := time.Now()
	asset.CreatedAt = now
	asset.UpdatedAt = now
	asset.LastScannedAt = &now

	var actualID uuid.UUID
	err := s.db.QueryRowContext(ctx, query,
		asset.ID, asset.AccountID, asset.ResourceType, asset.ResourceARN, asset.Region, asset.Name,
		asset.EncryptionStatus, asset.EncryptionKeyARN, asset.PublicAccess, asset.PublicAccessDetails,
		asset.VersioningEnabled, asset.LoggingEnabled, asset.SizeBytes, asset.ObjectCount,
		asset.Tags, asset.RawMetadata, asset.SensitivityLevel, pq.Array(asset.DataCategories), asset.ClassificationCount,
		asset.LastScannedAt, asset.ScanStatus, asset.CreatedAt, asset.UpdatedAt,
	).Scan(&actualID)
	if err != nil {
		return err
	}
	asset.ID = actualID
	return nil
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

	if filters.AccountID != nil {
		args = append(args, *filters.AccountID)
		baseQuery += fmt.Sprintf(" AND account_id = $%d", len(args))
	}
	if filters.ResourceType != nil {
		args = append(args, *filters.ResourceType)
		baseQuery += fmt.Sprintf(" AND resource_type = $%d", len(args))
	}
	if filters.SensitivityLevel != nil {
		args = append(args, *filters.SensitivityLevel)
		baseQuery += fmt.Sprintf(" AND sensitivity_level = $%d", len(args))
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
		SET sensitivity_level = $1, data_categories = $2, classification_count = classification_count + $3, updated_at = $4
		WHERE id = $5
	`
	_, err := s.db.ExecContext(ctx, query, sensitivity, pq.Array(categories), count, time.Now(), assetID)
	return err
}

// DeleteClassificationsForAsset removes all classifications for an asset before a new scan
func (s *Store) DeleteClassificationsForAsset(ctx context.Context, assetID uuid.UUID) error {
	query := `DELETE FROM classifications WHERE asset_id = $1`
	_, err := s.db.ExecContext(ctx, query, assetID)
	return err
}

// DeleteFindingsForAsset removes all findings for an asset before a new scan
func (s *Store) DeleteFindingsForAsset(ctx context.Context, assetID uuid.UUID) error {
	query := `DELETE FROM findings WHERE asset_id = $1`
	_, err := s.db.ExecContext(ctx, query, assetID)
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

// BatchCreateClassifications inserts multiple classifications at once using COPY for maximum performance.
// This can insert 10,000+ rows per second compared to ~100 rows/sec with individual inserts.
func (s *Store) BatchCreateClassifications(ctx context.Context, classifications []*models.Classification) error {
	if len(classifications) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create temp table for bulk load
	_, err = tx.ExecContext(ctx, `
		CREATE TEMP TABLE temp_classifications (
			id UUID, asset_id UUID, object_path TEXT, object_size BIGINT,
			rule_name TEXT, category TEXT, sensitivity TEXT,
			finding_count INT, sample_matches JSONB, match_locations JSONB,
			confidence_score FLOAT, validated BOOLEAN, discovered_at TIMESTAMP
		) ON COMMIT DROP
	`)
	if err != nil {
		return fmt.Errorf("create temp table: %w", err)
	}

	// Use COPY for bulk insert into temp table
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("temp_classifications",
		"id", "asset_id", "object_path", "object_size",
		"rule_name", "category", "sensitivity",
		"finding_count", "sample_matches", "match_locations",
		"confidence_score", "validated", "discovered_at",
	))
	if err != nil {
		return fmt.Errorf("prepare copy: %w", err)
	}

	now := time.Now()
	for _, c := range classifications {
		if c.ID == uuid.Nil {
			c.ID = uuid.New()
		}
		c.DiscoveredAt = now

		// Marshal JSONB fields to JSON strings for COPY command
		var sampleMatchesJSON, matchLocationsJSON []byte
		if c.SampleMatches != nil {
			sampleMatchesJSON, _ = json.Marshal(c.SampleMatches)
		} else {
			sampleMatchesJSON = []byte("{}")
		}
		if c.MatchLocations != nil {
			matchLocationsJSON, _ = json.Marshal(c.MatchLocations)
		} else {
			matchLocationsJSON = []byte("{}")
		}

		_, err = stmt.ExecContext(ctx,
			c.ID, c.AssetID, c.ObjectPath, c.ObjectSize,
			c.RuleName, string(c.Category), string(c.Sensitivity),
			c.FindingCount, string(sampleMatchesJSON), string(matchLocationsJSON),
			c.ConfidenceScore, c.Validated, c.DiscoveredAt,
		)
		if err != nil {
			stmt.Close()
			return fmt.Errorf("copy row: %w", err)
		}
	}

	if err = stmt.Close(); err != nil {
		return fmt.Errorf("close copy: %w", err)
	}

	// Upsert from temp table to main table
	_, err = tx.ExecContext(ctx, `
		INSERT INTO classifications (
			id, asset_id, object_path, object_size,
			rule_name, category, sensitivity,
			finding_count, sample_matches, match_locations,
			confidence_score, validated, discovered_at
		)
		SELECT * FROM temp_classifications
		ON CONFLICT (asset_id, object_path, rule_name) DO UPDATE SET
			finding_count = EXCLUDED.finding_count,
			sample_matches = EXCLUDED.sample_matches,
			match_locations = EXCLUDED.match_locations,
			confidence_score = EXCLUDED.confidence_score
	`)
	if err != nil {
		return fmt.Errorf("upsert from temp: %w", err)
	}

	return tx.Commit()
}

func (s *Store) ListClassificationsByAsset(ctx context.Context, assetID uuid.UUID) ([]models.Classification, error) {
	var classifications []models.Classification
	query := `SELECT * FROM classifications WHERE asset_id = $1 ORDER BY sensitivity DESC, discovered_at DESC`
	err := s.db.SelectContext(ctx, &classifications, query, assetID)
	return classifications, err
}

// ClassificationWithAsset is a classification with asset name included
type ClassificationWithAsset struct {
	models.Classification
	AssetName string `json:"asset_name" db:"asset_name"`
}

func (s *Store) ListAllClassifications(ctx context.Context, limit, offset int) ([]ClassificationWithAsset, int, error) {
	var classifications []ClassificationWithAsset
	var total int

	// Get total finding count (sum of all finding_count values to match dashboard)
	err := s.db.GetContext(ctx, &total, `SELECT COALESCE(SUM(finding_count), 0) FROM classifications`)
	if err != nil {
		return nil, 0, err
	}

	// Get classifications with limit/offset, including asset name
	query := `
		SELECT c.*, COALESCE(a.name, '') as asset_name
		FROM classifications c
		LEFT JOIN data_assets a ON c.asset_id = a.id
		ORDER BY c.sensitivity DESC, c.discovered_at DESC
		LIMIT $1 OFFSET $2
	`
	err = s.db.SelectContext(ctx, &classifications, query, limit, offset)
	return classifications, total, err
}

func (s *Store) GetClassification(ctx context.Context, id uuid.UUID) (*models.Classification, error) {
	var classification models.Classification
	query := `SELECT * FROM classifications WHERE id = $1`
	err := s.db.GetContext(ctx, &classification, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &classification, err
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
		pq.Array(finding.ComplianceFrameworks), finding.Evidence, finding.ResourceSnapshot,
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

	if filters.AccountID != nil {
		args = append(args, *filters.AccountID)
		baseQuery += fmt.Sprintf(" AND account_id = $%d", len(args))
	}
	if filters.AssetID != nil {
		args = append(args, *filters.AssetID)
		baseQuery += fmt.Sprintf(" AND asset_id = $%d", len(args))
	}
	if filters.Severity != nil {
		args = append(args, *filters.Severity)
		baseQuery += fmt.Sprintf(" AND severity = $%d", len(args))
	}
	if filters.Status != nil {
		args = append(args, *filters.Status)
		baseQuery += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if filters.FindingType != nil {
		args = append(args, *filters.FindingType)
		baseQuery += fmt.Sprintf(" AND finding_type = $%d", len(args))
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
		ORDER BY scheduled_at ASC NULLS LAST
		LIMIT $2
	`
	err := s.db.SelectContext(ctx, &jobs, query, models.ScanStatusPending, limit)
	return jobs, err
}

func (s *Store) ListAllScanJobs(ctx context.Context, limit int) ([]models.ScanJob, error) {
	var jobs []models.ScanJob
	query := `
		SELECT * FROM scan_jobs
		ORDER BY started_at DESC NULLS LAST, scheduled_at DESC NULLS LAST
		LIMIT $1
	`
	err := s.db.SelectContext(ctx, &jobs, query, limit)
	return jobs, err
}

func (s *Store) ClearStuckScanJobs(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE scan_jobs
		SET status = 'failed', completed_at = NOW()
		WHERE status IN ('running', 'pending')
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) CancelScanJob(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE scan_jobs
		SET status = 'cancelled', completed_at = NOW()
		WHERE id = $1
	`, id)
	return err
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
	HighFindings     int `db:"high_findings"`
	MediumFindings   int `db:"medium_findings"`
	LowFindings      int `db:"low_findings"`
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
			(SELECT COUNT(*) FROM findings WHERE severity = 'CRITICAL') AS critical_findings,
			(SELECT COUNT(*) FROM findings WHERE severity = 'HIGH') AS high_findings,
			(SELECT COUNT(*) FROM findings WHERE severity = 'MEDIUM') AS medium_findings,
			(SELECT COUNT(*) FROM findings WHERE severity = 'LOW') AS low_findings
	`

	err := s.db.GetContext(ctx, counts, query)
	if err != nil {
		return nil, fmt.Errorf("getting dashboard counts: %w", err)
	}

	return counts, nil
}

// ClearAllClassifications deletes all classifications from the database
func (s *Store) ClearAllClassifications(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM classifications`)
	if err != nil {
		return 0, err
	}
	// Also reset asset classification counts
	_, err = s.db.ExecContext(ctx, `
		UPDATE data_assets SET
			sensitivity_level = 'UNKNOWN',
			data_categories = '{}',
			classification_count = 0
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ClearAllFindings deletes all findings from the database
func (s *Store) ClearAllFindings(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM findings`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ClearAllScanData clears classifications, findings, and resets assets
func (s *Store) ClearAllScanData(ctx context.Context) (map[string]int64, error) {
	counts := make(map[string]int64)

	// Clear classifications first (may have FK to assets)
	classCount, err := s.ClearAllClassifications(ctx)
	if err != nil {
		return nil, fmt.Errorf("clearing classifications: %w", err)
	}
	counts["classifications"] = classCount

	// Clear findings
	findingCount, err := s.ClearAllFindings(ctx)
	if err != nil {
		return nil, fmt.Errorf("clearing findings: %w", err)
	}
	counts["findings"] = findingCount

	// Clear scan jobs
	result, err := s.db.ExecContext(ctx, `DELETE FROM scan_jobs`)
	if err != nil {
		return nil, fmt.Errorf("clearing scan jobs: %w", err)
	}
	scanCount, _ := result.RowsAffected()
	counts["scan_jobs"] = scanCount

	return counts, nil
}
