package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"simplied-iot-monitoring-go/internal/models"
)

func TestTopicRouter(t *testing.T) {
	t.Run("Create new topic router", func(t *testing.T) {
		defaultTopic := "default-topic"
		router := models.NewTopicRouter(defaultTopic)

		assert.Equal(t, defaultTopic, router.DefaultTopic)
		assert.NotNil(t, router.TopicMapping)

		// 检查默认映射
		assert.Equal(t, "device-data", router.TopicMapping[models.MessageTypeDeviceData])
		assert.Equal(t, "alerts", router.TopicMapping[models.MessageTypeAlert])
		assert.Equal(t, "system-status", router.TopicMapping[models.MessageTypeSystemStatus])
		assert.Equal(t, "heartbeat", router.TopicMapping[models.MessageTypeHeartbeat])
	})

	t.Run("Route message to topic", func(t *testing.T) {
		router := models.NewTopicRouter("default")

		// 测试已映射的消息类型
		deviceMessage := models.NewKafkaMessage(models.MessageTypeDeviceData, "test", nil)
		topic, err := router.Route(deviceMessage)
		assert.NoError(t, err)
		assert.Equal(t, "device-data", topic)

		alertMessage := models.NewKafkaMessage(models.MessageTypeAlert, "test", nil)
		topic, err = router.Route(alertMessage)
		assert.NoError(t, err)
		assert.Equal(t, "alerts", topic)

		// 测试未映射的消息类型
		commandMessage := models.NewKafkaMessage(models.MessageTypeCommand, "test", nil)
		topic, err = router.Route(commandMessage)
		assert.NoError(t, err)
		assert.Equal(t, "default", topic)
	})

	t.Run("Route message without default topic", func(t *testing.T) {
		router := models.NewTopicRouter("")

		commandMessage := models.NewKafkaMessage(models.MessageTypeCommand, "test", nil)
		topic, err := router.Route(commandMessage)
		assert.Error(t, err)
		assert.Empty(t, topic)
		assert.Contains(t, err.Error(), "no topic mapping found")
	})
}

func TestTopicRouterPartitioning(t *testing.T) {
	router := models.NewTopicRouter("default")

	t.Run("Get partition for device data", func(t *testing.T) {
		device := models.NewDevice("device_001", "sensor")
		payload := &models.DeviceDataPayload{Device: device}
		message := models.NewKafkaMessage(models.MessageTypeDeviceData, "test", payload)

		partition, err := router.GetPartition(message)
		assert.NoError(t, err)
		assert.True(t, partition >= 0 && partition < 3) // 应该在0-2之间

		// 相同设备ID应该得到相同分区
		partition2, err := router.GetPartition(message)
		assert.NoError(t, err)
		assert.Equal(t, partition, partition2)
	})

	t.Run("Get partition for non-device message", func(t *testing.T) {
		message := models.NewKafkaMessage(models.MessageTypeAlert, "test", nil)

		partition, err := router.GetPartition(message)
		assert.NoError(t, err)
		assert.Equal(t, int32(0), partition) // 默认分区
	})

	t.Run("Get partition for invalid payload", func(t *testing.T) {
		message := models.NewKafkaMessage(models.MessageTypeDeviceData, "test", "invalid-payload")

		partition, err := router.GetPartition(message)
		assert.NoError(t, err)
		assert.Equal(t, int32(0), partition) // 默认分区
	})
}

