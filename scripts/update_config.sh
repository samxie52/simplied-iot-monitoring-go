#!/bin/bash
# Industrial IoT 配置更新脚本
# 自动更新配置文件以连接到远程中间件服务

set -e

# 配置变量
SERVER_IP="192.168.5.16"
CONFIG_FILE="configs/development.yaml"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# 备份原配置文件
backup_config() {
    if [[ -f "$CONFIG_FILE" ]]; then
        cp "$CONFIG_FILE" "${CONFIG_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
        log_success "Backed up original config to ${CONFIG_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
    fi
}

# 更新配置文件
update_config() {
    log_info "Updating configuration file: $CONFIG_FILE"
    
    cat > "$CONFIG_FILE" << EOF
# Industrial IoT Monitoring - Development Configuration
# Updated for remote middleware deployment on $SERVER_IP

# 环境配置
environment: development

# 服务器配置
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
  max_header_bytes: 1048576

# 数据库配置 (PostgreSQL)
db:
  host: "$SERVER_IP"
  port: 5432
  user: "iot_user"
  password: "iot_password123"
  database: "iot_monitoring"
  sslmode: "disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

# Redis配置
redis:
  host: "$SERVER_IP"
  port: 6379
  password: "redis123"
  db: 0
  pool_size: 10
  min_idle_conns: 5
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
  pool_timeout: 4s
  idle_timeout: 300s

# Kafka配置
kafka:
  brokers:
    - "$SERVER_IP:9092"
  topic: "industrial-iot-data"
  client_id: "iot-producer-dev"
  
  # 生产者配置
  producer:
    max_message_bytes: 1000000
    required_acks: 1
    timeout: 10s
    compression: "snappy"
    flush_frequency: 100ms
    flush_messages: 100
    flush_bytes: 65536
    retry_max: 3
    retry_backoff: 100ms
    return_successes: true
    return_errors: true
    
    # 批处理配置
    batch_size: 100
    batch_timeout: 1s
    max_batch_size: 1000
    
    # 缓冲区配置
    channel_buffer_size: 256
    
  # 消费者配置 (如需要)
  consumer:
    group_id: "iot-consumer-group"
    session_timeout: 30s
    heartbeat_interval: 3s
    rebalance_timeout: 60s
    fetch_min: 1
    fetch_max: 1048576
    fetch_default: 1048576

# 生产者配置
producer:
  # 设备模拟配置
  device:
    count: 50
    sample_interval: 5s
    batch_size: 10
    
    # 传感器范围配置
    temperature:
      min: 15.0
      max: 45.0
      warning_threshold: 35.0
      error_threshold: 40.0
    
    humidity:
      min: 30.0
      max: 90.0
      warning_threshold: 80.0
      error_threshold: 85.0
    
    pressure:
      min: 900.0
      max: 1100.0
      warning_threshold: 1050.0
      error_threshold: 1080.0
    
    current:
      min: 0.5
      max: 15.0
      warning_threshold: 12.0
      error_threshold: 14.0
  
  # 工作池配置
  worker_pool:
    size: 10
    queue_size: 1000
    timeout: 30s
  
  # 连接池配置
  connection_pool:
    max_size: 20
    timeout: 10s

# 消费者配置 (如需要)
consumer:
  enabled: false
  topics:
    - "industrial-iot-data"
    - "device-alerts"
  group_id: "iot-monitoring-consumer"
  auto_offset_reset: "latest"
  enable_auto_commit: true
  auto_commit_interval: 1s

# WebSocket配置
websocket:
  enabled: true
  path: "/ws"
  read_buffer_size: 1024
  write_buffer_size: 1024
  check_origin: false
  enable_compression: true
  handshake_timeout: 10s
  read_deadline: 60s
  write_deadline: 10s
  ping_period: 54s
  pong_wait: 60s
  max_message_size: 512

# Web界面配置
web:
  enabled: true
  static_path: "./web/static"
  template_path: "./web/templates"
  upload_path: "./web/uploads"
  max_upload_size: 10485760  # 10MB

# 监控配置
monitoring:
  enabled: true
  metrics_path: "/metrics"
  health_path: "/health"
  ready_path: "/ready"
  
  # Prometheus配置
  prometheus:
    namespace: "iot_monitoring"
    subsystem: "producer"
    
  # 健康检查配置
  health_check:
    interval: 30s
    timeout: 5s
    
    # 检查项配置
    checks:
      kafka: true
      redis: true
      database: true
      memory: true
      disk: true

# 告警配置
alert:
  enabled: true
  
  # 告警规则
  rules:
    # 错误率告警
    error_rate:
      threshold: 0.05  # 5%
      window: "5m"
      severity: "warning"
    
    # 延迟告警
    latency:
      threshold: 1000  # 1秒
      window: "5m"
      severity: "warning"
    
    # 内存使用告警
    memory_usage:
      threshold: 0.8  # 80%
      window: "5m"
      severity: "critical"
    
    # CPU使用告警
    cpu_usage:
      threshold: 0.8  # 80%
      window: "5m"
      severity: "warning"
  
  # 告警通道配置
  channels:
    console:
      enabled: true
      level: "info"
    
    file:
      enabled: true
      path: "./logs/alerts.log"
      level: "warning"

# 日志配置
logging:
  level: "info"
  format: "json"
  output: "stdout"
  
  # 文件日志配置
  file:
    enabled: true
    path: "./logs/app.log"
    max_size: 100  # MB
    max_backups: 5
    max_age: 30    # days
    compress: true
  
  # 日志字段配置
  fields:
    service: "iot-producer"
    version: "1.0.0"
    environment: "development"

# 性能配置
performance:
  # 内存池配置
  memory_pool:
    enabled: true
    initial_size: 1024
    max_size: 10240
    
  # 零拷贝配置
  zero_copy:
    enabled: true
    buffer_size: 4096
    
  # 批处理配置
  batch_processing:
    enabled: true
    max_batch_size: 1000
    batch_timeout: 1s
    
  # 连接复用配置
  connection_reuse:
    enabled: true
    max_idle_conns: 10
    idle_timeout: 300s

# 安全配置
security:
  # TLS配置 (生产环境启用)
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
    
  # 认证配置 (生产环境启用)
  auth:
    enabled: false
    jwt_secret: ""
    token_expiry: "24h"
    
  # 限流配置
  rate_limit:
    enabled: true
    requests_per_second: 1000
    burst: 2000

# 开发配置
development:
  # 调试配置
  debug:
    enabled: true
    pprof: true
    pprof_port: 6060
    
  # 热重载配置
  hot_reload:
    enabled: true
    watch_paths:
      - "./configs"
      - "./internal"
    
  # 测试配置
  testing:
    mock_devices: true
    mock_kafka: false
    mock_redis: false
EOF

    log_success "Configuration file updated successfully"
}

