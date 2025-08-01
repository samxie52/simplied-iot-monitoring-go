package consumer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/services/consumer"
)

func TestPrometheusMetricsCollector_Creation(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.PrometheusConfig
		expectError bool
	}{
		{
			name:        "Nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "Valid config",
			config: &config.PrometheusConfig{
				Enabled:        true,
				Host:           "localhost",
				Port:           9090,
				Path:           "/metrics",
				ScrapeInterval: 15 * time.Second,
			},
			expectError: false,
		},
		{
			name: "Disabled config",
			config: &config.PrometheusConfig{
				Enabled:        false,
				Host:           "localhost",
				Port:           9091,
				Path:           "/metrics",
				ScrapeInterval: 15 * time.Second,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector, err := consumer.NewPrometheusMetricsCollector(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, collector)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, collector)

				if collector != nil {
					// 清理
					err := collector.Stop()
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestPrometheusMetricsCollector_StartStop(t *testing.T) {
	config := &config.PrometheusConfig{
		Enabled:        true,
		Host:           "localhost",
		Port:           9092,
		Path:           "/metrics",
		ScrapeInterval: 1 * time.Second,
	}

	collector, err := consumer.NewPrometheusMetricsCollector(config)
	require.NoError(t, err)
	require.NotNil(t, collector)

	// 测试启动
	err = collector.Start()
	assert.NoError(t, err)

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 测试健康检查端点
	resp, err := http.Get("http://localhost:9092/health")
	assert.NoError(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	}

	// 测试指标端点
	resp, err = http.Get("http://localhost:9092/metrics")
	assert.NoError(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Contains(t, string(body), "kafka_consumer_messages_consumed_total")
		resp.Body.Close()
	}

	// 测试停止
	err = collector.Stop()
	assert.NoError(t, err)

	// 验证服务器已停止
	time.Sleep(100 * time.Millisecond)
	_, err = http.Get("http://localhost:9092/health")
	assert.Error(t, err)
}

func TestPrometheusMetricsCollector_DisabledMode(t *testing.T) {
	config := &config.PrometheusConfig{
		Enabled:        false,
		Host:           "localhost",
		Port:           9093,
		Path:           "/metrics",
		ScrapeInterval: 15 * time.Second,
	}

	collector, err := consumer.NewPrometheusMetricsCollector(config)
	require.NoError(t, err)
	require.NotNil(t, collector)

	// 启动应该成功但不启动服务器
	err = collector.Start()
	assert.NoError(t, err)

	// 验证服务器未启动
	_, err = http.Get("http://localhost:9093/health")
	assert.Error(t, err)

	// 停止应该成功
	err = collector.Stop()
	assert.NoError(t, err)
}

func TestPrometheusMetricsCollector_CollectConsumerMetrics(t *testing.T) {
	config := &config.PrometheusConfig{
		Enabled:        true,
		Host:           "localhost",
		Port:           9094,
		Path:           "/metrics",
		ScrapeInterval: 1 * time.Second,
	}

	collector, err := consumer.NewPrometheusMetricsCollector(config)
	require.NoError(t, err)
	require.NotNil(t, collector)

	err = collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 收集测试指标
	testMetrics := &consumer.ConsumerMetrics{
		MessagesConsumed:     100,
		MessagesProcessed:    95,
		ProcessingErrors:     5,
		RebalanceCount:       2,
		PartitionCount:       3,
		LagTotal:             50,
		ProcessingLatencyMs:  25,
		ThroughputPerSecond:  1000,
		LastMessageTimestamp: time.Now().UnixMilli(),
	}

	collector.CollectConsumerMetrics(testMetrics)

	// 验证指标是否正确暴露
	resp, err := http.Get("http://localhost:9094/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	metricsText := string(body)

	// 验证关键指标存在
	assert.Contains(t, metricsText, "kafka_consumer_messages_consumed_total")
	assert.Contains(t, metricsText, "kafka_consumer_messages_processed_total")
	assert.Contains(t, metricsText, "kafka_consumer_processing_errors_total")
	assert.Contains(t, metricsText, "kafka_consumer_lag_total")
	assert.Contains(t, metricsText, "kafka_consumer_active_partitions")
}

func TestPrometheusMetricsCollector_CollectProcessorMetrics(t *testing.T) {
	config := &config.PrometheusConfig{
		Enabled:        true,
		Host:           "localhost",
		Port:           9095,
		Path:           "/metrics",
		ScrapeInterval: 1 * time.Second,
	}

	collector, err := consumer.NewPrometheusMetricsCollector(config)
	require.NoError(t, err)
	require.NotNil(t, collector)

	err = collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 收集处理器指标
	processorStats := consumer.ProcessorStats{
		ProcessedCount: 200,
		ErrorCount:     10,
		AvgLatencyMs:   15.5,
	}

	collector.CollectProcessorMetrics(processorStats)

	// 验证指标
	resp, err := http.Get("http://localhost:9095/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	metricsText := string(body)

	assert.Contains(t, metricsText, "kafka_processor_processed_total")
	assert.Contains(t, metricsText, "kafka_processor_errors_total")
	assert.Contains(t, metricsText, "kafka_processor_avg_latency_milliseconds")
}

func TestPrometheusMetricsCollector_CollectErrorMetrics(t *testing.T) {
	config := &config.PrometheusConfig{
		Enabled:        true,
		Host:           "localhost",
		Port:           9096,
		Path:           "/metrics",
		ScrapeInterval: 1 * time.Second,
	}

	collector, err := consumer.NewPrometheusMetricsCollector(config)
	require.NoError(t, err)
	require.NotNil(t, collector)

	err = collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 收集错误指标
	errorStats := consumer.ErrorStats{
		TotalErrors:     50,
		RetryableErrors: 30,
		FatalErrors:     20,
	}

	collector.CollectErrorMetrics(errorStats)

	// 验证指标
	resp, err := http.Get("http://localhost:9096/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	metricsText := string(body)

	assert.Contains(t, metricsText, "kafka_error_handler_total_errors")
	assert.Contains(t, metricsText, "kafka_error_handler_retryable_errors")
	assert.Contains(t, metricsText, "kafka_error_handler_fatal_errors")
}

func TestPrometheusMetricsCollector_ConcurrentAccess(t *testing.T) {
	config := &config.PrometheusConfig{
		Enabled:        true,
		Host:           "localhost",
		Port:           9097,
		Path:           "/metrics",
		ScrapeInterval: 1 * time.Second,
	}

	collector, err := consumer.NewPrometheusMetricsCollector(config)
	require.NoError(t, err)
	require.NotNil(t, collector)

	err = collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 并发收集指标
	var wg sync.WaitGroup
	numGoroutines := 10
	numIterations := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				testMetrics := &consumer.ConsumerMetrics{
					MessagesConsumed:     int64(id*numIterations + j),
					MessagesProcessed:    int64(id*numIterations + j - 1),
					ProcessingErrors:     1,
					RebalanceCount:       0,
					PartitionCount:       3,
					LagTotal:             int64(j),
					ProcessingLatencyMs:  int64(10 + j%20),
					ThroughputPerSecond:  1000,
					LastMessageTimestamp: time.Now().UnixMilli(),
				}
				collector.CollectConsumerMetrics(testMetrics)

				processorStats := consumer.ProcessorStats{
					ProcessedCount: int64(j),
					ErrorCount:     int64(j % 10),
					AvgLatencyMs:   float64(15 + j%10),
				}
				collector.CollectProcessorMetrics(processorStats)
			}
		}(i)
	}

	wg.Wait()

	// 验证指标端点仍然可访问
	resp, err := http.Get("http://localhost:9097/metrics")
	assert.NoError(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	}
}

func TestPrometheusMetricsCollector_GetCollectedMetrics(t *testing.T) {
	config := &config.PrometheusConfig{
		Enabled:        true,
		Host:           "localhost",
		Port:           9098,
		Path:           "/metrics",
		ScrapeInterval: 15 * time.Second,
	}

	collector, err := consumer.NewPrometheusMetricsCollector(config)
	require.NoError(t, err)
	require.NotNil(t, collector)

	metrics := collector.GetCollectedMetrics()
	assert.NotNil(t, metrics)

	// 验证返回的基本信息
	assert.Equal(t, true, metrics["prometheus_enabled"])
	assert.Equal(t, "localhost", metrics["prometheus_host"])
	assert.Equal(t, 9098, metrics["prometheus_port"])
	assert.Equal(t, "/metrics", metrics["prometheus_path"])
	assert.Contains(t, metrics["metrics_endpoint"], "http://localhost:9098/metrics")

	// 清理
	err = collector.Stop()
	assert.NoError(t, err)
}

func TestPrometheusMetricsCollector_GetMetricsEndpoint(t *testing.T) {
	// 测试启用状态
	enabledConfig := &config.PrometheusConfig{
		Enabled:        true,
		Host:           "localhost",
		Port:           9099,
		Path:           "/metrics",
		ScrapeInterval: 15 * time.Second,
	}

	collector, err := consumer.NewPrometheusMetricsCollector(enabledConfig)
	require.NoError(t, err)
	require.NotNil(t, collector)

	endpoint := collector.GetMetricsEndpoint()
	assert.Equal(t, "http://localhost:9099/metrics", endpoint)

	err = collector.Stop()
	assert.NoError(t, err)

	// 测试禁用状态
	disabledConfig := &config.PrometheusConfig{
		Enabled:        false,
		Host:           "localhost",
		Port:           9100,
		Path:           "/metrics",
		ScrapeInterval: 15 * time.Second,
	}

	collector2, err := consumer.NewPrometheusMetricsCollector(disabledConfig)
	require.NoError(t, err)
	require.NotNil(t, collector2)

	endpoint2 := collector2.GetMetricsEndpoint()
	assert.Empty(t, endpoint2)

	err = collector2.Stop()
	assert.NoError(t, err)
}

func TestPrometheusMetricsCollector_SystemMetrics(t *testing.T) {
	config := &config.PrometheusConfig{
		Enabled:        true,
		Host:           "localhost",
		Port:           9101,
		Path:           "/metrics",
		ScrapeInterval: 500 * time.Millisecond, // 更短的间隔用于测试
	}

	collector, err := consumer.NewPrometheusMetricsCollector(config)
	require.NoError(t, err)
	require.NotNil(t, collector)

	err = collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	// 等待系统指标收集
	time.Sleep(1 * time.Second)

	// 验证系统指标
	resp, err := http.Get("http://localhost:9101/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	metricsText := string(body)

	assert.Contains(t, metricsText, "kafka_consumer_memory_usage_bytes")
	assert.Contains(t, metricsText, "kafka_consumer_cpu_usage_percent")
	assert.Contains(t, metricsText, "kafka_consumer_goroutines")
}

// BenchmarkPrometheusMetricsCollector_CollectMetrics 性能基准测试
func BenchmarkPrometheusMetricsCollector_CollectMetrics(b *testing.B) {
	config := &config.PrometheusConfig{
		Enabled:        true,
		Host:           "localhost",
		Port:           9102,
		Path:           "/metrics",
		ScrapeInterval: 15 * time.Second,
	}

	collector, err := consumer.NewPrometheusMetricsCollector(config)
	require.NoError(b, err)
	require.NotNil(b, collector)

	err = collector.Start()
	require.NoError(b, err)
	defer collector.Stop()

	testMetrics := &consumer.ConsumerMetrics{
		MessagesConsumed:     1000,
		MessagesProcessed:    995,
		ProcessingErrors:     5,
		RebalanceCount:       1,
		PartitionCount:       5,
		LagTotal:             100,
		ProcessingLatencyMs:  20,
		ThroughputPerSecond:  2000,
		LastMessageTimestamp: time.Now().UnixMilli(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector.CollectConsumerMetrics(testMetrics)
		}
	})
}

// TestPrometheusMetricsCollector_Integration 集成测试
func TestPrometheusMetricsCollector_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &config.PrometheusConfig{
		Enabled:        true,
		Host:           "localhost",
		Port:           9103,
		Path:           "/metrics",
		ScrapeInterval: 1 * time.Second,
	}

	collector, err := consumer.NewPrometheusMetricsCollector(config)
	require.NoError(t, err)
	require.NotNil(t, collector)

	err = collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	// 模拟真实的指标收集场景
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		counter := int64(0)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				counter++
				testMetrics := &consumer.ConsumerMetrics{
					MessagesConsumed:     counter * 10,
					MessagesProcessed:    counter*10 - counter%5,
					ProcessingErrors:     counter % 5,
					RebalanceCount:       counter / 100,
					PartitionCount:       3,
					LagTotal:             counter % 50,
					ProcessingLatencyMs:  15 + counter%20,
					ThroughputPerSecond:  1000 + counter%500,
					LastMessageTimestamp: time.Now().UnixMilli(),
				}
				collector.CollectConsumerMetrics(testMetrics)

				processorStats := consumer.ProcessorStats{
					ProcessedCount: counter * 8,
					ErrorCount:     counter % 3,
					AvgLatencyMs:   float64(12 + counter%15),
				}
				collector.CollectProcessorMetrics(processorStats)

				errorStats := consumer.ErrorStats{
					TotalErrors:     counter % 10,
					RetryableErrors: counter % 7,
					FatalErrors:     counter % 3,
				}
				collector.CollectErrorMetrics(errorStats)
			}
		}
	}()

	// 定期检查指标端点
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resp, err := http.Get(fmt.Sprintf("http://localhost:9103/metrics"))
			assert.NoError(t, err)
			if resp != nil {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)
				assert.True(t, len(body) > 0)
				assert.True(t, strings.Contains(string(body), "kafka_consumer_"))
				resp.Body.Close()
			}
		}
	}
}
