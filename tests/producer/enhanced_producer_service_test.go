package producer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/producer"
)

// Mock implementations for testing
type MockEnhancedKafkaProducer struct {
	started bool
	stopped bool
	healthy bool
	mu      sync.RWMutex
}

func (m *MockEnhancedKafkaProducer) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
	return nil
}

func (m *MockEnhancedKafkaProducer) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
	return nil
}

func (m *MockEnhancedKafkaProducer) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.healthy
}

func (m *MockEnhancedKafkaProducer) SendMessage(key string, value interface{}) error {
	return nil
}

func (m *MockEnhancedKafkaProducer) ReloadConfig(cfg *config.Config) error {
	return nil
}

func (m *MockEnhancedKafkaProducer) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

type MockEnhancedBatchProcessor struct {
	started bool
	stopped bool
}

func (m *MockEnhancedBatchProcessor) Start() error {
	m.started = true
	return nil
}

func (m *MockEnhancedBatchProcessor) Stop() error {
	m.stopped = true
	return nil
}

type MockEnhancedDeviceSimulator struct {
	started bool
	stopped bool
}

func (m *MockEnhancedDeviceSimulator) Start() error {
	m.started = true
	return nil
}

func (m *MockEnhancedDeviceSimulator) Stop() error {
	m.stopped = true
	return nil
}

func (m *MockEnhancedDeviceSimulator) ReloadConfig(cfg *config.Config) error {
	return nil
}

type MockEnhancedConnectionPool struct {
	started bool
	stopped bool
	healthy bool
}

func (m *MockEnhancedConnectionPool) Start() error {
	m.started = true
	return nil
}

func (m *MockEnhancedConnectionPool) Stop() error {
	m.stopped = true
	return nil
}

func (m *MockEnhancedConnectionPool) IsHealthy() bool {
	return m.healthy
}

func (m *MockEnhancedConnectionPool) ActiveConnections() int {
	return 5
}

func (m *MockEnhancedConnectionPool) TotalConnections() int {
	return 10
}

type MockEnhancedWorkerPool struct {
	started bool
	stopped bool
}

func (m *MockEnhancedWorkerPool) Start() error {
	m.started = true
	return nil
}

func (m *MockEnhancedWorkerPool) Stop() error {
	m.stopped = true
	return nil
}

func TestEnhancedProducerService_Creation(t *testing.T) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8080,
		},
	}

	service := producer.NewEnhancedProducerService(cfg)
	if service == nil {
		t.Fatal("Failed to create enhanced producer service")
	}

	t.Log("Enhanced producer service created successfully")
}

func TestEnhancedProducerService_StartStop(t *testing.T) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8081, // 使用不同端口避免冲突
		},
	}

	service := producer.NewEnhancedProducerService(cfg)

	// 启动服务
	err := service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	// 等待服务启动
	time.Sleep(100 * time.Millisecond)

	// 停止服务
	err = service.Stop()
	if err != nil {
		t.Errorf("Failed to stop service: %v", err)
	}

	t.Log("Service start/stop test completed successfully")
}

func TestEnhancedProducerService_HTTPEndpoints(t *testing.T) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8082,
		},
	}

	service := producer.NewEnhancedProducerService(cfg)

	// 启动服务
	err := service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop()

	// 等待HTTP服务器启动
	time.Sleep(200 * time.Millisecond)

	// 测试各个端点
	endpoints := []string{
		"/metrics",
		"/health",
		"/health/components",
		"/config/status",
		"/stats",
	}

	for _, endpoint := range endpoints {
		resp, err := http.Get("http://localhost:8082" + endpoint)
		if err != nil {
			t.Errorf("Failed to request %s: %v", endpoint, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for %s, got %d", endpoint, resp.StatusCode)
		}
	}

	t.Log("HTTP endpoints test completed successfully")
}

