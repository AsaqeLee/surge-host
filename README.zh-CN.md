# surge-host

**自托管 Surge 规则与配置文件托管服务 — 上传、版本管理、语法校验，通过 Raw URL 一键订阅。**

📖 [English README](README.md)

```
https://your-domain.com/raw/{user}/{filename}
```

---

## 它解决什么问题？

Surge 的 `RULE-SET`、`DOMAIN-SET` 需要一条**稳定、纯文本、可直接拉取**的 HTTP 地址：

- 返回内容必须是 `text/plain`，不能带 HTML 包装
- 读端应对 Surge 开放，写端应受控
- 规则更新后，Surge 能自动同步，无需手动改本地文件

**surge-host** 把这件事做成一套轻量私有服务：

| 能力 | 说明 |
|------|------|
| Raw URL | 纯文本输出，Surge 直接解析 |
| Web 管理 | 上传、列表、在线编辑 |
| Git 版本控制 | 按文件追踪历史，预览与回滚 |
| 语法校验 | 上线前拦截明显错误 |
| Docker 部署 | NAS、VPS、内网一键运行 |

---

## 为什么比 GitHub Gist / 静态 Nginx 更适合 Surge？

| | GitHub Gist | 静态 Nginx | **surge-host** |
|---|---|---|---|
| 纯文本 Raw 输出 | ⚠️ 有包装 / 限流 | ✅ 需手动配置 | ✅ 专为 Surge 设计 |
| 在线编辑 + 校验 | ❌ | ❌ | ✅ |
| 版本历史与回滚 | ⚠️ 仅 Git 历史 | ❌ | ✅ 按文件管理 |
| 多文件管理 | ❌ 一个 Gist 一个文件 | ⚠️ 手动维护目录 | ✅ UI + API |
| 私有写入、公开读取 | ⚠️ Token / 可见性 | ⚠️ 自行实现 | ✅ JWT 管理，Raw 公开 |
| Surge 工作流 | ❌ | ❌ | ✅ 开箱即用 |

**Gist** 适合公开分享单文件，但不适合维护多套规则、回滚和语法检查。

**静态 Nginx** 能发文件，但编辑、版本、校验都要自己拼，每次改规则都要手动 `scp`。

**surge-host** 是专为 Surge 订阅设计的中间方案：**读端像静态站一样干净，写端像小型规则后台一样完整。**

---

## 快速部署与接入

### 前置要求

- Docker 20.10+ 与 Docker Compose v2+（推荐）
- 或 Go 1.22+ 用于本地开发
- 生产环境需 HTTPS 反代或隧道（Nginx、Caddy、Cloudflare Tunnel 等）

### 第一步：配置环境变量

```bash
git clone git@github.com:AsaqeLee/surge-host.git
cd surge-host
cp .env.example .env
```

编辑 `.env`（**切勿提交到 Git**）：

```env
# 宿主机映射端口（docker-compose 使用）
CONTAINER_NAME=surge-host
HOST_IP=
PANEL_APP_PORT_HTTP=28080

# 公网域名（用于生成 Raw URL）
SURGE_HOST_DOMAIN=rules.example.com

# 管理员账号
SURGE_HOST_ADMIN_USER=admin
SURGE_HOST_ADMIN_PASSWORD=请改为强密码

# JWT 签名密钥
SURGE_HOST_JWT_SECRET=请改为随机字符串
```

生成随机密钥：

```bash
openssl rand -hex 32
```

### 第二步：启动服务

```bash
docker compose up -d --build

# 健康检查
curl http://127.0.0.1:28080/healthz
# → {"status":"ok"}
```

> **1Panel 用户：** `docker-compose.yml` 默认接入 `1panel-network`，非 1Panel 环境请修改或删除 `networks` 配置。

### 第三步：暴露 HTTPS

将域名指向宿主机端口。以 **Cloudflare Tunnel** 为例：

```
Public Hostname : rules.example.com
Service         : http://192.168.1.x:28080
```

`SURGE_HOST_DOMAIN` 必须设为公网域名，否则 Raw URL 生成不正确。

**Nginx 反代示例：**

```nginx
server {
    listen 443 ssl http2;
    server_name rules.example.com;
    client_max_body_size 5m;

    location / {
        proxy_pass http://127.0.0.1:28080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 第四步：接入 Surge

1. 打开 `https://rules.example.com` → 上传规则文件
2. 从管理页复制 Raw URL
3. 写入 Surge 配置：

```ini
[Rule]
RULE-SET,https://rules.example.com/raw/admin/rules.list,PROXY
DOMAIN-SET,https://rules.example.com/raw/admin/domains.list,PROXY
FINAL,DIRECT
```

Surge 会定期拉取更新，**保存规则后无需手动同步。**

---

## 日常使用

### 网页功能

| 路径 | 说明 |
|------|------|
| `/` | 公开规则列表与快速入门 |
| `/upload` | 拖拽上传 |
| `/files` | 文件管理（重命名、删除、历史） |
| `/edit/{path}` | 在线编辑、语法高亮、校验 |

