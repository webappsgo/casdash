# CasDash

> The ultimate self-hosted service dashboard combining beautiful homepage functionality with comprehensive monitoring, container management, and security features.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)](https://docker.com)

## 🌟 Features

- **🏠 Beautiful Dashboard**: Homer/Dashy-style homepage with Dracula theme
- **📊 Comprehensive Monitoring**: Uptime Kuma-level monitoring capabilities
- **🔄 Update Management**: Watchtower-style container update automation
- **🔒 Enterprise Security**: Built-in vulnerability scanning and SSL monitoring
- **🔧 Service Discovery**: Automatic network scanning and container detection
- **📱 Mobile First**: Responsive design optimized for all devices
- **⚡ Single Binary**: Zero dependencies, embedded web assets
- **🌐 Multi-Mode**: Enterprise (self-hosted) and SaaS deployment options

## 🚀 Quick Start

### Using Docker (Recommended)

```bash
# Run with Docker
docker run -d \
  --name casdash \
  -p 64321:64321 \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v ./data:/data \
  ghcr.io/casapps/casdash:latest

# Or use Docker Compose
docker-compose up -d
```

### Development Setup

```bash
# Clone the repository
git clone https://github.com/casapps/casdash.git
cd casdash

# Build and run
make build
./casdash

# Or use Docker for development
make docker-dev
```

## 📖 Configuration

CasDash is designed with **zero configuration** by default. On first run, it will:

1. 🔍 Auto-detect available port (64000-65535 range)
2. 🗄️ Initialize SQLite database
3. 🔍 Discover services on your network
4. 📊 Start monitoring automatically

### Environment Variables (First Run Only)

After initial setup, all configuration is managed through the web interface.

```bash
# System Configuration
CASDASH_MODE=enterprise          # enterprise|saas
CASDASH_DB_TYPE=sqlite          # sqlite|postgres|mysql|mariadb
CASDASH_DB_PATH=./casdash.db    # SQLite database path

# Discovery Configuration
CASDASH_DISCOVERY_ENABLED=true
CASDASH_DISCOVERY_NETWORKS=auto-detect
CASDASH_DISCOVERY_PORTS=22,53,80,443,3000,5432,8080,8443,9000

# Security
CASDASH_SECRET_KEY=auto-generated
CASDASH_REGISTRATION=disabled
```

## 🏗️ Architecture

### Operating Modes

#### Enterprise Mode (Default)
- **Target**: Self-hosters, families, teams, enterprises
- **Users**: Internal (employees, family members, team members)
- **Services**: Organization-scoped with role-based permissions
- **Registration**: Admin-controlled (invite/LDAP/domain whitelist)
- **Billing**: Always disabled
- **Public Dashboard**: `/{username}` shows organization's public services

#### SaaS Mode
- **Target**: External paying customers/subscribers
- **Users**: External customers
- **Services**: User-scoped with personal ownership
- **Registration**: Open signup (configurable)
- **Billing**: Enabled if payment provider configured
- **Public Dashboard**: `/{username}` shows that user's public services

### Technical Stack

- **Backend**: Go 1.21+ with embedded assets
- **Database**: SQLite (default), PostgreSQL, MySQL, MariaDB
- **Frontend**: Vanilla JavaScript, Dracula theme CSS
- **WebSocket**: Real-time updates
- **API**: RESTful with comprehensive endpoints

## 📄 License

CasDash is released under the [MIT License](LICENSE).

## 🛠️ Development Status

**Current Phase**: Core Implementation ✅

### Completed Components

#### Foundation (✅ Complete)
- ✅ Project structure and Go module setup
- ✅ Docker development environment with multi-stage builds
- ✅ Database abstraction layer (SQLite, PostgreSQL, MySQL, MariaDB support)
- ✅ Core database schema (20+ tables, full spec compliance)
- ✅ Configuration system with mode detection (enterprise/saas)
- ✅ Migration system with embedded SQL migrations

#### Backend Infrastructure (✅ Complete)
- ✅ Web server with embedded static assets
- ✅ WebSocket system for real-time updates
- ✅ Service discovery engine with network scanning
- ✅ Monitoring engine with HTTP, SSL, and port checking
- ✅ User and service management models
- ✅ Session management and middleware
- ✅ Notification system with multi-channel support

#### Authentication & Security (✅ Complete)
- ✅ Authentication handlers (login, logout, registration)
- ✅ Role-based access control (RBAC)
- ✅ Session management with secure cookies
- ✅ Password hashing with BCrypt
- ✅ CSRF protection middleware

#### Monitoring & Health Checks (✅ Complete)
- ✅ HTTP/HTTPS health checking
- ✅ SSL certificate monitoring and expiry tracking
- ✅ TCP/UDP port monitoring
- ✅ Real-time status updates via WebSocket
- ✅ Performance metrics collection
- ✅ Service dependency tracking

#### Frontend & UI (✅ Complete)
- ✅ HTML templates with Dracula theme
- ✅ Dashboard interface
- ✅ Service management pages
- ✅ Monitoring views
- ✅ Settings and configuration pages
- ✅ Responsive mobile-first design

#### API Layer (✅ Complete)
- ✅ RESTful API endpoints
- ✅ Service CRUD operations
- ✅ Health check endpoints
- ✅ User management API
- ✅ WebSocket endpoints for real-time data

#### DevOps & Deployment (✅ Complete)
- ✅ Dockerfile with multi-stage build
- ✅ Docker Compose for easy deployment
- ✅ Development Docker Compose with hot reload
- ✅ Health checks and graceful shutdown
- ✅ Makefile for common operations

### Next Phase: Advanced Features
- 🔄 Email notification delivery
- 🔄 Slack/Discord webhook integrations
- 🔄 Container update management (Watchtower functionality)
- 🔄 Security vulnerability scanning
- 🔄 Backup and restore functionality
- 🔄 LDAP/OAuth authentication
- 🔄 Multi-tenant isolation (SaaS mode)
- 🔄 Billing integration (SaaS mode)
- 🔄 Advanced analytics and reporting
- 🔄 Comprehensive testing suite

See [TODO.md](TODO.md) for detailed roadmap.

## Author

🤖 casjay: [Github](https://github.com/casjay) 🤖

---

<div align="center">

**Made with ❤️ by [CasApps](https://github.com/casapps)**

</div>
