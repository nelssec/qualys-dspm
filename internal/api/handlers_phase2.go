package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qualys/dspm/internal/lineage"
	"github.com/qualys/dspm/internal/mlclassifier"
	"github.com/qualys/dspm/internal/models"
	"github.com/qualys/dspm/internal/remediation"
)

// =====================================================
// Data Lineage Handlers
// =====================================================

func (s *Server) getLineageOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	overview, err := s.lineageService.GetLineageOverview(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to get lineage overview", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get lineage overview")
		return
	}

	respondJSON(w, http.StatusOK, overview)
}

func (s *Server) getAssetLineage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	assetARN := chi.URLParam(r, "assetARN")
	if assetARN == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "assetARN is required")
		return
	}

	maxHops := 3
	if hopsStr := r.URL.Query().Get("max_hops"); hopsStr != "" {
		if parsed, err := strconv.Atoi(hopsStr); err == nil && parsed > 0 {
			maxHops = parsed
		}
	}

	graph, err := s.lineageService.GetAssetLineage(ctx, assetARN, maxHops)
	if err != nil {
		s.logger.Error("failed to get asset lineage", "error", err, "assetARN", assetARN)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get asset lineage")
		return
	}

	respondJSON(w, http.StatusOK, graph)
}

func (s *Server) findDataFlowPaths(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	req := &lineage.LineagePathRequest{
		SourceARN:      r.URL.Query().Get("source_arn"),
		DestinationARN: r.URL.Query().Get("dest_arn"),
		SensitiveOnly:  r.URL.Query().Get("sensitive_only") == "true",
	}

	if hopsStr := r.URL.Query().Get("max_hops"); hopsStr != "" {
		if parsed, err := strconv.Atoi(hopsStr); err == nil {
			req.MaxHops = parsed
		}
	}

	paths, err := s.lineageService.FindDataFlowPaths(ctx, accountID, req)
	if err != nil {
		s.logger.Error("failed to find data flow paths", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to find data flow paths")
		return
	}

	respondJSON(w, http.StatusOK, paths)
}

func (s *Server) getSensitiveDataFlows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	flows, err := s.lineageService.GetSensitiveDataFlows(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to get sensitive data flows", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get sensitive data flows")
		return
	}

	respondJSON(w, http.StatusOK, flows)
}

func (s *Server) triggerLineageScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		AccountID uuid.UUID `json:"account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	job := &models.ScanJob{
		AccountID: req.AccountID,
		ScanType:  "LINEAGE",
		Status:    models.ScanStatusPending,
	}
	if err := s.store.CreateScanJob(ctx, job); err != nil {
		s.logger.Error("failed to create lineage scan job", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to create scan job")
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"message": "Lineage scan triggered",
		"job_id":  job.ID,
	})
}

// =====================================================
// ML Classification Handlers
// =====================================================

func (s *Server) listMLModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	models, err := s.mlClassifier.ListMLModels(ctx)
	if err != nil {
		s.logger.Error("failed to list ML models", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list ML models")
		return
	}

	respondJSON(w, http.StatusOK, models)
}

func (s *Server) getMLModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	modelIDStr := chi.URLParam(r, "modelID")
	modelID, err := uuid.Parse(modelIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid modelID format")
		return
	}

	model, err := s.store.GetMLModel(ctx, modelID)
	if err != nil {
		s.logger.Error("failed to get ML model", "error", err, "modelID", modelID)
		respondError(w, http.StatusNotFound, "not_found", "ML model not found")
		return
	}

	respondJSON(w, http.StatusOK, model)
}

func (s *Server) getReviewQueue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := models.ReviewQueueStatusPending
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status = models.ReviewQueueStatus(statusStr)
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	items, err := s.mlClassifier.GetReviewQueue(ctx, status, limit)
	if err != nil {
		s.logger.Error("failed to get review queue", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get review queue")
		return
	}

	respondJSON(w, http.StatusOK, items)
}

func (s *Server) getReviewItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemIDStr := chi.URLParam(r, "itemID")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid itemID format")
		return
	}

	item, err := s.store.GetReviewQueueItem(ctx, itemID)
	if err != nil {
		s.logger.Error("failed to get review item", "error", err, "itemID", itemID)
		respondError(w, http.StatusNotFound, "not_found", "review item not found")
		return
	}

	respondJSON(w, http.StatusOK, item)
}

func (s *Server) resolveReviewItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemIDStr := chi.URLParam(r, "itemID")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid itemID format")
		return
	}

	var req mlclassifier.ReviewResolution
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	req.ItemID = itemID

	if err := s.mlClassifier.ResolveReviewItem(ctx, &req); err != nil {
		s.logger.Error("failed to resolve review item", "error", err, "itemID", itemID)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to resolve review item")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Review item resolved",
	})
}

func (s *Server) assignReviewItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemIDStr := chi.URLParam(r, "itemID")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid itemID format")
		return
	}

	var req struct {
		AssignedTo uuid.UUID `json:"assigned_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	item, err := s.store.GetReviewQueueItem(ctx, itemID)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "review item not found")
		return
	}

	item.AssignedTo = &req.AssignedTo
	item.Status = models.ReviewQueueStatusInReview

	if err := s.store.UpdateReviewQueueItem(ctx, item); err != nil {
		s.logger.Error("failed to assign review item", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to assign review item")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Review item assigned",
	})
}

