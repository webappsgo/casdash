package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/casapps/casdash/internal/app"
	"github.com/casapps/casdash/internal/config"
	"github.com/casapps/casdash/internal/database"
	"github.com/casapps/casdash/internal/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	// Version information
	Version   = "2.0.0"
	BuildDate = "2024-01-01"
	GitCommit = "development"
)

var (
	mode    string
	debug   bool
	rootCmd = &cobra.Command{
		Use:   "casdash",
		Short: "CasDash - Ultimate Self-Hosted Service Dashboard",
		Long: `CasDash combines beautiful homepage functionality with comprehensive monitoring,
container management, and security features in a single binary.`,
		Version: fmt.Sprintf("%s (built %s, commit %s)", Version, BuildDate, GitCommit),
		RunE:    runCasDash,
	}
)

func init() {
	rootCmd.Flags().StringVar(&mode, "mode", "enterprise", "Operating mode: enterprise or saas")
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runCasDash(cmd *cobra.Command, args []string) error {
	// Setup logging
	setupLogging()

	logrus.WithFields(logrus.Fields{
		"version":   Version,
		"buildDate": BuildDate,
		"gitCommit": GitCommit,
		"mode":      mode,
	}).Info("Starting CasDash")

	// Validate mode
	if mode != "enterprise" && mode != "saas" {
		return fmt.Errorf("invalid mode. Valid options: enterprise, saas")
	}

	// Load configuration
	cfg, err := config.Load(mode)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize database
	db, err := database.Initialize(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Create application instance
	application := app.New(cfg, db)

	// Initialize core components (starts monitoring and discovery)
	if err := application.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	// Start web server
	webServer := server.New(application)
	go func() {
		if err := webServer.Start(cfg.Server.Port); err != nil {
			logrus.WithError(err).Error("Web server failed to start")
		}
	}()

	logrus.WithField("port", cfg.Server.Port).Info("CasDash started successfully")

	// Wait for shutdown signal
	return waitForShutdown(application, webServer)
}

func setupLogging() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	logrus.SetOutput(os.Stdout)
}

func waitForShutdown(application *app.App, webServer *server.Server) error {
	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	logrus.WithField("signal", sig.String()).Info("Received shutdown signal")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown components in reverse order
	if webServer != nil {
		logrus.Info("Shutting down web server...")
		if err := webServer.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Error shutting down web server")
		}
	}

	if application != nil {
		logrus.Info("Shutting down application...")
		if err := application.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Error shutting down application")
		}
	}

	logrus.Info("CasDash shutdown complete")
	return nil
}