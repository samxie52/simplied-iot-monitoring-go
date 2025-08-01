package producer

import (
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/producer"
)

// TestKafkaProducerCreation 测试Kafka生产者创建
func TestKafkaProducerCreation(t *testing.T) {
	cfg := &config.KafkaProducer{
		ClientID:          "test-client",
		BatchSize:         100,
		BatchTimeout:      time.Millisecond * 100,
		CompressionType:   "gzip",
		MaxRetries:        3,
		RetryBackoff:      time.Millisecond * 100,
		RequiredAcks:      1,
		FlushFrequency:    time.Second,
		ChannelBufferSize: 1000,
		Timeout:           time.Second * 30,
	}

	// 注意：这个测试需要真实的Kafka集群才能完全通过
	// 在没有Kafka的情况下，创建会失败，这是预期的
	producer, err := producer.NewKafkaProducer(
		[]string{"localhost:9092"},
		"test-topic",
		cfg,
	)

	// 如果没有Kafka集群，这里会失败，这是正常的
	if err != nil {
		t.Logf("Expected error when Kafka is not available: %v", err)
		return
	}

	if producer == nil {
		t.Error("Producer should not be nil when creation succeeds")
	}

	// 清理
	if producer != nil {
		producer.Stop()
	}
}

// TestDeviceManagerBasicOperations 测试设备管理器基本操作
func TestDeviceManagerBasicOperations(t *testing.T) {
	manager := producer.NewDeviceManager()

	// 测试初始状态
	devices := manager.ListDevices()
	if len(devices) != 0 {
		t.Errorf("Expected 0 devices initially, got %d", len(devices))
	}

	// 测试设备创建（简化版本，不涉及复杂的类型转换）
	deviceCount := manager.GetDeviceCount()
	if deviceCount != 0 {
		t.Errorf("Expected device count 0, got %d", deviceCount)
	}

	// 测试设备状态更新（对不存在的设备）
	err := manager.UpdateDeviceStatus("non-existent", producer.DeviceStatusActive)
	if err == nil {
		t.Error("Expected error when updating non-existent device")
	}

	// 测试设备移除（对不存在的设备）
	err = manager.RemoveDevice("non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent device")
	}
}

// TestDeviceSimulatorCreation 测试设备模拟器创建
func TestDeviceSimulatorCreation(t *testing.T) {
	cfg := &config.DeviceSimulator{
		Enabled:         true,
		DeviceCount:     10,
		SampleInterval:  time.Second,
		DataVariation:   0.1,
		AnomalyRate:     0.05,
		TrendEnabled:    true,
		TrendStrength:   0.1,
		WorkerPoolSize:  4,
		QueueBufferSize: 1000,
	}

	// 创建一个模拟的Kafka生产者（实际上不会连接到Kafka）
	producerCfg := &config.KafkaProducer{
		ClientID:          "test-client",
		BatchSize:         100,
		BatchTimeout:      time.Millisecond * 100,
		CompressionType:   "gzip",
		MaxRetries:        3,
		RetryBackoff:      time.Millisecond * 100,
		RequiredAcks:      1,
		FlushFrequency:    time.Second,
		ChannelBufferSize: 1000,
		Timeout:           time.Second * 30,
	}

	// 尝试创建Kafka生产者（可能会失败）
	kafkaProducer, err := producer.NewKafkaProducer(
		[]string{"localhost:9092"},
		"test-topic",
		producerCfg,
	)

	// 如果Kafka不可用，跳过这个测试
	if err != nil {
		t.Logf("Skipping device simulator test due to Kafka unavailability: %v", err)
		return
	}

	simulator := producer.NewDeviceSimulator(cfg, kafkaProducer)
	if simulator == nil {
		t.Error("Device simulator should not be nil")
	}

	// 测试初始状态
	if simulator.IsRunning() {
		t.Error("Device simulator should not be running initially")
	}

	deviceCount := simulator.GetDeviceCount()
	if deviceCount != 0 {
		t.Errorf("Expected device count 0 initially, got %d", deviceCount)
	}

	activeCount := simulator.GetActiveDeviceCount()
	if activeCount != 0 {
		t.Errorf("Expected active device count 0 initially, got %d", activeCount)
	}

	// 清理
	kafkaProducer.Stop()
}

