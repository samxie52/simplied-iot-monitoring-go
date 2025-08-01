package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ConfigManager 配置管理器接口
type ConfigManager interface {
	// 配置加载
	Load(configPath string, env Environment) error
	LoadDefault() (*AppConfig, error)
	LoadFromEnv() error
	LoadFromCLI(args []string) error

	// 配置访问
	Get(key string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetDuration(key string) time.Duration

	// 配置验证
	Validate() error
	ValidateSection(section string) error

	// 配置监听
	Watch(callback ConfigWatchCallback) error
	StopWatch() error

	// 配置安全
	Encrypt(key string) error
	Decrypt(key string) error

	// 获取配置
	GetConfig() *AppConfig
	GetViper() *viper.Viper
}

// viperConfigManager Viper配置管理器实现
type viperConfigManager struct {
	viper     *viper.Viper
	config    *AppConfig
	env       Environment
	validator ConfigValidator

	watcher  *ConfigWatcher
	metadata *ConfigMetadata
}

// NewConfigManager 创建新的配置管理器
func NewConfigManager() ConfigManager {
	v := viper.New()

	return &viperConfigManager{
		viper:     v,
		config:    &AppConfig{}, // 初始化为空配置，完全从 YAML 加载
		env:       Development,
		validator: NewConfigValidator(),

		metadata: &ConfigMetadata{},
	}
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper) {
	// 应用默认值
	v.SetDefault("app.name", "industrial-iot-monitor")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.environment", "development")
	v.SetDefault("app.debug", true)
	v.SetDefault("app.log_level", "info")

	// Kafka默认值
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.topics.device_data", "device-data")
	v.SetDefault("kafka.topics.alerts", "alerts")
	v.SetDefault("kafka.producer.batch_size", 16384)
	v.SetDefault("kafka.producer.linger_ms", 10)
	v.SetDefault("kafka.producer.compression_type", "snappy")
	v.SetDefault("kafka.producer.retries", 3)
	v.SetDefault("kafka.producer.timeout", "30s")
	v.SetDefault("kafka.consumer.group_id", "iot-consumer-group")
	v.SetDefault("kafka.consumer.auto_offset_reset", "earliest")
	v.SetDefault("kafka.consumer.enable_auto_commit", true)
	v.SetDefault("kafka.consumer.session_timeout", "30s")
	v.SetDefault("kafka.consumer.max_poll_records", 500)
	v.SetDefault("kafka.timeout", "30s")

	// Redis默认值
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.database", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.timeout", "5s")

	// 数据库默认值
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.username", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.database", "iot_monitor")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "1h")

	// WebSocket默认值
	v.SetDefault("websocket.host", "0.0.0.0")
	v.SetDefault("websocket.port", 8081)
	v.SetDefault("websocket.path", "/ws")
	v.SetDefault("websocket.max_connections", 1000)
	v.SetDefault("websocket.read_buffer_size", 1024)
	v.SetDefault("websocket.write_buffer_size", 1024)
	v.SetDefault("websocket.heartbeat_interval", "30s")
	v.SetDefault("websocket.connection_timeout", "60s")

	// Web服务默认值
	v.SetDefault("web.host", "0.0.0.0")
	v.SetDefault("web.port", 8080)
	v.SetDefault("web.template_path", "web/templates")
	v.SetDefault("web.static_path", "web/static")
	v.SetDefault("web.read_timeout", "10s")
	v.SetDefault("web.write_timeout", "10s")
	v.SetDefault("web.max_header_bytes", 1048576)

	// 监控默认值
	v.SetDefault("monitoring.prometheus.enabled", true)
	v.SetDefault("monitoring.prometheus.host", "0.0.0.0")
	v.SetDefault("monitoring.prometheus.port", 9090)
	v.SetDefault("monitoring.prometheus.path", "/metrics")
	v.SetDefault("monitoring.prometheus.scrape_interval", "15s")
	v.SetDefault("monitoring.logging.level", "info")
	v.SetDefault("monitoring.logging.format", "json")
	v.SetDefault("monitoring.logging.output", []string{"stdout", "file"})
	v.SetDefault("monitoring.logging.file_path", "logs/app.log")
	v.SetDefault("monitoring.logging.max_size", "100MB")
	v.SetDefault("monitoring.logging.max_age", "7d")
	v.SetDefault("monitoring.logging.max_backups", 10)
}

