package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MessageType 消息类型枚举
type MessageType string

const (
	MessageTypeDeviceData   MessageType = "device_data"
	MessageTypeAlert        MessageType = "alert"
	MessageTypeSystemStatus MessageType = "system_status"
	MessageTypeHeartbeat    MessageType = "heartbeat"
	MessageTypeSubscription MessageType = "subscription"
	MessageTypeCommand      MessageType = "command"
)

// MessagePriority 消息优先级枚举
type MessagePriority int

const (
	PriorityLow      MessagePriority = 1
	PriorityNormal   MessagePriority = 5
	PriorityHigh     MessagePriority = 8
	PriorityCritical MessagePriority = 10
)

// KafkaMessage Kafka消息包装器 - 统一消息格式
type KafkaMessage struct {
	MessageID   string      `json:"message_id" validate:"required,uuid4"`
	MessageType MessageType `json:"message_type" validate:"required"`
	Timestamp   int64       `json:"timestamp" validate:"required,min=0"`
	Source      string      `json:"source" validate:"required"`

	// 消息载荷
	Payload interface{} `json:"payload" validate:"required"`

	// 消息元数据
	Headers  map[string]string `json:"headers,omitempty"`
	Priority MessagePriority   `json:"priority"`
	TTL      int64             `json:"ttl,omitempty"`

	// 路由信息
	Topic     string `json:"topic,omitempty"`
	Partition int32  `json:"partition,omitempty"`
	Key       string `json:"key,omitempty"`

	// 处理信息
	ProcessedAt *int64 `json:"processed_at,omitempty"`
	RetryCount  int    `json:"retry_count,omitempty"`
}

// MessageHeader 消息头信息
type MessageHeader struct {
	ContentType     string `json:"content_type"`
	ContentEncoding string `json:"content_encoding,omitempty"`
	CorrelationID   string `json:"correlation_id,omitempty"`
	ReplyTo         string `json:"reply_to,omitempty"`
	UserID          string `json:"user_id,omitempty"`
	AppID           string `json:"app_id,omitempty"`
}

// MessagePayload 通用消息载荷接口
type MessagePayload interface {
	GetType() string
	Validate() error
	ToJSON() ([]byte, error)
}

// DeviceDataPayload 设备数据载荷
type DeviceDataPayload struct {
	Device    *Device `json:"device" validate:"required"`
	BatchSize int     `json:"batch_size,omitempty"`
	BatchID   string  `json:"batch_id,omitempty"`
}

// AlertPayload 告警载荷
type AlertPayload struct {
	Alert    *Alert `json:"alert" validate:"required"`
	DeviceID string `json:"device_id" validate:"required"`
	Resolved bool   `json:"resolved,omitempty"`
}

