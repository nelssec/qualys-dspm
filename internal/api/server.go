package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/qualys/dspm/internal/auth"
	"github.com/qualys/dspm/internal/config"
	"github.com/qualys/dspm/internal/models"
	"github.com/qualys/dspm/internal/notifications"
	"github.com/qualys/dspm/internal/reports"
	"github.com/qualys/dspm/internal/rules"
	"github.com/qualys/dspm/internal/scheduler"
	"github.com/qualys/dspm/internal/store"
)

// Server represents the API server
type Server struct {
	cfg                *config.Config
	router             *chi.Mux
	store              *store.Store
	http               *http.Server
	logger             *slog.Logger

	// Auth
	authService *auth.Service
	userStore   auth.UserStore

	// Scheduler
	scheduler      *scheduler.Scheduler
	schedulerStore scheduler.Store

	// Rules
	rulesEngine *rules.Engine
	rulesStore  rules.Store

	// Reports
	reportGenerator *reports.Generator

	// Notifications
	notificationService *notifications.Service
	notificationConfig  notifications.Config
}

// ServerOption configures the server
type ServerOption func(*Server)

// WithLogger sets the logger
func WithLogger(logger *slog.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, opts ...ServerOption) (*Server, error) {
	// Initialize store
	st, err := store.New(store.Config{
		DSN:          cfg.Database.DSN(),
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		return nil, fmt.Errorf("initializing store: %w", err)
	}

	s := &Server{
		cfg:    cfg,
		router: chi.NewRouter(),
		store:  st,
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Initialize auth
	s.userStore = auth.NewPostgresUserStore(st.DB())
	s.authService = auth.NewService(auth.Config{
		JWTSecret:          cfg.Auth.JWTSecret,
		AccessTokenExpiry:  cfg.Auth.AccessTokenExpiry,
		RefreshTokenExpiry: cfg.Auth.RefreshTokenExpiry,
	}, s.userStore)

	// Initialize scheduler
	s.schedulerStore = scheduler.NewPostgresStore(st.DB())
	s.scheduler = scheduler.NewScheduler(s.schedulerStore, s.logger)

	// Initialize rules engine
	s.rulesStore = rules.NewPostgresStore(st.DB())
	s.rulesEngine = rules.NewEngine(s.rulesStore)

	// Initialize notifications
	s.notificationConfig = notifications.Config{
		Slack: notifications.SlackConfig{
			WebhookURL:  cfg.Notifications.Slack.WebhookURL,
			Channel:     cfg.Notifications.Slack.Channel,
			Username:    "DSPM Bot",
			IconEmoji:   ":shield:",
			Enabled:     cfg.Notifications.Slack.Enabled,
			MinSeverity: cfg.Notifications.MinSeverity,
		},
		Email: notifications.EmailConfig{
			SMTPHost:    cfg.Notifications.Email.SMTPHost,
			SMTPPort:    cfg.Notifications.Email.SMTPPort,
			Username:    cfg.Notifications.Email.Username,
			Password:    cfg.Notifications.Email.Password,
			From:        cfg.Notifications.Email.From,
			To:          cfg.Notifications.Email.To,
			Enabled:     cfg.Notifications.Email.Enabled,
			MinSeverity: cfg.Notifications.MinSeverity,
		},
	}
	s.notificationService = notifications.NewService(s.notificationConfig, s.logger)

	// Initialize report generator
	s.reportGenerator = reports.NewGenerator(&reportDataProvider{store: st})

	s.setupMiddleware()
	s.setupRoutes()

	s.http = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      s.router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return s, nil
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))
	s.router.Use(corsMiddleware)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) setupRoutes() {
	s.router.Get("/health", s.healthCheck)
	s.router.Get("/ready", s.readyCheck)

	// API v1 routes
	s.router.Route("/api/v1", func(r chi.Router) {
		// Public routes (no auth required)
		r.Post("/auth/login", s.login)
		r.Post("/auth/refresh", s.refresh)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(s.authService.Middleware)

			// Auth
			r.Post("/auth/logout", s.logout)
			r.Get("/auth/me", s.getCurrentUser)

			// User management (admin only)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(auth.RoleAdmin))
				r.Get("/users", s.listUsers)
				r.Post("/users", s.createUser)
			})

			// Accounts
			r.Route("/accounts", func(r chi.Router) {
				r.Get("/", s.listAccounts)
				r.Post("/", s.createAccount)
				r.Get("/{accountID}", s.getAccount)
				r.Delete("/{accountID}", s.deleteAccount)
				r.Post("/{accountID}/scan", s.triggerScan)
			})

			// Assets
			r.Route("/assets", func(r chi.Router) {
				r.Get("/", s.listAssets)
				r.Get("/{assetID}", s.getAsset)
				r.Get("/{assetID}/classifications", s.getAssetClassifications)
			})

			// Findings
			r.Route("/findings", func(r chi.Router) {
				r.Get("/", s.listFindings)
				r.Get("/{findingID}", s.getFinding)
				r.Patch("/{findingID}/status", s.updateFindingStatus)
			})

			// Scans
			r.Route("/scans", func(r chi.Router) {
				r.Get("/", s.listScans)
				r.Get("/{scanID}", s.getScan)
			})

			// Dashboard / Stats
			r.Route("/dashboard", func(r chi.Router) {
				r.Get("/summary", s.getDashboardSummary)
				r.Get("/classification-stats", s.getClassificationStats)
				r.Get("/finding-stats", s.getFindingStats)
			})

			// Scheduled Jobs
			r.Route("/jobs", func(r chi.Router) {
				r.Get("/", s.listScheduledJobs)
				r.Post("/", s.createScheduledJob)
				r.Get("/{jobID}", s.getScheduledJob)
				r.Put("/{jobID}", s.updateScheduledJob)
				r.Delete("/{jobID}", s.deleteScheduledJob)
				r.Post("/{jobID}/run", s.runScheduledJobNow)
				r.Get("/{jobID}/executions", s.getJobExecutions)
			})

			// Custom Rules
			r.Route("/rules", func(r chi.Router) {
				r.Get("/", s.listRules)
				r.Post("/", s.createRule)
				r.Get("/templates", s.getRuleTemplates)
				r.Post("/test", s.testRule)
				r.Get("/{ruleID}", s.getRule)
				r.Put("/{ruleID}", s.updateRule)
				r.Delete("/{ruleID}", s.deleteRule)
			})

			// Reports
			r.Route("/reports", func(r chi.Router) {
				r.Get("/types", s.getReportTypes)
				r.Post("/generate", s.generateReport)
				r.Get("/stream", s.streamCSVReport)
			})

			// Notification Settings
			r.Route("/notifications", func(r chi.Router) {
				r.Use(auth.RequireRole(auth.RoleAdmin))
				r.Get("/settings", s.getNotificationSettings)
				r.Put("/settings", s.updateNotificationSettings)
				r.Post("/test", s.testNotification)
			})
		})
	})
}

