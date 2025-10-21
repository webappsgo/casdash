-- Migration 002: Add remaining SPEC tables
-- This migration adds all tables specified in CLAUDE.md SPEC

-- Protocol & Port Monitoring
CREATE TABLE protocol_monitors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100),
    protocol VARCHAR(10), -- 'tcp', 'udp', 'both'
    default_port INTEGER,
    port_alternatives TEXT, -- JSON array
    test_method VARCHAR(50), -- 'banner', 'handshake', 'payload', 'connect'
    expected_response TEXT,
    timeout_ms INTEGER DEFAULT 5000,
    category VARCHAR(50)
);

CREATE TABLE protocol_tests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    protocol_id INTEGER REFERENCES protocol_monitors(id),
    host VARCHAR(255),
    port INTEGER,
    protocol VARCHAR(10),
    test_interval INTEGER DEFAULT 300,
    timeout_ms INTEGER DEFAULT 5000,
    enabled BOOLEAN DEFAULT TRUE,
    last_test TIMESTAMP,
    last_result VARCHAR(50),
    response_time_ms INTEGER,
    error_message TEXT
);

CREATE TABLE protocol_test_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    test_id INTEGER REFERENCES protocol_tests(id),
    tested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    success BOOLEAN,
    response_time_ms INTEGER,
    banner_received TEXT,
    ssl_info TEXT, -- JSON
    protocol_version VARCHAR(50),
    additional_data TEXT -- JSON
);

CREATE TABLE udp_monitoring_config (
    protocol_id INTEGER REFERENCES protocol_monitors(id),
    send_payload TEXT,
    expect_response BOOLEAN DEFAULT FALSE,
    response_timeout_ms INTEGER DEFAULT 2000,
    retry_count INTEGER DEFAULT 3,
    stateful_check BOOLEAN DEFAULT FALSE
);

