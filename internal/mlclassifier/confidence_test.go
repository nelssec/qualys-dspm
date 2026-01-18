package mlclassifier

import (
	"testing"

	"github.com/qualys/dspm/internal/models"
)

func TestConfidenceScorer_CalculateConfidence(t *testing.T) {
	scorer := NewConfidenceScorer()

	tests := []struct {
		name          string
		params        ConfidenceParams
		minConfidence float64
		maxConfidence float64
	}{
		{
			name: "high confidence - all indicators positive",
			params: ConfidenceParams{
				Match: &EnhancedMatch{
					Category:        models.CategoryPCI,
					RegexConfidence: 0.95,
				},
				Content:         "Credit card payment: 4111111111111111",
				Context:         "Process credit card payment for customer order",
				RegexValidated:  true,
				EntityConfirmed: true,
				FrequencyCount:  3,
			},
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name: "medium confidence - regex validated only",
			params: ConfidenceParams{
				Match: &EnhancedMatch{
					Category:        models.CategoryPCI,
					RegexConfidence: 0.8,
				},
				Content:         "4111111111111111",
				Context:         "",
				RegexValidated:  true,
				EntityConfirmed: false,
				FrequencyCount:  1,
			},
			minConfidence: 0.5,
			maxConfidence: 0.75,
		},
		{
			name: "low confidence - no validation, test context",
			params: ConfidenceParams{
				Match: &EnhancedMatch{
					Category:        models.CategoryPCI,
					RegexConfidence: 0.7,
				},
				Content:         "4111111111111111",
				Context:         "This is a test example card number",
				RegexValidated:  false,
				EntityConfirmed: false,
				FrequencyCount:  1,
			},
			minConfidence: 0.2,
			maxConfidence: 0.5,
		},
		{
			name: "high frequency boosts confidence",
			params: ConfidenceParams{
				Match: &EnhancedMatch{
					Category:        models.CategoryPII,
					RegexConfidence: 0.85,
				},
				Content:        "email@example.com",
				RegexValidated: true,
				FrequencyCount: 10,
			},
			minConfidence: 0.6,
			maxConfidence: 0.9,
		},
		{
			name: "NER confirmation boosts confidence",
			params: ConfidenceParams{
				Match: &EnhancedMatch{
					Category:        models.CategoryPII,
					RegexConfidence: 0.8,
				},
				Content:         "123-45-6789",
				Context:         "Customer SSN: 123-45-6789",
				RegexValidated:  true,
				EntityConfirmed: true,
				FrequencyCount:  1,
			},
			minConfidence: 0.7,
			maxConfidence: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := scorer.CalculateConfidence(tt.params)

			if confidence < tt.minConfidence || confidence > tt.maxConfidence {
				t.Errorf("CalculateConfidence() = %v, want between %v and %v",
					confidence, tt.minConfidence, tt.maxConfidence)
			}
		})
	}
}

func TestConfidenceScorer_PatternScore(t *testing.T) {
	scorer := NewConfidenceScorer()

	tests := []struct {
		name          string
		params        ConfidenceParams
		expectedScore float64
		tolerance     float64
	}{
		{
			name: "base score only",
			params: ConfidenceParams{
				RegexValidated: false,
				Match:          nil,
			},
			expectedScore: 0.5,
			tolerance:     0.01,
		},
		{
			name: "validated pattern",
			params: ConfidenceParams{
				RegexValidated: true,
				Match:          nil,
			},
			expectedScore: 0.9,
			tolerance:     0.01,
		},
		{
			name: "high regex confidence bonus",
			params: ConfidenceParams{
				RegexValidated: true,
				Match: &EnhancedMatch{
					RegexConfidence: 0.95,
				},
			},
			expectedScore: 1.0,
			tolerance:     0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.calculatePatternScore(tt.params)

			if score < tt.expectedScore-tt.tolerance || score > tt.expectedScore+tt.tolerance {
				t.Errorf("calculatePatternScore() = %v, want %v (tolerance %v)",
					score, tt.expectedScore, tt.tolerance)
			}
		})
	}
}

