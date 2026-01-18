package anomaly

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// Detector implements anomaly detection algorithms
type Detector struct {
	rules []DetectionRule
}

// NewDetector creates a new anomaly detector
func NewDetector() *Detector {
	return &Detector{
		rules: GetDefaultDetectionRules(),
	}
}

// SetRules sets the detection rules
func (d *Detector) SetRules(rules []DetectionRule) {
	d.rules = rules
}

// DetectAnomalies analyzes access events against baselines
func (d *Detector) DetectAnomalies(events []AccessEvent, baselines map[string]*AccessBaseline) []Anomaly {
	var anomalies []Anomaly

	// Group events by principal
	eventsByPrincipal := make(map[string][]AccessEvent)
	for _, event := range events {
		eventsByPrincipal[event.PrincipalID] = append(eventsByPrincipal[event.PrincipalID], event)
	}

	for principalID, principalEvents := range eventsByPrincipal {
		baseline := baselines[principalID]

		for _, rule := range d.rules {
			if !rule.Enabled {
				continue
			}

			var detected []Anomaly

			switch rule.AnomalyType {
			case AnomalyVolumeSpike:
				detected = d.detectVolumeSpike(principalEvents, baseline, rule)
			case AnomalyFrequencySpike:
				detected = d.detectFrequencySpike(principalEvents, baseline, rule)
			case AnomalyNewDestination:
				detected = d.detectNewDestination(principalEvents, baseline, rule)
			case AnomalyOffHoursAccess:
				detected = d.detectOffHoursAccess(principalEvents, baseline, rule)
			case AnomalyBulkDownload:
				detected = d.detectBulkDownload(principalEvents, baseline, rule)
			case AnomalyGeoAnomaly:
				detected = d.detectGeoAnomaly(principalEvents, baseline, rule)
			}

			anomalies = append(anomalies, detected...)
		}
	}

	return anomalies
}

// detectVolumeSpike detects unusual data volume
func (d *Detector) detectVolumeSpike(events []AccessEvent, baseline *AccessBaseline, rule DetectionRule) []Anomaly {
	var anomalies []Anomaly

	if len(events) == 0 {
		return anomalies
	}

	// Calculate total volume
	var totalVolume int64
	for _, event := range events {
		totalVolume += event.DataVolumeBytes
	}

	// Check minimum volume threshold
	minVolume := int64(10485760) // 10MB default
	if v, ok := rule.Conditions["min_volume_bytes"].(int); ok {
		minVolume = int64(v)
	}
	if totalVolume < minVolume {
		return anomalies
	}

	// Calculate deviation if baseline exists
	var deviationFactor float64
	var baselineValue float64

	if baseline != nil && baseline.StdDevDataVolume > 0 {
		baselineValue = baseline.AvgDataVolumeBytes
		deviationFactor = (float64(totalVolume) - baselineValue) / baseline.StdDevDataVolume

		if deviationFactor < rule.Threshold {
			return anomalies
		}
	} else {
		// No baseline - flag if over a large threshold
		if totalVolume < 104857600 { // 100MB
			return anomalies
		}
		deviationFactor = float64(totalVolume) / float64(minVolume)
	}

	// Create anomaly
	anomaly := Anomaly{
		ID:              uuid.New(),
		AccountID:       events[0].AccountID,
		PrincipalID:     events[0].PrincipalID,
		PrincipalType:   events[0].PrincipalType,
		PrincipalName:   events[0].PrincipalName,
		AnomalyType:     AnomalyVolumeSpike,
		Status:          StatusNew,
		Severity:        rule.Severity,
		Title:           "Unusual Data Volume Detected",
		Description:     "Data access volume significantly exceeds normal baseline",
		BaselineValue:   baselineValue,
		ObservedValue:   float64(totalVolume),
		DeviationFactor: deviationFactor,
		DetectedAt:      time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Details: map[string]interface{}{
			"total_bytes":     totalVolume,
			"event_count":     len(events),
			"time_period":     "analysis_window",
		},
	}

	anomalies = append(anomalies, anomaly)
	return anomalies
}

