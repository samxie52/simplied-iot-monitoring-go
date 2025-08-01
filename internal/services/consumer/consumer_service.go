package consumer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"simplied-iot-monitoring-go/internal/config"
)

// ConsumerService 消费者服务
type ConsumerService struct {
	// 配置
	config *config.AppConfig

	// Kafka消费者
	kafkaConsumer *KafkaConsumer

	// 消息处理器
	messageProcessor MessageProcessor

	// 错误处理器
	errorHandler ErrorHandler

	// 健康监控
	healthMonitor *ConsumerHealthMonitor

	// 控制和状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mutex     sync.RWMutex

	// 指标收集
	metricsCollector MetricsCollector
}

// ConsumerHealthMonitor 消费者健康监控器
type ConsumerHealthMonitor struct {
	consumer      *KafkaConsumer
	checkInterval time.Duration
	healthStatus  *config.HealthStatus
	mutex         sync.RWMutex

	// 健康检查配置
	maxProcessingErrors int64
	maxLagThreshold     int64
	minThroughput       int64
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	CollectConsumerMetrics(metrics *ConsumerMetrics)
	CollectProcessorMetrics(stats ProcessorStats)
	CollectErrorMetrics(stats ErrorStats)
	GetCollectedMetrics() map[string]interface{}
}

// NewConsumerService 创建消费者服务
func NewConsumerService(cfg *config.AppConfig) (*ConsumerService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建消息处理器
	processorConfig := ProcessorConfig{
		MaxRetries:           3,
		RetryDelay:           1 * time.Second,
		ProcessingTimeout:    30 * time.Second,
		EnableValidation:     true,
		EnableTransformation: true,
		EnableCaching:        false,
		BatchSize:            100,
		FlushInterval:        5 * time.Second,
	}
	messageProcessor := NewDefaultMessageProcessor(processorConfig)

	// 添加基础验证器和转换器
	messageProcessor.AddValidator(&BasicMessageValidator{})
	messageProcessor.AddTransformer(&TimestampTransformer{})
	messageProcessor.AddTransformer(&PayloadTransformer{})

	// 注册消息处理器
	messageProcessor.RegisterHandler(NewDeviceDataHandler(nil)) // TODO: 注入实际的设备管理器
	messageProcessor.RegisterHandler(NewAlertHandler(nil))      // TODO: 注入实际的告警管理器

	// 创建错误处理器
	errorConfig := ErrorHandlerConfig{
		MaxRetries:        3,
		RetryDelay:        1 * time.Second,
		BackoffMultiplier: 2.0,
		MaxRetryDelay:     30 * time.Second,
		EnableDLQ:         false, // 暂时禁用死信队列
		LogErrors:         true,
		AlertOnErrors:     false,
	}
	errorHandler := NewDefaultErrorHandler(errorConfig)

	// 创建Kafka消费者
	topics := []string{cfg.Kafka.Topics.DeviceData, cfg.Kafka.Topics.Alerts}
	kafkaConsumer, err := NewKafkaConsumer(
		cfg.Kafka.Brokers,
		topics,
		&cfg.Kafka.Consumer,
		messageProcessor,
		errorHandler,
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	// 创建健康监控器
	healthMonitor := &ConsumerHealthMonitor{
		consumer:            kafkaConsumer,
		checkInterval:       30 * time.Second,
		maxProcessingErrors: 100,
		maxLagThreshold:     1000,
		minThroughput:       10,
	}

	// 创建Prometheus指标收集器
	var metricsCollector MetricsCollector
	if cfg.Monitor.Prometheus.Enabled {
		prometheusCollector, err := NewPrometheusMetricsCollector(&cfg.Monitor.Prometheus)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create prometheus metrics collector: %w", err)
		}
		metricsCollector = prometheusCollector
		log.Println("Prometheus指标收集器已启用")
	} else {
		metricsCollector = NewSimpleMetricsCollector()
		log.Println("使用简单指标收集器")
	}

	service := &ConsumerService{
		config:           cfg,
		kafkaConsumer:    kafkaConsumer,
		messageProcessor: messageProcessor,
		errorHandler:     errorHandler,
		healthMonitor:    healthMonitor,
		metricsCollector: metricsCollector,
		ctx:              ctx,
		cancel:           cancel,
		isRunning:        false,
	}

	return service, nil
}

