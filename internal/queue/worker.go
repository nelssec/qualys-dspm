package queue

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/qualys/dspm/internal/classifier"
	"github.com/qualys/dspm/internal/config"
	"github.com/qualys/dspm/internal/connectors"
	awsconn "github.com/qualys/dspm/internal/connectors/aws"
	azureconn "github.com/qualys/dspm/internal/connectors/azure"
	gcpconn "github.com/qualys/dspm/internal/connectors/gcp"
	"github.com/qualys/dspm/internal/models"
	"github.com/qualys/dspm/internal/scanner"
	"github.com/qualys/dspm/internal/store"
)

type Worker struct {
	id         string
	queue      *Queue
	store      *store.Store
	config     *config.Config
	scanner    *scanner.Scanner
	classifier *classifier.Classifier

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	running bool
	mu      sync.Mutex
}

type WorkerConfig struct {
	Queue  *Queue
	Store  *store.Store
	Config *config.Config
}

func NewWorker(cfg WorkerConfig) *Worker {
	hostname, _ := os.Hostname()
	workerID := fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8])

	return &Worker{
		id:         workerID,
		queue:      cfg.Queue,
		store:      cfg.Store,
		config:     cfg.Config,
		scanner:    scanner.New(scanner.DefaultConfig()),
		classifier: classifier.New(),
	}
}

func (w *Worker) ID() string {
	return w.id
}

func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("worker already running")
	}
	w.running = true
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.mu.Unlock()

	log.Printf("[%s] Worker starting", w.id)

	w.wg.Add(1)
	go w.heartbeatLoop()

	w.wg.Add(1)
	go w.processLoop()

	w.wg.Add(1)
	go w.resultProcessor()

	return nil
}

func (w *Worker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	log.Printf("[%s] Worker stopping", w.id)
	w.cancel()
	w.wg.Wait()
	log.Printf("[%s] Worker stopped", w.id)
}

func (w *Worker) heartbeatLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.queue.WorkerHeartbeat(w.ctx, w.id)
		}
	}
}

func (w *Worker) processLoop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			job, err := w.queue.DequeueJob(w.ctx, w.id)
			if err != nil {
				log.Printf("[%s] Error dequeuing job: %v", w.id, err)
				time.Sleep(5 * time.Second)
				continue
			}

			if job == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			log.Printf("[%s] Processing job %s (type: %s, account: %s)",
				w.id, job.ID, job.ScanType, job.AccountID)

			if err := w.processJob(job); err != nil {
				log.Printf("[%s] Job %s failed: %v", w.id, job.ID, err)
				w.queue.RequeueJob(w.ctx, job, err.Error())
			} else {
				log.Printf("[%s] Job %s completed successfully", w.id, job.ID)
				w.queue.CompleteJob(w.ctx, job, true)
			}
		}
	}
}

func (w *Worker) processJob(job *Job) error {
	account, err := w.store.GetAccount(w.ctx, job.AccountID)
	if err != nil {
		return fmt.Errorf("getting account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found: %s", job.AccountID)
	}

	conn, err := w.createConnector(account)
	if err != nil {
		return fmt.Errorf("creating connector: %w", err)
	}
	defer conn.Close()

	if err := conn.Validate(w.ctx); err != nil {
		return fmt.Errorf("validating connection: %w", err)
	}

	scanJob := &models.ScanJob{
		ID:          job.ID,
		AccountID:   job.AccountID,
		ScanType:    job.ScanType,
		TriggeredBy: "worker",
		WorkerID:    w.id,
	}

	w.store.UpdateScanJobStatus(w.ctx, job.ID, models.ScanStatusRunning, w.id)

	switch job.ScanType {
	case models.ScanTypeFull, models.ScanTypeAssetDiscovery, models.ScanTypeClassification:
		return w.runStorageScan(job, conn, scanJob)
	case models.ScanTypeAccessAnalysis:
		return w.runAccessScan(job, conn, scanJob)
	default:
		return fmt.Errorf("unknown scan type: %s", job.ScanType)
	}
}

