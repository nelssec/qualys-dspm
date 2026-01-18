package api

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"

	"github.com/qualys/dspm/internal/classifier"
	awsconn "github.com/qualys/dspm/internal/connectors/aws"
	"github.com/qualys/dspm/internal/models"
	"github.com/qualys/dspm/internal/scanner"
	"github.com/qualys/dspm/internal/store"
)

// assetClassificationUpdate tracks classification summary for an asset
type assetClassificationUpdate struct {
	maxSensitivity models.Sensitivity
	categories     map[string]bool
	totalCount     int
}

// ScanExecutor handles background scan execution
type ScanExecutor struct {
	store      *store.Store
	scanner    *scanner.Scanner
	classifier *classifier.Classifier
	logger     *slog.Logger
	mu         sync.Mutex
	running    map[uuid.UUID]context.CancelFunc
	// Map scanner-generated asset IDs to actual database IDs
	assetIDMap   map[uuid.UUID]uuid.UUID
	assetIDMapMu sync.RWMutex
	// Batch size for bulk inserts (10K rows = ~100x faster than individual inserts)
	batchSize int
}

// NewScanExecutor creates a new scan executor
func NewScanExecutor(st *store.Store, logger *slog.Logger) *ScanExecutor {
	return &ScanExecutor{
		store:      st,
		scanner:    scanner.New(scanner.DefaultConfig()),
		classifier: classifier.New(),
		logger:     logger,
		running:    make(map[uuid.UUID]context.CancelFunc),
		assetIDMap: make(map[uuid.UUID]uuid.UUID),
		batchSize:  1000, // Batch 1000 classifications before bulk insert
	}
}

// ExecuteScan starts a scan in the background
func (e *ScanExecutor) ExecuteScan(ctx context.Context, job *models.ScanJob, account *models.CloudAccount) {
	e.mu.Lock()
	scanCtx, cancel := context.WithCancel(context.Background())
	e.running[job.ID] = cancel
	e.mu.Unlock()

	go func() {
		defer func() {
			e.mu.Lock()
			delete(e.running, job.ID)
			e.mu.Unlock()
		}()

		e.logger.Info("starting scan", "job_id", job.ID, "account_id", account.ID, "scan_type", job.ScanType)

		var finalStatus models.ScanStatus
		if err := e.runScan(scanCtx, job, account); err != nil {
			e.logger.Error("scan failed", "job_id", job.ID, "error", err)
			finalStatus = models.ScanStatusFailed
			if err := e.store.UpdateScanJobStatus(scanCtx, job.ID, models.ScanStatusFailed, "scan-executor"); err != nil {
				e.logger.Error("failed to update job status to failed", "job_id", job.ID, "error", err)
			}
		} else {
			e.logger.Info("scan completed successfully", "job_id", job.ID)
			finalStatus = models.ScanStatusCompleted
			if err := e.store.UpdateScanJobStatus(scanCtx, job.ID, models.ScanStatusCompleted, "scan-executor"); err != nil {
				e.logger.Error("failed to update job status to completed", "job_id", job.ID, "error", err)
			}
		}

		// Update account last_scanned_at with the FINAL status (not the original job.Status)
		e.logger.Info("updating account last scan", "account_id", account.ID, "status", finalStatus)
		if err := e.store.UpdateAccountLastScan(scanCtx, account.ID, string(finalStatus)); err != nil {
			e.logger.Error("failed to update account last scan", "account_id", account.ID, "error", err)
		}
		e.logger.Info("scan goroutine finished", "job_id", job.ID)
	}()
}

func (e *ScanExecutor) runScan(ctx context.Context, job *models.ScanJob, account *models.CloudAccount) error {
	e.logger.Info("runScan: updating job status to running", "job_id", job.ID)
	// Update job to running
	if err := e.store.UpdateScanJobStatus(ctx, job.ID, models.ScanStatusRunning, "scan-executor"); err != nil {
		return fmt.Errorf("updating job status: %w", err)
	}

	e.logger.Info("runScan: creating connector", "job_id", job.ID, "region", account.ConnectorConfig["region"])
	// Create connector
	conn, err := e.createConnector(ctx, account)
	if err != nil {
		return fmt.Errorf("creating connector: %w", err)
	}
	defer conn.Close()
	e.logger.Info("runScan: connector created successfully", "job_id", job.ID)

	// Validate connection
	e.logger.Info("runScan: validating connection", "job_id", job.ID)
	if err := conn.Validate(ctx); err != nil {
		return fmt.Errorf("validating connection: %w", err)
	}
	e.logger.Info("runScan: connection validated", "job_id", job.ID)

	// Run storage scan
	e.logger.Info("runScan: starting storage scan", "job_id", job.ID)
	return e.runStorageScan(ctx, job, account, conn)
}

