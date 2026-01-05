package classifier

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/qualys/dspm/internal/models"
)

type Rule struct {
	Name            string
	Category        models.Category
	Sensitivity     models.Sensitivity
	Patterns        []*regexp.Regexp
	ContextPatterns []*regexp.Regexp // Patterns that must appear nearby
	ContextRequired bool             // If true, requires context pattern match
	Validators      []Validator      // Additional validation functions
}

type Validator func(match string) bool

type Match struct {
	RuleName    string
	Category    models.Category
	Sensitivity models.Sensitivity
	Value       string // Redacted value
	Count       int
	LineNumbers []int
	Confidence  float64
}

type Result struct {
	Matches        []Match
	TotalFindings  int
	Categories     []models.Category
	MaxSensitivity models.Sensitivity
}

type Classifier struct {
	rules []*Rule
}

func New() *Classifier {
	return &Classifier{
		rules: DefaultRules(),
	}
}

func NewWithRules(rules []*Rule) *Classifier {
	return &Classifier{
		rules: rules,
	}
}

func (c *Classifier) AddRule(rule *Rule) {
	c.rules = append(c.rules, rule)
}

func (c *Classifier) Classify(content string) *Result {
	result := &Result{
		MaxSensitivity: models.SensitivityUnknown,
	}

	categorySet := make(map[models.Category]bool)
	lines := strings.Split(content, "\n")

	for _, rule := range c.rules {
		matches := c.findMatches(content, lines, rule)
		if len(matches) > 0 {
			match := Match{
				RuleName:    rule.Name,
				Category:    rule.Category,
				Sensitivity: rule.Sensitivity,
				Count:       len(matches),
				Confidence:  1.0, // Default confidence
			}

			for i, m := range matches {
				if i == 0 {
					match.Value = redact(m.value)
				}
				match.LineNumbers = append(match.LineNumbers, m.lineNum)
				if i >= 10 {
					break // Limit stored line numbers
				}
			}

			result.Matches = append(result.Matches, match)
			result.TotalFindings += match.Count
			categorySet[rule.Category] = true

			if compareSensitivity(rule.Sensitivity, result.MaxSensitivity) > 0 {
				result.MaxSensitivity = rule.Sensitivity
			}
		}
	}

	for cat := range categorySet {
		result.Categories = append(result.Categories, cat)
	}

	return result
}

type rawMatch struct {
	value   string
	lineNum int
}

func (c *Classifier) findMatches(content string, lines []string, rule *Rule) []rawMatch {
	var matches []rawMatch

	contextFound := !rule.ContextRequired
	if rule.ContextRequired && len(rule.ContextPatterns) > 0 {
		lowerContent := strings.ToLower(content)
		for _, cp := range rule.ContextPatterns {
			if cp.MatchString(lowerContent) {
				contextFound = true
				break
			}
		}
	}

	if !contextFound {
		return nil
	}

	for lineNum, line := range lines {
		for _, pattern := range rule.Patterns {
			found := pattern.FindAllString(line, -1)
			for _, match := range found {
				valid := true
				for _, validator := range rule.Validators {
					if !validator(match) {
						valid = false
						break
					}
				}

				if valid {
					matches = append(matches, rawMatch{
						value:   match,
						lineNum: lineNum + 1,
					})
				}
			}
		}
	}

	return matches
}

