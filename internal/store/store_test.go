package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/qualys/dspm/internal/models"
)

// getTestDSN returns the test database DSN from environment
func getTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=dspm password=dspm_password dbname=dspm_test sslmode=disable"
	}
	return dsn
}

// skipIfNoTestDB skips the test if no test database is available
func skipIfNoTestDB(t *testing.T) *Store {
	t.Helper()

	store, err := New(Config{
		DSN:          getTestDSN(),
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	})
	if err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := store.Ping(ctx); err != nil {
		t.Skipf("Skipping test, database not reachable: %v", err)
		return nil
	}

	return store
}

func TestStore_CloudAccounts(t *testing.T) {
	store := skipIfNoTestDB(t)
	if store == nil {
		return
	}
	defer store.Close()

	ctx := context.Background()

	// Create account
	account := &models.CloudAccount{
		Provider:    models.ProviderAWS,
		ExternalID:  "123456789012",
		DisplayName: "Test Account",
		ConnectorConfig: models.JSONB{
			"role_arn": "arn:aws:iam::123456789012:role/DSPMRole",
		},
		Status: "active",
	}

	err := store.CreateAccount(ctx, account)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	if account.ID == uuid.Nil {
		t.Error("Expected account ID to be set")
	}

	// Get account
	retrieved, err := store.GetAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}

	if retrieved.Provider != account.Provider {
		t.Errorf("Expected provider %s, got %s", account.Provider, retrieved.Provider)
	}
	if retrieved.ExternalID != account.ExternalID {
		t.Errorf("Expected external_id %s, got %s", account.ExternalID, retrieved.ExternalID)
	}

	// Get by external ID
	retrieved, err = store.GetAccountByExternalID(ctx, models.ProviderAWS, "123456789012")
	if err != nil {
		t.Fatalf("GetAccountByExternalID failed: %v", err)
	}
	if retrieved == nil {
		t.Error("Expected to find account by external ID")
	}

	// List accounts
	accounts, err := store.ListAccounts(ctx, nil, nil)
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}
	if len(accounts) == 0 {
		t.Error("Expected at least one account")
	}

	// Update status
	err = store.UpdateAccountStatus(ctx, account.ID, "scanning", "Running full scan")
	if err != nil {
		t.Fatalf("UpdateAccountStatus failed: %v", err)
	}

	// Verify update
	retrieved, _ = store.GetAccount(ctx, account.ID)
	if retrieved.Status != "scanning" {
		t.Errorf("Expected status 'scanning', got %s", retrieved.Status)
	}

	// Cleanup
	err = store.DeleteAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	// Verify deletion
	retrieved, _ = store.GetAccount(ctx, account.ID)
	if retrieved != nil {
		t.Error("Expected account to be deleted")
	}
}

func TestStore_DataAssets(t *testing.T) {
	store := skipIfNoTestDB(t)
	if store == nil {
		return
	}
	defer store.Close()

	ctx := context.Background()

	// Create account first
	account := &models.CloudAccount{
		Provider:        models.ProviderAWS,
		ExternalID:      "test-assets-" + uuid.New().String()[:8],
		DisplayName:     "Test Assets Account",
		ConnectorConfig: models.JSONB{},
	}
	store.CreateAccount(ctx, account)
	defer store.DeleteAccount(ctx, account.ID)

	// Create asset
	asset := &models.DataAsset{
		AccountID:        account.ID,
		ResourceType:     models.ResourceTypeS3Bucket,
		ResourceARN:      "arn:aws:s3:::test-bucket-" + uuid.New().String()[:8],
		Region:           "us-east-1",
		Name:             "test-bucket",
		EncryptionStatus: models.EncryptionSSE,
		PublicAccess:     false,
		Tags:             models.JSONB{"env": "test"},
	}

	err := store.UpsertAsset(ctx, asset)
	if err != nil {
		t.Fatalf("UpsertAsset failed: %v", err)
	}

	// Get asset
	retrieved, err := store.GetAsset(ctx, asset.ID)
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}

	if retrieved.Name != asset.Name {
		t.Errorf("Expected name %s, got %s", asset.Name, retrieved.Name)
	}

	// Get by ARN
	retrieved, err = store.GetAssetByARN(ctx, asset.ResourceARN)
	if err != nil {
		t.Fatalf("GetAssetByARN failed: %v", err)
	}
	if retrieved == nil {
		t.Error("Expected to find asset by ARN")
	}

	// List assets
	assets, total, err := store.ListAssets(ctx, ListAssetFilters{
		AccountID: &account.ID,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}
	if total == 0 || len(assets) == 0 {
		t.Error("Expected at least one asset")
	}

	// Update classification
	err = store.UpdateAssetClassification(ctx, asset.ID, models.SensitivityHigh, []string{"PII"}, 5)
	if err != nil {
		t.Fatalf("UpdateAssetClassification failed: %v", err)
	}

	// Verify update
	retrieved, _ = store.GetAsset(ctx, asset.ID)
	if retrieved.SensitivityLevel != models.SensitivityHigh {
		t.Errorf("Expected sensitivity HIGH, got %s", retrieved.SensitivityLevel)
	}
}

