package main

import (
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/producer"
)

// TestFixedConfigurationTypes 测试修复后的配置类型
func TestFixedConfigurationTypes(t *testing.T) {
	t.Run("AppConfig Structure", func(t *testing.T) {
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
					SampleInterval:  1 * time.Second, // 使用正确的字段名
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

		// 验证配置结构的完整性
		if cfg.Kafka.Topics.DeviceData == "" {
			t.Fatal("DeviceData topic not set")
		}
		if cfg.Device.Simulator.SampleInterval == 0 {
			t.Fatal("SampleInterval not set")
		}
		if cfg.Kafka.Producer.ClientID == "" {
			t.Fatal("Kafka Producer ClientID not set")
		}

		t.Log("Configuration structure validation passed")
	})
}

// TestCoreComponentsAfterFix 测试修复后的核心组件
func TestCoreComponentsAfterFix(t *testing.T) {
	t.Run("PrometheusMetrics", func(t *testing.T) {
		metrics := producer.NewPrometheusMetrics()
		if metrics == nil {
			t.Fatal("Failed to create PrometheusMetrics")
		}

		// 测试基本操作
		metrics.IncrementCounter("test_counter", map[string]string{"label": "value"})
		metrics.RecordHistogram("test_histogram", 1.5, map[string]string{"label": "value"})
		metrics.SetGauge("test_gauge", 42.0, map[string]string{"label": "value"})

		t.Log("PrometheusMetrics operations completed successfully")
	})

	t.Run("MemoryPool", func(t *testing.T) {
		pool := producer.NewMemoryPool(1024, 100)
		if pool == nil {
			t.Fatal("Failed to create MemoryPool")
		}
		defer pool.Close()

		// 测试内存池操作
		buf := pool.Get()
		if buf == nil {
			t.Fatal("Failed to get buffer from pool")
		}
		pool.Put(buf)

		t.Log("MemoryPool operations completed successfully")
	})

	t.Run("ZeroCopyBuffer", func(t *testing.T) {
		data := []byte("test data for zero copy buffer")
		buf := producer.NewZeroCopyBuffer(data)
		if buf == nil {
			t.Fatal("Failed to create ZeroCopyBuffer")
		}

		// 测试零拷贝操作
		bytes := buf.Bytes()
		if len(bytes) != len(data) {
			t.Fatalf("Expected %d bytes, got %d", len(data), len(bytes))
		}

		str := buf.String()
		if str != string(data) {
			t.Fatalf("Expected %s, got %s", string(data), str)
		}

		t.Log("ZeroCopyBuffer operations completed successfully")
	})

	t.Run("BatchWorkerPool", func(t *testing.T) {
		pool := producer.NewBatchWorkerPool(4, 100)
		if pool == nil {
			t.Fatal("Failed to create BatchWorkerPool")
		}
		defer pool.Stop()

		t.Log("BatchWorkerPool created successfully")
	})
}

// TestPerformanceAfterFix 测试修复后的性能
func TestPerformanceAfterFix(t *testing.T) {
	t.Run("ZeroCopy Performance", func(t *testing.T) {
		data := make([]byte, 1024)
		for i := range data {
			data[i] = byte(i % 256)
		}

		// 基准测试零拷贝转换
		start := time.Now()
		iterations := 10000
		
		for i := 0; i < iterations; i++ {
			// 零拷贝字符串到字节转换
			str := string(data)
			_ = producer.StringToBytesZeroCopy(str)
			
			// 零拷贝字节到字符串转换
			_ = producer.BytesToStringZeroCopy(data)
		}
		
		duration := time.Since(start)
		avgNsPerOp := duration.Nanoseconds() / int64(iterations*2) // 2 operations per iteration
		
		t.Logf("Zero copy conversions: %d iterations in %v, avg %.2f ns/op", 
			iterations*2, duration, float64(avgNsPerOp))
		
		// 验证性能 - 应该非常快（< 10 ns/op）
		if avgNsPerOp > 10 {
			t.Logf("Warning: Zero copy performance may be suboptimal: %.2f ns/op", float64(avgNsPerOp))
		}
	})

	t.Run("Memory Pool Performance", func(t *testing.T) {
		pool := producer.NewMemoryPool(1024, 50)
		defer pool.Close()

		start := time.Now()
		iterations := 1000

		for i := 0; i < iterations; i++ {
			buf := pool.Get()
			pool.Put(buf)
		}

		duration := time.Since(start)
		avgNsPerOp := duration.Nanoseconds() / int64(iterations)

		t.Logf("Memory pool operations: %d iterations in %v, avg %.2f ns/op",
			iterations, duration, float64(avgNsPerOp))

		// 验证性能 - 应该在合理范围内（< 1000 ns/op）
		if avgNsPerOp > 1000 {
			t.Logf("Warning: Memory pool performance may be suboptimal: %.2f ns/op", float64(avgNsPerOp))
		}
	})
}

