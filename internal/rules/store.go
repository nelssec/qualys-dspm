package rules

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/qualys/dspm/internal/models"
)

// PostgresStore implements Store with PostgreSQL
type PostgresStore struct {
	db *sqlx.DB
}

// NewPostgresStore creates a new PostgreSQL rules store
func NewPostgresStore(db *sqlx.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

type ruleRow struct {
	ID              string `db:"id"`
	Name            string `db:"name"`
	Description     string `db:"description"`
	Category        string `db:"category"`
	Sensitivity     string `db:"sensitivity"`
	ContextRequired bool   `db:"context_required"`
	Enabled         bool   `db:"enabled"`
	Priority        int    `db:"priority"`
	CreatedBy       string `db:"created_by"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

func (r *ruleRow) toRule() *CustomRule {
	return &CustomRule{
		ID:              r.ID,
		Name:            r.Name,
		Description:     r.Description,
		Category:        models.Category(r.Category),
		Sensitivity:     models.Sensitivity(r.Sensitivity),
		ContextRequired: r.ContextRequired,
		Enabled:         r.Enabled,
		Priority:        r.Priority,
		CreatedBy:       r.CreatedBy,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

// GetRule retrieves a rule by ID
func (s *PostgresStore) GetRule(ctx context.Context, id string) (*CustomRule, error) {
	var row ruleRow
	err := s.db.GetContext(ctx, &row, `
		SELECT id, name, description, category, sensitivity, context_required, enabled, priority, created_by, created_at, updated_at
		FROM custom_rules WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("rule not found")
		}
		return nil, err
	}
	return row.toRule(), nil
}

// ListRules lists rules
func (s *PostgresStore) ListRules(ctx context.Context, enabledOnly bool) ([]*CustomRule, error) {
	var rows []ruleRow
	var err error

	if enabledOnly {
		err = s.db.SelectContext(ctx, &rows, `
			SELECT id, name, description, category, sensitivity, context_required, enabled, priority, created_by, created_at, updated_at
			FROM custom_rules WHERE enabled = true ORDER BY priority DESC, created_at DESC
		`)
	} else {
		err = s.db.SelectContext(ctx, &rows, `
			SELECT id, name, description, category, sensitivity, context_required, enabled, priority, created_by, created_at, updated_at
			FROM custom_rules ORDER BY priority DESC, created_at DESC
		`)
	}

	if err != nil {
		return nil, err
	}

	rules := make([]*CustomRule, len(rows))
	for i, row := range rows {
		rules[i] = row.toRule()
	}
	return rules, nil
}

// CreateRule creates a new rule
func (s *PostgresStore) CreateRule(ctx context.Context, rule *CustomRule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO custom_rules (id, name, description, category, sensitivity, context_required, enabled, priority, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, rule.ID, rule.Name, rule.Description, string(rule.Category), string(rule.Sensitivity),
		rule.ContextRequired, rule.Enabled, rule.Priority, rule.CreatedBy, rule.CreatedAt, rule.UpdatedAt)
	return err
}

// UpdateRule updates a rule
func (s *PostgresStore) UpdateRule(ctx context.Context, rule *CustomRule) error {
	rule.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE custom_rules SET
			name = $2, description = $3, category = $4, sensitivity = $5,
			context_required = $6, enabled = $7, priority = $8, updated_at = $9
		WHERE id = $1
	`, rule.ID, rule.Name, rule.Description, string(rule.Category), string(rule.Sensitivity),
		rule.ContextRequired, rule.Enabled, rule.Priority, rule.UpdatedAt)
	return err
}

// DeleteRule deletes a rule
func (s *PostgresStore) DeleteRule(ctx context.Context, id string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete patterns first
	if _, err := tx.ExecContext(ctx, `DELETE FROM rule_patterns WHERE rule_id = $1`, id); err != nil {
		return err
	}

	// Delete rule
	if _, err := tx.ExecContext(ctx, `DELETE FROM custom_rules WHERE id = $1`, id); err != nil {
		return err
	}

	return tx.Commit()
}

// GetRulePatterns retrieves patterns for a rule
func (s *PostgresStore) GetRulePatterns(ctx context.Context, ruleID string) ([]string, []string, error) {
	var patterns, contextPatterns []string

	rows, err := s.db.QueryxContext(ctx, `
		SELECT pattern, is_context FROM rule_patterns WHERE rule_id = $1 ORDER BY id
	`, ruleID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pattern string
		var isContext bool
		if err := rows.Scan(&pattern, &isContext); err != nil {
			return nil, nil, err
		}
		if isContext {
			contextPatterns = append(contextPatterns, pattern)
		} else {
			patterns = append(patterns, pattern)
		}
	}

	return patterns, contextPatterns, rows.Err()
}

// SetRulePatterns sets patterns for a rule
func (s *PostgresStore) SetRulePatterns(ctx context.Context, ruleID string, patterns, contextPatterns []string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing patterns
	if _, err := tx.ExecContext(ctx, `DELETE FROM rule_patterns WHERE rule_id = $1`, ruleID); err != nil {
		return err
	}

	// Insert new patterns
	for _, p := range patterns {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO rule_patterns (id, rule_id, pattern, is_context) VALUES ($1, $2, $3, false)
		`, uuid.New().String(), ruleID, p); err != nil {
			return err
		}
	}

	// Insert context patterns
	for _, p := range contextPatterns {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO rule_patterns (id, rule_id, pattern, is_context) VALUES ($1, $2, $3, true)
		`, uuid.New().String(), ruleID, p); err != nil {
			return err
		}
	}

	return tx.Commit()
}
