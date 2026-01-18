package lineage

import (
	"context"
	"testing"

	"github.com/qualys/dspm/internal/models"
)

func TestInferenceEngine_InferFromLambdaConfig(t *testing.T) {
	engine := NewInferenceEngine()
	ctx := context.Background()

	tests := []struct {
		name           string
		config         *FunctionConfig
		expectedFlows  int
		expectedTypes  []models.FlowType
		expectedSources []string
	}{
		{
			name: "s3 bucket in env var",
			config: &FunctionConfig{
				FunctionARN:  "arn:aws:lambda:us-east-1:123456789:function:my-function",
				FunctionName: "my-function",
				Environment: map[string]string{
					"S3_BUCKET": "my-data-bucket",
				},
			},
			expectedFlows: 1,
			expectedTypes: []models.FlowType{models.FlowReadsFrom},
		},
		{
			name: "dynamodb table in env var",
			config: &FunctionConfig{
				FunctionARN:  "arn:aws:lambda:us-east-1:123456789:function:my-function",
				FunctionName: "my-function",
				Environment: map[string]string{
					"DYNAMODB_TABLE": "my-table",
				},
			},
			expectedFlows: 1,
			expectedTypes: []models.FlowType{models.FlowReadsFrom},
		},
		{
			name: "multiple resources in env vars",
			config: &FunctionConfig{
				FunctionARN:  "arn:aws:lambda:us-east-1:123456789:function:processor",
				FunctionName: "processor",
				Environment: map[string]string{
					"INPUT_BUCKET":  "input-bucket",
					"OUTPUT_BUCKET": "output-bucket",
					"TABLE_NAME":    "results-table",
				},
			},
			// 4 flows: input reads, output reads (bucket ref), output writes (output prefix), table reads
			expectedFlows: 4,
		},
		{
			name: "sqs event source",
			config: &FunctionConfig{
				FunctionARN:  "arn:aws:lambda:us-east-1:123456789:function:worker",
				FunctionName: "worker",
				EventSources: []EventSource{
					{
						EventSourceARN: "arn:aws:sqs:us-east-1:123456789:my-queue",
						Type:           "SQS",
						State:          "Enabled",
						BatchSize:      10,
					},
				},
			},
			expectedFlows: 1,
			expectedTypes: []models.FlowType{models.FlowReadsFrom},
		},
		{
			name: "kinesis event source",
			config: &FunctionConfig{
				FunctionARN:  "arn:aws:lambda:us-east-1:123456789:function:stream-processor",
				FunctionName: "stream-processor",
				EventSources: []EventSource{
					{
						EventSourceARN: "arn:aws:kinesis:us-east-1:123456789:stream/my-stream",
						Type:           "Kinesis",
						State:          "Enabled",
						BatchSize:      100,
					},
				},
			},
			expectedFlows: 1,
		},
		{
			name: "disabled event source",
			config: &FunctionConfig{
				FunctionARN:  "arn:aws:lambda:us-east-1:123456789:function:worker",
				FunctionName: "worker",
				EventSources: []EventSource{
					{
						EventSourceARN: "arn:aws:sqs:us-east-1:123456789:my-queue",
						Type:           "SQS",
						State:          "Disabled",
						BatchSize:      10,
					},
				},
			},
			expectedFlows: 0,
		},
		{
			name: "no resources",
			config: &FunctionConfig{
				FunctionARN:  "arn:aws:lambda:us-east-1:123456789:function:simple",
				FunctionName: "simple",
				Environment:  map[string]string{},
				EventSources: []EventSource{},
			},
			expectedFlows: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flows, err := engine.InferFromLambdaConfig(ctx, tt.config)
			if err != nil {
				t.Fatalf("InferFromLambdaConfig() error = %v", err)
			}

			if len(flows) != tt.expectedFlows {
				t.Errorf("Got %d flows, want %d", len(flows), tt.expectedFlows)
				for _, flow := range flows {
					t.Logf("  Flow: %s -> %s (%s)", flow.SourceARN, flow.TargetARN, flow.FlowType)
				}
			}

			if len(tt.expectedTypes) > 0 {
				for _, expectedType := range tt.expectedTypes {
					found := false
					for _, flow := range flows {
						if flow.FlowType == expectedType {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected flow type %s not found", expectedType)
					}
				}
			}
		})
	}
}

