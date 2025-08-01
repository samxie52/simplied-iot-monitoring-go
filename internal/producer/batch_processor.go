package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"simplied-iot-monitoring-go/internal/config"
)

// BatchProcessor 高性能批量消息处理器
type BatchProcessor struct {
	config          *config.KafkaProducer
	producer        sarama.AsyncProducer
	
	// 批处理配置
	batchSize       int
	batchTimeout    time.Duration
	compressionType string
	
	// 消息缓冲区
	messageBuffer   chan *ProducerMessage
	batchBuffer     []*ProducerMessage
	
	// 分区策略
	partitioner     Partitioner
	partitionCount  int32
	
	// 并发控制
	workerPool      *BatchWorkerPool
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	
	// 性能指标
	metrics         *BatchMetrics
	
	// 状态管理
	isRunning       bool
	mutex           sync.RWMutex
	
	// 错误处理
	errorHandler    ErrorHandler
	retryManager    *RetryManager
	
	// 流量控制
	rateLimiter     *RateLimiter
	backpressure    *BackpressureController
}

// ProducerMessage 生产者消息
type ProducerMessage struct {
	Key       string
	Value     interface{}
	Topic     string
	Partition int32
	Headers   map[string]string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// BatchMetrics 批处理指标
type BatchMetrics struct {
	// 消息统计
	TotalMessages       int64
	TotalBytes          int64
	TotalBatches        int64
	
	// 性能指标
	MessagesPerSecond   int64
	BytesPerSecond      int64
	BatchesPerSecond    int64
	AvgBatchSize        float64
	AvgLatency          time.Duration
	
	// 错误统计
	SendErrors          int64
	RetryAttempts       int64
	DroppedMessages     int64
	
	// 资源使用
	QueueDepth          int64
	MemoryUsage         int64
	GoroutineCount      int64
	
	// 分区统计
	PartitionDistribution map[int32]int64
	
	mutex               sync.RWMutex
}

// Partitioner 分区器接口
type Partitioner interface {
	Partition(message *ProducerMessage, partitionCount int32) int32
}

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	HandleError(err error, message *ProducerMessage) bool // 返回true表示重试
}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor(config *config.KafkaProducer, producer sarama.AsyncProducer, partitionCount int32) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	bp := &BatchProcessor{
		config:          config,
		producer:        producer,
		batchSize:       config.BatchSize,
		batchTimeout:    config.BatchTimeout,
		compressionType: config.CompressionType,
		
		messageBuffer:   make(chan *ProducerMessage, config.ChannelBufferSize),
		batchBuffer:     make([]*ProducerMessage, 0, config.BatchSize),
		
		partitioner:     NewHashPartitioner(),
		partitionCount:  partitionCount,
		
		workerPool:      NewBatchWorkerPool(4, 1000), // 4个工作协程，1000个任务缓冲
		ctx:             ctx,
		cancel:          cancel,
		
		metrics:         NewBatchMetrics(),
		
		isRunning:       false,
		
		errorHandler:    NewDefaultErrorHandler(),
		retryManager:    NewRetryManager(config.MaxRetries, config.RetryBackoff),
		
		rateLimiter:     NewRateLimiter(10000), // 10K messages/sec limit
		backpressure:    NewBackpressureController(),
	}
	
	return bp
}

// Start 启动批处理器
func (bp *BatchProcessor) Start() error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	if bp.isRunning {
		return fmt.Errorf("batch processor is already running")
	}
	
	// 启动工作池
	if err := bp.workerPool.Start(); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}
	
	// 启动批处理协程
	bp.wg.Add(3)
	go bp.batchCollector()
	go bp.metricsCollector()
	go bp.backpressureMonitor()
	
	bp.isRunning = true
	return nil
}

