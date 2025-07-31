package config

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ConfigValidator 配置验证器接口
type ConfigValidator interface {
	// 结构验证
	ValidateStruct(config *AppConfig) error
	ValidateField(field string, value interface{}) error
	ValidateSection(config *AppConfig, section string) error

	// 业务验证
	ValidateBusinessRules(config *AppConfig) error
	ValidateServiceDependencies(config *AppConfig) error

	// 自定义验证
	RegisterValidator(tag string, fn validator.Func) error
	ValidateCustom(config *AppConfig) error

	// 验证报告
	GetValidationErrors() []ValidationError
	FormatErrors(errors []ValidationError) string
}

// configValidator 配置验证器实现
type configValidator struct {
	validator *validator.Validate
	errors    []ValidationError
}

// NewConfigValidator 创建新的配置验证器
func NewConfigValidator() ConfigValidator {
	v := validator.New()
	cv := &configValidator{
		validator: v,
		errors:    make([]ValidationError, 0),
	}

	// 注册自定义验证函数
	cv.registerCustomValidators()

	return cv
}

// registerCustomValidators 注册自定义验证函数
func (cv *configValidator) registerCustomValidators() {
	// 注册自定义验证函数
	validators := map[string]validator.Func{
		"hostname_port": ValidateHostnamePort,
		"semver":        ValidateSemVer,
		"filepath":      ValidateFilePath,
		"dirpath":       ValidateDirPath,
		"reachable":     ValidateReachable,
		"file_exists":   ValidateFileExists,
	}

	for tag, fn := range validators {
		cv.validator.RegisterValidation(tag, fn)
	}
}

// ValidateStruct 验证整个配置结构
func (cv *configValidator) ValidateStruct(config *AppConfig) error {
	cv.errors = make([]ValidationError, 0)

	// 结构验证
	if err := cv.validator.Struct(config); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			for _, ve := range validationErrors {
				cv.errors = append(cv.errors, ValidationError{
					Field:   ve.Field(),
					Value:   ve.Value(),
					Tag:     ve.Tag(),
					Message: cv.formatValidationMessage(ve),
				})
			}
		}
	}

	// 业务规则验证
	if err := cv.ValidateBusinessRules(config); err != nil {
		return err
	}

	// 服务依赖验证
	if err := cv.ValidateServiceDependencies(config); err != nil {
		return err
	}

	// 自定义验证
	if err := cv.ValidateCustom(config); err != nil {
		return err
	}

	if len(cv.errors) > 0 {
		return fmt.Errorf("配置验证失败: %s", cv.FormatErrors(cv.errors))
	}

	return nil
}

// ValidateField 验证单个字段
func (cv *configValidator) ValidateField(field string, value interface{}) error {
	// 这里可以实现单个字段的验证逻辑
	return cv.validator.Var(value, field)
}

// ValidateSection 验证配置段
func (cv *configValidator) ValidateSection(config *AppConfig, section string) error {
	switch section {
	case "app":
		return cv.validator.Struct(&config.App)
	case "kafka":
		return cv.validator.Struct(&config.Kafka)
	case "redis":
		return cv.validator.Struct(&config.Redis)
	case "database":
		return cv.validator.Struct(&config.DB)
	case "producer":
		return cv.validator.Struct(&config.Producer)
	case "consumer":
		return cv.validator.Struct(&config.Consumer)
	case "websocket":
		return cv.validator.Struct(&config.WebSocket)
	case "web":
		return cv.validator.Struct(&config.Web)
	case "device":
		return cv.validator.Struct(&config.Device)
	case "alert":
		return cv.validator.Struct(&config.Alert)
	case "monitoring":
		return cv.validator.Struct(&config.Monitor)
	// security 配置已被移除
	default:
		return fmt.Errorf("未知的配置段: %s", section)
	}
}

// ValidateBusinessRules 验证业务规则
func (cv *configValidator) ValidateBusinessRules(config *AppConfig) error {
	// 验证端口不冲突
	ports := map[int]string{
		config.WebSocket.Port:          "websocket",
		config.Web.Port:                "web",
		config.Monitor.Prometheus.Port: "prometheus",
	}

	for port, service := range ports {
		for otherPort, otherService := range ports {
			if port == otherPort && service != otherService {
				cv.errors = append(cv.errors, ValidationError{
					Field:   fmt.Sprintf("%s.port", service),
					Value:   port,
					Tag:     "port_conflict",
					Message: fmt.Sprintf("端口 %d 与 %s 服务冲突", port, otherService),
				})
			}
		}
	}

	// 验证设备阈值逻辑
	if err := cv.validateThresholds(config); err != nil {
		return err
	}

	// 验证告警规则
	if err := cv.validateAlertRules(config); err != nil {
		return err
	}

	// 验证环境特定规则
	if err := cv.validateEnvironmentRules(config); err != nil {
		return err
	}

	return nil
}

