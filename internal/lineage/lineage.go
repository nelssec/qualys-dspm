package lineage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/qualys/dspm/internal/models"
)

// Service provides data lineage tracking functionality
type Service struct {
	store    Store
	inference *InferenceEngine
}

// Store defines the interface for lineage data persistence
type Store interface {
	// Lineage Events
	CreateLineageEvent(ctx context.Context, event *models.LineageEvent) error
	UpdateLineageEvent(ctx context.Context, event *models.LineageEvent) error
	GetLineageEvent(ctx context.Context, id uuid.UUID) (*models.LineageEvent, error)
	ListLineageEvents(ctx context.Context, accountID uuid.UUID) ([]*models.LineageEvent, error)
	GetLineageEventsBySource(ctx context.Context, sourceARN string) ([]*models.LineageEvent, error)
	GetLineageEventsByTarget(ctx context.Context, targetARN string) ([]*models.LineageEvent, error)
	DeleteLineageEvent(ctx context.Context, id uuid.UUID) error

	// Lineage Paths
	CreateLineagePath(ctx context.Context, path *models.LineagePath) error
	ListLineagePaths(ctx context.Context, accountID uuid.UUID) ([]*models.LineagePath, error)
	GetLineagePathsByOrigin(ctx context.Context, originARN string) ([]*models.LineagePath, error)
	GetLineagePathsByDestination(ctx context.Context, destARN string) ([]*models.LineagePath, error)
	GetSensitiveDataPaths(ctx context.Context, accountID uuid.UUID) ([]*models.LineagePath, error)
	DeleteLineagePaths(ctx context.Context, accountID uuid.UUID) error

	// Assets (for lookups and sensitivity info)
	GetDataAssetByARN(ctx context.Context, arn string) (*models.DataAsset, error)
	ListDataAssets(ctx context.Context, accountID uuid.UUID) ([]*models.DataAsset, error)
}

// NewService creates a new lineage service
func NewService(store Store) *Service {
	return &Service{
		store:    store,
		inference: NewInferenceEngine(),
	}
}

// GetLineageOverview returns an overview of data lineage for an account
func (s *Service) GetLineageOverview(ctx context.Context, accountID uuid.UUID) (*LineageOverview, error) {
	events, err := s.store.ListLineageEvents(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing lineage events: %w", err)
	}

	paths, err := s.store.ListLineagePaths(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing lineage paths: %w", err)
	}

	overview := &LineageOverview{
		AccountID:           accountID,
		TotalFlows:          len(events),
		FlowsByType:         make(map[string]int),
		TopDataSources:      []ResourceSummary{},
		TopDataDestinations: []ResourceSummary{},
		LastUpdated:         time.Now(),
	}

	// Count flows by type
	sourceCounts := make(map[string]*ResourceSummary)
	destCounts := make(map[string]*ResourceSummary)

	for _, event := range events {
		overview.FlowsByType[string(event.FlowType)]++

		// Track sources
		if _, ok := sourceCounts[event.SourceResourceARN]; !ok {
			sourceCounts[event.SourceResourceARN] = &ResourceSummary{
				ARN:  event.SourceResourceARN,
				Name: event.SourceResourceName,
				Type: event.SourceResourceType,
			}
		}
		sourceCounts[event.SourceResourceARN].FlowCount++

		// Track destinations
		if _, ok := destCounts[event.TargetResourceARN]; !ok {
			destCounts[event.TargetResourceARN] = &ResourceSummary{
				ARN:  event.TargetResourceARN,
				Name: event.TargetResourceName,
				Type: event.TargetResourceType,
			}
		}
		destCounts[event.TargetResourceARN].FlowCount++
	}

	// Count sensitive data flows
	for _, path := range paths {
		if path.ContainsSensitiveData {
			overview.SensitiveDataFlows++
		}
	}

	// Get top sources and destinations (simplified - would need sorting)
	for _, summary := range sourceCounts {
		overview.TopDataSources = append(overview.TopDataSources, *summary)
		if len(overview.TopDataSources) >= 10 {
			break
		}
	}

	for _, summary := range destCounts {
		overview.TopDataDestinations = append(overview.TopDataDestinations, *summary)
		if len(overview.TopDataDestinations) >= 10 {
			break
		}
	}

	return overview, nil
}

