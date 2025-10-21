package config

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Config represents the application configuration
type Config struct {
	Mode       string           `json:"mode"`
	Server     ServerConfig     `json:"server"`
	Database   DatabaseConfig   `json:"database"`
	Discovery  DiscoveryConfig  `json:"discovery"`
	Monitoring MonitoringConfig `json:"monitoring"`
	Security   SecurityConfig   `json:"security"`
	Debug      bool             `json:"debug"`
}

// ServerConfig contains web server configuration
type ServerConfig struct {
	Port      int    `json:"port"`
	Host      string `json:"host"`
	SecretKey string `json:"secret_key"`
}

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Type         string `json:"type"`
	Path         string `json:"path"`         // SQLite only
	Host         string `json:"host"`         // External DB
	Port         int    `json:"port"`         // External DB
	Name         string `json:"name"`         // External DB
	User         string `json:"user"`         // External DB
	Password     string `json:"password"`     // External DB
	SSLMode      string `json:"ssl_mode"`     // PostgreSQL only
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
	MaxLifetime  int    `json:"max_lifetime"` // seconds
}

// DiscoveryConfig contains service discovery configuration
type DiscoveryConfig struct {
	Enabled              bool     `json:"enabled"`
	Interval             int      `json:"interval"`              // seconds
	Networks             []string `json:"networks"`             // CIDR ranges
	Ports                []int    `json:"ports"`                // Ports to scan
	Timeout              int      `json:"timeout"`              // seconds
	ConfidenceThreshold  int      `json:"confidence_threshold"` // 0-100
	Privileged           bool     `json:"privileged"`           // For ICMP, SYN scanning
}

