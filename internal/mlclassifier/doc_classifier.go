package mlclassifier

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
)

// DocumentType represents document classification types
type DocumentType string

const (
	DocTypeMedical    DocumentType = "MEDICAL_RECORD"
	DocTypeFinancial  DocumentType = "FINANCIAL_STATEMENT"
	DocTypeLegal      DocumentType = "LEGAL_DOCUMENT"
	DocTypePII        DocumentType = "PII_DOCUMENT"
	DocTypeTechnical  DocumentType = "TECHNICAL_DOCUMENT"
	DocTypeGeneral    DocumentType = "GENERAL"
)

// ONNXDocumentClassifier classifies documents by type using ONNX models and rules
type ONNXDocumentClassifier struct {
	runtime      *ONNXRuntime
	tokenizer    *Tokenizer
	logger       *slog.Logger
	modelLoaded  bool
	labels       []DocumentType
	rulePatterns map[DocumentType][]documentPattern
}

// documentPattern defines a rule-based pattern for document classification
type documentPattern struct {
	Pattern     *regexp.Regexp
	Weight      float64
	Description string
}

// DocumentClassifierConfig contains configuration for the document classifier
type DocumentClassifierConfig struct {
	ModelPath    string
	VocabPath    string
	MaxSeqLength int
}

// NewONNXDocumentClassifier creates a new document classifier
func NewONNXDocumentClassifier(config DocumentClassifierConfig, runtime *ONNXRuntime, logger *slog.Logger) (*ONNXDocumentClassifier, error) {
	dc := &ONNXDocumentClassifier{
		runtime:     runtime,
		logger:      logger,
		modelLoaded: false,
		labels: []DocumentType{
			DocTypeMedical,
			DocTypeFinancial,
			DocTypeLegal,
			DocTypePII,
			DocTypeTechnical,
			DocTypeGeneral,
		},
		rulePatterns: make(map[DocumentType][]documentPattern),
	}

	// Initialize tokenizer
	tokConfig := TokenizerConfig{
		VocabFile: config.VocabPath,
		MaxLength: config.MaxSeqLength,
	}
	if tokConfig.MaxLength == 0 {
		tokConfig.MaxLength = 512
	}

	var err error
	dc.tokenizer, err = NewTokenizer(tokConfig)
	if err != nil {
		logger.Warn("failed to initialize tokenizer", "error", err)
	}

	// Initialize rule-based patterns
	dc.initializePatterns()

	// Load ONNX model if available
	if runtime != nil && runtime.IsAvailable() && config.ModelPath != "" {
		if err := runtime.LoadModel("doc_classifier", config.ModelPath); err != nil {
			logger.Warn("failed to load document classifier model", "error", err)
		} else {
			dc.modelLoaded = true
		}
	}

	return dc, nil
}

