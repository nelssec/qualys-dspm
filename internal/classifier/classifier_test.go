package classifier

import (
	"strings"
	"testing"

	"github.com/qualys/dspm/internal/models"
)

func TestClassifier_SSN(t *testing.T) {
	c := New()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"valid SSN with dashes", "My SSN is 123-45-6789", true},
		{"valid SSN with spaces", "SSN: 123 45 6789", true},
		{"invalid SSN area 000", "SSN: 000-12-3456", false},
		{"invalid SSN area 666", "SSN: 666-12-3456", false},
		{"invalid SSN area 900+", "SSN: 900-12-3456", false},
		{"no SSN", "Just some random text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Classify(tt.content)
			found := false
			for _, m := range result.Matches {
				if m.RuleName == "SSN" {
					found = true
					break
				}
			}
			if found != tt.expected {
				t.Errorf("expected SSN found=%v, got %v", tt.expected, found)
			}
		})
	}
}

func TestClassifier_Email(t *testing.T) {
	c := New()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"valid email", "Contact us at john.doe@acmecorp.com", true},
		{"email with subdomain", "Email: user@mail.bigcompany.com", true},
		{"email with plus", "Send to user+tag@mycompany.org", true},
		{"no email", "Just some text without email", false},
		{"invalid email", "Not an email@", false},
		// Test exclusions
		{"exclude example.com", "Contact us at test@example.com", false},
		{"exclude noreply", "From: noreply@acmecorp.com", false},
		// Test database connection string exclusion
		{"exclude db connection string", "DATABASE_URL=postgresql://admin:password@db.production.com:5432/app", false},
		{"exclude redis url", "REDIS_URL=redis://:secret@cache.myservice.io:6379", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Classify(tt.content)
			found := false
			for _, m := range result.Matches {
				if m.RuleName == "EMAIL" {
					found = true
					break
				}
			}
			if found != tt.expected {
				t.Errorf("expected EMAIL found=%v, got %v", tt.expected, found)
			}
		})
	}
}

func TestClassifier_CreditCard(t *testing.T) {
	c := New()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"valid Visa", "Card: 4532015112830366", true},
		{"valid Visa with spaces", "Card: 4532 0151 1283 0366", true},
		{"valid Mastercard", "Card: 5425233430109903", true},
		{"valid Amex", "Card: 374245455400126", true},
		{"invalid Luhn", "Card: 4532015112830367", false}, // fails Luhn
		{"no card", "Just some numbers 12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Classify(tt.content)
			found := false
			for _, m := range result.Matches {
				if m.RuleName == "CREDIT_CARD" {
					found = true
					break
				}
			}
			if found != tt.expected {
				t.Errorf("expected CREDIT_CARD found=%v, got %v", tt.expected, found)
			}
		})
	}
}

func TestClassifier_AWSKeys(t *testing.T) {
	c := New()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"AWS access key", "Key: AKIAIOSFODNN7EXAMPLE", true},
		{"AWS temp key", "Key: ASIAIOSFODNN7EXAMPLE", true},
		{"not AWS key", "Key: BKIAIOSFODNN7EXAMPLE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Classify(tt.content)
			found := false
			for _, m := range result.Matches {
				if m.RuleName == "AWS_ACCESS_KEY" {
					found = true
					break
				}
			}
			if found != tt.expected {
				t.Errorf("expected AWS_ACCESS_KEY found=%v, got %v", tt.expected, found)
			}
		})
	}
}

func TestClassifier_PrivateKey(t *testing.T) {
	c := New()

	content := `
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF9YQHV0U
-----END RSA PRIVATE KEY-----
`

	result := c.Classify(content)
	found := false
	for _, m := range result.Matches {
		if m.RuleName == "PRIVATE_KEY" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected PRIVATE_KEY to be found")
	}
}

func TestClassifier_JWT(t *testing.T) {
	c := New()

	// Valid JWT format (not a real token)
	content := "Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	result := c.Classify(content)
	found := false
	for _, m := range result.Matches {
		if m.RuleName == "JWT" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected JWT to be found")
	}
}

