package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresUserStore implements UserStore with PostgreSQL
type PostgresUserStore struct {
	db *sqlx.DB
}

// NewPostgresUserStore creates a new PostgreSQL user store
func NewPostgresUserStore(db *sqlx.DB) *PostgresUserStore {
	return &PostgresUserStore{db: db}
}

// GetUserByID retrieves a user by ID
func (s *PostgresUserStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := s.db.GetContext(ctx, &user, `
		SELECT id, email, name, password_hash, role, created_at, updated_at
		FROM users WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (s *PostgresUserStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := s.db.GetContext(ctx, &user, `
		SELECT id, email, name, password_hash, role, created_at, updated_at
		FROM users WHERE email = $1
	`, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// CreateUser creates a new user
func (s *PostgresUserStore) CreateUser(ctx context.Context, user *User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, user.ID, user.Email, user.Name, user.Password, user.Role, user.CreatedAt, user.UpdatedAt)
	return err
}

// UpdateUser updates a user
func (s *PostgresUserStore) UpdateUser(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET email = $2, name = $3, password_hash = $4, role = $5, updated_at = $6
		WHERE id = $1
	`, user.ID, user.Email, user.Name, user.Password, user.Role, user.UpdatedAt)
	return err
}

// DeleteUser deletes a user
func (s *PostgresUserStore) DeleteUser(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// ListUsers lists all users
func (s *PostgresUserStore) ListUsers(ctx context.Context) ([]*User, error) {
	var users []*User
	err := s.db.SelectContext(ctx, &users, `
		SELECT id, email, name, role, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`)
	return users, err
}

// StoreRefreshToken stores a refresh token
func (s *PostgresUserStore) StoreRefreshToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.New().String(), userID, token, expiresAt, time.Now())
	return err
}

// ValidateRefreshToken validates a refresh token
func (s *PostgresUserStore) ValidateRefreshToken(ctx context.Context, userID, token string) (bool, error) {
	var count int
	err := s.db.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM refresh_tokens
		WHERE user_id = $1 AND token = $2 AND expires_at > NOW() AND revoked_at IS NULL
	`, userID, token)
	return count > 0, err
}

// RevokeRefreshToken revokes a refresh token
func (s *PostgresUserStore) RevokeRefreshToken(ctx context.Context, userID, token string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW()
		WHERE user_id = $1 AND token = $2
	`, userID, token)
	return err
}

// RevokeAllRefreshTokens revokes all refresh tokens for a user
func (s *PostgresUserStore) RevokeAllRefreshTokens(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	return err
}
