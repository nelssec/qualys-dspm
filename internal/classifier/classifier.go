package classifier

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/qualys/dspm/internal/models"
)

type Rule struct {
	Name             string
	Category         models.Category
	Sensitivity      models.Sensitivity
	Patterns         []*regexp.Regexp
	ContextPatterns  []*regexp.Regexp // Patterns that must appear nearby
	NegativePatterns []*regexp.Regexp // Patterns that should NOT appear nearby (exclusions)
	ContextRequired  bool             // If true, requires context pattern match
	ContextDistance  int              // Max chars from match to check for context (0 = whole file)
	Validators       []Validator      // Additional validation functions
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
	// Enhanced match details for remediation
	SampleMatches []SampleMatch // Up to 5 sample matches with context
	ColumnName    string        // Column/field name if detected (for CSV/JSON)
}

// SampleMatch represents a single match with its context
type SampleMatch struct {
	LineNumber   int    // 1-based line number
	ColumnNumber int    // 1-based column position
	ColumnName   string // Header/field name if available
	MaskedValue  string // The matched value, masked for security
	Context      string // Surrounding text (also masked)
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

			// Build sample matches with full context (up to 5)
			for i, m := range matches {
				if i == 0 {
					match.Value = redact(m.value)
					if m.colName != "" {
						match.ColumnName = m.colName
					}
				}
				match.LineNumbers = append(match.LineNumbers, m.lineNum)

				// Add sample match with context (limit to 5)
				if i < 5 {
					sample := SampleMatch{
						LineNumber:   m.lineNum,
						ColumnNumber: m.colNum,
						ColumnName:   m.colName,
						MaskedValue:  redact(m.value),
						Context:      redactContext(m.context, m.value),
					}
					match.SampleMatches = append(match.SampleMatches, sample)
				}

				if len(match.LineNumbers) >= 10 {
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
	value      string
	lineNum    int
	colNum     int    // 1-based column position
	colName    string // Column header if CSV
	context    string // Surrounding text
	lineText   string // Full line for context extraction
}

func (c *Classifier) findMatches(content string, lines []string, rule *Rule) []rawMatch {
	var matches []rawMatch

	// For global context check (ContextDistance == 0), check entire content
	globalContextFound := !rule.ContextRequired
	if rule.ContextRequired && rule.ContextDistance == 0 && len(rule.ContextPatterns) > 0 {
		lowerContent := strings.ToLower(content)
		for _, cp := range rule.ContextPatterns {
			if cp.MatchString(lowerContent) {
				globalContextFound = true
				break
			}
		}
	}

	if !globalContextFound && rule.ContextDistance == 0 {
		return nil
	}

	// Try to detect CSV headers from first line
	var csvHeaders []string
	if len(lines) > 0 && strings.Contains(lines[0], ",") {
		csvHeaders = strings.Split(lines[0], ",")
		for i := range csvHeaders {
			csvHeaders[i] = strings.TrimSpace(csvHeaders[i])
		}
	}

	// Pre-calculate line offsets for local context checking
	lineOffsets := make([]int, len(lines)+1)
	offset := 0
	for i, line := range lines {
		lineOffsets[i] = offset
		offset += len(line) + 1 // +1 for newline
	}
	lineOffsets[len(lines)] = offset

	for lineNum, line := range lines {
		for _, pattern := range rule.Patterns {
			foundIndexes := pattern.FindAllStringIndex(line, -1)
			for _, idx := range foundIndexes {
				matchValue := line[idx[0]:idx[1]]

				valid := true
				for _, validator := range rule.Validators {
					if !validator(matchValue) {
						valid = false
						break
					}
				}

				if !valid {
					continue
				}

				// For local context checking (ContextDistance > 0)
				if rule.ContextRequired && rule.ContextDistance > 0 && len(rule.ContextPatterns) > 0 {
					// Get content around the match
					matchPos := lineOffsets[lineNum] + idx[0]
					contextStart := matchPos - rule.ContextDistance
					if contextStart < 0 {
						contextStart = 0
					}
					contextEnd := matchPos + idx[1] - idx[0] + rule.ContextDistance
					if contextEnd > len(content) {
						contextEnd = len(content)
					}
					localContext := strings.ToLower(content[contextStart:contextEnd])

					localContextFound := false
					for _, cp := range rule.ContextPatterns {
						if cp.MatchString(localContext) {
							localContextFound = true
							break
						}
					}
					if !localContextFound {
						continue
					}
				}

				// Check negative patterns (exclusions)
				if len(rule.NegativePatterns) > 0 {
					// Get broader context for exclusion check
					contextStart := idx[0] - 100
					if contextStart < 0 {
						contextStart = 0
					}
					contextEnd := idx[1] + 100
					if contextEnd > len(line) {
						contextEnd = len(line)
					}
					broadContext := strings.ToLower(line[contextStart:contextEnd])

					excluded := false
					for _, np := range rule.NegativePatterns {
						if np.MatchString(broadContext) {
							excluded = true
							break
						}
					}
					if excluded {
						continue
					}
				}

				// Calculate column position (1-based)
				colNum := idx[0] + 1

				// Try to find column name for CSV
				colName := ""
				if len(csvHeaders) > 0 && lineNum > 0 {
					// Count commas before the match to determine column
					commaCount := strings.Count(line[:idx[0]], ",")
					if commaCount < len(csvHeaders) {
						colName = csvHeaders[commaCount]
					}
				}

				// Extract context (30 chars before and after, masked)
				contextStart := idx[0] - 30
				if contextStart < 0 {
					contextStart = 0
				}
				contextEnd := idx[1] + 30
				if contextEnd > len(line) {
					contextEnd = len(line)
				}
				context := line[contextStart:contextEnd]

				matches = append(matches, rawMatch{
					value:    matchValue,
					lineNum:  lineNum + 1,
					colNum:   colNum,
					colName:  colName,
					context:  context,
					lineText: line,
				})
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
			NegativePatterns: []*regexp.Regexp{
				// Exclude system/automated/noreply emails
				regexp.MustCompile(`(?i)(noreply|no-reply|donotreply|notifications?@|alerts?@|system@|admin@|support@|info@|contact@|mailer@|postmaster@|hostmaster@|webmaster@)`),
				// Exclude example/test domains
				regexp.MustCompile(`(?i)@(example|test|localhost|invalid|sample)\.(com|org|net|local)`),
				// Exclude database connection strings (postgresql://user:pass@host, mongodb+srv://user:pass@host, etc.)
				regexp.MustCompile(`(?i)(://[^@\s]*@|redis://|mongodb://|mongodb\+srv://|postgresql://|postgres://|mysql://)`),
			},
		},
		{
			Name:        "PHONE_US",
			Category:    models.CategoryPII,
			Sensitivity: models.SensitivityMedium,
			Patterns: []*regexp.Regexp{
				// Require separators to avoid matching random 10-digit numbers
				regexp.MustCompile(`\b\d{3}[-.\s]\d{3}[-.\s]\d{4}\b`),
				regexp.MustCompile(`\b\(\d{3}\)\s?\d{3}[-.\s]?\d{4}\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(phone|tel|cell|mobile|fax|contact|call)`),
			},
			NegativePatterns: []*regexp.Regexp{
				// Exclude IP-like patterns, version numbers, timestamps
				regexp.MustCompile(`(?i)(ip|address|port|version|timestamp|serial|order|invoice|reference)`),
			},
			ContextRequired: true, // Require phone-related context
			ContextDistance: 150,  // Context must be within 150 chars
			Validators:      []Validator{ValidateUSPhone},
		},
		{
			Name:        "ADDRESS_US",
			Category:    models.CategoryPII,
			Sensitivity: models.SensitivityMedium,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)\b\d+\s+[A-Za-z]+\s+(Street|St|Avenue|Ave|Road|Rd|Boulevard|Blvd|Drive|Dr|Lane|Ln|Way|Court|Ct)\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(address|mailing|residence|shipping|billing|location|live|home)`),
			},
			NegativePatterns: []*regexp.Regexp{
				// Exclude code paths, file paths, variable names
				regexp.MustCompile(`(?i)(path|file|folder|directory|url|endpoint)`),
			},
			ContextRequired: true,
			ContextDistance: 200,
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
				regexp.MustCompile(`(?i)(dob|birth|born|birthday|date.of.birth)`),
			},
			NegativePatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(version|release|updated|created|modified|timestamp)`),
			},
			ContextRequired: true,  // Only match dates when birth-related context is present
			ContextDistance: 200,   // Context must be within 200 chars of the match (local check)
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
				// ICD-10 codes: Letter + 2 digits + optional decimal + 1-4 digits
				// More restrictive: require decimal for longer codes
				regexp.MustCompile(`\b[A-TV-Z]\d{2}\.\d{1,4}\b`),
				regexp.MustCompile(`\b[A-TV-Z]\d{2}\b`), // Short form without decimal
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(diagnosis|icd|icd-10|condition|dx|medical|clinical|patient)`),
			},
			NegativePatterns: []*regexp.Regexp{
				// Exclude version strings, platform names, product codes
				regexp.MustCompile(`(?i)(version|platform|gaia|firmware|release|model|sku|part.?number|serial)`),
				regexp.MustCompile(`(?i)(sfp|1gb|10gb|100gb|longwave|shortwave|multi.?mode|single.?mode)`),
			},
			ContextRequired: true,
			ContextDistance: 300, // Context must be within 300 chars (local check)
		},
		{
			Name:        "NDC",
			Category:    models.CategoryPHI,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b\d{5}-\d{4}-\d{2}\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(ndc|drug|medication|prescription|rx|pharmacy|dosage)`),
			},
			ContextRequired: true,
			ContextDistance: 200,
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
				// Require explicit banking context
				regexp.MustCompile(`(?i)(bank\s*account|checking\s*(account)?|savings\s*(account)?|account\s*#|acct\s*#|wire\s*transfer)`),
			},
			NegativePatterns: []*regexp.Regexp{
				// Exclude AWS account numbers, ECR URIs, docker registries
				regexp.MustCompile(`(?i)(\.ecr\.|\.dkr\.|amazonaws|docker|registry|arn:aws|aws_account)`),
				// Exclude other technical identifiers
				regexp.MustCompile(`(?i)(instance.?id|volume.?id|snapshot.?id|resource.?id|request.?id|trace.?id|span.?id)`),
			},
			ContextRequired: true,
			ContextDistance: 150, // Context must be within 150 chars (local check)
		},
		{
			Name:        "ROUTING_NUMBER",
			Category:    models.CategoryPCI,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`\b\d{9}\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(routing\s*(number)?|aba\s*(number)?|bank\s*transit)`),
			},
			NegativePatterns: []*regexp.Regexp{
				// Exclude SSN-like patterns, phone numbers, zip codes
				regexp.MustCompile(`(?i)(ssn|social|phone|zip|postal|timestamp|id)`),
			},
			ContextRequired: true,
			ContextDistance: 100,
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
			},
			NegativePatterns: []*regexp.Regexp{
				// Exclude already-masked/redacted values (contain asterisks or placeholder patterns)
				regexp.MustCompile(`\*{3,}`),
				// Exclude documentation describing password fields
				regexp.MustCompile(`(?i)(\(required\)|\(optional\)|password\s+for\s+t)`),
			},
		},
		{
			Name:        "GENERIC_API_KEY",
			Category:    models.CategorySecrets,
			Sensitivity: models.SensitivityHigh,
			Patterns: []*regexp.Regexp{
				// Require mix of letters and numbers to avoid hashes, UUIDs
				regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9]{31,63}\b`),
			},
			ContextPatterns: []*regexp.Regexp{
				// More specific context - must have "key" nearby
				regexp.MustCompile(`(?i)(api[_-]?key\s*[=:]|apikey\s*[=:]|x-api-key\s*[=:]|secret\s*key\s*[=:])`),
			},
			NegativePatterns: []*regexp.Regexp{
				// Exclude common hashes, UUIDs, checksums
				regexp.MustCompile(`(?i)(sha256|sha1|md5|hash|checksum|digest|uuid|guid|etag)`),
				// Exclude base64 encoded data
				regexp.MustCompile(`(?i)(base64|encoded|encryption)`),
			},
			ContextRequired: true,
			ContextDistance: 50, // Very close context required
		},
	}
}

func ValidateUSPhone(phone string) bool {
	// Extract digits only
	var digits strings.Builder
	for _, c := range phone {
		if unicode.IsDigit(c) {
			digits.WriteRune(c)
		}
	}
	clean := digits.String()

	if len(clean) != 10 {
		return false
	}

	// Area code cannot start with 0 or 1
	if clean[0] == '0' || clean[0] == '1' {
		return false
	}

	// Exchange (middle 3 digits) cannot start with 0 or 1
	if clean[3] == '0' || clean[3] == '1' {
		return false
	}

	// Reject obvious test/fake numbers
	if clean == "1234567890" || clean == "0000000000" ||
		clean == "1111111111" || clean == "5555555555" {
		return false
	}

	return true
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

// Patterns for additional sensitive data that should be masked in context
var sensitiveContextPatterns = []*regexp.Regexp{
	// SSN patterns (XXX-XX-XXXX or XXX XX XXXX)
	regexp.MustCompile(`\b\d{3}[-\s]\d{2}[-\s]\d{4}\b`),
	// Phone numbers (XXX-XXX-XXXX, (XXX) XXX-XXXX, XXX.XXX.XXXX)
	regexp.MustCompile(`\b\d{3}[-.\s]\d{3}[-.\s]\d{4}\b`),
	regexp.MustCompile(`\(\d{3}\)\s?\d{3}[-.\s]?\d{4}`),
	// Email addresses
	regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`),
	// Dates (MM/DD/YYYY, MM-DD-YYYY, YYYY-MM-DD) - potential DOB
	regexp.MustCompile(`\b\d{1,2}[/-]\d{1,2}[/-]\d{2,4}\b`),
	regexp.MustCompile(`\b\d{4}[/-]\d{1,2}[/-]\d{1,2}\b`),
	// Street addresses (number + street name)
	regexp.MustCompile(`\b\d+\s+[A-Za-z]+\s+(St|Street|Ave|Avenue|Rd|Road|Dr|Drive|Ln|Lane|Blvd|Boulevard|Way|Ct|Court|Pl|Place|Cir|Circle)\b`),
	// Bank account numbers (8-17 digits)
	regexp.MustCompile(`\b\d{8,17}\b`),
	// Credit card-like patterns
	regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`),
	// Long alphanumeric strings that might be secrets (32+ chars)
	regexp.MustCompile(`\b[A-Za-z0-9+/=]{32,}\b`),
	// Password-like values after = or :
	regexp.MustCompile(`(?i)(?:password|passwd|pwd|secret|key|token)\s*[=:]\s*\S+`),
	// Names after common labels (first name, last name patterns)
	regexp.MustCompile(`(?i)(?:name|first|last|customer|patient|user)[:\s]+[A-Z][a-z]+`),
}

// redactContext masks the sensitive value AND any other potentially sensitive patterns in the context string
func redactContext(context, sensitiveValue string) string {
	if context == "" {
		return context
	}

	result := context

	// First, mask the primary sensitive value
	if sensitiveValue != "" {
		masked := redact(sensitiveValue)
		result = strings.ReplaceAll(result, sensitiveValue, masked)
	}

	// Then, mask any other sensitive patterns found in the context
	for _, pattern := range sensitiveContextPatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Don't double-mask if already contains asterisks
			if strings.Contains(match, "*") {
				return match
			}
			return redact(match)
		})
	}

	return result
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
