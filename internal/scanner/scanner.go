package scanner

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/qualys/dspm/internal/classifier"
	"github.com/qualys/dspm/internal/connectors"
	"github.com/qualys/dspm/internal/models"
)

type Config struct {
	Workers         int
	MaxFileSize     int64
	SampleSize      int64
	FilesPerBucket  int
	RandomSamplePct float64
	ScanTimeout     time.Duration
}

func DefaultConfig() Config {
	return Config{
		Workers:         10,
		MaxFileSize:     100 * 1024 * 1024, // 100MB
		SampleSize:      1 * 1024 * 1024,   // 1MB
		FilesPerBucket:  1000,
		RandomSamplePct: 0.10,
		ScanTimeout:     5 * time.Minute,
	}
}

type Scanner struct {
	config     Config
	classifier *classifier.Classifier

	assetCh    chan *AssetResult
	classifyCh chan *ClassificationResult
	findingCh  chan *FindingResult
	errorCh    chan *ScanError
}

type AssetResult struct {
	Asset    *models.DataAsset
	Metadata map[string]interface{}
}

type ClassificationResult struct {
	AssetID      uuid.UUID
	ObjectPath   string
	ObjectSize   int64
	Matches      []classifier.Match
	ScannedBytes int64
}

type FindingResult struct {
	Finding *models.Finding
}

type ScanError struct {
	AssetARN string
	Phase    string
	Error    error
}

type ScanJob struct {
	ID        uuid.UUID
	AccountID uuid.UUID
	ScanType  models.ScanType
	Scope     *ScanScope
}

type ScanScope struct {
	Buckets  []string // Specific buckets to scan (empty = all)
	Regions  []string // Specific regions (empty = all)
	Prefixes []string // Object prefixes to include
	MaxDepth int      // Max directory depth
}

type ScanProgress struct {
	TotalAssets          int
	ScannedAssets        int
	TotalObjects         int
	ScannedObjects       int
	ClassificationsFound int
	FindingsFound        int
	Errors               int
	StartedAt            time.Time
	mu                   sync.Mutex
}

func New(config Config) *Scanner {
	return &Scanner{
		config:     config,
		classifier: classifier.New(),
		assetCh:    make(chan *AssetResult, 100),
		classifyCh: make(chan *ClassificationResult, 100),
		findingCh:  make(chan *FindingResult, 100),
		errorCh:    make(chan *ScanError, 100),
	}
}

func (s *Scanner) Results() (<-chan *AssetResult, <-chan *ClassificationResult, <-chan *FindingResult, <-chan *ScanError) {
	return s.assetCh, s.classifyCh, s.findingCh, s.errorCh
}

func (s *Scanner) ScanStorage(ctx context.Context, conn connectors.StorageConnector, job *ScanJob) (*ScanProgress, error) {
	progress := &ScanProgress{
		StartedAt: time.Now(),
	}

	buckets, err := conn.ListBuckets(ctx)
	if err != nil {
		return progress, fmt.Errorf("listing buckets: %w", err)
	}

	if job.Scope != nil && len(job.Scope.Buckets) > 0 {
		filtered := make([]connectors.BucketInfo, 0)
		bucketSet := make(map[string]bool)
		for _, b := range job.Scope.Buckets {
			bucketSet[b] = true
		}
		for _, bucket := range buckets {
			if bucketSet[bucket.Name] {
				filtered = append(filtered, bucket)
			}
		}
		buckets = filtered
	}

	progress.TotalAssets = len(buckets)

	bucketCh := make(chan connectors.BucketInfo, len(buckets))
	var wg sync.WaitGroup

	for i := 0; i < s.config.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for bucket := range bucketCh {
				s.scanBucket(ctx, conn, bucket, job, progress)
			}
		}()
	}

	for _, bucket := range buckets {
		select {
		case bucketCh <- bucket:
		case <-ctx.Done():
			close(bucketCh)
			return progress, ctx.Err()
		}
	}
	close(bucketCh)

	wg.Wait()

	return progress, nil
}