func (s *Server) submitTrainingFeedback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req mlclassifier.FeedbackSubmission
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if err := s.mlClassifier.SubmitFeedback(ctx, &req); err != nil {
		s.logger.Error("failed to submit training feedback", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to submit training feedback")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{
		"message": "Training feedback submitted",
	})
}

func (s *Server) getFeedbackStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := s.mlClassifier.GetReviewQueueStats(ctx)
	if err != nil {
		s.logger.Error("failed to get feedback stats", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get feedback stats")
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// =====================================================
// AI Source Tracking Handlers
// =====================================================

func (s *Server) listAIServices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	services, err := s.aiTrackingService.ListAIServices(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to list AI services", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list AI services")
		return
	}

	respondJSON(w, http.StatusOK, services)
}

func (s *Server) getAIService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	serviceIDStr := chi.URLParam(r, "serviceID")
	serviceID, err := uuid.Parse(serviceIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid serviceID format")
		return
	}

	service, err := s.store.GetAIService(ctx, serviceID)
	if err != nil {
		s.logger.Error("failed to get AI service", "error", err, "serviceID", serviceID)
		respondError(w, http.StatusNotFound, "not_found", "AI service not found")
		return
	}

	respondJSON(w, http.StatusOK, service)
}

func (s *Server) listAIModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	aiModels, err := s.aiTrackingService.ListAIModels(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to list AI models", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list AI models")
		return
	}

	respondJSON(w, http.StatusOK, aiModels)
}

func (s *Server) getAIModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	modelIDStr := chi.URLParam(r, "modelID")
	modelID, err := uuid.Parse(modelIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid modelID format")
		return
	}

	model, err := s.aiTrackingService.GetAIModel(ctx, modelID)
	if err != nil {
		s.logger.Error("failed to get AI model", "error", err, "modelID", modelID)
		respondError(w, http.StatusNotFound, "not_found", "AI model not found")
		return
	}

	respondJSON(w, http.StatusOK, model)
}

func (s *Server) getModelTrainingData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	modelIDStr := chi.URLParam(r, "modelID")
	modelID, err := uuid.Parse(modelIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid modelID format")
		return
	}

	analysis, err := s.aiTrackingService.GetModelTrainingDataAnalysis(ctx, modelID)
	if err != nil {
		s.logger.Error("failed to get model training data", "error", err, "modelID", modelID)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get model training data")
		return
	}

	respondJSON(w, http.StatusOK, analysis)
}

func (s *Server) listAIProcessingEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	events, err := s.store.ListAIProcessingEvents(ctx, accountID, limit)
	if err != nil {
		s.logger.Error("failed to list AI processing events", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list AI processing events")
		return
	}

	respondJSON(w, http.StatusOK, events)
}

func (s *Server) getSensitiveDataAccess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	summary, err := s.aiTrackingService.GetSensitiveDataAccessSummary(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to get sensitive data access", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get sensitive data access")
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

func (s *Server) getAIRiskReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	report, err := s.aiTrackingService.GetAIRiskReport(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to get AI risk report", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get AI risk report")
		return
	}

	respondJSON(w, http.StatusOK, report)
}

