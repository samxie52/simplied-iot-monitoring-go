package websocket

import (
	"encoding/json"
	"time"

	"simplied-iot-monitoring-go/internal/models"
)

// MessageType WebSocket消息类型
type MessageType string

const (
	// 客户端到服务器消息类型
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"
	MessageTypePing        MessageType = "ping"
	MessageTypeAuth        MessageType = "auth"
	MessageTypeFilter      MessageType = "filter"
	
	// 服务器到客户端消息类型
	MessageTypeDeviceData   MessageType = "device_data"
	MessageTypeAlert        MessageType = "alert"
	MessageTypeSystemStatus MessageType = "system_status"
	MessageTypePong         MessageType = "pong"
	MessageTypeAck          MessageType = "ack"
	MessageTypeError        MessageType = "error"
	MessageTypeWelcome      MessageType = "welcome"
)

// Message WebSocket消息基础结构
type Message struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id,omitempty"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// SubscribeMessage 订阅消息
type SubscribeMessage struct {
	Type    MessageType        `json:"type"`
	ID      string             `json:"id"`
	Filters *SubscriptionFilter `json:"filters"`
}

// UnsubscribeMessage 取消订阅消息
type UnsubscribeMessage struct {
	Type         MessageType `json:"type"`
	ID           string      `json:"id"`
	Subscription string      `json:"subscription"`
}

// AuthMessage 认证消息
type AuthMessage struct {
	Type     MessageType `json:"type"`
	ID       string      `json:"id"`
	Token    string      `json:"token"`
	UserID   string      `json:"user_id,omitempty"`
	ClientID string      `json:"client_id,omitempty"`
}

// FilterMessage 过滤器消息
type FilterMessage struct {
	Type   MessageType   `json:"type"`
	ID     string        `json:"id"`
	Filter *ClientFilter `json:"filter"`
}

// DeviceDataMessage 设备数据消息
type DeviceDataMessage struct {
	Type      MessageType                `json:"type"`
	ID        string                     `json:"id,omitempty"`
	Timestamp int64                      `json:"timestamp"`
	Data      *models.DeviceDataPayload  `json:"data"`
}

// AlertMessage 告警消息
type AlertMessage struct {
	Type      MessageType           `json:"type"`
	ID        string                `json:"id,omitempty"`
	Timestamp int64                 `json:"timestamp"`
	Data      *models.AlertPayload  `json:"data"`
}

// SystemStatusMessage 系统状态消息
type SystemStatusMessage struct {
	Type      MessageType    `json:"type"`
	ID        string         `json:"id,omitempty"`
	Timestamp int64          `json:"timestamp"`
	Data      *SystemStatus  `json:"data"`
}

// WelcomeMessage 欢迎消息
type WelcomeMessage struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id,omitempty"`
	Timestamp int64       `json:"timestamp"`
	Data      *WelcomeData `json:"data"`
}

// AckMessage 确认消息
type AckMessage struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id"`
	Timestamp int64       `json:"timestamp"`
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
}

// ErrorMessage 错误消息
type ErrorMessage struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id,omitempty"`
	Timestamp int64       `json:"timestamp"`
	Error     string      `json:"error"`
	Code      int         `json:"code,omitempty"`
}

// PingMessage Ping消息
type PingMessage struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// PongMessage Pong消息
type PongMessage struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// WelcomeData 欢迎消息数据
type WelcomeData struct {
	ClientID     string            `json:"client_id"`
	ServerTime   int64             `json:"server_time"`
	Version      string            `json:"version"`
	Capabilities []string          `json:"capabilities"`
	Limits       *ConnectionLimits `json:"limits"`
}

