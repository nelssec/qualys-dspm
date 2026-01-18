package api

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/qualys/dspm/internal/aitracking"
	"github.com/qualys/dspm/internal/auth"
	"github.com/qualys/dspm/internal/config"
	"github.com/qualys/dspm/internal/encryption"
	"github.com/qualys/dspm/internal/lineage"
	"github.com/qualys/dspm/internal/mlclassifier"
	"github.com/qualys/dspm/internal/models"
	"github.com/qualys/dspm/internal/notifications"
	"github.com/qualys/dspm/internal/remediation"
	"github.com/qualys/dspm/internal/reports"
	"github.com/qualys/dspm/internal/rules"
	"github.com/qualys/dspm/internal/scheduler"
	"github.com/qualys/dspm/internal/store"
)

//go:embed openapi.yaml
var openAPISpec string

func getOpenAPISpec() string {
	return openAPISpec
}

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

	// Phase 2 Services
	encryptionService  *encryption.Service
	lineageService     *lineage.Service
	aiTrackingService  *aitracking.Service
	mlClassifier       *mlclassifier.Service
	remediationService *remediation.Service

	// Scan executor for background scanning
	scanExecutor *ScanExecutor
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

	// Initialize Phase 2 services
	s.encryptionService = encryption.NewService(st)
	s.lineageService = lineage.NewService(st)
	s.aiTrackingService = aitracking.NewService(st)
	s.mlClassifier = mlclassifier.NewService(st)
	s.remediationService = remediation.NewService(&remediationStoreAdapter{st}, s.logger)

	// Initialize scan executor
	s.scanExecutor = NewScanExecutor(st, s.logger)

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

			// Only set JSON content type for API routes
			if r.URL.Path != "/" && r.URL.Path != "/health" && r.URL.Path != "/ready" {
				w.Header().Set("Content-Type", "application/json")
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) setupRoutes() {
	// Serve web UI
	s.router.Get("/", s.serveDashboard)

	s.router.Get("/health", s.healthCheck)
	s.router.Get("/ready", s.readyCheck)

	// Swagger UI and OpenAPI spec
	s.router.Get("/swagger", s.serveSwaggerUI)
	s.router.Get("/swagger/", s.serveSwaggerUI)
	s.router.Get("/api/openapi.yaml", s.serveOpenAPISpec)
	s.router.Get("/api/openapi.json", s.serveOpenAPISpecJSON)

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

			r.Route("/classifications", func(r chi.Router) {
				r.Get("/", s.listAllClassifications)
			})

			r.Route("/scans", func(r chi.Router) {
				r.Get("/", s.listScans)
				r.Post("/clear-stuck", s.clearStuckScans)
				r.Post("/clear-all", s.clearAllScanData)
				r.Get("/{scanID}", s.getScan)
				r.Post("/{scanID}/cancel", s.cancelScan)
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

			// Phase 2: Data Lineage Routes
			r.Route("/lineage", func(r chi.Router) {
				r.Get("/", s.getLineageOverview)
				r.Get("/asset/{assetARN}", s.getAssetLineage)
				r.Get("/paths", s.findDataFlowPaths)
				r.Get("/sensitive-flows", s.getSensitiveDataFlows)
				r.Post("/scan", s.triggerLineageScan)
			})

			// Phase 2: ML Classification Routes
			r.Route("/ml", func(r chi.Router) {
				r.Get("/models", s.listMLModels)
				r.Get("/models/{modelID}", s.getMLModel)
				r.Route("/review", func(r chi.Router) {
					r.Get("/queue", s.getReviewQueue)
					r.Get("/queue/{itemID}", s.getReviewItem)
					r.Post("/queue/{itemID}/resolve", s.resolveReviewItem)
					r.Post("/queue/{itemID}/assign", s.assignReviewItem)
				})
				r.Route("/feedback", func(r chi.Router) {
					r.Post("/", s.submitTrainingFeedback)
					r.Get("/stats", s.getFeedbackStats)
				})
			})

			// Phase 2: AI Source Tracking Routes
			r.Route("/ai", func(r chi.Router) {
				r.Get("/services", s.listAIServices)
				r.Get("/services/{serviceID}", s.getAIService)
				r.Get("/models", s.listAIModels)
				r.Get("/models/{modelID}", s.getAIModel)
				r.Get("/models/{modelID}/training-data", s.getModelTrainingData)
				r.Get("/events", s.listAIProcessingEvents)
				r.Get("/events/sensitive", s.getSensitiveDataAccess)
				r.Get("/risk-report", s.getAIRiskReport)
				r.Post("/scan", s.triggerAIScan)
			})

			// Phase 2: Enhanced Encryption Visibility Routes
			r.Route("/encryption", func(r chi.Router) {
				r.Get("/overview", s.getEncryptionOverview)
				r.Route("/keys", func(r chi.Router) {
					r.Get("/", s.listEncryptionKeys)
					r.Get("/{keyID}", s.getEncryptionKey)
					r.Get("/{keyID}/usage", s.getKeyUsage)
					r.Get("/{keyID}/assets", s.getKeyAssets)
				})
				r.Route("/compliance", func(r chi.Router) {
					r.Get("/summary", s.getComplianceSummary)
					r.Get("/asset/{assetID}", s.getAssetComplianceScore)
					r.Get("/account/{accountID}", s.getAccountComplianceScore)
					r.Get("/recommendations", s.getComplianceRecommendations)
				})
				r.Route("/transit", func(r chi.Router) {
					r.Get("/", s.listTransitEncryption)
					r.Get("/asset/{assetID}", s.getAssetTransitEncryption)
				})
				r.Post("/scan", s.triggerEncryptionScan)
			})

			// Remediation Routes
			r.Route("/remediation", func(r chi.Router) {
				r.Get("/", s.listRemediationActions)
				r.Post("/", s.createRemediationAction)
				r.Get("/summary", s.getRemediationSummary)
				r.Get("/definitions", s.getRemediationDefinitions)
				r.Get("/playbooks", s.getRemediationPlaybooks)
				r.Get("/{actionID}", s.getRemediationAction)
				r.Post("/{actionID}/approve", s.approveRemediationAction)
				r.Post("/{actionID}/reject", s.rejectRemediationAction)
				r.Post("/{actionID}/execute", s.executeRemediationAction)
				r.Post("/{actionID}/rollback", s.rollbackRemediationAction)
				r.Get("/asset/{assetID}", s.listAssetRemediations)
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

func (s *Server) serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Qualys DSPM API Documentation</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui.css">
    <style>
        body { margin: 0; padding: 0; }
        .swagger-ui .topbar { display: none; }
        .swagger-ui .info { margin: 20px 0; }
        .swagger-ui .info .title { color: #1991e1; }
        .header-bar {
            background: #0e1215;
            color: white;
            padding: 12px 20px;
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .header-bar svg { height: 28px; width: 28px; color: #9dbfe1; }
        .header-bar .title { font-size: 18px; font-weight: 500; }
        .header-bar .subtitle { font-size: 11px; color: #9dbfe1; text-transform: uppercase; letter-spacing: 1px; }
        .header-bar a { color: #9dbfe1; text-decoration: none; margin-left: auto; font-size: 14px; }
        .header-bar a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="header-bar">
        <svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/></svg>
        <div>
            <div class="title">Qualys <span class="subtitle">DSPM</span></div>
        </div>
        <a href="/">← Back to Dashboard</a>
    </div>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui-bundle.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: "/api/openapi.yaml",
                dom_id: '#swagger-ui',
                presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
                layout: "BaseLayout",
                deepLinking: true,
                showExtensions: true,
                showCommonExtensions: true,
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 1,
                docExpansion: "list",
                filter: true,
                tagsSorter: "alpha",
                operationsSorter: "alpha"
            });
        };
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func (s *Server) serveOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	// Read the OpenAPI spec from embedded file or filesystem
	spec := getOpenAPISpec()
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(spec))
}

func (s *Server) serveOpenAPISpecJSON(w http.ResponseWriter, r *http.Request) {
	// For JSON format, we'd need to convert YAML to JSON
	// For now, redirect to YAML
	http.Redirect(w, r, "/api/openapi.yaml", http.StatusTemporaryRedirect)
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

// remediationStoreAdapter adapts the store to the remediation.Store interface
type remediationStoreAdapter struct {
	store *store.Store
}

func (a *remediationStoreAdapter) CreateAction(ctx context.Context, action *remediation.Action) error {
	return a.store.CreateRemediationAction(ctx, action)
}

func (a *remediationStoreAdapter) GetAction(ctx context.Context, id uuid.UUID) (*remediation.Action, error) {
	return a.store.GetRemediationAction(ctx, id)
}

func (a *remediationStoreAdapter) UpdateAction(ctx context.Context, action *remediation.Action) error {
	return a.store.UpdateRemediationAction(ctx, action)
}

func (a *remediationStoreAdapter) ListActions(ctx context.Context, accountID uuid.UUID, status *remediation.ActionStatus, limit, offset int) ([]remediation.Action, int, error) {
	return a.store.ListRemediationActions(ctx, accountID, status, limit, offset)
}

func (a *remediationStoreAdapter) ListActionsForAsset(ctx context.Context, assetID uuid.UUID) ([]remediation.Action, error) {
	return a.store.ListRemediationActionsForAsset(ctx, assetID)
}

func (a *remediationStoreAdapter) GetActionSummary(ctx context.Context, accountID uuid.UUID) (*remediation.ActionSummary, error) {
	return a.store.GetRemediationActionSummary(ctx, accountID)
}
