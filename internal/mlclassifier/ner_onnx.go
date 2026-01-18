package mlclassifier

import (
	"context"
	"log/slog"
	"strings"
)

// ONNXEntityRecognizer implements NER using ONNX models
type ONNXEntityRecognizer struct {
	runtime     *ONNXRuntime
	tokenizer   *Tokenizer
	ruleBasedNER *RuleBasedNER
	logger      *slog.Logger
	labels      []string
	modelLoaded bool
}

// ONNXNERConfig contains configuration for ONNX NER
type ONNXNERConfig struct {
	ModelPath    string
	VocabPath    string
	MergesPath   string
	MaxSeqLength int
	Labels       []string
}

// DefaultONNXNERConfig returns default configuration
func DefaultONNXNERConfig() ONNXNERConfig {
	return ONNXNERConfig{
		MaxSeqLength: 512,
		Labels: []string{
			"O",           // Outside any entity
			"B-PERSON",    // Beginning of person name
			"I-PERSON",    // Inside person name
			"B-ORG",       // Beginning of organization
			"I-ORG",       // Inside organization
			"B-LOC",       // Beginning of location
			"I-LOC",       // Inside location
			"B-DATE",      // Beginning of date
			"I-DATE",      // Inside date
			"B-PII",       // Beginning of PII
			"I-PII",       // Inside PII
			"B-MEDICAL",   // Beginning of medical term
			"I-MEDICAL",   // Inside medical term
			"B-FINANCIAL", // Beginning of financial term
			"I-FINANCIAL", // Inside financial term
		},
	}
}

// NewONNXEntityRecognizer creates a new ONNX-based entity recognizer
func NewONNXEntityRecognizer(config ONNXNERConfig, runtime *ONNXRuntime, logger *slog.Logger) (*ONNXEntityRecognizer, error) {
	ner := &ONNXEntityRecognizer{
		runtime:      runtime,
		ruleBasedNER: NewRuleBasedNER(),
		logger:       logger,
		labels:       config.Labels,
		modelLoaded:  false,
	}

	if len(ner.labels) == 0 {
		ner.labels = DefaultONNXNERConfig().Labels
	}

	// Initialize tokenizer
	tokConfig := TokenizerConfig{
		VocabFile:  config.VocabPath,
		MergesFile: config.MergesPath,
		MaxLength:  config.MaxSeqLength,
	}

	var err error
	ner.tokenizer, err = NewTokenizer(tokConfig)
	if err != nil {
		logger.Warn("failed to initialize tokenizer, using basic tokenizer", "error", err)
	}

	// Load model if runtime is available and path provided
	if runtime != nil && runtime.IsAvailable() && config.ModelPath != "" {
		if err := runtime.LoadModel("ner", config.ModelPath); err != nil {
			logger.Warn("failed to load NER model, using rule-based fallback", "error", err)
		} else {
			ner.modelLoaded = true
		}
	}

	return ner, nil
}

// RecognizeEntities extracts entities from text using ONNX model with rule-based fallback
func (n *ONNXEntityRecognizer) RecognizeEntities(ctx context.Context, text string) ([]Entity, error) {
	// Always run rule-based NER for high-precision patterns
	ruleEntities, err := n.ruleBasedNER.RecognizeEntities(ctx, text)
	if err != nil {
		return nil, err
	}

	// If ONNX model is not available, return rule-based results
	if !n.modelLoaded || n.runtime == nil || !n.runtime.IsAvailable() {
		return ruleEntities, nil
	}

	// Run ONNX NER
	onnxEntities, err := n.runONNXInference(ctx, text)
	if err != nil {
		n.logger.Warn("ONNX inference failed, using rule-based results", "error", err)
		return ruleEntities, nil
	}

	// Merge results, preferring rule-based for overlapping entities
	merged := n.mergeEntities(ruleEntities, onnxEntities)

	return merged, nil
}

// runONNXInference runs the ONNX NER model
func (n *ONNXEntityRecognizer) runONNXInference(ctx context.Context, text string) ([]Entity, error) {
	// Tokenize text
	inputIDs, attentionMask := n.tokenizer.Encode(text)

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
	outputs, err := n.runtime.RunInference(ctx, "ner", inputs)
	if err != nil {
		return nil, err
	}

	if len(outputs) == 0 {
		return nil, nil
	}

	// Parse NER output
	entities := n.parseNEROutput(text, outputs[0].Data, outputs[0].Shape)

	return entities, nil
}

