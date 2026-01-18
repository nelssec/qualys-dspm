package encryption

import (
	"testing"

	"github.com/google/uuid"
	"github.com/qualys/dspm/internal/models"
)

func TestComplianceScorer_CalculateComplianceScore(t *testing.T) {
	scorer := NewComplianceScorer()

	tests := []struct {
		name          string
		profile       *AssetEncryptionProfile
		minScore      int
		maxScore      int
		expectedGrade string
		hasFindings   bool
	}{
		{
			name: "no encryption - grade F",
			profile: &AssetEncryptionProfile{
				AssetID:          uuid.New(),
				EncryptionStatus: models.EncryptionNone,
			},
			minScore:      15,
			maxScore:      35,
			expectedGrade: "F",
			hasFindings:   true,
		},
		{
			name: "SSE only - grade C/D",
			profile: &AssetEncryptionProfile{
				AssetID:          uuid.New(),
				EncryptionStatus: models.EncryptionSSE,
			},
			minScore:      50,
			maxScore:      70,
			expectedGrade: "", // C or D
			hasFindings:   false,
		},
		{
			name: "SSE-KMS without rotation - grade B",
			profile: &AssetEncryptionProfile{
				AssetID:            uuid.New(),
				EncryptionStatus:   models.EncryptionSSEKMS,
				KeyRotationEnabled: false,
			},
			minScore:    60,
			maxScore:    85,
			hasFindings: true, // Key rotation disabled finding
		},
		{
			name: "SSE-KMS with rotation - grade B/C",
			profile: &AssetEncryptionProfile{
				AssetID:            uuid.New(),
				EncryptionStatus:   models.EncryptionSSEKMS,
				KeyRotationEnabled: true,
			},
			minScore:    70,
			maxScore:    85,
			hasFindings: false,
		},
		{
			name: "CMK with all best practices - grade A",
			profile: &AssetEncryptionProfile{
				AssetID:            uuid.New(),
				EncryptionStatus:   models.EncryptionCMK,
				KeyRotationEnabled: true,
				Key: &models.EncryptionKey{
					Enabled:            true,
					RotationEnabled:    true,
					KeyManager:         "CUSTOMER",
					AllowsPublicAccess: false,
					KeyState:           models.KeyStateEnabled,
				},
				TransitEncryption: &models.TransitEncryption{
					TLSEnabled:                    true,
					TLSVersion:                    "TLSv1.3",
					SupportsPerfectForwardSecrecy: true,
					CertificateARN:                "arn:aws:acm:us-east-1:123456789:certificate/abc",
				},
			},
			minScore:      90,
			maxScore:      100,
			expectedGrade: "A",
			hasFindings:   false,
		},
		{
			name: "CMK with disabled key - grade D",
			profile: &AssetEncryptionProfile{
				AssetID:          uuid.New(),
				EncryptionStatus: models.EncryptionCMK,
				Key: &models.EncryptionKey{
					Enabled:  false,
					KeyState: models.KeyStateDisabled,
				},
			},
			minScore:    50,
			maxScore:    70,
			hasFindings: true,
		},
		{
			name: "transit encryption disabled - critical finding",
			profile: &AssetEncryptionProfile{
				AssetID:          uuid.New(),
				EncryptionStatus: models.EncryptionSSEKMS,
				TransitEncryption: &models.TransitEncryption{
					TLSEnabled: false,
				},
			},
			minScore:    30,
			maxScore:    60,
			hasFindings: true,
		},
		{
			name: "outdated TLS version",
			profile: &AssetEncryptionProfile{
				AssetID:          uuid.New(),
				EncryptionStatus: models.EncryptionSSEKMS,
				TransitEncryption: &models.TransitEncryption{
					TLSEnabled: true,
					TLSVersion: "TLSv1.0",
				},
			},
			minScore:    50,
			maxScore:    75,
			hasFindings: true,
		},
		{
			name: "key pending deletion",
			profile: &AssetEncryptionProfile{
				AssetID:          uuid.New(),
				EncryptionStatus: models.EncryptionCMK,
				Key: &models.EncryptionKey{
					Enabled:  true,
					KeyState: models.KeyStatePendingDeletion,
				},
			},
			minScore:    65,
			maxScore:    80,
			hasFindings: true,
		},
		{
			name: "public key policy",
			profile: &AssetEncryptionProfile{
				AssetID:          uuid.New(),
				EncryptionStatus: models.EncryptionCMK,
				Key: &models.EncryptionKey{
					Enabled:            true,
					AllowsPublicAccess: true,
					KeyState:           models.KeyStateEnabled,
				},
			},
			minScore:    65,
			maxScore:    80,
			hasFindings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scorer.CalculateComplianceScore(tt.profile)

			if result.Score < tt.minScore || result.Score > tt.maxScore {
				t.Errorf("Score = %d, want between %d and %d", result.Score, tt.minScore, tt.maxScore)
			}

			if tt.expectedGrade != "" && result.Grade != tt.expectedGrade {
				t.Errorf("Grade = %q, want %q", result.Grade, tt.expectedGrade)
			}

			if tt.hasFindings && len(result.Findings) == 0 {
				t.Error("Expected findings but got none")
			}

			if !tt.hasFindings && len(result.Findings) > 0 {
				t.Errorf("Expected no findings but got %d", len(result.Findings))
				for _, f := range result.Findings {
					t.Logf("  Finding: %s - %s", f.Type, f.Title)
				}
			}
		})
	}
}

