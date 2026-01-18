package mlclassifier

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/qualys/dspm/internal/models"
)

// Service provides ML-enhanced classification functionality
type Service struct {
	store            Store
	scorer           *ConfidenceScorer
	config           ClassifierConfig
	entityRecognizer EntityRecognizer
	docClassifier    DocumentClassifier
}

// Store defines the interface for ML classifier data persistence
type Store interface {
	// ML Models
	CreateMLModel(ctx context.Context, model *models.MLModel) error
	UpdateMLModel(ctx context.Context, model *models.MLModel) error
	GetMLModel(ctx context.Context, id uuid.UUID) (*models.MLModel, error)
	GetDefaultMLModel(ctx context.Context, modelType models.MLModelType) (*models.MLModel, error)
	ListMLModels(ctx context.Context) ([]*models.MLModel, error)

	// ML Predictions
	CreateMLPrediction(ctx context.Context, prediction *models.MLPrediction) error
	UpdateMLPrediction(ctx context.Context, prediction *models.MLPrediction) error
	GetMLPrediction(ctx context.Context, id uuid.UUID) (*models.MLPrediction, error)
	ListMLPredictionsByClassification(ctx context.Context, classificationID uuid.UUID) ([]*models.MLPrediction, error)

	// Review Queue
	CreateReviewQueueItem(ctx context.Context, item *models.ClassificationReviewQueue) error
	UpdateReviewQueueItem(ctx context.Context, item *models.ClassificationReviewQueue) error
	GetReviewQueueItem(ctx context.Context, id uuid.UUID) (*models.ClassificationReviewQueue, error)
	ListReviewQueue(ctx context.Context, status models.ReviewQueueStatus, limit int) ([]*models.ClassificationReviewQueue, error)
	GetReviewQueueStats(ctx context.Context) (map[string]int, error)

	// Training Feedback
	CreateTrainingFeedback(ctx context.Context, feedback *models.TrainingFeedback) error
	ListTrainingFeedback(ctx context.Context, modelID uuid.UUID, incorporated bool) ([]*models.TrainingFeedback, error)
	MarkFeedbackIncorporated(ctx context.Context, feedbackIDs []uuid.UUID, trainingRunID string) error

	// Classifications (for lookups)
	GetClassification(ctx context.Context, id uuid.UUID) (*models.Classification, error)
	UpdateClassificationConfidence(ctx context.Context, id uuid.UUID, confidence float64, validated bool) error
}

// EntityRecognizer defines the interface for NER functionality
type EntityRecognizer interface {
	// RecognizeEntities extracts named entities from text
	RecognizeEntities(ctx context.Context, text string) ([]Entity, error)
}

// DocumentClassifier defines the interface for document classification
type DocumentClassifier interface {
	// ClassifyDocument determines the document type
	ClassifyDocument(ctx context.Context, text string) (*DocumentClassification, error)
}

// NewService creates a new ML classifier service
func NewService(store Store) *Service {
	return &Service{
		store:  store,
		scorer: NewConfidenceScorer(),
		config: DefaultClassifierConfig(),
	}
}

// NewServiceWithConfig creates a service with custom configuration
func NewServiceWithConfig(store Store, config ClassifierConfig) *Service {
	return &Service{
		store:  store,
		scorer: NewConfidenceScorer(),
		config: config,
	}
}

// SetEntityRecognizer sets the NER implementation
func (s *Service) SetEntityRecognizer(er EntityRecognizer) {
	s.entityRecognizer = er
}

// SetDocumentClassifier sets the document classifier implementation
func (s *Service) SetDocumentClassifier(dc DocumentClassifier) {
	s.docClassifier = dc
}

// EnhanceClassification enhances regex matches with ML-based confidence scoring
func (s *Service) EnhanceClassification(ctx context.Context, content string, regexMatches []EnhancedMatch) (*MLResult, error) {
	result := &MLResult{
		Matches:           make([]EnhancedMatch, 0, len(regexMatches)),
		Entities:          []Entity{},
		OverallConfidence: 0,
		RequiresReview:    false,
	}

	// Get document classification if enabled
	var docType string
	if s.config.EnableDocClassifier && s.docClassifier != nil {
		docClassification, err := s.docClassifier.ClassifyDocument(ctx, content)
		if err == nil && docClassification != nil {
			result.DocumentType = docClassification
			docType = docClassification.Type
		}
	}

	// Extract entities if enabled
	var entities []Entity
	if s.config.EnableNER && s.entityRecognizer != nil {
		var err error
		entities, err = s.entityRecognizer.RecognizeEntities(ctx, content)
		if err == nil {
			result.Entities = entities
		}
	}

	// Enhance each match with ML confidence
	var totalConfidence float64
	for _, match := range regexMatches {
		enhanced := s.enhanceMatch(content, match, entities, docType)
		result.Matches = append(result.Matches, enhanced)
		totalConfidence += enhanced.CombinedConfidence
	}

	// Calculate overall confidence
	if len(result.Matches) > 0 {
		result.OverallConfidence = totalConfidence / float64(len(result.Matches))
	}

	// Determine if review is required
	result.RequiresReview, result.ReviewReason = s.shouldRequireReview(result)

	return result, nil
}

