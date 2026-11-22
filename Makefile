APP_NAME := photstatt_api
BIN      := bin/$(APP_NAME)
GOBIN    := $(shell go env GOPATH)/bin
AIR      := $(GOBIN)/air
PG_URL   := postgres://photstatt:photstatt_dev_pw@127.0.0.1:5432/photstatt?sslmode=disable

.PHONY: help run dev build tidy vet test lint db-up db-down db-docker psql

help: ## 显示可用命令
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

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

lint: ## 静态检查（需 golangci-lint）
	golangci-lint run ./...

db-up: ## 启动本机 Homebrew Postgres
	brew services start postgresql@18

db-down: ## 停止本机 Homebrew Postgres
	brew services stop postgresql@18

db-docker: ## 用 docker compose 启动 Postgres（与 db-up 二选一）
	docker compose up -d postgres

psql: ## 连接本地数据库
	psql "$(PG_URL)"
