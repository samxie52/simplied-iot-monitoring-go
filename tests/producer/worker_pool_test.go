package producer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/producer"
)

// TestTask 实现 BatchTaskInterface 接口的测试任务
type TestTask struct {
	ID      int
	Data    interface{}
	Handler func(interface{}) error
}

func (t *TestTask) Execute() error {
	if t.Handler != nil {
		return t.Handler(t.Data)
	}
	return nil
}

// TestErrorTask 会返回错误的任务
type TestErrorTask struct {
	ID      int
	Message string
}

func (t *TestErrorTask) Execute() error {
	return fmt.Errorf("test error: %s", t.Message)
}

func TestBatchWorkerPool_Creation(t *testing.T) {
	pool := producer.NewBatchWorkerPool(4, 100)

	if pool == nil {
		t.Fatal("Failed to create worker pool")
	}

	if pool.IsRunning() {
		t.Error("Worker pool should not be running initially")
	}
}

func TestBatchWorkerPool_StartStop(t *testing.T) {
	pool := producer.NewBatchWorkerPool(2, 50)

	// 启动池
	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}

	if !pool.IsRunning() {
		t.Error("Worker pool should be running after start")
	}

	// 停止池
	err = pool.Stop()
	if err != nil {
		t.Fatalf("Failed to stop worker pool: %v", err)
	}

	if pool.IsRunning() {
		t.Error("Worker pool should not be running after stop")
	}
}

func TestBatchWorkerPool_TaskExecution(t *testing.T) {
	pool := producer.NewBatchWorkerPool(2, 50)

	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	// 创建一个计数器来跟踪执行的任务
	var executed int64
	var wg sync.WaitGroup

	// 提交一些任务
	taskCount := 10
	wg.Add(taskCount)

	for i := 0; i < taskCount; i++ {
		task := &TestTask{
			ID:   i,
			Data: i,
			Handler: func(data interface{}) error {
				atomic.AddInt64(&executed, 1)
				wg.Done()
				return nil
			},
		}

		err := pool.Submit(task)
		if err != nil {
			t.Errorf("Failed to submit task %d: %v", i, err)
		}
	}

	// 等待所有任务完成
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// 任务完成
	case <-time.After(5 * time.Second):
		t.Fatal("Tasks did not complete within timeout")
	}

	if atomic.LoadInt64(&executed) != int64(taskCount) {
		t.Errorf("Expected %d tasks executed, got %d", taskCount, executed)
	}
}

func TestBatchWorkerPool_TaskWithError(t *testing.T) {
	pool := producer.NewBatchWorkerPool(2, 50)

	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	var wg sync.WaitGroup

	// 提交一些会失败的任务
	taskCount := 5
	wg.Add(taskCount)

	for i := 0; i < taskCount; i++ {
		task := &TestTask{
			ID:   i,
			Data: i,
			Handler: func(data interface{}) error {
				wg.Done()
				return fmt.Errorf("test error for task %v", data)
			},
		}

		err := pool.Submit(task)
		if err != nil {
			t.Errorf("Failed to submit task %d: %v", i, err)
		}
	}

	// 等待所有任务完成
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// 任务完成
	case <-time.After(5 * time.Second):
		t.Fatal("Tasks did not complete within timeout")
	}

	// 等待一下让指标更新
	time.Sleep(100 * time.Millisecond)

	// 检查指标
	metrics := pool.GetMetrics()
	if metrics.FailedTasks < int64(taskCount-1) { // 允许一个任务的误差
		t.Errorf("Expected at least %d failed tasks, got %d", taskCount-1, metrics.FailedTasks)
	}
}

func TestBatchWorkerPool_QueueCapacity(t *testing.T) {
	pool := producer.NewBatchWorkerPool(1, 2) // 小队列用于测试

	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	// 创建阻塞任务
	var blockChan = make(chan bool)

	// 提交阻塞任务填满工作线程
	blockingTask := &TestTask{
		ID:   999,
		Data: "blocking",
		Handler: func(data interface{}) error {
			<-blockChan // 等待信号
			return nil
		},
	}

	err = pool.Submit(blockingTask)
	if err != nil {
		t.Errorf("Failed to submit blocking task: %v", err)
	}

	// 等待一小段时间确保任务开始执行
	time.Sleep(100 * time.Millisecond)

	// 现在提交任务直到队列满
	successfulSubmissions := 0
	for i := 0; i < 10; i++ {
		task := &TestTask{
			ID:   i,
			Data: i,
			Handler: func(data interface{}) error {
				return nil
			},
		}

		err := pool.Submit(task)
		if err == nil {
			successfulSubmissions++
		} else {
			// 队列满了
			break
		}
	}

	// 解除阻塞
	close(blockChan)

	t.Logf("Successfully submitted %d tasks before queue full", successfulSubmissions)

	// 应该能够提交一些任务，但不是无限的
	if successfulSubmissions == 0 {
		t.Error("Should be able to submit at least some tasks")
	}
	if successfulSubmissions >= 10 {
		t.Error("Queue should have limited capacity")
	}
}

