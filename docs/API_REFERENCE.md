# Industrial IoT Kafka Producer - API 参考文档

## 📋 概述

本文档提供了 Industrial IoT Kafka Producer 系统的完整 API 参考，包括 HTTP 端点、指标接口、健康检查和管理接口。

## 🌐 HTTP API 端点

### 基础信息

- **Base URL**: `http://localhost:8080`
- **API Version**: v1
- **Content-Type**: `application/json`
- **Authentication**: 无 (内部服务)

### 1. 健康检查 API

#### GET /health
检查应用程序健康状态

**请求示例:**
```bash
curl -X GET http://localhost:8081/health
```

**响应示例:**
```json
{
  "status": "healthy",
  "timestamp": "2025-08-01T11:30:00Z",
  "version": "v1.0.0",
  "uptime": "2h30m15s",
  "checks": {
    "kafka": {
      "status": "healthy",
      "latency": "5ms",
      "last_check": "2025-08-01T11:29:55Z"
    },
    "database": {
      "status": "healthy",
      "connections": 5,
      "last_check": "2025-08-01T11:29:58Z"
    },
    "memory": {
      "status": "healthy",
      "usage": "45%",
      "available": "280MB"
    }
  }
}
```

**状态码:**
- `200 OK`: 应用程序健康
- `503 Service Unavailable`: 应用程序不健康

#### GET /ready
检查应用程序就绪状态

**请求示例:**
```bash
curl -X GET http://localhost:8081/ready
```

**响应示例:**
```json
{
  "status": "ready",
  "timestamp": "2025-08-01T11:30:00Z",
  "components": {
    "kafka_producer": "ready",
    "device_simulator": "ready",
    "metrics_collector": "ready"
  }
}
```

### 2. 指标 API

#### GET /metrics
获取 Prometheus 格式的指标数据

**请求示例:**
```bash
curl -X GET http://localhost:8080/metrics
```

**响应示例:**
```
# HELP kafka_messages_sent_total Total number of messages sent to Kafka
# TYPE kafka_messages_sent_total counter
kafka_messages_sent_total{topic="industrial-iot-data"} 12345

# HELP kafka_send_duration_seconds Time spent sending messages to Kafka
# TYPE kafka_send_duration_seconds histogram
kafka_send_duration_seconds_bucket{le="0.005"} 1000
kafka_send_duration_seconds_bucket{le="0.01"} 2000
kafka_send_duration_seconds_bucket{le="0.025"} 3000
kafka_send_duration_seconds_sum 45.5
kafka_send_duration_seconds_count 5000

# HELP device_data_generated_total Total device data points generated
# TYPE device_data_generated_total counter
device_data_generated_total{device_type="temperature"} 5000
device_data_generated_total{device_type="humidity"} 4800
device_data_generated_total{device_type="pressure"} 4900
```

### 3. 配置管理 API

#### GET /config
获取当前配置信息

**请求示例:**
```bash
curl -X GET http://localhost:8080/config
```

**响应示例:**
```json
{
  "environment": "production",
  "version": "v1.0.0",
  "kafka": {
    "brokers": ["kafka-cluster:9092"],
    "topic": "industrial-iot-data",
    "batch_size": 1000
  },
  "device": {
    "device_count": 100,
    "sample_interval": 1000
  },
  "last_reload": "2025-08-01T10:00:00Z"
}
```

#### POST /config/reload
重新加载配置文件

**请求示例:**
```bash
curl -X POST http://localhost:8080/config/reload
```

**响应示例:**
```json
{
  "status": "success",
  "message": "Configuration reloaded successfully",
  "timestamp": "2025-08-01T11:30:00Z",
  "changes": [
    "device.device_count: 100 -> 150",
    "device.sample_interval: 1000 -> 800"
  ]
}
```

### 4. 设备管理 API

#### GET /devices
获取设备列表和状态

**请求示例:**
```bash
curl -X GET http://localhost:8080/devices
```