// SendMessage 发送消息
func (bp *BatchProcessor) SendMessage(message *ProducerMessage) error {
	if !bp.isRunning {
		return fmt.Errorf("batch processor is not running")
	}
	
	// 流量控制
	if !bp.rateLimiter.Allow() {
		atomic.AddInt64(&bp.metrics.DroppedMessages, 1)
		return fmt.Errorf("rate limit exceeded")
	}
	
	// 背压控制
	if bp.backpressure.ShouldDrop() {
		atomic.AddInt64(&bp.metrics.DroppedMessages, 1)
		return fmt.Errorf("backpressure detected, dropping message")
	}
	
	// 设置分区
	if message.Partition == -1 {
		message.Partition = bp.partitioner.Partition(message, bp.partitionCount)
	}
	
	// 设置时间戳
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}
	
	select {
	case bp.messageBuffer <- message:
		atomic.AddInt64(&bp.metrics.QueueDepth, 1)
		return nil
	case <-bp.ctx.Done():
		return fmt.Errorf("batch processor is shutting down")
	default:
		atomic.AddInt64(&bp.metrics.DroppedMessages, 1)
		return fmt.Errorf("message buffer is full")
	}
}

// batchCollector 批量收集器
func (bp *BatchProcessor) batchCollector() {
	defer bp.wg.Done()
	
	ticker := time.NewTicker(bp.batchTimeout)
	defer ticker.Stop()
	
	for {
		select {
		case message := <-bp.messageBuffer:
			atomic.AddInt64(&bp.metrics.QueueDepth, -1)
			bp.batchBuffer = append(bp.batchBuffer, message)
			
			// 检查是否达到批次大小
			if len(bp.batchBuffer) >= bp.batchSize {
				bp.processBatch()
			}
			
		case <-ticker.C:
			// 超时处理
			if len(bp.batchBuffer) > 0 {
				bp.processBatch()
			}
			
		case <-bp.ctx.Done():
			// 处理剩余消息
			if len(bp.batchBuffer) > 0 {
				bp.processBatch()
			}
			return
		}
	}
}

// processBatch 处理批次
func (bp *BatchProcessor) processBatch() {
	if len(bp.batchBuffer) == 0 {
		return
	}
	
	batch := make([]*ProducerMessage, len(bp.batchBuffer))
	copy(batch, bp.batchBuffer)
	bp.batchBuffer = bp.batchBuffer[:0] // 重置缓冲区
	
	// 提交到工作池处理
	task := &BatchTask{
		Batch:      batch,
		Processor:  bp,
		StartTime:  time.Now(),
	}
	
	if err := bp.workerPool.Submit(task); err != nil {
		// 工作池满，直接处理
		task.Execute()
	}
}

// BatchTask 批处理任务
type BatchTask struct {
	Batch     []*ProducerMessage
	Processor *BatchProcessor
	StartTime time.Time
}

// Execute 执行批处理任务
func (bt *BatchTask) Execute() error {
	return bt.Processor.sendBatch(bt.Batch, bt.StartTime)
}

// sendBatch 发送批次
func (bp *BatchProcessor) sendBatch(batch []*ProducerMessage, startTime time.Time) error {
	if len(batch) == 0 {
		return nil
	}
	
	// 按分区分组
	partitionBatches := bp.groupByPartition(batch)
	
	// 并行发送到不同分区
	var wg sync.WaitGroup
	errors := make(chan error, len(partitionBatches))
	
	for partition, messages := range partitionBatches {
		wg.Add(1)
		go func(p int32, msgs []*ProducerMessage) {
			defer wg.Done()
			if err := bp.sendPartitionBatch(p, msgs); err != nil {
				errors <- err
			}
		}(partition, messages)
	}
	
	wg.Wait()
	close(errors)
	
	// 收集错误
	var lastError error
	errorCount := 0
	for err := range errors {
		lastError = err
		errorCount++
		atomic.AddInt64(&bp.metrics.SendErrors, 1)
	}
	
	// 更新指标
	bp.updateBatchMetrics(batch, startTime, errorCount == 0)
	
	if errorCount > 0 {
		return fmt.Errorf("batch send failed with %d errors, last error: %w", errorCount, lastError)
	}
	
	return nil
}