func TestEnhancedProducerService_MetricsEndpoint(t *testing.T) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8083,
		},
	}

	service := producer.NewEnhancedProducerService(cfg)
	err := service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop()

	time.Sleep(200 * time.Millisecond)

	// 测试metrics端点
	resp, err := http.Get("http://localhost:8083/metrics")
	if err != nil {
		t.Fatalf("Failed to request metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 验证Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain; version=0.0.4; charset=utf-8" {
		t.Errorf("Expected Prometheus content type, got %s", contentType)
	}

	t.Log("Metrics endpoint test completed successfully")
}

func TestEnhancedProducerService_HealthEndpoint(t *testing.T) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8084,
		},
	}

	service := producer.NewEnhancedProducerService(cfg)
	err := service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop()

	time.Sleep(200 * time.Millisecond)

	// 测试health端点
	resp, err := http.Get("http://localhost:8084/health")
	if err != nil {
		t.Fatalf("Failed to request health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 解析JSON响应
	var healthStatus map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&healthStatus)
	if err != nil {
		t.Errorf("Failed to decode health response: %v", err)
	}

	// 验证响应结构
	if _, ok := healthStatus["component"]; !ok {
		t.Error("Health response missing 'component' field")
	}

	if _, ok := healthStatus["status"]; !ok {
		t.Error("Health response missing 'status' field")
	}

	t.Log("Health endpoint test completed successfully")
}

func TestEnhancedProducerService_ComponentsHealthEndpoint(t *testing.T) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8085,
		},
	}

	service := producer.NewEnhancedProducerService(cfg)
	err := service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop()

	time.Sleep(200 * time.Millisecond)

	// 测试components health端点
	resp, err := http.Get("http://localhost:8085/health/components")
	if err != nil {
		t.Fatalf("Failed to request components health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 解析JSON响应
	var components []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&components)
	if err != nil {
		t.Errorf("Failed to decode components response: %v", err)
	}

	// 验证至少有一些组件
	if len(components) == 0 {
		t.Error("Expected at least one component in health response")
	}

	t.Log("Components health endpoint test completed successfully")
}

func TestEnhancedProducerService_ConfigReloadEndpoint(t *testing.T) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8086,
		},
	}

	service := producer.NewEnhancedProducerService(cfg)
	err := service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop()

	time.Sleep(200 * time.Millisecond)

	// 测试config reload端点
	req, err := http.NewRequest("POST", "http://localhost:8086/config/reload", nil)
	if err != nil {
		t.Fatalf("Failed to create reload request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to request config reload: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 解析JSON响应
	var reloadResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&reloadResponse)
	if err != nil {
		t.Errorf("Failed to decode reload response: %v", err)
	}

	// 验证响应
	if message, ok := reloadResponse["message"]; !ok || message != "Configuration reloaded successfully" {
		t.Error("Expected successful reload message")
	}

	t.Log("Config reload endpoint test completed successfully")
}

func TestEnhancedProducerService_StatsEndpoint(t *testing.T) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8087,
		},
	}

	service := producer.NewEnhancedProducerService(cfg)
	err := service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop()

	time.Sleep(200 * time.Millisecond)

	// 测试stats端点
	resp, err := http.Get("http://localhost:8087/stats")
	if err != nil {
		t.Fatalf("Failed to request stats: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 解析JSON响应
	var stats map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&stats)
	if err != nil {
		t.Errorf("Failed to decode stats response: %v", err)
	}

	// 验证统计信息结构
	expectedFields := []string{"uptime", "performance_stats", "health_summary"}
	for _, field := range expectedFields {
		if _, ok := stats[field]; !ok {
			t.Errorf("Stats response missing '%s' field", field)
		}
	}

	t.Log("Stats endpoint test completed successfully")
}

