package anomaly

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Store defines the interface for anomaly persistence
type Store interface {
	// Anomaly operations
	CreateAnomaly(ctx context.Context, anomaly *Anomaly) error
	GetAnomaly(ctx context.Context, id uuid.UUID) (*Anomaly, error)
	UpdateAnomaly(ctx context.Context, anomaly *Anomaly) error
	ListAnomalies(ctx context.Context, accountID uuid.UUID, status *AnomalyStatus, anomalyType *AnomalyType, limit, offset int) ([]Anomaly, int, error)
	GetAnomalySummary(ctx context.Context, accountID uuid.UUID) (*AnomalySummary, error)

	// Baseline operations
	CreateBaseline(ctx context.Context, baseline *AccessBaseline) error
	GetBaseline(ctx context.Context, accountID uuid.UUID, principalID string) (*AccessBaseline, error)
	ListBaselines(ctx context.Context, accountID uuid.UUID) ([]AccessBaseline, error)
	UpdateBaseline(ctx context.Context, baseline *AccessBaseline) error
	DeleteBaseline(ctx context.Context, id uuid.UUID) error

	// Threat score operations
	CreateThreatScore(ctx context.Context, score *ThreatScore) error
	GetThreatScore(ctx context.Context, accountID uuid.UUID, principalID string) (*ThreatScore, error)
	ListThreatScores(ctx context.Context, accountID uuid.UUID, minScore float64, limit, offset int) ([]ThreatScore, int, error)
	UpdateThreatScore(ctx context.Context, score *ThreatScore) error
}

// AccessEventSource provides access events for analysis
type AccessEventSource interface {
	GetAccessEvents(ctx context.Context, accountID uuid.UUID, since time.Time) ([]AccessEvent, error)
}

// Service provides anomaly detection and management capabilities
type Service struct {
	store     Store
	detector  *Detector
	logger    *slog.Logger
}

// NewService creates a new anomaly detection service
func NewService(store Store, logger *slog.Logger) *Service {
	return &Service{
		store:    store,
		detector: NewDetector(),
		logger:   logger,
	}
}

// GetAnomaly retrieves an anomaly by ID
func (s *Service) GetAnomaly(ctx context.Context, id uuid.UUID) (*Anomaly, error) {
	return s.store.GetAnomaly(ctx, id)
}

// ListAnomalies lists anomalies with optional filtering
func (s *Service) ListAnomalies(ctx context.Context, accountID uuid.UUID, status *AnomalyStatus, anomalyType *AnomalyType, limit, offset int) ([]Anomaly, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListAnomalies(ctx, accountID, status, anomalyType, limit, offset)
}

// UpdateAnomalyStatus updates the status of an anomaly
func (s *Service) UpdateAnomalyStatus(ctx context.Context, id uuid.UUID, req UpdateAnomalyStatusRequest) (*Anomaly, error) {
	anomaly, err := s.store.GetAnomaly(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting anomaly: %w", err)
	}

	anomaly.Status = req.Status
	anomaly.UpdatedAt = time.Now()

	if req.Status == StatusResolved || req.Status == StatusFalsePositive {
		now := time.Now()
		anomaly.ResolvedAt = &now
		anomaly.ResolvedBy = req.ResolvedBy
		anomaly.Resolution = req.Resolution
	}

	if err := s.store.UpdateAnomaly(ctx, anomaly); err != nil {
		return nil, fmt.Errorf("updating anomaly: %w", err)
	}

	s.logger.Info("anomaly status updated",
		"anomaly_id", id,
		"new_status", req.Status)

	return anomaly, nil
}

// GetAnomalySummary returns a summary of anomalies for an account
func (s *Service) GetAnomalySummary(ctx context.Context, accountID uuid.UUID) (*AnomalySummary, error) {
	return s.store.GetAnomalySummary(ctx, accountID)
}