// initializePatterns sets up rule-based classification patterns
func (dc *ONNXDocumentClassifier) initializePatterns() {
	// Medical document patterns
	dc.rulePatterns[DocTypeMedical] = []documentPattern{
		{regexp.MustCompile(`(?i)\b(diagnosis|diagnoses|diagnosed)\b`), 0.8, "diagnosis"},
		{regexp.MustCompile(`(?i)\b(patient|patients)\b`), 0.6, "patient"},
		{regexp.MustCompile(`(?i)\b(medical|clinical|hospital)\b`), 0.6, "medical terms"},
		{regexp.MustCompile(`(?i)\b(prescription|medication|drug|dosage)\b`), 0.7, "prescription"},
		{regexp.MustCompile(`(?i)\b(treatment|therapy|procedure)\b`), 0.6, "treatment"},
		{regexp.MustCompile(`(?i)\b(ICD-10|CPT|HIPAA)\b`), 0.9, "medical codes"},
		{regexp.MustCompile(`(?i)\b(symptoms|vitals|blood pressure|heart rate)\b`), 0.7, "medical measurements"},
		{regexp.MustCompile(`(?i)\b(MRI|CT scan|X-ray|ultrasound)\b`), 0.8, "imaging"},
		{regexp.MustCompile(`(?i)\b(physician|doctor|nurse|surgeon)\b`), 0.6, "medical personnel"},
		{regexp.MustCompile(`(?i)\b(insurance|coverage|claim|billing)\b.*\b(medical|health)\b`), 0.7, "medical insurance"},
	}

	// Financial document patterns
	dc.rulePatterns[DocTypeFinancial] = []documentPattern{
		{regexp.MustCompile(`(?i)\b(balance sheet|income statement|cash flow)\b`), 0.9, "financial statements"},
		{regexp.MustCompile(`(?i)\b(revenue|profit|loss|earnings)\b`), 0.6, "financial terms"},
		{regexp.MustCompile(`(?i)\b(assets|liabilities|equity)\b`), 0.7, "balance sheet items"},
		{regexp.MustCompile(`(?i)\b(bank|account|transaction|transfer)\b`), 0.6, "banking"},
		{regexp.MustCompile(`(?i)\b(investment|portfolio|stock|bond|mutual fund)\b`), 0.7, "investment"},
		{regexp.MustCompile(`(?i)\b(tax|IRS|W-2|1099|deduction)\b`), 0.8, "tax"},
		{regexp.MustCompile(`(?i)\b(credit|debit|interest|principal)\b`), 0.6, "financial operations"},
		{regexp.MustCompile(`(?i)\b(quarterly|annual|fiscal|FY\d{2,4})\b`), 0.6, "reporting period"},
		{regexp.MustCompile(`(?i)\$[\d,]+(\.\d{2})?`), 0.5, "dollar amounts"},
		{regexp.MustCompile(`(?i)\b(GAAP|IFRS|SOX|audit)\b`), 0.8, "financial standards"},
	}

	// Legal document patterns
	dc.rulePatterns[DocTypeLegal] = []documentPattern{
		{regexp.MustCompile(`(?i)\b(contract|agreement|terms and conditions)\b`), 0.8, "contracts"},
		{regexp.MustCompile(`(?i)\b(plaintiff|defendant|court|judge)\b`), 0.8, "litigation"},
		{regexp.MustCompile(`(?i)\b(attorney|lawyer|counsel|law firm)\b`), 0.7, "legal parties"},
		{regexp.MustCompile(`(?i)\b(hereby|whereas|notwithstanding|herein)\b`), 0.7, "legal language"},
		{regexp.MustCompile(`(?i)\b(liability|indemnify|warrant|represent)\b`), 0.7, "legal terms"},
		{regexp.MustCompile(`(?i)\b(intellectual property|patent|trademark|copyright)\b`), 0.8, "IP"},
		{regexp.MustCompile(`(?i)\b(subpoena|deposition|discovery|verdict)\b`), 0.8, "legal process"},
		{regexp.MustCompile(`(?i)\b(jurisdiction|venue|applicable law|governing law)\b`), 0.7, "jurisdiction"},
		{regexp.MustCompile(`(?i)§\s*\d+`), 0.7, "section references"},
		{regexp.MustCompile(`(?i)\b(NDA|non-disclosure|confidentiality agreement)\b`), 0.9, "NDA"},
	}

	// PII document patterns
	dc.rulePatterns[DocTypePII] = []documentPattern{
		{regexp.MustCompile(`(?i)\b(social security|SSN)\b`), 0.9, "SSN"},
		{regexp.MustCompile(`(?i)\b(date of birth|DOB|birthdate)\b`), 0.8, "DOB"},
		{regexp.MustCompile(`(?i)\b(driver'?s? license|DL number)\b`), 0.8, "drivers license"},
		{regexp.MustCompile(`(?i)\b(passport|passport number)\b`), 0.8, "passport"},
		{regexp.MustCompile(`(?i)\b(employee|applicant|candidate)\s+(information|record|data)\b`), 0.7, "employee data"},
		{regexp.MustCompile(`(?i)\b(personal|private|sensitive)\s+(information|data)\b`), 0.8, "personal info"},
		{regexp.MustCompile(`(?i)\b(name|address|phone|email)\s*:`), 0.6, "contact fields"},
		{regexp.MustCompile(`(?i)\b(gender|race|ethnicity|nationality)\b`), 0.6, "demographics"},
		{regexp.MustCompile(`(?i)\b(emergency contact|next of kin)\b`), 0.7, "emergency contact"},
		{regexp.MustCompile(`\b\d{3}[-.\s]?\d{2}[-.\s]?\d{4}\b`), 0.7, "SSN pattern"},
	}

	// Technical document patterns
	dc.rulePatterns[DocTypeTechnical] = []documentPattern{
		{regexp.MustCompile(`(?i)\b(API|REST|GraphQL|endpoint)\b`), 0.7, "API"},
		{regexp.MustCompile(`(?i)\b(database|SQL|NoSQL|schema)\b`), 0.7, "database"},
		{regexp.MustCompile(`(?i)\b(server|client|request|response)\b`), 0.5, "client-server"},
		{regexp.MustCompile(`(?i)\b(function|class|method|variable)\b`), 0.6, "programming"},
		{regexp.MustCompile(`(?i)\b(configuration|config|settings|parameters)\b`), 0.6, "configuration"},
		{regexp.MustCompile(`(?i)\b(deployment|infrastructure|kubernetes|docker)\b`), 0.7, "deployment"},
		{regexp.MustCompile(`(?i)\b(authentication|authorization|OAuth|JWT)\b`), 0.7, "auth"},
		{regexp.MustCompile(`(?i)\b(encryption|SSL|TLS|certificate)\b`), 0.7, "security"},
		{regexp.MustCompile("```|<code>|</code>"), 0.8, "code blocks"},
		{regexp.MustCompile(`(?i)\b(architecture|system design|technical specification)\b`), 0.7, "design docs"},
	}
}

