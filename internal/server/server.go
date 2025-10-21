package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/casapps/casdash/internal/app"
	"github.com/casapps/casdash/internal/handlers"
	"github.com/casapps/casdash/internal/middleware"
	"github.com/casapps/casdash/internal/websocket"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
)

//go:embed all:static
var staticFiles embed.FS

//go:embed all:templates
var templateFiles embed.FS

// Server represents the web server
type Server struct {
	app        *app.App
	router     *mux.Router
	server     *http.Server
	store      *sessions.CookieStore
	wsHub      *websocket.Hub
	handlers   *handlers.Handlers
}

// New creates a new web server instance
func New(application *app.App) *Server {
	server := &Server{
		app:   application,
		wsHub: application.WSHub, // Use the app's WebSocket hub
	}

	// Initialize components
	server.setupSessionStore()
	server.setupHandlers()
	server.setupRoutes()

	return server
}

// setupSessionStore initializes the session store
func (s *Server) setupSessionStore() {
	secretKey := s.app.Config.Server.SecretKey
	s.store = sessions.NewCookieStore([]byte(secretKey))
	s.store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
}

// setupHandlers initializes the HTTP handlers
func (s *Server) setupHandlers() {
	s.handlers = handlers.New(s.app, s.store, s.wsHub)
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	s.router = mux.NewRouter()

	// Apply global middleware
	s.router.Use(middleware.Logging)
	s.router.Use(middleware.Recovery)
	s.router.Use(middleware.Security)
	s.router.Use(middleware.Compression)

	// Static files
	s.setupStaticRoutes()

	// API routes
	s.setupAPIRoutes()

	// WebSocket routes
	s.setupWebSocketRoutes()

	// Web interface routes
	s.setupWebRoutes()

	// Public status page routes (must be last)
	s.setupPublicRoutes()
}

// setupStaticRoutes configures static file serving
func (s *Server) setupStaticRoutes() {
	// Serve embedded static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create static file system")
	}

	s.router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))),
	)

	// Favicon
	s.router.HandleFunc("/favicon.ico", s.handlers.Favicon).Methods("GET")
}

// setupAPIRoutes configures API endpoints
func (s *Server) setupAPIRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Apply API middleware
	api.Use(middleware.CORS)
	api.Use(middleware.RateLimit(1000)) // 1000 requests per hour
	api.Use(middleware.APIAuth(s.app))

	// Health check
	api.HandleFunc("/health", s.handlers.APIHealth).Methods("GET")

	// Auth
	auth := api.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/login", s.handlers.APILogin).Methods("POST")
	auth.HandleFunc("/user", s.handlers.APIGetUser).Methods("GET")

	// Services
	services := api.PathPrefix("/services").Subrouter()
	services.HandleFunc("", s.handlers.APIGetServices).Methods("GET")
	services.HandleFunc("", s.handlers.APICreateService).Methods("POST")
	services.HandleFunc("/{id:[0-9]+}", s.handlers.APIGetService).Methods("GET")
	services.HandleFunc("/{id:[0-9]+}", s.handlers.APIUpdateService).Methods("PUT")
	services.HandleFunc("/{id:[0-9]+}", s.handlers.APIDeleteService).Methods("DELETE")
	services.HandleFunc("/{id:[0-9]+}/check", s.handlers.APIForceCheck).Methods("POST")
	services.HandleFunc("/{id:[0-9]+}/status", s.handlers.APIGetServiceStatus).Methods("GET")
	services.HandleFunc("/{id:[0-9]+}/uptime", s.handlers.APIGetUptime).Methods("GET")
	services.HandleFunc("/{id:[0-9]+}/metrics", s.handlers.APIGetMetrics).Methods("GET")

	// Certificates
	certificates := api.PathPrefix("/certificates").Subrouter()
	certificates.HandleFunc("", s.handlers.APIGetCertificates).Methods("GET")

	// API Documentation
	api.HandleFunc("/docs", s.handlers.APIDocs).Methods("GET")

	// Note: Additional API endpoints for settings, security, updates, etc.
	// will be implemented as needed
}

