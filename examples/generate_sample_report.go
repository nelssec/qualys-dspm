// Package main demonstrates how DSPM collects data and generates reports
// This can be run standalone to see sample output
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ============================================================================
// DATA COLLECTION MODELS - What gets discovered during scanning
// ============================================================================

// CloudAccount represents a connected cloud account
type CloudAccount struct {
	ID           string    `json:"id"`
	Provider     string    `json:"provider"` // AWS, Azure, GCP
	AccountID    string    `json:"account_id"`
	AccountName  string    `json:"account_name"`
	Region       string    `json:"region"`
	Status       string    `json:"status"`
	LastScanned  time.Time `json:"last_scanned"`
	AssetsCount  int       `json:"assets_count"`
	FindingsOpen int       `json:"findings_open"`
}

// DataAsset represents a discovered data resource
type DataAsset struct {
	ID                string                 `json:"id"`
	ARN               string                 `json:"arn"`
	ResourceType      string                 `json:"resource_type"` // S3_BUCKET, RDS_INSTANCE, DYNAMODB_TABLE
	Name              string                 `json:"name"`
	Region            string                 `json:"region"`
	EncryptionStatus  string                 `json:"encryption_status"` // NONE, SSE, SSE_KMS, CMK
	PublicAccess      bool                   `json:"public_access"`
	SensitivityLevel  string                 `json:"sensitivity_level"` // CRITICAL, HIGH, MEDIUM, LOW
	Classifications   []Classification       `json:"classifications"`
	DiscoveredAt      time.Time              `json:"discovered_at"`
	LastScannedAt     time.Time              `json:"last_scanned_at"`
	Tags              map[string]string      `json:"tags"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// Classification represents detected sensitive data
type Classification struct {
	ID              string    `json:"id"`
	Category        string    `json:"category"` // PII, PHI, PCI, SECRETS
	SubCategory     string    `json:"sub_category"`
	Confidence      float64   `json:"confidence"`
	MatchCount      int       `json:"match_count"`
	SampleLocations []string  `json:"sample_locations"`
	DetectedAt      time.Time `json:"detected_at"`
}

// Finding represents a security finding
type Finding struct {
	ID              string                 `json:"id"`
	AssetARN        string                 `json:"asset_arn"`
	Severity        string                 `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	Remediation     string                 `json:"remediation"`
	Status          string                 `json:"status"` // open, in_progress, resolved, suppressed
	ComplianceRefs  []string               `json:"compliance_refs"`
	Evidence        map[string]interface{} `json:"evidence"`
	DetectedAt      time.Time              `json:"detected_at"`
	ResolvedAt      *time.Time             `json:"resolved_at,omitempty"`
}

// DataLineage represents data flow between resources
type DataLineage struct {
	ID              string    `json:"id"`
	SourceARN       string    `json:"source_arn"`
	SourceType      string    `json:"source_type"`
	TargetARN       string    `json:"target_arn"`
	TargetType      string    `json:"target_type"`
	FlowType        string    `json:"flow_type"` // READS_FROM, WRITES_TO, EXPORTS_TO
	InferredFrom    string    `json:"inferred_from"`
	Confidence      float64   `json:"confidence"`
	ContainsPII     bool      `json:"contains_pii"`
	FirstObserved   time.Time `json:"first_observed"`
}

// EncryptionCompliance represents encryption posture
type EncryptionCompliance struct {
	AssetARN           string  `json:"asset_arn"`
	AtRestScore        int     `json:"at_rest_score"`
	InTransitScore     int     `json:"in_transit_score"`
	KeyManagementScore int     `json:"key_management_score"`
	OverallScore       int     `json:"overall_score"`
	Grade              string  `json:"grade"`
	Recommendations    []string `json:"recommendations"`
}

// ============================================================================
// QUALYS INTEGRATION MODELS - Format for Qualys ingestion
// ============================================================================

// QualysAssetPayload is the format Qualys expects for asset data
type QualysAssetPayload struct {
	ExternalID    string                 `json:"externalId"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Provider      string                 `json:"provider"`
	Region        string                 `json:"region"`
	Tags          []QualysTag            `json:"tags"`
	Attributes    map[string]interface{} `json:"attributes"`
	LastSeen      string                 `json:"lastSeen"`
}

type QualysTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// QualysFindingPayload is the format Qualys expects for findings
type QualysFindingPayload struct {
	ExternalID    string   `json:"externalId"`
	AssetID       string   `json:"assetId"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Severity      int      `json:"severity"` // 1-5 scale
	Category      string   `json:"category"`
	Status        string   `json:"status"`
	Remediation   string   `json:"remediation"`
	ComplianceIDs []string `json:"complianceIds"`
	DetectedDate  string   `json:"detectedDate"`
	Evidence      string   `json:"evidence"`
}

