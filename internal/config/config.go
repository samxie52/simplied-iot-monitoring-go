package config

import (
	"time"
)

// AppConfig 应用全局配置结构
type AppConfig struct {
	// 应用基础配置
	App AppSection `yaml:"app" validate:"required"`

	// 中间件配置
	Kafka KafkaSection `yaml:"kafka" validate:"required"`
	Redis RedisSection `yaml:"redis" validate:"required"`
	DB    DBSection    `yaml:"database" validate:"required"`

	// 服务配置
	Producer  ProducerSection `yaml:"producer" validate:"required"`
	Consumer  ConsumerSection `yaml:"consumer" validate:"required"`
	WebSocket WSSection       `yaml:"websocket" validate:"required"`
	Web       WebSection      `yaml:"web" validate:"required"`

	// 业务配置
	Device DeviceSection `yaml:"device" validate:"required"`
	Alert  AlertSection  `yaml:"alert" validate:"required"`

	// 监控配置
	Monitor MonitorSection `yaml:"monitoring" validate:"required"`
}

// AppSection 应用基础配置
type AppSection struct {
	Name        string `yaml:"name" validate:"required"`
	Version     string `yaml:"version" validate:"required"`
	Environment string `yaml:"environment" validate:"required"`
	Debug       bool   `yaml:"debug"`
	LogLevel    string `yaml:"log_level" validate:"required"`
}

// KafkaSection Kafka配置段
type KafkaSection struct {
	Brokers  []string      `yaml:"brokers" validate:"required"`
	Topics   TopicConfig   `yaml:"topics" validate:"required"`
	Producer KafkaProducer `yaml:"producer" validate:"required"`
	Consumer KafkaConsumer `yaml:"consumer" validate:"required"`
	Security KafkaSecurity `yaml:"security"`
	Timeout  time.Duration `yaml:"timeout" validate:"required"`
}

// TopicConfig Kafka主题配置
type TopicConfig struct {
	DeviceData string `yaml:"device_data" validate:"required"`
	Alerts     string `yaml:"alerts" validate:"required"`
}

// KafkaProducer Kafka生产者配置
type KafkaProducer struct {
	ClientID         string        `yaml:"client_id" validate:"required"`
	BatchSize        int           `yaml:"batch_size" validate:"min=1,max=1000000"`
	BatchTimeout     time.Duration `yaml:"batch_timeout" validate:"min=1ms"`
	CompressionType  string        `yaml:"compression_type" validate:"oneof=none gzip snappy lz4 zstd"`
	MaxRetries       int           `yaml:"max_retries" validate:"min=0,max=10"`
	RetryBackoff     time.Duration `yaml:"retry_backoff" validate:"min=1ms"`
	RequiredAcks     int           `yaml:"required_acks" validate:"oneof=-1 0 1"`
	FlushFrequency   time.Duration `yaml:"flush_frequency" validate:"min=1ms"`
	ChannelBufferSize int          `yaml:"channel_buffer_size" validate:"min=1"`
	Timeout          time.Duration `yaml:"timeout" validate:"required"`
}

// KafkaConsumer Kafka消费者配置
type KafkaConsumer struct {
	GroupID          string        `yaml:"group_id" validate:"required"`
	AutoOffsetReset  string        `yaml:"auto_offset_reset" validate:"required"`
	EnableAutoCommit bool          `yaml:"enable_auto_commit"`
	SessionTimeout   time.Duration `yaml:"session_timeout" validate:"required"`
	MaxPollRecords   int           `yaml:"max_poll_records"`
}

// KafkaSecurity Kafka安全配置
type KafkaSecurity struct {
	Protocol string `yaml:"protocol"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// RedisSection Redis配置段
type RedisSection struct {
	Host       string        `yaml:"host" validate:"required"`
	Port       int           `yaml:"port" validate:"required"`
	Password   string        `yaml:"password"`
	Database   int           `yaml:"database"`
	PoolSize   int           `yaml:"pool_size" validate:"required"`
	MaxRetries int           `yaml:"max_retries"`
	Timeout    time.Duration `yaml:"timeout" validate:"required"`
}

// DBSection 数据库配置段
type DBSection struct {
	Host            string        `yaml:"host" validate:"required"`
	Port            int           `yaml:"port" validate:"required"`
	Username        string        `yaml:"username" validate:"required"`
	Password        string        `yaml:"password" validate:"required"`
	Database        string        `yaml:"database" validate:"required"`
	SSLMode         string        `yaml:"ssl_mode" validate:"required"`
	MaxOpenConns    int           `yaml:"max_open_conns" validate:"required"`
	MaxIdleConns    int           `yaml:"max_idle_conns" validate:"required"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" validate:"required"`
}

// ProducerSection 生产者服务配置
type ProducerSection struct {
	DeviceCount   int           `yaml:"device_count" validate:"required"`
	SendInterval  time.Duration `yaml:"send_interval" validate:"required"`
	DataVariance  float64       `yaml:"data_variance"`
	BatchSize     int           `yaml:"batch_size" validate:"required"`
	RetryAttempts int           `yaml:"retry_attempts"`
	Timeout       time.Duration `yaml:"timeout" validate:"required"`
}

// ConsumerSection 消费者服务配置
type ConsumerSection struct {
	WorkerCount       int           `yaml:"worker_count" validate:"required"`
	BufferSize        int           `yaml:"buffer_size" validate:"required"`
	BatchSize         int           `yaml:"batch_size" validate:"required"`
	ProcessingTimeout time.Duration `yaml:"processing_timeout" validate:"required"`
	RetryAttempts     int           `yaml:"retry_attempts"`
	DeadLetterQueue   bool          `yaml:"dead_letter_queue"`
}

