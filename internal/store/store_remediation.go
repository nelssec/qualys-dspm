package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/qualys/dspm/internal/remediation"
)

// CreateRemediationAction creates a new remediation action
func (s *Store) CreateRemediationAction(ctx context.Context, action *remediation.Action) error {
	paramsJSON, err := json.Marshal(action.Parameters)
	if err != nil {
		return fmt.Errorf("marshaling parameters: %w", err)
	}

	query := `
		INSERT INTO remediation_actions (
			id, account_id, asset_id, finding_id, action_type, status, risk_level,
			description, parameters, rollback_available, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = s.db.ExecContext(ctx, query,
		action.ID,
		action.AccountID,
		action.AssetID,
		action.FindingID,
		action.ActionType,
		action.Status,
		action.RiskLevel,
		action.Description,
		paramsJSON,
		action.RollbackAvailable,
		action.CreatedAt,
		action.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting remediation action: %w", err)
	}

	return nil
}

// GetRemediationAction retrieves a remediation action by ID
func (s *Store) GetRemediationAction(ctx context.Context, id uuid.UUID) (*remediation.Action, error) {
	query := `
		SELECT id, account_id, asset_id, finding_id, action_type, status, risk_level,
			   description, parameters, previous_state, new_state, created_at, updated_at,
			   approved_at, approved_by, executed_at, completed_at, error_message, rollback_available
		FROM remediation_actions
		WHERE id = $1
	`

	var action remediation.Action
	var paramsJSON, prevStateJSON, newStateJSON []byte
	var findingID sql.NullString
	var approvedAt, executedAt, completedAt sql.NullTime
	var approvedBy, errorMessage sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&action.ID,
		&action.AccountID,
		&action.AssetID,
		&findingID,
		&action.ActionType,
		&action.Status,
		&action.RiskLevel,
		&action.Description,
		&paramsJSON,
		&prevStateJSON,
		&newStateJSON,
		&action.CreatedAt,
		&action.UpdatedAt,
		&approvedAt,
		&approvedBy,
		&executedAt,
		&completedAt,
		&errorMessage,
		&action.RollbackAvailable,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("remediation action not found")
	}
	if err != nil {
		return nil, fmt.Errorf("querying remediation action: %w", err)
	}

	if findingID.Valid {
		fid, _ := uuid.Parse(findingID.String)
		action.FindingID = &fid
	}

	if len(paramsJSON) > 0 {
		if err := json.Unmarshal(paramsJSON, &action.Parameters); err != nil {
			return nil, fmt.Errorf("unmarshaling parameters: %w", err)
		}
	}

	if len(prevStateJSON) > 0 {
		if err := json.Unmarshal(prevStateJSON, &action.PreviousState); err != nil {
			return nil, fmt.Errorf("unmarshaling previous state: %w", err)
		}
	}

	if len(newStateJSON) > 0 {
		if err := json.Unmarshal(newStateJSON, &action.NewState); err != nil {
			return nil, fmt.Errorf("unmarshaling new state: %w", err)
		}
	}

	if approvedAt.Valid {
		action.ApprovedAt = &approvedAt.Time
	}
	if approvedBy.Valid {
		action.ApprovedBy = approvedBy.String
	}
	if executedAt.Valid {
		action.ExecutedAt = &executedAt.Time
	}
	if completedAt.Valid {
		action.CompletedAt = &completedAt.Time
	}
	if errorMessage.Valid {
		action.ErrorMessage = errorMessage.String
	}

	return &action, nil
}

// UpdateRemediationAction updates a remediation action
func (s *Store) UpdateRemediationAction(ctx context.Context, action *remediation.Action) error {
	prevStateJSON, err := json.Marshal(action.PreviousState)
	if err != nil {
		return fmt.Errorf("marshaling previous state: %w", err)
	}

	newStateJSON, err := json.Marshal(action.NewState)
	if err != nil {
		return fmt.Errorf("marshaling new state: %w", err)
	}

	query := `
		UPDATE remediation_actions SET
			status = $2,
			previous_state = $3,
			new_state = $4,
			approved_at = $5,
			approved_by = $6,
			executed_at = $7,
			completed_at = $8,
			error_message = $9,
			updated_at = $10
		WHERE id = $1
	`

	_, err = s.db.ExecContext(ctx, query,
		action.ID,
		action.Status,
		prevStateJSON,
		newStateJSON,
		action.ApprovedAt,
		action.ApprovedBy,
		action.ExecutedAt,
		action.CompletedAt,
		action.ErrorMessage,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("updating remediation action: %w", err)
	}

	return nil
}

// ListRemediationActions lists remediation actions with optional filtering
func (s *Store) ListRemediationActions(ctx context.Context, accountID uuid.UUID, status *remediation.ActionStatus, limit, offset int) ([]remediation.Action, int, error) {
	var args []interface{}
	argIdx := 1

	whereClause := fmt.Sprintf("WHERE account_id = $%d", argIdx)
	args = append(args, accountID)
	argIdx++

	if status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM remediation_actions %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting remediation actions: %w", err)
	}

	// Get actions
	query := fmt.Sprintf(`
		SELECT id, account_id, asset_id, finding_id, action_type, status, risk_level,
			   description, parameters, previous_state, new_state, created_at, updated_at,
			   approved_at, approved_by, executed_at, completed_at, error_message, rollback_available
		FROM remediation_actions
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying remediation actions: %w", err)
	}
	defer rows.Close()

	var actions []remediation.Action
	for rows.Next() {
		var action remediation.Action
		var paramsJSON, prevStateJSON, newStateJSON []byte
		var findingID sql.NullString
		var approvedAt, executedAt, completedAt sql.NullTime
		var approvedBy, errorMessage sql.NullString

		err := rows.Scan(
			&action.ID,
			&action.AccountID,
			&action.AssetID,
			&findingID,
			&action.ActionType,
			&action.Status,
			&action.RiskLevel,
			&action.Description,
			&paramsJSON,
			&prevStateJSON,
			&newStateJSON,
			&action.CreatedAt,
			&action.UpdatedAt,
			&approvedAt,
			&approvedBy,
			&executedAt,
			&completedAt,
			&errorMessage,
			&action.RollbackAvailable,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning remediation action: %w", err)
		}

		if findingID.Valid {
			fid, _ := uuid.Parse(findingID.String)
			action.FindingID = &fid
		}

		if len(paramsJSON) > 0 {
			json.Unmarshal(paramsJSON, &action.Parameters)
		}
		if len(prevStateJSON) > 0 {
			json.Unmarshal(prevStateJSON, &action.PreviousState)
		}
		if len(newStateJSON) > 0 {
			json.Unmarshal(newStateJSON, &action.NewState)
		}

		if approvedAt.Valid {
			action.ApprovedAt = &approvedAt.Time
		}
		if approvedBy.Valid {
			action.ApprovedBy = approvedBy.String
		}
		if executedAt.Valid {
			action.ExecutedAt = &executedAt.Time
		}
		if completedAt.Valid {
			action.CompletedAt = &completedAt.Time
		}
		if errorMessage.Valid {
			action.ErrorMessage = errorMessage.String
		}

		actions = append(actions, action)
	}

	return actions, total, nil
}