// QualysBulkImport represents a bulk import request to Qualys
type QualysBulkImport struct {
	Source    string                 `json:"source"`
	Timestamp string                 `json:"timestamp"`
	Assets    []QualysAssetPayload   `json:"assets"`
	Findings  []QualysFindingPayload `json:"findings"`
}

// ============================================================================
// SAMPLE DATA GENERATION
// ============================================================================

func generateSampleData() (*QualysBulkImport, []DataAsset, []Finding, []DataLineage) {
	now := time.Now()

	// Sample Assets
	assets := []DataAsset{
		{
			ID:               "asset-001",
			ARN:              "arn:aws:s3:::customer-data-prod",
			ResourceType:     "S3_BUCKET",
			Name:             "customer-data-prod",
			Region:           "us-east-1",
			EncryptionStatus: "SSE_KMS",
			PublicAccess:     false,
			SensitivityLevel: "CRITICAL",
			Classifications: []Classification{
				{
					ID:              "class-001",
					Category:        "PII",
					SubCategory:     "SSN",
					Confidence:      0.95,
					MatchCount:      1247,
					SampleLocations: []string{"customers/2024/q4/export.csv", "reports/annual/users.json"},
					DetectedAt:      now.Add(-2 * time.Hour),
				},
				{
					ID:              "class-002",
					Category:        "PII",
					SubCategory:     "EMAIL",
					Confidence:      0.98,
					MatchCount:      5832,
					SampleLocations: []string{"customers/contacts.csv"},
					DetectedAt:      now.Add(-2 * time.Hour),
				},
			},
			DiscoveredAt:  now.Add(-30 * 24 * time.Hour),
			LastScannedAt: now.Add(-2 * time.Hour),
			Tags: map[string]string{
				"Environment": "production",
				"Team":        "data-platform",
				"CostCenter":  "engineering",
			},
		},
		{
			ID:               "asset-002",
			ARN:              "arn:aws:s3:::app-logs-prod",
			ResourceType:     "S3_BUCKET",
			Name:             "app-logs-prod",
			Region:           "us-east-1",
			EncryptionStatus: "SSE",
			PublicAccess:     false,
			SensitivityLevel: "HIGH",
			Classifications: []Classification{
				{
					ID:              "class-003",
					Category:        "PCI",
					SubCategory:     "CREDIT_CARD",
					Confidence:      0.89,
					MatchCount:      23,
					SampleLocations: []string{"payment-service/2024-01-15.log"},
					DetectedAt:      now.Add(-5 * time.Hour),
				},
			},
			DiscoveredAt:  now.Add(-60 * 24 * time.Hour),
			LastScannedAt: now.Add(-5 * time.Hour),
			Tags: map[string]string{
				"Environment": "production",
				"Team":        "payments",
			},
		},
		{
			ID:               "asset-003",
			ARN:              "arn:aws:rds:us-east-1:123456789:db:healthcare-db",
			ResourceType:     "RDS_INSTANCE",
			Name:             "healthcare-db",
			Region:           "us-east-1",
			EncryptionStatus: "CMK",
			PublicAccess:     false,
			SensitivityLevel: "CRITICAL",
			Classifications: []Classification{
				{
					ID:              "class-004",
					Category:        "PHI",
					SubCategory:     "MEDICAL_RECORD",
					Confidence:      0.97,
					MatchCount:      15420,
					SampleLocations: []string{"patients table", "medical_history table"},
					DetectedAt:      now.Add(-1 * time.Hour),
				},
			},
			DiscoveredAt:  now.Add(-90 * 24 * time.Hour),
			LastScannedAt: now.Add(-1 * time.Hour),
			Tags: map[string]string{
				"Environment": "production",
				"Compliance":  "HIPAA",
			},
		},
		{
			ID:               "asset-004",
			ARN:              "arn:aws:s3:::deployments-config",
			ResourceType:     "S3_BUCKET",
			Name:             "deployments-config",
			Region:           "us-east-1",
			EncryptionStatus: "NONE",
			PublicAccess:     true,
			SensitivityLevel: "CRITICAL",
			Classifications: []Classification{
				{
					ID:              "class-005",
					Category:        "SECRETS",
					SubCategory:     "AWS_ACCESS_KEY",
					Confidence:      0.99,
					MatchCount:      3,
					SampleLocations: []string{"config/prod.env", "scripts/deploy.sh"},
					DetectedAt:      now.Add(-24 * time.Hour),
				},
			},
			DiscoveredAt:  now.Add(-45 * 24 * time.Hour),
			LastScannedAt: now.Add(-24 * time.Hour),
			Tags: map[string]string{
				"Environment": "production",
				"Team":        "devops",
			},
		},
	}

	// Sample Findings
	findings := []Finding{
		{
			ID:          "finding-001",
			AssetARN:    "arn:aws:s3:::customer-data-prod",
			Severity:    "CRITICAL",
			Title:       "Unencrypted SSN data detected in S3 bucket",
			Description: "Social Security Numbers were detected in plain text within CSV files. 1,247 instances found across 2 files.",
			Category:    "PII",
			Remediation: "1. Enable server-side encryption with KMS\n2. Implement data masking for SSN fields\n3. Review access policies",
			Status:      "open",
			ComplianceRefs: []string{"GDPR-Art32", "SOC2-CC6.1", "PCI-DSS-3.4"},
			Evidence: map[string]interface{}{
				"match_count":      1247,
				"sample_files":     []string{"customers/2024/q4/export.csv"},
				"pattern_detected": "XXX-XX-XXXX",
			},
			DetectedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:          "finding-002",
			AssetARN:    "arn:aws:s3:::app-logs-prod",
			Severity:    "CRITICAL",
			Title:       "Credit card numbers logged in application logs",
			Description: "Payment card numbers were found in application log files, violating PCI-DSS requirements.",
			Category:    "PCI",
			Remediation: "1. Implement log sanitization\n2. Update logging configuration to mask card numbers\n3. Rotate potentially exposed cards",
			Status:      "in_progress",
			ComplianceRefs: []string{"PCI-DSS-3.4", "PCI-DSS-10.2"},
			Evidence: map[string]interface{}{
				"match_count":   23,
				"log_file":      "payment-service/2024-01-15.log",
				"card_prefixes": []string{"4111****", "5500****"},
			},
			DetectedAt: now.Add(-5 * time.Hour),
		},
		{
			ID:          "finding-003",
			AssetARN:    "arn:aws:s3:::deployments-config",
			Severity:    "CRITICAL",
			Title:       "AWS access keys exposed in public bucket",
			Description: "Active AWS access keys were found in configuration files within a publicly accessible S3 bucket.",
			Category:    "SECRETS",
			Remediation: "1. IMMEDIATELY rotate exposed credentials\n2. Remove public access from bucket\n3. Enable bucket encryption\n4. Review CloudTrail for unauthorized access",
			Status:      "open",
			ComplianceRefs: []string{"CIS-AWS-1.14", "SOC2-CC6.1"},
			Evidence: map[string]interface{}{
				"key_prefix":     "AKIA****",
				"files_affected": []string{"config/prod.env", "scripts/deploy.sh"},
				"bucket_public":  true,
			},
			DetectedAt: now.Add(-24 * time.Hour),
		},
		{
			ID:          "finding-004",
			AssetARN:    "arn:aws:rds:us-east-1:123456789:db:healthcare-db",
			Severity:    "HIGH",
			Title:       "PHI data lacks field-level encryption",
			Description: "Protected Health Information in the patients table is not encrypted at the field level, only at rest.",
			Category:    "PHI",
			Remediation: "1. Implement application-level encryption for PHI fields\n2. Consider using AWS DynamoDB client-side encryption\n3. Update data handling procedures",
			Status:      "open",
			ComplianceRefs: []string{"HIPAA-164.312(a)(1)", "HIPAA-164.312(e)(1)"},
			Evidence: map[string]interface{}{
				"tables_affected": []string{"patients", "medical_history"},
				"phi_fields":      []string{"diagnosis", "treatment", "medications"},
			},
			DetectedAt: now.Add(-1 * time.Hour),
		},
	}

	// Sample Data Lineage
	lineage := []DataLineage{
		{
			ID:            "lineage-001",
			SourceARN:     "arn:aws:s3:::customer-data-prod",
			SourceType:    "S3_BUCKET",
			TargetARN:     "arn:aws:lambda:us-east-1:123456789:function:data-processor",
			TargetType:    "LAMBDA_FUNCTION",
			FlowType:      "READS_FROM",
			InferredFrom:  "ENV_VARIABLE",
			Confidence:    0.85,
			ContainsPII:   true,
			FirstObserved: now.Add(-7 * 24 * time.Hour),
		},
		{
			ID:            "lineage-002",
			SourceARN:     "arn:aws:lambda:us-east-1:123456789:function:data-processor",
			SourceType:    "LAMBDA_FUNCTION",
			TargetARN:     "arn:aws:dynamodb:us-east-1:123456789:table/users",
			TargetType:    "DYNAMODB_TABLE",
			FlowType:      "WRITES_TO",
			InferredFrom:  "IAM_POLICY",
			Confidence:    0.92,
			ContainsPII:   true,
			FirstObserved: now.Add(-7 * 24 * time.Hour),
		},
		{
			ID:            "lineage-003",
			SourceARN:     "arn:aws:dynamodb:us-east-1:123456789:table/users",
			SourceType:    "DYNAMODB_TABLE",
			TargetARN:     "arn:aws:sagemaker:us-east-1:123456789:training-job/customer-model",
			TargetType:    "SAGEMAKER_TRAINING",
			FlowType:      "READS_FROM",
			InferredFrom:  "IAM_POLICY",
			Confidence:    0.78,
			ContainsPII:   true,
			FirstObserved: now.Add(-3 * 24 * time.Hour),
		},
	}

	// Convert to Qualys format
	qualysPayload := convertToQualysFormat(assets, findings)

	return qualysPayload, assets, findings, lineage
}

