package producer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"simplied-iot-monitoring-go/internal/config"
)

// ConnectionPool Kafka连接池
type ConnectionPool struct {
	brokers     []string
	config      *sarama.Config
	connections chan sarama.Client
	maxSize     int
	currentSize int
	mutex       sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	isRunning   bool
}

// ConnectionHealthChecker 连接池健康检查器
type ConnectionHealthChecker struct {
	pool        *ConnectionPool
	producer    *KafkaProducer
	simulator   *DeviceSimulator
	checkInterval time.Duration
	timeout     time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	isRunning   bool
	mutex       sync.RWMutex
	lastCheck   time.Time
	healthStatus *config.HealthStatus
}

// ConnectionMetrics 连接池指标
type ConnectionMetrics struct {
	TotalConnections    int
	ActiveConnections   int
	IdleConnections     int
	FailedConnections   int
	ConnectionsCreated  int64
	ConnectionsDestroyed int64
	AverageLatency      time.Duration
	mutex               sync.RWMutex
}

// NewConnectionPool 创建连接池
func NewConnectionPool(brokers []string, kafkaConfig *config.KafkaProducer, maxSize int) (*ConnectionPool, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.ClientID = kafkaConfig.ClientID
	saramaConfig.Net.DialTimeout = kafkaConfig.Timeout
	saramaConfig.Net.ReadTimeout = kafkaConfig.Timeout
	saramaConfig.Net.WriteTimeout = kafkaConfig.Timeout

	ctx, cancel := context.WithCancel(context.Background())

	pool := &ConnectionPool{
		brokers:     brokers,
		config:      saramaConfig,
		connections: make(chan sarama.Client, maxSize),
		maxSize:     maxSize,
		currentSize: 0,
		ctx:         ctx,
		cancel:      cancel,
		isRunning:   false,
	}

	return pool, nil
}

// Start 启动连接池
func (cp *ConnectionPool) Start() error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if cp.isRunning {
		return fmt.Errorf("connection pool is already running")
	}

	// 预创建一些连接
	initialSize := cp.maxSize / 2
	if initialSize < 1 {
		initialSize = 1
	}

	for i := 0; i < initialSize; i++ {
		client, err := cp.createConnection()
		if err != nil {
			return fmt.Errorf("failed to create initial connection: %w", err)
		}
		cp.connections <- client
		cp.currentSize++
	}

	cp.isRunning = true
	return nil
}

// GetConnection 获取连接
func (cp *ConnectionPool) GetConnection() (sarama.Client, error) {
	if !cp.isRunning {
		return nil, fmt.Errorf("connection pool is not running")
	}

	select {
	case client := <-cp.connections:
		// 检查连接是否仍然有效
		if err := cp.validateConnection(client); err != nil {
			// 连接无效，创建新连接
			client.Close()
			cp.decrementCurrentSize()
			return cp.createAndReturnConnection()
		}
		return client, nil

	default:
		// 没有空闲连接，尝试创建新连接
		if cp.currentSize < cp.maxSize {
			return cp.createAndReturnConnection()
		}
		// 连接池已满，等待连接释放
		select {
		case client := <-cp.connections:
			if err := cp.validateConnection(client); err != nil {
				client.Close()
				cp.decrementCurrentSize()
				return cp.createAndReturnConnection()
			}
			return client, nil
		case <-time.After(5 * time.Second):
			return nil, fmt.Errorf("timeout waiting for connection")
		}
	}
}

// ReturnConnection 归还连接
func (cp *ConnectionPool) ReturnConnection(client sarama.Client) {
	if !cp.isRunning {
		client.Close()
		return
	}

	// 验证连接是否仍然有效
	if err := cp.validateConnection(client); err != nil {
		client.Close()
		cp.decrementCurrentSize()
		return
	}

	select {
	case cp.connections <- client:
		// 连接已归还到池中
	default:
		// 连接池已满，关闭连接
		client.Close()
		cp.decrementCurrentSize()
	}
}

// createConnection 创建新连接
func (cp *ConnectionPool) createConnection() (sarama.Client, error) {
	client, err := sarama.NewClient(cp.brokers, cp.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}
	return client, nil
}

// createAndReturnConnection 创建并返回新连接
func (cp *ConnectionPool) createAndReturnConnection() (sarama.Client, error) {
	client, err := cp.createConnection()
	if err != nil {
		return nil, err
	}
	cp.incrementCurrentSize()
	return client, nil
}

