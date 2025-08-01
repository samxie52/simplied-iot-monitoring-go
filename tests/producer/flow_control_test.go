package producer

import (
	"context"
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/producer"
)

func TestRateLimiter_Creation(t *testing.T) {
	limiter := producer.NewRateLimiter(100) // 100 requests/sec

	if limiter == nil {
		t.Fatal("Failed to create rate limiter")
	}

	if limiter.GetRate() != 100 {
		t.Errorf("Expected rate 100, got %d", limiter.GetRate())
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter := producer.NewRateLimiter(10) // 10 requests/sec

	// 初始应该允许请求
	if !limiter.Allow() {
		t.Error("First request should be allowed")
	}

	// 快速连续请求应该被限制
	allowed := 0
	for i := 0; i < 20; i++ {
		if limiter.Allow() {
			allowed++
		}
	}

	// 应该允许一些请求，但不是全部
	if allowed == 0 {
		t.Error("No requests were allowed")
	}
	if allowed >= 20 {
		t.Error("Too many requests were allowed")
	}

	t.Logf("Allowed %d out of 20 requests", allowed)
}

func TestRateLimiter_Wait(t *testing.T) {
	limiter := producer.NewRateLimiter(5) // 5 requests/sec

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()

	// 第一个请求应该立即通过
	err := limiter.Wait(ctx)
	if err != nil {
		t.Errorf("First wait should not error: %v", err)
	}

	// 第二个请求可能需要等待
	err = limiter.Wait(ctx)
	if err != nil {
		t.Errorf("Second wait should not error: %v", err)
	}

	elapsed := time.Since(start)
	t.Logf("Two requests took %v", elapsed)

	// 测试上下文取消
	fastCtx, fastCancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer fastCancel()

	// 快速耗尽令牌
	for i := 0; i < 10; i++ {
		limiter.Allow()
	}

	err = limiter.Wait(fastCtx)
	if err == nil {
		t.Error("Expected context timeout error")
	}
}

func TestRateLimiter_SetRate(t *testing.T) {
	limiter := producer.NewRateLimiter(10)

	// 更改速率
	limiter.SetRate(20)
	if limiter.GetRate() != 20 {
		t.Errorf("Expected rate 20 after SetRate, got %d", limiter.GetRate())
	}

	// 测试新速率是否生效
	allowed := 0
	for i := 0; i < 30; i++ {
		if limiter.Allow() {
			allowed++
		}
	}

	t.Logf("With rate 20, allowed %d out of 30 requests", allowed)
}

func TestBackpressureController_Creation(t *testing.T) {
	controller := producer.NewBackpressureController()

	if controller == nil {
		t.Fatal("Failed to create backpressure controller")
	}

	status := controller.GetStatus()
	if status != producer.StatusNormal {
		t.Errorf("Expected initial status Normal, got %v", status)
	}

	load := controller.GetCurrentLoad()
	if load != 0.0 {
		t.Errorf("Expected initial load 0.0, got %f", load)
	}
}

func TestBackpressureController_Update(t *testing.T) {
	controller := producer.NewBackpressureController()

	// 测试正常负载
	controller.Update(30, 100) // 30%负载
	if controller.GetStatus() != producer.StatusNormal {
		t.Error("Expected Normal status with 30% load")
	}
	if controller.GetCurrentLoad() != 0.3 {
		t.Errorf("Expected load 0.3, got %f", controller.GetCurrentLoad())
	}

	// 测试警告负载
	controller.Update(75, 100) // 75%负载
	if controller.GetStatus() != producer.StatusWarning {
		t.Error("Expected Warning status with 75% load")
	}

	// 测试临界负载
	controller.Update(95, 100) // 95%负载
	if controller.GetStatus() != producer.StatusCritical {
		t.Error("Expected Critical status with 95% load")
	}
}

func TestBackpressureController_ShouldDrop(t *testing.T) {
	controller := producer.NewBackpressureController()

	// 正常状态不应该丢弃
	controller.Update(30, 100)
	shouldDrop := false
	for i := 0; i < 10; i++ {
		if controller.ShouldDrop() {
			shouldDrop = true
			break
		}
	}
	if shouldDrop {
		t.Error("Should not drop requests in normal status")
	}

	// 临界状态应该丢弃一些请求
	controller.Update(98, 100)
	dropCount := 0
	totalRequests := 100
	for i := 0; i < totalRequests; i++ {
		if controller.ShouldDrop() {
			dropCount++
		}
	}

	if dropCount == 0 {
		t.Error("Expected some requests to be dropped in critical status")
	}
	// 在临界状态下，由于负载>0.95，可能会丢弃大部分或全部请求
	// 这是正常的背压行为
	if dropCount < totalRequests/4 {
		t.Errorf("Expected at least 25%% of requests to be dropped, got %d%%", dropCount*100/totalRequests)
	}

	t.Logf("Dropped %d out of %d requests in critical status (%d%%)", dropCount, totalRequests, dropCount*100/totalRequests)
}

func TestBackpressureController_ShouldThrottle(t *testing.T) {
	controller := producer.NewBackpressureController()

	// 正常状态不应该限流
	controller.Update(30, 100)
	if controller.ShouldThrottle() {
		t.Error("Should not throttle in normal status")
	}

	// 警告状态应该限流
	controller.Update(75, 100)
	if !controller.ShouldThrottle() {
		t.Error("Should throttle in warning status")
	}

	// 临界状态应该限流
	controller.Update(95, 100)
	if !controller.ShouldThrottle() {
		t.Error("Should throttle in critical status")
	}
}

func TestBackpressureController_GetThrottleDelay(t *testing.T) {
	controller := producer.NewBackpressureController()

	// 正常状态没有延迟
	controller.Update(30, 100)
	delay := controller.GetThrottleDelay()
	if delay != 0 {
		t.Errorf("Expected no delay in normal status, got %v", delay)
	}

	// 警告状态有适中延迟
	controller.Update(75, 100)
	delay = controller.GetThrottleDelay()
	if delay <= 0 {
		t.Error("Expected positive delay in warning status")
	}
	if delay > 100*time.Millisecond {
		t.Errorf("Warning delay too high: %v", delay)
	}

	// 临界状态有更高延迟
	controller.Update(95, 100)
	delay = controller.GetThrottleDelay()
	if delay <= 100*time.Millisecond {
		t.Error("Expected higher delay in critical status")
	}

	t.Logf("Critical status delay: %v", delay)
}

func TestBackpressureController_Statistics(t *testing.T) {
	controller := producer.NewBackpressureController()

	// 生成一些请求
	controller.Update(80, 100) // 警告状态

	totalRequests := 100
	for i := 0; i < totalRequests; i++ {
		controller.ShouldDrop()
		controller.ShouldThrottle()
	}

	stats := controller.GetStatistics()

	if stats.TotalRequests != int64(totalRequests) {
		t.Errorf("Expected %d total requests, got %d", totalRequests, stats.TotalRequests)
	}

	if stats.CurrentLoad != 0.8 {
		t.Errorf("Expected load 0.8, got %f", stats.CurrentLoad)
	}

	if stats.Status != producer.StatusWarning {
		t.Errorf("Expected Warning status, got %v", stats.Status)
	}

	if stats.ThrottledRequests != int64(totalRequests) {
		t.Errorf("Expected %d throttled requests, got %d", totalRequests, stats.ThrottledRequests)
	}

	if stats.ThrottleRate != 1.0 {
		t.Errorf("Expected throttle rate 1.0, got %f", stats.ThrottleRate)
	}

	t.Logf("Statistics: %+v", stats)
}

func TestBackpressureController_Reset(t *testing.T) {
	controller := producer.NewBackpressureController()

	// 生成一些活动
	controller.Update(90, 100)
	for i := 0; i < 50; i++ {
		controller.ShouldDrop()
		controller.ShouldThrottle()
	}

	// 重置
	controller.Reset()

	stats := controller.GetStatistics()
	if stats.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests after reset, got %d", stats.TotalRequests)
	}
	if stats.DroppedRequests != 0 {
		t.Errorf("Expected 0 dropped requests after reset, got %d", stats.DroppedRequests)
	}
	if stats.ThrottledRequests != 0 {
		t.Errorf("Expected 0 throttled requests after reset, got %d", stats.ThrottledRequests)
	}
	if stats.CurrentLoad != 0.0 {
		t.Errorf("Expected 0.0 load after reset, got %f", stats.CurrentLoad)
	}
	if stats.Status != producer.StatusNormal {
		t.Errorf("Expected Normal status after reset, got %v", stats.Status)
	}
}

