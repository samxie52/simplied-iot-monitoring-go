# Industrial IoT Kafka Producer - 运维手册

## 📋 概述

本运维手册提供了 Industrial IoT Kafka Producer 系统的日常运维指南，包括监控、故障排查、性能调优和维护操作。

## 🎯 运维目标

### 服务等级目标 (SLO)
- **可用性**: 99.9% (每月最多43.2分钟停机)
- **响应时间**: 95%的请求在100ms内完成
- **吞吐量**: 支持10,000+ 消息/秒
- **错误率**: < 0.1%

### 关键性能指标 (KPI)
- **消息发送成功率**: > 99.9%
- **平均延迟**: < 50ms
- **内存使用率**: < 80%
- **CPU使用率**: < 70%

## 📊 监控和告警

### 1. 关键监控指标

#### 业务指标
```promql
# 消息发送速率
rate(kafka_messages_sent_total[5m])

# 错误率
rate(kafka_send_errors_total[5m]) / rate(kafka_messages_sent_total[5m]) * 100

# 设备数据生成速率
rate(device_data_generated_total[5m])

# 批处理效率
rate(kafka_batches_sent_total[5m]) / rate(kafka_messages_sent_total[5m])
```

#### 系统指标
```promql
# 内存使用率
process_resident_memory_bytes / 1024 / 1024

# CPU使用率
rate(process_cpu_seconds_total[5m]) * 100

# Goroutine数量
go_goroutines

# GC延迟
rate(go_gc_duration_seconds_sum[5m])
```

### 2. 告警规则配置

#### 关键告警
```yaml
# 高错误率告警
- alert: HighErrorRate
  expr: rate(kafka_send_errors_total[5m]) / rate(kafka_messages_sent_total[5m]) > 0.01
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "消息发送错误率过高"
    description: "错误率: {{ $value | humanizePercentage }}"

# 高延迟告警
- alert: HighLatency
  expr: histogram_quantile(0.95, rate(kafka_send_duration_seconds_bucket[5m])) > 0.1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "消息发送延迟过高"
    description: "95%延迟: {{ $value }}s"

# 内存使用告警
- alert: HighMemoryUsage
  expr: process_resident_memory_bytes / 1024 / 1024 > 400
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "内存使用率过高"
    description: "内存使用: {{ $value }}MB"
```

### 3. 监控仪表板

#### Grafana 仪表板关键面板
1. **系统概览**: 消息速率、错误率、延迟
2. **资源使用**: CPU、内存、网络
3. **业务指标**: 设备状态、数据生成
4. **告警状态**: 当前告警和历史趋势

## 🚨 故障排查指南

### 1. 常见故障场景

#### 场景1: 应用无法启动

**症状**:
- Pod处于CrashLoopBackOff状态
- 健康检查失败

**排查步骤**:
```bash
# 1. 查看Pod状态
kubectl get pods -n iot-monitoring

# 2. 查看Pod事件
kubectl describe pod <pod-name> -n iot-monitoring

# 3. 查看应用日志
kubectl logs <pod-name> -n iot-monitoring

# 4. 检查配置
kubectl get configmap iot-producer-config -n iot-monitoring -o yaml
```

**常见原因和解决方案**:
- **配置文件错误**: 验证YAML语法和字段名
- **依赖服务不可用**: 检查Kafka、数据库连接
- **资源不足**: 增加内存/CPU限制
- **权限问题**: 检查ServiceAccount和RBAC

#### 场景2: Kafka连接失败

**症状**:
- 大量连接错误日志
- 消息发送失败

**排查步骤**:
```bash
# 1. 测试Kafka连接
kubectl run kafka-test --rm -i --tty --image=confluentinc/cp-kafka:latest -- bash
kafka-console-producer --broker-list kafka:9092 --topic test

# 2. 检查网络策略
kubectl get networkpolicy -n iot-monitoring

# 3. 检查DNS解析
nslookup kafka-cluster.kafka.svc.cluster.local

# 4. 查看Kafka日志
kubectl logs -f deployment/kafka -n kafka
```

**解决方案**:
- 验证Kafka集群状态
- 检查网络连通性
- 更新broker地址配置
- 检查认证凭据

#### 场景3: 内存泄漏

**症状**:
- 内存使用持续增长
- Pod被OOMKilled

