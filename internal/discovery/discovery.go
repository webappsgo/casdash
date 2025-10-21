package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/casapps/casdash/internal/config"
	"github.com/casapps/casdash/internal/database"
	"github.com/go-ping/ping"
	"github.com/sirupsen/logrus"
)

// Service represents the discovery service
type Service struct {
	db     *database.DB
	config config.DiscoveryConfig

	// Control channels
	stop   chan struct{}
	ticker *time.Ticker

	// Current session
	currentSession *DiscoverySession
	sessionMutex   sync.RWMutex
}

// DiscoverySession represents an active discovery session
type DiscoverySession struct {
	ID          int
	StartedAt   time.Time
	Type        string
	Target      string
	Status      string
	ServicesFound int
	ServicesAdded int
}

// DiscoveredService represents a discovered service
type DiscoveredService struct {
	Host            string
	Port            int
	ServiceType     string
	Version         string
	Fingerprint     string
	ConfidenceScore int
	AdditionalInfo  map[string]interface{}
}

// New creates a new discovery service
func New(db *database.DB, config config.DiscoveryConfig) *Service {
	return &Service{
		db:     db,
		config: config,
		stop:   make(chan struct{}),
	}
}

// Start starts the discovery service
func (s *Service) Start() error {
	if !s.config.Enabled {
		logrus.Info("Service discovery is disabled")
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"interval": s.config.Interval,
		"networks": s.config.Networks,
		"ports":    s.config.Ports,
	}).Info("Starting service discovery")

	// Set up periodic discovery
	s.ticker = time.NewTicker(time.Duration(s.config.Interval) * time.Second)

	// Run initial discovery
	go func() {
		if err := s.RunDiscovery("network", "scheduled"); err != nil {
			logrus.WithError(err).Error("Initial discovery failed")
		}
	}()

	// Start periodic discovery
	go func() {
		for {
			select {
			case <-s.ticker.C:
				if err := s.RunDiscovery("network", "scheduled"); err != nil {
					logrus.WithError(err).Error("Scheduled discovery failed")
				}
			case <-s.stop:
				s.ticker.Stop()
				return
			}
		}
	}()

	return nil
}

// Shutdown gracefully stops the discovery service
func (s *Service) Shutdown(ctx context.Context) error {
	logrus.Info("Shutting down discovery service")

	close(s.stop)

	if s.ticker != nil {
		s.ticker.Stop()
	}

	// Wait for current session to complete or timeout
	s.sessionMutex.RLock()
	if s.currentSession != nil {
		logrus.Info("Waiting for current discovery session to complete")
		// TODO: Implement graceful session termination
	}
	s.sessionMutex.RUnlock()

	return nil
}

// RunDiscovery starts a new discovery session
func (s *Service) RunDiscovery(discoveryType, trigger string) error {
	s.sessionMutex.Lock()
	if s.currentSession != nil {
		s.sessionMutex.Unlock()
		return fmt.Errorf("discovery session already running")
	}

	session := &DiscoverySession{
		StartedAt: time.Now(),
		Type:      discoveryType,
		Target:    "network_scan",
		Status:    "running",
	}
	s.currentSession = session
	s.sessionMutex.Unlock()

	logrus.WithFields(logrus.Fields{
		"type":    discoveryType,
		"trigger": trigger,
	}).Info("Starting discovery session")

	defer func() {
		s.sessionMutex.Lock()
		s.currentSession = nil
		s.sessionMutex.Unlock()
	}()

	// Save session to database
	sessionID, err := s.createDiscoverySession(session)
	if err != nil {
		return fmt.Errorf("failed to create discovery session: %w", err)
	}
	session.ID = sessionID

	var discoveredServices []DiscoveredService

	switch discoveryType {
	case "network":
		discoveredServices, err = s.runNetworkDiscovery()
	case "docker":
		discoveredServices, err = s.runDockerDiscovery()
	case "kubernetes":
		discoveredServices, err = s.runKubernetesDiscovery()
	default:
		err = fmt.Errorf("unknown discovery type: %s", discoveryType)
	}

	if err != nil {
		s.updateDiscoverySession(sessionID, "failed", 0, 0, err.Error())
		return fmt.Errorf("discovery failed: %w", err)
	}

	// Store discovered services
	servicesAdded := 0
	for _, service := range discoveredServices {
		if err := s.storeDiscoveredService(sessionID, service); err != nil {
			logrus.WithError(err).Warn("Failed to store discovered service")
			continue
		}

		// Auto-add services with high confidence
		if service.ConfidenceScore >= s.config.ConfidenceThreshold {
			if err := s.autoAddService(service); err != nil {
				logrus.WithError(err).Warn("Failed to auto-add service")
			} else {
				servicesAdded++
			}
		}
	}

	// Update session
	s.updateDiscoverySession(sessionID, "completed", len(discoveredServices), servicesAdded, "")

	logrus.WithFields(logrus.Fields{
		"services_found": len(discoveredServices),
		"services_added": servicesAdded,
		"session_id":     sessionID,
	}).Info("Discovery session completed")

	return nil
}

