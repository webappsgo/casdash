package monitoring

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/casapps/casdash/internal/config"
	"github.com/casapps/casdash/internal/database"
	"github.com/casapps/casdash/internal/websocket"
	"github.com/sirupsen/logrus"
)

// Engine represents the monitoring engine
type Engine struct {
	db     *database.DB
	config config.MonitoringConfig
	wsHub  *websocket.Hub

	// Checkers
	httpChecker *HTTPChecker
	sslChecker  *SSLChecker
	portChecker *PortChecker

	// Control channels
	stop   chan struct{}
	ticker *time.Ticker

	// Worker pool
	workers   int
	workQueue chan *MonitoringJob
	wg        sync.WaitGroup
}

// MonitoringJob represents a monitoring job
type MonitoringJob struct {
	ServiceID   int
	ServiceURL  string
	CheckType   string
	Config      map[string]interface{}
}

// MonitoringResult represents the result of a monitoring check
type MonitoringResult struct {
	ServiceID      int
	CheckType      string
	CheckTime      time.Time
	Success        bool
	ResponseTime   int // milliseconds
	StatusCode     int
	ErrorMessage   string
	Details        map[string]interface{}
}

// New creates a new monitoring engine
func New(db *database.DB, config config.MonitoringConfig, wsHub *websocket.Hub) *Engine {
	return &Engine{
		db:          db,
		config:      config,
		wsHub:       wsHub,
		httpChecker: NewHTTPChecker(config.CheckTimeout, true),
		sslChecker:  NewSSLChecker(config.CheckTimeout),
		portChecker: NewPortChecker(config.CheckTimeout),
		stop:        make(chan struct{}),
		workers:     config.ConcurrentChecks,
		workQueue:   make(chan *MonitoringJob, 1000),
	}
}

// Start starts the monitoring engine
func (e *Engine) Start() error {
	logrus.WithFields(logrus.Fields{
		"workers":        e.workers,
		"check_interval": e.config.CheckInterval,
	}).Info("Starting monitoring engine")

	// Start worker pool
	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.worker(i)
	}

	// Set up periodic monitoring
	e.ticker = time.NewTicker(time.Duration(e.config.CheckInterval) * time.Second)

	// Start scheduler
	go e.scheduler()

	return nil
}

// Shutdown gracefully stops the monitoring engine
func (e *Engine) Shutdown(ctx context.Context) error {
	logrus.Info("Shutting down monitoring engine")

	// Stop scheduler
	if e.ticker != nil {
		e.ticker.Stop()
	}

	// Stop workers
	close(e.stop)
	close(e.workQueue)

	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	// Wait for graceful shutdown or timeout
	select {
	case <-done:
		logrus.Info("Monitoring engine stopped gracefully")
	case <-ctx.Done():
		logrus.Warn("Monitoring engine shutdown timed out")
	}

	return nil
}

// scheduler runs the monitoring scheduler
func (e *Engine) scheduler() {
	logrus.Info("Starting monitoring scheduler")

	// Run initial check
	if err := e.scheduleChecks(); err != nil {
		logrus.WithError(err).Error("Initial monitoring check failed")
	}

	// Schedule periodic checks
	for {
		select {
		case <-e.ticker.C:
			if err := e.scheduleChecks(); err != nil {
				logrus.WithError(err).Error("Scheduled monitoring check failed")
			}
		case <-e.stop:
			return
		}
	}
}

// scheduleChecks schedules monitoring checks for all enabled services
func (e *Engine) scheduleChecks() error {
	logrus.Debug("Scheduling monitoring checks")

	// Get all services that need monitoring
	services, err := e.getServicesForMonitoring()
	if err != nil {
		return fmt.Errorf("failed to get services for monitoring: %w", err)
	}

	// Queue monitoring jobs
	for _, service := range services {
		// Always queue health check
		healthJob := &MonitoringJob{
			ServiceID:  service.ID,
			ServiceURL: service.URL,
			CheckType:  "health",
			Config:     service.Config,
		}

		select {
		case e.workQueue <- healthJob:
		default:
			logrus.Warn("Monitoring work queue is full, dropping health job")
		}

		// Auto-detect SSL monitoring for HTTPS services
		// If ssl_monitoring_enabled is NULL and URL starts with https://, enable SSL monitoring
		if e.shouldEnableSSLMonitoring(service) {
			sslJob := &MonitoringJob{
				ServiceID:  service.ID,
				ServiceURL: service.URL,
				CheckType:  "ssl",
				Config:     service.Config,
			}

			select {
			case e.workQueue <- sslJob:
			default:
				logrus.Warn("Monitoring work queue is full, dropping SSL job")
			}
		}
	}

	logrus.WithField("services", len(services)).Debug("Scheduled monitoring checks")
	return nil
}