// enhanceMatch enhances a single match with ML confidence
func (s *Service) enhanceMatch(content string, match EnhancedMatch, entities []Entity, docType string) EnhancedMatch {
	// Extract context around the match
	contextWindow := s.extractContext(content, match.Value)

	// Check if NER confirmed this entity
	entityConfirmed := s.isEntityConfirmed(match, entities)
	entityType := ""
	if entityConfirmed {
		entityType = s.findMatchingEntityType(match, entities)
	}

	// Calculate ML confidence
	params := ConfidenceParams{
		Match:           &match,
		Content:         content,
		Context:         contextWindow,
		RegexValidated:  match.RegexConfidence >= 0.8,
		EntityConfirmed: entityConfirmed,
		FrequencyCount:  match.Count,
		DocumentType:    docType,
	}

	mlConfidence := s.scorer.CalculateConfidence(params)

	// Adjust for document type
	mlConfidence = AdjustForDocumentType(mlConfidence, docType, string(match.Category))

	// Combine regex and ML confidence
	combinedConfidence := CombineConfidenceScores(match.RegexConfidence, mlConfidence, 0.4)

	match.MLConfidence = mlConfidence
	match.CombinedConfidence = combinedConfidence
	if params.Context != "" {
		match.ContextScore = s.scorer.calculateContextScore(params)
	}
	match.EntityType = entityType

	return match
}

// extractContext extracts surrounding text for context analysis
func (s *Service) extractContext(content, value string) string {
	idx := findSubstringIndex(content, value)
	if idx < 0 {
		return ""
	}

	windowSize := s.config.ContextWindowSize
	start := max(0, idx-windowSize)
	end := min(len(content), idx+len(value)+windowSize)

	return content[start:end]
}

// isEntityConfirmed checks if NER found a matching entity
func (s *Service) isEntityConfirmed(match EnhancedMatch, entities []Entity) bool {
	for _, entity := range entities {
		// Check if the entity text matches the match value
		if entity.Text == match.Value || containsSubstring(entity.Text, match.Value) {
			return true
		}
	}
	return false
}

// findMatchingEntityType finds the NER entity type for a match
func (s *Service) findMatchingEntityType(match EnhancedMatch, entities []Entity) string {
	for _, entity := range entities {
		if entity.Text == match.Value || containsSubstring(entity.Text, match.Value) {
			return entity.Type
		}
	}
	return ""
}

// shouldRequireReview determines if a result needs human review
func (s *Service) shouldRequireReview(result *MLResult) (bool, string) {
	thresholds := s.config.Thresholds

	// Check if any match is below auto-approve but above auto-reject
	for _, match := range result.Matches {
		if match.CombinedConfidence < thresholds.AutoApprove &&
			match.CombinedConfidence >= thresholds.RequireReview {
			return true, "LOW_CONFIDENCE"
		}
	}

	// Check for conflicting entity types
	if len(result.Matches) > 1 && s.hasConflictingTypes(result.Matches) {
		return true, "CONFLICTING_PREDICTIONS"
	}

	// Check for high sensitivity data
	for _, match := range result.Matches {
		if match.Sensitivity == models.SensitivityCritical &&
			match.CombinedConfidence < thresholds.AutoApprove {
			return true, "SENSITIVE_DATA"
		}
	}

	return false, ""
}

// hasConflictingTypes checks if matches have conflicting categorizations
func (s *Service) hasConflictingTypes(matches []EnhancedMatch) bool {
	categories := make(map[models.Category]bool)
	for _, match := range matches {
		if categories[match.Category] {
			continue
		}
		categories[match.Category] = true
	}
	// Multiple different categories for similar matches might indicate confusion
	return len(categories) > 2
}

// QueueForReview adds a classification to the review queue
func (s *Service) QueueForReview(ctx context.Context, classificationID uuid.UUID, predictionID *uuid.UUID, reason string, confidence float64) error {
	item := &models.ClassificationReviewQueue{
		ID:                 uuid.New(),
		ClassificationID:   classificationID,
		PredictionID:       predictionID,
		Priority:           s.calculatePriority(reason, confidence),
		Reason:             reason,
		OriginalConfidence: confidence,
		Status:             models.ReviewQueueStatusPending,
		CreatedAt:          time.Now(),
	}

	return s.store.CreateReviewQueueItem(ctx, item)
}

