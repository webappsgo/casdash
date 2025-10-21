package handlers

import (
	"net/http"

	"github.com/casapps/casdash/internal/models"
	"github.com/sirupsen/logrus"
)

// DashboardData represents the dashboard page data
type DashboardData struct {
	BaseData
	Services      []*ServiceCard
	Stats         DashboardStats
	Categories    []string
	RecentAlerts  []Alert
}

// ServiceCard represents a service card for the dashboard
type ServiceCard struct {
	ID               int
	Name             string
	URL              string
	ServiceType      string
	Category         string
	Description      string
	Icon             string
	Status           string
	LastResponseTime int
	LastChecked      string
	MaintenanceMode  bool
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalServices     int
	OnlineServices    int
	OfflineServices   int
	WarningServices   int
	UptimePercentage  float64
}

// Alert represents an alert/issue
type Alert struct {
	ID          int
	Title       string
	ServiceName string
	Severity    string
	CreatedAt   string
}

// Dashboard renders the main dashboard
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	// Get all services for the user
	services, err := h.app.Services.ListServices(user.ID)
	if err != nil {
		logrus.WithError(err).Error("Failed to list services")
		h.renderError(w, "Failed to load services", http.StatusInternalServerError)
		return
	}

	// Convert to service cards
	serviceCards := make([]*ServiceCard, 0, len(services))
	categories := make(map[string]bool)
	stats := DashboardStats{}

	for _, service := range services {
		// Get service status
		status := "unknown"
		lastResponseTime := 0
		lastChecked := "Never"

		// Try to get monitoring status (simplified for now)
		status = "online" // TODO: Get actual status from monitoring engine

		// Create service card
		card := &ServiceCard{
			ID:               service.ID,
			Name:             service.Name,
			URL:              service.URL,
			ServiceType:      service.ServiceType,
			Category:         service.Category,
			Description:      service.Description,
			Icon:             service.Icon,
			Status:           status,
			LastResponseTime: lastResponseTime,
			LastChecked:      lastChecked,
			MaintenanceMode:  service.MaintenanceMode,
		}

		serviceCards = append(serviceCards, card)

		// Track categories
		if service.Category != "" {
			categories[service.Category] = true
		}

		// Update stats
		stats.TotalServices++
		if status == "online" {
			stats.OnlineServices++
		} else if status == "offline" {
			stats.OfflineServices++
		} else if status == "warning" {
			stats.WarningServices++
		}
	}

	// Calculate uptime percentage
	if stats.TotalServices > 0 {
		stats.UptimePercentage = float64(stats.OnlineServices) / float64(stats.TotalServices) * 100
	}

	// Convert categories map to slice
	categoryList := make([]string, 0, len(categories))
	for category := range categories {
		categoryList = append(categoryList, category)
	}

	// Get recent alerts (simplified for now)
	alerts := []Alert{} // TODO: Get actual alerts from database

	// Prepare dashboard data
	data := DashboardData{
		BaseData:     h.getBaseData(r, user),
		Services:     serviceCards,
		Stats:        stats,
		Categories:   categoryList,
		RecentAlerts: alerts,
	}
	data.ActivePage = "dashboard"

	h.renderTemplate(w, "dashboard.html", data)
}

