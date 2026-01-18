package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/qualys/dspm/internal/anomaly"
)

// CreateAnomaly creates a new anomaly record
func (s *Store) CreateAnomaly(ctx context.Context, a *anomaly.Anomaly) error {
	detailsJSON, err := json.Marshal(a.Details)
	if err != nil {
		return fmt.Errorf("marshaling details: %w", err)
	}

	query := `
		INSERT INTO anomalies (
			id, account_id, asset_id, principal_id, principal_type, principal_name,
			anomaly_type, status, severity, title, description, details,
			baseline_value, observed_value, deviation_factor, detected_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err = s.db.ExecContext(ctx, query,
		a.ID,
		a.AccountID,
		a.AssetID,
		a.PrincipalID,
		a.PrincipalType,
		a.PrincipalName,
		a.AnomalyType,
		a.Status,
		a.Severity,
		a.Title,
		a.Description,
		detailsJSON,
		a.BaselineValue,
		a.ObservedValue,
		a.DeviationFactor,
		a.DetectedAt,
		a.CreatedAt,
		a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting anomaly: %w", err)
	}

	return nil
}

// GetAnomaly retrieves an anomaly by ID
func (s *Store) GetAnomaly(ctx context.Context, id uuid.UUID) (*anomaly.Anomaly, error) {
	query := `
		SELECT id, account_id, asset_id, principal_id, principal_type, principal_name,
			   anomaly_type, status, severity, title, description, details,
			   baseline_value, observed_value, deviation_factor, detected_at,
			   resolved_at, resolved_by, resolution, created_at, updated_at
		FROM anomalies
		WHERE id = $1
	`

	var a anomaly.Anomaly
	var detailsJSON []byte
	var assetID sql.NullString
	var resolvedAt sql.NullTime
	var resolvedBy, resolution sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID,
		&a.AccountID,
		&assetID,
		&a.PrincipalID,
		&a.PrincipalType,
		&a.PrincipalName,
		&a.AnomalyType,
		&a.Status,
		&a.Severity,
		&a.Title,
		&a.Description,
		&detailsJSON,
		&a.BaselineValue,
		&a.ObservedValue,
		&a.DeviationFactor,
		&a.DetectedAt,
		&resolvedAt,
		&resolvedBy,
		&resolution,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("anomaly not found")
	}
	if err != nil {
		return nil, fmt.Errorf("querying anomaly: %w", err)
	}

	if assetID.Valid {
		aid, _ := uuid.Parse(assetID.String)
		a.AssetID = &aid
	}
	if len(detailsJSON) > 0 {
		json.Unmarshal(detailsJSON, &a.Details)
	}
	if resolvedAt.Valid {
		a.ResolvedAt = &resolvedAt.Time
	}
	if resolvedBy.Valid {
		a.ResolvedBy = resolvedBy.String
	}
	if resolution.Valid {
		a.Resolution = resolution.String
	}

	return &a, nil
}

// UpdateAnomaly updates an anomaly record
func (s *Store) UpdateAnomaly(ctx context.Context, a *anomaly.Anomaly) error {
	query := `
		UPDATE anomalies SET
			status = $2,
			resolved_at = $3,
			resolved_by = $4,
			resolution = $5,
			updated_at = $6
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query,
		a.ID,
		a.Status,
		a.ResolvedAt,
		a.ResolvedBy,
		a.Resolution,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("updating anomaly: %w", err)
	}

	return nil
}

