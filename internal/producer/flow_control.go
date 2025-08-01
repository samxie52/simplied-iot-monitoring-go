package producer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	rate     int64 // 每秒允许的请求数
	tokens   int64 // 当前令牌数
	capacity int64 // 令牌桶容量
	lastTime int64 // 上次更新时间（纳秒）
	mutex    sync.Mutex
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(rate int64) *RateLimiter {
	now := time.Now().UnixNano()
	return &RateLimiter{
		rate:     rate,
		tokens:   rate,
		capacity: rate,
		lastTime: now,
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now().UnixNano()
	elapsed := now - rl.lastTime
	
	// 添加令牌
	tokensToAdd := elapsed * rl.rate / int64(time.Second)
	rl.tokens += tokensToAdd
	if rl.tokens > rl.capacity {
		rl.tokens = rl.capacity
	}
	
	rl.lastTime = now
	
	// 检查是否有可用令牌
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	
	return false
}

// Wait 等待直到可以发送请求
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow() {
			return nil
		}
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond):
			// 短暂等待后重试
		}
	}
}

// SetRate 设置新的速率
func (rl *RateLimiter) SetRate(rate int64) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	rl.rate = rate
	rl.capacity = rate
	if rl.tokens > rl.capacity {
		rl.tokens = rl.capacity
	}
}

// GetRate 获取当前速率
func (rl *RateLimiter) GetRate() int64 {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	return rl.rate
}

// BackpressureController 背压控制器
type BackpressureController struct {
	// 阈值配置
	warningThreshold  float64 // 警告阈值（队列使用率）
	criticalThreshold float64 // 临界阈值（队列使用率）
	
	// 当前状态
	currentLoad       float64 // 当前负载
	status            BackpressureStatus
	
	// 统计信息
	totalRequests     int64
	droppedRequests   int64
	throttledRequests int64
	
	// 自适应参数
	adaptiveEnabled   bool
	loadHistory       []float64
	historySize       int
	historyIndex      int
	
	mutex             sync.RWMutex
}

// BackpressureStatus 背压状态
type BackpressureStatus int

const (
	StatusNormal BackpressureStatus = iota
	StatusWarning
	StatusCritical
)

// NewBackpressureController 创建背压控制器
func NewBackpressureController() *BackpressureController {
	return &BackpressureController{
		warningThreshold:  0.7,  // 70%
		criticalThreshold: 0.9,  // 90%
		status:           StatusNormal,
		adaptiveEnabled:  true,
		loadHistory:      make([]float64, 100), // 保留100个历史记录
		historySize:      100,
		historyIndex:     0,
	}
}

// Update 更新背压状态
func (bc *BackpressureController) Update(currentQueueSize, maxQueueSize int64) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	// 计算当前负载
	if maxQueueSize > 0 {
		bc.currentLoad = float64(currentQueueSize) / float64(maxQueueSize)
	} else {
		bc.currentLoad = 0
	}
	
	// 更新历史记录
	if bc.adaptiveEnabled {
		bc.loadHistory[bc.historyIndex] = bc.currentLoad
		bc.historyIndex = (bc.historyIndex + 1) % bc.historySize
	}
	
	// 更新状态
	oldStatus := bc.status
	if bc.currentLoad >= bc.criticalThreshold {
		bc.status = StatusCritical
	} else if bc.currentLoad >= bc.warningThreshold {
		bc.status = StatusWarning
	} else {
		bc.status = StatusNormal
	}
	
	// 自适应调整阈值
	if bc.adaptiveEnabled && oldStatus != bc.status {
		bc.adaptThresholds()
	}
}

// ShouldDrop 检查是否应该丢弃请求
func (bc *BackpressureController) ShouldDrop() bool {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	
	atomic.AddInt64(&bc.totalRequests, 1)
	
	switch bc.status {
	case StatusCritical:
		// 临界状态：丢弃50%的请求
		atomic.AddInt64(&bc.droppedRequests, 1)
		return bc.currentLoad > 0.95 || (time.Now().UnixNano()%2 == 0)
		
	case StatusWarning:
		// 警告状态：丢弃10%的请求
		if bc.currentLoad > 0.85 && (time.Now().UnixNano()%10 == 0) {
			atomic.AddInt64(&bc.droppedRequests, 1)
			return true
		}
		
	case StatusNormal:
		// 正常状态：不丢弃
		return false
	}
	
	return false
}