func TestClassifier_GitHubToken(t *testing.T) {
	c := New()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"personal access token", "Token: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", true},
		{"oauth token", "Token: gho_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", true},
		{"not github token", "Token: ghx_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Classify(tt.content)
			found := false
			for _, m := range result.Matches {
				if m.RuleName == "GITHUB_TOKEN" {
					found = true
					break
				}
			}
			if found != tt.expected {
				t.Errorf("expected GITHUB_TOKEN found=%v, got %v", tt.expected, found)
			}
		})
	}
}

func TestClassifier_DBConnectionString(t *testing.T) {
	c := New()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		// True positives - real connection strings
		{"postgresql url", "DATABASE_URL=postgresql://admin:secretpass@db.production.com:5432/app_db", true},
		{"postgres url", "DB_URI=postgres://user:password123@localhost:5432/mydb", true},
		{"mysql url", "MYSQL_URL=mysql://root:rootpass@mysql.server.com:3306/database", true},
		{"mongodb url", "MONGO_URI=mongodb://admin:mongopass@cluster.mongodb.net:27017/db", true},
		{"redis url", "REDIS_URL=redis://user:redispass@cache.server.com:6379", true},
		// False positives - should NOT match
		{"masked placeholder", "Qualys scanner IP addresses. pa************e} (Required for new vault) The", false},
		{"documentation text", "ieberman ERPM server account. pa************e} (Required) The password for t", false},
		{"asterisk mask", "he username must contain '@'. pa************e} (Required to create record, o", false},
		{"no connection string", "Just some regular text without any database URLs", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Classify(tt.content)
			found := false
			for _, m := range result.Matches {
				if m.RuleName == "DB_CONNECTION_STRING" {
					found = true
					break
				}
			}
			if found != tt.expected {
				t.Errorf("expected DB_CONNECTION_STRING found=%v, got %v for content: %s", tt.expected, found, tt.content)
			}
		})
	}
}

func TestClassifier_Sensitivity(t *testing.T) {
	c := New()

	// Content with SSN (CRITICAL) and Email (MEDIUM)
	content := "SSN: 123-45-6789, Email: john.doe@acmecorp.com"

	result := c.Classify(content)

	if result.MaxSensitivity != models.SensitivityCritical {
		t.Errorf("expected max sensitivity CRITICAL, got %s", result.MaxSensitivity)
	}
}

func TestClassifier_Categories(t *testing.T) {
	c := New()

	// Content with PII (SSN) and PCI (credit card)
	content := "SSN: 123-45-6789, Card: 4532015112830366"

	result := c.Classify(content)

	hasPII := false
	hasPCI := false
	for _, cat := range result.Categories {
		if cat == models.CategoryPII {
			hasPII = true
		}
		if cat == models.CategoryPCI {
			hasPCI = true
		}
	}

	if !hasPII {
		t.Error("expected PII category")
	}
	if !hasPCI {
		t.Error("expected PCI category")
	}
}

