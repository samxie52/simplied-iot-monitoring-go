package consumer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simplied-iot-monitoring-go/internal/models"
	"simplied-iot-monitoring-go/internal/services/consumer"
	wsService "simplied-iot-monitoring-go/internal/services/websocket"
)

// 测试用的消息处理器
type TestDeviceDataHandler struct{}

func (h *TestDeviceDataHandler) HandleMessage(ctx context.Context, message *models.KafkaMessage) error {
	// 模拟设备数据处理
	return nil
}

func (h *TestDeviceDataHandler) GetHandlerType() models.MessageType {
	return models.MessageTypeDeviceData
}

type TestAlertHandler struct{}

func (h *TestAlertHandler) HandleMessage(ctx context.Context, message *models.KafkaMessage) error {
	// 模拟告警处理
	return nil
}

func (h *TestAlertHandler) GetHandlerType() models.MessageType {
	return models.MessageTypeAlert
}

func TestWebSocketIntegratedProcessor(t *testing.T) {
	// 创建数据推送器
	pusherConfig := &wsService.DataPusherConfig{
		QueueSize:         100,
		WorkerCount:       2,
		WriteTimeout:      5 * time.Second,
		PingInterval:      30 * time.Second,
		ClientTimeout:     60 * time.Second,
		MaxMessageSize:    1024 * 1024,
		EnableCompression: true,
	}
	dataPusher := wsService.NewWebSocketDataPusher(pusherConfig)

	// 启动数据推送器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := dataPusher.Start(ctx)
	require.NoError(t, err)
	defer dataPusher.Stop()

	// 创建WebSocket集成处理器
	processorConfig := consumer.ProcessorConfig{
		MaxRetries:           3,
		RetryDelay:           1 * time.Second,
		ProcessingTimeout:    10 * time.Second,
		EnableValidation:     true,
		EnableTransformation: true,
		EnableCaching:        false,
		BatchSize:            100,
		FlushInterval:        5 * time.Second,
	}

	wsConfig := &consumer.WebSocketProcessorConfig{
		EnablePush:     true,
		PushTimeout:    5 * time.Second,
		AsyncPush:      false, // 同步推送便于测试
		PushRetries:    3,
		PushRetryDelay: 1 * time.Second,
	}

	processor := consumer.NewWebSocketIntegratedProcessor(processorConfig, wsConfig, dataPusher)

	// 注册消息处理器
	testDeviceHandler := &TestDeviceDataHandler{}
	testAlertHandler := &TestAlertHandler{}
	processor.RegisterHandler(testDeviceHandler)
	processor.RegisterHandler(testAlertHandler)

	// 测试设备数据处理和推送
	t.Run("ProcessDeviceData", func(t *testing.T) {
		// 创建测试设备数据
		device := &models.Device{
			DeviceID:   "test-device-001",
			DeviceType: "sensor",
			Timestamp:  time.Now().UnixMilli(),
			DeviceInfo: models.DeviceInfo{
				Model:           "TempSensor-v1",
				Manufacturer:    "TestCorp",
				FirmwareVersion: "1.0.0",
				HardwareVersion: "1.0.0",
				SerialNumber:    "TS001",
				BatteryLevel:    85,
				SignalStrength:  -45,
				NetworkType:     models.NetworkTypeWiFi,
			},
			Location: models.LocationInfo{
				Building: "Building-A",
				Floor:    1,
				Room:     "Room-101",
				Zone:     "Zone-1",
			},
			SensorData: models.SensorData{
				Temperature: &models.TemperatureSensor{
					Value:  25.5,
					Unit:   "°C",
					Status: models.SensorStatusNormal,
				},
				Humidity: &models.HumiditySensor{
					Value:  60.0,
					Unit:   "%",
					Status: models.SensorStatusNormal,
				},
			},
			Status:   models.DeviceStatusOnline,
			LastSeen: time.Now().UnixMilli(),
		}

		devicePayload := &models.DeviceDataPayload{
			Device: device,
		}

		message := &models.KafkaMessage{
			MessageID:   "msg-001",
			MessageType: models.MessageTypeDeviceData,
			Timestamp:   time.Now().Unix(),
			Payload:     devicePayload,
		}

		// 处理消息
		err := processor.ProcessMessage(ctx, message)
		assert.NoError(t, err)

		// 验证推送统计
		stats := processor.GetPushStats()
		assert.Equal(t, int64(1), stats.PushSuccessCount)
		assert.Equal(t, int64(0), stats.PushFailureCount)
		assert.True(t, stats.EnablePush)
	})

	// 测试告警数据处理和推送
	t.Run("ProcessAlert", func(t *testing.T) {
		// 创建测试告警数据
		alert := &models.Alert{
			AlertID:     "alert-001",
			DeviceID:    "test-device-001",
			AlertType:   models.AlertTypeThreshold,
			Severity:    models.AlertSeverityWarning,
			Title:       "Temperature High",
			Description: "Temperature exceeds threshold",
			Timestamp:   time.Now().UnixMilli(),
			Status:      models.AlertStatusActive,
		}

		alertPayload := &models.AlertPayload{
			Alert:    alert,
			DeviceID: "test-device-001",
			Resolved: false,
		}

		message := &models.KafkaMessage{
			MessageID:   "msg-002",
			MessageType: models.MessageTypeAlert,
			Timestamp:   time.Now().Unix(),
			Payload:     alertPayload,
		}

		// 处理消息
		err := processor.ProcessMessage(ctx, message)
		assert.NoError(t, err)

		// 验证推送统计
		stats := processor.GetPushStats()
		assert.Equal(t, int64(2), stats.PushSuccessCount) // 包括之前的设备数据
		assert.Equal(t, int64(0), stats.PushFailureCount)
	})

	// 测试WebSocket推送开关
	t.Run("EnableDisablePush", func(t *testing.T) {
		// 禁用推送
		processor.DisableWebSocketPush()
		assert.False(t, processor.IsWebSocketPushEnabled())

		// 启用推送
		processor.EnableWebSocketPush()
		assert.True(t, processor.IsWebSocketPushEnabled())
	})
}

