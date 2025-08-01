package producer

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/producer"
)

// MockConfigChangeHandler 用于测试的配置变更处理器
type MockConfigChangeHandler struct {
	name         string
	handleCount  int
	lastConfig   interface{}
	shouldError  bool
	mu           sync.RWMutex
}

func NewMockConfigChangeHandler(name string) *MockConfigChangeHandler {
	return &MockConfigChangeHandler{
		name: name,
	}
}

func (m *MockConfigChangeHandler) Name() string {
	return m.name
}

func (m *MockConfigChangeHandler) HandleConfigChange(newConfig interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.handleCount++
	m.lastConfig = newConfig

	if m.shouldError {
		return &config.ConfigError{
			Type:    config.ErrorTypeValidation,
			Field:   "test",
			Message: "Mock error",
		}
	}

	return nil
}

func (m *MockConfigChangeHandler) GetHandleCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.handleCount
}

func (m *MockConfigChangeHandler) GetLastConfig() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastConfig
}

func (m *MockConfigChangeHandler) SetShouldError(shouldError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
}

func TestConfigWatcher_Creation(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")

	// 创建测试配置文件
	configContent := `
producer:
  batch_size: 100
  timeout: 5s
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	watcher := producer.NewConfigWatcher(configFile, 100*time.Millisecond)
	if watcher == nil {
		t.Fatal("Failed to create config watcher")
	}

	t.Log("Config watcher created successfully")
}

func TestConfigWatcher_AddHandler(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")

	configContent := `
producer:
  batch_size: 100
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	watcher := producer.NewConfigWatcher(configFile, 100*time.Millisecond)
	handler := NewMockConfigChangeHandler("test_handler")

	watcher.AddHandler(handler)

	t.Log("Handler added successfully")
}

