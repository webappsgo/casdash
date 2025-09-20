```
PROJECT: CasDash - Self-Hosted Infrastructure Monitoring Dashboard

COMPLETE IMPLEMENTATION SPECIFICATION FOR AI ASSISTANTS

=== CRITICAL IMPLEMENTATION REQUIREMENTS ===
This specification is COMPLETE and EXHAUSTIVE. Any AI assistant implementing this project must create a fully functional, production-ready application from this specification alone. No additional clarification should be required. The resulting code must compile and run successfully on first attempt.

=== PROJECT IDENTITY ===
Repository: github.com/casapps/casdash
License: MIT License with full text in LICENSE file
Language: Go (minimum version 1.21)
Binary Name: casdash
Target Platforms: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
Copyright: Copyright (c) 2024 CasApps - CasjaysDev Applications

=== PROJECT PHILOSOPHY ===
Primary Goal: Create the ultimate self-hosted service dashboard that combines beautiful homepage functionality (like Homer/Dashy) with comprehensive monitoring capabilities (like Uptime Kuma) plus enterprise-grade features, all in a single binary with zero external dependencies.

Target Users: Self-hosters (primary), families sharing infrastructure, small teams, MSP clients, enterprises
Design Principles: Mobile-first responsive design, security-first approach, zero-config defaults with deep customization options, single binary deployment, beautiful Dracula theme default

=== OPERATING MODES ===
CasDash operates in exactly TWO modes, selected at startup. Mode cannot be changed after initialization.

COMMAND LINE MODE SELECTION:
Default (Enterprise Mode):
casdash
casdash --mode enterprise
casdash --mode=enterprise

SaaS Mode:
casdash --mode saas
casdash --mode=saas

ENVIRONMENT VARIABLE:
CASDASH_MODE=enterprise (default)
CASDASH_MODE=saas

MODE VALIDATION:
Valid values: enterprise, saas
Invalid mode: Exit with error message "Invalid mode. Valid options: enterprise, saas"
Mode is immutable after database initialization

=== MODE BEHAVIORAL DIFFERENCES ===
ENTERPRISE MODE (Default):
- Target: Self-hosters, families, teams, enterprises
- Users: Internal (company employees, family members, team members)
- Service Scope: Organization-scoped with role-based permissions
- Data Query: SELECT * FROM services WHERE organization_id = ? AND user_has_permission()
- Service Ownership: All services belong to organization, users have permissions
- Registration: Admin-controlled (invite/LDAP/domain/manual)
- Billing: Always disabled
- Public Dashboard: /{username} shows organization's public services (same content for all usernames)
- Custom Domains: Not available (users deploy on their own infrastructure)
- User Mental Model: "Monitor our infrastructure"

SAAS MODE:
- Target: External customers/subscribers
- Users: External paying customers
- Service Scope: User-scoped with personal ownership
- Data Query: SELECT * FROM services WHERE user_id = ?
- Service Ownership: Users personally own their services
- Registration: Open signup (unless admin disables)
- Billing: Enabled if billing provider configured, free tier available
- Public Dashboard: /{username} shows that user's personal public services
- Custom Domains: Available with automatic SSL
- User Mental Model: "Monitor my infrastructure"

CRITICAL: Same features, same UI, same automation capabilities. Only data scoping and target users differ.

=== STARTUP AND CONFIGURATION ===
CONFIGURATION HIERARCHY:
1. Environment variables used ONLY during first startup/initialization
2. Database becomes single source of truth after first run
3. All settings configurable via admin UI after initialization
4. Sane defaults provided for everything

PORT MANAGEMENT:
- NEVER use standard or well-known ports (no 8080, 3000, 80, 443, etc.)
- On startup: Scan for available port in 64000-65535 range
- Select random unused port from this range
- Write selected port to database for persistence
- Always use database port on subsequent starts
- Admin can change port via UI with availability validation
- Design philosophy: Prefer running behind reverse proxy

ENVIRONMENT VARIABLES (Initialization Only):
CASDASH_MODE=enterprise (enterprise|saas)
CASDASH_DB_TYPE=sqlite (sqlite|postgres|mysql|mariadb)
CASDASH_DB_PATH=./casdash.db (for SQLite only)
CASDASH_DB_HOST=localhost (for external DBs)
CASDASH_DB_PORT=5432 (for external DBs)
CASDASH_DB_NAME=casdash (for external DBs)
CASDASH_DB_USER=casdash (for external DBs)
CASDASH_DB_PASSWORD=password (for external DBs)
CASDASH_SECRET_KEY=auto-generated-32-char-key
CASDASH_MULTIUSER=false (true|false)
CASDASH_REGISTRATION=disabled (open|approval|disabled)
CASDASH_DEBUG=false (true|false)
CASDASH_DISCOVERY_ENABLED=true (true|false)
CASDASH_DISCOVERY_INTERVAL=24h
CASDASH_DISCOVERY_NETWORKS=auto-detect
CASDASH_DISCOVERY_PORTS=22,53,80,443,3000,5432,8080,8443,9000
CASDASH_DISCOVERY_PRIVILEGED=true (true|false)

=== ARCHITECTURE OVERVIEW ===
Single Go binary with embedded web assets, templates, and database migrations
Multi-process design: Main web application (unprivileged) + Discovery service (privileged for network scanning)
Database abstraction layer supporting SQLite (default), PostgreSQL, MySQL, MariaDB
Template-based configuration generation system with service-specific optimizations
Real-time updates via WebSocket connections
RESTful API with comprehensive endpoints
Embedded frontend assets using Go's embed package
No runtime dependencies except chosen database

=== USER SYSTEM ===
AUTHENTICATION:
- First user registration becomes primary admin automatically in both modes
- Bcrypt password hashing with cost 12
- Secure session management with configurable timeouts (default 24 hours)
- CSRF protection on all state-changing operations
- Rate limiting: 5 failed login attempts per IP per 15 minutes
- 2FA support via TOTP (optional)
- API key authentication for programmatic access

USER TYPES AND ROLES:
- Primary Admin: First user, immutable role, full system access
- Admin: Full system access, can manage users and settings
- User: Standard user with service management capabilities
- Support: Can access support tickets and chat across all users (both modes)
- View-Only: Read-only access to permitted services

MULTI-USER MODE:
- Disabled by default (CASDASH_MULTIUSER=false)
- Enable via CASDASH_MULTIUSER=true environment variable
- Controls whether multiple user accounts are allowed
- Single user mode: Only primary admin account exists
- Multi-user mode: Multiple accounts with role-based access

REGISTRATION MODES (Enterprise):
- disabled: No new registrations allowed
- approval: Admin must approve new registrations
- open: Anyone can register (not recommended for Enterprise)
- LDAP integration: Use corporate directory
- Email domain whitelist: Restrict to specific domains
- Invitation-only: Admin sends invitation links

REGISTRATION MODES (SaaS):
- open: Default, anyone can signup
- disabled: Admin can disable new signups
- approval: Admin approval required (for private SaaS instances)

GUEST ACCESS:
- Always enabled for public status pages (/{username})
- No authentication required for public service viewing
- Configurable rate limits for guest access

=== SERVICE MANAGEMENT ===
SERVICE DISCOVERY:
- Automatic network scanning on startup and scheduled intervals
- Privileged background process for network access
- Discovery methods: Port scanning, HTTP fingerprinting, Container APIs, mDNS, SNMP
- Configurable network ranges and port lists
- 600+ service fingerprints for automatic recognition
- Confidence scoring for discovered services
- One-click service enablement from discovery results

SUPPORTED SERVICE TYPES (600+ with templates):
Web Services: HTTP, HTTPS, REST APIs, GraphQL, WebSocket
Databases: PostgreSQL, MySQL, MongoDB, Redis, InfluxDB, CouchDB, Cassandra, MariaDB
Container Platforms: Docker, Kubernetes, LXD, Incus, Podman, Docker Swarm
VPS/Cloud: AWS EC2, Azure VMs, Google Cloud, DigitalOcean, Linode, Vultr
Network Services: DNS, DHCP, NTP, SNMP, SSH, Telnet, FTP, SFTP
Email Services: SMTP, IMAP, POP3, Exchange, Postfix, Dovecot
File Services: SMB, NFS, WebDAV, S3, FTP, SFTP
Media Services: Plex, Jellyfin, Emby, Subsonic, Airsonic
Home Automation: Home Assistant, OpenHAB, Domoticz
Development: GitLab, GitHub, Jenkins, SonarQube, Nexus
Monitoring: Prometheus, Grafana, Nagios, Zabbix
Networking: pfSense, OPNsense, UniFi, Mikrotik
And 500+ more with optimized monitoring configurations

SERVICE CONFIGURATION FIELDS:
- name (string, required): Display name for service
- url (string, required): Full URL or connection string
- service_type (string, auto-detected): One of 600+ predefined service types
- category (string): User-defined or auto-suggested category
- description (string, optional): Service description
- icon (string): Icon source (library/upload/url/favicon)
- auth_type (enum): none/basic/bearer/api_key/oauth2/custom_headers
- auth_credentials (encrypted JSON): Authentication details stored with AES-256
- custom_headers (JSON): Additional headers for requests
- monitoring_enabled (boolean, default true): Enable health monitoring
- check_interval (integer, default 300): Seconds between health checks
- timeout (integer, default 30): Request timeout in seconds
- expected_status_codes (JSON array): HTTP status codes considered healthy (default [200])
- expected_content (string): Text that must be present in response
- follow_redirects (boolean, default true): Follow HTTP redirects
- ssl_verify (boolean, default true): Verify SSL certificates
- public_visible (boolean, default false): Show on public status page
- public_name (string): Different name for public display
- public_description (string): Different description for public display
- maintenance_mode (boolean, default false): Temporarily disable monitoring
- position_x (integer): Card position for drag-drop ordering
- position_y (integer): Card position for drag-drop ordering
- card_size (enum): small/medium/large card display size
- created_at (timestamp): Service creation time
- updated_at (timestamp): Last modification time
- created_by (integer): User ID who created service

SERVICE TYPE SELECTION UX:
PROBLEM: 600+ service types too many for dropdown, especially mobile
SOLUTION: Multi-approach selection system

Approach 1 - Categorized Selection:
Step 1: Select Category (Web Services, Databases, Network Services, Containers, etc.)
Step 2: Select Specific Service within category

Approach 2 - Search with Auto-complete:
Search box with real-time filtering
User types "mys" → Shows MySQL, MySQL Cluster
User types "doc" → Shows Docker, Docker Swarm, Docker Registry

Approach 3 - Popular + Recent + Search:
Mobile-optimized layout showing:
- Popular Services (top 20 most used)
- Recently Added (user's last 5 types)
- Search bar for everything else
- "Browse All Categories" button

Approach 4 - Smart Detection:
URL-based suggestions when user enters service URL
"https://mysql.example.com:3306" → "Looks like MySQL - use MySQL monitoring?"

=== MONITORING ENGINE ===
HEALTH CHECK TYPES:
- HTTP/HTTPS requests with full response validation
- TCP port connectivity testing
- ICMP ping with packet loss tracking
- SSL certificate validation and expiration monitoring
- DNS resolution with response time tracking
- Database connectivity with query execution
- Custom script execution with exit code validation
- Container health checks via Docker/Kubernetes APIs
- SNMP monitoring for network equipment
- Service-specific health endpoints

MONITORING STATES:
- active: Normal monitoring active
- paused_dependency: Paused due to dependency failure
- paused_maintenance: Paused for scheduled maintenance
- paused_manual: Manually paused by user
- investigating: Issue under investigation, monitoring continues

DEPENDENCY DETECTION AND MANAGEMENT:
- Automatic detection of service dependencies
- Docker containers depend on Docker daemon
- Kubernetes pods depend on cluster
- Database applications depend on database servers
- Web applications depend on reverse proxies
- Smart cascade management to prevent alert spam
- Progressive recovery validation when dependencies restore

REAL-TIME MONITORING:
- WebSocket-based status updates to dashboard
- Sub-second status propagation to connected clients
- Configurable check intervals per service (30 seconds to 24 hours)
- Performance metrics: Response time, uptime percentage
- Historical data retention with configurable periods
- Real-time dashboard updates without page refresh

ALERTING SYSTEM:
- Configurable notification channels: Email (SMTP), webhooks (Discord/Slack/custom)
- Smart deduplication to prevent alert spam
- Dependency-aware suppression (don't alert on dependent services)
- Escalation policies with progressive notifications
- Quiet hours configuration per user
- Maintenance window awareness (auto-suspend alerts)
- Alert grouping for related failures

=== SECURITY SYSTEM ===
SECURITY ASSESSMENT ENGINE:
- Per-service security scoring (1-10 scale)
- Automatic vulnerability detection for known service types
- SSL/TLS configuration analysis
- HTTP security header validation
- Authentication method assessment
- Exposure risk evaluation (public vs private)
- Compliance checking against security standards

SECURITY RECOMMENDATIONS:
- Service-specific security guides with step-by-step instructions
- Complete hardening workflows with copy-paste commands
- Configuration file generation for security improvements
- Regular security scan scheduling
- Automatic security issue creation for vulnerabilities
- Integration with vulnerability databases

SUPPORTED SECURITY IMPROVEMENTS:
- HTTPS enforcement configuration
- Reverse proxy configuration generation (Nginx, Apache, Caddy, Traefik, HAProxy)
- Firewall rule creation for service isolation
- SSL certificate management and automation
- Authentication strengthening recommendations
- Security header implementation guides

SECURITY INTEGRATION:
- Continuous security monitoring with scheduled scans
- Automatic security issue creation for detected problems
- Compliance tracking and reporting
- Security audit logging for all actions
- Integration with security standards (OWASP, NIST)

=== ISSUES SYSTEM ===
AUTOMATED ISSUE CREATION:
Issues are automatically created by monitoring system for:
- Service downtime lasting more than 5 minutes
- SSL certificate expiration warnings (30, 7, and 1 day before expiry)
- Performance degradation (response time >50% slower than baseline)
- Security vulnerabilities detected in scans
- Configuration drift from recommended settings
- Dependency failures affecting multiple services

ISSUE TYPES AND CATEGORIES:
- Bug reports: Service malfunctions and errors
- Performance problems: Slow response times, timeouts
- Security issues: Vulnerabilities, misconfigurations
- Maintenance planning: Scheduled work coordination
- Feature requests: User-requested improvements

ISSUE LIFECYCLE:
- Open: Issue created, awaiting action
- In Progress: Work actively happening
- Resolved: Problem fixed, awaiting verification
- Closed: Confirmed resolved and archived

ISSUE FEATURES:
- GitHub-style issue tracking interface
- Service-specific issue templates for common problems
- Automatic issue resolution when underlying problems fix
- Issue analytics: Pattern recognition, reliability metrics
- Smart deduplication to group related issues
- Rich issue descriptions with monitoring data attached

ISSUE MANAGEMENT:
- Assignment to users or teams
- Priority levels (Critical, High, Medium, Low)
- Labels and categorization
- Comments and collaboration
- Time tracking and resolution metrics
- Integration with monitoring data and service context

=== SUPPORT SYSTEM ===
SUPPORT CHANNELS:
Three distinct support communication methods:

1. BOT CHAT (Automated):
- Integrated into live chat system
- Bot responds first to all chat inquiries
- Handles simple data queries instantly:
  * "Is service X down?" → Bot checks monitoring data
  * "SSL expiry date?" → Bot responds with certificate info
  * "Uptime this month?" → Bot responds with statistics
- Offers human escalation: "Would you like to speak with a human?"
- Seamless handoff to live chat with context preservation

2. LIVE CHAT (Human Support):
- Real-time chat when admin/support team is online
- WebSocket-based communication
- Chat history preservation
- Context of user's services available
- Immediate escalation path from bot chat
- Status indicator showing admin online/offline

3. SUPPORT TICKETS (Formal Support):
- Complete help desk functionality
- User-submitted tickets for complex issues
- Admin/support team ticket management
- Ticket status tracking (Open, In Progress, Resolved, Closed)
- Priority levels and categorization
- Integration with service context and monitoring data
- Ticket templates for common support scenarios
- SLA tracking and response time monitoring

SUPPORT USER MANAGEMENT:
- Dedicated support role separate from admin
- Support users can access all user chats/tickets across both modes
- Enterprise Mode: Create support users OR link to LDAP groups
- SaaS Mode: Create dedicated support users for customer service
- Support permissions: Chat/ticket access, service viewing (read-only), knowledge base management
- Support cannot: Modify server config, manage users, change billing

KNOWLEDGE BASE:
- Searchable documentation system
- Service-specific help articles
- Community-contributed content
- Automatic linking of relevant articles to tickets
- Admin-editable content management

SUPPORT ANALYTICS:
- Response time tracking
- Resolution rate monitoring
- User satisfaction scoring
- Support team performance metrics
- Common issue identification

=== MAINTENANCE SYSTEM ===
MAINTENANCE WINDOW MANAGEMENT:
- Scheduled maintenance with automatic monitoring suspension
- Maintenance types: Single service, service groups, infrastructure platforms, network segments, entire system
- User notifications before, during, and after maintenance
- Progress tracking and status updates

MAINTENANCE WORKFLOW:
- Pre-maintenance notifications to affected users
- Automatic status updates during maintenance
- Post-maintenance validation and service restoration
- Impact analysis with automatic dependency assessment
- Stakeholder notification based on affected services

MAINTENANCE CALENDAR:
- Calendar interface for maintenance scheduling
- Conflict detection and resolution
- Recurring maintenance pattern support
- Integration with external calendar systems
- Maintenance history and analytics

AUTO-DETECTION:
- Service-initiated maintenance detection via API responses
- HTTP header analysis for maintenance mode
- Automatic maintenance window creation for detected maintenance
- Smart correlation with planned maintenance windows

=== NOTIFICATION SYSTEM ===
NOTIFICATION CHANNELS:
- Email (SMTP) with HTML templates
- Webhooks for Discord, Slack, and custom integrations
- In-app notifications with real-time delivery
- Browser push notifications (optional)

SMART NOTIFICATION FEATURES:
- Deduplication to prevent alert spam
- Grouping of related alerts
- Intelligent escalation timing
- Dependency-aware suppression
- Maintenance window respect (auto-suspend during maintenance)

NOTIFICATION RULES:
- Threshold-based alerting (response time, uptime percentage)
- Dependency-aware suppression rules
- Escalation policies with progressive alert escalation
- On-call schedule integration
- Emergency override capabilities for critical services

USER PREFERENCES:
- Per-user notification settings
- Quiet hours configuration (no alerts during specified times)
- Alert priority filtering (only critical, high priority, etc.)
- Channel preferences per notification type
- Notification frequency controls

=== DASHBOARD SYSTEM ===
PUBLIC DASHBOARDS:
Enterprise Mode:
- /{username} shows organization's public services
- Same content regardless of which username is accessed
- Shows only services marked as public_visible=true
- Cannot be disabled, always available
- Shows welcome message if no public services

SaaS Mode:
- /{username} shows that specific user's public services
- Different content per user
- Shows only that user's services marked as public_visible=true
- Each user has unique public dashboard

PUBLIC DASHBOARD BEHAVIOR:
- No authentication required
- Shows only services marked as public
- All services created as private by default (public_visible=false)
- User must explicitly mark services as public
- Service cards show: name, status, uptime, response time
- Same design system and layout as private dashboard

EMPTY PUBLIC DASHBOARD:
When /{username} has no public services:
- Return HTTP 404 status code
- Serve custom "About CasDash" page (not generic 404 error)
- Self-publicity content showcasing CasDash capabilities
- Professional project marketing and feature highlights
- Links to project documentation and repository
- Same uniform design system as rest of application
- Admin can fully customize this content via UI
- Default content promotes CasDash project

PRIVATE DASHBOARDS:
- Full service management interface requiring authentication
- Real-time status with WebSocket updates
- Mobile-first responsive design
- Service cards in uniform layout (280px × 160px)
- Three-button system per card: [Open] [Details] [Menu]
- Drag-and-drop reordering with position persistence
- 1-6 column responsive grid based on screen width

SERVICE CARDS:
- Uniform card dimensions: 280px × 160px
- Service icon + name + status + response time display
- Status indicators via colored borders (green=up, red=down, yellow=warning)
- Card actions: Open service (new tab), view details, access management menu
- Hover effects and smooth animations
- Touch-friendly for mobile (44px minimum touch targets)

LAYOUT OPTIONS:
- Responsive grid system (1-6 columns based on screen width)
- Auto-sizing based on available space
- Drag-and-drop reordering with visual feedback
- Position persistence in database
- Card size options (small/medium/large)

=== DOMAIN MAPPING (SaaS Mode Only) ===
CUSTOM DOMAIN SUPPORT:
- SaaS users can map custom FQDNs to their dashboard
- Domain verification required before SSL issuance
- Automatic Let's Encrypt SSL certificate provisioning
- DNS challenge support with multiple providers

DOMAIN VERIFICATION PROCESS:
1. User adds custom domain (e.g., monitoring.acme-corp.com)
2. CasDash checks DNS records
3. Validates domain points to SaaS instance external IP(s)
4. Only proceeds with SSL certificate request after verification
5. Prevents unauthorized certificate requests and domain hijacking

DNS VALIDATION:
- Check A records and CNAME records
- Validate domain points to configured external IP addresses
- Support for multiple IPs (load balancer configurations)
- Reject domains pointing to wrong infrastructure
- Clear error messages with setup instructions

AUTOMATIC SSL:
- Let's Encrypt integration with HTTP-01 challenge (default for users)
- Automatic certificate renewal
- Certificate expiration monitoring per domain
- Support for wildcard certificates where appropriate

DOMAIN CONFIGURATION:
- Admin configures SaaS instance external IP addresses
- User adds custom domain in dashboard
- System provides DNS setup instructions
- Automatic validation and certificate provisioning
- Professional custom domain experience for SaaS users

=== LET'S ENCRYPT INTEGRATION ===
BUILT-IN ACME CLIENT:
- Complete Let's Encrypt ACME v2 client implementation
- HTTP-01 challenge support (default for SaaS users)
- DNS-01 challenge support (default RFC2136, plus 50+ providers)
- Automatic certificate renewal (30 days before expiry)
- Certificate monitoring and expiration alerting

DNS CHALLENGE PROVIDERS:
Default: RFC2136 (DNS UPDATE standard)
Supported Providers:
- Cloudflare, AWS Route53, Google Cloud DNS, Azure DNS
- DigitalOcean, Namecheap, GoDaddy, Hover
- Vultr, Linode, Hetzner, OVH
- And 40+ additional DNS providers
- Custom provider plugin system

HTTP CHALLENGE:
- Automatic via reverse proxy configuration
- Works with generated Nginx, Apache, Caddy configs
- No additional setup required for standard deployments
- Fallback option if DNS challenge fails

CERTIFICATE MANAGEMENT:
- Automatic renewal 30 days before expiration
- Renewal failure notifications and retry logic
- Certificate storage in database (encrypted)
- Integration with reverse proxy configuration generation
- Certificate expiration monitoring with alerts

SSL CONFIGURATION GENERATION:
- Complete reverse proxy configs with SSL
- Modern cipher suites and security headers
- HSTS, OCSP stapling, perfect forward secrecy
- HTTP to HTTPS redirect configuration
- Ready-to-use configurations requiring no manual SSL setup

=== ROUTES ARCHITECTURE ===
CORE ROUTES:
/ - Dashboard home (redirects to login if not authenticated)
/login - User authentication
/register - User registration (if enabled for mode)
/{username} - Public dashboard (no auth required, shows public services only)

SERVICE MANAGEMENT:
/add - Add new service wizard with service type selection
/services - All services list view
/service/{id} - Individual service details and monitoring data
/service/{id}/edit - Edit service configuration
/service/{id}/monitor - Detailed monitoring data and charts
/service/{id}/secure - Service-specific security assessment and guide

SECURITY SYSTEM:
/security - Security overview dashboard
/security/{service} - Service-specific security guide
/security/{service}/fix - Step-by-step security remediation
/security/{service}/scan - Run security assessment

TOOLS & CONFIGURATION:
/tools - Configuration generation tools
/tools/proxy - Reverse proxy configuration generator
/tools/ssl - SSL certificate setup helper
/tools/firewall - Firewall rules generator
/tools/backup - Backup and export utilities

SUPPORT SYSTEM:
/support - Support ticket dashboard
/support/new - Create support ticket
/support/tickets - View all support tickets
/support/chat - Live support chat interface
/support/docs - Knowledge base and documentation

USER MANAGEMENT:
/users - User management (admin) and account settings
/users/account - Current user account settings
/users/add - Add new user (admin only)
/users/{id} - View/edit user details (admin only)

MAINTENANCE SYSTEM:
/maintenance - Maintenance overview and scheduling
/maintenance/calendar - Maintenance calendar view
/maintenance/history - Maintenance history and analytics
/maintenance/settings - Maintenance configuration

ISSUES SYSTEM:
/issues - All service issues dashboard
/service/{id}/issues - Service-specific issues
/issues/analytics - Issue trends and insights

ADMINISTRATION:
/admin - Admin system settings
/admin/discovery - Network discovery configuration
/admin/health - CasDash self-monitoring
/admin/appearance - UI themes and branding
/admin/users - User management
/admin/billing - Billing configuration (SaaS mode only)

API ROUTES:
/api/service/{id} - Service CRUD operations
/api/service/{id}/check - Force health check
/api/service/{id}/public - Toggle public visibility
/api/monitoring/uptime - Uptime data for charts
/api/security/{service} - Security recommendations
/api/status - Public status data (no auth required)
/api/health - CasDash system health
/api/users - User management
/api/issues - Issue management
/api/support - Support ticket management
/api/maintenance - Maintenance window management

WEBSOCKET ROUTES:
/ws/status - Real-time status updates
/ws/chat - Live chat communication
/ws/notifications - Real-time notifications

=== BILLING SYSTEM (SaaS Mode Only) ===
BILLING ACTIVATION:
- Billing only enabled if payment provider configured
- If no billing provider: Full unlimited access (like Enterprise mode)
- If billing provider configured: Free tier + paid plans available

FREE TIER LOGIC:
- When no billing provider: Unlimited everything
- When billing provider configured: Free tier = 50% of lowest paid plan
- Example: If Basic plan = 50 services, Free tier = 25 services
- All features available in free tier, just with usage limits

BILLING PLANS (SaaS Mode):
FREE TIER (when billing enabled):
- 25 services (50% of Basic)
- 250 checks/hour (50% of Basic)
- 15 days data retention (50% of Basic)
- All features available with usage limits
- Basic support (bot + tickets)

BASIC PLAN - $5/month:
- 50 services
- 500 checks/hour (every 7.2 seconds)
- 30 days data retention
- All monitoring features
- Security assessments
- Public status page
- Email notifications
- Standard support
- SSL certificate monitoring

PRO PLAN - $10/month:
- 150 services
- 1,500 checks/hour (every 2.4 seconds)
- 90 days data retention
- Custom check intervals (down to 30 seconds)
- Webhook notifications
- Maintenance scheduling
- Advanced security features
- Configuration generation tools
- Priority support
- API access

ENTERPRISE PLAN - $20/month:
- 500 services
- 5,000 checks/hour (every 0.72 seconds)
- 1 year data retention
- Real-time monitoring (10-second intervals)
- Advanced automation
- White-label status pages
- LDAP/SSO integration
- Advanced reporting
- Priority support with SLA
- Full API access + webhooks

MULTI-PROVIDER SUPPORT:
- Stripe (primary)
- PayPal
- Square
- User selects preferred provider
- Admin must supply working API credentials
- All providers can be enabled simultaneously

TAX SYSTEM:
- EU VIES VAT ID lookup and validation
- VATComply for VAT validation and rate lookup
- OpenTax for US/CA tax calculations
- CloudRates for fallback/other countries
- B2B reverse-charge support
- Tax-exempt profiles
- All invoices show detailed tax breakdowns

INVOICING:
- One invoice per billing event
- Statuses: draft, pending, paid, failed, canceled
- PDF, HTML, JSON formats available
- Line items, discounts, overages, tax included
- Payment method and cycle information
- Automatic invoice generation even if provider down

BILLING CYCLES:
- Monthly (default)
- Annually (10% discount)
- Biannually (15% discount)
- Triennially (20% discount)
- Proration on plan changes
- Trial periods per plan
- Grace periods on payment failures (7 days default)

PAYMENT FAILURE HANDLING:
- Provider unavailable: Wait 24h before user notification
- Notify with provider status and billing update encouragement
- Retry every 3 hours
- Auto-subscription detection with cancel page links
- If not restored in 2 months: Account suspension
- Suspended users retain: Login, invoice access, billing updates, downgrades
- Public resources remain accessible with guest rate limits

USAGE TRACKING:
- Per-user, per-resource metering
- Types: included usage, overage usage
- Triggers: Alerts, auto-upgrades, overage billing
- Enforcement: Hard limits, soft limits, grace limits
- Real-time usage monitoring and notifications

=== THEMES AND UI DESIGN SYSTEM ===
DEFAULT THEME - DRACULA:
- Background: #282a36 (dark slate)
- Primary: #bd93f9 (purple)
- Secondary: #8be9fd (cyan)
- Accent: #50fa7b (green)
- Text: #f8f8f2 (off-white)
- Error: #ff5555 (red)
- Warning: #f1fa8c (yellow)
- Success: #50fa7b (green)

ADDITIONAL THEMES:
- Light theme (high contrast, WCAG AA compliant)
- Dark theme (alternative dark option)
- Admin can upload custom themes
- Theme components: colors, fonts, spacing, component styling

TYPOGRAPHY:
- Primary Font: Inter (web font)
- Monospace Font: JetBrains Mono (for code, logs, technical data)
- Font weights: 400 (regular), 500 (medium), 600 (semibold), 700 (bold)
- Responsive font sizes with fluid scaling

MOBILE DESIGN REQUIREMENTS:
- Mobile-first responsive design approach
- Touch-friendly 44px minimum touch targets
- Responsive grid system (CSS Grid + Flexbox)
- Bottom navigation for mobile devices
- Gesture support for card interactions
- Optimized performance on mobile devices

COMPONENT SYSTEM:
- Uniform 280px × 160px service cards
- Consistent spacing and typography throughout
- Status color coding (green/red/yellow borders)
- Smooth animations and hover effects
- Loading states and skeleton screens
- Consistent form styling and validation

ACCESSIBILITY REQUIREMENTS:
- WCAG 2.1 AA compliance
- Keyboard navigation support
- Screen reader compatibility
- High contrast mode options
- Semantic HTML structure
- Proper ARIA labels and roles

DESIGN CONSISTENCY:
- Every single page uses identical design system
- Same header/navigation structure throughout
- Same footer system across all pages
- Same component library and styling
- No page exceptions, no standalone designs
- Uniform experience from login to admin panels

=== FOOTER SYSTEM ===
DEFAULT FOOTER LAYOUT:
- Three default elements: Execution Time | Powered by CasDash | Version
- Centered on page as a unit
- Evenly spaced with | separators
- Example: "Generated in 0.234ms | Powered by CasDash | v1.2.3"

ADMIN FOOTER MANAGEMENT:
- Drag and drop footer builder interface
- Available elements: Execution Time, Powered By, Version, Custom Text, Links, Timestamp, Custom HTML
- Admin can drag elements from sidebar to footer
- Reorder elements by dragging
- Remove elements (drag off or delete button)
- Enable/disable individual elements toggle
- Edit content within customizable elements
- Real-time preview of changes

FOOTER BUILDER FEATURES:
- Live preview of footer layout
- Proper spacing and styling preview
- Responsive design preview (desktop/tablet/mobile)
- Apply/cancel changes functionality
- Reset to defaults option
- Save configurations to database

ELEMENT MANAGEMENT:
```
ELEMENT MANAGEMENT:
Each footer element includes:
- Toggle on/off switch
- Edit content button (for customizable elements)
- Drag handle for reordering
- Delete button (for removable elements)
- Preview of how element appears
- Element-specific settings and configuration