// Load 加载配置文件
func (cm *viperConfigManager) Load(configPath string, env Environment) error {
	cm.env = env

	// 检查是否是完整的文件路径
	if strings.HasSuffix(configPath, ".yaml") || strings.HasSuffix(configPath, ".yml") {
		// 完整的文件路径
		cm.viper.SetConfigFile(configPath)
		if err := cm.viper.ReadInConfig(); err != nil {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}
	} else {
		// 目录路径
		if configPath == "" {
			configPath = "config"
		}

		cm.viper.SetConfigName("config")
		cm.viper.SetConfigType("yaml")
		cm.viper.AddConfigPath(configPath)
		cm.viper.AddConfigPath(".")
		cm.viper.AddConfigPath("./config")
		cm.viper.AddConfigPath("./configs")

		// 读取基础配置文件
		if err := cm.viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// 配置文件不存在，返回错误
				return fmt.Errorf("配置文件不存在: %s", configPath)
			} else {
				return fmt.Errorf("读取配置文件失败: %w", err)
			}
		}

		// 加载环境特定配置
		if err := cm.loadEnvironmentConfig(configPath, env); err != nil {
			return fmt.Errorf("加载环境配置失败: %w", err)
		}
	}

	// 绑定环境变量
	if err := cm.LoadFromEnv(); err != nil {
		return fmt.Errorf("绑定环境变量失败: %w", err)
	}

	// 由于Unmarshal有问题，使用直接读取viper值的方式
	cm.config = cm.buildConfigFromViper()

	// 验证配置
	if err := cm.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 记录审计日志

	// 更新元数据
	cm.metadata.LoadTime = time.Now()
	cm.metadata.Environment = env
	cm.metadata.Source = SourceFile

	return nil
}