// WSSection WebSocket服务配置
type WSSection struct {
	Host              string        `yaml:"host" validate:"required"`
	Port              int           `yaml:"port" validate:"required"`
	Path              string        `yaml:"path" validate:"required"`
	MaxConnections    int           `yaml:"max_connections" validate:"required"`
	ReadBufferSize    int           `yaml:"read_buffer_size" validate:"required"`
	WriteBufferSize   int           `yaml:"write_buffer_size" validate:"required"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval" validate:"required"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout" validate:"required"`
}

// WebSection Web服务配置
type WebSection struct {
	Host           string        `yaml:"host" validate:"required"`
	Port           int           `yaml:"port" validate:"required"`
	TemplatePath   string        `yaml:"template_path" validate:"required"`
	StaticPath     string        `yaml:"static_path" validate:"required"`
	ReadTimeout    time.Duration `yaml:"read_timeout" validate:"required"`
	WriteTimeout   time.Duration `yaml:"write_timeout" validate:"required"`
	MaxHeaderBytes int           `yaml:"max_header_bytes" validate:"required"`
}

// DeviceSection 设备配置段
type DeviceSection struct {
	Simulator  DeviceSimulator  `yaml:"simulator" validate:"required"`
	Thresholds DeviceThresholds `yaml:"thresholds" validate:"required"`
}

// DeviceSimulator 设备模拟器配置
type DeviceSimulator struct {
	Enabled         bool          `yaml:"enabled"`
	DeviceCount     int           `yaml:"device_count" validate:"min=1,max=100000"`
	SampleInterval  time.Duration `yaml:"sample_interval" validate:"min=100ms"`
	DataVariation   float64       `yaml:"data_variation" validate:"min=0,max=1"`
	AnomalyRate     float64       `yaml:"anomaly_rate" validate:"min=0,max=0.1"`
	TrendEnabled    bool          `yaml:"trend_enabled"`
	TrendStrength   float64       `yaml:"trend_strength" validate:"min=0,max=1"`
	WorkerPoolSize  int           `yaml:"worker_pool_size" validate:"min=1,max=1000"`
	QueueBufferSize int           `yaml:"queue_buffer_size" validate:"min=100"`
}

// DeviceThresholds 设备阈值配置
type DeviceThresholds struct {
	Temperature TemperatureThreshold `yaml:"temperature" validate:"required"`
	Humidity    HumidityThreshold    `yaml:"humidity" validate:"required"`
	Battery     BatteryThreshold     `yaml:"battery" validate:"required"`
}

// TemperatureThreshold 温度阈值
type TemperatureThreshold struct {
	Min      float64 `yaml:"min"`
	Max      float64 `yaml:"max"`
	Warning  float64 `yaml:"warning"`
	Critical float64 `yaml:"critical"`
}

// HumidityThreshold 湿度阈值
type HumidityThreshold struct {
	Min      float64 `yaml:"min"`
	Max      float64 `yaml:"max"`
	Warning  float64 `yaml:"warning"`
	Critical float64 `yaml:"critical"`
}

// BatteryThreshold 电池阈值
type BatteryThreshold struct {
	Min      float64 `yaml:"min"`
	Max      float64 `yaml:"max"`
	Warning  float64 `yaml:"warning"`
	Critical float64 `yaml:"critical"`
}

// AlertSection 告警配置段
type AlertSection struct {
	Enabled       bool              `yaml:"enabled"`
	Rules         []AlertRule       `yaml:"rules" validate:"dive"`
	Notifications AlertNotification `yaml:"notifications" validate:"required"`
}

// AlertRule 告警规则
type AlertRule struct {
	Name      string        `yaml:"name" validate:"required"`
	Condition string        `yaml:"condition" validate:"required"`
	Severity  string        `yaml:"severity" validate:"required"`
	Cooldown  time.Duration `yaml:"cooldown" validate:"required"`
}

// AlertNotification 告警通知配置
type AlertNotification struct {
	Email   EmailNotification   `yaml:"email"`
	Webhook WebhookNotification `yaml:"webhook"`
}

// EmailNotification 邮件通知配置
type EmailNotification struct {
	Enabled  bool   `yaml:"enabled"`
	SMTPHost string `yaml:"smtp_host" validate:"required_if=Enabled true"`
	SMTPPort int    `yaml:"smtp_port" validate:"required_if=Enabled true"`
}

// WebhookNotification Webhook通知配置
type WebhookNotification struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url" validate:"required_if=Enabled true"`
}

// MonitorSection 监控配置段
type MonitorSection struct {
	Prometheus PrometheusConfig `yaml:"prometheus" validate:"required"`
	Logging    LoggingConfig    `yaml:"logging" validate:"required"`
}

// PrometheusConfig Prometheus配置
type PrometheusConfig struct {
	Enabled        bool          `yaml:"enabled"`
	Host           string        `yaml:"host" validate:"required_if=Enabled true"`
	Port           int           `yaml:"port" validate:"required_if=Enabled true"`
	Path           string        `yaml:"path" validate:"required_if=Enabled true"`
	ScrapeInterval time.Duration `yaml:"scrape_interval" validate:"required_if=Enabled true"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string   `yaml:"level" validate:"required"`
	Format     string   `yaml:"format" validate:"required"`
	Output     []string `yaml:"output" validate:"required"`
	FilePath   string   `yaml:"file_path"`
	MaxSize    string   `yaml:"max_size"`
	MaxAge     string   `yaml:"max_age"`
	MaxBackups int      `yaml:"max_backups"`
}
