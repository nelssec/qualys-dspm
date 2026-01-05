package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresStore implements Store with PostgreSQL
type PostgresStore struct {
	db *sqlx.DB
}

// NewPostgresStore creates a new PostgreSQL scheduler store
func NewPostgresStore(db *sqlx.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

type jobRow struct {
	ID          string     `db:"id"`
	Name        string     `db:"name"`
	Description string     `db:"description"`
	Schedule    string     `db:"schedule"`
	JobType     string     `db:"job_type"`
	Config      []byte     `db:"config"`
	Enabled     bool       `db:"enabled"`
	LastRun     *time.Time `db:"last_run"`
	NextRun     *time.Time `db:"next_run"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

func (r *jobRow) toJob() (*Job, error) {
	var config map[string]string
	if len(r.Config) > 0 {
		if err := json.Unmarshal(r.Config, &config); err != nil {
			return nil, err
		}
	}

	return &Job{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Schedule:    r.Schedule,
		JobType:     JobType(r.JobType),
		Config:      config,
		Enabled:     r.Enabled,
		LastRun:     r.LastRun,
		NextRun:     r.NextRun,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}, nil
}

// GetJob retrieves a job by ID
func (s *PostgresStore) GetJob(ctx context.Context, id string) (*Job, error) {
	var row jobRow
	err := s.db.GetContext(ctx, &row, `
		SELECT id, name, description, schedule, job_type, config, enabled, last_run, next_run, created_at, updated_at
		FROM scheduled_jobs WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("job not found")
		}
		return nil, err
	}
	return row.toJob()
}

// ListJobs lists all jobs
func (s *PostgresStore) ListJobs(ctx context.Context) ([]*Job, error) {
	var rows []jobRow
	err := s.db.SelectContext(ctx, &rows, `
		SELECT id, name, description, schedule, job_type, config, enabled, last_run, next_run, created_at, updated_at
		FROM scheduled_jobs ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}

	jobs := make([]*Job, len(rows))
	for i, row := range rows {
		job, err := row.toJob()
		if err != nil {
			return nil, err
		}
		jobs[i] = job
	}
	return jobs, nil
}

// CreateJob creates a new job
func (s *PostgresStore) CreateJob(ctx context.Context, job *Job) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now

	configJSON, err := json.Marshal(job.Config)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO scheduled_jobs (id, name, description, schedule, job_type, config, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, job.ID, job.Name, job.Description, job.Schedule, string(job.JobType), configJSON, job.Enabled, job.CreatedAt, job.UpdatedAt)
	return err
}

// UpdateJob updates a job
func (s *PostgresStore) UpdateJob(ctx context.Context, job *Job) error {
	job.UpdatedAt = time.Now()

	configJSON, err := json.Marshal(job.Config)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE scheduled_jobs SET
			name = $2, description = $3, schedule = $4, job_type = $5,
			config = $6, enabled = $7, next_run = $8, updated_at = $9
		WHERE id = $1
	`, job.ID, job.Name, job.Description, job.Schedule, string(job.JobType), configJSON, job.Enabled, job.NextRun, job.UpdatedAt)
	return err
}

// DeleteJob deletes a job
func (s *PostgresStore) DeleteJob(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM scheduled_jobs WHERE id = $1`, id)
	return err
}

// UpdateLastRun updates the last run time
func (s *PostgresStore) UpdateLastRun(ctx context.Context, id string, lastRun time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE scheduled_jobs SET last_run = $2, updated_at = NOW()
		WHERE id = $1
	`, id, lastRun)
	return err
}

// CreateExecution creates a job execution record
func (s *PostgresStore) CreateExecution(ctx context.Context, exec *JobExecution) error {
	if exec.ID == "" {
		exec.ID = uuid.New().String()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO job_executions (id, job_id, status, started_at, error, output)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, exec.ID, exec.JobID, string(exec.Status), exec.StartedAt, exec.Error, exec.Output)
	return err
}

// UpdateExecution updates a job execution record
func (s *PostgresStore) UpdateExecution(ctx context.Context, exec *JobExecution) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE job_executions SET status = $2, ended_at = $3, error = $4, output = $5
		WHERE id = $1
	`, exec.ID, string(exec.Status), exec.EndedAt, exec.Error, exec.Output)
	return err
}

// GetJobExecutions gets recent executions for a job
func (s *PostgresStore) GetJobExecutions(ctx context.Context, jobID string, limit int) ([]*JobExecution, error) {
	var execs []*JobExecution
	err := s.db.SelectContext(ctx, &execs, `
		SELECT id, job_id, status, started_at, ended_at, error, output
		FROM job_executions
		WHERE job_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, jobID, limit)
	return execs, err
}
