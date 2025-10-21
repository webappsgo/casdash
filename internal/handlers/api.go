package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	. "github.com/casapps/casdash/internal/models"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// sendJSON sends a JSON response
func sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// sendSuccess sends a successful API response
func sendSuccess(w http.ResponseWriter, data interface{}) {
	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// sendError sends an error API response
func sendError(w http.ResponseWriter, statusCode int, message string) {
	sendJSON(w, statusCode, APIResponse{
		Success: false,
		Error:   message,
	})
}

// APIHealth returns the API health status
func (h *Handlers) APIHealth(w http.ResponseWriter, r *http.Request) {
	// Check application health
	if err := h.app.Health(); err != nil {
		sendError(w, http.StatusServiceUnavailable, "Application unhealthy")
		return
	}

	sendSuccess(w, map[string]interface{}{
		"status":  "healthy",
		"version": h.app.GetVersion(),
		"uptime":  time.Since(startTime).Seconds(),
	})
}

var startTime = time.Now()

// APIGetServices returns all services
func (h *Handlers) APIGetServices(w http.ResponseWriter, r *http.Request) {
	// Get user from session
	session, _ := h.sessionStore.Get(r, "casdash-session")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get user
	user, err := h.app.Users.GetUserByID(userID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	// Get services
	services, err := h.app.Services.ListServices(user.ID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to get services")
		return
	}

	sendSuccess(w, services)
}

// APIGetService returns a specific service
func (h *Handlers) APIGetService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid service ID")
		return
	}

	// Get user from session
	session, _ := h.sessionStore.Get(r, "casdash-session")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get service
	service, err := h.app.Services.GetServiceByID(serviceID)
	if err != nil {
		sendError(w, http.StatusNotFound, "Service not found")
		return
	}

	// TODO: Check user permissions for this service
	_ = userID

	sendSuccess(w, service)
}

