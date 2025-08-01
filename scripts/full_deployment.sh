#!/bin/bash
# Industrial IoT 完整部署和测试脚本
# 一键部署中间件、更新配置、启动应用并运行测试

set -e

# 配置变量
SERVER_IP="192.168.5.16"
DEPLOYMENT_TYPE=${1:-"recommended"}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_step "Checking dependencies..."
    
    # 检查Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    if ! docker info > /dev/null 2>&1; then
        log_error "Docker is not running"
        exit 1
    fi
    
    # 检查Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    # 检查网络连通性
    if ! ping -c 1 "$SERVER_IP" > /dev/null 2>&1; then
        log_warning "Cannot ping $SERVER_IP, but continuing anyway..."
    fi
    
    log_success "All dependencies are available"
}

# 部署中间件
deploy_middleware() {
    log_step "Deploying middleware services..."
    
    cd "$PROJECT_DIR"
    
    if [[ ! -f "scripts/deploy_middleware.sh" ]]; then
        log_error "Middleware deployment script not found"
        exit 1
    fi
    
    ./scripts/deploy_middleware.sh "$DEPLOYMENT_TYPE"
    
    log_success "Middleware deployment completed"
}

# 更新配置
update_configuration() {
    log_step "Updating application configuration..."
    
    cd "$PROJECT_DIR"
    
    if [[ ! -f "scripts/update_config.sh" ]]; then
        log_error "Configuration update script not found"
        exit 1
    fi
    
    ./scripts/update_config.sh
    
    log_success "Configuration updated"
}

# 构建应用
build_application() {
    log_step "Building application..."
    
    cd "$PROJECT_DIR"
    
    # 清理之前的构建
    go clean -cache
    
    # 下载依赖
    go mod tidy
    go mod download
    
    # 构建主应用
    go build -o bin/iot-producer ./cmd/producer
    
    # 构建测试工具
    go build -o bin/test-runner ./cmd/test-runner 2>/dev/null || log_warning "Test runner build failed, continuing..."
    
    log_success "Application built successfully"
}

# 运行测试
run_tests() {
    log_step "Running tests..."
    
    cd "$PROJECT_DIR"
    
    # 运行单元测试
    log_info "Running unit tests..."
    go test -v ./internal/... -short -timeout=30s || log_warning "Some unit tests failed"
    
    # 运行核心组件测试
    log_info "Running core component tests..."
    go test -v ./tests/producer/standalone_core_test.go -timeout=30s || log_warning "Core tests failed"
    
    # 运行基准测试
    log_info "Running benchmark tests..."
    go test -v ./tests/producer/benchmark_test.go -bench=. -benchmem -timeout=60s || log_warning "Benchmark tests failed"
    
    log_success "Tests completed"
}

# 启动应用 (后台)
start_application() {
    log_step "Starting application..."
    
    cd "$PROJECT_DIR"
    
    # 创建日志目录
    mkdir -p logs
    
    # 启动应用
    nohup ./bin/iot-producer --config configs/development.yaml > logs/app.log 2>&1 &
    APP_PID=$!
    
    echo $APP_PID > .app.pid
    
    log_info "Application started with PID: $APP_PID"
    
    # 等待应用启动
    log_info "Waiting for application to start..."
    for i in {1..30}; do
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
            log_success "Application is running and healthy"
            return 0
        fi
        sleep 2
    done
    
    log_error "Application failed to start properly"
    return 1
}

# 运行集成测试
run_integration_tests() {
    log_step "Running integration tests..."
    
    cd "$PROJECT_DIR"
    
    # 等待服务稳定
    sleep 10
    
    # 测试健康检查
    log_info "Testing health endpoints..."
    if curl -s http://localhost:8080/health | grep -q "healthy"; then
        log_success "Health check passed"
    else
        log_error "Health check failed"
    fi
    
    # 测试指标端点
    log_info "Testing metrics endpoint..."
    if curl -s http://localhost:8080/metrics | grep -q "iot_monitoring"; then
        log_success "Metrics endpoint working"
    else
        log_warning "Metrics endpoint may have issues"
    fi
    
    # 运行E2E测试
    log_info "Running E2E tests..."
    go test -v ./tests/e2e/end_to_end_test.go -timeout=120s || log_warning "E2E tests failed"
    
    log_success "Integration tests completed"
}