// buildConfigFromViper 直接从Viper构建配置，绕过Unmarshal问题
func (cm *viperConfigManager) buildConfigFromViper() *AppConfig {
	// 初始化空配置，完全从 YAML 加载
	config := &AppConfig{}

	// 应用配置
	config.App.Name = cm.viper.GetString("app.name")
	config.App.Version = cm.viper.GetString("app.version")
	config.App.Environment = cm.viper.GetString("app.environment")
	config.App.Debug = cm.viper.GetBool("app.debug")
	config.App.LogLevel = cm.viper.GetString("app.log_level")

	// Kafka配置
	config.Kafka.Brokers = cm.viper.GetStringSlice("kafka.brokers")
	config.Kafka.Topics.DeviceData = cm.viper.GetString("kafka.topics.device_data")
	config.Kafka.Topics.Alerts = cm.viper.GetString("kafka.topics.alerts")
	config.Kafka.Producer.ClientID = cm.viper.GetString("kafka.producer.client_id")
	config.Kafka.Producer.BatchSize = cm.viper.GetInt("kafka.producer.batch_size")
	config.Kafka.Producer.BatchTimeout = cm.viper.GetDuration("kafka.producer.batch_timeout")
	config.Kafka.Producer.CompressionType = cm.viper.GetString("kafka.producer.compression_type")
	config.Kafka.Producer.MaxRetries = cm.viper.GetInt("kafka.producer.max_retries")
	config.Kafka.Producer.RetryBackoff = cm.viper.GetDuration("kafka.producer.retry_backoff")
	config.Kafka.Producer.RequiredAcks = cm.viper.GetInt("kafka.producer.required_acks")
	config.Kafka.Producer.FlushFrequency = cm.viper.GetDuration("kafka.producer.flush_frequency")
	config.Kafka.Producer.ChannelBufferSize = cm.viper.GetInt("kafka.producer.channel_buffer_size")
	config.Kafka.Producer.Timeout = cm.viper.GetDuration("kafka.producer.timeout")
	config.Kafka.Consumer.GroupID = cm.viper.GetString("kafka.consumer.group_id")
	config.Kafka.Consumer.AutoOffsetReset = cm.viper.GetString("kafka.consumer.auto_offset_reset")
	config.Kafka.Consumer.EnableAutoCommit = cm.viper.GetBool("kafka.consumer.enable_auto_commit")
	config.Kafka.Consumer.SessionTimeout = cm.viper.GetDuration("kafka.consumer.session_timeout")
	config.Kafka.Consumer.MaxPollRecords = cm.viper.GetInt("kafka.consumer.max_poll_records")
	config.Kafka.Security.Protocol = cm.viper.GetString("kafka.security.protocol")
	config.Kafka.Security.Username = cm.viper.GetString("kafka.security.username")
	config.Kafka.Security.Password = cm.viper.GetString("kafka.security.password")
	config.Kafka.Timeout = cm.viper.GetDuration("kafka.timeout")

	// Redis配置
	config.Redis.Host = cm.viper.GetString("redis.host")
	config.Redis.Port = cm.viper.GetInt("redis.port")
	config.Redis.Password = cm.viper.GetString("redis.password")
	config.Redis.Database = cm.viper.GetInt("redis.database")
	config.Redis.PoolSize = cm.viper.GetInt("redis.pool_size")
	config.Redis.MaxRetries = cm.viper.GetInt("redis.max_retries")
	config.Redis.Timeout = cm.viper.GetDuration("redis.timeout")

	// 数据库配置
	config.DB.Host = cm.viper.GetString("database.host")
	config.DB.Port = cm.viper.GetInt("database.port")
	config.DB.Username = cm.viper.GetString("database.username")
	config.DB.Password = cm.viper.GetString("database.password")
	config.DB.Database = cm.viper.GetString("database.database")
	config.DB.SSLMode = cm.viper.GetString("database.ssl_mode")
	config.DB.MaxOpenConns = cm.viper.GetInt("database.max_open_conns")
	config.DB.MaxIdleConns = cm.viper.GetInt("database.max_idle_conns")
	config.DB.ConnMaxLifetime = cm.viper.GetDuration("database.conn_max_lifetime")

	// 生产者配置
	config.Producer.DeviceCount = cm.viper.GetInt("producer.device_count")
	config.Producer.SendInterval = cm.viper.GetDuration("producer.send_interval")
	config.Producer.DataVariance = cm.viper.GetFloat64("producer.data_variance")
	config.Producer.BatchSize = cm.viper.GetInt("producer.batch_size")
	config.Producer.RetryAttempts = cm.viper.GetInt("producer.retry_attempts")
	config.Producer.Timeout = cm.viper.GetDuration("producer.timeout")

	// 消费者配置
	config.Consumer.WorkerCount = cm.viper.GetInt("consumer.worker_count")
	config.Consumer.BufferSize = cm.viper.GetInt("consumer.buffer_size")
	config.Consumer.BatchSize = cm.viper.GetInt("consumer.batch_size")
	config.Consumer.ProcessingTimeout = cm.viper.GetDuration("consumer.processing_timeout")
	config.Consumer.RetryAttempts = cm.viper.GetInt("consumer.retry_attempts")
	config.Consumer.DeadLetterQueue = cm.viper.GetBool("consumer.dead_letter_queue")

	// WebSocket配置
	config.WebSocket.Host = cm.viper.GetString("websocket.host")
	config.WebSocket.Port = cm.viper.GetInt("websocket.port")
	config.WebSocket.Path = cm.viper.GetString("websocket.path")
	config.WebSocket.MaxConnections = cm.viper.GetInt("websocket.max_connections")
	config.WebSocket.ReadBufferSize = cm.viper.GetInt("websocket.read_buffer_size")
	config.WebSocket.WriteBufferSize = cm.viper.GetInt("websocket.write_buffer_size")
	config.WebSocket.HeartbeatInterval = cm.viper.GetDuration("websocket.heartbeat_interval")
	config.WebSocket.ConnectionTimeout = cm.viper.GetDuration("websocket.connection_timeout")

	// Web配置
	config.Web.Host = cm.viper.GetString("web.host")
	config.Web.Port = cm.viper.GetInt("web.port")
	config.Web.TemplatePath = cm.viper.GetString("web.template_path")
	config.Web.StaticPath = cm.viper.GetString("web.static_path")
	config.Web.ReadTimeout = cm.viper.GetDuration("web.read_timeout")
	config.Web.WriteTimeout = cm.viper.GetDuration("web.write_timeout")
	config.Web.MaxHeaderBytes = cm.viper.GetInt("web.max_header_bytes")

	// 设备配置
	config.Device.Simulator.Enabled = cm.viper.GetBool("device.simulator.enabled")
	config.Device.Simulator.DeviceCount = cm.viper.GetInt("device.simulator.device_count")
	config.Device.Simulator.SampleInterval = cm.viper.GetDuration("device.simulator.sample_interval")
	config.Device.Simulator.DataVariation = cm.viper.GetFloat64("device.simulator.data_variation")
	config.Device.Simulator.AnomalyRate = cm.viper.GetFloat64("device.simulator.anomaly_rate")
	config.Device.Simulator.TrendEnabled = cm.viper.GetBool("device.simulator.trend_enabled")
	config.Device.Simulator.TrendStrength = cm.viper.GetFloat64("device.simulator.trend_strength")
	config.Device.Simulator.WorkerPoolSize = cm.viper.GetInt("device.simulator.worker_pool_size")
	config.Device.Simulator.QueueBufferSize = cm.viper.GetInt("device.simulator.queue_buffer_size")

	// 设备阈值配置
	config.Device.Thresholds.Temperature.Min = cm.viper.GetFloat64("device.thresholds.temperature.min")
	config.Device.Thresholds.Temperature.Max = cm.viper.GetFloat64("device.thresholds.temperature.max")
	config.Device.Thresholds.Temperature.Warning = cm.viper.GetFloat64("device.thresholds.temperature.warning")
	config.Device.Thresholds.Temperature.Critical = cm.viper.GetFloat64("device.thresholds.temperature.critical")
	config.Device.Thresholds.Humidity.Min = cm.viper.GetFloat64("device.thresholds.humidity.min")
	config.Device.Thresholds.Humidity.Max = cm.viper.GetFloat64("device.thresholds.humidity.max")
	config.Device.Thresholds.Humidity.Warning = cm.viper.GetFloat64("device.thresholds.humidity.warning")
	config.Device.Thresholds.Humidity.Critical = cm.viper.GetFloat64("device.thresholds.humidity.critical")
	// 告警配置
	config.Alert.Enabled = cm.viper.GetBool("alert.enabled")
	config.Alert.Notifications.Email.Enabled = cm.viper.GetBool("alert.notifications.email.enabled")
	config.Alert.Notifications.Email.SMTPHost = cm.viper.GetString("alert.notifications.email.smtp_host")
	config.Alert.Notifications.Email.SMTPPort = cm.viper.GetInt("alert.notifications.email.smtp_port")
	config.Alert.Notifications.Webhook.Enabled = cm.viper.GetBool("alert.notifications.webhook.enabled")
	config.Alert.Notifications.Webhook.URL = cm.viper.GetString("alert.notifications.webhook.url")

	// 监控配置
	config.Monitor.Prometheus.Enabled = cm.viper.GetBool("monitoring.prometheus.enabled")
	config.Monitor.Prometheus.Host = cm.viper.GetString("monitoring.prometheus.host")
	config.Monitor.Prometheus.Port = cm.viper.GetInt("monitoring.prometheus.port")
	config.Monitor.Prometheus.Path = cm.viper.GetString("monitoring.prometheus.path")
	config.Monitor.Prometheus.ScrapeInterval = cm.viper.GetDuration("monitoring.prometheus.scrape_interval")
	config.Monitor.Logging.Level = cm.viper.GetString("monitoring.logging.level")
	config.Monitor.Logging.Format = cm.viper.GetString("monitoring.logging.format")
	config.Monitor.Logging.Output = cm.viper.GetStringSlice("monitoring.logging.output")
	config.Monitor.Logging.FilePath = cm.viper.GetString("monitoring.logging.file_path")
	config.Monitor.Logging.MaxSize = cm.viper.GetString("monitoring.logging.max_size")
	config.Monitor.Logging.MaxAge = cm.viper.GetString("monitoring.logging.max_age")
	config.Monitor.Logging.MaxBackups = cm.viper.GetInt("monitoring.logging.max_backups")

	// 电池阈值
	config.Device.Thresholds.Battery.Min = cm.viper.GetFloat64("device.thresholds.battery.min")
	config.Device.Thresholds.Battery.Max = cm.viper.GetFloat64("device.thresholds.battery.max")
	config.Device.Thresholds.Battery.Warning = cm.viper.GetFloat64("device.thresholds.battery.warning")
	config.Device.Thresholds.Battery.Critical = cm.viper.GetFloat64("device.thresholds.battery.critical")

	return config
}

