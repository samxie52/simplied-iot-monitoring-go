package producer

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultErrorHandler 默认错误处理器
type DefaultErrorHandler struct {
	retryableErrors map[string]bool
	maxRetries      int
	retryCount      map[string]int64
	mutex           sync.RWMutex
}

// NewDefaultErrorHandler 创建默认错误处理器
func NewDefaultErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{
		retryableErrors: map[string]bool{
			"kafka: client has run out of available brokers":     true,
			"kafka: broker received message that exceeds":        false,
			"kafka: message was too large":                       false,
			"kafka: invalid topic":                               false,
			"kafka: leader not available":                        true,
			"kafka: not enough replicas":                         true,
			"kafka: request timed out":                           true,
			"kafka: network exception":                           true,
		},
		maxRetries: 3,
		retryCount: make(map[string]int64),
	}
}

// HandleError 处理错误
func (eh *DefaultErrorHandler) HandleError(err error, message *ProducerMessage) bool {
	eh.mutex.Lock()
	defer eh.mutex.Unlock()
	
	errorMsg := err.Error()
	
	// 检查是否是可重试的错误
	for pattern, retryable := range eh.retryableErrors {
		if contains(errorMsg, pattern) {
			if retryable {
				// 检查重试次数
				key := fmt.Sprintf("%s_%s", message.Key, pattern)
				count := eh.retryCount[key]
				if count < int64(eh.maxRetries) {
					eh.retryCount[key] = count + 1
					return true // 可以重试
				}
			}
			return false // 不可重试或超过重试次数
		}
	}
	
	// 未知错误，默认重试一次
	key := fmt.Sprintf("%s_unknown", message.Key)
	count := eh.retryCount[key]
	if count < 1 {
		eh.retryCount[key] = count + 1
		return true
	}
	
	return false
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 indexOf(s, substr) >= 0)))
}

// indexOf 查找子串位置
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// RetryManager 重试管理器
type RetryManager struct {
	maxRetries   int
	baseDelay    time.Duration
	maxDelay     time.Duration
	backoffType  BackoffType
	retryMetrics *RetryMetrics
}

// BackoffType 退避类型
type BackoffType int

const (
	FixedBackoff BackoffType = iota
	LinearBackoff
	ExponentialBackoff
	JitteredBackoff
)

// RetryMetrics 重试指标
type RetryMetrics struct {
	TotalRetries    int64
	SuccessRetries  int64
	FailedRetries   int64
	AvgRetryDelay   time.Duration
	MaxRetryDelay   time.Duration
	mutex           sync.RWMutex
}

// NewRetryManager 创建重试管理器
func NewRetryManager(maxRetries int, baseDelay time.Duration) *RetryManager {
	return &RetryManager{
		maxRetries:   maxRetries,
		baseDelay:    baseDelay,
		maxDelay:     baseDelay * 60, // 最大延迟为基础延迟的60倍
		backoffType:  ExponentialBackoff,
		retryMetrics: &RetryMetrics{},
	}
}

// Retry 执行重试
func (rm *RetryManager) Retry(operation func() error) error {
	var lastErr error
	
	for attempt := 0; attempt <= rm.maxRetries; attempt++ {
		if attempt > 0 {
			// 计算延迟
			delay := rm.calculateDelay(attempt)
			
			// 更新指标
			atomic.AddInt64(&rm.retryMetrics.TotalRetries, 1)
			rm.updateDelayMetrics(delay)
			
			// 等待
			time.Sleep(delay)
		}
		
		// 执行操作
		err := operation()
		if err == nil {
			if attempt > 0 {
				atomic.AddInt64(&rm.retryMetrics.SuccessRetries, 1)
			}
			return nil // 成功
		}
		
		lastErr = err
	}
	
	// 所有重试都失败
	atomic.AddInt64(&rm.retryMetrics.FailedRetries, 1)
	return fmt.Errorf("operation failed after %d retries, last error: %w", rm.maxRetries, lastErr)
}

// calculateDelay 计算延迟时间
func (rm *RetryManager) calculateDelay(attempt int) time.Duration {
	var delay time.Duration
	
	switch rm.backoffType {
	case FixedBackoff:
		delay = rm.baseDelay
		
	case LinearBackoff:
		delay = time.Duration(attempt) * rm.baseDelay
		
	case ExponentialBackoff:
		delay = rm.baseDelay * time.Duration(1<<uint(attempt-1))
		
	case JitteredBackoff:
		exponentialDelay := rm.baseDelay * time.Duration(1<<uint(attempt-1))
		jitter := time.Duration(float64(exponentialDelay) * 0.1 * (2.0*rand.Float64() - 1.0))
		delay = exponentialDelay + jitter
		
	default:
		delay = rm.baseDelay
	}
	
	// 限制最大延迟
	if delay > rm.maxDelay {
		delay = rm.maxDelay
	}
	
	return delay
}

// updateDelayMetrics 更新延迟指标
func (rm *RetryManager) updateDelayMetrics(delay time.Duration) {
	rm.retryMetrics.mutex.Lock()
	defer rm.retryMetrics.mutex.Unlock()
	
	// 更新平均延迟
	totalRetries := atomic.LoadInt64(&rm.retryMetrics.TotalRetries)
	if totalRetries == 1 {
		rm.retryMetrics.AvgRetryDelay = delay
	} else {
		currentAvg := rm.retryMetrics.AvgRetryDelay
		rm.retryMetrics.AvgRetryDelay = time.Duration(
			(int64(currentAvg)*(totalRetries-1) + int64(delay)) / totalRetries,
		)
	}
	
	// 更新最大延迟
	if delay > rm.retryMetrics.MaxRetryDelay {
		rm.retryMetrics.MaxRetryDelay = delay
	}
}

// GetMetrics 获取重试指标
func (rm *RetryManager) GetMetrics() RetryMetrics {
	rm.retryMetrics.mutex.RLock()
	defer rm.retryMetrics.mutex.RUnlock()
	
	return RetryMetrics{
		TotalRetries:   atomic.LoadInt64(&rm.retryMetrics.TotalRetries),
		SuccessRetries: atomic.LoadInt64(&rm.retryMetrics.SuccessRetries),
		FailedRetries:  atomic.LoadInt64(&rm.retryMetrics.FailedRetries),
		AvgRetryDelay:  rm.retryMetrics.AvgRetryDelay,
		MaxRetryDelay:  rm.retryMetrics.MaxRetryDelay,
	}
}

// SetBackoffType 设置退避类型
func (rm *RetryManager) SetBackoffType(backoffType BackoffType) {
	rm.backoffType = backoffType
}

// SetMaxDelay 设置最大延迟
func (rm *RetryManager) SetMaxDelay(maxDelay time.Duration) {
	rm.maxDelay = maxDelay
}

// Reset 重置指标
func (rm *RetryManager) Reset() {
	atomic.StoreInt64(&rm.retryMetrics.TotalRetries, 0)
	atomic.StoreInt64(&rm.retryMetrics.SuccessRetries, 0)
	atomic.StoreInt64(&rm.retryMetrics.FailedRetries, 0)
	
	rm.retryMetrics.mutex.Lock()
	rm.retryMetrics.AvgRetryDelay = 0
	rm.retryMetrics.MaxRetryDelay = 0
	rm.retryMetrics.mutex.Unlock()
}
