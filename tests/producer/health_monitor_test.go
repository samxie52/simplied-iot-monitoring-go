package producer

import (
	"context"
	"sync"
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/producer"
)

// MockHealthChecker 用于测试的健康检查器
type MockHealthChecker struct {
	name    string
	healthy bool
	delay   time.Duration
	mu      sync.RWMutex
}

func NewMockHealthChecker(name string, healthy bool, delay time.Duration) *MockHealthChecker {
	return &MockHealthChecker{
		name:    name,
		healthy: healthy,
		delay:   delay,
	}
}

func (m *MockHealthChecker) Name() string {
	return m.name
}

func (m *MockHealthChecker) Check(ctx context.Context) producer.HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 模拟检查延迟
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return producer.HealthStatus{
				Component: m.name,
				Status:    producer.HealthStatusUnknown,
				Message:   "Check cancelled",
				Timestamp: time.Now(),
			}
		}
	}

	status := producer.HealthStatusHealthy
	message := "OK"
	if !m.healthy {
		status = producer.HealthStatusUnhealthy
		message = "Mock failure"
	}

	return producer.HealthStatus{
		Component: m.name,
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
	}
}

func (m *MockHealthChecker) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

func TestHealthMonitor_Creation(t *testing.T) {
	monitor := producer.NewHealthMonitor(5*time.Second, nil)

	if monitor == nil {
		t.Fatal("Failed to create health monitor")
	}
}

func TestHealthMonitor_AddChecker(t *testing.T) {
	monitor := producer.NewHealthMonitor(5*time.Second, nil)

	checker1 := NewMockHealthChecker("test1", true, 0)
	checker2 := NewMockHealthChecker("test2", true, 0)

	monitor.AddChecker(checker1)
	monitor.AddChecker(checker2)

	// 验证检查器已添加
	status := monitor.GetOverallStatus()
	if status.Status != producer.HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %v", status.Status)
	}

	t.Log("Health checkers added successfully")
}

func TestHealthMonitor_SingleHealthyChecker(t *testing.T) {
	monitor := producer.NewHealthMonitor(5*time.Second, nil)
	checker := NewMockHealthChecker("healthy_service", true, 0)

	monitor.AddChecker(checker)

	status := monitor.GetOverallStatus()
	if status.Status != producer.HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %v", status.Status)
	}

	if status.Component != "overall" {
		t.Errorf("Expected component 'overall', got %s", status.Component)
	}

	t.Log("Single healthy checker test passed")
}

func TestHealthMonitor_SingleUnhealthyChecker(t *testing.T) {
	monitor := producer.NewHealthMonitor(5*time.Second, nil)
	checker := NewMockHealthChecker("unhealthy_service", false, 0)

	monitor.AddChecker(checker)

	status := monitor.GetOverallStatus()
	if status.Status != producer.HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status, got %v", status.Status)
	}

	t.Log("Single unhealthy checker test passed")
}

func TestHealthMonitor_MixedCheckers(t *testing.T) {
	monitor := producer.NewHealthMonitor(5*time.Second, nil)

	healthyChecker := NewMockHealthChecker("healthy_service", true, 0)
	unhealthyChecker := NewMockHealthChecker("unhealthy_service", false, 0)

	monitor.AddChecker(healthyChecker)
	monitor.AddChecker(unhealthyChecker)

	status := monitor.GetOverallStatus()
	if status.Status != producer.HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status (due to mixed), got %v", status.Status)
	}

	t.Log("Mixed checkers test passed")
}

func TestHealthMonitor_GetComponentStatus(t *testing.T) {
	monitor := producer.NewHealthMonitor(5*time.Second, nil)

	checker1 := NewMockHealthChecker("service1", true, 0)
	checker2 := NewMockHealthChecker("service2", false, 0)

	monitor.AddChecker(checker1)
	monitor.AddChecker(checker2)

	// 获取特定组件状态
	status1 := monitor.GetComponentStatus("service1")
	if status1.Status != producer.HealthStatusHealthy {
		t.Errorf("Expected service1 to be healthy, got %v", status1.Status)
	}

	status2 := monitor.GetComponentStatus("service2")
	if status2.Status != producer.HealthStatusUnhealthy {
		t.Errorf("Expected service2 to be unhealthy, got %v", status2.Status)
	}

	// 获取不存在的组件状态
	statusNone := monitor.GetComponentStatus("nonexistent")
	if statusNone.Status != producer.HealthStatusUnknown {
		t.Errorf("Expected unknown status for nonexistent component, got %v", statusNone.Status)
	}

	t.Log("Component status retrieval test passed")
}