// Run starts the server
func (s *Server) Run(ctx context.Context) error {
	// Start scheduler
	if err := s.scheduler.Start(ctx); err != nil {
		s.logger.Error("failed to start scheduler", "error", err)
	}

	// Load custom rules
	if err := s.rulesEngine.LoadRules(ctx); err != nil {
		s.logger.Error("failed to load custom rules", "error", err)
	}

	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("starting server", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.scheduler.Stop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.http.Shutdown(shutdownCtx)
	}
}

// reportDataProvider implements reports.DataProvider
type reportDataProvider struct {
	store *store.Store
}

func (p *reportDataProvider) GetFindings(ctx context.Context, filters reports.FindingsFilter) ([]*models.Finding, error) {
	// Convert filters and call store
	storeFilters := store.ListFindingFilters{
		Limit: 10000,
	}
	findings, _, err := p.store.ListFindings(ctx, storeFilters)
	if err != nil {
		return nil, err
	}

	// Convert to reports model
	result := make([]*models.Finding, len(findings))
	for i, f := range findings {
		result[i] = &models.Finding{
			ID:          f.ID.String(),
			Title:       f.FindingType,
			Description: f.Details["description"].(string),
			Severity:    models.Sensitivity(f.Severity),
			Category:    models.Category(f.FindingType),
			Status:      f.Status,
			AssetID:     f.AssetID.String(),
			CreatedAt:   f.CreatedAt,
			UpdatedAt:   f.UpdatedAt,
		}
	}
	return result, nil
}

