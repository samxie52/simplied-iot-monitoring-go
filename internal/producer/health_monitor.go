package producer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"simplied-iot-monitoring-go/internal/config"
)

// HealthStatus 健康状态枚举
type HealthStatus int

const (
	HealthStatusUnknown HealthStatus = iota
	HealthStatusHealthy
	HealthStatusWarning
	HealthStatusCritical
	HealthStatusDown
)

func (hs HealthStatus) String() string {
	switch hs {
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusWarning:
		return "warning"
	case HealthStatusCritical:
		return "critical"
	case HealthStatusDown:
		return "down"
	default:
		return "unknown"
	}
}

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message"`
	LastCheck   time.Time              `json:"last_check"`
	CheckCount  int64                  `json:"check_count"`
	ErrorCount  int64                  `json:"error_count"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SystemHealth 系统整体健康状态
type SystemHealth struct {
	OverallStatus HealthStatus                    `json:"overall_status"`
	Components    map[string]*ComponentHealth     `json:"components"`
	Timestamp     time.Time                       `json:"timestamp"`
	Uptime        time.Duration                   `json:"uptime"`
	Version       string                          `json:"version"`
	Metadata      map[string]interface{}          `json:"metadata"`
}

// HealthChecker 健康检查接口
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) *ComponentHealth
	IsEnabled() bool
}

// HealthMonitor 健康监控器
type HealthMonitor struct {
	checkers        []HealthChecker
	checkInterval   time.Duration
	timeout         time.Duration
	systemHealth    *SystemHealth
	metrics         *PrometheusMetrics
	startTime       time.Time
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	mutex           sync.RWMutex
	
	// 告警配置
	alertThresholds map[string]int64
	alertCallbacks  []AlertCallback
}

// AlertCallback 告警回调函数
type AlertCallback func(component string, health *ComponentHealth)

// NewHealthMonitor 创建健康监控器
func NewHealthMonitor(checkInterval, timeout time.Duration, metrics *PrometheusMetrics) *HealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &HealthMonitor{
		checkers:        make([]HealthChecker, 0),
		checkInterval:   checkInterval,
		timeout:         timeout,
		systemHealth:    &SystemHealth{
			Components: make(map[string]*ComponentHealth),
			Version:    "1.0.0",
			Metadata:   make(map[string]interface{}),
		},
		metrics:         metrics,
		startTime:       time.Now(),
		ctx:             ctx,
		cancel:          cancel,
		alertThresholds: make(map[string]int64),
		alertCallbacks:  make([]AlertCallback, 0),
	}
}

// RegisterChecker 注册健康检查器
func (hm *HealthMonitor) RegisterChecker(checker HealthChecker) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	
	hm.checkers = append(hm.checkers, checker)
	hm.systemHealth.Components[checker.Name()] = &ComponentHealth{
		Name:       checker.Name(),
		Status:     HealthStatusUnknown,
		Message:    "Not checked yet",
		LastCheck:  time.Time{},
		CheckCount: 0,
		ErrorCount: 0,
		Metadata:   make(map[string]interface{}),
	}
}

// RegisterAlertCallback 注册告警回调
func (hm *HealthMonitor) RegisterAlertCallback(callback AlertCallback) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	hm.alertCallbacks = append(hm.alertCallbacks, callback)
}

// SetAlertThreshold 设置告警阈值
func (hm *HealthMonitor) SetAlertThreshold(component string, threshold int64) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	hm.alertThresholds[component] = threshold
}

// Start 启动健康监控
func (hm *HealthMonitor) Start() error {
	hm.wg.Add(1)
	go hm.monitorLoop()
	return nil
}

// Stop 停止健康监控
func (hm *HealthMonitor) Stop() error {
	hm.cancel()
	hm.wg.Wait()
	return nil
}

// monitorLoop 监控循环
func (hm *HealthMonitor) monitorLoop() {
	defer hm.wg.Done()
	
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()
	
	// 立即执行一次检查
	hm.performHealthCheck()
	
	for {
		select {
		case <-ticker.C:
			hm.performHealthCheck()
		case <-hm.ctx.Done():
			return
		}
	}
}

// performHealthCheck 执行健康检查
func (hm *HealthMonitor) performHealthCheck() {
	hm.mutex.Lock()
	checkers := make([]HealthChecker, len(hm.checkers))
	copy(checkers, hm.checkers)
	hm.mutex.Unlock()
	
	// 并发执行所有健康检查
	var wg sync.WaitGroup
	results := make(chan *ComponentHealth, len(checkers))
	
	for _, checker := range checkers {
		if !checker.IsEnabled() {
			continue
		}
		
		wg.Add(1)
		go func(c HealthChecker) {
			defer wg.Done()
			
			ctx, cancel := context.WithTimeout(hm.ctx, hm.timeout)
			defer cancel()
			
			health := c.Check(ctx)
			if health != nil {
				results <- health
			}
		}(checker)
	}
	
	// 等待所有检查完成
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// 收集结果
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	
	for health := range results {
		oldHealth := hm.systemHealth.Components[health.Name]
		hm.systemHealth.Components[health.Name] = health
		
		// 更新Prometheus指标
		hm.metrics.SetComponentStatus(health.Name, health.Status == HealthStatusHealthy)
		
		// 检查是否需要触发告警
		hm.checkAlert(health, oldHealth)
	}
	
	// 更新系统整体状态
	hm.updateOverallStatus()
	hm.systemHealth.Timestamp = time.Now()
	hm.systemHealth.Uptime = time.Since(hm.startTime)
}

// updateOverallStatus 更新系统整体状态
func (hm *HealthMonitor) updateOverallStatus() {
	overallStatus := HealthStatusHealthy
	
	for _, component := range hm.systemHealth.Components {
		switch component.Status {
		case HealthStatusDown, HealthStatusCritical:
			overallStatus = HealthStatusCritical
		case HealthStatusWarning:
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusWarning
			}
		}
	}
	
	hm.systemHealth.OverallStatus = overallStatus
	hm.metrics.SetHealthStatus(overallStatus == HealthStatusHealthy)
}

// checkAlert 检查是否需要触发告警
func (hm *HealthMonitor) checkAlert(current, previous *ComponentHealth) {
	// 检查状态变化
	if previous != nil && current.Status != previous.Status {
		// 状态发生变化，触发告警
		for _, callback := range hm.alertCallbacks {
			go callback(current.Name, current)
		}
	}
	
	// 检查错误计数阈值
	if threshold, exists := hm.alertThresholds[current.Name]; exists {
		if current.ErrorCount >= threshold {
			for _, callback := range hm.alertCallbacks {
				go callback(current.Name, current)
			}
		}
	}
}

// GetHealth 获取系统健康状态
func (hm *HealthMonitor) GetHealth() *SystemHealth {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()
	
	// 返回深拷贝
	health := &SystemHealth{
		OverallStatus: hm.systemHealth.OverallStatus,
		Components:    make(map[string]*ComponentHealth),
		Timestamp:     hm.systemHealth.Timestamp,
		Uptime:        hm.systemHealth.Uptime,
		Version:       hm.systemHealth.Version,
		Metadata:      make(map[string]interface{}),
	}
	
	for name, component := range hm.systemHealth.Components {
		health.Components[name] = &ComponentHealth{
			Name:       component.Name,
			Status:     component.Status,
			Message:    component.Message,
			LastCheck:  component.LastCheck,
			CheckCount: component.CheckCount,
			ErrorCount: component.ErrorCount,
			Metadata:   make(map[string]interface{}),
		}
		
		for k, v := range component.Metadata {
			health.Components[name].Metadata[k] = v
		}
	}
	
	for k, v := range hm.systemHealth.Metadata {
		health.Metadata[k] = v
	}
	
	return health
}

// GetComponentHealth 获取特定组件健康状态
func (hm *HealthMonitor) GetComponentHealth(name string) *ComponentHealth {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()
	
	if component, exists := hm.systemHealth.Components[name]; exists {
		return &ComponentHealth{
			Name:       component.Name,
			Status:     component.Status,
			Message:    component.Message,
			LastCheck:  component.LastCheck,
			CheckCount: component.CheckCount,
			ErrorCount: component.ErrorCount,
			Metadata:   component.Metadata,
		}
	}
	
	return nil
}

// KafkaHealthChecker Kafka健康检查器
type KafkaHealthChecker struct {
	name     string
	producer *KafkaProducer
	enabled  bool
}

// NewKafkaHealthChecker 创建Kafka健康检查器
func NewKafkaHealthChecker(producer *KafkaProducer) *KafkaHealthChecker {
	return &KafkaHealthChecker{
		name:     "kafka_producer",
		producer: producer,
		enabled:  true,
	}
}

// Name 返回检查器名称
func (khc *KafkaHealthChecker) Name() string {
	return khc.name
}

// IsEnabled 检查是否启用
func (khc *KafkaHealthChecker) IsEnabled() bool {
	return khc.enabled
}

// Check 执行健康检查
func (khc *KafkaHealthChecker) Check(ctx context.Context) *ComponentHealth {
	health := &ComponentHealth{
		Name:       khc.name,
		LastCheck:  time.Now(),
		CheckCount: 0,
		ErrorCount: 0,
		Metadata:   make(map[string]interface{}),
	}
	
	if khc.producer == nil {
		health.Status = HealthStatusDown
		health.Message = "Kafka producer is not initialized"
		health.ErrorCount++
		return health
	}
	
	if !khc.producer.IsRunning() {
		health.Status = HealthStatusDown
		health.Message = "Kafka producer is not running"
		health.ErrorCount++
		return health
	}
	
	// 获取生产者指标
	metrics := khc.producer.GetMetrics()
	if metrics == nil {
		health.Status = HealthStatusWarning
		health.Message = "Unable to get producer metrics"
		return health
	}
	
	// 检查错误率
	errorRate := float64(metrics.SendErrors) / float64(metrics.MessagesPerSecond+1)
	if errorRate > 0.1 { // 错误率超过10%
		health.Status = HealthStatusCritical
		health.Message = fmt.Sprintf("High error rate: %.2f%%", errorRate*100)
		health.ErrorCount = metrics.SendErrors
	} else if errorRate > 0.05 { // 错误率超过5%
		health.Status = HealthStatusWarning
		health.Message = fmt.Sprintf("Elevated error rate: %.2f%%", errorRate*100)
		health.ErrorCount = metrics.SendErrors
	} else {
		health.Status = HealthStatusHealthy
		health.Message = "Kafka producer is healthy"
	}
	
	// 添加元数据
	health.Metadata["messages_per_second"] = metrics.MessagesPerSecond
	health.Metadata["bytes_per_second"] = metrics.BytesPerSecond
	health.Metadata["send_errors"] = metrics.SendErrors
	health.Metadata["queue_depth"] = metrics.QueueDepth
	health.Metadata["error_rate"] = errorRate
	
	health.CheckCount++
	return health
}

// ConnectionPoolHealthChecker 连接池健康检查器
type ConnectionPoolHealthChecker struct {
	name           string
	connectionPool *ConnectionPool
	enabled        bool
}

// NewConnectionPoolHealthChecker 创建连接池健康检查器
func NewConnectionPoolHealthChecker(pool *ConnectionPool) *ConnectionPoolHealthChecker {
	return &ConnectionPoolHealthChecker{
		name:           "connection_pool",
		connectionPool: pool,
		enabled:        true,
	}
}

// Name 返回检查器名称
func (cphc *ConnectionPoolHealthChecker) Name() string {
	return cphc.name
}

// IsEnabled 检查是否启用
func (cphc *ConnectionPoolHealthChecker) IsEnabled() bool {
	return cphc.enabled
}

// Check 执行健康检查
func (cphc *ConnectionPoolHealthChecker) Check(ctx context.Context) *ComponentHealth {
	health := &ComponentHealth{
		Name:       cphc.name,
		LastCheck:  time.Now(),
		CheckCount: 1,
		ErrorCount: 0,
		Metadata:   make(map[string]interface{}),
	}
	
	if cphc.connectionPool == nil {
		health.Status = HealthStatusDown
		health.Message = "Connection pool is not initialized"
		health.ErrorCount++
		return health
	}
	
	// 获取连接池统计
	stats := cphc.connectionPool.GetStats()
	
	activeConnections := stats["active_connections"].(int)
	maxConnections := stats["max_connections"].(int)
	utilizationRate := float64(activeConnections) / float64(maxConnections)
	
	if utilizationRate > 0.9 { // 使用率超过90%
		health.Status = HealthStatusCritical
		health.Message = fmt.Sprintf("High connection utilization: %.1f%%", utilizationRate*100)
	} else if utilizationRate > 0.7 { // 使用率超过70%
		health.Status = HealthStatusWarning
		health.Message = fmt.Sprintf("Elevated connection utilization: %.1f%%", utilizationRate*100)
	} else {
		health.Status = HealthStatusHealthy
		health.Message = "Connection pool is healthy"
	}
	
	// 添加元数据
	health.Metadata["active_connections"] = activeConnections
	health.Metadata["max_connections"] = maxConnections
	health.Metadata["utilization_rate"] = utilizationRate
	health.Metadata["total_created"] = stats["total_created"]
	health.Metadata["total_closed"] = stats["total_closed"]
	
	return health
}

// SystemResourceHealthChecker 系统资源健康检查器
type SystemResourceHealthChecker struct {
	name    string
	enabled bool
}

// NewSystemResourceHealthChecker 创建系统资源健康检查器
func NewSystemResourceHealthChecker() *SystemResourceHealthChecker {
	return &SystemResourceHealthChecker{
		name:    "system_resources",
		enabled: true,
	}
}

// Name 返回检查器名称
func (srhc *SystemResourceHealthChecker) Name() string {
	return srhc.name
}

// IsEnabled 检查是否启用
func (srhc *SystemResourceHealthChecker) IsEnabled() bool {
	return srhc.enabled
}

// Check 执行健康检查
func (srhc *SystemResourceHealthChecker) Check(ctx context.Context) *ComponentHealth {
	health := &ComponentHealth{
		Name:       srhc.name,
		LastCheck:  time.Now(),
		CheckCount: 1,
		ErrorCount: 0,
		Metadata:   make(map[string]interface{}),
	}
	
	// 检查内存使用情况
	memStats := config.ReadMemStats()
	
	memUsagePercent := float64(memStats.Alloc) / float64(memStats.Sys) * 100
	
	if memUsagePercent > 90 {
		health.Status = HealthStatusCritical
		health.Message = fmt.Sprintf("High memory usage: %.1f%%", memUsagePercent)
	} else if memUsagePercent > 75 {
		health.Status = HealthStatusWarning
		health.Message = fmt.Sprintf("Elevated memory usage: %.1f%%", memUsagePercent)
	} else {
		health.Status = HealthStatusHealthy
		health.Message = "System resources are healthy"
	}
	
	// 添加元数据
	health.Metadata["memory_usage_percent"] = memUsagePercent
	health.Metadata["memory_alloc"] = memStats.Alloc
	health.Metadata["memory_sys"] = memStats.Sys
	health.Metadata["gc_count"] = memStats.NumGC
	
	return health
}