func TestClassifier_Redact(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1234567890", "12******90"},
		{"abc", "****"},
		{"test@example.com", "te************om"},
	}

	for _, tt := range tests {
		result := redact(tt.input)
		if result != tt.expected {
			t.Errorf("redact(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestRedactContext(t *testing.T) {
	tests := []struct {
		name           string
		context        string
		sensitiveValue string
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name:           "masks bank account and routing number together",
			context:        "Bank Account: 933135462 (Routing: 916912404)",
			sensitiveValue: "933135462",
			shouldContain:  []string{"93*****62", "91*****04"},
			shouldNotContain: []string{"933135462", "916912404"},
		},
		{
			name:           "masks SSN pattern in context",
			context:        "Customer SSN: 123-45-6789 lives at address",
			sensitiveValue: "123-45-6789",
			shouldContain:  []string{"12*******89"},
			shouldNotContain: []string{"123-45-6789"},
		},
		{
			name:           "masks long numeric strings",
			context:        "Account number 12345678901234567 confirmed",
			sensitiveValue: "12345678901234567",
			shouldContain:  []string{"12*************67"},
			shouldNotContain: []string{"12345678901234567"},
		},
		{
			name:           "does not double-mask already masked values",
			context:        "Account: 93*****62 is masked",
			sensitiveValue: "",
			shouldContain:  []string{"93*****62"},
		},
		{
			name:           "masks phone numbers in context",
			context:        "Contact: john@email.com,856-380-4673,123-45-6789,04/05/1975,5734 Elm St,Dallas",
			sensitiveValue: "123-45-6789",
			shouldContain:  []string{"12*******89"}, // SSN masked
			shouldNotContain: []string{"856-380-4673", "john@email.com", "04/05/1975", "5734 Elm St"},
		},
		{
			name:           "masks email and DOB",
			context:        "User test.user@company.com born 12/25/1990 at 123 Main Ave",
			sensitiveValue: "",
			shouldNotContain: []string{"test.user@company.com", "12/25/1990", "123 Main Ave"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactContext(tt.context, tt.sensitiveValue)
			for _, s := range tt.shouldContain {
				if !strings.Contains(result, s) {
					t.Errorf("redactContext result should contain %q, got: %q", s, result)
				}
			}
			for _, s := range tt.shouldNotContain {
				if strings.Contains(result, s) {
					t.Errorf("redactContext result should NOT contain %q, got: %q", s, result)
				}
			}
		})
	}
}

func TestValidateLuhn(t *testing.T) {
	tests := []struct {
		number   string
		expected bool
	}{
		{"4532015112830366", true},  // Valid Visa
		{"5425233430109903", true},  // Valid Mastercard
		{"374245455400126", true},   // Valid Amex
		{"4532015112830367", false}, // Invalid (wrong check digit)
		{"1234567890123456", false}, // Invalid
		{"123", false},              // Too short
	}

	for _, tt := range tests {
		result := ValidateLuhn(tt.number)
		if result != tt.expected {
			t.Errorf("ValidateLuhn(%s) = %v, expected %v", tt.number, result, tt.expected)
		}
	}
}

func TestValidateSSN(t *testing.T) {
	tests := []struct {
		ssn      string
		expected bool
	}{
		{"123-45-6789", true},
		{"123 45 6789", true},
		{"000-12-3456", false}, // Invalid area
		{"666-12-3456", false}, // Invalid area
		{"900-12-3456", false}, // Invalid area
		{"123-00-6789", false}, // Invalid group
		{"123-45-0000", false}, // Invalid serial
	}

	for _, tt := range tests {
		result := ValidateSSN(tt.ssn)
		if result != tt.expected {
			t.Errorf("ValidateSSN(%s) = %v, expected %v", tt.ssn, result, tt.expected)
		}
	}
}

func TestValidateABARouting(t *testing.T) {
	tests := []struct {
		routing  string
		expected bool
	}{
		{"021000021", true},  // JPMorgan Chase
		{"011401533", true},  // Bank of America
		{"123456789", false}, // Invalid checksum
	}

	for _, tt := range tests {
		result := ValidateABARouting(tt.routing)
		if result != tt.expected {
			t.Errorf("ValidateABARouting(%s) = %v, expected %v", tt.routing, result, tt.expected)
		}
	}
}

func TestValidateIBAN(t *testing.T) {
	tests := []struct {
		iban     string
		expected bool
	}{
		{"GB82WEST12345698765432", true},  // Valid UK IBAN
		{"DE89370400440532013000", true},  // Valid German IBAN
		{"GB82WEST12345698765433", false}, // Invalid checksum
	}

	for _, tt := range tests {
		result := ValidateIBAN(tt.iban)
		if result != tt.expected {
			t.Errorf("ValidateIBAN(%s) = %v, expected %v", tt.iban, result, tt.expected)
		}
	}
}

func BenchmarkClassifier(b *testing.B) {
	c := New()
	content := `
		Name: John Doe
		Email: john.doe@acmecorp.com
		SSN: 123-45-6789
		Phone: (555) 123-4567
		Card: 4532 0151 1283 0366
		Address: 123 Main Street, New York, NY 10001
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Classify(content)
	}
}
