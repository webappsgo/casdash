package app

import (
	"context"
	"fmt"

	"github.com/casapps/casdash/internal/config"
	"github.com/casapps/casdash/internal/database"
	"github.com/casapps/casdash/internal/discovery"
	"github.com/casapps/casdash/internal/models"
	"github.com/casapps/casdash/internal/monitoring"
	"github.com/casapps/casdash/internal/notifications"
	"github.com/casapps/casdash/internal/websocket"
	"github.com/sirupsen/logrus"
)

// App represents the main application instance
type App struct {
	Config        *config.Config
	DB            *database.DB
	Mode          string
	Settings      *models.Settings
	Users         *models.UserManager
	Services      *models.ServiceManager
	Monitoring    *monitoring.Engine
	Discovery     *discovery.Service
	Notifications *notifications.Manager
	WSHub         *websocket.Hub
}

// New creates a new application instance
func New(cfg *config.Config, db *database.DB) *App {
	return &App{
		Config: cfg,
		DB:     db,
		Mode:   cfg.Mode,
	}
}

// Initialize sets up the application components
func (a *App) Initialize() error {
	logrus.Info("Initializing application")

	// Initialize WebSocket hub first
	a.WSHub = websocket.NewHub()
	go a.WSHub.Run()
	logrus.Info("WebSocket hub started")

	// Initialize settings manager
	settings, err := models.NewSettings(a.DB)
	if err != nil {
		return fmt.Errorf("failed to initialize settings: %w", err)
	}
	a.Settings = settings

	// Validate mode immutability - mode cannot change after database initialization
	if err := a.validateMode(); err != nil {
		return err
	}

	// Check if this is first run and create primary admin if needed
	if err := a.ensurePrimaryAdmin(); err != nil {
		return fmt.Errorf("failed to ensure primary admin: %w", err)
	}

	// Initialize user manager
	users, err := models.NewUserManager(a.DB, a.Settings)
	if err != nil {
		return fmt.Errorf("failed to initialize user manager: %w", err)
	}
	a.Users = users

	// Initialize service manager
	services, err := models.NewServiceManager(a.DB, a.Settings, a.Mode)
	if err != nil {
		return fmt.Errorf("failed to initialize service manager: %w", err)
	}
	a.Services = services

	// Initialize notification manager
	notificationManager := notifications.New(a.DB, a.WSHub)
	a.Notifications = notificationManager

	// Start notification manager
	if err := a.Notifications.Start(); err != nil {
		return fmt.Errorf("failed to start notification manager: %w", err)
	}
	logrus.Info("Notification manager started successfully")

	// Initialize monitoring engine with WebSocket hub
	monitoringEngine := monitoring.New(a.DB, a.Config.Monitoring, a.WSHub)
	a.Monitoring = monitoringEngine

	// Start monitoring engine
	if err := a.Monitoring.Start(); err != nil {
		return fmt.Errorf("failed to start monitoring engine: %w", err)
	}
	logrus.Info("Monitoring engine started successfully")

	// Initialize discovery service
	discoveryService := discovery.New(a.DB, a.Config.Discovery)
	a.Discovery = discoveryService

	// Start discovery service
	if err := a.Discovery.Start(); err != nil {
		return fmt.Errorf("failed to start discovery service: %w", err)
	}
	logrus.Info("Discovery service started successfully")

	// Apply mode-specific configuration
	if err := a.applyModeConfiguration(); err != nil {
		return fmt.Errorf("failed to apply mode configuration: %w", err)
	}

	logrus.WithField("mode", a.Mode).Info("Application initialized successfully")
	return nil
}

// validateMode ensures mode immutability after database initialization
func (a *App) validateMode() error {
	// Get the stored mode from database
	storedMode, err := a.Settings.Get("mode")
	if err != nil {
		// If mode is not set, this is first run - store the current mode
		if err := a.Settings.Set("mode", a.Mode); err != nil {
			return fmt.Errorf("failed to store initial mode: %w", err)
		}
		logrus.WithField("mode", a.Mode).Info("Initial mode set in database")
		return nil
	}

	// If mode exists in database, validate it matches current mode
	if storedMode != a.Mode {
		return fmt.Errorf("mode cannot be changed after database initialization (stored: %s, requested: %s)", storedMode, a.Mode)
	}

	logrus.WithField("mode", a.Mode).Debug("Mode validation passed")
	return nil
}