FOOTER ELEMENT TYPES:
- Execution Time: "Generated in X ms" (auto-populated)
- Powered By: "Powered by CasDash" (customizable text)
- Version: "v1.2.3" (auto-populated from build)
- Custom Text: User-defined static content
- Links: Admin-created clickable links
- Timestamp: Current date/time (various formats)
- Custom HTML: Advanced users can add HTML content

FOOTER CONSISTENCY:
- Same footer appears on every single page
- Identical styling and spacing throughout application
- Changes apply globally across entire application
- Mobile responsive footer layout
- Professional appearance maintained

=== CONFIGURATION GENERATION SYSTEM ===
REVERSE PROXY SUPPORT:
Supported Web Servers:
- Nginx (default, most comprehensive templates)
- Apache HTTP Server
- Caddy Server
- Traefik
- HAProxy

GENERATED CONFIGURATIONS INCLUDE:
- Complete virtual host configurations
- SSL/TLS setup with modern cipher suites
- Security headers (HSTS, CSP, X-Frame-Options, etc.)
- Gzip compression configuration
- Rate limiting rules
- Proxy pass configuration to CasDash
- HTTP to HTTPS redirect rules
- Custom error pages
- Access logging configuration

SSL PROVIDERS INTEGRATION:
- Certbot/Let's Encrypt (default, auto-renewal scripts)
- ACME.sh integration
- Manual certificate support
- Self-signed certificate generation
- Commercial certificate integration