**响应示例:**
```json
{
  "total_devices": 100,
  "active_devices": 98,
  "devices": [
    {
      "id": "device-001",
      "type": "temperature_sensor",
      "status": "active",
      "last_data": "2025-08-01T11:29:58Z",
      "location": {
        "building": "Factory-A",
        "floor": 2,
        "room": "Production-Line-1"
      },
      "sensor_data": {
        "temperature": 25.6,
        "status": "normal"
      }
    }
  ]
}
```

#### GET /devices/{device_id}
获取特定设备详细信息

**请求示例:**
```bash
curl -X GET http://localhost:8080/devices/device-001
```

**响应示例:**
```json
{
  "id": "device-001",
  "type": "temperature_sensor",
  "status": "active",
  "created_at": "2025-08-01T09:00:00Z",
  "last_data": "2025-08-01T11:29:58Z",
  "location": {
    "building": "Factory-A",
    "floor": 2,
    "room": "Production-Line-1",
    "coordinates": {
      "x": 10.5,
      "y": 20.3
    }
  },
  "sensor_data": {
    "temperature": 25.6,
    "humidity": 45.2,
    "status": "normal",
    "battery_level": 85
  },
  "statistics": {
    "messages_sent": 1500,
    "last_24h_avg": 24.8,
    "error_count": 0
  }
}
```

### 5. 统计信息 API

#### GET /stats
获取系统统计信息

**请求示例:**
```bash
curl -X GET http://localhost:8080/stats
```

**响应示例:**
```json
{
  "uptime": "2h30m15s",
  "start_time": "2025-08-01T09:00:00Z",
  "kafka": {
    "messages_sent": 125000,
    "messages_per_second": 45.2,
    "errors": 12,
    "error_rate": 0.0096,
    "avg_latency_ms": 15.6
  },
  "devices": {
    "total": 100,
    "active": 98,
    "inactive": 2,
    "data_points_generated": 150000
  },
  "system": {
    "memory_usage_mb": 280,
    "cpu_usage_percent": 35.2,
    "goroutines": 45,
    "gc_cycles": 156
  },
  "batching": {
    "batches_sent": 1250,
    "avg_batch_size": 100,
    "batch_efficiency": 0.95
  }
}
```

### 6. 管理操作 API

#### POST /admin/shutdown
优雅关闭应用程序

**请求示例:**
```bash
curl -X POST http://localhost:8080/admin/shutdown \
  -H "Content-Type: application/json" \
  -d '{"timeout": 30}'
```

**响应示例:**
```json
{
  "status": "shutting_down",
  "message": "Graceful shutdown initiated",
  "timeout": 30,
  "timestamp": "2025-08-01T11:30:00Z"
}
```

#### POST /admin/flush
强制刷新所有缓冲区

**请求示例:**
```bash
curl -X POST http://localhost:8080/admin/flush
```

**响应示例:**
```json
{
  "status": "success",
  "message": "All buffers flushed successfully",
  "flushed_messages": 245,
  "timestamp": "2025-08-01T11:30:00Z"
}
```

## 📊 指标详细说明

### Kafka 相关指标

| 指标名称 | 类型 | 描述 |
|---------|------|------|
| `kafka_messages_sent_total` | Counter | 发送到Kafka的消息总数 |
| `kafka_send_errors_total` | Counter | 发送失败的消息总数 |
| `kafka_send_duration_seconds` | Histogram | 消息发送耗时分布 |
| `kafka_batches_sent_total` | Counter | 发送的批次总数 |
| `kafka_producer_buffer_depth` | Gauge | 生产者缓冲区深度 |
| `kafka_connection_failures_total` | Counter | Kafka连接失败次数 |

### 设备相关指标

| 指标名称 | 类型 | 描述 |
|---------|------|------|
| `device_data_generated_total` | Counter | 生成的设备数据点总数 |
| `device_count_active` | Gauge | 活跃设备数量 |
| `device_errors_total` | Counter | 设备错误总数 |
| `device_data_generation_rate` | Gauge | 数据生成速率 |

