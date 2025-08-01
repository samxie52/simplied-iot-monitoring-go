package producer

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// BatchWorkerPool 批处理工作池
type BatchWorkerPool struct {
	workerCount    int
	taskQueue      chan BatchTaskInterface
	workers        []*BatchWorker
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	isRunning      bool
	mutex          sync.RWMutex
	
	// 性能指标
	metrics        *WorkerPoolMetrics
	
	// 自适应配置
	adaptive       bool
	minWorkers     int
	maxWorkers     int
	scaleThreshold float64
	scaleInterval  time.Duration
	lastScaleTime  time.Time
}

// BatchTaskInterface 批处理任务接口
type BatchTaskInterface interface {
	Execute() error
}

// BatchWorker 批处理工作者
type BatchWorker struct {
	id          int
	pool        *BatchWorkerPool
	taskCount   int64
	errorCount  int64
	totalTime   time.Duration
	isActive    bool
	mutex       sync.RWMutex
}

// WorkerPoolMetrics 工作池指标
type WorkerPoolMetrics struct {
	// 基础指标
	ActiveWorkers     int64
	TotalTasks        int64
	CompletedTasks    int64
	FailedTasks       int64
	QueuedTasks       int64
	
	// 性能指标
	TasksPerSecond    float64
	AvgTaskDuration   time.Duration
	MaxTaskDuration   time.Duration
	MinTaskDuration   time.Duration
	
	// 资源使用
	MemoryUsage       int64
	CPUUsage          float64
	GoroutineCount    int64
	
	// 队列指标
	QueueUtilization  float64
	MaxQueueSize      int64
	
	mutex             sync.RWMutex
}

// NewBatchWorkerPool 创建批处理工作池
func NewBatchWorkerPool(workerCount, queueSize int) *BatchWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &BatchWorkerPool{
		workerCount:    workerCount,
		taskQueue:      make(chan BatchTaskInterface, queueSize),
		workers:        make([]*BatchWorker, 0, workerCount),
		ctx:            ctx,
		cancel:         cancel,
		isRunning:      false,
		metrics:        NewWorkerPoolMetrics(),
		adaptive:       true,
		minWorkers:     1,
		maxWorkers:     runtime.NumCPU() * 4,
		scaleThreshold: 0.8, // 80%队列使用率触发扩容
		scaleInterval:  30 * time.Second,
		lastScaleTime:  time.Now(),
	}
	
	return pool
}

// Start 启动工作池
func (pool *BatchWorkerPool) Start() error {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	
	if pool.isRunning {
		return fmt.Errorf("worker pool is already running")
	}
	
	// 创建初始工作者
	for i := 0; i < pool.workerCount; i++ {
		worker := &BatchWorker{
			id:        i,
			pool:      pool,
			taskCount: 0,
			errorCount: 0,
			isActive:  true,
		}
		pool.workers = append(pool.workers, worker)
		
		pool.wg.Add(1)
		go worker.run()
	}
	
	atomic.StoreInt64(&pool.metrics.ActiveWorkers, int64(pool.workerCount))
	
	// 启动监控协程
	if pool.adaptive {
		pool.wg.Add(2)
		go pool.metricsCollector()
		go pool.autoScaler()
	}
	
	pool.isRunning = true
	return nil
}

// Submit 提交任务
func (pool *BatchWorkerPool) Submit(task BatchTaskInterface) error {
	if !pool.isRunning {
		return fmt.Errorf("worker pool is not running")
	}
	
	select {
	case pool.taskQueue <- task:
		atomic.AddInt64(&pool.metrics.QueuedTasks, 1)
		atomic.AddInt64(&pool.metrics.TotalTasks, 1)
		return nil
	case <-pool.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	default:
		return fmt.Errorf("task queue is full")
	}
}

// run 工作者运行循环
func (worker *BatchWorker) run() {
	defer worker.pool.wg.Done()
	
	for {
		select {
		case task := <-worker.pool.taskQueue:
			atomic.AddInt64(&worker.pool.metrics.QueuedTasks, -1)
			worker.executeTask(task)
			
		case <-worker.pool.ctx.Done():
			return
		}
	}
}

// executeTask 执行任务
func (worker *BatchWorker) executeTask(task BatchTaskInterface) {
	startTime := time.Now()
	
	// 执行任务
	err := task.Execute()
	
	duration := time.Since(startTime)
	
	// 更新工作者统计
	worker.mutex.Lock()
	worker.taskCount++
	worker.totalTime += duration
	if err != nil {
		worker.errorCount++
		atomic.AddInt64(&worker.pool.metrics.FailedTasks, 1)
	} else {
		atomic.AddInt64(&worker.pool.metrics.CompletedTasks, 1)
	}
	worker.mutex.Unlock()
	
	// 更新池级别指标
	worker.pool.updateTaskMetrics(duration, err != nil)
}