func TestTopicRouterKeys(t *testing.T) {
	router := models.NewTopicRouter("default")

	t.Run("Get key for device data", func(t *testing.T) {
		device := models.NewDevice("device_001", "sensor")
		payload := &models.DeviceDataPayload{Device: device}
		message := models.NewKafkaMessage(models.MessageTypeDeviceData, "test", payload)

		key, err := router.GetKey(message)
		assert.NoError(t, err)
		assert.Equal(t, "device_001", key)
	})

	t.Run("Get key for alert", func(t *testing.T) {
		payload := &models.AlertPayload{DeviceID: "device_002"}
		message := models.NewKafkaMessage(models.MessageTypeAlert, "test", payload)

		key, err := router.GetKey(message)
		assert.NoError(t, err)
		assert.Equal(t, "device_002", key)
	})

	t.Run("Get key for other message types", func(t *testing.T) {
		message := models.NewKafkaMessage(models.MessageTypeHeartbeat, "test", nil)

		key, err := router.GetKey(message)
		assert.NoError(t, err)
		assert.Equal(t, message.MessageID, key)
	})
}

func TestMessageBatch(t *testing.T) {
	t.Run("Create new message batch", func(t *testing.T) {
		batchID := "batch_123"
		batchSize := 10

		batch := models.NewMessageBatch(batchID, batchSize)

		assert.Equal(t, batchID, batch.BatchID)
		assert.Equal(t, batchSize, batch.BatchSize)
		assert.Equal(t, models.BatchStatusPending, batch.Status)
		assert.NotNil(t, batch.Messages)
		assert.True(t, batch.IsEmpty())
		assert.False(t, batch.IsFull())
		assert.NotNil(t, batch.Metadata)
	})

	t.Run("Add messages to batch", func(t *testing.T) {
		batch := models.NewMessageBatch("batch_123", 2)

		message1 := models.NewKafkaMessage(models.MessageTypeDeviceData, "test", nil)
		message2 := models.NewKafkaMessage(models.MessageTypeAlert, "test", nil)
		message3 := models.NewKafkaMessage(models.MessageTypeHeartbeat, "test", nil)

		// 添加第一条消息
		err := batch.AddMessage(message1)
		assert.NoError(t, err)
		assert.Len(t, batch.Messages, 1)
		assert.False(t, batch.IsEmpty())
		assert.False(t, batch.IsFull())

		// 添加第二条消息
		err = batch.AddMessage(message2)
		assert.NoError(t, err)
		assert.Len(t, batch.Messages, 2)
		assert.True(t, batch.IsFull())

		// 尝试添加第三条消息（应该失败）
		err = batch.AddMessage(message3)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "batch is full")
		assert.Len(t, batch.Messages, 2)
	})

	t.Run("Batch status management", func(t *testing.T) {
		batch := models.NewMessageBatch("batch_123", 5)

		assert.Equal(t, models.BatchStatusPending, batch.Status)
		assert.Nil(t, batch.ProcessedAt)

		batch.SetProcessing()
		assert.Equal(t, models.BatchStatusProcessing, batch.Status)

		batch.SetCompleted()
		assert.Equal(t, models.BatchStatusCompleted, batch.Status)
		assert.NotNil(t, batch.ProcessedAt)

		// 重置状态测试失败
		batch2 := models.NewMessageBatch("batch_456", 5)
		batch2.SetFailed()
		assert.Equal(t, models.BatchStatusFailed, batch2.Status)
		assert.NotNil(t, batch2.ProcessedAt)
	})
}

