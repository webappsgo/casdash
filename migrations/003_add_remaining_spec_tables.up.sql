-- Migration 003: Add remaining SPEC tables (Support, Maintenance, Billing, API, WebSocket, etc.)

-- Support System
CREATE TABLE bot_responses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trigger_pattern TEXT,
    response_template TEXT,
    category VARCHAR(50),
    requires_data BOOLEAN DEFAULT FALSE,
    data_query TEXT,
    priority INTEGER DEFAULT 0
);

CREATE TABLE chat_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    support_user_id INTEGER REFERENCES users(id),
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    status VARCHAR(50),
    rating INTEGER,
    transcript TEXT -- JSON
);

CREATE TABLE chat_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER REFERENCES chat_sessions(id),
    sender_id INTEGER REFERENCES users(id),
    message TEXT,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    read_at TIMESTAMP,
    message_type VARCHAR(20)
);

CREATE TABLE support_tickets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_number VARCHAR(50) UNIQUE,
    user_id INTEGER REFERENCES users(id),
    assigned_to INTEGER REFERENCES users(id),
    subject VARCHAR(255),
    description TEXT,
    category VARCHAR(50),
    status VARCHAR(50),
    priority VARCHAR(20),
    service_id INTEGER REFERENCES services(id),
    related_issue_id INTEGER REFERENCES issues(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    resolved_at TIMESTAMP,
    closed_at TIMESTAMP,
    sla_response_due TIMESTAMP,
    sla_resolution_due TIMESTAMP,
    sla_breached BOOLEAN DEFAULT FALSE
);

CREATE TABLE knowledge_base_articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title VARCHAR(255),
    slug VARCHAR(255) UNIQUE,
    content TEXT,
    category VARCHAR(100),
    tags TEXT, -- JSON array
    author_id INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    published BOOLEAN DEFAULT FALSE,
    view_count INTEGER DEFAULT 0,
    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0
);

-- Maintenance System
CREATE TABLE maintenance_windows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title VARCHAR(255),
    description TEXT,
    affected_services TEXT, -- JSON array
    affect_all_services BOOLEAN DEFAULT FALSE,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    timezone VARCHAR(50),
    recurring BOOLEAN DEFAULT FALSE,
    recurrence_pattern VARCHAR(100),
    recurrence_end DATE,
    advance_notice_sent BOOLEAN DEFAULT FALSE,
    reminder_sent BOOLEAN DEFAULT FALSE,
    completion_sent BOOLEAN DEFAULT FALSE,
    status VARCHAR(50),
    actual_start TIMESTAMP,
    actual_end TIMESTAMP,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE maintenance_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    window_id INTEGER REFERENCES maintenance_windows(id),
    service_id INTEGER REFERENCES services(id),
    task_type VARCHAR(50),
    description TEXT,
    script TEXT,
    estimated_duration INTEGER,
    actual_duration INTEGER,
    status VARCHAR(50),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    execution_order INTEGER,
    can_parallel BOOLEAN DEFAULT FALSE
);

CREATE TABLE maintenance_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255),
    description TEXT,
    service_types TEXT, -- JSON array
    tasks TEXT, -- JSON
    estimated_duration INTEGER,
    requires_downtime BOOLEAN DEFAULT TRUE
);

-- Billing System (SaaS Mode)
CREATE TABLE billing_plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100),
    slug VARCHAR(100) UNIQUE,
    price_monthly DECIMAL,
    price_annual DECIMAL,
    currency VARCHAR(3) DEFAULT 'USD',
    max_services INTEGER,
    max_checks_per_hour INTEGER,
    data_retention_days INTEGER,
    features TEXT, -- JSON
    active BOOLEAN DEFAULT TRUE,
    available_for_signup BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    plan_id INTEGER REFERENCES billing_plans(id),
    status VARCHAR(50),
    billing_cycle VARCHAR(20),
    started_at TIMESTAMP,
    current_period_start TIMESTAMP,
    current_period_end TIMESTAMP,
    cancelled_at TIMESTAMP,
    expires_at TIMESTAMP,
    payment_method_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),
    paypal_subscription_id VARCHAR(255),
    trial_ends_at TIMESTAMP,
    trial_used BOOLEAN DEFAULT FALSE
);

CREATE TABLE invoices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    invoice_number VARCHAR(50) UNIQUE,
    user_id INTEGER REFERENCES users(id),
    subscription_id INTEGER REFERENCES subscriptions(id),
    subtotal DECIMAL,
    tax DECIMAL,
    total DECIMAL,
    currency VARCHAR(3),
    status VARCHAR(50),
    issued_at TIMESTAMP,
    due_at TIMESTAMP,
    paid_at TIMESTAMP,
    payment_method VARCHAR(50),
    transaction_id VARCHAR(255),
    line_items TEXT, -- JSON
    tax_info TEXT, -- JSON
    pdf_url TEXT
);

CREATE TABLE usage_tracking (
    user_id INTEGER REFERENCES users(id),
    metric_type VARCHAR(50),
    metric_value INTEGER,
    period_start DATE,
    period_end DATE,
    PRIMARY KEY (user_id, metric_type, period_start)
);

