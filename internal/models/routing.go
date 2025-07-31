package models

import (
	"fmt"
	"strings"
	"time"
)

// KafkaMessageRouter Kafka消息路由器接口
type KafkaMessageRouter interface {
	Route(message *KafkaMessage) (string, error)
	GetPartition(message *KafkaMessage) (int32, error)
	GetKey(message *KafkaMessage) (string, error)
}

// TopicRouter Topic路由器
type TopicRouter struct {
	DefaultTopic string
	TopicMapping map[MessageType]string
}

// WSMessageRouter WebSocket消息路由器
type WSMessageRouter struct {
	Routes map[WSAction]WSRouteHandler
}

// WSRouteHandler WebSocket路由处理器
type WSRouteHandler interface {
	Handle(message *WebSocketMessage) (*WebSocketMessage, error)
	GetSubscriptionType() SubscriptionType
}

// MessageQueue 消息队列接口
type MessageQueue interface {
	Enqueue(message interface{}) error
	Dequeue() (interface{}, error)
	Size() int
	IsEmpty() bool
}

// MessageBatch 消息批次
type MessageBatch struct {
	BatchID     string                 `json:"batch_id"`
	Messages    []*KafkaMessage        `json:"messages"`
	BatchSize   int                    `json:"batch_size"`
	CreatedAt   time.Time              `json:"created_at"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
	Status      BatchStatus            `json:"status"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BatchStatus 批次状态
type BatchStatus string

const (
	BatchStatusPending    BatchStatus = "pending"
	BatchStatusProcessing BatchStatus = "processing"
	BatchStatusCompleted  BatchStatus = "completed"
	BatchStatusFailed     BatchStatus = "failed"
)

// DeadLetterQueue 死信队列
type DeadLetterQueue struct {
	QueueName    string           `json:"queue_name"`
	Messages     []*FailedMessage `json:"messages"`
	MaxSize      int              `json:"max_size"`
	RetentionTTL int64            `json:"retention_ttl"` // 保留时间(秒)
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// FailedMessage 失败消息
type FailedMessage struct {
	MessageID     string                 `json:"message_id"`
	OriginalMsg   *KafkaMessage          `json:"original_message"`
	FailureReason string                 `json:"failure_reason"`
	FailedAt      time.Time              `json:"failed_at"`
	RetryCount    int                    `json:"retry_count"`
	LastRetryAt   *time.Time             `json:"last_retry_at,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// MessageFilter 消息过滤器
type MessageFilter struct {
	FilterID   string            `json:"filter_id"`
	Name       string            `json:"name"`
	Conditions []FilterCondition `json:"conditions"`
	Action     FilterAction      `json:"action"`
	Enabled    bool              `json:"enabled"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// FilterCondition 过滤条件
type FilterCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, contains, regex
	Value    interface{} `json:"value"`
}

// FilterAction 过滤动作
type FilterAction string

const (
	FilterActionAllow  FilterAction = "allow"
	FilterActionDeny   FilterAction = "deny"
	FilterActionModify FilterAction = "modify"
	FilterActionRoute  FilterAction = "route"
)

// ConnectionManager 连接管理器
type ConnectionManager struct {
	Connections map[string]*WSConnection `json:"connections"`
	Sessions    map[string]*WSSession    `json:"sessions"`
	Rooms       map[string]*WSRoom       `json:"rooms"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

// WSConnection WebSocket连接
type WSConnection struct {
	ConnectionID  string                 `json:"connection_id"`
	ClientID      string                 `json:"client_id"`
	SessionID     string                 `json:"session_id"`
	UserID        string                 `json:"user_id,omitempty"`
	RemoteAddr    string                 `json:"remote_addr"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	ConnectedAt   time.Time              `json:"connected_at"`
	LastPingAt    time.Time              `json:"last_ping_at"`
	Status        ConnectionStatus       `json:"status"`
	Subscriptions []string               `json:"subscriptions"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ConnectionStatus 连接状态
type ConnectionStatus string

const (
	ConnectionStatusConnected    ConnectionStatus = "connected"
	ConnectionStatusDisconnected ConnectionStatus = "disconnected"
	ConnectionStatusIdle         ConnectionStatus = "idle"
	ConnectionStatusBusy         ConnectionStatus = "busy"
)

// WSSession WebSocket会话
type WSSession struct {
	SessionID     string                 `json:"session_id"`
	ClientID      string                 `json:"client_id"`
	UserID        string                 `json:"user_id,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	ExpiresAt     time.Time              `json:"expires_at"`
	IsActive      bool                   `json:"is_active"`
	Subscriptions []SubscriptionInfo     `json:"subscriptions"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SubscriptionInfo 订阅信息
type SubscriptionInfo struct {
	SubscriptionID string           `json:"subscription_id"`
	Type           SubscriptionType `json:"type"`
	Target         string           `json:"target"`
	Filters        []string         `json:"filters,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	IsActive       bool             `json:"is_active"`
}

// WSRoom WebSocket房间
type WSRoom struct {
	RoomID      string    `json:"room_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Members     []string  `json:"members"` // ClientID列表
	MaxMembers  int       `json:"max_members"`
	IsPrivate   bool      `json:"is_private"`
	CreatedBy   string    `json:"created_by,omitempty"`
}

// NewTopicRouter 创建Topic路由器
func NewTopicRouter(defaultTopic string) *TopicRouter {
	return &TopicRouter{
		DefaultTopic: defaultTopic,
		TopicMapping: map[MessageType]string{
			MessageTypeDeviceData:   "device-data",
			MessageTypeAlert:        "alerts",
			MessageTypeSystemStatus: "system-status",
			MessageTypeHeartbeat:    "heartbeat",
		},
	}
}

// Route 路由消息到Topic
func (tr *TopicRouter) Route(message *KafkaMessage) (string, error) {
	if topic, exists := tr.TopicMapping[message.MessageType]; exists {
		return topic, nil
	}
	if tr.DefaultTopic != "" {
		return tr.DefaultTopic, nil
	}
	return "", fmt.Errorf("no topic mapping found for message type: %s", message.MessageType)
}

// GetPartition 获取分区
func (tr *TopicRouter) GetPartition(message *KafkaMessage) (int32, error) {
	// 基于设备ID进行分区
	if message.MessageType == MessageTypeDeviceData {
		if payload, ok := message.Payload.(*DeviceDataPayload); ok {
			if payload.Device != nil {
				// 简单哈希分区
				hash := int32(0)
				for _, b := range payload.Device.DeviceID {
					hash = hash*31 + int32(b)
				}
				if hash < 0 {
					hash = -hash
				}
				return hash % 3, nil // 假设3个分区
			}
		}
	}
	return 0, nil // 默认分区
}

// GetKey 获取消息键
func (tr *TopicRouter) GetKey(message *KafkaMessage) (string, error) {
	switch message.MessageType {
	case MessageTypeDeviceData:
		if payload, ok := message.Payload.(*DeviceDataPayload); ok {
			if payload.Device != nil {
				return payload.Device.DeviceID, nil
			}
		}
	case MessageTypeAlert:
		if payload, ok := message.Payload.(*AlertPayload); ok {
			return payload.DeviceID, nil
		}
	}
	return message.MessageID, nil
}

// NewWSMessageRouter 创建WebSocket消息路由器
func NewWSMessageRouter() *WSMessageRouter {
	return &WSMessageRouter{
		Routes: make(map[WSAction]WSRouteHandler),
	}
}

// RegisterHandler 注册路由处理器
func (wr *WSMessageRouter) RegisterHandler(action WSAction, handler WSRouteHandler) {
	wr.Routes[action] = handler
}

// Route 路由WebSocket消息
func (wr *WSMessageRouter) Route(message *WebSocketMessage) (*WebSocketMessage, error) {
	handler, exists := wr.Routes[message.Action]
	if !exists {
		return nil, fmt.Errorf("no handler found for action: %s", message.Action)
	}
	return handler.Handle(message)
}

// MessageBatch方法

// NewMessageBatch 创建新的消息批次
func NewMessageBatch(batchID string, batchSize int) *MessageBatch {
	return &MessageBatch{
		BatchID:   batchID,
		Messages:  make([]*KafkaMessage, 0, batchSize),
		BatchSize: batchSize,
		CreatedAt: time.Now(),
		Status:    BatchStatusPending,
		Metadata:  make(map[string]interface{}),
	}
}

// AddMessage 添加消息到批次
func (mb *MessageBatch) AddMessage(message *KafkaMessage) error {
	if len(mb.Messages) >= mb.BatchSize {
		return fmt.Errorf("batch is full, cannot add more messages")
	}
	mb.Messages = append(mb.Messages, message)
	return nil
}

// IsFull 检查批次是否已满
func (mb *MessageBatch) IsFull() bool {
	return len(mb.Messages) >= mb.BatchSize
}

// IsEmpty 检查批次是否为空
func (mb *MessageBatch) IsEmpty() bool {
	return len(mb.Messages) == 0
}

// SetProcessing 设置为处理中状态
func (mb *MessageBatch) SetProcessing() {
	mb.Status = BatchStatusProcessing
}

// SetCompleted 设置为完成状态
func (mb *MessageBatch) SetCompleted() {
	mb.Status = BatchStatusCompleted
	now := time.Now()
	mb.ProcessedAt = &now
}

// SetFailed 设置为失败状态
func (mb *MessageBatch) SetFailed() {
	mb.Status = BatchStatusFailed
	now := time.Now()
	mb.ProcessedAt = &now
}

// DeadLetterQueue方法

// NewDeadLetterQueue 创建死信队列
func NewDeadLetterQueue(queueName string, maxSize int, retentionTTL int64) *DeadLetterQueue {
	return &DeadLetterQueue{
		QueueName:    queueName,
		Messages:     make([]*FailedMessage, 0),
		MaxSize:      maxSize,
		RetentionTTL: retentionTTL,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// AddFailedMessage 添加失败消息
func (dlq *DeadLetterQueue) AddFailedMessage(message *KafkaMessage, reason string) error {
	if len(dlq.Messages) >= dlq.MaxSize {
		// 移除最老的消息
		dlq.Messages = dlq.Messages[1:]
	}

	failedMsg := &FailedMessage{
		MessageID:     message.MessageID,
		OriginalMsg:   message,
		FailureReason: reason,
		FailedAt:      time.Now(),
		RetryCount:    message.RetryCount,
		Metadata:      make(map[string]interface{}),
	}

	dlq.Messages = append(dlq.Messages, failedMsg)
	dlq.UpdatedAt = time.Now()
	return nil
}

// GetFailedMessage 获取失败消息
func (dlq *DeadLetterQueue) GetFailedMessage(messageID string) (*FailedMessage, bool) {
	for _, msg := range dlq.Messages {
		if msg.MessageID == messageID {
			return msg, true
		}
	}
	return nil, false
}

// RemoveExpiredMessages 移除过期消息
func (dlq *DeadLetterQueue) RemoveExpiredMessages() int {
	if dlq.RetentionTTL <= 0 {
		return 0
	}

	cutoff := time.Now().Unix() - dlq.RetentionTTL
	originalCount := len(dlq.Messages)

	validMessages := make([]*FailedMessage, 0)
	for _, msg := range dlq.Messages {
		if msg.FailedAt.Unix() > cutoff {
			validMessages = append(validMessages, msg)
		}
	}

	dlq.Messages = validMessages
	dlq.UpdatedAt = time.Now()

	return originalCount - len(validMessages)
}

// ConnectionManager方法

// NewConnectionManager 创建连接管理器
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		Connections: make(map[string]*WSConnection),
		Sessions:    make(map[string]*WSSession),
		Rooms:       make(map[string]*WSRoom),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// AddConnection 添加连接
func (cm *ConnectionManager) AddConnection(conn *WSConnection) {
	cm.Connections[conn.ConnectionID] = conn
	cm.UpdatedAt = time.Now()
}

// RemoveConnection 移除连接
func (cm *ConnectionManager) RemoveConnection(connectionID string) {
	delete(cm.Connections, connectionID)
	cm.UpdatedAt = time.Now()
}

// GetConnection 获取连接
func (cm *ConnectionManager) GetConnection(connectionID string) (*WSConnection, bool) {
	conn, exists := cm.Connections[connectionID]
	return conn, exists
}

// GetConnectionsByClientID 根据客户端ID获取连接
func (cm *ConnectionManager) GetConnectionsByClientID(clientID string) []*WSConnection {
	connections := make([]*WSConnection, 0)
	for _, conn := range cm.Connections {
		if conn.ClientID == clientID {
			connections = append(connections, conn)
		}
	}
	return connections
}

// CreateSession 创建会话
func (cm *ConnectionManager) CreateSession(clientID, userID string, duration time.Duration) *WSSession {
	session := &WSSession{
		SessionID:     fmt.Sprintf("session_%s_%d", clientID, time.Now().Unix()),
		ClientID:      clientID,
		UserID:        userID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(duration),
		IsActive:      true,
		Subscriptions: make([]SubscriptionInfo, 0),
		Metadata:      make(map[string]interface{}),
	}

	cm.Sessions[session.SessionID] = session
	cm.UpdatedAt = time.Now()
	return session
}

// GetSession 获取会话
func (cm *ConnectionManager) GetSession(sessionID string) (*WSSession, bool) {
	session, exists := cm.Sessions[sessionID]
	return session, exists
}

// CreateRoom 创建房间
func (cm *ConnectionManager) CreateRoom(name, description, createdBy string, maxMembers int, isPrivate bool) *WSRoom {
	room := &WSRoom{
		RoomID:      fmt.Sprintf("room_%s_%d", strings.ReplaceAll(name, " ", "_"), time.Now().Unix()),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Members:     make([]string, 0),
		MaxMembers:  maxMembers,
		IsPrivate:   isPrivate,
		CreatedBy:   createdBy,
	}

	cm.Rooms[room.RoomID] = room
	cm.UpdatedAt = time.Now()
	return room
}

// JoinRoom 加入房间
func (cm *ConnectionManager) JoinRoom(roomID, clientID string) error {
	room, exists := cm.Rooms[roomID]
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	// 检查是否已在房间中
	for _, member := range room.Members {
		if member == clientID {
			return nil // 已在房间中
		}
	}

	// 检查房间容量
	if len(room.Members) >= room.MaxMembers {
		return fmt.Errorf("room is full")
	}

	room.Members = append(room.Members, clientID)
	room.UpdatedAt = time.Now()
	cm.UpdatedAt = time.Now()

	return nil
}

// LeaveRoom 离开房间
func (cm *ConnectionManager) LeaveRoom(roomID, clientID string) error {
	room, exists := cm.Rooms[roomID]
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	for i, member := range room.Members {
		if member == clientID {
			room.Members = append(room.Members[:i], room.Members[i+1:]...)
			room.UpdatedAt = time.Now()
			cm.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("client not in room")
}

// GetRoomMembers 获取房间成员
func (cm *ConnectionManager) GetRoomMembers(roomID string) ([]string, error) {
	room, exists := cm.Rooms[roomID]
	if !exists {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}
	return room.Members, nil
}

// CleanupExpiredSessions 清理过期会话
func (cm *ConnectionManager) CleanupExpiredSessions() int {
	now := time.Now()
	expiredCount := 0

	for sessionID, session := range cm.Sessions {
		if now.After(session.ExpiresAt) {
			delete(cm.Sessions, sessionID)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		cm.UpdatedAt = time.Now()
	}

	return expiredCount
}
