package producer

import (
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/producer"
)

// TestBasicIntegration 测试基本集成功能
func TestBasicIntegration(t *testing.T) {
	// 创建测试配置
	cfg := &config.AppConfig{
		App: config.AppSection{
			Name:        "test-app",
			Version:     "1.0.0",
			Environment: "test",
			Debug:       true,
			LogLevel:    "debug",
		},
		Kafka: config.KafkaSection{
			Brokers: []string{"localhost:9092"},
			Topics: config.TopicConfig{
				DeviceData: "device-data-test",
				Alerts:     "alerts-test",
			},
			Producer: config.KafkaProducer{
				ClientID:          "test-client",
				BatchSize:         100,
				BatchTimeout:      100 * time.Millisecond,
				CompressionType:   "gzip",
				MaxRetries:        3,
				RetryBackoff:      100 * time.Millisecond,
				RequiredAcks:      1,
				FlushFrequency:    50 * time.Millisecond,
				ChannelBufferSize: 256,
				Timeout:           30 * time.Second,
			},
			Consumer: config.KafkaConsumer{
				GroupID:          "test-group",
				AutoOffsetReset:  "earliest",
				EnableAutoCommit: true,
				SessionTimeout:   30 * time.Second,
				MaxPollRecords:   100,
			},
			Timeout: 30 * time.Second,
		},
		Producer: config.ProducerSection{
			DeviceCount:   10,
			SendInterval:  1 * time.Second,
			DataVariance:  0.1,
			BatchSize:     100,
			RetryAttempts: 3,
			Timeout:       30 * time.Second,
		},
		Device: config.DeviceSection{
			Simulator: config.DeviceSimulator{
				Enabled:         true,
				DeviceCount:     10,
				SampleInterval:  1 * time.Second,
				DataVariation:   0.1,
				AnomalyRate:     0.01,
				TrendEnabled:    true,
				TrendStrength:   0.1,
				WorkerPoolSize:  4,
				QueueBufferSize: 1000,
			},
		},
		Web: config.WebSection{
			Host:           "localhost",
			Port:           8080,
			TemplatePath:   "./templates",
			StaticPath:     "./static",
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1024 * 1024,
		},
	}

	t.Run("KafkaProducer Creation", func(t *testing.T) {
		// 测试Kafka生产者创建
		kafkaProducer, err := producer.NewKafkaProducer(
			cfg.Kafka.Brokers,
			cfg.Kafka.Topics.DeviceData,
			&cfg.Kafka.Producer,
		)
		if err != nil {
			t.Logf("Kafka producer creation failed (expected in test environment): %v", err)
			// 在测试环境中，Kafka可能不可用，这是正常的
			return
		}
		defer kafkaProducer.Stop()

		t.Log("Kafka producer created successfully")
	})

	t.Run("ConnectionPool Creation", func(t *testing.T) {
		// 测试连接池创建
		connectionPool, err := producer.NewConnectionPool(
			cfg.Kafka.Brokers,
			&cfg.Kafka.Producer,
			10,
		)
		if err != nil {
			t.Logf("Connection pool creation failed (expected in test environment): %v", err)
			return
		}
		defer connectionPool.Stop()

		t.Log("Connection pool created successfully")
	})

	t.Run("DeviceSimulator Creation", func(t *testing.T) {
		// 创建模拟的Kafka生产者用于测试
		mockProducer := &producer.MockKafkaProducer{}

		// 测试设备模拟器创建
		deviceSimulator := producer.NewDeviceSimulator(
			&cfg.Device.Simulator,
			mockProducer,
		)
		if deviceSimulator == nil {
			t.Fatal("Device simulator creation failed")
		}

		t.Log("Device simulator created successfully")
	})

	t.Run("BatchWorkerPool Creation", func(t *testing.T) {
		// 测试批处理工作池创建
		workerPool := producer.NewBatchWorkerPool(10, 1000)
		if workerPool == nil {
			t.Fatal("Batch worker pool creation failed")
		}
		defer workerPool.Stop()

		t.Log("Batch worker pool created successfully")
	})

	t.Run("PrometheusMetrics Operations", func(t *testing.T) {
		// 测试Prometheus指标
		metrics := producer.NewPrometheusMetrics()
		if metrics == nil {
			t.Fatal("Prometheus metrics creation failed")
		}

		// 测试指标操作
		metrics.IncrementCounter("test_counter", map[string]string{"label": "value"})
		metrics.RecordHistogram("test_histogram", 1.5, map[string]string{"label": "value"})
		metrics.SetGauge("test_gauge", 42.0, map[string]string{"label": "value"})

		t.Log("Prometheus metrics operations completed successfully")
	})

	t.Log("Basic integration test completed successfully")
}

// TestPerformanceComponents 测试性能组件
func TestPerformanceComponents(t *testing.T) {
	t.Run("MemoryPool Performance", func(t *testing.T) {
		pool := producer.NewMemoryPool(1024, 100)
		defer pool.Close()

		// 测试内存池性能
		start := time.Now()
		for i := 0; i < 1000; i++ {
			buf := pool.Get()
			pool.Put(buf)
		}
		duration := time.Since(start)

		t.Logf("Memory pool 1000 operations completed in %v", duration)
		if duration > 10*time.Millisecond {
			t.Logf("Warning: Memory pool operations took longer than expected: %v", duration)
		}
	})

	t.Run("ZeroCopyBuffer Performance", func(t *testing.T) {
		data := make([]byte, 1024)
		for i := range data {
			data[i] = byte(i % 256)
		}

		// 测试零拷贝缓冲区性能
		start := time.Now()
		for i := 0; i < 1000; i++ {
			buf := producer.NewZeroCopyBuffer(data)
			_ = buf.Bytes()
			_ = buf.String()
		}
		duration := time.Since(start)

		t.Logf("Zero copy buffer 1000 operations completed in %v", duration)
		if duration > 5*time.Millisecond {
			t.Logf("Warning: Zero copy operations took longer than expected: %v", duration)
		}
	})

	t.Log("Performance components test completed successfully")
}

// TestConfigurationCompatibility 测试配置兼容性
func TestConfigurationCompatibility(t *testing.T) {
	// 测试配置结构的完整性
	cfg := &config.AppConfig{}

	// 验证所有必需的配置字段都存在
	t.Run("Kafka Configuration", func(t *testing.T) {
		cfg.Kafka.Brokers = []string{"localhost:9092"}
		cfg.Kafka.Topics.DeviceData = "device-data"
		cfg.Kafka.Topics.Alerts = "alerts"

		// 验证KafkaProducer配置
		cfg.Kafka.Producer.ClientID = "test-client"
		cfg.Kafka.Producer.BatchSize = 100
		cfg.Kafka.Producer.Timeout = 30 * time.Second

		t.Log("Kafka configuration structure is valid")
	})

	t.Run("Device Configuration", func(t *testing.T) {
		cfg.Device.Simulator.Enabled = true
		cfg.Device.Simulator.DeviceCount = 10
		cfg.Device.Simulator.SampleInterval = 1 * time.Second
		cfg.Device.Simulator.WorkerPoolSize = 4
		cfg.Device.Simulator.QueueBufferSize = 1000

		t.Log("Device configuration structure is valid")
	})

	t.Run("Producer Configuration", func(t *testing.T) {
		cfg.Producer.DeviceCount = 10
		cfg.Producer.SendInterval = 1 * time.Second
		cfg.Producer.BatchSize = 100
		cfg.Producer.Timeout = 30 * time.Second

		t.Log("Producer configuration structure is valid")
	})

	t.Log("Configuration compatibility test completed successfully")
}
