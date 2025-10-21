package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/casapps/casdash/internal/config"
	"github.com/golang-migrate/migrate/v4"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// DB represents the database connection with additional metadata
type DB struct {
	*sql.DB
	Type   string
	Config config.DatabaseConfig
}

// Initialize creates and configures the database connection
func Initialize(cfg config.DatabaseConfig) (*DB, error) {
	logrus.WithField("type", cfg.Type).Info("Initializing database connection")

	// Create connection string
	connStr, err := buildConnectionString(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build connection string: %w", err)
	}

	// Map database type to driver name
	driverName := getDriverName(cfg.Type)

	// Open database connection
	sqlDB, err := sql.Open(driverName, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	configureConnectionPool(sqlDB, cfg)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		DB:     sqlDB,
		Type:   cfg.Type,
		Config: cfg,
	}

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize default data
	if err := db.InitializeDefaults(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize defaults: %w", err)
	}

	logrus.Info("Database initialized successfully")
	return db, nil
}

// getDriverName maps database type to SQL driver name
func getDriverName(dbType string) string {
	switch dbType {
	case "sqlite":
		return "sqlite3"
	case "postgres", "postgresql":
		return "postgres"
	case "mysql", "mariadb":
		return "mysql"
	default:
		return dbType
	}
}

// buildConnectionString creates the appropriate connection string for each database type
func buildConnectionString(cfg config.DatabaseConfig) (string, error) {
	switch cfg.Type {
	case "sqlite":
		if cfg.Path == "" {
			return "", fmt.Errorf("database path is required for SQLite")
		}
		return cfg.Path, nil

	case "postgres", "postgresql":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode), nil

	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name), nil

	case "mariadb":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name), nil

	default:
		return "", fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

// configureConnectionPool sets up connection pool parameters
func configureConnectionPool(db *sql.DB, cfg config.DatabaseConfig) {
	// Set maximum number of open connections
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(25) // Default
	}

	// Set maximum number of idle connections
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(5) // Default
	}

	// Set maximum connection lifetime
	if cfg.MaxLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)
	} else {
		db.SetConnMaxLifetime(5 * time.Minute) // Default
	}
}

// RunMigrations executes database migrations
func (db *DB) RunMigrations() error {
	logrus.Info("Running database migrations")

	// Create migrate instance based on database type
	var driver migratedb.Driver
	var err error

	switch db.Type {
	case "sqlite":
		driver, err = sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	case "postgres", "postgresql":
		driver, err = postgres.WithInstance(db.DB, &postgres.Config{})
	case "mysql", "mariadb":
		driver, err = mysql.WithInstance(db.DB, &mysql.Config{})
	default:
		return fmt.Errorf("unsupported database type for migrations: %s", db.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create file source for migrations
	sourceDriver, err := (&file.File{}).Open("file://migrations")
	if err != nil {
		return fmt.Errorf("failed to open migrations source: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("file", sourceDriver, db.Type, driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logrus.Info("Database migrations completed successfully")
	return nil
}

// InitializeDefaults populates the database with default data
func (db *DB) InitializeDefaults() error {
	logrus.Info("Initializing default data")

	// Check if this is first run
	isFirstRun, err := db.IsFirstRun()
	if err != nil {
		return fmt.Errorf("failed to check first run: %w", err)
	}

	if !isFirstRun {
		logrus.Debug("Not first run, skipping default data initialization")
		return nil
	}

	// Initialize default settings
	if err := db.InitializeSettings(); err != nil {
		return fmt.Errorf("failed to initialize settings: %w", err)
	}

	// Initialize service types
	if err := db.InitializeServiceTypes(); err != nil {
		return fmt.Errorf("failed to initialize service types: %w", err)
	}

	// Initialize themes
	if err := db.InitializeThemes(); err != nil {
		return fmt.Errorf("failed to initialize themes: %w", err)
	}

	logrus.Info("Default data initialized successfully")
	return nil
}

// IsFirstRun checks if this is the first time the application is running
func (db *DB) IsFirstRun() (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM settings WHERE key = 'initialized'`

	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return false, err
	}

	return count == 0, nil
}

// MarkInitialized marks the database as initialized
func (db *DB) MarkInitialized() error {
	query := `INSERT INTO settings (key, value, type, category, description)
			  VALUES ('initialized', 'true', 'boolean', 'system', 'Marks database as initialized')`

	_, err := db.Exec(query)
	return err
}

// GetDatabaseType returns the database type as a string
func (db *DB) GetDatabaseType() string {
	return db.Type
}

// SupportsJSON returns true if the database supports JSON columns
func (db *DB) SupportsJSON() bool {
	switch db.Type {
	case "postgres", "postgresql":
		return true
	case "mysql", "mariadb":
		return true
	case "sqlite":
		return true // SQLite 3.38+ supports JSON
	default:
		return false
	}
}

// GetJSONType returns the appropriate JSON column type for the database
func (db *DB) GetJSONType() string {
	switch db.Type {
	case "postgres", "postgresql":
		return "JSONB"
	case "mysql", "mariadb":
		return "JSON"
	case "sqlite":
		return "TEXT" // SQLite stores JSON as TEXT
	default:
		return "TEXT"
	}
}

// BeginTx starts a database transaction
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.Begin()
}

// Health checks the database connection health
func (db *DB) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

// Close closes the database connection
func (db *DB) Close() error {
	logrus.Info("Closing database connection")
	return db.DB.Close()
}