package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/qualys/dspm/internal/auth"
	"github.com/qualys/dspm/internal/models"
	"github.com/qualys/dspm/internal/reports"
	"github.com/qualys/dspm/internal/rules"
	"github.com/qualys/dspm/internal/scheduler"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "email and password are required")
		return
	}

	tokens, err := s.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "auth_error", "Invalid credentials")
		return
	}

	respondJSON(w, http.StatusOK, tokens)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	tokens, err := s.authService.RefreshTokens(r.Context(), req.RefreshToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "auth_error", "Invalid refresh token")
		return
	}

	respondJSON(w, http.StatusOK, tokens)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "auth_error", "Not authenticated")
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		_ = s.authService.LogoutAll(r.Context(), claims.UserID)
	} else {

		_ = s.authService.Logout(r.Context(), claims.UserID, req.RefreshToken)
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (s *Server) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "auth_error", "Not authenticated")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
	})
}

type createUserRequest struct {
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Password string    `json:"password"`
	Role     auth.Role `json:"role"`
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "email and password are required")
		return
	}

	if req.Role == "" {
		req.Role = auth.RoleViewer
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "server_error", "Failed to process password")
		return
	}

	user := &auth.User{
		Email:    req.Email,
		Name:     req.Name,
		Password: hashedPassword,
		Role:     req.Role,
	}

	if err := s.userStore.CreateUser(r.Context(), user); err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	user.Password = ""
	respondJSON(w, http.StatusCreated, user)
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.userStore.ListUsers(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, users)
}

func (s *Server) listScheduledJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.schedulerStore.ListJobs(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, jobs)
}

type createJobRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Schedule    string            `json:"schedule"`
	JobType     scheduler.JobType `json:"job_type"`
	Config      map[string]string `json:"config"`
	Enabled     bool              `json:"enabled"`
}

func (s *Server) createScheduledJob(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if req.Name == "" || req.Schedule == "" || req.JobType == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "name, schedule, and job_type are required")
		return
	}

	job := &scheduler.Job{
		Name:        req.Name,
		Description: req.Description,
		Schedule:    req.Schedule,
		JobType:     req.JobType,
		Config:      req.Config,
		Enabled:     req.Enabled,
	}

	if err := s.scheduler.AddJob(r.Context(), job); err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, job)
}

func (s *Server) getScheduledJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "jobID")
	job, err := s.schedulerStore.GetJob(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "Job not found")
		return
	}

	respondJSON(w, http.StatusOK, job)
}

func (s *Server) updateScheduledJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "jobID")

	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	job := &scheduler.Job{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Schedule:    req.Schedule,
		JobType:     req.JobType,
		Config:      req.Config,
		Enabled:     req.Enabled,
	}

	if err := s.scheduler.UpdateJob(r.Context(), job); err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, job)
}

func (s *Server) deleteScheduledJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "jobID")

	if err := s.scheduler.DeleteJob(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) runScheduledJobNow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "jobID")

	if err := s.scheduler.RunJobNow(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "job_error", err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]string{"status": "triggered"})
}

func (s *Server) getJobExecutions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "jobID")

	execs, err := s.schedulerStore.GetJobExecutions(r.Context(), id, 50)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, execs)
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	rulesList, err := s.rulesEngine.GetRules(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, rulesList)
}

type createRuleRequest struct {
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	Category        models.Category    `json:"category"`
	Sensitivity     models.Sensitivity `json:"sensitivity"`
	Patterns        []string           `json:"patterns"`
	ContextPatterns []string           `json:"context_patterns"`
	ContextRequired bool               `json:"context_required"`
	Priority        int                `json:"priority"`
	Enabled         bool               `json:"enabled"`
}

func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	var req createRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if req.Name == "" || len(req.Patterns) == 0 {
		respondError(w, http.StatusBadRequest, "validation_error", "name and at least one pattern are required")
		return
	}

	for _, p := range req.Patterns {
		if err := rules.ValidatePattern(p); err != nil {
			respondError(w, http.StatusBadRequest, "validation_error", "Invalid pattern: "+err.Error())
			return
		}
	}

	claims, _ := auth.GetUserFromContext(r.Context())
	createdBy := ""
	if claims != nil {
		createdBy = claims.UserID
	}

	rule := &rules.CustomRule{
		Name:            req.Name,
		Description:     req.Description,
		Category:        req.Category,
		Sensitivity:     req.Sensitivity,
		Patterns:        req.Patterns,
		ContextPatterns: req.ContextPatterns,
		ContextRequired: req.ContextRequired,
		Priority:        req.Priority,
		Enabled:         req.Enabled,
		CreatedBy:       createdBy,
	}

	if err := s.rulesEngine.CreateRule(r.Context(), rule); err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, rule)
}

func (s *Server) getRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "ruleID")

	rule, err := s.rulesEngine.GetRule(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "Rule not found")
		return
	}

	respondJSON(w, http.StatusOK, rule)
}

func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "ruleID")

	var req createRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	existing, err := s.rulesEngine.GetRule(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "Rule not found")
		return
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Category = req.Category
	existing.Sensitivity = req.Sensitivity
	existing.Patterns = req.Patterns
	existing.ContextPatterns = req.ContextPatterns
	existing.ContextRequired = req.ContextRequired
	existing.Priority = req.Priority
	existing.Enabled = req.Enabled

	if err := s.rulesEngine.UpdateRule(r.Context(), existing); err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, existing)
}

