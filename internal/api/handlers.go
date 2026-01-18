package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/qualys/dspm/internal/models"
	"github.com/qualys/dspm/internal/store"
)

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	var provider *models.Provider
	var status *string

	if p := r.URL.Query().Get("provider"); p != "" {
		prov := models.Provider(p)
		provider = &prov
	}
	if st := r.URL.Query().Get("status"); st != "" {
		status = &st
	}

	accounts, err := s.store.ListAccounts(r.Context(), provider, status)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, accounts)
}

type createAccountRequest struct {
	Provider        models.Provider `json:"provider"`
	ExternalID      string          `json:"external_id"`
	DisplayName     string          `json:"display_name"`
	ConnectorConfig models.JSONB    `json:"connector_config"`
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var req createAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if req.Provider == "" || req.ExternalID == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "provider and external_id are required")
		return
	}

	existing, err := s.store.GetAccountByExternalID(r.Context(), req.Provider, req.ExternalID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if existing != nil {
		respondError(w, http.StatusConflict, "account_exists", "Account already exists")
		return
	}

	account := &models.CloudAccount{
		Provider:        req.Provider,
		ExternalID:      req.ExternalID,
		DisplayName:     req.DisplayName,
		ConnectorConfig: req.ConnectorConfig,
		Status:          "active",
	}

	if err := s.store.CreateAccount(r.Context(), account); err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, account)
}

func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "accountID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid account ID")
		return
	}

	account, err := s.store.GetAccount(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if account == nil {
		respondError(w, http.StatusNotFound, "not_found", "Account not found")
		return
	}

	respondJSON(w, http.StatusOK, account)
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "accountID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid account ID")
		return
	}

	if err := s.store.DeleteAccount(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type triggerScanRequest struct {
	ScanType models.ScanType `json:"scan_type"`
	Scope    *scanScope      `json:"scope,omitempty"`
}

type scanScope struct {
	Buckets []string `json:"buckets,omitempty"`
	Regions []string `json:"regions,omitempty"`
}

func (s *Server) triggerScan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "accountID")
	accountID, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid account ID")
		return
	}

	var req triggerScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if req.ScanType == "" {
		req.ScanType = models.ScanTypeFull
	} else {
		// Normalize scan type to uppercase to match constants
		req.ScanType = models.ScanType(strings.ToUpper(string(req.ScanType)))
	}

	account, err := s.store.GetAccount(r.Context(), accountID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if account == nil {
		respondError(w, http.StatusNotFound, "not_found", "Account not found")
		return
	}

	job := &models.ScanJob{
		AccountID:   accountID,
		ScanType:    req.ScanType,
		TriggeredBy: "api",
	}

	if req.Scope != nil {
		job.ScanScope = models.JSONB{
			"buckets": req.Scope.Buckets,
			"regions": req.Scope.Regions,
		}
	}

	if err := s.store.CreateScanJob(r.Context(), job); err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	// Start the scan in the background
	s.scanExecutor.ExecuteScan(r.Context(), job, account)

	respondJSON(w, http.StatusAccepted, job)
}

func (s *Server) listAssets(w http.ResponseWriter, r *http.Request) {
	filters := store.ListAssetFilters{
		Limit:  100,
		Offset: 0,
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if limit, err := strconv.Atoi(l); err == nil {
			filters.Limit = limit
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if offset, err := strconv.Atoi(o); err == nil {
			filters.Offset = offset
		}
	}
	if accountID := r.URL.Query().Get("account_id"); accountID != "" {
		if id, err := uuid.Parse(accountID); err == nil {
			filters.AccountID = &id
		}
	}
	if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
		rt := models.ResourceType(resourceType)
		filters.ResourceType = &rt
	}
	if sensitivity := r.URL.Query().Get("sensitivity"); sensitivity != "" {
		sens := models.Sensitivity(sensitivity)
		filters.SensitivityLevel = &sens
	}
	if r.URL.Query().Get("public_only") == "true" {
		filters.PublicOnly = true
	}

	assets, total, err := s.store.ListAssets(r.Context(), filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSONWithMeta(w, http.StatusOK, assets, &apiMeta{
		Total:  total,
		Limit:  filters.Limit,
		Offset: filters.Offset,
	})
}

func (s *Server) getAsset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "assetID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid asset ID")
		return
	}

	asset, err := s.store.GetAsset(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if asset == nil {
		respondError(w, http.StatusNotFound, "not_found", "Asset not found")
		return
	}

	respondJSON(w, http.StatusOK, asset)
}

func (s *Server) getAssetClassifications(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "assetID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid asset ID")
		return
	}

	classifications, err := s.store.ListClassificationsByAsset(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, classifications)
}