// detectFrequencySpike detects unusual access frequency
func (d *Detector) detectFrequencySpike(events []AccessEvent, baseline *AccessBaseline, rule DetectionRule) []Anomaly {
	var anomalies []Anomaly

	accessCount := float64(len(events))

	// Check minimum access count
	minCount := float64(10)
	if v, ok := rule.Conditions["min_access_count"].(int); ok {
		minCount = float64(v)
	}
	if accessCount < minCount {
		return anomalies
	}

	var deviationFactor float64
	var baselineValue float64

	if baseline != nil && baseline.StdDevAccessCount > 0 {
		baselineValue = baseline.AvgDailyAccessCount
		deviationFactor = (accessCount - baselineValue) / baseline.StdDevAccessCount

		if deviationFactor < rule.Threshold {
			return anomalies
		}
	} else {
		// No baseline - flag if significantly high
		if accessCount < 50 {
			return anomalies
		}
		deviationFactor = accessCount / minCount
	}

	if len(events) == 0 {
		return anomalies
	}

	anomaly := Anomaly{
		ID:              uuid.New(),
		AccountID:       events[0].AccountID,
		PrincipalID:     events[0].PrincipalID,
		PrincipalType:   events[0].PrincipalType,
		PrincipalName:   events[0].PrincipalName,
		AnomalyType:     AnomalyFrequencySpike,
		Status:          StatusNew,
		Severity:        rule.Severity,
		Title:           "Unusual Access Frequency Detected",
		Description:     "Access frequency significantly exceeds normal baseline",
		BaselineValue:   baselineValue,
		ObservedValue:   accessCount,
		DeviationFactor: deviationFactor,
		DetectedAt:      time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Details: map[string]interface{}{
			"access_count": int(accessCount),
		},
	}

	anomalies = append(anomalies, anomaly)
	return anomalies
}

