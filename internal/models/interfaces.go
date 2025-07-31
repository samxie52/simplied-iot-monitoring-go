package models

// DataConverter 数据转换接口 - 格式转换
type DataConverter interface {
	ToJSON() ([]byte, error)
	FromJSON(data []byte) error
	Validate() *ValidationResult
}

// DeviceDataProcessor 设备数据处理器接口
type DeviceDataProcessor interface {
	ProcessDeviceData(device *Device) error
	ValidateDevice(device *Device) (*ValidationResult, error)
	TransformToMessage(device *Device, msgType string) (interface{}, error)
	HandleValidationError(err *ValidationError) error
}

// SensorRegistry 传感器注册表接口
type SensorRegistry interface {
	RegisterSensorType(sensorType string, factory SensorFactory) error
	CreateSensor(sensorType string, config map[string]interface{}) (interface{}, error)
	GetSupportedTypes() []string
	ValidateSensorConfig(sensorType string, config map[string]interface{}) error
}

// SensorFactory 传感器工厂接口
type SensorFactory interface {
	CreateSensor(config map[string]interface{}) (interface{}, error)
	ValidateConfig(config map[string]interface{}) error
	GetSensorType() string
}

// DataValidator 数据验证器接口
type DataValidator interface {
	ValidateStruct(data interface{}) *ValidationResult
	ValidateBusinessRules(data interface{}) *ValidationResult
	ValidateRange(value float64, min, max float64, fieldName string) *ValidationResult
	ValidateTimestamp(timestamp int64, fieldName string) *ValidationResult
}

// MessageRouter 消息路由器接口
type MessageRouter interface {
	RouteMessage(message interface{}, messageType string) error
	RegisterHandler(messageType string, handler MessageHandler) error
	GetSupportedMessageTypes() []string
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleMessage(message interface{}) error
	GetMessageType() string
	ValidateMessage(message interface{}) error
}

// DeviceManager 设备管理器接口
type DeviceManager interface {
	AddDevice(device *Device) error
	UpdateDevice(device *Device) error
	RemoveDevice(deviceID string) error
	GetDevice(deviceID string) (*Device, error)
	GetAllDevices() ([]*Device, error)
	GetDevicesByType(deviceType string) ([]*Device, error)
	GetDevicesByStatus(status DeviceStatus) ([]*Device, error)
	GetDevicesByLocation(building string, floor int, room string) ([]*Device, error)
}

// SensorDataManager 传感器数据管理器接口
type SensorDataManager interface {
	UpdateSensorData(deviceID string, sensorData *SensorData) error
	GetSensorData(deviceID string) (*SensorData, error)
	GetSensorHistory(deviceID string, sensorType string, startTime, endTime int64) ([]interface{}, error)
	ValidateSensorData(sensorData *SensorData) *ValidationResult
	CleanupOldData(maxAge int64) error
}

// AlertManager 告警管理器接口
type AlertManager interface {
	CreateAlert(deviceID string, alertType string, severity string, message string) error
	UpdateAlertStatus(alertID string, status string) error
	GetActiveAlerts() ([]interface{}, error)
	GetAlertsByDevice(deviceID string) ([]interface{}, error)
	GetAlertsByType(alertType string) ([]interface{}, error)
	ResolveAlert(alertID string) error
}

// ConfigManager 配置管理器接口
type ConfigManager interface {
	LoadConfig(configPath string) error
	GetConfig() interface{}
	ValidateConfig() error
	ReloadConfig() error
	WatchConfig(callback func()) error
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	RecordDeviceCount(count int)
	RecordSensorReading(deviceID string, sensorType string, value float64)
	RecordValidationError(errorType string, field string)
	RecordProcessingTime(operation string, duration int64)
	GetMetrics() map[string]interface{}
}

// Logger 日志记录器接口
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
}

// CacheManager 缓存管理器接口
type CacheManager interface {
	Set(key string, value interface{}, ttl int64) error
	Get(key string) (interface{}, error)
	Delete(key string) error
	Clear() error
	GetKeys(pattern string) ([]string, error)
	Exists(key string) bool
}

// DatabaseManager 数据库管理器接口
type DatabaseManager interface {
	Connect() error
	Disconnect() error
	SaveDevice(device *Device) error
	LoadDevice(deviceID string) (*Device, error)
	SaveSensorData(deviceID string, sensorData *SensorData) error
	LoadSensorData(deviceID string) (*SensorData, error)
	Query(query string, args ...interface{}) ([]map[string]interface{}, error)
	Execute(query string, args ...interface{}) error
}

// EventPublisher 事件发布器接口
type EventPublisher interface {
	PublishDeviceEvent(deviceID string, eventType string, data interface{}) error
	PublishSensorEvent(deviceID string, sensorType string, data interface{}) error
	PublishAlertEvent(alertID string, eventType string, data interface{}) error
	Subscribe(eventType string, handler EventHandler) error
	Unsubscribe(eventType string, handler EventHandler) error
}

// EventHandler 事件处理器接口
type EventHandler interface {
	HandleEvent(eventType string, data interface{}) error
	GetEventTypes() []string
}

// HealthChecker 健康检查器接口
type HealthChecker interface {
	CheckHealth() error
	GetHealthStatus() map[string]interface{}
	RegisterHealthCheck(name string, checker func() error) error
	UnregisterHealthCheck(name string) error
}

// SecurityManager 安全管理器接口 (预留)
type SecurityManager interface {
	ValidateToken(token string) error
	GenerateToken(userID string) (string, error)
	HashPassword(password string) (string, error)
	VerifyPassword(password, hash string) error
	EncryptData(data []byte) ([]byte, error)
	DecryptData(data []byte) ([]byte, error)
}

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(key string) bool
	SetLimit(key string, limit int, window int64) error
	GetLimit(key string) (int, int64, error)
	Reset(key string) error
}

// CircuitBreaker 熔断器接口
type CircuitBreaker interface {
	Call(operation func() error) error
	GetState() string
	Reset() error
	ForceOpen() error
	ForceClose() error
}

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	SelectEndpoint(endpoints []string) (string, error)
	AddEndpoint(endpoint string) error
	RemoveEndpoint(endpoint string) error
	GetEndpoints() []string
	UpdateEndpointHealth(endpoint string, healthy bool) error
}
