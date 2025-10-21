-- CasDash Initial Database Schema
-- This migration creates all core tables as specified in CLAUDE.md

-- Users table
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user', -- 'primary_admin', 'admin', 'user', 'support', 'view_only'
    is_primary_admin BOOLEAN DEFAULT FALSE, -- First user, immutable
    organization_id INTEGER, -- For Enterprise mode
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    two_fa_secret VARCHAR(255),
    two_fa_enabled BOOLEAN DEFAULT FALSE,
    api_key VARCHAR(255) UNIQUE,
    api_key_created TIMESTAMP,
    session_token VARCHAR(255),
    session_expires TIMESTAMP,
    active BOOLEAN DEFAULT TRUE,
    email_verified BOOLEAN DEFAULT FALSE,
    password_reset_token VARCHAR(255),
    password_reset_expires TIMESTAMP,
    preferences TEXT, -- JSON user preferences
    metadata TEXT -- JSON additional user data
);

-- Services table
CREATE TABLE services (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    -- Basic Information
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    service_type VARCHAR(100), -- One of 2000+ types
    category VARCHAR(100),
    description TEXT,
    icon TEXT, -- URL, base64, or icon library reference

    -- Authentication
    auth_type VARCHAR(50) DEFAULT 'none', -- 'none', 'basic', 'bearer', 'api_key', 'oauth2', 'custom'
    auth_credentials TEXT, -- Encrypted JSON
    custom_headers TEXT, -- JSON

    -- Monitoring Configuration
    monitoring_enabled BOOLEAN DEFAULT TRUE,
    check_interval INTEGER DEFAULT 300, -- seconds
    timeout INTEGER DEFAULT 30, -- seconds
    expected_status_codes TEXT DEFAULT '[200]', -- JSON array
    expected_content TEXT,
    follow_redirects BOOLEAN DEFAULT TRUE,
    ssl_verify BOOLEAN DEFAULT TRUE,
    ssl_monitoring_enabled BOOLEAN, -- NULL = auto-detect
    ssl_check_interval INTEGER DEFAULT 86400,
    ssl_hostname VARCHAR(255),
    ssl_port INTEGER,

    -- Public Access
    public_visible BOOLEAN DEFAULT FALSE,
    public_name VARCHAR(255),
    public_description TEXT,

    -- Management
    maintenance_mode BOOLEAN DEFAULT FALSE,
    maintenance_until TIMESTAMP,

    -- UI Configuration
    position_x INTEGER,
    position_y INTEGER,
    card_size VARCHAR(20) DEFAULT 'medium', -- 'small', 'medium', 'large'
    card_color VARCHAR(7), -- Hex color override

    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER REFERENCES users(id),
    organization_id INTEGER, -- For Enterprise mode
    user_id INTEGER, -- For SaaS mode
    tags TEXT, -- JSON array of tags
    custom_fields TEXT, -- JSON extensible fields

    -- Dependencies
    depends_on TEXT, -- JSON array of service IDs
    dependency_type VARCHAR(50) DEFAULT 'soft', -- 'hard', 'soft', 'optional'

    -- Container/VM specific
    container_id VARCHAR(255),
    container_image VARCHAR(255),
    vm_id VARCHAR(255),
    host_server VARCHAR(255)
);

-- Settings table (single source of truth)
CREATE TABLE settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'string', 'integer', 'boolean', 'json'
    category VARCHAR(100) NOT NULL, -- 'system', 'monitoring', 'security', etc.
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by INTEGER REFERENCES users(id)
);

-- Service types definition (2000+ types)
CREATE TABLE service_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) UNIQUE NOT NULL,
    category VARCHAR(50),
    default_port INTEGER,
    default_check_type VARCHAR(50),
    health_endpoint VARCHAR(255),
    auth_type VARCHAR(50),
    icon VARCHAR(255),
    docker_image VARCHAR(255), -- Common Docker image
    documentation_url TEXT,
    configuration_template TEXT, -- JSON template
    monitoring_config TEXT -- JSON service-specific monitoring settings
);

-- Monitoring results
CREATE TABLE monitoring_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    check_type VARCHAR(50), -- 'health', 'ssl', 'port', 'performance'
    check_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    success BOOLEAN,
    response_time_ms INTEGER,
    status_code INTEGER,
    error_message TEXT,
    details TEXT -- JSON detailed check results
);

