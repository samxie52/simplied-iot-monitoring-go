package consumer

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"simplied-iot-monitoring-go/internal/models"
	"simplied-iot-monitoring-go/internal/services/websocket"
)

// WebSocketIntegratedProcessor WebSocket集成消息处理器
// 嵌入默认消息处理器并添加实时数据推送功能
type WebSocketIntegratedProcessor struct {
	*DefaultMessageProcessor
	
	// WebSocket数据推送器
	dataPusher websocket.DataPusher
	
	// 推送统计
	pushSuccessCount int64
	pushFailureCount int64
	
	// 配置
	config *WebSocketProcessorConfig
}

// WebSocketProcessorConfig WebSocket处理器配置
type WebSocketProcessorConfig struct {
	// 是否启用WebSocket推送
	EnablePush bool `yaml:"enable_push"`
	
	// 推送超时时间
	PushTimeout time.Duration `yaml:"push_timeout"`
	
	// 是否异步推送
	AsyncPush bool `yaml:"async_push"`
	
	// 推送重试次数
	PushRetries int `yaml:"push_retries"`
	
	// 推送重试延迟
	PushRetryDelay time.Duration `yaml:"push_retry_delay"`
}

// NewWebSocketIntegratedProcessor 创建WebSocket集成消息处理器
func NewWebSocketIntegratedProcessor(
	processorConfig ProcessorConfig,
	wsConfig *WebSocketProcessorConfig,
	dataPusher websocket.DataPusher,
) *WebSocketIntegratedProcessor {
	
	// 创建默认消息处理器
	defaultProcessor := NewDefaultMessageProcessor(processorConfig)
	
	// 设置默认WebSocket配置
	if wsConfig == nil {
		wsConfig = &WebSocketProcessorConfig{
			EnablePush:     true,
			PushTimeout:    5 * time.Second,
			AsyncPush:      true,
			PushRetries:    3,
			PushRetryDelay: 1 * time.Second,
		}
	}
	
	return &WebSocketIntegratedProcessor{
		DefaultMessageProcessor: defaultProcessor,
		dataPusher:              dataPusher,
		config:                  wsConfig,
	}
}

// ProcessMessage 处理消息并推送到WebSocket客户端
func (wip *WebSocketIntegratedProcessor) ProcessMessage(ctx context.Context, message *models.KafkaMessage) error {
	// 首先调用默认处理器处理消息
	if err := wip.DefaultMessageProcessor.ProcessMessage(ctx, message); err != nil {
		return err
	}
	
	// 如果启用了WebSocket推送，则推送数据
	if wip.config.EnablePush && wip.dataPusher != nil {
		if wip.config.AsyncPush {
			// 异步推送
			go wip.pushMessageAsync(ctx, message)
		} else {
			// 同步推送
			if err := wip.pushMessage(ctx, message); err != nil {
				log.Printf("WebSocket推送失败: %v", err)
				// 推送失败不影响消息处理结果
			}
		}
	}
	
	return nil
}

// pushMessage 推送消息到WebSocket客户端
func (wip *WebSocketIntegratedProcessor) pushMessage(ctx context.Context, message *models.KafkaMessage) error {
	// 创建带超时的上下文
	pushCtx, cancel := context.WithTimeout(ctx, wip.config.PushTimeout)
	defer cancel()
	
	var err error
	
	// 根据消息类型推送数据
	switch message.MessageType {
	case models.MessageTypeDeviceData:
		err = wip.pushDeviceData(pushCtx, message)
	case models.MessageTypeAlert:
		err = wip.pushAlert(pushCtx, message)
	default:
		log.Printf("未知的消息类型，跳过WebSocket推送: %s", message.MessageType)
		return nil
	}
	
	// 更新推送统计
	if err != nil {
		atomic.AddInt64(&wip.pushFailureCount, 1)
		return fmt.Errorf("WebSocket推送失败: %w", err)
	}
	
	atomic.AddInt64(&wip.pushSuccessCount, 1)
	return nil
}

