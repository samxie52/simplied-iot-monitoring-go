package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simplied-iot-monitoring-go/internal/config"
)

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "industrial-iot-monitor", cfg.App.Name)
	assert.Equal(t, "1.0.0", cfg.App.Version)
	assert.Equal(t, "development", cfg.App.Environment)
	assert.True(t, cfg.App.Debug)
	assert.Equal(t, "info", cfg.App.LogLevel)

	// 验证Kafka配置
	assert.NotEmpty(t, cfg.Kafka.Brokers)
	assert.NotEmpty(t, cfg.Kafka.Topics.DeviceData)
	assert.NotEmpty(t, cfg.Kafka.Topics.Alerts)

	// 验证数据库配置
	assert.NotEmpty(t, cfg.DB.Host)
	assert.Greater(t, cfg.DB.Port, 0)
	assert.NotEmpty(t, cfg.DB.Database)

	// 验证Redis配置
	assert.NotEmpty(t, cfg.Redis.Host)
	assert.Greater(t, cfg.Redis.Port, 0)
}

// TestEnvironmentTypes 测试环境类型
func TestEnvironmentTypes(t *testing.T) {
	tests := []struct {
		name string
		env  config.Environment
	}{
		{"Development", config.Development},
		{"Testing", config.Testing},
		{"Staging", config.Staging},
		{"Production", config.Production},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, string(tt.env))
			assert.True(t, tt.env.IsValid())
		})
	}
}

// TestConfigManager 测试配置管理器基本功能
func TestConfigManager(t *testing.T) {
	manager := config.NewConfigManager()
	assert.NotNil(t, manager)

	// 测试获取配置
	cfg := manager.GetConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, "industrial-iot-monitor", cfg.App.Name)
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	manager := config.NewConfigManager()

	// 测试有效配置
	validConfig := config.DefaultConfig()
	err := manager.Validate()
	assert.NoError(t, err)

	// 验证配置结构完整性
	assert.NotNil(t, validConfig.App)
	assert.NotNil(t, validConfig.Kafka)
	assert.NotNil(t, validConfig.DB)
	assert.NotNil(t, validConfig.Redis)
	assert.NotNil(t, validConfig.Monitor)
}

// TestConfigFromFile 测试从文件加载配置
func TestConfigFromFile(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
app:
  name: "test-app"
  version: "1.0.0"
  environment: "testing"
  debug: true
  log_level: "debug"

kafka:
  brokers: ["localhost:9092"]
  topics:
    device_data: "device-data"
    alerts: "alerts"

redis:
  host: "localhost"
  port: 6379
  database: 1

db:
  host: "localhost"
  port: 5432
  username: "test_user"
  password: "test_pass"
  database: "test_db"

producer:
  device_count: 10
  send_interval: "5s"
  data_variance: 0.1
  batch_size: 100
  retry_attempts: 3
  timeout: "30s"

consumer:
  worker_count: 4
  buffer_size: 1000
  batch_size: 100
  processing_timeout: "30s"
  retry_attempts: 3
  dead_letter_queue: false

websocket:
  enabled: true
  host: "localhost"
  port: 8080
  path: "/ws"
  read_buffer_size: 1024
  write_buffer_size: 1024
  handshake_timeout: "10s"
  ping_period: "54s"
  pong_wait: "60s"
  write_wait: "10s"
  max_message_size: 512

web:
  enabled: true
  host: "localhost"
  port: 8081
  template_path: "templates"
  static_path: "static"
  read_timeout: "30s"
  write_timeout: "30s"
  max_header_bytes: 1048576

device:
  simulator:
    enabled: true
    device_count: 10
    send_interval: "5s"
    data_variance: 0.1
  thresholds:
    temperature:
      min: -10.0
      max: 50.0
      warning: 35.0
      critical: 45.0
    humidity:
      min: 0.0
      max: 100.0
      warning: 80.0
      critical: 90.0
    battery:
      min: 0.0
      max: 100.0
      warning: 20.0
      critical: 10.0

alert:
  enabled: true
  rules:
    - name: "temperature_high"
      condition: "temperature > 40"
      severity: "warning"
      cooldown: "5m"
  notifications:
    email:
      enabled: false
      smtp_host: "smtp.example.com"
      smtp_port: 587
    webhook:
      enabled: false
      url: "http://example.com/webhook"

monitor:
  logging:
    level: "info"
    file: "test.log"
  metrics:
    enabled: true
    port: 9090
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// 测试配置加载
	manager := config.NewConfigManager()
	err = manager.Load(configFile, config.Testing)
	require.NoError(t, err)

	cfg := manager.GetConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, "test-app", cfg.App.Name)
	assert.Equal(t, "1.0.0", cfg.App.Version)
	assert.Equal(t, "testing", cfg.App.Environment)
	assert.True(t, cfg.App.Debug)
	assert.Equal(t, "info", cfg.App.LogLevel)

	assert.Equal(t, []string{"localhost:9092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "device-data", cfg.Kafka.Topics.DeviceData)
	assert.Equal(t, "alerts", cfg.Kafka.Topics.Alerts)

	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.Equal(t, 1, cfg.Redis.Database)

	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "test_user", cfg.DB.Username)
	assert.Equal(t, "test_pass", cfg.DB.Password)
	assert.Equal(t, "test_db", cfg.DB.Database)
}