func convertToQualysFormat(assets []DataAsset, findings []Finding) *QualysBulkImport {
	qualysAssets := make([]QualysAssetPayload, 0, len(assets))
	qualysFindings := make([]QualysFindingPayload, 0, len(findings))

	for _, asset := range assets {
		tags := make([]QualysTag, 0)
		for k, v := range asset.Tags {
			tags = append(tags, QualysTag{Key: k, Value: v})
		}
		// Add DSPM-specific tags
		tags = append(tags, QualysTag{Key: "dspm:sensitivity", Value: asset.SensitivityLevel})
		tags = append(tags, QualysTag{Key: "dspm:encryption", Value: asset.EncryptionStatus})

		qualysAssets = append(qualysAssets, QualysAssetPayload{
			ExternalID: asset.ARN,
			Name:       asset.Name,
			Type:       asset.ResourceType,
			Provider:   "AWS",
			Region:     asset.Region,
			Tags:       tags,
			Attributes: map[string]interface{}{
				"publicAccess":        asset.PublicAccess,
				"sensitivityLevel":    asset.SensitivityLevel,
				"encryptionStatus":    asset.EncryptionStatus,
				"classificationCount": len(asset.Classifications),
			},
			LastSeen: asset.LastScannedAt.Format(time.RFC3339),
		})
	}

	severityMap := map[string]int{
		"CRITICAL": 5,
		"HIGH":     4,
		"MEDIUM":   3,
		"LOW":      2,
		"INFO":     1,
	}

	for _, finding := range findings {
		evidence, _ := json.Marshal(finding.Evidence)
		qualysFindings = append(qualysFindings, QualysFindingPayload{
			ExternalID:    finding.ID,
			AssetID:       finding.AssetARN,
			Title:         finding.Title,
			Description:   finding.Description,
			Severity:      severityMap[finding.Severity],
			Category:      finding.Category,
			Status:        finding.Status,
			Remediation:   finding.Remediation,
			ComplianceIDs: finding.ComplianceRefs,
			DetectedDate:  finding.DetectedAt.Format(time.RFC3339),
			Evidence:      string(evidence),
		})
	}

	return &QualysBulkImport{
		Source:    "qualys-dspm",
		Timestamp: time.Now().Format(time.RFC3339),
		Assets:    qualysAssets,
		Findings:  qualysFindings,
	}
}

