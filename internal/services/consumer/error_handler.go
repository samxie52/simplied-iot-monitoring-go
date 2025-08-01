package consumer

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
)

// DefaultErrorHandler 默认错误处理器实现
type DefaultErrorHandler struct {
	// 统计信息
	totalErrors    int64
	retryableErrors int64
	fatalErrors    int64

	// 配置
	config ErrorHandlerConfig

	// 重试策略
	retryPolicy RetryPolicy
	
	// 错误分类器
	classifier ErrorClassifier
	
	// 死信队列
	dlq DeadLetterQueue
}

// ErrorHandlerConfig 错误处理器配置
type ErrorHandlerConfig struct {
	MaxRetries        int
	RetryDelay        time.Duration
	BackoffMultiplier float64
	MaxRetryDelay     time.Duration
	EnableDLQ         bool
	DLQTopic          string
	LogErrors         bool
	AlertOnErrors     bool
}

// RetryPolicy 重试策略接口
type RetryPolicy interface {
	ShouldRetry(ctx context.Context, err error, attempt int) bool
	GetRetryDelay(attempt int) time.Duration
}

// ErrorClassifier 错误分类器接口
type ErrorClassifier interface {
	ClassifyError(err error) ErrorType
	IsRetryable(err error) bool
	IsFatal(err error) bool
}

// DeadLetterQueue 死信队列接口
type DeadLetterQueue interface {
	SendToDLQ(ctx context.Context, message *sarama.ConsumerMessage, err error) error
	GetDLQStats() DLQStats
}

// ErrorType 错误类型
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeRetryable
	ErrorTypeFatal
	ErrorTypeTemporary
	ErrorTypePermanent
	ErrorTypeTimeout
	ErrorTypeValidation
	ErrorTypeSerialization
	ErrorTypeNetwork
	ErrorTypeResource
)

// DLQStats 死信队列统计
type DLQStats struct {
	MessagesSent int64
	SendErrors   int64
	QueueDepth   int64
}

// NewDefaultErrorHandler 创建默认错误处理器
func NewDefaultErrorHandler(config ErrorHandlerConfig) *DefaultErrorHandler {
	return &DefaultErrorHandler{
		config:      config,
		retryPolicy: NewExponentialBackoffRetryPolicy(config),
		classifier:  NewDefaultErrorClassifier(),
	}
}

// HandleError 处理错误
func (deh *DefaultErrorHandler) HandleError(ctx context.Context, err error, message *sarama.ConsumerMessage) bool {
	atomic.AddInt64(&deh.totalErrors, 1)

	// 记录错误日志
	if deh.config.LogErrors {
		log.Printf("处理消息错误: Topic=%s, Partition=%d, Offset=%d, Error=%v",
			message.Topic, message.Partition, message.Offset, err)
	}

	// 分类错误
	deh.classifier.ClassifyError(err)
	
	// 判断是否可重试
	if !deh.classifier.IsRetryable(err) {
		atomic.AddInt64(&deh.fatalErrors, 1)
		log.Printf("不可重试错误: %v", err)
		
		// 发送到死信队列
		if deh.config.EnableDLQ && deh.dlq != nil {
			if dlqErr := deh.dlq.SendToDLQ(ctx, message, err); dlqErr != nil {
				log.Printf("发送到死信队列失败: %v", dlqErr)
			}
		}
		
		return false // 不重试
	}

	atomic.AddInt64(&deh.retryableErrors, 1)
	
	// 检查重试策略
	retryAttempt := deh.getRetryAttempt(message)
	if !deh.retryPolicy.ShouldRetry(ctx, err, retryAttempt) {
		log.Printf("超过最大重试次数: %d", retryAttempt)
		
		// 发送到死信队列
		if deh.config.EnableDLQ && deh.dlq != nil {
			if dlqErr := deh.dlq.SendToDLQ(ctx, message, err); dlqErr != nil {
				log.Printf("发送到死信队列失败: %v", dlqErr)
			}
		}
		
		return false
	}

	// 计算重试延迟
	retryDelay := deh.retryPolicy.GetRetryDelay(retryAttempt)
	log.Printf("将在 %v 后重试 (第%d次)", retryDelay, retryAttempt+1)
	
	// 等待重试延迟
	select {
	case <-time.After(retryDelay):
		return true // 重试
	case <-ctx.Done():
		return false // 上下文取消，不重试
	}
}

// getRetryAttempt 获取重试次数
func (deh *DefaultErrorHandler) getRetryAttempt(message *sarama.ConsumerMessage) int {
	// 这里可以从消息头中获取重试次数
	// 简化实现，返回0
	return 0
}