// pushMessageAsync 异步推送消息
func (wip *WebSocketIntegratedProcessor) pushMessageAsync(ctx context.Context, message *models.KafkaMessage) {
	// 带重试的推送
	for attempt := 0; attempt <= wip.config.PushRetries; attempt++ {
		if attempt > 0 {
			// 等待重试延迟
			select {
			case <-time.After(wip.config.PushRetryDelay):
			case <-ctx.Done():
				return
			}
		}
		
		if err := wip.pushMessage(ctx, message); err != nil {
			log.Printf("WebSocket异步推送失败 (第%d次): %v", attempt+1, err)
			continue
		}
		
		// 推送成功，退出重试循环
		return
	}
	
	log.Printf("WebSocket异步推送最终失败，已重试%d次: MessageID=%s", 
		wip.config.PushRetries, message.MessageID)
}

// pushDeviceData 推送设备数据
func (wip *WebSocketIntegratedProcessor) pushDeviceData(ctx context.Context, message *models.KafkaMessage) error {
	// 提取设备数据载荷
	payload, ok := message.Payload.(*models.DeviceDataPayload)
	if !ok {
		return fmt.Errorf("无效的设备数据载荷类型")
	}
	
	// 推送到WebSocket客户端
	if err := wip.dataPusher.PushDeviceData(ctx, payload); err != nil {
		return fmt.Errorf("推送设备数据失败: %w", err)
	}
	
	log.Printf("设备数据已推送到WebSocket客户端: DeviceID=%s, ClientCount=%d", 
		payload.Device.DeviceID, wip.dataPusher.GetClientCount())
	
	return nil
}

// pushAlert 推送告警数据
func (wip *WebSocketIntegratedProcessor) pushAlert(ctx context.Context, message *models.KafkaMessage) error {
	// 提取告警数据载荷
	payload, ok := message.Payload.(*models.AlertPayload)
	if !ok {
		return fmt.Errorf("无效的告警数据载荷类型")
	}
	
	// 推送到WebSocket客户端
	if err := wip.dataPusher.PushAlert(ctx, payload); err != nil {
		return fmt.Errorf("推送告警数据失败: %w", err)
	}
	
	log.Printf("告警数据已推送到WebSocket客户端: AlertID=%s, DeviceID=%s, ClientCount=%d", 
		payload.Alert.AlertID, payload.DeviceID, wip.dataPusher.GetClientCount())
	
	return nil
}

// GetPushStats 获取推送统计信息
func (wip *WebSocketIntegratedProcessor) GetPushStats() *WebSocketPushStats {
	return &WebSocketPushStats{
		PushSuccessCount: atomic.LoadInt64(&wip.pushSuccessCount),
		PushFailureCount: atomic.LoadInt64(&wip.pushFailureCount),
		ClientCount:      int64(wip.dataPusher.GetClientCount()),
		EnablePush:       wip.config.EnablePush,
		AsyncPush:        wip.config.AsyncPush,
	}
}

// WebSocketPushStats WebSocket推送统计信息
type WebSocketPushStats struct {
	PushSuccessCount int64 `json:"push_success_count"`
	PushFailureCount int64 `json:"push_failure_count"`
	ClientCount      int64 `json:"client_count"`
	EnablePush       bool  `json:"enable_push"`
	AsyncPush        bool  `json:"async_push"`
}

// SetDataPusher 设置数据推送器
func (wip *WebSocketIntegratedProcessor) SetDataPusher(dataPusher websocket.DataPusher) {
	wip.dataPusher = dataPusher
}

// GetDataPusher 获取数据推送器
func (wip *WebSocketIntegratedProcessor) GetDataPusher() websocket.DataPusher {
	return wip.dataPusher
}

// EnableWebSocketPush 启用WebSocket推送
func (wip *WebSocketIntegratedProcessor) EnableWebSocketPush() {
	wip.config.EnablePush = true
	log.Printf("WebSocket推送已启用")
}

// DisableWebSocketPush 禁用WebSocket推送
func (wip *WebSocketIntegratedProcessor) DisableWebSocketPush() {
	wip.config.EnablePush = false
	log.Printf("WebSocket推送已禁用")
}

// IsWebSocketPushEnabled 检查WebSocket推送是否启用
func (wip *WebSocketIntegratedProcessor) IsWebSocketPushEnabled() bool {
	return wip.config.EnablePush
}

// UpdateWebSocketConfig 更新WebSocket配置
func (wip *WebSocketIntegratedProcessor) UpdateWebSocketConfig(config *WebSocketProcessorConfig) {
	if config != nil {
		wip.config = config
		log.Printf("WebSocket处理器配置已更新: EnablePush=%v, AsyncPush=%v", 
			config.EnablePush, config.AsyncPush)
	}
}