TEMPLATE SYSTEM:
- Global templates with service-specific injections
- Security-optimized default configurations
- Performance tuning included
- Service-specific optimizations based on service type
- Custom template support for advanced users

GENERATED OUTPUTS:
- Complete configuration files ready for deployment
- Setup scripts for automated installation
- Step-by-step installation instructions
- Testing commands for validation
- Troubleshooting guides
- Update and maintenance procedures

FIREWALL INTEGRATION:
- UFW (Ubuntu Firewall) rule generation
- iptables rule generation
- pfSense configuration generation
- Custom firewall rule templates
- Service-specific port and protocol rules

=== NETWORK DISCOVERY SYSTEM ===
DISCOVERY ENGINE:
- Privileged background process for network access
- Automatic startup scan on first run
- Scheduled rescans (configurable interval, default 24 hours)
- Manual discovery triggers from admin interface

DISCOVERY METHODS:
- Network port scanning (configurable port ranges)
- HTTP service fingerprinting and banner detection
- Container platform API integration (Docker, Kubernetes)
- mDNS/Bonjour service discovery
- SNMP device detection and identification
- Cloud provider API integration (AWS, Azure, GCP)

DISCOVERY SCOPE:
- Configurable network ranges (CIDR notation support)
- Common service ports: 22, 53, 80, 443, 3000, 5432, 8080, 8443, 9000
- Extended port ranges for comprehensive discovery
- Exclude lists for known non-service ports
- Container platform service discovery