// APICreateService creates a new service
func (h *Handlers) APICreateService(w http.ResponseWriter, r *http.Request) {
	// Get user from session
	session, _ := h.sessionStore.Get(r, "casdash-session")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var req struct {
		Name               string                 `json:"name"`
		URL                string                 `json:"url"`
		ServiceType        string                 `json:"service_type"`
		Category           string                 `json:"category"`
		Description        string                 `json:"description"`
		Icon               string                 `json:"icon"`
		MonitoringEnabled  bool                   `json:"monitoring_enabled"`
		CheckInterval      int                    `json:"check_interval"`
		ExpectedStatusCodes []int                 `json:"expected_status_codes"`
		Tags               []string               `json:"tags"`
		CustomFields       map[string]interface{} `json:"custom_fields"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" || req.URL == "" {
		sendError(w, http.StatusBadRequest, "Name and URL are required")
		return
	}

	// Create service object
	service := &Service{
		Name:              req.Name,
		URL:               req.URL,
		ServiceType:       req.ServiceType,
		Category:          req.Category,
		Description:       req.Description,
		Icon:              req.Icon,
		MonitoringEnabled: req.MonitoringEnabled,
		CheckInterval:     req.CheckInterval,
		Tags:              req.Tags,
		CustomFields:      req.CustomFields,
		CreatedBy:         userID,
	}

	// Set expected status codes
	if len(req.ExpectedStatusCodes) > 0 {
		service.ExpectedStatusCodes = req.ExpectedStatusCodes
	} else {
		service.ExpectedStatusCodes = []int{200, 201, 202, 204}
	}

	// Set user or organization ID based on mode
	if h.app.IsSaaSMode() {
		service.UserID = &userID
	} else {
		// TODO: Get user's organization ID
		// For now, leave it nil
	}

	// Create service
	createdService, err := h.app.Services.CreateService(service)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create service")
		return
	}

	sendSuccess(w, map[string]interface{}{
		"id":      createdService.ID,
		"message": "Service created successfully",
	})
}

// APIUpdateService updates a service
func (h *Handlers) APIUpdateService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid service ID")
		return
	}

	// Get user from session
	session, _ := h.sessionStore.Get(r, "casdash-session")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get existing service
	service, err := h.app.Services.GetServiceByID(serviceID)
	if err != nil {
		sendError(w, http.StatusNotFound, "Service not found")
		return
	}

	// TODO: Check user permissions for this service
	_ = userID

	// Parse request body
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Apply updates to service object
	// This is simplified - a proper implementation would validate and apply each field
	if name, ok := updates["name"].(string); ok {
		service.Name = name
	}
	if url, ok := updates["url"].(string); ok {
		service.URL = url
	}
	if desc, ok := updates["description"].(string); ok {
		service.Description = desc
	}

	// Update service
	if err := h.app.Services.UpdateService(service); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to update service")
		return
	}

	sendSuccess(w, map[string]interface{}{
		"message": "Service updated successfully",
	})
}

// APIDeleteService deletes a service
func (h *Handlers) APIDeleteService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid service ID")
		return
	}

	// Get user from session
	session, _ := h.sessionStore.Get(r, "casdash-session")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// TODO: Check user permissions for this service
	_ = userID

	// Delete service
	if err := h.app.Services.DeleteService(serviceID); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to delete service")
		return
	}

	sendSuccess(w, map[string]interface{}{
		"message": "Service deleted successfully",
	})
}

// APIGetServiceStatus returns the current status of a service
func (h *Handlers) APIGetServiceStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid service ID")
		return
	}

	// Get status from monitoring engine
	status, err := h.app.Monitoring.GetServiceStatus(serviceID)
	if err != nil {
		sendError(w, http.StatusNotFound, "Status not found")
		return
	}

	sendSuccess(w, status)
}

// APIForceCheck forces an immediate health check for a service
func (h *Handlers) APIForceCheck(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid service ID")
		return
	}

	// Get user from session
	session, _ := h.sessionStore.Get(r, "casdash-session")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Verify service exists
	_, err = h.app.Services.GetServiceByID(serviceID)
	if err != nil {
		sendError(w, http.StatusNotFound, "Service not found")
		return
	}

	// TODO: Check user permissions
	// TODO: Implement force check
	_ = userID

	// For now, return success
	sendSuccess(w, map[string]interface{}{
		"message": "Check initiated",
	})
}

// APIGetUptime returns uptime statistics for a service
func (h *Handlers) APIGetUptime(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid service ID")
		return
	}

	// Get period from query params
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "day"
	}

	// Get uptime stats
	stats, err := h.app.Monitoring.GetUptimeStats(serviceID, period)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to get uptime stats")
		return
	}

	sendSuccess(w, stats)
}

// APIGetMetrics returns performance metrics for a service
func (h *Handlers) APIGetMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid service ID")
		return
	}

	// Get time range from query params
	since := r.URL.Query().Get("since")
	until := r.URL.Query().Get("until")

	// Parse times
	var sinceTime, untilTime time.Time
	if since != "" {
		sinceTime, _ = time.Parse(time.RFC3339, since)
	} else {
		sinceTime = time.Now().Add(-24 * time.Hour)
	}
	if until != "" {
		untilTime, _ = time.Parse(time.RFC3339, until)
	} else {
		untilTime = time.Now()
	}

	// Get metrics from database
	query := `SELECT check_time, response_time_ms, success
			  FROM monitoring_results
			  WHERE service_id = ? AND check_time BETWEEN ? AND ?
			  ORDER BY check_time ASC`

	rows, err := h.app.DB.Query(query, serviceID, sinceTime, untilTime)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to get metrics")
		return
	}
	defer rows.Close()

	var metrics []map[string]interface{}
	for rows.Next() {
		var checkTime time.Time
		var responseTime int
		var success bool

		if err := rows.Scan(&checkTime, &responseTime, &success); err != nil {
			continue
		}

		metrics = append(metrics, map[string]interface{}{
			"timestamp":     checkTime.Unix(),
			"response_time": responseTime,
			"success":       success,
		})
	}

	sendSuccess(w, metrics)
}

// APIGetCertificates returns SSL certificate information
func (h *Handlers) APIGetCertificates(w http.ResponseWriter, r *http.Request) {
	// Get user from session
	session, _ := h.sessionStore.Get(r, "casdash-session")
	_, ok := session.Values["user_id"].(int)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get all SSL certificates
	query := `SELECT id, hostname, port, common_name, issuer_cn, not_before, not_after,
			  CAST((julianday(not_after) - julianday('now')) AS INTEGER) as days_until_expiry,
			  is_self_signed
			  FROM ssl_certificates
			  WHERE monitoring_enabled = 1
			  ORDER BY days_until_expiry ASC`

	rows, err := h.app.DB.Query(query)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to get certificates")
		return
	}
	defer rows.Close()

	var certificates []map[string]interface{}
	for rows.Next() {
		var id int
		var hostname, commonName, issuerCN string
		var port, daysUntilExpiry int
		var notBefore, notAfter time.Time
		var isSelfSigned bool

		if err := rows.Scan(&id, &hostname, &port, &commonName, &issuerCN, &notBefore, &notAfter, &daysUntilExpiry, &isSelfSigned); err != nil {
			continue
		}

		certificates = append(certificates, map[string]interface{}{
			"id":                id,
			"hostname":          hostname,
			"port":              port,
			"common_name":       commonName,
			"issuer":            issuerCN,
			"not_before":        notBefore,
			"not_after":         notAfter,
			"days_until_expiry": daysUntilExpiry,
			"is_self_signed":    isSelfSigned,
			"is_expired":        daysUntilExpiry < 0,
		})
	}

	sendSuccess(w, certificates)
}

// APILogin handles API authentication
func (h *Handlers) APILogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Authenticate user
	user, err := h.app.Users.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Create session
	session, _ := h.sessionStore.Get(r, "casdash-session")
	session.Values["user_id"] = user.ID
	session.Values["authenticated"] = true
	session.Save(r, w)

	// Generate API token
	// TODO: Implement proper API token generation
	token := "api_token_placeholder"

	sendSuccess(w, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// APIGetUser returns current user information
func (h *Handlers) APIGetUser(w http.ResponseWriter, r *http.Request) {
	// Get user from session
	session, _ := h.sessionStore.Get(r, "casdash-session")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get user
	user, err := h.app.Users.GetUserByID(userID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	sendSuccess(w, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

// APIDocs serves API documentation
func (h *Handlers) APIDocs(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"version": "1.0.0",
		"base_url": "/api/v1",
		"endpoints": map[string]interface{}{
			"health": map[string]interface{}{
				"method": "GET",
				"path":   "/health",
				"description": "Check API health",
			},
			"services": map[string]interface{}{
				"list": map[string]interface{}{
					"method": "GET",
					"path":   "/services",
					"description": "List all services",
				},
				"get": map[string]interface{}{
					"method": "GET",
					"path":   "/services/{id}",
					"description": "Get service by ID",
				},
				"create": map[string]interface{}{
					"method": "POST",
					"path":   "/services",
					"description": "Create new service",
				},
				"update": map[string]interface{}{
					"method": "PUT",
					"path":   "/services/{id}",
					"description": "Update service",
				},
				"delete": map[string]interface{}{
					"method": "DELETE",
					"path":   "/services/{id}",
					"description": "Delete service",
				},
			},
			"monitoring": map[string]interface{}{
				"status": map[string]interface{}{
					"method": "GET",
					"path":   "/services/{id}/status",
					"description": "Get service status",
				},
				"check": map[string]interface{}{
					"method": "POST",
					"path":   "/services/{id}/check",
					"description": "Force health check",
				},
				"uptime": map[string]interface{}{
					"method": "GET",
					"path":   "/services/{id}/uptime",
					"description": "Get uptime statistics",
				},
				"metrics": map[string]interface{}{
					"method": "GET",
					"path":   "/services/{id}/metrics",
					"description": "Get performance metrics",
				},
			},
			"certificates": map[string]interface{}{
				"list": map[string]interface{}{
					"method": "GET",
					"path":   "/certificates",
					"description": "List SSL certificates",
				},
			},
			"auth": map[string]interface{}{
				"login": map[string]interface{}{
					"method": "POST",
					"path":   "/auth/login",
					"description": "Authenticate and get token",
				},
				"user": map[string]interface{}{
					"method": "GET",
					"path":   "/auth/user",
					"description": "Get current user",
				},
			},
		},
	}

	sendSuccess(w, docs)
}

// LogAPIRequest logs API requests for debugging
func LogAPIRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log the request
		logrus.WithFields(logrus.Fields{
			"method":   r.Method,
			"path":     r.URL.Path,
			"duration": time.Since(start).Milliseconds(),
			"ip":       r.RemoteAddr,
		}).Debug("API request")
	})
}