func TestComplianceScorer_AtRestScore(t *testing.T) {
	scorer := NewComplianceScorer()

	tests := []struct {
		name      string
		status    models.EncryptionStatus
		rotation  bool
		wantScore int
	}{
		{
			name:      "no encryption",
			status:    models.EncryptionNone,
			wantScore: 0,
		},
		{
			name:      "SSE only",
			status:    models.EncryptionSSE,
			wantScore: 40,
		},
		{
			name:      "SSE-KMS without rotation",
			status:    models.EncryptionSSEKMS,
			rotation:  false,
			wantScore: 70, // 40 + 30
		},
		{
			name:      "SSE-KMS with rotation",
			status:    models.EncryptionSSEKMS,
			rotation:  true,
			wantScore: 80, // 40 + 30 + 10
		},
		{
			name:      "CMK without rotation",
			status:    models.EncryptionCMK,
			rotation:  false,
			wantScore: 90, // 40 + 30 + 20
		},
		{
			name:      "CMK with rotation",
			status:    models.EncryptionCMK,
			rotation:  true,
			wantScore: 100, // 40 + 30 + 20 + 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &AssetEncryptionProfile{
				EncryptionStatus:   tt.status,
				KeyRotationEnabled: tt.rotation,
			}
			result := &ComplianceResult{
				Findings:        []EncryptionFinding{},
				Recommendations: []string{},
			}

			score := scorer.calculateAtRestScore(profile, result)
			if score != tt.wantScore {
				t.Errorf("At-rest score = %d, want %d", score, tt.wantScore)
			}
		})
	}
}

