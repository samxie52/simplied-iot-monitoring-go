# Industrial IoT 中间件部署指南

## 📋 中间件清单总览

基于你的Industrial IoT Kafka Producer系统配置，以下是需要部署的中间件和推荐镜像：

| 中间件 | 用途 | 推荐镜像 | 镜像大小 | 优先级 |
|--------|------|----------|----------|--------|
| **Kafka** | 消息队列 | `bitnami/kafka:latest` | 454MB | 🔴 必需 |
| **Redis** | 缓存/会话存储 | `redis:7-alpine` | 32MB | 🟡 推荐 |
| **PostgreSQL** | 关系数据库 | `postgres:15-alpine` | 78MB | 🟡 推荐 |
| **Prometheus** | 监控指标收集 | `prom/prometheus:latest` | 249MB | 🟡 推荐 |
| **Grafana** | 监控可视化 | `grafana/grafana:latest` | 398MB | 🟢 可选 |
| **Jaeger** | 分布式链路追踪 | `jaegertracing/all-in-one:latest` | 65MB | 🟢 可选 |

## 🐳 Docker 部署命令

### 1. Kafka (消息队列) - 🔴 必需

**推荐镜像**: `bitnami/kafka:latest`
**部署位置**: 192.168.5.16 (已部署)

```bash
# 在192.168.5.16服务器上运行
docker run -d \
  --name kafka-server \
  -p 9092:9092 \
  -e KAFKA_ENABLE_KRAFT=yes \
  -e KAFKA_CFG_PROCESS_ROLES=controller,broker \
  -e KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER \
  -e KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093 \
  -e KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT \
  -e KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://192.168.5.16:9092 \
  -e KAFKA_BROKER_ID=1 \
  -e KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=1@127.0.0.1:9093 \
  -e ALLOW_PLAINTEXT_LISTENER=yes \
  -e KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=true \
  -e KAFKA_CFG_LOG_RETENTION_HOURS=168 \
  -e KAFKA_CFG_NUM_PARTITIONS=3 \
  -e KAFKA_CFG_DEFAULT_REPLICATION_FACTOR=1 \
  -v kafka_data:/bitnami/kafka \
  --restart unless-stopped \
  bitnami/kafka:latest
```

### 2. Redis (缓存系统) - 🟡 推荐

**推荐镜像**: `redis:7-alpine` (轻量级，32MB)

```bash
# 在192.168.5.16服务器上运行
docker run -d \
  --name redis-server \
  -p 6379:6379 \
  -e REDIS_PASSWORD=redis123 \
  -v redis_data:/data \
  --restart unless-stopped \
  redis:7-alpine \
  redis-server --requirepass redis123 --appendonly yes
```

**验证Redis部署**:
```bash
# 测试连接
docker exec -it redis-server redis-cli -a redis123 ping
# 应该返回 PONG

# 测试基本操作
docker exec -it redis-server redis-cli -a redis123 set test "Hello IoT"
docker exec -it redis-server redis-cli -a redis123 get test
```

### 3. PostgreSQL (关系数据库) - 🟡 推荐

**推荐镜像**: `postgres:15-alpine` (轻量级，78MB)

```bash
# 在192.168.5.16服务器上运行
docker run -d \
  --name postgres-server \
  -p 5432:5432 \
  -e POSTGRES_DB=iot_monitoring \
  -e POSTGRES_USER=iot_user \
  -e POSTGRES_PASSWORD=iot_password123 \
  -e POSTGRES_INITDB_ARGS="--encoding=UTF-8 --lc-collate=C --lc-ctype=C" \
  -v postgres_data:/var/lib/postgresql/data \
  --restart unless-stopped \
  postgres:15-alpine
```

**验证PostgreSQL部署**:
```bash
# 测试连接
docker exec -it postgres-server psql -U iot_user -d iot_monitoring -c "SELECT version();"

# 创建IoT相关表
docker exec -it postgres-server psql -U iot_user -d iot_monitoring -c "
CREATE TABLE IF NOT EXISTS device_data (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    sensor_type VARCHAR(20),
    value DECIMAL(10,2),
    status VARCHAR(20)
);
"
```

### 4. Prometheus (监控指标收集) - 🟡 推荐

**推荐镜像**: `prom/prometheus:latest`

```bash
# 首先创建Prometheus配置文件
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

# 在192.168.5.16服务器上运行
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
  --web.enable-lifecycle
```

**验证Prometheus部署**:
```bash
# 访问Prometheus Web UI
curl http://192.168.5.16:9090/api/v1/label/__name__/values
```

### 5. Grafana (监控可视化) - 🟢 可选

**推荐镜像**: `grafana/grafana:latest`

```bash
# 在192.168.5.16服务器上运行
docker run -d \
  --name grafana-server \
  -p 3000:3000 \
  -e GF_SECURITY_ADMIN_USER=admin \
  -e GF_SECURITY_ADMIN_PASSWORD=admin123 \
  -e GF_INSTALL_PLUGINS=grafana-clock-panel,grafana-simple-json-datasource \
  -v grafana_data:/var/lib/grafana \
  --restart unless-stopped \
  grafana/grafana:latest
```

**验证Grafana部署**:
```bash
# 访问Grafana Web UI: http://192.168.5.16:3000
# 用户名: admin, 密码: admin123
curl -u admin:admin123 http://192.168.5.16:3000/api/health
```

### 6. Jaeger (分布式链路追踪) - 🟢 可选

**推荐镜像**: `jaegertracing/all-in-one:latest`

```bash
# 在192.168.5.16服务器上运行
docker run -d \
  --name jaeger-server \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -e COLLECTOR_OTLP_ENABLED=true \
  --restart unless-stopped \
  jaegertracing/all-in-one:latest
```

