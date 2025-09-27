# CasDash - Complete Implementation Specification

## PROJECT IDENTITY
- **Repository**: github.com/casapps/casdash
- **License**: MIT License with full text in LICENSE file
- **Language**: Go (minimum version 1.21)
- **Binary Name**: casdash
- **Target Platforms**: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- **Copyright**: Copyright (c) 2024 CasApps - CasjaysDev Applications

## PROJECT PHILOSOPHY
**Primary Goal**: Create the ultimate self-hosted service dashboard that combines beautiful homepage functionality (like Homer/Dashy) with comprehensive monitoring capabilities (like Uptime Kuma) plus enterprise-grade features, all in a single binary with zero external dependencies.

**Target Users**: Self-hosters (primary), families sharing infrastructure, small teams, MSP clients, enterprises

**Design Principles**: 
- Mobile-first responsive design
- Security-first approach
- Zero-config defaults with deep customization options
- Single binary deployment
- Beautiful Dracula theme default
- Database-driven configuration (no config files)

## OPERATING MODES

### Mode Selection
CasDash operates in exactly TWO modes, selected at startup. Mode cannot be changed after initialization.

**Command Line**:
```bash
casdash                    # Default: Enterprise Mode
casdash --mode enterprise
casdash --mode saas
```

**Environment Variable**:
```bash
CASDASH_MODE=enterprise    # Default
CASDASH_MODE=saas
```

### Mode Behavioral Differences

**Enterprise Mode (Default)**:
- Target: Self-hosters, families, teams, enterprises
- Users: Internal (company employees, family members, team members)
- Service Scope: Organization-scoped with role-based permissions
- Service Ownership: All services belong to organization, users have permissions
- Registration: Admin-controlled (invite/LDAP/domain/manual)
- Billing: Always disabled
- Public Dashboard: /{username} shows organization's public services
- Custom Domains: Not available
- User Mental Model: "Monitor our infrastructure"

**SaaS Mode**:
- Target: External customers/subscribers
- Users: External paying customers
- Service Scope: User-scoped with personal ownership
- Service Ownership: Users personally own their services
- Registration: Open signup (unless admin disables)
- Billing: Enabled if billing provider configured
- Public Dashboard: /{username} shows that user's personal public services
- Custom Domains: Available with automatic SSL
- User Mental Model: "Monitor my infrastructure"

## STARTUP AND CONFIGURATION

### Configuration Hierarchy
1. Environment variables used ONLY during first startup/initialization
2. Database becomes single source of truth after first run
3. All settings configurable via admin UI after initialization
4. Sane defaults provided for everything

### Port Management
- NEVER use standard ports (no 8080, 3000, 80, 443)
- On startup: Scan for available port in 64000-65535 range
- Select random unused port from this range
- Write selected port to database for persistence
- Always use database port on subsequent starts
- Admin can change port via UI with availability validation

### Environment Variables (Initialization Only)
```bash
CASDASH_MODE=enterprise
CASDASH_DB_TYPE=sqlite
CASDASH_DB_PATH=./casdash.db
CASDASH_DB_HOST=localhost
CASDASH_DB_PORT=5432
CASDASH_DB_NAME=casdash
CASDASH_DB_USER=casdash
CASDASH_DB_PASSWORD=password
CASDASH_SECRET_KEY=auto-generated-32-char-key
CASDASH_MULTIUSER=false
CASDASH_REGISTRATION=disabled
CASDASH_DEBUG=false
```

## DATABASE ARCHITECTURE

### Database Support
- SQLite (default) - Zero configuration
- PostgreSQL - Production scalability
- MySQL/MariaDB - Enterprise compatibility

### Core Schema

