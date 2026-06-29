# surge-host

**Self-hosted Surge rule & config hosting — upload, version, validate, subscribe via Raw URL.**

📖 [简体中文文档](README.zh-CN.md)

```
https://your-domain.com/raw/{user}/{filename}
```

---

## Why surge-host?

Surge's `RULE-SET` and `DOMAIN-SET` need a **stable, plain-text HTTP URL** that returns config content directly — no HTML wrapper, no auth wall on read, no CDN quirks. When you update a rule, Surge should pick it up automatically instead of you manually syncing local files and redistributing links.

**surge-host** is a minimal private host built for that workflow:

- **Raw URL** — `text/plain` output, Surge-ready
- **Web UI** — upload, manage, edit in browser
- **Git versioning** — per-file history, preview, rollback
- **Syntax validation** — catch broken rules before they go live
- **Single-binary + Docker** — deploy on your NAS, VPS, or homelab

---

## vs GitHub Gist / Static Nginx

| | GitHub Gist | Static Nginx | **surge-host** |
|---|---|---|---|
| Raw plain text | ⚠️ Wrapper / rate limits | ✅ Manual setup | ✅ Purpose-built |
| Online edit + validate | ❌ | ❌ | ✅ |
| Version history & rollback | ⚠️ Git history only | ❌ | ✅ Per-file Git |
| Multi-file management | ❌ One gist = one file | ⚠️ Manual dirs | ✅ UI + API |
| Private write, public read | ⚠️ Token / visibility | ⚠️ DIY | ✅ JWT admin, open Raw |
| Surge-specific workflow | ❌ | ❌ | ✅ |

**Gist** works for a single public snippet, but gets painful with multiple rule sets, rollbacks, and syntax checks.

**Static Nginx** can serve files, but editing, versioning, and validation require separate tooling — every change means manual `scp` and reload.

**surge-host** is the dedicated middle ground: **as simple as a static file server for Surge, as capable as a small CMS for your rules.**

---

## Quick Start

### Requirements

- Docker 20.10+ & Docker Compose v2+ **(recommended)**
- Or Go 1.22+ for local development
- A reverse proxy or tunnel for HTTPS (Nginx, Caddy, Cloudflare Tunnel, etc.)

### 1. Configure

```bash
git clone git@github.com:AsaqeLee/surge-host.git
cd surge-host
cp .env.example .env
```

Edit `.env` — **never commit this file**:

```env
# Host port mapping (docker-compose)
CONTAINER_NAME=surge-host
HOST_IP=
PANEL_APP_PORT_HTTP=28080

# Public domain for Raw URL generation
SURGE_HOST_DOMAIN=rules.example.com

# Admin credentials
SURGE_HOST_ADMIN_USER=admin
SURGE_HOST_ADMIN_PASSWORD=change-me-to-a-strong-password

# JWT secret — use a long random string
SURGE_HOST_JWT_SECRET=change-me-to-a-random-secret
```

Generate a secret:

```bash
openssl rand -hex 32
```

### 2. Start

```bash
docker compose up -d --build

curl http://127.0.0.1:28080/healthz
# → {"status":"ok"}
```

> **1Panel users:** `docker-compose.yml` joins `1panel-network` by default. Change or remove the `networks` section if not using 1Panel.

### 3. Expose via HTTPS

Point your domain to the host port (e.g. `28080`). Example with **Cloudflare Tunnel**:

```
Public Hostname : rules.example.com
Service         : http://192.168.1.x:28080
```

Set `SURGE_HOST_DOMAIN` to the public hostname so Raw URLs are generated correctly.

### 4. Connect Surge

1. Open `https://rules.example.com` → **Upload**
2. Copy the Raw URL from the dashboard
3. Reference in your Surge config:

```ini
[Rule]
RULE-SET,https://rules.example.com/raw/admin/rules.list,PROXY
DOMAIN-SET,https://rules.example.com/raw/admin/domains.list,PROXY
FINAL,DIRECT
```

Surge polls the URL periodically — **no manual sync needed after updates.**

---

## Daily Workflow

### Web UI

| Path | Description |
|------|-------------|
| `/` | Public file list + quick start |
| `/upload` | Drag-and-drop upload |
| `/files` | List, rename, delete, history |
| `/edit/{path}` | Online editor + syntax highlight + validate |

Log in with admin credentials when auth is enabled. Token is stored in `localStorage`.

### Raw URL

```
GET /raw/{user}/{path...}
```

| Property | Value |
|----------|-------|
| Content-Type | `text/plain; charset=utf-8` |
| Auth | None (public read-only) |
| Subdirs | Supported |
| Security | Path traversal blocked, extension whitelist |

### Version Control

Each user gets an isolated Git bare repo. Every upload, edit, rename, or delete auto-commits.

