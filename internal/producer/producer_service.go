package producer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"simplied-iot-monitoring-go/internal/config"
)

// ProducerService 生产者服务
type ProducerService struct {
	config         *config.AppConfig
	kafkaProducer  *KafkaProducer
	deviceManager  *DeviceManager
	deviceSimulator *DeviceSimulator
	connectionPool *ConnectionPool
	healthChecker  *ConnectionHealthChecker
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	isRunning      bool
	mutex          sync.RWMutex
}

// ServiceMetrics 服务指标
type ServiceMetrics struct {
	StartTime         time.Time
	Uptime            time.Duration
	TotalMessages     int64
	TotalErrors       int64
	DevicesActive     int
	DevicesTotal      int
	ConnectionsActive int
	ConnectionsTotal  int
	MemoryUsage       int64
	CPUUsage          float64
	mutex             sync.RWMutex
}

// NewProducerService 创建生产者服务
func NewProducerService(cfg *config.AppConfig) (*ProducerService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	service := &ProducerService{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
		isRunning: false,
	}

	// 初始化组件
	if err := service.initializeComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return service, nil
}

// initializeComponents 初始化组件
func (ps *ProducerService) initializeComponents() error {
	var err error

	// 1. 创建连接池
	ps.connectionPool, err = NewConnectionPool(
		ps.config.Kafka.Brokers,
		&ps.config.Kafka.Producer,
		20, // 最大连接数
	)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// 2. 创建Kafka生产者
	ps.kafkaProducer, err = NewKafkaProducer(
		ps.config.Kafka.Brokers,
		ps.config.Kafka.Topics.DeviceData,
		&ps.config.Kafka.Producer,
	)
	if err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// 3. 创建设备管理器
	ps.deviceManager = NewDeviceManager()

	// 4. 创建设备模拟器
	ps.deviceSimulator = NewDeviceSimulator(&ps.config.Device.Simulator, ps.kafkaProducer)

	// 5. 创建健康检查器
	ps.healthChecker = NewHealthChecker(ps.connectionPool, ps.kafkaProducer, ps.deviceSimulator)

	return nil
}

// Start 启动生产者服务
func (ps *ProducerService) Start() error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if ps.isRunning {
		return fmt.Errorf("producer service is already running")
	}

	// 启动连接池
	if err := ps.connectionPool.Start(); err != nil {
		return fmt.Errorf("failed to start connection pool: %w", err)
	}

	// 启动Kafka生产者
	if err := ps.kafkaProducer.Start(); err != nil {
		return fmt.Errorf("failed to start Kafka producer: %w", err)
	}

	// 启动设备模拟器（如果启用）
	if ps.config.Device.Simulator.Enabled {
		if err := ps.deviceSimulator.Start(); err != nil {
			return fmt.Errorf("failed to start device simulator: %w", err)
		}
	}

	// 启动健康检查器
	if err := ps.healthChecker.Start(); err != nil {
		return fmt.Errorf("failed to start health checker: %w", err)
	}

	ps.isRunning = true

	// 启动监控协程
	ps.wg.Add(1)
	go ps.monitoringLoop()

	fmt.Println("Producer service started successfully")
	return nil
}

// monitoringLoop 监控循环
func (ps *ProducerService) monitoringLoop() {
	defer ps.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ps.logMetrics()

		case <-ps.ctx.Done():
			return
		}
	}
}

// logMetrics 记录指标
func (ps *ProducerService) logMetrics() {
	metrics := ps.GetMetrics()
	healthStatus := ps.healthChecker.GetHealthStatus()

	fmt.Printf("[METRICS] Uptime: %v, Active Devices: %d/%d, Messages: %d, Errors: %d, Healthy: %v\n",
		metrics.Uptime.Round(time.Second),
		metrics.DevicesActive,
		metrics.DevicesTotal,
		metrics.TotalMessages,
		metrics.TotalErrors,
		healthStatus.Healthy,
	)
}

// Stop 停止生产者服务
func (ps *ProducerService) Stop() error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if !ps.isRunning {
		return nil
	}

	fmt.Println("Stopping producer service...")

	// 停止监控循环
	ps.cancel()
	ps.wg.Wait()

	// 停止健康检查器
	if err := ps.healthChecker.Stop(); err != nil {
		fmt.Printf("Error stopping health checker: %v\n", err)
	}

	// 停止设备模拟器
	if err := ps.deviceSimulator.Stop(); err != nil {
		fmt.Printf("Error stopping device simulator: %v\n", err)
	}

	// 停止Kafka生产者
	if err := ps.kafkaProducer.Stop(); err != nil {
		fmt.Printf("Error stopping Kafka producer: %v\n", err)
	}

	// 停止连接池
	if err := ps.connectionPool.Stop(); err != nil {
		fmt.Printf("Error stopping connection pool: %v\n", err)
	}

	ps.isRunning = false
	fmt.Println("Producer service stopped")
	return nil
}

