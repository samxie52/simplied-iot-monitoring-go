# Industrial IoT Monitoring System Makefile
# 企业级工业IoT监控系统构建脚本

# 项目配置
PROJECT_NAME := simplied-iot-monitoring-go
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date +%Y-%m-%d_%H:%M:%S)
GO_VERSION := $(shell go version | awk '{print $$3}')

# 构建配置
BUILD_DIR := build
BINARY_DIR := $(BUILD_DIR)/bin
DOCKER_REGISTRY := localhost:5000
DOCKER_TAG := $(VERSION)

# Go构建标志
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GoVersion=$(GO_VERSION)"
GCFLAGS := -gcflags="all=-trimpath=$(PWD)"
ASMFLAGS := -asmflags="all=-trimpath=$(PWD)"

# 默认目标
.PHONY: all
all: clean deps build test

# 帮助信息
.PHONY: help
help:
	@echo "Industrial IoT Monitoring System - Build Commands"
	@echo "=================================================="
	@echo "Development Commands:"
	@echo "  deps                 下载并安装依赖包"
	@echo "  build                构建所有服务"
	@echo "  test                 运行所有测试"
	@echo "  run-producer         运行数据生产者服务"
	@echo "  run-consumer         运行数据消费者服务"
	@echo "  run-websocket        运行WebSocket服务"
	@echo "  run-web              运行Web服务"
	@echo ""
	@echo "Build Commands:"
	@echo "  build-producer       构建生产者服务"
	@echo "  build-consumer       构建消费者服务"
	@echo "  build-websocket      构建WebSocket服务"
	@echo "  build-web            构建Web服务"
	@echo ""
	@echo "Quality Commands:"
	@echo "  fmt                  格式化代码"
	@echo "  lint                 代码静态检查"
	@echo "  vet                  Go代码检查"
	@echo "  test-unit            运行单元测试"
	@echo "  test-integration     运行集成测试"
	@echo "  test-coverage        生成测试覆盖率报告"
	@echo ""
	@echo "Docker Commands:"
	@echo "  docker-build         构建所有Docker镜像"
	@echo "  docker-build-producer    构建生产者服务镜像"
	@echo "  docker-build-consumer    构建消费者服务镜像"
	@echo "  docker-build-websocket   构建 WebSocket服务镜像"
	@echo "  docker-build-web         构建 Web服务镜像"
	@echo "  docker-build-service     构建指定服务镜像 (SERVICE=xxx)"
	@echo "  docker-up            启动Docker服务"
	@echo "  docker-down          停止Docker服务"
	@echo ""
	@echo "Utility Commands:"
	@echo "  clean                清理构建产物"
	@echo "  dev                  启动开发环境"

# 依赖管理
.PHONY: deps
deps:
	@echo "📦 下载依赖包..."
	go mod download
	go mod verify
	@echo "✅ 依赖包下载完成"

# 代码格式化
.PHONY: fmt
fmt:
	@echo "🎨 格式化代码..."
	go fmt ./...
	@echo "✅ 代码格式化完成"

# 代码检查
.PHONY: vet
vet:
	@echo "🔍 Go代码检查..."
	go vet ./...
	@echo "✅ 代码检查完成"

# 静态检查 (需要安装golangci-lint)
.PHONY: lint
lint:
	@echo "🔍 代码静态检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "⚠️  golangci-lint 未安装，跳过静态检查"; \
		echo "   安装命令: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi
	@echo "✅ 静态检查完成"

# 创建构建目录
$(BINARY_DIR):
	@mkdir -p $(BINARY_DIR)

# 构建所有服务
.PHONY: build
build: build-producer build-consumer build-websocket build-web
	@echo "🎉 所有服务构建完成"

# 构建生产者服务
.PHONY: build-producer
build-producer: $(BINARY_DIR)
	@echo "🔨 构建生产者服务..."
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) \
		-o $(BINARY_DIR)/producer ./cmd/producer
	@echo "✅ 生产者服务构建完成"