// groupByPartition 按分区分组
func (bp *BatchProcessor) groupByPartition(batch []*ProducerMessage) map[int32][]*ProducerMessage {
	partitionBatches := make(map[int32][]*ProducerMessage)
	
	for _, message := range batch {
		partition := message.Partition
		partitionBatches[partition] = append(partitionBatches[partition], message)
	}
	
	return partitionBatches
}

// sendPartitionBatch 发送分区批次
func (bp *BatchProcessor) sendPartitionBatch(partition int32, messages []*ProducerMessage) error {
	for _, message := range messages {
		// 序列化消息
		valueBytes, err := json.Marshal(message.Value)
		if err != nil {
			if bp.errorHandler.HandleError(err, message) {
				// 重试逻辑
				return bp.retryManager.Retry(func() error {
					return bp.sendSingleMessage(message)
				})
			}
			continue
		}
		
		// 创建Sarama消息
		saramaMsg := &sarama.ProducerMessage{
			Topic:     message.Topic,
			Key:       sarama.StringEncoder(message.Key),
			Value:     sarama.ByteEncoder(valueBytes),
			Partition: partition,
			Timestamp: message.Timestamp,
		}
		
		// 添加头部信息
		if len(message.Headers) > 0 {
			headers := make([]sarama.RecordHeader, 0, len(message.Headers))
			for k, v := range message.Headers {
				headers = append(headers, sarama.RecordHeader{
					Key:   []byte(k),
					Value: []byte(v),
				})
			}
			saramaMsg.Headers = headers
		}
		
		// 发送到Kafka
		select {
		case bp.producer.Input() <- saramaMsg:
			atomic.AddInt64(&bp.metrics.TotalMessages, 1)
			atomic.AddInt64(&bp.metrics.TotalBytes, int64(len(valueBytes)))
			
			// 更新分区统计
			bp.metrics.mutex.Lock()
			if bp.metrics.PartitionDistribution == nil {
				bp.metrics.PartitionDistribution = make(map[int32]int64)
			}
			bp.metrics.PartitionDistribution[partition]++
			bp.metrics.mutex.Unlock()
			
		case <-bp.ctx.Done():
			return fmt.Errorf("producer is shutting down")
		}
	}
	
	return nil
}

// sendSingleMessage 发送单个消息（重试用）
func (bp *BatchProcessor) sendSingleMessage(message *ProducerMessage) error {
	valueBytes, err := json.Marshal(message.Value)
	if err != nil {
		return err
	}
	
	saramaMsg := &sarama.ProducerMessage{
		Topic:     message.Topic,
		Key:       sarama.StringEncoder(message.Key),
		Value:     sarama.ByteEncoder(valueBytes),
		Partition: message.Partition,
		Timestamp: message.Timestamp,
	}
	
	select {
	case bp.producer.Input() <- saramaMsg:
		return nil
	case <-bp.ctx.Done():
		return fmt.Errorf("producer is shutting down")
	}
}

// updateBatchMetrics 更新批次指标
func (bp *BatchProcessor) updateBatchMetrics(batch []*ProducerMessage, startTime time.Time, success bool) {
	latency := time.Since(startTime)
	batchSize := len(batch)
	
	atomic.AddInt64(&bp.metrics.TotalBatches, 1)
	atomic.AddInt64(&bp.metrics.BatchesPerSecond, 1)
	
	if success {
		// 更新平均批次大小
		bp.metrics.mutex.Lock()
		currentAvg := bp.metrics.AvgBatchSize
		totalBatches := float64(atomic.LoadInt64(&bp.metrics.TotalBatches))
		bp.metrics.AvgBatchSize = (currentAvg*(totalBatches-1) + float64(batchSize)) / totalBatches
		
		// 更新平均延迟
		currentLatency := bp.metrics.AvgLatency
		bp.metrics.AvgLatency = time.Duration((int64(currentLatency)*(int64(totalBatches)-1) + int64(latency)) / int64(totalBatches))
		bp.metrics.mutex.Unlock()
	}
}

