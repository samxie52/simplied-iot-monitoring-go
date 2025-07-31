package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvironmentManager 环境管理器接口
type EnvironmentManager interface {
	// 环境检测
	GetCurrentEnv() Environment
	SetEnv(env Environment) error
	IsProduction() bool
	IsDevelopment() bool

	// 环境配置
	LoadEnvConfig(env Environment) (*AppConfig, error)
	GetEnvConfigPath(env Environment) string

	// 环境变量处理
	BindEnvVars() error
	SetEnvPrefix(prefix string)
	GetEnvVar(key string) string

	// 环境验证
	ValidateEnv() error
	CheckEnvRequirements() error
}

// environmentManager 环境管理器实现
type environmentManager struct {
	currentEnv Environment
	envPrefix  string
	configPath string
	envVars    map[string]string
}

// NewEnvironmentManager 创建新的环境管理器
func NewEnvironmentManager() EnvironmentManager {
	return &environmentManager{
		currentEnv: Development,
		envPrefix:  "IOT",
		configPath: "configs",
		envVars:    make(map[string]string),
	}
}

// GetCurrentEnv 获取当前环境
func (em *environmentManager) GetCurrentEnv() Environment {
	// 优先从环境变量获取
	if envStr := os.Getenv("GO_ENV"); envStr != "" {
		if env := Environment(envStr); env.IsValid() {
			em.currentEnv = env
			return env
		}
	}

	if envStr := os.Getenv("IOT_APP_ENVIRONMENT"); envStr != "" {
		if env := Environment(envStr); env.IsValid() {
			em.currentEnv = env
			return env
		}
	}

	if envStr := os.Getenv("ENVIRONMENT"); envStr != "" {
		if env := Environment(envStr); env.IsValid() {
			em.currentEnv = env
			return env
		}
	}

	// 根据其他环境变量推断环境
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		em.currentEnv = Production
		return Production
	}

	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		em.currentEnv = Testing
		return Testing
	}

	// 默认为开发环境
	return em.currentEnv
}

// SetEnv 设置当前环境
func (em *environmentManager) SetEnv(env Environment) error {
	if !env.IsValid() {
		return fmt.Errorf("无效的环境类型: %s", env)
	}
	
	em.currentEnv = env
	
	// 设置环境变量
	if err := os.Setenv("GO_ENV", env.String()); err != nil {
		return fmt.Errorf("设置环境变量失败: %w", err)
	}
	
	return nil
}

// IsProduction 检查是否为生产环境
func (em *environmentManager) IsProduction() bool {
	return em.GetCurrentEnv().IsProduction()
}

// IsDevelopment 检查是否为开发环境
func (em *environmentManager) IsDevelopment() bool {
	return em.GetCurrentEnv().IsDevelopment()
}

// LoadEnvConfig 加载环境特定配置
func (em *environmentManager) LoadEnvConfig(env Environment) (*AppConfig, error) {
	// 获取基础配置
	config := GetDefaultByEnvironment(env)
	
	// 加载环境特定的配置文件
	envConfigPath := em.GetEnvConfigPath(env)
	if _, err := os.Stat(envConfigPath); err == nil {
		// 配置文件存在，可以进一步处理
		// 这里可以集成Viper来读取环境配置文件
	}
	
	// 应用环境变量覆盖
	if err := em.applyEnvOverrides(config); err != nil {
		return nil, fmt.Errorf("应用环境变量覆盖失败: %w", err)
	}
	
	return config, nil
}

// GetEnvConfigPath 获取环境配置文件路径
func (em *environmentManager) GetEnvConfigPath(env Environment) string {
	return filepath.Join(em.configPath, fmt.Sprintf("%s.yaml", env.String()))
}

// BindEnvVars 绑定环境变量
func (em *environmentManager) BindEnvVars() error {
	// 预定义的环境变量映射
	envMappings := map[string]string{
		"IOT_APP_NAME":             "app.name",
		"IOT_APP_VERSION":          "app.version",
		"IOT_APP_ENVIRONMENT":      "app.environment",
		"IOT_APP_DEBUG":            "app.debug",
		"IOT_APP_LOG_LEVEL":        "app.log_level",
		
		"IOT_KAFKA_BROKERS":        "kafka.brokers",
		"IOT_KAFKA_USERNAME":       "kafka.security.username",
		"IOT_KAFKA_PASSWORD":       "kafka.security.password",
		
		"IOT_REDIS_HOST":           "redis.host",
		"IOT_REDIS_PORT":           "redis.port",
		"IOT_REDIS_PASSWORD":       "redis.password",
		"IOT_REDIS_DATABASE":       "redis.database",
		
		"IOT_DB_HOST":              "database.host",
		"IOT_DB_PORT":              "database.port",
		"IOT_DB_USERNAME":          "database.username",
		"IOT_DB_PASSWORD":          "database.password",
		"IOT_DB_DATABASE":          "database.database",
		"IOT_DB_SSL_MODE":          "database.ssl_mode",
		
		"IOT_WS_HOST":              "websocket.host",
		"IOT_WS_PORT":              "websocket.port",
		"IOT_WS_PATH":              "websocket.path",
		"IOT_WS_MAX_CONNECTIONS":   "websocket.max_connections",
		
		"IOT_WEB_HOST":             "web.host",
		"IOT_WEB_PORT":             "web.port",
		"IOT_WEB_TEMPLATE_PATH":    "web.template_path",
		"IOT_WEB_STATIC_PATH":      "web.static_path",
		
		"IOT_PROMETHEUS_ENABLED":   "monitoring.prometheus.enabled",
		"IOT_PROMETHEUS_PORT":      "monitoring.prometheus.port",
		"IOT_LOG_LEVEL":            "monitoring.logging.level",
		"IOT_LOG_FORMAT":           "monitoring.logging.format",
		
		"IOT_JWT_SECRET":           "security.auth.jwt_secret",
		"IOT_ENCRYPTION_KEY_FILE":  "security.encryption.key_file",
	}
	
	// 绑定环境变量
	for envKey, configKey := range envMappings {
		if value := os.Getenv(envKey); value != "" {
			em.envVars[configKey] = value
		}
	}
	
	return nil
}