// updateTaskMetrics 更新任务指标
func (pool *BatchWorkerPool) updateTaskMetrics(duration time.Duration, failed bool) {
	pool.metrics.mutex.Lock()
	defer pool.metrics.mutex.Unlock()
	
	// 更新任务持续时间统计
	if pool.metrics.MinTaskDuration == 0 || duration < pool.metrics.MinTaskDuration {
		pool.metrics.MinTaskDuration = duration
	}
	if duration > pool.metrics.MaxTaskDuration {
		pool.metrics.MaxTaskDuration = duration
	}
	
	// 更新平均持续时间
	completedTasks := atomic.LoadInt64(&pool.metrics.CompletedTasks)
	if completedTasks > 0 {
		currentAvg := pool.metrics.AvgTaskDuration
		pool.metrics.AvgTaskDuration = time.Duration(
			(int64(currentAvg)*(completedTasks-1) + int64(duration)) / completedTasks,
		)
	}
}

// metricsCollector 指标收集器
func (pool *BatchWorkerPool) metricsCollector() {
	defer pool.wg.Done()
	
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	var lastCompleted int64
	
	for {
		select {
		case <-ticker.C:
			// 计算每秒任务数
			currentCompleted := atomic.LoadInt64(&pool.metrics.CompletedTasks)
			pool.metrics.mutex.Lock()
			pool.metrics.TasksPerSecond = float64(currentCompleted - lastCompleted)
			pool.metrics.mutex.Unlock()
			lastCompleted = currentCompleted
			
			// 更新队列使用率
			queuedTasks := atomic.LoadInt64(&pool.metrics.QueuedTasks)
			maxQueueSize := int64(cap(pool.taskQueue))
			pool.metrics.mutex.Lock()
			pool.metrics.QueueUtilization = float64(queuedTasks) / float64(maxQueueSize)
			pool.metrics.MaxQueueSize = maxQueueSize
			pool.metrics.mutex.Unlock()
			
			// 更新资源使用情况
			pool.updateResourceMetrics()
			
		case <-pool.ctx.Done():
			return
		}
	}
}

// updateResourceMetrics 更新资源指标
func (pool *BatchWorkerPool) updateResourceMetrics() {
	// 获取内存使用情况
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	pool.metrics.mutex.Lock()
	pool.metrics.MemoryUsage = int64(m.Alloc)
	pool.metrics.GoroutineCount = int64(runtime.NumGoroutine())
	pool.metrics.mutex.Unlock()
}

// autoScaler 自动扩缩容
func (pool *BatchWorkerPool) autoScaler() {
	defer pool.wg.Done()
	
	ticker := time.NewTicker(pool.scaleInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			pool.checkAndScale()
			
		case <-pool.ctx.Done():
			return
		}
	}
}

// checkAndScale 检查并执行扩缩容
func (pool *BatchWorkerPool) checkAndScale() {
	pool.metrics.mutex.RLock()
	queueUtilization := pool.metrics.QueueUtilization
	tasksPerSecond := pool.metrics.TasksPerSecond
	pool.metrics.mutex.RUnlock()
	
	currentWorkers := int(atomic.LoadInt64(&pool.metrics.ActiveWorkers))
	
	// 扩容条件：队列使用率高且任务处理速度跟不上
	if queueUtilization > pool.scaleThreshold && tasksPerSecond > 0 && currentWorkers < pool.maxWorkers {
		pool.scaleUp()
	}
	
	// 缩容条件：队列使用率低且工作者数量超过最小值
	if queueUtilization < 0.2 && currentWorkers > pool.minWorkers && time.Since(pool.lastScaleTime) > pool.scaleInterval*2 {
		pool.scaleDown()
	}
}

// scaleUp 扩容
func (pool *BatchWorkerPool) scaleUp() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	
	if !pool.isRunning {
		return
	}
	
	currentWorkers := len(pool.workers)
	if currentWorkers >= pool.maxWorkers {
		return
	}
	
	// 添加新工作者
	newWorkerCount := minInt(pool.maxWorkers-currentWorkers, maxInt(1, currentWorkers/4)) // 每次增加25%或至少1个
	
	for i := 0; i < newWorkerCount; i++ {
		worker := &BatchWorker{
			id:        currentWorkers + i,
			pool:      pool,
			taskCount: 0,
			errorCount: 0,
			isActive:  true,
		}
		pool.workers = append(pool.workers, worker)
		
		pool.wg.Add(1)
		go worker.run()
	}
	
	atomic.AddInt64(&pool.metrics.ActiveWorkers, int64(newWorkerCount))
	pool.lastScaleTime = time.Now()
	
	fmt.Printf("Scaled up worker pool: added %d workers, total: %d\n", newWorkerCount, len(pool.workers))
}

