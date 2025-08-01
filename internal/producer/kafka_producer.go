package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"simplied-iot-monitoring-go/internal/config"
)

// KafkaProducer Kafka生产者实现
type KafkaProducer struct {
	producer     sarama.AsyncProducer
	config       *config.KafkaProducer
	topic        string
	batchBuffer  chan *sarama.ProducerMessage
	errorChan    chan *sarama.ProducerError
	successChan  chan *sarama.ProducerMessage
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	isRunning    bool
	mutex        sync.RWMutex
	metrics      *ProducerMetrics
}

// ProducerMetrics 生产者指标
type ProducerMetrics struct {
	MessagesPerSecond    int64
	BytesPerSecond       int64
	BatchesPerSecond     int64
	SendErrors           int64
	RetryAttempts        int64
	DroppedMessages      int64
	GoroutineCount       int64
	MemoryUsage          int64
	CPUUsage             float64
	DeviceCount          int64
	ActiveConnections    int64
	QueueDepth           int64
	mutex                sync.RWMutex
}

// NewKafkaProducer 创建新的Kafka生产者
func NewKafkaProducer(brokers []string, topic string, producerConfig *config.KafkaProducer) (*KafkaProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.RequiredAcks = sarama.RequiredAcks(producerConfig.RequiredAcks)
	saramaConfig.Producer.Retry.Max = producerConfig.MaxRetries
	saramaConfig.Producer.Retry.Backoff = producerConfig.RetryBackoff
	saramaConfig.Producer.Flush.Frequency = producerConfig.FlushFrequency
	saramaConfig.Producer.Flush.Messages = producerConfig.BatchSize
	saramaConfig.ClientID = producerConfig.ClientID

	// 设置压缩类型
	switch producerConfig.CompressionType {
	case "gzip":
		saramaConfig.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		saramaConfig.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		saramaConfig.Producer.Compression = sarama.CompressionLZ4
	case "zstd":
		saramaConfig.Producer.Compression = sarama.CompressionZSTD
	default:
		saramaConfig.Producer.Compression = sarama.CompressionNone
	}

	producer, err := sarama.NewAsyncProducer(brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	kp := &KafkaProducer{
		producer:    producer,
		config:      producerConfig,
		topic:       topic,
		batchBuffer: make(chan *sarama.ProducerMessage, producerConfig.ChannelBufferSize),
		errorChan:   make(chan *sarama.ProducerError, 100),
		successChan: make(chan *sarama.ProducerMessage, 100),
		ctx:         ctx,
		cancel:      cancel,
		isRunning:   false,
		metrics:     &ProducerMetrics{},
	}

	return kp, nil
}

// Start 启动生产者
func (kp *KafkaProducer) Start() error {
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	if kp.isRunning {
		return fmt.Errorf("producer is already running")
	}

	kp.isRunning = true

	// 启动消息处理协程
	kp.wg.Add(3)
	go kp.handleSuccesses()
	go kp.handleErrors()
	go kp.batchProcessor()

	return nil
}

// SendMessage 发送消息
func (kp *KafkaProducer) SendMessage(key string, value interface{}) error {
	if !kp.isRunning {
		return fmt.Errorf("producer is not running")
	}

	// 序列化消息
	valueBytes, err := json.Marshal(value)
	if err != nil {
		kp.incrementSendErrors()
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	message := &sarama.ProducerMessage{
		Topic:     kp.topic,
		Key:       sarama.StringEncoder(key),
		Value:     sarama.ByteEncoder(valueBytes),
		Timestamp: time.Now(),
	}

	select {
	case kp.batchBuffer <- message:
		return nil
	case <-kp.ctx.Done():
		return fmt.Errorf("producer is shutting down")
	default:
		kp.incrementDroppedMessages()
		return fmt.Errorf("message buffer is full")
	}
}

// batchProcessor 批量处理消息
func (kp *KafkaProducer) batchProcessor() {
	defer kp.wg.Done()

	ticker := time.NewTicker(kp.config.BatchTimeout)
	defer ticker.Stop()

	batch := make([]*sarama.ProducerMessage, 0, kp.config.BatchSize)

	for {
		select {
		case message := <-kp.batchBuffer:
			batch = append(batch, message)

			if len(batch) >= kp.config.BatchSize {
				kp.sendBatch(batch)
				batch = batch[:0] // 重置切片
			}

		case <-ticker.C:
			if len(batch) > 0 {
				kp.sendBatch(batch)
				batch = batch[:0]
			}

		case <-kp.ctx.Done():
			// 发送剩余消息
			if len(batch) > 0 {
				kp.sendBatch(batch)
			}
			return
		}
	}
}

// sendBatch 发送批量消息
func (kp *KafkaProducer) sendBatch(batch []*sarama.ProducerMessage) {
	start := time.Now()

	for _, message := range batch {
		select {
		case kp.producer.Input() <- message:
			kp.incrementMessagesPerSecond()
		case <-kp.ctx.Done():
			return
		}
	}

	kp.incrementBatchesPerSecond()
	// 记录发送延迟（这里简化处理，实际应该使用Prometheus Histogram）
	_ = time.Since(start)
}

// handleSuccesses 处理成功消息
func (kp *KafkaProducer) handleSuccesses() {
	defer kp.wg.Done()

	for {
		select {
		case success := <-kp.producer.Successes():
			kp.incrementBytesPerSecond(int64(len(success.Value.(sarama.ByteEncoder))))

		case <-kp.ctx.Done():
			return
		}
	}
}

// handleErrors 处理错误消息
func (kp *KafkaProducer) handleErrors() {
	defer kp.wg.Done()

	for {
		select {
		case err := <-kp.producer.Errors():
			kp.incrementSendErrors()
			// 这里可以添加错误日志记录
			fmt.Printf("Kafka producer error: %v\n", err)

		case <-kp.ctx.Done():
			return
		}
	}
}

// Stop 停止生产者
func (kp *KafkaProducer) Stop() error {
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	if !kp.isRunning {
		return nil
	}

	kp.cancel()
	kp.wg.Wait()

	if err := kp.producer.Close(); err != nil {
		return fmt.Errorf("failed to close producer: %w", err)
	}

	kp.isRunning = false
	return nil
}

// GetMetrics 获取生产者指标
func (kp *KafkaProducer) GetMetrics() *ProducerMetrics {
	kp.metrics.mutex.RLock()
	defer kp.metrics.mutex.RUnlock()

	// 返回指标副本
	return &ProducerMetrics{
		MessagesPerSecond:    kp.metrics.MessagesPerSecond,
		BytesPerSecond:       kp.metrics.BytesPerSecond,
		BatchesPerSecond:     kp.metrics.BatchesPerSecond,
		SendErrors:           kp.metrics.SendErrors,
		RetryAttempts:        kp.metrics.RetryAttempts,
		DroppedMessages:      kp.metrics.DroppedMessages,
		GoroutineCount:       kp.metrics.GoroutineCount,
		MemoryUsage:          kp.metrics.MemoryUsage,
		CPUUsage:             kp.metrics.CPUUsage,
		DeviceCount:          kp.metrics.DeviceCount,
		ActiveConnections:    kp.metrics.ActiveConnections,
		QueueDepth:           kp.metrics.QueueDepth,
	}
}

// IsRunning 检查生产者是否运行中
func (kp *KafkaProducer) IsRunning() bool {
	kp.mutex.RLock()
	defer kp.mutex.RUnlock()
	return kp.isRunning
}

// GetHealth 获取健康状态
func (kp *KafkaProducer) GetHealth() *config.HealthStatus {
	metrics := kp.GetMetrics()
	
	healthy := kp.IsRunning() && metrics.SendErrors < 100 // 简单的健康检查逻辑
	
	return &config.HealthStatus{
		Healthy:   healthy,
		Timestamp: time.Now(),
		Services: map[string]bool{
			"kafka_producer": kp.IsRunning(),
		},
		Metrics: map[string]string{
			"messages_per_second": fmt.Sprintf("%d", metrics.MessagesPerSecond),
			"send_errors":         fmt.Sprintf("%d", metrics.SendErrors),
			"queue_depth":         fmt.Sprintf("%d", metrics.QueueDepth),
		},
	}
}

// 指标更新方法
func (kp *KafkaProducer) incrementMessagesPerSecond() {
	kp.metrics.mutex.Lock()
	defer kp.metrics.mutex.Unlock()
	kp.metrics.MessagesPerSecond++
}

func (kp *KafkaProducer) incrementBytesPerSecond(bytes int64) {
	kp.metrics.mutex.Lock()
	defer kp.metrics.mutex.Unlock()
	kp.metrics.BytesPerSecond += bytes
}

func (kp *KafkaProducer) incrementBatchesPerSecond() {
	kp.metrics.mutex.Lock()
	defer kp.metrics.mutex.Unlock()
	kp.metrics.BatchesPerSecond++
}

func (kp *KafkaProducer) incrementSendErrors() {
	kp.metrics.mutex.Lock()
	defer kp.metrics.mutex.Unlock()
	kp.metrics.SendErrors++
}

func (kp *KafkaProducer) incrementDroppedMessages() {
	kp.metrics.mutex.Lock()
	defer kp.metrics.mutex.Unlock()
	kp.metrics.DroppedMessages++
}

func (kp *KafkaProducer) updateQueueDepth(depth int64) {
	kp.metrics.mutex.Lock()
	defer kp.metrics.mutex.Unlock()
	kp.metrics.QueueDepth = depth
}