SERVICE RECOGNITION:
- 600+ service fingerprints for automatic identification
- HTTP response analysis for service type detection
- Port-based service identification
- Banner analysis and pattern matching
- Version detection where possible
- Confidence scoring (1-100) for each detection

DISCOVERY RESULTS:
- Automatic service suggestions with one-click enablement
- Confidence scoring for each discovered service
- Duplicate detection and consolidation
- Integration with service templates for optimal configuration
- Discovery history and change tracking

DISCOVERY CONFIGURATION:
- Network ranges: Auto-detect local networks or manual CIDR
- Port lists: Configurable port ranges to scan
- Scan intervals: From hourly to weekly
- Scan intensity: Light, normal, comprehensive
- Exclusion lists: Skip known infrastructure

=== DATABASE SCHEMA ===
DATABASE ABSTRACTION:
- Support for SQLite (default), PostgreSQL, MySQL, MariaDB
- Database migration system with version tracking
- Connection pooling and optimization
- Automatic backup scheduling
- Data retention policies

CORE TABLES:

users table:
- id (PRIMARY KEY, AUTO_INCREMENT)
- email (VARCHAR(255), UNIQUE, NOT NULL)
- password_hash (VARCHAR(255), NOT NULL) -- bcrypt with cost 12
- name (VARCHAR(255), NOT NULL)
- role (ENUM: admin, user, support, view_only)
- created_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP)
- updated_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)
- last_login (TIMESTAMP, NULL)
- settings_json (TEXT) -- JSON for user preferences
- custom_domain (VARCHAR(255), NULL) -- SaaS mode only
- totp_secret (VARCHAR(255), NULL) -- 2FA secret, encrypted
- api_key (VARCHAR(255), NULL) -- API access key
- organization_id (INTEGER, FOREIGN KEY, NOT NULL)

