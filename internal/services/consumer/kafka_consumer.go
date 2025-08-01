package consumer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/models"
)

// KafkaConsumer Kafka消费者实现
type KafkaConsumer struct {
	// 基础配置
	config       *config.KafkaConsumer
	brokers      []string
	topics       []string
	groupID      string
	saramaConfig *sarama.Config

	// Sarama组件
	consumerGroup sarama.ConsumerGroup
	handler       *ConsumerGroupHandler

	// 控制和状态
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	isRunning   bool
	mutex       sync.RWMutex

	// 指标和监控
	metrics *ConsumerMetrics

	// 消息处理
	messageProcessor MessageProcessor
	errorHandler     ErrorHandler
}

// ConsumerMetrics 消费者指标
type ConsumerMetrics struct {
	MessagesConsumed     int64
	MessagesProcessed    int64
	ProcessingErrors     int64
	RebalanceCount       int64
	PartitionCount       int64
	LagTotal             int64
	ProcessingLatencyMs  int64
	ThroughputPerSecond  int64
	ActivePartitions     map[string][]int32
	LastMessageTimestamp int64
	mutex                sync.RWMutex
}

// MessageProcessor 消息处理器接口
type MessageProcessor interface {
	ProcessMessage(ctx context.Context, message *models.KafkaMessage) error
	GetStats() ProcessorStats
}

// ProcessorStats 处理器统计信息
type ProcessorStats struct {
	ProcessedCount int64
	ErrorCount     int64
	AvgLatencyMs   float64
}

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	HandleError(ctx context.Context, err error, message *sarama.ConsumerMessage) bool
	GetErrorStats() ErrorStats
}

// ErrorStats 错误统计信息
type ErrorStats struct {
	TotalErrors    int64
	RetryableErrors int64
	FatalErrors    int64
}

// NewKafkaConsumer 创建新的Kafka消费者
func NewKafkaConsumer(
	brokers []string,
	topics []string,
	consumerConfig *config.KafkaConsumer,
	processor MessageProcessor,
	errorHandler ErrorHandler,
) (*KafkaConsumer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("brokers cannot be empty")
	}
	if len(topics) == 0 {
		return nil, fmt.Errorf("topics cannot be empty")
	}
	if consumerConfig == nil {
		return nil, fmt.Errorf("consumer config cannot be nil")
	}

	// 创建Sarama配置
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_6_0_0
	saramaConfig.Consumer.Group.Session.Timeout = consumerConfig.SessionTimeout
	saramaConfig.Consumer.Group.Heartbeat.Interval = consumerConfig.SessionTimeout / 3
	saramaConfig.Consumer.MaxProcessingTime = consumerConfig.SessionTimeout
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	
	// 设置自动提交
	if consumerConfig.EnableAutoCommit {
		saramaConfig.Consumer.Offsets.AutoCommit.Enable = true
		saramaConfig.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
	}

	// 设置分区分配策略
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin

	// 设置网络超时
	saramaConfig.Net.DialTimeout = 10 * time.Second
	saramaConfig.Net.ReadTimeout = 10 * time.Second
	saramaConfig.Net.WriteTimeout = 10 * time.Second

	ctx, cancel := context.WithCancel(context.Background())

	consumer := &KafkaConsumer{
		config:       consumerConfig,
		brokers:      brokers,
		topics:       topics,
		groupID:      consumerConfig.GroupID,
		saramaConfig: saramaConfig,
		ctx:          ctx,
		cancel:       cancel,
		isRunning:    false,
		metrics: &ConsumerMetrics{
			ActivePartitions: make(map[string][]int32),
		},
		messageProcessor: processor,
		errorHandler:     errorHandler,
	}

	// 创建消费者组处理器
	consumer.handler = NewConsumerGroupHandler(consumer)

	return consumer, nil
}

