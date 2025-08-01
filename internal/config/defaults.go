package config

import (
	"time"
)

// DefaultConfig 返回默认配置
func DefaultConfig() *AppConfig {
	return &AppConfig{
		App: AppSection{
			Name:        "industrial-iot-monitor",
			Version:     "1.0.0",
			Environment: "development",
			Debug:       true,
			LogLevel:    "info",
		},
		Kafka: KafkaSection{
			Brokers: []string{"localhost:9092"},
			Topics: TopicConfig{
				DeviceData: "device-data",
				Alerts:     "alerts",
			},
			Producer: KafkaProducer{
				ClientID:         "iot-producer",
				BatchSize:        16384,
				BatchTimeout:     10 * time.Millisecond,
				CompressionType:  "snappy",
				MaxRetries:       3,
				RetryBackoff:     100 * time.Millisecond,
				RequiredAcks:     1,
				FlushFrequency:   1 * time.Second,
				ChannelBufferSize: 1000,
				Timeout:          30 * time.Second,
			},
			Consumer: KafkaConsumer{
				GroupID:          "iot-consumer-group",
				AutoOffsetReset:  "earliest",
				EnableAutoCommit: true,
				SessionTimeout:   30 * time.Second,
				MaxPollRecords:   500,
			},
			Security: KafkaSecurity{
				Protocol: "PLAINTEXT",
				Username: "",
				Password: "",
			},
			Timeout: 30 * time.Second,
		},
		Redis: RedisSection{
			Host:       "localhost",
			Port:       6379,
			Password:   "",
			Database:   0,
			PoolSize:   10,
			MaxRetries: 3,
			Timeout:    5 * time.Second,
		},
		DB: DBSection{
			Host:            "localhost",
			Port:            5432,
			Username:        "postgres",
			Password:        "postgres",
			Database:        "iot_monitor",
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 1 * time.Hour,
		},
		Producer: ProducerSection{
			DeviceCount:   50,
			SendInterval:  5 * time.Second,
			DataVariance:  0.1,
			BatchSize:     100,
			RetryAttempts: 3,
			Timeout:       10 * time.Second,
		},
		Consumer: ConsumerSection{
			WorkerCount:       5,
			BufferSize:        1000,
			BatchSize:         50,
			ProcessingTimeout: 30 * time.Second,
			RetryAttempts:     3,
			DeadLetterQueue:   true,
		},
		WebSocket: WSSection{
			Host:              "0.0.0.0",
			Port:              8081,
			Path:              "/ws",
			MaxConnections:    1000,
			ReadBufferSize:    1024,
			WriteBufferSize:   1024,
			HeartbeatInterval: 30 * time.Second,
			ConnectionTimeout: 60 * time.Second,
		},
		Web: WebSection{
			Host:           "0.0.0.0",
			Port:           8080,
			TemplatePath:   "web/templates",
			StaticPath:     "web/static",
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1048576,
		},
		Device: DeviceSection{
			Simulator: DeviceSimulator{
				Enabled:         true,
				DeviceCount:     50,
				SampleInterval:  5 * time.Second,
				DataVariation:   0.1,
				AnomalyRate:     0.01,
				TrendEnabled:    true,
				TrendStrength:   0.1,
				WorkerPoolSize:  10,
				QueueBufferSize: 1000,
			},
			Thresholds: DeviceThresholds{
				Temperature: TemperatureThreshold{
					Min:      -20.0,
					Max:      80.0,
					Warning:  70.0,
					Critical: 75.0,
				},
				Humidity: HumidityThreshold{
					Min:      0.0,
					Max:      100.0,
					Warning:  85.0,
					Critical: 90.0,
				},
				Battery: BatteryThreshold{
					Min:      0.0,
					Max:      100.0,
					Warning:  20.0,
					Critical: 10.0,
				},
			},
		},
		Alert: AlertSection{
			Enabled: true,
			Rules: []AlertRule{
				{
					Name:      "high_temperature",
					Condition: "temperature > 75",
					Severity:  "critical",
					Cooldown:  5 * time.Minute,
				},
				{
					Name:      "low_battery",
					Condition: "battery < 10",
					Severity:  "medium",
					Cooldown:  1 * time.Hour,
				},
			},
			Notifications: AlertNotification{
				Email: EmailNotification{
					Enabled:  false,
					SMTPHost: "smtp.example.com",
					SMTPPort: 587,
				},
				Webhook: WebhookNotification{
					Enabled: true,
					URL:     "http://localhost:8080/webhook/alerts",
				},
			},
		},
		Monitor: MonitorSection{
			Prometheus: PrometheusConfig{
				Enabled:        true,
				Host:           "0.0.0.0",
				Port:           9090,
				Path:           "/metrics",
				ScrapeInterval: 15 * time.Second,
			},
			Logging: LoggingConfig{
				Level:      "info",
				Format:     "json",
				Output:     []string{"stdout", "file"},
				FilePath:   "logs/app.log",
				MaxSize:    "100MB",
				MaxAge:     "7d",
				MaxBackups: 10,
			},
		},
	}
}

// DefaultDevelopmentConfig 返回开发环境默认配置
func DefaultDevelopmentConfig() *AppConfig {
	config := DefaultConfig()
	config.App.Environment = "development"
	config.App.Debug = true
	config.App.LogLevel = "debug"
	config.Monitor.Logging.Level = "debug"
	return config
}

// DefaultProductionConfig 返回生产环境默认配置
func DefaultProductionConfig() *AppConfig {
	config := DefaultConfig()
	config.App.Environment = "production"
	config.App.Debug = false
	config.App.LogLevel = "warn"
	config.Monitor.Logging.Level = "warn"
	return config
}

// DefaultTestingConfig 返回测试环境默认配置
func DefaultTestingConfig() *AppConfig {
	config := DefaultConfig()
	config.App.Environment = "testing"
	config.App.Debug = false
	config.App.LogLevel = "info"
	config.Producer.DeviceCount = 10
	config.Consumer.WorkerCount = 2
	config.WebSocket.MaxConnections = 100
	return config
}

// GetDefaultByEnvironment 根据环境返回默认配置
func GetDefaultByEnvironment(env Environment) *AppConfig {
	switch env {
	case Development:
		return DefaultDevelopmentConfig()
	case Testing:
		return DefaultTestingConfig()
	case Production:
		return DefaultProductionConfig()
	default:
		return DefaultConfig()
	}
}