func (s *Server) triggerAIScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		AccountID uuid.UUID `json:"account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	job := &models.ScanJob{
		AccountID: req.AccountID,
		ScanType:  "AI_SERVICES",
		Status:    models.ScanStatusPending,
	}
	if err := s.store.CreateScanJob(ctx, job); err != nil {
		s.logger.Error("failed to create AI scan job", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to create scan job")
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"message": "AI scan triggered",
		"job_id":  job.ID,
	})
}

// =====================================================
// Enhanced Encryption Visibility Handlers
// =====================================================

func (s *Server) getEncryptionOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	overview, err := s.encryptionService.GetEncryptionOverview(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to get encryption overview", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get encryption overview")
		return
	}

	respondJSON(w, http.StatusOK, overview)
}

func (s *Server) listEncryptionKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	keys, err := s.encryptionService.ListEncryptionKeys(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to list encryption keys", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list encryption keys")
		return
	}

	respondJSON(w, http.StatusOK, keys)
}

func (s *Server) getEncryptionKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	keyIDStr := chi.URLParam(r, "keyID")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid keyID format")
		return
	}

	key, err := s.encryptionService.GetEncryptionKey(ctx, keyID)
	if err != nil {
		s.logger.Error("failed to get encryption key", "error", err, "keyID", keyID)
		respondError(w, http.StatusNotFound, "not_found", "encryption key not found")
		return
	}

	respondJSON(w, http.StatusOK, key)
}

func (s *Server) getKeyUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	keyIDStr := chi.URLParam(r, "keyID")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid keyID format")
		return
	}

	summary, err := s.encryptionService.GetKeyUsageSummary(ctx, keyID)
	if err != nil {
		s.logger.Error("failed to get key usage", "error", err, "keyID", keyID)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get key usage")
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

func (s *Server) getKeyAssets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	keyIDStr := chi.URLParam(r, "keyID")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid keyID format")
		return
	}

	summary, err := s.encryptionService.GetKeyUsageSummary(ctx, keyID)
	if err != nil {
		s.logger.Error("failed to get key assets", "error", err, "keyID", keyID)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get key assets")
		return
	}

	respondJSON(w, http.StatusOK, summary.Assets)
}

func (s *Server) getComplianceSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	summary, err := s.encryptionService.GetComplianceSummary(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to get compliance summary", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get compliance summary")
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

func (s *Server) getAssetComplianceScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	assetIDStr := chi.URLParam(r, "assetID")
	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid assetID format")
		return
	}

	compliance, err := s.encryptionService.GetAssetComplianceScore(ctx, assetID)
	if err != nil {
		s.logger.Error("failed to get asset compliance score", "error", err, "assetID", assetID)
		respondError(w, http.StatusNotFound, "not_found", "compliance score not found")
		return
	}

	respondJSON(w, http.StatusOK, compliance)
}

func (s *Server) getAccountComplianceScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := chi.URLParam(r, "accountID")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid accountID format")
		return
	}

	compliance, err := s.store.ListEncryptionCompliance(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to get account compliance", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get account compliance")
		return
	}

	// Calculate aggregate score
	var totalScore int
	var criticalFindings int
	gradeCount := make(map[string]int)

	for _, c := range compliance {
		totalScore += c.ComplianceScore
		criticalFindings += c.CriticalFindings
		gradeCount[c.Grade]++
	}

	avgScore := 0.0
	if len(compliance) > 0 {
		avgScore = float64(totalScore) / float64(len(compliance))
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"account_id":        accountID,
		"assets_evaluated":  len(compliance),
		"average_score":     avgScore,
		"critical_findings": criticalFindings,
		"grade_distribution": gradeCount,
	})
}

func (s *Server) getComplianceRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	compliance, err := s.store.ListEncryptionCompliance(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to get compliance", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get compliance")
		return
	}

	// Aggregate all recommendations
	recommendationSet := make(map[string]bool)
	for _, c := range compliance {
		for _, rec := range c.Recommendations {
			recommendationSet[rec] = true
		}
	}

	recommendations := make([]string, 0, len(recommendationSet))
	for rec := range recommendationSet {
		recommendations = append(recommendations, rec)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"account_id":      accountID,
		"recommendations": recommendations,
	})
}

func (s *Server) listTransitEncryption(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	transit, err := s.encryptionService.ListTransitEncryption(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to list transit encryption", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list transit encryption")
		return
	}

	respondJSON(w, http.StatusOK, transit)
}

func (s *Server) getAssetTransitEncryption(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	assetIDStr := chi.URLParam(r, "assetID")
	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid assetID format")
		return
	}

	transit, err := s.store.GetTransitEncryption(ctx, assetID)
	if err != nil {
		s.logger.Error("failed to get asset transit encryption", "error", err, "assetID", assetID)
		respondError(w, http.StatusNotFound, "not_found", "transit encryption not found")
		return
	}

	respondJSON(w, http.StatusOK, transit)
}

func (s *Server) triggerEncryptionScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		AccountID uuid.UUID `json:"account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	job := &models.ScanJob{
		AccountID: req.AccountID,
		ScanType:  "ENCRYPTION",
		Status:    models.ScanStatusPending,
	}
	if err := s.store.CreateScanJob(ctx, job); err != nil {
		s.logger.Error("failed to create encryption scan job", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to create scan job")
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"message": "Encryption scan triggered",
		"job_id":  job.ID,
	})
}