-- SSL/TLS Extended Tables
CREATE TABLE ssl_certificates_extended (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    hostname VARCHAR(255),
    port INTEGER,
    protocol VARCHAR(50),
    discovery_method VARCHAR(50),
    fingerprint_sha256 VARCHAR(95) UNIQUE,
    fingerprint_sha1 VARCHAR(59),
    serial_number VARCHAR(100),
    version INTEGER,
    subject_dn TEXT,
    common_name VARCHAR(255),
    organization VARCHAR(255),
    organizational_unit VARCHAR(255),
    country VARCHAR(2),
    state_province VARCHAR(100),
    locality VARCHAR(100),
    email VARCHAR(255),
    issuer_dn TEXT,
    issuer_cn VARCHAR(255),
    issuer_org VARCHAR(255),
    ca_type VARCHAR(50),
    is_self_signed BOOLEAN,
    not_before TIMESTAMP,
    not_after TIMESTAMP,
    validity_days INTEGER,
    key_algorithm VARCHAR(50),
    key_size INTEGER,
    key_usage TEXT, -- JSON array
    extended_key_usage TEXT, -- JSON array
    san_dns_names TEXT, -- JSON array
    san_ip_addresses TEXT, -- JSON array
    san_emails TEXT, -- JSON array
    wildcard_cert BOOLEAN DEFAULT FALSE,
    chain_length INTEGER,
    root_ca VARCHAR(255),
    intermediate_cas TEXT, -- JSON array
    chain_complete BOOLEAN,
    chain_valid BOOLEAN,
    chain_issues TEXT, -- JSON array
    ct_logged BOOLEAN,
    ct_log_ids TEXT, -- JSON array
    sct_count INTEGER,
    caa_compliant BOOLEAN,
    caa_records TEXT, -- JSON array
    ev_certificate BOOLEAN,
    monitoring_enabled BOOLEAN DEFAULT TRUE,
    check_interval INTEGER DEFAULT 86400,
    auto_renew_enabled BOOLEAN DEFAULT FALSE,
    renewal_method VARCHAR(50),
    status VARCHAR(50),
    revocation_status VARCHAR(50),
    revocation_reason VARCHAR(100),
    revoked_at TIMESTAMP,
    cost DECIMAL,
    vendor VARCHAR(100),
    purchase_order VARCHAR(100),
    notes TEXT,
    tags TEXT, -- JSON array
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ssl_security_assessment (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    certificate_id INTEGER REFERENCES ssl_certificates_extended(id),
    assessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    overall_grade VARCHAR(2),
    score INTEGER,
    protocols_supported TEXT, -- JSON array
    protocols_insecure TEXT, -- JSON array
    best_protocol VARCHAR(20),
    cipher_suites_strong TEXT, -- JSON array
    cipher_suites_weak TEXT, -- JSON array
    cipher_suites_insecure TEXT, -- JSON array
    forward_secrecy BOOLEAN,
    cipher_order_ok BOOLEAN,
    vulnerable_to TEXT, -- JSON array
    hsts_enabled BOOLEAN,
    hsts_max_age INTEGER,
    hpkp_enabled BOOLEAN,
    ocsp_stapling_enabled BOOLEAN,
    session_resumption BOOLEAN,
    secure_renegotiation BOOLEAN,
    pci_compliant BOOLEAN,
    hipaa_compliant BOOLEAN,
    fips_compliant BOOLEAN,
    nist_compliant BOOLEAN,
    issues TEXT, -- JSON
    warnings TEXT, -- JSON
    recommendations TEXT -- JSON
);

CREATE TABLE ssl_ocsp_status (
    certificate_id INTEGER PRIMARY KEY REFERENCES ssl_certificates_extended(id),
    ocsp_url TEXT,
    status VARCHAR(50),
    reason VARCHAR(100),
    revoked_at TIMESTAMP,
    this_update TIMESTAMP,
    next_update TIMESTAMP,
    response_time_ms INTEGER,
    stapling_enabled BOOLEAN,
    must_staple BOOLEAN,
    last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ssl_certificate_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    hostname VARCHAR(255),
    old_fingerprint VARCHAR(95),
    new_fingerprint VARCHAR(95),
    change_type VARCHAR(50),
    change_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    days_before_expiry INTEGER,
    initiated_by VARCHAR(50),
    notes TEXT
);

CREATE TABLE service_ssl_config (
    service_id INTEGER PRIMARY KEY REFERENCES services(id),
    auto_detected BOOLEAN DEFAULT TRUE,
    monitoring_enabled BOOLEAN DEFAULT TRUE,
    check_certificate BOOLEAN DEFAULT TRUE,
    check_chain BOOLEAN DEFAULT TRUE,
    check_protocols BOOLEAN DEFAULT TRUE,
    check_ciphers BOOLEAN DEFAULT TRUE,
    check_vulnerabilities BOOLEAN DEFAULT TRUE,
    check_compliance BOOLEAN DEFAULT FALSE,
    check_ct_logs BOOLEAN DEFAULT FALSE,
    expiry_warning_days INTEGER DEFAULT 30,
    expiry_critical_days INTEGER DEFAULT 7,
    min_key_size INTEGER DEFAULT 2048,
    min_protocol VARCHAR(10) DEFAULT 'TLSv1.2',
    required_grade VARCHAR(2) DEFAULT 'B'
);

-- ACME/Let's Encrypt
CREATE TABLE acme_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider VARCHAR(50),
    email VARCHAR(255),
    account_key TEXT,
    account_url TEXT,
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    rate_limit_remaining INTEGER,
    rate_limit_reset TIMESTAMP
);

CREATE TABLE acme_certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    certificate_id INTEGER REFERENCES ssl_certificates_extended(id),
    acme_account_id INTEGER REFERENCES acme_accounts(id),
    order_url TEXT,
    domains TEXT, -- JSON array
    challenge_type VARCHAR(50),
    status VARCHAR(50),
    auto_renewal BOOLEAN DEFAULT TRUE,
    renewal_days_before INTEGER DEFAULT 30
);

CREATE TABLE acme_dns_providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(50),
    provider_type VARCHAR(50),
    api_credentials TEXT,
    enabled BOOLEAN DEFAULT TRUE
);

-- Update Management
CREATE TABLE update_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    from_version VARCHAR(100),
    to_version VARCHAR(100),
    update_type VARCHAR(50),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    status VARCHAR(50),
    initiated_by VARCHAR(50),
    error_message TEXT,
    rollback_performed BOOLEAN DEFAULT FALSE
);

CREATE TABLE container_registries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    registry_url VARCHAR(255),
    registry_type VARCHAR(50),
    auth_type VARCHAR(50),
    credentials TEXT,
    scan_interval INTEGER DEFAULT 3600
);

-- Virtualization & Container Platforms
CREATE TABLE platform_credentials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    platform_type VARCHAR(50),
    name VARCHAR(255),
    endpoint VARCHAR(255),
    username VARCHAR(255),
    password_encrypted TEXT,
    token_encrypted TEXT,
    certificate TEXT,
    options TEXT, -- JSON
    last_connected TIMESTAMP,
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE virtual_resources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    platform_id INTEGER REFERENCES platform_credentials(id),
    resource_type VARCHAR(50),
    resource_id VARCHAR(255),
    name VARCHAR(255),
    state VARCHAR(50),
    cpu_count INTEGER,
    memory_mb INTEGER,
    disk_gb INTEGER,
    ip_addresses TEXT, -- JSON array
    image VARCHAR(255),
    created_at TIMESTAMP,
    uptime_seconds BIGINT
);