func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "ruleID")

	if err := s.rulesEngine.DeleteRule(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type testRuleRequest struct {
	Rule    createRuleRequest `json:"rule"`
	Content string            `json:"content"`
}

func (s *Server) testRule(w http.ResponseWriter, r *http.Request) {
	var req testRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	rule := &rules.CustomRule{
		Name:            req.Rule.Name,
		Patterns:        req.Rule.Patterns,
		ContextPatterns: req.Rule.ContextPatterns,
		ContextRequired: req.Rule.ContextRequired,
		Category:        req.Rule.Category,
		Sensitivity:     req.Rule.Sensitivity,
	}

	match, err := s.rulesEngine.TestRule(r.Context(), rule, req.Content)
	if err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"matched": match != nil,
		"match":   match,
	})
}

func (s *Server) getRuleTemplates(w http.ResponseWriter, r *http.Request) {
	templates := rules.GetTemplates()
	respondJSON(w, http.StatusOK, templates)
}

type generateReportRequest struct {
	Type       reports.ReportType   `json:"type"`
	Format     reports.ReportFormat `json:"format"`
	Title      string               `json:"title"`
	AccountIDs []string             `json:"account_ids,omitempty"`
	DateFrom   *time.Time           `json:"date_from,omitempty"`
	DateTo     *time.Time           `json:"date_to,omitempty"`
	Severities []string             `json:"severities,omitempty"`
	Categories []string             `json:"categories,omitempty"`
	Statuses   []string             `json:"statuses,omitempty"`
}

func (s *Server) generateReport(w http.ResponseWriter, r *http.Request) {
	var req generateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if req.Type == "" || req.Format == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "type and format are required")
		return
	}

	if req.Title == "" {
		req.Title = string(req.Type) + " Report"
	}

	reportReq := &reports.ReportRequest{
		Type:       req.Type,
		Format:     req.Format,
		Title:      req.Title,
		AccountIDs: req.AccountIDs,
		DateFrom:   req.DateFrom,
		DateTo:     req.DateTo,
		Severities: req.Severities,
		Categories: req.Categories,
		Statuses:   req.Statuses,
	}

	report, err := s.reportGenerator.Generate(r.Context(), reportReq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "report_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", report.MimeType)
	w.Header().Set("Content-Disposition", "attachment; filename="+report.Filename)
	w.Header().Set("Content-Length", string(rune(len(report.Data))))
	_, _ = w.Write(report.Data)
}

func (s *Server) getReportTypes(w http.ResponseWriter, r *http.Request) {
	types := []map[string]string{
		{"type": "findings", "name": "Findings Report", "description": "List of all security findings"},
		{"type": "assets", "name": "Asset Inventory", "description": "Complete asset inventory"},
		{"type": "classification", "name": "Classification Report", "description": "Data classification summary"},
		{"type": "executive", "name": "Executive Summary", "description": "High-level security posture"},
		{"type": "compliance", "name": "Compliance Report", "description": "Compliance framework status"},
	}
	respondJSON(w, http.StatusOK, types)
}

func (s *Server) streamCSVReport(w http.ResponseWriter, r *http.Request) {
	reportType := r.URL.Query().Get("type")
	if reportType == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "type is required")
		return
	}

	req := &reports.ReportRequest{
		Type:   reports.ReportType(reportType),
		Format: reports.FormatCSV,
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename="+reportType+"_export.csv")
	w.Header().Set("Transfer-Encoding", "chunked")

	if err := s.reportGenerator.StreamCSV(r.Context(), w, req); err != nil {

		s.logger.Error("streaming error", "error", err)
	}
}

type notificationSettingsRequest struct {
	SlackEnabled    bool     `json:"slack_enabled"`
	SlackWebhookURL string   `json:"slack_webhook_url"`
	SlackChannel    string   `json:"slack_channel"`
	EmailEnabled    bool     `json:"email_enabled"`
	EmailRecipients []string `json:"email_recipients"`
	MinSeverity     string   `json:"min_severity"`
}

func (s *Server) getNotificationSettings(w http.ResponseWriter, r *http.Request) {

	settings := map[string]interface{}{
		"slack_enabled":    s.notificationConfig.Slack.Enabled,
		"slack_channel":    s.notificationConfig.Slack.Channel,
		"email_enabled":    s.notificationConfig.Email.Enabled,
		"email_recipients": s.notificationConfig.Email.To,
		"min_severity":     string(s.notificationConfig.Slack.MinSeverity),
	}
	respondJSON(w, http.StatusOK, settings)
}

func (s *Server) updateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	var req notificationSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.notificationConfig.Slack.Enabled = req.SlackEnabled
	if req.SlackWebhookURL != "" {
		s.notificationConfig.Slack.WebhookURL = req.SlackWebhookURL
	}
	s.notificationConfig.Slack.Channel = req.SlackChannel

	s.notificationConfig.Email.Enabled = req.EmailEnabled
	s.notificationConfig.Email.To = req.EmailRecipients

	if req.MinSeverity != "" {
		s.notificationConfig.Slack.MinSeverity = models.Sensitivity(req.MinSeverity)
		s.notificationConfig.Email.MinSeverity = models.Sensitivity(req.MinSeverity)
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) testNotification(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "slack"
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "sent",
		"channel": channel,
	})
}
