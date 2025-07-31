package config

import (
	"fmt"
	"strings"
)

// ConfigError 配置错误类型
type ConfigError struct {
	Type    string
	Field   string
	Value   interface{}
	Message string
	Cause   error
}

// Error 实现error接口
func (ce *ConfigError) Error() string {
	if ce.Cause != nil {
		return fmt.Sprintf("配置错误 [%s.%s]: %s (原因: %v)", ce.Type, ce.Field, ce.Message, ce.Cause)
	}
	return fmt.Sprintf("配置错误 [%s.%s]: %s", ce.Type, ce.Field, ce.Message)
}

// Unwrap 实现错误解包
func (ce *ConfigError) Unwrap() error {
	return ce.Cause
}

// ConfigErrorType 配置错误类型常量
const (
	ErrorTypeValidation   = "validation"
	ErrorTypeLoad         = "load"
	ErrorTypeParse        = "parse"
	ErrorTypeEnvironment  = "environment"

	ErrorTypeNetwork      = "network"
	ErrorTypeFileSystem   = "filesystem"
)

// NewConfigError 创建新的配置错误
func NewConfigError(errorType, field, message string, cause error) *ConfigError {
	return &ConfigError{
		Type:    errorType,
		Field:   field,
		Message: message,
		Cause:   cause,
	}
}

// NewValidationError 创建验证错误
func NewValidationError(field, message string) *ConfigError {
	return &ConfigError{
		Type:    ErrorTypeValidation,
		Field:   field,
		Message: message,
	}
}

// NewLoadError 创建加载错误
func NewLoadError(field, message string, cause error) *ConfigError {
	return &ConfigError{
		Type:    ErrorTypeLoad,
		Field:   field,
		Message: message,
		Cause:   cause,
	}
}

// NewParseError 创建解析错误
func NewParseError(field, message string, cause error) *ConfigError {
	return &ConfigError{
		Type:    ErrorTypeParse,
		Field:   field,
		Message: message,
		Cause:   cause,
	}
}

// ConfigErrorCollection 配置错误集合
type ConfigErrorCollection struct {
	errors []error
}

// NewConfigErrorCollection 创建新的错误集合
func NewConfigErrorCollection() *ConfigErrorCollection {
	return &ConfigErrorCollection{
		errors: make([]error, 0),
	}
}

// Add 添加错误
func (cec *ConfigErrorCollection) Add(err error) {
	if err != nil {
		cec.errors = append(cec.errors, err)
	}
}

// AddConfigError 添加配置错误
func (cec *ConfigErrorCollection) AddConfigError(errorType, field, message string, cause error) {
	cec.Add(NewConfigError(errorType, field, message, cause))
}

// HasErrors 检查是否有错误
func (cec *ConfigErrorCollection) HasErrors() bool {
	return len(cec.errors) > 0
}

// Count 获取错误数量
func (cec *ConfigErrorCollection) Count() int {
	return len(cec.errors)
}

// Errors 获取所有错误
func (cec *ConfigErrorCollection) Errors() []error {
	return cec.errors
}

// Error 实现error接口
func (cec *ConfigErrorCollection) Error() string {
	if len(cec.errors) == 0 {
		return ""
	}

	if len(cec.errors) == 1 {
		return cec.errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("发现 %d 个配置错误:\n", len(cec.errors)))
	
	for i, err := range cec.errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}

	return sb.String()
}

// ErrorRecovery 错误恢复接口
type ErrorRecovery interface {
	// 尝试恢复错误
	Recover(err error) error
	
	// 检查错误是否可恢复
	IsRecoverable(err error) bool
	
	// 获取恢复建议
	GetRecoverySuggestion(err error) string
}

// configErrorRecovery 配置错误恢复实现
type configErrorRecovery struct {
	defaultConfig *AppConfig
}

// NewErrorRecovery 创建错误恢复器
func NewErrorRecovery() ErrorRecovery {
	return &configErrorRecovery{
		defaultConfig: DefaultConfig(),
	}
}

// Recover 尝试恢复错误
func (cer *configErrorRecovery) Recover(err error) error {
	if configErr, ok := err.(*ConfigError); ok {
		switch configErr.Type {
		case ErrorTypeValidation:
			return cer.recoverValidationError(configErr)
		case ErrorTypeLoad:
			return cer.recoverLoadError(configErr)
		case ErrorTypeParse:
			return cer.recoverParseError(configErr)
		case ErrorTypeEnvironment:
			return cer.recoverEnvironmentError(configErr)

		default:
			return fmt.Errorf("无法恢复错误类型: %s", configErr.Type)
		}
	}
	
	return fmt.Errorf("无法恢复未知错误类型")
}