func TestComplianceScorer_InTransitScore(t *testing.T) {
	scorer := NewComplianceScorer()

	tests := []struct {
		name      string
		transit   *models.TransitEncryption
		wantScore int
	}{
		{
			name:      "no transit info - default score",
			transit:   nil,
			wantScore: 80,
		},
		{
			name: "TLS disabled",
			transit: &models.TransitEncryption{
				TLSEnabled: false,
			},
			wantScore: 0,
		},
		{
			name: "TLS 1.0",
			transit: &models.TransitEncryption{
				TLSEnabled: true,
				TLSVersion: "TLSv1.0",
			},
			wantScore: 50,
		},
		{
			name: "TLS 1.2",
			transit: &models.TransitEncryption{
				TLSEnabled: true,
				TLSVersion: "TLSv1.2",
			},
			wantScore: 75, // 50 + 25
		},
		{
			name: "TLS 1.3",
			transit: &models.TransitEncryption{
				TLSEnabled: true,
				TLSVersion: "TLSv1.3",
			},
			wantScore: 85, // 50 + 25 + 10
		},
		{
			name: "TLS 1.3 with PFS",
			transit: &models.TransitEncryption{
				TLSEnabled:                    true,
				TLSVersion:                    "TLSv1.3",
				SupportsPerfectForwardSecrecy: true,
			},
			wantScore: 95, // 50 + 25 + 10 + 10
		},
		{
			name: "TLS 1.3 with PFS and cert",
			transit: &models.TransitEncryption{
				TLSEnabled:                    true,
				TLSVersion:                    "TLSv1.3",
				SupportsPerfectForwardSecrecy: true,
				CertificateARN:                "arn:aws:acm:...",
			},
			wantScore: 100, // 50 + 25 + 10 + 10 + 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &AssetEncryptionProfile{
				TransitEncryption: tt.transit,
			}
			result := &ComplianceResult{
				Findings:        []EncryptionFinding{},
				Recommendations: []string{},
			}

			score := scorer.calculateInTransitScore(profile, result)
			if score != tt.wantScore {
				t.Errorf("In-transit score = %d, want %d", score, tt.wantScore)
			}
		})
	}
}

func TestComplianceScorer_KeyManagementScore(t *testing.T) {
	scorer := NewComplianceScorer()

	tests := []struct {
		name      string
		status    models.EncryptionStatus
		key       *models.EncryptionKey
		wantScore int
	}{
		{
			name:      "no encryption",
			status:    models.EncryptionNone,
			key:       nil,
			wantScore: 0,
		},
		{
			name:      "SSE (AWS managed)",
			status:    models.EncryptionSSE,
			key:       nil,
			wantScore: 50,
		},
		{
			name:      "KMS without key info",
			status:    models.EncryptionSSEKMS,
			key:       nil,
			wantScore: 60,
		},
		{
			name:   "disabled key",
			status: models.EncryptionCMK,
			key: &models.EncryptionKey{
				Enabled: false,
			},
			wantScore: 0,
		},
		{
			name:   "AWS managed key with rotation",
			status: models.EncryptionSSEKMS,
			key: &models.EncryptionKey{
				Enabled:            true,
				RotationEnabled:    true,
				KeyManager:         "AWS",
				AllowsPublicAccess: false,
				KeyState:           models.KeyStateEnabled,
			},
			wantScore: 80, // 30 + 25 + 0 + 15 + 10
		},
		{
			name:   "customer managed key with all best practices",
			status: models.EncryptionCMK,
			key: &models.EncryptionKey{
				Enabled:            true,
				RotationEnabled:    true,
				KeyManager:         "CUSTOMER",
				AllowsPublicAccess: false,
				KeyState:           models.KeyStateEnabled,
			},
			wantScore: 100, // 30 + 25 + 20 + 15 + 10
		},
		{
			name:   "key pending deletion",
			status: models.EncryptionCMK,
			key: &models.EncryptionKey{
				Enabled:            true,
				RotationEnabled:    true,
				KeyManager:         "CUSTOMER",
				AllowsPublicAccess: false,
				KeyState:           models.KeyStatePendingDeletion,
			},
			wantScore: 90, // 30 + 25 + 20 + 15 + 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &AssetEncryptionProfile{
				EncryptionStatus: tt.status,
				Key:              tt.key,
			}
			result := &ComplianceResult{
				Findings:        []EncryptionFinding{},
				Recommendations: []string{},
			}

			score := scorer.calculateKeyManagementScore(profile, result)
			if score != tt.wantScore {
				t.Errorf("Key management score = %d, want %d", score, tt.wantScore)
			}
		})
	}
}