# 验证配置文件
verify_config() {
    log_info "Verifying configuration file..."
    
    if [[ ! -f "$CONFIG_FILE" ]]; then
        log_error "Configuration file not found: $CONFIG_FILE"
        return 1
    fi
    
    # 检查YAML语法
    if command -v python3 &> /dev/null; then
        python3 -c "import yaml; yaml.safe_load(open('$CONFIG_FILE'))" 2>/dev/null
        if [[ $? -eq 0 ]]; then
            log_success "Configuration file syntax is valid"
        else
            log_error "Configuration file has syntax errors"
            return 1
        fi
    else
        log_warning "Python3 not found, skipping YAML syntax validation"
    fi
    
    # 检查关键配置项
    if grep -q "$SERVER_IP" "$CONFIG_FILE"; then
        log_success "Server IP ($SERVER_IP) configured correctly"
    else
        log_error "Server IP not found in configuration"
        return 1
    fi
    
    return 0
}

# 显示配置摘要
show_config_summary() {
    echo
    echo "📋 Configuration Summary:"
    echo "========================"
    echo "Server IP: $SERVER_IP"
    echo "Config File: $CONFIG_FILE"
    echo
    echo "Services configured:"
    echo "  - Kafka: $SERVER_IP:9092"
    echo "  - Redis: $SERVER_IP:6379 (password: redis123)"
    echo "  - PostgreSQL: $SERVER_IP:5432 (user: iot_user)"
    echo
    echo "Application settings:"
    echo "  - Device count: 50"
    echo "  - Sample interval: 5s"
    echo "  - Batch size: 10"
    echo "  - Worker pool: 10 workers"
    echo
    echo "🚀 Ready to start the application:"
    echo "   go run ./cmd/producer --config $CONFIG_FILE"
}

# 主函数
main() {
    echo "🔧 Industrial IoT Configuration Update"
    echo "====================================="
    echo "Target server: $SERVER_IP"
    echo "Config file: $CONFIG_FILE"
    echo

    # 创建配置目录
    mkdir -p "$(dirname "$CONFIG_FILE")"
    
    # 备份和更新配置
    backup_config
    update_config
    
    # 验证配置
    if verify_config; then
        show_config_summary
        log_success "Configuration update completed successfully!"
    else
        log_error "Configuration update failed!"
        exit 1
    fi
}

# 运行主函数
main "$@"