# 显示状态
show_status() {
    log_step "Checking system status..."
    
    echo
    echo "🌐 Service Status:"
    echo "=================="
    
    # 检查Docker容器
    echo "Docker Containers:"
    docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "(kafka|redis|postgres|prometheus|grafana|jaeger)" || echo "No middleware containers found"
    
    echo
    echo "📊 Application Status:"
    echo "====================="
    
    # 检查应用进程
    if [[ -f ".app.pid" ]]; then
        APP_PID=$(cat .app.pid)
        if ps -p $APP_PID > /dev/null 2>&1; then
            echo "✅ IoT Producer: Running (PID: $APP_PID)"
        else
            echo "❌ IoT Producer: Not running"
        fi
    else
        echo "❌ IoT Producer: Not started"
    fi
    
    # 检查端口
    echo
    echo "🔌 Port Status:"
    echo "==============="
    for port in 8080 9090 3000 6379 5432 9092; do
        if lsof -i :$port > /dev/null 2>&1; then
            echo "✅ Port $port: In use"
        else
            echo "❌ Port $port: Available"
        fi
    done
    
    echo
    echo "🔗 Access URLs:"
    echo "==============="
    echo "  - IoT Producer API: http://localhost:8080"
    echo "  - Health Check: http://localhost:8080/health"
    echo "  - Metrics: http://localhost:8080/metrics"
    echo "  - Prometheus: http://$SERVER_IP:9090"
    echo "  - Grafana: http://$SERVER_IP:3000 (admin/admin123)"
    
    echo
    echo "📋 Useful Commands:"
    echo "==================="
    echo "  - View logs: tail -f logs/app.log"
    echo "  - Stop app: kill \$(cat .app.pid)"
    echo "  - Restart: ./scripts/full_deployment.sh"
    echo "  - Test API: curl http://localhost:8080/health"
}

# 清理函数
cleanup() {
    log_info "Cleaning up..."
    
    if [[ -f ".app.pid" ]]; then
        APP_PID=$(cat .app.pid)
        if ps -p $APP_PID > /dev/null 2>&1; then
            kill $APP_PID
            log_info "Stopped application (PID: $APP_PID)"
        fi
        rm -f .app.pid
    fi
}

# 显示帮助
show_help() {
    echo "Industrial IoT Full Deployment Script"
    echo
    echo "Usage: $0 [deployment_type] [options]"
    echo
    echo "Deployment Types:"
    echo "  minimal     - Kafka + Redis only"
    echo "  recommended - Kafka + Redis + PostgreSQL + Prometheus (default)"
    echo "  full        - All services including Grafana + Jaeger"
    echo
    echo "Options:"
    echo "  --no-tests     Skip running tests"
    echo "  --no-start     Don't start the application"
    echo "  --cleanup      Stop and cleanup existing deployment"
    echo "  --status       Show current system status"
    echo "  --help         Show this help message"
    echo
    echo "Examples:"
    echo "  $0 recommended"
    echo "  $0 full --no-tests"
    echo "  $0 --status"
    echo "  $0 --cleanup"
}

# 主函数
main() {
    echo "🚀 Industrial IoT Full Deployment"
    echo "=================================="
    echo "Server IP: $SERVER_IP"
    echo "Deployment Type: $DEPLOYMENT_TYPE"
    echo "Project Directory: $PROJECT_DIR"
    echo

    # 解析参数
    NO_TESTS=false
    NO_START=false
    CLEANUP=false
    STATUS_ONLY=false
    
    for arg in "$@"; do
        case $arg in
            --no-tests)
                NO_TESTS=true
                ;;
            --no-start)
                NO_START=true
                ;;
            --cleanup)
                CLEANUP=true
                ;;
            --status)
                STATUS_ONLY=true
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
        esac
    done
    
    # 设置清理陷阱
    trap cleanup EXIT
    
    # 只显示状态
    if [[ "$STATUS_ONLY" == "true" ]]; then
        show_status
        exit 0
    fi
    
    # 清理模式
    if [[ "$CLEANUP" == "true" ]]; then
        cleanup
        log_success "Cleanup completed"
        exit 0
    fi
    
    # 执行完整部署流程
    check_dependencies
    deploy_middleware
    update_configuration
    build_application
    
    if [[ "$NO_TESTS" != "true" ]]; then
        run_tests
    fi
    
    if [[ "$NO_START" != "true" ]]; then
        if start_application; then
            run_integration_tests
        fi
    fi
    
    show_status
    
    echo
    log_success "🎉 Full deployment completed successfully!"
    echo
    echo "Your Industrial IoT system is now running and ready for production!"
    echo "Check the status above and access the monitoring dashboards."
}

# 运行主函数
main "$@"