func (p *reportDataProvider) GetAssets(ctx context.Context, filters reports.AssetsFilter) ([]*models.DataAsset, error) {
	storeFilters := store.ListAssetFilters{
		Limit: 10000,
	}
	assets, _, err := p.store.ListAssets(ctx, storeFilters)
	if err != nil {
		return nil, err
	}

	result := make([]*models.DataAsset, len(assets))
	for i, a := range assets {
		result[i] = &models.DataAsset{
			ID:        a.ID.String(),
			Name:      a.ResourceID,
			AssetType: a.ResourceType,
			Provider:  models.Provider(a.AccountID.String()),
			Region:    a.Region,
			Classification: models.ClassificationSummary{
				MaxSensitivity: a.SensitivityLevel,
			},
			CreatedAt: a.CreatedAt,
		}
	}
	return result, nil
}

func (p *reportDataProvider) GetClassifications(ctx context.Context, assetID string) ([]*models.Classification, error) {
	return nil, nil
}

func (p *reportDataProvider) GetAccounts(ctx context.Context) ([]*models.CloudAccount, error) {
	accounts, err := p.store.ListAccounts(ctx, nil, nil)
	if err != nil {
		return nil, err
	}

	result := make([]*models.CloudAccount, len(accounts))
	for i, a := range accounts {
		result[i] = &models.CloudAccount{
			ID:       a.ID.String(),
			Name:     a.DisplayName,
			Provider: a.Provider,
			Status:   models.AccountStatus(a.Status),
		}
	}
	return result, nil
}

func (p *reportDataProvider) GetStats(ctx context.Context) (*reports.Stats, error) {
	stats := &reports.Stats{
		ClassificationCounts: make(map[models.Category]int),
		SensitivityCounts:    make(map[models.Sensitivity]int),
	}

	// Get accounts count
	accounts, _ := p.store.ListAccounts(ctx, nil, nil)
	stats.TotalAccounts = len(accounts)

	// Get assets count
	_, total, _ := p.store.ListAssets(ctx, store.ListAssetFilters{Limit: 1})
	stats.TotalAssets = total

	// Get findings
	findings, findingTotal, _ := p.store.ListFindings(ctx, store.ListFindingFilters{Limit: 10000})
	stats.TotalFindings = findingTotal
	for _, f := range findings {
		switch f.Severity {
		case models.SeverityCritical:
			stats.CriticalFindings++
		case models.SeverityHigh:
			stats.HighFindings++
		case models.SeverityMedium:
			stats.MediumFindings++
		case models.SeverityLow:
			stats.LowFindings++
		}
		if f.Status == models.FindingStatusOpen {
			stats.OpenFindings++
		} else if f.Status == models.FindingStatusResolved {
			stats.ResolvedFindings++
		}
	}

	// Get classification stats
	classStats, _ := p.store.GetClassificationStats(ctx, nil)
	for cat, count := range classStats {
		stats.ClassificationCounts[models.Category(cat)] = count
	}

	return stats, nil
}

// Response helpers

type apiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *apiError   `json:"error,omitempty"`
	Meta    *apiMeta    `json:"meta,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type apiMeta struct {
	Total  int `json:"total,omitempty"`
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	})
}

func respondJSONWithMeta(w http.ResponseWriter, status int, data interface{}, meta *apiMeta) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
		Meta:    meta,
	})
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiResponse{
		Success: false,
		Error: &apiError{
			Code:    code,
			Message: message,
		},
	})
}

// Health checks

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

func (s *Server) readyCheck(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		respondError(w, http.StatusServiceUnavailable, "db_unavailable", "Database not available")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}
