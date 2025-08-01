package producer

import (
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusMetrics Prometheus指标收集器
type PrometheusMetrics struct {
	// 吞吐量指标
	MessagesPerSecond prometheus.Counter
	BytesPerSecond    prometheus.Counter
	BatchesPerSecond  prometheus.Counter

	// 延迟指标
	SendLatency          prometheus.Histogram
	SerializationLatency prometheus.Histogram
	CompressionLatency   prometheus.Histogram
	BatchLatency         prometheus.Histogram

	// 错误指标
	SendErrors      prometheus.Counter
	RetryAttempts   prometheus.Counter
	DroppedMessages prometheus.Counter
	TimeoutErrors   prometheus.Counter

	// 资源使用指标
	GoroutineCount    prometheus.Gauge
	MemoryUsage       prometheus.Gauge
	CPUUsage          prometheus.Gauge
	HeapSize          prometheus.Gauge
	GCPauseTime       prometheus.Histogram

	// 业务指标
	DeviceCount       prometheus.Gauge
	ActiveConnections prometheus.Gauge
	QueueDepth        prometheus.Gauge
	BatchSize         prometheus.Histogram
	MessageSize       prometheus.Histogram

	// 健康状态指标
	HealthStatus      prometheus.Gauge
	UptimeSeconds     prometheus.Counter
	LastSuccessTime   prometheus.Gauge
	ComponentStatus   *prometheus.GaugeVec

	// 性能优化指标
	ZeroCopyOperations prometheus.Counter
	MemoryPoolHits     prometheus.Counter
	MemoryPoolMisses   prometheus.Counter
	ConnectionReuse    prometheus.Counter

	// 内部状态
	startTime time.Time
	mutex     sync.RWMutex
}

// NewPrometheusMetrics 创建新的Prometheus指标收集器
func NewPrometheusMetrics(namespace, subsystem string) *PrometheusMetrics {
	labels := prometheus.Labels{
		"service": "iot-kafka-producer",
		"version": "1.0.0",
	}

	return &PrometheusMetrics{
		// 吞吐量指标
		MessagesPerSecond: promauto.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "messages_sent_total",
			Help:        "Total number of messages sent to Kafka",
			ConstLabels: labels,
		}),
		BytesPerSecond: promauto.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "bytes_sent_total",
			Help:        "Total bytes sent to Kafka",
			ConstLabels: labels,
		}),
		BatchesPerSecond: promauto.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "batches_sent_total",
			Help:        "Total number of batches sent to Kafka",
			ConstLabels: labels,
		}),

		// 延迟指标
		SendLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "send_latency_seconds",
			Help:        "Message send latency in seconds",
			ConstLabels: labels,
			Buckets:     prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to 16s
		}),
		SerializationLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "serialization_latency_seconds",
			Help:        "Message serialization latency in seconds",
			ConstLabels: labels,
			Buckets:     prometheus.ExponentialBuckets(0.0001, 2, 12), // 0.1ms to 400ms
		}),
		CompressionLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "compression_latency_seconds",
			Help:        "Message compression latency in seconds",
			ConstLabels: labels,
			Buckets:     prometheus.ExponentialBuckets(0.0001, 2, 12),
		}),
		BatchLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "batch_latency_seconds",
			Help:        "Batch processing latency in seconds",
			ConstLabels: labels,
			Buckets:     prometheus.ExponentialBuckets(0.001, 2, 15),
		}),

		// 错误指标
		SendErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "send_errors_total",
			Help:        "Total number of send errors",
			ConstLabels: labels,
		}),
		RetryAttempts: promauto.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "retry_attempts_total",
			Help:        "Total number of retry attempts",
			ConstLabels: labels,
		}),
		DroppedMessages: promauto.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "dropped_messages_total",
			Help:        "Total number of dropped messages",
			ConstLabels: labels,
		}),
		TimeoutErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "timeout_errors_total",
			Help:        "Total number of timeout errors",
			ConstLabels: labels,
		}),

		// 资源使用指标
		GoroutineCount: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "goroutines_count",
			Help:        "Current number of goroutines",
			ConstLabels: labels,
		}),
		MemoryUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "memory_usage_bytes",
			Help:        "Current memory usage in bytes",
			ConstLabels: labels,
		}),
		CPUUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "cpu_usage_percent",
			Help:        "Current CPU usage percentage",
			ConstLabels: labels,
		}),
		HeapSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "heap_size_bytes",
			Help:        "Current heap size in bytes",
			ConstLabels: labels,
		}),
		GCPauseTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "gc_pause_seconds",
			Help:        "Garbage collection pause time in seconds",
			ConstLabels: labels,
			Buckets:     prometheus.ExponentialBuckets(0.0001, 2, 15),
		}),

		// 业务指标
		DeviceCount: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "devices_count",
			Help:        "Current number of simulated devices",
			ConstLabels: labels,
		}),
		ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "active_connections_count",
			Help:        "Current number of active Kafka connections",
			ConstLabels: labels,
		}),
		QueueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "queue_depth",
			Help:        "Current queue depth",
			ConstLabels: labels,
		}),
		BatchSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "batch_size",
			Help:        "Distribution of batch sizes",
			ConstLabels: labels,
			Buckets:     prometheus.LinearBuckets(1, 10, 20), // 1 to 200
		}),
		MessageSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "message_size_bytes",
			Help:        "Distribution of message sizes in bytes",
			ConstLabels: labels,
			Buckets:     prometheus.ExponentialBuckets(64, 2, 15), // 64B to 1MB
		}),

		// 健康状态指标
		HealthStatus: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "health_status",
			Help:        "Overall health status (1=healthy, 0=unhealthy)",
			ConstLabels: labels,
		}),
		UptimeSeconds: promauto.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "uptime_seconds_total",
			Help:        "Total uptime in seconds",
			ConstLabels: labels,
		}),
		LastSuccessTime: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "last_success_timestamp",
			Help:        "Timestamp of last successful operation",
			ConstLabels: labels,
		}),
		ComponentStatus: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "component_status",
			Help:        "Status of individual components (1=healthy, 0=unhealthy)",
			ConstLabels: labels,
		}, []string{"component"}),

		// 性能优化指标
		ZeroCopyOperations: promauto.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "zero_copy_operations_total",
			Help:        "Total number of zero-copy operations",
			ConstLabels: labels,
		}),
		MemoryPoolHits: promauto.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "memory_pool_hits_total",
			Help:        "Total number of memory pool hits",
			ConstLabels: labels,
		}),
		MemoryPoolMisses: promauto.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "memory_pool_misses_total",
			Help:        "Total number of memory pool misses",
			ConstLabels: labels,
		}),
		ConnectionReuse: promauto.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "connection_reuse_total",
			Help:        "Total number of connection reuses",
			ConstLabels: labels,
		}),

		startTime: time.Now(),
	}
}