// ensurePrimaryAdmin creates the primary admin user if this is the first run
func (a *App) ensurePrimaryAdmin() error {
	// Check if any users exist
	var userCount int
	err := a.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if userCount == 0 {
		logrus.Info("No users found, this appears to be first run")
		// We'll create the primary admin user later through the setup wizard
		// For now, just log that this is first run
		return a.Settings.Set("first_run", "true")
	}

	return nil
}

// applyModeConfiguration applies mode-specific settings and constraints
func (a *App) applyModeConfiguration() error {
	logrus.WithField("mode", a.Mode).Info("Applying mode-specific configuration")

	switch a.Mode {
	case "enterprise":
		return a.configureEnterpriseMode()
	case "saas":
		return a.configureSaaSMode()
	default:
		return fmt.Errorf("unknown mode: %s", a.Mode)
	}
}

// configureEnterpriseMode sets up enterprise-specific configuration
func (a *App) configureEnterpriseMode() error {
	logrus.Info("Configuring Enterprise mode")

	// Set enterprise-specific defaults
	defaults := map[string]string{
		"registration":           "disabled", // Admin-controlled
		"billing_enabled":        "false",    // Always disabled
		"custom_domains":         "false",    // Not available
		"user_service_isolation": "false",    // Shared services with permissions
	}

	for key, value := range defaults {
		if err := a.Settings.SetDefault(key, value); err != nil {
			return fmt.Errorf("failed to set enterprise default %s: %w", key, err)
		}
	}

	return nil
}

// configureSaaSMode sets up SaaS-specific configuration
func (a *App) configureSaaSMode() error {
	logrus.Info("Configuring SaaS mode")

	// Set SaaS-specific defaults
	defaults := map[string]string{
		"registration":           "open",  // Open signup by default
		"billing_enabled":        "false", // Disabled until payment provider configured
		"custom_domains":         "true",  // Available with Let's Encrypt
		"user_service_isolation": "true",  // Users own their services
	}

	for key, value := range defaults {
		if err := a.Settings.SetDefault(key, value); err != nil {
			return fmt.Errorf("failed to set SaaS default %s: %w", key, err)
		}
	}

	return nil
}

// IsFirstRun returns true if this is the first time the application is running
func (a *App) IsFirstRun() bool {
	firstRun, _ := a.Settings.GetBool("first_run")
	return firstRun
}

// GetMode returns the current operating mode
func (a *App) GetMode() string {
	return a.Mode
}

// IsEnterpriseMode returns true if running in enterprise mode
func (a *App) IsEnterpriseMode() bool {
	return a.Mode == "enterprise"
}

// IsSaaSMode returns true if running in SaaS mode
func (a *App) IsSaaSMode() bool {
	return a.Mode == "saas"
}

// Health checks the application health
func (a *App) Health() error {
	// Check database health
	if err := a.DB.Health(); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Check settings accessibility
	if _, err := a.Settings.Get("initialized"); err != nil {
		return fmt.Errorf("settings health check failed: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown(ctx context.Context) error {
	logrus.Info("Shutting down application")

	// Stop discovery service
	if a.Discovery != nil {
		if err := a.Discovery.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Error stopping discovery service")
		}
	}

	// Stop monitoring engine
	if a.Monitoring != nil {
		if err := a.Monitoring.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Error stopping monitoring engine")
		}
	}

	// Close database connection
	if a.DB != nil {
		if err := a.DB.Close(); err != nil {
			logrus.WithError(err).Error("Error closing database")
			return err
		}
	}

	logrus.Info("Application shutdown complete")
	return nil
}

// GetVersion returns version information
func (a *App) GetVersion() map[string]string {
	return map[string]string{
		"version":   "2.0.0",
		"mode":      a.Mode,
		"db_type":   a.DB.Type,
		"build":     "development",
	}
}