package producer

import (
	"errors"
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/producer"
)

func TestDefaultErrorHandler_Creation(t *testing.T) {
	handler := producer.NewDefaultErrorHandler()

	if handler == nil {
		t.Fatal("Failed to create default error handler")
	}
}

func TestDefaultErrorHandler_HandleError(t *testing.T) {
	handler := producer.NewDefaultErrorHandler()

	// 创建测试消息
	message := &producer.ProducerMessage{
		Key:   "test-key",
		Value: map[string]interface{}{"data": "test"},
		Topic: "test-topic",
	}

	// 测试处理可重试的错误
	retryableErr := errors.New("kafka: leader not available")
	shouldRetry := handler.HandleError(retryableErr, message)

	if !shouldRetry {
		t.Error("Should retry for leader not available error")
	}

	// 测试处理不可重试的错误
	nonRetryableErr := errors.New("kafka: message was too large")
	shouldRetry = handler.HandleError(nonRetryableErr, message)

	if shouldRetry {
		t.Error("Should not retry for message too large error")
	}

	// 测试未知错误（应该重试一次）
	unknownErr := errors.New("unknown error")
	shouldRetry = handler.HandleError(unknownErr, message)

	if !shouldRetry {
		t.Error("Should retry once for unknown error")
	}

	// 第二次相同的未知错误应该不重试
	shouldRetry = handler.HandleError(unknownErr, message)
	if shouldRetry {
		t.Error("Should not retry unknown error twice")
	}
}

func TestDefaultErrorHandler_RetryLimit(t *testing.T) {
	handler := producer.NewDefaultErrorHandler()

	message := &producer.ProducerMessage{
		Key:   "retry-test-key",
		Value: map[string]interface{}{"data": "test"},
		Topic: "test-topic",
	}

	// 测试重试次数限制
	retryableErr := errors.New("kafka: request timed out")
	
	// 前3次应该可以重试
	for i := 0; i < 3; i++ {
		shouldRetry := handler.HandleError(retryableErr, message)
		if !shouldRetry {
			t.Errorf("Should retry attempt %d for timeout error", i+1)
		}
	}

	// 第4次应该不能重试
	shouldRetry := handler.HandleError(retryableErr, message)
	if shouldRetry {
		t.Error("Should not retry after max retries exceeded")
	}
}

func TestRetryManager_Creation(t *testing.T) {
	manager := producer.NewRetryManager(3, 100*time.Millisecond)

	if manager == nil {
		t.Fatal("Failed to create retry manager")
	}
}