// TestEnvironmentValidation 测试环境验证
func TestEnvironmentValidation(t *testing.T) {
	tests := []struct {
		name string
		env  config.Environment
	}{
		{"Development", config.Development},
		{"Testing", config.Testing},
		{"Staging", config.Staging},
		{"Production", config.Production},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.env.IsValid())
			assert.NotEmpty(t, tt.env.String())
		})
	}
}

// TestConfigWatcher 测试配置监听器
func TestConfigWatcher(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "watch_config.yaml")

	initialContent := `
app:
  name: "watch-test"
  log_level: "info"
`

	err := os.WriteFile(configFile, []byte(initialContent), 0644)
	require.NoError(t, err)

	// 创建配置管理器
	manager := config.NewConfigManager()
	err = manager.Load(configFile, config.Development)
	require.NoError(t, err)

	callback := func(updatedConfig *config.AppConfig) error {
		// 回调函数被调用时的处理逻辑
		return nil
	}

	watcher := config.NewConfigWatcher(manager.GetViper(), callback)

	// 开始监听
	err = watcher.Start()
	assert.NoError(t, err)
	assert.True(t, watcher.IsRunning())

	// 停止监听
	err = watcher.Stop()
	assert.NoError(t, err)
	assert.False(t, watcher.IsRunning())
}

// TestErrorRecovery 测试错误恢复
func TestErrorRecovery(t *testing.T) {
	recovery := config.NewErrorRecovery()

	// 测试可恢复错误
	validationErr := config.NewValidationError("app.log_level", "invalid log level")
	assert.True(t, recovery.IsRecoverable(validationErr))

	suggestion := recovery.GetRecoverySuggestion(validationErr)
	assert.NotEmpty(t, suggestion)

	// 测试错误恢复
	err := recovery.Recover(validationErr)
	assert.NoError(t, err) // 验证错误应该可以恢复
}

// TestConfigErrorCollection 测试配置错误收集
func TestConfigErrorCollection(t *testing.T) {
	collection := config.NewConfigErrorCollection()

	assert.False(t, collection.HasErrors())
	assert.Equal(t, 0, collection.Count())

	// 添加错误
	err1 := config.NewValidationError("field1", "error1")
	err2 := config.NewValidationError("field2", "error2")

	collection.Add(err1)
	collection.Add(err2)

	assert.True(t, collection.HasErrors())
	assert.Equal(t, 2, collection.Count())

	errors := collection.Errors()
	assert.Len(t, errors, 2)

	// 测试错误消息
	errorMsg := collection.Error()
	assert.Contains(t, errorMsg, "2")
	assert.Contains(t, errorMsg, "field1")
	assert.Contains(t, errorMsg, "field2")
}

// TestConfigLoadOptions 测试配置加载选项
func TestConfigLoadOptions(t *testing.T) {
	options := config.DefaultConfigLoadOptions()
	assert.NotNil(t, options)
	assert.NotEmpty(t, options.ConfigPath)
	assert.Equal(t, config.Development, options.Environment)
	assert.NotEmpty(t, options.EnvPrefix)
	assert.NotEmpty(t, options.Sources)
}

// TestValidationError 测试验证错误
func TestValidationError(t *testing.T) {
	err := config.NewValidationError("test.field", "test error message")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test error message")

	// 测试ConfigError字段
	configErr := err
	assert.Equal(t, "test.field", configErr.Field)
	assert.Equal(t, "test error message", configErr.Message)
	assert.Equal(t, config.ErrorTypeValidation, configErr.Type)
}