// worker processes monitoring jobs
func (e *Engine) worker(id int) {
	defer e.wg.Done()

	logrus.WithField("worker_id", id).Debug("Starting monitoring worker")

	for {
		select {
		case job, ok := <-e.workQueue:
			if !ok {
				return // Channel closed
			}

			result := e.executeMonitoringJob(job)
			if err := e.storeMonitoringResult(result); err != nil {
				logrus.WithError(err).Error("Failed to store monitoring result")
			}

			// Broadcast status update via WebSocket
			if e.wsHub != nil {
				e.broadcastStatusUpdate(job, result)
			}

		case <-e.stop:
			return
		}
	}
}

// executeMonitoringJob executes a monitoring job
func (e *Engine) executeMonitoringJob(job *MonitoringJob) *MonitoringResult {
	start := time.Now()

	result := &MonitoringResult{
		ServiceID: job.ServiceID,
		CheckType: job.CheckType,
		CheckTime: start,
		Details:   make(map[string]interface{}),
	}

	// Execute check based on type
	switch job.CheckType {
	case "health":
		e.executeHealthCheck(job, result)
	case "ssl":
		e.executeSSLCheck(job, result)
	case "port":
		e.executePortCheck(job, result)
	case "performance":
		e.executePerformanceCheck(job, result)
	default:
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Unknown check type: %s", job.CheckType)
	}

	// Calculate response time
	result.ResponseTime = int(time.Since(start).Milliseconds())

	logrus.WithFields(logrus.Fields{
		"service_id":    job.ServiceID,
		"check_type":    job.CheckType,
		"success":       result.Success,
		"response_time": result.ResponseTime,
	}).Debug("Monitoring check completed")

	return result
}

// executeHealthCheck performs a health check
func (e *Engine) executeHealthCheck(job *MonitoringJob, result *MonitoringResult) {
	// Create HTTP check configuration
	checkConfig := &HTTPCheckConfig{
		URL:                 job.ServiceURL,
		Method:              "GET",
		ExpectedStatusCodes: []int{200, 201, 202, 204, 301, 302, 307, 308},
		FollowRedirects:     true,
		VerifySSL:           true,
		Timeout:             e.config.CheckTimeout,
	}

	// Extract any custom configuration from job config
	if method, ok := job.Config["http_method"].(string); ok {
		checkConfig.Method = method
	}
	if expectedCodes, ok := job.Config["expected_status_codes"].([]int); ok {
		checkConfig.ExpectedStatusCodes = expectedCodes
	}
	if expectedContent, ok := job.Config["expected_content"].(string); ok {
		checkConfig.ExpectedContent = expectedContent
	}

	// Perform the check
	httpResult := e.httpChecker.Check(checkConfig)

	// Map result
	result.Success = httpResult.Success
	result.StatusCode = httpResult.StatusCode
	result.ErrorMessage = httpResult.ErrorMessage
	result.Details["check_type"] = "health"
	result.Details["url"] = job.ServiceURL
	result.Details["response_size"] = httpResult.ResponseSize
	result.Details["tls_version"] = httpResult.TLSVersion
	result.Details["content_matched"] = httpResult.ContentMatched

	if httpResult.CertificateInfo != nil {
		result.Details["certificate"] = httpResult.CertificateInfo
	}

	logrus.WithFields(logrus.Fields{
		"service_id":  job.ServiceID,
		"success":     result.Success,
		"status_code": result.StatusCode,
	}).Debug("Health check completed")
}

// executeSSLCheck performs an SSL certificate check
func (e *Engine) executeSSLCheck(job *MonitoringJob, result *MonitoringResult) {
	// Extract hostname and port from service URL
	hostname, port := e.parseHostnamePort(job.ServiceURL)
	if hostname == "" {
		result.Success = false
		result.ErrorMessage = "Invalid hostname"
		return
	}

	// Create SSL check configuration
	checkConfig := &SSLCheckConfig{
		Hostname: hostname,
		Port:     port,
	}

	// Perform the check
	sslResult := e.sslChecker.Check(checkConfig)

	// Map result
	result.Success = sslResult.Success
	result.ErrorMessage = sslResult.ErrorMessage
	result.Details["check_type"] = "ssl"
	result.Details["hostname"] = hostname
	result.Details["port"] = port
	result.Details["days_until_expiry"] = sslResult.DaysUntilExpiry
	result.Details["is_expired"] = sslResult.IsExpired
	result.Details["is_self_signed"] = sslResult.IsSelfSigned
	result.Details["chain_valid"] = sslResult.ChainValid
	result.Details["chain_length"] = sslResult.ChainLength
	result.Details["protocol_version"] = sslResult.ProtocolVersion
	result.Details["cipher_suite"] = sslResult.CipherSuite
	result.Details["vulnerabilities"] = sslResult.Vulnerabilities

	if sslResult.Certificate != nil {
		result.Details["certificate"] = map[string]interface{}{
			"common_name":   sslResult.Certificate.CommonName,
			"sans":          sslResult.Certificate.SANs,
			"issuer":        sslResult.Certificate.Issuer,
			"not_before":    sslResult.Certificate.NotBefore,
			"not_after":     sslResult.Certificate.NotAfter,
			"is_wildcard":   sslResult.Certificate.IsWildcard,
			"fingerprint":   sslResult.Certificate.FingerprintSHA256,
		}
	}

	logrus.WithFields(logrus.Fields{
		"service_id":        job.ServiceID,
		"success":           result.Success,
		"days_until_expiry": sslResult.DaysUntilExpiry,
	}).Debug("SSL check completed")
}