func TestInferenceEngine_InferFromIAMPolicy(t *testing.T) {
	engine := NewInferenceEngine()

	tests := []struct {
		name          string
		principalARN  string
		principalName string
		principalType string
		policy        map[string]interface{}
		expectedFlows int
		expectedTypes []models.FlowType
	}{
		{
			name:          "s3 read policy",
			principalARN:  "arn:aws:iam::123456789:role/reader-role",
			principalName: "reader-role",
			principalType: "iam_role",
			policy: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":   "Allow",
						"Action":   []interface{}{"s3:GetObject", "s3:ListBucket"},
						"Resource": []interface{}{"arn:aws:s3:::my-bucket/*"},
					},
				},
			},
			expectedFlows: 1,
			expectedTypes: []models.FlowType{models.FlowReadsFrom},
		},
		{
			name:          "s3 write policy",
			principalARN:  "arn:aws:iam::123456789:role/writer-role",
			principalName: "writer-role",
			principalType: "iam_role",
			policy: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":   "Allow",
						"Action":   "s3:PutObject",
						"Resource": "arn:aws:s3:::output-bucket/*",
					},
				},
			},
			expectedFlows: 1,
			expectedTypes: []models.FlowType{models.FlowWritesTo},
		},
		{
			name:          "dynamodb full access",
			principalARN:  "arn:aws:iam::123456789:role/dynamo-role",
			principalName: "dynamo-role",
			principalType: "iam_role",
			policy: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":   "Allow",
						"Action":   []interface{}{"dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem"},
						"Resource": []interface{}{"arn:aws:dynamodb:us-east-1:123456789:table/my-table"},
					},
				},
			},
			expectedFlows: 1,
			expectedTypes: []models.FlowType{models.FlowWritesTo}, // Write takes precedence
		},
		{
			name:          "deny statement ignored",
			principalARN:  "arn:aws:iam::123456789:role/restricted-role",
			principalName: "restricted-role",
			principalType: "iam_role",
			policy: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":   "Deny",
						"Action":   []interface{}{"s3:*"},
						"Resource": []interface{}{"arn:aws:s3:::sensitive-bucket/*"},
					},
				},
			},
			expectedFlows: 0,
		},
		{
			name:          "wildcard resource ignored",
			principalARN:  "arn:aws:iam::123456789:role/admin-role",
			principalName: "admin-role",
			principalType: "iam_role",
			policy: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":   "Allow",
						"Action":   []interface{}{"s3:*"},
						"Resource": "*",
					},
				},
			},
			expectedFlows: 0,
		},
		{
			name:          "multiple statements",
			principalARN:  "arn:aws:iam::123456789:role/etl-role",
			principalName: "etl-role",
			principalType: "iam_role",
			policy: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":   "Allow",
						"Action":   []interface{}{"s3:GetObject"},
						"Resource": []interface{}{"arn:aws:s3:::source-bucket/*"},
					},
					map[string]interface{}{
						"Effect":   "Allow",
						"Action":   []interface{}{"s3:PutObject"},
						"Resource": []interface{}{"arn:aws:s3:::dest-bucket/*"},
					},
				},
			},
			expectedFlows: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flows, err := engine.InferFromIAMPolicy(tt.principalARN, tt.principalName, tt.principalType, tt.policy)
			if err != nil {
				t.Fatalf("InferFromIAMPolicy() error = %v", err)
			}

			if len(flows) != tt.expectedFlows {
				t.Errorf("Got %d flows, want %d", len(flows), tt.expectedFlows)
				for _, flow := range flows {
					t.Logf("  Flow: %s -> %s (%s)", flow.SourceARN, flow.TargetARN, flow.FlowType)
				}
			}

			if len(tt.expectedTypes) > 0 {
				for _, expectedType := range tt.expectedTypes {
					found := false
					for _, flow := range flows {
						if flow.FlowType == expectedType {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected flow type %s not found", expectedType)
					}
				}
			}
		})
	}
}