// validateConnection 验证连接
func (cp *ConnectionPool) validateConnection(client sarama.Client) error {
	if client.Closed() {
		return fmt.Errorf("connection is closed")
	}

	// 尝试获取broker列表来验证连接
	brokers := client.Brokers()
	if len(brokers) == 0 {
		return fmt.Errorf("no brokers available")
	}

	return nil
}

// incrementCurrentSize 增加当前连接数
func (cp *ConnectionPool) incrementCurrentSize() {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	cp.currentSize++
}

// decrementCurrentSize 减少当前连接数
func (cp *ConnectionPool) decrementCurrentSize() {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	if cp.currentSize > 0 {
		cp.currentSize--
	}
}

// Stop 停止连接池
func (cp *ConnectionPool) Stop() error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if !cp.isRunning {
		return nil
	}

	cp.cancel()

	// 关闭所有连接
	close(cp.connections)
	for client := range cp.connections {
		client.Close()
	}

	cp.currentSize = 0
	cp.isRunning = false
	return nil
}

// GetMetrics 获取连接池指标
func (cp *ConnectionPool) GetMetrics() *ConnectionMetrics {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	idleConnections := len(cp.connections)
	activeConnections := cp.currentSize - idleConnections

	return &ConnectionMetrics{
		TotalConnections:  cp.currentSize,
		ActiveConnections: activeConnections,
		IdleConnections:   idleConnections,
	}
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(pool *ConnectionPool, producer *KafkaProducer, simulator *DeviceSimulator) *ConnectionHealthChecker {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConnectionHealthChecker{
		pool:          pool,
		producer:      producer,
		simulator:     simulator,
		checkInterval: 30 * time.Second,
		timeout:       10 * time.Second,
		ctx:           ctx,
		cancel:        cancel,
		isRunning:     false,
		healthStatus: &config.HealthStatus{
			Healthy:   false,
			Timestamp: time.Now(),
			Services:  make(map[string]bool),
			Errors:    make([]string, 0),
			Metrics:   make(map[string]string),
		},
	}
}

// Start 启动健康检查
func (hc *ConnectionHealthChecker) Start() error {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if hc.isRunning {
		return fmt.Errorf("health checker is already running")
	}

	hc.isRunning = true
	go hc.healthCheckLoop()

	return nil
}

// healthCheckLoop 健康检查循环
func (hc *ConnectionHealthChecker) healthCheckLoop() {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	// 立即执行一次健康检查
	hc.performHealthCheck()

	for {
		select {
		case <-ticker.C:
			hc.performHealthCheck()

		case <-hc.ctx.Done():
			return
		}
	}
}

// performHealthCheck 执行健康检查
func (hc *ConnectionHealthChecker) performHealthCheck() {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	hc.lastCheck = time.Now()
	hc.healthStatus.Timestamp = hc.lastCheck
	hc.healthStatus.Errors = hc.healthStatus.Errors[:0] // 清空错误列表

	// 检查连接池
	poolHealthy := hc.checkConnectionPool()
	hc.healthStatus.Services["connection_pool"] = poolHealthy

	// 检查Kafka生产者
	producerHealthy := hc.checkKafkaProducer()
	hc.healthStatus.Services["kafka_producer"] = producerHealthy

	// 检查设备模拟器
	simulatorHealthy := hc.checkDeviceSimulator()
	hc.healthStatus.Services["device_simulator"] = simulatorHealthy

	// 更新整体健康状态
	hc.healthStatus.Healthy = poolHealthy && producerHealthy && simulatorHealthy

	// 更新指标
	hc.updateMetrics()
}

// checkConnectionPool 检查连接池健康状态
func (hc *ConnectionHealthChecker) checkConnectionPool() bool {
	if hc.pool == nil {
		hc.healthStatus.Errors = append(hc.healthStatus.Errors, "connection pool is nil")
		return false
	}

	metrics := hc.pool.GetMetrics()
	if metrics.TotalConnections == 0 {
		hc.healthStatus.Errors = append(hc.healthStatus.Errors, "no connections in pool")
		return false
	}

	// 尝试获取一个连接来测试
	client, err := hc.pool.GetConnection()
	if err != nil {
		hc.healthStatus.Errors = append(hc.healthStatus.Errors, fmt.Sprintf("failed to get connection: %v", err))
		return false
	}

	// 归还连接
	hc.pool.ReturnConnection(client)
	return true
}

// checkKafkaProducer 检查Kafka生产者健康状态
func (hc *ConnectionHealthChecker) checkKafkaProducer() bool {
	if hc.producer == nil {
		hc.healthStatus.Errors = append(hc.healthStatus.Errors, "kafka producer is nil")
		return false
	}

	if !hc.producer.IsRunning() {
		hc.healthStatus.Errors = append(hc.healthStatus.Errors, "kafka producer is not running")
		return false
	}

	metrics := hc.producer.GetMetrics()
	if metrics.SendErrors > 1000 { // 错误数量过多
		hc.healthStatus.Errors = append(hc.healthStatus.Errors, "too many send errors")
		return false
	}

	return true
}

// checkDeviceSimulator 检查设备模拟器健康状态
func (hc *ConnectionHealthChecker) checkDeviceSimulator() bool {
	if hc.simulator == nil {
		hc.healthStatus.Errors = append(hc.healthStatus.Errors, "device simulator is nil")
		return false
	}

	if !hc.simulator.IsRunning() {
		hc.healthStatus.Errors = append(hc.healthStatus.Errors, "device simulator is not running")
		return false
	}

	activeDevices := hc.simulator.GetActiveDeviceCount()
	if activeDevices == 0 {
		hc.healthStatus.Errors = append(hc.healthStatus.Errors, "no active devices")
		return false
	}

	return true
}

// updateMetrics 更新指标
func (hc *ConnectionHealthChecker) updateMetrics() {
	// 连接池指标
	if hc.pool != nil {
		poolMetrics := hc.pool.GetMetrics()
		hc.healthStatus.Metrics["pool_total_connections"] = fmt.Sprintf("%d", poolMetrics.TotalConnections)
		hc.healthStatus.Metrics["pool_active_connections"] = fmt.Sprintf("%d", poolMetrics.ActiveConnections)
		hc.healthStatus.Metrics["pool_idle_connections"] = fmt.Sprintf("%d", poolMetrics.IdleConnections)
	}

	// 生产者指标
	if hc.producer != nil {
		producerMetrics := hc.producer.GetMetrics()
		hc.healthStatus.Metrics["producer_messages_per_second"] = fmt.Sprintf("%d", producerMetrics.MessagesPerSecond)
		hc.healthStatus.Metrics["producer_send_errors"] = fmt.Sprintf("%d", producerMetrics.SendErrors)
		hc.healthStatus.Metrics["producer_queue_depth"] = fmt.Sprintf("%d", producerMetrics.QueueDepth)
	}

	// 模拟器指标
	if hc.simulator != nil {
		hc.healthStatus.Metrics["simulator_device_count"] = fmt.Sprintf("%d", hc.simulator.GetDeviceCount())
		hc.healthStatus.Metrics["simulator_active_devices"] = fmt.Sprintf("%d", hc.simulator.GetActiveDeviceCount())
	}

	hc.healthStatus.Metrics["last_check"] = hc.lastCheck.Format(time.RFC3339)
}

// GetHealthStatus 获取健康状态
func (hc *ConnectionHealthChecker) GetHealthStatus() *config.HealthStatus {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	// 返回健康状态副本
	status := &config.HealthStatus{
		Healthy:   hc.healthStatus.Healthy,
		Timestamp: hc.healthStatus.Timestamp,
		Services:  make(map[string]bool),
		Errors:    make([]string, len(hc.healthStatus.Errors)),
		Metrics:   make(map[string]string),
	}

	// 复制服务状态
	for k, v := range hc.healthStatus.Services {
		status.Services[k] = v
	}

	// 复制错误列表
	copy(status.Errors, hc.healthStatus.Errors)

	// 复制指标
	for k, v := range hc.healthStatus.Metrics {
		status.Metrics[k] = v
	}

	return status
}

// Stop 停止健康检查
func (hc *ConnectionHealthChecker) Stop() error {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if !hc.isRunning {
		return nil
	}

	hc.cancel()
	hc.isRunning = false
	return nil
}

// IsRunning 检查健康检查器是否运行中
func (hc *ConnectionHealthChecker) IsRunning() bool {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()
	return hc.isRunning
}

// SetCheckInterval 设置检查间隔
func (hc *ConnectionHealthChecker) SetCheckInterval(interval time.Duration) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.checkInterval = interval
}

// SetTimeout 设置超时时间
func (hc *ConnectionHealthChecker) SetTimeout(timeout time.Duration) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.timeout = timeout
}