// scaleDown 缩容
func (pool *BatchWorkerPool) scaleDown() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	
	if !pool.isRunning {
		return
	}
	
	currentWorkers := len(pool.workers)
	if currentWorkers <= pool.minWorkers {
		return
	}
	
	// 减少工作者数量
	removeCount := maxInt(1, currentWorkers/8) // 每次减少12.5%或至少1个
	if currentWorkers-removeCount < pool.minWorkers {
		removeCount = currentWorkers - pool.minWorkers
	}
	
	// 标记工作者为非活跃（它们会在下次循环时退出）
	for i := currentWorkers - removeCount; i < currentWorkers; i++ {
		pool.workers[i].mutex.Lock()
		pool.workers[i].isActive = false
		pool.workers[i].mutex.Unlock()
	}
	
	// 从切片中移除
	pool.workers = pool.workers[:currentWorkers-removeCount]
	
	atomic.AddInt64(&pool.metrics.ActiveWorkers, -int64(removeCount))
	pool.lastScaleTime = time.Now()
	
	fmt.Printf("Scaled down worker pool: removed %d workers, total: %d\n", removeCount, len(pool.workers))
}

// Stop 停止工作池
func (pool *BatchWorkerPool) Stop() error {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	
	if !pool.isRunning {
		return nil
	}
	
	pool.cancel()
	pool.wg.Wait()
	
	pool.isRunning = false
	return nil
}

// GetMetrics 获取指标
func (pool *BatchWorkerPool) GetMetrics() *WorkerPoolMetrics {
	pool.metrics.mutex.RLock()
	defer pool.metrics.mutex.RUnlock()
	
	return &WorkerPoolMetrics{
		ActiveWorkers:     atomic.LoadInt64(&pool.metrics.ActiveWorkers),
		TotalTasks:        atomic.LoadInt64(&pool.metrics.TotalTasks),
		CompletedTasks:    atomic.LoadInt64(&pool.metrics.CompletedTasks),
		FailedTasks:       atomic.LoadInt64(&pool.metrics.FailedTasks),
		QueuedTasks:       atomic.LoadInt64(&pool.metrics.QueuedTasks),
		TasksPerSecond:    pool.metrics.TasksPerSecond,
		AvgTaskDuration:   pool.metrics.AvgTaskDuration,
		MaxTaskDuration:   pool.metrics.MaxTaskDuration,
		MinTaskDuration:   pool.metrics.MinTaskDuration,
		MemoryUsage:       pool.metrics.MemoryUsage,
		CPUUsage:          pool.metrics.CPUUsage,
		GoroutineCount:    pool.metrics.GoroutineCount,
		QueueUtilization:  pool.metrics.QueueUtilization,
		MaxQueueSize:      pool.metrics.MaxQueueSize,
	}
}

// GetWorkerStats 获取工作者统计
func (pool *BatchWorkerPool) GetWorkerStats() []WorkerStats {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()
	
	stats := make([]WorkerStats, len(pool.workers))
	for i, worker := range pool.workers {
		worker.mutex.RLock()
		stats[i] = WorkerStats{
			ID:           worker.id,
			TaskCount:    worker.taskCount,
			ErrorCount:   worker.errorCount,
			TotalTime:    worker.totalTime,
			IsActive:     worker.isActive,
			AvgDuration:  time.Duration(int64(worker.totalTime) / maxInt64(1, worker.taskCount)),
		}
		worker.mutex.RUnlock()
	}
	
	return stats
}

// WorkerStats 工作者统计
type WorkerStats struct {
	ID          int           `json:"id"`
	TaskCount   int64         `json:"task_count"`
	ErrorCount  int64         `json:"error_count"`
	TotalTime   time.Duration `json:"total_time"`
	IsActive    bool          `json:"is_active"`
	AvgDuration time.Duration `json:"avg_duration"`
}

// NewWorkerPoolMetrics 创建工作池指标
func NewWorkerPoolMetrics() *WorkerPoolMetrics {
	return &WorkerPoolMetrics{}
}

// IsRunning 检查是否运行中
func (pool *BatchWorkerPool) IsRunning() bool {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()
	return pool.isRunning
}

// SetAdaptive 设置自适应模式
func (pool *BatchWorkerPool) SetAdaptive(enabled bool) {
	pool.adaptive = enabled
}

// SetScaleParams 设置扩缩容参数
func (pool *BatchWorkerPool) SetScaleParams(minWorkers, maxWorkers int, threshold float64, interval time.Duration) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	
	pool.minWorkers = minWorkers
	pool.maxWorkers = maxWorkers
	pool.scaleThreshold = threshold
	pool.scaleInterval = interval
}

// 辅助函数
func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