func TestInferenceEngine_DetermineFlowTypeFromActions(t *testing.T) {
	engine := NewInferenceEngine()

	tests := []struct {
		name     string
		actions  []string
		expected models.FlowType
	}{
		{
			name:     "read actions",
			actions:  []string{"s3:GetObject", "s3:ListBucket"},
			expected: models.FlowReadsFrom,
		},
		{
			name:     "write actions",
			actions:  []string{"s3:PutObject", "s3:DeleteObject"},
			expected: models.FlowWritesTo,
		},
		{
			name:     "mixed - write takes precedence",
			actions:  []string{"s3:GetObject", "s3:PutObject"},
			expected: models.FlowWritesTo,
		},
		{
			name:     "describe action",
			actions:  []string{"dynamodb:DescribeTable"},
			expected: models.FlowReadsFrom,
		},
		{
			name:     "update action",
			actions:  []string{"dynamodb:UpdateItem"},
			expected: models.FlowWritesTo,
		},
		{
			name:     "create action",
			actions:  []string{"s3:CreateBucket"},
			expected: models.FlowWritesTo,
		},
		{
			name:     "no recognized actions",
			actions:  []string{"iam:PassRole"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.determineFlowTypeFromActions(tt.actions)
			if result != tt.expected {
				t.Errorf("determineFlowTypeFromActions(%v) = %v, want %v", tt.actions, result, tt.expected)
			}
		})
	}
}

func TestInferenceEngine_GetResourceTypeFromARN(t *testing.T) {
	engine := NewInferenceEngine()

	tests := []struct {
		arn      string
		expected string
	}{
		{"arn:aws:s3:::my-bucket", "s3_bucket"},
		{"arn:aws:s3:::my-bucket/path/to/object", "s3_bucket"},
		{"arn:aws:dynamodb:us-east-1:123456789:table/my-table", "dynamodb_table"},
		{"arn:aws:sqs:us-east-1:123456789:my-queue", "sqs_queue"},
		{"arn:aws:sns:us-east-1:123456789:my-topic", "sns_topic"},
		{"arn:aws:kinesis:us-east-1:123456789:stream/my-stream", "kinesis_stream"},
		{"arn:aws:rds:us-east-1:123456789:db:my-database", "rds_instance"},
		{"arn:aws:lambda:us-east-1:123456789:function:my-function", "lambda_function"},
		{"not-an-arn", "unknown"},
		{"arn:aws:unknown:us-east-1:123456789:resource", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.arn, func(t *testing.T) {
			result := engine.getResourceTypeFromARN(tt.arn)
			if result != tt.expected {
				t.Errorf("getResourceTypeFromARN(%q) = %q, want %q", tt.arn, result, tt.expected)
			}
		})
	}
}

func TestInferenceEngine_ExtractResourceName(t *testing.T) {
	engine := NewInferenceEngine()

	tests := []struct {
		arn      string
		expected string
	}{
		{"arn:aws:s3:::my-bucket", "my-bucket"},
		{"arn:aws:dynamodb:us-east-1:123456789:table/my-table", "my-table"},
		{"arn:aws:sqs:us-east-1:123456789:my-queue", "my-queue"},
		{"arn:aws:kinesis:us-east-1:123456789:stream/data-stream", "data-stream"},
		{"not-an-arn", "not-an-arn"},
		{"simple-name", "simple-name"},
	}

	for _, tt := range tests {
		t.Run(tt.arn, func(t *testing.T) {
			result := engine.extractResourceName(tt.arn)
			if result != tt.expected {
				t.Errorf("extractResourceName(%q) = %q, want %q", tt.arn, result, tt.expected)
			}
		})
	}
}

func TestInferenceEngine_ConstructARN(t *testing.T) {
	engine := NewInferenceEngine()

	tests := []struct {
		resourceType string
		value        string
		expected     string
	}{
		{"s3_bucket", "my-bucket", "arn:aws:s3:::my-bucket"},
		{"dynamodb_table", "my-table", "arn:aws:dynamodb:*:*:table/my-table"},
		{"sqs_queue", "arn:aws:sqs:us-east-1:123:queue", "arn:aws:sqs:us-east-1:123:queue"},
		{"sqs_queue", "https://sqs.us-east-1.amazonaws.com/123/queue", "https://sqs.us-east-1.amazonaws.com/123/queue"},
		{"unknown_type", "value", ""},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType+"/"+tt.value, func(t *testing.T) {
			result := engine.constructARN(tt.resourceType, tt.value)
			if result != tt.expected {
				t.Errorf("constructARN(%q, %q) = %q, want %q", tt.resourceType, tt.value, result, tt.expected)
			}
		})
	}
}