// SystemStatusPayload 系统状态载荷
type SystemStatusPayload struct {
	ServiceName string                 `json:"service_name" validate:"required"`
	Status      string                 `json:"status" validate:"required,oneof=healthy degraded unhealthy"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
	Timestamp   int64                  `json:"timestamp" validate:"required,min=0"`
}

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	MessageID   string      `json:"message_id" validate:"required,uuid4"`
	MessageType MessageType `json:"message_type" validate:"required"`
	Timestamp   int64       `json:"timestamp" validate:"required,min=0"`

	// WebSocket特有字段
	ClientID  string   `json:"client_id,omitempty"`
	SessionID string   `json:"session_id,omitempty"`
	Action    WSAction `json:"action" validate:"required"`

	// 消息载荷
	Payload interface{} `json:"payload,omitempty"`

	// 元数据
	Headers  map[string]string `json:"headers,omitempty"`
	Priority MessagePriority   `json:"priority"`

	// 响应信息
	RequestID string `json:"request_id,omitempty"`
	Success   bool   `json:"success,omitempty"`
	Error     string `json:"error,omitempty"`
}

// WSAction WebSocket动作类型
type WSAction string

const (
	WSActionSubscribe   WSAction = "subscribe"
	WSActionUnsubscribe WSAction = "unsubscribe"
	WSActionPublish     WSAction = "publish"
	WSActionHeartbeat   WSAction = "heartbeat"
	WSActionAuth        WSAction = "auth"
	WSActionResponse    WSAction = "response"
	WSActionBroadcast   WSAction = "broadcast"
)

// SubscriptionPayload 订阅载荷
type SubscriptionPayload struct {
	Type    SubscriptionType       `json:"type" validate:"required"`
	Target  string                 `json:"target" validate:"required"`
	Filters []string               `json:"filters,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// SubscriptionType 订阅类型
type SubscriptionType string

const (
	SubscriptionTypeDevice SubscriptionType = "device"
	SubscriptionTypeRoom   SubscriptionType = "room"
	SubscriptionTypeAlert  SubscriptionType = "alert"
	SubscriptionTypeAll    SubscriptionType = "all"
)

// HeartbeatPayload 心跳载荷
type HeartbeatPayload struct {
	ClientID  string `json:"client_id" validate:"required"`
	Timestamp int64  `json:"timestamp" validate:"required,min=0"`
	Status    string `json:"status" validate:"oneof=alive busy idle"`
}

// NewKafkaMessage 创建新的Kafka消息
func NewKafkaMessage(messageType MessageType, source string, payload interface{}) *KafkaMessage {
	return &KafkaMessage{
		MessageID:   uuid.New().String(),
		MessageType: messageType,
		Timestamp:   time.Now().Unix(),
		Source:      source,
		Payload:     payload,
		Headers:     make(map[string]string),
		Priority:    PriorityNormal,
		RetryCount:  0,
	}
}

// NewWebSocketMessage 创建新的WebSocket消息
func NewWebSocketMessage(messageType MessageType, action WSAction, payload interface{}) *WebSocketMessage {
	return &WebSocketMessage{
		MessageID:   uuid.New().String(),
		MessageType: messageType,
		Timestamp:   time.Now().Unix(),
		Action:      action,
		Payload:     payload,
		Headers:     make(map[string]string),
		Priority:    PriorityNormal,
		Success:     true,
	}
}

// SetHeader 设置消息头
func (km *KafkaMessage) SetHeader(key, value string) {
	if km.Headers == nil {
		km.Headers = make(map[string]string)
	}
	km.Headers[key] = value
}

// GetHeader 获取消息头
func (km *KafkaMessage) GetHeader(key string) (string, bool) {
	if km.Headers == nil {
		return "", false
	}
	value, exists := km.Headers[key]
	return value, exists
}

// SetPriority 设置消息优先级
func (km *KafkaMessage) SetPriority(priority MessagePriority) {
	km.Priority = priority
}

// SetTTL 设置消息生存时间（秒）
func (km *KafkaMessage) SetTTL(ttl time.Duration) {
	// 使用毫秒转换以避免精度损失，然后除以1000得到秒
	if ttl < time.Second {
		// 对于小于1秒的时间，设置为1秒以确保正常工作
		km.TTL = 1
	} else {
		km.TTL = int64(ttl.Seconds())
	}
}

// IsExpired 检查消息是否过期
func (km *KafkaMessage) IsExpired() bool {
	if km.TTL <= 0 {
		return false
	}
	// 将时间戳转换为秒进行比较
	timestampInSeconds := km.Timestamp
	if km.Timestamp > 1e12 { // 如果是毫秒时间戳，转换为秒
		timestampInSeconds = km.Timestamp / 1000
	}
	return time.Now().Unix()-timestampInSeconds >= km.TTL
}

// IncrementRetry 增加重试次数
func (km *KafkaMessage) IncrementRetry() {
	km.RetryCount++
}

// SetProcessed 标记消息已处理
func (km *KafkaMessage) SetProcessed() {
	now := time.Now().Unix()
	km.ProcessedAt = &now
}

// ToJSON 转换为JSON
func (km *KafkaMessage) ToJSON() ([]byte, error) {
	return json.Marshal(km)
}

// FromJSON 从JSON解析
func (km *KafkaMessage) FromJSON(data []byte) error {
	return json.Unmarshal(data, km)
}

// Validate 验证消息
func (km *KafkaMessage) Validate() error {
	if km.MessageID == "" {
		return fmt.Errorf("message_id is required")
	}
	if km.MessageType == "" {
		return fmt.Errorf("message_type is required")
	}
	if km.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}
	if km.Source == "" {
		return fmt.Errorf("source is required")
	}
	if km.Payload == nil {
		return fmt.Errorf("payload is required")
	}
	if km.Priority < 1 || km.Priority > 10 {
		return fmt.Errorf("priority must be between 1 and 10")
	}
	return nil
}