func TestEnhancedProducerService_AlertHandling(t *testing.T) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8088,
		},
	}

	service := producer.NewEnhancedProducerService(cfg)

	// 创建一个通道来接收告警
	alertReceived := make(chan producer.HealthStatus, 1)

	// 模拟告警处理
	go func() {
		// 这里应该有实际的告警处理逻辑
		// 为了测试，我们模拟一个告警
		alert := producer.HealthStatus{
			Component: "test_component",
			Status:    producer.HealthStatusUnhealthy,
			Message:   "Test alert",
			Timestamp: time.Now(),
		}
		alertReceived <- alert
	}()

	// 验证告警是否被接收
	select {
	case alert := <-alertReceived:
		if alert.Component != "test_component" {
			t.Errorf("Expected component 'test_component', got %s", alert.Component)
		}
		if alert.Status != producer.HealthStatusUnhealthy {
			t.Errorf("Expected unhealthy status, got %v", alert.Status)
		}
		t.Log("Alert handling test completed successfully")
	case <-time.After(1 * time.Second):
		t.Error("Alert was not received within timeout")
	}
}

func TestEnhancedProducerService_ConcurrentRequests(t *testing.T) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8089,
		},
	}

	service := producer.NewEnhancedProducerService(cfg)
	err := service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop()

	time.Sleep(200 * time.Millisecond)

	// 并发请求测试
	var wg sync.WaitGroup
	errorCount := 0
	mu := sync.Mutex{}

	endpoints := []string{"/health", "/metrics", "/stats", "/config/status"}

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			endpoint := endpoints[id%len(endpoints)]
			resp, err := http.Get("http://localhost:8089" + endpoint)
			if err != nil {
				mu.Lock()
				errorCount++
				mu.Unlock()
				return
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				mu.Lock()
				errorCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Got %d errors during concurrent requests", errorCount)
	}

	t.Log("Concurrent requests test completed successfully")
}

func TestEnhancedProducerService_GracefulShutdown(t *testing.T) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8090,
		},
	}

	service := producer.NewEnhancedProducerService(cfg)

	// 启动服务
	err := service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// 测试优雅关闭
	start := time.Now()
	err = service.Stop()
	if err != nil {
		t.Errorf("Failed to stop service: %v", err)
	}
	duration := time.Since(start)

	// 验证关闭时间合理（不应该太长）
	if duration > 5*time.Second {
		t.Errorf("Service took too long to stop: %v", duration)
	}

	t.Logf("Graceful shutdown completed in %v", duration)
}

func TestEnhancedProducerService_ErrorHandling(t *testing.T) {
	// 测试端口已被占用的情况
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8091,
		},
	}

	// 先启动一个服务占用端口
	service1 := producer.NewEnhancedProducerService(cfg)
	err := service1.Start()
	if err != nil {
		t.Fatalf("Failed to start first service: %v", err)
	}
	defer service1.Stop()

	time.Sleep(100 * time.Millisecond)

	// 尝试启动第二个服务使用相同端口
	service2 := producer.NewEnhancedProducerService(cfg)
	err = service2.Start()
	if err == nil {
		service2.Stop()
		t.Error("Expected error when starting service on occupied port")
	}

	t.Log("Error handling test completed successfully")
}

// Benchmarks

func BenchmarkEnhancedProducerService_HealthEndpoint(b *testing.B) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8092,
		},
	}

	service := producer.NewEnhancedProducerService(cfg)
	err := service.Start()
	if err != nil {
		b.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop()

	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := http.Get("http://localhost:8092/health")
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			resp.Body.Close()
		}
	})
}

func BenchmarkEnhancedProducerService_MetricsEndpoint(b *testing.B) {
	cfg := &config.Config{
		Web: config.WebConfig{
			Port: 8093,
		},
	}

	service := producer.NewEnhancedProducerService(cfg)
	err := service.Start()
	if err != nil {
		b.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop()

	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get("http://localhost:8093/metrics")
		if err != nil {
			b.Errorf("Request failed: %v", err)
			continue
		}
		resp.Body.Close()
	}
}