func TestDeadLetterQueue(t *testing.T) {
	t.Run("Create new dead letter queue", func(t *testing.T) {
		queueName := "failed-messages"
		maxSize := 100
		retentionTTL := int64(3600) // 1小时

		dlq := models.NewDeadLetterQueue(queueName, maxSize, retentionTTL)

		assert.Equal(t, queueName, dlq.QueueName)
		assert.Equal(t, maxSize, dlq.MaxSize)
		assert.Equal(t, retentionTTL, dlq.RetentionTTL)
		assert.NotNil(t, dlq.Messages)
		assert.Len(t, dlq.Messages, 0)
	})

	t.Run("Add failed messages", func(t *testing.T) {
		dlq := models.NewDeadLetterQueue("test-queue", 3, 3600)

		message1 := models.NewKafkaMessage(models.MessageTypeDeviceData, "test", nil)
		message2 := models.NewKafkaMessage(models.MessageTypeAlert, "test", nil)

		err := dlq.AddFailedMessage(message1, "connection timeout")
		assert.NoError(t, err)
		assert.Len(t, dlq.Messages, 1)

		err = dlq.AddFailedMessage(message2, "invalid payload")
		assert.NoError(t, err)
		assert.Len(t, dlq.Messages, 2)

		// 检查失败消息内容
		failedMsg, exists := dlq.GetFailedMessage(message1.MessageID)
		assert.True(t, exists)
		assert.Equal(t, message1.MessageID, failedMsg.MessageID)
		assert.Equal(t, "connection timeout", failedMsg.FailureReason)
		assert.Equal(t, message1, failedMsg.OriginalMsg)
	})

	t.Run("Queue size limit", func(t *testing.T) {
		dlq := models.NewDeadLetterQueue("test-queue", 2, 3600)

		message1 := models.NewKafkaMessage(models.MessageTypeDeviceData, "test", nil)
		message2 := models.NewKafkaMessage(models.MessageTypeAlert, "test", nil)
		message3 := models.NewKafkaMessage(models.MessageTypeHeartbeat, "test", nil)

		dlq.AddFailedMessage(message1, "error1")
		dlq.AddFailedMessage(message2, "error2")
		assert.Len(t, dlq.Messages, 2)

		// 添加第三条消息应该移除最老的
		dlq.AddFailedMessage(message3, "error3")
		assert.Len(t, dlq.Messages, 2)

		// 第一条消息应该被移除
		_, exists := dlq.GetFailedMessage(message1.MessageID)
		assert.False(t, exists)

		// 第二和第三条消息应该存在
		_, exists = dlq.GetFailedMessage(message2.MessageID)
		assert.True(t, exists)
		_, exists = dlq.GetFailedMessage(message3.MessageID)
		assert.True(t, exists)
	})

	t.Run("Remove expired messages", func(t *testing.T) {
		dlq := models.NewDeadLetterQueue("test-queue", 10, 1) // 1秒TTL

		message1 := models.NewKafkaMessage(models.MessageTypeDeviceData, "test", nil)
		message2 := models.NewKafkaMessage(models.MessageTypeAlert, "test", nil)

		dlq.AddFailedMessage(message1, "error1")

		time.Sleep(2 * time.Second) // 等待过期

		dlq.AddFailedMessage(message2, "error2") // 新消息

		removedCount := dlq.RemoveExpiredMessages()
		assert.Equal(t, 1, removedCount)
		assert.Len(t, dlq.Messages, 1)

		// 只有新消息应该存在
		_, exists := dlq.GetFailedMessage(message1.MessageID)
		assert.False(t, exists)
		_, exists = dlq.GetFailedMessage(message2.MessageID)
		assert.True(t, exists)
	})
}

