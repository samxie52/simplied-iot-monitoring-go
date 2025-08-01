package consumer

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"simplied-iot-monitoring-go/internal/config"
)

// PrometheusMetricsCollector Prometheus指标收集器
type PrometheusMetricsCollector struct {
	// 消费者指标
	messagesConsumed    prometheus.Counter
	messagesProcessed   prometheus.Counter
	processingErrors    prometheus.Counter
	processingLatency   prometheus.Histogram
	consumerLag         prometheus.Gauge
	activePartitions    prometheus.Gauge
	rebalanceCount      prometheus.Counter
	throughputPerSecond prometheus.Gauge

	// 处理器指标
	processorProcessedCount prometheus.Counter
	processorErrorCount     prometheus.Counter
	processorAvgLatency     prometheus.Gauge

	// 错误处理指标
	totalErrors     prometheus.Counter
	retryableErrors prometheus.Counter
	fatalErrors     prometheus.Counter

	// 系统指标
	memoryUsage prometheus.Gauge
	cpuUsage    prometheus.Gauge
	goroutines  prometheus.Gauge

	// HTTP服务器
	httpServer *http.Server
	registry   *prometheus.Registry
	config     *config.PrometheusConfig

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mutex  sync.RWMutex
}

// NewPrometheusMetricsCollector 创建Prometheus指标收集器
func NewPrometheusMetricsCollector(cfg *config.PrometheusConfig) (*PrometheusMetricsCollector, error) {
	if cfg == nil {
		return nil, fmt.Errorf("prometheus config cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建自定义注册表
	registry := prometheus.NewRegistry()

	collector := &PrometheusMetricsCollector{
		config:   cfg,
		registry: registry,
		ctx:      ctx,
		cancel:   cancel,
	}

	// 初始化指标
	if err := collector.initMetrics(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	// 注册指标到注册表
	if err := collector.registerMetrics(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to register metrics: %w", err)
	}

	return collector, nil
}

// initMetrics 初始化所有指标
func (pmc *PrometheusMetricsCollector) initMetrics() error {
	// 消费者指标
	pmc.messagesConsumed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kafka_consumer_messages_consumed_total",
		Help: "Total number of messages consumed from Kafka",
	})

	pmc.messagesProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kafka_consumer_messages_processed_total",
		Help: "Total number of messages successfully processed",
	})

	pmc.processingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kafka_consumer_processing_errors_total",
		Help: "Total number of message processing errors",
	})

	pmc.processingLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "kafka_consumer_processing_latency_seconds",
		Help:    "Message processing latency in seconds",
		Buckets: prometheus.DefBuckets,
	})

	pmc.consumerLag = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kafka_consumer_lag_total",
		Help: "Total consumer lag across all partitions",
	})

	pmc.activePartitions = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kafka_consumer_active_partitions",
		Help: "Number of active partitions being consumed",
	})

	pmc.rebalanceCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kafka_consumer_rebalance_total",
		Help: "Total number of consumer group rebalances",
	})

	pmc.throughputPerSecond = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kafka_consumer_throughput_per_second",
		Help: "Current message processing throughput per second",
	})

	// 处理器指标
	pmc.processorProcessedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kafka_processor_processed_total",
		Help: "Total number of messages processed by message processor",
	})

	pmc.processorErrorCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kafka_processor_errors_total",
		Help: "Total number of message processor errors",
	})

	pmc.processorAvgLatency = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kafka_processor_avg_latency_milliseconds",
		Help: "Average message processing latency in milliseconds",
	})

	// 错误处理指标
	pmc.totalErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kafka_error_handler_total_errors",
		Help: "Total number of errors handled",
	})

	pmc.retryableErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kafka_error_handler_retryable_errors",
		Help: "Total number of retryable errors",
	})

	pmc.fatalErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kafka_error_handler_fatal_errors",
		Help: "Total number of fatal errors",
	})

	// 系统指标
	pmc.memoryUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kafka_consumer_memory_usage_bytes",
		Help: "Current memory usage in bytes",
	})

	pmc.cpuUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kafka_consumer_cpu_usage_percent",
		Help: "Current CPU usage percentage",
	})

	pmc.goroutines = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kafka_consumer_goroutines",
		Help: "Current number of goroutines",
	})

	return nil
}

// registerMetrics 注册所有指标
func (pmc *PrometheusMetricsCollector) registerMetrics() error {
	metrics := []prometheus.Collector{
		// 消费者指标
		pmc.messagesConsumed,
		pmc.messagesProcessed,
		pmc.processingErrors,
		pmc.processingLatency,
		pmc.consumerLag,
		pmc.activePartitions,
		pmc.rebalanceCount,
		pmc.throughputPerSecond,

		// 处理器指标
		pmc.processorProcessedCount,
		pmc.processorErrorCount,
		pmc.processorAvgLatency,

		// 错误处理指标
		pmc.totalErrors,
		pmc.retryableErrors,
		pmc.fatalErrors,

		// 系统指标
		pmc.memoryUsage,
		pmc.cpuUsage,
		pmc.goroutines,
	}

	for _, metric := range metrics {
		if err := pmc.registry.Register(metric); err != nil {
			return fmt.Errorf("failed to register metric: %w", err)
		}
	}

	return nil
}