func DefaultRules() []*Rule {
	return []*Rule{
		{
			Name:        "SSN",
			Category:    models.CategoryPII,
			Sensitivity: models.SensitivityCritical,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
				regexp.MustCompile(`\b\d{3}\s\d{2}\s\d{4}\b`),
			},
			Validators: []Validator{ValidateSSN},
		},
		{
			Name:        "EMAIL",
			Category:    models.CategoryPII,
			Sensitivity: models.SensitivityMedium,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`),
			},
		},
		{
			Name:        "PHONE_US",
			Category:    models.CategoryPII,
			Sensitivity: models.SensitivityMedium,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b`),
				regexp.MustCompile(`\b\(\d{3}\)\s?\d{3}[-.\s]?\d{4}\b`),
			},
		},
		{
			Name:        "ADDRESS_US",
			Category:    models.CategoryPII,
			Sensitivity: models.SensitivityMedium,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)\b\d+\s+[A-Za-z]+\s+(Street|St|Avenue|Ave|Road|Rd|Boulevard|Blvd|Drive|Dr|Lane|Ln|Way|Court|Ct)\b`),
			},
		},
		{
			Name:        "DOB",
			Category:    models.CategoryPII,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b(0?[1-9]|1[0-2])[-/](0?[1-9]|[12]\d|3[01])[-/](19|20)\d{2}\b`),
				regexp.MustCompile(`\b(19|20)\d{2}[-/](0?[1-9]|1[0-2])[-/](0?[1-9]|[12]\d|3[01])\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(dob|birth|born|birthday)`),
			},
		},
		{
			Name:        "PASSPORT",
			Category:    models.CategoryPII,
			Sensitivity: models.SensitivityCritical,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b[A-Z]{1,2}\d{6,9}\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(passport|travel\s*document)`),
			},
			ContextRequired: true,
		},

		{
			Name:        "MRN",
			Category:    models.CategoryPHI,
			Sensitivity: models.SensitivityCritical,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)\bMRN[:\s#]?\d{6,10}\b`),
				regexp.MustCompile(`(?i)\bmedical\s*record[:\s#]?\d{6,10}\b`),
			},
		},
		{
			Name:        "ICD_CODE",
			Category:    models.CategoryPHI,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b[A-TV-Z]\d{2}(\.\d{1,4})?\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(diagnosis|icd|condition|dx)`),
			},
			ContextRequired: true,
		},
		{
			Name:        "NDC",
			Category:    models.CategoryPHI,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b\d{5}-\d{4}-\d{2}\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(ndc|drug|medication|prescription|rx)`),
			},
		},

		{
			Name:        "CREDIT_CARD",
			Category:    models.CategoryPCI,
			Sensitivity: models.SensitivityCritical,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b4\d{3}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`),
				regexp.MustCompile(`\b5[1-5]\d{2}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`),
				regexp.MustCompile(`\b2[2-7]\d{2}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`),
				regexp.MustCompile(`\b3[47]\d{2}[-\s]?\d{6}[-\s]?\d{5}\b`),
				regexp.MustCompile(`\b6(?:011|5\d{2})[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`),
			},
			Validators: []Validator{ValidateLuhn},
		},
		{
			Name:        "BANK_ACCOUNT",
			Category:    models.CategoryPCI,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b\d{8,17}\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(account\s*number|bank\s*account|checking|savings|acct)`),
			},
			ContextRequired: true,
		},
		{
			Name:        "ROUTING_NUMBER",
			Category:    models.CategoryPCI,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b\d{9}\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(routing|aba|transit)`),
			},
			ContextRequired: true,
			Validators:      []Validator{ValidateABARouting},
		},
		{
			Name:        "IBAN",
			Category:    models.CategoryPCI,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b[A-Z]{2}\d{2}[A-Z0-9]{4}\d{7}[A-Z0-9]{0,16}\b`),
			},
			Validators: []Validator{ValidateIBAN},
		},

		{
			Name:        "AWS_ACCESS_KEY",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityCritical,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
				regexp.MustCompile(`\bASIA[0-9A-Z]{16}\b`),
			},
		},
		{
			Name:        "AWS_SECRET_KEY",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityCritical,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b[A-Za-z0-9/+=]{40}\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(aws_secret|secret_access_key|secretaccesskey)`),
			},
			ContextRequired: true,
		},
		{
			Name:        "PRIVATE_KEY",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityCritical,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
				regexp.MustCompile(`-----BEGIN PGP PRIVATE KEY BLOCK-----`),
			},
		},
		{
			Name:        "JWT",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\beyJ[A-Za-z0-9-_]+\.eyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_.+/=]+\b`),
			},
		},
		{
			Name:        "GITHUB_TOKEN",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityCritical,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\bghp_[A-Za-z0-9]{36}\b`),
				regexp.MustCompile(`\bgho_[A-Za-z0-9]{36}\b`),
				regexp.MustCompile(`\bghu_[A-Za-z0-9]{36}\b`),
				regexp.MustCompile(`\bghs_[A-Za-z0-9]{36}\b`),
				regexp.MustCompile(`\bghr_[A-Za-z0-9]{36}\b`),
			},
		},
		{
			Name:        "SLACK_TOKEN",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\bxox[baprs]-[0-9A-Za-z-]{10,}\b`),
			},
		},
		{
			Name:        "GOOGLE_API_KEY",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\bAIza[0-9A-Za-z-_]{35}\b`),
			},
		},
		{
			Name:        "AZURE_CONNECTION_STRING",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityCritical,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`DefaultEndpointsProtocol=https?;AccountName=[^;]+;AccountKey=[A-Za-z0-9+/=]+`),
				regexp.MustCompile(`AccountKey=[A-Za-z0-9+/=]{86}==`),
			},
		},
		{
			Name:        "DB_CONNECTION_STRING",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityCritical,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b(mysql|postgresql|postgres|mongodb|redis)://[^:]+:[^@]+@`),
				regexp.MustCompile(`(?i)password\s*=\s*[^\s&;]+`),
			},
		},
		{
			Name:        "GENERIC_API_KEY",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b[a-zA-Z0-9]{32,64}\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(api[_-]?key|apikey|x-api-key|authorization\s*:\s*bearer)`),
			},
			ContextRequired: true,
		},
	}
}

func ValidateSSN(ssn string) bool {
	clean := strings.ReplaceAll(strings.ReplaceAll(ssn, "-", ""), " ", "")
	if len(clean) != 9 {
		return false
	}

	for _, c := range clean {
		if !unicode.IsDigit(c) {
			return false
		}
	}

	area := 0
	for i := 0; i < 3; i++ {
		area = area*10 + int(clean[i]-'0')
	}

	if area == 0 || area == 666 || area >= 900 {
		return false
	}

	group := int(clean[3]-'0')*10 + int(clean[4]-'0')
	if group == 0 {
		return false
	}

	serial := 0
	for i := 5; i < 9; i++ {
		serial = serial*10 + int(clean[i]-'0')
	}
	return serial != 0
}

func ValidateLuhn(number string) bool {
	var clean strings.Builder
	for _, c := range number {
		if unicode.IsDigit(c) {
			clean.WriteRune(c)
		}
	}
	digits := clean.String()

	if len(digits) < 13 || len(digits) > 19 {
		return false
	}

	sum := 0
	alternate := false

	for i := len(digits) - 1; i >= 0; i-- {
		n := int(digits[i] - '0')

		if alternate {
			n *= 2
			if n > 9 {
				n = n%10 + 1
			}
		}

		sum += n
		alternate = !alternate
	}

	return sum%10 == 0
}

func ValidateABARouting(routing string) bool {
	if len(routing) != 9 {
		return false
	}

	for _, c := range routing {
		if !unicode.IsDigit(c) {
			return false
		}
	}

	d := make([]int, 9)
	for i, c := range routing {
		d[i] = int(c - '0')
	}

	checksum := 3*(d[0]+d[3]+d[6]) + 7*(d[1]+d[4]+d[7]) + (d[2] + d[5] + d[8])
	return checksum%10 == 0
}

func ValidateIBAN(iban string) bool {
	clean := strings.ReplaceAll(strings.ToUpper(iban), " ", "")

	if len(clean) < 15 || len(clean) > 34 {
		return false
	}

	rearranged := clean[4:] + clean[:4]

	var numeric strings.Builder
	for _, c := range rearranged {
		if c >= 'A' && c <= 'Z' {
			numeric.WriteString(strconv.Itoa(int(c-'A') + 10))
		} else {
			numeric.WriteRune(c)
		}
	}

	remainder := 0
	for _, c := range numeric.String() {
		remainder = (remainder*10 + int(c-'0')) % 97
	}

	return remainder == 1
}

func redact(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

func compareSensitivity(a, b models.Sensitivity) int {
	order := map[models.Sensitivity]int{
		models.SensitivityCritical: 4,
		models.SensitivityHigh:     3,
		models.SensitivityMedium:   2,
		models.SensitivityLow:      1,
		models.SensitivityUnknown:  0,
	}
	return order[a] - order[b]
}