func (e *ScanExecutor) createConnector(ctx context.Context, account *models.CloudAccount) (*awsconn.Connector, error) {
	if account.Provider != models.ProviderAWS {
		return nil, fmt.Errorf("unsupported provider: %s", account.Provider)
	}

	cfg := awsconn.Config{
		Region: getStringFromConfig(account.ConnectorConfig, "region", "us-east-1"),
	}
	if roleArn, ok := account.ConnectorConfig["role_arn"].(string); ok {
		cfg.AssumeRoleARN = roleArn
	}
	if extID, ok := account.ConnectorConfig["external_id"].(string); ok {
		cfg.ExternalID = extID
	}

	return awsconn.New(ctx, cfg)
}

func (e *ScanExecutor) runStorageScan(ctx context.Context, job *models.ScanJob, account *models.CloudAccount, conn *awsconn.Connector) error {
	// Parse scope from job
	var scope *scanner.ScanScope
	if job.ScanScope != nil {
		scope = &scanner.ScanScope{}
		if buckets, ok := job.ScanScope["buckets"].([]interface{}); ok {
			for _, b := range buckets {
				if bs, ok := b.(string); ok {
					scope.Buckets = append(scope.Buckets, bs)
				}
			}
		}
		if regions, ok := job.ScanScope["regions"].([]interface{}); ok {
			for _, r := range regions {
				if rs, ok := r.(string); ok {
					scope.Regions = append(scope.Regions, rs)
				}
			}
		}
	}

	scannerJob := &scanner.ScanJob{
		ID:        job.ID,
		AccountID: job.AccountID,
		ScanType:  job.ScanType,
		Scope:     scope,
	}

	// Create a new scanner for this job
	e.logger.Info("runStorageScan: creating scanner instance", "job_id", job.ID)
	scannerInstance := scanner.New(scanner.DefaultConfig())
	assetCh, classifyCh, findingCh, errorCh := scannerInstance.Results()

	// Collect results in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.logger.Info("collectResults: starting", "job_id", job.ID)
		e.collectResults(ctx, job.ID, account.ID, assetCh, classifyCh, findingCh, errorCh)
		e.logger.Info("collectResults: finished", "job_id", job.ID)
	}()

	// Run the scan
	e.logger.Info("runStorageScan: calling ScanStorage", "job_id", job.ID)
	progress, err := scannerInstance.ScanStorage(ctx, conn, scannerJob)
	e.logger.Info("runStorageScan: ScanStorage returned", "job_id", job.ID, "error", err)

	// Close scanner to signal end of results
	e.logger.Info("runStorageScan: closing scanner channels", "job_id", job.ID)
	scannerInstance.Close()
	e.logger.Info("runStorageScan: scanner channels closed", "job_id", job.ID)

	// Wait for results to be collected
	e.logger.Info("runStorageScan: waiting for collector goroutine", "job_id", job.ID)
	wg.Wait()
	e.logger.Info("runStorageScan: collector goroutine finished", "job_id", job.ID)

	// Update progress
	if progress != nil {
		if err := e.store.UpdateScanJobProgress(ctx, job.ID,
			progress.ScannedAssets, progress.FindingsFound, progress.ClassificationsFound); err != nil {
			e.logger.Error("failed to update scan job progress", "job_id", job.ID, "error", err)
		}
		e.logger.Info("scan progress",
			"job_id", job.ID,
			"assets", progress.ScannedAssets,
			"objects", progress.ScannedObjects,
			"classifications", progress.ClassificationsFound,
			"findings", progress.FindingsFound)
	}

	e.logger.Info("runStorageScan: returning", "job_id", job.ID)
	return err
}

