# 🚀 Industrial IoT 系统完整部署指南

欢迎使用 Industrial IoT 实时监控系统！本指南将帮助您完成生产者、消费者和Web服务的完整部署流程。

## 📋 系统架构概览

本系统包含三个核心服务：

- **🏗️ Kafka生产者**: 模拟工业IoT设备数据生成
- **📊 Kafka消费者**: 实时处理设备数据和告警
- **🌐 Web服务**: 实时数据可视化和监控界面

## 📋 前置要求

- **Docker**: 已安装并运行
- **Go**: 版本 1.19+
- **网络**: 能够访问中间件服务器 (默认配置中指定)
- **系统**: macOS/Linux (推荐)
- **端口**: 确保 8080, 8090, 9090 端口可用

## ⚙️ 配置说明

在开始部署之前，请检查和更新以下配置文件：

```bash
# 编辑主配置文件
vim configs/development.yaml

# 更新以下关键配置：
# - Kafka brokers: 修改为您的Kafka服务器地址
# - Redis host: 修改为您的Redis服务器地址
# - Database host: 修改为您的PostgreSQL服务器地址
# - Prometheus/Grafana URLs: 修改为您的监控服务器地址
```

**示例配置替换：**

```yaml
kafka:
  brokers: ["<KAFKA_HOST>:9092"]  # 替换为实际Kafka服务器地址

redis:
  host: "<REDIS_HOST>"  # 替换为实际Redis服务器地址

db:
  host: "<DATABASE_HOST>"  # 替换为实际数据库服务器地址
```

## ⚡ 一键部署 (推荐)

### 完整自动化部署

```bash
# 克隆项目并进入目录
git clone <repository-url>
cd simplied-iot-monitoring-go

# 一键部署所有服务 (推荐配置)
./scripts/full_deployment.sh recommended

# 或者部署完整版本 (包含Grafana和Jaeger)
./scripts/full_deployment.sh full
```

**自动化部署包含：**

- ✅ 中间件服务部署 (Kafka, Redis, PostgreSQL, Prometheus)
- ✅ 配置文件自动更新
- ✅ 三个服务程序构建 (producer, consumer, web)
- ✅ 自动化测试验证
- ✅ 服务启动和健康检查
- ✅ 集成测试验证

## 🏗️ 分步骤部署流程

### 步骤1: 部署中间件服务

```bash
# 部署Kafka、Redis、PostgreSQL、Prometheus等中间件
./scripts/deploy_middleware.sh recommended

# 验证中间件服务状态
docker ps -a
docker logs kafka-server
docker logs redis-server
```

### 步骤2: 更新配置文件

```bash
# 自动更新配置文件以连接中间件服务
./scripts/update_config.sh

# 验证配置文件
cat configs/development.yaml
```

### 步骤3: 构建所有服务

```bash
# 安装依赖
go mod tidy

# 构建生产者服务
go build -o bin/producer ./cmd/producer

# 构建消费者服务
go build -o bin/consumer ./cmd/consumer

# 构建Web服务
go build -o bin/web ./cmd/web

# 验证构建结果
ls -la bin/
```

## 🚀 服务启动和管理

### Kafka生产者服务

```bash
# 启动生产者 (前台运行)
./bin/producer start --config configs/development.yaml --verbose

# 启动生产者 (后台运行)
nohup ./bin/producer start --config configs/development.yaml --verbose > logs/producer.log 2>&1 &
echo $! > .producer.pid

# 查看生产者状态
./bin/producer status --config configs/development.yaml

# 查看生产者帮助
./bin/producer --help

# 停止生产者
./bin/producer stop --config configs/development.yaml
# 或者使用PID文件
kill $(cat .producer.pid)
```

### Kafka消费者服务

```bash
# 启动消费者 (前台运行)
./bin/consumer start --config configs/development.yaml --verbose

# 启动消费者 (后台运行)
nohup ./bin/consumer start --config configs/development.yaml --verbose > logs/consumer.log 2>&1 &
echo $! > .consumer.pid

# 查看消费者状态
./bin/consumer status --config configs/development.yaml

# 查看消费者健康状态
./bin/consumer health --config configs/development.yaml

# 停止消费者
kill $(cat .consumer.pid)
```

### Web监控服务

```bash
# 启动Web服务 (前台运行)
./bin/web --port 8090

# 启动Web服务 (后台运行)
nohup ./bin/web --port 8090 > logs/web.log 2>&1 &
echo $! > .web.pid

# 停止Web服务
kill $(cat .web.pid)
```

