package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Job struct {
	ID          string            `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	Schedule    string            `json:"schedule" db:"schedule"` // Cron expression
	JobType     JobType           `json:"job_type" db:"job_type"`
	Config      map[string]string `json:"config" db:"config"`
	Enabled     bool              `json:"enabled" db:"enabled"`
	LastRun     *time.Time        `json:"last_run,omitempty" db:"last_run"`
	NextRun     *time.Time        `json:"next_run,omitempty" db:"next_run"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

type JobType string

const (
	JobTypeScanAccount     JobType = "scan_account"
	JobTypeScanAllAccounts JobType = "scan_all_accounts"
	JobTypeCleanupOld      JobType = "cleanup_old"
	JobTypeGenerateReport  JobType = "generate_report"
	JobTypeSyncAccessGraph JobType = "sync_access_graph"
)

type JobExecution struct {
	ID        string          `json:"id" db:"id"`
	JobID     string          `json:"job_id" db:"job_id"`
	Status    ExecutionStatus `json:"status" db:"status"`
	StartedAt time.Time       `json:"started_at" db:"started_at"`
	EndedAt   *time.Time      `json:"ended_at,omitempty" db:"ended_at"`
	Error     string          `json:"error,omitempty" db:"error"`
	Output    string          `json:"output,omitempty" db:"output"`
}

type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
)

type JobHandler func(ctx context.Context, job *Job) error

type Store interface {
	GetJob(ctx context.Context, id string) (*Job, error)
	ListJobs(ctx context.Context) ([]*Job, error)
	CreateJob(ctx context.Context, job *Job) error
	UpdateJob(ctx context.Context, job *Job) error
	DeleteJob(ctx context.Context, id string) error
	UpdateLastRun(ctx context.Context, id string, lastRun time.Time) error
	CreateExecution(ctx context.Context, exec *JobExecution) error
	UpdateExecution(ctx context.Context, exec *JobExecution) error
	GetJobExecutions(ctx context.Context, jobID string, limit int) ([]*JobExecution, error)
}

type Scheduler struct {
	cron     *cron.Cron
	store    Store
	handlers map[JobType]JobHandler
	entries  map[string]cron.EntryID
	mu       sync.RWMutex
	logger   *slog.Logger
}

func NewScheduler(store Store, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler{
		cron: cron.New(cron.WithParser(cron.NewParser(
			cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		))),
		store:    store,
		handlers: make(map[JobType]JobHandler),
		entries:  make(map[string]cron.EntryID),
		logger:   logger,
	}
}

func (s *Scheduler) RegisterHandler(jobType JobType, handler JobHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[jobType] = handler
}

func (s *Scheduler) Start(ctx context.Context) error {
	jobs, err := s.store.ListJobs(ctx)
	if err != nil {
		return fmt.Errorf("failed to load jobs: %w", err)
	}

	for _, job := range jobs {
		if job.Enabled {
			if err := s.scheduleJob(job); err != nil {
				s.logger.Error("failed to schedule job",
					"job_id", job.ID,
					"job_name", job.Name,
					"error", err)
			}
		}
	}

	s.cron.Start()
	s.logger.Info("scheduler started", "jobs_count", len(jobs))

	return nil
}

func (s *Scheduler) Stop() context.Context {
	return s.cron.Stop()
}

func (s *Scheduler) AddJob(ctx context.Context, job *Job) error {
	if err := s.store.CreateJob(ctx, job); err != nil {
		return err
	}

	if job.Enabled {
		return s.scheduleJob(job)
	}

	return nil
}

func (s *Scheduler) UpdateJob(ctx context.Context, job *Job) error {
	s.unscheduleJob(job.ID)

	if err := s.store.UpdateJob(ctx, job); err != nil {
		return err
	}

	if job.Enabled {
		return s.scheduleJob(job)
	}

	return nil
}

func (s *Scheduler) DeleteJob(ctx context.Context, id string) error {
	s.unscheduleJob(id)
	return s.store.DeleteJob(ctx, id)
}

func (s *Scheduler) EnableJob(ctx context.Context, id string) error {
	job, err := s.store.GetJob(ctx, id)
	if err != nil {
		return err
	}

	job.Enabled = true
	if err := s.store.UpdateJob(ctx, job); err != nil {
		return err
	}

	return s.scheduleJob(job)
}

func (s *Scheduler) DisableJob(ctx context.Context, id string) error {
	job, err := s.store.GetJob(ctx, id)
	if err != nil {
		return err
	}

	job.Enabled = false
	s.unscheduleJob(id)

	return s.store.UpdateJob(ctx, job)
}

func (s *Scheduler) RunJobNow(ctx context.Context, id string) error {
	job, err := s.store.GetJob(ctx, id)
	if err != nil {
		return err
	}

	go s.executeJob(job)
	return nil
}