func TestRetryManager_SuccessfulRetry(t *testing.T) {
	manager := producer.NewRetryManager(3, 10*time.Millisecond)

	attemptCount := 0
	operation := func() error {
		attemptCount++
		if attemptCount < 3 {
			return errors.New("temporary failure")
		}
		return nil // 第3次成功
	}

	err := manager.Retry(operation)
	if err != nil {
		t.Errorf("Expected successful retry, got error: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestRetryManager_FailedRetry(t *testing.T) {
	manager := producer.NewRetryManager(2, 10*time.Millisecond)

	attemptCount := 0
	operation := func() error {
		attemptCount++
		return errors.New("persistent failure")
	}

	err := manager.Retry(operation)
	if err == nil {
		t.Error("Expected retry to fail after max attempts")
	}

	// 应该尝试3次（初始 + 2次重试）
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestRetryManager_BackoffTypes(t *testing.T) {
	testCases := []struct {
		name        string
		backoffType producer.BackoffType
	}{
		{"Fixed", producer.FixedBackoff},
		{"Linear", producer.LinearBackoff},
		{"Exponential", producer.ExponentialBackoff},
		{"Jittered", producer.JitteredBackoff},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := producer.NewRetryManager(2, 10*time.Millisecond)
			manager.SetBackoffType(tc.backoffType)

			attemptCount := 0
			operation := func() error {
				attemptCount++
				return errors.New("test failure")
			}

			err := manager.Retry(operation)
			if err == nil {
				t.Error("Expected retry to fail")
			}

			if attemptCount != 3 {
				t.Errorf("Expected 3 attempts, got %d", attemptCount)
			}
		})
	}
}

func TestRetryManager_MaxDelay(t *testing.T) {
	manager := producer.NewRetryManager(3, 100*time.Millisecond)
	manager.SetMaxDelay(200 * time.Millisecond)

	start := time.Now()
	attemptCount := 0
	operation := func() error {
		attemptCount++
		return errors.New("test failure")
	}

	manager.Retry(operation)
	elapsed := time.Since(start)

	// 验证总时间不会过长（由于最大延迟限制）
	maxExpectedTime := 1 * time.Second // 给一些缓冲
	if elapsed > maxExpectedTime {
		t.Errorf("Retry took too long: %v, expected less than %v", elapsed, maxExpectedTime)
	}

	t.Logf("Retry with max delay took %v for %d attempts", elapsed, attemptCount)
}

func TestRetryManager_Metrics(t *testing.T) {
	manager := producer.NewRetryManager(2, 10*time.Millisecond)

	// 执行一些成功的重试
	successOperation := func() error {
		return nil // 立即成功
	}
	manager.Retry(successOperation)

	// 执行一些失败的重试
	failOperation := func() error {
		return errors.New("always fail")
	}
	manager.Retry(failOperation)

	metrics := manager.GetMetrics()

	if metrics.TotalRetries < 2 {
		t.Errorf("Expected at least 2 total retries, got %d", metrics.TotalRetries)
	}

	if metrics.FailedRetries < 1 {
		t.Errorf("Expected at least 1 failed retry, got %d", metrics.FailedRetries)
	}

	t.Logf("Retry metrics: Total=%d, Success=%d, Failed=%d, AvgDelay=%v, MaxDelay=%v",
		metrics.TotalRetries, metrics.SuccessRetries, metrics.FailedRetries,
		metrics.AvgRetryDelay, metrics.MaxRetryDelay)
}

func TestRetryManager_Reset(t *testing.T) {
	manager := producer.NewRetryManager(2, 10*time.Millisecond)

	// 执行一些重试以生成指标
	operation := func() error {
		return errors.New("test failure")
	}
	manager.Retry(operation)

	// 重置指标
	manager.Reset()

	// 检查指标是否被重置
	metrics := manager.GetMetrics()
	if metrics.TotalRetries != 0 {
		t.Errorf("Expected 0 total retries after reset, got %d", metrics.TotalRetries)
	}
	if metrics.SuccessRetries != 0 {
		t.Errorf("Expected 0 success retries after reset, got %d", metrics.SuccessRetries)
	}
	if metrics.FailedRetries != 0 {
		t.Errorf("Expected 0 failed retries after reset, got %d", metrics.FailedRetries)
	}
	if metrics.AvgRetryDelay != 0 {
		t.Errorf("Expected 0 avg delay after reset, got %v", metrics.AvgRetryDelay)
	}
	if metrics.MaxRetryDelay != 0 {
		t.Errorf("Expected 0 max delay after reset, got %v", metrics.MaxRetryDelay)
	}
}

func TestRetryManager_ConcurrentAccess(t *testing.T) {
	manager := producer.NewRetryManager(2, 10*time.Millisecond)

	// 并发执行重试
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			operation := func() error {
				if id%2 == 0 {
					return nil // 偶数ID成功
				}
				return errors.New("odd id failure") // 奇数ID失败
			}
			manager.Retry(operation)
			done <- true
		}(i)
	}

	// 等待所有协程完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 检查最终指标
	metrics := manager.GetMetrics()
	t.Logf("Concurrent retry metrics: Total=%d, Success=%d, Failed=%d",
		metrics.TotalRetries, metrics.SuccessRetries, metrics.FailedRetries)

	if metrics.TotalRetries == 0 {
		t.Error("Expected some retries from concurrent access")
	}
}

func TestErrorHandler_ErrorPatterns(t *testing.T) {
	handler := producer.NewDefaultErrorHandler()

	testCases := []struct {
		errorMsg     string
		shouldRetry  bool
		description  string
	}{
		{"kafka: leader not available", true, "leader not available should be retryable"},
		{"kafka: request timed out", true, "timeout should be retryable"},
		{"kafka: message was too large", false, "message too large should not be retryable"},
		{"kafka: invalid topic", false, "invalid topic should not be retryable"},
		{"some random error", true, "unknown errors should be retryable once"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			message := &producer.ProducerMessage{
				Key:   "pattern-test-key",
				Value: map[string]interface{}{"data": "test"},
				Topic: "test-topic",
			}

			err := errors.New(tc.errorMsg)
			shouldRetry := handler.HandleError(err, message)

			if shouldRetry != tc.shouldRetry {
				t.Errorf("Error '%s': expected retry=%v, got retry=%v", 
					tc.errorMsg, tc.shouldRetry, shouldRetry)
			}
		})
	}
}

func BenchmarkDefaultErrorHandler_HandleError(b *testing.B) {
	handler := producer.NewDefaultErrorHandler()
	err := errors.New("kafka: request timed out")
	message := &producer.ProducerMessage{
		Key:   "bench-key",
		Value: map[string]interface{}{"data": "benchmark"},
		Topic: "bench-topic",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.HandleError(err, message)
	}
}

func BenchmarkRetryManager_Retry(b *testing.B) {
	manager := producer.NewRetryManager(3, 1*time.Millisecond)

	operation := func() error {
		return nil // 立即成功，避免实际延迟
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Retry(operation)
	}
}

func BenchmarkRetryManager_Concurrent(b *testing.B) {
	manager := producer.NewRetryManager(3, 1*time.Millisecond)

	operation := func() error {
		return nil
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			manager.Retry(operation)
		}
	})
}