// executePortCheck performs a port connectivity check
func (e *Engine) executePortCheck(job *MonitoringJob, result *MonitoringResult) {
	// Extract hostname and port from service URL
	hostname, port := e.parseHostnamePort(job.ServiceURL)
	if hostname == "" || port == 0 {
		result.Success = false
		result.ErrorMessage = "Invalid hostname or port"
		return
	}

	// Create port check configuration
	checkConfig := &PortCheckConfig{
		Host:     hostname,
		Port:     port,
		Protocol: "tcp", // Default to TCP
	}

	// Check if UDP is specified in config
	if protocol, ok := job.Config["protocol"].(string); ok {
		checkConfig.Protocol = protocol
	}

	// Perform the check
	portResult := e.portChecker.Check(checkConfig)

	// Map result
	result.Success = portResult.Success && portResult.IsOpen
	result.ErrorMessage = portResult.ErrorMessage
	result.Details["check_type"] = "port"
	result.Details["hostname"] = hostname
	result.Details["port"] = port
	result.Details["protocol"] = portResult.Protocol
	result.Details["is_open"] = portResult.IsOpen
	result.Details["banner"] = portResult.BannerGrabbed

	logrus.WithFields(logrus.Fields{
		"service_id": job.ServiceID,
		"success":    result.Success,
		"is_open":    portResult.IsOpen,
	}).Debug("Port check completed")
}

// executePerformanceCheck performs a performance check
func (e *Engine) executePerformanceCheck(job *MonitoringJob, result *MonitoringResult) {
	// TODO: Implement performance check
	result.Success = true
	result.Details["check_type"] = "performance"

	logrus.WithField("service_id", job.ServiceID).Debug("Performance check not fully implemented")
}

// ServiceForMonitoring represents a service configured for monitoring
type ServiceForMonitoring struct {
	ID     int
	URL    string
	Config map[string]interface{}
}