func (s *Scanner) scanBucket(ctx context.Context, conn connectors.StorageConnector, bucket connectors.BucketInfo, job *ScanJob, progress *ScanProgress) {
	metadata, err := conn.GetBucketMetadata(ctx, bucket.Name)
	if err != nil {
		s.errorCh <- &ScanError{
			AssetARN: bucket.ARN,
			Phase:    "metadata",
			Error:    err,
		}
		return
	}

	asset := &models.DataAsset{
		ID:           uuid.New(),
		AccountID:    job.AccountID,
		ResourceType: models.ResourceTypeS3Bucket, // Adjust based on provider
		ResourceARN:  bucket.ARN,
		Region:       bucket.Region,
		Name:         bucket.Name,
		Tags:         models.JSONB(convertTags(metadata.Tags)),
	}

	if metadata.Encryption.Enabled {
		asset.EncryptionStatus = metadata.Encryption.Type
		asset.EncryptionKeyARN = metadata.Encryption.KeyARN
	} else {
		asset.EncryptionStatus = models.EncryptionNone
	}

	asset.VersioningEnabled = metadata.Versioning
	asset.LoggingEnabled = metadata.Logging.Enabled

	asset.PublicAccess = !metadata.PublicAccessBlock.BlockPublicAcls ||
		!metadata.PublicAccessBlock.BlockPublicPolicy

	policy, err := conn.GetBucketPolicy(ctx, bucket.Name)
	if err == nil && policy != nil && policy.IsPublic {
		asset.PublicAccess = true
		asset.PublicAccessDetails = models.JSONB{
			"policy_public":  true,
			"public_actions": policy.PublicActions,
		}
	}

	acl, err := conn.GetBucketACL(ctx, bucket.Name)
	if err == nil && acl != nil {
		for _, grant := range acl.Grants {
			if grant.IsPublic {
				asset.PublicAccess = true
				if asset.PublicAccessDetails == nil {
					asset.PublicAccessDetails = models.JSONB{}
				}
				asset.PublicAccessDetails["acl_public"] = true
				break
			}
		}
	}

	s.generateBucketFindings(asset, metadata, job.AccountID)

	s.assetCh <- &AssetResult{
		Asset: asset,
		Metadata: map[string]interface{}{
			"encryption": metadata.Encryption,
			"logging":    metadata.Logging,
			"versioning": metadata.Versioning,
		},
	}

	if job.ScanType == models.ScanTypeFull || job.ScanType == models.ScanTypeClassification {
		s.scanBucketContents(ctx, conn, bucket.Name, asset.ID, job, progress)
	}

	progress.mu.Lock()
	progress.ScannedAssets++
	progress.mu.Unlock()
}

func (s *Scanner) scanBucketContents(ctx context.Context, conn connectors.StorageConnector, bucketName string, assetID uuid.UUID, job *ScanJob, progress *ScanProgress) {
	objects, err := conn.ListObjects(ctx, bucketName, "", s.config.FilesPerBucket)
	if err != nil {
		s.errorCh <- &ScanError{
			AssetARN: bucketName,
			Phase:    "list_objects",
			Error:    err,
		}
		return
	}

	progress.mu.Lock()
	progress.TotalObjects += len(objects)
	progress.mu.Unlock()

	scannable := s.filterScannable(objects)

	objectCh := make(chan connectors.ObjectInfo, len(scannable))
	var wg sync.WaitGroup

	workers := s.config.Workers / 2
	if workers < 1 {
		workers = 1
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for obj := range objectCh {
				s.scanObject(ctx, conn, bucketName, obj, assetID, progress)
			}
		}()
	}

	for _, obj := range scannable {
		select {
		case objectCh <- obj:
		case <-ctx.Done():
			close(objectCh)
			return
		}
	}
	close(objectCh)

	wg.Wait()
}

func (s *Scanner) scanObject(ctx context.Context, conn connectors.StorageConnector, bucketName string, obj connectors.ObjectInfo, assetID uuid.UUID, progress *ScanProgress) {
	defer func() {
		progress.mu.Lock()
		progress.ScannedObjects++
		progress.mu.Unlock()
	}()

	var byteRange *connectors.ByteRange
	if obj.Size > s.config.SampleSize {
		byteRange = &connectors.ByteRange{
			Start: 0,
			End:   s.config.SampleSize - 1,
		}
	}

	reader, err := conn.GetObject(ctx, bucketName, obj.Key, byteRange)
	if err != nil {
		s.errorCh <- &ScanError{
			AssetARN: fmt.Sprintf("%s/%s", bucketName, obj.Key),
			Phase:    "get_object",
			Error:    err,
		}
		return
	}
	defer reader.Close()

	content, err := io.ReadAll(io.LimitReader(reader, s.config.SampleSize))
	if err != nil {
		s.errorCh <- &ScanError{
			AssetARN: fmt.Sprintf("%s/%s", bucketName, obj.Key),
			Phase:    "read_object",
			Error:    err,
		}
		return
	}

	result := s.classifier.Classify(string(content))

	if len(result.Matches) > 0 {
		s.classifyCh <- &ClassificationResult{
			AssetID:      assetID,
			ObjectPath:   obj.Key,
			ObjectSize:   obj.Size,
			Matches:      result.Matches,
			ScannedBytes: int64(len(content)),
		}

		progress.mu.Lock()
		progress.ClassificationsFound += result.TotalFindings
		progress.mu.Unlock()
	}
}