// loadEnvironmentConfig 加载环境特定配置
func (cm *viperConfigManager) loadEnvironmentConfig(configPath string, env Environment) error {
	envConfigFile := filepath.Join(configPath, fmt.Sprintf("%s.yaml", env.String()))

	// 检查环境配置文件是否存在
	if _, err := os.Stat(envConfigFile); os.IsNotExist(err) {
		// 环境配置文件不存在，跳过
		return nil
	}

	// 创建新的viper实例来读取环境配置
	envViper := viper.New()
	envViper.SetConfigFile(envConfigFile)

	if err := envViper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取环境配置文件 %s 失败: %w", envConfigFile, err)
	}

	// 合并环境配置到主配置
	if err := cm.viper.MergeConfigMap(envViper.AllSettings()); err != nil {
		return fmt.Errorf("合并环境配置失败: %w", err)
	}

	return nil
}

// LoadFromEnv 从环境变量加载配置
func (cm *viperConfigManager) LoadFromEnv() error {
	cm.viper.SetEnvPrefix("IOT")
	cm.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	cm.viper.AutomaticEnv()

	// 绑定特定的环境变量
	envBindings := map[string]string{
		"app.environment":          "IOT_APP_ENVIRONMENT",
		"app.debug":                "IOT_APP_DEBUG",
		"app.log_level":            "IOT_APP_LOG_LEVEL",
		"database.host":            "IOT_DB_HOST",
		"database.port":            "IOT_DB_PORT",
		"database.username":        "IOT_DB_USERNAME",
		"database.password":        "IOT_DB_PASSWORD",
		"database.database":        "IOT_DB_DATABASE",
		"redis.host":               "IOT_REDIS_HOST",
		"redis.port":               "IOT_REDIS_PORT",
		"redis.password":           "IOT_REDIS_PASSWORD",
		"kafka.brokers":            "IOT_KAFKA_BROKERS",
		"kafka.security.username":  "IOT_KAFKA_USERNAME",
		"kafka.security.password":  "IOT_KAFKA_PASSWORD",
		"security.auth.jwt_secret": "IOT_JWT_SECRET",
	}

	for configKey, envKey := range envBindings {
		if err := cm.viper.BindEnv(configKey, envKey); err != nil {
			return fmt.Errorf("绑定环境变量 %s 失败: %w", envKey, err)
		}
	}

	return nil
}