## 🔍 系统验证和监控

### 健康检查

```bash
# 检查生产者健康状态
curl http://localhost:8080/health

# 检查Web服务健康状态
curl http://localhost:8090/health

# 检查API状态
curl http://localhost:8090/api/status

# 查看系统指标
curl http://localhost:8080/metrics
```

### 访问监控界面

- **🌐 Web监控界面**: http://localhost:8090
- **📊 生产者API**: http://localhost:8080
- **💓 健康检查**: http://localhost:8090/health
- **📈 Prometheus**: http://<MIDDLEWARE_HOST>:9090
- **📊 Grafana**: http://<MIDDLEWARE_HOST>:3000 (admin/admin123)
- **🔗 WebSocket测试**: ws://localhost:8090/ws

### 日志查看

```bash
# 查看生产者日志
tail -f logs/producer.log

# 查看消费者日志
tail -f logs/consumer.log

# 查看Web服务日志
tail -f logs/web.log

# 查看Docker容器日志
docker logs -f kafka-server
docker logs -f redis-server
docker logs -f postgres-db

# 查看系统所有日志
tail -f logs/*.log
```

### 实时数据监控

```bash
# 监控Kafka消息生产
./bin/producer stats --config configs/development.yaml

# 监控Kafka消息消费
./bin/consumer stats --config configs/development.yaml

# 监控WebSocket连接
curl http://localhost:8090/api/websocket/stats

# 查看设备状态统计
curl http://localhost:8090/api/devices/stats
```

## 🚀 性能测试和基准测试

### 生产者性能测试

```bash
# 运行生产者基准测试
go test -bench=BenchmarkProducer -benchmem ./internal/producer/...

# 运行完整的性能测试套件
go test -bench=. -benchmem ./tests/performance/...

# 测试高并发场景
./bin/producer benchmark --config configs/development.yaml --devices 10000 --duration 60s
```

### 消费者性能测试

```bash
# 运行消费者基准测试
go test -bench=BenchmarkConsumer -benchmem ./internal/consumer/...

# 测试消息处理性能
./bin/consumer benchmark --config configs/development.yaml --messages 100000
```

### WebSocket性能测试

```bash
# 测试WebSocket并发连接
go test -bench=BenchmarkWebSocket -benchmem ./internal/websocket/...

# 使用外部工具测试WebSocket
npm install -g wscat
wscat -c ws://localhost:8090/ws
```

### 系统集成测试

```bash
# 运行完整的集成测试
go test -v ./tests/integration/...

# 运行端到端测试
go test -v ./tests/e2e/...

# 运行所有测试
go test -v ./...
```

## 📊 监控和管理

### 实时日志查看

```bash
# 查看各服务日志
tail -f logs/producer.log
tail -f logs/consumer.log
tail -f logs/web.log

# 查看所有日志
tail -f logs/*.log

# 查看Docker容器日志
docker logs kafka-server
docker logs redis-server
docker logs postgres-db
docker logs prometheus-server
```

### 性能测试

```bash
# 运行基准测试
go test -v ./tests/producer/benchmark_test.go -bench=. -benchmem

# 运行端到端测试
go test -v ./tests/e2e/end_to_end_test.go -timeout=120s
```

## 🛠️ 常用操作

### 重启所有服务

```bash
# 停止所有服务
kill $(cat .producer.pid) 2>/dev/null || true
kill $(cat .consumer.pid) 2>/dev/null || true
kill $(cat .web.pid) 2>/dev/null || true

# 重新启动所有服务
nohup ./bin/producer start --config configs/development.yaml > logs/producer.log 2>&1 &
echo $! > .producer.pid

nohup ./bin/consumer start --config configs/development.yaml > logs/consumer.log 2>&1 &
echo $! > .consumer.pid

nohup ./bin/web --port 8090 > logs/web.log 2>&1 &
echo $! > .web.pid
```

### 清理和重置

```bash
# 停止所有服务和清理
./scripts/full_deployment.sh --cleanup

# 重新部署
./scripts/full_deployment.sh recommended
```

### 配置修改

