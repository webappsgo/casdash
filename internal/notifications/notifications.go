package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/casapps/casdash/internal/database"
	"github.com/casapps/casdash/internal/websocket"
	"github.com/sirupsen/logrus"
)

// Priority levels for notifications
const (
	PriorityLow      = "low"
	PriorityMedium   = "medium"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)

// Notification types
const (
	TypeServiceDown     = "service_down"
	TypeServiceUp       = "service_up"
	TypeSSLExpiring     = "ssl_expiring"
	TypeSSLExpired      = "ssl_expired"
	TypeUpdateAvailable = "update_available"
	TypeSecurityIssue   = "security_issue"
	TypeCustom          = "custom"
)

// Manager manages the notification system
type Manager struct {
	db       *database.DB
	wsHub    *websocket.Hub
	channels map[int]*Channel // channel ID -> Channel
	mutex    sync.RWMutex
	stop     chan struct{}
}

// Channel represents a notification channel
type Channel struct {
	ID           int
	Name         string
	Type         string // email, slack, discord, webhook, sms
	Enabled      bool
	Config       map[string]interface{}
	RateLimit    int // messages per window
	RateLimitWin int // window in seconds
	lastSent     time.Time
	sentCount    int
	mutex        sync.Mutex
}

// Notification represents a notification message
type Notification struct {
	ID        int
	Title     string
	Message   string
	Priority  string
	Type      string
	ServiceID *int
	UserID    *int
	Timestamp time.Time
	Sent      bool
	Read      bool
	Metadata  map[string]interface{}
}

// Rule represents a notification rule
type Rule struct {
	ID         int
	Name       string
	ServiceID  *int // nil for global rules
	ChannelID  int
	Events     []string
	Conditions map[string]interface{}
	Enabled    bool
}

// New creates a new notification manager
func New(db *database.DB, wsHub *websocket.Hub) *Manager {
	return &Manager{
		db:       db,
		wsHub:    wsHub,
		channels: make(map[int]*Channel),
		stop:     make(chan struct{}),
	}
}

// Start starts the notification manager
func (m *Manager) Start() error {
	logrus.Info("Starting notification manager")

	// Load notification channels
	if err := m.loadChannels(); err != nil {
		return fmt.Errorf("failed to load notification channels: %w", err)
	}

	// Start background processor
	go m.processNotifications()

	return nil
}

// Shutdown gracefully stops the notification manager
func (m *Manager) Shutdown(ctx context.Context) error {
	logrus.Info("Shutting down notification manager")
	close(m.stop)
	return nil
}