// Start 启动消费者服务
func (cs *ConsumerService) Start() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if cs.isRunning {
		return fmt.Errorf("consumer service is already running")
	}

	log.Println("正在启动消费者服务...")

	// 启动Kafka消费者
	if err := cs.kafkaConsumer.Start(); err != nil {
		return fmt.Errorf("failed to start kafka consumer: %w", err)
	}

	// 启动健康监控
	cs.wg.Add(1)
	go cs.runHealthMonitor()

	// 启动Prometheus指标收集器
	if prometheusCollector, ok := cs.metricsCollector.(*PrometheusMetricsCollector); ok {
		if err := prometheusCollector.Start(); err != nil {
			log.Printf("Failed to start Prometheus metrics collector: %v", err)
		}
	}

	// 启动指标收集
	if cs.metricsCollector != nil {
		cs.wg.Add(1)
		go cs.runMetricsCollection()
	}

	cs.isRunning = true
	log.Println("消费者服务启动成功")

	return nil
}

// Stop 停止消费者服务
func (cs *ConsumerService) Stop() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if !cs.isRunning {
		return nil
	}

	log.Println("正在停止消费者服务...")

	// 取消上下文
	cs.cancel()

	// 停止Kafka消费者
	if err := cs.kafkaConsumer.Stop(); err != nil {
		log.Printf("停止Kafka消费者时发生错误: %v", err)
	}

	// 停止Prometheus指标收集器
	if prometheusCollector, ok := cs.metricsCollector.(*PrometheusMetricsCollector); ok {
		if err := prometheusCollector.Stop(); err != nil {
			log.Printf("Failed to stop Prometheus metrics collector: %v", err)
		}
	}

	// 等待所有协程结束
	cs.wg.Wait()

	cs.isRunning = false
	log.Println("消费者服务已停止")

	return nil
}

// IsRunning 检查服务是否运行中
func (cs *ConsumerService) IsRunning() bool {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	return cs.isRunning
}

// GetHealth 获取健康状态
func (cs *ConsumerService) GetHealth() *config.HealthStatus {
	cs.healthMonitor.mutex.RLock()
	defer cs.healthMonitor.mutex.RUnlock()

	if cs.healthMonitor.healthStatus != nil {
		return cs.healthMonitor.healthStatus
	}

	// 返回默认健康状态
	return &config.HealthStatus{
		Healthy:   cs.IsRunning(),
		Timestamp: time.Now(),
		Services: map[string]bool{
			"consumer_service": cs.IsRunning(),
			"kafka_consumer":   cs.kafkaConsumer.IsRunning(),
		},
		Metrics: map[string]string{
			"status": "unknown",
		},
	}
}