启用认证后使用管理员账号登录，Token 保存在浏览器 `localStorage`。

### Raw URL

```
GET /raw/{user}/{path...}
```

- `Content-Type: text/plain; charset=utf-8`
- 无需认证（公开读取）
- 支持子目录路径
- 防路径穿越、扩展名白名单

### 版本管理

每个用户拥有独立 Git 裸仓库，上传、编辑、重命名、删除均自动提交。

**网页操作：** 管理页 → **历史** → 预览或回滚

**API 示例：**

```bash
# 提交历史
curl -H "Authorization: Bearer $TOKEN" \
  https://rules.example.com/api/git/log/rules.list

# 查看指定版本
curl -H "Authorization: Bearer $TOKEN" \
  "https://rules.example.com/api/git/show/rules.list?commit=abc1234"

# 回滚
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"commit":"abc1234"}' \
  https://rules.example.com/api/git/restore/rules.list
```

### 语法校验

对 `.list`、`.conf`、`.module` 文件进行校验：

- 逗号分隔规则格式
- `.conf` 段名识别（`[Rule]`、`[General]` 等）
- 常见规则类型：`DOMAIN-SUFFIX`、`RULE-SET`、`GEOIP`、`IP-CIDR`、`FINAL` 等
- 严格模式（`SURGE_HOST_VALIDATE_STRICT=true`）将未知类型视为错误

编辑器内点 **校验**，或通过 API：

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"rules.list","content":"DOMAIN-SUFFIX,example.com,PROXY"}' \
  https://rules.example.com/api/validate
```

上传或保存失败时返回 HTTP `422` 及详细问题列表。

---

## 部署参考

### Docker Compose 常用命令

```bash
docker compose up -d          # 启动
docker compose logs -f        # 日志
docker compose restart        # 重启
docker compose down           # 停止
```

### 数据目录

绑定挂载 `./data` → 容器 `/app/data`：

```
data/
├── surge-host.db         # SQLite 元数据
├── users/{username}/     # 规则文件
└── repos/{username}.git  # Git 裸仓库
```

**备份：**

```bash
tar czf surge-host-backup-$(date +%F).tar.gz -C data .
```

### 源码运行

```bash
go mod tidy
export SURGE_HOST_DOMAIN=localhost
export SURGE_HOST_ADMIN_PASSWORD=dev-password
export SURGE_HOST_JWT_SECRET=dev-secret
go run ./cmd/server
```

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SURGE_HOST_PORT` | `8080` | 容器监听端口 |
| `SURGE_HOST_DATA_DIR` | `./data` | 数据根目录 |
| `SURGE_HOST_DOMAIN` | `localhost` | 公网域名（生成 Raw URL） |
| `SURGE_HOST_ADMIN_USER` | `admin` | 管理员用户名 |
| `SURGE_HOST_ADMIN_PASSWORD` | _(空)_ | 密码；为空则开发模式 |
| `SURGE_HOST_JWT_SECRET` | `change-me-in-production` | JWT 密钥 |
| `SURGE_HOST_MAX_FILE_SIZE` | `5242880` | 单文件上限（5 MB） |
| `SURGE_HOST_ALLOWED_EXTENSIONS` | `.conf,.list,.txt,.module,.yaml,.yml` | 允许扩展名 |
| `SURGE_HOST_GIT_ENABLED` | `true` | Git 版本控制 |
| `SURGE_HOST_VALIDATE_ENABLED` | `true` | 语法校验 |
| `SURGE_HOST_VALIDATE_STRICT` | `false` | 严格校验 |
| `SURGE_HOST_CORS_ORIGINS` | _(空)_ | CORS 白名单 |

---

## API 概览

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| `GET` | `/healthz` | — | 健康检查 |
| `GET` | `/raw/{user}/{path...}` | — | Raw 纯文本 |
| `POST` | `/api/auth/login` | — | 获取 JWT |
| `GET` | `/api/files` | ✅ | 文件列表 |
| `POST` | `/api/files` | ✅ | 上传 |
| `GET/PUT/DELETE/PATCH` | `/api/files/{path...}` | ✅ | 读/写/删/重命名 |
| `GET` | `/api/git/log/{path...}` | ✅ | 提交历史 |
| `GET` | `/api/git/show/{path...}` | ✅ | 查看版本 |
| `POST` | `/api/git/restore/{path...}` | ✅ | 回滚 |
| `POST` | `/api/validate` | ✅ | 语法校验 |

---

## 安全说明

- 生产环境务必设置强 `SURGE_HOST_ADMIN_PASSWORD` 与 `SURGE_HOST_JWT_SECRET`
- `.env` 已被 gitignore，仓库中仅保留 `.env.example`
- Raw URL 设计为公开读取；写入操作需认证
- 防路径穿越、扩展名白名单、文件大小限制

---

## 项目结构

```
surge-host/
├── cmd/server/           # 入口
├── internal/             # 核心业务逻辑
├── pkg/validator/        # Surge 语法校验
├── web/                  # 前端模板与静态资源
├── data/                 # 运行时数据（不提交）
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

---

## License

MIT