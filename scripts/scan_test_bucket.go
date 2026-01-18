package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/qualys/dspm/internal/classifier"
)

type ScanResult struct {
	ObjectKey       string             `json:"object_key"`
	Size            int64              `json:"size"`
	ContentType     string             `json:"content_type"`
	Matches         []classifier.Match `json:"matches"`
	TotalFindings   int                `json:"total_findings"`
	MaxSensitivity  string             `json:"max_sensitivity"`
	Category        string             `json:"category"`
	ScanDuration    string             `json:"scan_duration"`
}

type ScanSummary struct {
	TotalObjects     int            `json:"total_objects"`
	ScannedObjects   int            `json:"scanned_objects"`
	ObjectsWithData  int            `json:"objects_with_sensitive_data"`
	TotalFindings    int            `json:"total_findings"`
	ByCategory       map[string]int `json:"by_category"`
	BySensitivity    map[string]int `json:"by_sensitivity"`
	ByClassification map[string]int `json:"by_classification"`
	TotalDuration    string         `json:"total_duration"`
}

func main() {
	bucketName := os.Getenv("DSPM_TEST_BUCKET")
	if bucketName == "" {
		bucketName = "dspm-test-data-314104994032"
	}

	prefix := "test-data/"
	maxObjects := 500

	fmt.Println("=============================================================")
	fmt.Println("DSPM Test Scanner")
	fmt.Println("=============================================================")
	fmt.Printf("Bucket: s3://%s\n", bucketName)
	fmt.Printf("Prefix: %s\n", prefix)
	fmt.Printf("Max Objects: %d\n", maxObjects)
	fmt.Println("=============================================================")

	// Initialize AWS client
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		fmt.Printf("Error loading AWS config: %v\n", err)
		os.Exit(1)
	}

	s3Client := s3.NewFromConfig(cfg)

	// Initialize classifier
	clf := classifier.New()

	// List objects
	fmt.Println("\nListing objects...")
	var objects []string
	var continuationToken *string

	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            &bucketName,
			Prefix:            &prefix,
			ContinuationToken: continuationToken,
		}

		resp, err := s3Client.ListObjectsV2(ctx, input)
		if err != nil {
			fmt.Printf("Error listing objects: %v\n", err)
			os.Exit(1)
		}

		for _, obj := range resp.Contents {
			objects = append(objects, *obj.Key)
			if len(objects) >= maxObjects {
				break
			}
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated || len(objects) >= maxObjects {
			break
		}
		continuationToken = resp.NextContinuationToken
	}

	fmt.Printf("Found %d objects to scan\n\n", len(objects))

	// Scan objects
	summary := ScanSummary{
		TotalObjects:     len(objects),
		ByCategory:       make(map[string]int),
		BySensitivity:    make(map[string]int),
		ByClassification: make(map[string]int),
	}

	startTime := time.Now()
	var results []ScanResult

	for i, objKey := range objects {
		if (i+1)%50 == 0 {
			fmt.Printf("Progress: %d/%d objects scanned...\n", i+1, len(objects))
		}

		// Get object
		getResp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &bucketName,
			Key:    &objKey,
		})
		if err != nil {
			fmt.Printf("Warning: error getting object %s: %v\n", objKey, err)
			continue
		}

		// Read content (limit to 1MB)
		content, err := io.ReadAll(io.LimitReader(getResp.Body, 1024*1024))
		getResp.Body.Close()
		if err != nil {
			fmt.Printf("Warning: error reading object %s: %v\n", objKey, err)
			continue
		}

		// Skip empty files
		if len(content) == 0 {
			continue
		}

		// Classify
		scanStart := time.Now()
		classResult := clf.Classify(string(content))
		scanDuration := time.Since(scanStart)

		// Determine category from path
		category := "CLEAN"
		if strings.Contains(objKey, "/pii/") {
			category = "PII"
		} else if strings.Contains(objKey, "/phi/") {
			category = "PHI"
		} else if strings.Contains(objKey, "/pci/") {
			category = "PCI"
		} else if strings.Contains(objKey, "/secrets/") {
			category = "SECRETS"
		} else if strings.Contains(objKey, "/mixed/") {
			category = "MIXED"
		}

		// Update summary
		if classResult.TotalFindings > 0 {
			summary.ObjectsWithData++
			summary.TotalFindings += classResult.TotalFindings
			summary.ByCategory[category]++
			summary.BySensitivity[string(classResult.MaxSensitivity)]++

			for _, m := range classResult.Matches {
				summary.ByClassification[m.RuleName] += m.Count
			}
		}

		result := ScanResult{
			ObjectKey:      objKey,
			Size:           int64(len(content)),
			ContentType:    getContentType(objKey),
			Matches:        classResult.Matches,
			TotalFindings:  classResult.TotalFindings,
			MaxSensitivity: string(classResult.MaxSensitivity),
			Category:       category,
			ScanDuration:   scanDuration.String(),
		}
		results = append(results, result)
		summary.ScannedObjects++
	}

	summary.TotalDuration = time.Since(startTime).String()

	// Print results
	fmt.Println("\n=============================================================")
	fmt.Println("SCAN RESULTS")
	fmt.Println("=============================================================")

	// Summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total Objects:              %d\n", summary.TotalObjects)
	fmt.Printf("  Scanned Objects:            %d\n", summary.ScannedObjects)
	fmt.Printf("  Objects with Sensitive Data: %d\n", summary.ObjectsWithData)
	fmt.Printf("  Total Findings:             %d\n", summary.TotalFindings)
	fmt.Printf("  Scan Duration:              %s\n", summary.TotalDuration)

	// By Category
	fmt.Println("\nObjects with Findings by Category:")
	printSortedMap(summary.ByCategory)

	// By Sensitivity
	fmt.Println("\nObjects by Max Sensitivity:")
	printSortedMap(summary.BySensitivity)

	// By Classification Type
	fmt.Println("\nFindings by Classification Type:")
	printSortedMap(summary.ByClassification)

	// Sample findings
	fmt.Println("\n=============================================================")
	fmt.Println("SAMPLE FINDINGS (first 10 with classifications)")
	fmt.Println("=============================================================")

	count := 0
	for _, r := range results {
		if len(r.Matches) > 0 && count < 10 {
			fmt.Printf("\n%s\n", r.ObjectKey)
			fmt.Printf("  Size: %d bytes, Category: %s, Max Sensitivity: %s\n", r.Size, r.Category, r.MaxSensitivity)
			for _, m := range r.Matches {
				fmt.Printf("  - %s (%s): %q [count: %d, confidence: %.2f]\n",
					m.RuleName, m.Sensitivity, m.Value, m.Count, m.Confidence)
			}
			count++
		}
	}

	// Write full results to JSON
	outputFile := "/tmp/dspm-scan-results.json"
	fullResults := map[string]interface{}{
		"summary": summary,
		"results": results,
	}
	jsonData, _ := json.MarshalIndent(fullResults, "", "  ")
	os.WriteFile(outputFile, jsonData, 0644)
	fmt.Printf("\n\nFull results written to: %s\n", outputFile)
}

func getContentType(filename string) string {
	if strings.HasSuffix(filename, ".json") {
		return "application/json"
	} else if strings.HasSuffix(filename, ".csv") {
		return "text/csv"
	} else if strings.HasSuffix(filename, ".txt") {
		return "text/plain"
	} else if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		return "application/x-yaml"
	} else if strings.HasSuffix(filename, ".env") {
		return "text/plain"
	} else if strings.HasSuffix(filename, ".log") {
		return "text/plain"
	}
	return "application/octet-stream"
}

func printSortedMap(m map[string]int) {
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})
	for _, kv := range sorted {
		fmt.Printf("  %-20s: %d\n", kv.Key, kv.Value)
	}
}