// validateThresholds 验证阈值设置
func (cv *configValidator) validateThresholds(config *AppConfig) error {
	// 温度阈值验证
	temp := config.Device.Thresholds.Temperature
	if temp.Min >= temp.Max {
		cv.errors = append(cv.errors, ValidationError{
			Field:   "device.thresholds.temperature",
			Value:   temp,
			Tag:     "threshold_range",
			Message: "温度最小值必须小于最大值",
		})
	}
	if temp.Warning >= temp.Critical {
		cv.errors = append(cv.errors, ValidationError{
			Field:   "device.thresholds.temperature",
			Value:   temp,
			Tag:     "threshold_warning",
			Message: "温度警告阈值必须小于临界阈值",
		})
	}

	// 湿度阈值验证
	humidity := config.Device.Thresholds.Humidity
	if humidity.Min >= humidity.Max {
		cv.errors = append(cv.errors, ValidationError{
			Field:   "device.thresholds.humidity",
			Value:   humidity,
			Tag:     "threshold_range",
			Message: "湿度最小值必须小于最大值",
		})
	}

	// 电池阈值验证
	battery := config.Device.Thresholds.Battery
	if battery.Warning <= battery.Critical {
		cv.errors = append(cv.errors, ValidationError{
			Field:   "device.thresholds.battery",
			Value:   battery,
			Tag:     "threshold_warning",
			Message: "电池警告阈值必须大于临界阈值",
		})
	}

	return nil
}

// validateAlertRules 验证告警规则
func (cv *configValidator) validateAlertRules(config *AppConfig) error {
	if !config.Alert.Enabled {
		return nil
	}

	// 验证告警规则名称唯一性
	ruleNames := make(map[string]bool)
	for _, rule := range config.Alert.Rules {
		if ruleNames[rule.Name] {
			cv.errors = append(cv.errors, ValidationError{
				Field:   "alert.rules",
				Value:   rule.Name,
				Tag:     "unique",
				Message: fmt.Sprintf("告警规则名称重复: %s", rule.Name),
			})
		}
		ruleNames[rule.Name] = true

		// 验证条件表达式
		if err := cv.validateAlertCondition(rule.Condition); err != nil {
			cv.errors = append(cv.errors, ValidationError{
				Field:   "alert.rules.condition",
				Value:   rule.Condition,
				Tag:     "condition_syntax",
				Message: fmt.Sprintf("告警条件语法错误: %s", err.Error()),
			})
		}
	}

	return nil
}

// validateAlertCondition 验证告警条件
func (cv *configValidator) validateAlertCondition(condition string) error {
	// 简单的条件语法验证
	validOperators := []string{">", "<", ">=", "<=", "==", "!="}
	validFields := []string{"temperature", "humidity", "battery", "pressure", "current"}

	hasOperator := false
	for _, op := range validOperators {
		if strings.Contains(condition, op) {
			hasOperator = true
			break
		}
	}

	if !hasOperator {
		return fmt.Errorf("条件必须包含比较操作符")
	}

	hasField := false
	for _, field := range validFields {
		if strings.Contains(condition, field) {
			hasField = true
			break
		}
	}

	if !hasField {
		return fmt.Errorf("条件必须包含有效的字段名")
	}

	return nil
}

// validateEnvironmentRules 验证环境特定规则
func (cv *configValidator) validateEnvironmentRules(config *AppConfig) error {
	env := Environment(config.App.Environment)

	switch env {
	case Production:
		return cv.validateProductionRules(config)
	case Development:
		return cv.validateDevelopmentRules(config)
	case Testing:
		return cv.validateTestingRules(config)
	default:
		return nil
	}
}

// validateProductionRules 验证生产环境规则
func (cv *configValidator) validateProductionRules(config *AppConfig) error {

	// 生产环境不应该启用调试模式
	if config.App.Debug {
		cv.errors = append(cv.errors, ValidationError{
			Field:   "app.debug",
			Value:   true,
			Tag:     "production_debug",
			Message: "生产环境不应该启用调试模式",
		})
	}

	// 生产环境日志级别不应该是debug
	if config.App.LogLevel == "debug" {
		cv.errors = append(cv.errors, ValidationError{
			Field:   "app.log_level",
			Value:   "debug",
			Tag:     "production_log_level",
			Message: "生产环境不应该使用debug日志级别",
		})
	}

	return nil
}

// validateDevelopmentRules 验证开发环境规则
func (cv *configValidator) validateDevelopmentRules(config *AppConfig) error {
	// 开发环境规则相对宽松
	return nil
}

// validateTestingRules 验证测试环境规则
func (cv *configValidator) validateTestingRules(config *AppConfig) error {
	// 测试环境设备数量不应该太大
	if config.Producer.DeviceCount > 100 {
		cv.errors = append(cv.errors, ValidationError{
			Field:   "producer.device_count",
			Value:   config.Producer.DeviceCount,
			Tag:     "testing_device_limit",
			Message: "测试环境设备数量不应该超过100",
		})
	}

	return nil
}