// ListAnomalies lists anomalies with optional filtering
func (s *Store) ListAnomalies(ctx context.Context, accountID uuid.UUID, status *anomaly.AnomalyStatus, anomalyType *anomaly.AnomalyType, limit, offset int) ([]anomaly.Anomaly, int, error) {
	var args []interface{}
	argIdx := 1

	whereClause := fmt.Sprintf("WHERE account_id = $%d", argIdx)
	args = append(args, accountID)
	argIdx++

	if status != nil && *status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}

	if anomalyType != nil && *anomalyType != "" {
		whereClause += fmt.Sprintf(" AND anomaly_type = $%d", argIdx)
		args = append(args, *anomalyType)
		argIdx++
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM anomalies %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting anomalies: %w", err)
	}

	// List query
	query := fmt.Sprintf(`
		SELECT id, account_id, asset_id, principal_id, principal_type, principal_name,
			   anomaly_type, status, severity, title, description, details,
			   baseline_value, observed_value, deviation_factor, detected_at,
			   resolved_at, resolved_by, resolution, created_at, updated_at
		FROM anomalies
		%s
		ORDER BY detected_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying anomalies: %w", err)
	}
	defer rows.Close()

	var anomalies []anomaly.Anomaly
	for rows.Next() {
		var a anomaly.Anomaly
		var detailsJSON []byte
		var assetID sql.NullString
		var resolvedAt sql.NullTime
		var resolvedBy, resolution sql.NullString

		err := rows.Scan(
			&a.ID,
			&a.AccountID,
			&assetID,
			&a.PrincipalID,
			&a.PrincipalType,
			&a.PrincipalName,
			&a.AnomalyType,
			&a.Status,
			&a.Severity,
			&a.Title,
			&a.Description,
			&detailsJSON,
			&a.BaselineValue,
			&a.ObservedValue,
			&a.DeviationFactor,
			&a.DetectedAt,
			&resolvedAt,
			&resolvedBy,
			&resolution,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning anomaly: %w", err)
		}

		if assetID.Valid {
			aid, _ := uuid.Parse(assetID.String)
			a.AssetID = &aid
		}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &a.Details)
		}
		if resolvedAt.Valid {
			a.ResolvedAt = &resolvedAt.Time
		}
		if resolvedBy.Valid {
			a.ResolvedBy = resolvedBy.String
		}
		if resolution.Valid {
			a.Resolution = resolution.String
		}

		anomalies = append(anomalies, a)
	}

	return anomalies, total, nil
}

// GetAnomalySummary returns a summary of anomalies
func (s *Store) GetAnomalySummary(ctx context.Context, accountID uuid.UUID) (*anomaly.AnomalySummary, error) {
	summary := &anomaly.AnomalySummary{
		BySeverity: make(map[string]int),
		ByType:     make(map[string]int),
	}

	// Count by status
	statusQuery := `
		SELECT status, COUNT(*)
		FROM anomalies
		WHERE account_id = $1
		GROUP BY status
	`
	rows, err := s.db.QueryContext(ctx, statusQuery, accountID)
	if err != nil {
		return nil, fmt.Errorf("querying status counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scanning status count: %w", err)
		}
		summary.TotalAnomalies += count
		switch anomaly.AnomalyStatus(status) {
		case anomaly.StatusNew:
			summary.NewCount = count
		case anomaly.StatusInvestigating:
			summary.InvestigatingCount = count
		case anomaly.StatusConfirmed:
			summary.ConfirmedCount = count
		}
	}

	// Count by severity
	severityQuery := `
		SELECT severity, COUNT(*)
		FROM anomalies
		WHERE account_id = $1
		GROUP BY severity
	`
	rows, err = s.db.QueryContext(ctx, severityQuery, accountID)
	if err != nil {
		return nil, fmt.Errorf("querying severity counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("scanning severity count: %w", err)
		}
		summary.BySeverity[severity] = count
	}

	// Count by type
	typeQuery := `
		SELECT anomaly_type, COUNT(*)
		FROM anomalies
		WHERE account_id = $1
		GROUP BY anomaly_type
	`
	rows, err = s.db.QueryContext(ctx, typeQuery, accountID)
	if err != nil {
		return nil, fmt.Errorf("querying type counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var anomalyType string
		var count int
		if err := rows.Scan(&anomalyType, &count); err != nil {
			return nil, fmt.Errorf("scanning type count: %w", err)
		}
		summary.ByType[anomalyType] = count
	}

	// Get recent anomalies
	recentQuery := `
		SELECT id, account_id, principal_id, principal_name, anomaly_type, status, severity, title, detected_at
		FROM anomalies
		WHERE account_id = $1
		ORDER BY detected_at DESC
		LIMIT 10
	`
	rows, err = s.db.QueryContext(ctx, recentQuery, accountID)
	if err != nil {
		return nil, fmt.Errorf("querying recent anomalies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var a anomaly.Anomaly
		err := rows.Scan(
			&a.ID,
			&a.AccountID,
			&a.PrincipalID,
			&a.PrincipalName,
			&a.AnomalyType,
			&a.Status,
			&a.Severity,
			&a.Title,
			&a.DetectedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning recent anomaly: %w", err)
		}
		summary.RecentAnomalies = append(summary.RecentAnomalies, a)
	}

	return summary, nil
}

// CreateBaseline creates a new access baseline
func (s *Store) CreateBaseline(ctx context.Context, baseline *anomaly.AccessBaseline) error {
	normalHoursJSON, _ := json.Marshal(baseline.NormalAccessHours)
	normalDaysJSON, _ := json.Marshal(baseline.NormalAccessDays)
	commonAssetsJSON, _ := json.Marshal(baseline.CommonAssets)
	commonOpsJSON, _ := json.Marshal(baseline.CommonOperations)
	commonIPsJSON, _ := json.Marshal(baseline.CommonSourceIPs)
	commonGeoJSON, _ := json.Marshal(baseline.CommonGeoLocations)

	query := `
		INSERT INTO access_baselines (
			id, account_id, principal_id, principal_type, principal_name,
			time_window, baseline_period_start, baseline_period_end,
			avg_daily_access_count, std_dev_access_count,
			avg_data_volume_bytes, std_dev_data_volume,
			normal_access_hours, normal_access_days,
			common_assets, common_operations, common_source_ips, common_geo_locations,
			access_count_threshold, data_volume_threshold,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`

	_, err := s.db.ExecContext(ctx, query,
		baseline.ID,
		baseline.AccountID,
		baseline.PrincipalID,
		baseline.PrincipalType,
		baseline.PrincipalName,
		baseline.TimeWindow,
		baseline.BaselinePeriodStart,
		baseline.BaselinePeriodEnd,
		baseline.AvgDailyAccessCount,
		baseline.StdDevAccessCount,
		baseline.AvgDataVolumeBytes,
		baseline.StdDevDataVolume,
		normalHoursJSON,
		normalDaysJSON,
		commonAssetsJSON,
		commonOpsJSON,
		commonIPsJSON,
		commonGeoJSON,
		baseline.AccessCountThreshold,
		baseline.DataVolumeThreshold,
		baseline.CreatedAt,
		baseline.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting baseline: %w", err)
	}

	return nil
}

// GetBaseline retrieves a baseline for a principal
func (s *Store) GetBaseline(ctx context.Context, accountID uuid.UUID, principalID string) (*anomaly.AccessBaseline, error) {
	query := `
		SELECT id, account_id, principal_id, principal_type, principal_name,
			   time_window, baseline_period_start, baseline_period_end,
			   avg_daily_access_count, std_dev_access_count,
			   avg_data_volume_bytes, std_dev_data_volume,
			   normal_access_hours, normal_access_days,
			   common_assets, common_operations, common_source_ips, common_geo_locations,
			   access_count_threshold, data_volume_threshold,
			   created_at, updated_at
		FROM access_baselines
		WHERE account_id = $1 AND principal_id = $2
	`

	var b anomaly.AccessBaseline
	var normalHoursJSON, normalDaysJSON, commonAssetsJSON, commonOpsJSON, commonIPsJSON, commonGeoJSON []byte

	err := s.db.QueryRowContext(ctx, query, accountID, principalID).Scan(
		&b.ID,
		&b.AccountID,
		&b.PrincipalID,
		&b.PrincipalType,
		&b.PrincipalName,
		&b.TimeWindow,
		&b.BaselinePeriodStart,
		&b.BaselinePeriodEnd,
		&b.AvgDailyAccessCount,
		&b.StdDevAccessCount,
		&b.AvgDataVolumeBytes,
		&b.StdDevDataVolume,
		&normalHoursJSON,
		&normalDaysJSON,
		&commonAssetsJSON,
		&commonOpsJSON,
		&commonIPsJSON,
		&commonGeoJSON,
		&b.AccessCountThreshold,
		&b.DataVolumeThreshold,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying baseline: %w", err)
	}

	json.Unmarshal(normalHoursJSON, &b.NormalAccessHours)
	json.Unmarshal(normalDaysJSON, &b.NormalAccessDays)
	json.Unmarshal(commonAssetsJSON, &b.CommonAssets)
	json.Unmarshal(commonOpsJSON, &b.CommonOperations)
	json.Unmarshal(commonIPsJSON, &b.CommonSourceIPs)
	json.Unmarshal(commonGeoJSON, &b.CommonGeoLocations)

	return &b, nil
}

// ListBaselines lists all baselines for an account
func (s *Store) ListBaselines(ctx context.Context, accountID uuid.UUID) ([]anomaly.AccessBaseline, error) {
	query := `
		SELECT id, account_id, principal_id, principal_type, principal_name,
			   time_window, baseline_period_start, baseline_period_end,
			   avg_daily_access_count, std_dev_access_count,
			   avg_data_volume_bytes, std_dev_data_volume,
			   normal_access_hours, normal_access_days,
			   common_assets, common_operations, common_source_ips, common_geo_locations,
			   access_count_threshold, data_volume_threshold,
			   created_at, updated_at
		FROM access_baselines
		WHERE account_id = $1
		ORDER BY principal_name
	`

	rows, err := s.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("querying baselines: %w", err)
	}
	defer rows.Close()

	var baselines []anomaly.AccessBaseline
	for rows.Next() {
		var b anomaly.AccessBaseline
		var normalHoursJSON, normalDaysJSON, commonAssetsJSON, commonOpsJSON, commonIPsJSON, commonGeoJSON []byte

		err := rows.Scan(
			&b.ID,
			&b.AccountID,
			&b.PrincipalID,
			&b.PrincipalType,
			&b.PrincipalName,
			&b.TimeWindow,
			&b.BaselinePeriodStart,
			&b.BaselinePeriodEnd,
			&b.AvgDailyAccessCount,
			&b.StdDevAccessCount,
			&b.AvgDataVolumeBytes,
			&b.StdDevDataVolume,
			&normalHoursJSON,
			&normalDaysJSON,
			&commonAssetsJSON,
			&commonOpsJSON,
			&commonIPsJSON,
			&commonGeoJSON,
			&b.AccessCountThreshold,
			&b.DataVolumeThreshold,
			&b.CreatedAt,
			&b.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning baseline: %w", err)
		}

		json.Unmarshal(normalHoursJSON, &b.NormalAccessHours)
		json.Unmarshal(normalDaysJSON, &b.NormalAccessDays)
		json.Unmarshal(commonAssetsJSON, &b.CommonAssets)
		json.Unmarshal(commonOpsJSON, &b.CommonOperations)
		json.Unmarshal(commonIPsJSON, &b.CommonSourceIPs)
		json.Unmarshal(commonGeoJSON, &b.CommonGeoLocations)

		baselines = append(baselines, b)
	}

	return baselines, nil
}

// UpdateBaseline updates an access baseline
func (s *Store) UpdateBaseline(ctx context.Context, baseline *anomaly.AccessBaseline) error {
	normalHoursJSON, _ := json.Marshal(baseline.NormalAccessHours)
	normalDaysJSON, _ := json.Marshal(baseline.NormalAccessDays)
	commonAssetsJSON, _ := json.Marshal(baseline.CommonAssets)
	commonOpsJSON, _ := json.Marshal(baseline.CommonOperations)
	commonIPsJSON, _ := json.Marshal(baseline.CommonSourceIPs)
	commonGeoJSON, _ := json.Marshal(baseline.CommonGeoLocations)

	query := `
		UPDATE access_baselines SET
			baseline_period_start = $2,
			baseline_period_end = $3,
			avg_daily_access_count = $4,
			std_dev_access_count = $5,
			avg_data_volume_bytes = $6,
			std_dev_data_volume = $7,
			normal_access_hours = $8,
			normal_access_days = $9,
			common_assets = $10,
			common_operations = $11,
			common_source_ips = $12,
			common_geo_locations = $13,
			access_count_threshold = $14,
			data_volume_threshold = $15,
			updated_at = $16
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query,
		baseline.ID,
		baseline.BaselinePeriodStart,
		baseline.BaselinePeriodEnd,
		baseline.AvgDailyAccessCount,
		baseline.StdDevAccessCount,
		baseline.AvgDataVolumeBytes,
		baseline.StdDevDataVolume,
		normalHoursJSON,
		normalDaysJSON,
		commonAssetsJSON,
		commonOpsJSON,
		commonIPsJSON,
		commonGeoJSON,
		baseline.AccessCountThreshold,
		baseline.DataVolumeThreshold,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("updating baseline: %w", err)
	}

	return nil
}