// metricsCollector 指标收集器
func (bp *BatchProcessor) metricsCollector() {
	defer bp.wg.Done()
	
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	var lastMessages, lastBytes, lastBatches int64
	
	for {
		select {
		case <-ticker.C:
			// 计算每秒指标
			currentMessages := atomic.LoadInt64(&bp.metrics.TotalMessages)
			currentBytes := atomic.LoadInt64(&bp.metrics.TotalBytes)
			currentBatches := atomic.LoadInt64(&bp.metrics.TotalBatches)
			
			atomic.StoreInt64(&bp.metrics.MessagesPerSecond, currentMessages-lastMessages)
			atomic.StoreInt64(&bp.metrics.BytesPerSecond, currentBytes-lastBytes)
			atomic.StoreInt64(&bp.metrics.BatchesPerSecond, currentBatches-lastBatches)
			
			lastMessages = currentMessages
			lastBytes = currentBytes
			lastBatches = currentBatches
			
		case <-bp.ctx.Done():
			return
		}
	}
}

// backpressureMonitor 背压监控
func (bp *BatchProcessor) backpressureMonitor() {
	defer bp.wg.Done()
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			queueDepth := atomic.LoadInt64(&bp.metrics.QueueDepth)
			bp.backpressure.Update(queueDepth, int64(cap(bp.messageBuffer)))
			
		case <-bp.ctx.Done():
			return
		}
	}
}

// Stop 停止批处理器
func (bp *BatchProcessor) Stop() error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	if !bp.isRunning {
		return nil
	}
	
	bp.cancel()
	bp.wg.Wait()
	
	if err := bp.workerPool.Stop(); err != nil {
		return fmt.Errorf("failed to stop worker pool: %w", err)
	}
	
	bp.isRunning = false
	return nil
}

// GetMetrics 获取指标
func (bp *BatchProcessor) GetMetrics() *BatchMetrics {
	bp.metrics.mutex.RLock()
	defer bp.metrics.mutex.RUnlock()
	
	// 返回指标副本
	partitionDist := make(map[int32]int64)
	for k, v := range bp.metrics.PartitionDistribution {
		partitionDist[k] = v
	}
	
	return &BatchMetrics{
		TotalMessages:         atomic.LoadInt64(&bp.metrics.TotalMessages),
		TotalBytes:           atomic.LoadInt64(&bp.metrics.TotalBytes),
		TotalBatches:         atomic.LoadInt64(&bp.metrics.TotalBatches),
		MessagesPerSecond:    atomic.LoadInt64(&bp.metrics.MessagesPerSecond),
		BytesPerSecond:       atomic.LoadInt64(&bp.metrics.BytesPerSecond),
		BatchesPerSecond:     atomic.LoadInt64(&bp.metrics.BatchesPerSecond),
		AvgBatchSize:         bp.metrics.AvgBatchSize,
		AvgLatency:           bp.metrics.AvgLatency,
		SendErrors:           atomic.LoadInt64(&bp.metrics.SendErrors),
		RetryAttempts:        atomic.LoadInt64(&bp.metrics.RetryAttempts),
		DroppedMessages:      atomic.LoadInt64(&bp.metrics.DroppedMessages),
		QueueDepth:           atomic.LoadInt64(&bp.metrics.QueueDepth),
		MemoryUsage:          atomic.LoadInt64(&bp.metrics.MemoryUsage),
		GoroutineCount:       atomic.LoadInt64(&bp.metrics.GoroutineCount),
		PartitionDistribution: partitionDist,
	}
}

// NewBatchMetrics 创建批处理指标
func NewBatchMetrics() *BatchMetrics {
	return &BatchMetrics{
		PartitionDistribution: make(map[int32]int64),
	}
}

// IsRunning 检查是否运行中
func (bp *BatchProcessor) IsRunning() bool {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	return bp.isRunning
}