// UpdateResourceMetrics 更新资源使用指标
func (pm *PrometheusMetrics) UpdateResourceMetrics() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// 更新Goroutine数量
	pm.GoroutineCount.Set(float64(runtime.NumGoroutine()))

	// 更新内存使用情况
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	pm.MemoryUsage.Set(float64(memStats.Alloc))
	pm.HeapSize.Set(float64(memStats.HeapAlloc))
	
	// 记录GC暂停时间
	if memStats.NumGC > 0 {
		pm.GCPauseTime.Observe(float64(memStats.PauseNs[(memStats.NumGC+255)%256]) / 1e9)
	}

	// 更新运行时间
	pm.UptimeSeconds.Add(1)
}

// RecordSendLatency 记录发送延迟
func (pm *PrometheusMetrics) RecordSendLatency(duration time.Duration) {
	pm.SendLatency.Observe(duration.Seconds())
}

// RecordSerializationLatency 记录序列化延迟
func (pm *PrometheusMetrics) RecordSerializationLatency(duration time.Duration) {
	pm.SerializationLatency.Observe(duration.Seconds())
}

// RecordCompressionLatency 记录压缩延迟
func (pm *PrometheusMetrics) RecordCompressionLatency(duration time.Duration) {
	pm.CompressionLatency.Observe(duration.Seconds())
}

