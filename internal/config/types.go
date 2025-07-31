package config

import (
	"time"
)

// Environment 环境类型枚举
type Environment string

const (
	Development Environment = "development"
	Testing     Environment = "testing"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

// String 返回环境类型的字符串表示
func (e Environment) String() string {
	return string(e)
}

// IsValid 检查环境类型是否有效
func (e Environment) IsValid() bool {
	switch e {
	case Development, Testing, Staging, Production:
		return true
	default:
		return false
	}
}

// IsDevelopment 检查是否为开发环境
func (e Environment) IsDevelopment() bool {
	return e == Development
}

// IsProduction 检查是否为生产环境
func (e Environment) IsProduction() bool {
	return e == Production
}

// ConfigSource 配置源类型
type ConfigSource int

const (
	SourceFile ConfigSource = iota
	SourceEnv
	SourceCLI
	SourceRemote
)

// String 返回配置源的字符串表示
func (cs ConfigSource) String() string {
	switch cs {
	case SourceFile:
		return "file"
	case SourceEnv:
		return "env"
	case SourceCLI:
		return "cli"
	case SourceRemote:
		return "remote"
	default:
		return "unknown"
	}
}

// Priority 配置源优先级（数值越大优先级越高）
func (cs ConfigSource) Priority() int {
	switch cs {
	case SourceFile:
		return 1
	case SourceEnv:
		return 2
	case SourceCLI:
		return 3
	case SourceRemote:
		return 4
	default:
		return 0
	}
}

// ValidationError 配置验证错误
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Tag     string      `json:"tag"`
	Message string      `json:"message"`
}

// Error 实现error接口
func (ve ValidationError) Error() string {
	return ve.Message
}



// ConfigLoadOptions 配置加载选项
type ConfigLoadOptions struct {
	ConfigPath    string
	Environment   Environment
	EnvPrefix     string
	WatchEnabled  bool
	ValidateOnly  bool
	IgnoreEnvVars bool
	Sources       []ConfigSource
}

// DefaultConfigLoadOptions 返回默认的配置加载选项
func DefaultConfigLoadOptions() *ConfigLoadOptions {
	return &ConfigLoadOptions{
		ConfigPath:    "config/config.yaml",
		Environment:   Development,
		EnvPrefix:     "IOT",
		WatchEnabled:  false,
		ValidateOnly:  false,
		IgnoreEnvVars: false,
		Sources:       []ConfigSource{SourceFile, SourceEnv, SourceCLI},
	}
}

// ConfigWatchCallback 配置监听回调函数类型
type ConfigWatchCallback func(*AppConfig) error



// ConfigMetadata 配置元数据
type ConfigMetadata struct {
	LoadTime    time.Time
	Source      ConfigSource
	Environment Environment
	Version     string
	Checksum    string
}

// HealthStatus 健康状态
type HealthStatus struct {
	Healthy     bool              `json:"healthy"`
	Timestamp   time.Time         `json:"timestamp"`
	Services    map[string]bool   `json:"services"`
	Errors      []string          `json:"errors,omitempty"`
	Metrics     map[string]string `json:"metrics,omitempty"`
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name      string    `json:"name"`
	Healthy   bool      `json:"healthy"`
	LastCheck time.Time `json:"last_check"`
	Error     string    `json:"error,omitempty"`
	Uptime    string    `json:"uptime"`
}