// LoadFromCLI 从命令行参数加载配置
func (cm *viperConfigManager) LoadFromCLI(args []string) error {
	// 这里可以集成pflag或其他CLI库
	// 暂时简单实现
	for _, arg := range args {
		if strings.HasPrefix(arg, "--config=") {
			configPath := strings.TrimPrefix(arg, "--config=")
			cm.viper.SetConfigFile(configPath)
		} else if strings.HasPrefix(arg, "--env=") {
			envStr := strings.TrimPrefix(arg, "--env=")
			cm.env = Environment(envStr)
		}
	}
	return nil
}

// Get 获取配置值
func (cm *viperConfigManager) Get(key string) interface{} {

	return cm.viper.Get(key)
}

// GetString 获取字符串配置值
func (cm *viperConfigManager) GetString(key string) string {

	return cm.viper.GetString(key)
}

// GetInt 获取整数配置值
func (cm *viperConfigManager) GetInt(key string) int {

	return cm.viper.GetInt(key)
}

// GetBool 获取布尔配置值
func (cm *viperConfigManager) GetBool(key string) bool {

	return cm.viper.GetBool(key)
}

// GetDuration 获取时间间隔配置值
func (cm *viperConfigManager) GetDuration(key string) time.Duration {

	return cm.viper.GetDuration(key)
}