// runNetworkDiscovery performs network-based service discovery
func (s *Service) runNetworkDiscovery() ([]DiscoveredService, error) {
	var allServices []DiscoveredService
	var wg sync.WaitGroup
	servicesChan := make(chan DiscoveredService, 100)

	// Worker pool for concurrent scanning
	const maxWorkers = 50
	semaphore := make(chan struct{}, maxWorkers)

	// Collect results
	go func() {
		for service := range servicesChan {
			allServices = append(allServices, service)
		}
	}()

	// Scan each network
	for _, network := range s.config.Networks {
		wg.Add(1)
		go func(network string) {
			defer wg.Done()

			hosts, err := s.expandNetwork(network)
			if err != nil {
				logrus.WithError(err).WithField("network", network).Warn("Failed to expand network")
				return
			}

			// Scan each host
			for _, host := range hosts {
				for _, port := range s.config.Ports {
					wg.Add(1)
					go func(host string, port int) {
						defer wg.Done()

						// Rate limiting
						semaphore <- struct{}{}
						defer func() { <-semaphore }()

						if service := s.scanHostPort(host, port); service != nil {
							servicesChan <- *service
						}
					}(host, port)
				}
			}
		}(network)
	}

	// Wait for all scans to complete
	wg.Wait()
	close(servicesChan)

	return allServices, nil
}

// runDockerDiscovery discovers services from Docker
func (s *Service) runDockerDiscovery() ([]DiscoveredService, error) {
	// TODO: Implement Docker discovery
	logrus.Info("Docker discovery not yet implemented")
	return []DiscoveredService{}, nil
}

// runKubernetesDiscovery discovers services from Kubernetes
func (s *Service) runKubernetesDiscovery() ([]DiscoveredService, error) {
	// TODO: Implement Kubernetes discovery
	logrus.Info("Kubernetes discovery not yet implemented")
	return []DiscoveredService{}, nil
}

// expandNetwork expands a CIDR network into individual host IPs
func (s *Service) expandNetwork(network string) ([]string, error) {
	_, ipNet, err := net.ParseCIDR(network)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %s", network)
	}

	var hosts []string

	// For small networks, expand all IPs
	// For large networks, sample representative IPs
	maskSize, _ := ipNet.Mask.Size()
	maxHosts := 1 << (32 - maskSize)

	if maxHosts > 1024 {
		// Large network - sample every 10th IP
		hosts = s.sampleNetwork(ipNet, 10)
	} else {
		// Small network - scan all IPs
		hosts = s.expandAllIPs(ipNet)
	}

	return hosts, nil
}

// expandAllIPs expands all IPs in a network
func (s *Service) expandAllIPs(ipNet *net.IPNet) []string {
	var hosts []string

	ip := ipNet.IP.Mask(ipNet.Mask)
	for ipNet.Contains(ip) {
		hosts = append(hosts, ip.String())
		// Increment IP
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	return hosts
}

// sampleNetwork samples IPs from a large network
func (s *Service) sampleNetwork(ipNet *net.IPNet, step int) []string {
	var hosts []string
	count := 0

	ip := ipNet.IP.Mask(ipNet.Mask)
	for ipNet.Contains(ip) && len(hosts) < 1000 { // Limit to 1000 hosts
		if count%step == 0 {
			hosts = append(hosts, ip.String())
		}
		count++

		// Increment IP
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	return hosts
}

// scanHostPort scans a specific host and port
func (s *Service) scanHostPort(host string, port int) *DiscoveredService {
	// First, ping the host to see if it's alive
	if !s.isHostAlive(host) {
		return nil
	}

	// Then check if the port is open
	if !s.isPortOpen(host, port) {
		return nil
	}

	// Try to identify the service
	serviceType, version, confidence := s.identifyService(host, port)

	if confidence < 50 { // Minimum confidence threshold
		return nil
	}

	return &DiscoveredService{
		Host:            host,
		Port:            port,
		ServiceType:     serviceType,
		Version:         version,
		ConfidenceScore: confidence,
		Fingerprint:     fmt.Sprintf("%s:%d", host, port),
		AdditionalInfo:  make(map[string]interface{}),
	}
}

// isHostAlive checks if a host is reachable
func (s *Service) isHostAlive(host string) bool {
	if !s.config.Privileged {
		// Use TCP connect instead of ICMP if not privileged
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, "80"),
			time.Duration(s.config.Timeout)*time.Second)
		if err == nil {
			conn.Close()
			return true
		}
		return false
	}

	// Use ICMP ping if privileged
	pinger, err := ping.NewPinger(host)
	if err != nil {
		return false
	}
	pinger.SetPrivileged(true)
	pinger.Count = 1
	pinger.Timeout = time.Duration(s.config.Timeout) * time.Second

	err = pinger.Run()
	return err == nil && pinger.Statistics().PacketsRecv > 0
}

