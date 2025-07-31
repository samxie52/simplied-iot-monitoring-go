package config

import (
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidateHostnamePort 验证主机名:端口格式
func ValidateHostnamePort(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return false
	}

	// 验证主机名或IP地址
	if net.ParseIP(host) == nil {
		// 不是IP地址，验证主机名格式
		if !isValidHostnameFormat(host) {
			return false
		}
	}

	// 验证端口范围
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return false
	}

	return portNum >= 1 && portNum <= 65535
}

// ValidateSemVer 验证语义版本号格式
func ValidateSemVer(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	// 语义版本号正则表达式
	semverRegex := regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	return semverRegex.MatchString(value)
}

// ValidateFilePath 验证文件路径
func ValidateFilePath(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	// 检查路径格式
	if !filepath.IsAbs(value) && !isValidRelativePath(value) {
		return false
	}

	// 检查路径中是否包含非法字符
	if containsIllegalChars(value) {
		return false
	}

	return true
}

// ValidateDirPath 验证目录路径
func ValidateDirPath(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	// 使用与文件路径相同的验证逻辑
	return ValidateFilePath(fl)
}

// ValidateFileExists 验证文件是否存在
func ValidateFileExists(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	// 检查文件是否存在
	info, err := os.Stat(value)
	if err != nil {
		return false
	}

	// 确保是文件而不是目录
	return !info.IsDir()
}

// ValidateDirExists 验证目录是否存在
func ValidateDirExists(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	// 检查目录是否存在
	info, err := os.Stat(value)
	if err != nil {
		return false
	}

	// 确保是目录而不是文件
	return info.IsDir()
}

// ValidateReachable 验证网络地址是否可达
func ValidateReachable(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	// 解析URL
	parsedURL, err := url.Parse(value)
	if err != nil {
		return false
	}

	// 检查协议
	if parsedURL.Scheme == "" {
		return false
	}

	// 检查主机
	if parsedURL.Host == "" {
		return false
	}

	// 这里可以添加实际的网络连接测试
	// 但在配置验证阶段，通常只做格式验证以避免网络依赖
	return true
}

// ValidatePortRange 验证端口范围
func ValidatePortRange(fl validator.FieldLevel) bool {
	port := int(fl.Field().Int())
	return port >= 1024 && port <= 65535
}

// ValidateLogLevel 验证日志级别
func ValidateLogLevel(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}

	for _, level := range validLevels {
		if strings.EqualFold(value, level) {
			return true
		}
	}

	return false
}

// ValidateEnvironment 验证环境类型
func ValidateEnvironment(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	env := Environment(value)
	return env.IsValid()
}

// ValidateCompressionType 验证压缩类型
func ValidateCompressionType(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validTypes := []string{"none", "gzip", "snappy", "lz4", "zstd"}

	for _, compressionType := range validTypes {
		if strings.EqualFold(value, compressionType) {
			return true
		}
	}

	return false
}

// ValidateSSLMode 验证SSL模式
func ValidateSSLMode(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validModes := []string{"disable", "require", "verify-ca", "verify-full"}

	for _, mode := range validModes {
		if strings.EqualFold(value, mode) {
			return true
		}
	}

	return false
}

// ValidateAlertSeverity 验证告警严重级别
func ValidateAlertSeverity(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validSeverities := []string{"low", "medium", "high", "critical"}

	for _, severity := range validSeverities {
		if strings.EqualFold(value, severity) {
			return true
		}
	}

	return false
}

// ValidateEncryptionAlgorithm 验证加密算法
func ValidateEncryptionAlgorithm(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validAlgorithms := []string{"AES-128", "AES-192", "AES-256"}

	for _, algorithm := range validAlgorithms {
		if strings.EqualFold(value, algorithm) {
			return true
		}
	}

	return false
}

// ValidateKafkaProtocol 验证Kafka安全协议
func ValidateKafkaProtocol(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validProtocols := []string{"PLAINTEXT", "SASL_PLAINTEXT", "SASL_SSL", "SSL"}

	for _, protocol := range validProtocols {
		if strings.EqualFold(value, protocol) {
			return true
		}
	}

	return false
}

// ValidateJSONFormat 验证JSON格式
func ValidateJSONFormat(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validFormats := []string{"text", "json"}

	for _, format := range validFormats {
		if strings.EqualFold(value, format) {
			return true
		}
	}

	return false
}

// ValidateOutputType 验证输出类型
func ValidateOutputType(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validOutputs := []string{"stdout", "stderr", "file"}

	for _, output := range validOutputs {
		if strings.EqualFold(value, output) {
			return true
		}
	}

	return false
}

// ValidateDataSize 验证数据大小格式（如100MB, 1GB等）
func ValidateDataSize(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	// 数据大小正则表达式
	dataSizeRegex := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(B|KB|MB|GB|TB)$`)
	return dataSizeRegex.MatchString(strings.ToUpper(value))
}

// ValidateTimeDuration 验证时间间隔格式（如1h, 30m, 5s等）
func ValidateTimeDuration(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	// 时间间隔正则表达式
	durationRegex := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(ns|us|µs|ms|s|m|h|d)$`)
	return durationRegex.MatchString(value)
}

// ValidateIPAddress 验证IP地址
func ValidateIPAddress(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	return net.ParseIP(value) != nil
}

// ValidateHostname 验证主机名
func ValidateHostname(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	return isValidHostnameFormat(value)
}

// ValidateURL 验证URL格式
func ValidateURL(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	parsedURL, err := url.Parse(value)
	if err != nil {
		return false
	}

	return parsedURL.Scheme != "" && parsedURL.Host != ""
}

// ValidateURI 验证URI格式
func ValidateURI(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	_, err := url.ParseRequestURI(value)
	return err == nil
}

// 辅助函数

// isValidHostnameFormat 验证主机名格式
func isValidHostnameFormat(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}

	// 主机名正则表达式
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	return hostnameRegex.MatchString(hostname)
}

// isValidRelativePath 验证相对路径格式
func isValidRelativePath(path string) bool {
	// 检查是否包含".."等不安全的路径组件
	if strings.Contains(path, "..") {
		return false
	}

	// 检查路径分隔符
	if strings.Contains(path, "\\") && filepath.Separator != '\\' {
		return false
	}

	return true
}

// containsIllegalChars 检查路径是否包含非法字符
func containsIllegalChars(path string) bool {
	// 在不同操作系统上，非法字符可能不同
	// 这里列出一些常见的非法字符
	illegalChars := []string{"<", ">", ":", "\"", "|", "?", "*"}

	for _, char := range illegalChars {
		if strings.Contains(path, char) {
			return true
		}
	}

	return false
}