// GetMetrics 获取服务指标
func (cs *ConsumerService) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Kafka消费者指标
	consumerMetrics := cs.kafkaConsumer.GetMetrics()
	metrics["consumer"] = map[string]interface{}{
		"messages_consumed":     consumerMetrics.MessagesConsumed,
		"messages_processed":    consumerMetrics.MessagesProcessed,
		"processing_errors":     consumerMetrics.ProcessingErrors,
		"rebalance_count":       consumerMetrics.RebalanceCount,
		"partition_count":       consumerMetrics.PartitionCount,
		"throughput_per_second": consumerMetrics.ThroughputPerSecond,
		"processing_latency_ms": consumerMetrics.ProcessingLatencyMs,
		"active_partitions":     consumerMetrics.ActivePartitions,
	}

	// 消息处理器指标
	processorStats := cs.messageProcessor.GetStats()
	metrics["processor"] = map[string]interface{}{
		"processed_count": processorStats.ProcessedCount,
		"error_count":     processorStats.ErrorCount,
		"avg_latency_ms":  processorStats.AvgLatencyMs,
	}

	// 错误处理器指标
	errorStats := cs.errorHandler.GetErrorStats()
	metrics["error_handler"] = map[string]interface{}{
		"total_errors":     errorStats.TotalErrors,
		"retryable_errors": errorStats.RetryableErrors,
		"fatal_errors":     errorStats.FatalErrors,
	}

	// 添加收集的指标
	if cs.metricsCollector != nil {
		collectedMetrics := cs.metricsCollector.GetCollectedMetrics()
		for key, value := range collectedMetrics {
			metrics[key] = value
		}
	}

	return metrics
}

// runHealthMonitor 运行健康监控
func (cs *ConsumerService) runHealthMonitor() {
	defer cs.wg.Done()

	ticker := time.NewTicker(cs.healthMonitor.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cs.performHealthCheck()
		case <-cs.ctx.Done():
			return
		}
	}
}

// performHealthCheck 执行健康检查
func (cs *ConsumerService) performHealthCheck() {
	cs.healthMonitor.mutex.Lock()
	defer cs.healthMonitor.mutex.Unlock()

	// 获取消费者指标
	metrics := cs.kafkaConsumer.GetMetrics()

	// 检查健康状态
	healthy := true
	issues := make([]string, 0)

	// 检查是否运行
	if !cs.kafkaConsumer.IsRunning() {
		healthy = false
		issues = append(issues, "kafka consumer not running")
	}

	// 检查处理错误数
	if metrics.ProcessingErrors > cs.healthMonitor.maxProcessingErrors {
		healthy = false
		issues = append(issues, fmt.Sprintf("too many processing errors: %d", metrics.ProcessingErrors))
	}

	// 检查消息延迟
	if metrics.LagTotal > cs.healthMonitor.maxLagThreshold {
		healthy = false
		issues = append(issues, fmt.Sprintf("message lag too high: %d", metrics.LagTotal))
	}

	// 检查吞吐量
	if metrics.ThroughputPerSecond < cs.healthMonitor.minThroughput {
		healthy = false
		issues = append(issues, fmt.Sprintf("throughput too low: %d", metrics.ThroughputPerSecond))
	}

	// 更新健康状态
	cs.healthMonitor.healthStatus = &config.HealthStatus{
		Healthy:   healthy,
		Timestamp: time.Now(),
		Services: map[string]bool{
			"consumer_service": cs.IsRunning(),
			"kafka_consumer":   cs.kafkaConsumer.IsRunning(),
		},
		Metrics: map[string]string{
			"messages_consumed":     fmt.Sprintf("%d", metrics.MessagesConsumed),
			"messages_processed":    fmt.Sprintf("%d", metrics.MessagesProcessed),
			"processing_errors":     fmt.Sprintf("%d", metrics.ProcessingErrors),
			"throughput_per_second": fmt.Sprintf("%d", metrics.ThroughputPerSecond),
			"lag_total":             fmt.Sprintf("%d", metrics.LagTotal),
		},
	}

	if !healthy {
		log.Printf("健康检查失败: %v", issues)
	}
}

// runMetricsCollection 运行指标收集
func (cs *ConsumerService) runMetricsCollection() {
	defer cs.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // 每10秒收集一次指标
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cs.collectMetrics()
		case <-cs.ctx.Done():
			return
		}
	}
}