func (e *ScanExecutor) collectResults(ctx context.Context, jobID, accountID uuid.UUID,
	assetCh <-chan *scanner.AssetResult,
	classifyCh <-chan *scanner.ClassificationResult,
	findingCh <-chan *scanner.FindingResult,
	errorCh <-chan *scanner.ScanError) {

	// Batch classifications for bulk insert
	var classificationBatch []*models.Classification
	// Track asset updates for batch summary updates
	assetUpdates := make(map[uuid.UUID]*assetClassificationUpdate)
	// Queue findings to save after all assets are processed
	var findingsQueue []*scanner.FindingResult

	flushBatch := func() {
		if len(classificationBatch) == 0 {
			return
		}
		if err := e.store.BatchCreateClassifications(ctx, classificationBatch); err != nil {
			e.logger.Error("failed to batch insert classifications", "count", len(classificationBatch), "error", err)
			// Fallback to individual inserts
			for _, c := range classificationBatch {
				if err := e.store.CreateClassification(ctx, c); err != nil {
					e.logger.Error("failed to save classification", "object", c.ObjectPath, "error", err)
				}
			}
		} else {
			e.logger.Debug("batch inserted classifications", "count", len(classificationBatch))
		}

		// Update asset classification summaries
		for assetID, update := range assetUpdates {
			categories := make([]string, 0, len(update.categories))
			for cat := range update.categories {
				categories = append(categories, cat)
			}
			if err := e.store.UpdateAssetClassification(ctx, assetID, update.maxSensitivity, categories, update.totalCount); err != nil {
				e.logger.Error("failed to update asset classification", "asset_id", assetID, "error", err)
			}
		}
		classificationBatch = nil
		assetUpdates = make(map[uuid.UUID]*assetClassificationUpdate)
	}

	// Save all queued findings after assets are processed
	saveQueuedFindings := func() {
		for _, finding := range findingsQueue {
			e.saveFinding(ctx, accountID, finding)
		}
		findingsQueue = nil
	}

	for {
		// Check exit condition BEFORE entering select to avoid blocking on nil channels
		if assetCh == nil && classifyCh == nil && findingCh == nil && errorCh == nil {
			e.logger.Info("collectResults: all channels closed, exiting", "job_id", jobID)
			// Save all queued findings now that assets are processed
			saveQueuedFindings()
			return
		}

		select {
		case <-ctx.Done():
			flushBatch()
			saveQueuedFindings()
			return

		case asset, ok := <-assetCh:
			if !ok {
				e.logger.Info("collectResults: assetCh closed", "job_id", jobID)
				assetCh = nil
				continue
			}
			if asset != nil {
				e.saveAsset(ctx, accountID, asset)
			}

		case classification, ok := <-classifyCh:
			if !ok {
				e.logger.Info("collectResults: classifyCh closed", "job_id", jobID)
				classifyCh = nil
				flushBatch() // Flush remaining when channel closes
				continue
			}
			if classification != nil {
				batch := e.convertClassificationResult(classification)
				classificationBatch = append(classificationBatch, batch...)

				// Track asset summary updates
				for _, c := range batch {
					update := assetUpdates[c.AssetID]
					if update == nil {
						update = &assetClassificationUpdate{
							maxSensitivity: models.SensitivityLow,
							categories:     make(map[string]bool),
						}
						assetUpdates[c.AssetID] = update
					}
					update.categories[string(c.Category)] = true
					update.totalCount += c.FindingCount
					if compareSensitivity(c.Sensitivity, update.maxSensitivity) > 0 {
						update.maxSensitivity = c.Sensitivity
					}
				}

				if len(classificationBatch) >= e.batchSize {
					flushBatch()
				}
			}

		case finding, ok := <-findingCh:
			if !ok {
				e.logger.Info("collectResults: findingCh closed", "job_id", jobID)
				findingCh = nil
				continue
			}
			if finding != nil {
				// Queue findings to save after all assets are processed
				findingsQueue = append(findingsQueue, finding)
			}

		case scanErr, ok := <-errorCh:
			if !ok {
				e.logger.Info("collectResults: errorCh closed", "job_id", jobID)
				errorCh = nil
				continue
			}
			if scanErr != nil {
				e.logger.Warn("scan error", "job_id", jobID, "error", scanErr.Error, "asset", scanErr.AssetARN)
			}
		}
	}
}

// convertClassificationResult converts a scanner result to model classifications
func (e *ScanExecutor) convertClassificationResult(result *scanner.ClassificationResult) []*models.Classification {
	if len(result.Matches) == 0 {
		return nil
	}

	// Look up the actual database ID for this asset
	assetID := result.AssetID
	e.assetIDMapMu.RLock()
	if mappedID, ok := e.assetIDMap[assetID]; ok {
		assetID = mappedID
	}
	e.assetIDMapMu.RUnlock()

	var classifications []*models.Classification
	for _, match := range result.Matches {
		// Build sample matches for the API/UI
		var sampleMatches []map[string]interface{}
		for _, sm := range match.SampleMatches {
			sampleMatches = append(sampleMatches, map[string]interface{}{
				"line":        sm.LineNumber,
				"column":      sm.ColumnNumber,
				"column_name": sm.ColumnName,
				"value":       sm.MaskedValue,
				"context":     sm.Context,
			})
		}

		// Build match locations (line numbers)
		var matchLocations []map[string]interface{}
		for i, lineNum := range match.LineNumbers {
			loc := map[string]interface{}{
				"line": lineNum,
			}
			if i < len(match.SampleMatches) {
				loc["column"] = match.SampleMatches[i].ColumnNumber
				if match.SampleMatches[i].ColumnName != "" {
					loc["column_name"] = match.SampleMatches[i].ColumnName
				}
			}
			matchLocations = append(matchLocations, loc)
		}

		classifications = append(classifications, &models.Classification{
			AssetID:         assetID,
			ObjectPath:      result.ObjectPath,
			ObjectSize:      result.ObjectSize,
			RuleName:        match.RuleName,
			Category:        match.Category,
			Sensitivity:     match.Sensitivity,
			FindingCount:    match.Count,
			ConfidenceScore: match.Confidence,
			SampleMatches:   models.JSONB{"samples": sampleMatches},
			MatchLocations:  models.JSONB{"locations": matchLocations},
		})
	}
	return classifications
}