// DeleteBaseline deletes an access baseline
func (s *Store) DeleteBaseline(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM access_baselines WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting baseline: %w", err)
	}
	return nil
}

// CreateThreatScore creates a new threat score
func (s *Store) CreateThreatScore(ctx context.Context, score *anomaly.ThreatScore) error {
	factorsJSON, _ := json.Marshal(score.Factors)
	detailsJSON, _ := json.Marshal(score.Details)

	query := `
		INSERT INTO threat_scores (
			id, account_id, principal_id, principal_type, principal_name,
			score, risk_level, factors, recent_anomalies, trend_direction,
			details, last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := s.db.ExecContext(ctx, query,
		score.ID,
		score.AccountID,
		score.PrincipalID,
		score.PrincipalType,
		score.PrincipalName,
		score.Score,
		score.RiskLevel,
		factorsJSON,
		score.RecentAnomalies,
		score.TrendDirection,
		detailsJSON,
		score.LastUpdated,
	)
	if err != nil {
		return fmt.Errorf("inserting threat score: %w", err)
	}

	return nil
}

// GetThreatScore retrieves a threat score for a principal
func (s *Store) GetThreatScore(ctx context.Context, accountID uuid.UUID, principalID string) (*anomaly.ThreatScore, error) {
	query := `
		SELECT id, account_id, principal_id, principal_type, principal_name,
			   score, risk_level, factors, recent_anomalies, trend_direction,
			   details, last_updated
		FROM threat_scores
		WHERE account_id = $1 AND principal_id = $2
	`

	var score anomaly.ThreatScore
	var factorsJSON, detailsJSON []byte

	err := s.db.QueryRowContext(ctx, query, accountID, principalID).Scan(
		&score.ID,
		&score.AccountID,
		&score.PrincipalID,
		&score.PrincipalType,
		&score.PrincipalName,
		&score.Score,
		&score.RiskLevel,
		&factorsJSON,
		&score.RecentAnomalies,
		&score.TrendDirection,
		&detailsJSON,
		&score.LastUpdated,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying threat score: %w", err)
	}

	json.Unmarshal(factorsJSON, &score.Factors)
	json.Unmarshal(detailsJSON, &score.Details)

	return &score, nil
}

// ListThreatScores lists threat scores for an account
func (s *Store) ListThreatScores(ctx context.Context, accountID uuid.UUID, minScore float64, limit, offset int) ([]anomaly.ThreatScore, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM threat_scores WHERE account_id = $1 AND score >= $2`
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, accountID, minScore).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting threat scores: %w", err)
	}

	// List query
	query := `
		SELECT id, account_id, principal_id, principal_type, principal_name,
			   score, risk_level, factors, recent_anomalies, trend_direction,
			   details, last_updated
		FROM threat_scores
		WHERE account_id = $1 AND score >= $2
		ORDER BY score DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := s.db.QueryContext(ctx, query, accountID, minScore, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying threat scores: %w", err)
	}
	defer rows.Close()

	var scores []anomaly.ThreatScore
	for rows.Next() {
		var score anomaly.ThreatScore
		var factorsJSON, detailsJSON []byte

		err := rows.Scan(
			&score.ID,
			&score.AccountID,
			&score.PrincipalID,
			&score.PrincipalType,
			&score.PrincipalName,
			&score.Score,
			&score.RiskLevel,
			&factorsJSON,
			&score.RecentAnomalies,
			&score.TrendDirection,
			&detailsJSON,
			&score.LastUpdated,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning threat score: %w", err)
		}

		json.Unmarshal(factorsJSON, &score.Factors)
		json.Unmarshal(detailsJSON, &score.Details)

		scores = append(scores, score)
	}

	return scores, total, nil
}

// UpdateThreatScore updates a threat score
func (s *Store) UpdateThreatScore(ctx context.Context, score *anomaly.ThreatScore) error {
	factorsJSON, _ := json.Marshal(score.Factors)
	detailsJSON, _ := json.Marshal(score.Details)

	query := `
		UPDATE threat_scores SET
			score = $2,
			risk_level = $3,
			factors = $4,
			recent_anomalies = $5,
			trend_direction = $6,
			details = $7,
			last_updated = $8
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query,
		score.ID,
		score.Score,
		score.RiskLevel,
		factorsJSON,
		score.RecentAnomalies,
		score.TrendDirection,
		detailsJSON,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("updating threat score: %w", err)
	}

	return nil
}