// Classify classifies a document and returns the type with confidence
func (dc *ONNXDocumentClassifier) Classify(ctx context.Context, text string) (*DocumentClassification, error) {
	// Run rule-based classification
	ruleResult := dc.classifyWithRules(text)

	// If ONNX model is loaded, combine with ML classification
	if dc.modelLoaded && dc.runtime != nil && dc.runtime.IsAvailable() {
		mlResult, err := dc.classifyWithONNX(ctx, text)
		if err != nil {
			dc.logger.Warn("ONNX classification failed, using rule-based result", "error", err)
			return ruleResult, nil
		}

		// Combine results (weighted average)
		combined := dc.combineResults(ruleResult, mlResult)
		return combined, nil
	}

	return ruleResult, nil
}

// classifyWithRules performs rule-based classification
func (dc *ONNXDocumentClassifier) classifyWithRules(text string) *DocumentClassification {
	scores := make(map[DocumentType]float64)
	indicators := make(map[DocumentType][]string)

	textLower := strings.ToLower(text)

	for docType, patterns := range dc.rulePatterns {
		for _, pattern := range patterns {
			matches := pattern.Pattern.FindAllString(textLower, -1)
			if len(matches) > 0 {
				// Score increases with more matches, but with diminishing returns
				matchScore := pattern.Weight * (1.0 + 0.1*float64(min(len(matches)-1, 5)))
				scores[docType] += matchScore
				indicators[docType] = append(indicators[docType], pattern.Description)
			}
		}
	}

	// Find best match
	var bestType DocumentType = DocTypeGeneral
	var bestScore float64

	for docType, score := range scores {
		if score > bestScore {
			bestScore = score
			bestType = docType
		}
	}

	// Normalize confidence (0-1 range)
	confidence := bestScore / 10.0 // Normalize assuming max practical score ~10
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.1 && bestType != DocTypeGeneral {
		bestType = DocTypeGeneral
		confidence = 0.5
	}

	// Get unique indicators
	uniqueIndicators := uniqueStrings(indicators[bestType])
	if len(uniqueIndicators) > 5 {
		uniqueIndicators = uniqueIndicators[:5]
	}

	return &DocumentClassification{
		Type:       string(bestType),
		Confidence: confidence,
		Indicators: uniqueIndicators,
	}
}

