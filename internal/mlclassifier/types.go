package mlclassifier

import (
	"time"

	"github.com/google/uuid"
	"github.com/qualys/dspm/internal/models"
)

// MLResult contains the results of ML-enhanced classification
type MLResult struct {
	Matches           []EnhancedMatch       `json:"matches"`
	DocumentType      *DocumentClassification `json:"document_type,omitempty"`
	Entities          []Entity              `json:"entities"`
	OverallConfidence float64               `json:"overall_confidence"`
	RequiresReview    bool                  `json:"requires_review"`
	ReviewReason      string                `json:"review_reason,omitempty"`
}

// EnhancedMatch extends a regex match with ML-derived confidence
type EnhancedMatch struct {
	RuleName          string            `json:"rule_name"`
	Category          models.Category   `json:"category"`
	Sensitivity       models.Sensitivity `json:"sensitivity"`
	Value             string            `json:"value"`
	Count             int               `json:"count"`
	LineNumbers       []int             `json:"line_numbers,omitempty"`
	RegexConfidence   float64           `json:"regex_confidence"`
	MLConfidence      float64           `json:"ml_confidence"`
	CombinedConfidence float64          `json:"combined_confidence"`
	EntityType        string            `json:"entity_type,omitempty"`
	ContextScore      float64           `json:"context_score"`
}

// Entity represents a named entity recognized by NER
type Entity struct {
	Text        string  `json:"text"`
	Type        string  `json:"type"` // PERSON, ORGANIZATION, LOCATION, DATE, MEDICAL_TERM
	StartOffset int     `json:"start_offset"`
	EndOffset   int     `json:"end_offset"`
	Confidence  float64 `json:"confidence"`
}

// DocumentClassification represents document type classification
type DocumentClassification struct {
	Type       string   `json:"type"` // MEDICAL_RECORD, FINANCIAL_STATEMENT, LEGAL_DOCUMENT
	Confidence float64  `json:"confidence"`
	Indicators []string `json:"indicators,omitempty"`
}

// ReviewQueueItem represents an item in the review queue
type ReviewQueueItem struct {
	ID                 uuid.UUID                      `json:"id"`
	Classification     *models.Classification         `json:"classification"`
	Prediction         *models.MLPrediction           `json:"prediction,omitempty"`
	Priority           int                            `json:"priority"`
	Reason             string                         `json:"reason"`
	OriginalConfidence float64                        `json:"original_confidence"`
	Status             models.ReviewQueueStatus       `json:"status"`
	AssignedTo         *uuid.UUID                     `json:"assigned_to,omitempty"`
	CreatedAt          time.Time                      `json:"created_at"`
}

// ReviewResolution represents the resolution of a review item
type ReviewResolution struct {
	ItemID          uuid.UUID `json:"item_id"`
	Resolution      string    `json:"resolution"` // CONFIRMED, REJECTED, MODIFIED
	FinalLabel      string    `json:"final_label,omitempty"`
	FinalConfidence float64   `json:"final_confidence"`
	Notes           string    `json:"notes,omitempty"`
	ReviewedBy      uuid.UUID `json:"reviewed_by"`
}

// FeedbackSubmission represents training feedback from human review
type FeedbackSubmission struct {
	PredictionID       uuid.UUID `json:"prediction_id,omitempty"`
	OriginalPrediction string    `json:"original_prediction"`
	CorrectedLabel     string    `json:"corrected_label"`
	FeedbackType       string    `json:"feedback_type"` // CORRECTION, CONFIRMATION, FALSE_POSITIVE, FALSE_NEGATIVE
	SampleContent      string    `json:"sample_content,omitempty"`
	ContextWindow      string    `json:"context_window,omitempty"`
	SubmittedBy        uuid.UUID `json:"submitted_by"`
}

// ConfidenceParams contains parameters for confidence calculation
type ConfidenceParams struct {
	Match           *EnhancedMatch
	Content         string
	Context         string // Surrounding text
	RegexValidated  bool   // Did regex validators pass
	EntityConfirmed bool   // Did NER confirm the entity type
	FrequencyCount  int    // Number of occurrences
	DocumentType    string // Type of document if known
}

// ModelInfo contains information about a loaded ML model
type ModelInfo struct {
	ID            uuid.UUID         `json:"id"`
	Name          string            `json:"name"`
	Type          models.MLModelType `json:"type"`
	Version       string            `json:"version"`
	Framework     string            `json:"framework"`
	Status        string            `json:"status"`
	Accuracy      float64           `json:"accuracy,omitempty"`
	LastUsed      *time.Time        `json:"last_used,omitempty"`
}

// ClassifierConfig contains configuration for the ML classifier
type ClassifierConfig struct {
	Thresholds          models.ConfidenceThresholds `json:"thresholds"`
	EnableNER           bool                        `json:"enable_ner"`
	EnableDocClassifier bool                        `json:"enable_doc_classifier"`
	ContextWindowSize   int                         `json:"context_window_size"`
	MaxEntitiesPerDoc   int                         `json:"max_entities_per_doc"`
}

// DefaultClassifierConfig returns default configuration
func DefaultClassifierConfig() ClassifierConfig {
	return ClassifierConfig{
		Thresholds:          models.DefaultConfidenceThresholds(),
		EnableNER:           true,
		EnableDocClassifier: true,
		ContextWindowSize:   200,
		MaxEntitiesPerDoc:   1000,
	}
}

// SupportedEntityTypes returns the supported NER entity types
func SupportedEntityTypes() []string {
	return []string{
		"PERSON",
		"ORGANIZATION",
		"LOCATION",
		"DATE",
		"MEDICAL_TERM",
		"FINANCIAL_TERM",
		"EMAIL",
		"PHONE",
		"ADDRESS",
	}
}

// SupportedDocumentTypes returns the supported document types
func SupportedDocumentTypes() []string {
	return []string{
		"MEDICAL_RECORD",
		"FINANCIAL_STATEMENT",
		"LEGAL_DOCUMENT",
		"PII_DOCUMENT",
		"TECHNICAL_DOCUMENT",
		"GENERAL",
	}
}

// EntityTypeToCategory maps NER entity types to data categories
func EntityTypeToCategory(entityType string) models.Category {
	switch entityType {
	case "PERSON", "EMAIL", "PHONE", "ADDRESS":
		return models.CategoryPII
	case "MEDICAL_TERM":
		return models.CategoryPHI
	case "FINANCIAL_TERM":
		return models.CategoryPCI
	default:
		return models.CategoryCustom
	}
}
