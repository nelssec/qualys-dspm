package aitracking

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/qualys/dspm/internal/models"
)

// Service provides AI/ML tracking and risk assessment functionality
type Service struct {
	store       Store
	riskWeights RiskFactorWeights
}

// Store defines the interface for AI tracking data persistence
type Store interface {
	// AI Services
	CreateAIService(ctx context.Context, service *models.AIService) error
	UpdateAIService(ctx context.Context, service *models.AIService) error
	GetAIService(ctx context.Context, id uuid.UUID) (*models.AIService, error)
	GetAIServiceByARN(ctx context.Context, arn string) (*models.AIService, error)
	ListAIServices(ctx context.Context, accountID uuid.UUID) ([]*models.AIService, error)
	DeleteAIService(ctx context.Context, id uuid.UUID) error

	// AI Models
	CreateAIModel(ctx context.Context, model *models.AIModel) error
	UpdateAIModel(ctx context.Context, model *models.AIModel) error
	GetAIModel(ctx context.Context, id uuid.UUID) (*models.AIModel, error)
	GetAIModelByARN(ctx context.Context, arn string) (*models.AIModel, error)
	ListAIModels(ctx context.Context, accountID uuid.UUID) ([]*models.AIModel, error)
	ListAIModelsByService(ctx context.Context, serviceID uuid.UUID) ([]*models.AIModel, error)
	DeleteAIModel(ctx context.Context, id uuid.UUID) error

	// Training Data
	CreateAITrainingData(ctx context.Context, data *models.AITrainingData) error
	UpdateAITrainingData(ctx context.Context, data *models.AITrainingData) error
	GetAITrainingDataByModel(ctx context.Context, modelID uuid.UUID) ([]*models.AITrainingData, error)
	ListSensitiveTrainingData(ctx context.Context, accountID uuid.UUID) ([]*models.AITrainingData, error)

	// Processing Events
	CreateAIProcessingEvent(ctx context.Context, event *models.AIProcessingEvent) error
	GetAIProcessingEvent(ctx context.Context, id uuid.UUID) (*models.AIProcessingEvent, error)
	ListAIProcessingEvents(ctx context.Context, accountID uuid.UUID, limit int) ([]*models.AIProcessingEvent, error)
	ListAIProcessingEventsByModel(ctx context.Context, modelID uuid.UUID) ([]*models.AIProcessingEvent, error)
	ListSensitiveDataAccessEvents(ctx context.Context, accountID uuid.UUID) ([]*models.AIProcessingEvent, error)

	// Assets (for lookups)
	GetDataAsset(ctx context.Context, id uuid.UUID) (*models.DataAsset, error)
	GetDataAssetByARN(ctx context.Context, arn string) (*models.DataAsset, error)
}

// NewService creates a new AI tracking service
func NewService(store Store) *Service {
	return &Service{
		store:       store,
		riskWeights: DefaultRiskFactorWeights(),
	}
}

// NewServiceWithWeights creates an AI tracking service with custom risk weights
func NewServiceWithWeights(store Store, weights RiskFactorWeights) *Service {
	return &Service{
		store:       store,
		riskWeights: weights,
	}
}

// GetAIServiceOverview returns an overview of AI services for an account
func (s *Service) GetAIServiceOverview(ctx context.Context, accountID uuid.UUID) (*AIServiceOverview, error) {
	services, err := s.store.ListAIServices(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing AI services: %w", err)
	}

	models, err := s.store.ListAIModels(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing AI models: %w", err)
	}

	events, err := s.store.ListAIProcessingEvents(ctx, accountID, 100)
	if err != nil {
		return nil, fmt.Errorf("listing AI events: %w", err)
	}

	overview := &AIServiceOverview{
		AccountID:      accountID,
		ServicesByType: make(map[string]int),
		ModelsByType:   make(map[string]int),
		EventsByType:   make(map[string]int),
		RecentEvents:   events,
		LastUpdated:    time.Now(),
	}

	for _, svc := range services {
		overview.ServicesByType[string(svc.ServiceType)]++
	}

	for _, model := range models {
		overview.ModelsByType[string(model.ModelType)]++
	}

	for _, event := range events {
		overview.EventsByType[string(event.EventType)]++
	}

	return overview, nil
}