### 系统相关指标

| 指标名称 | 类型 | 描述 |
|---------|------|------|
| `process_resident_memory_bytes` | Gauge | 进程内存使用量 |
| `process_cpu_seconds_total` | Counter | 进程CPU使用时间 |
| `go_goroutines` | Gauge | Goroutine数量 |
| `go_gc_duration_seconds` | Summary | GC耗时 |

## 🔧 配置参数

### HTTP 服务器配置

```yaml
web:
  port: 8080              # HTTP服务端口
  health_port: 8081       # 健康检查端口
  read_timeout: 30        # 读取超时(秒)
  write_timeout: 30       # 写入超时(秒)
  max_header_bytes: 1048576  # 最大请求头大小
```

### API 限制配置

```yaml
api:
  rate_limit: 1000        # 每分钟请求限制
  max_connections: 100    # 最大并发连接数
  timeout: 30             # API超时时间(秒)
```

## 🚨 错误代码

### HTTP 状态码

| 状态码 | 描述 | 示例场景 |
|--------|------|----------|
| 200 | 成功 | 正常请求 |
| 400 | 请求错误 | 参数格式错误 |
| 404 | 资源不存在 | 设备ID不存在 |
| 500 | 内部服务器错误 | 系统异常 |
| 503 | 服务不可用 | 健康检查失败 |

### 错误响应格式

```json
{
  "error": {
    "code": "DEVICE_NOT_FOUND",
    "message": "Device with ID 'device-999' not found",
    "timestamp": "2025-08-01T11:30:00Z",
    "request_id": "req-12345"
  }
}
```

## 🔐 安全考虑

### 1. 访问控制
- 健康检查端点 (8081) 仅限内部网络访问
- 管理端点需要额外的认证机制
- 指标端点应限制访问来源

### 2. 速率限制
```yaml
# 建议的Nginx配置
location /api/ {
    limit_req zone=api burst=20 nodelay;
    proxy_pass http://iot-producer:8080;
}
```

### 3. 日志记录
所有API请求都会记录访问日志：
```
2025-08-01T11:30:00Z INFO api_access method=GET path=/health status=200 duration=5ms client_ip=10.0.0.1
```

## 📝 使用示例

### 1. 监控脚本示例

```bash
#!/bin/bash
# health_check.sh

HEALTH_URL="http://localhost:8081/health"
METRICS_URL="http://localhost:8080/metrics"

# 检查健康状态
health_status=$(curl -s -o /dev/null -w "%{http_code}" $HEALTH_URL)
if [ $health_status -eq 200 ]; then
    echo "✅ Application is healthy"
else
    echo "❌ Application is unhealthy (HTTP $health_status)"
    exit 1
fi

# 获取关键指标
messages_sent=$(curl -s $METRICS_URL | grep kafka_messages_sent_total | awk '{print $2}')
echo "📊 Messages sent: $messages_sent"
```

### 2. Python 客户端示例

```python
import requests
import json

class IoTProducerClient:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
    
    def get_health(self):
        response = requests.get(f"{self.base_url}:8081/health")
        return response.json()
    
    def get_devices(self):
        response = requests.get(f"{self.base_url}/devices")
        return response.json()
    
    def get_stats(self):
        response = requests.get(f"{self.base_url}/stats")
        return response.json()
    
    def reload_config(self):
        response = requests.post(f"{self.base_url}/config/reload")
        return response.json()

# 使用示例
client = IoTProducerClient()
health = client.get_health()
print(f"Application status: {health['status']}")
```

## 📞 支持信息

- **API版本**: v1.0.0
- **文档版本**: 2025-08-01
- **技术支持**: support@your-company.com
- **GitHub**: https://github.com/your-org/iot-producer

---

*最后更新: 2025-08-01*  
*API版本: v1.0.0*