// Start 启动Prometheus指标服务器
func (pmc *PrometheusMetricsCollector) Start() error {
	if !pmc.config.Enabled {
		return nil
	}

	pmc.mutex.Lock()
	defer pmc.mutex.Unlock()

	// 创建HTTP处理器
	mux := http.NewServeMux()
	mux.Handle(pmc.config.Path, promhttp.HandlerFor(pmc.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))

	// 添加健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 创建HTTP服务器
	addr := fmt.Sprintf("%s:%d", pmc.config.Host, pmc.config.Port)
	pmc.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动系统指标收集
	pmc.wg.Add(1)
	go pmc.collectSystemMetrics()

	// 启动HTTP服务器
	pmc.wg.Add(1)
	go func() {
		defer pmc.wg.Done()
		if err := pmc.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Prometheus metrics server error: %v\n", err)
		}
	}()

	fmt.Printf("Prometheus metrics server started on %s%s\n", addr, pmc.config.Path)
	return nil
}

// Stop 停止Prometheus指标服务器
func (pmc *PrometheusMetricsCollector) Stop() error {
	pmc.mutex.Lock()
	defer pmc.mutex.Unlock()

	// 取消上下文
	if pmc.cancel != nil {
		pmc.cancel()
	}

	// 停止HTTP服务器
	if pmc.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := pmc.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown prometheus server: %w", err)
		}
	}

	// 等待所有goroutine结束
	pmc.wg.Wait()

	fmt.Println("Prometheus metrics server stopped")
	return nil
}

// CollectConsumerMetrics 收集消费者指标
func (pmc *PrometheusMetricsCollector) CollectConsumerMetrics(metrics *ConsumerMetrics) {
	if !pmc.config.Enabled {
		return
	}

	pmc.mutex.RLock()
	defer pmc.mutex.RUnlock()

	// 更新计数器指标（只能增加）
	pmc.messagesConsumed.Add(float64(metrics.MessagesConsumed))
	pmc.messagesProcessed.Add(float64(metrics.MessagesProcessed))
	pmc.processingErrors.Add(float64(metrics.ProcessingErrors))
	pmc.rebalanceCount.Add(float64(metrics.RebalanceCount))

	// 更新仪表指标（可以设置任意值）
	pmc.consumerLag.Set(float64(metrics.LagTotal))
	pmc.activePartitions.Set(float64(metrics.PartitionCount))
	pmc.throughputPerSecond.Set(float64(metrics.ThroughputPerSecond))

	// 记录处理延迟
	if metrics.ProcessingLatencyMs > 0 {
		pmc.processingLatency.Observe(float64(metrics.ProcessingLatencyMs) / 1000.0)
	}
}

// CollectProcessorMetrics 收集处理器指标
func (pmc *PrometheusMetricsCollector) CollectProcessorMetrics(stats ProcessorStats) {
	if !pmc.config.Enabled {
		return
	}

	pmc.mutex.RLock()
	defer pmc.mutex.RUnlock()

	pmc.processorProcessedCount.Add(float64(stats.ProcessedCount))
	pmc.processorErrorCount.Add(float64(stats.ErrorCount))
	pmc.processorAvgLatency.Set(stats.AvgLatencyMs)
}

// CollectErrorMetrics 收集错误指标
func (pmc *PrometheusMetricsCollector) CollectErrorMetrics(stats ErrorStats) {
	if !pmc.config.Enabled {
		return
	}

	pmc.mutex.RLock()
	defer pmc.mutex.RUnlock()

	pmc.totalErrors.Add(float64(stats.TotalErrors))
	pmc.retryableErrors.Add(float64(stats.RetryableErrors))
	pmc.fatalErrors.Add(float64(stats.FatalErrors))
}

// GetCollectedMetrics 获取收集的指标（兼容接口）
func (pmc *PrometheusMetricsCollector) GetCollectedMetrics() map[string]interface{} {
	// Prometheus指标通过HTTP端点暴露，这里返回基本信息
	return map[string]interface{}{
		"prometheus_enabled": pmc.config.Enabled,
		"prometheus_host":    pmc.config.Host,
		"prometheus_port":    pmc.config.Port,
		"prometheus_path":    pmc.config.Path,
		"metrics_endpoint":   fmt.Sprintf("http://%s:%d%s", pmc.config.Host, pmc.config.Port, pmc.config.Path),
	}
}

// collectSystemMetrics 收集系统指标
func (pmc *PrometheusMetricsCollector) collectSystemMetrics() {
	defer pmc.wg.Done()

	ticker := time.NewTicker(pmc.config.ScrapeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pmc.ctx.Done():
			return
		case <-ticker.C:
			pmc.updateSystemMetrics()
		}
	}
}

// updateSystemMetrics 更新系统指标
func (pmc *PrometheusMetricsCollector) updateSystemMetrics() {
	pmc.mutex.RLock()
	defer pmc.mutex.RUnlock()

	// 获取内存统计
	memStats := config.ReadMemStats()
	pmc.memoryUsage.Set(float64(memStats.HeapInuse))

	// 获取goroutine数量
	pmc.goroutines.Set(float64(memStats.NumGC))

	// CPU使用率需要更复杂的计算，这里简化处理
	pmc.cpuUsage.Set(memStats.GCCPUFraction * 100)
}

// GetMetricsEndpoint 获取指标端点URL
func (pmc *PrometheusMetricsCollector) GetMetricsEndpoint() string {
	if !pmc.config.Enabled {
		return ""
	}
	return fmt.Sprintf("http://%s:%d%s", pmc.config.Host, pmc.config.Port, pmc.config.Path)
}

// IsEnabled 检查Prometheus是否启用
func (pmc *PrometheusMetricsCollector) IsEnabled() bool {
	return pmc.config.Enabled
}