-- Create indexes for monitoring results
CREATE INDEX idx_monitoring_service_time ON monitoring_results(service_id, check_time DESC);
CREATE INDEX idx_monitoring_check_time ON monitoring_results(check_time DESC);

-- SSL Certificates
CREATE TABLE ssl_certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    hostname VARCHAR(255),
    port INTEGER,

    -- Certificate Details
    fingerprint_sha256 VARCHAR(95) UNIQUE,
    serial_number VARCHAR(100),

    -- Subject Information
    common_name VARCHAR(255),
    organization VARCHAR(255),

    -- Issuer Information
    issuer_cn VARCHAR(255),
    issuer_org VARCHAR(255),
    is_self_signed BOOLEAN,

    -- Validity
    not_before TIMESTAMP,
    not_after TIMESTAMP,

    -- Technical Details
    key_algorithm VARCHAR(50),
    key_size INTEGER,
    signature_algorithm VARCHAR(50),

    -- SAN
    san_dns_names TEXT, -- JSON array
    wildcard_cert BOOLEAN DEFAULT FALSE,

    -- Chain
    chain_valid BOOLEAN,
    chain_complete BOOLEAN,

    -- Status
    status VARCHAR(50), -- 'active', 'expiring', 'expired', 'revoked'

    -- Metadata
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    monitoring_enabled BOOLEAN DEFAULT TRUE
);

-- SSL certificate indexes
CREATE INDEX idx_ssl_service_id ON ssl_certificates(service_id);
CREATE INDEX idx_ssl_hostname_port ON ssl_certificates(hostname, port);
CREATE INDEX idx_ssl_not_after ON ssl_certificates(not_after);

-- Service dependencies
CREATE TABLE service_dependencies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    depends_on_id INTEGER REFERENCES services(id),
    dependency_type VARCHAR(50) DEFAULT 'soft', -- 'hard', 'soft', 'optional'

    -- Failure handling
    on_dependency_failure VARCHAR(50) DEFAULT 'alert_only', -- 'cascade_stop', 'alert_only', 'ignore'
    health_impact VARCHAR(20) DEFAULT 'minimal', -- 'critical', 'degraded', 'minimal'

    -- Auto-detected
    auto_detected BOOLEAN DEFAULT FALSE,
    detection_method VARCHAR(50),
    confidence INTEGER, -- 0-100

    UNIQUE(service_id, depends_on_id)
);

-- Issues table for automated issue tracking
CREATE TABLE issues (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    issue_type VARCHAR(50), -- 'downtime', 'performance', 'security', 'config'
    severity VARCHAR(20), -- 'critical', 'high', 'medium', 'low'
    title VARCHAR(255),
    description TEXT,

    -- Automation
    auto_created BOOLEAN DEFAULT FALSE,
    auto_resolved BOOLEAN DEFAULT FALSE,
    resolution_script TEXT,

    -- Status
    status VARCHAR(50) DEFAULT 'open', -- 'open', 'in_progress', 'resolved', 'closed'
    assigned_to INTEGER REFERENCES users(id),

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    closed_at TIMESTAMP,

    -- Relationships
    related_issues TEXT, -- JSON array of issue IDs
    caused_by_issue INTEGER REFERENCES issues(id),

    -- Metrics
    time_to_detect INTEGER, -- seconds
    time_to_resolve INTEGER, -- seconds
    recurrence_count INTEGER DEFAULT 0
);

-- Notification channels
CREATE TABLE notification_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255),
    channel_type VARCHAR(50), -- 'email', 'slack', 'discord', 'webhook', 'sms'
    enabled BOOLEAN DEFAULT TRUE,

    -- Configuration (encrypted)
    config TEXT, -- JSON

    -- Rate limiting
    rate_limit_count INTEGER,
    rate_limit_window INTEGER, -- seconds

    -- Testing
    last_test TIMESTAMP,
    test_successful BOOLEAN,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Notification rules