// GetAIRiskReport generates a comprehensive AI risk report for an account
func (s *Service) GetAIRiskReport(ctx context.Context, accountID uuid.UUID) (*AIRiskReport, error) {
	services, err := s.store.ListAIServices(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	models, err := s.store.ListAIModels(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing models: %w", err)
	}

	sensitiveData, err := s.store.ListSensitiveTrainingData(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing sensitive training data: %w", err)
	}

	sensitiveEvents, err := s.store.ListSensitiveDataAccessEvents(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing sensitive events: %w", err)
	}

	report := &AIRiskReport{
		AccountID:              accountID,
		TotalAIServices:        len(services),
		TotalAIModels:          len(models),
		SensitiveTrainingJobs:  len(sensitiveData),
		RiskByCategory:         make(map[string]int),
		TopRiskyModels:         []ModelRiskSummary{},
		Recommendations:        []string{},
		GeneratedAt:            time.Now(),
	}

	// Count models accessing sensitive data
	modelSensitiveAccess := make(map[uuid.UUID]bool)
	for _, data := range sensitiveData {
		modelSensitiveAccess[data.ModelID] = true
	}
	report.ModelsAccessingSensitive = len(modelSensitiveAccess)

	// Count high risk events and by category
	for _, event := range sensitiveEvents {
		if event.RiskScore >= 70 {
			report.HighRiskEvents++
		}
		for _, cat := range event.AccessedCategories {
			report.RiskByCategory[cat]++
		}
	}

	// Calculate risk for each model
	for _, model := range models {
		riskSummary := s.calculateModelRisk(ctx, model)
		report.TopRiskyModels = append(report.TopRiskyModels, riskSummary)
	}

	// Sort by risk score (simplified)
	// In production, would use proper sorting

	// Generate recommendations
	report.Recommendations = s.generateRecommendations(report)

	return report, nil
}

// calculateModelRisk calculates the risk score for an AI model
func (s *Service) calculateModelRisk(ctx context.Context, model *models.AIModel) ModelRiskSummary {
	summary := ModelRiskSummary{
		Model:       model,
		RiskScore:   0,
		RiskFactors: []string{},
	}

	// Get training data for this model
	trainingData, _ := s.store.GetAITrainingDataByModel(ctx, model.ID)

	for _, data := range trainingData {
		if data.ContainsSensitiveData {
			summary.SensitiveDataSources++
		}
	}

	// Calculate risk based on sensitive data access
	if summary.SensitiveDataSources > 0 {
		summary.RiskScore += int(float64(30) * s.riskWeights.SensitiveDataAccess)
		summary.RiskFactors = append(summary.RiskFactors,
			fmt.Sprintf("Model trained on %d sensitive data sources", summary.SensitiveDataSources))
	}

	// Check for critical data
	for _, data := range trainingData {
		if data.SensitivityLevel == models.SensitivityCritical {
			summary.RiskScore += int(float64(25) * s.riskWeights.CriticalDataTraining)
			summary.RiskFactors = append(summary.RiskFactors, "Model uses critical sensitivity data for training")
			break
		}
	}

	// Get recent events
	events, _ := s.store.ListAIProcessingEventsByModel(ctx, model.ID)
	if len(events) > 0 {
		lastEvent := events[0].EventTime
		summary.LastAccessEvent = &lastEvent
	}

	// Cap risk score at 100
	if summary.RiskScore > 100 {
		summary.RiskScore = 100
	}

	return summary
}

// generateRecommendations generates recommendations based on the risk report
func (s *Service) generateRecommendations(report *AIRiskReport) []string {
	var recommendations []string

	if report.ModelsAccessingSensitive > 0 {
		recommendations = append(recommendations,
			"Review AI models with access to sensitive data and ensure proper data handling procedures")
	}

	if report.SensitiveTrainingJobs > 0 {
		recommendations = append(recommendations,
			"Implement data anonymization or pseudonymization for training data containing PII")
	}

	if report.HighRiskEvents > 10 {
		recommendations = append(recommendations,
			"Investigate high-risk AI data access events and implement additional access controls")
	}

	if piiCount := report.RiskByCategory["PII"]; piiCount > 0 {
		recommendations = append(recommendations,
			"Consider implementing differential privacy for models trained on PII data")
	}

	if phiCount := report.RiskByCategory["PHI"]; phiCount > 0 {
		recommendations = append(recommendations,
			"Ensure HIPAA compliance for AI models processing protected health information")
	}

	return recommendations
}

// GetModelTrainingDataAnalysis analyzes training data for a specific model
func (s *Service) GetModelTrainingDataAnalysis(ctx context.Context, modelID uuid.UUID) (*TrainingDataAnalysis, error) {
	trainingData, err := s.store.GetAITrainingDataByModel(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("getting training data: %w", err)
	}

	analysis := &TrainingDataAnalysis{
		ModelID:            modelID,
		TotalDataSources:   len(trainingData),
		DataCategories:     []string{},
		HighestSensitivity: models.SensitivityUnknown,
		DataSources:        trainingData,
	}

	categorySet := make(map[string]bool)
	sensitivityOrder := map[models.Sensitivity]int{
		models.SensitivityUnknown:  0,
		models.SensitivityLow:      1,
		models.SensitivityMedium:   2,
		models.SensitivityHigh:     3,
		models.SensitivityCritical: 4,
	}

	for _, data := range trainingData {
		analysis.TotalDataSizeBytes += data.DataSizeBytes

		if data.ContainsSensitiveData {
			analysis.SensitiveDataSources++
		}

		for _, cat := range data.SensitivityCategories {
			categorySet[cat] = true
		}

		if sensitivityOrder[data.SensitivityLevel] > sensitivityOrder[analysis.HighestSensitivity] {
			analysis.HighestSensitivity = data.SensitivityLevel
		}
	}

	for cat := range categorySet {
		analysis.DataCategories = append(analysis.DataCategories, cat)
	}

	return analysis, nil
}

// GetSensitiveDataAccessSummary returns a summary of AI access to sensitive data
func (s *Service) GetSensitiveDataAccessSummary(ctx context.Context, accountID uuid.UUID) (*SensitiveDataAccessSummary, error) {
	events, err := s.store.ListSensitiveDataAccessEvents(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("listing sensitive events: %w", err)
	}

	summary := &SensitiveDataAccessSummary{
		TotalEvents:        len(events),
		EventsBySensitivity: make(map[string]int),
		EventsByCategory:   make(map[string]int),
		EventsByModel:      make(map[string]int),
		MostAccessedAssets: []AssetAccessCount{},
	}

	assetAccess := make(map[string]*AssetAccessCount)

	for _, event := range events {
		summary.EventsBySensitivity[string(event.AccessedSensitivityLevel)]++

		for _, cat := range event.AccessedCategories {
			summary.EventsByCategory[cat]++
		}

		if event.ModelID != nil {
			summary.EventsByModel[event.ModelID.String()]++
		}

		if event.DataSourceARN != "" {
			if _, ok := assetAccess[event.DataSourceARN]; !ok {
				assetAccess[event.DataSourceARN] = &AssetAccessCount{
					AssetARN:         event.DataSourceARN,
					SensitivityLevel: event.AccessedSensitivityLevel,
				}
				if event.DataAssetID != nil {
					assetAccess[event.DataSourceARN].AssetID = *event.DataAssetID
				}
			}
			assetAccess[event.DataSourceARN].AccessCount++
			assetAccess[event.DataSourceARN].LastAccessed = event.EventTime
		}
	}

	for _, count := range assetAccess {
		summary.MostAccessedAssets = append(summary.MostAccessedAssets, *count)
	}

	return summary, nil
}

// RecordAIService records a discovered AI service
func (s *Service) RecordAIService(ctx context.Context, service *models.AIService) error {
	service.ID = uuid.New()
	service.CreatedAt = time.Now()
	service.UpdatedAt = time.Now()

	err := s.store.CreateAIService(ctx, service)
	if err != nil {
		return s.store.UpdateAIService(ctx, service)
	}
	return nil
}

// RecordAIModel records a discovered AI model
func (s *Service) RecordAIModel(ctx context.Context, model *models.AIModel) error {
	model.ID = uuid.New()
	model.CreatedAt = time.Now()
	model.UpdatedAt = time.Now()

	err := s.store.CreateAIModel(ctx, model)
	if err != nil {
		return s.store.UpdateAIModel(ctx, model)
	}
	return nil
}

// RecordTrainingData records training data usage for a model
func (s *Service) RecordTrainingData(ctx context.Context, data *models.AITrainingData) error {
	data.ID = uuid.New()
	data.DiscoveredAt = time.Now()

	// Check if data source contains sensitive data
	asset, _ := s.store.GetDataAssetByARN(ctx, data.DataSourceARN)
	if asset != nil {
		data.ContainsSensitiveData = asset.SensitivityLevel != models.SensitivityUnknown &&
			asset.SensitivityLevel != models.SensitivityLow
		data.SensitivityLevel = asset.SensitivityLevel
		data.SensitivityCategories = asset.DataCategories
	}

	return s.store.CreateAITrainingData(ctx, data)
}

// RecordProcessingEvent records an AI data processing event
func (s *Service) RecordProcessingEvent(ctx context.Context, event *models.AIProcessingEvent) error {
	event.ID = uuid.New()
	event.CreatedAt = time.Now()

	// Calculate risk score
	event.RiskScore, event.RiskFactors = s.calculateEventRisk(ctx, event)

	return s.store.CreateAIProcessingEvent(ctx, event)
}

// calculateEventRisk calculates the risk score for a processing event
func (s *Service) calculateEventRisk(ctx context.Context, event *models.AIProcessingEvent) (int, []string) {
	score := 0
	var factors []string

	// Risk based on sensitivity level
	switch event.AccessedSensitivityLevel {
	case models.SensitivityCritical:
		score += 40
		factors = append(factors, "Accessing critical sensitivity data")
	case models.SensitivityHigh:
		score += 30
		factors = append(factors, "Accessing high sensitivity data")
	case models.SensitivityMedium:
		score += 15
	}

	// Risk based on data categories
	for _, cat := range event.AccessedCategories {
		switch cat {
		case "PII":
			score += 15
			factors = append(factors, "Accessing PII data")
		case "PHI":
			score += 20
			factors = append(factors, "Accessing PHI data")
		case "PCI":
			score += 15
			factors = append(factors, "Accessing PCI data")
		}
	}

	// Risk based on event type
	if event.EventType == models.AIEventTrainingJob {
		score += 10
		factors = append(factors, "Training job accessing sensitive data")
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score, factors
}

// ListAIServices returns all AI services for an account
func (s *Service) ListAIServices(ctx context.Context, accountID uuid.UUID) ([]*models.AIService, error) {
	return s.store.ListAIServices(ctx, accountID)
}

// ListAIModels returns all AI models for an account
func (s *Service) ListAIModels(ctx context.Context, accountID uuid.UUID) ([]*models.AIModel, error) {
	return s.store.ListAIModels(ctx, accountID)
}

// GetAIModel returns a specific AI model
func (s *Service) GetAIModel(ctx context.Context, modelID uuid.UUID) (*models.AIModel, error) {
	return s.store.GetAIModel(ctx, modelID)
}