// GetAssetLineage returns the lineage graph for a specific asset
func (s *Service) GetAssetLineage(ctx context.Context, assetARN string, maxHops int) (*LineageGraph, error) {
	if maxHops <= 0 {
		maxHops = 3
	}

	graph := &LineageGraph{
		Nodes: []LineageNode{},
		Edges: []LineageEdge{},
	}

	// Track visited nodes to avoid cycles
	visited := make(map[string]bool)
	nodeMap := make(map[string]bool)

	// Get both upstream and downstream lineage
	if err := s.traverseLineage(ctx, assetARN, maxHops, true, visited, graph, nodeMap); err != nil {
		return nil, fmt.Errorf("traversing upstream lineage: %w", err)
	}

	visited = make(map[string]bool) // Reset for downstream
	if err := s.traverseLineage(ctx, assetARN, maxHops, false, visited, graph, nodeMap); err != nil {
		return nil, fmt.Errorf("traversing downstream lineage: %w", err)
	}

	return graph, nil
}

// traverseLineage recursively traverses the lineage graph
func (s *Service) traverseLineage(ctx context.Context, arn string, hopsLeft int, upstream bool, visited map[string]bool, graph *LineageGraph, nodeMap map[string]bool) error {
	if hopsLeft <= 0 || visited[arn] {
		return nil
	}
	visited[arn] = true

	// Add node if not already present
	if !nodeMap[arn] {
		asset, _ := s.store.GetDataAssetByARN(ctx, arn)
		node := LineageNode{
			ID:   arn,
			ARN:  arn,
			Name: arn, // Default to ARN
			Type: "unknown",
		}
		if asset != nil {
			node.Name = asset.Name
			node.Type = string(asset.ResourceType)
			node.SensitivityLevel = asset.SensitivityLevel
			node.DataCategories = asset.DataCategories
		}
		graph.Nodes = append(graph.Nodes, node)
		nodeMap[arn] = true
	}

	// Get connected events
	var events []*models.LineageEvent
	var err error
	if upstream {
		events, err = s.store.GetLineageEventsByTarget(ctx, arn)
	} else {
		events, err = s.store.GetLineageEventsBySource(ctx, arn)
	}
	if err != nil {
		return err
	}

	for _, event := range events {
		// Add edge
		edge := LineageEdge{
			ID:              event.ID.String(),
			Source:          event.SourceResourceARN,
			Target:          event.TargetResourceARN,
			FlowType:        event.FlowType,
			InferredFrom:    string(event.InferredFrom),
			ConfidenceScore: event.ConfidenceScore,
			AccessMethod:    event.AccessMethod,
		}
		graph.Edges = append(graph.Edges, edge)

		// Continue traversal
		nextARN := event.SourceResourceARN
		if !upstream {
			nextARN = event.TargetResourceARN
		}
		if err := s.traverseLineage(ctx, nextARN, hopsLeft-1, upstream, visited, graph, nodeMap); err != nil {
			return err
		}
	}

	return nil
}

// FindDataFlowPaths finds all paths between data sources matching criteria
func (s *Service) FindDataFlowPaths(ctx context.Context, accountID uuid.UUID, req *LineagePathRequest) ([]*models.LineagePath, error) {
	if req.SensitiveOnly {
		return s.store.GetSensitiveDataPaths(ctx, accountID)
	}

	if req.SourceARN != "" {
		return s.store.GetLineagePathsByOrigin(ctx, req.SourceARN)
	}

	if req.DestinationARN != "" {
		return s.store.GetLineagePathsByDestination(ctx, req.DestinationARN)
	}

	return s.store.ListLineagePaths(ctx, accountID)
}

