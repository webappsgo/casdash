package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/casapps/casdash/internal/database"
	"github.com/sirupsen/logrus"
)

// Service represents a monitored service
type Service struct {
	ID                     int       `json:"id"`
	Name                   string    `json:"name"`
	URL                    string    `json:"url"`
	ServiceType            string    `json:"service_type"`
	Category               string    `json:"category"`
	Description            string    `json:"description"`
	Icon                   string    `json:"icon"`
	AuthType               string    `json:"auth_type"`
	AuthCredentials        string    `json:"-"` // Encrypted, never expose
	CustomHeaders          string    `json:"custom_headers,omitempty"`
	MonitoringEnabled      bool      `json:"monitoring_enabled"`
	CheckInterval          int       `json:"check_interval"`
	Timeout                int       `json:"timeout"`
	ExpectedStatusCodes    []int     `json:"expected_status_codes"`
	ExpectedContent        string    `json:"expected_content,omitempty"`
	FollowRedirects        bool      `json:"follow_redirects"`
	SSLVerify              bool      `json:"ssl_verify"`
	SSLMonitoringEnabled   *bool     `json:"ssl_monitoring_enabled,omitempty"`
	SSLCheckInterval       int       `json:"ssl_check_interval"`
	SSLHostname            string    `json:"ssl_hostname,omitempty"`
	SSLPort                int       `json:"ssl_port,omitempty"`
	PublicVisible          bool      `json:"public_visible"`
	PublicName             string    `json:"public_name,omitempty"`
	PublicDescription      string    `json:"public_description,omitempty"`
	MaintenanceMode        bool      `json:"maintenance_mode"`
	MaintenanceUntil       *time.Time `json:"maintenance_until,omitempty"`
	PositionX              int       `json:"position_x"`
	PositionY              int       `json:"position_y"`
	CardSize               string    `json:"card_size"`
	CardColor              string    `json:"card_color,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
	CreatedBy              int       `json:"created_by"`
	OrganizationID         *int      `json:"organization_id,omitempty"`
	UserID                 *int      `json:"user_id,omitempty"`
	Tags                   []string  `json:"tags"`
	CustomFields           map[string]interface{} `json:"custom_fields,omitempty"`
	DependsOn              []int     `json:"depends_on"`
	DependencyType         string    `json:"dependency_type"`
	ContainerID            string    `json:"container_id,omitempty"`
	ContainerImage         string    `json:"container_image,omitempty"`
	VMID                   string    `json:"vm_id,omitempty"`
	HostServer             string    `json:"host_server,omitempty"`
}

// ServiceManager handles service operations
type ServiceManager struct {
	db       *database.DB
	settings *Settings
	mode     string
}

// NewServiceManager creates a new service manager
func NewServiceManager(db *database.DB, settings *Settings, mode string) (*ServiceManager, error) {
	return &ServiceManager{
		db:       db,
		settings: settings,
		mode:     mode,
	}, nil
}

// CreateService creates a new service
func (sm *ServiceManager) CreateService(service *Service) (*Service, error) {
	// Validate required fields
	if service.Name == "" {
		return nil, fmt.Errorf("service name is required")
	}
	if service.URL == "" {
		return nil, fmt.Errorf("service URL is required")
	}

	// Set defaults
	if service.CheckInterval == 0 {
		interval, _ := sm.settings.GetInt("check_interval")
		if interval == 0 {
			interval = 300
		}
		service.CheckInterval = interval
	}

	if service.Timeout == 0 {
		timeout, _ := sm.settings.GetInt("check_timeout")
		if timeout == 0 {
			timeout = 30
		}
		service.Timeout = timeout
	}

	if service.AuthType == "" {
		service.AuthType = "none"
	}

	if service.CardSize == "" {
		service.CardSize = "medium"
	}

	if service.DependencyType == "" {
		service.DependencyType = "soft"
	}

	if len(service.ExpectedStatusCodes) == 0 {
		service.ExpectedStatusCodes = []int{200}
	}

	// Convert arrays to JSON
	statusCodesJSON, _ := json.Marshal(service.ExpectedStatusCodes)
	tagsJSON, _ := json.Marshal(service.Tags)
	customFieldsJSON, _ := json.Marshal(service.CustomFields)
	dependsOnJSON, _ := json.Marshal(service.DependsOn)

	// Insert service
	query := `INSERT INTO services (
		name, url, service_type, category, description, icon,
		auth_type, auth_credentials, custom_headers,
		monitoring_enabled, check_interval, timeout, expected_status_codes,
		expected_content, follow_redirects, ssl_verify, ssl_monitoring_enabled,
		ssl_check_interval, ssl_hostname, ssl_port,
		public_visible, public_name, public_description,
		maintenance_mode, maintenance_until,
		position_x, position_y, card_size, card_color,
		created_at, updated_at, created_by, organization_id, user_id,
		tags, custom_fields, depends_on, dependency_type,
		container_id, container_image, vm_id, host_server
	) VALUES (
		?, ?, ?, ?, ?, ?,
		?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?,
		?, ?, ?,
		?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?
	)`

	now := time.Now()
	result, err := sm.db.Exec(query,
		service.Name, service.URL, service.ServiceType, service.Category,
		service.Description, service.Icon,
		service.AuthType, service.AuthCredentials, service.CustomHeaders,
		service.MonitoringEnabled, service.CheckInterval, service.Timeout,
		string(statusCodesJSON), service.ExpectedContent, service.FollowRedirects,
		service.SSLVerify, service.SSLMonitoringEnabled, service.SSLCheckInterval,
		service.SSLHostname, service.SSLPort,
		service.PublicVisible, service.PublicName, service.PublicDescription,
		service.MaintenanceMode, service.MaintenanceUntil,
		service.PositionX, service.PositionY, service.CardSize, service.CardColor,
		now, now, service.CreatedBy, service.OrganizationID, service.UserID,
		string(tagsJSON), string(customFieldsJSON), string(dependsOnJSON),
		service.DependencyType,
		service.ContainerID, service.ContainerImage, service.VMID, service.HostServer,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	serviceID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get service ID: %w", err)
	}

	service.ID = int(serviceID)
	service.CreatedAt = now
	service.UpdatedAt = now

	logrus.WithFields(logrus.Fields{
		"service_id": serviceID,
		"name":       service.Name,
		"type":       service.ServiceType,
		"url":        service.URL,
	}).Info("Service created successfully")

	return service, nil
}

// GetServiceByID retrieves a service by ID
func (sm *ServiceManager) GetServiceByID(id int) (*Service, error) {
	query := `SELECT id, name, url, service_type, category, description, icon,
					 auth_type, custom_headers, monitoring_enabled, check_interval,
					 timeout, expected_status_codes, expected_content, follow_redirects,
					 ssl_verify, ssl_monitoring_enabled, ssl_check_interval,
					 ssl_hostname, ssl_port, public_visible, public_name,
					 public_description, maintenance_mode, maintenance_until,
					 position_x, position_y, card_size, card_color,
					 created_at, updated_at, created_by, organization_id, user_id,
					 tags, custom_fields, depends_on, dependency_type,
					 container_id, container_image, vm_id, host_server
			  FROM services WHERE id = ?`

	service := &Service{}
	var orgID, userID sql.NullInt64
	var sslMonitoring sql.NullBool
	var maintenanceUntil sql.NullTime
	var statusCodesJSON, tagsJSON, customFieldsJSON, dependsOnJSON string
	var customHeaders, expectedContent, publicName, publicDescription sql.NullString
	var cardColor, containerID, containerImage, vmID, hostServer sql.NullString
	var sslHostname sql.NullString
	var sslPort sql.NullInt64

	err := sm.db.QueryRow(query, id).Scan(
		&service.ID, &service.Name, &service.URL, &service.ServiceType,
		&service.Category, &service.Description, &service.Icon,
		&service.AuthType, &customHeaders, &service.MonitoringEnabled,
		&service.CheckInterval, &service.Timeout, &statusCodesJSON,
		&expectedContent, &service.FollowRedirects, &service.SSLVerify,
		&sslMonitoring, &service.SSLCheckInterval, &sslHostname, &sslPort,
		&service.PublicVisible, &publicName, &publicDescription,
		&service.MaintenanceMode, &maintenanceUntil,
		&service.PositionX, &service.PositionY, &service.CardSize, &cardColor,
		&service.CreatedAt, &service.UpdatedAt, &service.CreatedBy,
		&orgID, &userID, &tagsJSON, &customFieldsJSON, &dependsOnJSON,
		&service.DependencyType, &containerID, &containerImage, &vmID, &hostServer,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("service not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Handle nullable fields
	if orgID.Valid {
		orgIDInt := int(orgID.Int64)
		service.OrganizationID = &orgIDInt
	}
	if userID.Valid {
		userIDInt := int(userID.Int64)
		service.UserID = &userIDInt
	}
	if sslMonitoring.Valid {
		service.SSLMonitoringEnabled = &sslMonitoring.Bool
	}
	if maintenanceUntil.Valid {
		service.MaintenanceUntil = &maintenanceUntil.Time
	}
	if customHeaders.Valid {
		service.CustomHeaders = customHeaders.String
	}
	if expectedContent.Valid {
		service.ExpectedContent = expectedContent.String
	}
	if publicName.Valid {
		service.PublicName = publicName.String
	}
	if publicDescription.Valid {
		service.PublicDescription = publicDescription.String
	}
	if cardColor.Valid {
		service.CardColor = cardColor.String
	}
	if containerID.Valid {
		service.ContainerID = containerID.String
	}
	if containerImage.Valid {
		service.ContainerImage = containerImage.String
	}
	if vmID.Valid {
		service.VMID = vmID.String
	}
	if hostServer.Valid {
		service.HostServer = hostServer.String
	}
	if sslHostname.Valid {
		service.SSLHostname = sslHostname.String
	}
	if sslPort.Valid {
		service.SSLPort = int(sslPort.Int64)
	}

	// Parse JSON fields
	json.Unmarshal([]byte(statusCodesJSON), &service.ExpectedStatusCodes)
	json.Unmarshal([]byte(tagsJSON), &service.Tags)
	json.Unmarshal([]byte(customFieldsJSON), &service.CustomFields)
	json.Unmarshal([]byte(dependsOnJSON), &service.DependsOn)

	return service, nil
}

// ListServices returns services based on mode and user
func (sm *ServiceManager) ListServices(userID int) ([]*Service, error) {
	var query string
	var args []interface{}

	switch sm.mode {
	case "enterprise":
		// In enterprise mode, return all services (with future role-based filtering)
		query = `SELECT id FROM services ORDER BY name`
	case "saas":
		// In SaaS mode, only return user's services
		query = `SELECT id FROM services WHERE user_id = ? ORDER BY name`
		args = append(args, userID)
	default:
		return nil, fmt.Errorf("unknown mode: %s", sm.mode)
	}

	rows, err := sm.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	defer rows.Close()

	var services []*Service
	for rows.Next() {
		var serviceID int
		if err := rows.Scan(&serviceID); err != nil {
			return nil, fmt.Errorf("failed to scan service ID: %w", err)
		}

		service, err := sm.GetServiceByID(serviceID)
		if err != nil {
			logrus.WithError(err).WithField("service_id", serviceID).Warn("Failed to get service")
			continue
		}

		services = append(services, service)
	}

	return services, nil
}

// ListPublicServices returns services marked as public for status page
func (sm *ServiceManager) ListPublicServices(username string) ([]*Service, error) {
	var query string
	var args []interface{}

	switch sm.mode {
	case "enterprise":
		// In enterprise mode, show organization's public services
		query = `SELECT s.id FROM services s
				 JOIN users u ON s.organization_id = u.organization_id
				 WHERE u.username = ? AND s.public_visible = 1
				 ORDER BY s.name`
		args = append(args, username)
	case "saas":
		// In SaaS mode, show user's public services
		query = `SELECT s.id FROM services s
				 JOIN users u ON s.user_id = u.id
				 WHERE u.username = ? AND s.public_visible = 1
				 ORDER BY s.name`
		args = append(args, username)
	default:
		return nil, fmt.Errorf("unknown mode: %s", sm.mode)
	}

	rows, err := sm.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list public services: %w", err)
	}
	defer rows.Close()

	var services []*Service
	for rows.Next() {
		var serviceID int
		if err := rows.Scan(&serviceID); err != nil {
			return nil, fmt.Errorf("failed to scan service ID: %w", err)
		}

		service, err := sm.GetServiceByID(serviceID)
		if err != nil {
			logrus.WithError(err).WithField("service_id", serviceID).Warn("Failed to get public service")
			continue
		}

		services = append(services, service)
	}

	return services, nil
}

// UpdateService updates an existing service
func (sm *ServiceManager) UpdateService(service *Service) error {
	// Convert arrays to JSON
	statusCodesJSON, _ := json.Marshal(service.ExpectedStatusCodes)
	tagsJSON, _ := json.Marshal(service.Tags)
	customFieldsJSON, _ := json.Marshal(service.CustomFields)
	dependsOnJSON, _ := json.Marshal(service.DependsOn)

	query := `UPDATE services SET
		name = ?, url = ?, service_type = ?, category = ?, description = ?, icon = ?,
		auth_type = ?, custom_headers = ?,
		monitoring_enabled = ?, check_interval = ?, timeout = ?, expected_status_codes = ?,
		expected_content = ?, follow_redirects = ?, ssl_verify = ?, ssl_monitoring_enabled = ?,
		ssl_check_interval = ?, ssl_hostname = ?, ssl_port = ?,
		public_visible = ?, public_name = ?, public_description = ?,
		maintenance_mode = ?, maintenance_until = ?,
		position_x = ?, position_y = ?, card_size = ?, card_color = ?,
		updated_at = ?, tags = ?, custom_fields = ?, depends_on = ?, dependency_type = ?,
		container_id = ?, container_image = ?, vm_id = ?, host_server = ?
		WHERE id = ?`

	_, err := sm.db.Exec(query,
		service.Name, service.URL, service.ServiceType, service.Category,
		service.Description, service.Icon, service.AuthType, service.CustomHeaders,
		service.MonitoringEnabled, service.CheckInterval, service.Timeout,
		string(statusCodesJSON), service.ExpectedContent, service.FollowRedirects,
		service.SSLVerify, service.SSLMonitoringEnabled, service.SSLCheckInterval,
		service.SSLHostname, service.SSLPort,
		service.PublicVisible, service.PublicName, service.PublicDescription,
		service.MaintenanceMode, service.MaintenanceUntil,
		service.PositionX, service.PositionY, service.CardSize, service.CardColor,
		time.Now(), string(tagsJSON), string(customFieldsJSON), string(dependsOnJSON),
		service.DependencyType, service.ContainerID, service.ContainerImage,
		service.VMID, service.HostServer, service.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	logrus.WithField("service_id", service.ID).Info("Service updated successfully")
	return nil
}

// DeleteService removes a service
func (sm *ServiceManager) DeleteService(id int) error {
	// TODO: Check for dependencies and handle cleanup

	query := `DELETE FROM services WHERE id = ?`
	result, err := sm.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("service not found: %d", id)
	}

	logrus.WithField("service_id", id).Info("Service deleted successfully")
	return nil
}

// GetServicesByType returns services of a specific type
func (sm *ServiceManager) GetServicesByType(serviceType string) ([]*Service, error) {
	query := `SELECT id FROM services WHERE service_type = ? ORDER BY name`

	rows, err := sm.db.Query(query, serviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get services by type: %w", err)
	}
	defer rows.Close()

	var services []*Service
	for rows.Next() {
		var serviceID int
		if err := rows.Scan(&serviceID); err != nil {
			return nil, fmt.Errorf("failed to scan service ID: %w", err)
		}

		service, err := sm.GetServiceByID(serviceID)
		if err != nil {
			logrus.WithError(err).WithField("service_id", serviceID).Warn("Failed to get service")
			continue
		}

		services = append(services, service)
	}

	return services, nil
}