CREATE TABLE platform_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    platform_id INTEGER REFERENCES platform_credentials(id),
    metric_type VARCHAR(50),
    metric_name VARCHAR(100),
    metric_value DECIMAL,
    unit VARCHAR(20),
    collected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Media & Automation Services
CREATE TABLE media_services (
    service_id INTEGER PRIMARY KEY REFERENCES services(id),
    media_type VARCHAR(50),
    total_items INTEGER,
    active_streams INTEGER,
    transcoding_sessions INTEGER,
    bandwidth_usage_mbps DECIMAL,
    storage_used_gb DECIMAL,
    storage_available_gb DECIMAL,
    library_scanning BOOLEAN DEFAULT FALSE,
    last_scan TIMESTAMP,
    version VARCHAR(50),
    update_available BOOLEAN
);

CREATE TABLE arr_services (
    service_id INTEGER PRIMARY KEY REFERENCES services(id),
    arr_type VARCHAR(50),
    queue_size INTEGER,
    queue_warnings INTEGER,
    queue_errors INTEGER,
    missing_items INTEGER,
    monitored_items INTEGER,
    disk_space_free_gb DECIMAL,
    indexer_status TEXT, -- JSON
    download_client_status TEXT, -- JSON
    recent_grabs INTEGER,
    recent_failures INTEGER
);

CREATE TABLE media_stacks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255),
    stack_type VARCHAR(50),
    primary_service_id INTEGER REFERENCES services(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE media_stack_members (
    stack_id INTEGER REFERENCES media_stacks(id),
    service_id INTEGER REFERENCES services(id),
    role VARCHAR(50),
    PRIMARY KEY (stack_id, service_id)
);

CREATE TABLE download_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    item_id VARCHAR(255),
    title VARCHAR(255),
    status VARCHAR(50),
    progress_percent DECIMAL,
    size_mb DECIMAL,
    download_speed_mbps DECIMAL,
    eta_seconds INTEGER,
    added_at TIMESTAMP,
    completed_at TIMESTAMP
);

-- Security
CREATE TABLE security_assessments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    assessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    overall_score INTEGER,
    ssl_score INTEGER,
    auth_score INTEGER,
    headers_score INTEGER,
    ports_score INTEGER,
    updates_score INTEGER,
    config_score INTEGER,
    critical_issues INTEGER DEFAULT 0,
    high_issues INTEGER DEFAULT 0,
    medium_issues INTEGER DEFAULT 0,
    low_issues INTEGER DEFAULT 0,
    vulnerabilities TEXT, -- JSON
    recommendations TEXT, -- JSON
    compliance_status TEXT -- JSON
);

CREATE TABLE security_recommendations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    category VARCHAR(50),
    severity VARCHAR(20),
    title VARCHAR(255),
    description TEXT,
    solution TEXT,
    commands TEXT, -- JSON array
    config_files TEXT, -- JSON
    estimated_time INTEGER,
    auto_fixable BOOLEAN DEFAULT FALSE,
    applied BOOLEAN DEFAULT FALSE,
    ignored BOOLEAN DEFAULT FALSE
);

CREATE TABLE compliance_requirements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    standard VARCHAR(50),
    requirement VARCHAR(255),
    status VARCHAR(50),
    last_audit DATE,
    next_audit DATE,
    evidence_url TEXT,
    notes TEXT
);

CREATE TABLE vulnerabilities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER REFERENCES services(id),
    cve_id VARCHAR(50),
    severity VARCHAR(20),
    cvss_score DECIMAL,
    description TEXT,
    affected_component VARCHAR(255),
    fixed_version VARCHAR(50),
    patch_available BOOLEAN,
    discovered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    ignored BOOLEAN DEFAULT FALSE
);

-- Advanced Monitoring
CREATE TABLE health_check_config (
    service_id INTEGER PRIMARY KEY REFERENCES services(id),
    check_type VARCHAR(50),
    check_interval INTEGER DEFAULT 300,
    timeout INTEGER DEFAULT 30,
    retry_count INTEGER DEFAULT 2,
    retry_delay INTEGER DEFAULT 10,
    http_method VARCHAR(10) DEFAULT 'GET',
    http_body TEXT,
    expected_status_codes TEXT, -- JSON array
    expected_content TEXT,
    follow_redirects BOOLEAN DEFAULT TRUE,
    send_data TEXT,
    expect_data TEXT,
    depends_on TEXT, -- JSON array of service IDs
    cascade_failure BOOLEAN DEFAULT FALSE
);