// MonitoringConfig contains monitoring engine configuration
type MonitoringConfig struct {
	CheckInterval          int   `json:"check_interval"`           // seconds
	CheckTimeout           int   `json:"check_timeout"`            // seconds
	CheckRetries           int   `json:"check_retries"`
	ExpectedStatusCodes    []int `json:"expected_status_codes"`
	SSLExpiryWarning       int   `json:"ssl_expiry_warning"`       // days
	ResponseTimeWarning    int   `json:"response_time_warning"`    // ms
	ResponseTimeCritical   int   `json:"response_time_critical"`   // ms
	ConcurrentChecks       int   `json:"concurrent_checks"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	PasswordMinLength int  `json:"password_min_length"`
	BCryptCost        int  `json:"bcrypt_cost"`
	TwoFAEnabled      bool `json:"two_fa_enabled"`
	SessionTimeout    int  `json:"session_timeout"` // seconds
}

// Load creates configuration from environment variables and defaults
func Load(mode string) (*Config, error) {
	logrus.Info("Loading configuration")

	config := &Config{
		Mode:  mode,
		Debug: getBoolEnv("CASDASH_DEBUG", false),
	}

	// Load server configuration
	serverConfig, err := loadServerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load server config: %w", err)
	}
	config.Server = serverConfig

	// Load database configuration
	dbConfig, err := loadDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load database config: %w", err)
	}
	config.Database = dbConfig

	// Load discovery configuration
	discoveryConfig, err := loadDiscoveryConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load discovery config: %w", err)
	}
	config.Discovery = discoveryConfig

	// Load monitoring configuration
	monitoringConfig, err := loadMonitoringConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load monitoring config: %w", err)
	}
	config.Monitoring = monitoringConfig

	// Load security configuration
	securityConfig, err := loadSecurityConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load security config: %w", err)
	}
	config.Security = securityConfig

	logrus.WithFields(logrus.Fields{
		"mode":      config.Mode,
		"port":      config.Server.Port,
		"db_type":   config.Database.Type,
		"discovery": config.Discovery.Enabled,
	}).Info("Configuration loaded successfully")

	return config, nil
}

// loadServerConfig loads server-related configuration
func loadServerConfig() (ServerConfig, error) {
	config := ServerConfig{
		Host: getEnv("CASDASH_HOST", "0.0.0.0"),
	}

	// Port selection: Use env var if set, otherwise find available port
	if portStr := getEnv("CASDASH_PORT", ""); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return config, fmt.Errorf("invalid port: %s", portStr)
		}
		config.Port = port
	} else {
		port, err := findAvailablePort()
		if err != nil {
			return config, fmt.Errorf("failed to find available port: %w", err)
		}
		config.Port = port
	}

	// Secret key generation
	if secretKey := getEnv("CASDASH_SECRET_KEY", ""); secretKey != "" {
		config.SecretKey = secretKey
	} else {
		config.SecretKey = generateSecretKey()
	}

	return config, nil
}

// loadDatabaseConfig loads database configuration
func loadDatabaseConfig() (DatabaseConfig, error) {
	config := DatabaseConfig{
		Type:         getEnv("CASDASH_DB_TYPE", "sqlite"),
		Path:         getEnv("CASDASH_DB_PATH", "./casdash.db"),
		Host:         getEnv("CASDASH_DB_HOST", "localhost"),
		Port:         getIntEnv("CASDASH_DB_PORT", 5432),
		Name:         getEnv("CASDASH_DB_NAME", "casdash"),
		User:         getEnv("CASDASH_DB_USER", "casdash"),
		Password:     getEnv("CASDASH_DB_PASSWORD", ""),
		SSLMode:      getEnv("CASDASH_DB_SSLMODE", "prefer"),
		MaxOpenConns: getIntEnv("CASDASH_DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns: getIntEnv("CASDASH_DB_MAX_IDLE_CONNS", 5),
		MaxLifetime:  getIntEnv("CASDASH_DB_MAX_LIFETIME", 300),
	}

	// Validate database type
	validTypes := []string{"sqlite", "postgres", "postgresql", "mysql", "mariadb"}
	if !contains(validTypes, config.Type) {
		return config, fmt.Errorf("unsupported database type: %s", config.Type)
	}

	// Set default port based on database type
	if config.Port == 5432 && config.Type != "postgres" && config.Type != "postgresql" {
		switch config.Type {
		case "mysql", "mariadb":
			config.Port = 3306
		}
	}

	return config, nil
}

// loadDiscoveryConfig loads service discovery configuration
func loadDiscoveryConfig() (DiscoveryConfig, error) {
	config := DiscoveryConfig{
		Enabled:             getBoolEnv("CASDASH_DISCOVERY_ENABLED", true),
		Interval:            getIntEnv("CASDASH_DISCOVERY_INTERVAL", 86400), // 24 hours
		Timeout:             getIntEnv("CASDASH_DISCOVERY_TIMEOUT", 2),
		ConfidenceThreshold: getIntEnv("CASDASH_DISCOVERY_CONFIDENCE_THRESHOLD", 70),
		Privileged:          getBoolEnv("CASDASH_DISCOVERY_PRIVILEGED", true),
	}

	// Parse networks
	networksStr := getEnv("CASDASH_DISCOVERY_NETWORKS", "auto-detect")
	if networksStr == "auto-detect" {
		networks, err := autoDetectNetworks()
		if err != nil {
			logrus.WithError(err).Warn("Failed to auto-detect networks, using defaults")
			config.Networks = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
		} else {
			config.Networks = networks
		}
	} else {
		config.Networks = strings.Split(networksStr, ",")
	}

	// Parse ports
	portsStr := getEnv("CASDASH_DISCOVERY_PORTS", "22,53,80,443,3000,3306,5432,6379,8080,8443,9000")
	ports := []int{}
	for _, portStr := range strings.Split(portsStr, ",") {
		if port, err := strconv.Atoi(strings.TrimSpace(portStr)); err == nil {
			ports = append(ports, port)
		}
	}
	config.Ports = ports

	return config, nil
}

// loadMonitoringConfig loads monitoring engine configuration
func loadMonitoringConfig() (MonitoringConfig, error) {
	config := MonitoringConfig{
		CheckInterval:        getIntEnv("CASDASH_CHECK_INTERVAL", 300),         // 5 minutes
		CheckTimeout:         getIntEnv("CASDASH_CHECK_TIMEOUT", 30),           // 30 seconds
		CheckRetries:         getIntEnv("CASDASH_CHECK_RETRIES", 2),
		SSLExpiryWarning:     getIntEnv("CASDASH_SSL_EXPIRY_WARNING", 30),      // 30 days
		ResponseTimeWarning:  getIntEnv("CASDASH_RESPONSE_TIME_WARNING", 1000), // 1 second
		ResponseTimeCritical: getIntEnv("CASDASH_RESPONSE_TIME_CRITICAL", 5000), // 5 seconds
		ConcurrentChecks:     getIntEnv("CASDASH_CONCURRENT_CHECKS", 10),
	}

	// Parse expected status codes
	statusCodesStr := getEnv("CASDASH_EXPECTED_STATUS_CODES", "200,201,202,204")
	statusCodes := []int{}
	for _, codeStr := range strings.Split(statusCodesStr, ",") {
		if code, err := strconv.Atoi(strings.TrimSpace(codeStr)); err == nil {
			statusCodes = append(statusCodes, code)
		}
	}
	config.ExpectedStatusCodes = statusCodes

	return config, nil
}

// loadSecurityConfig loads security configuration
func loadSecurityConfig() (SecurityConfig, error) {
	config := SecurityConfig{
		PasswordMinLength: getIntEnv("CASDASH_PASSWORD_MIN_LENGTH", 12),
		BCryptCost:        getIntEnv("CASDASH_PASSWORD_BCRYPT_COST", 12),
		TwoFAEnabled:      getBoolEnv("CASDASH_2FA_ENABLED", false),
		SessionTimeout:    getIntEnv("CASDASH_SESSION_TIMEOUT", 86400), // 24 hours
	}

	return config, nil
}

// findAvailablePort finds an available port in the range 64000-65535
func findAvailablePort() (int, error) {
	const (
		startPort = 64000
		endPort   = 65535
	)

	// Try a few random ports first
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10; i++ {
		port := rand.Intn(endPort-startPort+1) + startPort
		if isPortAvailable(port) {
			return port, nil
		}
	}

	// If random selection fails, scan sequentially
	for port := startPort; port <= endPort; port++ {
		if isPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports found in range %d-%d", startPort, endPort)
}

// isPortAvailable checks if a port is available for binding
func isPortAvailable(port int) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// autoDetectNetworks detects local network ranges
func autoDetectNetworks() ([]string, error) {
	networks := []string{}

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					networks = append(networks, ipnet.String())
				}
			}
		}
	}

	return networks, nil
}

// generateSecretKey generates a random 32-character secret key
func generateSecretKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())

	key := make([]byte, 32)
	for i := range key {
		key[i] = charset[rand.Intn(len(charset))]
	}

	return string(key)
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}