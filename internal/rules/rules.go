package rules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/qualys/dspm/internal/models"
)

// CustomRule represents a user-defined classification rule
type CustomRule struct {
	ID              string             `json:"id" db:"id"`
	Name            string             `json:"name" db:"name"`
	Description     string             `json:"description" db:"description"`
	Category        models.Category    `json:"category" db:"category"`
	Sensitivity     models.Sensitivity `json:"sensitivity" db:"sensitivity"`
	Patterns        []string           `json:"patterns" db:"-"`
	ContextPatterns []string           `json:"context_patterns,omitempty" db:"-"`
	ContextRequired bool               `json:"context_required" db:"context_required"`
	Enabled         bool               `json:"enabled" db:"enabled"`
	Priority        int                `json:"priority" db:"priority"` // Higher priority runs first
	CreatedBy       string             `json:"created_by" db:"created_by"`
	CreatedAt       time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" db:"updated_at"`
}

// CompiledRule is a rule with compiled regex patterns
type CompiledRule struct {
	Rule            *CustomRule
	Patterns        []*regexp.Regexp
	ContextPatterns []*regexp.Regexp
}

// Store defines the interface for rule persistence
type Store interface {
	GetRule(ctx context.Context, id string) (*CustomRule, error)
	ListRules(ctx context.Context, enabledOnly bool) ([]*CustomRule, error)
	CreateRule(ctx context.Context, rule *CustomRule) error
	UpdateRule(ctx context.Context, rule *CustomRule) error
	DeleteRule(ctx context.Context, id string) error
	GetRulePatterns(ctx context.Context, ruleID string) ([]string, []string, error)
	SetRulePatterns(ctx context.Context, ruleID string, patterns, contextPatterns []string) error
}

// Engine manages custom rules and classification
type Engine struct {
	store         Store
	compiledRules []*CompiledRule
}

// NewEngine creates a new rules engine
func NewEngine(store Store) *Engine {
	return &Engine{
		store:         store,
		compiledRules: make([]*CompiledRule, 0),
	}
}

// LoadRules loads and compiles all enabled rules
func (e *Engine) LoadRules(ctx context.Context) error {
	rules, err := e.store.ListRules(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	e.compiledRules = make([]*CompiledRule, 0, len(rules))

	for _, rule := range rules {
		patterns, contextPatterns, err := e.store.GetRulePatterns(ctx, rule.ID)
		if err != nil {
			return fmt.Errorf("failed to load patterns for rule %s: %w", rule.ID, err)
		}
		rule.Patterns = patterns
		rule.ContextPatterns = contextPatterns

		compiled, err := e.compileRule(rule)
		if err != nil {
			return fmt.Errorf("failed to compile rule %s: %w", rule.ID, err)
		}
		e.compiledRules = append(e.compiledRules, compiled)
	}

	return nil
}

// compileRule compiles a rule's patterns
func (e *Engine) compileRule(rule *CustomRule) (*CompiledRule, error) {
	compiled := &CompiledRule{
		Rule:            rule,
		Patterns:        make([]*regexp.Regexp, 0, len(rule.Patterns)),
		ContextPatterns: make([]*regexp.Regexp, 0, len(rule.ContextPatterns)),
	}

	for _, p := range rule.Patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", p, err)
		}
		compiled.Patterns = append(compiled.Patterns, re)
	}

	for _, p := range rule.ContextPatterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid context pattern %q: %w", p, err)
		}
		compiled.ContextPatterns = append(compiled.ContextPatterns, re)
	}

	return compiled, nil
}

// Match represents a classification match
type Match struct {
	RuleID      string
	RuleName    string
	Category    models.Category
	Sensitivity models.Sensitivity
	Matches     []string
	Context     []string
	Confidence  float64
}

// Classify classifies content using custom rules
func (e *Engine) Classify(content string) []*Match {
	var matches []*Match

	for _, compiled := range e.compiledRules {
		match := e.matchRule(compiled, content)
		if match != nil {
			matches = append(matches, match)
		}
	}

	return matches
}

// matchRule checks if content matches a rule
func (e *Engine) matchRule(compiled *CompiledRule, content string) *Match {
	var foundMatches []string
	var contextMatches []string

	// Check main patterns
	for _, re := range compiled.Patterns {
		if matches := re.FindAllString(content, -1); len(matches) > 0 {
			foundMatches = append(foundMatches, matches...)
		}
	}

	if len(foundMatches) == 0 {
		return nil
	}

	// Check context patterns if required
	if compiled.Rule.ContextRequired && len(compiled.ContextPatterns) > 0 {
		hasContext := false
		for _, re := range compiled.ContextPatterns {
			if matches := re.FindAllString(content, -1); len(matches) > 0 {
				contextMatches = append(contextMatches, matches...)
				hasContext = true
			}
		}
		if !hasContext {
			return nil
		}
	}

	// Calculate confidence based on matches and context
	confidence := e.calculateConfidence(len(foundMatches), len(contextMatches), compiled.Rule.ContextRequired)

	return &Match{
		RuleID:      compiled.Rule.ID,
		RuleName:    compiled.Rule.Name,
		Category:    compiled.Rule.Category,
		Sensitivity: compiled.Rule.Sensitivity,
		Matches:     foundMatches,
		Context:     contextMatches,
		Confidence:  confidence,
	}
}

