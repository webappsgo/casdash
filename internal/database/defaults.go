package database

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// InitializeSettings populates the settings table with default values
func (db *DB) InitializeSettings() error {
	logrus.Info("Initializing default settings")

	settings := []struct {
		Key         string
		Value       string
		Type        string
		Category    string
		Description string
	}{
		// System
		{"mode", "enterprise", "string", "system", "Operating mode: enterprise or saas"},
		{"port", "64321", "integer", "system", "Server port"},
		{"multiuser", "false", "boolean", "system", "Enable multi-user mode"},
		{"registration", "disabled", "string", "system", "User registration policy"},
		{"session_timeout", "86400", "integer", "system", "Session timeout in seconds"},

		// Discovery
		{"discovery_enabled", "true", "boolean", "discovery", "Enable service discovery"},
		{"discovery_interval", "86400", "integer", "discovery", "Discovery interval in seconds"},
		{"discovery_networks", `["10.0.0.0/8","172.16.0.0/12","192.168.0.0/16"]`, "json", "discovery", "Networks to scan"},
		{"discovery_ports", `[22,53,80,443,3000,3306,5432,6379,8080,8443,9000]`, "json", "discovery", "Ports to scan"},
		{"discovery_timeout", "2", "integer", "discovery", "Discovery timeout in seconds"},
		{"discovery_confidence_threshold", "70", "integer", "discovery", "Minimum confidence for auto-adding services"},

		// Monitoring
		{"check_interval", "300", "integer", "monitoring", "Default check interval in seconds"},
		{"check_timeout", "30", "integer", "monitoring", "Default check timeout in seconds"},
		{"check_retries", "2", "integer", "monitoring", "Default number of retries"},
		{"expected_status_codes", `[200,201,202,204]`, "json", "monitoring", "Default expected HTTP status codes"},
		{"ssl_expiry_warning", "30", "integer", "monitoring", "SSL expiry warning threshold in days"},
		{"response_time_warning", "1000", "integer", "monitoring", "Response time warning threshold in ms"},
		{"response_time_critical", "5000", "integer", "monitoring", "Response time critical threshold in ms"},

		// Updates (Watchtower)
		{"update_enabled", "true", "boolean", "update", "Enable update management"},
		{"update_policy", "manual", "string", "update", "Default update policy"},
		{"update_check_interval", "21600", "integer", "update", "Update check interval in seconds"},
		{"update_backup_before", "true", "boolean", "update", "Backup before updates"},
		{"update_rollback_on_failure", "true", "boolean", "update", "Rollback on update failure"},
		{"update_exclude_tags", `["alpha","beta","rc","dev"]`, "json", "update", "Tags to exclude from updates"},
		{"update_respect_watchtower_labels", "true", "boolean", "update", "Respect Watchtower labels"},

		// Security
		{"security_scan_enabled", "true", "boolean", "security", "Enable security scanning"},
		{"security_scan_interval", "604800", "integer", "security", "Security scan interval in seconds"},
		{"password_min_length", "12", "integer", "security", "Minimum password length"},
		{"password_bcrypt_cost", "12", "integer", "security", "BCrypt cost factor"},
		{"2fa_enabled", "false", "boolean", "security", "Enable two-factor authentication"},

		// Notifications
		{"notifications_enabled", "true", "boolean", "notifications", "Enable notifications"},
		{"notification_delay", "300", "integer", "notifications", "Notification delay in seconds"},
		{"notification_deduplication", "3600", "integer", "notifications", "Deduplication window in seconds"},
		{"notification_grouping", "true", "boolean", "notifications", "Enable notification grouping"},

		// Data Retention
		{"retention_monitoring_data", "7776000", "integer", "retention", "Monitoring data retention in seconds"},
		{"retention_logs", "2592000", "integer", "retention", "Log retention in seconds"},
		{"retention_audit", "31536000", "integer", "retention", "Audit log retention in seconds"},
		{"retention_issues", "0", "integer", "retention", "Issue retention in seconds (0 = forever)"},

		// Performance
		{"cache_enabled", "true", "boolean", "performance", "Enable caching"},
		{"cache_ttl", "300", "integer", "performance", "Cache TTL in seconds"},
		{"compression_enabled", "true", "boolean", "performance", "Enable compression"},
		{"concurrent_checks", "10", "integer", "performance", "Number of concurrent checks"},

		// UI/UX
		{"ui_theme", "dracula", "string", "ui", "Default UI theme"},
		{"ui_auto_refresh", "true", "boolean", "ui", "Enable auto-refresh"},
		{"ui_auto_refresh_interval", "30", "integer", "ui", "Auto-refresh interval in seconds"},
		{"dashboard_card_size", "medium", "string", "ui", "Default card size"},

		// API
		{"api_enabled", "true", "boolean", "api", "Enable API"},
		{"api_rate_limit", "1000", "integer", "api", "API rate limit per hour"},
		{"api_anonymous_rate_limit", "100", "integer", "api", "Anonymous API rate limit per hour"},
	}

	query := `INSERT INTO settings (key, value, type, category, description)
			  VALUES (?, ?, ?, ?, ?)`

	for _, setting := range settings {
		_, err := db.Exec(query, setting.Key, setting.Value, setting.Type, setting.Category, setting.Description)
		if err != nil {
			return fmt.Errorf("failed to insert setting %s: %w", setting.Key, err)
		}
	}

	// Mark as initialized
	if err := db.MarkInitialized(); err != nil {
		return fmt.Errorf("failed to mark as initialized: %w", err)
	}

	logrus.Info("Default settings initialized successfully")
	return nil
}