# 构建消费者服务
.PHONY: build-consumer
build-consumer: $(BINARY_DIR)
	@echo "🔨 构建消费者服务..."
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) \
		-o $(BINARY_DIR)/consumer ./cmd/consumer
	@echo "✅ 消费者服务构建完成"

# 构建WebSocket服务
.PHONY: build-websocket
build-websocket: $(BINARY_DIR)
	@echo "🔨 构建WebSocket服务..."
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) \
		-o $(BINARY_DIR)/websocket ./cmd/websocket
	@echo "✅ WebSocket服务构建完成"

# 构建Web服务
.PHONY: build-web
build-web: $(BINARY_DIR)
	@echo "🔨 构建Web服务..."
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) \
		-o $(BINARY_DIR)/web ./cmd/web
	@echo "✅ Web服务构建完成"

# 运行测试
.PHONY: test
test: test-unit
	@echo "🧪 所有测试完成"

# 单元测试
.PHONY: test-unit
test-unit:
	@echo "🧪 运行单元测试..."
	go test -v -race ./tests/unit/...
	@echo "✅ 单元测试完成"

# 集成测试
.PHONY: test-integration
test-integration:
	@echo "🧪 运行集成测试..."
	go test -v -race ./tests/integration/...
	@echo "✅ 集成测试完成"

# 端到端测试
.PHONY: test-e2e
test-e2e:
	@echo "🧪 运行端到端测试..."
	go test -v -race ./tests/e2e/...
	@echo "✅ 端到端测试完成"

# 测试覆盖率
.PHONY: test-coverage
test-coverage:
	@echo "📊 生成测试覆盖率报告..."
	@mkdir -p $(BUILD_DIR)/coverage
	go test -v -race -coverprofile=$(BUILD_DIR)/coverage/coverage.out ./...
	go tool cover -html=$(BUILD_DIR)/coverage/coverage.out -o $(BUILD_DIR)/coverage/coverage.html
	@echo "✅ 测试覆盖率报告生成完成: $(BUILD_DIR)/coverage/coverage.html"

# 运行服务
.PHONY: run-producer
run-producer:
	@echo "🚀 启动生产者服务..."
	go run ./cmd/producer

.PHONY: run-consumer
run-consumer:
	@echo "🚀 启动消费者服务..."
	go run ./cmd/consumer

.PHONY: run-websocket
run-websocket:
	@echo "🚀 启动WebSocket服务..."
	go run ./cmd/websocket

.PHONY: run-web
run-web:
	@echo "🚀 启动Web服务..."
	go run ./cmd/web

# Docker相关
.PHONY: docker-build
docker-build: docker-build-producer docker-build-consumer docker-build-websocket docker-build-web
	@echo "🎉 所有Docker镜像构建完成"

# 构建单个服务Docker镜像
.PHONY: docker-build-producer
docker-build-producer:
	@echo "🐳 构建生产者服务Docker镜像..."
	docker build --target producer \
		--build-arg SERVICE=producer \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME="$(BUILD_TIME)" \
		--build-arg GIT_COMMIT="$(shell git rev-parse HEAD 2>/dev/null || echo 'unknown')" \
		-t $(DOCKER_REGISTRY)/$(PROJECT_NAME)-producer:$(DOCKER_TAG) .
	@echo "✅ 生产者服务Docker镜像构建完成"

.PHONY: docker-build-consumer
docker-build-consumer:
	@echo "🐳 构建消费者服务Docker镜像..."
	docker build --target consumer \
		--build-arg SERVICE=consumer \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME="$(BUILD_TIME)" \
		--build-arg GIT_COMMIT="$(shell git rev-parse HEAD 2>/dev/null || echo 'unknown')" \
		-t $(DOCKER_REGISTRY)/$(PROJECT_NAME)-consumer:$(DOCKER_TAG) .
	@echo "✅ 消费者服务Docker镜像构建完成"