func TestInferenceEngine_ExtractStringOrSlice(t *testing.T) {
	engine := NewInferenceEngine()

	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "single string",
			input:    "value",
			expected: []string{"value"},
		},
		{
			name:     "slice of strings",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "slice of interface{}",
			input:    []interface{}{"x", "y", "z"},
			expected: []string{"x", "y", "z"},
		},
		{
			name:     "nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "mixed interface slice",
			input:    []interface{}{"valid", 123, "another"},
			expected: []string{"valid", "another"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.extractStringOrSlice(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("extractStringOrSlice() got %d elements, want %d", len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("extractStringOrSlice()[%d] = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestInferenceEngine_ConfidenceScores(t *testing.T) {
	engine := NewInferenceEngine()
	ctx := context.Background()

	// Event source should have high confidence
	eventSourceConfig := &FunctionConfig{
		FunctionARN:  "arn:aws:lambda:us-east-1:123456789:function:worker",
		FunctionName: "worker",
		EventSources: []EventSource{
			{
				EventSourceARN: "arn:aws:sqs:us-east-1:123456789:my-queue",
				Type:           "SQS",
				State:          "Enabled",
			},
		},
	}

	flows, _ := engine.InferFromLambdaConfig(ctx, eventSourceConfig)
	if len(flows) > 0 && flows[0].ConfidenceScore < 0.9 {
		t.Errorf("Event source flow confidence = %v, want >= 0.9", flows[0].ConfidenceScore)
	}

	// Env var should have medium confidence
	envVarConfig := &FunctionConfig{
		FunctionARN:  "arn:aws:lambda:us-east-1:123456789:function:processor",
		FunctionName: "processor",
		Environment: map[string]string{
			"S3_BUCKET": "my-bucket",
		},
	}

	flows, _ = engine.InferFromLambdaConfig(ctx, envVarConfig)
	if len(flows) > 0 {
		if flows[0].ConfidenceScore < 0.7 || flows[0].ConfidenceScore > 0.85 {
			t.Errorf("Env var flow confidence = %v, want between 0.7 and 0.85", flows[0].ConfidenceScore)
		}
	}

	// IAM policy should have lower confidence
	policy := map[string]interface{}{
		"Statement": []interface{}{
			map[string]interface{}{
				"Effect":   "Allow",
				"Action":   "s3:GetObject",
				"Resource": "arn:aws:s3:::bucket/*",
			},
		},
	}

	flows, _ = engine.InferFromIAMPolicy("arn:aws:iam::123:role/role", "role", "iam_role", policy)
	if len(flows) > 0 && flows[0].ConfidenceScore > 0.7 {
		t.Errorf("IAM policy flow confidence = %v, want <= 0.7", flows[0].ConfidenceScore)
	}
}

func TestParsePolicyDocument(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name: "valid policy",
			json: `{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Action": "s3:GetObject",
						"Resource": "arn:aws:s3:::bucket/*"
					}
				]
			}`,
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantErr: true,
		},
		{
			name:    "empty json",
			json:    `{}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePolicyDocument(tt.json)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePolicyDocument() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Benchmark tests
func BenchmarkInferenceEngine_EnvVars(b *testing.B) {
	engine := NewInferenceEngine()
	ctx := context.Background()

	config := &FunctionConfig{
		FunctionARN:  "arn:aws:lambda:us-east-1:123456789:function:processor",
		FunctionName: "processor",
		Environment: map[string]string{
			"S3_BUCKET":       "bucket1",
			"DYNAMODB_TABLE":  "table1",
			"SQS_QUEUE_URL":   "https://sqs.us-east-1.amazonaws.com/123/queue",
			"OUTPUT_BUCKET":   "bucket2",
			"CACHE_TABLE":     "cache-table",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.InferFromLambdaConfig(ctx, config)
	}
}

func BenchmarkInferenceEngine_IAMPolicy(b *testing.B) {
	engine := NewInferenceEngine()

	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []interface{}{
			map[string]interface{}{
				"Effect":   "Allow",
				"Action":   []interface{}{"s3:GetObject", "s3:ListBucket"},
				"Resource": []interface{}{"arn:aws:s3:::bucket1/*", "arn:aws:s3:::bucket2/*"},
			},
			map[string]interface{}{
				"Effect":   "Allow",
				"Action":   []interface{}{"dynamodb:GetItem", "dynamodb:PutItem"},
				"Resource": []interface{}{"arn:aws:dynamodb:us-east-1:123:table/table1"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.InferFromIAMPolicy("arn:aws:iam::123:role/role", "role", "iam_role", policy)
	}
}