CREATE TABLE notification_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255),
    service_id INTEGER REFERENCES services(id), -- NULL for global
    channel_id INTEGER REFERENCES notification_channels(id),

    -- Triggers
    trigger_events TEXT, -- JSON array ['service_down', 'ssl_expiring', etc.]

    -- Conditions
    condition_type VARCHAR(50) DEFAULT 'immediate', -- 'immediate', 'threshold', 'pattern'
    condition_config TEXT, -- JSON

    -- Deduplication
    dedupe_window INTEGER, -- seconds
    group_window INTEGER, -- seconds for grouping

    -- Schedule
    active_hours_only BOOLEAN DEFAULT FALSE,
    active_hours_start TIME,
    active_hours_end TIME,
    active_days TEXT, -- JSON array [0,1,2,3,4,5,6] where 0=Sunday

    -- Escalation
    escalation_enabled BOOLEAN DEFAULT FALSE,
    escalation_after INTEGER, -- seconds
    escalation_channel_id INTEGER REFERENCES notification_channels(id),

    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Notification history
CREATE TABLE notification_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id INTEGER REFERENCES notification_rules(id),
    channel_id INTEGER REFERENCES notification_channels(id),
    service_id INTEGER REFERENCES services(id),

    -- Content
    title VARCHAR(255),
    message TEXT,
    priority VARCHAR(20),

    -- Status
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    delivered BOOLEAN,
    delivery_error TEXT,

    -- Grouping
    group_id VARCHAR(100),
    grouped_count INTEGER DEFAULT 1
);

-- Themes table
CREATE TABLE themes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100),
    slug VARCHAR(100) UNIQUE,

    -- Colors (JSON)
    colors TEXT,

    -- Typography (JSON)
    fonts TEXT,

    -- Components (JSON)
    components TEXT,

    -- Status
    is_default BOOLEAN DEFAULT FALSE,
    is_custom BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User preferences
CREATE TABLE user_preferences (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),

    -- Theme
    theme_id INTEGER REFERENCES themes(id),

    -- Display
    language VARCHAR(10) DEFAULT 'en-US',
    timezone VARCHAR(50),
    date_format VARCHAR(50) DEFAULT 'YYYY-MM-DD',
    time_format VARCHAR(10) DEFAULT '24h',

    -- Dashboard
    dashboard_layout VARCHAR(20) DEFAULT 'grid',
    cards_per_row INTEGER DEFAULT 4,
    card_size VARCHAR(20) DEFAULT 'medium',
    show_favicon BOOLEAN DEFAULT TRUE,

    -- Notifications
    email_notifications BOOLEAN DEFAULT TRUE,
    browser_notifications BOOLEAN DEFAULT FALSE,
    notification_sound BOOLEAN DEFAULT TRUE,

    -- Accessibility
    high_contrast BOOLEAN DEFAULT FALSE,
    reduce_motion BOOLEAN DEFAULT FALSE,
    font_size VARCHAR(20) DEFAULT 'medium',

    -- Advanced
    show_advanced_options BOOLEAN DEFAULT FALSE,
    developer_mode BOOLEAN DEFAULT FALSE,

    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- API keys
CREATE TABLE api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    key_hash VARCHAR(255) UNIQUE,
    name VARCHAR(255),

    -- Permissions
    permissions TEXT, -- JSON array ['read', 'write', 'admin']
    allowed_ips TEXT, -- JSON array

    -- Rate limiting
    rate_limit INTEGER DEFAULT 1000, -- per hour

    -- Usage
    last_used TIMESTAMP,
    usage_count INTEGER DEFAULT 0,

    -- Status
    active BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Discovery sessions
CREATE TABLE discovery_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    discovery_type VARCHAR(50), -- 'network', 'docker', 'kubernetes', etc.
    target VARCHAR(255), -- IP range, docker socket, k8s cluster
    services_found INTEGER,
    services_added INTEGER,
    status VARCHAR(50), -- 'running', 'completed', 'failed'
    error_message TEXT,
    initiated_by INTEGER REFERENCES users(id)
);

-- Discovered services awaiting confirmation
CREATE TABLE discovered_services (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER REFERENCES discovery_sessions(id),
    host VARCHAR(255),
    port INTEGER,
    service_type VARCHAR(100),
    confidence_score INTEGER, -- 0-100
    fingerprint TEXT,
    version VARCHAR(50),
    additional_info TEXT, -- JSON
    added_as_service BOOLEAN DEFAULT FALSE,
    ignored BOOLEAN DEFAULT FALSE,
    discovered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Update tracking
CREATE TABLE update_checks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    check_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    current_version VARCHAR(100),
    latest_version VARCHAR(100),
    update_available BOOLEAN,
    update_type VARCHAR(50), -- 'major', 'minor', 'patch', 'security'
    release_notes TEXT,
    download_url TEXT,
    breaking_changes BOOLEAN DEFAULT FALSE
);