```sql
-- Users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    is_primary_admin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    two_fa_secret VARCHAR(255),
    api_key VARCHAR(255) UNIQUE,
    active BOOLEAN DEFAULT TRUE
);

-- Services table
CREATE TABLE services (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    service_type VARCHAR(100),
    category VARCHAR(100),
    description TEXT,
    icon TEXT,
    auth_type VARCHAR(50),
    auth_credentials TEXT, -- Encrypted JSON
    custom_headers TEXT, -- JSON
    monitoring_enabled BOOLEAN DEFAULT TRUE,
    check_interval INTEGER DEFAULT 300,
    timeout INTEGER DEFAULT 30,
    expected_status_codes TEXT DEFAULT '[200]', -- JSON array
    expected_content TEXT,
    follow_redirects BOOLEAN DEFAULT TRUE,
    ssl_verify BOOLEAN DEFAULT TRUE,
    ssl_monitoring_enabled BOOLEAN DEFAULT NULL,
    public_visible BOOLEAN DEFAULT FALSE,
    public_name VARCHAR(255),
    public_description TEXT,
    maintenance_mode BOOLEAN DEFAULT FALSE,
    position_x INTEGER,
    position_y INTEGER,
    card_size VARCHAR(20) DEFAULT 'medium',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER REFERENCES users(id),
    organization_id INTEGER -- For Enterprise mode
);

-- Settings table (configuration storage)
CREATE TABLE settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,
    category VARCHAR(100) NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by INTEGER REFERENCES users(id)
);
```

## SERVICE DISCOVERY & MONITORING

### Automatic Service Discovery
- Network scanning on startup and scheduled intervals
- Discovery methods: Port scanning, HTTP fingerprinting, Container APIs, mDNS, SNMP
- 2000+ service fingerprints for automatic recognition
- Confidence scoring for discovered services
- One-click service enablement from discovery results

### Supported Service Types (2000+)

**Categories**:
- **Web Services**: HTTP, HTTPS, REST APIs, GraphQL, WebSocket
- **Databases**: PostgreSQL, MySQL, MongoDB, Redis, InfluxDB, CouchDB, Cassandra, MariaDB
- **Container Platforms**: Docker, Kubernetes, LXD, Incus, Podman, Docker Swarm
- **Virtualization**: VMware ESXi/vCenter, Proxmox, XenServer/XCP-ng, Hyper-V, KVM/Libvirt
- **VPS/Cloud**: AWS EC2, Azure VMs, Google Cloud, DigitalOcean, Linode, Vultr
- **Network Services**: DNS, DHCP, NTP, SNMP, SSH, Telnet, FTP, SFTP
- **Email Services**: SMTP, IMAP, POP3, Exchange, Postfix, Dovecot
- **Media Services**: Plex, Jellyfin, Emby, Tautulli, *arr suite (Sonarr, Radarr, etc.)
- **Home Automation**: Home Assistant, OpenHAB, Domoticz, Node-RED
- **Development**: GitLab, GitHub, Jenkins, SonarQube, Nexus
- **Security**: pfSense, OPNsense, WireGuard, OpenVPN, Authentik, Authelia
- **Backup**: Veeam, Duplicati, Restic, Proxmox Backup Server
- **And 1900+ more with optimized monitoring configurations**

### Protocol & Port Monitoring

**TCP/UDP Protocol Support**:
- Email: SMTP (25/587/465), IMAP (143/993), POP3 (110/995)
- File Transfer: FTP (21), SFTP (22), FTPS (990), TFTP (69/UDP)
- Remote Access: SSH (22), Telnet (23), RDP (3389), VNC (5900)
- Web: HTTP (80), HTTPS (443), WebSocket (80/443)
- DNS: TCP/UDP (53)
- VPN: OpenVPN (1194), WireGuard (51820/UDP), IPSec (500/4500/UDP)
- Database: MySQL (3306), PostgreSQL (5432), MongoDB (27017), Redis (6379)
- And many more protocols with appropriate health checks

### Health Check Types
- HTTP/HTTPS requests with full response validation
- TCP port connectivity testing
- UDP protocol testing
- ICMP ping with packet loss tracking
- SSL certificate validation and expiration monitoring
- DNS resolution with response time tracking
- Database connectivity with query execution
- Custom script execution with exit code validation
- Container health checks via Docker/Kubernetes APIs
- SNMP monitoring for network equipment
- Service-specific health endpoints
- Protocol-specific checks (SMTP, IMAP, SSH, etc.)

## SSL/TLS CERTIFICATE MONITORING

### Automatic SSL Detection
- Automatically detects HTTPS and SSL/TLS services
- Toggle SSL monitoring on/off per service
- Monitors any SSL/TLS service (HTTPS, SMTPS, IMAPS, FTPS, etc.)