func TestStore_Classifications(t *testing.T) {
	store := skipIfNoTestDB(t)
	if store == nil {
		return
	}
	defer store.Close()

	ctx := context.Background()

	// Create account and asset
	account := &models.CloudAccount{
		Provider:        models.ProviderAWS,
		ExternalID:      "test-class-" + uuid.New().String()[:8],
		ConnectorConfig: models.JSONB{},
	}
	store.CreateAccount(ctx, account)
	defer store.DeleteAccount(ctx, account.ID)

	asset := &models.DataAsset{
		AccountID:    account.ID,
		ResourceType: models.ResourceTypeS3Bucket,
		ResourceARN:  "arn:aws:s3:::test-class-" + uuid.New().String()[:8],
		Name:         "test-bucket",
	}
	store.UpsertAsset(ctx, asset)

	// Create classification
	classification := &models.Classification{
		AssetID:         asset.ID,
		ObjectPath:      "data/users.csv",
		ObjectSize:      1024,
		RuleName:        "EMAIL",
		Category:        models.CategoryPII,
		Sensitivity:     models.SensitivityMedium,
		FindingCount:    10,
		ConfidenceScore: 0.95,
	}

	err := store.CreateClassification(ctx, classification)
	if err != nil {
		t.Fatalf("CreateClassification failed: %v", err)
	}

	// List classifications
	classifications, err := store.ListClassificationsByAsset(ctx, asset.ID)
	if err != nil {
		t.Fatalf("ListClassificationsByAsset failed: %v", err)
	}
	if len(classifications) == 0 {
		t.Error("Expected at least one classification")
	}

	// Get stats
	stats, err := store.GetClassificationStats(ctx, &account.ID)
	if err != nil {
		t.Fatalf("GetClassificationStats failed: %v", err)
	}
	if stats["PII"] == 0 {
		t.Error("Expected PII count > 0")
	}
}

