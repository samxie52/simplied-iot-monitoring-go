# Docker 使用示例

## 多服务 Docker 构建和部署指南

本项目支持构建4个独立的微服务Docker镜像：producer、consumer、websocket、web。

### 🏗️ 构建Docker镜像

#### 构建所有服务镜像
```bash
make docker-build
```

#### 构建单个服务镜像
```bash
# 构建生产者服务
make docker-build-producer

# 构建消费者服务  
make docker-build-consumer

# 构建WebSocket服务
make docker-build-websocket

# 构建Web服务
make docker-build-web
```

#### 使用通用构建命令
```bash
# 构建指定服务
make docker-build-service SERVICE=producer
make docker-build-service SERVICE=consumer
make docker-build-service SERVICE=websocket
make docker-build-service SERVICE=web
```

### 🐳 直接使用Docker命令

#### 构建生产者服务镜像
```bash
docker build --target producer \
  --build-arg SERVICE=producer \
  --build-arg VERSION=v1.0.0 \
  --build-arg BUILD_TIME="$(date +%Y-%m-%d_%H:%M:%S)" \
  --build-arg GIT_COMMIT="$(git rev-parse HEAD)" \
  -t industrial-iot-producer:latest .
```

#### 构建消费者服务镜像
```bash
docker build --target consumer \
  --build-arg SERVICE=consumer \
  --build-arg VERSION=v1.0.0 \
  --build-arg BUILD_TIME="$(date +%Y-%m-%d_%H:%M:%S)" \
  --build-arg GIT_COMMIT="$(git rev-parse HEAD)" \
  -t industrial-iot-consumer:latest .
```

#### 构建web服务镜像
```bash
docker build --target web --build-arg SERVICE=web --build-arg VERSION=v1.0.0 --build-arg BUILD_TIME="$(date +%Y-%m-%d_%H:%M:%S)" --build-arg GIT_COMMIT="$(git rev-parse HEAD)" -t industrial-iot-web:latest .
```

### 🚀 运行Docker容器

#### 运行生产者服务
```bash
docker run -d \
  --name iot-producer \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs \
  -e CONFIG_PATH=/app/configs/development.yaml \
  industrial-iot-producer:latest start
```

#### 运行消费者服务
```bash
docker run -d \
  --name iot-consumer \
  -p 8081:8081 \
  -v $(pwd)/configs:/app/configs \
  -e CONFIG_PATH=/app/configs/development.yaml \
  industrial-iot-consumer:latest
```


#### 运行Web服务
```bash
docker run -d \
  --name iot-web \
  -p 8082:8082 \
  -v $(pwd)/configs:/app/configs \
  -v $(pwd)/web:/app/web \
  -e CONFIG_PATH=/app/configs/development.yaml \
  industrial-iot-web:latest
```

```
docker run -d --name iot-web -p 8082:8082 -v $(pwd)/configs:/app/configs -v $(pwd)/web:/app/web -e CONFIG_PATH=/app/configs/development.yaml industrial-iot-web:latest
```

### 📋 Docker Compose 示例

创建 `docker-compose.services.yml` 文件：

```yaml
version: '3.8'

services:
  iot-producer:
    image: industrial-iot-producer:latest
    build:
      context: .
      target: producer
      args:
        SERVICE: producer
        VERSION: v1.0.0
    ports:
      - "8080:8080"
    volumes:
      - ./configs:/app/configs
    environment:
      - CONFIG_PATH=/app/configs/development.yaml
      - LOG_LEVEL=info
    depends_on:
      - kafka
      - redis

  iot-consumer:
    image: industrial-iot-consumer:latest
    build:
      context: .
      target: consumer
      args:
        SERVICE: consumer
        VERSION: v1.0.0
    ports:
      - "8081:8081"
    volumes:
      - ./configs:/app/configs
    environment:
      - CONFIG_PATH=/app/configs/development.yaml
      - LOG_LEVEL=info
    depends_on:
      - kafka
      - redis

  iot-websocket:
    image: industrial-iot-websocket:latest
    build:
      context: .
      target: websocket
      args:
        SERVICE: websocket
        VERSION: v1.0.0
    ports:
      - "8083:8083"
    volumes:
      - ./configs:/app/configs
    environment:
      - CONFIG_PATH=/app/configs/development.yaml
      - LOG_LEVEL=info
    depends_on:
      - kafka
      - redis

  iot-web:
    image: industrial-iot-web:latest
    build:
      context: .
      target: web
      args:
        SERVICE: web
        VERSION: v1.0.0
    ports:
      - "8082:8082"
    volumes:
      - ./configs:/app/configs
      - ./web:/app/web
    environment:
      - CONFIG_PATH=/app/configs/development.yaml
      - LOG_LEVEL=info

  # 中间件服务
  kafka:
    image: confluentinc/cp-kafka:7.4.0
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    depends_on:
      - zookeeper

  zookeeper:
    image: confluentinc/cp-zookeeper:7.4.0
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

### 🔧 环境变量配置

每个服务支持以下环境变量：

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `CONFIG_PATH` | `/app/configs/development.yaml` | 配置文件路径 |
| `LOG_LEVEL` | `info` | 日志级别 |
| `TZ` | `Asia/Shanghai` | 时区设置 |
| `SERVICE_NAME` | `${SERVICE}` | 服务名称 |

### 📊 健康检查

所有服务都包含健康检查：

```bash
# 检查容器健康状态
docker ps

# 查看健康检查日志
docker inspect --format='{{json .State.Health}}' iot-producer
```

### 🔍 故障排查

#### 查看容器日志
```bash
docker logs -f --tail=20 iot-producer
docker logs -f --tail=20 iot-consumer
docker logs -f --tail=20 iot-websocket
docker logs -f --tail=20 iot-web
```

#### 进入容器调试
```bash
# 注意：由于使用alpine基础镜像，需要使用sh而不是bash
docker exec -it iot-producer sh
```

#### 检查服务状态
```bash
# 检查端口是否正常监听
docker exec iot-producer netstat -tlnp
```

### 📈 性能优化

1. **多阶段构建**：减少最终镜像大小
2. **Alpine基础镜像**：轻量级运行环境
3. **非root用户**：提高安全性
4. **健康检查**：确保服务可用性
5. **.dockerignore**：优化构建速度

### 🔐 安全最佳实践

1. 使用非root用户运行服务
2. 不在镜像中包含敏感配置
3. 通过环境变量或挂载卷提供配置
4. 定期更新基础镜像
5. 使用多阶段构建减少攻击面