**排查步骤**:
```bash
# 1. 查看内存趋势
# 在Grafana中查看内存使用图表

# 2. 获取内存分析
kubectl port-forward pod/<pod-name> 6060:6060 -n iot-monitoring
go tool pprof http://localhost:6060/debug/pprof/heap

# 3. 分析GC情况
curl http://localhost:8080/metrics | grep go_gc

# 4. 检查Goroutine泄漏
curl http://localhost:6060/debug/pprof/goroutine?debug=1
```

**解决方案**:
- 调整GOGC参数
- 增加内存限制
- 修复代码中的内存泄漏
- 优化数据结构使用

### 2. 性能问题排查

#### 高延迟问题
```bash
# 1. 查看延迟分布
curl http://localhost:8080/metrics | grep kafka_send_duration_seconds

# 2. 分析CPU使用
kubectl top pod <pod-name> -n iot-monitoring

# 3. 检查网络延迟
ping kafka-cluster.kafka.svc.cluster.local

# 4. 分析慢查询
# 查看应用日志中的慢操作
```

#### 吞吐量下降
```bash
# 1. 检查批处理效率
curl http://localhost:8080/metrics | grep kafka_batches_sent_total

# 2. 查看队列深度
curl http://localhost:8080/metrics | grep kafka_producer_buffer_depth

# 3. 分析资源瓶颈
kubectl top pod <pod-name> -n iot-monitoring

# 4. 检查Kafka集群性能
# 查看Kafka监控指标
```

## 🔧 日常维护操作

### 1. 例行检查清单

#### 每日检查
- [ ] 检查应用健康状态
- [ ] 查看关键指标趋势
- [ ] 检查错误日志
- [ ] 验证告警配置

```bash
#!/bin/bash
# daily_check.sh

echo "=== 每日健康检查 ==="

# 检查Pod状态
echo "1. Pod状态检查:"
kubectl get pods -n iot-monitoring

# 检查健康端点
echo "2. 健康检查:"
kubectl port-forward svc/iot-producer-service 8081:8081 -n iot-monitoring &
sleep 2
curl -f http://localhost:8081/health || echo "健康检查失败"
kill %1

# 检查关键指标
echo "3. 关键指标:"
kubectl port-forward svc/iot-producer-service 8080:8080 -n iot-monitoring &
sleep 2
curl -s http://localhost:8080/metrics | grep -E "(kafka_messages_sent_total|kafka_send_errors_total)" | tail -5
kill %1

echo "=== 检查完成 ==="
```

#### 每周检查
- [ ] 分析性能趋势
- [ ] 检查资源使用情况
- [ ] 更新监控仪表板
- [ ] 备份配置文件

```bash
#!/bin/bash
# weekly_check.sh

echo "=== 每周维护检查 ==="

# 备份配置
kubectl get configmap iot-producer-config -n iot-monitoring -o yaml > backup/config-$(date +%Y%m%d).yaml

# 检查资源使用趋势
echo "资源使用情况:"
kubectl top pods -n iot-monitoring

# 清理旧日志
find /var/log/iot-producer -name "*.log" -mtime +7 -delete

echo "=== 每周检查完成 ==="
```

### 2. 配置管理

#### 配置更新流程
1. **准备阶段**
   ```bash
   # 备份当前配置
   kubectl get configmap iot-producer-config -n iot-monitoring -o yaml > backup/config-backup.yaml
   ```

2. **更新配置**
   ```bash
   # 编辑配置
   kubectl edit configmap iot-producer-config -n iot-monitoring
   
   # 或者应用新配置文件
   kubectl apply -f k8s/configmap.yaml
   ```

3. **验证更新**
   ```bash
   # 检查配置重载
   kubectl logs -f deployment/iot-producer -n iot-monitoring | grep "config reloaded"
   
   # 验证新配置生效
   curl http://localhost:8080/config
   ```

#### 密钥轮换
```bash
# 1. 生成新密钥
echo -n 'new-password' | base64

# 2. 更新Secret
kubectl patch secret iot-producer-secrets -n iot-monitoring -p '{"data":{"db-password":"bmV3LXBhc3N3b3Jk"}}'

# 3. 重启Pod使新密钥生效
kubectl rollout restart deployment/iot-producer -n iot-monitoring
```

### 3. 性能调优

#### 应用层调优
```yaml
# 调整批处理参数
producer:
  batch_size: 1000      # 增加批处理大小
  flush_interval: 500   # 减少刷新间隔
  worker_count: 8       # 增加工作线程

# 调整设备模拟参数
device:
  sample_interval: 1000 # 调整采样间隔
  data_variation: 0.1   # 调整数据变化幅度
```

