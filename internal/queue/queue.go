package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/qualys/dspm/internal/models"
)

const (
	ScanJobsQueue      = "dspm:jobs:scan"
	ScanJobsProcessing = "dspm:jobs:processing"
	ScanJobsCompleted  = "dspm:jobs:completed"
	ScanJobsFailed     = "dspm:jobs:failed"
	WorkerHeartbeatKey = "dspm:workers:heartbeat"
	JobStatusPrefix    = "dspm:job:status:"
	JobProgressPrefix  = "dspm:job:progress:"
)

type Config struct {
	Addr     string
	Password string
	DB       int
}

type Queue struct {
	client *redis.Client
}

func New(cfg Config) (*Queue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	return &Queue{client: client}, nil
}

func (q *Queue) Close() error {
	return q.client.Close()
}

type Job struct {
	ID        uuid.UUID       `json:"id"`
	Type      string          `json:"type"`
	AccountID uuid.UUID       `json:"account_id"`
	ScanType  models.ScanType `json:"scan_type"`
	Scope     *ScanScope      `json:"scope,omitempty"`
	Priority  int             `json:"priority"`
	CreatedAt time.Time       `json:"created_at"`
	Attempts  int             `json:"attempts"`
}

type ScanScope struct {
	Buckets  []string `json:"buckets,omitempty"`
	Regions  []string `json:"regions,omitempty"`
	Prefixes []string `json:"prefixes,omitempty"`
}

type JobProgress struct {
	JobID                uuid.UUID         `json:"job_id"`
	Status               models.ScanStatus `json:"status"`
	TotalAssets          int               `json:"total_assets"`
	ScannedAssets        int               `json:"scanned_assets"`
	TotalObjects         int               `json:"total_objects"`
	ScannedObjects       int               `json:"scanned_objects"`
	ClassificationsFound int               `json:"classifications_found"`
	FindingsFound        int               `json:"findings_found"`
	Errors               []string          `json:"errors"`
	StartedAt            *time.Time        `json:"started_at,omitempty"`
	UpdatedAt            time.Time         `json:"updated_at"`
	CompletedAt          *time.Time        `json:"completed_at,omitempty"`
	WorkerID             string            `json:"worker_id,omitempty"`
}

func (q *Queue) EnqueueScanJob(ctx context.Context, job *Job) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	job.CreatedAt = time.Now()

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshaling job: %w", err)
	}

	score := float64(time.Now().Unix()) - float64(job.Priority*1000)

	if err := q.client.ZAdd(ctx, ScanJobsQueue, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err(); err != nil {
		return fmt.Errorf("enqueueing job: %w", err)
	}

	progress := &JobProgress{
		JobID:     job.ID,
		Status:    models.ScanStatusPending,
		UpdatedAt: time.Now(),
	}
	if err := q.UpdateProgress(ctx, progress); err != nil {
		return fmt.Errorf("initializing progress: %w", err)
	}

	return nil
}

func (q *Queue) DequeueJob(ctx context.Context, workerID string) (*Job, error) {
	results, err := q.client.ZPopMin(ctx, ScanJobsQueue, 1).Result()
	if err != nil {
		return nil, fmt.Errorf("dequeuing job: %w", err)
	}

	if len(results) == 0 {
		return nil, nil // No jobs available
	}

	var job Job
	if err := json.Unmarshal([]byte(results[0].Member.(string)), &job); err != nil {
		return nil, fmt.Errorf("unmarshaling job: %w", err)
	}

	data, _ := json.Marshal(job)
	if err := q.client.SAdd(ctx, ScanJobsProcessing, string(data)).Err(); err != nil {
		q.client.ZAdd(ctx, ScanJobsQueue, redis.Z{
			Score:  results[0].Score,
			Member: results[0].Member,
		})
		return nil, fmt.Errorf("marking job as processing: %w", err)
	}

	now := time.Now()
	progress := &JobProgress{
		JobID:     job.ID,
		Status:    models.ScanStatusRunning,
		StartedAt: &now,
		UpdatedAt: now,
		WorkerID:  workerID,
	}
	_ = q.UpdateProgress(ctx, progress)

	return &job, nil
}

func (q *Queue) CompleteJob(ctx context.Context, job *Job, success bool) error {
	data, _ := json.Marshal(job)

	q.client.SRem(ctx, ScanJobsProcessing, string(data))

	targetSet := ScanJobsCompleted
	status := models.ScanStatusCompleted
	if !success {
		targetSet = ScanJobsFailed
		status = models.ScanStatusFailed
	}

	if err := q.client.SAdd(ctx, targetSet, string(data)).Err(); err != nil {
		return fmt.Errorf("marking job complete: %w", err)
	}

	now := time.Now()
	progress, _ := q.GetProgress(ctx, job.ID)
	if progress == nil {
		progress = &JobProgress{JobID: job.ID}
	}
	progress.Status = status
	progress.CompletedAt = &now
	progress.UpdatedAt = now
	_ = q.UpdateProgress(ctx, progress)

	return nil
}