organizations table:
- id (PRIMARY KEY, AUTO_INCREMENT)
- name (VARCHAR(255), NOT NULL)
- created_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP)
- settings_json (TEXT) -- JSON for org settings

services table:
- id (PRIMARY KEY, AUTO_INCREMENT)
- name (VARCHAR(255), NOT NULL)
- description (TEXT)
- url (VARCHAR(500), NOT NULL)
- service_type (VARCHAR(100), NOT NULL)
- category (VARCHAR(100))
- icon_type (ENUM: library, upload, url, favicon)
- icon_value (VARCHAR(500))
- auth_type (ENUM: none, basic, bearer, api_key, oauth2, custom_headers)
- auth_credentials_encrypted (TEXT) -- AES-256 encrypted JSON
- custom_headers (TEXT) -- JSON
- monitoring_enabled (BOOLEAN, DEFAULT TRUE)
- check_interval (INTEGER, DEFAULT 300) -- seconds
- timeout (INTEGER, DEFAULT 30) -- seconds
- expected_status_codes (TEXT) -- JSON array
- expected_content (TEXT)
- follow_redirects (BOOLEAN, DEFAULT TRUE)
- ssl_verify (BOOLEAN, DEFAULT TRUE)
- public_visible (BOOLEAN, DEFAULT FALSE)
- public_name (VARCHAR(255))
- public_description (TEXT)
- maintenance_mode (BOOLEAN, DEFAULT FALSE)
- position_x (INTEGER, DEFAULT 0)
- position_y (INTEGER, DEFAULT 0)
- card_size (ENUM: small, medium, large, DEFAULT medium)
- created_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP)
- updated_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)
- created_by (INTEGER, FOREIGN KEY REFERENCES users(id))
- user_id (INTEGER, FOREIGN KEY REFERENCES users(id)) -- SaaS mode ownership
- organization_id (INTEGER, FOREIGN KEY REFERENCES organizations(id))