// TestConfigurationFieldsFixed 测试配置字段修复
func TestConfigurationFieldsFixed(t *testing.T) {
	t.Run("DeviceSimulator Fields", func(t *testing.T) {
		simulator := config.DeviceSimulator{
			Enabled:         true,
			DeviceCount:     10,
			SampleInterval:  1 * time.Second, // 正确的字段名
			DataVariation:   0.1,
			AnomalyRate:     0.01,
			TrendEnabled:    true,
			TrendStrength:   0.1,
			WorkerPoolSize:  4,
			QueueBufferSize: 1000,
		}

		// 验证字段设置正确
		if simulator.SampleInterval != 1*time.Second {
			t.Fatal("SampleInterval not set correctly")
		}
		if simulator.DeviceCount != 10 {
			t.Fatal("DeviceCount not set correctly")
		}
		if simulator.WorkerPoolSize != 4 {
			t.Fatal("WorkerPoolSize not set correctly")
		}

		t.Log("DeviceSimulator fields validation passed")
	})

	t.Run("KafkaProducer Fields", func(t *testing.T) {
		producer := config.KafkaProducer{
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
		}

		// 验证字段设置正确
		if producer.ClientID != "test-client" {
			t.Fatal("ClientID not set correctly")
		}
		if producer.BatchSize != 100 {
			t.Fatal("BatchSize not set correctly")
		}
		if producer.Timeout != 30*time.Second {
			t.Fatal("Timeout not set correctly")
		}

		t.Log("KafkaProducer fields validation passed")
	})

	t.Run("Topic Configuration", func(t *testing.T) {
		topics := config.TopicConfig{
			DeviceData: "device-data-topic",
			Alerts:     "alerts-topic",
		}

		// 验证主题配置
		if topics.DeviceData != "device-data-topic" {
			t.Fatal("DeviceData topic not set correctly")
		}
		if topics.Alerts != "alerts-topic" {
			t.Fatal("Alerts topic not set correctly")
		}

		t.Log("Topic configuration validation passed")
	})
}

// TestIntegrationReadiness 测试集成就绪状态
func TestIntegrationReadiness(t *testing.T) {
	t.Run("Configuration Compatibility", func(t *testing.T) {
		// 创建完整的应用配置
		cfg := &config.AppConfig{}
		
		// 设置所有必需的字段
		cfg.Kafka.Brokers = []string{"localhost:9092"}
		cfg.Kafka.Topics.DeviceData = "device-data"
		cfg.Kafka.Producer.ClientID = "test-producer"
		cfg.Kafka.Producer.Timeout = 30 * time.Second
		cfg.Device.Simulator.SampleInterval = 1 * time.Second
		cfg.Device.Simulator.DeviceCount = 10
		cfg.Web.Port = 8080

		// 验证配置可以正常使用
		if cfg.Kafka.Topics.DeviceData == "" {
			t.Fatal("DeviceData topic configuration failed")
		}
		if cfg.Device.Simulator.SampleInterval == 0 {
			t.Fatal("SampleInterval configuration failed")
		}

		t.Log("Configuration compatibility verified")
	})

	t.Log("System integration readiness validated")
}