func (s *Scanner) filterScannable(objects []connectors.ObjectInfo) []connectors.ObjectInfo {
	highPriority := map[string]bool{
		".csv": true, ".json": true, ".xlsx": true, ".xls": true,
		".parquet": true, ".sql": true, ".log": true, ".txt": true,
		".tsv": true, ".xml": true, ".yaml": true, ".yml": true,
	}

	skip := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".mp4": true, ".mp3": true, ".wav": true, ".avi": true,
		".zip": true, ".gz": true, ".tar": true, ".rar": true,
		".exe": true, ".dll": true, ".so": true, ".bin": true,
		".pdf": true, ".doc": true, ".docx": true,
	}

	var high, medium []connectors.ObjectInfo

	for _, obj := range objects {
		if obj.Size > s.config.MaxFileSize {
			continue
		}

		if obj.Size == 0 {
			continue
		}

		ext := strings.ToLower(filepath.Ext(obj.Key))

		if skip[ext] {
			continue
		}

		if highPriority[ext] {
			high = append(high, obj)
		} else {
			medium = append(medium, obj)
		}
	}

	result := append(high, medium...)

	if len(result) > s.config.FilesPerBucket {
		result = result[:s.config.FilesPerBucket]
	}

	return result
}

func (s *Scanner) generateBucketFindings(asset *models.DataAsset, metadata *connectors.BucketMetadata, accountID uuid.UUID) {
	now := time.Now()

	if asset.PublicAccess {
		s.findingCh <- &FindingResult{
			Finding: &models.Finding{
				ID:          uuid.New(),
				AccountID:   accountID,
				AssetID:     &asset.ID,
				FindingType: "PUBLIC_BUCKET",
				Severity:    models.SeverityCritical,
				Title:       fmt.Sprintf("Public access enabled on bucket %s", asset.Name),
				Description: "This storage bucket allows public access which could expose sensitive data.",
				Remediation: "Disable public access by enabling Block Public Access settings and reviewing bucket policies.",
				Status:      models.FindingStatusOpen,
				ComplianceFrameworks: []string{
					"GDPR-Art32", "HIPAA-164.312", "PCI-DSS-1.3", "SOC2-CC6.1",
				},
				Evidence: models.JSONB{
					"public_access_details": asset.PublicAccessDetails,
				},
				CreatedAt:   now,
				UpdatedAt:   now,
				FirstSeenAt: now,
				LastSeenAt:  now,
			},
		}
	}

	if asset.EncryptionStatus == models.EncryptionNone {
		s.findingCh <- &FindingResult{
			Finding: &models.Finding{
				ID:          uuid.New(),
				AccountID:   accountID,
				AssetID:     &asset.ID,
				FindingType: "UNENCRYPTED_STORAGE",
				Severity:    models.SeverityHigh,
				Title:       fmt.Sprintf("Encryption not enabled on bucket %s", asset.Name),
				Description: "This storage bucket does not have server-side encryption enabled.",
				Remediation: "Enable default encryption using SSE-S3 or SSE-KMS.",
				Status:      models.FindingStatusOpen,
				ComplianceFrameworks: []string{
					"GDPR-Art32", "HIPAA-164.312(a)(2)(iv)", "PCI-DSS-3.4",
				},
				CreatedAt:   now,
				UpdatedAt:   now,
				FirstSeenAt: now,
				LastSeenAt:  now,
			},
		}
	}

	if !asset.VersioningEnabled {
		s.findingCh <- &FindingResult{
			Finding: &models.Finding{
				ID:          uuid.New(),
				AccountID:   accountID,
				AssetID:     &asset.ID,
				FindingType: "VERSIONING_DISABLED",
				Severity:    models.SeverityMedium,
				Title:       fmt.Sprintf("Versioning not enabled on bucket %s", asset.Name),
				Description: "This storage bucket does not have versioning enabled, making data recovery difficult.",
				Remediation: "Enable versioning to protect against accidental deletion and maintain object history.",
				Status:      models.FindingStatusOpen,
				ComplianceFrameworks: []string{
					"SOC2-A1.2",
				},
				CreatedAt:   now,
				UpdatedAt:   now,
				FirstSeenAt: now,
				LastSeenAt:  now,
			},
		}
	}

	if !asset.LoggingEnabled {
		s.findingCh <- &FindingResult{
			Finding: &models.Finding{
				ID:          uuid.New(),
				AccountID:   accountID,
				AssetID:     &asset.ID,
				FindingType: "LOGGING_DISABLED",
				Severity:    models.SeverityLow,
				Title:       fmt.Sprintf("Access logging not enabled on bucket %s", asset.Name),
				Description: "This storage bucket does not have access logging enabled.",
				Remediation: "Enable server access logging to track requests made to the bucket.",
				Status:      models.FindingStatusOpen,
				ComplianceFrameworks: []string{
					"SOC2-CC7.2", "PCI-DSS-10.2",
				},
				CreatedAt:   now,
				UpdatedAt:   now,
				FirstSeenAt: now,
				LastSeenAt:  now,
			},
		}
	}
}

func convertTags(tags map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range tags {
		result[k] = v
	}
	return result
}

func (s *Scanner) Close() {
	close(s.assetCh)
	close(s.classifyCh)
	close(s.findingCh)
	close(s.errorCh)
}