// GetErrorStats 获取错误统计
func (deh *DefaultErrorHandler) GetErrorStats() ErrorStats {
	return ErrorStats{
		TotalErrors:     atomic.LoadInt64(&deh.totalErrors),
		RetryableErrors: atomic.LoadInt64(&deh.retryableErrors),
		FatalErrors:     atomic.LoadInt64(&deh.fatalErrors),
	}
}

// SetRetryPolicy 设置重试策略
func (deh *DefaultErrorHandler) SetRetryPolicy(policy RetryPolicy) {
	deh.retryPolicy = policy
}

// SetErrorClassifier 设置错误分类器
func (deh *DefaultErrorHandler) SetErrorClassifier(classifier ErrorClassifier) {
	deh.classifier = classifier
}

// SetDeadLetterQueue 设置死信队列
func (deh *DefaultErrorHandler) SetDeadLetterQueue(dlq DeadLetterQueue) {
	deh.dlq = dlq
}

// ExponentialBackoffRetryPolicy 指数退避重试策略
type ExponentialBackoffRetryPolicy struct {
	maxRetries        int
	baseDelay         time.Duration
	backoffMultiplier float64
	maxDelay          time.Duration
}

// NewExponentialBackoffRetryPolicy 创建指数退避重试策略
func NewExponentialBackoffRetryPolicy(config ErrorHandlerConfig) *ExponentialBackoffRetryPolicy {
	return &ExponentialBackoffRetryPolicy{
		maxRetries:        config.MaxRetries,
		baseDelay:         config.RetryDelay,
		backoffMultiplier: config.BackoffMultiplier,
		maxDelay:          config.MaxRetryDelay,
	}
}

// ShouldRetry 判断是否应该重试
func (ebrp *ExponentialBackoffRetryPolicy) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	if attempt >= ebrp.maxRetries {
		return false
	}
	
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}

// GetRetryDelay 获取重试延迟
func (ebrp *ExponentialBackoffRetryPolicy) GetRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return ebrp.baseDelay
	}
	
	// 计算指数退避延迟
	delay := float64(ebrp.baseDelay)
	for i := 0; i < attempt; i++ {
		delay *= ebrp.backoffMultiplier
	}
	
	resultDelay := time.Duration(delay)
	if resultDelay > ebrp.maxDelay {
		resultDelay = ebrp.maxDelay
	}
	
	return resultDelay
}

// DefaultErrorClassifier 默认错误分类器
type DefaultErrorClassifier struct{}

// NewDefaultErrorClassifier 创建默认错误分类器
func NewDefaultErrorClassifier() *DefaultErrorClassifier {
	return &DefaultErrorClassifier{}
}

// ClassifyError 分类错误
func (dec *DefaultErrorClassifier) ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}
	
	errStr := err.Error()
	
	// 网络错误
	if containsAny(errStr, []string{"connection", "network", "timeout", "dial"}) {
		return ErrorTypeNetwork
	}
	
	// 序列化错误
	if containsAny(errStr, []string{"json", "unmarshal", "marshal", "decode", "encode"}) {
		return ErrorTypeSerialization
	}
	
	// 验证错误
	if containsAny(errStr, []string{"validation", "invalid", "required", "format"}) {
		return ErrorTypeValidation
	}
	
	// 资源错误
	if containsAny(errStr, []string{"resource", "memory", "disk", "cpu"}) {
		return ErrorTypeResource
	}
	
	return ErrorTypeUnknown
}

// IsRetryable 判断错误是否可重试
func (dec *DefaultErrorClassifier) IsRetryable(err error) bool {
	errorType := dec.ClassifyError(err)
	
	switch errorType {
	case ErrorTypeNetwork, ErrorTypeTimeout, ErrorTypeResource:
		return true
	case ErrorTypeValidation, ErrorTypeSerialization:
		return false
	default:
		return true // 默认可重试
	}
}

// IsFatal 判断错误是否致命
func (dec *DefaultErrorClassifier) IsFatal(err error) bool {
	return !dec.IsRetryable(err)
}

// containsAny 检查字符串是否包含任意一个子字符串
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// SimpleDLQ 简单死信队列实现
type SimpleDLQ struct {
	producer   sarama.AsyncProducer
	topic      string
	messagesSent int64
	sendErrors   int64
}

// NewSimpleDLQ 创建简单死信队列
func NewSimpleDLQ(producer sarama.AsyncProducer, topic string) *SimpleDLQ {
	return &SimpleDLQ{
		producer: producer,
		topic:    topic,
	}
}