### Certificate Monitoring Features
- Expiration tracking with configurable alerts (90, 60, 30, 14, 7, 1 days)
- Security grading (A+ to F) based on configuration
- Certificate chain validation
- Protocol version checking (SSL/TLS versions)
- Cipher suite analysis
- Vulnerability scanning (Heartbleed, POODLE, BEAST, etc.)
- OCSP stapling verification
- Certificate Transparency log monitoring
- Certificate pinning support
- Automatic renewal via Let's Encrypt/ACME

### SSL Security Assessment
- Comprehensive security scoring
- PCI-DSS, HIPAA, NIST compliance checking
- Recommendations for improvements
- Certificate cost tracking
- Multi-cloud certificate discovery (AWS ACM, Azure, GCP)

## WATCHTOWER-STYLE UPDATE MANAGEMENT

### Docker Label Support
Fully compatible with Watchtower labels:
- `com.centurylinklabs.watchtower.enable`
- `com.centurylinklabs.watchtower.monitor-only`
- `com.centurylinklabs.watchtower.stop-signal`
- `com.centurylinklabs.watchtower.pre-update-command`
- `com.centurylinklabs.watchtower.post-update-command`
- Plus CasDash-specific labels for enhanced control

### Update Policies
- **Manual**: Notify only, never auto-update
- **Automatic**: Update immediately when available
- **Scheduled**: Update during maintenance windows only
- **Staged**: Update dev → staging → production with delays
- **Approval Required**: Update after admin/user approval
- **Test First**: Update test instance, wait for validation, then production

### Update Features
- Container image update detection
- Version tracking for all service types
- Automatic rollback on failure
- Pre-update backups
- Dependency-aware updates
- Update coordination across service stacks
- Security update prioritization

## SECURITY SYSTEM

### Security Assessment Engine
- Per-service security scoring (1-10 scale)
- Automatic vulnerability detection
- SSL/TLS configuration analysis
- HTTP security header validation
- Authentication method assessment
- Exposure risk evaluation

### Security Features
- Service-specific security guides
- Complete hardening workflows
- Configuration file generation for security improvements
- Firewall rule creation templates
- Reverse proxy configuration generation
- Integration with vulnerability databases
- Compliance tracking (GDPR, HIPAA, SOC2, PCI-DSS)

## INTELLIGENT AUTOMATION

### Issues System
Automated issue creation for:
- Service downtime lasting >5 minutes
- SSL certificate expiration warnings
- Performance degradation
- Security vulnerabilities detected
- Configuration drift from recommended settings
- Dependency failures

### Maintenance System
- Scheduled maintenance windows with monitoring suspension
- Maintenance types: Single service, service groups, infrastructure
- User notifications before, during, and after maintenance
- Progress tracking and status updates
- Automatic dependency assessment
- Calendar interface for scheduling

### Support System
**Three-tier support**:
1. **Bot Chat**: Instant automated responses for common queries
2. **Live Chat**: Real-time human support when available
3. **Support Tickets**: Complete help desk functionality

Features:
- Knowledge base integration
- Service context awareness
- SLA tracking
- Support analytics

## NOTIFICATION SYSTEM

### Channels
- Email (SMTP) with HTML templates
- Webhooks (Discord, Slack, custom)
- In-app notifications with real-time delivery
- Browser push notifications

### Smart Features
- Deduplication to prevent alert spam
- Grouping of related alerts
- Dependency-aware suppression
- Maintenance window respect
- Escalation policies
- Quiet hours configuration

## DASHBOARD SYSTEM

### Private Dashboard
- Real-time status with WebSocket updates
- Mobile-first responsive design (1-6 column grid)
- Service cards (280px × 160px uniform size)
- Drag-and-drop reordering
- SSL status badges on cards
- Three-button system: [Open] [Details] [Menu]

### Public Dashboard
- Available at /{username}
- No authentication required
- Shows only services marked as public
- Returns 404 with marketing page if no public services
- Same design system as private dashboard

### Dashboard Widgets
- Service status overview
- SSL certificate expiry timeline
- Update availability summary
- Performance metrics charts
- Security score distribution
- Maintenance calendar
- Issue tracker summary