.PHONY: docker-build-websocket
docker-build-websocket:
	@echo "🐳 构建 WebSocket服务Docker镜像..."
	docker build --target websocket \
		--build-arg SERVICE=websocket \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME="$(BUILD_TIME)" \
		--build-arg GIT_COMMIT="$(shell git rev-parse HEAD 2>/dev/null || echo 'unknown')" \
		-t $(DOCKER_REGISTRY)/$(PROJECT_NAME)-websocket:$(DOCKER_TAG) .
	@echo "✅ WebSocket服务Docker镜像构建完成"

.PHONY: docker-build-web
docker-build-web:
	@echo "🐳 构建 Web服务Docker镜像..."
	docker build --target web \
		--build-arg SERVICE=web \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME="$(BUILD_TIME)" \
		--build-arg GIT_COMMIT="$(shell git rev-parse HEAD 2>/dev/null || echo 'unknown')" \
		-t $(DOCKER_REGISTRY)/$(PROJECT_NAME)-web:$(DOCKER_TAG) .
	@echo "✅ Web服务Docker镜像构建完成"

# 构建通用Docker镜像（可指定服务类型）
.PHONY: docker-build-service
docker-build-service:
	@if [ -z "$(SERVICE)" ]; then \
		echo "❌ 请指定服务类型: make docker-build-service SERVICE=producer"; \
		exit 1; \
	fi
	@echo "🐳 构建 $(SERVICE) 服务Docker镜像..."
	docker build --target $(SERVICE) \
		--build-arg SERVICE=$(SERVICE) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME="$(BUILD_TIME)" \
		--build-arg GIT_COMMIT="$(shell git rev-parse HEAD 2>/dev/null || echo 'unknown')" \
		-t $(DOCKER_REGISTRY)/$(PROJECT_NAME)-$(SERVICE):$(DOCKER_TAG) .
	@echo "✅ $(SERVICE) 服务Docker镜像构建完成"

.PHONY: docker-up
docker-up:
	@echo "🐳 启动Docker服务..."
	docker-compose up -d
	@echo "✅ Docker服务启动完成"

.PHONY: docker-down
docker-down:
	@echo "🐳 停止Docker服务..."
	docker-compose down
	@echo "✅ Docker服务停止完成"

# 开发环境
.PHONY: dev
dev:
	@echo "🚀 启动开发环境..."
	@echo "1. 启动Kafka和相关服务..."
	docker-compose up -d kafka zookeeper
	@echo "2. 等待服务启动..."
	sleep 10
	@echo "3. 启动所有应用服务..."
	@echo "   - 生产者服务: make run-producer"
	@echo "   - 消费者服务: make run-consumer"
	@echo "   - WebSocket服务: make run-websocket"
	@echo "   - Web服务: make run-web"
	@echo "✅ 开发环境准备完成"

# 清理
.PHONY: clean
clean:
	@echo "🧹 清理构建产物..."
	rm -rf $(BUILD_DIR)
	go clean -cache
	go clean -testcache
	@echo "✅ 清理完成"

# 安装开发工具
.PHONY: install-tools
install-tools:
	@echo "🔧 安装开发工具..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "✅ 开发工具安装完成"

# 检查环境
.PHONY: check-env
check-env:
	@echo "🔍 检查开发环境..."
	@echo "Go版本: $(GO_VERSION)"
	@echo "项目版本: $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)"
	@which docker >/dev/null 2>&1 && echo "✅ Docker已安装" || echo "❌ Docker未安装"
	@which docker-compose >/dev/null 2>&1 && echo "✅ Docker Compose已安装" || echo "❌ Docker Compose未安装"
	@which golangci-lint >/dev/null 2>&1 && echo "✅ golangci-lint已安装" || echo "❌ golangci-lint未安装"
	@echo "✅ 环境检查完成"
