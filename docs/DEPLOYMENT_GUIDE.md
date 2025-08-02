# Industrial IoT Kafka Producer - 部署指南

## 📋 概述

本文档提供了 Industrial IoT Kafka Producer 系统的完整部署指南，包括本地开发、Docker容器化部署和Kubernetes生产环境部署。

## 🛠️ 系统要求

### 最低系统要求
- **CPU**: 2 核心
- **内存**: 4GB RAM
- **存储**: 10GB 可用空间
- **网络**: 稳定的网络连接

### 推荐生产环境要求
- **CPU**: 4+ 核心
- **内存**: 8GB+ RAM
- **存储**: 50GB+ SSD
- **网络**: 千兆网络

### 依赖服务
- **Kafka**: 2.8+ (推荐 3.0+)
- **Zookeeper**: 3.6+ (如果使用旧版Kafka)
- **Prometheus**: 2.30+ (监控)
- **Grafana**: 8.0+ (可视化)

## 🚀 快速开始

### 1. 本地开发部署

```bash
# 克隆项目
git clone <repository-url>
cd simplied-iot-monitoring-go

# 安装依赖
go mod download

# 复制配置文件
cp configs/development.yaml.example configs/development.yaml

# 编辑配置文件
vim configs/development.yaml

# 构建应用
go build -o bin/iot-producer ./cmd/producer

# 运行应用
./bin/iot-producer start --config configs/development.yaml
```

### 2. Docker 部署

```bash
# 构建Docker镜像
docker build -t industrial-iot-producer:v1.0.0 .

# 运行容器
docker run -d \
  --name iot-producer \
  -p 8080:8080 \
  -p 8081:8081 \
  -v $(pwd)/configs:/etc/config \
  -e CONFIG_PATH=/etc/config/production.yaml \
  industrial-iot-producer:v1.0.0
```

### 3. Docker Compose 部署

```yaml
# docker-compose.yml
version: '3.8'
services:
  iot-producer:
    image: industrial-iot-producer:v1.0.0
    ports:
      - "8080:8080"
      - "8081:8081"
    environment:
      - CONFIG_PATH=/etc/config/production.yaml
      - LOG_LEVEL=info
    volumes:
      - ./configs:/etc/config
    depends_on:
      - kafka
      - prometheus
    restart: unless-stopped

  kafka:
    image: confluentinc/cp-kafka:latest
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    ports:
      - "9092:9092"

  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
    ports:
      - "2181:2181"

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus-config.yaml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin123
```

## ☸️ Kubernetes 生产部署

### 1. 准备工作

```bash
# 创建命名空间
kubectl apply -f k8s/namespace.yaml

# 创建配置和密钥
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml

# 创建服务账户和RBAC
kubectl apply -f k8s/serviceaccount.yaml
```

### 2. 部署应用

```bash
# 部署应用
kubectl apply -f k8s/deployment.yaml

# 创建服务
kubectl apply -f k8s/service.yaml

# 配置自动扩缩容
kubectl apply -f k8s/hpa.yaml
```

### 3. 验证部署

```bash
# 检查Pod状态
kubectl get pods -n iot-monitoring

# 检查服务状态
kubectl get svc -n iot-monitoring

# 查看日志
kubectl logs -f deployment/iot-producer -n iot-monitoring

# 检查指标端点
kubectl port-forward svc/iot-producer-service 8080:8080 -n iot-monitoring
curl http://localhost:8080/metrics
```

## 📊 监控部署

### 1. Prometheus 配置

```bash
# 应用Prometheus配置
kubectl create configmap prometheus-config \
  --from-file=monitoring/prometheus-config.yaml \
  -n monitoring

# 部署Prometheus (使用Helm)
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --values monitoring/prometheus-values.yaml
```

### 2. Grafana 仪表板

```bash
# 导入仪表板
kubectl create configmap grafana-dashboard \
  --from-file=monitoring/grafana-dashboard.json \
  -n monitoring

# 或通过Grafana UI导入
# 1. 登录Grafana (admin/admin123)
# 2. 导航到 Dashboards > Import
# 3. 上传 monitoring/grafana-dashboard.json
```

## 🔧 配置管理

### 1. 环境配置

#### 开发环境 (development.yaml)
```yaml
environment: development
kafka:
  brokers: ["localhost:9092"]
device:
  device_count: 10
  sample_interval: 5000
logging:
  level: debug
```

#### 测试环境 (testing.yaml)
```yaml
environment: testing
kafka:
  brokers: ["kafka-test:9092"]
device:
  device_count: 50
  sample_interval: 2000
logging:
  level: info
```

#### 生产环境 (production.yaml)
```yaml
environment: production
kafka:
  brokers: ["kafka-cluster:9092"]
device:
  device_count: 100
  sample_interval: 1000
logging:
  level: warn
```