**UI:** Files page → **History** → preview or rollback

**API:**

```bash
# List commits
curl -H "Authorization: Bearer $TOKEN" \
  https://rules.example.com/api/git/log/rules.list

# View a version
curl -H "Authorization: Bearer $TOKEN" \
  "https://rules.example.com/api/git/show/rules.list?commit=abc1234"

# Rollback
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"commit":"abc1234"}' \
  https://rules.example.com/api/git/restore/rules.list
```

### Syntax Validation

Validates `.list`, `.conf`, `.module` files:

- Comma-separated rule format
- Section headers in `.conf` (`[Rule]`, `[General]`, …)
- Common rule types: `DOMAIN-SUFFIX`, `RULE-SET`, `GEOIP`, `IP-CIDR`, `FINAL`, …
- **Strict mode** (`SURGE_HOST_VALIDATE_STRICT=true`) treats unknown types as errors

Validate in the editor before save, or via API:

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"rules.list","content":"DOMAIN-SUFFIX,example.com,PROXY"}' \
  https://rules.example.com/api/validate
```

Failed upload/save returns HTTP `422` with issue details.

---

## Deployment Reference

### Docker Compose

```bash
docker compose up -d          # start
docker compose logs -f        # logs
docker compose restart        # restart
docker compose down           # stop
```

**Data layout** (bind mount `./data` → `/app/data`):

```
data/
├── surge-host.db         # SQLite metadata
├── users/{username}/     # Rule files
└── repos/{username}.git  # Git bare repos
```

**Backup:**

```bash
tar czf surge-host-backup-$(date +%F).tar.gz -C data .
```

### Run from source

```bash
go mod tidy
export SURGE_HOST_DOMAIN=localhost
export SURGE_HOST_ADMIN_PASSWORD=dev-password
export SURGE_HOST_JWT_SECRET=dev-secret
go run ./cmd/server
```

### Reverse proxy (Nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name rules.example.com;

    client_max_body_size 5m;

    location / {
        proxy_pass http://127.0.0.1:28080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SURGE_HOST_PORT` | `8080` | Container listen port |
| `SURGE_HOST_DATA_DIR` | `./data` | Data root |
| `SURGE_HOST_DOMAIN` | `localhost` | Public domain for Raw URL |
| `SURGE_HOST_ADMIN_USER` | `admin` | Admin username |
| `SURGE_HOST_ADMIN_PASSWORD` | _(empty)_ | Password; empty = dev mode |
| `SURGE_HOST_JWT_SECRET` | `change-me-in-production` | JWT signing key |
| `SURGE_HOST_MAX_FILE_SIZE` | `5242880` | Max file size (5 MB) |
| `SURGE_HOST_ALLOWED_EXTENSIONS` | `.conf,.list,.txt,.module,.yaml,.yml` | Allowed extensions |
| `SURGE_HOST_GIT_ENABLED` | `true` | Git versioning |
| `SURGE_HOST_VALIDATE_ENABLED` | `true` | Syntax validation |
| `SURGE_HOST_VALIDATE_STRICT` | `false` | Strict validation |
| `SURGE_HOST_CORS_ORIGINS` | _(empty)_ | CORS whitelist |

---

## API Overview

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/healthz` | — | Health check |
| `GET` | `/raw/{user}/{path...}` | — | Raw plain-text file |
| `POST` | `/api/auth/login` | — | Get JWT token |
| `GET` | `/api/files` | ✅ | List files |
| `POST` | `/api/files` | ✅ | Upload (multipart) |
| `GET/PUT/DELETE/PATCH` | `/api/files/{path...}` | ✅ | Read / update / delete / rename |
| `GET` | `/api/git/log/{path...}` | ✅ | Commit history |
| `GET` | `/api/git/show/{path...}` | ✅ | View version |
| `POST` | `/api/git/restore/{path...}` | ✅ | Rollback |
| `POST` | `/api/validate` | ✅ | Validate content |

```bash
curl -X POST https://rules.example.com/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-password"}'
```

---

## Security

- Set strong `SURGE_HOST_ADMIN_PASSWORD` and `SURGE_HOST_JWT_SECRET` in production
- `.env` is gitignored — use `.env.example` as template only
- Raw URLs are **public read** by design; write operations require auth
- Path traversal protection, extension whitelist, 5 MB size limit

---

## Project Structure

```
surge-host/
├── cmd/server/           # Entrypoint
├── internal/
│   ├── auth/             # JWT authentication
│   ├── db/               # SQLite
│   ├── handler/          # HTTP handlers
│   ├── store/            # File storage
│   └── vcs/              # Git versioning
├── pkg/validator/        # Surge syntax checker
├── web/                  # Templates + static assets
├── data/                 # Runtime data (gitignored)
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

---

## License

MIT