// ConnectionLimits 连接限制
type ConnectionLimits struct {
	MaxMessageSize    int64         `json:"max_message_size"`
	MaxSubscriptions  int           `json:"max_subscriptions"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	IdleTimeout       time.Duration `json:"idle_timeout"`
}

// SystemStatus 系统状态
type SystemStatus struct {
	Timestamp         int64                    `json:"timestamp"`
	ConnectedClients  int                      `json:"connected_clients"`
	TotalConnections  int64                    `json:"total_connections"`
	MessagesSent      int64                    `json:"messages_sent"`
	MessagesReceived  int64                    `json:"messages_received"`
	SystemHealth      string                   `json:"system_health"`
	KafkaStatus       *KafkaStatus             `json:"kafka_status,omitempty"`
	ResourceUsage     *ResourceUsage           `json:"resource_usage,omitempty"`
}

// KafkaStatus Kafka状态
type KafkaStatus struct {
	Connected        bool              `json:"connected"`
	ConsumerGroups   []string          `json:"consumer_groups"`
	TopicPartitions  map[string]int    `json:"topic_partitions"`
	MessageRate      float64           `json:"message_rate"`
	LastMessageTime  int64             `json:"last_message_time"`
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsed    int64   `json:"memory_used"`
	MemoryTotal   int64   `json:"memory_total"`
	GoroutineCount int    `json:"goroutine_count"`
	HeapSize      int64   `json:"heap_size"`
}

// MessageRouter 消息路由器接口
type MessageRouter interface {
	// RouteMessage 路由消息到匹配的客户端
	RouteMessage(message *Message) error
	
	// RouteDeviceData 路由设备数据
	RouteDeviceData(data *models.DeviceDataPayload) error
	
	// RouteAlert 路由告警
	RouteAlert(alert *models.AlertPayload) error
	
	// RouteSystemStatus 路由系统状态
	RouteSystemStatus(status *SystemStatus) error
	
	// Subscribe 添加订阅
	Subscribe(clientID string, filter *SubscriptionFilter) error
	
	// Unsubscribe 移除订阅
	Unsubscribe(clientID string, filterID string) error
	
	// GetSubscriptions 获取客户端订阅
	GetSubscriptions(clientID string) []*SubscriptionFilter
	
	// UpdateFilter 更新客户端过滤器
	UpdateFilter(clientID string, filter *ClientFilter) error
}

// DefaultMessageRouter 默认消息路由器实现
type DefaultMessageRouter struct {
	hub *Hub
}

// NewDefaultMessageRouter 创建默认消息路由器
func NewDefaultMessageRouter(hub *Hub) *DefaultMessageRouter {
	return &DefaultMessageRouter{
		hub: hub,
	}
}

// RouteMessage 路由消息
func (r *DefaultMessageRouter) RouteMessage(message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	
	return r.hub.BroadcastToAll(data)
}

// RouteDeviceData 路由设备数据
func (r *DefaultMessageRouter) RouteDeviceData(data *models.DeviceDataPayload) error {
	return r.hub.PushDeviceData(data)
}

// RouteAlert 路由告警
func (r *DefaultMessageRouter) RouteAlert(alert *models.AlertPayload) error {
	return r.hub.PushAlert(alert)
}

// RouteSystemStatus 路由系统状态
func (r *DefaultMessageRouter) RouteSystemStatus(status *SystemStatus) error {
	message := &SystemStatusMessage{
		Type:      MessageTypeSystemStatus,
		Timestamp: time.Now().Unix(),
		Data:      status,
	}
	
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	
	return r.hub.BroadcastToAll(data)
}

// Subscribe 添加订阅
func (r *DefaultMessageRouter) Subscribe(clientID string, filter *SubscriptionFilter) error {
	client := r.hub.GetClient(clientID)
	if client == nil {
		return ErrClientNotFound
	}
	
	client.AddSubscription(filter)
	return nil
}

// Unsubscribe 移除订阅
func (r *DefaultMessageRouter) Unsubscribe(clientID string, filterID string) error {
	client := r.hub.GetClient(clientID)
	if client == nil {
		return ErrClientNotFound
	}
	
	if !client.RemoveSubscription(filterID) {
		return ErrSubscriptionNotFound
	}
	
	return nil
}

// GetSubscriptions 获取客户端订阅
func (r *DefaultMessageRouter) GetSubscriptions(clientID string) []*SubscriptionFilter {
	client := r.hub.GetClient(clientID)
	if client == nil {
		return nil
	}
	
	return client.GetAllSubscriptions()
}

// UpdateFilter 更新客户端过滤器
func (r *DefaultMessageRouter) UpdateFilter(clientID string, filter *ClientFilter) error {
	client := r.hub.GetClient(clientID)
	if client == nil {
		return ErrClientNotFound
	}
	
	client.UpdateFilter(filter)
	return nil
}

// CreateMessage 创建基础消息
func CreateMessage(msgType MessageType, data interface{}) *Message {
	return &Message{
		Type:      msgType,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
}

// CreateDeviceDataMessage 创建设备数据消息
func CreateDeviceDataMessage(data *models.DeviceDataPayload) *DeviceDataMessage {
	return &DeviceDataMessage{
		Type:      MessageTypeDeviceData,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
}

// CreateAlertMessage 创建告警消息
func CreateAlertMessage(alert *models.AlertPayload) *AlertMessage {
	return &AlertMessage{
		Type:      MessageTypeAlert,
		Timestamp: time.Now().Unix(),
		Data:      alert,
	}
}

// CreateWelcomeMessage 创建欢迎消息
func CreateWelcomeMessage(clientID string, version string) *WelcomeMessage {
	return &WelcomeMessage{
		Type:      MessageTypeWelcome,
		Timestamp: time.Now().Unix(),
		Data: &WelcomeData{
			ClientID:   clientID,
			ServerTime: time.Now().Unix(),
			Version:    version,
			Capabilities: []string{
				"device_data",
				"alerts",
				"system_status",
				"subscriptions",
				"filters",
			},
			Limits: &ConnectionLimits{
				MaxMessageSize:    1024 * 1024, // 1MB
				MaxSubscriptions:  100,
				HeartbeatInterval: 15 * time.Second,
				IdleTimeout:       60 * time.Second,
			},
		},
	}
}

// CreateAckMessage 创建确认消息
func CreateAckMessage(id string, success bool, message string) *AckMessage {
	return &AckMessage{
		Type:      MessageTypeAck,
		ID:        id,
		Timestamp: time.Now().Unix(),
		Success:   success,
		Message:   message,
	}
}

// CreateErrorMessage 创建错误消息
func CreateErrorMessage(id string, err error, code int) *ErrorMessage {
	return &ErrorMessage{
		Type:      MessageTypeError,
		ID:        id,
		Timestamp: time.Now().Unix(),
		Error:     err.Error(),
		Code:      code,
	}
}

// CreatePongMessage 创建Pong消息
func CreatePongMessage(id string) *PongMessage {
	return &PongMessage{
		Type:      MessageTypePong,
		ID:        id,
		Timestamp: time.Now().Unix(),
	}
}

// ParseMessage 解析消息
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, ErrInvalidMessageFormat
	}
	
	if msg.Type == "" {
		return nil, ErrMissingMessageType
	}
	
	return &msg, nil
}

// ValidateMessage 验证消息格式
func ValidateMessage(msg *Message) error {
	if msg == nil {
		return ErrInvalidMessageFormat
	}
	
	if msg.Type == "" {
		return ErrMissingMessageType
	}
	
	// 根据消息类型进行特定验证
	switch msg.Type {
	case MessageTypeSubscribe:
		// 验证订阅消息
		if msg.Data == nil {
			return ErrInvalidSubscription
		}
	case MessageTypeUnsubscribe:
		// 验证取消订阅消息
		if msg.ID == "" {
			return ErrInvalidMessageFormat
		}
	case MessageTypeAuth:
		// 验证认证消息
		if msg.Data == nil {
			return ErrInvalidMessageFormat
		}
	}
	
	return nil
}