// calculateConfidence calculates match confidence
func (e *Engine) calculateConfidence(matchCount, contextCount int, contextRequired bool) float64 {
	base := 0.5
	if matchCount > 1 {
		base += 0.1 * float64(min(matchCount-1, 5))
	}
	if contextCount > 0 {
		base += 0.2
	}
	if contextRequired && contextCount > 0 {
		base += 0.1
	}
	return min(base, 1.0)
}

// CreateRule creates a new custom rule
func (e *Engine) CreateRule(ctx context.Context, rule *CustomRule) error {
	// Validate patterns
	for _, p := range rule.Patterns {
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("invalid pattern %q: %w", p, err)
		}
	}
	for _, p := range rule.ContextPatterns {
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("invalid context pattern %q: %w", p, err)
		}
	}

	if err := e.store.CreateRule(ctx, rule); err != nil {
		return err
	}

	if err := e.store.SetRulePatterns(ctx, rule.ID, rule.Patterns, rule.ContextPatterns); err != nil {
		return err
	}

	// Reload rules if enabled
	if rule.Enabled {
		return e.LoadRules(ctx)
	}

	return nil
}

// UpdateRule updates an existing rule
func (e *Engine) UpdateRule(ctx context.Context, rule *CustomRule) error {
	// Validate patterns
	for _, p := range rule.Patterns {
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("invalid pattern %q: %w", p, err)
		}
	}
	for _, p := range rule.ContextPatterns {
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("invalid context pattern %q: %w", p, err)
		}
	}

	if err := e.store.UpdateRule(ctx, rule); err != nil {
		return err
	}

	if err := e.store.SetRulePatterns(ctx, rule.ID, rule.Patterns, rule.ContextPatterns); err != nil {
		return err
	}

	return e.LoadRules(ctx)
}

// DeleteRule deletes a rule
func (e *Engine) DeleteRule(ctx context.Context, id string) error {
	if err := e.store.DeleteRule(ctx, id); err != nil {
		return err
	}
	return e.LoadRules(ctx)
}

// EnableRule enables a rule
func (e *Engine) EnableRule(ctx context.Context, id string) error {
	rule, err := e.store.GetRule(ctx, id)
	if err != nil {
		return err
	}
	rule.Enabled = true
	if err := e.store.UpdateRule(ctx, rule); err != nil {
		return err
	}
	return e.LoadRules(ctx)
}

// DisableRule disables a rule
func (e *Engine) DisableRule(ctx context.Context, id string) error {
	rule, err := e.store.GetRule(ctx, id)
	if err != nil {
		return err
	}
	rule.Enabled = false
	if err := e.store.UpdateRule(ctx, rule); err != nil {
		return err
	}
	return e.LoadRules(ctx)
}

// TestRule tests a rule against sample content
func (e *Engine) TestRule(ctx context.Context, rule *CustomRule, content string) (*Match, error) {
	compiled, err := e.compileRule(rule)
	if err != nil {
		return nil, err
	}
	return e.matchRule(compiled, content), nil
}

// GetRules returns all rules
func (e *Engine) GetRules(ctx context.Context) ([]*CustomRule, error) {
	rules, err := e.store.ListRules(ctx, false)
	if err != nil {
		return nil, err
	}

	// Load patterns for each rule
	for _, rule := range rules {
		patterns, contextPatterns, err := e.store.GetRulePatterns(ctx, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Patterns = patterns
		rule.ContextPatterns = contextPatterns
	}

	return rules, nil
}

// GetRule returns a single rule by ID
func (e *Engine) GetRule(ctx context.Context, id string) (*CustomRule, error) {
	rule, err := e.store.GetRule(ctx, id)
	if err != nil {
		return nil, err
	}

	patterns, contextPatterns, err := e.store.GetRulePatterns(ctx, rule.ID)
	if err != nil {
		return nil, err
	}
	rule.Patterns = patterns
	rule.ContextPatterns = contextPatterns

	return rule, nil
}

// ValidatePattern validates a regex pattern
func ValidatePattern(pattern string) error {
	if pattern == "" {
		return errors.New("pattern cannot be empty")
	}
	_, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex: %w", err)
	}
	return nil
}