// RecordBatchLatency 记录批处理延迟
func (pm *PrometheusMetrics) RecordBatchLatency(duration time.Duration) {
	pm.BatchLatency.Observe(duration.Seconds())
}

// RecordBatchSize 记录批次大小
func (pm *PrometheusMetrics) RecordBatchSize(size int) {
	pm.BatchSize.Observe(float64(size))
}

// RecordMessageSize 记录消息大小
func (pm *PrometheusMetrics) RecordMessageSize(size int) {
	pm.MessageSize.Observe(float64(size))
}

// IncrementMessages 增加消息计数
func (pm *PrometheusMetrics) IncrementMessages() {
	pm.MessagesPerSecond.Inc()
}

// IncrementBytes 增加字节计数
func (pm *PrometheusMetrics) IncrementBytes(bytes int64) {
	pm.BytesPerSecond.Add(float64(bytes))
}

// IncrementBatches 增加批次计数
func (pm *PrometheusMetrics) IncrementBatches() {
	pm.BatchesPerSecond.Inc()
}

// IncrementSendErrors 增加发送错误计数
func (pm *PrometheusMetrics) IncrementSendErrors() {
	pm.SendErrors.Inc()
}

// IncrementRetryAttempts 增加重试计数
func (pm *PrometheusMetrics) IncrementRetryAttempts() {
	pm.RetryAttempts.Inc()
}

// IncrementDroppedMessages 增加丢弃消息计数
func (pm *PrometheusMetrics) IncrementDroppedMessages() {
	pm.DroppedMessages.Inc()
}

// IncrementTimeoutErrors 增加超时错误计数
func (pm *PrometheusMetrics) IncrementTimeoutErrors() {
	pm.TimeoutErrors.Inc()
}

// IncrementZeroCopyOperations 增加零拷贝操作计数
func (pm *PrometheusMetrics) IncrementZeroCopyOperations() {
	pm.ZeroCopyOperations.Inc()
}

// IncrementMemoryPoolHits 增加内存池命中计数
func (pm *PrometheusMetrics) IncrementMemoryPoolHits() {
	pm.MemoryPoolHits.Inc()
}

// IncrementMemoryPoolMisses 增加内存池未命中计数
func (pm *PrometheusMetrics) IncrementMemoryPoolMisses() {
	pm.MemoryPoolMisses.Inc()
}

// IncrementConnectionReuse 增加连接复用计数
func (pm *PrometheusMetrics) IncrementConnectionReuse() {
	pm.ConnectionReuse.Inc()
}

// SetDeviceCount 设置设备数量
func (pm *PrometheusMetrics) SetDeviceCount(count int) {
	pm.DeviceCount.Set(float64(count))
}

// SetActiveConnections 设置活跃连接数
func (pm *PrometheusMetrics) SetActiveConnections(count int) {
	pm.ActiveConnections.Set(float64(count))
}

// SetQueueDepth 设置队列深度
func (pm *PrometheusMetrics) SetQueueDepth(depth int) {
	pm.QueueDepth.Set(float64(depth))
}

// SetHealthStatus 设置健康状态
func (pm *PrometheusMetrics) SetHealthStatus(healthy bool) {
	if healthy {
		pm.HealthStatus.Set(1)
	} else {
		pm.HealthStatus.Set(0)
	}
}

// SetComponentStatus 设置组件状态
func (pm *PrometheusMetrics) SetComponentStatus(component string, healthy bool) {
	if healthy {
		pm.ComponentStatus.WithLabelValues(component).Set(1)
	} else {
		pm.ComponentStatus.WithLabelValues(component).Set(0)
	}
}

// UpdateLastSuccessTime 更新最后成功时间
func (pm *PrometheusMetrics) UpdateLastSuccessTime() {
	pm.LastSuccessTime.Set(float64(time.Now().Unix()))
}

// GetUptime 获取运行时间
func (pm *PrometheusMetrics) GetUptime() time.Duration {
	return time.Since(pm.startTime)
}