func TestConfidenceScorer_ContextScore(t *testing.T) {
	scorer := NewConfidenceScorer()

	tests := []struct {
		name     string
		params   ConfidenceParams
		minScore float64
		maxScore float64
	}{
		{
			name: "no context",
			params: ConfidenceParams{
				Context: "",
			},
			minScore: 0.45,
			maxScore: 0.55, // Should be ~0.5
		},
		{
			name: "relevant PCI context",
			params: ConfidenceParams{
				Context: "Credit card payment transaction for billing",
				Match: &EnhancedMatch{
					Category: models.CategoryPCI,
				},
			},
			minScore: 0.6,
			maxScore: 1.0,
		},
		{
			name: "relevant PHI context",
			params: ConfidenceParams{
				Context: "Patient medical record diagnosis treatment",
				Match: &EnhancedMatch{
					Category: models.CategoryPHI,
				},
			},
			minScore: 0.6,
			maxScore: 1.0,
		},
		{
			name: "negative indicator - test",
			params: ConfidenceParams{
				Context: "This is a test example for demo purposes",
				Match: &EnhancedMatch{
					Category: models.CategoryPCI,
				},
			},
			minScore: 0.0,
			maxScore: 0.3,
		},
		{
			name: "negative indicator - fake",
			params: ConfidenceParams{
				Context: "Fake sample data for placeholder",
				Match: &EnhancedMatch{
					Category: models.CategoryPII,
				},
			},
			minScore: 0.0,
			maxScore: 0.3,
		},
		{
			name: "relevant PII context",
			params: ConfidenceParams{
				Context: "Customer personal contact email phone address",
				Match: &EnhancedMatch{
					Category: models.CategoryPII,
				},
			},
			minScore: 0.6,
			maxScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.calculateContextScore(tt.params)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateContextScore() = %v, want between %v and %v",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestConfidenceScorer_FrequencyScore(t *testing.T) {
	scorer := NewConfidenceScorer()

	tests := []struct {
		name          string
		frequency     int
		expectedScore float64
	}{
		{"zero frequency", 0, 0.6},
		{"single occurrence", 1, 0.6},
		{"two occurrences", 2, 0.8},
		{"five occurrences", 5, 0.8},
		{"ten occurrences", 10, 0.9},
		{"twenty occurrences", 20, 0.9},
		{"many occurrences", 100, 0.7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := ConfidenceParams{FrequencyCount: tt.frequency}
			score := scorer.calculateFrequencyScore(params)

			if score != tt.expectedScore {
				t.Errorf("calculateFrequencyScore(%d) = %v, want %v",
					tt.frequency, score, tt.expectedScore)
			}
		})
	}
}

func TestConfidenceScorer_NERScore(t *testing.T) {
	scorer := NewConfidenceScorer()

	tests := []struct {
		name          string
		params        ConfidenceParams
		expectedScore float64
	}{
		{
			name: "NER confirmed",
			params: ConfidenceParams{
				EntityConfirmed: true,
			},
			expectedScore: 0.95,
		},
		{
			name: "NER not confirmed, entity type present",
			params: ConfidenceParams{
				EntityConfirmed: false,
				Match: &EnhancedMatch{
					EntityType: "EMAIL",
				},
			},
			expectedScore: 0.4,
		},
		{
			name: "NER not available",
			params: ConfidenceParams{
				EntityConfirmed: false,
				Match:           nil,
			},
			expectedScore: 0.5,
		},
		{
			name: "NER available but no entity type",
			params: ConfidenceParams{
				EntityConfirmed: false,
				Match: &EnhancedMatch{
					EntityType: "",
				},
			},
			expectedScore: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.calculateNERScore(tt.params)

			if score != tt.expectedScore {
				t.Errorf("calculateNERScore() = %v, want %v", score, tt.expectedScore)
			}
		})
	}
}