// InitializeServiceTypes populates the service_types table with predefined service types
func (db *DB) InitializeServiceTypes() error {
	logrus.Info("Initializing service types")

	serviceTypes := []struct {
		Name                string
		Category            string
		DefaultPort         *int
		DefaultCheckType    string
		HealthEndpoint      string
		AuthType            string
		Icon                string
		DockerImage         string
		DocumentationURL    string
		ConfigurationTemplate string
	}{
		// Web Services & Proxies
		{"nginx", "web", intPtr(80), "http", "/", "none", "nginx", "nginx:latest", "https://nginx.org/", "{}"},
		{"apache", "web", intPtr(80), "http", "/", "none", "apache", "httpd:latest", "https://httpd.apache.org/", "{}"},
		{"caddy", "web", intPtr(80), "http", "/", "none", "caddy", "caddy:latest", "https://caddyserver.com/", "{}"},
		{"traefik", "web", intPtr(8080), "http", "/dashboard/", "none", "traefik", "traefik:latest", "https://traefik.io/", "{}"},
		{"haproxy", "web", intPtr(80), "tcp", "", "none", "haproxy", "haproxy:latest", "https://www.haproxy.org/", "{}"},
		{"nginx_proxy_manager", "web", intPtr(81), "http", "/", "basic", "nginx", "jc21/nginx-proxy-manager:latest", "https://nginxproxymanager.com/", "{}"},

		// Databases - Relational
		{"postgresql", "database", intPtr(5432), "tcp", "", "basic", "postgresql", "postgres:latest", "https://postgresql.org/", "{}"},
		{"mysql", "database", intPtr(3306), "tcp", "", "basic", "mysql", "mysql:latest", "https://mysql.com/", "{}"},
		{"mariadb", "database", intPtr(3306), "tcp", "", "basic", "mariadb", "mariadb:latest", "https://mariadb.org/", "{}"},
		{"sqlite", "database", nil, "file", "", "none", "sqlite", "", "https://sqlite.org/", "{}"},

		// Databases - NoSQL
		{"mongodb", "database", intPtr(27017), "tcp", "", "basic", "mongodb", "mongo:latest", "https://mongodb.com/", "{}"},
		{"redis", "database", intPtr(6379), "tcp", "", "basic", "redis", "redis:latest", "https://redis.io/", "{}"},
		{"elasticsearch", "database", intPtr(9200), "http", "/_cluster/health", "basic", "elasticsearch", "elasticsearch:latest", "https://elastic.co/", "{}"},

		// Container Platforms
		{"docker", "container", intPtr(2375), "tcp", "", "none", "docker", "", "https://docker.com/", "{}"},
		{"kubernetes", "container", intPtr(6443), "tcp", "", "bearer", "kubernetes", "", "https://kubernetes.io/", "{}"},
		{"portainer", "container", intPtr(9000), "http", "/api/status", "basic", "portainer", "portainer/portainer-ce:latest", "https://portainer.io/", "{}"},

		// Media Servers
		{"plex", "media", intPtr(32400), "http", "/web", "none", "plex", "plexinc/pms-docker:latest", "https://plex.tv/", "{}"},
		{"jellyfin", "media", intPtr(8096), "http", "/web", "none", "jellyfin", "jellyfin/jellyfin:latest", "https://jellyfin.org/", "{}"},
		{"emby", "media", intPtr(8096), "http", "/web", "basic", "emby", "emby/embyserver:latest", "https://emby.media/", "{}"},

		// Automation (*arr Suite)
		{"sonarr", "automation", intPtr(8989), "http", "/api/v3/system/status", "api_key", "sonarr", "linuxserver/sonarr:latest", "https://sonarr.tv/", "{}"},
		{"radarr", "automation", intPtr(7878), "http", "/api/v3/system/status", "api_key", "radarr", "linuxserver/radarr:latest", "https://radarr.video/", "{}"},
		{"lidarr", "automation", intPtr(8686), "http", "/api/v1/system/status", "api_key", "lidarr", "linuxserver/lidarr:latest", "https://lidarr.audio/", "{}"},
		{"prowlarr", "automation", intPtr(9696), "http", "/api/v1/system/status", "api_key", "prowlarr", "linuxserver/prowlarr:latest", "https://prowlarr.com/", "{}"},

		// Download Clients
		{"qbittorrent", "download", intPtr(8080), "http", "/api/v2/app/version", "basic", "qbittorrent", "linuxserver/qbittorrent:latest", "https://qbittorrent.org/", "{}"},
		{"transmission", "download", intPtr(9091), "http", "/transmission/rpc", "basic", "transmission", "linuxserver/transmission:latest", "https://transmissionbt.com/", "{}"},
		{"sabnzbd", "download", intPtr(8080), "http", "/api", "api_key", "sabnzbd", "linuxserver/sabnzbd:latest", "https://sabnzbd.org/", "{}"},

		// Network & Security
		{"pihole", "network", intPtr(80), "http", "/admin", "none", "pihole", "pihole/pihole:latest", "https://pi-hole.net/", "{}"},
		{"adguard_home", "network", intPtr(3000), "http", "/", "basic", "adguard", "adguard/adguardhome:latest", "https://adguard.com/", "{}"},
		{"unifi_controller", "network", intPtr(8443), "https", "/manage", "basic", "unifi", "linuxserver/unifi-controller:latest", "https://unifi-network.ui.com/", "{}"},

		// Monitoring & Observability
		{"prometheus", "monitoring", intPtr(9090), "http", "/-/healthy", "none", "prometheus", "prom/prometheus:latest", "https://prometheus.io/", "{}"},
		{"grafana", "monitoring", intPtr(3000), "http", "/api/health", "basic", "grafana", "grafana/grafana:latest", "https://grafana.com/", "{}"},
		{"uptime_kuma", "monitoring", intPtr(3001), "http", "/", "basic", "uptime-kuma", "louislam/uptime-kuma:latest", "https://uptime.kuma.pet/", "{}"},

		// Home Automation
		{"home_assistant", "automation", intPtr(8123), "http", "/api/", "bearer", "home-assistant", "homeassistant/home-assistant:latest", "https://home-assistant.io/", "{}"},
		{"node_red", "automation", intPtr(1880), "http", "/", "basic", "node-red", "nodered/node-red:latest", "https://nodered.org/", "{}"},

		// Development & CI/CD
		{"gitlab", "development", intPtr(80), "http", "/users/sign_in", "none", "gitlab", "gitlab/gitlab-ce:latest", "https://gitlab.com/", "{}"},
		{"gitea", "development", intPtr(3000), "http", "/", "basic", "gitea", "gitea/gitea:latest", "https://gitea.io/", "{}"},
		{"jenkins", "development", intPtr(8080), "http", "/", "basic", "jenkins", "jenkins/jenkins:lts", "https://jenkins.io/", "{}"},

		// Storage & NAS
		{"nextcloud", "storage", intPtr(443), "https", "/status.php", "basic", "nextcloud", "nextcloud:latest", "https://nextcloud.com/", "{}"},
		{"syncthing", "storage", intPtr(8384), "http", "/rest/system/status", "api_key", "syncthing", "syncthing/syncthing:latest", "https://syncthing.net/", "{}"},
		{"minio", "storage", intPtr(9000), "http", "/minio/health/live", "basic", "minio", "minio/minio:latest", "https://min.io/", "{}"},
	}

	query := `INSERT INTO service_types (name, category, default_port, default_check_type, health_endpoint, auth_type, icon, docker_image, documentation_url, configuration_template)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	for _, st := range serviceTypes {
		_, err := db.Exec(query, st.Name, st.Category, st.DefaultPort, st.DefaultCheckType, st.HealthEndpoint, st.AuthType, st.Icon, st.DockerImage, st.DocumentationURL, st.ConfigurationTemplate)
		if err != nil {
			return fmt.Errorf("failed to insert service type %s: %w", st.Name, err)
		}
	}

	logrus.WithField("count", len(serviceTypes)).Info("Core service types initialized successfully")

	// Additional service types from SPEC for comprehensive coverage
	// These provide discovery hints and can be enhanced with full metadata later
	additionalServiceTypes := []struct {
		Name     string
		Category string
		Port     *int
	}{
		// Web Services (remaining)
		{"envoy", "web", intPtr(9901)},
		{"kong", "web", intPtr(8001)},
		{"istio", "web", intPtr(15021)},
		{"cloudflare_tunnel", "web", nil},

		// Databases (remaining)
		{"mssql", "database", intPtr(1433)},
		{"oracle", "database", intPtr(1521)},
		{"cockroachdb", "database", intPtr(26257)},
		{"yugabyte", "database", intPtr(7000)},
		{"tidb", "database", intPtr(4000)},
		{"cassandra", "database", intPtr(9042)},
		{"couchdb", "database", intPtr(5984)},
		{"influxdb", "database", intPtr(8086)},
		{"neo4j", "database", intPtr(7474)},
		{"arangodb", "database", intPtr(8529)},
		{"rethinkdb", "database", intPtr(28015)},

		// Container Platforms (remaining)
		{"docker_swarm", "container", intPtr(2377)},
		{"k3s", "container", intPtr(6443)},
		{"podman", "container", nil},
		{"lxd", "container", intPtr(8443)},
		{"incus", "container", intPtr(8443)},
		{"proxmox", "container", intPtr(8006)},

		// Virtualization
		{"vmware_esxi", "virtualization", intPtr(443)},
		{"vmware_vcenter", "virtualization", intPtr(443)},
		{"proxmox_ve", "virtualization", intPtr(8006)},
		{"xenserver", "virtualization", intPtr(443)},
		{"xcp_ng", "virtualization", intPtr(443)},
		{"hyper_v", "virtualization", intPtr(5985)},
		{"kvm_libvirt", "virtualization", intPtr(16509)},
		{"virtualbox", "virtualization", intPtr(18083)},
		{"nutanix", "virtualization", intPtr(9440)},
		{"openstack", "virtualization", intPtr(5000)},

		// Media (remaining)
		{"kodi", "media", intPtr(8080)},
		{"subsonic", "media", intPtr(4040)},
		{"navidrome", "media", intPtr(4533)},
		{"airsonic", "media", intPtr(4040)},
		{"funkwhale", "media", intPtr(3000)},
		{"photoprism", "media", intPtr(2342)},
		{"immich", "media", intPtr(3001)},

		// Automation (remaining *arr)
		{"readarr", "automation", intPtr(8787)},
		{"bazarr", "automation", intPtr(6767)},
		{"overseerr", "automation", intPtr(5055)},
		{"jellyseerr", "automation", intPtr(5055)},
		{"ombi", "automation", intPtr(3579)},
		{"tautulli", "automation", intPtr(8181)},

		// Download Clients (remaining)
		{"nzbget", "download", intPtr(6789)},
		{"deluge", "download", intPtr(8112)},
		{"rutorrent", "download", intPtr(80)},
		{"aria2", "download", intPtr(6800)},
		{"jackett", "download", intPtr(9117)},
		{"nzbhydra2", "download", intPtr(5076)},
		{"flaresolverr", "download", intPtr(8191)},

		// Network & Security (remaining)
		{"pfsense", "network", intPtr(443)},
		{"opnsense", "network", intPtr(443)},
		{"openwrt", "network", intPtr(80)},
		{"wireguard", "vpn", intPtr(51820)},
		{"openvpn", "vpn", intPtr(1194)},
		{"tailscale", "vpn", intPtr(41641)},
		{"zerotier", "vpn", intPtr(9993)},

		// Authentication & Identity
		{"authentik", "auth", intPtr(9000)},
		{"authelia", "auth", intPtr(9091)},
		{"keycloak", "auth", intPtr(8080)},
		{"freeipa", "auth", intPtr(443)},
		{"active_directory", "auth", intPtr(389)},
		{"openldap", "auth", intPtr(389)},
		{"oauth2_proxy", "auth", intPtr(4180)},
		{"dex", "auth", intPtr(5556)},
		{"zitadel", "auth", intPtr(8080)},
		{"fusionauth", "auth", intPtr(9011)},

		// Monitoring (remaining)
		{"loki", "monitoring", intPtr(3100)},
		{"kibana", "monitoring", intPtr(5601)},
		{"datadog_agent", "monitoring", intPtr(8126)},
		{"new_relic", "monitoring", nil},
		{"sentry", "monitoring", intPtr(9000)},
		{"healthchecks", "monitoring", intPtr(8000)},

		// Development & CI/CD (remaining)
		{"drone", "development", intPtr(80)},
		{"woodpecker", "development", intPtr(8000)},
		{"argocd", "development", intPtr(8080)},
		{"flux", "development", intPtr(9090)},
		{"sonarqube", "development", intPtr(9000)},
		{"harbor", "development", intPtr(443)},
		{"nexus", "development", intPtr(8081)},

		// Communication & Collaboration
		{"mattermost", "communication", intPtr(8065)},
		{"rocketchat", "communication", intPtr(3000)},
		{"matrix_synapse", "communication", intPtr(8008)},
		{"element", "communication", intPtr(80)},
		{"discord_bot", "communication", nil},
		{"slack_bot", "communication", nil},
		{"zulip", "communication", intPtr(443)},
		{"discourse", "communication", intPtr(80)},
		{"xenforo", "communication", intPtr(80)},
		{"nodebb", "communication", intPtr(4567)},

		// Email Services
		{"postfix", "email", intPtr(25)},
		{"dovecot", "email", intPtr(143)},
		{"mailcow", "email", intPtr(443)},
		{"mailu", "email", intPtr(443)},
		{"poste_io", "email", intPtr(443)},
		{"zimbra", "email", intPtr(443)},
		{"exchange", "email", intPtr(443)},
		{"roundcube", "email", intPtr(80)},
		{"rainloop", "email", intPtr(80)},
		{"sogo", "email", intPtr(80)},

		// Backup Solutions
		{"veeam", "backup", intPtr(9443)},
		{"duplicati", "backup", intPtr(8200)},
		{"restic", "backup", intPtr(8000)},
		{"borgbackup", "backup", nil},
		{"kopia", "backup", intPtr(51515)},
		{"urbackup", "backup", intPtr(55414)},
		{"proxmox_backup", "backup", intPtr(8007)},
		{"bacula", "backup", intPtr(9101)},
		{"rclone", "backup", intPtr(5572)},

		// Storage & NAS (remaining)
		{"truenas", "storage", intPtr(443)},
		{"unraid", "storage", intPtr(443)},
		{"openmediavault", "storage", intPtr(80)},
		{"synology_dsm", "storage", intPtr(5000)},
		{"qnap", "storage", intPtr(443)},
		{"owncloud", "storage", intPtr(443)},
		{"seafile", "storage", intPtr(8000)},
		{"ceph", "storage", intPtr(7480)},

		// Home Automation (remaining)
		{"openhab", "automation", intPtr(8080)},
		{"domoticz", "automation", intPtr(8080)},
		{"mosquitto", "iot", intPtr(1883)},
		{"zigbee2mqtt", "iot", intPtr(8080)},
		{"zwavejs2mqtt", "iot", intPtr(8091)},
		{"frigate", "automation", intPtr(5000)},
		{"homebridge", "automation", intPtr(8581)},
		{"hubitat", "automation", intPtr(80)},

		// Game Servers
		{"minecraft", "gaming", intPtr(25565)},
		{"pterodactyl", "gaming", intPtr(80)},
		{"amp", "gaming", intPtr(8080)},
		{"csgo", "gaming", intPtr(27015)},
		{"rust_game", "gaming", intPtr(28015)},
		{"valheim", "gaming", intPtr(2456)},
		{"terraria", "gaming", intPtr(7777)},
		{"factorio", "gaming", intPtr(34197)},
		{"ark_server", "gaming", intPtr(27015)},
		{"satisfactory", "gaming", intPtr(15777)},

		// Business & Productivity
		{"onlyoffice", "office", intPtr(443)},
		{"collabora", "office", intPtr(9980)},
		{"cryptpad", "office", intPtr(3000)},
		{"etherpad", "office", intPtr(9001)},
		{"hedgedoc", "office", intPtr(3000)},
		{"bookstack", "office", intPtr(80)},
		{"outline", "office", intPtr(3000)},
		{"firefly_iii", "finance", intPtr(80)},
		{"invoice_ninja", "finance", intPtr(80)},
		{"akaunting", "finance", intPtr(80)},

		// Analytics & BI
		{"metabase", "analytics", intPtr(3000)},
		{"redash", "analytics", intPtr(5000)},
		{"superset", "analytics", intPtr(8088)},
		{"plausible", "analytics", intPtr(8000)},
		{"matomo", "analytics", intPtr(80)},
		{"umami", "analytics", intPtr(3000)},
		{"posthog", "analytics", intPtr(8000)},
		{"splunk", "analytics", intPtr(8000)},
	}

	// Insert additional service types with basic metadata
	for _, st := range additionalServiceTypes {
		_, err := db.Exec(query, st.Name, st.Category, st.Port, "http", "/", "none", st.Name, "", "", "{}")
		if err != nil {
			return fmt.Errorf("failed to insert additional service type %s: %w", st.Name, err)
		}
	}

	totalCount := len(serviceTypes) + len(additionalServiceTypes)
	logrus.WithField("total_count", totalCount).Info("All service types initialized successfully")
	return nil
}

// InitializeThemes populates the themes table with default themes
func (db *DB) InitializeThemes() error {
	logrus.Info("Initializing themes")

	draculaColors := `{
		"background": "#282a36",
		"foreground": "#f8f8f2",
		"primary": "#bd93f9",
		"secondary": "#8be9fd",
		"accent": "#50fa7b",
		"error": "#ff5555",
		"warning": "#f1fa8c",
		"success": "#50fa7b",
		"info": "#8be9fd"
	}`

	draculaFonts := `{
		"primary": "Inter",
		"monospace": "JetBrains Mono",
		"weights": [400, 500, 600, 700]
	}`

	query := `INSERT INTO themes (name, slug, colors, fonts, is_default, is_custom)
			  VALUES (?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query, "Dracula", "dracula", draculaColors, draculaFonts, true, false)
	if err != nil {
		return fmt.Errorf("failed to insert Dracula theme: %w", err)
	}

	logrus.Info("Themes initialized successfully")
	return nil
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}