### 2. 配置热重载

系统支持配置文件热重载，无需重启应用：

```bash
# 修改配置文件
vim configs/production.yaml

# 配置会自动重载，查看日志确认
kubectl logs -f deployment/iot-producer -n iot-monitoring | grep "config reloaded"
```

## 🔒 安全配置

### 1. TLS/SSL 配置

```yaml
# 在配置文件中启用TLS
kafka:
  security:
    protocol: SSL
    ssl:
      ca_cert_file: /etc/ssl/certs/ca.pem
      cert_file: /etc/ssl/certs/client.pem
      key_file: /etc/ssl/private/client-key.pem
```

### 2. SASL 认证

```yaml
kafka:
  security:
    protocol: SASL_SSL
    sasl:
      mechanism: PLAIN
      username: ${KAFKA_USERNAME}
      password: ${KAFKA_PASSWORD}
```

### 3. 网络策略

```yaml
# k8s/network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: iot-producer-netpol
  namespace: iot-monitoring
spec:
  podSelector:
    matchLabels:
      app: industrial-iot-producer
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: kafka
    ports:
    - protocol: TCP
      port: 9092
```

## 📈 性能调优

### 1. JVM 调优 (如果使用Kafka客户端)

```bash
export KAFKA_OPTS="-Xmx512m -Xms512m -XX:+UseG1GC"
```

### 2. Go 应用调优

```bash
# 设置环境变量
export GOGC=100
export GOMAXPROCS=4
export GOMEMLIMIT=512MiB
```

### 3. Kubernetes 资源限制

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

## 🚨 故障排查

### 1. 常见问题

#### 应用无法启动
```bash
# 检查配置文件
kubectl describe configmap iot-producer-config -n iot-monitoring

# 检查密钥
kubectl describe secret iot-producer-secrets -n iot-monitoring

# 查看Pod事件
kubectl describe pod <pod-name> -n iot-monitoring
```

#### Kafka连接失败
```bash
# 测试Kafka连接
kubectl run kafka-test --rm -i --tty --image=confluentinc/cp-kafka:latest -- bash
kafka-console-producer --broker-list kafka:9092 --topic test
```

#### 内存不足
```bash
# 检查内存使用
kubectl top pods -n iot-monitoring

# 调整资源限制
kubectl patch deployment iot-producer -n iot-monitoring -p '{"spec":{"template":{"spec":{"containers":[{"name":"iot-producer","resources":{"limits":{"memory":"1Gi"}}}]}}}}'
```

### 2. 日志分析

```bash
# 查看应用日志
kubectl logs -f deployment/iot-producer -n iot-monitoring

# 查看特定时间段日志
kubectl logs deployment/iot-producer -n iot-monitoring --since=1h

# 查看错误日志
kubectl logs deployment/iot-producer -n iot-monitoring | grep ERROR
```

### 3. 性能分析

```bash
# 查看指标
curl http://localhost:8080/metrics

# 查看健康状态
curl http://localhost:8081/health

# 性能分析
go tool pprof http://localhost:8080/debug/pprof/profile
```

## 🔄 升级和回滚

### 1. 滚动升级

```bash
# 更新镜像
kubectl set image deployment/iot-producer iot-producer=industrial-iot-producer:v1.1.0 -n iot-monitoring

# 查看升级状态
kubectl rollout status deployment/iot-producer -n iot-monitoring
```

### 2. 回滚

```bash
# 查看升级历史
kubectl rollout history deployment/iot-producer -n iot-monitoring

# 回滚到上一版本
kubectl rollout undo deployment/iot-producer -n iot-monitoring

# 回滚到特定版本
kubectl rollout undo deployment/iot-producer --to-revision=2 -n iot-monitoring
```

## 📝 维护任务

### 1. 定期维护

```bash
# 清理旧日志
find /var/log/iot-producer -name "*.log" -mtime +7 -delete

# 清理旧指标数据
# (通过Prometheus配置retention策略)

# 备份配置
kubectl get configmap iot-producer-config -n iot-monitoring -o yaml > backup/config-$(date +%Y%m%d).yaml
```

### 2. 监控检查

```bash
# 检查应用健康状态
curl -f http://localhost:8081/health || echo "Health check failed"

# 检查指标收集
curl -s http://localhost:8080/metrics | grep kafka_messages_sent_total

# 检查告警状态
curl -s http://prometheus:9090/api/v1/alerts | jq '.data.alerts[] | select(.state=="firing")'
```

## 📞 支持和联系

- **文档**: [项目文档](./README.md)
- **问题报告**: [GitHub Issues](https://github.com/your-org/iot-producer/issues)
- **技术支持**: support@your-company.com

---

*最后更新: 2025-08-01*  
*版本: v1.0.0*