// classifyWithONNX performs ONNX model-based classification
func (dc *ONNXDocumentClassifier) classifyWithONNX(ctx context.Context, text string) (*DocumentClassification, error) {
	// Tokenize
	inputIDs, attentionMask := dc.tokenizer.Encode(text)

	// Prepare inputs
	inputs := []InferenceInput{
		{
			Name:  "input_ids",
			Data:  Int32ToFloat32(inputIDs),
			Shape: []int64{1, int64(len(inputIDs))},
		},
		{
			Name:  "attention_mask",
			Data:  Int32ToFloat32(attentionMask),
			Shape: []int64{1, int64(len(attentionMask))},
		},
	}

	// Run inference
	outputs, err := dc.runtime.RunInference(ctx, "doc_classifier", inputs)
	if err != nil {
		return nil, err
	}

	if len(outputs) == 0 {
		return &DocumentClassification{
			Type:       string(DocTypeGeneral),
			Confidence: 0.5,
		}, nil
	}

	// Process output logits
	logits := outputs[0].Data
	probs := Softmax(logits)

	// Get best prediction
	bestIdx := Argmax(probs)
	confidence := float64(probs[bestIdx])

	var docType DocumentType = DocTypeGeneral
	if bestIdx < len(dc.labels) {
		docType = dc.labels[bestIdx]
	}

	return &DocumentClassification{
		Type:       string(docType),
		Confidence: confidence,
	}, nil
}

// combineResults combines rule-based and ML results
func (dc *ONNXDocumentClassifier) combineResults(ruleResult, mlResult *DocumentClassification) *DocumentClassification {
	// Weight rule-based higher for specific patterns
	ruleWeight := 0.6
	mlWeight := 0.4

	// If results agree, boost confidence
	if ruleResult.Type == mlResult.Type {
		avgConfidence := ruleResult.Confidence*ruleWeight + mlResult.Confidence*mlWeight
		boostedConfidence := avgConfidence * 1.2
		if boostedConfidence > 1.0 {
			boostedConfidence = 1.0
		}
		return &DocumentClassification{
			Type:       ruleResult.Type,
			Confidence: boostedConfidence,
			Indicators: ruleResult.Indicators,
		}
	}

	// Results disagree - use the one with higher confidence
	ruleAdjusted := ruleResult.Confidence * ruleWeight
	mlAdjusted := mlResult.Confidence * mlWeight

	if ruleAdjusted >= mlAdjusted {
		return ruleResult
	}
	return mlResult
}

// ClassifyBatch classifies multiple documents
func (dc *ONNXDocumentClassifier) ClassifyBatch(ctx context.Context, texts []string) ([]*DocumentClassification, error) {
	results := make([]*DocumentClassification, len(texts))

	for i, text := range texts {
		result, err := dc.Classify(ctx, text)
		if err != nil {
			results[i] = &DocumentClassification{
				Type:       string(DocTypeGeneral),
				Confidence: 0.0,
			}
			continue
		}
		results[i] = result
	}

	return results, nil
}

// GetDocumentTypeDescription returns a description for a document type
func GetDocumentTypeDescription(docType DocumentType) string {
	descriptions := map[DocumentType]string{
		DocTypeMedical:   "Medical records, health information, clinical documents",
		DocTypeFinancial: "Financial statements, banking documents, tax records",
		DocTypeLegal:     "Legal contracts, agreements, litigation documents",
		DocTypePII:       "Documents containing personal identifiable information",
		DocTypeTechnical: "Technical documentation, specifications, code",
		DocTypeGeneral:   "General purpose documents",
	}
	if desc, ok := descriptions[docType]; ok {
		return desc
	}
	return "Unknown document type"
}

// GetSensitivityForDocType returns the typical sensitivity level for a document type
func GetSensitivityForDocType(docType DocumentType) string {
	sensitivities := map[DocumentType]string{
		DocTypeMedical:   "HIGH",
		DocTypeFinancial: "HIGH",
		DocTypeLegal:     "MEDIUM",
		DocTypePII:       "CRITICAL",
		DocTypeTechnical: "LOW",
		DocTypeGeneral:   "LOW",
	}
	if sens, ok := sensitivities[docType]; ok {
		return sens
	}
	return "LOW"
}

// IsModelLoaded returns whether the ONNX model is loaded
func (dc *ONNXDocumentClassifier) IsModelLoaded() bool {
	return dc.modelLoaded
}

// Helper functions

func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// ClassifyDocument implements the DocumentClassifier interface
func (dc *ONNXDocumentClassifier) ClassifyDocument(ctx context.Context, text string) (*DocumentClassification, error) {
	return dc.Classify(ctx, text)
}