func (s *Scheduler) GetNextRuns(id string, count int) []time.Time {
	s.mu.RLock()
	entryID, ok := s.entries[id]
	s.mu.RUnlock()

	if !ok {
		return nil
	}

	entry := s.cron.Entry(entryID)
	if entry.ID == 0 {
		return nil
	}

	runs := make([]time.Time, 0, count)
	next := entry.Next
	for i := 0; i < count; i++ {
		runs = append(runs, next)
		next = entry.Schedule.Next(next)
	}

	return runs
}

func (s *Scheduler) scheduleJob(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.entries[job.ID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, job.ID)
	}

	entryID, err := s.cron.AddFunc(job.Schedule, func() {
		s.executeJob(job)
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	s.entries[job.ID] = entryID

	entry := s.cron.Entry(entryID)
	nextRun := entry.Next
	job.NextRun = &nextRun

	s.logger.Info("scheduled job",
		"job_id", job.ID,
		"job_name", job.Name,
		"schedule", job.Schedule,
		"next_run", nextRun)

	return nil
}

func (s *Scheduler) unscheduleJob(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.entries[id]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, id)
	}
}

func (s *Scheduler) executeJob(job *Job) {
	ctx := context.Background()
	startTime := time.Now()

	exec := &JobExecution{
		ID:        fmt.Sprintf("exec-%d", startTime.UnixNano()),
		JobID:     job.ID,
		Status:    StatusRunning,
		StartedAt: startTime,
	}

	if err := s.store.CreateExecution(ctx, exec); err != nil {
		s.logger.Error("failed to create execution record", "error", err)
	}

	s.logger.Info("executing job",
		"job_id", job.ID,
		"job_name", job.Name,
		"execution_id", exec.ID)

	s.mu.RLock()
	handler, ok := s.handlers[job.JobType]
	s.mu.RUnlock()

	if !ok {
		exec.Status = StatusFailed
		exec.Error = fmt.Sprintf("no handler registered for job type: %s", job.JobType)
		endTime := time.Now()
		exec.EndedAt = &endTime
		_ = s.store.UpdateExecution(ctx, exec)
		return
	}

	err := handler(ctx, job)
	endTime := time.Now()
	exec.EndedAt = &endTime

	if err != nil {
		exec.Status = StatusFailed
		exec.Error = err.Error()
		s.logger.Error("job execution failed",
			"job_id", job.ID,
			"job_name", job.Name,
			"error", err,
			"duration", endTime.Sub(startTime))
	} else {
		exec.Status = StatusCompleted
		s.logger.Info("job execution completed",
			"job_id", job.ID,
			"job_name", job.Name,
			"duration", endTime.Sub(startTime))
	}

	_ = s.store.UpdateExecution(ctx, exec)
	_ = s.store.UpdateLastRun(ctx, job.ID, startTime)
}

type DefaultHandlers struct {
	ScanFunc       func(ctx context.Context, accountID string) error
	ScanAllFunc    func(ctx context.Context) error
	CleanupFunc    func(ctx context.Context, olderThan time.Duration) error
	ReportFunc     func(ctx context.Context, config map[string]string) error
	SyncAccessFunc func(ctx context.Context) error
}

func (h *DefaultHandlers) Register(s *Scheduler) {
	if h.ScanFunc != nil {
		s.RegisterHandler(JobTypeScanAccount, func(ctx context.Context, job *Job) error {
			accountID := job.Config["account_id"]
			if accountID == "" {
				return fmt.Errorf("account_id not specified in job config")
			}
			return h.ScanFunc(ctx, accountID)
		})
	}

	if h.ScanAllFunc != nil {
		s.RegisterHandler(JobTypeScanAllAccounts, func(ctx context.Context, job *Job) error {
			return h.ScanAllFunc(ctx)
		})
	}

	if h.CleanupFunc != nil {
		s.RegisterHandler(JobTypeCleanupOld, func(ctx context.Context, job *Job) error {
			days := 30
			if d, ok := job.Config["retention_days"]; ok {
				_, _ = fmt.Sscanf(d, "%d", &days)
			}
			return h.CleanupFunc(ctx, time.Duration(days)*24*time.Hour)
		})
	}

	if h.ReportFunc != nil {
		s.RegisterHandler(JobTypeGenerateReport, func(ctx context.Context, job *Job) error {
			return h.ReportFunc(ctx, job.Config)
		})
	}

	if h.SyncAccessFunc != nil {
		s.RegisterHandler(JobTypeSyncAccessGraph, func(ctx context.Context, job *Job) error {
			return h.SyncAccessFunc(ctx)
		})
	}
}