#### Kubernetes资源调优
```yaml
resources:
  requests:
    memory: "512Mi"     # 增加内存请求
    cpu: "200m"         # 增加CPU请求
  limits:
    memory: "1Gi"       # 增加内存限制
    cpu: "1000m"        # 增加CPU限制
```

#### JVM调优 (如果适用)
```bash
export KAFKA_OPTS="-Xmx1g -Xms1g -XX:+UseG1GC -XX:MaxGCPauseMillis=20"
```

## 📈 容量规划

### 1. 资源需求评估

#### 计算公式
```
# 内存需求 (MB)
Memory = Base_Memory + (Device_Count * 2) + (Buffer_Size * 0.001)

# CPU需求 (millicores)
CPU = Base_CPU + (Message_Rate / 1000 * 10)

# 存储需求 (GB)
Storage = Log_Size + Config_Size + Temp_Files
```

#### 示例计算
```
设备数量: 1000
消息速率: 10,000/秒
批处理大小: 1000

内存需求: 256 + (1000 * 2) + (10000 * 0.001) = 2266 MB ≈ 2.3 GB
CPU需求: 100 + (10000 / 1000 * 10) = 200 millicores
```

### 2. 扩容策略

#### 水平扩容
```yaml
# HPA配置
spec:
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

#### 垂直扩容
```bash
# 增加资源限制
kubectl patch deployment iot-producer -n iot-monitoring -p '{"spec":{"template":{"spec":{"containers":[{"name":"iot-producer","resources":{"limits":{"memory":"2Gi","cpu":"1000m"}}}]}}}}'
```

## 🔄 备份和恢复

### 1. 配置备份

#### 自动备份脚本
```bash
#!/bin/bash
# backup_config.sh

BACKUP_DIR="/backup/iot-producer"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# 备份ConfigMap
kubectl get configmap iot-producer-config -n iot-monitoring -o yaml > $BACKUP_DIR/configmap-$DATE.yaml

# 备份Secret
kubectl get secret iot-producer-secrets -n iot-monitoring -o yaml > $BACKUP_DIR/secret-$DATE.yaml

# 备份Deployment
kubectl get deployment iot-producer -n iot-monitoring -o yaml > $BACKUP_DIR/deployment-$DATE.yaml

echo "备份完成: $BACKUP_DIR"
```

### 2. 灾难恢复

#### 恢复流程
```bash
# 1. 恢复命名空间
kubectl apply -f k8s/namespace.yaml

# 2. 恢复配置
kubectl apply -f backup/configmap-latest.yaml
kubectl apply -f backup/secret-latest.yaml

# 3. 恢复应用
kubectl apply -f backup/deployment-latest.yaml

# 4. 验证恢复
kubectl get pods -n iot-monitoring
curl http://localhost:8081/health
```

## 📞 应急响应

### 1. 应急联系人

| 角色 | 姓名 | 电话 | 邮箱 | 职责 |
|------|------|------|------|------|
| 主要负责人 | 张三 | 138-0000-0001 | zhang@company.com | 系统架构和重大故障 |
| 运维工程师 | 李四 | 138-0000-0002 | li@company.com | 日常运维和监控 |
| 开发工程师 | 王五 | 138-0000-0003 | wang@company.com | 代码问题和性能调优 |

### 2. 应急处理流程

#### P0级故障 (服务完全不可用)
1. **立即响应** (5分钟内)
   - 确认故障范围
   - 启动应急响应
   - 通知相关人员

2. **快速恢复** (15分钟内)
   - 执行回滚操作
   - 切换到备用系统
   - 恢复服务可用性

3. **根因分析** (2小时内)
   - 分析故障原因
   - 制定修复方案
   - 更新文档和流程

#### P1级故障 (部分功能不可用)
1. **响应时间**: 30分钟内
2. **修复时间**: 2小时内
3. **跟进措施**: 优化监控和告警

## 📋 运维检查表

### 部署前检查
- [ ] 配置文件语法正确
- [ ] 依赖服务可用
- [ ] 资源配额充足
- [ ] 监控告警配置
- [ ] 备份策略就绪

### 部署后验证
- [ ] 健康检查通过
- [ ] 指标数据正常
- [ ] 日志输出正常
- [ ] 性能指标达标
- [ ] 告警规则生效

### 日常运维
- [ ] 监控仪表板检查
- [ ] 日志分析
- [ ] 性能趋势分析
- [ ] 容量规划评估
- [ ] 安全漏洞扫描

---

*最后更新: 2025-08-01*  
*版本: v1.0.0*  
*维护团队: IoT Platform Team*