func TestConnectionManager(t *testing.T) {
	t.Run("Create new connection manager", func(t *testing.T) {
		cm := models.NewConnectionManager()

		assert.NotNil(t, cm.Connections)
		assert.NotNil(t, cm.Sessions)
		assert.NotNil(t, cm.Rooms)
		assert.Len(t, cm.Connections, 0)
		assert.Len(t, cm.Sessions, 0)
		assert.Len(t, cm.Rooms, 0)
	})

	t.Run("Manage connections", func(t *testing.T) {
		cm := models.NewConnectionManager()

		conn := &models.WSConnection{
			ConnectionID:  "conn_123",
			ClientID:      "client_456",
			SessionID:     "session_789",
			RemoteAddr:    "192.168.1.100",
			ConnectedAt:   time.Now(),
			LastPingAt:    time.Now(),
			Status:        models.ConnectionStatusConnected,
			Subscriptions: []string{"device_001", "alerts"},
		}

		// 添加连接
		cm.AddConnection(conn)
		assert.Len(t, cm.Connections, 1)

		// 获取连接
		retrievedConn, exists := cm.GetConnection("conn_123")
		assert.True(t, exists)
		assert.Equal(t, conn.ConnectionID, retrievedConn.ConnectionID)
		assert.Equal(t, conn.ClientID, retrievedConn.ClientID)

		// 根据客户端ID获取连接
		connections := cm.GetConnectionsByClientID("client_456")
		assert.Len(t, connections, 1)
		assert.Equal(t, conn.ConnectionID, connections[0].ConnectionID)

		// 移除连接
		cm.RemoveConnection("conn_123")
		assert.Len(t, cm.Connections, 0)

		_, exists = cm.GetConnection("conn_123")
		assert.False(t, exists)
	})

	t.Run("Manage sessions", func(t *testing.T) {
		cm := models.NewConnectionManager()

		session := cm.CreateSession("client_123", "user_456", 1*time.Hour)

		assert.NotEmpty(t, session.SessionID)
		assert.Equal(t, "client_123", session.ClientID)
		assert.Equal(t, "user_456", session.UserID)
		assert.True(t, session.IsActive)
		assert.True(t, session.ExpiresAt.After(time.Now()))

		// 获取会话
		retrievedSession, exists := cm.GetSession(session.SessionID)
		assert.True(t, exists)
		assert.Equal(t, session.SessionID, retrievedSession.SessionID)
		assert.Equal(t, session.ClientID, retrievedSession.ClientID)
	})

	t.Run("Manage rooms", func(t *testing.T) {
		cm := models.NewConnectionManager()

		room := cm.CreateRoom("Test Room", "A test room", "user_123", 10, false)

		assert.NotEmpty(t, room.RoomID)
		assert.Equal(t, "Test Room", room.Name)
		assert.Equal(t, "A test room", room.Description)
		assert.Equal(t, "user_123", room.CreatedBy)
		assert.Equal(t, 10, room.MaxMembers)
		assert.False(t, room.IsPrivate)
		assert.Len(t, room.Members, 0)

		// 加入房间
		err := cm.JoinRoom(room.RoomID, "client_001")
		assert.NoError(t, err)
		assert.Len(t, room.Members, 1)
		assert.Contains(t, room.Members, "client_001")

		err = cm.JoinRoom(room.RoomID, "client_002")
		assert.NoError(t, err)
		assert.Len(t, room.Members, 2)

		// 重复加入
		err = cm.JoinRoom(room.RoomID, "client_001")
		assert.NoError(t, err)
		assert.Len(t, room.Members, 2) // 没有变化

		// 获取房间成员
		members, err := cm.GetRoomMembers(room.RoomID)
		assert.NoError(t, err)
		assert.Len(t, members, 2)
		assert.Contains(t, members, "client_001")
		assert.Contains(t, members, "client_002")

		// 离开房间
		err = cm.LeaveRoom(room.RoomID, "client_001")
		assert.NoError(t, err)
		assert.Len(t, room.Members, 1)
		assert.NotContains(t, room.Members, "client_001")
		assert.Contains(t, room.Members, "client_002")
	})

	t.Run("Room capacity limit", func(t *testing.T) {
		cm := models.NewConnectionManager()

		room := cm.CreateRoom("Small Room", "", "user_123", 2, false)

		err := cm.JoinRoom(room.RoomID, "client_001")
		assert.NoError(t, err)

		err = cm.JoinRoom(room.RoomID, "client_002")
		assert.NoError(t, err)

		// 第三个客户端应该被拒绝
		err = cm.JoinRoom(room.RoomID, "client_003")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "room is full")
		assert.Len(t, room.Members, 2)
	})

	t.Run("Room and session errors", func(t *testing.T) {
		cm := models.NewConnectionManager()

		// 加入不存在的房间
		err := cm.JoinRoom("non-existent", "client_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "room not found")

		// 离开不存在的房间
		err = cm.LeaveRoom("non-existent", "client_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "room not found")

		// 获取不存在房间的成员
		_, err = cm.GetRoomMembers("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "room not found")

		// 离开未加入的房间
		room := cm.CreateRoom("Test Room", "", "user_123", 10, false)
		err = cm.LeaveRoom(room.RoomID, "client_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client not in room")
	})

	t.Run("Cleanup expired sessions", func(t *testing.T) {
		cm := models.NewConnectionManager()

		// 创建一个很快过期的会话
		session1 := cm.CreateSession("client_001", "user_001", 1*time.Millisecond)
		session2 := cm.CreateSession("client_002", "user_002", 1*time.Hour)

		assert.Len(t, cm.Sessions, 2)

		time.Sleep(10 * time.Millisecond) // 等待第一个会话过期

		expiredCount := cm.CleanupExpiredSessions()
		assert.Equal(t, 1, expiredCount)
		assert.Len(t, cm.Sessions, 1)

		// 第一个会话应该被移除
		_, exists := cm.GetSession(session1.SessionID)
		assert.False(t, exists)

		// 第二个会话应该还存在
		_, exists = cm.GetSession(session2.SessionID)
		assert.True(t, exists)
	})
}

func TestWSMessageRouter(t *testing.T) {
	t.Run("Create new WebSocket message router", func(t *testing.T) {
		router := models.NewWSMessageRouter()

		assert.NotNil(t, router.Routes)
		assert.Len(t, router.Routes, 0)
	})

	t.Run("Register and route handlers", func(t *testing.T) {
		router := models.NewWSMessageRouter()

		// 创建模拟处理器
		mockHandler := &MockWSRouteHandler{}

		// 注册处理器
		router.RegisterHandler(models.WSActionSubscribe, mockHandler)
		assert.Len(t, router.Routes, 1)

		// 路由消息
		message := models.NewWebSocketMessage(models.MessageTypeSubscription, models.WSActionSubscribe, nil)
		response, err := router.Route(message)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, mockHandler.HandleCalled)
	})

	t.Run("Route to non-existent handler", func(t *testing.T) {
		router := models.NewWSMessageRouter()

		message := models.NewWebSocketMessage(models.MessageTypeSubscription, models.WSActionSubscribe, nil)
		response, err := router.Route(message)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "no handler found for action")
	})
}

