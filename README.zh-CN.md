# surge-host

**自托管多平台代理配置托管服务 — 上传、版本管理、语法校验，通过 Raw URL 统一分发 Surge、Meta/Mihomo、sing-box 等配置文件。**

> **项目状态（2026-07-01）：** 开发已**正式结束**。v2.4.1 为最终功能版本，生产环境运行于 [rules.asaqe.site](https://rules.asaqe.site)。更新记录见 [CHANGELOG.md](CHANGELOG.md)。

📖 [English README](README.md)

```
https://your-domain.com/raw/{user}/{filename}
```

---

## 它解决什么问题？

规则与代理配置通常需要一条**稳定、纯文本、可直接拉取**的 HTTP 地址：

- 返回内容必须是 `text/plain`，不能带 HTML 包装
- 读端应对客户端或同步脚本开放，写端应受控
- 规则或配置更新后，客户端能自动同步，无需手动改本地文件

**surge-host** 把这件事做成一套轻量私有服务：

| 能力 | 说明 |
|------|------|
| Raw URL | 纯文本输出，适合规则集、YAML、JSON 等配置直接拉取 |
| Web 管理 | 上传、列表、在线编辑 |
| Git 版本控制 | 按文件追踪历史，预览与回滚 |
| 语法校验 | 上线前拦截明显错误 |
| Docker 部署 | NAS、VPS、内网一键运行 |

---

## 为什么比 GitHub Gist / 静态 Nginx 更适合配置托管？

| | GitHub Gist | 静态 Nginx | **surge-host** |
|---|---|---|---|
| 纯文本 Raw 输出 | ⚠️ 有包装 / 限流 | ✅ 需手动配置 | ✅ 面向多平台配置分发设计 |
| 在线编辑 + 校验 | ❌ | ❌ | ✅ |
| 版本历史与回滚 | ⚠️ 仅 Git 历史 | ❌ | ✅ 按文件管理 |
| 多文件管理 | ❌ 一个 Gist 一个文件 | ⚠️ 手动维护目录 | ✅ UI + API |
| 私有写入、公开读取 | ⚠️ Token / 可见性 | ⚠️ 自行实现 | ✅ JWT 管理，Raw 公开 |
| Surge 工作流 | ❌ | ❌ | ✅ 开箱即用 |

**Gist** 适合公开分享单文件，但不适合维护多套规则、回滚和语法检查。

**静态 Nginx** 能发文件，但编辑、版本、校验都要自己拼，每次改规则都要手动 `scp`。

**surge-host** 是面向多平台代理配置托管的中间方案：**读端像静态站一样干净，写端像小型配置后台一样完整。**

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

> 修改 `web/` 模板或静态资源后，需执行 `docker compose build` 再 `up -d --force-recreate`；仅重启容器不会更新镜像内前端文件。

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

### 第四步：接入客户端

1. 打开 `https://rules.example.com` → 上传规则或配置文件
2. 从管理页复制 Raw URL
3. 在客户端或同步脚本中引用：

```text
https://rules.example.com/raw/admin/rules.list
https://rules.example.com/raw/admin/meta.yaml
https://rules.example.com/raw/admin/sing-box.json
```

保存后即可通过同一条 Raw URL 拉取新内容，适合 Surge、Meta/Mihomo、sing-box 或同步脚本统一消费。

---

## 日常使用

### 网页功能

| 路径 | 说明 |
|------|------|
| `/` | 产品概览、系统状态、平台说明、公开注册表 |
| `/upload` | 拖拽上传 |
| `/files` | 文件管理（重命名、删除、历史、复制 Raw URL） |
| `/edit/{path}` | 在线编辑、语法高亮、校验 |

顶栏导航：`[ Home ]` `[ Upload ]` `[ Manage ]`。界面为黑白网格化控制台风格（`system.css`）。

复制 Raw URL：点击 **Copy** 按钮、点击 URL 文本，或在 URL 输入框上右键。

启用认证后使用管理员账号登录，Token 保存在浏览器 `localStorage`。

### Raw URL

```
GET /raw/{user}/{path...}
```

- `Content-Type: text/plain; charset=utf-8`
- 无需认证（公开读取）
- 支持子目录路径
- 防路径穿越、扩展名白名单

### 健康检查

`/healthz` 会检查以下依赖状态：

- SQLite 连接可用
- `data/` 目录可写
- `data/users/` 目录存在
- 开启 Git 时 `data/repos/` 目录存在

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

当前内置以下格式校验：

- Surge：`.list`、`.conf`、`.module`
- Meta / Mihomo：`.yaml`、`.yml`
- sing-box：`.json`

其中：

- Surge 校验规则行格式、常见规则类型与配置段结构
- Meta / Mihomo 校验 YAML 语法，并检查 `proxies`、`proxy-groups`、`rules`、`rule-providers`、`payload` 等常见结构
- sing-box 校验 JSON 语法，并检查 `inbounds`、`outbounds`、`route`、`dns`、`log` 等顶层结构
- 严格模式（`SURGE_HOST_VALIDATE_STRICT=true`）会对未识别类型给出更严格提示

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

当 `SURGE_HOST_DOMAIN` 不是回环地址（`localhost`、`127.0.0.1`、`::1`）时，服务启动将强制要求：

- 必须设置非空 `SURGE_HOST_ADMIN_PASSWORD`
- 必须将 `SURGE_HOST_JWT_SECRET` 改为非默认值

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SURGE_HOST_PORT` | `8080` | 容器监听端口 |
| `SURGE_HOST_DATA_DIR` | `./data` | 数据根目录 |
| `SURGE_HOST_DOMAIN` | `localhost` | 公网域名（生成 Raw URL） |
| `SURGE_HOST_ADMIN_USER` | `admin` | 管理员用户名 |
| `SURGE_HOST_ADMIN_PASSWORD` | _(空)_ | 密码；仅回环开发环境可为空 |
| `SURGE_HOST_JWT_SECRET` | `change-me-in-production` | JWT 密钥；非回环部署必须改掉 |
| `SURGE_HOST_MAX_FILE_SIZE` | `5242880` | 单文件上限（5 MB） |
| `SURGE_HOST_ALLOWED_EXTENSIONS` | `.conf,.list,.txt,.module,.yaml,.yml,.json` | 允许扩展名 |
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
├── pkg/validator/        # 多格式配置校验
├── web/
│   ├── templates/        # Go HTML 模板
│   └── static/css/       # system.css 统一设计系统
├── data/                 # 运行时数据（不提交）
├── CHANGELOG.md          # 版本与结项记录
├── Dockerfile.release    # 1Panel 发布镜像（预编译二进制）
├── docker-compose.yml
└── .env.example
```

---

## 结项说明

本项目目标已全部达成：

- 多平台配置托管（Surge / Meta/Mihomo / sing-box）
- 稳定 Raw URL 纯文本分发
- Web 上传、编辑、校验、Git 版本管理
- Docker / 1Panel 一键部署
- 生产环境 [rules.asaqe.site](https://rules.asaqe.site) 已上线

后续仅接受关键安全修复，不再规划新功能。感谢使用。

---

## License

MIT