// ValidateServiceDependencies 验证服务依赖
func (cv *configValidator) ValidateServiceDependencies(config *AppConfig) error {
	// 验证Kafka连接
	if err := cv.validateKafkaConnection(config.Kafka); err != nil {
		cv.errors = append(cv.errors, ValidationError{
			Field:   "kafka",
			Value:   config.Kafka.Brokers,
			Tag:     "service_dependency",
			Message: fmt.Sprintf("Kafka连接验证失败: %s", err.Error()),
		})
	}

	// 验证Redis连接
	if err := cv.validateRedisConnection(config.Redis); err != nil {
		cv.errors = append(cv.errors, ValidationError{
			Field:   "redis",
			Value:   fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
			Tag:     "service_dependency",
			Message: fmt.Sprintf("Redis连接验证失败: %s", err.Error()),
		})
	}

	// 验证数据库连接
	if err := cv.validateDatabaseConnection(config.DB); err != nil {
		cv.errors = append(cv.errors, ValidationError{
			Field:   "database",
			Value:   fmt.Sprintf("%s:%d", config.DB.Host, config.DB.Port),
			Tag:     "service_dependency",
			Message: fmt.Sprintf("数据库连接验证失败: %s", err.Error()),
		})
	}

	return nil
}

// validateKafkaConnection 验证Kafka连接
func (cv *configValidator) validateKafkaConnection(kafka KafkaSection) error {
	for _, broker := range kafka.Brokers {
		if err := cv.validateNetworkAddress(broker); err != nil {
			return fmt.Errorf("Kafka broker %s 不可达: %w", broker, err)
		}
	}
	return nil
}

// validateRedisConnection 验证Redis连接
func (cv *configValidator) validateRedisConnection(redis RedisSection) error {
	address := fmt.Sprintf("%s:%d", redis.Host, redis.Port)
	return cv.validateNetworkAddress(address)
}

// validateDatabaseConnection 验证数据库连接
func (cv *configValidator) validateDatabaseConnection(db DBSection) error {
	address := fmt.Sprintf("%s:%d", db.Host, db.Port)
	return cv.validateNetworkAddress(address)
}

// validateNetworkAddress 验证网络地址可达性
func (cv *configValidator) validateNetworkAddress(address string) error {
	// 在实际环境中，这里可能需要更复杂的连接测试
	// 为了避免在配置验证时进行实际的网络连接，这里只做格式验证
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("地址格式错误: %w", err)
	}

	// 验证主机名或IP地址
	if net.ParseIP(host) == nil {
		// 不是IP地址，验证主机名格式
		if !isValidHostname(host) {
			return fmt.Errorf("无效的主机名: %s", host)
		}
	}

	// 验证端口范围
	if portNum, err := strconv.Atoi(port); err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("无效的端口号: %s", port)
	}

	return nil
}

// RegisterValidator 注册自定义验证函数
func (cv *configValidator) RegisterValidator(tag string, fn validator.Func) error {
	return cv.validator.RegisterValidation(tag, fn)
}

// ValidateCustom 自定义验证
func (cv *configValidator) ValidateCustom(config *AppConfig) error {
	// 这里可以添加更多自定义验证逻辑
	return nil
}

// GetValidationErrors 获取验证错误
func (cv *configValidator) GetValidationErrors() []ValidationError {
	return cv.errors
}

// FormatErrors 格式化错误信息
func (cv *configValidator) FormatErrors(errors []ValidationError) string {
	if len(errors) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, err := range errors {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return sb.String()
}

// formatValidationMessage 格式化验证消息
func (cv *configValidator) formatValidationMessage(ve validator.FieldError) string {
	switch ve.Tag() {
	case "required":
		return fmt.Sprintf("字段 %s 是必需的", ve.Field())
	case "min":
		return fmt.Sprintf("字段 %s 的值必须大于等于 %s", ve.Field(), ve.Param())
	case "max":
		return fmt.Sprintf("字段 %s 的值必须小于等于 %s", ve.Field(), ve.Param())
	case "oneof":
		return fmt.Sprintf("字段 %s 的值必须是以下之一: %s", ve.Field(), ve.Param())
	case "hostname":
		return fmt.Sprintf("字段 %s 必须是有效的主机名", ve.Field())
	case "ip":
		return fmt.Sprintf("字段 %s 必须是有效的IP地址", ve.Field())
	case "url":
		return fmt.Sprintf("字段 %s 必须是有效的URL", ve.Field())
	case "uri":
		return fmt.Sprintf("字段 %s 必须是有效的URI", ve.Field())
	case "hostname_port":
		return fmt.Sprintf("字段 %s 必须是有效的主机名:端口格式", ve.Field())
	case "semver":
		return fmt.Sprintf("字段 %s 必须是有效的语义版本号", ve.Field())
	case "filepath":
		return fmt.Sprintf("字段 %s 必须是有效的文件路径", ve.Field())
	case "dirpath":
		return fmt.Sprintf("字段 %s 必须是有效的目录路径", ve.Field())
	default:
		return fmt.Sprintf("字段 %s 验证失败: %s", ve.Field(), ve.Tag())
	}
}

// isValidHostname 验证主机名格式
func isValidHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}

	// 简单的主机名格式验证
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	return hostnameRegex.MatchString(hostname)
}