// IsRecoverable 检查错误是否可恢复
func (cer *configErrorRecovery) IsRecoverable(err error) bool {
	if configErr, ok := err.(*ConfigError); ok {
		switch configErr.Type {
		case ErrorTypeValidation:
			return cer.isValidationRecoverable(configErr)
		case ErrorTypeLoad:
			return true // 加载错误通常可以通过使用默认配置恢复
		case ErrorTypeParse:
			return false // 解析错误通常需要修复配置文件
		case ErrorTypeEnvironment:
			return true // 环境错误可以通过设置默认值恢复

		case ErrorTypeNetwork:
			return false // 网络错误需要检查网络连接
		case ErrorTypeFileSystem:
			return true // 文件系统错误可以尝试创建目录等
		default:
			return false
		}
	}
	return false
}

// GetRecoverySuggestion 获取恢复建议
func (cer *configErrorRecovery) GetRecoverySuggestion(err error) string {
	if configErr, ok := err.(*ConfigError); ok {
		switch configErr.Type {
		case ErrorTypeValidation:
			return cer.getValidationSuggestion(configErr)
		case ErrorTypeLoad:
			return "建议检查配置文件路径是否正确，或使用默认配置"
		case ErrorTypeParse:
			return "建议检查配置文件语法是否正确，特别是YAML格式"
		case ErrorTypeEnvironment:
			return "建议检查环境变量设置，或使用默认环境配置"

		case ErrorTypeNetwork:
			return "建议检查网络连接和服务可用性"
		case ErrorTypeFileSystem:
			return "建议检查文件系统权限，确保目录存在且可写"
		default:
			return "建议查看详细错误信息并手动修复"
		}
	}
	return "建议查看错误日志获取更多信息"
}

// recoverValidationError 恢复验证错误
func (cer *configErrorRecovery) recoverValidationError(err *ConfigError) error {
	// 对于某些验证错误，可以使用默认值
	switch err.Field {
	case "app.log_level":
		return nil // 可以使用默认日志级别
	case "websocket.port", "web.port":
		return nil // 可以使用默认端口
	default:
		return fmt.Errorf("无法自动恢复验证错误: %s", err.Field)
	}
}

// recoverLoadError 恢复加载错误
func (cer *configErrorRecovery) recoverLoadError(err *ConfigError) error {
	// 加载错误通常可以通过使用默认配置恢复
	return nil
}

// recoverParseError 恢复解析错误
func (cer *configErrorRecovery) recoverParseError(err *ConfigError) error {
	// 解析错误通常需要修复配置文件，无法自动恢复
	return fmt.Errorf("解析错误需要手动修复配置文件")
}

// recoverEnvironmentError 恢复环境错误
func (cer *configErrorRecovery) recoverEnvironmentError(err *ConfigError) error {
	// 环境错误可以通过设置默认值恢复
	return nil
}

// recoverSecurityError 恢复安全错误
func (cer *configErrorRecovery) recoverSecurityError(err *ConfigError) error {
	// 安全错误不应该自动恢复
	return fmt.Errorf("安全错误需要手动修复")
}

// isValidationRecoverable 检查验证错误是否可恢复
func (cer *configErrorRecovery) isValidationRecoverable(err *ConfigError) bool {
	recoverableFields := []string{
		"app.log_level",
		"websocket.port",
		"web.port",
		"producer.device_count",
		"consumer.worker_count",
	}
	
	for _, field := range recoverableFields {
		if err.Field == field {
			return true
		}
	}
	
	return false
}

// getValidationSuggestion 获取验证错误建议
func (cer *configErrorRecovery) getValidationSuggestion(err *ConfigError) string {
	switch err.Field {
	case "app.log_level":
		return "建议使用有效的日志级别: debug, info, warn, error"
	case "websocket.port", "web.port":
		return "建议使用1024-65535范围内的端口号"
	case "kafka.brokers":
		return "建议检查Kafka代理地址格式: host:port"
	case "database.password":
		return "建议设置数据库密码"
	case "redis.host":
		return "建议检查Redis主机地址"
	default:
		return fmt.Sprintf("建议检查字段 %s 的配置值", err.Field)
	}
}