**验证Jaeger部署**:
```bash
# 访问Jaeger Web UI: http://192.168.5.16:16686
curl http://192.168.5.16:16686/api/services
```

## 🔧 配置文件更新

根据部署的中间件，更新你的配置文件：

### development.yaml 更新建议

```yaml
# Kafka配置 (已更新)
kafka:
  brokers: ["192.168.5.16:9092"]

# Redis配置
redis:
  host: "192.168.5.16"
  port: 6379
  password: "redis123"
  database: 1
  pool_size: 10

# 数据库配置
db:
  host: "192.168.5.16"
  port: 5432
  name: "iot_monitoring"
  user: "iot_user"
  password: "iot_password123"
  ssl_mode: "disable"
  max_open_conns: 25
  max_idle_conns: 5

# 监控配置
monitoring:
  prometheus:
    enabled: true
    port: 8080
    path: "/metrics"
  jaeger:
    enabled: true
    endpoint: "http://192.168.5.16:14268/api/traces"
```

## 🚀 一键部署脚本

创建一个便捷的部署脚本：

```bash
#!/bin/bash
# deploy_middleware.sh - 一键部署所有中间件

SERVER_IP="192.168.5.16"

echo "🚀 开始部署Industrial IoT中间件..."

# 1. 部署Kafka (如果未部署)
echo "📦 部署Kafka..."
docker run -d --name kafka-server -p 9092:9092 \
  -e KAFKA_ENABLE_KRAFT=yes \
  -e KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://$SERVER_IP:9092 \
  --restart unless-stopped bitnami/kafka:latest

# 2. 部署Redis
echo "📦 部署Redis..."
docker run -d --name redis-server -p 6379:6379 \
  -e REDIS_PASSWORD=redis123 \
  --restart unless-stopped redis:7-alpine \
  redis-server --requirepass redis123 --appendonly yes

# 3. 部署PostgreSQL
echo "📦 部署PostgreSQL..."
docker run -d --name postgres-server -p 5432:5432 \
  -e POSTGRES_DB=iot_monitoring \
  -e POSTGRES_USER=iot_user \
  -e POSTGRES_PASSWORD=iot_password123 \
  --restart unless-stopped postgres:15-alpine

# 4. 部署Prometheus
echo "📦 部署Prometheus..."
mkdir -p /tmp/prometheus
cat > /tmp/prometheus/prometheus.yml << 'EOF'
global:
  scrape_interval: 15s
scrape_configs:
  - job_name: 'iot-producer'
    static_configs:
      - targets: ['host.docker.internal:8080']
EOF

docker run -d --name prometheus-server -p 9090:9090 \
  -v /tmp/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml \
  --restart unless-stopped prom/prometheus:latest

# 5. 部署Grafana
echo "📦 部署Grafana..."
docker run -d --name grafana-server -p 3000:3000 \
  -e GF_SECURITY_ADMIN_PASSWORD=admin123 \
  --restart unless-stopped grafana/grafana:latest

echo "✅ 所有中间件部署完成!"
echo "🌐 访问地址:"
echo "  - Kafka: $SERVER_IP:9092"
echo "  - Redis: $SERVER_IP:6379"
echo "  - PostgreSQL: $SERVER_IP:5432"
echo "  - Prometheus: http://$SERVER_IP:9090"
echo "  - Grafana: http://$SERVER_IP:3000 (admin/admin123)"
```

## 📊 资源使用统计

| 中间件 | 内存使用 | CPU使用 | 存储需求 | 网络端口 |
|--------|----------|---------|----------|----------|
| Kafka | ~512MB | 低-中 | 1-5GB | 9092 |
| Redis | ~50MB | 低 | 100MB-1GB | 6379 |
| PostgreSQL | ~128MB | 低-中 | 500MB-2GB | 5432 |
| Prometheus | ~200MB | 低-中 | 1-10GB | 9090 |
| Grafana | ~150MB | 低 | 100MB-500MB | 3000 |
| Jaeger | ~100MB | 低 | 500MB-2GB | 16686,14268 |

**总计资源需求**:
- **内存**: ~1.2GB
- **存储**: ~5-20GB
- **端口**: 6个端口需要开放

## 🔒 安全配置建议

### 防火墙配置
```bash
# 在192.168.5.16服务器上配置防火墙
sudo ufw allow 9092  # Kafka
sudo ufw allow 6379  # Redis
sudo ufw allow 5432  # PostgreSQL
sudo ufw allow 9090  # Prometheus
sudo ufw allow 3000  # Grafana
sudo ufw allow 16686 # Jaeger
```

### 网络安全
- 考虑使用Docker网络隔离
- 为生产环境配置TLS/SSL
- 使用强密码和认证机制
- 限制外部访问权限

## 📝 验证部署清单

部署完成后，使用以下命令验证所有服务：

```bash
# 检查所有容器状态
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 测试各服务连通性
echo "Testing Kafka..."
telnet 192.168.5.16 9092

echo "Testing Redis..."
docker exec redis-server redis-cli -a redis123 ping

echo "Testing PostgreSQL..."
docker exec postgres-server pg_isready -U iot_user

echo "Testing Prometheus..."
curl -f http://192.168.5.16:9090/-/healthy

echo "Testing Grafana..."
curl -f http://192.168.5.16:3000/api/health
```

---

**部署建议**:
1. **最小化部署**: Kafka + Redis (核心功能)
2. **推荐部署**: + PostgreSQL + Prometheus (完整功能)
3. **完整部署**: + Grafana + Jaeger (全功能监控)

选择适合你需求的部署方案，逐步扩展中间件栈！