```bash
# 编辑配置文件
vim configs/development.yaml

# 重启相关服务以应用新配置
# 重启生产者
kill $(cat .producer.pid) 2>/dev/null || true
nohup ./bin/producer start --config configs/development.yaml > logs/producer.log 2>&1 &
echo $! > .producer.pid

# 重启消费者
kill $(cat .consumer.pid) 2>/dev/null || true
nohup ./bin/consumer start --config configs/development.yaml > logs/consumer.log 2>&1 &
echo $! > .consumer.pid
```

### 检查服务状态

```bash
# 检查所有服务进程
ps aux | grep -E "(producer|consumer|web)" | grep -v grep

# 检查PID文件
ls -la .*.pid

# 检查服务健康状态
curl -s http://localhost:8080/health | jq .
curl -s http://localhost:8090/health | jq .
```

## 🔧 故障排除和常见问题

### 常见问题解决

**1. Kafka连接失败**

```bash
# 检查Kafka服务状态
docker ps | grep kafka
docker logs kafka-server

# 重启 Kafka服务
docker restart kafka-server

# 检查网络连接
telnet <KAFKA_HOST> 9092
```

**2. 数据库连接问题**

```bash
# 检查PostgreSQL状态
docker ps | grep postgres
docker logs postgres-db

# 测试数据库连接
psql -h <DATABASE_HOST> -U iot_user -d iot_monitoring

# 重启数据库
docker restart postgres-db
```

**3. WebSocket连接问题**

```bash
# 检查Web服务状态
curl http://localhost:8090/health

# 测试WebSocket连接
wscat -c ws://localhost:8090/ws

# 检查端口占用
lsof -i :8090
```

**4. 内存不足问题**

```bash
# 检查系统资源使用
top -p $(pgrep -f "producer|consumer|web")

# 检查内存使用
free -h

# 调整JVM堆内存(如果使用Java组件)
export KAFKA_HEAP_OPTS="-Xmx2G -Xms2G"
```

### 性能优化建议

**1. 生产者优化**

```bash
# 调整批处理大小
./bin/producer start --config configs/development.yaml --batch-size 1000

# 调整发送间隔
./bin/producer start --config configs/development.yaml --send-interval 100ms

# 启用压缩
./bin/producer start --config configs/development.yaml --compression gzip
```

**2. 消费者优化**

```bash
# 调整并发数
./bin/consumer start --config configs/development.yaml --workers 10

# 调整缓冲区大小
./bin/consumer start --config configs/development.yaml --buffer-size 10000
```

**3. WebSocket优化**

```bash
# 调整最大连接数
./bin/web --max-connections 10000

# 调整心跳间隔
./bin/web --heartbeat-interval 30s
```

### 日志分析

```bash
# 查看所有服务的错误日志
grep -i error logs/*.log

# 查看性能指标
grep -i "metrics" logs/*.log

# 实时监控所有日志
tail -f logs/*.log | grep -E "(ERROR|WARN|metrics)"

# 分别查看各服务日志
grep -i error logs/producer.log
grep -i error logs/consumer.log
grep -i error logs/web.log
```

## 📈 性能基准

系统已经过全面优化，具备以下性能特征：

- **零拷贝优化**: 0.26 ns/op (192倍性能提升)
- **内存池管理**: 19.79-116.4 ns/op
- **Prometheus指标**: 6.883 ns/op
- **并发处理**: 支持高并发安全操作
- **消息吞吐**: 每秒处理数千条IoT消息

## 🎯 生产环境部署

对于生产环境，请参考详细文档：

- [部署指南](docs/DEPLOYMENT_GUIDE.md) - 完整的生产部署流程
- [API参考](docs/API_REFERENCE.md) - 详细的API文档
- [运维手册](docs/OPERATIONS_MANUAL.md) - 企业级运维指南
- [中间件部署](docs/MIDDLEWARE_DEPLOYMENT_GUIDE.md) - 中间件详细配置

## 🆘 获取帮助

```bash
# 查看脚本帮助
./scripts/full_deployment.sh --help
./scripts/deploy_middleware.sh help
./scripts/update_config.sh --help

# 查看应用帮助
./bin/producer --help
```

## 🎉 成功

如果您看到以下输出，说明系统已成功运行：

```text
✅ IoT Producer: Running (PID: xxxx)
✅ Port 8080: In use
✅ Health Check: Passed
✅ Metrics: Available
```

您的 Industrial IoT 系统现已准备就绪，可以开始处理工业物联网数据！

---

**技术支持**: 如遇问题，请查看 `logs/` 目录下的服务日志文件或参考详细文档。