func TestComplianceScorer_CalculateGrade(t *testing.T) {
	scorer := NewComplianceScorer()

	tests := []struct {
		score int
		grade string
	}{
		{100, "A"},
		{95, "A"},
		{90, "A"},
		{89, "B"},
		{85, "B"},
		{80, "B"},
		{79, "C"},
		{75, "C"},
		{70, "C"},
		{69, "D"},
		{65, "D"},
		{60, "D"},
		{59, "F"},
		{50, "F"},
		{0, "F"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.score)), func(t *testing.T) {
			grade := scorer.calculateGrade(tt.score)
			if grade != tt.grade {
				t.Errorf("Grade for score %d = %q, want %q", tt.score, grade, tt.grade)
			}
		})
	}
}

func TestComplianceScorer_CustomWeights(t *testing.T) {
	// Custom weights: prioritize in-transit over at-rest
	customWeights := ScoringWeights{
		AtRest:        0.2,
		InTransit:     0.6,
		KeyManagement: 0.2,
	}

	scorer := NewComplianceScorerWithWeights(customWeights)

	// Profile with good at-rest but poor transit
	profileBadTransit := &AssetEncryptionProfile{
		EncryptionStatus: models.EncryptionCMK,
		TransitEncryption: &models.TransitEncryption{
			TLSEnabled: false,
		},
		Key: &models.EncryptionKey{
			Enabled:  true,
			KeyState: models.KeyStateEnabled,
		},
	}

	// Profile with poor at-rest but good transit
	profileGoodTransit := &AssetEncryptionProfile{
		EncryptionStatus: models.EncryptionSSE,
		TransitEncryption: &models.TransitEncryption{
			TLSEnabled:                    true,
			TLSVersion:                    "TLSv1.3",
			SupportsPerfectForwardSecrecy: true,
		},
	}

	resultBadTransit := scorer.CalculateComplianceScore(profileBadTransit)
	resultGoodTransit := scorer.CalculateComplianceScore(profileGoodTransit)

	// With custom weights prioritizing transit, good transit should score better
	if resultGoodTransit.Score <= resultBadTransit.Score {
		t.Errorf("With transit-heavy weights, good transit (%d) should score better than bad transit (%d)",
			resultGoodTransit.Score, resultBadTransit.Score)
	}
}

func TestComplianceScorer_FindingSeverities(t *testing.T) {
	scorer := NewComplianceScorer()

	// Test that critical issues generate critical findings
	criticalProfile := &AssetEncryptionProfile{
		EncryptionStatus: models.EncryptionCMK,
		TransitEncryption: &models.TransitEncryption{
			TLSEnabled: false, // Critical finding
		},
		Key: &models.EncryptionKey{
			Enabled:            true,
			AllowsPublicAccess: true, // Critical finding
			KeyState:           models.KeyStateEnabled,
		},
	}

	result := scorer.CalculateComplianceScore(criticalProfile)

	criticalCount := 0
	for _, finding := range result.Findings {
		if finding.Severity == models.SeverityCritical {
			criticalCount++
		}
	}

	if criticalCount < 2 {
		t.Errorf("Expected at least 2 critical findings, got %d", criticalCount)
	}
}

// Benchmark tests
func BenchmarkComplianceScorer_FullProfile(b *testing.B) {
	scorer := NewComplianceScorer()
	profile := &AssetEncryptionProfile{
		AssetID:            uuid.New(),
		EncryptionStatus:   models.EncryptionCMK,
		KeyRotationEnabled: true,
		Key: &models.EncryptionKey{
			Enabled:            true,
			RotationEnabled:    true,
			KeyManager:         "CUSTOMER",
			AllowsPublicAccess: false,
			KeyState:           models.KeyStateEnabled,
		},
		TransitEncryption: &models.TransitEncryption{
			TLSEnabled:                    true,
			TLSVersion:                    "TLSv1.3",
			SupportsPerfectForwardSecrecy: true,
			CertificateARN:                "arn:aws:acm:...",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scorer.CalculateComplianceScore(profile)
	}
}
