# surge-host

A lightweight, version-controlled service for hosting, syncing, and validating Surge configuration and rule files. Built with Go for performance and reliability.

## Overview

**surge-host** provides a centralized, private platform to manage your Surge rule sets and configurations. Unlike static file hosting, it integrates native version control (Git), syntax validation, and a dedicated web dashboard to ensure your network configurations are always correct and accessible.

### Core Pillars

- **🚀 Performance:** Low-latency delivery of raw text files, optimized for Surge's subscription model.
- **🛡️ Validation:** Integrated syntax checker for Surge-specific rules (`RULE-SET`, `IP-CIDR`, etc.) to prevent configuration errors.
- **📜 Versioning:** Built-in Git support tracks every change, enabling instant history viewing and one-click rollbacks.
- **🖥️ Management:** A minimalist web interface featuring drag-and-drop uploads and an online editor with syntax highlighting.

---

## Technical Specifications

| Component | Technology |
| :--- | :--- |
| **Backend** | Go (Golang) 1.22+ |
| **Database** | SQLite (Embedded) |
| **VCS** | Native Git Implementation |
| **Frontend** | Go Templates + Vanilla JS |
| **Deployment** | Docker & Docker Compose |

---

## Quick Start

### Docker Deployment (Recommended)

The simplest way to get started is via Docker Compose:

```bash
# 1. Clone the repository
git clone https://github.com/AsaqeLee/surge-host.git
cd surge-host

# 2. Configure your environment
cp .env.example .env
# Edit .env with your domain and credentials

# 3. Launch the service
docker compose up -d --build
```

### Local Development

Ensure you have Go 1.22+ and Git installed:

```bash
go mod tidy
go run ./cmd/server
```
The server will be available at `http://localhost:8080` by default.

---

## Configuration

Surge-host is configured via environment variables. Key parameters include:

- `SURGE_HOST_DOMAIN`: The domain used to generate raw URLs.
- `SURGE_HOST_ADMIN_PASSWORD`: Enables authentication for the dashboard and API.
- `SURGE_HOST_GIT_ENABLED`: Toggles the internal Git versioning system.
- `SURGE_HOST_ALLOWED_EXTENSIONS`: Restricts file types (Default: `.conf, .list, .txt, .module, .yaml`).

---

## API & Integration

### Raw URL Subscription
In Surge, reference your hosted files using the following format:
```ini
[Rule]
RULE-SET,https://your-domain.com/raw/admin/proxy.list,PROXY
```

### REST API
Comprehensive endpoints for automation:
- `GET /api/files`: List managed files.
- `POST /api/validate`: Dry-run validation of Surge syntax.
- `GET /api/git/log/{path}`: Access full version history.

---

## License

Distributed under the [MIT License](LICENSE). Built by [Asaqe Lee](https://asaqe.org).