// GetMetrics 获取服务指标
func (ps *ProducerService) GetMetrics() *ServiceMetrics {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	metrics := &ServiceMetrics{
		StartTime: time.Now(), // 这里应该记录实际启动时间
		Uptime:    time.Since(time.Now()), // 这里应该计算实际运行时间
	}

	// 获取生产者指标
	if ps.kafkaProducer != nil {
		producerMetrics := ps.kafkaProducer.GetMetrics()
		metrics.TotalMessages = producerMetrics.MessagesPerSecond
		metrics.TotalErrors = producerMetrics.SendErrors
	}

	// 获取设备指标
	if ps.deviceSimulator != nil {
		metrics.DevicesActive = ps.deviceSimulator.GetActiveDeviceCount()
		metrics.DevicesTotal = ps.deviceSimulator.GetDeviceCount()
	}

	// 获取连接池指标
	if ps.connectionPool != nil {
		poolMetrics := ps.connectionPool.GetMetrics()
		metrics.ConnectionsActive = poolMetrics.ActiveConnections
		metrics.ConnectionsTotal = poolMetrics.TotalConnections
	}

	return metrics
}

// GetHealthStatus 获取健康状态
func (ps *ProducerService) GetHealthStatus() *config.HealthStatus {
	if ps.healthChecker != nil {
		return ps.healthChecker.GetHealthStatus()
	}

	return &config.HealthStatus{
		Healthy:   false,
		Timestamp: time.Now(),
		Services:  make(map[string]bool),
		Errors:    []string{"health checker not initialized"},
		Metrics:   make(map[string]string),
	}
}

// IsRunning 检查服务是否运行中
func (ps *ProducerService) IsRunning() bool {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.isRunning
}

// GetKafkaProducer 获取Kafka生产者
func (ps *ProducerService) GetKafkaProducer() *KafkaProducer {
	return ps.kafkaProducer
}

// GetDeviceManager 获取设备管理器
func (ps *ProducerService) GetDeviceManager() *DeviceManager {
	return ps.deviceManager
}

// GetDeviceSimulator 获取设备模拟器
func (ps *ProducerService) GetDeviceSimulator() *DeviceSimulator {
	return ps.deviceSimulator
}

// GetConnectionPool 获取连接池
func (ps *ProducerService) GetConnectionPool() *ConnectionPool {
	return ps.connectionPool
}

// GetHealthChecker 获取健康检查器
func (ps *ProducerService) GetHealthChecker() *HealthChecker {
	return ps.healthChecker
}

// SendMessage 发送消息
func (ps *ProducerService) SendMessage(key string, value interface{}) error {
	if !ps.isRunning {
		return fmt.Errorf("producer service is not running")
	}

	return ps.kafkaProducer.SendMessage(key, value)
}

// CreateDevice 创建设备
func (ps *ProducerService) CreateDevice(deviceID, deviceType string, location interface{}, config map[string]interface{}) error {
	if !ps.isRunning {
		return fmt.Errorf("producer service is not running")
	}

	// 这里需要类型转换，简化处理
	// location应该是models.LocationInfo类型
	// 实际实现中需要正确的类型转换
	return fmt.Errorf("device creation not implemented in this version")
}

// RemoveDevice 移除设备
func (ps *ProducerService) RemoveDevice(deviceID string) error {
	if !ps.isRunning {
		return fmt.Errorf("producer service is not running")
	}

	return ps.deviceManager.RemoveDevice(deviceID)
}

// ListDevices 列出设备
func (ps *ProducerService) ListDevices() map[string]*SimulatedDevice {
	if !ps.isRunning {
		return make(map[string]*SimulatedDevice)
	}

	return ps.deviceManager.ListDevices()
}

// UpdateDeviceStatus 更新设备状态
func (ps *ProducerService) UpdateDeviceStatus(deviceID string, status string) error {
	if !ps.isRunning {
		return fmt.Errorf("producer service is not running")
	}

	// 转换状态字符串为DeviceStatus类型
	var deviceStatus DeviceStatus
	switch status {
	case "active":
		deviceStatus = DeviceStatusActive
	case "inactive":
		deviceStatus = DeviceStatusInactive
	case "error":
		deviceStatus = DeviceStatusError
	case "maintenance":
		deviceStatus = DeviceStatusMaintenance
	default:
		return fmt.Errorf("invalid device status: %s", status)
	}

	return ps.deviceManager.UpdateDeviceStatus(deviceID, deviceStatus)
}

// WaitForShutdown 等待关闭信号
func (ps *ProducerService) WaitForShutdown() {
	<-ps.ctx.Done()
}

// Restart 重启服务
func (ps *ProducerService) Restart() error {
	if err := ps.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// 等待一段时间确保资源完全释放
	time.Sleep(2 * time.Second)

	if err := ps.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}
