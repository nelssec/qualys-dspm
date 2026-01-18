package mlclassifier

import (
	"context"
	"regexp"
	"testing"
)

func TestRuleBasedNER_RecognizeEntities(t *testing.T) {
	ner := NewRuleBasedNER()
	ctx := context.Background()

	tests := []struct {
		name           string
		text           string
		expectedTypes  []string
		expectedCount  int
		expectedValues []string
	}{
		{
			name:           "email recognition",
			text:           "Contact us at support@example.com for help",
			expectedTypes:  []string{"EMAIL"},
			expectedCount:  1,
			expectedValues: []string{"support@example.com"},
		},
		{
			name:           "multiple emails",
			text:           "Email john@test.com or jane@company.org",
			expectedTypes:  []string{"EMAIL", "EMAIL"},
			expectedCount:  2,
			expectedValues: []string{"john@test.com", "jane@company.org"},
		},
		{
			name:           "phone number US format",
			text:           "Call us at 555-123-4567",
			expectedTypes:  []string{"PHONE"},
			expectedCount:  1,
			expectedValues: []string{"555-123-4567"},
		},
		{
			name:           "phone with area code parentheses",
			text:           "Phone: (555) 123-4567",
			expectedTypes:  []string{"PHONE"},
			expectedCount:  1,
			expectedValues: []string{"(555) 123-4567"},
		},
		{
			name:           "SSN with dashes",
			text:           "SSN: 123-45-6789",
			expectedTypes:  []string{"SSN"},
			expectedCount:  1,
			expectedValues: []string{"123-45-6789"},
		},
		{
			name:           "SSN without separators",
			text:           "Social: 123456789",
			expectedTypes:  []string{"SSN"},
			expectedCount:  1,
			expectedValues: []string{"123456789"},
		},
		{
			name:          "invalid SSN area 000",
			text:          "SSN: 000-12-3456",
			expectedCount: 0,
		},
		{
			name:          "invalid SSN area 666",
			text:          "SSN: 666-12-3456",
			expectedCount: 0,
		},
		{
			name:           "credit card Visa",
			text:           "Card: 4111111111111111",
			expectedTypes:  []string{"CREDIT_CARD"},
			expectedCount:  1,
			expectedValues: []string{"4111111111111111"},
		},
		{
			name:           "credit card Mastercard",
			text:           "Payment with 5500000000000004",
			expectedTypes:  []string{"CREDIT_CARD"},
			expectedCount:  1,
			expectedValues: []string{"5500000000000004"},
		},
		{
			name:          "invalid credit card (fails Luhn)",
			text:          "Card: 4111111111111112",
			expectedCount: 0,
		},
		{
			name:           "IPv4 address",
			text:           "Server IP: 192.168.1.100",
			expectedTypes:  []string{"IP_ADDRESS"},
			expectedCount:  1,
			expectedValues: []string{"192.168.1.100"},
		},
		{
			name:           "multiple IP addresses",
			text:           "From 10.0.0.1 to 10.0.0.255",
			expectedTypes:  []string{"IP_ADDRESS", "IP_ADDRESS"},
			expectedCount:  2,
			expectedValues: []string{"10.0.0.1", "10.0.0.255"},
		},
		{
			name:           "AWS ARN",
			text:           "Resource: arn:aws:s3:::my-bucket/path/to/object",
			expectedTypes:  []string{"AWS_ARN"},
			expectedCount:  1,
			expectedValues: []string{"arn:aws:s3:::my-bucket/path/to/object"},
		},
		{
			name:           "AWS access key",
			text:           "Key: AKIAIOSFODNN7EXAMPLE",
			expectedTypes:  []string{"AWS_ACCESS_KEY"},
			expectedCount:  1,
			expectedValues: []string{"AKIAIOSFODNN7EXAMPLE"},
		},
		{
			name:           "AWS temporary key",
			text:           "Temp: ASIAISAMPLEKEYID1234",
			expectedTypes:  []string{"AWS_ACCESS_KEY"},
			expectedCount:  1,
			expectedValues: []string{"ASIAISAMPLEKEYID1234"},
		},
		{
			name:           "UUID",
			text:           "ID: 550e8400-e29b-41d4-a716-446655440000",
			expectedTypes:  []string{"UUID"},
			expectedCount:  1,
			expectedValues: []string{"550e8400-e29b-41d4-a716-446655440000"},
		},
		{
			name:           "date MM/DD/YYYY",
			text:           "Date: 12/25/2024",
			expectedTypes:  []string{"DATE"},
			expectedCount:  1,
			expectedValues: []string{"12/25/2024"},
		},
		{
			name:           "person with title",
			text:           "Contact Dr. John Smith for details",
			expectedTypes:  []string{"PERSON"},
			expectedCount:  1,
			expectedValues: []string{"Dr. John Smith"},
		},
		{
			name:           "IBAN",
			text:           "Bank account: DE89370400440532013000",
			expectedTypes:  []string{"IBAN"},
			expectedCount:  1,
			expectedValues: []string{"DE89370400440532013000"},
		},
		{
			name:          "no entities",
			text:          "This is just regular text without any sensitive data.",
			expectedCount: 0,
		},
		{
			name:           "mixed entities",
			text:           "Contact john@example.com at 555-123-4567, SSN 123-45-6789",
			expectedTypes:  []string{"EMAIL", "PHONE", "SSN"},
			expectedCount:  3,
			expectedValues: []string{"john@example.com", "555-123-4567", "123-45-6789"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities, err := ner.RecognizeEntities(ctx, tt.text)
			if err != nil {
				t.Fatalf("RecognizeEntities() error = %v", err)
			}

			if len(entities) != tt.expectedCount {
				t.Errorf("RecognizeEntities() got %d entities, want %d", len(entities), tt.expectedCount)
				for _, e := range entities {
					t.Logf("  Found: %s (%s)", e.Text, e.Type)
				}
				return
			}

			if tt.expectedCount > 0 {
				// Check that expected values are found
				for _, expectedValue := range tt.expectedValues {
					found := false
					for _, entity := range entities {
						if entity.Text == expectedValue {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find entity with value %q", expectedValue)
					}
				}

				// Check that expected types are found
				for _, expectedType := range tt.expectedTypes {
					found := false
					for _, entity := range entities {
						if entity.Type == expectedType {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find entity of type %q", expectedType)
					}
				}
			}
		})
	}
}

func TestRuleBasedNER_EntityOffsets(t *testing.T) {
	ner := NewRuleBasedNER()
	ctx := context.Background()

	text := "Email: test@example.com"
	entities, err := ner.RecognizeEntities(ctx, text)
	if err != nil {
		t.Fatalf("RecognizeEntities() error = %v", err)
	}

	if len(entities) != 1 {
		t.Fatalf("Expected 1 entity, got %d", len(entities))
	}

	entity := entities[0]
	extractedText := text[entity.StartOffset:entity.EndOffset]
	if extractedText != entity.Text {
		t.Errorf("Offset mismatch: extracted %q, entity.Text = %q", extractedText, entity.Text)
	}
}

func TestRuleBasedNER_Confidence(t *testing.T) {
	ner := NewRuleBasedNER()
	ctx := context.Background()

	tests := []struct {
		name              string
		text              string
		entityType        string
		minConfidence     float64
	}{
		{
			name:          "email high confidence",
			text:          "test@example.com",
			entityType:    "EMAIL",
			minConfidence: 0.90,
		},
		{
			name:          "IP address high confidence",
			text:          "192.168.1.1",
			entityType:    "IP_ADDRESS",
			minConfidence: 0.90,
		},
		{
			name:          "UUID high confidence",
			text:          "550e8400-e29b-41d4-a716-446655440000",
			entityType:    "UUID",
			minConfidence: 0.90,
		},
		{
			name:          "person lower confidence",
			text:          "Dr. John Smith",
			entityType:    "PERSON",
			minConfidence: 0.60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities, err := ner.RecognizeEntities(ctx, tt.text)
			if err != nil {
				t.Fatalf("RecognizeEntities() error = %v", err)
			}

			if len(entities) == 0 {
				t.Fatalf("Expected at least 1 entity")
			}

			var found bool
			for _, entity := range entities {
				if entity.Type == tt.entityType {
					found = true
					if entity.Confidence < tt.minConfidence {
						t.Errorf("Entity %s confidence = %v, want >= %v", tt.entityType, entity.Confidence, tt.minConfidence)
					}
				}
			}

			if !found {
				t.Errorf("Entity type %s not found", tt.entityType)
			}
		})
	}
}

func TestRuleBasedNER_AddCustomPattern(t *testing.T) {
	ner := NewRuleBasedNER()
	ctx := context.Background()

	// Add a custom pattern for employee IDs
	ner.AddPattern("EMPLOYEE_ID", &EntityPattern{
		Regex:       regexp.MustCompile(`\bEMP-[0-9]{6}\b`),
		EntityType:  "EMPLOYEE_ID",
		Confidence:  0.95,
		Description: "Employee ID",
	})

	text := "Employee EMP-123456 submitted the request"
	entities, err := ner.RecognizeEntities(ctx, text)
	if err != nil {
		t.Fatalf("RecognizeEntities() error = %v", err)
	}

	var found bool
	for _, entity := range entities {
		if entity.Type == "EMPLOYEE_ID" && entity.Text == "EMP-123456" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Custom EMPLOYEE_ID pattern not recognized")
	}
}

func TestValidateSSN(t *testing.T) {
	tests := []struct {
		ssn   string
		valid bool
	}{
		{"123-45-6789", true},
		{"123 45 6789", true},
		{"123456789", true},
		{"000-12-3456", false}, // area 000
		{"666-12-3456", false}, // area 666
		{"900-12-3456", false}, // area 9xx
		{"123-00-4567", false}, // group 00
		{"123-45-0000", false}, // serial 0000
		{"12345678", false},    // too short
		{"1234567890", false},  // too long
	}

	for _, tt := range tests {
		t.Run(tt.ssn, func(t *testing.T) {
			if got := validateSSN(tt.ssn); got != tt.valid {
				t.Errorf("validateSSN(%q) = %v, want %v", tt.ssn, got, tt.valid)
			}
		})
	}
}

func TestValidateLuhn(t *testing.T) {
	tests := []struct {
		number string
		valid  bool
	}{
		{"4111111111111111", true},  // Visa test number
		{"5500000000000004", true},  // Mastercard test number
		{"340000000000009", true},   // Amex test number
		{"4111111111111112", false}, // Invalid
		{"1234567890123456", false}, // Random invalid
		{"", false},                 // Empty
		{"12345", false},            // Too short
	}

	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			if got := validateLuhn(tt.number); got != tt.valid {
				t.Errorf("validateLuhn(%q) = %v, want %v", tt.number, got, tt.valid)
			}
		})
	}
}