// PredefinedRules returns built-in classification rules
func PredefinedRules() []*CustomRule {
	return []*CustomRule{
		{
			Name:        "Social Security Numbers",
			Description: "Detects US Social Security Numbers",
			Category:    models.CategoryPII,
			Sensitivity: models.SensitivityHigh,
			Patterns:    []string{`\b\d{3}-\d{2}-\d{4}\b`, `\b\d{9}\b`},
			ContextPatterns: []string{`(?i)ssn|social\s*security|tax\s*id`},
			ContextRequired: true,
			Priority:    100,
		},
		{
			Name:        "Credit Card Numbers",
			Description: "Detects credit card numbers (Visa, MasterCard, Amex, etc.)",
			Category:    models.CategoryPCI,
			Sensitivity: models.SensitivityCritical,
			Patterns:    []string{`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b`},
			Priority:    100,
		},
		{
			Name:        "Email Addresses",
			Description: "Detects email addresses",
			Category:    models.CategoryPII,
			Sensitivity: models.SensitivityMedium,
			Patterns:    []string{`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`},
			Priority:    50,
		},
		{
			Name:        "AWS Access Keys",
			Description: "Detects AWS access key IDs",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityCritical,
			Patterns:    []string{`\b(?:AKIA|ABIA|ACCA|ASIA)[A-Z0-9]{16}\b`},
			Priority:    100,
		},
		{
			Name:        "Private Keys",
			Description: "Detects RSA/EC private keys",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityCritical,
			Patterns:    []string{`-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`},
			Priority:    100,
		},
		{
			Name:        "Medical Record Numbers",
			Description: "Detects medical record numbers with context",
			Category:    models.CategoryPHI,
			Sensitivity: models.SensitivityHigh,
			Patterns:    []string{`\b[A-Z]{2,3}[0-9]{6,10}\b`},
			ContextPatterns: []string{`(?i)medical|patient|mrn|health|hospital|clinic|diagnosis`},
			ContextRequired: true,
			Priority:    80,
		},
		{
			Name:        "IP Addresses",
			Description: "Detects IPv4 addresses",
			Category:    models.CategoryPII,
			Sensitivity: models.SensitivityLow,
			Patterns:    []string{`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`},
			Priority:    30,
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RuleTemplate is a pre-made rule configuration
type RuleTemplate struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Patterns    []string `json:"patterns"`
	Hint        string   `json:"hint"`
}

// GetTemplates returns available rule templates
func GetTemplates() []RuleTemplate {
	return []RuleTemplate{
		{
			Name:        "Phone Numbers",
			Description: "US phone number formats",
			Patterns:    []string{`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`, `\(\d{3}\)\s?\d{3}[-.]?\d{4}`},
			Hint:        "Matches (555) 123-4567, 555-123-4567, 5551234567",
		},
		{
			Name:        "Date of Birth",
			Description: "Common date formats",
			Patterns:    []string{`\b\d{1,2}[/-]\d{1,2}[/-]\d{2,4}\b`, `\b\d{4}[/-]\d{2}[/-]\d{2}\b`},
			Hint:        "Matches 01/15/1990, 1990-01-15",
		},
		{
			Name:        "Passport Numbers",
			Description: "US passport format",
			Patterns:    []string{`\b[A-Z]{1,2}[0-9]{6,9}\b`},
			Hint:        "Use with context patterns for accuracy",
		},
		{
			Name:        "Bank Account Numbers",
			Description: "Bank routing and account numbers",
			Patterns:    []string{`\b\d{9,17}\b`},
			Hint:        "Add context patterns like 'account', 'routing' for better accuracy",
		},
		{
			Name:        "JWT Tokens",
			Description: "JSON Web Tokens",
			Patterns:    []string{`eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*`},
			Hint:        "Detects Bearer tokens in format header.payload.signature",
		},
		{
			Name:        "API Keys (Generic)",
			Description: "Generic API key patterns",
			Patterns:    []string{`(?i)api[_-]?key['\"]?\s*[:=]\s*['\"]?[a-z0-9]{20,}`, `(?i)secret[_-]?key['\"]?\s*[:=]\s*['\"]?[a-z0-9]{20,}`},
			Hint:        "Catches common API key assignment patterns",
		},
	}
}

// MarshalPatterns converts patterns to JSON for storage
func MarshalPatterns(patterns []string) ([]byte, error) {
	return json.Marshal(patterns)
}

// UnmarshalPatterns converts JSON to patterns
func UnmarshalPatterns(data []byte) ([]string, error) {
	var patterns []string
	if err := json.Unmarshal(data, &patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}