func TestCombineConfidenceScores(t *testing.T) {
	tests := []struct {
		name        string
		regexConf   float64
		mlConf      float64
		regexWeight float64
		expected    float64
	}{
		{
			name:        "equal weights",
			regexConf:   0.8,
			mlConf:      0.6,
			regexWeight: 0.5,
			expected:    0.7,
		},
		{
			name:        "favor regex",
			regexConf:   0.9,
			mlConf:      0.5,
			regexWeight: 0.8,
			expected:    0.82, // 0.9*0.8 + 0.5*0.2
		},
		{
			name:        "favor ML",
			regexConf:   0.5,
			mlConf:      0.9,
			regexWeight: 0.2,
			expected:    0.82, // 0.5*0.2 + 0.9*0.8
		},
		{
			name:        "both high",
			regexConf:   1.0,
			mlConf:      1.0,
			regexWeight: 0.5,
			expected:    1.0,
		},
		{
			name:        "both low",
			regexConf:   0.2,
			mlConf:      0.3,
			regexWeight: 0.5,
			expected:    0.25,
		},
		{
			name:        "invalid weight - uses default",
			regexConf:   0.8,
			mlConf:      0.6,
			regexWeight: 1.5, // Invalid, should default to 0.5
			expected:    0.7,
		},
		{
			name:        "negative weight - uses default",
			regexConf:   0.8,
			mlConf:      0.6,
			regexWeight: -0.2, // Invalid, should default to 0.5
			expected:    0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CombineConfidenceScores(tt.regexConf, tt.mlConf, tt.regexWeight)

			tolerance := 0.01
			if result < tt.expected-tolerance || result > tt.expected+tolerance {
				t.Errorf("CombineConfidenceScores(%v, %v, %v) = %v, want %v",
					tt.regexConf, tt.mlConf, tt.regexWeight, result, tt.expected)
			}
		})
	}
}

func TestAdjustForDocumentType(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		docType    string
		category   string
		minResult  float64
		maxResult  float64
	}{
		{
			name:       "PHI in medical document - boosted",
			confidence: 0.7,
			docType:    "MEDICAL_RECORD",
			category:   "PHI",
			minResult:  0.8,
			maxResult:  0.85,
		},
		{
			name:       "PCI in financial document - boosted",
			confidence: 0.7,
			docType:    "FINANCIAL_STATEMENT",
			category:   "PCI",
			minResult:  0.8,
			maxResult:  0.85,
		},
		{
			name:       "technical document - reduced",
			confidence: 0.8,
			docType:    "TECHNICAL_DOCUMENT",
			category:   "PII",
			minResult:  0.65,
			maxResult:  0.70,
		},
		{
			name:       "generic document - unchanged",
			confidence: 0.7,
			docType:    "GENERAL",
			category:   "PII",
			minResult:  0.69,
			maxResult:  0.71,
		},
		{
			name:       "PHI in non-medical - unchanged",
			confidence: 0.7,
			docType:    "FINANCIAL_STATEMENT",
			category:   "PHI",
			minResult:  0.69,
			maxResult:  0.71,
		},
		{
			name:       "max confidence capping",
			confidence: 0.95,
			docType:    "MEDICAL_RECORD",
			category:   "PHI",
			minResult:  1.0,
			maxResult:  1.0, // Should be capped at 1.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AdjustForDocumentType(tt.confidence, tt.docType, tt.category)

			if result < tt.minResult || result > tt.maxResult {
				t.Errorf("AdjustForDocumentType(%v, %q, %q) = %v, want between %v and %v",
					tt.confidence, tt.docType, tt.category, result, tt.minResult, tt.maxResult)
			}
		})
	}
}

func TestGetContextKeywords(t *testing.T) {
	tests := []struct {
		category      string
		shouldContain []string
	}{
		{
			category:      "PII",
			shouldContain: []string{"name", "email", "ssn", "customer"},
		},
		{
			category:      "PHI",
			shouldContain: []string{"patient", "medical", "diagnosis", "treatment"},
		},
		{
			category:      "PCI",
			shouldContain: []string{"card", "credit", "payment", "cvv"},
		},
		{
			category:      "SECRETS",
			shouldContain: []string{"key", "secret", "token", "password"},
		},
		{
			category:      "UNKNOWN",
			shouldContain: []string{}, // Should return empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			keywords := getContextKeywords(tt.category)

			for _, expected := range tt.shouldContain {
				found := false
				for _, kw := range keywords {
					if kw == expected {
						found = true
						break
					}
				}
				if !found && len(tt.shouldContain) > 0 {
					t.Errorf("getContextKeywords(%q) missing expected keyword %q", tt.category, expected)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkConfidenceScorer_CalculateConfidence(b *testing.B) {
	scorer := NewConfidenceScorer()
	params := ConfidenceParams{
		Match: &EnhancedMatch{
			Category:        models.CategoryPCI,
			RegexConfidence: 0.9,
		},
		Content:         "Credit card: 4111111111111111",
		Context:         "Process payment transaction for customer",
		RegexValidated:  true,
		EntityConfirmed: true,
		FrequencyCount:  5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scorer.CalculateConfidence(params)
	}
}

func BenchmarkCombineConfidenceScores(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CombineConfidenceScores(0.85, 0.75, 0.4)
	}
}
