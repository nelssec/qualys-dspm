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

type Server struct {
	cfg    *config.Config
	router *chi.Mux
	store  *store.Store
	http   *http.Server
	logger *slog.Logger

	authService *auth.Service
	userStore   auth.UserStore

	scheduler      *scheduler.Scheduler
	schedulerStore scheduler.Store

	rulesEngine *rules.Engine
	rulesStore  rules.Store

	reportGenerator *reports.Generator

	notificationService *notifications.Service
	notificationConfig  notifications.Config
}

type ServerOption func(*Server)

func WithLogger(logger *slog.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

func NewServer(cfg *config.Config, opts ...ServerOption) (*Server, error) {
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

	s.userStore = auth.NewPostgresUserStore(st.DB())
	s.authService = auth.NewService(auth.Config{
		JWTSecret:          cfg.Auth.JWTSecret,
		AccessTokenExpiry:  cfg.Auth.AccessTokenExpiry,
		RefreshTokenExpiry: cfg.Auth.RefreshTokenExpiry,
	}, s.userStore)

	s.schedulerStore = scheduler.NewPostgresStore(st.DB())
	s.scheduler = scheduler.NewScheduler(s.schedulerStore, s.logger)

	s.rulesStore = rules.NewPostgresStore(st.DB())
	s.rulesEngine = rules.NewEngine(s.rulesStore)

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
	s.router.Use(s.corsMiddleware())
}

func (s *Server) corsMiddleware() func(http.Handler) http.Handler {
	allowOrigin := s.cfg.Server.CORSAllowOrigin
	if allowOrigin == "" {
		allowOrigin = "*"
		s.logger.Warn("CORS Allow-Origin set to '*' - configure server.cors_allow_origin in production")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) setupRoutes() {
	s.router.Get("/health", s.healthCheck)
	s.router.Get("/ready", s.readyCheck)

	s.router.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", s.login)
		r.Post("/auth/refresh", s.refresh)

		r.Group(func(r chi.Router) {
			r.Use(s.authService.Middleware)

			r.Post("/auth/logout", s.logout)
			r.Get("/auth/me", s.getCurrentUser)

			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(auth.RoleAdmin))
				r.Get("/users", s.listUsers)
				r.Post("/users", s.createUser)
			})

			r.Route("/accounts", func(r chi.Router) {
				r.Get("/", s.listAccounts)
				r.Post("/", s.createAccount)
				r.Get("/{accountID}", s.getAccount)
				r.Delete("/{accountID}", s.deleteAccount)
				r.Post("/{accountID}/scan", s.triggerScan)
			})

			r.Route("/assets", func(r chi.Router) {
				r.Get("/", s.listAssets)
				r.Get("/{assetID}", s.getAsset)
				r.Get("/{assetID}/classifications", s.getAssetClassifications)
			})

			r.Route("/findings", func(r chi.Router) {
				r.Get("/", s.listFindings)
				r.Get("/{findingID}", s.getFinding)
				r.Patch("/{findingID}/status", s.updateFindingStatus)
			})

			r.Route("/scans", func(r chi.Router) {
				r.Get("/", s.listScans)
				r.Get("/{scanID}", s.getScan)
			})

			r.Route("/dashboard", func(r chi.Router) {
				r.Get("/summary", s.getDashboardSummary)
				r.Get("/classification-stats", s.getClassificationStats)
				r.Get("/finding-stats", s.getFindingStats)
			})

			r.Route("/jobs", func(r chi.Router) {
				r.Get("/", s.listScheduledJobs)
				r.Post("/", s.createScheduledJob)
				r.Get("/{jobID}", s.getScheduledJob)
				r.Put("/{jobID}", s.updateScheduledJob)
				r.Delete("/{jobID}", s.deleteScheduledJob)
				r.Post("/{jobID}/run", s.runScheduledJobNow)
				r.Get("/{jobID}/executions", s.getJobExecutions)
			})

			r.Route("/rules", func(r chi.Router) {
				r.Get("/", s.listRules)
				r.Post("/", s.createRule)
				r.Get("/templates", s.getRuleTemplates)
				r.Post("/test", s.testRule)
				r.Get("/{ruleID}", s.getRule)
				r.Put("/{ruleID}", s.updateRule)
				r.Delete("/{ruleID}", s.deleteRule)
			})

			r.Route("/reports", func(r chi.Router) {
				r.Get("/types", s.getReportTypes)
				r.Post("/generate", s.generateReport)
				r.Get("/stream", s.streamCSVReport)
			})

			r.Route("/notifications", func(r chi.Router) {
				r.Use(auth.RequireRole(auth.RoleAdmin))
				r.Get("/settings", s.getNotificationSettings)
				r.Put("/settings", s.updateNotificationSettings)
				r.Post("/test", s.testNotification)
			})
		})
	})
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.scheduler.Start(ctx); err != nil {
		s.logger.Error("failed to start scheduler", "error", err)
	}

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

