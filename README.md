# surge-host

A lightweight, self-hosted service for hosting, versioning, and validating proxy configuration files. Deliver Surge, Meta/Mihomo, and sing-box configs through stable Raw URLs.

📖 [中文文档](README.zh-CN.md)

```
https://your-domain.com/raw/{user}/{filename}
```

## Overview

Proxy rules and client configs need a **stable, plain-text HTTP endpoint** — no HTML wrappers, controlled writes, and automatic sync on the client side.

**surge-host** provides a private platform for that workflow:

| Capability | Description |
|------------|-------------|
| Raw URL | Plain-text delivery for rule lists, YAML, JSON, and other config files |
| Web UI | Upload, file list, online editor with syntax highlighting |
| Git versioning | Per-file history, preview, and rollback |
| Validation | Catch syntax and structural errors before they go live |
| Docker | One-command deploy on NAS, VPS, or homelab |

### Supported formats

| Extension | Client / use case | Validation |
|-----------|-------------------|------------|
| `.list` | Surge rule sets | Rule-line syntax |
| `.conf`, `.module` | Surge configuration | Section and rule checks |
| `.yaml`, `.yml` | Meta / Mihomo | YAML syntax + structure |
| `.json` | sing-box | JSON syntax + top-level structure |
| `.txt` | Plain text | No validation |

Validation is **format-level / heuristic**, not a full upstream schema parser.

---

## Why not GitHub Gist or static Nginx?

| | GitHub Gist | Static Nginx | **surge-host** |
|---|---|---|---|
| Plain-text Raw output | ⚠️ Wrappers / rate limits | ✅ Manual setup | ✅ Built for config delivery |
| Online edit + validation | ❌ | ❌ | ✅ |
| Per-file history & rollback | ⚠️ Git history only | ❌ | ✅ |
| Multi-file management | ❌ One file per Gist | ⚠️ Manual dirs | ✅ UI + API |
| Private write, public read | ⚠️ Tokens / visibility | ⚠️ DIY | ✅ JWT admin, public Raw |
| Multi-client workflow | ❌ | ❌ | ✅ Out of the box |

---

## Technical Specifications

| Component | Technology |
| :--- | :--- |
| **Backend** | Go 1.22+ |
| **Database** | SQLite (embedded) |
| **VCS** | Native Git |
| **Frontend** | Go templates + vanilla JS |
| **Deployment** | Docker & Docker Compose |

---

## Quick Start

### Docker (recommended)

```bash
git clone https://github.com/AsaqeLee/surge-host.git
cd surge-host
cp .env.example .env
# Edit .env: domain, admin password, JWT secret
docker compose up -d --build
```

### Local development

```bash
go mod tidy
go run ./cmd/server
```

Default: `http://localhost:8080`

---

## Configuration

Key environment variables:

| Variable | Description |
|----------|-------------|
| `SURGE_HOST_DOMAIN` | Public domain for Raw URL generation |
| `SURGE_HOST_ADMIN_USER` | Dashboard admin username |
| `SURGE_HOST_ADMIN_PASSWORD` | Admin password (enables auth) |
| `SURGE_HOST_JWT_SECRET` | JWT signing secret |
| `SURGE_HOST_ALLOWED_EXTENSIONS` | Allowed file types (default includes `.json`) |
| `SURGE_HOST_VALIDATE_ENABLED` | Toggle syntax validation |
| `SURGE_HOST_GIT_ENABLED` | Toggle Git versioning |

See `.env.example` for the full list.

---

## Integration

### Raw URL examples

```text
https://your-domain.com/raw/user/rules.list
https://your-domain.com/raw/user/meta.yaml
https://your-domain.com/raw/user/sing-box.json
```

**Surge** — reference in `surge.conf`:

```ini
[Rule]
RULE-SET,https://your-domain.com/raw/user/rules.list,PROXY
```

**Meta / Mihomo** — use the Raw URL in `rule-providers` or subscription fields.

**sing-box** — point your client or sync script at the Raw JSON URL.

### REST API

| Endpoint | Description |
|----------|-------------|
| `GET /api/files` | List files |
| `POST /api/files` | Upload (multipart) |
| `PUT /api/files/{path}` | Replace content |
| `POST /api/validate` | Dry-run validation |
| `GET /api/git/log/{path}` | Version history |

Authenticated endpoints require `Authorization: Bearer <token>` from `POST /api/auth/login`.

---

## License

[MIT License](LICENSE) · [Asaqe Lee](https://asaqe.org)