// ListRemediationActionsForAsset lists remediation actions for a specific asset
func (s *Store) ListRemediationActionsForAsset(ctx context.Context, assetID uuid.UUID) ([]remediation.Action, error) {
	query := `
		SELECT id, account_id, asset_id, finding_id, action_type, status, risk_level,
			   description, parameters, previous_state, new_state, created_at, updated_at,
			   approved_at, approved_by, executed_at, completed_at, error_message, rollback_available
		FROM remediation_actions
		WHERE asset_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, assetID)
	if err != nil {
		return nil, fmt.Errorf("querying remediation actions: %w", err)
	}
	defer rows.Close()

	var actions []remediation.Action
	for rows.Next() {
		var action remediation.Action
		var paramsJSON, prevStateJSON, newStateJSON []byte
		var findingID sql.NullString
		var approvedAt, executedAt, completedAt sql.NullTime
		var approvedBy, errorMessage sql.NullString

		err := rows.Scan(
			&action.ID,
			&action.AccountID,
			&action.AssetID,
			&findingID,
			&action.ActionType,
			&action.Status,
			&action.RiskLevel,
			&action.Description,
			&paramsJSON,
			&prevStateJSON,
			&newStateJSON,
			&action.CreatedAt,
			&action.UpdatedAt,
			&approvedAt,
			&approvedBy,
			&executedAt,
			&completedAt,
			&errorMessage,
			&action.RollbackAvailable,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning remediation action: %w", err)
		}

		if findingID.Valid {
			fid, _ := uuid.Parse(findingID.String)
			action.FindingID = &fid
		}

		if len(paramsJSON) > 0 {
			json.Unmarshal(paramsJSON, &action.Parameters)
		}
		if len(prevStateJSON) > 0 {
			json.Unmarshal(prevStateJSON, &action.PreviousState)
		}
		if len(newStateJSON) > 0 {
			json.Unmarshal(newStateJSON, &action.NewState)
		}

		if approvedAt.Valid {
			action.ApprovedAt = &approvedAt.Time
		}
		if approvedBy.Valid {
			action.ApprovedBy = approvedBy.String
		}
		if executedAt.Valid {
			action.ExecutedAt = &executedAt.Time
		}
		if completedAt.Valid {
			action.CompletedAt = &completedAt.Time
		}
		if errorMessage.Valid {
			action.ErrorMessage = errorMessage.String
		}

		actions = append(actions, action)
	}

	return actions, nil
}

// GetRemediationActionSummary returns a summary of remediation actions
func (s *Store) GetRemediationActionSummary(ctx context.Context, accountID uuid.UUID) (*remediation.ActionSummary, error) {
	summary := &remediation.ActionSummary{
		ByActionType: make(map[string]int),
		ByRiskLevel:  make(map[string]int),
	}

	// Get counts by status
	statusQuery := `
		SELECT status, COUNT(*)
		FROM remediation_actions
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
		summary.TotalActions += count
		switch remediation.ActionStatus(status) {
		case remediation.StatusPending:
			summary.PendingCount = count
		case remediation.StatusApproved:
			summary.ApprovedCount = count
		case remediation.StatusCompleted:
			summary.CompletedCount = count
		case remediation.StatusFailed:
			summary.FailedCount = count
		}
	}

	// Get counts by action type
	typeQuery := `
		SELECT action_type, COUNT(*)
		FROM remediation_actions
		WHERE account_id = $1
		GROUP BY action_type
	`
	rows, err = s.db.QueryContext(ctx, typeQuery, accountID)
	if err != nil {
		return nil, fmt.Errorf("querying type counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var actionType string
		var count int
		if err := rows.Scan(&actionType, &count); err != nil {
			return nil, fmt.Errorf("scanning type count: %w", err)
		}
		summary.ByActionType[actionType] = count
	}

	// Get counts by risk level
	riskQuery := `
		SELECT risk_level, COUNT(*)
		FROM remediation_actions
		WHERE account_id = $1
		GROUP BY risk_level
	`
	rows, err = s.db.QueryContext(ctx, riskQuery, accountID)
	if err != nil {
		return nil, fmt.Errorf("querying risk counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var riskLevel string
		var count int
		if err := rows.Scan(&riskLevel, &count); err != nil {
			return nil, fmt.Errorf("scanning risk count: %w", err)
		}
		summary.ByRiskLevel[riskLevel] = count
	}

	// Get recently executed actions
	recentQuery := `
		SELECT id, account_id, asset_id, finding_id, action_type, status, risk_level,
			   description, parameters, created_at, updated_at, completed_at, rollback_available
		FROM remediation_actions
		WHERE account_id = $1 AND status = 'COMPLETED'
		ORDER BY completed_at DESC
		LIMIT 5
	`
	rows, err = s.db.QueryContext(ctx, recentQuery, accountID)
	if err != nil {
		return nil, fmt.Errorf("querying recent actions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var action remediation.Action
		var paramsJSON []byte
		var findingID sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(
			&action.ID,
			&action.AccountID,
			&action.AssetID,
			&findingID,
			&action.ActionType,
			&action.Status,
			&action.RiskLevel,
			&action.Description,
			&paramsJSON,
			&action.CreatedAt,
			&action.UpdatedAt,
			&completedAt,
			&action.RollbackAvailable,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning recent action: %w", err)
		}

		if findingID.Valid {
			fid, _ := uuid.Parse(findingID.String)
			action.FindingID = &fid
		}
		if len(paramsJSON) > 0 {
			json.Unmarshal(paramsJSON, &action.Parameters)
		}
		if completedAt.Valid {
			action.CompletedAt = &completedAt.Time
		}

		summary.RecentlyExecuted = append(summary.RecentlyExecuted, action)
	}

	return summary, nil
}