// SendToDLQ 发送消息到死信队列
func (sdlq *SimpleDLQ) SendToDLQ(ctx context.Context, message *sarama.ConsumerMessage, err error) error {
	if sdlq.producer == nil {
		return fmt.Errorf("DLQ producer not configured")
	}
	
	// 创建死信消息
	dlqMessage := &sarama.ProducerMessage{
		Topic: sdlq.topic,
		Key:   sarama.ByteEncoder(message.Key),
		Value: sarama.ByteEncoder(message.Value),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("original_topic"),
				Value: []byte(message.Topic),
			},
			{
				Key:   []byte("original_partition"),
				Value: []byte(fmt.Sprintf("%d", message.Partition)),
			},
			{
				Key:   []byte("original_offset"),
				Value: []byte(fmt.Sprintf("%d", message.Offset)),
			},
			{
				Key:   []byte("error"),
				Value: []byte(err.Error()),
			},
			{
				Key:   []byte("timestamp"),
				Value: []byte(fmt.Sprintf("%d", time.Now().Unix())),
			},
		},
	}
	
	// 复制原始消息头
	for _, header := range message.Headers {
		dlqMessage.Headers = append(dlqMessage.Headers, sarama.RecordHeader{
			Key:   append([]byte("original_"), header.Key...),
			Value: header.Value,
		})
	}
	
	// 发送到死信队列
	select {
	case sdlq.producer.Input() <- dlqMessage:
		atomic.AddInt64(&sdlq.messagesSent, 1)
		log.Printf("消息已发送到死信队列: %s", sdlq.topic)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		atomic.AddInt64(&sdlq.sendErrors, 1)
		return fmt.Errorf("DLQ producer channel full")
	}
}

// GetDLQStats 获取死信队列统计
func (sdlq *SimpleDLQ) GetDLQStats() DLQStats {
	return DLQStats{
		MessagesSent: atomic.LoadInt64(&sdlq.messagesSent),
		SendErrors:   atomic.LoadInt64(&sdlq.sendErrors),
		QueueDepth:   0, // 简化实现，不统计队列深度
	}
}

// CircuitBreakerErrorHandler 熔断器错误处理器
type CircuitBreakerErrorHandler struct {
	*DefaultErrorHandler
	
	// 熔断器状态
	state         CircuitBreakerState
	failureCount  int64
	lastFailTime  time.Time
	successCount  int64
	
	// 熔断器配置
	failureThreshold int64
	timeout          time.Duration
	successThreshold int64
}

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// NewCircuitBreakerErrorHandler 创建熔断器错误处理器
func NewCircuitBreakerErrorHandler(config ErrorHandlerConfig, failureThreshold int64, timeout time.Duration) *CircuitBreakerErrorHandler {
	return &CircuitBreakerErrorHandler{
		DefaultErrorHandler: NewDefaultErrorHandler(config),
		state:               CircuitBreakerClosed,
		failureThreshold:    failureThreshold,
		timeout:             timeout,
		successThreshold:    5, // 半开状态需要5次成功才能关闭
	}
}

// HandleError 带熔断器的错误处理
func (cbeh *CircuitBreakerErrorHandler) HandleError(ctx context.Context, err error, message *sarama.ConsumerMessage) bool {
	// 检查熔断器状态
	if cbeh.state == CircuitBreakerOpen {
		if time.Since(cbeh.lastFailTime) > cbeh.timeout {
			cbeh.state = CircuitBreakerHalfOpen
			cbeh.successCount = 0
			log.Println("熔断器进入半开状态")
		} else {
			log.Println("熔断器开启，拒绝处理")
			return false
		}
	}
	
	// 调用默认错误处理
	shouldRetry := cbeh.DefaultErrorHandler.HandleError(ctx, err, message)
	
	// 更新熔断器状态
	if shouldRetry {
		atomic.AddInt64(&cbeh.failureCount, 1)
		cbeh.lastFailTime = time.Now()
		
		if cbeh.state == CircuitBreakerHalfOpen {
			cbeh.state = CircuitBreakerOpen
			log.Println("熔断器重新开启")
		} else if atomic.LoadInt64(&cbeh.failureCount) >= cbeh.failureThreshold {
			cbeh.state = CircuitBreakerOpen
			log.Printf("熔断器开启，失败次数: %d", cbeh.failureCount)
		}
	} else {
		// 处理成功
		if cbeh.state == CircuitBreakerHalfOpen {
			atomic.AddInt64(&cbeh.successCount, 1)
			if atomic.LoadInt64(&cbeh.successCount) >= cbeh.successThreshold {
				cbeh.state = CircuitBreakerClosed
				atomic.StoreInt64(&cbeh.failureCount, 0)
				log.Println("熔断器关闭")
			}
		}
	}
	
	return shouldRetry
}

// GetCircuitBreakerState 获取熔断器状态
func (cbeh *CircuitBreakerErrorHandler) GetCircuitBreakerState() CircuitBreakerState {
	return cbeh.state
}

// GetFailureCount 获取失败次数
func (cbeh *CircuitBreakerErrorHandler) GetFailureCount() int64 {
	return atomic.LoadInt64(&cbeh.failureCount)
}