func (w *Worker) createConnector(account *models.CloudAccount) (connectors.Connector, error) {
	switch account.Provider {
	case models.ProviderAWS:
		cfg := awsconn.Config{
			Region: getStringFromConfig(account.ConnectorConfig, "region", "us-east-1"),
		}
		if roleArn, ok := account.ConnectorConfig["role_arn"].(string); ok {
			cfg.AssumeRoleARN = roleArn
		}
		if extID, ok := account.ConnectorConfig["external_id"].(string); ok {
			cfg.ExternalID = extID
		}
		return awsconn.New(w.ctx, cfg)

	case models.ProviderAzure:
		cfg := azureconn.Config{
			TenantID:       getStringFromConfig(account.ConnectorConfig, "tenant_id", ""),
			ClientID:       getStringFromConfig(account.ConnectorConfig, "client_id", ""),
			ClientSecret:   getStringFromConfig(account.ConnectorConfig, "client_secret", ""),
			SubscriptionID: getStringFromConfig(account.ConnectorConfig, "subscription_id", ""),
		}
		return azureconn.New(w.ctx, cfg)

	case models.ProviderGCP:
		cfg := gcpconn.Config{
			ProjectID:       getStringFromConfig(account.ConnectorConfig, "project_id", ""),
			CredentialsFile: getStringFromConfig(account.ConnectorConfig, "credentials_file", ""),
		}
		return gcpconn.New(w.ctx, cfg)

	default:
		return nil, fmt.Errorf("unsupported provider: %s", account.Provider)
	}
}

func (w *Worker) runStorageScan(job *Job, conn connectors.Connector, scanJob *models.ScanJob) error {
	storageConn, ok := conn.(connectors.StorageConnector)
	if !ok {
		return fmt.Errorf("connector does not support storage operations")
	}

	var scope *scanner.ScanScope
	if job.Scope != nil {
		scope = &scanner.ScanScope{
			Buckets:  job.Scope.Buckets,
			Regions:  job.Scope.Regions,
			Prefixes: job.Scope.Prefixes,
		}
	}

	scannerJob := &scanner.ScanJob{
		ID:        job.ID,
		AccountID: job.AccountID,
		ScanType:  job.ScanType,
		Scope:     scope,
	}

	assetCh, classifyCh, findingCh, errorCh := w.scanner.Results()

	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		w.collectResults(job.ID, assetCh, classifyCh, findingCh, errorCh)
	}()

	progress, err := w.scanner.ScanStorage(w.ctx, storageConn, scannerJob)

	w.scanner.Close()

	resultWg.Wait()

	if progress != nil {
		w.queue.UpdateProgress(w.ctx, &JobProgress{
			JobID:                job.ID,
			Status:               models.ScanStatusCompleted,
			TotalAssets:          progress.TotalAssets,
			ScannedAssets:        progress.ScannedAssets,
			TotalObjects:         progress.TotalObjects,
			ScannedObjects:       progress.ScannedObjects,
			ClassificationsFound: progress.ClassificationsFound,
			FindingsFound:        progress.FindingsFound,
		})

		w.store.UpdateScanJobProgress(w.ctx, job.ID,
			progress.ScannedAssets, progress.FindingsFound, progress.ClassificationsFound)
	}

	return err
}

func (w *Worker) runAccessScan(job *Job, conn connectors.Connector, scanJob *models.ScanJob) error {
	iamConn, ok := conn.(connectors.IAMConnector)
	if !ok {
		return fmt.Errorf("connector does not support IAM operations")
	}

	users, err := iamConn.ListUsers(w.ctx)
	if err != nil {
		log.Printf("[%s] Error listing users: %v", w.id, err)
	}

	roles, err := iamConn.ListRoles(w.ctx)
	if err != nil {
		log.Printf("[%s] Error listing roles: %v", w.id, err)
	}

	policies, err := iamConn.ListPolicies(w.ctx)
	if err != nil {
		log.Printf("[%s] Error listing policies: %v", w.id, err)
	}

	log.Printf("[%s] Access scan found %d users, %d roles, %d policies",
		w.id, len(users), len(roles), len(policies))

	for _, policy := range policies {
		policyDoc, _ := iamConn.GetPolicy(w.ctx, policy.ARN)

		accessPolicy := &models.AccessPolicy{
			ID:         uuid.New(),
			AccountID:  job.AccountID,
			PolicyARN:  policy.ARN,
			PolicyName: policy.Name,
			PolicyType: policy.Type,
		}

		if policyDoc != nil {
			accessPolicy.PolicyDocument = models.JSONB{
				"raw":        policyDoc.Raw,
				"statements": policyDoc.Statements,
			}
		}

	}

	return nil
}