// BuildBaseline builds access baselines for an account
func (s *Service) BuildBaseline(ctx context.Context, req BaselineRequest, events []AccessEvent) (int, error) {
	if req.DaysToAnalyze <= 0 {
		req.DaysToAnalyze = 30
	}

	cutoff := time.Now().AddDate(0, 0, -req.DaysToAnalyze)

	// Filter events within the time range
	var filteredEvents []AccessEvent
	for _, event := range events {
		if event.Timestamp.After(cutoff) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	// Group events by principal
	eventsByPrincipal := make(map[string][]AccessEvent)
	principalInfo := make(map[string]struct{ pType, pName string })

	for _, event := range filteredEvents {
		eventsByPrincipal[event.PrincipalID] = append(eventsByPrincipal[event.PrincipalID], event)
		principalInfo[event.PrincipalID] = struct{ pType, pName string }{event.PrincipalType, event.PrincipalName}
	}

	// Build baseline for each principal (or specific principal if requested)
	var count int
	for principalID, principalEvents := range eventsByPrincipal {
		if req.PrincipalID != "" && principalID != req.PrincipalID {
			continue
		}

		info := principalInfo[principalID]
		baseline := s.detector.BuildBaseline(principalEvents, principalID, info.pType, info.pName, req.AccountID)
		if baseline == nil {
			continue
		}

		// Check if baseline already exists
		existing, _ := s.store.GetBaseline(ctx, req.AccountID, principalID)
		if existing != nil {
			baseline.ID = existing.ID
			if err := s.store.UpdateBaseline(ctx, baseline); err != nil {
				s.logger.Error("failed to update baseline", "error", err, "principal_id", principalID)
				continue
			}
		} else {
			if err := s.store.CreateBaseline(ctx, baseline); err != nil {
				s.logger.Error("failed to create baseline", "error", err, "principal_id", principalID)
				continue
			}
		}

		count++
	}

	s.logger.Info("baselines built",
		"account_id", req.AccountID,
		"baseline_count", count)

	return count, nil
}

// GetBaseline retrieves a baseline for a principal
func (s *Service) GetBaseline(ctx context.Context, accountID uuid.UUID, principalID string) (*AccessBaseline, error) {
	return s.store.GetBaseline(ctx, accountID, principalID)
}

// ListBaselines lists all baselines for an account
func (s *Service) ListBaselines(ctx context.Context, accountID uuid.UUID) ([]AccessBaseline, error) {
	return s.store.ListBaselines(ctx, accountID)
}

// DetectAnomalies analyzes events and detects anomalies
func (s *Service) DetectAnomalies(ctx context.Context, accountID uuid.UUID, events []AccessEvent) ([]Anomaly, error) {
	// Load baselines
	baselines, err := s.store.ListBaselines(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("loading baselines: %w", err)
	}

	// Create baseline map
	baselineMap := make(map[string]*AccessBaseline)
	for i := range baselines {
		baselineMap[baselines[i].PrincipalID] = &baselines[i]
	}

	// Detect anomalies
	detected := s.detector.DetectAnomalies(events, baselineMap)

	// Store detected anomalies
	var stored []Anomaly
	for _, anomaly := range detected {
		anomaly.AccountID = accountID
		if err := s.store.CreateAnomaly(ctx, &anomaly); err != nil {
			s.logger.Error("failed to store anomaly", "error", err)
			continue
		}
		stored = append(stored, anomaly)
	}

	s.logger.Info("anomalies detected",
		"account_id", accountID,
		"anomaly_count", len(stored))

	return stored, nil
}

// GetThreatScores retrieves threat scores for an account
func (s *Service) GetThreatScores(ctx context.Context, accountID uuid.UUID, minScore float64, limit, offset int) ([]ThreatScore, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListThreatScores(ctx, accountID, minScore, limit, offset)
}

// CalculateThreatScores calculates and updates threat scores
func (s *Service) CalculateThreatScores(ctx context.Context, accountID uuid.UUID, recentDays int) ([]ThreatScore, error) {
	if recentDays <= 0 {
		recentDays = 30
	}

	// Get recent anomalies
	anomalies, _, err := s.store.ListAnomalies(ctx, accountID, nil, nil, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("listing anomalies: %w", err)
	}

	// Group anomalies by principal
	principalAnomalies := make(map[string][]Anomaly)
	principalInfo := make(map[string]struct{ pType, pName string })

	for _, a := range anomalies {
		principalAnomalies[a.PrincipalID] = append(principalAnomalies[a.PrincipalID], a)
		principalInfo[a.PrincipalID] = struct{ pType, pName string }{a.PrincipalType, a.PrincipalName}
	}

	// Calculate scores
	var scores []ThreatScore
	for principalID, pAnomalies := range principalAnomalies {
		info := principalInfo[principalID]
		score := s.detector.CalculateThreatScore(principalID, info.pType, info.pName, accountID, pAnomalies, recentDays)

		// Check if score already exists
		existing, _ := s.store.GetThreatScore(ctx, accountID, principalID)
		if existing != nil {
			score.ID = existing.ID
			if err := s.store.UpdateThreatScore(ctx, &score); err != nil {
				s.logger.Error("failed to update threat score", "error", err, "principal_id", principalID)
				continue
			}
		} else {
			if err := s.store.CreateThreatScore(ctx, &score); err != nil {
				s.logger.Error("failed to create threat score", "error", err, "principal_id", principalID)
				continue
			}
		}

		scores = append(scores, score)
	}

	s.logger.Info("threat scores calculated",
		"account_id", accountID,
		"score_count", len(scores))

	return scores, nil
}

// GenerateReport generates an anomaly report
func (s *Service) GenerateReport(ctx context.Context, accountID uuid.UUID, periodDays int) (*AnomalyReport, error) {
	if periodDays <= 0 {
		periodDays = 30
	}

	summary, err := s.store.GetAnomalySummary(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("getting summary: %w", err)
	}

	scores, _, err := s.store.ListThreatScores(ctx, accountID, 0, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("getting threat scores: %w", err)
	}

	// Find high-risk principals
	var highRisk []PrincipalAnomaly
	for _, score := range scores {
		if score.RiskLevel == SeverityHigh || score.RiskLevel == SeverityCritical {
			highRisk = append(highRisk, PrincipalAnomaly{
				PrincipalID:   score.PrincipalID,
				PrincipalName: score.PrincipalName,
				AnomalyCount:  score.RecentAnomalies,
				ThreatScore:   score.Score,
				RiskLevel:     score.RiskLevel,
			})
		}
	}

	// Get critical anomalies
	criticalStatus := AnomalyStatus(StatusNew)
	criticalType := AnomalyType("")
	critical, _, err := s.store.ListAnomalies(ctx, accountID, &criticalStatus, &criticalType, 20, 0)
	if err != nil {
		critical = []Anomaly{}
	}

	// Filter to critical severity
	var criticalAnomalies []Anomaly
	for _, a := range critical {
		if a.Severity == SeverityCritical || a.Severity == SeverityHigh {
			criticalAnomalies = append(criticalAnomalies, a)
		}
	}

	// Generate recommendations
	recommendations := s.generateRecommendations(summary, scores, criticalAnomalies)

	report := &AnomalyReport{
		GeneratedAt:        time.Now(),
		ReportPeriod:       fmt.Sprintf("Last %d days", periodDays),
		AccountID:          accountID,
		Summary:            *summary,
		ThreatScores:       scores,
		HighRiskPrincipals: highRisk,
		CriticalAnomalies:  criticalAnomalies,
		Recommendations:    recommendations,
	}

	return report, nil
}

// generateRecommendations generates security recommendations based on anomaly analysis
func (s *Service) generateRecommendations(summary *AnomalySummary, scores []ThreatScore, critical []Anomaly) []string {
	var recommendations []string

	// Check for high volume of new anomalies
	if summary.NewCount > 10 {
		recommendations = append(recommendations, "High volume of new anomalies detected. Consider increasing monitoring frequency and reviewing access policies.")
	}

	// Check for bulk download patterns
	if bulkCount, ok := summary.ByType["BULK_DOWNLOAD"]; ok && bulkCount > 0 {
		recommendations = append(recommendations, "Bulk data download events detected. Review data loss prevention policies and consider implementing rate limiting.")
	}

	// Check for off-hours access
	if offHoursCount, ok := summary.ByType["OFF_HOURS_ACCESS"]; ok && offHoursCount > 5 {
		recommendations = append(recommendations, "Significant off-hours access detected. Consider implementing time-based access controls for sensitive data.")
	}

	// Check for high-risk principals
	highRiskCount := 0
	for _, score := range scores {
		if score.RiskLevel == SeverityCritical || score.RiskLevel == SeverityHigh {
			highRiskCount++
		}
	}
	if highRiskCount > 0 {
		recommendations = append(recommendations, fmt.Sprintf("%d principals with elevated threat scores. Conduct detailed review of their access patterns.", highRiskCount))
	}

	// Check for geographic anomalies
	if geoCount, ok := summary.ByType["GEO_ANOMALY"]; ok && geoCount > 0 {
		recommendations = append(recommendations, "Geographic anomalies detected. Review IP allowlists and consider implementing location-based access controls.")
	}

	// Default recommendation if none
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Continue regular monitoring. No critical issues require immediate attention.")
	}

	return recommendations
}

// GetDetectionRules returns the current detection rules
func (s *Service) GetDetectionRules() []DetectionRule {
	return GetDefaultDetectionRules()
}