func TestHealthMonitor_GetAllStatuses(t *testing.T) {
	monitor := producer.NewHealthMonitor(5*time.Second, nil)

	checker1 := NewMockHealthChecker("service1", true, 0)
	checker2 := NewMockHealthChecker("service2", false, 0)
	checker3 := NewMockHealthChecker("service3", true, 0)

	monitor.AddChecker(checker1)
	monitor.AddChecker(checker2)
	monitor.AddChecker(checker3)

	statuses := monitor.GetAllStatuses()

	if len(statuses) != 3 {
		t.Errorf("Expected 3 statuses, got %d", len(statuses))
	}

	// 验证每个状态
	statusMap := make(map[string]producer.HealthStatus)
	for _, status := range statuses {
		statusMap[status.Component] = status
	}

	if statusMap["service1"].Status != producer.HealthStatusHealthy {
		t.Error("service1 should be healthy")
	}

	if statusMap["service2"].Status != producer.HealthStatusUnhealthy {
		t.Error("service2 should be unhealthy")
	}

	if statusMap["service3"].Status != producer.HealthStatusHealthy {
		t.Error("service3 should be healthy")
	}

	t.Log("All statuses retrieval test passed")
}

func TestHealthMonitor_AlertCallback(t *testing.T) {
	alertCalled := make(chan producer.HealthStatus, 10)
	alertCallback := func(status producer.HealthStatus) {
		alertCalled <- status
	}

	monitor := producer.NewHealthMonitor(100*time.Millisecond, alertCallback)
	checker := NewMockHealthChecker("test_service", true, 0)

	monitor.AddChecker(checker)

	// 启动监控
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitor.Start(ctx)

	// 等待初始检查
	time.Sleep(150 * time.Millisecond)

	// 改变健康状态
	checker.SetHealthy(false)

	// 等待状态变化检测
	time.Sleep(150 * time.Millisecond)

	// 验证是否收到告警
	select {
	case status := <-alertCalled:
		if status.Status != producer.HealthStatusUnhealthy {
			t.Errorf("Expected unhealthy alert, got %v", status.Status)
		}
		t.Log("Alert callback triggered successfully")
	case <-time.After(1 * time.Second):
		t.Error("Alert callback was not triggered")
	}
}

func TestHealthMonitor_PeriodicChecking(t *testing.T) {
	checkCount := 0
	mu := sync.Mutex{}

	// 创建一个会计数检查次数的检查器
	checker := &MockHealthChecker{
		name:    "counting_checker",
		healthy: true,
	}

	// 重写Check方法以计数
	originalCheck := checker.Check
	checker.Check = func(ctx context.Context) producer.HealthStatus {
		mu.Lock()
		checkCount++
		mu.Unlock()
		return originalCheck(ctx)
	}

	monitor := producer.NewHealthMonitor(50*time.Millisecond, nil)
	monitor.AddChecker(checker)

	// 启动监控
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	monitor.Start(ctx)

	// 等待监控完成
	<-ctx.Done()

	mu.Lock()
	finalCount := checkCount
	mu.Unlock()

	// 应该至少检查了几次
	if finalCount < 3 {
		t.Errorf("Expected at least 3 checks, got %d", finalCount)
	}

	t.Logf("Periodic checking completed with %d checks", finalCount)
}

func TestHealthMonitor_ContextCancellation(t *testing.T) {
	monitor := producer.NewHealthMonitor(1*time.Second, nil)
	checker := NewMockHealthChecker("test_service", true, 0)

	monitor.AddChecker(checker)

	ctx, cancel := context.WithCancel(context.Background())

	// 启动监控
	done := make(chan bool)
	go func() {
		monitor.Start(ctx)
		done <- true
	}()

	// 等待一段时间后取消
	time.Sleep(100 * time.Millisecond)
	cancel()

	// 验证监控是否正确停止
	select {
	case <-done:
		t.Log("Health monitor stopped correctly on context cancellation")
	case <-time.After(1 * time.Second):
		t.Error("Health monitor did not stop within timeout")
	}
}

func TestHealthMonitor_SlowChecker(t *testing.T) {
	monitor := producer.NewHealthMonitor(100*time.Millisecond, nil)

	// 创建一个慢检查器
	slowChecker := NewMockHealthChecker("slow_service", true, 200*time.Millisecond)
	fastChecker := NewMockHealthChecker("fast_service", true, 0)

	monitor.AddChecker(slowChecker)
	monitor.AddChecker(fastChecker)

	start := time.Now()
	statuses := monitor.GetAllStatuses()
	duration := time.Since(start)

	// 验证所有检查器都返回了状态
	if len(statuses) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(statuses))
	}

	// 验证检查时间合理（应该等待慢检查器）
	if duration < 200*time.Millisecond {
		t.Errorf("Expected duration >= 200ms, got %v", duration)
	}

	t.Logf("Slow checker test completed in %v", duration)
}