CREATE TABLE payment_methods (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    type VARCHAR(50),
    last4 VARCHAR(4),
    brand VARCHAR(20),
    exp_month INTEGER,
    exp_year INTEGER,
    stripe_payment_method_id VARCHAR(255),
    paypal_billing_agreement_id VARCHAR(255),
    is_default BOOLEAN DEFAULT FALSE,
    verified BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- API Management
CREATE TABLE api_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key_id INTEGER REFERENCES api_keys(id),
    method VARCHAR(10),
    path TEXT,
    query_params TEXT,
    body_size INTEGER,
    status_code INTEGER,
    response_size INTEGER,
    response_time_ms INTEGER,
    ip_address VARCHAR(45),
    user_agent TEXT,
    requested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE webhooks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    name VARCHAR(255),
    url TEXT,
    events TEXT, -- JSON array
    auth_type VARCHAR(50),
    auth_config TEXT, -- JSON
    active BOOLEAN DEFAULT TRUE,
    verify_ssl BOOLEAN DEFAULT TRUE,
    retry_count INTEGER DEFAULT 3,
    retry_delay INTEGER DEFAULT 5,
    last_triggered TIMESTAMP,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0
);

CREATE TABLE webhook_deliveries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    webhook_id INTEGER REFERENCES webhooks(id),
    event VARCHAR(100),
    request_headers TEXT, -- JSON
    request_body TEXT,
    response_status INTEGER,
    response_headers TEXT, -- JSON
    response_body TEXT,
    response_time_ms INTEGER,
    success BOOLEAN,
    retry_count INTEGER DEFAULT 0,
    delivered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- WebSocket Management
CREATE TABLE websocket_connections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    connection_id VARCHAR(100) UNIQUE,
    user_id INTEGER REFERENCES users(id),
    ip_address VARCHAR(45),
    user_agent TEXT,
    subscribed_channels TEXT, -- JSON array
    subscribed_services TEXT, -- JSON array
    connected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_ping TIMESTAMP,
    disconnected_at TIMESTAMP
);

CREATE TABLE websocket_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    connection_id VARCHAR(100),
    channel VARCHAR(50),
    message_type VARCHAR(50),
    payload TEXT, -- JSON
    queued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    delivered BOOLEAN DEFAULT FALSE,
    delivered_at TIMESTAMP
);

-- User Management (uses session_token in users table per SPEC)

-- Dashboard Layouts
CREATE TABLE dashboard_layouts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    name VARCHAR(255),
    layout_data TEXT, -- JSON
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_protocol_tests_service_id ON protocol_tests(service_id);
CREATE INDEX idx_protocol_test_results_test_id ON protocol_test_results(test_id);
CREATE INDEX idx_ssl_extended_service_id ON ssl_certificates_extended(service_id);
CREATE INDEX idx_ssl_extended_fingerprint ON ssl_certificates_extended(fingerprint_sha256);
CREATE INDEX idx_ssl_security_cert_id ON ssl_security_assessment(certificate_id);
CREATE INDEX idx_update_history_service_id ON update_history(service_id);
CREATE INDEX idx_virtual_resources_platform ON virtual_resources(platform_id);
CREATE INDEX idx_platform_metrics_platform ON platform_metrics(platform_id);
CREATE INDEX idx_media_services_service_id ON media_services(service_id);
CREATE INDEX idx_arr_services_service_id ON arr_services(service_id);
CREATE INDEX idx_download_queue_service_id ON download_queue(service_id);
CREATE INDEX idx_security_assessments_service_id ON security_assessments(service_id);
CREATE INDEX idx_security_recommendations_service_id ON security_recommendations(service_id);
CREATE INDEX idx_vulnerabilities_service_id ON vulnerabilities(service_id);
CREATE INDEX idx_vulnerabilities_cve ON vulnerabilities(cve_id);
CREATE INDEX idx_chat_sessions_user ON chat_sessions(user_id);
CREATE INDEX idx_chat_messages_session ON chat_messages(session_id);
CREATE INDEX idx_support_tickets_user ON support_tickets(user_id);
CREATE INDEX idx_support_tickets_number ON support_tickets(ticket_number);
CREATE INDEX idx_maintenance_windows_start ON maintenance_windows(start_time);
CREATE INDEX idx_maintenance_tasks_window ON maintenance_tasks(window_id);
CREATE INDEX idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX idx_invoices_user ON invoices(user_id);
CREATE INDEX idx_api_requests_key ON api_requests(api_key_id);
CREATE INDEX idx_api_requests_time ON api_requests(requested_at);
CREATE INDEX idx_webhooks_user ON webhooks(user_id);
CREATE INDEX idx_webhook_deliveries_webhook ON webhook_deliveries(webhook_id);
CREATE INDEX idx_websocket_connections_user ON websocket_connections(user_id);
CREATE INDEX idx_dashboard_layouts_user ON dashboard_layouts(user_id);

-- Populate default billing plans (for SaaS mode)
INSERT INTO billing_plans (name, slug, price_monthly, price_annual, max_services, max_checks_per_hour, data_retention_days, features) VALUES
('Free', 'free', 0, 0, 25, 250, 15, '{"ssl_monitoring": true, "basic_alerts": true}'),
('Basic', 'basic', 5, 50, 50, 500, 30, '{"ssl_monitoring": true, "advanced_alerts": true, "api_access": true}'),
('Pro', 'pro', 10, 100, 150, 1500, 90, '{"ssl_monitoring": true, "advanced_alerts": true, "api_access": true, "custom_domains": true}'),
('Enterprise', 'enterprise', 20, 200, 500, 5000, 365, '{"ssl_monitoring": true, "advanced_alerts": true, "api_access": true, "custom_domains": true, "priority_support": true}');