// parseNEROutput converts ONNX NER output to entities
func (n *ONNXEntityRecognizer) parseNEROutput(text string, logits []float32, shape []int64) []Entity {
	var entities []Entity

	if len(shape) < 3 {
		return entities
	}

	seqLen := int(shape[1])
	numLabels := int(shape[2])

	if len(logits) != seqLen*numLabels {
		return entities
	}

	// Track current entity being built
	var currentEntity *struct {
		entityType string
		tokens     []int
		startIdx   int
	}

	// Process each position
	for pos := 0; pos < seqLen; pos++ {
		// Get logits for this position
		posLogits := logits[pos*numLabels : (pos+1)*numLabels]

		// Apply softmax and get prediction
		probs := Softmax(posLogits)
		predLabel := Argmax(probs)
		confidence := float64(probs[predLabel])

		// Skip low confidence predictions
		if confidence < 0.5 {
			predLabel = 0 // O tag
		}

		// Get label name
		labelName := "O"
		if predLabel < len(n.labels) {
			labelName = n.labels[predLabel]
		}

		// Process BIO tags
		if labelName == "O" {
			// End current entity if any
			if currentEntity != nil {
				entity := n.buildEntity(text, currentEntity.entityType, currentEntity.startIdx, pos, confidence)
				if entity != nil {
					entities = append(entities, *entity)
				}
				currentEntity = nil
			}
		} else if strings.HasPrefix(labelName, "B-") {
			// Begin new entity
			if currentEntity != nil {
				entity := n.buildEntity(text, currentEntity.entityType, currentEntity.startIdx, pos, confidence)
				if entity != nil {
					entities = append(entities, *entity)
				}
			}
			entityType := strings.TrimPrefix(labelName, "B-")
			currentEntity = &struct {
				entityType string
				tokens     []int
				startIdx   int
			}{
				entityType: entityType,
				tokens:     []int{pos},
				startIdx:   pos,
			}
		} else if strings.HasPrefix(labelName, "I-") {
			// Continue current entity
			if currentEntity != nil {
				expectedType := strings.TrimPrefix(labelName, "I-")
				if expectedType == currentEntity.entityType {
					currentEntity.tokens = append(currentEntity.tokens, pos)
				} else {
					// Type mismatch, end current and start new
					entity := n.buildEntity(text, currentEntity.entityType, currentEntity.startIdx, pos, confidence)
					if entity != nil {
						entities = append(entities, *entity)
					}
					currentEntity = &struct {
						entityType string
						tokens     []int
						startIdx   int
					}{
						entityType: expectedType,
						tokens:     []int{pos},
						startIdx:   pos,
					}
				}
			} else {
				// I- without B-, treat as B-
				entityType := strings.TrimPrefix(labelName, "I-")
				currentEntity = &struct {
					entityType string
					tokens     []int
					startIdx   int
				}{
					entityType: entityType,
					tokens:     []int{pos},
					startIdx:   pos,
				}
			}
		}
	}

	// Handle final entity
	if currentEntity != nil {
		entity := n.buildEntity(text, currentEntity.entityType, currentEntity.startIdx, seqLen, 0.5)
		if entity != nil {
			entities = append(entities, *entity)
		}
	}

	return entities
}

// buildEntity creates an entity from token positions
func (n *ONNXEntityRecognizer) buildEntity(text, entityType string, startToken, endToken int, confidence float64) *Entity {
	// Map token positions to character positions
	// This is a simplified mapping; in production, use the tokenizer's offset mapping
	words := strings.Fields(text)
	if startToken >= len(words) {
		return nil
	}

	// Find character offsets
	charPos := 0
	startChar := 0
	endChar := len(text)

	for i, word := range words {
		wordStart := strings.Index(text[charPos:], word)
		if wordStart == -1 {
			break
		}
		wordStart += charPos

		if i == startToken {
			startChar = wordStart
		}
		if i == endToken-1 || i == len(words)-1 {
			endChar = wordStart + len(word)
		}

		charPos = wordStart + len(word)
	}

	if startChar >= endChar || startChar >= len(text) {
		return nil
	}

	if endChar > len(text) {
		endChar = len(text)
	}

	return &Entity{
		Text:        text[startChar:endChar],
		Type:        n.mapEntityType(entityType),
		StartOffset: startChar,
		EndOffset:   endChar,
		Confidence:  confidence,
	}
}

