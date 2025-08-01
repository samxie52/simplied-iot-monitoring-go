package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"simplied-iot-monitoring-go/internal/models"
)

// DefaultMessageProcessor 默认消息处理器实现
type DefaultMessageProcessor struct {
	// 统计信息
	processedCount int64
	errorCount     int64
	totalLatency   int64
	mutex          sync.RWMutex

	// 处理器配置
	config ProcessorConfig

	// 数据处理管道
	validators    []MessageValidator
	transformers  []MessageTransformer
	handlers      map[models.MessageType]TypedMessageHandler
	
	// 缓存和存储
	cache   CacheManager
	storage StorageManager
}

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	MaxRetries          int
	RetryDelay          time.Duration
	ProcessingTimeout   time.Duration
	EnableValidation    bool
	EnableTransformation bool
	EnableCaching       bool
	BatchSize           int
	FlushInterval       time.Duration
}

// MessageValidator 消息验证器接口
type MessageValidator interface {
	Validate(ctx context.Context, message *models.KafkaMessage) error
	GetValidatorName() string
}

// MessageTransformer 消息转换器接口
type MessageTransformer interface {
	Transform(ctx context.Context, message *models.KafkaMessage) (*models.KafkaMessage, error)
	GetTransformerName() string
}

// TypedMessageHandler 类型化消息处理器接口
type TypedMessageHandler interface {
	HandleMessage(ctx context.Context, message *models.KafkaMessage) error
	GetHandlerType() models.MessageType
}

// CacheManager 缓存管理器接口
type CacheManager interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (interface{}, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) bool
}

// StorageManager 存储管理器接口
type StorageManager interface {
	Store(ctx context.Context, message *models.KafkaMessage) error
	Retrieve(ctx context.Context, messageID string) (*models.KafkaMessage, error)
	Delete(ctx context.Context, messageID string) error
}

// NewDefaultMessageProcessor 创建默认消息处理器
func NewDefaultMessageProcessor(config ProcessorConfig) *DefaultMessageProcessor {
	return &DefaultMessageProcessor{
		config:       config,
		validators:   make([]MessageValidator, 0),
		transformers: make([]MessageTransformer, 0),
		handlers:     make(map[models.MessageType]TypedMessageHandler),
	}
}

// ProcessMessage 处理消息
func (dmp *DefaultMessageProcessor) ProcessMessage(ctx context.Context, message *models.KafkaMessage) error {
	startTime := time.Now()
	
	// 创建带超时的上下文
	processCtx, cancel := context.WithTimeout(ctx, dmp.config.ProcessingTimeout)
	defer cancel()

	// 处理消息
	err := dmp.processMessageWithRetry(processCtx, message)
	
	// 更新统计信息
	processingTime := time.Since(startTime)
	atomic.AddInt64(&dmp.totalLatency, processingTime.Nanoseconds())
	
	if err != nil {
		atomic.AddInt64(&dmp.errorCount, 1)
		return err
	}
	
	atomic.AddInt64(&dmp.processedCount, 1)
	return nil
}

// processMessageWithRetry 带重试的消息处理
func (dmp *DefaultMessageProcessor) processMessageWithRetry(ctx context.Context, message *models.KafkaMessage) error {
	var lastErr error
	
	for attempt := 0; attempt <= dmp.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// 等待重试延迟
			select {
			case <-time.After(dmp.config.RetryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
			log.Printf("重试处理消息 %s (第%d次)", message.MessageID, attempt)
		}

		err := dmp.doProcessMessage(ctx, message)
		if err == nil {
			return nil
		}

		lastErr = err
		log.Printf("处理消息失败 (第%d次): %v", attempt+1, err)
	}

	return fmt.Errorf("消息处理失败，已重试%d次: %w", dmp.config.MaxRetries, lastErr)
}

// doProcessMessage 执行消息处理
func (dmp *DefaultMessageProcessor) doProcessMessage(ctx context.Context, message *models.KafkaMessage) error {
	// 1. 验证消息
	if dmp.config.EnableValidation {
		if err := dmp.validateMessage(ctx, message); err != nil {
			return fmt.Errorf("消息验证失败: %w", err)
		}
	}

	// 2. 转换消息
	processedMessage := message
	if dmp.config.EnableTransformation {
		transformed, err := dmp.transformMessage(ctx, message)
		if err != nil {
			return fmt.Errorf("消息转换失败: %w", err)
		}
		processedMessage = transformed
	}

	// 3. 缓存处理
	if dmp.config.EnableCaching && dmp.cache != nil {
		if err := dmp.handleCaching(ctx, processedMessage); err != nil {
			log.Printf("缓存处理警告: %v", err)
		}
	}

	// 4. 类型化处理
	if err := dmp.handleTypedMessage(ctx, processedMessage); err != nil {
		return fmt.Errorf("类型化处理失败: %w", err)
	}

	// 5. 存储处理
	if dmp.storage != nil {
		if err := dmp.storage.Store(ctx, processedMessage); err != nil {
			log.Printf("存储处理警告: %v", err)
		}
	}

	return nil
}

