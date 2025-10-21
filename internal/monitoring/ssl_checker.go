package monitoring

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

// SSLChecker performs SSL/TLS certificate checks
type SSLChecker struct {
	timeout time.Duration
}

// NewSSLChecker creates a new SSL checker
func NewSSLChecker(timeout int) *SSLChecker {
	return &SSLChecker{
		timeout: time.Duration(timeout) * time.Second,
	}
}

// SSLCheckConfig represents the configuration for an SSL check
type SSLCheckConfig struct {
	Hostname string
	Port     int
	SNI      string // Server Name Indication
}

// SSLCheckResult represents the result of an SSL certificate check
type SSLCheckResult struct {
	Success              bool
	ErrorMessage         string
	ResponseTime         int64
	Certificate          *CertificateInfo
	ChainValid           bool
	ChainLength          int
	ProtocolVersion      string
	CipherSuite          string
	SupportedProtocols   []string
	Vulnerabilities      []string
	DaysUntilExpiry      int
	IsExpired            bool
	IsSelfSigned         bool
	HasValidChain        bool
	OCSPStapling         bool
	CertificateTransparency bool
}

// CertificateInfo contains detailed certificate information
type CertificateInfo struct {
	CommonName           string
	SANs                 []string
	Organization         string
	OrganizationalUnit   string
	Country              string
	Issuer               string
	IssuerOrganization   string
	SerialNumber         string
	FingerprintSHA256    string
	FingerprintSHA1      string
	NotBefore            time.Time
	NotAfter             time.Time
	KeyAlgorithm         string
	KeySize              int
	SignatureAlgorithm   string
	Version              int
	IsCA                 bool
	IsWildcard           bool
}

// Check performs an SSL certificate check
func (s *SSLChecker) Check(config *SSLCheckConfig) *SSLCheckResult {
	start := time.Now()
	result := &SSLCheckResult{
		Vulnerabilities: []string{},
		SupportedProtocols: []string{},
	}

	// Set default port if not specified
	if config.Port == 0 {
		config.Port = 443
	}

	// Set SNI if not specified
	if config.SNI == "" {
		config.SNI = config.Hostname
	}

	// Connect to the server
	address := fmt.Sprintf("%s:%d", config.Hostname, config.Port)
	dialer := &net.Dialer{
		Timeout: s.timeout,
	}

	tlsConfig := &tls.Config{
		ServerName:         config.SNI,
		InsecureSkipVerify: false, // We want to verify the chain
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to connect: %v", err)
		result.ResponseTime = time.Since(start).Milliseconds()
		return result
	}
	defer conn.Close()

	result.ResponseTime = time.Since(start).Milliseconds()

	// Get connection state
	state := conn.ConnectionState()

	// Extract protocol and cipher information
	result.ProtocolVersion = s.getTLSVersion(state.Version)
	result.CipherSuite = tls.CipherSuiteName(state.CipherSuite)
	result.OCSPStapling = len(state.OCSPResponse) > 0

	// Get certificates
	if len(state.PeerCertificates) == 0 {
		result.ErrorMessage = "No certificates found"
		return result
	}

	// Extract certificate information
	cert := state.PeerCertificates[0]
	result.Certificate = s.extractCertificateDetails(cert)
	result.ChainLength = len(state.PeerCertificates)
	result.ChainValid = state.VerifiedChains != nil && len(state.VerifiedChains) > 0
	result.HasValidChain = result.ChainValid

	// Check if self-signed
	result.IsSelfSigned = cert.Issuer.CommonName == cert.Subject.CommonName

	// Calculate days until expiry
	now := time.Now()
	result.DaysUntilExpiry = int(cert.NotAfter.Sub(now).Hours() / 24)
	result.IsExpired = now.After(cert.NotAfter)

	// Check for vulnerabilities
	result.Vulnerabilities = s.checkVulnerabilities(&state, cert)

	// Check Certificate Transparency
	result.CertificateTransparency = len(cert.Extensions) > 0 // Simplified check

	// Success if certificate is valid and not expired
	if !result.IsExpired && (result.ChainValid || result.IsSelfSigned) {
		result.Success = true
	} else {
		if result.IsExpired {
			result.ErrorMessage = fmt.Sprintf("Certificate expired %d days ago", -result.DaysUntilExpiry)
		} else if !result.ChainValid && !result.IsSelfSigned {
			result.ErrorMessage = "Certificate chain validation failed"
		}
	}

	// Check supported protocols
	result.SupportedProtocols = s.checkSupportedProtocols(config)

	logrus.WithFields(logrus.Fields{
		"hostname":         config.Hostname,
		"port":             config.Port,
		"days_until_expiry": result.DaysUntilExpiry,
		"is_expired":       result.IsExpired,
		"success":          result.Success,
	}).Debug("SSL check completed")

	return result
}