// SetEnvPrefix 设置环境变量前缀
func (em *environmentManager) SetEnvPrefix(prefix string) {
	em.envPrefix = prefix
}

// GetEnvVar 获取环境变量值
func (em *environmentManager) GetEnvVar(key string) string {
	// 尝试直接获取
	if value := os.Getenv(key); value != "" {
		return value
	}
	
	// 尝试带前缀获取
	prefixedKey := fmt.Sprintf("%s_%s", em.envPrefix, strings.ToUpper(key))
	if value := os.Getenv(prefixedKey); value != "" {
		return value
	}
	
	// 从缓存的环境变量中获取
	if value, exists := em.envVars[key]; exists {
		return value
	}
	
	return ""
}

// ValidateEnv 验证环境配置
func (em *environmentManager) ValidateEnv() error {
	env := em.GetCurrentEnv()
	
	// 验证环境类型
	if !env.IsValid() {
		return fmt.Errorf("无效的环境类型: %s", env)
	}
	
	// 生产环境特殊验证
	if env.IsProduction() {
		requiredEnvVars := []string{
			"IOT_DB_PASSWORD",
			"IOT_JWT_SECRET",
		}
		
		for _, envVar := range requiredEnvVars {
			if os.Getenv(envVar) == "" {
				return fmt.Errorf("生产环境缺少必需的环境变量: %s", envVar)
			}
		}
	}
	
	return nil
}

// CheckEnvRequirements 检查环境要求
func (em *environmentManager) CheckEnvRequirements() error {
	env := em.GetCurrentEnv()
	
	// 检查Go版本
	if goVersion := os.Getenv("GO_VERSION"); goVersion != "" {
		// 可以添加Go版本检查逻辑
	}
	
	// 检查必需的目录
	requiredDirs := []string{
		"logs",
		"configs",
	}
	
	for _, dir := range requiredDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("创建必需目录 %s 失败: %w", dir, err)
			}
		}
	}
	
	// 环境特定检查
	switch env {
	case Production:
		return em.checkProductionRequirements()
	case Development:
		return em.checkDevelopmentRequirements()
	case Testing:
		return em.checkTestingRequirements()
	default:
		return nil
	}
}

// checkProductionRequirements 检查生产环境要求
func (em *environmentManager) checkProductionRequirements() error {
	// 检查SSL证书
	if sslCert := os.Getenv("IOT_SSL_CERT_FILE"); sslCert != "" {
		if _, err := os.Stat(sslCert); os.IsNotExist(err) {
			return fmt.Errorf("SSL证书文件不存在: %s", sslCert)
		}
	}
	
	// 检查加密密钥文件
	if keyFile := os.Getenv("IOT_ENCRYPTION_KEY_FILE"); keyFile != "" {
		if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			return fmt.Errorf("加密密钥文件不存在: %s", keyFile)
		}
	}
	
	return nil
}

// checkDevelopmentRequirements 检查开发环境要求
func (em *environmentManager) checkDevelopmentRequirements() error {
	// 开发环境通常要求较少
	return nil
}

// checkTestingRequirements 检查测试环境要求
func (em *environmentManager) checkTestingRequirements() error {
	// 检查测试数据目录
	testDataDir := "testdata"
	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testDataDir, 0755); err != nil {
			return fmt.Errorf("创建测试数据目录失败: %w", err)
		}
	}
	
	return nil
}

// applyEnvOverrides 应用环境变量覆盖
func (em *environmentManager) applyEnvOverrides(config *AppConfig) error {
	// 绑定环境变量
	if err := em.BindEnvVars(); err != nil {
		return err
	}
	
	// 应用环境变量覆盖
	for configKey, envValue := range em.envVars {
		if err := em.setConfigValue(config, configKey, envValue); err != nil {
			return fmt.Errorf("设置配置值 %s 失败: %w", configKey, err)
		}
	}
	
	return nil
}

// setConfigValue 设置配置值
func (em *environmentManager) setConfigValue(config *AppConfig, key, value string) error {
	// 这里可以使用反射或者switch语句来设置配置值
	// 为了简化，这里只处理常见的配置项
	
	switch key {
	case "app.environment":
		config.App.Environment = value
	case "app.debug":
		config.App.Debug = (value == "true")
	case "app.log_level":
		config.App.LogLevel = value
	case "database.host":
		config.DB.Host = value
	case "database.password":
		config.DB.Password = value
	case "redis.host":
		config.Redis.Host = value
	case "redis.password":
		config.Redis.Password = value
	// security 配置已被移除
	// 可以添加更多配置项
	default:
		// 对于未知的配置项，可以记录警告日志
		return fmt.Errorf("未知的配置键: %s", key)
	}
	
	return nil
}
