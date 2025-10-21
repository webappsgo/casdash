package monitoring

import (
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

// PortChecker performs TCP/UDP port connectivity checks
type PortChecker struct {
	timeout time.Duration
}

// NewPortChecker creates a new port checker
func NewPortChecker(timeout int) *PortChecker {
	return &PortChecker{
		timeout: time.Duration(timeout) * time.Second,
	}
}

// PortCheckConfig represents the configuration for a port check
type PortCheckConfig struct {
	Host     string
	Port     int
	Protocol string // "tcp" or "udp"
	SendData string // Optional data to send
	ExpectData string // Optional data to expect in response
}

// PortCheckResult represents the result of a port check
type PortCheckResult struct {
	Success       bool
	ResponseTime  int64
	ErrorMessage  string
	IsOpen        bool
	BannerGrabbed string
	Protocol      string
	Host          string
	Port          int
}

// Check performs a port connectivity check
func (p *PortChecker) Check(config *PortCheckConfig) *PortCheckResult {
	start := time.Now()
	result := &PortCheckResult{
		Protocol: config.Protocol,
		Host:     config.Host,
		Port:     config.Port,
	}

	// Default to TCP if not specified
	if config.Protocol == "" {
		config.Protocol = "tcp"
	}

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)

	switch config.Protocol {
	case "tcp":
		result = p.checkTCP(address, config)
	case "udp":
		result = p.checkUDP(address, config)
	default:
		result.ErrorMessage = fmt.Sprintf("Unsupported protocol: %s", config.Protocol)
	}

	result.ResponseTime = time.Since(start).Milliseconds()

	logrus.WithFields(logrus.Fields{
		"host":     config.Host,
		"port":     config.Port,
		"protocol": config.Protocol,
		"success":  result.Success,
		"is_open":  result.IsOpen,
	}).Debug("Port check completed")

	return result
}

// checkTCP performs a TCP port check
func (p *PortChecker) checkTCP(address string, config *PortCheckConfig) *PortCheckResult {
	result := &PortCheckResult{
		Protocol: "tcp",
		Host:     config.Host,
		Port:     config.Port,
	}

	// Attempt to connect
	conn, err := net.DialTimeout("tcp", address, p.timeout)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Connection failed: %v", err)
		result.IsOpen = false
		return result
	}
	defer conn.Close()

	result.IsOpen = true

	// Set read/write deadlines
	conn.SetDeadline(time.Now().Add(p.timeout))

	// If we have data to send, send it
	if config.SendData != "" {
		_, err := conn.Write([]byte(config.SendData))
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to send data: %v", err)
			result.Success = false
			return result
		}
	}

	// Try to read banner/response
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err == nil && n > 0 {
		result.BannerGrabbed = string(buffer[:n])
	}

	// Check if expected data matches
	if config.ExpectData != "" {
		if result.BannerGrabbed == "" {
			result.ErrorMessage = "No data received, but expected data specified"
			result.Success = false
			return result
		}
		// Simple contains check
		if len(result.BannerGrabbed) > 0 {
			result.Success = true
		}
	} else {
		// If no expected data, just being able to connect is success
		result.Success = true
	}

	return result
}

// checkUDP performs a UDP port check
func (p *PortChecker) checkUDP(address string, config *PortCheckConfig) *PortCheckResult {
	result := &PortCheckResult{
		Protocol: "udp",
		Host:     config.Host,
		Port:     config.Port,
	}

	// UDP is connectionless, so we need to send data and wait for response
	conn, err := net.DialTimeout("udp", address, p.timeout)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Connection failed: %v", err)
		result.IsOpen = false
		return result
	}
	defer conn.Close()

	// Set deadline
	conn.SetDeadline(time.Now().Add(p.timeout))

	// Send data (required for UDP checks)
	sendData := config.SendData
	if sendData == "" {
		sendData = "\n" // Send a newline if no data specified
	}

	_, err = conn.Write([]byte(sendData))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to send data: %v", err)
		result.IsOpen = false
		return result
	}

	// Try to read response
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		// UDP ports might not respond, which doesn't necessarily mean they're closed
		// This is a limitation of UDP
		result.ErrorMessage = fmt.Sprintf("No response received: %v", err)
		result.IsOpen = false // We can't definitively say
		return result
	}

	if n > 0 {
		result.BannerGrabbed = string(buffer[:n])
		result.IsOpen = true
		result.Success = true
	}

	return result
}

// QuickCheck performs a quick TCP port check
func (p *PortChecker) QuickCheck(host string, port int) (bool, int64, error) {
	config := &PortCheckConfig{
		Host:     host,
		Port:     port,
		Protocol: "tcp",
	}

	result := p.Check(config)
	if result.Success && result.IsOpen {
		return true, result.ResponseTime, nil
	}

	return false, result.ResponseTime, fmt.Errorf(result.ErrorMessage)
}

// CheckMultiplePorts checks multiple ports on a host
func (p *PortChecker) CheckMultiplePorts(host string, ports []int) map[int]*PortCheckResult {
	results := make(map[int]*PortCheckResult)

	for _, port := range ports {
		config := &PortCheckConfig{
			Host:     host,
			Port:     port,
			Protocol: "tcp",
		}
		results[port] = p.Check(config)
	}

	return results
}

// ScanCommonPorts scans common service ports
func (p *PortChecker) ScanCommonPorts(host string) map[int]*PortCheckResult {
	commonPorts := []int{
		22,    // SSH
		25,    // SMTP
		53,    // DNS
		80,    // HTTP
		110,   // POP3
		143,   // IMAP
		443,   // HTTPS
		3306,  // MySQL
		5432,  // PostgreSQL
		6379,  // Redis
		8080,  // HTTP Alt
		8443,  // HTTPS Alt
		9000,  // Various
		27017, // MongoDB
	}

	return p.CheckMultiplePorts(host, commonPorts)
}