func (s *Server) listAllClassifications(w http.ResponseWriter, r *http.Request) {
	limit := 500
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 5000 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	classifications, total, err := s.store.ListAllClassifications(r.Context(), limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSONWithMeta(w, http.StatusOK, classifications, &apiMeta{
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (s *Server) listFindings(w http.ResponseWriter, r *http.Request) {
	filters := store.ListFindingFilters{
		Limit:  100,
		Offset: 0,
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if limit, err := strconv.Atoi(l); err == nil {
			filters.Limit = limit
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if offset, err := strconv.Atoi(o); err == nil {
			filters.Offset = offset
		}
	}
	if accountID := r.URL.Query().Get("account_id"); accountID != "" {
		if id, err := uuid.Parse(accountID); err == nil {
			filters.AccountID = &id
		}
	}
	if assetID := r.URL.Query().Get("asset_id"); assetID != "" {
		if id, err := uuid.Parse(assetID); err == nil {
			filters.AssetID = &id
		}
	}
	if severity := r.URL.Query().Get("severity"); severity != "" {
		sev := models.FindingSeverity(severity)
		filters.Severity = &sev
	}
	if status := r.URL.Query().Get("status"); status != "" {
		st := models.FindingStatus(status)
		filters.Status = &st
	}
	if findingType := r.URL.Query().Get("type"); findingType != "" {
		filters.FindingType = &findingType
	}

	findings, total, err := s.store.ListFindings(r.Context(), filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSONWithMeta(w, http.StatusOK, findings, &apiMeta{
		Total:  total,
		Limit:  filters.Limit,
		Offset: filters.Offset,
	})
}

func (s *Server) getFinding(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "findingID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid finding ID")
		return
	}

	finding, err := s.store.GetFinding(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if finding == nil {
		respondError(w, http.StatusNotFound, "not_found", "Finding not found")
		return
	}

	respondJSON(w, http.StatusOK, finding)
}

type updateFindingStatusRequest struct {
	Status models.FindingStatus `json:"status"`
	Reason string               `json:"reason,omitempty"`
}

func (s *Server) updateFindingStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "findingID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid finding ID")
		return
	}

	var req updateFindingStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if req.Status == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "status is required")
		return
	}

	if err := s.store.UpdateFindingStatus(r.Context(), id, req.Status, req.Reason); err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	finding, err := s.store.GetFinding(r.Context(), id)
	if err != nil {

		respondJSON(w, http.StatusOK, map[string]string{"id": id.String(), "status": string(req.Status)})
		return
	}
	respondJSON(w, http.StatusOK, finding)
}

func (s *Server) listScans(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	scans, err := s.store.ListAllScanJobs(r.Context(), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, scans)
}

func (s *Server) getScan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "scanID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid scan ID")
		return
	}

	scan, err := s.store.GetScanJob(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if scan == nil {
		respondError(w, http.StatusNotFound, "not_found", "Scan not found")
		return
	}

	respondJSON(w, http.StatusOK, scan)
}

func (s *Server) clearStuckScans(w http.ResponseWriter, r *http.Request) {
	count, err := s.store.ClearStuckScanJobs(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"cleared": count,
		"message": fmt.Sprintf("Cleared %d stuck scan jobs", count),
	})
}

func (s *Server) cancelScan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "scanID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid scan ID")
		return
	}

	if err := s.store.CancelScanJob(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (s *Server) clearAllScanData(w http.ResponseWriter, r *http.Request) {
	counts, err := s.store.ClearAllScanData(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"cleared": counts,
		"message": "All scan data cleared successfully",
	})
}

type dashboardSummary struct {
	Accounts struct {
		Total  int `json:"total"`
		Active int `json:"active"`
	} `json:"accounts"`
	Assets struct {
		Total    int `json:"total"`
		Public   int `json:"public"`
		Critical int `json:"critical"`
	} `json:"assets"`
	Findings struct {
		Total    int `json:"total"`
		Open     int `json:"open"`
		Critical int `json:"critical"`
		High     int `json:"high"`
		Medium   int `json:"medium"`
		Low      int `json:"low"`
	} `json:"findings"`
	Classifications struct {
		Total      int            `json:"total"`
		ByCategory map[string]int `json:"by_category"`
	} `json:"classifications"`
}

func (s *Server) getDashboardSummary(w http.ResponseWriter, r *http.Request) {
	summary := dashboardSummary{}

	counts, err := s.store.GetDashboardCounts(r.Context())
	if err != nil {
		s.logger.Error("failed to get dashboard counts", "error", err)
		respondError(w, http.StatusInternalServerError, "db_error", "Failed to load dashboard")
		return
	}

	summary.Accounts.Total = counts.TotalAccounts
	summary.Accounts.Active = counts.ActiveAccounts
	summary.Assets.Total = counts.TotalAssets
	summary.Assets.Public = counts.PublicAssets
	summary.Assets.Critical = counts.CriticalAssets
	summary.Findings.Total = counts.TotalFindings
	summary.Findings.Open = counts.OpenFindings
	summary.Findings.Critical = counts.CriticalFindings
	summary.Findings.High = counts.HighFindings
	summary.Findings.Medium = counts.MediumFindings
	summary.Findings.Low = counts.LowFindings

	classStats, err := s.store.GetClassificationStats(r.Context(), nil)
	if err != nil {
		s.logger.Warn("failed to get classification stats", "error", err)
		classStats = make(map[string]int)
	}
	summary.Classifications.ByCategory = classStats
	for _, count := range classStats {
		summary.Classifications.Total += count
	}

	respondJSON(w, http.StatusOK, summary)
}

func (s *Server) getClassificationStats(w http.ResponseWriter, r *http.Request) {
	var accountID *uuid.UUID
	if id := r.URL.Query().Get("account_id"); id != "" {
		if parsed, err := uuid.Parse(id); err == nil {
			accountID = &parsed
		}
	}

	stats, err := s.store.GetClassificationStats(r.Context(), accountID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

func (s *Server) getFindingStats(w http.ResponseWriter, r *http.Request) {
	var accountID *uuid.UUID
	if id := r.URL.Query().Get("account_id"); id != "" {
		if parsed, err := uuid.Parse(id); err == nil {
			accountID = &parsed
		}
	}

	stats, err := s.store.GetFindingStats(r.Context(), accountID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}
