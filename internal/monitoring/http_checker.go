package monitoring

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// HTTPChecker performs HTTP/HTTPS health checks
type HTTPChecker struct {
	client *http.Client
}

// NewHTTPChecker creates a new HTTP checker
func NewHTTPChecker(timeout int, verifySSL bool) *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: !verifySSL,
				},
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 10 redirects
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
	}
}

// HTTPCheckConfig represents the configuration for an HTTP health check
type HTTPCheckConfig struct {
	URL                string
	Method             string
	Headers            map[string]string
	Body               string
	ExpectedStatusCodes []int
	ExpectedContent    string
	FollowRedirects    bool
	VerifySSL          bool
	Timeout            int
	AuthType           string
	AuthCredentials    map[string]string
}

// HTTPCheckResult represents the result of an HTTP health check
type HTTPCheckResult struct {
	Success         bool
	StatusCode      int
	ResponseTime    int64
	ResponseSize    int64
	ErrorMessage    string
	Headers         map[string]string
	ContentMatched  bool
	RedirectChain   []string
	TLSVersion      string
	CertificateInfo map[string]interface{}
}

// Check performs an HTTP health check
func (h *HTTPChecker) Check(config *HTTPCheckConfig) *HTTPCheckResult {
	start := time.Now()
	result := &HTTPCheckResult{
		Headers:       make(map[string]string),
		RedirectChain: []string{},
	}

	// Create request
	req, err := http.NewRequest(config.Method, config.URL, strings.NewReader(config.Body))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}

	// Add custom headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// Add authentication
	if config.AuthType != "" {
		h.addAuthentication(req, config)
	}

	// Set User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "CasDash-Monitor/2.0")
	}

	// Perform request
	resp, err := h.client.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Request failed: %v", err)
		result.ResponseTime = time.Since(start).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	// Record response time
	result.ResponseTime = time.Since(start).Milliseconds()
	result.StatusCode = resp.StatusCode

	// Store response headers
	for key := range resp.Header {
		result.Headers[key] = resp.Header.Get(key)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to read response body: %v", err)
		return result
	}
	result.ResponseSize = int64(len(body))

	// Check TLS information
	if resp.TLS != nil {
		result.TLSVersion = h.getTLSVersion(resp.TLS.Version)
		result.CertificateInfo = h.extractCertificateInfo(resp.TLS)
	}

	// Check status code
	if len(config.ExpectedStatusCodes) > 0 {
		statusOK := false
		for _, expectedCode := range config.ExpectedStatusCodes {
			if resp.StatusCode == expectedCode {
				statusOK = true
				break
			}
		}
		if !statusOK {
			result.ErrorMessage = fmt.Sprintf("Unexpected status code: %d (expected: %v)", resp.StatusCode, config.ExpectedStatusCodes)
			return result
		}
	}

	// Check expected content
	if config.ExpectedContent != "" {
		bodyStr := string(body)
		matched, err := h.matchContent(bodyStr, config.ExpectedContent)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("Content matching error: %v", err)
			return result
		}
		result.ContentMatched = matched
		if !matched {
			result.ErrorMessage = fmt.Sprintf("Expected content not found: %s", config.ExpectedContent)
			return result
		}
	} else {
		result.ContentMatched = true
	}

	// Success!
	result.Success = true

	logrus.WithFields(logrus.Fields{
		"url":           config.URL,
		"status_code":   result.StatusCode,
		"response_time": result.ResponseTime,
		"success":       result.Success,
	}).Debug("HTTP check completed")

	return result
}

// addAuthentication adds authentication to the request
func (h *HTTPChecker) addAuthentication(req *http.Request, config *HTTPCheckConfig) {
	switch config.AuthType {
	case "basic":
		if username, ok := config.AuthCredentials["username"]; ok {
			if password, ok := config.AuthCredentials["password"]; ok {
				req.SetBasicAuth(username, password)
			}
		}
	case "bearer":
		if token, ok := config.AuthCredentials["token"]; ok {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	case "api_key":
		if key, ok := config.AuthCredentials["api_key"]; ok {
			if headerName, ok := config.AuthCredentials["header_name"]; ok {
				req.Header.Set(headerName, key)
			} else {
				req.Header.Set("X-API-Key", key)
			}
		}
	}
}

// matchContent checks if the response body matches the expected content
func (h *HTTPChecker) matchContent(body, expected string) (bool, error) {
	// Try as literal match first
	if strings.Contains(body, expected) {
		return true, nil
	}

	// Try as regex
	matched, err := regexp.MatchString(expected, body)
	if err != nil {
		return false, err
	}

	return matched, nil
}

// getTLSVersion returns a human-readable TLS version
func (h *HTTPChecker) getTLSVersion(version uint16) string {
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

// extractCertificateInfo extracts basic certificate information
func (h *HTTPChecker) extractCertificateInfo(tlsState *tls.ConnectionState) map[string]interface{} {
	info := make(map[string]interface{})

	if len(tlsState.PeerCertificates) > 0 {
		cert := tlsState.PeerCertificates[0]

		info["common_name"] = cert.Subject.CommonName
		info["issuer"] = cert.Issuer.CommonName
		info["not_before"] = cert.NotBefore.Format(time.RFC3339)
		info["not_after"] = cert.NotAfter.Format(time.RFC3339)
		info["days_until_expiry"] = int(time.Until(cert.NotAfter).Hours() / 24)
		info["is_expired"] = time.Now().After(cert.NotAfter)
		info["san_dns_names"] = cert.DNSNames
		info["signature_algorithm"] = cert.SignatureAlgorithm.String()

		// Check if it's self-signed
		info["is_self_signed"] = cert.Issuer.CommonName == cert.Subject.CommonName
	}

	info["negotiated_protocol"] = tlsState.NegotiatedProtocol
	info["cipher_suite"] = tls.CipherSuiteName(tlsState.CipherSuite)
	info["server_name"] = tlsState.ServerName

	return info
}

// QuickCheck performs a quick HTTP check with default settings
func (h *HTTPChecker) QuickCheck(url string) (bool, int, int64, error) {
	config := &HTTPCheckConfig{
		URL:                 url,
		Method:              "GET",
		ExpectedStatusCodes: []int{200, 201, 202, 204},
		FollowRedirects:     true,
		VerifySSL:           true,
		Timeout:             30,
	}

	result := h.Check(config)
	if result.Success {
		return true, result.StatusCode, result.ResponseTime, nil
	}

	return false, result.StatusCode, result.ResponseTime, fmt.Errorf(result.ErrorMessage)
}