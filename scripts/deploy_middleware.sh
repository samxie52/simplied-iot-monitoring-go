#!/bin/bash
# Industrial IoT 中间件一键部署脚本
# 使用方法: ./deploy_middleware.sh [minimal|recommended|full]

set -e

# 配置变量
SERVER_IP="192.168.5.16"
DEPLOYMENT_TYPE=${1:-"recommended"}

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
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

# 检查Docker是否运行
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        log_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    log_success "Docker is running"
}

# 清理现有容器
cleanup_containers() {
    log_info "Cleaning up existing containers..."
    
    containers=("kafka-server" "redis-server" "postgres-server" "prometheus-server" "grafana-server" "jaeger-server")
    
    for container in "${containers[@]}"; do
        if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
            log_warning "Removing existing container: $container"
            docker rm -f $container > /dev/null 2>&1 || true
        fi
    done
}

# 部署Kafka
deploy_kafka() {
    log_info "Deploying Kafka..."
    
    docker run -d \
        --name kafka-server \
        -p 9092:9092 \
        -e KAFKA_ENABLE_KRAFT=yes \
        -e KAFKA_CFG_PROCESS_ROLES=controller,broker \
        -e KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER \
        -e KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093 \
        -e KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT \
        -e KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://$SERVER_IP:9092 \
        -e KAFKA_BROKER_ID=1 \
        -e KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=1@127.0.0.1:9093 \
        -e ALLOW_PLAINTEXT_LISTENER=yes \
        -e KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=true \
        -e KAFKA_CFG_LOG_RETENTION_HOURS=168 \
        -e KAFKA_CFG_NUM_PARTITIONS=3 \
        -e KAFKA_CFG_DEFAULT_REPLICATION_FACTOR=1 \
        -v kafka_data:/bitnami/kafka \
        --restart unless-stopped \
        bitnami/kafka:latest > /dev/null
    
    log_success "Kafka deployed successfully"
}

# 部署Redis
deploy_redis() {
    log_info "Deploying Redis..."
    
    docker run -d \
        --name redis-server \
        -p 6379:6379 \
        -e REDIS_PASSWORD=redis123 \
        -v redis_data:/data \
        --restart unless-stopped \
        redis:7-alpine \
        redis-server --requirepass redis123 --appendonly yes > /dev/null
    
    log_success "Redis deployed successfully"
}

# 部署PostgreSQL
deploy_postgres() {
    log_info "Deploying PostgreSQL..."
    
    docker run -d \
        --name postgres-server \
        -p 5432:5432 \
        -e POSTGRES_DB=iot_monitoring \
        -e POSTGRES_USER=iot_user \
        -e POSTGRES_PASSWORD=iot_password123 \
        -e POSTGRES_INITDB_ARGS="--encoding=UTF-8 --lc-collate=C --lc-ctype=C" \
        -v postgres_data:/var/lib/postgresql/data \
        --restart unless-stopped \
        postgres:15-alpine > /dev/null
    
    log_success "PostgreSQL deployed successfully"
}

# 部署Prometheus
deploy_prometheus() {
    log_info "Deploying Prometheus..."
    
    # 创建Prometheus配置
    mkdir -p /tmp/prometheus
    cat > /tmp/prometheus/prometheus.yml << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'iot-producer'
    static_configs:
      - targets: ['host.docker.internal:8080']
    metrics_path: /metrics
    scrape_interval: 10s

  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
EOF

    docker run -d \
        --name prometheus-server \
        -p 9090:9090 \
        -v /tmp/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml \
        -v prometheus_data:/prometheus \
        --restart unless-stopped \
        prom/prometheus:latest \
        --config.file=/etc/prometheus/prometheus.yml \
        --storage.tsdb.path=/prometheus \
        --web.console.libraries=/etc/prometheus/console_libraries \
        --web.console.templates=/etc/prometheus/consoles \
        --storage.tsdb.retention.time=15d \
        --web.enable-lifecycle > /dev/null
    
    log_success "Prometheus deployed successfully"
}

# 部署Grafana
deploy_grafana() {
    log_info "Deploying Grafana..."
    
    docker run -d \
        --name grafana-server \
        -p 3000:3000 \
        -e GF_SECURITY_ADMIN_USER=admin \
        -e GF_SECURITY_ADMIN_PASSWORD=admin123 \
        -e GF_INSTALL_PLUGINS=grafana-clock-panel,grafana-simple-json-datasource \
        -v grafana_data:/var/lib/grafana \
        --restart unless-stopped \
        grafana/grafana:latest > /dev/null
    
    log_success "Grafana deployed successfully"
}

# 部署Jaeger
deploy_jaeger() {
    log_info "Deploying Jaeger..."
    
    docker run -d \
        --name jaeger-server \
        -p 16686:16686 \
        -p 14268:14268 \
        -p 14250:14250 \
        -e COLLECTOR_OTLP_ENABLED=true \
        --restart unless-stopped \
        jaegertracing/all-in-one:latest > /dev/null
    
    log_success "Jaeger deployed successfully"
}

# 等待服务启动
wait_for_services() {
    log_info "Waiting for services to start..."
    
    # 等待Kafka启动
    log_info "Waiting for Kafka..."
    for i in {1..30}; do
        if docker exec kafka-server kafka-topics.sh --bootstrap-server localhost:9092 --list > /dev/null 2>&1; then
            break
        fi
        sleep 2
    done
    
    # 等待Redis启动
    if docker ps --format '{{.Names}}' | grep -q redis-server; then
        log_info "Waiting for Redis..."
        for i in {1..15}; do
            if docker exec redis-server redis-cli -a redis123 ping > /dev/null 2>&1; then
                break
            fi
            sleep 1
        done
    fi
    
    # 等待PostgreSQL启动
    if docker ps --format '{{.Names}}' | grep -q postgres-server; then
        log_info "Waiting for PostgreSQL..."
        for i in {1..30}; do
            if docker exec postgres-server pg_isready -U iot_user > /dev/null 2>&1; then
                break
            fi
            sleep 2
        done
    fi
    
    log_success "All services are ready"
}