func TestValidateIBAN(t *testing.T) {
	tests := []struct {
		iban  string
		valid bool
	}{
		{"DE89370400440532013000", true},  // Germany
		{"GB82WEST12345698765432", true},  // UK
		{"FR7630006000011234567890189", true}, // France
		{"1234567890", false},              // No country code
		{"XX12345", false},                 // Too short
		{"12DE370400440532013000", false},  // Country code not at start
	}

	for _, tt := range tests {
		t.Run(tt.iban, func(t *testing.T) {
			if got := validateIBAN(tt.iban); got != tt.valid {
				t.Errorf("validateIBAN(%q) = %v, want %v", tt.iban, got, tt.valid)
			}
		})
	}
}

// Benchmark tests
func BenchmarkNER_SmallText(b *testing.B) {
	ner := NewRuleBasedNER()
	ctx := context.Background()
	text := "Contact john@example.com at 555-123-4567"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ner.RecognizeEntities(ctx, text)
	}
}

func BenchmarkNER_LargeText(b *testing.B) {
	ner := NewRuleBasedNER()
	ctx := context.Background()

	// Generate a larger text with multiple entities
	text := `
		Customer Report
		================
		Name: Dr. John Smith
		Email: john.smith@example.com
		Phone: (555) 123-4567
		SSN: 123-45-6789

		Payment Information:
		Card: 4111111111111111
		Billing Address: 123 Main St

		Server Details:
		IP: 192.168.1.100
		Instance: arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0

		Additional contacts:
		- jane@company.org
		- support@test.com
		- (800) 555-0123

		Reference ID: 550e8400-e29b-41d4-a716-446655440000
		Date: 12/25/2024
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ner.RecognizeEntities(ctx, text)
	}
}