// =====================================================
// Remediation Handlers
// =====================================================

func (s *Server) listRemediationActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	// Parse optional status filter
	var status *remediation.ActionStatus
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		s := remediation.ActionStatus(statusStr)
		status = &s
	}

	limit := 50
	offset := 0

	actions, total, err := s.remediationService.ListActions(ctx, accountID, status, limit, offset)
	if err != nil {
		s.logger.Error("failed to list remediation actions", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list remediation actions")
		return
	}

	respondJSONWithMeta(w, http.StatusOK, actions, &apiMeta{
		Total: total,
		Limit: limit,
	})
}

func (s *Server) createRemediationAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req remediation.CreateActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	action, err := s.remediationService.CreateAction(ctx, req)
	if err != nil {
		s.logger.Error("failed to create remediation action", "error", err)
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, action)
}

func (s *Server) getRemediationSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "missing_param", "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid account_id format")
		return
	}

	summary, err := s.remediationService.GetActionSummary(ctx, accountID)
	if err != nil {
		s.logger.Error("failed to get remediation summary", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get remediation summary")
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

func (s *Server) getRemediationDefinitions(w http.ResponseWriter, r *http.Request) {
	definitions := s.remediationService.GetActionDefinitions()
	respondJSON(w, http.StatusOK, definitions)
}

func (s *Server) getRemediationPlaybooks(w http.ResponseWriter, r *http.Request) {
	playbooks := s.remediationService.GetPlaybooks()
	respondJSON(w, http.StatusOK, playbooks)
}

func (s *Server) getRemediationAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actionIDStr := chi.URLParam(r, "actionID")
	actionID, err := uuid.Parse(actionIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid actionID format")
		return
	}

	action, err := s.remediationService.GetAction(ctx, actionID)
	if err != nil {
		s.logger.Error("failed to get remediation action", "error", err, "actionID", actionID)
		respondError(w, http.StatusNotFound, "not_found", "remediation action not found")
		return
	}

	respondJSON(w, http.StatusOK, action)
}

func (s *Server) approveRemediationAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actionIDStr := chi.URLParam(r, "actionID")
	actionID, err := uuid.Parse(actionIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid actionID format")
		return
	}

	var req remediation.ApproveActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.ApprovedBy = "admin" // Default if not provided
	}

	action, err := s.remediationService.ApproveAction(ctx, actionID, req)
	if err != nil {
		s.logger.Error("failed to approve remediation action", "error", err, "actionID", actionID)
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, action)
}

func (s *Server) rejectRemediationAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actionIDStr := chi.URLParam(r, "actionID")
	actionID, err := uuid.Parse(actionIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid actionID format")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Reason = "Rejected by admin"
	}

	action, err := s.remediationService.RejectAction(ctx, actionID, req.Reason)
	if err != nil {
		s.logger.Error("failed to reject remediation action", "error", err, "actionID", actionID)
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, action)
}

func (s *Server) executeRemediationAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actionIDStr := chi.URLParam(r, "actionID")
	actionID, err := uuid.Parse(actionIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid actionID format")
		return
	}

	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Provider = "aws" // Default provider
	}

	action, err := s.remediationService.ExecuteAction(ctx, actionID, req.Provider)
	if err != nil {
		s.logger.Error("failed to execute remediation action", "error", err, "actionID", actionID)
		respondError(w, http.StatusBadRequest, "execution_failed", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, action)
}

func (s *Server) rollbackRemediationAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actionIDStr := chi.URLParam(r, "actionID")
	actionID, err := uuid.Parse(actionIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid actionID format")
		return
	}

	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Provider = "aws"
	}

	action, err := s.remediationService.RollbackAction(ctx, actionID, req.Provider)
	if err != nil {
		s.logger.Error("failed to rollback remediation action", "error", err, "actionID", actionID)
		respondError(w, http.StatusBadRequest, "rollback_failed", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, action)
}

func (s *Server) listAssetRemediations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	assetIDStr := chi.URLParam(r, "assetID")
	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_param", "invalid assetID format")
		return
	}

	actions, err := s.remediationService.ListActionsForAsset(ctx, assetID)
	if err != nil {
		s.logger.Error("failed to list asset remediations", "error", err, "assetID", assetID)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list asset remediations")
		return
	}

	respondJSON(w, http.StatusOK, actions)
}