// calculatePriority calculates review priority based on reason and confidence
func (s *Service) calculatePriority(reason string, confidence float64) int {
	priority := 0

	// Lower confidence = higher priority
	if confidence < 0.5 {
		priority += 30
	} else if confidence < 0.7 {
		priority += 20
	}

	// Reason-based priority
	switch reason {
	case "SENSITIVE_DATA":
		priority += 50
	case "CONFLICTING_PREDICTIONS":
		priority += 30
	case "LOW_CONFIDENCE":
		priority += 20
	}

	return priority
}

// GetReviewQueue returns items in the review queue
func (s *Service) GetReviewQueue(ctx context.Context, status models.ReviewQueueStatus, limit int) ([]*ReviewQueueItem, error) {
	items, err := s.store.ListReviewQueue(ctx, status, limit)
	if err != nil {
		return nil, fmt.Errorf("listing review queue: %w", err)
	}

	result := make([]*ReviewQueueItem, 0, len(items))
	for _, item := range items {
		queueItem := &ReviewQueueItem{
			ID:                 item.ID,
			Priority:           item.Priority,
			Reason:             item.Reason,
			OriginalConfidence: item.OriginalConfidence,
			Status:             item.Status,
			AssignedTo:         item.AssignedTo,
			CreatedAt:          item.CreatedAt,
		}

		// Get classification details
		classification, err := s.store.GetClassification(ctx, item.ClassificationID)
		if err == nil {
			queueItem.Classification = classification
		}

		// Get prediction details if available
		if item.PredictionID != nil {
			prediction, err := s.store.GetMLPrediction(ctx, *item.PredictionID)
			if err == nil {
				queueItem.Prediction = prediction
			}
		}

		result = append(result, queueItem)
	}

	return result, nil
}

// ResolveReviewItem resolves a review queue item
func (s *Service) ResolveReviewItem(ctx context.Context, resolution *ReviewResolution) error {
	// Get the queue item
	item, err := s.store.GetReviewQueueItem(ctx, resolution.ItemID)
	if err != nil {
		return fmt.Errorf("getting review item: %w", err)
	}

	// Update queue item
	now := time.Now()
	item.Status = models.ReviewQueueStatusResolved
	item.ResolvedAt = &now
	item.Resolution = resolution.Resolution
	item.FinalLabel = resolution.FinalLabel
	item.FinalConfidence = resolution.FinalConfidence

	if err := s.store.UpdateReviewQueueItem(ctx, item); err != nil {
		return fmt.Errorf("updating review item: %w", err)
	}

	// Update classification confidence if confirmed
	if resolution.Resolution == "CONFIRMED" || resolution.Resolution == "MODIFIED" {
		if err := s.store.UpdateClassificationConfidence(ctx, item.ClassificationID, resolution.FinalConfidence, true); err != nil {
			return fmt.Errorf("updating classification: %w", err)
		}
	}

	return nil
}

// SubmitFeedback submits training feedback from human review
func (s *Service) SubmitFeedback(ctx context.Context, submission *FeedbackSubmission) error {
	// Hash the sample content for deduplication
	hash := sha256.Sum256([]byte(submission.SampleContent))
	hashStr := hex.EncodeToString(hash[:])

	feedback := &models.TrainingFeedback{
		ID:                 uuid.New(),
		PredictionID:       &submission.PredictionID,
		OriginalPrediction: submission.OriginalPrediction,
		CorrectedLabel:     submission.CorrectedLabel,
		FeedbackType:       submission.FeedbackType,
		SampleContent:      submission.SampleContent,
		SampleHash:         hashStr,
		ContextWindow:      submission.ContextWindow,
		SubmittedBy:        &submission.SubmittedBy,
		SubmittedAt:        time.Now(),
	}

	return s.store.CreateTrainingFeedback(ctx, feedback)
}

// ListMLModels returns all registered ML models
func (s *Service) ListMLModels(ctx context.Context) ([]*ModelInfo, error) {
	models, err := s.store.ListMLModels(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*ModelInfo, 0, len(models))
	for _, m := range models {
		result = append(result, &ModelInfo{
			ID:        m.ID,
			Name:      m.Name,
			Type:      m.ModelType,
			Version:   m.Version,
			Framework: m.Framework,
			Status:    string(m.Status),
			Accuracy:  m.Accuracy,
		})
	}

	return result, nil
}

// GetReviewQueueStats returns statistics about the review queue
func (s *Service) GetReviewQueueStats(ctx context.Context) (map[string]int, error) {
	return s.store.GetReviewQueueStats(ctx)
}

// Helper functions

func findSubstringIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func containsSubstring(s, substr string) bool {
	return findSubstringIndex(s, substr) >= 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
