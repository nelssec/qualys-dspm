package mlclassifier

import (
	"math"
	"strings"
)

// ConfidenceScorer calculates refined confidence scores
type ConfidenceScorer struct {
	contextWeight   float64
	patternWeight   float64
	frequencyWeight float64
	nerWeight       float64
}

// NewConfidenceScorer creates a new confidence scorer with default weights
func NewConfidenceScorer() *ConfidenceScorer {
	return &ConfidenceScorer{
		contextWeight:   0.25,
		patternWeight:   0.35,
		frequencyWeight: 0.15,
		nerWeight:       0.25,
	}
}

// CalculateConfidence computes a weighted confidence score
func (cs *ConfidenceScorer) CalculateConfidence(params ConfidenceParams) float64 {
	// Pattern match quality (Luhn check passed, format valid, etc.)
	patternScore := cs.calculatePatternScore(params)

	// Context relevance (surrounding text analysis)
	contextScore := cs.calculateContextScore(params)

	// Frequency analysis (single vs. multiple occurrences)
	frequencyScore := cs.calculateFrequencyScore(params)

	// NER confirmation (entity type matches expected)
	nerScore := cs.calculateNERScore(params)

	combined := patternScore*cs.patternWeight +
		contextScore*cs.contextWeight +
		frequencyScore*cs.frequencyWeight +
		nerScore*cs.nerWeight

	return math.Min(combined, 1.0)
}

// calculatePatternScore scores based on pattern match quality
func (cs *ConfidenceScorer) calculatePatternScore(params ConfidenceParams) float64 {
	score := 0.5 // Base score for regex match

	// If validators passed (e.g., Luhn check for credit cards)
	if params.RegexValidated {
		score += 0.4
	}

	// Exact format match bonus
	if params.Match != nil && params.Match.RegexConfidence >= 0.9 {
		score += 0.1
	}

	return math.Min(score, 1.0)
}

// calculateContextScore scores based on surrounding context
func (cs *ConfidenceScorer) calculateContextScore(params ConfidenceParams) float64 {
	if params.Context == "" {
		return 0.5 // Neutral score when no context available
	}

	score := 0.3 // Base score
	contextLower := strings.ToLower(params.Context)

	// Check for relevant context keywords based on category
	if params.Match != nil {
		keywords := getContextKeywords(string(params.Match.Category))
		matchedKeywords := 0

		for _, kw := range keywords {
			if strings.Contains(contextLower, kw) {
				matchedKeywords++
			}
		}

		// More keyword matches = higher confidence
		if matchedKeywords > 0 {
			score += math.Min(float64(matchedKeywords)*0.15, 0.6)
		}
	}

	// Check for negative indicators (likely false positive)
	negativeIndicators := []string{
		"example", "test", "sample", "demo", "fake",
		"placeholder", "dummy", "mock", "xxx",
	}

	for _, indicator := range negativeIndicators {
		if strings.Contains(contextLower, indicator) {
			score -= 0.3
			break
		}
	}

	return math.Max(0, math.Min(score, 1.0))
}

// calculateFrequencyScore scores based on occurrence frequency
func (cs *ConfidenceScorer) calculateFrequencyScore(params ConfidenceParams) float64 {
	count := params.FrequencyCount
	if count <= 0 {
		count = 1
	}

	// Single occurrence is slightly suspicious for some types
	if count == 1 {
		return 0.6
	}

	// Multiple occurrences increase confidence up to a point
	if count >= 2 && count <= 5 {
		return 0.8
	}

	// Many occurrences might indicate either real data or a pattern/template
	if count > 5 && count <= 20 {
		return 0.9
	}

	// Very high frequency might indicate template/generated data
	return 0.7
}

// calculateNERScore scores based on NER confirmation
func (cs *ConfidenceScorer) calculateNERScore(params ConfidenceParams) float64 {
	if params.EntityConfirmed {
		return 0.95 // High confidence when NER confirms
	}

	// If NER is available but didn't confirm, slight penalty
	if params.Match != nil && params.Match.EntityType != "" {
		return 0.4
	}

	// NER not applicable or not available
	return 0.5
}

// getContextKeywords returns relevant keywords for a category
func getContextKeywords(category string) []string {
	switch category {
	case "PII":
		return []string{
			"name", "address", "phone", "email", "ssn", "social security",
			"date of birth", "dob", "passport", "license", "personal",
			"contact", "customer", "employee", "user", "member",
		}
	case "PHI":
		return []string{
			"patient", "medical", "diagnosis", "prescription", "doctor",
			"hospital", "clinic", "health", "treatment", "medication",
			"symptom", "condition", "mrn", "record", "insurance",
		}
	case "PCI":
		return []string{
			"card", "credit", "debit", "payment", "transaction",
			"cvv", "expir", "account", "bank", "billing", "charge",
			"purchase", "amount", "visa", "mastercard", "amex",
		}
	case "SECRETS":
		return []string{
			"key", "secret", "token", "password", "credential",
			"api", "auth", "access", "private", "connection",
		}
	default:
		return []string{}
	}
}

// CombineConfidenceScores combines regex and ML confidence scores
func CombineConfidenceScores(regexConf, mlConf float64, regexWeight float64) float64 {
	if regexWeight < 0 || regexWeight > 1 {
		regexWeight = 0.5 // Default to equal weighting
	}

	mlWeight := 1.0 - regexWeight

	combined := regexConf*regexWeight + mlConf*mlWeight

	return math.Min(combined, 1.0)
}

// AdjustForDocumentType adjusts confidence based on document type
func AdjustForDocumentType(confidence float64, docType, category string) float64 {
	// Medical documents should have higher confidence for PHI
	if docType == "MEDICAL_RECORD" && category == "PHI" {
		return math.Min(confidence*1.15, 1.0)
	}

	// Financial documents should have higher confidence for PCI
	if docType == "FINANCIAL_STATEMENT" && category == "PCI" {
		return math.Min(confidence*1.15, 1.0)
	}

	// Technical documents often have false positives
	if docType == "TECHNICAL_DOCUMENT" {
		return confidence * 0.85
	}

	return confidence
}