func TestStore_Findings(t *testing.T) {
	store := skipIfNoTestDB(t)
	if store == nil {
		return
	}
	defer store.Close()

	ctx := context.Background()

	// Create account
	account := &models.CloudAccount{
		Provider:        models.ProviderAWS,
		ExternalID:      "test-findings-" + uuid.New().String()[:8],
		ConnectorConfig: models.JSONB{},
	}
	store.CreateAccount(ctx, account)
	defer store.DeleteAccount(ctx, account.ID)

	// Create finding
	finding := &models.Finding{
		AccountID:            account.ID,
		FindingType:          "PUBLIC_BUCKET",
		Severity:             models.SeverityCritical,
		Title:                "Test bucket is public",
		Description:          "The bucket allows public access",
		Remediation:          "Disable public access",
		Status:               models.FindingStatusOpen,
		ComplianceFrameworks: []string{"GDPR-Art32", "PCI-DSS-1.3"},
	}

	err := store.CreateFinding(ctx, finding)
	if err != nil {
		t.Fatalf("CreateFinding failed: %v", err)
	}

	// Get finding
	retrieved, err := store.GetFinding(ctx, finding.ID)
	if err != nil {
		t.Fatalf("GetFinding failed: %v", err)
	}
	if retrieved.Title != finding.Title {
		t.Errorf("Expected title %s, got %s", finding.Title, retrieved.Title)
	}

	// List findings
	findings, total, err := store.ListFindings(ctx, ListFindingFilters{
		AccountID: &account.ID,
		Severity:  ptrTo(models.SeverityCritical),
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListFindings failed: %v", err)
	}
	if total == 0 || len(findings) == 0 {
		t.Error("Expected at least one finding")
	}

	// Update status
	err = store.UpdateFindingStatus(ctx, finding.ID, models.FindingStatusResolved, "Fixed")
	if err != nil {
		t.Fatalf("UpdateFindingStatus failed: %v", err)
	}

	// Verify update
	retrieved, _ = store.GetFinding(ctx, finding.ID)
	if retrieved.Status != models.FindingStatusResolved {
		t.Errorf("Expected status 'resolved', got %s", retrieved.Status)
	}
	if retrieved.ResolvedAt == nil {
		t.Error("Expected resolved_at to be set")
	}

	// Get stats
	stats, err := store.GetFindingStats(ctx, &account.ID)
	if err != nil {
		t.Fatalf("GetFindingStats failed: %v", err)
	}
	if stats["CRITICAL"] == nil {
		t.Error("Expected CRITICAL stats")
	}
}

func TestStore_ScanJobs(t *testing.T) {
	store := skipIfNoTestDB(t)
	if store == nil {
		return
	}
	defer store.Close()

	ctx := context.Background()

	// Create account
	account := &models.CloudAccount{
		Provider:        models.ProviderAWS,
		ExternalID:      "test-scans-" + uuid.New().String()[:8],
		ConnectorConfig: models.JSONB{},
	}
	store.CreateAccount(ctx, account)
	defer store.DeleteAccount(ctx, account.ID)

	// Create scan job
	job := &models.ScanJob{
		AccountID:   account.ID,
		ScanType:    models.ScanTypeFull,
		TriggeredBy: "test",
	}

	err := store.CreateScanJob(ctx, job)
	if err != nil {
		t.Fatalf("CreateScanJob failed: %v", err)
	}

	// Get job
	retrieved, err := store.GetScanJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetScanJob failed: %v", err)
	}
	if retrieved.Status != models.ScanStatusPending {
		t.Errorf("Expected status 'pending', got %s", retrieved.Status)
	}

	// Update status
	err = store.UpdateScanJobStatus(ctx, job.ID, models.ScanStatusRunning, "worker-1")
	if err != nil {
		t.Fatalf("UpdateScanJobStatus failed: %v", err)
	}

	// Update progress
	err = store.UpdateScanJobProgress(ctx, job.ID, 10, 5, 20)
	if err != nil {
		t.Fatalf("UpdateScanJobProgress failed: %v", err)
	}

	// Verify updates
	retrieved, _ = store.GetScanJob(ctx, job.ID)
	if retrieved.Status != models.ScanStatusRunning {
		t.Errorf("Expected status 'running', got %s", retrieved.Status)
	}
	if retrieved.ScannedAssets != 10 {
		t.Errorf("Expected scanned_assets 10, got %d", retrieved.ScannedAssets)
	}

	// List pending
	pending, err := store.ListPendingScanJobs(ctx, 10)
	if err != nil {
		t.Fatalf("ListPendingScanJobs failed: %v", err)
	}
	// Should not include our running job
	for _, j := range pending {
		if j.ID == job.ID {
			t.Error("Running job should not be in pending list")
		}
	}
}

// Helper function
func ptrTo[T any](v T) *T {
	return &v
}