// getServicesForMonitoring retrieves all services that need monitoring
func (e *Engine) getServicesForMonitoring() ([]ServiceForMonitoring, error) {
	query := `SELECT id, url FROM services WHERE monitoring_enabled = 1`

	rows, err := e.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []ServiceForMonitoring
	for rows.Next() {
		var service ServiceForMonitoring
		service.Config = make(map[string]interface{})

		if err := rows.Scan(&service.ID, &service.URL); err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	return services, nil
}

// shouldEnableSSLMonitoring determines if SSL monitoring should be automatically enabled
// Per SPEC: If ssl_monitoring_enabled is NULL and URL starts with https://, enable SSL monitoring
func (e *Engine) shouldEnableSSLMonitoring(service ServiceForMonitoring) bool {
	// Check if URL starts with https://
	if !strings.HasPrefix(strings.ToLower(service.URL), "https://") {
		return false
	}

	// Query the service's ssl_monitoring_enabled setting
	var sslEnabled *bool
	query := `SELECT ssl_monitoring_enabled FROM services WHERE id = ?`
	err := e.db.QueryRow(query, service.ID).Scan(&sslEnabled)
	if err != nil {
		logrus.WithError(err).WithField("service_id", service.ID).Debug("Failed to check SSL monitoring setting")
		return false
	}

	// NULL means auto-detect (enabled for HTTPS)
	// TRUE means explicitly enabled
	// FALSE means explicitly disabled
	if sslEnabled == nil {
		logrus.WithField("service_id", service.ID).Debug("Auto-enabling SSL monitoring for HTTPS service")
		return true
	}

	return *sslEnabled
}

// storeMonitoringResult stores a monitoring result in the database
func (e *Engine) storeMonitoringResult(result *MonitoringResult) error {
	query := `INSERT INTO monitoring_results (service_id, check_type, check_time, success, response_time_ms, status_code, error_message, details)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	detailsJSON := "{}"
	if len(result.Details) > 0 {
		// TODO: Properly serialize details to JSON
		detailsJSON = `{"check_completed": true}`
	}

	_, err := e.db.Exec(query,
		result.ServiceID,
		result.CheckType,
		result.CheckTime,
		result.Success,
		result.ResponseTime,
		result.StatusCode,
		result.ErrorMessage,
		detailsJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to store monitoring result: %w", err)
	}

	return nil
}

// GetServiceStatus returns the current status of a service
func (e *Engine) GetServiceStatus(serviceID int) (*ServiceStatus, error) {
	query := `SELECT success, response_time_ms, check_time, error_message
			  FROM monitoring_results
			  WHERE service_id = ?
			  ORDER BY check_time DESC
			  LIMIT 1`

	var status ServiceStatus
	var checkTime time.Time

	err := e.db.QueryRow(query, serviceID).Scan(
		&status.Success,
		&status.ResponseTime,
		&checkTime,
		&status.ErrorMessage,
	)

	if err != nil {
		return nil, err
	}

	status.LastChecked = checkTime
	status.Status = "online"
	if !status.Success {
		status.Status = "offline"
	}

	return &status, nil
}

// ServiceStatus represents the current status of a service
type ServiceStatus struct {
	Success      bool      `json:"success"`
	Status       string    `json:"status"`
	ResponseTime int       `json:"response_time"`
	LastChecked  time.Time `json:"last_checked"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// GetUptimeStats returns uptime statistics for a service
func (e *Engine) GetUptimeStats(serviceID int, period string) (*UptimeStats, error) {
	// Default to last 24 hours
	var since time.Time
	switch period {
	case "hour":
		since = time.Now().Add(-1 * time.Hour)
	case "day":
		since = time.Now().Add(-24 * time.Hour)
	case "week":
		since = time.Now().Add(-7 * 24 * time.Hour)
	case "month":
		since = time.Now().Add(-30 * 24 * time.Hour)
	default:
		since = time.Now().Add(-24 * time.Hour)
	}

	query := `SELECT COUNT(*) as total, COUNT(CASE WHEN success = 1 THEN 1 END) as successful
			  FROM monitoring_results
			  WHERE service_id = ? AND check_time >= ?`

	var total, successful int
	err := e.db.QueryRow(query, serviceID, since).Scan(&total, &successful)
	if err != nil {
		return nil, err
	}

	uptime := float64(0)
	if total > 0 {
		uptime = float64(successful) / float64(total) * 100
	}

	return &UptimeStats{
		Period:     period,
		Uptime:     uptime,
		TotalChecks: total,
		SuccessfulChecks: successful,
		Since:      since,
	}, nil
}

// UptimeStats represents uptime statistics
type UptimeStats struct {
	Period           string    `json:"period"`
	Uptime           float64   `json:"uptime"`
	TotalChecks      int       `json:"total_checks"`
	SuccessfulChecks int       `json:"successful_checks"`
	Since            time.Time `json:"since"`
}

// parseHostnamePort extracts hostname and port from a URL
func (e *Engine) parseHostnamePort(urlStr string) (string, int) {
	// Try to parse as URL first
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "http://" + urlStr
	}

	// Simple parsing
	parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(urlStr, "http://"), "https://"), "/")
	if len(parts) == 0 {
		return "", 0
	}

	hostPort := parts[0]

	// Check if port is specified
	if strings.Contains(hostPort, ":") {
		hostParts := strings.Split(hostPort, ":")
		if len(hostParts) == 2 {
			host := hostParts[0]
			portStr := hostParts[1]
			port := 0
			fmt.Sscanf(portStr, "%d", &port)
			return host, port
		}
	}

	// Default ports based on scheme
	defaultPort := 80
	if strings.HasPrefix(urlStr, "https://") {
		defaultPort = 443
	}

	return hostPort, defaultPort
}

// broadcastStatusUpdate sends monitoring updates via WebSocket
func (e *Engine) broadcastStatusUpdate(job *MonitoringJob, result *MonitoringResult) {
	// Get service name from database
	serviceName := ""
	err := e.db.QueryRow("SELECT name FROM services WHERE id = ?", job.ServiceID).Scan(&serviceName)
	if err != nil {
		serviceName = job.ServiceURL
	}

	// Determine status
	status := "down"
	if result.Success {
		status = "up"
	}

	// Create status update
	update := websocket.StatusUpdate{
		ServiceID:    job.ServiceID,
		ServiceName:  serviceName,
		Status:       status,
		ResponseTime: result.ResponseTime,
		StatusCode:   result.StatusCode,
		ErrorMessage: result.ErrorMessage,
		Timestamp:    result.CheckTime,
	}

	// Broadcast to WebSocket clients
	e.wsHub.BroadcastStatusUpdate(update)
}