// GetSensitiveDataFlows returns all data flows involving sensitive data
func (s *Service) GetSensitiveDataFlows(ctx context.Context, accountID uuid.UUID) ([]*SensitiveDataFlow, error) {
	events, err := s.store.ListLineageEvents(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}

	var sensitiveFlows []*SensitiveDataFlow

	for _, event := range events {
		// Check if source or target has sensitive data
		sourceAsset, _ := s.store.GetDataAssetByARN(ctx, event.SourceResourceARN)
		targetAsset, _ := s.store.GetDataAssetByARN(ctx, event.TargetResourceARN)

		var sensitivity models.Sensitivity
		var categories []string

		if sourceAsset != nil && sourceAsset.SensitivityLevel != models.SensitivityUnknown {
			sensitivity = sourceAsset.SensitivityLevel
			categories = sourceAsset.DataCategories
		} else if targetAsset != nil && targetAsset.SensitivityLevel != models.SensitivityUnknown {
			sensitivity = targetAsset.SensitivityLevel
			categories = targetAsset.DataCategories
		} else {
			continue // Not a sensitive flow
		}

		// Calculate risk score
		riskScore, riskFactors := s.calculateFlowRisk(event, sourceAsset, targetAsset)

		sensitiveFlows = append(sensitiveFlows, &SensitiveDataFlow{
			Flow:             event,
			SensitivityLevel: sensitivity,
			DataCategories:   categories,
			RiskScore:        riskScore,
			RiskFactors:      riskFactors,
		})
	}

	return sensitiveFlows, nil
}

// calculateFlowRisk calculates the risk score for a data flow
func (s *Service) calculateFlowRisk(event *models.LineageEvent, source, target *models.DataAsset) (int, []string) {
	score := 0
	var factors []string

	// Base score from sensitivity
	if source != nil {
		switch source.SensitivityLevel {
		case models.SensitivityCritical:
			score += 40
			factors = append(factors, "Source contains critical data")
		case models.SensitivityHigh:
			score += 30
			factors = append(factors, "Source contains high sensitivity data")
		case models.SensitivityMedium:
			score += 20
		}
	}

	// Low confidence inference
	if event.ConfidenceScore < 0.7 {
		score += 10
		factors = append(factors, "Low confidence lineage inference")
	}

	// Cross-service flow
	if source != nil && target != nil && source.ResourceType != target.ResourceType {
		score += 10
		factors = append(factors, "Cross-service data flow")
	}

	// Export flows are higher risk
	if event.FlowType == models.FlowExportsTo || event.FlowType == models.FlowReplicatesTo {
		score += 15
		factors = append(factors, "Data export/replication flow")
	}

	return min(score, 100), factors
}

// RecordLineageEvent records a new data lineage event
func (s *Service) RecordLineageEvent(ctx context.Context, accountID uuid.UUID, flow *InferredFlow) error {
	event := &models.LineageEvent{
		ID:                 uuid.New(),
		AccountID:          accountID,
		SourceResourceARN:  flow.SourceARN,
		SourceResourceType: flow.SourceType,
		SourceResourceName: flow.SourceName,
		TargetResourceARN:  flow.TargetARN,
		TargetResourceType: flow.TargetType,
		TargetResourceName: flow.TargetName,
		FlowType:           flow.FlowType,
		InferredFrom:       flow.InferredFrom,
		ConfidenceScore:    flow.ConfidenceScore,
		Evidence:           models.JSONB(flow.Evidence),
		FirstObservedAt:    time.Now(),
		LastObservedAt:     time.Now(),
		CreatedAt:          time.Now(),
	}

	err := s.store.CreateLineageEvent(ctx, event)
	if err != nil {
		// Update if already exists
		return s.store.UpdateLineageEvent(ctx, event)
	}
	return nil
}

// InferLineageFromFunction infers lineage from a Lambda function configuration
func (s *Service) InferLineageFromFunction(ctx context.Context, accountID uuid.UUID, fn *FunctionConfig) error {
	flows, err := s.inference.InferFromLambdaConfig(ctx, fn)
	if err != nil {
		return fmt.Errorf("inferring lineage: %w", err)
	}

	for _, flow := range flows {
		if err := s.RecordLineageEvent(ctx, accountID, &flow); err != nil {
			// Log but continue with other flows
			continue
		}
	}

	return nil
}

// InferLineageFromPolicy infers lineage from an IAM policy
func (s *Service) InferLineageFromPolicy(ctx context.Context, accountID uuid.UUID, principalARN, principalName, principalType string, policyDoc map[string]interface{}) error {
	flows, err := s.inference.InferFromIAMPolicy(principalARN, principalName, principalType, policyDoc)
	if err != nil {
		return fmt.Errorf("inferring lineage from policy: %w", err)
	}

	for _, flow := range flows {
		if err := s.RecordLineageEvent(ctx, accountID, &flow); err != nil {
			continue
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