// TestConnectionPoolCreation 测试连接池创建
func TestConnectionPoolCreation(t *testing.T) {
	cfg := &config.KafkaProducer{
		ClientID:          "test-client",
		BatchSize:         100,
		BatchTimeout:      time.Millisecond * 100,
		CompressionType:   "gzip",
		MaxRetries:        3,
		RetryBackoff:      time.Millisecond * 100,
		RequiredAcks:      1,
		FlushFrequency:    time.Second,
		ChannelBufferSize: 1000,
		Timeout:           time.Second * 30,
	}

	pool, err := producer.NewConnectionPool(
		[]string{"localhost:9092"},
		cfg,
		10,
	)

	if err != nil {
		t.Errorf("Failed to create connection pool: %v", err)
		return
	}

	if pool == nil {
		t.Error("Connection pool should not be nil")
	}

	// 测试指标
	metrics := pool.GetMetrics()
	if metrics == nil {
		t.Error("Connection pool metrics should not be nil")
	}

	if metrics.TotalConnections != 0 {
		t.Errorf("Expected 0 total connections initially, got %d", metrics.TotalConnections)
	}

	// 清理
	pool.Stop()
}

// TestHealthCheckerCreation 测试健康检查器创建
func TestHealthCheckerCreation(t *testing.T) {
	// 创建模拟组件
	cfg := &config.KafkaProducer{
		ClientID:          "test-client",
		BatchSize:         100,
		BatchTimeout:      time.Millisecond * 100,
		CompressionType:   "gzip",
		MaxRetries:        3,
		RetryBackoff:      time.Millisecond * 100,
		RequiredAcks:      1,
		FlushFrequency:    time.Second,
		ChannelBufferSize: 1000,
		Timeout:           time.Second * 30,
	}

	pool, err := producer.NewConnectionPool(
		[]string{"localhost:9092"},
		cfg,
		10,
	)
	if err != nil {
		t.Logf("Skipping health checker test due to connection pool creation failure: %v", err)
		return
	}

	// 创建健康检查器（不需要真实的生产者和模拟器）
	checker := producer.NewHealthChecker(pool, nil, nil)
	if checker == nil {
		t.Error("Health checker should not be nil")
	}

	// 测试初始状态
	if checker.IsRunning() {
		t.Error("Health checker should not be running initially")
	}

	// 获取健康状态
	status := checker.GetHealthStatus()
	if status == nil {
		t.Error("Health status should not be nil")
	}

	if status.Healthy {
		t.Error("Health status should be false initially")
	}

	// 清理
	pool.Stop()
}

// TestProducerServiceCreation 测试生产者服务创建
func TestProducerServiceCreation(t *testing.T) {
	// 创建测试配置
	cfg := &config.AppConfig{
		Kafka: config.KafkaSection{
			Brokers: []string{"localhost:9092"},
			Topics: config.TopicConfig{
				DeviceData: "device-data",
				Alerts:     "alerts",
			},
			Producer: config.KafkaProducer{
				ClientID:          "test-client",
				BatchSize:         100,
				BatchTimeout:      time.Millisecond * 100,
				CompressionType:   "gzip",
				MaxRetries:        3,
				RetryBackoff:      time.Millisecond * 100,
				RequiredAcks:      1,
				FlushFrequency:    time.Second,
				ChannelBufferSize: 1000,
				Timeout:           time.Second * 30,
			},
		},
		Device: config.DeviceSection{
			Simulator: config.DeviceSimulator{
				Enabled:         false, // 禁用模拟器避免Kafka依赖
				DeviceCount:     10,
				SampleInterval:  time.Second,
				DataVariation:   0.1,
				AnomalyRate:     0.05,
				TrendEnabled:    true,
				TrendStrength:   0.1,
				WorkerPoolSize:  4,
				QueueBufferSize: 1000,
			},
		},
	}

	service, err := producer.NewProducerService(cfg)

	// 如果Kafka不可用，服务创建会失败
	if err != nil {
		t.Logf("Expected error when Kafka is not available: %v", err)
		return
	}

	if service == nil {
		t.Error("Producer service should not be nil when creation succeeds")
	}

	// 测试初始状态
	if service.IsRunning() {
		t.Error("Producer service should not be running initially")
	}

	// 测试获取组件
	if service.GetDeviceManager() == nil {
		t.Error("Device manager should not be nil")
	}

	if service.GetKafkaProducer() == nil {
		t.Error("Kafka producer should not be nil")
	}

	if service.GetDeviceSimulator() == nil {
		t.Error("Device simulator should not be nil")
	}

	if service.GetConnectionPool() == nil {
		t.Error("Connection pool should not be nil")
	}

	if service.GetHealthChecker() == nil {
		t.Error("Health checker should not be nil")
	}

	// 测试指标获取
	metrics := service.GetMetrics()
	if metrics == nil {
		t.Error("Service metrics should not be nil")
	}

	// 测试健康状态获取
	healthStatus := service.GetHealthStatus()
	if healthStatus == nil {
		t.Error("Health status should not be nil")
	}

	// 清理
	service.Stop()
}