// mapEntityType maps ONNX label to standard entity type
func (n *ONNXEntityRecognizer) mapEntityType(labelType string) string {
	switch labelType {
	case "PERSON":
		return "PERSON"
	case "ORG":
		return "ORGANIZATION"
	case "LOC":
		return "LOCATION"
	case "DATE":
		return "DATE"
	case "PII":
		return "PII"
	case "MEDICAL":
		return "MEDICAL_TERM"
	case "FINANCIAL":
		return "FINANCIAL_TERM"
	default:
		return labelType
	}
}

// mergeEntities combines rule-based and ONNX entities
func (n *ONNXEntityRecognizer) mergeEntities(ruleEntities, onnxEntities []Entity) []Entity {
	// Create a map of character positions covered by rule entities
	ruleCovered := make(map[int]bool)
	for _, e := range ruleEntities {
		for i := e.StartOffset; i < e.EndOffset; i++ {
			ruleCovered[i] = true
		}
	}

	// Start with rule entities (higher precision)
	merged := make([]Entity, len(ruleEntities))
	copy(merged, ruleEntities)

	// Add ONNX entities that don't overlap with rule entities
	for _, e := range onnxEntities {
		overlaps := false
		for i := e.StartOffset; i < e.EndOffset; i++ {
			if ruleCovered[i] {
				overlaps = true
				break
			}
		}

		if !overlaps {
			merged = append(merged, e)
		}
	}

	return merged
}

// ExtractPIIEntities extracts PII-specific entities
func (n *ONNXEntityRecognizer) ExtractPIIEntities(ctx context.Context, text string) ([]Entity, error) {
	entities, err := n.RecognizeEntities(ctx, text)
	if err != nil {
		return nil, err
	}

	// Filter to PII-relevant entity types
	piiTypes := map[string]bool{
		"PERSON":         true,
		"EMAIL":          true,
		"PHONE":          true,
		"SSN":            true,
		"CREDIT_CARD":    true,
		"ADDRESS":        true,
		"DATE":           true,
		"IP_ADDRESS":     true,
		"AWS_ACCESS_KEY": true,
		"API_KEY":        true,
		"PII":            true,
	}

	var piiEntities []Entity
	for _, e := range entities {
		if piiTypes[e.Type] {
			piiEntities = append(piiEntities, e)
		}
	}

	return piiEntities, nil
}

// GetEntityConfidence returns confidence-boosted score based on context
func (n *ONNXEntityRecognizer) GetEntityConfidence(entity Entity, context string) float64 {
	baseConfidence := entity.Confidence

	// Boost confidence based on surrounding context
	contextLower := strings.ToLower(context)

	// PII-indicating keywords
	piiKeywords := map[string][]string{
		"PERSON":       {"name", "person", "contact", "employee", "patient"},
		"EMAIL":        {"email", "e-mail", "contact", "address"},
		"PHONE":        {"phone", "tel", "mobile", "cell", "contact"},
		"SSN":          {"ssn", "social security", "social", "security number"},
		"CREDIT_CARD":  {"card", "credit", "payment", "visa", "mastercard"},
		"ADDRESS":      {"address", "street", "city", "state", "zip"},
		"DATE":         {"date", "born", "birth", "dob"},
		"MEDICAL_TERM": {"diagnosis", "treatment", "condition", "medical", "patient"},
	}

	if keywords, ok := piiKeywords[entity.Type]; ok {
		for _, keyword := range keywords {
			if strings.Contains(contextLower, keyword) {
				baseConfidence += 0.1
			}
		}
	}

	// Cap at 1.0
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	return baseConfidence
}

// IsModelLoaded returns whether the ONNX model is loaded
func (n *ONNXEntityRecognizer) IsModelLoaded() bool {
	return n.modelLoaded
}