func printSection(title string) {
	fmt.Printf("\n%s\n", "=======================================================================")
	fmt.Printf("  %s\n", title)
	fmt.Printf("%s\n\n", "=======================================================================")
}

func main() {
	qualysPayload, assets, findings, lineage := generateSampleData()

	// Print Data Collection Summary
	printSection("DSPM DATA COLLECTION SUMMARY")
	fmt.Printf("Scan completed at: %s\n\n", time.Now().Format(time.RFC3339))

	fmt.Println("DISCOVERED ASSETS:")
	fmt.Println("-----------------")
	for _, asset := range assets {
		fmt.Printf("  [%s] %s\n", asset.SensitivityLevel, asset.ARN)
		fmt.Printf("       Type: %s | Encryption: %s | Public: %v\n",
			asset.ResourceType, asset.EncryptionStatus, asset.PublicAccess)
		for _, c := range asset.Classifications {
			fmt.Printf("       - %s/%s: %d matches (%.0f%% confidence)\n",
				c.Category, c.SubCategory, c.MatchCount, c.Confidence*100)
		}
		fmt.Println()
	}

	printSection("SECURITY FINDINGS")
	for _, f := range findings {
		fmt.Printf("[%s] %s\n", f.Severity, f.Title)
		fmt.Printf("  Asset: %s\n", f.AssetARN)
		fmt.Printf("  Category: %s | Status: %s\n", f.Category, f.Status)
		fmt.Printf("  Compliance: %v\n", f.ComplianceRefs)
		fmt.Println()
	}

	printSection("DATA LINEAGE (Sensitive Data Flow)")
	for _, l := range lineage {
		piiWarning := ""
		if l.ContainsPII {
			piiWarning = " [CONTAINS PII]"
		}
		fmt.Printf("  %s\n", l.SourceARN)
		fmt.Printf("    |-- %s -->\n", l.FlowType)
		fmt.Printf("  %s%s\n", l.TargetARN, piiWarning)
		fmt.Printf("  (Confidence: %.0f%%, Inferred from: %s)\n\n", l.Confidence*100, l.InferredFrom)
	}

	printSection("QUALYS INTEGRATION PAYLOAD (JSON)")
	fmt.Println("This payload can be sent to Qualys via REST API or webhook:")
	fmt.Println()

	qualysJSON, _ := json.MarshalIndent(qualysPayload, "", "  ")
	fmt.Println(string(qualysJSON))

	printSection("QUALYS INTEGRATION OPTIONS")
	fmt.Println(`
1. REST API PUSH (Recommended)
   POST https://qualysapi.qualys.com/api/v2/assets/import
   Authorization: Bearer <token>
   Content-Type: application/json
   Body: <QualysBulkImport payload>

2. WEBHOOK INTEGRATION
   Configure Qualys webhook endpoint in DSPM:
   POST /api/v1/notifications/settings
   {
     "type": "webhook",
     "url": "https://qualysapi.qualys.com/webhook/dspm",
     "events": ["finding.created", "asset.classified"]
   }

3. SCHEDULED SYNC
   Configure periodic export job:
   POST /api/v1/jobs
   {
     "type": "qualys_sync",
     "schedule": "0 */6 * * *",  // Every 6 hours
     "config": {
       "qualys_api_url": "https://qualysapi.qualys.com",
       "include_findings": true,
       "include_assets": true,
       "severity_threshold": "HIGH"
     }
   }

4. SIEM INTEGRATION (Splunk, etc.)
   Stream events via syslog or HTTP Event Collector:
   - Real-time finding alerts
   - Classification events
   - Scan completion notifications`)

	// Save sample outputs
	os.MkdirAll("output", 0755)

	// Save Qualys payload
	qualysFile, _ := os.Create("output/qualys_import.json")
	qualysFile.WriteString(string(qualysJSON))
	qualysFile.Close()

	// Save findings CSV
	csvFile, _ := os.Create("output/findings_report.csv")
	csvFile.WriteString("ID,Severity,Title,Asset,Category,Status,Compliance,Detected\n")
	for _, f := range findings {
		csvFile.WriteString(fmt.Sprintf("%s,%s,\"%s\",%s,%s,%s,\"%v\",%s\n",
			f.ID, f.Severity, f.Title, f.AssetARN, f.Category, f.Status,
			f.ComplianceRefs, f.DetectedAt.Format(time.RFC3339)))
	}
	csvFile.Close()

	fmt.Println("\nOutput files generated:")
	fmt.Println("  - output/qualys_import.json  (Qualys API payload)")
	fmt.Println("  - output/findings_report.csv (CSV report)")
	fmt.Println("\nOpen examples/sample_dashboard.html in a browser to see the visual dashboard.")
}