# 创建Kafka主题
create_kafka_topics() {
    log_info "Creating Kafka topics..."
    
    topics=("industrial-iot-data" "device-alerts" "system-metrics")
    
    for topic in "${topics[@]}"; do
        docker exec kafka-server kafka-topics.sh \
            --bootstrap-server localhost:9092 \
            --create \
            --if-not-exists \
            --topic $topic \
            --partitions 3 \
            --replication-factor 1 > /dev/null 2>&1
        log_success "Created topic: $topic"
    done
}

# 初始化数据库
init_database() {
    if docker ps --format '{{.Names}}' | grep -q postgres-server; then
        log_info "Initializing database..."
        
        docker exec postgres-server psql -U iot_user -d iot_monitoring -c "
        CREATE TABLE IF NOT EXISTS device_data (
            id SERIAL PRIMARY KEY,
            device_id VARCHAR(50) NOT NULL,
            timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            sensor_type VARCHAR(20),
            value DECIMAL(10,2),
            status VARCHAR(20),
            INDEX idx_device_timestamp (device_id, timestamp)
        );" > /dev/null 2>&1
        
        log_success "Database initialized"
    fi
}

# 验证部署
verify_deployment() {
    log_info "Verifying deployment..."
    
    services=()
    
    # 检查各服务状态
    if docker ps --format '{{.Names}}' | grep -q kafka-server; then
        services+=("Kafka:$SERVER_IP:9092")
    fi
    
    if docker ps --format '{{.Names}}' | grep -q redis-server; then
        services+=("Redis:$SERVER_IP:6379")
    fi
    
    if docker ps --format '{{.Names}}' | grep -q postgres-server; then
        services+=("PostgreSQL:$SERVER_IP:5432")
    fi
    
    if docker ps --format '{{.Names}}' | grep -q prometheus-server; then
        services+=("Prometheus:http://$SERVER_IP:9090")
    fi
    
    if docker ps --format '{{.Names}}' | grep -q grafana-server; then
        services+=("Grafana:http://$SERVER_IP:3000")
    fi
    
    if docker ps --format '{{.Names}}' | grep -q jaeger-server; then
        services+=("Jaeger:http://$SERVER_IP:16686")
    fi
    
    log_success "Deployment completed successfully!"
    echo
    echo "🌐 Available Services:"
    for service in "${services[@]}"; do
        echo "  - $service"
    done
    
    echo
    echo "📋 Default Credentials:"
    echo "  - Redis password: redis123"
    echo "  - PostgreSQL: iot_user/iot_password123"
    echo "  - Grafana: admin/admin123"
    
    echo
    echo "🚀 Next Steps:"
    echo "  1. Update your configs/development.yaml with the new service addresses"
    echo "  2. Run your IoT Producer: go run ./cmd/producer --config configs/development.yaml"
    echo "  3. Access monitoring dashboards at the URLs above"
}

# 显示帮助信息
show_help() {
    echo "Industrial IoT Middleware Deployment Script"
    echo
    echo "Usage: $0 [deployment_type]"
    echo
    echo "Deployment Types:"
    echo "  minimal     - Kafka + Redis (core functionality)"
    echo "  recommended - Kafka + Redis + PostgreSQL + Prometheus (default)"
    echo "  full        - All services including Grafana + Jaeger"
    echo
    echo "Examples:"
    echo "  $0 minimal"
    echo "  $0 recommended"
    echo "  $0 full"
}

# 主函数
main() {
    echo "🚀 Industrial IoT Middleware Deployment"
    echo "========================================"
    echo "Server IP: $SERVER_IP"
    echo "Deployment Type: $DEPLOYMENT_TYPE"
    echo

    case $DEPLOYMENT_TYPE in
        "help"|"-h"|"--help")
            show_help
            exit 0
            ;;
        "minimal")
            log_info "Deploying minimal stack (Kafka + Redis)..."
            ;;
        "recommended")
            log_info "Deploying recommended stack (Kafka + Redis + PostgreSQL + Prometheus)..."
            ;;
        "full")
            log_info "Deploying full stack (All services)..."
            ;;
        *)
            log_error "Unknown deployment type: $DEPLOYMENT_TYPE"
            show_help
            exit 1
            ;;
    esac

    # 执行部署流程
    check_docker
    cleanup_containers
    
    # 核心服务 (所有部署类型都需要)
    deploy_kafka
    deploy_redis
    
    # 推荐和完整部署
    if [[ "$DEPLOYMENT_TYPE" == "recommended" || "$DEPLOYMENT_TYPE" == "full" ]]; then
        deploy_postgres
        deploy_prometheus
    fi
    
    # 完整部署
    if [[ "$DEPLOYMENT_TYPE" == "full" ]]; then
        deploy_grafana
        deploy_jaeger
    fi
    
    # 等待服务启动
    wait_for_services
    
    # 初始化配置
    create_kafka_topics
    
    if [[ "$DEPLOYMENT_TYPE" == "recommended" || "$DEPLOYMENT_TYPE" == "full" ]]; then
        init_database
    fi
    
    # 验证部署
    verify_deployment
}

# 运行主函数
main "$@"
