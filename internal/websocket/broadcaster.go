package websocket

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
)

// StatusUpdate represents a service status update
type StatusUpdate struct {
	ServiceID    int       `json:"service_id"`
	ServiceName  string    `json:"service_name"`
	Status       string    `json:"status"`
	ResponseTime int       `json:"response_time_ms"`
	StatusCode   int       `json:"status_code,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// MonitoringUpdate represents a monitoring metrics update
type MonitoringUpdate struct {
	ServiceID     int                    `json:"service_id"`
	Metrics       map[string]interface{} `json:"metrics"`
	Timestamp     time.Time              `json:"timestamp"`
}

// NotificationUpdate represents a notification
type NotificationUpdate struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Priority  string    `json:"priority"`
	ServiceID int       `json:"service_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// LogUpdate represents a log message
type LogUpdate struct {
	ServiceID int       `json:"service_id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// BroadcastStatusUpdate broadcasts a service status update
func (h *Hub) BroadcastStatusUpdate(update StatusUpdate) {
	msg := Message{
		Type:    "status_update",
		Channel: "status",
		Payload: update,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal status update")
		return
	}

	h.BroadcastToChannel("status", data)
}

// BroadcastMonitoringUpdate broadcasts monitoring metrics
func (h *Hub) BroadcastMonitoringUpdate(update MonitoringUpdate) {
	msg := Message{
		Type:    "monitoring_update",
		Channel: "monitoring",
		Payload: update,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal monitoring update")
		return
	}

	h.BroadcastToChannel("monitoring", data)
}

// BroadcastNotification broadcasts a notification
func (h *Hub) BroadcastNotification(notification NotificationUpdate) {
	msg := Message{
		Type:    "notification",
		Channel: "notifications",
		Payload: notification,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal notification")
		return
	}

	h.BroadcastToChannel("notifications", data)
}

// BroadcastLog broadcasts a log message
func (h *Hub) BroadcastLog(log LogUpdate) {
	msg := Message{
		Type:    "log",
		Channel: "logs",
		Payload: log,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal log")
		return
	}

	h.BroadcastToChannel("logs", data)
}

// BroadcastServiceUpdate broadcasts a general service update
func (h *Hub) BroadcastServiceUpdate(serviceID int, updateType string, payload interface{}) {
	msg := Message{
		Type:    updateType,
		Channel: "status",
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal service update")
		return
	}

	h.BroadcastToChannel("status", data)
}