// TestProducerServiceLifecycle 测试生产者服务生命周期
func TestProducerServiceLifecycle(t *testing.T) {
	// 创建测试配置
	cfg := &config.AppConfig{
		Kafka: config.KafkaSection{
			Brokers: []string{"localhost:9092"},
			Topics: config.TopicConfig{
				DeviceData: "device-data",
				Alerts:     "alerts",
			},
			Producer: config.KafkaProducer{
				ClientID:          "test-client",
				BatchSize:         100,
				BatchTimeout:      time.Millisecond * 100,
				CompressionType:   "gzip",
				MaxRetries:        3,
				RetryBackoff:      time.Millisecond * 100,
				RequiredAcks:      1,
				FlushFrequency:    time.Second,
				ChannelBufferSize: 1000,
				Timeout:           time.Second * 30,
			},
		},
		Device: config.DeviceSection{
			Simulator: config.DeviceSimulator{
				Enabled:         false, // 禁用模拟器避免Kafka依赖
				DeviceCount:     5,
				SampleInterval:  time.Second,
				DataVariation:   0.1,
				AnomalyRate:     0.05,
				TrendEnabled:    false,
				TrendStrength:   0.1,
				WorkerPoolSize:  2,
				QueueBufferSize: 100,
			},
		},
	}

	service, err := producer.NewProducerService(cfg)
	if err != nil {
		t.Logf("Skipping lifecycle test due to service creation failure: %v", err)
		return
	}

	// 测试启动
	err = service.Start()
	if err != nil {
		t.Logf("Expected error when starting service without Kafka: %v", err)
		return
	}

	// 如果启动成功，测试运行状态
	if !service.IsRunning() {
		t.Error("Service should be running after start")
	}

	// 等待一小段时间让服务稳定
	time.Sleep(100 * time.Millisecond)

	// 测试停止
	err = service.Stop()
	if err != nil {
		t.Errorf("Failed to stop service: %v", err)
	}

	if service.IsRunning() {
		t.Error("Service should not be running after stop")
	}
}

// TestConcurrentOperations 测试并发操作
func TestConcurrentOperations(t *testing.T) {
	manager := producer.NewDeviceManager()

	// 并发测试设备管理器
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// 并发获取设备列表
			devices := manager.ListDevices()
			if devices == nil {
				t.Errorf("Devices list should not be nil in goroutine %d", id)
				return
			}

			// 并发获取设备数量
			count := manager.GetDeviceCount()
			if count < 0 {
				t.Errorf("Device count should not be negative in goroutine %d", id)
				return
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// OK
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for concurrent operations")
			return
		}
	}
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	// 测试无效配置
	invalidCfg := &config.AppConfig{
		Kafka: config.KafkaSection{
			Brokers: []string{}, // 空的broker列表
			Topics: config.TopicConfig{
				DeviceData: "",
				Alerts:     "",
			},
		},
	}

	service, err := producer.NewProducerService(invalidCfg)
	if err == nil {
		t.Error("Expected error with invalid configuration")
		if service != nil {
			service.Stop()
		}
		return
	}

	t.Logf("Got expected error with invalid config: %v", err)
}

// BenchmarkDeviceManagerOperations 基准测试设备管理器操作
func BenchmarkDeviceManagerOperations(b *testing.B) {
	manager := producer.NewDeviceManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.ListDevices()
		manager.GetDeviceCount()
	}
}

// BenchmarkConcurrentDeviceAccess 基准测试并发设备访问
func BenchmarkConcurrentDeviceAccess(b *testing.B) {
	manager := producer.NewDeviceManager()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			manager.ListDevices()
		}
	})
}
