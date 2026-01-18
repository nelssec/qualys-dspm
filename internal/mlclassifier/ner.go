package mlclassifier

import (
	"context"
	"regexp"
	"strings"
)

// RuleBasedNER implements EntityRecognizer using pattern matching
// This serves as a baseline until ML-based NER is integrated
type RuleBasedNER struct {
	patterns map[string]*EntityPattern
}

// EntityPattern defines a pattern for entity recognition
type EntityPattern struct {
	Regex       *regexp.Regexp
	EntityType  string
	Confidence  float64
	Validators  []func(string) bool
	Description string
}

// NewRuleBasedNER creates a new rule-based NER with default patterns
func NewRuleBasedNER() *RuleBasedNER {
	ner := &RuleBasedNER{
		patterns: make(map[string]*EntityPattern),
	}

	// Initialize with common PII patterns
	ner.initializePatterns()

	return ner
}

// initializePatterns sets up the default entity patterns
func (n *RuleBasedNER) initializePatterns() {
	// Email addresses
	n.patterns["EMAIL"] = &EntityPattern{
		Regex:       regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
		EntityType:  "EMAIL",
		Confidence:  0.95,
		Description: "Email address",
	}

	// Phone numbers (various formats) - matches at word boundary or start of parenthesis group
	n.patterns["PHONE"] = &EntityPattern{
		Regex:       regexp.MustCompile(`(?:\b\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}\b`),
		EntityType:  "PHONE",
		Confidence:  0.85,
		Description: "Phone number",
		Validators:  []func(string) bool{validatePhoneContext},
	}

	// SSN (US Social Security Number) - uses validator for detailed checks
	n.patterns["SSN"] = &EntityPattern{
		Regex:       regexp.MustCompile(`\b\d{3}[-\s]?\d{2}[-\s]?\d{4}\b`),
		EntityType:  "SSN",
		Confidence:  0.90,
		Validators:  []func(string) bool{validateSSN},
		Description: "US Social Security Number",
	}

	// Credit card numbers
	n.patterns["CREDIT_CARD"] = &EntityPattern{
		Regex:       regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b`),
		EntityType:  "CREDIT_CARD",
		Confidence:  0.85,
		Validators:  []func(string) bool{validateLuhn},
		Description: "Credit card number",
	}

	// IP addresses (IPv4)
	n.patterns["IP_ADDRESS"] = &EntityPattern{
		Regex:       regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`),
		EntityType:  "IP_ADDRESS",
		Confidence:  0.95,
		Description: "IPv4 address",
	}

	// Date patterns (various formats)
	n.patterns["DATE"] = &EntityPattern{
		Regex:       regexp.MustCompile(`\b(?:\d{1,2}[-/]\d{1,2}[-/]\d{2,4}|\d{4}[-/]\d{1,2}[-/]\d{1,2}|\w+\s+\d{1,2},?\s+\d{4})\b`),
		EntityType:  "DATE",
		Confidence:  0.80,
		Description: "Date",
	}

	// Person names (simplified - looks for title + capitalized words)
	n.patterns["PERSON"] = &EntityPattern{
		Regex:       regexp.MustCompile(`\b(?:Mr\.|Mrs\.|Ms\.|Dr\.|Prof\.)\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\b`),
		EntityType:  "PERSON",
		Confidence:  0.70,
		Description: "Person name with title",
	}

	// AWS ARN
	n.patterns["AWS_ARN"] = &EntityPattern{
		Regex:       regexp.MustCompile(`\barn:aws:[a-z0-9-]+:[a-z0-9-]*:[0-9]*:[a-z0-9-_/:.]+\b`),
		EntityType:  "AWS_ARN",
		Confidence:  0.95,
		Description: "AWS Resource ARN",
	}

	// AWS Access Key ID
	n.patterns["AWS_ACCESS_KEY"] = &EntityPattern{
		Regex:       regexp.MustCompile(`\b(?:AKIA|ABIA|ACCA|ASIA)[0-9A-Z]{16}\b`),
		EntityType:  "AWS_ACCESS_KEY",
		Confidence:  0.95,
		Description: "AWS Access Key ID",
	}

	// API Key (generic pattern)
	n.patterns["API_KEY"] = &EntityPattern{
		Regex:       regexp.MustCompile(`\b(?:api[_-]?key|apikey)[=:]\s*['"]?([a-zA-Z0-9_-]{20,})['"]?\b`),
		EntityType:  "API_KEY",
		Confidence:  0.80,
		Description: "API Key",
	}

	// UUID
	n.patterns["UUID"] = &EntityPattern{
		Regex:       regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`),
		EntityType:  "UUID",
		Confidence:  0.95,
		Description: "UUID",
	}

	// IBAN
	n.patterns["IBAN"] = &EntityPattern{
		Regex:       regexp.MustCompile(`\b[A-Z]{2}[0-9]{2}[A-Z0-9]{4,30}\b`),
		EntityType:  "IBAN",
		Confidence:  0.85,
		Validators:  []func(string) bool{validateIBAN},
		Description: "International Bank Account Number",
	}
}

// RecognizeEntities extracts named entities from text
func (n *RuleBasedNER) RecognizeEntities(ctx context.Context, text string) ([]Entity, error) {
	var entities []Entity

	for _, pattern := range n.patterns {
		matches := pattern.Regex.FindAllStringIndex(text, -1)

		for _, match := range matches {
			value := text[match[0]:match[1]]

			// Apply validators if present
			valid := true
			if len(pattern.Validators) > 0 {
				for _, validator := range pattern.Validators {
					if !validator(value) {
						valid = false
						break
					}
				}
			}

			if valid {
				entities = append(entities, Entity{
					Text:        value,
					Type:        pattern.EntityType,
					StartOffset: match[0],
					EndOffset:   match[1],
					Confidence:  pattern.Confidence,
				})
			}
		}
	}

	// Remove duplicates and overlapping entities
	entities = n.deduplicateEntities(entities)

	return entities, nil
}

// deduplicateEntities removes overlapping entities, keeping highest confidence
func (n *RuleBasedNER) deduplicateEntities(entities []Entity) []Entity {
	if len(entities) == 0 {
		return entities
	}

	// First pass: remove exact duplicates
	seen := make(map[string]bool)
	var unique []Entity

	for _, e := range entities {
		key := e.Text + e.Type
		if !seen[key] {
			seen[key] = true
			unique = append(unique, e)
		}
	}

	// Second pass: remove entities contained within other higher-confidence entities
	var result []Entity
	for i, e := range unique {
		isContained := false
		for j, other := range unique {
			if i == j {
				continue
			}
			// Check if entity e is contained within entity other
			if e.StartOffset >= other.StartOffset && e.EndOffset <= other.EndOffset {
				// e is contained within other
				if other.Confidence >= e.Confidence {
					isContained = true
					break
				}
			}
		}
		if !isContained {
			result = append(result, e)
		}
	}

	return result
}

// AddPattern adds a custom pattern to the NER
func (n *RuleBasedNER) AddPattern(name string, pattern *EntityPattern) {
	n.patterns[name] = pattern
}

// Validator functions

func validateSSN(ssn string) bool {
	// Remove separators
	clean := strings.ReplaceAll(strings.ReplaceAll(ssn, "-", ""), " ", "")
	if len(clean) != 9 {
		return false
	}

	// Check area number (first 3 digits)
	area := clean[:3]
	if area == "000" || area == "666" || area[0] == '9' {
		return false
	}

	// Check group number (middle 2 digits)
	group := clean[3:5]
	if group == "00" {
		return false
	}

	// Check serial number (last 4 digits)
	serial := clean[5:]
	if serial == "0000" {
		return false
	}

	return true
}

func validateLuhn(cardNumber string) bool {
	// Remove any spaces or dashes
	clean := strings.ReplaceAll(strings.ReplaceAll(cardNumber, " ", ""), "-", "")

	if len(clean) < 13 || len(clean) > 19 {
		return false
	}

	sum := 0
	alternate := false

	for i := len(clean) - 1; i >= 0; i-- {
		n := int(clean[i] - '0')
		if n < 0 || n > 9 {
			return false
		}

		if alternate {
			n *= 2
			if n > 9 {
				n = (n % 10) + 1
			}
		}
		sum += n
		alternate = !alternate
	}

	return sum%10 == 0
}

func validateIBAN(iban string) bool {
	// Basic IBAN validation
	clean := strings.ReplaceAll(strings.ToUpper(iban), " ", "")

	if len(clean) < 15 || len(clean) > 34 {
		return false
	}

	// Check country code (first 2 chars should be letters)
	if !isLetter(rune(clean[0])) || !isLetter(rune(clean[1])) {
		return false
	}

	// Check digits (chars 3-4 should be numbers)
	if !isDigit(rune(clean[2])) || !isDigit(rune(clean[3])) {
		return false
	}

	return true
}

func isLetter(c rune) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

func validatePhoneContext(phone string) bool {
	// Phone numbers should have reasonable digit count (10-11 digits for US)
	digitCount := 0
	digitFreq := make(map[rune]int)
	for _, c := range phone {
		if c >= '0' && c <= '9' {
			digitCount++
			digitFreq[c]++
		}
	}

	// US phone numbers have 10 digits (or 11 with country code)
	if digitCount < 10 || digitCount > 11 {
		return false
	}

	// Phone numbers should have some variety (not dominated by one digit)
	// This filters out patterns like "1111111112" which are unlikely to be real phone numbers
	maxFreq := 0
	for _, freq := range digitFreq {
		if freq > maxFreq {
			maxFreq = freq
		}
	}
	// If more than 70% of digits are the same, likely not a real phone number
	if digitCount > 0 && float64(maxFreq)/float64(digitCount) > 0.7 {
		return false
	}

	return true
}