// collectMetrics 收集指标
func (cs *ConsumerService) collectMetrics() {
	if cs.metricsCollector == nil {
		return
	}

	// 收集消费者指标
	consumerMetrics := cs.kafkaConsumer.GetMetrics()
	cs.metricsCollector.CollectConsumerMetrics(consumerMetrics)

	// 收集处理器指标
	processorStats := cs.messageProcessor.GetStats()
	cs.metricsCollector.CollectProcessorMetrics(processorStats)

	// 收集错误处理器指标
	errorStats := cs.errorHandler.GetErrorStats()
	cs.metricsCollector.CollectErrorMetrics(errorStats)
}

// SetMetricsCollector 设置指标收集器
func (cs *ConsumerService) SetMetricsCollector(collector MetricsCollector) {
	cs.metricsCollector = collector
}

// GetKafkaConsumer 获取Kafka消费者
func (cs *ConsumerService) GetKafkaConsumer() *KafkaConsumer {
	return cs.kafkaConsumer
}

// GetMessageProcessor 获取消息处理器
func (cs *ConsumerService) GetMessageProcessor() MessageProcessor {
	return cs.messageProcessor
}

// GetErrorHandler 获取错误处理器
func (cs *ConsumerService) GetErrorHandler() ErrorHandler {
	return cs.errorHandler
}

// UpdateConfig 更新配置
func (cs *ConsumerService) UpdateConfig(newConfig *config.AppConfig) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if cs.isRunning {
		return fmt.Errorf("cannot update config while service is running")
	}

	cs.config = newConfig
	log.Println("消费者服务配置已更新")

	return nil
}

// GetConfig 获取配置
func (cs *ConsumerService) GetConfig() *config.AppConfig {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	return cs.config
}

// SimpleMetricsCollector 简单指标收集器实现
type SimpleMetricsCollector struct {
	metrics map[string]interface{}
	mutex   sync.RWMutex
}

// NewSimpleMetricsCollector 创建简单指标收集器
func NewSimpleMetricsCollector() *SimpleMetricsCollector {
	return &SimpleMetricsCollector{
		metrics: make(map[string]interface{}),
	}
}

// CollectConsumerMetrics 收集消费者指标
func (smc *SimpleMetricsCollector) CollectConsumerMetrics(metrics *ConsumerMetrics) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()

	smc.metrics["consumer_messages_consumed"] = metrics.MessagesConsumed
	smc.metrics["consumer_messages_processed"] = metrics.MessagesProcessed
	smc.metrics["consumer_processing_errors"] = metrics.ProcessingErrors
	smc.metrics["consumer_rebalance_count"] = metrics.RebalanceCount
	smc.metrics["consumer_partition_count"] = metrics.PartitionCount
	smc.metrics["consumer_throughput_per_second"] = metrics.ThroughputPerSecond
	smc.metrics["consumer_processing_latency_ms"] = metrics.ProcessingLatencyMs
	smc.metrics["consumer_last_message_timestamp"] = metrics.LastMessageTimestamp
}

// CollectProcessorMetrics 收集处理器指标
func (smc *SimpleMetricsCollector) CollectProcessorMetrics(stats ProcessorStats) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()

	smc.metrics["processor_processed_count"] = stats.ProcessedCount
	smc.metrics["processor_error_count"] = stats.ErrorCount
	smc.metrics["processor_avg_latency_ms"] = stats.AvgLatencyMs
}

// CollectErrorMetrics 收集错误指标
func (smc *SimpleMetricsCollector) CollectErrorMetrics(stats ErrorStats) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()

	smc.metrics["error_total_errors"] = stats.TotalErrors
	smc.metrics["error_retryable_errors"] = stats.RetryableErrors
	smc.metrics["error_fatal_errors"] = stats.FatalErrors
}

// GetCollectedMetrics 获取收集的指标
func (smc *SimpleMetricsCollector) GetCollectedMetrics() map[string]interface{} {
	smc.mutex.RLock()
	defer smc.mutex.RUnlock()

	result := make(map[string]interface{})
	for key, value := range smc.metrics {
		result[key] = value
	}

	return result
}