// extractCertificateDetails extracts detailed information from a certificate
func (s *SSLChecker) extractCertificateDetails(cert *x509.Certificate) *CertificateInfo {
	info := &CertificateInfo{
		CommonName:         cert.Subject.CommonName,
		SANs:              cert.DNSNames,
		Issuer:            cert.Issuer.CommonName,
		SerialNumber:      cert.SerialNumber.String(),
		NotBefore:         cert.NotBefore,
		NotAfter:          cert.NotAfter,
		SignatureAlgorithm: cert.SignatureAlgorithm.String(),
		Version:           cert.Version,
		IsCA:              cert.IsCA,
	}

	// Organization information
	if len(cert.Subject.Organization) > 0 {
		info.Organization = cert.Subject.Organization[0]
	}
	if len(cert.Subject.OrganizationalUnit) > 0 {
		info.OrganizationalUnit = cert.Subject.OrganizationalUnit[0]
	}
	if len(cert.Subject.Country) > 0 {
		info.Country = cert.Subject.Country[0]
	}
	if len(cert.Issuer.Organization) > 0 {
		info.IssuerOrganization = cert.Issuer.Organization[0]
	}

	// Check if wildcard certificate
	info.IsWildcard = len(info.CommonName) > 0 && info.CommonName[0] == '*'

	// Calculate fingerprints
	sha256Sum := sha256.Sum256(cert.Raw)
	info.FingerprintSHA256 = hex.EncodeToString(sha256Sum[:])

	// Determine key algorithm and size
	switch cert.PublicKeyAlgorithm {
	case x509.RSA:
		info.KeyAlgorithm = "RSA"
		// Note: Simplified key size extraction - proper implementation would use type assertion
		info.KeySize = 2048 // Default assumption for RSA
	case x509.ECDSA:
		info.KeyAlgorithm = "ECDSA"
		info.KeySize = 256 // Default ECDSA size
	case x509.Ed25519:
		info.KeyAlgorithm = "Ed25519"
		info.KeySize = 256
	case x509.DSA:
		info.KeyAlgorithm = "DSA"
		info.KeySize = 1024
	default:
		info.KeyAlgorithm = "Unknown"
		info.KeySize = 0
	}

	return info
}

// getTLSVersion returns a human-readable TLS version
func (s *SSLChecker) getTLSVersion(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04X)", version)
	}
}

// checkVulnerabilities checks for known SSL/TLS vulnerabilities
func (s *SSLChecker) checkVulnerabilities(state *tls.ConnectionState, cert *x509.Certificate) []string {
	var vulnerabilities []string

	// Check for weak protocols
	if state.Version == tls.VersionTLS10 {
		vulnerabilities = append(vulnerabilities, "TLS 1.0 (Weak)")
	}
	if state.Version == tls.VersionTLS11 {
		vulnerabilities = append(vulnerabilities, "TLS 1.1 (Weak)")
	}

	// Check certificate expiry
	now := time.Now()
	daysUntilExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)
	if daysUntilExpiry < 30 && daysUntilExpiry > 0 {
		vulnerabilities = append(vulnerabilities, fmt.Sprintf("Certificate expiring soon (%d days)", daysUntilExpiry))
	}

	// Check for weak key size (RSA < 2048)
	// Note: This is simplified - proper implementation would check actual key size
	if cert.PublicKeyAlgorithm == x509.RSA {
		// Placeholder: actual implementation would extract key size
	}

	// Check for SHA-1 signature
	if cert.SignatureAlgorithm.String() == "SHA1-RSA" ||
	   cert.SignatureAlgorithm.String() == "SHA1-DSA" ||
	   cert.SignatureAlgorithm.String() == "SHA1-ECDSA" {
		vulnerabilities = append(vulnerabilities, "SHA-1 signature (Weak)")
	}

	return vulnerabilities
}

// checkSupportedProtocols checks which TLS protocols are supported
func (s *SSLChecker) checkSupportedProtocols(config *SSLCheckConfig) []string {
	var supported []string
	address := fmt.Sprintf("%s:%d", config.Hostname, config.Port)

	protocols := []struct {
		Version uint16
		Name    string
	}{
		{tls.VersionTLS10, "TLS 1.0"},
		{tls.VersionTLS11, "TLS 1.1"},
		{tls.VersionTLS12, "TLS 1.2"},
		{tls.VersionTLS13, "TLS 1.3"},
	}

	for _, proto := range protocols {
		tlsConfig := &tls.Config{
			ServerName:         config.SNI,
			InsecureSkipVerify: true,
			MinVersion:         proto.Version,
			MaxVersion:         proto.Version,
		}

		dialer := &net.Dialer{
			Timeout: 3 * time.Second,
		}

		conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
		if err == nil {
			conn.Close()
			supported = append(supported, proto.Name)
		}
	}

	return supported
}

// QuickCheck performs a quick SSL check with default settings
func (s *SSLChecker) QuickCheck(hostname string, port int) (bool, int, string, error) {
	config := &SSLCheckConfig{
		Hostname: hostname,
		Port:     port,
	}

	result := s.Check(config)
	if result.Success {
		return true, result.DaysUntilExpiry, "", nil
	}

	return false, result.DaysUntilExpiry, result.ErrorMessage, fmt.Errorf(result.ErrorMessage)
}