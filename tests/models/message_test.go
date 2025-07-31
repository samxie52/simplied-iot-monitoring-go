package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simplied-iot-monitoring-go/internal/models"
)

func TestKafkaMessage(t *testing.T) {
	t.Run("Create new Kafka message", func(t *testing.T) {
		device := models.NewDevice("device_001", "sensor")
		payload := &models.DeviceDataPayload{
			Device: device,
		}

		message := models.NewKafkaMessage(models.MessageTypeDeviceData, "test-producer", payload)

		assert.NotEmpty(t, message.MessageID)
		assert.Equal(t, models.MessageTypeDeviceData, message.MessageType)
		assert.Equal(t, "test-producer", message.Source)
		assert.Equal(t, payload, message.Payload)
		assert.Equal(t, models.PriorityNormal, message.Priority)
		assert.Equal(t, 0, message.RetryCount)
		assert.NotNil(t, message.Headers)
	})

	t.Run("Set and get headers", func(t *testing.T) {
		message := models.NewKafkaMessage(models.MessageTypeHeartbeat, "test", nil)

		message.SetHeader("Content-Type", "application/json")
		message.SetHeader("User-ID", "user123")

		contentType, exists := message.GetHeader("Content-Type")
		assert.True(t, exists)
		assert.Equal(t, "application/json", contentType)

		userID, exists := message.GetHeader("User-ID")
		assert.True(t, exists)
		assert.Equal(t, "user123", userID)

		_, exists = message.GetHeader("NonExistent")
		assert.False(t, exists)
	})

	t.Run("Set priority and TTL", func(t *testing.T) {
		message := models.NewKafkaMessage(models.MessageTypeAlert, "test", nil)

		message.SetPriority(models.PriorityCritical)
		assert.Equal(t, models.PriorityCritical, message.Priority)

		message.SetTTL(5 * time.Minute)
		assert.Equal(t, int64(300), message.TTL)
	})

	t.Run("Check message expiration", func(t *testing.T) {
		message := models.NewKafkaMessage(models.MessageTypeAlert, "test", nil)

		// 未设置TTL，不会过期
		assert.False(t, message.IsExpired())

		// 设置很短的TTL
		message.SetTTL(1 * time.Second)
		time.Sleep(1100 * time.Millisecond) // 等待1.1秒确保过期
		assert.True(t, message.IsExpired())
	})

	t.Run("Increment retry count", func(t *testing.T) {
		message := models.NewKafkaMessage(models.MessageTypeAlert, "test", nil)

		assert.Equal(t, 0, message.RetryCount)

		message.IncrementRetry()
		assert.Equal(t, 1, message.RetryCount)

		message.IncrementRetry()
		assert.Equal(t, 2, message.RetryCount)
	})

	t.Run("Set processed", func(t *testing.T) {
		message := models.NewKafkaMessage(models.MessageTypeAlert, "test", nil)

		assert.Nil(t, message.ProcessedAt)

		message.SetProcessed()
		assert.NotNil(t, message.ProcessedAt)
		assert.True(t, *message.ProcessedAt > 0)
	})
}