// validateMessage 验证消息
func (dmp *DefaultMessageProcessor) validateMessage(ctx context.Context, message *models.KafkaMessage) error {
	for _, validator := range dmp.validators {
		if err := validator.Validate(ctx, message); err != nil {
			return fmt.Errorf("验证器 %s 失败: %w", validator.GetValidatorName(), err)
		}
	}
	return nil
}

// transformMessage 转换消息
func (dmp *DefaultMessageProcessor) transformMessage(ctx context.Context, message *models.KafkaMessage) (*models.KafkaMessage, error) {
	result := message
	
	for _, transformer := range dmp.transformers {
		transformed, err := transformer.Transform(ctx, result)
		if err != nil {
			return nil, fmt.Errorf("转换器 %s 失败: %w", transformer.GetTransformerName(), err)
		}
		result = transformed
	}
	
	return result, nil
}

// handleCaching 处理缓存
func (dmp *DefaultMessageProcessor) handleCaching(ctx context.Context, message *models.KafkaMessage) error {
	cacheKey := fmt.Sprintf("message:%s", message.MessageID)
	
	// 检查是否已处理
	if dmp.cache.Exists(ctx, cacheKey) {
		return fmt.Errorf("消息已处理: %s", message.MessageID)
	}
	
	// 缓存消息
	ttl := 1 * time.Hour // 默认TTL
	if message.TTL > 0 {
		ttl = time.Duration(message.TTL) * time.Second
	}
	
	return dmp.cache.Set(ctx, cacheKey, message, ttl)
}

// handleTypedMessage 处理类型化消息
func (dmp *DefaultMessageProcessor) handleTypedMessage(ctx context.Context, message *models.KafkaMessage) error {
	handler, exists := dmp.handlers[message.MessageType]
	if !exists {
		return fmt.Errorf("未找到消息类型 %s 的处理器", message.MessageType)
	}
	
	return handler.HandleMessage(ctx, message)
}

// GetStats 获取处理器统计信息
func (dmp *DefaultMessageProcessor) GetStats() ProcessorStats {
	processedCount := atomic.LoadInt64(&dmp.processedCount)
	errorCount := atomic.LoadInt64(&dmp.errorCount)
	totalLatency := atomic.LoadInt64(&dmp.totalLatency)
	
	var avgLatency float64
	if processedCount > 0 {
		avgLatency = float64(totalLatency) / float64(processedCount) / 1000000 // 转换为毫秒
	}
	
	return ProcessorStats{
		ProcessedCount: processedCount,
		ErrorCount:     errorCount,
		AvgLatencyMs:   avgLatency,
	}
}

// AddValidator 添加验证器
func (dmp *DefaultMessageProcessor) AddValidator(validator MessageValidator) {
	dmp.mutex.Lock()
	defer dmp.mutex.Unlock()
	dmp.validators = append(dmp.validators, validator)
}

// AddTransformer 添加转换器
func (dmp *DefaultMessageProcessor) AddTransformer(transformer MessageTransformer) {
	dmp.mutex.Lock()
	defer dmp.mutex.Unlock()
	dmp.transformers = append(dmp.transformers, transformer)
}

// RegisterHandler 注册类型化处理器
func (dmp *DefaultMessageProcessor) RegisterHandler(handler TypedMessageHandler) {
	dmp.mutex.Lock()
	defer dmp.mutex.Unlock()
	dmp.handlers[handler.GetHandlerType()] = handler
}

// SetCacheManager 设置缓存管理器
func (dmp *DefaultMessageProcessor) SetCacheManager(cache CacheManager) {
	dmp.cache = cache
}

// SetStorageManager 设置存储管理器
func (dmp *DefaultMessageProcessor) SetStorageManager(storage StorageManager) {
	dmp.storage = storage
}

// 具体的验证器实现

// BasicMessageValidator 基础消息验证器
type BasicMessageValidator struct{}

func (bmv *BasicMessageValidator) Validate(ctx context.Context, message *models.KafkaMessage) error {
	return message.Validate()
}

func (bmv *BasicMessageValidator) GetValidatorName() string {
	return "BasicMessageValidator"
}

// SchemaValidator 模式验证器
type SchemaValidator struct {
	schemas map[models.MessageType]interface{}
}

func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		schemas: make(map[models.MessageType]interface{}),
	}
}

func (sv *SchemaValidator) Validate(ctx context.Context, message *models.KafkaMessage) error {
	schema, exists := sv.schemas[message.MessageType]
	if !exists {
		return nil // 没有模式定义，跳过验证
	}
	
	// 这里应该实现具体的模式验证逻辑
	// 例如使用JSON Schema验证
	_ = schema
	return nil
}

func (sv *SchemaValidator) GetValidatorName() string {
	return "SchemaValidator"
}

func (sv *SchemaValidator) RegisterSchema(messageType models.MessageType, schema interface{}) {
	sv.schemas[messageType] = schema
}