// Home redirects to dashboard or login
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	if h.isAuthenticated(r) {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}

	// Check if first run
	if h.app.IsFirstRun() {
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// Services lists all services
func (h *Handlers) Services(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	// Get all services
	services, err := h.app.Services.ListServices(user.ID)
	if err != nil {
		logrus.WithError(err).Error("Failed to list services")
		h.renderError(w, "Failed to load services", http.StatusInternalServerError)
		return
	}

	data := struct {
		BaseData
		Services []*models.Service
	}{
		BaseData: h.getBaseData(r, user),
		Services: services,
	}
	data.ActivePage = "services"

	h.renderTemplate(w, "services.html", data)
}

// AddService displays the add service form
func (h *Handlers) AddService(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	if r.Method == "GET" {
		data := h.getBaseData(r, user)
		data.ActivePage = "services"
		h.renderTemplate(w, "add-service.html", data)
		return
	}

	// Handle POST - create service
	if err := r.ParseForm(); err != nil {
		h.renderError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	service := &models.Service{
		Name:              r.FormValue("name"),
		URL:               r.FormValue("url"),
		ServiceType:       r.FormValue("service_type"),
		Category:          r.FormValue("category"),
		Description:       r.FormValue("description"),
		MonitoringEnabled: r.FormValue("monitoring_enabled") == "on",
		CreatedBy:         user.ID,
	}

	// Set user_id or organization_id based on mode
	if h.app.IsSaaSMode() {
		service.UserID = &user.ID
	} else if user.OrganizationID != nil {
		service.OrganizationID = user.OrganizationID
	}

	// Create service
	_, err := h.app.Services.CreateService(service)
	if err != nil {
		logrus.WithError(err).Error("Failed to create service")
		h.renderError(w, "Failed to create service: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to services list
	http.Redirect(w, r, "/services", http.StatusFound)
}

// ServiceDetails displays service details
func (h *Handlers) ServiceDetails(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	// Get service ID from URL
	// TODO: Extract ID from path using mux.Vars(r)

	data := h.getBaseData(r, user)
	data.ActivePage = "services"
	h.renderTemplate(w, "service-details.html", data)
}

// DiscoverServices displays the service discovery interface
func (h *Handlers) DiscoverServices(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	if r.Method == "GET" {
		data := h.getBaseData(r, user)
		data.ActivePage = "services"
		h.renderTemplate(w, "discover-services.html", data)
		return
	}

	// Handle POST - start discovery
	// TODO: Trigger discovery scan

	h.jsonResponse(w, map[string]interface{}{
		"success": true,
		"message": "Discovery scan started",
	})
}

// Monitoring displays the monitoring overview
func (h *Handlers) Monitoring(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	data := h.getBaseData(r, user)
	data.ActivePage = "monitoring"
	h.renderTemplate(w, "monitoring.html", data)
}

// Security displays the security overview
func (h *Handlers) Security(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	data := h.getBaseData(r, user)
	data.ActivePage = "security"
	h.renderTemplate(w, "security.html", data)
}

// Certificates displays SSL certificates
func (h *Handlers) Certificates(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	data := h.getBaseData(r, user)
	data.ActivePage = "certificates"
	h.renderTemplate(w, "certificates.html", data)
}

// Updates displays the updates overview
func (h *Handlers) Updates(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	data := h.getBaseData(r, user)
	data.ActivePage = "updates"
	h.renderTemplate(w, "updates.html", data)
}

// Support displays the support dashboard
func (h *Handlers) Support(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	data := h.getBaseData(r, user)
	data.ActivePage = "support"
	h.renderTemplate(w, "support.html", data)
}

// Maintenance displays maintenance overview
func (h *Handlers) Maintenance(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	data := h.getBaseData(r, user)
	data.ActivePage = "maintenance"
	h.renderTemplate(w, "maintenance.html", data)
}

// Profile displays user profile
func (h *Handlers) Profile(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	data := h.getBaseData(r, user)
	data.ActivePage = "profile"
	h.renderTemplate(w, "profile.html", data)
}

// Preferences displays user preferences
func (h *Handlers) Preferences(w http.ResponseWriter, r *http.Request) {
	user := h.requireAuth(w, r)
	if user == nil {
		return
	}

	data := h.getBaseData(r, user)
	data.ActivePage = "preferences"
	h.renderTemplate(w, "preferences.html", data)
}

// Admin displays admin dashboard
func (h *Handlers) Admin(w http.ResponseWriter, r *http.Request) {
	user := h.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	data := h.getBaseData(r, user)
	data.ActivePage = "admin"
	h.renderTemplate(w, "admin.html", data)
}

// Users displays user management
func (h *Handlers) Users(w http.ResponseWriter, r *http.Request) {
	user := h.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	// Get all users
	users, err := h.app.Users.ListUsers()
	if err != nil {
		logrus.WithError(err).Error("Failed to list users")
		h.renderError(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	data := struct {
		BaseData
		Users []*models.User
	}{
		BaseData: h.getBaseData(r, user),
		Users:    users,
	}
	data.ActivePage = "users"

	h.renderTemplate(w, "users.html", data)
}

// PublicStatusPage displays a user's public status page
func (h *Handlers) PublicStatusPage(w http.ResponseWriter, r *http.Request) {
	// Get username from URL
	// TODO: Extract username from path using mux.Vars(r)
	username := "demo" // Placeholder

	// Get public services
	services, err := h.app.Services.ListPublicServices(username)
	if err != nil {
		logrus.WithError(err).Error("Failed to list public services")
		h.renderError(w, "User not found", http.StatusNotFound)
		return
	}

	data := struct {
		BaseData
		Username string
		Services []*models.Service
	}{
		BaseData: h.getBaseData(r, nil),
		Username: username,
		Services: services,
	}

	h.renderTemplate(w, "public-status.html", data)
}

// Placeholder handlers for other routes
func (h *Handlers) EditService(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) ServiceMonitoring(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) ServiceSSL(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) ServiceSecurity(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) ServiceIssues(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) ServiceMaintenance(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) ServiceLogs(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) UptimeStats(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) PerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) AlertConfiguration(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) DependencyMap(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) SecurityScan(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) Vulnerabilities(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) Compliance(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) SecurityRecommendations(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) CertificateDetails(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) ExpiringCertificates(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) UpdatePolicies(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) UpdateHistory(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) ScheduledUpdates(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) LiveChat(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) SupportTickets(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) NewTicket(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) TicketDetails(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) KnowledgeBase(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) MaintenanceCalendar(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) ScheduleMaintenance(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) MaintenanceTemplates(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) MaintenanceHistory(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) InviteUser(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) UserDetails(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) SecuritySettings(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) AdminSettings(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) DiscoveryConfiguration(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) UICustomization(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) BackupManagement(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) SystemLogs(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) SystemHealth(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) BillingConfiguration(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) EmailConfiguration(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *Handlers) Integrations(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, "Not implemented yet", http.StatusNotImplemented)
}