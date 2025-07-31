package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simplied-iot-monitoring-go/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	
	assert.NotNil(t, cfg)
	assert.Equal(t, "industrial-iot-monitor", cfg.App.Name)
	assert.Equal(t, "1.0.0", cfg.App.Version)
	assert.Equal(t, "development", cfg.App.Environment)
	assert.True(t, cfg.App.Debug)
	assert.Equal(t, "info", cfg.App.LogLevel)
}

func TestConfigManager_Load(t *testing.T) {
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
    device_data: "test-device-data"
    alerts: "test-alerts"
  producer:
    batch_size: 1024
    linger_ms: 5
    compression_type: "snappy"
    retries: 3
    timeout: "30s"
  consumer:
    group_id: "test-group"
    auto_offset_reset: "earliest"
    enable_auto_commit: true
    session_timeout: "30s"
    max_poll_records: 500
  timeout: "30s"

redis:
  host: "localhost"
  port: 6379
  database: 1
  pool_size: 5
  max_retries: 3
  timeout: "5s"

database:
  host: "localhost"
  port: 5432
  username: "test_user"
  password: "test_pass"
  database: "test_db"
  ssl_mode: "disable"
  max_open_conns: 10
  max_idle_conns: 2
  conn_max_lifetime: "1h"

producer:
  device_count: 100
  send_interval: "5s"
  data_variance: 0.1
  batch_size: 50
  retry_attempts: 3
  timeout: "10s"

consumer:
  worker_count: 2
  buffer_size: 100
  batch_size: 25
  processing_timeout: "30s"
  retry_attempts: 3
  dead_letter_queue: true

websocket:
  host: "localhost"
  port: 8081
  path: "/ws"
  max_connections: 100
  read_buffer_size: 1024
  write_buffer_size: 1024
  heartbeat_interval: "30s"
  connection_timeout: "60s"

web:
  host: "localhost"
  port: 8080
  template_path: "web/templates"
  static_path: "web/static"
  read_timeout: "10s"
  write_timeout: "10s"
  max_header_bytes: 1048576

device:
  simulator:
    enabled: true
    device_count: 100
    send_interval: "5s"
    data_variance: 0.1
  thresholds:
    temperature:
      min: -20.0
      max: 80.0
      warning: 70.0
      critical: 75.0
    humidity:
      min: 0.0
      max: 100.0
      warning: 85.0
      critical: 90.0
    battery:
      min: 0.0
      max: 100.0
      warning: 20.0
      critical: 10.0

alert:
  enabled: true
  rules:
    - name: "high_temperature"
      condition: "temperature > 75"
      severity: "critical"
      cooldown: "5m"
  notifications:
    email:
      enabled: false
      smtp_host: "smtp.example.com"
      smtp_port: 587
    webhook:
      enabled: true
      url: "http://localhost:8080/webhook/alerts"

monitoring:
  prometheus:
    enabled: true
    host: "localhost"
    port: 9090
    path: "/metrics"
    scrape_interval: "15s"
  logging:
    level: "info"
    format: "json"
    output: ["stdout"]
    file_path: "logs/app.log"
    max_size: "100MB"
    max_age: "7d"
    max_backups: 10

security:
  encryption:
    enabled: false
    algorithm: "AES-256"
    key_file: "configs/encryption.key"
  auth:
    enabled: false
    jwt_secret: ""
    token_expiry: "24h"
  rate_limiting:
    enabled: false
    requests_per_minute: 1000
    burst_size: 100
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// 测试配置加载
	manager := config.NewConfigManager()
	err = manager.Load(configFile, config.Development)
	
	require.NoError(t, err)
	cfg := manager.GetConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, "test-app", cfg.App.Name)
	assert.Equal(t, "1.0.0", cfg.App.Version)
	assert.Equal(t, "testing", cfg.App.Environment)
	assert.True(t, cfg.App.Debug)
	assert.Equal(t, "debug", cfg.App.LogLevel)
	
	assert.Equal(t, []string{"localhost:9092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "test-device-data", cfg.Kafka.Topics.DeviceData)
	assert.Equal(t, "test-alerts", cfg.Kafka.Topics.Alerts)
	
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.Equal(t, 1, cfg.Redis.Database)
	
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "test_user", cfg.DB.Username)
	assert.Equal(t, "test_pass", cfg.DB.Password)
	assert.Equal(t, "test_db", cfg.DB.Database)
}

func TestConfigManager_LoadWithEnvironmentVariables(t *testing.T) {
	// 设置环境变量
	os.Setenv("APP_NAME", "env-test-app")
	os.Setenv("APP_LOG_LEVEL", "warn")
	os.Setenv("KAFKA_BROKERS", "kafka1:9092,kafka2:9092")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5433")
	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("APP_LOG_LEVEL")
		os.Unsetenv("KAFKA_BROKERS")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
	}()

	manager := config.NewConfigManager()
	cfg, err := manager.LoadDefault()
	
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	
	// 验证环境变量覆盖了默认值
	assert.Equal(t, "env-test-app", cfg.App.Name)
	assert.Equal(t, "warn", cfg.App.LogLevel)
	assert.Equal(t, []string{"kafka1:9092", "kafka2:9092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "db.example.com", cfg.DB.Host)
	assert.Equal(t, 5433, cfg.DB.Port)
}

func TestConfigValidator(t *testing.T) {
	validator := config.NewConfigValidator()
	
	// 测试有效配置
	validConfig := config.DefaultConfig()
	err := validator.ValidateStruct(validConfig)
	assert.NoError(t, err)
	
	// 测试无效配置
	invalidConfig := &config.AppConfig{
		App: config.AppSection{
			Name:        "", // 空名称应该失败
			Version:     "invalid-version",
			Environment: "invalid",
			LogLevel:    "invalid-level",
		},
		WebSocket: config.WSSection{
			Port: -1, // 无效端口
		},
		Web: config.WebSection{
			Port: 99999, // 无效端口
		},
	}
	
	err = validator.ValidateStruct(invalidConfig)
	assert.Error(t, err)
}



func TestErrorRecovery(t *testing.T) {
	recovery := config.NewErrorRecovery()
	
	// 测试可恢复错误
	validationErr := config.NewValidationError("app.log_level", "invalid log level")
	assert.True(t, recovery.IsRecoverable(validationErr))
	
	suggestion := recovery.GetRecoverySuggestion(validationErr)
	assert.NotEmpty(t, suggestion)
	assert.Contains(t, suggestion, "日志级别")
	
	// 测试不可恢复错误
	securityErr := config.NewConfigError(config.ErrorTypeSecurity, "encryption.key", "key not found", nil)
	assert.False(t, recovery.IsRecoverable(securityErr))
}

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
	assert.Contains(t, errorMsg, "2 个配置错误")
	assert.Contains(t, errorMsg, "field1")
	assert.Contains(t, errorMsg, "field2")
}