func (e *ScanExecutor) saveAsset(ctx context.Context, accountID uuid.UUID, result *scanner.AssetResult) {
	if result.Asset == nil {
		return
	}
	asset := result.Asset
	originalID := asset.ID // Capture the scanner-generated ID
	asset.AccountID = accountID
	asset.ScanStatus = "scanned"

	if err := e.store.UpsertAsset(ctx, asset); err != nil {
		e.logger.Error("failed to save asset", "arn", asset.ResourceARN, "error", err)
		return
	}

	// Clear old classifications and findings for this asset before adding new scan results
	// This ensures we only show data from the latest scan
	if err := e.store.DeleteClassificationsForAsset(ctx, asset.ID); err != nil {
		e.logger.Error("failed to clear old classifications", "asset_id", asset.ID, "error", err)
	}
	if err := e.store.DeleteFindingsForAsset(ctx, asset.ID); err != nil {
		e.logger.Error("failed to clear old findings", "asset_id", asset.ID, "error", err)
	}

	// Track the mapping from scanner-generated ID to actual database ID
	if originalID != asset.ID {
		e.assetIDMapMu.Lock()
		e.assetIDMap[originalID] = asset.ID
		e.assetIDMapMu.Unlock()
		e.logger.Debug("mapped asset ID", "scanner_id", originalID, "db_id", asset.ID)
	}
}

func (e *ScanExecutor) saveClassification(ctx context.Context, result *scanner.ClassificationResult) {
	if len(result.Matches) == 0 {
		return
	}

	// Look up the actual database ID for this asset
	assetID := result.AssetID
	e.assetIDMapMu.RLock()
	if mappedID, ok := e.assetIDMap[assetID]; ok {
		assetID = mappedID
	}
	e.assetIDMapMu.RUnlock()

	// Find the highest sensitivity match
	var maxSensitivity models.Sensitivity = models.SensitivityLow
	var categories []string
	categorySet := make(map[string]bool)

	for _, match := range result.Matches {
		// Create a classification for each match
		classification := &models.Classification{
			AssetID:         assetID,
			ObjectPath:      result.ObjectPath,
			ObjectSize:      result.ObjectSize,
			RuleName:        match.RuleName,
			Category:        match.Category,
			Sensitivity:     match.Sensitivity,
			FindingCount:    match.Count,
			ConfidenceScore: match.Confidence,
		}

		if err := e.store.CreateClassification(ctx, classification); err != nil {
			e.logger.Error("failed to save classification", "object", result.ObjectPath, "rule", match.RuleName, "error", err)
			continue
		}

		// Track categories and max sensitivity
		if !categorySet[string(match.Category)] {
			categorySet[string(match.Category)] = true
			categories = append(categories, string(match.Category))
		}
		if compareSensitivity(match.Sensitivity, maxSensitivity) > 0 {
			maxSensitivity = match.Sensitivity
		}
	}

	// Update asset classification summary
	if err := e.store.UpdateAssetClassification(ctx, assetID, maxSensitivity, categories, len(result.Matches)); err != nil {
		e.logger.Error("failed to update asset classification", "asset_id", assetID, "error", err)
	}
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

func (e *ScanExecutor) saveFinding(ctx context.Context, accountID uuid.UUID, result *scanner.FindingResult) {
	if result.Finding == nil {
		return
	}

	finding := result.Finding
	finding.AccountID = accountID
	if finding.Status == "" {
		finding.Status = models.FindingStatusOpen
	}

	// Look up the actual database ID for this asset
	if finding.AssetID != nil {
		e.assetIDMapMu.RLock()
		if mappedID, ok := e.assetIDMap[*finding.AssetID]; ok {
			finding.AssetID = &mappedID
		}
		e.assetIDMapMu.RUnlock()
	}

	if err := e.store.CreateFinding(ctx, finding); err != nil {
		e.logger.Error("failed to save finding", "title", finding.Title, "error", err)
	}
}

func getStringFromConfig(config models.JSONB, key, defaultVal string) string {
	if config == nil {
		return defaultVal
	}
	if val, ok := config[key].(string); ok && val != "" {
		return val
	}
	return defaultVal
}