func TestBackpressureController_AdaptiveThresholds(t *testing.T) {
	controller := producer.NewBackpressureController()

	// 模拟持续高负载以触发自适应调整
	for i := 0; i < 150; i++ { // 超过历史大小以确保自适应
		load := 0.85 + float64(i%10)*0.01 // 85%-95%之间的负载
		controller.Update(int64(load*100), 100)

		// 生成一些请求以触发统计
		controller.ShouldDrop()
		controller.ShouldThrottle()
	}

	stats := controller.GetStatistics()

	// 在持续高负载后，阈值可能会被调整
	t.Logf("Adaptive thresholds - Warning: %f, Critical: %f",
		stats.WarningThreshold, stats.CriticalThreshold)

	// 阈值应该在合理范围内
	if stats.WarningThreshold < 0.3 || stats.WarningThreshold > 0.9 {
		t.Errorf("Warning threshold out of reasonable range: %f", stats.WarningThreshold)
	}
	if stats.CriticalThreshold < 0.5 || stats.CriticalThreshold > 1.0 {
		t.Errorf("Critical threshold out of reasonable range: %f", stats.CriticalThreshold)
	}
	if stats.WarningThreshold >= stats.CriticalThreshold {
		t.Error("Warning threshold should be less than critical threshold")
	}
}

func BenchmarkRateLimiter_Allow(b *testing.B) {
	limiter := producer.NewRateLimiter(1000000) // 高速率避免限制

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}

func BenchmarkBackpressureController_ShouldDrop(b *testing.B) {
	controller := producer.NewBackpressureController()
	controller.Update(80, 100) // 设置警告状态

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		controller.ShouldDrop()
	}
}

func BenchmarkBackpressureController_Concurrent(b *testing.B) {
	controller := producer.NewBackpressureController()
	controller.Update(80, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			controller.ShouldDrop()
			controller.ShouldThrottle()
		}
	})
}