func TestKafkaMessageValidation(t *testing.T) {
	tests := []struct {
		name          string
		message       *models.KafkaMessage
		expectedError string
	}{
		{
			name: "Valid message",
			message: &models.KafkaMessage{
				MessageID:   "msg-123",
				MessageType: models.MessageTypeDeviceData,
				Timestamp:   time.Now().Unix(),
				Source:      "test-source",
				Payload:     "test-payload",
				Priority:    models.PriorityNormal,
			},
			expectedError: "",
		},
		{
			name: "Missing message ID",
			message: &models.KafkaMessage{
				MessageType: models.MessageTypeDeviceData,
				Timestamp:   time.Now().Unix(),
				Source:      "test-source",
				Payload:     "test-payload",
				Priority:    models.PriorityNormal,
			},
			expectedError: "message_id is required",
		},
		{
			name: "Missing message type",
			message: &models.KafkaMessage{
				MessageID: "msg-123",
				Timestamp: time.Now().Unix(),
				Source:    "test-source",
				Payload:   "test-payload",
				Priority:  models.PriorityNormal,
			},
			expectedError: "message_type is required",
		},
		{
			name: "Invalid timestamp",
			message: &models.KafkaMessage{
				MessageID:   "msg-123",
				MessageType: models.MessageTypeDeviceData,
				Timestamp:   0,
				Source:      "test-source",
				Payload:     "test-payload",
				Priority:    models.PriorityNormal,
			},
			expectedError: "timestamp must be positive",
		},
		{
			name: "Missing source",
			message: &models.KafkaMessage{
				MessageID:   "msg-123",
				MessageType: models.MessageTypeDeviceData,
				Timestamp:   time.Now().Unix(),
				Payload:     "test-payload",
				Priority:    models.PriorityNormal,
			},
			expectedError: "source is required",
		},
		{
			name: "Missing payload",
			message: &models.KafkaMessage{
				MessageID:   "msg-123",
				MessageType: models.MessageTypeDeviceData,
				Timestamp:   time.Now().Unix(),
				Source:      "test-source",
				Priority:    models.PriorityNormal,
			},
			expectedError: "payload is required",
		},
		{
			name: "Invalid priority low",
			message: &models.KafkaMessage{
				MessageID:   "msg-123",
				MessageType: models.MessageTypeDeviceData,
				Timestamp:   time.Now().Unix(),
				Source:      "test-source",
				Payload:     "test-payload",
				Priority:    0,
			},
			expectedError: "priority must be between 1 and 10",
		},
		{
			name: "Invalid priority high",
			message: &models.KafkaMessage{
				MessageID:   "msg-123",
				MessageType: models.MessageTypeDeviceData,
				Timestamp:   time.Now().Unix(),
				Source:      "test-source",
				Payload:     "test-payload",
				Priority:    11,
			},
			expectedError: "priority must be between 1 and 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.message.Validate()
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestKafkaMessageSerialization(t *testing.T) {
	t.Run("JSON serialization", func(t *testing.T) {
		device := models.NewDevice("device_001", "sensor")
		payload := &models.DeviceDataPayload{
			Device: device,
		}

		message := models.NewKafkaMessage(models.MessageTypeDeviceData, "test-producer", payload)
		message.SetHeader("Content-Type", "application/json")
		message.SetPriority(models.PriorityHigh)

		// 序列化
		data, err := message.ToJSON()
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// 反序列化
		var deserializedMessage models.KafkaMessage
		err = deserializedMessage.FromJSON(data)
		require.NoError(t, err)

		assert.Equal(t, message.MessageID, deserializedMessage.MessageID)
		assert.Equal(t, message.MessageType, deserializedMessage.MessageType)
		assert.Equal(t, message.Source, deserializedMessage.Source)
		assert.Equal(t, message.Priority, deserializedMessage.Priority)

		// 检查头部信息
		contentType, exists := deserializedMessage.GetHeader("Content-Type")
		assert.True(t, exists)
		assert.Equal(t, "application/json", contentType)
	})
}

func TestWebSocketMessage(t *testing.T) {
	t.Run("Create new WebSocket message", func(t *testing.T) {
		payload := &models.SubscriptionPayload{
			Type:   models.SubscriptionTypeDevice,
			Target: "device_001",
		}

		message := models.NewWebSocketMessage(models.MessageTypeSubscription, models.WSActionSubscribe, payload)

		assert.NotEmpty(t, message.MessageID)
		assert.Equal(t, models.MessageTypeSubscription, message.MessageType)
		assert.Equal(t, models.WSActionSubscribe, message.Action)
		assert.Equal(t, payload, message.Payload)
		assert.Equal(t, models.PriorityNormal, message.Priority)
		assert.True(t, message.Success)
		assert.Empty(t, message.Error)
	})

	t.Run("Set client and session ID", func(t *testing.T) {
		message := models.NewWebSocketMessage(models.MessageTypeHeartbeat, models.WSActionHeartbeat, nil)

		message.SetClientID("client_123")
		message.SetSessionID("session_456")

		assert.Equal(t, "client_123", message.ClientID)
		assert.Equal(t, "session_456", message.SessionID)
	})

	t.Run("Set error", func(t *testing.T) {
		message := models.NewWebSocketMessage(models.MessageTypeSubscription, models.WSActionSubscribe, nil)

		assert.True(t, message.Success)
		assert.Empty(t, message.Error)

		err := assert.AnError
		message.SetError(err)

		assert.False(t, message.Success)
		assert.Equal(t, err.Error(), message.Error)
	})

	t.Run("Set success", func(t *testing.T) {
		message := models.NewWebSocketMessage(models.MessageTypeSubscription, models.WSActionSubscribe, nil)

		// 先设置错误
		message.SetError(assert.AnError)
		assert.False(t, message.Success)
		assert.NotEmpty(t, message.Error)

		// 再设置成功
		message.SetSuccess(true)
		assert.True(t, message.Success)
		assert.Empty(t, message.Error)
	})
}

func TestWebSocketMessageValidation(t *testing.T) {
	tests := []struct {
		name          string
		message       *models.WebSocketMessage
		expectedError string
	}{
		{
			name: "Valid message",
			message: &models.WebSocketMessage{
				MessageID:   "msg-123",
				MessageType: models.MessageTypeSubscription,
				Action:      models.WSActionSubscribe,
				Timestamp:   time.Now().Unix(),
				Priority:    models.PriorityNormal,
			},
			expectedError: "",
		},
		{
			name: "Missing message ID",
			message: &models.WebSocketMessage{
				MessageType: models.MessageTypeSubscription,
				Action:      models.WSActionSubscribe,
				Timestamp:   time.Now().Unix(),
				Priority:    models.PriorityNormal,
			},
			expectedError: "message_id is required",
		},
		{
			name: "Missing message type",
			message: &models.WebSocketMessage{
				MessageID: "msg-123",
				Action:    models.WSActionSubscribe,
				Timestamp: time.Now().Unix(),
				Priority:  models.PriorityNormal,
			},
			expectedError: "message_type is required",
		},
		{
			name: "Missing action",
			message: &models.WebSocketMessage{
				MessageID:   "msg-123",
				MessageType: models.MessageTypeSubscription,
				Timestamp:   time.Now().Unix(),
				Priority:    models.PriorityNormal,
			},
			expectedError: "action is required",
		},
		{
			name: "Invalid timestamp",
			message: &models.WebSocketMessage{
				MessageID:   "msg-123",
				MessageType: models.MessageTypeSubscription,
				Action:      models.WSActionSubscribe,
				Timestamp:   0,
				Priority:    models.PriorityNormal,
			},
			expectedError: "timestamp must be positive",
		},
		{
			name: "Invalid priority",
			message: &models.WebSocketMessage{
				MessageID:   "msg-123",
				MessageType: models.MessageTypeSubscription,
				Action:      models.WSActionSubscribe,
				Timestamp:   time.Now().Unix(),
				Priority:    0,
			},
			expectedError: "priority must be between 1 and 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.message.Validate()
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestMessagePayloads(t *testing.T) {
	t.Run("DeviceDataPayload", func(t *testing.T) {
		device := models.NewDevice("device_001", "sensor")
		// 设置必需的设备信息以通过验证
		device.DeviceInfo.Model = "Test-Model"
		device.DeviceInfo.Manufacturer = "Test-Manufacturer"
		device.DeviceInfo.FirmwareVersion = "1.0.0"
		device.DeviceInfo.HardwareVersion = "1.0.0"
		device.DeviceInfo.SerialNumber = "SN123456"
		device.DeviceInfo.BatteryLevel = 85
		device.DeviceInfo.SignalStrength = -45
		device.DeviceInfo.NetworkType = models.NetworkTypeWiFi
		device.Location.Building = "Building A"
		device.Location.Floor = 1
		device.Location.Room = "Room 101"
		device.Location.Zone = "Zone A"
		// 设置传感器数据的LastUpdate字段
		device.SensorData.LastUpdate = time.Now().Unix()

		payload := &models.DeviceDataPayload{
			Device:    device,
			BatchSize: 10,
			BatchID:   "batch_123",
		}

		assert.Equal(t, string(models.MessageTypeDeviceData), payload.GetType())

		err := payload.Validate()
		assert.NoError(t, err)

		data, err := payload.ToJSON()
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		// 验证JSON包含预期字段
		var jsonData map[string]interface{}
		err = json.Unmarshal(data, &jsonData)
		require.NoError(t, err)
		assert.Contains(t, jsonData, "device")
		assert.Contains(t, jsonData, "batch_size")
		assert.Contains(t, jsonData, "batch_id")
	})

	t.Run("AlertPayload", func(t *testing.T) {
		alert := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "High Temperature", "Temperature exceeds threshold")
		payload := &models.AlertPayload{
			Alert:    alert,
			DeviceID: "device_001",
			Resolved: false,
		}

		assert.Equal(t, string(models.MessageTypeAlert), payload.GetType())

		err := payload.Validate()
		assert.NoError(t, err)

		data, err := payload.ToJSON()
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("SystemStatusPayload", func(t *testing.T) {
		payload := &models.SystemStatusPayload{
			ServiceName: "kafka-consumer",
			Status:      "healthy",
			Metrics: map[string]interface{}{
				"cpu_usage":    45.2,
				"memory_usage": 67.8,
				"uptime":       3600,
			},
			Timestamp: time.Now().Unix(),
		}

		assert.Equal(t, string(models.MessageTypeSystemStatus), payload.GetType())

		err := payload.Validate()
		assert.NoError(t, err)

		data, err := payload.ToJSON()
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})
}

func TestMessagePayloadValidation(t *testing.T) {
	t.Run("DeviceDataPayload validation", func(t *testing.T) {
		// 无效载荷 - 缺少设备
		payload := &models.DeviceDataPayload{}
		err := payload.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "device is required")
	})

	t.Run("AlertPayload validation", func(t *testing.T) {
		// 无效载荷 - 缺少告警
		payload := &models.AlertPayload{
			DeviceID: "device_001",
		}
		err := payload.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "alert is required")

		// 无效载荷 - 缺少设备ID
		alert := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Test", "Test message")
		payload = &models.AlertPayload{
			Alert: alert,
		}
		err = payload.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "device_id is required")
	})

	t.Run("SystemStatusPayload validation", func(t *testing.T) {
		// 无效载荷 - 缺少服务名
		payload := &models.SystemStatusPayload{
			Status:    "healthy",
			Timestamp: time.Now().Unix(),
		}
		err := payload.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service_name is required")

		// 无效载荷 - 缺少状态
		payload = &models.SystemStatusPayload{
			ServiceName: "test-service",
			Timestamp:   time.Now().Unix(),
		}
		err = payload.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status is required")

		// 无效载荷 - 无效时间戳
		payload = &models.SystemStatusPayload{
			ServiceName: "test-service",
			Status:      "healthy",
			Timestamp:   0,
		}
		err = payload.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timestamp must be positive")
	})
}

func TestWebSocketMessageSerialization(t *testing.T) {
	t.Run("JSON serialization", func(t *testing.T) {
		payload := &models.HeartbeatPayload{
			ClientID:  "client_123",
			Timestamp: time.Now().Unix(),
			Status:    "alive",
		}

		message := models.NewWebSocketMessage(models.MessageTypeHeartbeat, models.WSActionHeartbeat, payload)
		message.SetClientID("client_123")
		message.SetSessionID("session_456")

		// 序列化
		data, err := message.ToJSON()
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// 反序列化
		var deserializedMessage models.WebSocketMessage
		err = deserializedMessage.FromJSON(data)
		require.NoError(t, err)

		assert.Equal(t, message.MessageID, deserializedMessage.MessageID)
		assert.Equal(t, message.MessageType, deserializedMessage.MessageType)
		assert.Equal(t, message.Action, deserializedMessage.Action)
		assert.Equal(t, message.ClientID, deserializedMessage.ClientID)
		assert.Equal(t, message.SessionID, deserializedMessage.SessionID)
		assert.Equal(t, message.Success, deserializedMessage.Success)
	})
}

func TestMessageTypes(t *testing.T) {
	t.Run("Message type constants", func(t *testing.T) {
		assert.Equal(t, models.MessageType("device_data"), models.MessageTypeDeviceData)
		assert.Equal(t, models.MessageType("alert"), models.MessageTypeAlert)
		assert.Equal(t, models.MessageType("system_status"), models.MessageTypeSystemStatus)
		assert.Equal(t, models.MessageType("heartbeat"), models.MessageTypeHeartbeat)
		assert.Equal(t, models.MessageType("subscription"), models.MessageTypeSubscription)
		assert.Equal(t, models.MessageType("command"), models.MessageTypeCommand)
	})

	t.Run("WebSocket action constants", func(t *testing.T) {
		assert.Equal(t, models.WSAction("subscribe"), models.WSActionSubscribe)
		assert.Equal(t, models.WSAction("unsubscribe"), models.WSActionUnsubscribe)
		assert.Equal(t, models.WSAction("publish"), models.WSActionPublish)
		assert.Equal(t, models.WSAction("heartbeat"), models.WSActionHeartbeat)
		assert.Equal(t, models.WSAction("auth"), models.WSActionAuth)
		assert.Equal(t, models.WSAction("response"), models.WSActionResponse)
		assert.Equal(t, models.WSAction("broadcast"), models.WSActionBroadcast)
	})

	t.Run("Priority constants", func(t *testing.T) {
		assert.Equal(t, models.MessagePriority(1), models.PriorityLow)
		assert.Equal(t, models.MessagePriority(5), models.PriorityNormal)
		assert.Equal(t, models.MessagePriority(8), models.PriorityHigh)
		assert.Equal(t, models.MessagePriority(10), models.PriorityCritical)
	})

	t.Run("Subscription type constants", func(t *testing.T) {
		assert.Equal(t, models.SubscriptionType("device"), models.SubscriptionTypeDevice)
		assert.Equal(t, models.SubscriptionType("room"), models.SubscriptionTypeRoom)
		assert.Equal(t, models.SubscriptionType("alert"), models.SubscriptionTypeAlert)
		assert.Equal(t, models.SubscriptionType("all"), models.SubscriptionTypeAll)
	})
}