## API SYSTEM

### RESTful API
- Complete CRUD operations for all entities
- OpenAPI/Swagger documentation
- API key authentication
- Rate limiting (1000/hour authenticated, 100/hour anonymous)
- Webhook support for automation

### WebSocket Endpoints
- Real-time status updates
- Live chat communication
- Real-time notifications

## BILLING SYSTEM (SaaS Mode Only)

### When Billing Disabled
- Full unlimited access to all features

### When Billing Enabled

**Free Tier** (50% of Basic limits):
- 25 services
- 250 checks/hour
- 15 days data retention

**Basic Plan** ($5/month):
- 50 services
- 500 checks/hour
- 30 days data retention

**Pro Plan** ($10/month):
- 150 services
- 1,500 checks/hour
- 90 days data retention

**Enterprise Plan** ($20/month):
- 500 services
- 5,000 checks/hour
- 1 year data retention

### Payment Support
- Stripe, PayPal, Square
- EU VAT handling
- Automatic invoicing
- Grace periods
- Subscription management

## UI/UX DESIGN

### Default Theme - Dracula
- Background: #282a36
- Primary: #bd93f9
- Secondary: #8be9fd
- Accent: #50fa7b
- Text: #f8f8f2

### Design Requirements
- Mobile-first responsive design
- Touch-friendly (44px minimum targets)
- WCAG 2.1 AA compliance
- Consistent component system
- Smooth animations
- Loading states and skeleton screens

## DEFAULT CONFIGURATION

All defaults stored in database after initialization:

### Monitoring Defaults
- Check interval: 5 minutes
- Timeout: 30 seconds
- Expected status codes: [200, 201, 202, 204]
- SSL expiry warning: 30 days
- Response time warning: 1000ms

### Update Defaults
- Policy: Manual (notify only)
- Check interval: 6 hours
- Backup before update: Yes
- Rollback on failure: Yes
- Exclude tags: alpha, beta, rc, dev

### Security Defaults
- Password minimum: 12 characters
- Session timeout: 24 hours
- Bcrypt cost: 12
- Security scan interval: Weekly

### Data Retention Defaults
- Monitoring data: 90 days
- Logs: 30 days
- Audit trail: 1 year
- Issues: Forever
- Support tickets: Forever

## DEPLOYMENT

### Binary Distribution
- Single Go binary with embedded assets
- No runtime dependencies except database
- Cross-platform support
- Auto-updates supported

### Container Deployment
- Official Docker images
- Multi-architecture support
- Docker Compose examples
- Kubernetes manifests
- Helm charts

### Requirements
- Minimum: 512MB RAM, 1 CPU core
- Recommended: 2GB RAM, 2 CPU cores
- Storage: 100MB + database size
- Database: SQLite (included) or external

## ROUTES ARCHITECTURE

### Core Routes
- `/` - Dashboard home
- `/login` - Authentication
- `/register` - Registration
- `/{username}` - Public dashboard

### Service Routes
- `/add` - Add service wizard
- `/services` - Service list
- `/service/{id}` - Service details
- `/service/{id}/edit` - Edit service
- `/service/{id}/monitor` - Monitoring data

### Admin Routes
- `/admin` - System settings
- `/admin/discovery` - Network discovery
- `/admin/users` - User management
- `/admin/billing` - Billing (SaaS only)

### API Routes
- `/api/service/{id}` - Service CRUD
- `/api/monitoring/uptime` - Uptime data
- `/api/status` - Public status
- `/api/health` - System health

## CRITICAL IMPLEMENTATION NOTES

1. **No Configuration Files**: Everything in database after initialization
2. **Zero Dependencies**: Single binary includes everything
3. **Automatic Everything**: Discovery, SSL detection, updates
4. **Security First**: Encrypted credentials, secure defaults
5. **Mobile First**: Every interface works on mobile
6. **Real-time Updates**: WebSocket for live data
7. **Intelligent Monitoring**: Service-specific optimizations
8. **Complete Solution**: Homepage + Monitoring + Updates + Security

This specification is complete and exhaustive. The resulting implementation must be a fully functional, production-ready application that compiles and runs successfully on first attempt.