// setupWebSocketRoutes configures WebSocket endpoints
func (s *Server) setupWebSocketRoutes() {
	ws := s.router.PathPrefix("/ws").Subrouter()
	ws.Use(middleware.WebSocketAuth(s.app))

	ws.HandleFunc("/status", s.handlers.WSStatus)
	ws.HandleFunc("/monitoring", s.handlers.WSMonitoring)
	ws.HandleFunc("/notifications", s.handlers.WSNotifications)
	ws.HandleFunc("/logs", s.handlers.WSLogs)
	ws.HandleFunc("/chat", s.handlers.WSChat)
}

// setupWebRoutes configures web interface routes
func (s *Server) setupWebRoutes() {
	// Authentication middleware for web routes
	webAuth := middleware.RequireAuth(s.app, s.store)

	// Public routes (no auth required)
	s.router.HandleFunc("/", s.handlers.Home)
	s.router.HandleFunc("/login", s.handlers.Login).Methods("GET", "POST")
	s.router.HandleFunc("/register", s.handlers.Register).Methods("GET", "POST")
	s.router.HandleFunc("/setup", s.handlers.Setup).Methods("GET", "POST")
	s.router.HandleFunc("/logout", s.handlers.Logout).Methods("POST")

	// Protected routes (auth required)
	protected := s.router.NewRoute().Subrouter()
	protected.Use(webAuth)

	// Dashboard
	protected.HandleFunc("/dashboard", s.handlers.Dashboard).Methods("GET")

	// Services
	protected.HandleFunc("/services", s.handlers.Services).Methods("GET")
	protected.HandleFunc("/services/add", s.handlers.AddService).Methods("GET", "POST")
	protected.HandleFunc("/services/discover", s.handlers.DiscoverServices).Methods("GET", "POST")
	protected.HandleFunc("/services/{id:[0-9]+}", s.handlers.ServiceDetails).Methods("GET")
	protected.HandleFunc("/services/{id:[0-9]+}/edit", s.handlers.EditService).Methods("GET", "POST")
	protected.HandleFunc("/services/{id:[0-9]+}/monitor", s.handlers.ServiceMonitoring).Methods("GET")
	protected.HandleFunc("/services/{id:[0-9]+}/ssl", s.handlers.ServiceSSL).Methods("GET")
	protected.HandleFunc("/services/{id:[0-9]+}/security", s.handlers.ServiceSecurity).Methods("GET")
	protected.HandleFunc("/services/{id:[0-9]+}/issues", s.handlers.ServiceIssues).Methods("GET")
	protected.HandleFunc("/services/{id:[0-9]+}/maintenance", s.handlers.ServiceMaintenance).Methods("GET")
	protected.HandleFunc("/services/{id:[0-9]+}/logs", s.handlers.ServiceLogs).Methods("GET")

	// Monitoring
	protected.HandleFunc("/monitoring", s.handlers.Monitoring).Methods("GET")
	protected.HandleFunc("/monitoring/uptime", s.handlers.UptimeStats).Methods("GET")
	protected.HandleFunc("/monitoring/performance", s.handlers.PerformanceMetrics).Methods("GET")
	protected.HandleFunc("/monitoring/alerts", s.handlers.AlertConfiguration).Methods("GET", "POST")
	protected.HandleFunc("/monitoring/dependencies", s.handlers.DependencyMap).Methods("GET")

	// Security
	protected.HandleFunc("/security", s.handlers.Security).Methods("GET")
	protected.HandleFunc("/security/scan", s.handlers.SecurityScan).Methods("GET", "POST")
	protected.HandleFunc("/security/vulnerabilities", s.handlers.Vulnerabilities).Methods("GET")
	protected.HandleFunc("/security/compliance", s.handlers.Compliance).Methods("GET")
	protected.HandleFunc("/security/recommendations", s.handlers.SecurityRecommendations).Methods("GET")

	// SSL/TLS
	protected.HandleFunc("/certificates", s.handlers.Certificates).Methods("GET")
	protected.HandleFunc("/certificates/{id:[0-9]+}", s.handlers.CertificateDetails).Methods("GET")
	protected.HandleFunc("/certificates/expiring", s.handlers.ExpiringCertificates).Methods("GET")

	// Updates
	protected.HandleFunc("/updates", s.handlers.Updates).Methods("GET")
	protected.HandleFunc("/updates/policies", s.handlers.UpdatePolicies).Methods("GET", "POST")
	protected.HandleFunc("/updates/history", s.handlers.UpdateHistory).Methods("GET")
	protected.HandleFunc("/updates/schedule", s.handlers.ScheduledUpdates).Methods("GET", "POST")

	// Support
	protected.HandleFunc("/support", s.handlers.Support).Methods("GET")
	protected.HandleFunc("/support/chat", s.handlers.LiveChat).Methods("GET")
	protected.HandleFunc("/support/tickets", s.handlers.SupportTickets).Methods("GET")
	protected.HandleFunc("/support/tickets/new", s.handlers.NewTicket).Methods("GET", "POST")
	protected.HandleFunc("/support/tickets/{id:[0-9]+}", s.handlers.TicketDetails).Methods("GET")
	protected.HandleFunc("/support/kb", s.handlers.KnowledgeBase).Methods("GET")

	// Maintenance
	protected.HandleFunc("/maintenance", s.handlers.Maintenance).Methods("GET")
	protected.HandleFunc("/maintenance/calendar", s.handlers.MaintenanceCalendar).Methods("GET")
	protected.HandleFunc("/maintenance/schedule", s.handlers.ScheduleMaintenance).Methods("GET", "POST")
	protected.HandleFunc("/maintenance/templates", s.handlers.MaintenanceTemplates).Methods("GET")
	protected.HandleFunc("/maintenance/history", s.handlers.MaintenanceHistory).Methods("GET")

	// User management
	adminAuth := middleware.RequireRole("admin")
	admin := protected.NewRoute().Subrouter()
	admin.Use(adminAuth)

	admin.HandleFunc("/users", s.handlers.Users).Methods("GET")
	admin.HandleFunc("/users/invite", s.handlers.InviteUser).Methods("GET", "POST")
	admin.HandleFunc("/users/{id:[0-9]+}", s.handlers.UserDetails).Methods("GET")

	// User profile
	protected.HandleFunc("/profile", s.handlers.Profile).Methods("GET", "POST")
	protected.HandleFunc("/preferences", s.handlers.Preferences).Methods("GET", "POST")
	protected.HandleFunc("/security-settings", s.handlers.SecuritySettings).Methods("GET", "POST")

	// Admin settings
	admin.HandleFunc("/admin", s.handlers.Admin).Methods("GET")
	admin.HandleFunc("/admin/settings", s.handlers.AdminSettings).Methods("GET", "POST")
	admin.HandleFunc("/admin/discovery", s.handlers.DiscoveryConfiguration).Methods("GET", "POST")
	admin.HandleFunc("/admin/appearance", s.handlers.UICustomization).Methods("GET", "POST")
	admin.HandleFunc("/admin/backup", s.handlers.BackupManagement).Methods("GET", "POST")
	admin.HandleFunc("/admin/logs", s.handlers.SystemLogs).Methods("GET")
	admin.HandleFunc("/admin/health", s.handlers.SystemHealth).Methods("GET")

	// SaaS mode specific routes
	if s.app.IsSaaSMode() {
		admin.HandleFunc("/admin/billing", s.handlers.BillingConfiguration).Methods("GET", "POST")
	}

	admin.HandleFunc("/admin/email", s.handlers.EmailConfiguration).Methods("GET", "POST")
	admin.HandleFunc("/admin/integrations", s.handlers.Integrations).Methods("GET", "POST")
}

// setupPublicRoutes configures public status page routes
func (s *Server) setupPublicRoutes() {
	// Public status pages (no auth required)
	// These must be last to avoid conflicting with other routes
	s.router.HandleFunc("/{username:[a-zA-Z0-9_-]+}", s.handlers.PublicStatusPage).Methods("GET")
}

// Start starts the web server
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf("%s:%d", s.app.Config.Server.Host, port)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logrus.WithField("address", addr).Info("Starting web server")

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the web server
func (s *Server) Shutdown(ctx context.Context) error {
	logrus.Info("Shutting down web server")

	if s.wsHub != nil {
		s.wsHub.Stop()
	}

	if s.server != nil {
		return s.server.Shutdown(ctx)
	}

	return nil
}

// GetTemplateFS returns the embedded template file system
func (s *Server) GetTemplateFS() fs.FS {
	templateFS, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create template file system")
	}
	return templateFS
}

// GetStaticFS returns the embedded static file system
func (s *Server) GetStaticFS() fs.FS {
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create static file system")
	}
	return staticFS
}