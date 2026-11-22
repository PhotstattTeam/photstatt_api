# photstatt_api

Photstatt 后端服务：Go + Gin + GORM + Postgres。

## 环境要求

| 工具 | 版本 | 说明 |
|---|---|---|
| Go | 1.26+ | `go version` |
| Postgres | 18 | 本机 Homebrew 或 docker compose 二选一 |
| air | 最新 | 热重载，`go install github.com/air-verse/air@latest` |
| golangci-lint | 最新 | 可选，`brew install golangci-lint` |

## 快速开始

```bash
# 1. 配置（首次）
cp .env.example .env

# 2. 启动 Postgres（二选一）
make db-up        # 本机 Homebrew 服务
# make db-docker  # 或 docker compose

# 3. 首次需要建库建用户（Homebrew 方式）
psql -d postgres -c "CREATE ROLE photstatt LOGIN PASSWORD 'photstatt_dev_pw'"
createdb -O photstatt photstatt

# 4. 跑起来
make dev          # 热重载
# make run        # 或普通运行
```

启动后：

- `GET http://localhost:8080/api/v1/ping` → `{"message":"pong"}`
- `GET http://localhost:8080/healthz` → 含数据库连通性

## 目录结构

```
cmd/main.go              程序入口：装配配置、数据库、路由、优雅关闭
internal/config/         环境变量加载与校验（.env + os env）
internal/database/       GORM/Postgres 连接与连接池
.air.toml                热重载配置
docker-compose.yml       开发用 Postgres
Makefile                 常用命令（make help 查看）
```

## 配置项

全部通过环境变量注入，本地由 `.env` 提供（已 gitignore）。见 [.env.example](.env.example)。

生产环境建议设置 `APP_ENV=production`、`GIN_MODE=release`，并通过部署平台注入 `DB_PASSWORD` 等敏感值，不要提交 `.env`。

## 与前端联调

- 后端接口统一前缀 `/api/v1`
- Vue 开发服务器（5173）已配置代理：`/api` → `http://127.0.0.1:8080`
- 跨域白名单由 `CORS_ALLOW_ORIGINS` 控制

## 常用命令

```bash
make help      # 查看全部命令
make dev       # 热重载
make build     # 编译到 bin/photstatt_api
make tidy      # 同步依赖
make vet       # 静态检查
make test      # 测试
make lint      # golangci-lint
make psql      # 连接数据库
```