service_checks table:
- id (PRIMARY KEY, AUTO_INCREMENT)
- service_id (INTEGER, FOREIGN KEY REFERENCES services(id), NOT NULL)
- status (ENUM: up, down, warning, unknown)
- response_time (INTEGER) -- milliseconds
- status_code (INTEGER)
- error_message (TEXT)
- check_time (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP)
- ssl_expiry_date (TIMESTAMP, NULL)
- ssl_issuer (VARCHAR(255))
- ssl_subject (VARCHAR(255))

issues table:
- id (PRIMARY KEY, AUTO_INCREMENT)
- service_id (INTEGER, FOREIGN KEY REFERENCES services(id))
- title (VARCHAR(255), NOT NULL)
- description (TEXT)
- issue_type (ENUM: bug, performance, security, maintenance, feature_request)
- priority (ENUM: critical, high, medium, low)
- status (ENUM: open, in_progress, resolved, closed)
- created_by (INTEGER, FOREIGN KEY REFERENCES users(id))
- assigned_to (INTEGER, FOREIGN KEY REFERENCES users(id), NULL)
- created_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP)
- updated_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)
- closed_at (TIMESTAMP, NULL)
- auto_generated (BOOLEAN, DEFAULT FALSE)
- source_type (VARCHAR(100)) -- monitoring, security_scan, manual

support_tickets table:
- id (PRIMARY KEY, AUTO_INCREMENT)
- user_id (INTEGER, FOREIGN KEY REFERENCES users(id), NOT NULL)
- service_id (INTEGER, FOREIGN KEY REFERENCES services(id), NULL)
- subject (VARCHAR(255), NOT NULL)
- description (TEXT, NOT NULL)
- priority (ENUM: critical, high, medium, low)
- status (ENUM: open, in_progress, resolved, closed)
- assigned_to (INTEGER, FOREIGN KEY REFERENCES users(id), NULL)
- created_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP)
- updated_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)
- closed_at (TIMESTAMP, NULL)
- satisfaction_rating (INTEGER) -- 1-5 scale

maintenance_windows table:
- id (PRIMARY KEY, AUTO_INCREMENT)
- title (VARCHAR(255), NOT NULL)
- description (TEXT)
- start_time (TIMESTAMP, NOT NULL)
- end_time (TIMESTAMP, NOT NULL)
- maintenance_type (ENUM: service, group, platform, network, system)
- affected_services (TEXT) -- JSON array of service IDs
- created_by (INTEGER, FOREIGN KEY REFERENCES users(id), NOT NULL)
- status (ENUM: scheduled, in_progress, completed, cancelled)
- actual_start (TIMESTAMP, NULL)
- actual_end (TIMESTAMP, NULL)
- created_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP)

notifications table:
- id (PRIMARY KEY, AUTO_INCREMENT)
- user_id (INTEGER, FOREIGN KEY REFERENCES users(id), NOT NULL)
- type (VARCHAR(100), NOT NULL) -- alert, info, warning, error
- title (VARCHAR(255), NOT NULL)
- message (TEXT, NOT NULL)
- read_at (TIMESTAMP, NULL)
- created_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP)
- related_id (INTEGER, NULL) -- ID of related object
- related_type (VARCHAR(100), NULL) -- Type of related object

system_settings table:
- key (VARCHAR(255), PRIMARY KEY)
- value (TEXT, NOT NULL)
- updated_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)
- updated_by (INTEGER, FOREIGN KEY REFERENCES users(id), NULL)

discovery_results table:
- id (PRIMARY KEY, AUTO_INCREMENT)
- ip_address (VARCHAR(45), NOT NULL) -- IPv4/IPv6 support
- hostname (VARCHAR(255))
- ports_open (TEXT) -- JSON array
- service_type (VARCHAR(100))
- confidence (INTEGER) -- 1-100 scale
- discovered_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP)
- enabled (BOOLEAN, DEFAULT FALSE)

user_permissions table (Enterprise mode):
- id (PRIMARY KEY, AUTO_INCREMENT)
- user_id (INTEGER, FOREIGN KEY REFERENCES users(id), NOT NULL)
- service_id (INTEGER, FOREIGN KEY REFERENCES services(id), NOT NULL)
- permission_level (ENUM: read, write, admin)
- granted_by (INTEGER, FOREIGN KEY REFERENCES users(id), NOT NULL)
- created_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP)

billing_plans table (SaaS mode):
- id (PRIMARY KEY, AUTO_INCREMENT)
- name (VARCHAR(255), NOT NULL)
- price (DECIMAL(10,2), NOT NULL)
- currency (VARCHAR(3), DEFAULT 'USD')
- billing_cycle (ENUM: monthly, annually, biannually, triennially)
- features (TEXT) -- JSON with limits and capabilities
- trial_days (INTEGER, DEFAULT 0)
- grace_period_days (INTEGER, DEFAULT 7)
- active (BOOLEAN, DEFAULT TRUE)
- created_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP)

user_subscriptions table (SaaS mode):
- id (PRIMARY KEY, AUTO_INCREMENT)
- user_id (INTEGER, FOREIGN KEY REFERENCES users(id), NOT NULL)
- plan_id (INTEGER, FOREIGN KEY REFERENCES billing_plans(id), NOT NULL)
- status (ENUM: active, cancelled, expired, suspended)
- started_at (TIMESTAMP, NOT NULL)
- expires_at (TIMESTAMP, NOT NULL)
- auto_renew (BOOLEAN, DEFAULT TRUE)
- payment_provider (VARCHAR(100))
- external_subscription_id (VARCHAR(255))

=== API SPECIFICATION ===
API ARCHITECTURE:
- RESTful API design with consistent endpoints
- JSON request/response format
- HTTP status codes for all responses
- Comprehensive error handling with detailed messages
- Rate limiting per user and per IP
- API versioning support (/api/v1/)

AUTHENTICATION:
- JWT tokens for session-based authentication
- API keys for programmatic access
- Session-based authentication for web interface
- Role-based authorization for all endpoints
- Rate limiting: 1000 requests/hour per user, 100/hour per IP for unauthenticated

API ENDPOINTS:

Service Management:
GET /api/v1/services - List all services (scoped by mode)
POST /api/v1/services - Create new service
GET /api/v1/services/{id} - Get service details
PUT /api/v1/services/{id} - Update service
DELETE /api/v1/services/{id} - Delete service
POST /api/v1/services/{id}/check - Force health check
PUT /api/v1/services/{id}/public - Toggle public visibility

Monitoring:
GET /api/v1/monitoring/status - Current status of all services
GET /api/v1/monitoring/uptime/{id} - Uptime data for specific service
GET /api/v1/monitoring/history/{id} - Historical monitoring data
GET /api/v1/monitoring/performance/{id} - Performance metrics

Security:
GET /api/v1/security/scan/{id} - Run security assessment
GET /api/v1/security/recommendations/{id} - Get security recommendations
GET /api/v1/security/score/{id} - Get security score

Issues:
GET /api/v1/issues - List all issues (scoped by mode)
POST /api/v1/issues - Create new issue
GET /api/v1/issues/{id} - Get issue details
PUT /api/v1/issues/{id} - Update issue
DELETE /api/v1/issues/{id} - Delete issue