type reportDataProvider struct {
	store *store.Store
}

func (p *reportDataProvider) GetFindings(ctx context.Context, filters reports.FindingsFilter) ([]*reports.ReportFinding, error) {
	storeFilters := store.ListFindingFilters{
		Limit: 10000,
	}
	findings, _, err := p.store.ListFindings(ctx, storeFilters)
	if err != nil {
		return nil, err
	}

	result := make([]*reports.ReportFinding, len(findings))
	for i, f := range findings {
		assetID := ""
		if f.AssetID != nil {
			assetID = f.AssetID.String()
		}
		result[i] = &reports.ReportFinding{
			ID:          f.ID.String(),
			Title:       f.Title,
			Description: f.Description,
			Severity:    string(f.Severity),
			Category:    f.FindingType,
			Status:      string(f.Status),
			AssetID:     assetID,
			Remediation: f.Remediation,
			CreatedAt:   f.CreatedAt,
			UpdatedAt:   f.UpdatedAt,
		}
	}
	return result, nil
}

func (p *reportDataProvider) GetAssets(ctx context.Context, filters reports.AssetsFilter) ([]*reports.ReportAsset, error) {
	storeFilters := store.ListAssetFilters{
		Limit: 10000,
	}
	assets, _, err := p.store.ListAssets(ctx, storeFilters)
	if err != nil {
		return nil, err
	}

	result := make([]*reports.ReportAsset, len(assets))
	for i, a := range assets {
		result[i] = &reports.ReportAsset{
			ID:            a.ID.String(),
			Name:          a.Name,
			AssetType:     string(a.ResourceType),
			Provider:      "cloud",
			AccountID:     a.AccountID.String(),
			Region:        a.Region,
			Sensitivity:   string(a.SensitivityLevel),
			LastScannedAt: a.LastScannedAt,
			CreatedAt:     a.CreatedAt,
		}
	}
	return result, nil
}

func (p *reportDataProvider) GetAccounts(ctx context.Context) ([]*reports.ReportAccount, error) {
	accounts, err := p.store.ListAccounts(ctx, nil, nil)
	if err != nil {
		return nil, err
	}

	result := make([]*reports.ReportAccount, len(accounts))
	for i, a := range accounts {
		result[i] = &reports.ReportAccount{
			ID:       a.ID.String(),
			Name:     a.DisplayName,
			Provider: string(a.Provider),
			Status:   a.Status,
		}
	}
	return result, nil
}

func (p *reportDataProvider) GetStats(ctx context.Context) (*reports.Stats, error) {
	stats := &reports.Stats{
		ClassificationCounts: make(map[string]int),
		SensitivityCounts:    make(map[string]int),
	}

	accounts, _ := p.store.ListAccounts(ctx, nil, nil)
	stats.TotalAccounts = len(accounts)

	_, total, _ := p.store.ListAssets(ctx, store.ListAssetFilters{Limit: 1})
	stats.TotalAssets = total

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

	classStats, _ := p.store.GetClassificationStats(ctx, nil)
	for cat, count := range classStats {
		stats.ClassificationCounts[cat] = count
	}

	return stats, nil
}

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
	_ = json.NewEncoder(w).Encode(apiResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	})
}

func respondJSONWithMeta(w http.ResponseWriter, status int, data interface{}, meta *apiMeta) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
		Meta:    meta,
	})
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{
		Success: false,
		Error: &apiError{
			Code:    code,
			Message: message,
		},
	})
}

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
