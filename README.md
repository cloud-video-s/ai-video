# AI Video

AI Video 是一个 Go + Vue 3 的视频内容管理服务，包含客户端 API、管理后台、模板与 Banner 管理、用户及积分/VIP 管理、上传、RBAC 和运行时配置等模块。

## 技术栈

- 后端：Go、Gin、GORM、Casbin、Redis、Viper、Zap
- 数据库：MySQL（默认），代码同时包含 PostgreSQL 与 SQLite 驱动
- 前端：Vue 3、TypeScript、Vite、Pinia、Vue Router、Element Plus
- API 文档：服务运行后访问 `/docs`，OpenAPI JSON 位于 `/docs/openapi.json`

## 目录结构

```text
cmd/                    可执行程序入口
config/                 本地配置与 Casbin 模型
docs/                   接口和业务配置文档
internal/app/           应用初始化、迁移与种子数据
internal/model/         数据模型
internal/repository/    数据访问层
internal/server/        API 与管理后台模块
internal/pkg/           可复用基础组件
web/admin/              Vue 管理后台
```

## 本地开发

### 前置条件

- Go 1.26.4（以 `go.mod` 为准）
- Node.js 与 npm
- MySQL
- Redis

### 配置

默认配置位于 `config/config.yaml`。启动前至少确认数据库、Redis 和 JWT 配置。嵌套配置可以通过大写下划线环境变量覆盖，例如 `DATABASE_HOST` 覆盖 `database.host`。

生产环境必须使用 `server.mode: release`，并设置至少 32 字节且不同于默认值的 `jwt.secret`。不要提交真实密码、密钥或第三方平台凭据。

### 启动后端

```bash
go mod download
go run ./cmd/admin-server
```

也可以指定配置文件：

```bash
go run ./cmd/admin-server -config path/to/config.yaml
```

服务默认监听 `http://localhost:8080`。首次初始化会创建管理员 `admin/admin123`，登录后应立即修改密码。

### 启动管理后台

```bash
cd web/admin
npm ci
npm run dev
```

Vite 开发服务器默认把 `/admin`、`/api` 和 `/uploads` 代理到 `http://localhost:8080`；可用 `VITE_PROXY_TARGET` 修改目标。

### 检查与构建

```bash
go test ./...
go vet ./...
gofmt -w path/to/changed.go

cd web/admin
npm run build
```

### Podman 分离构建

```bash
podman build --file deploy/Containerfile.backend --tag ai-video-go:latest .
podman build --file deploy/Containerfile.web --tag ai-video-web:latest .
```

离线镜像导出、导入及运行方式见 [Podman 镜像构建与打包](docs/PODMAN.md)。

提交规范和检查清单见 [CONTRIBUTING.md](CONTRIBUTING.md)，文档导航见 [docs/README.md](docs/README.md)。

## 安全提示

仓库中的配置只适合本地开发。部署前必须更换数据库密码、Redis 密码和 JWT 密钥，并配置可信的 CORS 来源、OAuth Client ID、对象存储凭据及 HTTPS 反向代理。