-- Update policies
CREATE TABLE update_policies (
    service_id INTEGER PRIMARY KEY REFERENCES services(id),
    policy VARCHAR(50) DEFAULT 'manual', -- 'manual', 'automatic', 'scheduled', 'approval'
    schedule_cron VARCHAR(100),
    delay_hours INTEGER,
    exclude_major BOOLEAN DEFAULT FALSE,
    exclude_tags TEXT, -- JSON array ['beta', 'rc', 'alpha']
    rollback_on_failure BOOLEAN DEFAULT TRUE,
    backup_before_update BOOLEAN DEFAULT TRUE,
    test_instance_id INTEGER REFERENCES services(id),
    approval_required BOOLEAN DEFAULT FALSE,
    approved_by INTEGER REFERENCES users(id),
    max_versions_behind INTEGER
);

-- Docker labels (Watchtower compatibility)
CREATE TABLE docker_labels (
    service_id INTEGER REFERENCES services(id) ON DELETE CASCADE,
    label_key VARCHAR(255),
    label_value TEXT,
    PRIMARY KEY (service_id, label_key)
);

-- Supported Watchtower labels:
-- com.centurylinklabs.watchtower.enable
-- com.centurylinklabs.watchtower.monitor-only
-- com.centurylinklabs.watchtower.stop-signal
-- com.centurylinklabs.watchtower.pre-update-command
-- com.centurylinklabs.watchtower.post-update-command
-- com.centurylinklabs.watchtower.lifecycle.*
-- com.centurylinklabs.watchtower.scope
-- com.centurylinklabs.watchtower.depends-on
--
-- CasDash extended labels:
-- com.casdash.update.enable
-- com.casdash.update.policy
-- com.casdash.update.schedule
-- com.casdash.update.rollback
-- com.casdash.monitor.interval
-- com.casdash.public.visible

-- Footer configuration
CREATE TABLE footer_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    elements TEXT DEFAULT '[
        {"type": "execution_time", "text": "Generated in {time}ms", "order": 1},
        {"type": "separator", "text": " | ", "order": 2},
        {"type": "powered_by", "text": "Powered by CasDash", "link": "https://github.com/casapps/casdash", "order": 3},
        {"type": "separator", "text": " | ", "order": 4},
        {"type": "version", "text": "v{version}", "order": 5}
    ]',
    alignment VARCHAR(20) DEFAULT 'center',
    custom_css TEXT,
    custom_html TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by INTEGER REFERENCES users(id)
);

-- Create additional indexes for performance
CREATE INDEX idx_services_type ON services(service_type);
CREATE INDEX idx_services_category ON services(category);
CREATE INDEX idx_services_user_id ON services(user_id);
CREATE INDEX idx_services_organization_id ON services(organization_id);
CREATE INDEX idx_services_monitoring_enabled ON services(monitoring_enabled);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_organization_id ON users(organization_id);

CREATE INDEX idx_settings_category ON settings(category);

CREATE INDEX idx_issues_service_id ON issues(service_id);
CREATE INDEX idx_issues_status ON issues(status);
CREATE INDEX idx_issues_severity ON issues(severity);
CREATE INDEX idx_issues_created_at ON issues(created_at);

CREATE INDEX idx_notification_history_service_id ON notification_history(service_id);
CREATE INDEX idx_notification_history_sent_at ON notification_history(sent_at);

CREATE INDEX idx_discovery_sessions_status ON discovery_sessions(status);
CREATE INDEX idx_discovery_sessions_started_at ON discovery_sessions(started_at);

CREATE INDEX idx_discovered_services_session_id ON discovered_services(session_id);
CREATE INDEX idx_discovered_services_confidence ON discovered_services(confidence_score);

CREATE INDEX idx_update_checks_service_id ON update_checks(service_id);
CREATE INDEX idx_update_checks_time ON update_checks(check_time);

CREATE INDEX idx_docker_labels_service_id ON docker_labels(service_id);
CREATE INDEX idx_docker_labels_key ON docker_labels(label_key);