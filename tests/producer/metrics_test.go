package producer

import (
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/producer"
)

func TestPrometheusMetrics_Creation(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	if metrics == nil {
		t.Fatal("Failed to create Prometheus metrics")
	}
}

func TestPrometheusMetrics_IncrementCounters(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	// 测试消息计数
	metrics.IncrementMessages()
	metrics.IncrementMessages()
	metrics.IncrementMessages()

	// 测试字节计数
	metrics.IncrementBytes(1024)
	metrics.IncrementBytes(2048)

	// 测试批次计数
	metrics.IncrementBatches()
	metrics.IncrementBatches()

	// 测试错误计数
	metrics.IncrementSendErrors()
	metrics.IncrementRetryAttempts()
	metrics.IncrementDroppedMessages()
	metrics.IncrementTimeoutErrors()

	// 测试性能优化计数
	metrics.IncrementZeroCopyOperations()
	metrics.IncrementMemoryPoolHits()
	metrics.IncrementMemoryPoolMisses()
	metrics.IncrementConnectionReuse()

	t.Log("All counter increments completed successfully")
}

func TestPrometheusMetrics_SetGauges(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	// 测试设备数量
	metrics.SetDeviceCount(100)
	metrics.SetDeviceCount(150)

	// 测试活跃连接数
	metrics.SetActiveConnections(10)
	metrics.SetActiveConnections(15)

	// 测试队列深度
	metrics.SetQueueDepth(50)
	metrics.SetQueueDepth(75)

	// 测试健康状态
	metrics.SetHealthStatus(true)
	metrics.SetHealthStatus(false)
	metrics.SetHealthStatus(true)

	// 测试组件状态
	metrics.SetComponentStatus("kafka_producer", true)
	metrics.SetComponentStatus("connection_pool", false)
	metrics.SetComponentStatus("worker_pool", true)

	t.Log("All gauge settings completed successfully")
}

func TestPrometheusMetrics_RecordHistograms(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	// 测试延迟记录
	metrics.RecordSendLatency(10 * time.Millisecond)
	metrics.RecordSendLatency(25 * time.Millisecond)
	metrics.RecordSendLatency(50 * time.Millisecond)

	metrics.RecordSerializationLatency(1 * time.Millisecond)
	metrics.RecordSerializationLatency(2 * time.Millisecond)

	metrics.RecordCompressionLatency(5 * time.Millisecond)
	metrics.RecordCompressionLatency(8 * time.Millisecond)

	metrics.RecordBatchLatency(100 * time.Millisecond)
	metrics.RecordBatchLatency(150 * time.Millisecond)

	// 测试大小记录
	metrics.RecordBatchSize(10)
	metrics.RecordBatchSize(25)
	metrics.RecordBatchSize(50)

	metrics.RecordMessageSize(512)
	metrics.RecordMessageSize(1024)
	metrics.RecordMessageSize(2048)

	t.Log("All histogram recordings completed successfully")
}

func TestPrometheusMetrics_UpdateResourceMetrics(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	// 更新资源指标
	metrics.UpdateResourceMetrics()

	// 多次更新以测试稳定性
	for i := 0; i < 5; i++ {
		metrics.UpdateResourceMetrics()
		time.Sleep(10 * time.Millisecond)
	}

	t.Log("Resource metrics updated successfully")
}

func TestPrometheusMetrics_UpdateLastSuccessTime(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	// 更新最后成功时间
	metrics.UpdateLastSuccessTime()

	// 等待一段时间后再次更新
	time.Sleep(100 * time.Millisecond)
	metrics.UpdateLastSuccessTime()

	t.Log("Last success time updated successfully")
}

func TestPrometheusMetrics_GetUptime(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	// 等待一段时间
	time.Sleep(100 * time.Millisecond)

	uptime := metrics.GetUptime()
	if uptime <= 0 {
		t.Errorf("Expected positive uptime, got %v", uptime)
	}

	if uptime < 100*time.Millisecond {
		t.Errorf("Expected uptime >= 100ms, got %v", uptime)
	}

	t.Logf("Uptime: %v", uptime)
}

func TestPrometheusMetrics_ConcurrentAccess(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	// 并发测试
	done := make(chan bool, 10)

	// 启动多个goroutine并发更新指标
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < 100; j++ {
				metrics.IncrementMessages()
				metrics.IncrementBytes(int64(j * 10))
				metrics.SetDeviceCount(id * 10)
				metrics.RecordSendLatency(time.Duration(j) * time.Millisecond)
				metrics.UpdateResourceMetrics()
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	t.Log("Concurrent access test completed successfully")
}

func TestPrometheusMetrics_ComponentStatusTracking(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	components := []string{
		"kafka_producer",
		"connection_pool",
		"worker_pool",
		"device_simulator",
		"batch_processor",
	}

	// 设置所有组件为健康状态
	for _, component := range components {
		metrics.SetComponentStatus(component, true)
	}

	// 模拟一些组件出现问题
	metrics.SetComponentStatus("kafka_producer", false)
	metrics.SetComponentStatus("connection_pool", false)

	// 恢复健康状态
	metrics.SetComponentStatus("kafka_producer", true)
	metrics.SetComponentStatus("connection_pool", true)

	t.Log("Component status tracking test completed successfully")
}

func TestPrometheusMetrics_PerformanceOptimizationMetrics(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	// 模拟性能优化操作
	for i := 0; i < 1000; i++ {
		if i%3 == 0 {
			metrics.IncrementZeroCopyOperations()
		}
		if i%2 == 0 {
			metrics.IncrementMemoryPoolHits()
		} else {
			metrics.IncrementMemoryPoolMisses()
		}
		if i%5 == 0 {
			metrics.IncrementConnectionReuse()
		}
	}

	t.Log("Performance optimization metrics test completed successfully")
}

func BenchmarkPrometheusMetrics_IncrementMessages(b *testing.B) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.IncrementMessages()
	}
}

func BenchmarkPrometheusMetrics_RecordSendLatency(b *testing.B) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")
	duration := 10 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordSendLatency(duration)
	}
}

func BenchmarkPrometheusMetrics_UpdateResourceMetrics(b *testing.B) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.UpdateResourceMetrics()
	}
}

func BenchmarkPrometheusMetrics_ConcurrentIncrements(b *testing.B) {
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			metrics.IncrementMessages()
			metrics.IncrementBytes(1024)
			metrics.SetDeviceCount(100)
		}
	})
}