// WebSocket消息方法

// SetClientID 设置客户端ID
func (wsm *WebSocketMessage) SetClientID(clientID string) {
	wsm.ClientID = clientID
}

// SetSessionID 设置会话ID
func (wsm *WebSocketMessage) SetSessionID(sessionID string) {
	wsm.SessionID = sessionID
}

// SetError 设置错误信息
func (wsm *WebSocketMessage) SetError(err error) {
	wsm.Success = false
	wsm.Error = err.Error()
}

// SetSuccess 设置成功状态
func (wsm *WebSocketMessage) SetSuccess(success bool) {
	wsm.Success = success
	if success {
		wsm.Error = ""
	}
}

// ToJSON WebSocket消息转JSON
func (wsm *WebSocketMessage) ToJSON() ([]byte, error) {
	return json.Marshal(wsm)
}

// FromJSON WebSocket消息从JSON解析
func (wsm *WebSocketMessage) FromJSON(data []byte) error {
	return json.Unmarshal(data, wsm)
}

// Validate 验证WebSocket消息
func (wsm *WebSocketMessage) Validate() error {
	if wsm.MessageID == "" {
		return fmt.Errorf("message_id is required")
	}
	if wsm.MessageType == "" {
		return fmt.Errorf("message_type is required")
	}
	if wsm.Action == "" {
		return fmt.Errorf("action is required")
	}
	if wsm.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}
	if wsm.Priority < 1 || wsm.Priority > 10 {
		return fmt.Errorf("priority must be between 1 and 10")
	}
	return nil
}

// 载荷类型实现

// GetType 实现MessagePayload接口
func (ddp *DeviceDataPayload) GetType() string {
	return string(MessageTypeDeviceData)
}

// Validate 验证设备数据载荷
func (ddp *DeviceDataPayload) Validate() error {
	if ddp.Device == nil {
		return fmt.Errorf("device is required")
	}
	validationResult := ddp.Device.Validate()
	if !validationResult.IsValid {
		return fmt.Errorf("device validation failed: %d errors", len(validationResult.Errors))
	}
	return nil
}

// ToJSON 设备数据载荷转JSON
func (ddp *DeviceDataPayload) ToJSON() ([]byte, error) {
	return json.Marshal(ddp)
}

// GetType 实现MessagePayload接口
func (ap *AlertPayload) GetType() string {
	return string(MessageTypeAlert)
}

// Validate 验证告警载荷
func (ap *AlertPayload) Validate() error {
	if ap.Alert == nil {
		return fmt.Errorf("alert is required")
	}
	if ap.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	return ap.Alert.Validate()
}

// ToJSON 告警载荷转JSON
func (ap *AlertPayload) ToJSON() ([]byte, error) {
	return json.Marshal(ap)
}

// GetType 实现MessagePayload接口
func (ssp *SystemStatusPayload) GetType() string {
	return string(MessageTypeSystemStatus)
}

// Validate 验证系统状态载荷
func (ssp *SystemStatusPayload) Validate() error {
	if ssp.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}
	if ssp.Status == "" {
		return fmt.Errorf("status is required")
	}
	if ssp.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}
	return nil
}

// ToJSON 系统状态载荷转JSON
func (ssp *SystemStatusPayload) ToJSON() ([]byte, error) {
	return json.Marshal(ssp)
}