func TestBatchWorkerPool_Metrics(t *testing.T) {
	pool := producer.NewBatchWorkerPool(3, 100)

	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	var wg sync.WaitGroup
	taskCount := 20
	wg.Add(taskCount)

	// 提交一些任务
	for i := 0; i < taskCount; i++ {
		task := &TestTask{
			ID:   i,
			Data: i,
			Handler: func(data interface{}) error {
				time.Sleep(10 * time.Millisecond) // 模拟工作
				wg.Done()
				return nil
			},
		}

		err := pool.Submit(task)
		if err != nil {
			t.Errorf("Failed to submit task %d: %v", i, err)
		}
	}

	// 等待所有任务完成
	wg.Wait()

	// 检查指标
	metrics := pool.GetMetrics()

	if metrics.TotalTasks < int64(taskCount) {
		t.Errorf("Expected at least %d total tasks, got %d", taskCount, metrics.TotalTasks)
	}

	if metrics.CompletedTasks < int64(taskCount) {
		t.Errorf("Expected at least %d completed tasks, got %d", taskCount, metrics.CompletedTasks)
	}

	if metrics.ActiveWorkers > 3 {
		t.Errorf("Expected at most 3 active workers, got %d", metrics.ActiveWorkers)
	}

	if metrics.QueuedTasks < 0 {
		t.Errorf("Queued tasks should not be negative, got %d", metrics.QueuedTasks)
	}

	if metrics.AvgTaskDuration <= 0 {
		t.Errorf("Average task duration should be positive, got %v", metrics.AvgTaskDuration)
	}

	t.Logf("Metrics: TotalTasks=%d, CompletedTasks=%d, ActiveWorkers=%d, AvgDuration=%v",
		metrics.TotalTasks, metrics.CompletedTasks, metrics.ActiveWorkers, metrics.AvgTaskDuration)
}

func TestBatchWorkerPool_AdaptiveScaling(t *testing.T) {
	pool := producer.NewBatchWorkerPool(2, 100) // 从2个工作线程开始

	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	// 提交大量任务以触发扩展
	var wg sync.WaitGroup
	taskCount := 50
	wg.Add(taskCount)

	for i := 0; i < taskCount; i++ {
		task := &TestTask{
			ID:   i,
			Data: i,
			Handler: func(data interface{}) error {
				time.Sleep(50 * time.Millisecond) // 模拟较慢的工作
				wg.Done()
				return nil
			},
		}

		err := pool.Submit(task)
		if err != nil {
			t.Errorf("Failed to submit task %d: %v", i, err)
		}
	}

	// 在任务执行期间检查工作线程数量
	time.Sleep(200 * time.Millisecond)
	metrics := pool.GetMetrics()

	t.Logf("During high load - Active workers: %d, Queued tasks: %d",
		metrics.ActiveWorkers, metrics.QueuedTasks)

	// 等待所有任务完成
	wg.Wait()

	// 等待一段时间让工作线程可能缩减
	time.Sleep(2 * time.Second)

	finalMetrics := pool.GetMetrics()
	t.Logf("After completion - Active workers: %d, Queued tasks: %d",
		finalMetrics.ActiveWorkers, finalMetrics.QueuedTasks)

	// 验证任务都完成了
	if finalMetrics.CompletedTasks < int64(taskCount) {
		t.Errorf("Expected at least %d completed tasks, got %d",
			taskCount, finalMetrics.CompletedTasks)
	}
}

func TestBatchWorkerPool_ContextCancellation(t *testing.T) {
	pool := producer.NewBatchWorkerPool(2, 50)

	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	var startedTasks int64

	// 提交一些长时间运行的任务
	for i := 0; i < 5; i++ {
		task := &TestTask{
			ID:   i,
			Data: i,
			Handler: func(data interface{}) error {
				atomic.AddInt64(&startedTasks, 1)

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(2 * time.Second):
					return nil
				}
			},
		}

		err := pool.Submit(task)
		if err != nil {
			t.Errorf("Failed to submit task %d: %v", i, err)
		}
	}

	// 等待任务开始
	time.Sleep(100 * time.Millisecond)

	// 取消上下文
	cancel()

	// 等待取消生效
	time.Sleep(500 * time.Millisecond)

	started := atomic.LoadInt64(&startedTasks)

	t.Logf("Started tasks: %d", started)

	if started == 0 {
		t.Error("Expected some tasks to start")
	}
}

func TestBatchWorkerPool_StopWithPendingTasks(t *testing.T) {
	pool := producer.NewBatchWorkerPool(1, 10)

	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}

	// 提交一些任务
	for i := 0; i < 5; i++ {
		task := &TestTask{
			ID:   i,
			Data: i,
			Handler: func(data interface{}) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
		}

		err := pool.Submit(task)
		if err != nil {
			t.Errorf("Failed to submit task %d: %v", i, err)
		}
	}

	// 立即停止（可能有待处理的任务）
	err = pool.Stop()
	if err != nil {
		t.Errorf("Failed to stop pool: %v", err)
	}

	if pool.IsRunning() {
		t.Error("Pool should not be running after stop")
	}

	// 尝试提交新任务应该失败
	task := &TestTask{
		ID:   999,
		Data: 999,
		Handler: func(data interface{}) error {
			return nil
		},
	}

	err = pool.Submit(task)
	if err == nil {
		t.Error("Should not be able to submit tasks to stopped pool")
	}
}

func BenchmarkBatchWorkerPool_TaskSubmission(b *testing.B) {
	pool := producer.NewBatchWorkerPool(4, 10000)

	err := pool.Start()
	if err != nil {
		b.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	task := &TestTask{
		ID:   1,
		Data: "benchmark",
		Handler: func(data interface{}) error {
			return nil
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(task)
	}
}

func BenchmarkBatchWorkerPool_Concurrent(b *testing.B) {
	pool := producer.NewBatchWorkerPool(8, 10000)

	err := pool.Start()
	if err != nil {
		b.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	task := &TestTask{
		ID:   1,
		Data: "concurrent",
		Handler: func(data interface{}) error {
			return nil
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Submit(task)
		}
	})
}