func TestConfigWatcher_FileChange(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")

	// 创建初始配置文件
	initialContent := `
producer:
  batch_size: 100
  timeout: 5s
`
	err := os.WriteFile(configFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	watcher := producer.NewConfigWatcher(configFile, 50*time.Millisecond)
	handler := NewMockConfigChangeHandler("test_handler")
	watcher.AddHandler(handler)

	// 启动监控
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go watcher.Start(ctx)

	// 等待监控启动
	time.Sleep(100 * time.Millisecond)

	// 修改配置文件
	updatedContent := `
producer:
  batch_size: 200
  timeout: 10s
`
	err = os.WriteFile(configFile, []byte(updatedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// 等待文件变更检测和处理
	time.Sleep(200 * time.Millisecond)

	// 验证处理器是否被调用
	handleCount := handler.GetHandleCount()
	if handleCount == 0 {
		t.Error("Expected handler to be called, but it wasn't")
	}

	t.Logf("Handler was called %d times", handleCount)
}

func TestConfigWatcher_MultipleHandlers(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")

	configContent := `
producer:
  batch_size: 100
device:
  count: 50
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	watcher := producer.NewConfigWatcher(configFile, 50*time.Millisecond)

	handler1 := NewMockConfigChangeHandler("handler1")
	handler2 := NewMockConfigChangeHandler("handler2")
	handler3 := NewMockConfigChangeHandler("handler3")

	watcher.AddHandler(handler1)
	watcher.AddHandler(handler2)
	watcher.AddHandler(handler3)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go watcher.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// 修改配置文件
	updatedContent := `
producer:
  batch_size: 300
device:
  count: 75
`
	err = os.WriteFile(configFile, []byte(updatedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 验证所有处理器都被调用
	handlers := []*MockConfigChangeHandler{handler1, handler2, handler3}
	for i, handler := range handlers {
		count := handler.GetHandleCount()
		if count == 0 {
			t.Errorf("Handler %d was not called", i+1)
		}
	}

	t.Log("All handlers were called successfully")
}

func TestConfigWatcher_HandlerError(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")

	configContent := `
producer:
  batch_size: 100
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	watcher := producer.NewConfigWatcher(configFile, 50*time.Millisecond)

	goodHandler := NewMockConfigChangeHandler("good_handler")
	errorHandler := NewMockConfigChangeHandler("error_handler")
	errorHandler.SetShouldError(true)

	watcher.AddHandler(goodHandler)
	watcher.AddHandler(errorHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go watcher.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// 修改配置文件
	updatedContent := `
producer:
  batch_size: 200
`
	err = os.WriteFile(configFile, []byte(updatedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 验证好的处理器仍然被调用，即使有错误的处理器
	goodCount := goodHandler.GetHandleCount()
	errorCount := errorHandler.GetHandleCount()

	if goodCount == 0 {
		t.Error("Good handler should have been called")
	}

	if errorCount == 0 {
		t.Error("Error handler should have been called (even though it errors)")
	}

	t.Log("Handler error test completed successfully")
}

func TestConfigWatcher_DebounceLogic(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")

	configContent := `
producer:
  batch_size: 100
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// 使用较长的防抖时间
	watcher := producer.NewConfigWatcher(configFile, 200*time.Millisecond)
	handler := NewMockConfigChangeHandler("debounce_handler")
	watcher.AddHandler(handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go watcher.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// 快速连续修改文件多次
	for i := 0; i < 5; i++ {
		content := `
producer:
  batch_size: ` + string(rune('1'+i)) + `00
`
		err = os.WriteFile(configFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to update config file: %v", err)
		}
		time.Sleep(20 * time.Millisecond) // 短于防抖时间
	}

	// 等待防抖时间过去
	time.Sleep(300 * time.Millisecond)

	// 由于防抖，处理器应该只被调用一次（或很少次数）
	handleCount := handler.GetHandleCount()
	if handleCount > 2 {
		t.Errorf("Expected debounce to limit calls, but got %d calls", handleCount)
	}

	t.Logf("Debounce test completed with %d handler calls", handleCount)
}

func TestConfigWatcher_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")

	configContent := `
producer:
  batch_size: 100
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	watcher := producer.NewConfigWatcher(configFile, 50*time.Millisecond)
	handler := NewMockConfigChangeHandler("cancel_handler")
	watcher.AddHandler(handler)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		watcher.Start(ctx)
		done <- true
	}()

	// 等待监控启动
	time.Sleep(100 * time.Millisecond)

	// 取消上下文
	cancel()

	// 验证监控是否正确停止
	select {
	case <-done:
		t.Log("Config watcher stopped correctly on context cancellation")
	case <-time.After(1 * time.Second):
		t.Error("Config watcher did not stop within timeout")
	}
}

func TestKafkaProducerConfigHandler(t *testing.T) {
	// 创建模拟的Kafka生产者
	mockProducer := &MockKafkaProducerForConfig{}
	handler := producer.NewKafkaProducerConfigHandler(mockProducer)

	if handler.Name() != "kafka_producer" {
		t.Errorf("Expected name 'kafka_producer', got %s", handler.Name())
	}

	// 创建测试配置
	cfg := &config.Config{
		Producer: config.ProducerConfig{
			BatchSize: 200,
			Timeout:   "10s",
		},
	}

	err := handler.HandleConfigChange(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !mockProducer.reloadCalled {
		t.Error("Expected ReloadConfig to be called")
	}

	t.Log("Kafka producer config handler test passed")
}

func TestDeviceSimulatorConfigHandler(t *testing.T) {
	mockSimulator := &MockDeviceSimulator{}
	handler := producer.NewDeviceSimulatorConfigHandler(mockSimulator)

	if handler.Name() != "device_simulator" {
		t.Errorf("Expected name 'device_simulator', got %s", handler.Name())
	}

	cfg := &config.Config{
		Device: config.DeviceConfig{
			Count:    100,
			Interval: "2s",
		},
	}

	err := handler.HandleConfigChange(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !mockSimulator.reloadCalled {
		t.Error("Expected ReloadConfig to be called")
	}

	t.Log("Device simulator config handler test passed")
}

func TestConfigReloader_Creation(t *testing.T) {
	reloader := producer.NewConfigReloader()
	if reloader == nil {
		t.Fatal("Failed to create config reloader")
	}

	t.Log("Config reloader created successfully")
}

func TestConfigReloader_TriggerReload(t *testing.T) {
	reloader := producer.NewConfigReloader()

	// 触发重载
	err := reloader.TriggerReload()
	if err != nil {
		t.Errorf("Unexpected error triggering reload: %v", err)
	}

	// 获取状态
	status := reloader.GetStatus()
	if status.LastReloadTime.IsZero() {
		t.Error("Expected last reload time to be set")
	}

	if status.ReloadCount != 1 {
		t.Errorf("Expected reload count 1, got %d", status.ReloadCount)
	}

	t.Log("Config reload triggered successfully")
}

func TestConfigReloader_MultipleReloads(t *testing.T) {
	reloader := producer.NewConfigReloader()

	// 触发多次重载
	for i := 0; i < 5; i++ {
		err := reloader.TriggerReload()
		if err != nil {
			t.Errorf("Unexpected error on reload %d: %v", i+1, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	status := reloader.GetStatus()
	if status.ReloadCount != 5 {
		t.Errorf("Expected reload count 5, got %d", status.ReloadCount)
	}

	t.Log("Multiple reloads completed successfully")
}

func TestConfigReloader_ConcurrentReloads(t *testing.T) {
	reloader := producer.NewConfigReloader()

	var wg sync.WaitGroup
	errorCount := 0
	mu := sync.Mutex{}

	// 并发触发重载
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := reloader.TriggerReload()
			if err != nil {
				mu.Lock()
				errorCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Got %d errors during concurrent reloads", errorCount)
	}

	status := reloader.GetStatus()
	if status.ReloadCount != 10 {
		t.Errorf("Expected reload count 10, got %d", status.ReloadCount)
	}

	t.Log("Concurrent reloads completed successfully")
}

// Mock implementations for testing

type MockKafkaProducerForConfig struct {
	reloadCalled bool
}

func (m *MockKafkaProducerForConfig) ReloadConfig(cfg *config.Config) error {
	m.reloadCalled = true
	return nil
}

type MockDeviceSimulator struct {
	reloadCalled bool
}

func (m *MockDeviceSimulator) ReloadConfig(cfg *config.Config) error {
	m.reloadCalled = true
	return nil
}

// Benchmarks

func BenchmarkConfigWatcher_HandleChange(b *testing.B) {
	tempDir := b.TempDir()
	configFile := filepath.Join(tempDir, "bench_config.yaml")

	configContent := `
producer:
  batch_size: 100
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create test config file: %v", err)
	}

	watcher := producer.NewConfigWatcher(configFile, 10*time.Millisecond)
	handler := NewMockConfigChangeHandler("bench_handler")
	watcher.AddHandler(handler)

	cfg := &config.Config{
		Producer: config.ProducerConfig{
			BatchSize: 100,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.HandleConfigChange(cfg)
	}
}

func BenchmarkConfigReloader_TriggerReload(b *testing.B) {
	reloader := producer.NewConfigReloader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reloader.TriggerReload()
	}
}