// loadChannels loads notification channels from database
func (m *Manager) loadChannels() error {
	rows, err := m.db.Query(`
		SELECT id, name, channel_type, enabled, config,
		       COALESCE(rate_limit_count, 0), COALESCE(rate_limit_window, 0)
		FROM notification_channels
		WHERE enabled = true
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for rows.Next() {
		channel := &Channel{}
		var configJSON string

		err := rows.Scan(
			&channel.ID,
			&channel.Name,
			&channel.Type,
			&channel.Enabled,
			&configJSON,
			&channel.RateLimit,
			&channel.RateLimitWin,
		)
		if err != nil {
			logrus.WithError(err).Error("Failed to scan notification channel")
			continue
		}

		// Parse config JSON
		if err := json.Unmarshal([]byte(configJSON), &channel.Config); err != nil {
			logrus.WithError(err).WithField("channel_id", channel.ID).Error("Failed to parse channel config")
			continue
		}

		m.channels[channel.ID] = channel
	}

	logrus.WithField("count", len(m.channels)).Info("Loaded notification channels")
	return nil
}

// Send sends a notification
func (m *Manager) Send(notification *Notification) error {
	// Store notification in database
	if err := m.storeNotification(notification); err != nil {
		return fmt.Errorf("failed to store notification: %w", err)
	}

	// Send via WebSocket for real-time delivery
	if m.wsHub != nil {
		m.sendViaWebSocket(notification)
	}

	// Get applicable rules
	rules, err := m.getApplicableRules(notification)
	if err != nil {
		logrus.WithError(err).Error("Failed to get notification rules")
		return err
	}

	// Send to each channel based on rules
	for _, rule := range rules {
		if err := m.sendToChannel(rule.ChannelID, notification); err != nil {
			logrus.WithError(err).WithField("channel_id", rule.ChannelID).Error("Failed to send notification")
		}
	}

	return nil
}

// SendServiceDown sends a service down notification
func (m *Manager) SendServiceDown(serviceID int, serviceName string, errorMessage string) error {
	notification := &Notification{
		Title:     fmt.Sprintf("Service Down: %s", serviceName),
		Message:   fmt.Sprintf("Service %s is down. Error: %s", serviceName, errorMessage),
		Priority:  PriorityCritical,
		Type:      TypeServiceDown,
		ServiceID: &serviceID,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"error": errorMessage,
		},
	}

	return m.Send(notification)
}

// SendServiceUp sends a service recovery notification
func (m *Manager) SendServiceUp(serviceID int, serviceName string, downtime time.Duration) error {
	notification := &Notification{
		Title:     fmt.Sprintf("Service Recovered: %s", serviceName),
		Message:   fmt.Sprintf("Service %s is back up. Downtime: %s", serviceName, downtime),
		Priority:  PriorityMedium,
		Type:      TypeServiceUp,
		ServiceID: &serviceID,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"downtime_seconds": downtime.Seconds(),
		},
	}

	return m.Send(notification)
}

// SendSSLExpiring sends an SSL expiring notification
func (m *Manager) SendSSLExpiring(serviceID int, serviceName string, daysUntilExpiry int) error {
	priority := PriorityMedium
	if daysUntilExpiry <= 7 {
		priority = PriorityHigh
	}
	if daysUntilExpiry <= 3 {
		priority = PriorityCritical
	}

	notification := &Notification{
		Title:     fmt.Sprintf("SSL Certificate Expiring: %s", serviceName),
		Message:   fmt.Sprintf("SSL certificate for %s expires in %d days", serviceName, daysUntilExpiry),
		Priority:  priority,
		Type:      TypeSSLExpiring,
		ServiceID: &serviceID,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"days_until_expiry": daysUntilExpiry,
		},
	}

	return m.Send(notification)
}

// storeNotification stores a notification in the database
func (m *Manager) storeNotification(n *Notification) error {
	metadataJSON, err := json.Marshal(n.Metadata)
	if err != nil {
		return err
	}

	result, err := m.db.Exec(`
		INSERT INTO notifications
		(title, message, priority, type, service_id, user_id, created_at, sent, read, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, n.Title, n.Message, n.Priority, n.Type, n.ServiceID, n.UserID, n.Timestamp, n.Sent, n.Read, string(metadataJSON))
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err == nil {
		n.ID = int(id)
	}

	return nil
}

// sendViaWebSocket sends notification via WebSocket
func (m *Manager) sendViaWebSocket(n *Notification) {
	update := websocket.NotificationUpdate{
		ID:        n.ID,
		Title:     n.Title,
		Message:   n.Message,
		Priority:  n.Priority,
		Timestamp: n.Timestamp,
	}

	if n.ServiceID != nil {
		update.ServiceID = *n.ServiceID
	}

	m.wsHub.BroadcastNotification(update)
}

// getApplicableRules returns rules that apply to this notification
func (m *Manager) getApplicableRules(n *Notification) ([]*Rule, error) {
	// Query database for matching rules
	query := `
		SELECT id, name, service_id, channel_id, events, conditions, enabled
		FROM notification_rules
		WHERE enabled = true
		AND (service_id IS NULL OR service_id = ?)
	`

	rows, err := m.db.Query(query, n.ServiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*Rule
	for rows.Next() {
		rule := &Rule{}
		var eventsJSON, conditionsJSON string

		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.ServiceID,
			&rule.ChannelID,
			&eventsJSON,
			&conditionsJSON,
			&rule.Enabled,
		)
		if err != nil {
			continue
		}

		// Parse JSON fields
		json.Unmarshal([]byte(eventsJSON), &rule.Events)
		json.Unmarshal([]byte(conditionsJSON), &rule.Conditions)

		// Check if rule applies to this notification type
		if m.ruleMatches(rule, n) {
			rules = append(rules, rule)
		}
	}

	return rules, nil
}

// ruleMatches checks if a rule matches a notification
func (m *Manager) ruleMatches(rule *Rule, n *Notification) bool {
	// Check if event type matches
	eventMatches := false
	for _, event := range rule.Events {
		if event == n.Type {
			eventMatches = true
			break
		}
	}

	if !eventMatches {
		return false
	}

	// Check priority conditions if specified
	if minPriority, ok := rule.Conditions["min_priority"].(string); ok {
		if !m.priorityMeetsMinimum(n.Priority, minPriority) {
			return false
		}
	}

	return true
}

// priorityMeetsMinimum checks if priority meets minimum requirement
func (m *Manager) priorityMeetsMinimum(priority, minPriority string) bool {
	priorities := map[string]int{
		PriorityLow:      1,
		PriorityMedium:   2,
		PriorityHigh:     3,
		PriorityCritical: 4,
	}

	return priorities[priority] >= priorities[minPriority]
}

// sendToChannel sends notification to a specific channel
func (m *Manager) sendToChannel(channelID int, n *Notification) error {
	m.mutex.RLock()
	channel, exists := m.channels[channelID]
	m.mutex.RUnlock()

	if !exists || !channel.Enabled {
		return fmt.Errorf("channel %d not found or disabled", channelID)
	}

	// Check rate limiting
	if !channel.checkRateLimit() {
		logrus.WithField("channel_id", channelID).Debug("Rate limit exceeded, skipping notification")
		return nil
	}

	// Send based on channel type
	switch channel.Type {
	case "email":
		return m.sendEmail(channel, n)
	case "slack":
		return m.sendSlack(channel, n)
	case "discord":
		return m.sendDiscord(channel, n)
	case "webhook":
		return m.sendWebhook(channel, n)
	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}

// checkRateLimit checks if channel can send another message
func (c *Channel) checkRateLimit() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.RateLimit == 0 {
		return true // No rate limit
	}

	now := time.Now()
	window := time.Duration(c.RateLimitWin) * time.Second

	// Reset counter if outside window
	if now.Sub(c.lastSent) > window {
		c.sentCount = 0
		c.lastSent = now
	}

	// Check if under limit
	if c.sentCount < c.RateLimit {
		c.sentCount++
		return true
	}

	return false
}

// sendEmail sends notification via email (placeholder)
func (m *Manager) sendEmail(channel *Channel, n *Notification) error {
	// TODO: Implement email sending
	logrus.WithFields(logrus.Fields{
		"channel": channel.Name,
		"title":   n.Title,
	}).Debug("Email notification (not implemented)")
	return nil
}

// sendSlack sends notification to Slack (placeholder)
func (m *Manager) sendSlack(channel *Channel, n *Notification) error {
	// TODO: Implement Slack webhook
	logrus.WithFields(logrus.Fields{
		"channel": channel.Name,
		"title":   n.Title,
	}).Debug("Slack notification (not implemented)")
	return nil
}

// sendDiscord sends notification to Discord (placeholder)
func (m *Manager) sendDiscord(channel *Channel, n *Notification) error {
	// TODO: Implement Discord webhook
	logrus.WithFields(logrus.Fields{
		"channel": channel.Name,
		"title":   n.Title,
	}).Debug("Discord notification (not implemented)")
	return nil
}

// sendWebhook sends notification to generic webhook (placeholder)
func (m *Manager) sendWebhook(channel *Channel, n *Notification) error {
	// TODO: Implement generic webhook
	logrus.WithFields(logrus.Fields{
		"channel": channel.Name,
		"title":   n.Title,
	}).Debug("Webhook notification (not implemented)")
	return nil
}

// processNotifications processes queued notifications
func (m *Manager) processNotifications() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Process any pending notifications
			// TODO: Implement retry logic for failed notifications
		case <-m.stop:
			return
		}
	}
}

// GetUnreadNotifications returns unread notifications for a user
func (m *Manager) GetUnreadNotifications(userID int, limit int) ([]*Notification, error) {
	query := `
		SELECT id, title, message, priority, type, service_id, created_at, metadata
		FROM notifications
		WHERE (user_id IS NULL OR user_id = ?)
		AND read = false
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := m.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		n := &Notification{}
		var metadataJSON string

		err := rows.Scan(
			&n.ID,
			&n.Title,
			&n.Message,
			&n.Priority,
			&n.Type,
			&n.ServiceID,
			&n.Timestamp,
			&metadataJSON,
		)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(metadataJSON), &n.Metadata)
		notifications = append(notifications, n)
	}

	return notifications, nil
}

// MarkAsRead marks a notification as read
func (m *Manager) MarkAsRead(notificationID int) error {
	_, err := m.db.Exec("UPDATE notifications SET read = true WHERE id = ?", notificationID)
	return err
}