package lineage

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/qualys/dspm/internal/models"
)

// InferenceEngine infers data lineage from various sources
type InferenceEngine struct {
	envVarPatterns []EnvironmentVariablePattern
}

// NewInferenceEngine creates a new inference engine with default patterns
func NewInferenceEngine() *InferenceEngine {
	return &InferenceEngine{
		envVarPatterns: DefaultEnvVarPatterns(),
	}
}

// InferFromLambdaConfig infers data flows from Lambda function configuration
func (e *InferenceEngine) InferFromLambdaConfig(ctx context.Context, fn *FunctionConfig) ([]InferredFlow, error) {
	var flows []InferredFlow

	// Infer from environment variables
	envFlows := e.inferFromEnvVars(fn.FunctionARN, fn.FunctionName, fn.Environment)
	flows = append(flows, envFlows...)

	// Infer from event sources
	eventFlows := e.inferFromEventSources(fn.FunctionARN, fn.FunctionName, fn.EventSources)
	flows = append(flows, eventFlows...)

	// Infer from IAM role (would need policy parsing)
	// This would be called separately with the actual policy document

	return flows, nil
}

// inferFromEnvVars infers data flows from environment variables
func (e *InferenceEngine) inferFromEnvVars(functionARN, functionName string, envVars map[string]string) []InferredFlow {
	var flows []InferredFlow

	for envName, envValue := range envVars {
		for _, pattern := range e.envVarPatterns {
			nameMatch, _ := regexp.MatchString(pattern.NamePattern, envName)
			if !nameMatch {
				continue
			}

			valueMatch, _ := regexp.MatchString(pattern.ValuePattern, envValue)
			if !valueMatch {
				continue
			}

			// Construct ARN based on resource type
			targetARN := e.constructARN(pattern.ResourceType, envValue)
			if targetARN == "" {
				continue
			}

			flow := InferredFlow{
				SourceARN:       functionARN,
				SourceType:      "lambda_function",
				SourceName:      functionName,
				TargetARN:       targetARN,
				TargetType:      pattern.ResourceType,
				TargetName:      envValue,
				FlowType:        pattern.FlowType,
				InferredFrom:    models.InferEnvVariable,
				ConfidenceScore: 0.75, // Medium confidence for env var inference
				Evidence: map[string]interface{}{
					"env_var_name":  envName,
					"env_var_value": envValue,
					"pattern_used":  pattern.NamePattern,
				},
			}

			// Swap source/target if this is a READS_FROM flow
			if pattern.FlowType == models.FlowReadsFrom {
				flow.SourceARN, flow.TargetARN = flow.TargetARN, flow.SourceARN
				flow.SourceType, flow.TargetType = flow.TargetType, flow.SourceType
				flow.SourceName, flow.TargetName = flow.TargetName, flow.SourceName
			}

			flows = append(flows, flow)
		}
	}

	return flows
}

// inferFromEventSources infers data flows from event source mappings
func (e *InferenceEngine) inferFromEventSources(functionARN, functionName string, eventSources []EventSource) []InferredFlow {
	var flows []InferredFlow

	for _, es := range eventSources {
		if es.State != "Enabled" {
			continue
		}

		// Determine resource type from event source ARN
		resourceType := e.getResourceTypeFromARN(es.EventSourceARN)
		resourceName := e.extractResourceName(es.EventSourceARN)

		flow := InferredFlow{
			SourceARN:       es.EventSourceARN,
			SourceType:      resourceType,
			SourceName:      resourceName,
			TargetARN:       functionARN,
			TargetType:      "lambda_function",
			TargetName:      functionName,
			FlowType:        models.FlowReadsFrom,
			InferredFrom:    models.InferEventSource,
			ConfidenceScore: 0.95, // High confidence for event source mappings
			Evidence: map[string]interface{}{
				"event_source_arn": es.EventSourceARN,
				"batch_size":       es.BatchSize,
				"event_type":       es.Type,
			},
		}

		flows = append(flows, flow)
	}

	return flows
}