// MockWSRouteHandler 模拟WebSocket路由处理器
type MockWSRouteHandler struct {
	HandleCalled bool
}

func (m *MockWSRouteHandler) Handle(message *models.WebSocketMessage) (*models.WebSocketMessage, error) {
	m.HandleCalled = true
	response := models.NewWebSocketMessage(models.MessageTypeSubscription, models.WSActionResponse, nil)
	response.RequestID = message.MessageID
	return response, nil
}

func (m *MockWSRouteHandler) GetSubscriptionType() models.SubscriptionType {
	return models.SubscriptionTypeDevice
}

func TestConnectionStatus(t *testing.T) {
	t.Run("Connection status constants", func(t *testing.T) {
		assert.Equal(t, models.ConnectionStatus("connected"), models.ConnectionStatusConnected)
		assert.Equal(t, models.ConnectionStatus("disconnected"), models.ConnectionStatusDisconnected)
		assert.Equal(t, models.ConnectionStatus("idle"), models.ConnectionStatusIdle)
		assert.Equal(t, models.ConnectionStatus("busy"), models.ConnectionStatusBusy)
	})
}

func TestBatchStatus(t *testing.T) {
	t.Run("Batch status constants", func(t *testing.T) {
		assert.Equal(t, models.BatchStatus("pending"), models.BatchStatusPending)
		assert.Equal(t, models.BatchStatus("processing"), models.BatchStatusProcessing)
		assert.Equal(t, models.BatchStatus("completed"), models.BatchStatusCompleted)
		assert.Equal(t, models.BatchStatus("failed"), models.BatchStatusFailed)
	})
}