func (q *Queue) RequeueJob(ctx context.Context, job *Job, errorMsg string) error {
	data, _ := json.Marshal(job)

	q.client.SRem(ctx, ScanJobsProcessing, string(data))

	job.Attempts++

	if job.Attempts >= 3 {
		return q.CompleteJob(ctx, job, false)
	}

	newData, _ := json.Marshal(job)
	backoff := time.Duration(job.Attempts*30) * time.Second
	score := float64(time.Now().Add(backoff).Unix())

	if err := q.client.ZAdd(ctx, ScanJobsQueue, redis.Z{
		Score:  score,
		Member: string(newData),
	}).Err(); err != nil {
		return fmt.Errorf("requeuing job: %w", err)
	}

	progress, _ := q.GetProgress(ctx, job.ID)
	if progress == nil {
		progress = &JobProgress{JobID: job.ID}
	}
	progress.Status = models.ScanStatusPending
	progress.Errors = append(progress.Errors, errorMsg)
	progress.UpdatedAt = time.Now()
	_ = q.UpdateProgress(ctx, progress)

	return nil
}

func (q *Queue) UpdateProgress(ctx context.Context, progress *JobProgress) error {
	progress.UpdatedAt = time.Now()
	data, err := json.Marshal(progress)
	if err != nil {
		return fmt.Errorf("marshaling progress: %w", err)
	}

	key := JobProgressPrefix + progress.JobID.String()
	if err := q.client.Set(ctx, key, string(data), 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("updating progress: %w", err)
	}

	return nil
}

func (q *Queue) GetProgress(ctx context.Context, jobID uuid.UUID) (*JobProgress, error) {
	key := JobProgressPrefix + jobID.String()
	data, err := q.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting progress: %w", err)
	}

	var progress JobProgress
	if err := json.Unmarshal([]byte(data), &progress); err != nil {
		return nil, fmt.Errorf("unmarshaling progress: %w", err)
	}

	return &progress, nil
}

func (q *Queue) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	pending, _ := q.client.ZCard(ctx, ScanJobsQueue).Result()
	processing, _ := q.client.SCard(ctx, ScanJobsProcessing).Result()
	completed, _ := q.client.SCard(ctx, ScanJobsCompleted).Result()
	failed, _ := q.client.SCard(ctx, ScanJobsFailed).Result()

	stats["pending"] = pending
	stats["processing"] = processing
	stats["completed"] = completed
	stats["failed"] = failed

	return stats, nil
}

func (q *Queue) WorkerHeartbeat(ctx context.Context, workerID string) error {
	return q.client.HSet(ctx, WorkerHeartbeatKey, workerID, time.Now().Unix()).Err()
}

func (q *Queue) GetActiveWorkers(ctx context.Context, timeout time.Duration) ([]string, error) {
	workers, err := q.client.HGetAll(ctx, WorkerHeartbeatKey).Result()
	if err != nil {
		return nil, fmt.Errorf("getting workers: %w", err)
	}

	var active []string
	cutoff := time.Now().Add(-timeout).Unix()

	for workerID, lastSeen := range workers {
		var ts int64
		_, _ = fmt.Sscanf(lastSeen, "%d", &ts)
		if ts > cutoff {
			active = append(active, workerID)
		}
	}

	return active, nil
}

func (q *Queue) CleanupStaleJobs(ctx context.Context, timeout time.Duration) (int, error) {
	jobs, err := q.client.SMembers(ctx, ScanJobsProcessing).Result()
	if err != nil {
		return 0, fmt.Errorf("getting processing jobs: %w", err)
	}

	cleaned := 0
	for _, jobData := range jobs {
		var job Job
		if err := json.Unmarshal([]byte(jobData), &job); err != nil {
			continue
		}

		progress, err := q.GetProgress(ctx, job.ID)
		if err != nil || progress == nil {
			continue
		}

		if time.Since(progress.UpdatedAt) > timeout {
			q.client.SRem(ctx, ScanJobsProcessing, jobData)

			job.Attempts++
			if job.Attempts < 3 {
				newData, _ := json.Marshal(job)
				q.client.ZAdd(ctx, ScanJobsQueue, redis.Z{
					Score:  float64(time.Now().Unix()),
					Member: string(newData),
				})
			} else {
				q.client.SAdd(ctx, ScanJobsFailed, jobData)
			}
			cleaned++
		}
	}

	return cleaned, nil
}