CREATE TABLE performance_baselines (
    service_id INTEGER PRIMARY KEY REFERENCES services(id),
    metric VARCHAR(50),
    baseline_value DECIMAL,
    threshold_warning DECIMAL,
    threshold_critical DECIMAL,
    calculated_at TIMESTAMP,
    sample_count INTEGER
);

CREATE TABLE monitoring_aggregated (
    service_id INTEGER,
    period_start TIMESTAMP,
    period_type VARCHAR(20),
    checks_total INTEGER,
    checks_success INTEGER,
    uptime_percent DECIMAL,
    avg_response_time_ms INTEGER,
    min_response_time_ms INTEGER,
    max_response_time_ms INTEGER,
    p95_response_time_ms INTEGER,
    p99_response_time_ms INTEGER,
    PRIMARY KEY (service_id, period_start, period_type)
);

-- Issues & Automation
CREATE TABLE issue_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_type VARCHAR(100),
    issue_type VARCHAR(50),
    title_template TEXT,
    description_template TEXT,
    auto_assign_role VARCHAR(50),
    priority VARCHAR(20),
    tags TEXT -- JSON array
);

CREATE TABLE issue_automation_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255),
    trigger_condition TEXT,
    action_type VARCHAR(50),
    action_config TEXT, -- JSON
    enabled BOOLEAN DEFAULT TRUE,
    last_triggered TIMESTAMP
);

CREATE TABLE dependency_cascades (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    root_service_id INTEGER REFERENCES services(id),
    cascade_started TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    affected_services TEXT, -- JSON array
    cascade_stopped TIMESTAMP,
    manual_intervention BOOLEAN DEFAULT FALSE
);


-- Populate protocol_monitors with common protocols from SPEC
INSERT INTO protocol_monitors (name, protocol, default_port, test_method, expected_response, category) VALUES
-- Email Protocols
('smtp', 'tcp', 25, 'banner', '220*SMTP*', 'email'),
('smtp_tls', 'tcp', 587, 'banner', '220*SMTP*', 'email'),
('smtps', 'tcp', 465, 'banner', NULL, 'email'),
('pop3', 'tcp', 110, 'banner', '+OK*', 'email'),
('pop3s', 'tcp', 995, 'banner', NULL, 'email'),
('imap', 'tcp', 143, 'banner', '* OK*', 'email'),
('imaps', 'tcp', 993, 'banner', NULL, 'email'),
-- File Transfer
('ftp', 'tcp', 21, 'banner', '220*', 'transfer'),
('ftps', 'tcp', 990, 'banner', NULL, 'transfer'),
('sftp', 'tcp', 22, 'banner', 'SSH-*', 'transfer'),
('tftp', 'udp', 69, 'payload', NULL, 'transfer'),
('rsync', 'tcp', 873, 'banner', '@RSYNCD*', 'transfer'),
-- Remote Access
('ssh', 'tcp', 22, 'banner', 'SSH-*', 'remote'),
('telnet', 'tcp', 23, 'connect', NULL, 'remote'),
('rdp', 'tcp', 3389, 'handshake', NULL, 'remote'),
('vnc', 'tcp', 5900, 'handshake', 'RFB*', 'remote'),
-- DNS & Directory
('dns', 'udp', 53, 'payload', NULL, 'network'),
('dns_tcp', 'tcp', 53, 'payload', NULL, 'network'),
('ldap', 'tcp', 389, 'handshake', NULL, 'directory'),
('ldaps', 'tcp', 636, 'handshake', NULL, 'directory'),
-- VPN Protocols
('openvpn', 'udp', 1194, 'payload', NULL, 'vpn'),
('wireguard', 'udp', 51820, 'payload', NULL, 'vpn'),
('ipsec_ike', 'udp', 500, 'handshake', NULL, 'vpn'),
('l2tp', 'udp', 1701, 'handshake', NULL, 'vpn'),
-- Messaging
('mqtt', 'tcp', 1883, 'handshake', NULL, 'messaging'),
('amqp', 'tcp', 5672, 'handshake', 'AMQP*', 'messaging'),
('xmpp', 'tcp', 5222, 'handshake', '<?xml*', 'messaging');