// 具体的转换器实现

// TimestampTransformer 时间戳转换器
type TimestampTransformer struct{}

func (tt *TimestampTransformer) Transform(ctx context.Context, message *models.KafkaMessage) (*models.KafkaMessage, error) {
	// 确保时间戳是毫秒级别
	if message.Timestamp < 1000000000000 { // 小于2001年的时间戳，可能是秒级
		message.Timestamp *= 1000
	}
	
	return message, nil
}

func (tt *TimestampTransformer) GetTransformerName() string {
	return "TimestampTransformer"
}

// PayloadTransformer 载荷转换器
type PayloadTransformer struct{}

func (pt *PayloadTransformer) Transform(ctx context.Context, message *models.KafkaMessage) (*models.KafkaMessage, error) {
	// 根据消息类型转换载荷
	switch message.MessageType {
	case models.MessageTypeDeviceData:
		return pt.transformDeviceData(message)
	case models.MessageTypeAlert:
		return pt.transformAlert(message)
	default:
		return message, nil
	}
}

func (pt *PayloadTransformer) transformDeviceData(message *models.KafkaMessage) (*models.KafkaMessage, error) {
	// 确保载荷是正确的设备数据格式
	if payloadMap, ok := message.Payload.(map[string]interface{}); ok {
		var payload models.DeviceDataPayload
		payloadBytes, err := json.Marshal(payloadMap)
		if err != nil {
			return nil, err
		}
		
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, err
		}
		
		message.Payload = &payload
	}
	
	return message, nil
}

func (pt *PayloadTransformer) transformAlert(message *models.KafkaMessage) (*models.KafkaMessage, error) {
	// 确保载荷是正确的告警格式
	if payloadMap, ok := message.Payload.(map[string]interface{}); ok {
		var payload models.AlertPayload
		payloadBytes, err := json.Marshal(payloadMap)
		if err != nil {
			return nil, err
		}
		
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, err
		}
		
		message.Payload = &payload
	}
	
	return message, nil
}

func (pt *PayloadTransformer) GetTransformerName() string {
	return "PayloadTransformer"
}

// 具体的处理器实现

// DeviceDataHandler 设备数据处理器
type DeviceDataHandler struct {
	deviceManager DeviceManager
}

type DeviceManager interface {
	UpdateDevice(ctx context.Context, device *models.Device) error
	GetDevice(ctx context.Context, deviceID string) (*models.Device, error)
}

func NewDeviceDataHandler(deviceManager DeviceManager) *DeviceDataHandler {
	return &DeviceDataHandler{
		deviceManager: deviceManager,
	}
}

func (ddh *DeviceDataHandler) HandleMessage(ctx context.Context, message *models.KafkaMessage) error {
	payload, ok := message.Payload.(*models.DeviceDataPayload)
	if !ok {
		return fmt.Errorf("无效的设备数据载荷类型")
	}
	
	// 检查设备数据是否为空
	if payload.Device == nil {
		return fmt.Errorf("设备数据为空")
	}
	
	log.Printf("处理设备数据: DeviceID=%s", payload.Device.DeviceID)
	
	// 更新设备状态
	if ddh.deviceManager != nil {
		if err := ddh.deviceManager.UpdateDevice(ctx, payload.Device); err != nil {
			return fmt.Errorf("更新设备失败: %w", err)
		}
	}
	
	return nil
}

func (ddh *DeviceDataHandler) GetHandlerType() models.MessageType {
	return models.MessageTypeDeviceData
}

// AlertHandler 告警处理器
type AlertHandler struct {
	alertManager AlertManager
}

type AlertManager interface {
	ProcessAlert(ctx context.Context, alert *models.Alert) error
	ResolveAlert(ctx context.Context, alertID string) error
}

func NewAlertHandler(alertManager AlertManager) *AlertHandler {
	return &AlertHandler{
		alertManager: alertManager,
	}
}

func (ah *AlertHandler) HandleMessage(ctx context.Context, message *models.KafkaMessage) error {
	payload, ok := message.Payload.(*models.AlertPayload)
	if !ok {
		return fmt.Errorf("无效的告警载荷类型")
	}
	
	// 检查告警数据是否为空
	if payload.Alert == nil {
		return fmt.Errorf("告警数据为空")
	}
	
	log.Printf("处理告警: AlertID=%s, DeviceID=%s", payload.Alert.AlertID, payload.DeviceID)
	
	// 处理告警
	if ah.alertManager != nil {
		if payload.Resolved {
			if err := ah.alertManager.ResolveAlert(ctx, payload.Alert.AlertID); err != nil {
				return fmt.Errorf("解决告警失败: %w", err)
			}
		} else {
			if err := ah.alertManager.ProcessAlert(ctx, payload.Alert); err != nil {
				return fmt.Errorf("处理告警失败: %w", err)
			}
		}
	}
	
	return nil
}

func (ah *AlertHandler) GetHandlerType() models.MessageType {
	return models.MessageTypeAlert
}