func TestHealthMonitor_ConcurrentAccess(t *testing.T) {
	monitor := producer.NewHealthMonitor(50*time.Millisecond, nil)

	// 添加多个检查器
	for i := 0; i < 5; i++ {
		checker := NewMockHealthChecker("service"+string(rune('1'+i)), true, 0)
		monitor.AddChecker(checker)
	}

	// 启动监控
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitor.Start(ctx)

	// 并发访问监控器
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = monitor.GetOverallStatus()
				_ = monitor.GetAllStatuses()
				_ = monitor.GetComponentStatus("service1")
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
	t.Log("Concurrent access test completed successfully")
}

func TestKafkaProducerHealthChecker(t *testing.T) {
	// 创建一个模拟的Kafka生产者
	mockProducer := &MockKafkaProducer{healthy: true}
	checker := producer.NewKafkaProducerHealthChecker(mockProducer)

	if checker.Name() != "kafka_producer" {
		t.Errorf("Expected name 'kafka_producer', got %s", checker.Name())
	}

	// 测试健康状态
	ctx := context.Background()
	status := checker.Check(ctx)

	if status.Status != producer.HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %v", status.Status)
	}

	// 测试不健康状态
	mockProducer.healthy = false
	status = checker.Check(ctx)

	if status.Status != producer.HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status, got %v", status.Status)
	}

	t.Log("Kafka producer health checker test passed")
}

func TestConnectionPoolHealthChecker(t *testing.T) {
	mockPool := &MockConnectionPool{healthy: true}
	checker := producer.NewConnectionPoolHealthChecker(mockPool)

	if checker.Name() != "connection_pool" {
		t.Errorf("Expected name 'connection_pool', got %s", checker.Name())
	}

	ctx := context.Background()
	status := checker.Check(ctx)

	if status.Status != producer.HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %v", status.Status)
	}

	t.Log("Connection pool health checker test passed")
}

func TestSystemResourceHealthChecker(t *testing.T) {
	checker := producer.NewSystemResourceHealthChecker(90.0, 90.0)

	if checker.Name() != "system_resources" {
		t.Errorf("Expected name 'system_resources', got %s", checker.Name())
	}

	ctx := context.Background()
	status := checker.Check(ctx)

	// 系统资源检查应该返回一个状态（具体状态取决于系统）
	if status.Component != "system_resources" {
		t.Errorf("Expected component 'system_resources', got %s", status.Component)
	}

	t.Log("System resource health checker test passed")
}

// Mock implementations for testing

type MockKafkaProducer struct {
	healthy bool
}

func (m *MockKafkaProducer) IsHealthy() bool {
	return m.healthy
}

type MockConnectionPool struct {
	healthy bool
}

func (m *MockConnectionPool) IsHealthy() bool {
	return m.healthy
}

func (m *MockConnectionPool) ActiveConnections() int {
	if m.healthy {
		return 5
	}
	return 0
}

func (m *MockConnectionPool) TotalConnections() int {
	return 10
}

// Benchmarks

func BenchmarkHealthMonitor_GetOverallStatus(b *testing.B) {
	monitor := producer.NewHealthMonitor(5*time.Second, nil)

	for i := 0; i < 5; i++ {
		checker := NewMockHealthChecker("service"+string(rune('1'+i)), true, 0)
		monitor.AddChecker(checker)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.GetOverallStatus()
	}
}

func BenchmarkHealthMonitor_GetAllStatuses(b *testing.B) {
	monitor := producer.NewHealthMonitor(5*time.Second, nil)

	for i := 0; i < 10; i++ {
		checker := NewMockHealthChecker("service"+string(rune('1'+i)), true, 0)
		monitor.AddChecker(checker)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.GetAllStatuses()
	}
}

func BenchmarkHealthMonitor_ConcurrentAccess(b *testing.B) {
	monitor := producer.NewHealthMonitor(5*time.Second, nil)

	for i := 0; i < 5; i++ {
		checker := NewMockHealthChecker("service"+string(rune('1'+i)), true, 0)
		monitor.AddChecker(checker)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = monitor.GetOverallStatus()
			_ = monitor.GetComponentStatus("service1")
		}
	})
}