// ShouldThrottle 检查是否应该限流
func (bc *BackpressureController) ShouldThrottle() bool {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	
	if bc.status == StatusWarning || bc.status == StatusCritical {
		atomic.AddInt64(&bc.throttledRequests, 1)
		return true
	}
	
	return false
}

// GetThrottleDelay 获取限流延迟
func (bc *BackpressureController) GetThrottleDelay() time.Duration {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	
	switch bc.status {
	case StatusWarning:
		return time.Duration(bc.currentLoad*100) * time.Millisecond
	case StatusCritical:
		return time.Duration(bc.currentLoad*500) * time.Millisecond
	default:
		return 0
	}
}

// adaptThresholds 自适应调整阈值
func (bc *BackpressureController) adaptThresholds() {
	if !bc.adaptiveEnabled {
		return
	}
	
	// 计算历史负载的统计信息
	var sum, max float64
	count := 0
	
	for i := 0; i < bc.historySize; i++ {
		if bc.loadHistory[i] > 0 {
			sum += bc.loadHistory[i]
			if bc.loadHistory[i] > max {
				max = bc.loadHistory[i]
			}
			count++
		}
	}
	
	if count < 10 {
		return // 数据不足，不调整
	}
	
	avg := sum / float64(count)
	
	// 基于历史数据调整阈值
	if avg > 0.8 && max > 0.95 {
		// 系统经常高负载，降低阈值
		bc.warningThreshold = maxFloat64(0.5, bc.warningThreshold-0.05)
		bc.criticalThreshold = maxFloat64(0.7, bc.criticalThreshold-0.05)
	} else if avg < 0.3 && max < 0.6 {
		// 系统负载较低，提高阈值
		bc.warningThreshold = minFloat64(0.8, bc.warningThreshold+0.05)
		bc.criticalThreshold = minFloat64(0.95, bc.criticalThreshold+0.05)
	}
}

// GetStatus 获取当前状态
func (bc *BackpressureController) GetStatus() BackpressureStatus {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	return bc.status
}

// GetCurrentLoad 获取当前负载
func (bc *BackpressureController) GetCurrentLoad() float64 {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	return bc.currentLoad
}

// GetStatistics 获取统计信息
func (bc *BackpressureController) GetStatistics() BackpressureStats {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	
	total := atomic.LoadInt64(&bc.totalRequests)
	dropped := atomic.LoadInt64(&bc.droppedRequests)
	throttled := atomic.LoadInt64(&bc.throttledRequests)
	
	var dropRate, throttleRate float64
	if total > 0 {
		dropRate = float64(dropped) / float64(total)
		throttleRate = float64(throttled) / float64(total)
	}
	
	return BackpressureStats{
		CurrentLoad:       bc.currentLoad,
		Status:           bc.status,
		WarningThreshold: bc.warningThreshold,
		CriticalThreshold: bc.criticalThreshold,
		TotalRequests:    total,
		DroppedRequests:  dropped,
		ThrottledRequests: throttled,
		DropRate:         dropRate,
		ThrottleRate:     throttleRate,
	}
}

// BackpressureStats 背压统计信息
type BackpressureStats struct {
	CurrentLoad       float64            `json:"current_load"`
	Status           BackpressureStatus `json:"status"`
	WarningThreshold float64            `json:"warning_threshold"`
	CriticalThreshold float64           `json:"critical_threshold"`
	TotalRequests    int64              `json:"total_requests"`
	DroppedRequests  int64              `json:"dropped_requests"`
	ThrottledRequests int64             `json:"throttled_requests"`
	DropRate         float64            `json:"drop_rate"`
	ThrottleRate     float64            `json:"throttle_rate"`
}

// Reset 重置统计信息
func (bc *BackpressureController) Reset() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	atomic.StoreInt64(&bc.totalRequests, 0)
	atomic.StoreInt64(&bc.droppedRequests, 0)
	atomic.StoreInt64(&bc.throttledRequests, 0)
	
	bc.currentLoad = 0
	bc.status = StatusNormal
	
	for i := range bc.loadHistory {
		bc.loadHistory[i] = 0
	}
	bc.historyIndex = 0
}

// 辅助函数
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