// detectNewDestination detects data flow to new destinations
func (d *Detector) detectNewDestination(events []AccessEvent, baseline *AccessBaseline, rule DetectionRule) []Anomaly {
	var anomalies []Anomaly

	if baseline == nil || len(baseline.CommonAssets) == 0 {
		return anomalies
	}

	// Create set of common assets
	commonAssets := make(map[string]bool)
	for _, asset := range baseline.CommonAssets {
		commonAssets[asset] = true
	}

	// Find new assets
	newAssets := make(map[string]AccessEvent)
	for _, event := range events {
		assetID := event.AssetID.String()
		if !commonAssets[assetID] {
			newAssets[assetID] = event
		}
	}

	// Create anomalies for new destinations
	for _, event := range newAssets {
		anomaly := Anomaly{
			ID:              uuid.New(),
			AccountID:       event.AccountID,
			AssetID:         &event.AssetID,
			PrincipalID:     event.PrincipalID,
			PrincipalType:   event.PrincipalType,
			PrincipalName:   event.PrincipalName,
			AnomalyType:     AnomalyNewDestination,
			Status:          StatusNew,
			Severity:        rule.Severity,
			Title:           "New Data Destination Detected",
			Description:     "Data accessed from a resource not in normal baseline",
			DetectedAt:      time.Now(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			Details: map[string]interface{}{
				"asset_id":    event.AssetID.String(),
				"asset_name":  event.AssetName,
				"operation":   event.Operation,
				"source_ip":   event.SourceIP,
			},
		}
		anomalies = append(anomalies, anomaly)
	}

	return anomalies
}

// detectOffHoursAccess detects access outside normal working hours
func (d *Detector) detectOffHoursAccess(events []AccessEvent, baseline *AccessBaseline, rule DetectionRule) []Anomaly {
	var anomalies []Anomaly

	offHoursStart := 22 // 10 PM
	offHoursEnd := 6    // 6 AM

	if v, ok := rule.Conditions["off_hours_start"].(int); ok {
		offHoursStart = v
	}
	if v, ok := rule.Conditions["off_hours_end"].(int); ok {
		offHoursEnd = v
	}

	// If baseline has normal hours, use those
	normalHours := make(map[int]bool)
	if baseline != nil && len(baseline.NormalAccessHours) > 0 {
		for _, hour := range baseline.NormalAccessHours {
			normalHours[hour] = true
		}
	} else {
		// Default: 6 AM to 10 PM are normal hours
		for h := offHoursEnd; h < offHoursStart; h++ {
			normalHours[h] = true
		}
	}

	// Find off-hours events
	for _, event := range events {
		hour := event.Timestamp.Hour()
		if !normalHours[hour] {
			anomaly := Anomaly{
				ID:              uuid.New(),
				AccountID:       event.AccountID,
				AssetID:         &event.AssetID,
				PrincipalID:     event.PrincipalID,
				PrincipalType:   event.PrincipalType,
				PrincipalName:   event.PrincipalName,
				AnomalyType:     AnomalyOffHoursAccess,
				Status:          StatusNew,
				Severity:        rule.Severity,
				Title:           "Off-Hours Data Access Detected",
				Description:     "Data accessed outside normal working hours",
				DetectedAt:      time.Now(),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				Details: map[string]interface{}{
					"access_hour":    hour,
					"access_time":    event.Timestamp.Format(time.RFC3339),
					"asset_name":     event.AssetName,
					"operation":      event.Operation,
					"source_ip":      event.SourceIP,
				},
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// detectBulkDownload detects large-scale data extraction
func (d *Detector) detectBulkDownload(events []AccessEvent, baseline *AccessBaseline, rule DetectionRule) []Anomaly {
	var anomalies []Anomaly

	minVolume := int64(104857600) // 100MB default
	if v, ok := rule.Conditions["min_volume_bytes"].(int); ok {
		minVolume = int64(v)
	}

	timeWindow := 60 // 60 minutes default
	if v, ok := rule.Conditions["time_window_minutes"].(int); ok {
		timeWindow = v
	}

	if len(events) == 0 {
		return anomalies
	}

	// Group events into time windows and check volume
	windowDuration := time.Duration(timeWindow) * time.Minute

	// Sort events by time (assuming they're roughly sorted)
	windowStart := events[0].Timestamp
	var windowVolume int64
	var windowEvents []AccessEvent

	for _, event := range events {
		if event.Timestamp.Sub(windowStart) > windowDuration {
			// Check if current window exceeds threshold
			if windowVolume >= minVolume {
				anomaly := d.createBulkDownloadAnomaly(windowEvents, windowVolume, baseline, rule)
				anomalies = append(anomalies, anomaly)
			}

			// Start new window
			windowStart = event.Timestamp
			windowVolume = 0
			windowEvents = nil
		}

		windowVolume += event.DataVolumeBytes
		windowEvents = append(windowEvents, event)
	}

	// Check final window
	if windowVolume >= minVolume {
		anomaly := d.createBulkDownloadAnomaly(windowEvents, windowVolume, baseline, rule)
		anomalies = append(anomalies, anomaly)
	}

	return anomalies
}

func (d *Detector) createBulkDownloadAnomaly(events []AccessEvent, totalVolume int64, baseline *AccessBaseline, rule DetectionRule) Anomaly {
	var deviationFactor float64
	var baselineValue float64

	if baseline != nil && baseline.StdDevDataVolume > 0 {
		baselineValue = baseline.AvgDataVolumeBytes
		deviationFactor = (float64(totalVolume) - baselineValue) / baseline.StdDevDataVolume
	} else {
		deviationFactor = float64(totalVolume) / 104857600 // deviation from 100MB
	}

	// Collect unique assets
	assets := make(map[string]bool)
	for _, event := range events {
		assets[event.AssetName] = true
	}
	assetList := make([]string, 0, len(assets))
	for asset := range assets {
		assetList = append(assetList, asset)
	}

	return Anomaly{
		ID:              uuid.New(),
		AccountID:       events[0].AccountID,
		PrincipalID:     events[0].PrincipalID,
		PrincipalType:   events[0].PrincipalType,
		PrincipalName:   events[0].PrincipalName,
		AnomalyType:     AnomalyBulkDownload,
		Status:          StatusNew,
		Severity:        SeverityCritical,
		Title:           "Bulk Data Download Detected",
		Description:     "Large-scale data extraction detected within short time window",
		BaselineValue:   baselineValue,
		ObservedValue:   float64(totalVolume),
		DeviationFactor: deviationFactor,
		DetectedAt:      time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Details: map[string]interface{}{
			"total_bytes":   totalVolume,
			"event_count":   len(events),
			"assets":        assetList,
			"time_start":    events[0].Timestamp.Format(time.RFC3339),
			"time_end":      events[len(events)-1].Timestamp.Format(time.RFC3339),
		},
	}
}

// detectGeoAnomaly detects access from unusual geographic locations
func (d *Detector) detectGeoAnomaly(events []AccessEvent, baseline *AccessBaseline, rule DetectionRule) []Anomaly {
	var anomalies []Anomaly

	if baseline == nil || len(baseline.CommonGeoLocations) == 0 {
		return anomalies
	}

	// Create set of common locations
	commonLocations := make(map[string]bool)
	for _, loc := range baseline.CommonGeoLocations {
		commonLocations[loc] = true
	}

	// Find anomalous locations
	for _, event := range events {
		if event.GeoLocation != "" && !commonLocations[event.GeoLocation] {
			anomaly := Anomaly{
				ID:              uuid.New(),
				AccountID:       event.AccountID,
				AssetID:         &event.AssetID,
				PrincipalID:     event.PrincipalID,
				PrincipalType:   event.PrincipalType,
				PrincipalName:   event.PrincipalName,
				AnomalyType:     AnomalyGeoAnomaly,
				Status:          StatusNew,
				Severity:        rule.Severity,
				Title:           "Access from Unusual Location",
				Description:     "Data accessed from a geographic location not in normal baseline",
				DetectedAt:      time.Now(),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				Details: map[string]interface{}{
					"geo_location":      event.GeoLocation,
					"source_ip":         event.SourceIP,
					"asset_name":        event.AssetName,
					"operation":         event.Operation,
					"common_locations":  baseline.CommonGeoLocations,
				},
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// CalculateThreatScore calculates insider threat score for a principal
func (d *Detector) CalculateThreatScore(principalID, principalType, principalName string, accountID uuid.UUID, anomalies []Anomaly, recentDays int) ThreatScore {
	score := ThreatScore{
		ID:            uuid.New(),
		AccountID:     accountID,
		PrincipalID:   principalID,
		PrincipalType: principalType,
		PrincipalName: principalName,
		LastUpdated:   time.Now(),
		Details:       make(map[string]interface{}),
	}

	// Filter anomalies for this principal
	var principalAnomalies []Anomaly
	cutoff := time.Now().AddDate(0, 0, -recentDays)

	for _, a := range anomalies {
		if a.PrincipalID == principalID && a.DetectedAt.After(cutoff) {
			principalAnomalies = append(principalAnomalies, a)
		}
	}

	score.RecentAnomalies = len(principalAnomalies)

	// Calculate factor scores
	var factors []ThreatFactor

	// Volume anomalies factor
	volumeScore := d.calculateFactorScore(principalAnomalies, AnomalyVolumeSpike, AnomalyBulkDownload)
	factors = append(factors, ThreatFactor{
		Factor:      "data_volume",
		Weight:      0.3,
		Score:       volumeScore,
		Description: "Unusual data volume patterns",
	})

	// Access pattern anomalies factor
	accessScore := d.calculateFactorScore(principalAnomalies, AnomalyFrequencySpike, AnomalyOffHoursAccess)
	factors = append(factors, ThreatFactor{
		Factor:      "access_patterns",
		Weight:      0.25,
		Score:       accessScore,
		Description: "Unusual access frequency or timing",
	})

	// New destination factor
	destScore := d.calculateFactorScore(principalAnomalies, AnomalyNewDestination)
	factors = append(factors, ThreatFactor{
		Factor:      "new_destinations",
		Weight:      0.25,
		Score:       destScore,
		Description: "Access to unusual data destinations",
	})

	// Geographic anomalies factor
	geoScore := d.calculateFactorScore(principalAnomalies, AnomalyGeoAnomaly)
	factors = append(factors, ThreatFactor{
		Factor:      "geographic",
		Weight:      0.2,
		Score:       geoScore,
		Description: "Access from unusual locations",
	})

	score.Factors = factors

	// Calculate weighted total score
	var totalScore float64
	for _, f := range factors {
		totalScore += f.Score * f.Weight
	}
	score.Score = math.Min(100, totalScore)

	// Determine risk level
	switch {
	case score.Score >= 80:
		score.RiskLevel = SeverityCritical
	case score.Score >= 60:
		score.RiskLevel = SeverityHigh
	case score.Score >= 40:
		score.RiskLevel = SeverityMedium
	default:
		score.RiskLevel = SeverityLow
	}

	// Determine trend (would need historical data for real implementation)
	score.TrendDirection = "STABLE"

	return score
}

// calculateFactorScore calculates score for specific anomaly types
func (d *Detector) calculateFactorScore(anomalies []Anomaly, types ...AnomalyType) float64 {
	typeSet := make(map[AnomalyType]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	var count int
	var severitySum float64

	for _, a := range anomalies {
		if typeSet[a.AnomalyType] {
			count++
			switch a.Severity {
			case SeverityCritical:
				severitySum += 100
			case SeverityHigh:
				severitySum += 75
			case SeverityMedium:
				severitySum += 50
			case SeverityLow:
				severitySum += 25
			}
		}
	}

	if count == 0 {
		return 0
	}

	// Score based on count and average severity
	avgSeverity := severitySum / float64(count)
	countFactor := math.Min(1.0, float64(count)/5.0) // Cap at 5 anomalies

	return avgSeverity * countFactor
}

// BuildBaseline creates an access baseline from historical events
func (d *Detector) BuildBaseline(events []AccessEvent, principalID, principalType, principalName string, accountID uuid.UUID) *AccessBaseline {
	if len(events) == 0 {
		return nil
	}

	baseline := &AccessBaseline{
		ID:            uuid.New(),
		AccountID:     accountID,
		PrincipalID:   principalID,
		PrincipalType: principalType,
		PrincipalName: principalName,
		TimeWindow:    "DAILY",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Calculate time range
	var minTime, maxTime time.Time
	for i, event := range events {
		if i == 0 || event.Timestamp.Before(minTime) {
			minTime = event.Timestamp
		}
		if i == 0 || event.Timestamp.After(maxTime) {
			maxTime = event.Timestamp
		}
	}
	baseline.BaselinePeriodStart = minTime
	baseline.BaselinePeriodEnd = maxTime

	// Group by day
	dailyData := make(map[string]struct {
		accessCount int
		volume      int64
	})

	for _, event := range events {
		day := event.Timestamp.Format("2006-01-02")
		data := dailyData[day]
		data.accessCount++
		data.volume += event.DataVolumeBytes
		dailyData[day] = data
	}

	// Calculate averages and standard deviations
	var accessCounts, volumes []float64
	for _, data := range dailyData {
		accessCounts = append(accessCounts, float64(data.accessCount))
		volumes = append(volumes, float64(data.volume))
	}

	baseline.AvgDailyAccessCount = mean(accessCounts)
	baseline.StdDevAccessCount = stdDev(accessCounts)
	baseline.AvgDataVolumeBytes = mean(volumes)
	baseline.StdDevDataVolume = stdDev(volumes)

	// Set thresholds (3 standard deviations)
	baseline.AccessCountThreshold = baseline.AvgDailyAccessCount + (3 * baseline.StdDevAccessCount)
	baseline.DataVolumeThreshold = baseline.AvgDataVolumeBytes + (3 * baseline.StdDevDataVolume)

	// Collect common patterns
	hourCounts := make(map[int]int)
	dayCounts := make(map[int]int)
	assetCounts := make(map[string]int)
	opCounts := make(map[string]int)
	ipCounts := make(map[string]int)
	geoCounts := make(map[string]int)

	for _, event := range events {
		hourCounts[event.Timestamp.Hour()]++
		dayCounts[int(event.Timestamp.Weekday())]++
		assetCounts[event.AssetID.String()]++
		opCounts[event.Operation]++
		if event.SourceIP != "" {
			ipCounts[event.SourceIP]++
		}
		if event.GeoLocation != "" {
			geoCounts[event.GeoLocation]++
		}
	}

	// Find most common hours (top 80% of activity)
	baseline.NormalAccessHours = topKeys(hourCounts, 0.8)
	baseline.NormalAccessDays = topKeys(dayCounts, 0.8)
	baseline.CommonAssets = topStringKeys(assetCounts, 0.8)
	baseline.CommonOperations = topStringKeys(opCounts, 0.8)
	baseline.CommonSourceIPs = topStringKeys(ipCounts, 0.8)
	baseline.CommonGeoLocations = topStringKeys(geoCounts, 0.8)

	return baseline
}

// Helper functions

func mean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	var sum float64
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func stdDev(data []float64) float64 {
	if len(data) < 2 {
		return 0
	}
	m := mean(data)
	var sum float64
	for _, v := range data {
		sum += (v - m) * (v - m)
	}
	return math.Sqrt(sum / float64(len(data)-1))
}

func topKeys(counts map[int]int, threshold float64) []int {
	var total int
	for _, count := range counts {
		total += count
	}

	target := int(float64(total) * threshold)
	var current int
	var result []int

	// Sort by count descending (simple implementation)
	for current < target {
		maxKey := -1
		maxCount := 0
		for key, count := range counts {
			if count > maxCount {
				found := false
				for _, r := range result {
					if r == key {
						found = true
						break
					}
				}
				if !found {
					maxKey = key
					maxCount = count
				}
			}
		}
		if maxKey == -1 {
			break
		}
		result = append(result, maxKey)
		current += maxCount
	}

	return result
}

func topStringKeys(counts map[string]int, threshold float64) []string {
	var total int
	for _, count := range counts {
		total += count
	}

	target := int(float64(total) * threshold)
	var current int
	var result []string

	for current < target && len(result) < len(counts) {
		maxKey := ""
		maxCount := 0
		for key, count := range counts {
			if count > maxCount {
				found := false
				for _, r := range result {
					if r == key {
						found = true
						break
					}
				}
				if !found {
					maxKey = key
					maxCount = count
				}
			}
		}
		if maxKey == "" {
			break
		}
		result = append(result, maxKey)
		current += maxCount
	}

	return result
}