// InferFromIAMPolicy infers data flows from IAM policy statements
func (e *InferenceEngine) InferFromIAMPolicy(principalARN, principalName, principalType string, policyDoc map[string]interface{}) ([]InferredFlow, error) {
	var flows []InferredFlow

	// Parse policy document
	statementsRaw, ok := policyDoc["Statement"]
	if !ok {
		return flows, nil
	}

	statements, ok := statementsRaw.([]interface{})
	if !ok {
		return flows, nil
	}

	for _, stmtRaw := range statements {
		stmt, ok := stmtRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// Only process Allow statements
		effect, _ := stmt["Effect"].(string)
		if effect != "Allow" {
			continue
		}

		// Get actions and resources
		actions := e.extractStringOrSlice(stmt["Action"])
		resources := e.extractStringOrSlice(stmt["Resource"])

		// Analyze actions to determine flow type
		for _, resource := range resources {
			if resource == "*" {
				continue // Skip wildcard resources
			}

			flowType := e.determineFlowTypeFromActions(actions)
			if flowType == "" {
				continue
			}

			resourceType := e.getResourceTypeFromARN(resource)
			resourceName := e.extractResourceName(resource)

			flow := InferredFlow{
				SourceARN:       principalARN,
				SourceType:      principalType,
				SourceName:      principalName,
				TargetARN:       resource,
				TargetType:      resourceType,
				TargetName:      resourceName,
				FlowType:        flowType,
				InferredFrom:    models.InferIAMPolicy,
				ConfidenceScore: 0.60, // Lower confidence for policy-based inference
				Evidence: map[string]interface{}{
					"actions":   actions,
					"resources": resources,
				},
			}

			// Adjust source/target based on flow direction
			if flowType == models.FlowReadsFrom {
				flow.SourceARN, flow.TargetARN = flow.TargetARN, flow.SourceARN
				flow.SourceType, flow.TargetType = flow.TargetType, flow.SourceType
				flow.SourceName, flow.TargetName = flow.TargetName, flow.SourceName
			}

			flows = append(flows, flow)
		}
	}

	return flows, nil
}

// determineFlowTypeFromActions determines the flow type based on IAM actions
func (e *InferenceEngine) determineFlowTypeFromActions(actions []string) models.FlowType {
	hasRead := false
	hasWrite := false

	for _, action := range actions {
		actionLower := strings.ToLower(action)

		// Read actions
		if strings.Contains(actionLower, "get") ||
			strings.Contains(actionLower, "list") ||
			strings.Contains(actionLower, "describe") ||
			strings.Contains(actionLower, "read") ||
			strings.Contains(actionLower, "select") {
			hasRead = true
		}

		// Write actions
		if strings.Contains(actionLower, "put") ||
			strings.Contains(actionLower, "create") ||
			strings.Contains(actionLower, "write") ||
			strings.Contains(actionLower, "update") ||
			strings.Contains(actionLower, "delete") ||
			strings.Contains(actionLower, "insert") {
			hasWrite = true
		}
	}

	// If both read and write, prioritize write
	if hasWrite {
		return models.FlowWritesTo
	}
	if hasRead {
		return models.FlowReadsFrom
	}

	return ""
}

// constructARN constructs an ARN from resource type and name
func (e *InferenceEngine) constructARN(resourceType, value string) string {
	switch resourceType {
	case "s3_bucket":
		return fmt.Sprintf("arn:aws:s3:::%s", value)
	case "dynamodb_table":
		return fmt.Sprintf("arn:aws:dynamodb:*:*:table/%s", value)
	case "sqs_queue":
		if strings.HasPrefix(value, "arn:") {
			return value
		}
		return value // Return URL as-is for SQS
	case "sns_topic":
		return value
	default:
		return ""
	}
}

// getResourceTypeFromARN extracts the resource type from an ARN
func (e *InferenceEngine) getResourceTypeFromARN(arn string) string {
	if !strings.HasPrefix(arn, "arn:") {
		return "unknown"
	}

	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return "unknown"
	}

	service := parts[2]
	switch service {
	case "s3":
		return "s3_bucket"
	case "dynamodb":
		return "dynamodb_table"
	case "sqs":
		return "sqs_queue"
	case "sns":
		return "sns_topic"
	case "kinesis":
		return "kinesis_stream"
	case "rds":
		return "rds_instance"
	case "lambda":
		return "lambda_function"
	default:
		return service
	}
}

// extractResourceName extracts the resource name from an ARN
func (e *InferenceEngine) extractResourceName(arn string) string {
	if !strings.HasPrefix(arn, "arn:") {
		return arn
	}

	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return arn
	}

	resource := parts[5]
	// Handle resource paths like "table/MyTable"
	if idx := strings.LastIndex(resource, "/"); idx != -1 {
		return resource[idx+1:]
	}
	return resource
}

// extractStringOrSlice extracts a string or slice of strings from an interface
func (e *InferenceEngine) extractStringOrSlice(v interface{}) []string {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case string:
		return []string{val}
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return val
	}

	return nil
}

// ParsePolicyDocument parses a JSON policy document
func ParsePolicyDocument(policyJSON string) (map[string]interface{}, error) {
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(policyJSON), &doc); err != nil {
		return nil, fmt.Errorf("parsing policy document: %w", err)
	}
	return doc, nil
}
