# Podman 镜像构建与打包

以下命令均在项目根目录执行。Web 与 Go 后端构建为两个独立镜像。

## 构建镜像

```bash
podman build --file deploy/Containerfile.backend --tag ai-video-go:latest .
podman build --file deploy/Containerfile.web --tag ai-video-web:latest .
```

如需给镜像指定发布版本：

```bash
podman build --file deploy/Containerfile.backend --tag ai-video-go:1.0.0 .
podman build --file deploy/Containerfile.web --tag ai-video-web:1.0.0 .
```

## 导出离线镜像包

```bash
podman save --format docker-archive --output ai-video-go.tar ai-video-go:latest
podman save --format docker-archive --output ai-video-web.tar ai-video-web:latest
```

在目标机器导入：

```bash
podman load --input ai-video-go.tar
podman load --input ai-video-web.tar
```

## 运行示例

先创建前后端共用网络和持久化卷：

```bash
podman network create ai-video-net
podman volume create ai-video-storage
podman volume create ai-video-logs
```

启动 Go 后端：

```bash
podman run -d --name ai-video-go --network ai-video-net -p 8080:8080 -v ai-video-storage:/app/storage -v ai-video-logs:/app/logs -e SERVER_MODE=release -e DATABASE_HOST=mysql -e REDIS_HOST=redis -e JWT_SECRET=replace-with-at-least-32-random-characters ai-video-go:latest
```

启动 Web。Nginx 默认通过容器名 `ai-video-go:8080` 代理 `/admin`、`/api`、`/docs` 和 `/uploads`：

```bash
podman run -d --name ai-video-web --network ai-video-net -p 80:80 ai-video-web:latest
```

如果后端容器名或端口不同，通过环境变量覆盖：

```bash
podman run -d --name ai-video-web --network ai-video-net -p 80:80 -e BACKEND_HOST=my-backend -e BACKEND_PORT=8080 ai-video-web:latest
```

后端镜像内置 `config/` 默认配置。生产环境应使用环境变量覆盖数据库、Redis、JWT、对象存储等敏感配置，也可以把完整配置目录只读挂载到 `/app/config`。