// isPortOpen checks if a port is open on a host
func (s *Service) isPortOpen(host string, port int) bool {
	timeout := time.Duration(s.config.Timeout) * time.Second
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// identifyService attempts to identify the service running on a port
func (s *Service) identifyService(host string, port int) (serviceType, version string, confidence int) {
	// Common port mappings
	commonPorts := map[int]string{
		22:    "ssh",
		23:    "telnet",
		25:    "smtp",
		53:    "dns",
		80:    "http",
		110:   "pop3",
		143:   "imap",
		443:   "https",
		993:   "imaps",
		995:   "pop3s",
		3306:  "mysql",
		5432:  "postgresql",
		6379:  "redis",
		27017: "mongodb",
	}

	// Start with port-based identification
	if knownService, exists := commonPorts[port]; exists {
		serviceType = knownService
		confidence = 70
	} else {
		serviceType = "unknown"
		confidence = 30
	}

	// Try to get a banner or probe the service
	if banner, err := s.getBanner(host, port); err == nil && banner != "" {
		if detectedType, detectedVersion := s.analyzeBanner(banner); detectedType != "" {
			serviceType = detectedType
			version = detectedVersion
			confidence = 90
		}
	}

	// HTTP-specific detection
	if port == 80 || port == 443 || port == 8080 || port == 8443 {
		if httpInfo := s.probeHTTP(host, port); httpInfo != nil {
			if httpInfo.ServiceType != "" {
				serviceType = httpInfo.ServiceType
				version = httpInfo.Version
				confidence = 85
			}
		}
	}

	return serviceType, version, confidence
}

// getBanner attempts to get a service banner
func (s *Service) getBanner(host string, port int) (string, error) {
	timeout := time.Duration(s.config.Timeout) * time.Second
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Read banner
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", err
	}

	return string(buffer[:n]), nil
}

// analyzeBanner analyzes a service banner to identify the service
func (s *Service) analyzeBanner(banner string) (serviceType, version string) {
	// Simple banner analysis - this can be expanded significantly
	bannerPatterns := map[string]string{
		"SSH":      "ssh",
		"HTTP":     "http",
		"FTP":      "ftp",
		"SMTP":     "smtp",
		"MySQL":    "mysql",
		"PostgreSQL": "postgresql",
		"Redis":    "redis",
		"MongoDB":  "mongodb",
	}

	for pattern, service := range bannerPatterns {
		if len(banner) > 0 && banner[:min(len(banner), len(pattern))] == pattern {
			return service, ""
		}
	}

	return "", ""
}

// HTTPInfo contains HTTP service information
type HTTPInfo struct {
	ServiceType string
	Version     string
	Title       string
	Server      string
}

// probeHTTP probes HTTP services for additional information
func (s *Service) probeHTTP(host string, port int) *HTTPInfo {
	// TODO: Implement HTTP probing
	return nil
}

// Database operations

// createDiscoverySession creates a new discovery session in the database
func (s *Service) createDiscoverySession(session *DiscoverySession) (int, error) {
	query := `INSERT INTO discovery_sessions (started_at, discovery_type, target, status)
			  VALUES (?, ?, ?, ?) RETURNING id`

	var id int
	err := s.db.QueryRow(query, session.StartedAt, session.Type, session.Target, session.Status).Scan(&id)
	return id, err
}

// updateDiscoverySession updates a discovery session
func (s *Service) updateDiscoverySession(id int, status string, found, added int, errorMsg string) error {
	query := `UPDATE discovery_sessions
			  SET completed_at = ?, status = ?, services_found = ?, services_added = ?, error_message = ?
			  WHERE id = ?`

	_, err := s.db.Exec(query, time.Now(), status, found, added, errorMsg, id)
	return err
}

// storeDiscoveredService stores a discovered service
func (s *Service) storeDiscoveredService(sessionID int, service DiscoveredService) error {
	query := `INSERT INTO discovered_services (session_id, host, port, service_type, confidence_score, fingerprint, version, discovered_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, sessionID, service.Host, service.Port, service.ServiceType,
		service.ConfidenceScore, service.Fingerprint, service.Version, time.Now())
	return err
}

// autoAddService automatically adds a high-confidence discovered service
func (s *Service) autoAddService(discovered DiscoveredService) error {
	// TODO: Implement automatic service addition
	logrus.WithFields(logrus.Fields{
		"host":       discovered.Host,
		"port":       discovered.Port,
		"type":       discovered.ServiceType,
		"confidence": discovered.ConfidenceScore,
	}).Debug("Auto-adding service (not implemented)")

	return nil
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}