func TestWebSocketServer(t *testing.T) {
	// 创建数据推送器
	pusherConfig := &wsService.DataPusherConfig{
		QueueSize:         100,
		WorkerCount:       2,
		WriteTimeout:      5 * time.Second,
		PingInterval:      30 * time.Second,
		ClientTimeout:     60 * time.Second,
		MaxMessageSize:    1024 * 1024,
		EnableCompression: true,
	}
	dataPusher := wsService.NewWebSocketDataPusher(pusherConfig)

	// 创建WebSocket服务器
	serverConfig := &wsService.ServerConfig{
		Host:              "127.0.0.1",
		Port:              8082, // 使用不同端口避免冲突
		Path:              "/ws",
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		HandshakeTimeout:  10 * time.Second,
		CheckOrigin:       false,
		EnableCompression: true,
		MaxMessageSize:    1024 * 1024,
	}

	server := wsService.NewWebSocketServer(serverConfig, dataPusher)

	// 启动服务器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server.Start(ctx)
	require.NoError(t, err)
	defer server.Stop()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	t.Run("WebSocketConnection", func(t *testing.T) {
		// 连接WebSocket
		u := url.URL{Scheme: "ws", Host: "127.0.0.1:8082", Path: "/ws"}
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		require.NoError(t, err)
		defer conn.Close()

		// 验证连接成功
		assert.Equal(t, 1, server.GetClientCount())

		// 发送过滤器消息
		filterMsg := map[string]interface{}{
			"type": "filter",
			"data": map[string]interface{}{
				"device_ids": []string{"test-device-001"},
			},
		}

		err = conn.WriteJSON(filterMsg)
		assert.NoError(t, err)

		// 发送ping消息
		pingMsg := map[string]interface{}{
			"type": "ping",
		}

		err = conn.WriteJSON(pingMsg)
		assert.NoError(t, err)

		// 读取pong响应
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		var response map[string]interface{}
		err = conn.ReadJSON(&response)
		assert.NoError(t, err)
		assert.Equal(t, "pong", response["type"])
	})

	t.Run("HealthEndpoint", func(t *testing.T) {
		resp, err := http.Get("http://127.0.0.1:8082/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", health["status"])
	})

	t.Run("StatsEndpoint", func(t *testing.T) {
		resp, err := http.Get("http://127.0.0.1:8082/stats")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var stats map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&stats)
		assert.NoError(t, err)
		assert.Contains(t, stats, "server")
		assert.Contains(t, stats, "pusher")
	})
}

func TestWebSocketDataPusher(t *testing.T) {
	// 创建数据推送器
	config := &wsService.DataPusherConfig{
		QueueSize:         100,
		WorkerCount:       2,
		WriteTimeout:      5 * time.Second,
		PingInterval:      30 * time.Second,
		ClientTimeout:     60 * time.Second,
		MaxMessageSize:    1024 * 1024,
		EnableCompression: true,
	}

	pusher := wsService.NewWebSocketDataPusher(config)

	// 启动推送器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := pusher.Start(ctx)
	require.NoError(t, err)
	defer pusher.Stop()

	t.Run("ClientManagement", func(t *testing.T) {
		// 跳过客户端管理测试，因为需要真实的WebSocket连接
		// 这里只测试基本的统计功能
		assert.Equal(t, 0, pusher.GetClientCount())

		// 验证初始状态
		stats := pusher.GetStats()
		assert.Equal(t, int64(0), stats.TotalMessages)
	})

	t.Run("DataPushing", func(t *testing.T) {
		// 创建测试设备数据
		device := &models.Device{
			DeviceID:   "test-device-001",
			DeviceType: "sensor",
			Timestamp:  time.Now().UnixMilli(),
			Status:     models.DeviceStatusOnline,
		}

		devicePayload := &models.DeviceDataPayload{
			Device: device,
		}

		// 推送设备数据
		err := pusher.PushDeviceData(ctx, devicePayload)
		assert.NoError(t, err)

		// 创建测试告警数据
		alert := &models.Alert{
			AlertID:   "alert-001",
			DeviceID:  "test-device-001",
			AlertType: models.AlertTypeThreshold,
			Severity:  models.AlertSeverityWarning,
			Timestamp: time.Now().UnixMilli(),
		}

		alertPayload := &models.AlertPayload{
			Alert:    alert,
			DeviceID: "device-001",
		}

		// 推送告警数据
		err = pusher.PushAlert(ctx, alertPayload)
		assert.NoError(t, err)

		// 验证统计信息
		stats := pusher.GetStats()
		assert.Equal(t, int64(2), stats.TotalMessages)
		assert.Equal(t, int64(1), stats.DeviceMessages)
		assert.Equal(t, int64(1), stats.AlertMessages)
	})
}