Support:
GET /api/v1/support/tickets - List support tickets
POST /api/v1/support/tickets - Create support ticket
GET /api/v1/support/tickets/{id} - Get ticket details
PUT /api/v1/support/tickets/{id} - Update ticket

User Management:
GET /api/v1/users - List users (admin only)
POST /api/v1/users - Create user (admin only)
GET /api/v1/users/{id} - Get user details
PUT /api/v1/users/{id} - Update user
DELETE /api/v1/users/{id} - Delete user (admin only)

System:
GET /api/v1/health - System health check
GET /api/v1/status - Public system status (no auth required)
GET /api/v1/settings - Get system settings (admin only)
PUT /api/v1/settings - Update system settings (admin only)

Discovery:
GET /api/v1/discovery/results - Get discovery results
POST /api/v1/discovery/scan - Trigger discovery scan
PUT /api/v1/discovery/enable/{id} - Enable discovered service

Billing (SaaS mode):
GET /api/v1/billing/plans - List available plans
GET /api/v1/billing/subscription - Get user subscription
PUT /api/v1/billing/subscription - Update subscription
GET /api/v1/billing/invoices - List user invoices
GET /api/v1/billing/usage - Get usage statistics

ERROR HANDLING:
- Consistent JSON error responses
- HTTP status codes: 200 (success), 400 (bad request), 401 (unauthorized), 403 (forbidden), 404 (not found), 500 (server error)
- Detailed error messages with field-specific validation errors
- Error codes for programmatic handling

RATE LIMITING:
- Per-user limits: 1000 requests/hour for authenticated users
- Per-IP limits: 100 requests/hour for unauthenticated requests
- API key exemptions: Higher limits for API key authentication
- Configurable thresholds via admin interface

PAGINATION:
- Standard pagination for list endpoints
- Query parameters: page, per_page, sort, order
- Response includes: total, page, per_page, total_pages
- Maximum per_page: 100 items

=== WEBSOCKET INTEGRATION ===
REAL-TIME FEATURES:
- Service status updates pushed to connected clients
- Live chat communication
- Real-time notifications
- Dashboard updates without page refresh
- Monitoring data streaming

WEBSOCKET ENDPOINTS:
/ws/status - Real-time service status updates
/ws/chat - Live support chat communication
/ws/notifications - Real-time notification delivery
/ws/monitoring - Live monitoring data streams

CONNECTION MANAGEMENT:
- Automatic reconnection on connection loss
- Connection authentication and authorization
- Per-user connection limits
- Graceful degradation when WebSocket unavailable

=== SECURITY REQUIREMENTS ===
AUTHENTICATION SECURITY:
- Bcrypt password hashing with cost factor 12
- Secure session management with HttpOnly cookies
- CSRF protection on all state-changing operations
- Rate limiting on authentication endpoints (5 attempts per 15 minutes)
- 2FA support via TOTP with QR code generation
- Session timeout configuration (default 24 hours)

AUTHORIZATION:
- Role-based access control (RBAC)
- Resource-level permissions
- API key authentication with scoped permissions
- Service ownership validation (SaaS mode)
- Organization membership validation (Enterprise mode)

DATA PROTECTION:
- AES-256 encryption for sensitive data (passwords, API keys, credentials)
- Secure credential storage with key derivation
- Database encryption at rest option
- Audit logging for all administrative actions
- Personal data handling compliance (GDPR-ready)

NETWORK SECURITY:
- HTTPS enforcement in production with HSTS headers
- Security headers: CSP, X-Frame-Options, X-Content-Type-Options
- Input validation and sanitization on all endpoints
- SQL injection prevention with parameterized queries
- XSS prevention with output encoding

CONTAINER SECURITY:
- Non-root user execution inside containers
- Minimal container images with no unnecessary packages
- Security scanning integration for container images
- Least privilege principle for container permissions
- Read-only root filesystem where possible

=== BUILD AND DEPLOYMENT ===
BUILD SYSTEM:
- Go modules for dependency management
- Embedded assets using Go 1.16+ embed package
- Cross-compilation for multiple platforms
- Automated build pipeline with GitHub Actions
- Version embedding from Git tags

DEPENDENCIES:
- Minimal external dependencies
- No runtime dependencies except chosen database
- All web assets embedded in binary
- Static linking for maximum portability

INSTALLATION METHODS:
- Single binary download (primary method)
- Docker containers with multi-arch support
- System package managers (future: .deb, .rpm, .pkg)
- System service integration (systemd, launchd, Windows service)

STARTUP PROCESS:
1. Parse command line arguments and environment variables
2. Initialize database connection and run migrations
3. Start network discovery service (if enabled and privileged)
4. Drop privileges after discovery service startup
5. Start web server on selected port
6. Begin monitoring loops for all enabled services

DOCKER DEPLOYMENT:
- Multi-stage build for minimal image size
- Multi-arch images (linux/amd64, linux/arm64)
- Non-root user execution
- Volume mounting for data persistence
- Health checks included
- Docker Compose examples provided

=== TESTING REQUIREMENTS ===
UNIT TESTS:
- Minimum 80% code coverage requirement
- All business logic functions tested
- Database abstraction layer tested with SQLite
- Service monitoring logic tested with mock services
- Authentication and authorization logic tested

INTEGRATION TESTS:
- Full API endpoint testing with real database
- Database migration testing across all supported databases
- Authentication flow testing (login, logout, 2FA)
- Service discovery integration testing
- WebSocket connection testing

END-TO-END TESTS:
- Complete user workflow testing (registration to service monitoring)
- Cross-browser compatibility testing (Chrome, Firefox, Safari, Edge)
- Mobile device testing (iOS Safari, Android Chrome)
- Performance testing under load

PERFORMANCE TESTS:
- Load testing for monitoring at scale (1000+ services)
- Database performance testing with large datasets
- Memory usage validation and leak detection
- WebSocket performance under concurrent connections

=== DOCUMENTATION REQUIREMENTS ===
README.md:
- Project overview with feature highlights
- Quick start guide (download, run, access)
- Installation instructions for all platforms
- Basic configuration examples
- Link to comprehensive documentation

INSTALLATION GUIDE:
- Detailed setup for Linux, macOS, Windows
- Docker deployment instructions
- Database setup for all supported types
- Reverse proxy configuration examples
- SSL certificate setup guide
- System service installation

CONFIGURATION GUIDE:
- Complete environment variable reference
- Database configuration for all supported types
- SSL/TLS configuration options
- Network discovery configuration
- Performance tuning recommendations

API DOCUMENTATION:
- Complete endpoint documentation with examples
- Authentication methods and examples
- Error response documentation
- Rate limiting information
- SDK information and examples

USER GUIDE:
- Dashboard usage and navigation
- Service configuration and management
- Security assessment and hardening
- Maintenance window scheduling
- Support ticket and chat usage

DEVELOPER GUIDE:
- Architecture overview and design decisions
- Development environment setup
- Contributing guidelines and code standards
- Testing procedures and requirements
- Build and release process

SECURITY GUIDE:
- Security best practices for deployment
- Hardening recommendations for production
- Compliance information (GDPR, SOC 2 preparation)
- Incident response procedures
- Regular security maintenance tasks

=== MAKEFILE TARGETS ===
Required Makefile with these exact targets:

build: Build for current platform
	go build -ldflags="-X main.Version=$(shell git describe --tags --always)" -o casdash