// Start 启动消费者
func (kc *KafkaConsumer) Start() error {
	kc.mutex.Lock()
	defer kc.mutex.Unlock()

	if kc.isRunning {
		return fmt.Errorf("consumer is already running")
	}

	log.Printf("正在启动Kafka消费者，连接到brokers: %v", kc.brokers)
	log.Printf("消费者组ID: %s, 订阅主题: %v", kc.groupID, kc.topics)

	// 创建消费者组
	consumerGroup, err := sarama.NewConsumerGroup(kc.brokers, kc.groupID, kc.saramaConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	kc.consumerGroup = consumerGroup
	kc.isRunning = true

	// 启动错误处理协程
	kc.wg.Add(1)
	go kc.handleErrors()

	// 启动消费协程
	kc.wg.Add(1)
	go kc.consume()

	log.Println("Kafka消费者启动成功")
	return nil
}

// consume 消费消息
func (kc *KafkaConsumer) consume() {
	defer kc.wg.Done()

	for {
		select {
		case <-kc.ctx.Done():
			log.Println("消费者上下文已取消，停止消费")
			return
		default:
			// 开始消费
			err := kc.consumerGroup.Consume(kc.ctx, kc.topics, kc.handler)
			if err != nil {
				log.Printf("消费者组消费错误: %v", err)
				// 如果是上下文取消，则退出
				if kc.ctx.Err() != nil {
					return
				}
				// 其他错误，等待一段时间后重试
				time.Sleep(5 * time.Second)
			}
		}
	}
}

// handleErrors 处理错误
func (kc *KafkaConsumer) handleErrors() {
	defer kc.wg.Done()

	for {
		select {
		case err := <-kc.consumerGroup.Errors():
			if err != nil {
				log.Printf("消费者组错误: %v", err)
				kc.incrementProcessingErrors()
			}
		case <-kc.ctx.Done():
			return
		}
	}
}

// Stop 停止消费者
func (kc *KafkaConsumer) Stop() error {
	kc.mutex.Lock()
	defer kc.mutex.Unlock()

	if !kc.isRunning {
		return nil
	}

	log.Println("正在停止Kafka消费者...")

	// 取消上下文
	kc.cancel()

	// 等待所有协程结束
	kc.wg.Wait()

	// 关闭消费者组
	if kc.consumerGroup != nil {
		if err := kc.consumerGroup.Close(); err != nil {
			log.Printf("关闭消费者组时发生错误: %v", err)
			return err
		}
	}

	kc.isRunning = false
	log.Println("Kafka消费者已停止")
	return nil
}

// IsRunning 检查消费者是否运行中
func (kc *KafkaConsumer) IsRunning() bool {
	kc.mutex.RLock()
	defer kc.mutex.RUnlock()
	return kc.isRunning
}

// GetMetrics 获取消费者指标
func (kc *KafkaConsumer) GetMetrics() *ConsumerMetrics {
	kc.metrics.mutex.RLock()
	defer kc.metrics.mutex.RUnlock()

	// 返回指标副本
	activePartitions := make(map[string][]int32)
	for topic, partitions := range kc.metrics.ActivePartitions {
		activePartitions[topic] = make([]int32, len(partitions))
		copy(activePartitions[topic], partitions)
	}

	return &ConsumerMetrics{
		MessagesConsumed:     kc.metrics.MessagesConsumed,
		MessagesProcessed:    kc.metrics.MessagesProcessed,
		ProcessingErrors:     kc.metrics.ProcessingErrors,
		RebalanceCount:       kc.metrics.RebalanceCount,
		PartitionCount:       kc.metrics.PartitionCount,
		LagTotal:             kc.metrics.LagTotal,
		ProcessingLatencyMs:  kc.metrics.ProcessingLatencyMs,
		ThroughputPerSecond:  kc.metrics.ThroughputPerSecond,
		ActivePartitions:     activePartitions,
		LastMessageTimestamp: kc.metrics.LastMessageTimestamp,
	}
}

// GetHealth 获取健康状态
func (kc *KafkaConsumer) GetHealth() *config.HealthStatus {
	metrics := kc.GetMetrics()
	
	// 简单的健康检查逻辑
	healthy := kc.IsRunning() && metrics.ProcessingErrors < 100
	
	return &config.HealthStatus{
		Healthy:   healthy,
		Timestamp: time.Now(),
		Services: map[string]bool{
			"kafka_consumer": kc.IsRunning(),
		},
		Metrics: map[string]string{
			"messages_consumed":    fmt.Sprintf("%d", metrics.MessagesConsumed),
			"messages_processed":   fmt.Sprintf("%d", metrics.MessagesProcessed),
			"processing_errors":    fmt.Sprintf("%d", metrics.ProcessingErrors),
			"rebalance_count":      fmt.Sprintf("%d", metrics.RebalanceCount),
			"partition_count":      fmt.Sprintf("%d", metrics.PartitionCount),
			"throughput_per_second": fmt.Sprintf("%d", metrics.ThroughputPerSecond),
		},
	}
}

// 指标更新方法
func (kc *KafkaConsumer) incrementMessagesConsumed() {
	kc.metrics.mutex.Lock()
	defer kc.metrics.mutex.Unlock()
	kc.metrics.MessagesConsumed++
	kc.metrics.LastMessageTimestamp = time.Now().Unix()
}

func (kc *KafkaConsumer) incrementMessagesProcessed() {
	kc.metrics.mutex.Lock()
	defer kc.metrics.mutex.Unlock()
	kc.metrics.MessagesProcessed++
}

func (kc *KafkaConsumer) incrementProcessingErrors() {
	kc.metrics.mutex.Lock()
	defer kc.metrics.mutex.Unlock()
	kc.metrics.ProcessingErrors++
}

func (kc *KafkaConsumer) incrementRebalanceCount() {
	kc.metrics.mutex.Lock()
	defer kc.metrics.mutex.Unlock()
	kc.metrics.RebalanceCount++
}

func (kc *KafkaConsumer) updatePartitionCount(count int64) {
	kc.metrics.mutex.Lock()
	defer kc.metrics.mutex.Unlock()
	kc.metrics.PartitionCount = count
}

func (kc *KafkaConsumer) updateActivePartitions(topic string, partitions []int32) {
	kc.metrics.mutex.Lock()
	defer kc.metrics.mutex.Unlock()
	kc.metrics.ActivePartitions[topic] = partitions
}

func (kc *KafkaConsumer) updateProcessingLatency(latencyMs int64) {
	kc.metrics.mutex.Lock()
	defer kc.metrics.mutex.Unlock()
	kc.metrics.ProcessingLatencyMs = latencyMs
}
