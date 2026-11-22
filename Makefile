APP_NAME := photstatt_api
BIN      := bin/$(APP_NAME)
GOBIN    := $(shell go env GOPATH)/bin
AIR      := $(GOBIN)/air
GOOSE    := $(GOBIN)/goose
PG_URL   := postgres://photstatt:photstatt_dev_pw@127.0.0.1:5432/photstatt?sslmode=disable
DB_URL   := $(PG_URL)
MIG_DIR  := migrations

.PHONY: help run dev build tidy fmt vet test lint db-up db-down db-docker psql \
        migrate-create migrate-up migrate-status migrate-down migrate-redo

help: ## 显示可用命令
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

run: ## 直接运行
	go run ./cmd

dev: ## 热重载（需 air）
	$(AIR) -c .air.toml

build: ## 编译到 bin/
	go build -o $(BIN) ./cmd

tidy: ## 同步 go.mod / go.sum
	go mod tidy

vet: ## go vet
	go vet ./...

test: ## 运行测试
	go test ./...

lint: ## 静态检查（golangci-lint）
	golangci-lint run ./...

fmt: ## 格式化（gofmt + goimports）
	golangci-lint fmt ./...

db-up: ## 启动本机 Homebrew Postgres
	brew services start postgresql@18

db-down: ## 停止本机 Homebrew Postgres
	brew services stop postgresql@18

db-docker: ## 用 docker compose 启动 Postgres（与 db-up 二选一）
	docker compose up -d postgres

psql: ## 连接本地数据库
	psql "$(PG_URL)"

migrate-create: ## 新建迁移：make migrate-create name=add_users_table
	@test -n "$(name)" || (echo "用法: make migrate-create name=add_users_table" && exit 1)
	$(GOOSE) -dir $(MIG_DIR) postgres "$(DB_URL)" create $(name) sql

migrate-up: ## 执行所有未执行的迁移
	$(GOOSE) -dir $(MIG_DIR) postgres "$(DB_URL)" up

migrate-status: ## 查看迁移状态
	@if ls $(MIG_DIR)/*.sql >/dev/null 2>&1; then \
		$(GOOSE) -dir $(MIG_DIR) postgres "$(DB_URL)" status; \
	else \
		echo "还没有迁移文件，先运行: make migrate-create name=xxx"; \
	fi

migrate-down: ## 回滚最近一个迁移
	$(GOOSE) -dir $(MIG_DIR) postgres "$(DB_URL)" down

migrate-redo: ## 回滚并重新执行最近一个迁移
	$(GOOSE) -dir $(MIG_DIR) postgres "$(DB_URL)" redo