build-all: Cross-compile for all platforms
	GOOS=linux GOARCH=amd64 go build -ldflags="-X main.Version=$(shell git describe --tags --always)" -o dist/casdash-linux-amd64
	GOOS=linux GOARCH=arm64 go build -ldflags="-X main.Version=$(shell git describe --tags --always)" -o dist/casdash-linux-arm64
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.Version=$(shell git describe --tags --always)" -o dist/casdash-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.Version=$(shell git describe --tags --always)" -o dist/casdash-darwin-arm64
	GOOS=windows GOARCH=amd64 go build -ldflags="-X main.Version=$(shell git describe --tags --always)" -o dist/casdash-windows-amd64.exe

test: Run all tests
	go test -v ./...

test-coverage: Generate coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

docker: Build Docker image
	docker build -t casdash:latest .

clean: Clean build artifacts
	rm -rf dist/
	rm -f casdash
	rm -f coverage.out coverage.html

deps: Install dependencies
	go mod download
	go mod tidy

lint: Run code linting
	golangci-lint run

fmt: Format code
	go fmt ./...
	goimports -w .

dev: Start development server with hot reload
	air -c .air.toml

install: Install binary to system
	go install

=== DEVELOPMENT SETUP ===
PREREQUISITES:
- Go 1.21 or higher
- Node.js 18+ for frontend build tools
- Docker for testing and deployment
- Make for build automation
- Git for version control

DEVELOPMENT DATABASE:
- SQLite for development (no setup required)
- PostgreSQL for testing (Docker container recommended)
- MySQL/MariaDB for compatibility testing

FRONTEND BUILD:
- Modern JavaScript (ES2022)
- CSS with modern features (Grid, Flexbox, Custom Properties)
- No heavy frameworks - vanilla JS with Web Components
- Build system: esbuild for speed
- Hot reload during development

CODE QUALITY:
- gofmt for code formatting
- golangci-lint for comprehensive linting
- gosec for security scanning
- go vet for static analysis
- Pre-commit hooks for code quality

GIT HOOKS:
- Pre-commit: Run fmt, lint, and basic tests
- Pre-push: Run full test suite
- Commit message validation

=== ERROR HANDLING ===
ERROR CATEGORIES:
- User errors (4xx): Invalid input, authentication failures, permission denied
- System errors (5xx): Database failures, external service unavailable, configuration errors
- Validation errors: Field-specific validation with detailed messages
- Authentication errors: Invalid credentials, expired sessions, insufficient permissions

ERROR RESPONSES:
- Consistent JSON format for API errors
- HTML error pages for web interface with same design system
- Descriptive error messages suitable for end users
- Error codes for programmatic handling
- Stack traces in debug mode only

LOGGING:
- Structured logging with configurable levels (debug, info, warn, error)
- Audit trail for security events and administrative actions
- Performance metrics logging (response times, error rates)
- Log rotation and retention policies
- Integration with external logging systems

RECOVERY:
- Graceful degradation when external services unavailable
- Automatic retry logic with exponential backoff
- Circuit breaker patterns for external service calls
- Health checks with automatic service recovery
- Database connection pooling with automatic reconnection

=== PERFORMANCE REQUIREMENTS ===
RESPONSE TIME TARGETS:
- Dashboard loads: <100ms for authenticated users
- API calls: <500ms for simple operations, <2s for complex operations
- Discovery scans: <2s for small networks, <30s for comprehensive scans
- Database queries: <50ms for simple queries, <500ms for complex analytics

SCALABILITY:
- Support 1000+ services per instance
- Support 100+ concurrent users
- Minimal memory footprint (<512MB base usage)
- Efficient database queries with proper indexing
- Connection pooling for database and external services

CACHING:
- In-memory caching for frequently accessed data
- Redis support for distributed caching (optional)
- Efficient WebSocket handling with connection pooling
- Static asset caching with proper ETags
- Database query result caching where appropriate

MONITORING PERFORMANCE:
- Configurable check intervals per service (30 seconds to 24 hours)
- Parallel monitoring execution with worker pools
- Efficient dependency checking to prevent cascade failures
- Smart scheduling to distribute load evenly
- Performance metrics collection and analysis

=== COMPLIANCE FEATURES ===
AUDIT LOGGING:
- All user actions logged with timestamps and user identification
- Configuration changes tracked with before/after values
- Access attempts recorded (successful and failed)
- Data retention according to compliance requirements
- Tamper-evident logging with integrity checks

DATA RETENTION:
- Configurable retention policies for all data types
- Automatic data archival after retention period
- Secure data deletion with verification
- Backup retention separate from operational data
- Legal hold capabilities for compliance investigations

ACCESS CONTROL:
- Role-based permissions with principle of least privilege
- Audit trail for all permission changes
- Administrative oversight with approval workflows
- Regular access reviews and reporting
- Integration with enterprise identity systems

SECURITY STANDARDS:
- OWASP Top 10 compliance with regular assessments
- Security header implementation (HSTS, CSP, etc.)
- Regular vulnerability scanning integration
- Penetration testing support and documentation
- Security incident response procedures

GDPR COMPLIANCE:
- Personal data identification and classification
- Data subject rights implementation (access, rectification, erasure)
- Privacy by design principles
- Data processing agreements templates
- Breach notification procedures

=== BACKUP AND RECOVERY ===
BACKUP FEATURES:
- Automatic database backups on configurable schedule
- Full system configuration export
- Service configuration backup with encryption
- Incremental backups for large datasets
- Cross-platform backup compatibility

RECOVERY OPTIONS:
- Point-in-time recovery for databases
- Configuration restore from backup files
- Disaster recovery procedures and documentation
- Database migration tools for platform changes
- Service configuration import/export

BACKUP STORAGE:
- Local filesystem backup storage
- Cloud storage integration (S3, Google Cloud, Azure)
- Encrypted backup files with key management
- Backup verification and integrity checking
- Automated backup retention management

=== SELF-MONITORING ===
HEALTH ENDPOINTS:
- /api/health for comprehensive system status
- Component health checks (database, discovery service, monitoring)
- Dependency validation (external services, DNS resolution)
- Resource usage monitoring (CPU, memory, disk)
- Performance metrics collection

INTERNAL METRICS:
- Application performance monitoring
- Database query performance tracking
- Error rate monitoring and alerting
- Resource usage trending
- User activity analytics

SELF-SERVICE MONITORING:
- CasDash monitors itself as a regular service
- Appears in service list with system icon
- Follows same monitoring rules as user services
- Self-healing capabilities where possible
- Administrative alerts for system issues

=== PROJECT STRUCTURE ===
Required directory structure:

casdash/
├── cmd/
│   └── casdash/
│       └── main.go
├── internal/
│   ├── api/
│   ├── auth/
│   ├── config/
│   ├── database/
│   ├── discovery/
│   ├── monitoring/
│   ├── security/
│   ├── billing/
│   └── web/
├── pkg/
│   ├── models/
│   ├── services/
│   └── utils/
├── web/
│   ├── static/
│   ├── templates/
│   └── assets/
├── migrations/
├── docs/
├── scripts/
├── docker/
├── .github/
│   └── workflows/
├── Makefile
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── README.md
├── LICENSE
├── .gitignore
└── .air.toml

=== FINAL IMPLEMENTATION NOTES ===
This specification is COMPLETE and EXHAUSTIVE. Any AI assistant implementing this project must:

1. Create a fully functional, production-ready application
2. Include ALL specified features and functionality
3. Follow ALL architectural requirements and constraints
4. Implement proper error handling and security measures
5. Create comprehensive documentation
6. Ensure the application compiles and runs on first attempt
7. Include all build artifacts (Makefile, Dockerfile, etc.)
8. Follow Go best practices and coding standards
9. Include proper testing coverage
10. Create a professional, production-ready solution

The resulting application should be immediately deployable in production environments and compete with commercial monitoring solutions while maintaining the simplicity and reliability expected from self-hosted software.

NO ADDITIONAL CLARIFICATION SHOULD BE REQUIRED. This specification contains every detail necessary for complete implementation.
```