// Validate 验证配置
func (cm *viperConfigManager) Validate() error {
	return cm.validator.ValidateStruct(cm.config)
}

// ValidateSection 验证配置段
func (cm *viperConfigManager) ValidateSection(section string) error {
	return cm.validator.ValidateSection(cm.config, section)
}

// Watch 监听配置变化
func (cm *viperConfigManager) Watch(callback ConfigWatchCallback) error {
	if cm.watcher == nil {
		cm.watcher = NewConfigWatcher(cm.viper, callback)
	}
	return cm.watcher.Start()
}

// StopWatch 停止监听配置变化
func (cm *viperConfigManager) StopWatch() error {
	if cm.watcher != nil {
		return cm.watcher.Stop()
	}
	return nil
}

// Encrypt 加密配置值
func (cm *viperConfigManager) Encrypt(key string) error {
	value := cm.viper.GetString(key)
	if value == "" {
		return fmt.Errorf("配置键 %s 不存在或为空", key)
	}

	return fmt.Errorf("加密功能已移除")
}

// Decrypt 解密配置值
func (cm *viperConfigManager) Decrypt(key string) error {
	value := cm.viper.GetString(key)
	if value == "" {
		return fmt.Errorf("配置键 %s 不存在或为空", key)
	}

	return fmt.Errorf("解密功能已移除")
}

// GetViper 获取Viper实例
func (cm *viperConfigManager) GetViper() *viper.Viper {
	return cm.viper
}

// LoadDefault 加载默认配置
func (cm *viperConfigManager) LoadDefault() (*AppConfig, error) {
	// 直接使用默认配置
	config := DefaultConfig()

	// 绑定环境变量
	cm.LoadFromEnv()

	// 如果有环境变量覆盖，则应用它们
	if cm.viper.AllKeys() != nil && len(cm.viper.AllKeys()) > 0 {
		// 只在有配置时才解析
		if err := cm.viper.Unmarshal(config); err != nil {
			return nil, NewParseError("", "解析环境变量失败", err)
		}
	}

	// 验证配置
	if err := cm.validator.ValidateStruct(config); err != nil {
		return nil, err
	}

	// 记录审计日志

	return config, nil
}

// applyDefaults 应用默认值
func (cm *viperConfigManager) applyDefaults(config, defaultConfig *AppConfig) {
	// 如果配置中的值为空，使用默认值
	if config.App.Name == "" {
		config.App.Name = defaultConfig.App.Name
	}
	if config.App.Version == "" {
		config.App.Version = defaultConfig.App.Version
	}
	if config.App.Environment == "" {
		config.App.Environment = defaultConfig.App.Environment
	}
	if config.App.LogLevel == "" {
		config.App.LogLevel = defaultConfig.App.LogLevel
	}

	// 应用其他默认值...
	if len(config.Kafka.Brokers) == 0 {
		config.Kafka.Brokers = defaultConfig.Kafka.Brokers
	}
	if config.Kafka.Topics.DeviceData == "" {
		config.Kafka.Topics.DeviceData = defaultConfig.Kafka.Topics.DeviceData
	}
	if config.Kafka.Topics.Alerts == "" {
		config.Kafka.Topics.Alerts = defaultConfig.Kafka.Topics.Alerts
	}

	if config.Redis.Host == "" {
		config.Redis.Host = defaultConfig.Redis.Host
	}
	if config.Redis.Port == 0 {
		config.Redis.Port = defaultConfig.Redis.Port
	}

	if config.DB.Host == "" {
		config.DB.Host = defaultConfig.DB.Host
	}
	if config.DB.Port == 0 {
		config.DB.Port = defaultConfig.DB.Port
	}
	if config.DB.Username == "" {
		config.DB.Username = defaultConfig.DB.Username
	}
	if config.DB.Password == "" {
		config.DB.Password = defaultConfig.DB.Password
	}
	if config.DB.Database == "" {
		config.DB.Database = defaultConfig.DB.Database
	}
}

// GetConfig 获取配置对象
func (cm *viperConfigManager) GetConfig() *AppConfig {
	return cm.config
}

// ...