func (w *Worker) collectResults(jobID uuid.UUID,
	assetCh <-chan *scanner.AssetResult,
	classifyCh <-chan *scanner.ClassificationResult,
	findingCh <-chan *scanner.FindingResult,
	errorCh <-chan *scanner.ScanError) {

	for {
		select {
		case asset, ok := <-assetCh:
			if !ok {
				assetCh = nil
				continue
			}
			if asset != nil {
				if err := w.store.UpsertAsset(w.ctx, asset.Asset); err != nil {
					log.Printf("[%s] Error storing asset: %v", w.id, err)
				}
			}

		case classification, ok := <-classifyCh:
			if !ok {
				classifyCh = nil
				continue
			}
			if classification != nil {
				for _, match := range classification.Matches {
					class := &models.Classification{
						AssetID:         classification.AssetID,
						ObjectPath:      classification.ObjectPath,
						ObjectSize:      classification.ObjectSize,
						RuleName:        match.RuleName,
						Category:        match.Category,
						Sensitivity:     match.Sensitivity,
						FindingCount:    match.Count,
						ConfidenceScore: match.Confidence,
						SampleMatches: models.JSONB{
							"value": match.Value,
						},
						MatchLocations: models.JSONB{
							"lines": match.LineNumbers,
						},
					}
					if err := w.store.CreateClassification(w.ctx, class); err != nil {
						log.Printf("[%s] Error storing classification: %v", w.id, err)
					}
				}

				if len(classification.Matches) > 0 {
					maxSens := models.SensitivityUnknown
					categories := make(map[models.Category]bool)
					count := 0

					for _, m := range classification.Matches {
						count += m.Count
						categories[m.Category] = true
						if compareSensitivity(m.Sensitivity, maxSens) > 0 {
							maxSens = m.Sensitivity
						}
					}

					var cats []string
					for c := range categories {
						cats = append(cats, string(c))
					}

					w.store.UpdateAssetClassification(w.ctx, classification.AssetID, maxSens, cats, count)
				}
			}

		case finding, ok := <-findingCh:
			if !ok {
				findingCh = nil
				continue
			}
			if finding != nil && finding.Finding != nil {
				if err := w.store.CreateFinding(w.ctx, finding.Finding); err != nil {
					log.Printf("[%s] Error storing finding: %v", w.id, err)
				}
			}

		case scanErr, ok := <-errorCh:
			if !ok {
				errorCh = nil
				continue
			}
			if scanErr != nil {
				log.Printf("[%s] Scan error for %s (%s): %v",
					w.id, scanErr.AssetARN, scanErr.Phase, scanErr.Error)
			}
		}

		if assetCh == nil && classifyCh == nil && findingCh == nil && errorCh == nil {
			return
		}
	}
}

func (w *Worker) resultProcessor() {
	defer w.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			cleaned, err := w.queue.CleanupStaleJobs(w.ctx, 30*time.Minute)
			if err != nil {
				log.Printf("[%s] Error cleaning stale jobs: %v", w.id, err)
			} else if cleaned > 0 {
				log.Printf("[%s] Cleaned up %d stale jobs", w.id, cleaned)
			}
		}
	}
}

func getStringFromConfig(cfg models.JSONB, key, defaultVal string) string {
	if val, ok := cfg[key].(string); ok {
		return val
	}
	return defaultVal
}

func compareSensitivity(a, b models.Sensitivity) int {
	order := map[models.Sensitivity]int{
		models.SensitivityCritical: 4,
		models.SensitivityHigh:     3,
		models.SensitivityMedium:   2,
		models.SensitivityLow:      1,
		models.SensitivityUnknown:  0,
	}
	return order[a] - order[b]
}
