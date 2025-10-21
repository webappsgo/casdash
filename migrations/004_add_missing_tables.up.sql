-- Migration 004: Add missing tables to match SPEC exactly

-- Real-time monitoring data (simplified for SQLite, would be partitioned in PostgreSQL)
CREATE TABLE monitoring_realtime (
    service_id INTEGER,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    response_time_ms INTEGER,
    status_code INTEGER,
    success BOOLEAN,
    error_message TEXT,
    PRIMARY KEY (service_id, timestamp)
);

-- Notification templates
CREATE TABLE notification_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255),
    channel_type VARCHAR(50),
    event_type VARCHAR(50),
    title_template TEXT,
    body_template TEXT,
    format VARCHAR(20), -- 'plain', 'html', 'markdown'
    available_vars TEXT -- JSON array
);

-- Create indexes
CREATE INDEX idx_monitoring_realtime_service ON monitoring_realtime(service_id);
CREATE INDEX idx_monitoring_realtime_timestamp ON monitoring_realtime(timestamp DESC);
CREATE INDEX idx_notification_templates_channel ON notification_templates(channel_type);
CREATE INDEX idx_notification_templates_event ON notification_templates(event_type);
