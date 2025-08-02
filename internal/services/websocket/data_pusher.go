package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"simplied-iot-monitoring-go/internal/models"
)

// DataPusher WebSocket数据推送接口
type DataPusher interface {
	// PushDeviceData 推送设备数据到WebSocket客户端
	PushDeviceData(ctx context.Context, deviceData *models.DeviceDataPayload) error
	
	// PushAlert 推送告警数据到WebSocket客户端
	PushAlert(ctx context.Context, alertData *models.AlertPayload) error
	
	// AddClient 添加WebSocket客户端
	AddClient(client *Client) error
	
	// RemoveClient 移除WebSocket客户端
	RemoveClient(clientID string) error
	
	// GetClientCount 获取当前客户端数量
	GetClientCount() int
	
	// Start 启动数据推送服务
	Start(ctx context.Context) error
	
	// Stop 停止数据推送服务
	Stop() error
}



// WebSocketDataPusher WebSocket数据推送器实现
type WebSocketDataPusher struct {
	clients    map[string]*Client
	clientsMux sync.RWMutex
	
	// 消息队列
	deviceDataQueue chan *models.DeviceDataPayload
	alertQueue      chan *models.AlertPayload
	
	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	// 配置
	config *DataPusherConfig
	
	// 统计
	stats *PusherStats
}

// DataPusherConfig 数据推送器配置
type DataPusherConfig struct {
	QueueSize        int           `yaml:"queue_size"`
	WorkerCount      int           `yaml:"worker_count"`
	WriteTimeout     time.Duration `yaml:"write_timeout"`
	PingInterval     time.Duration `yaml:"ping_interval"`
	ClientTimeout    time.Duration `yaml:"client_timeout"`
	MaxMessageSize   int64         `yaml:"max_message_size"`
	EnableCompression bool         `yaml:"enable_compression"`
}

// PusherStats 推送器统计信息
type PusherStats struct {
	TotalMessages    int64 `json:"total_messages"`
	DeviceMessages   int64 `json:"device_messages"`
	AlertMessages    int64 `json:"alert_messages"`
	FailedMessages   int64 `json:"failed_messages"`
	ActiveClients    int64 `json:"active_clients"`
	TotalClients     int64 `json:"total_clients"`
	LastMessageTime  time.Time `json:"last_message_time"`
	mutex            sync.RWMutex
}

// NewWebSocketDataPusher 创建WebSocket数据推送器
func NewWebSocketDataPusher(config *DataPusherConfig) *WebSocketDataPusher {
	if config == nil {
		config = &DataPusherConfig{
			QueueSize:        1000,
			WorkerCount:      4,
			WriteTimeout:     10 * time.Second,
			PingInterval:     54 * time.Second,
			ClientTimeout:    60 * time.Second,
			MaxMessageSize:   1024 * 1024, // 1MB
			EnableCompression: true,
		}
	}
	
	return &WebSocketDataPusher{
		clients:         make(map[string]*Client),
		deviceDataQueue: make(chan *models.DeviceDataPayload, config.QueueSize),
		alertQueue:      make(chan *models.AlertPayload, config.QueueSize),
		config:          config,
		stats:           &PusherStats{},
	}
}

// Start 启动数据推送服务
func (dp *WebSocketDataPusher) Start(ctx context.Context) error {
	dp.ctx, dp.cancel = context.WithCancel(ctx)
	
	// 启动工作协程
	for i := 0; i < dp.config.WorkerCount; i++ {
		dp.wg.Add(1)
		go dp.deviceDataWorker()
		
		dp.wg.Add(1)
		go dp.alertWorker()
	}
	
	// 启动客户端管理协程
	dp.wg.Add(1)
	go dp.clientManager()
	
	log.Printf("WebSocket数据推送器已启动，工作协程数: %d", dp.config.WorkerCount*2)
	return nil
}

// Stop 停止数据推送服务
func (dp *WebSocketDataPusher) Stop() error {
	if dp.cancel != nil {
		dp.cancel()
	}
	
	// 关闭队列
	close(dp.deviceDataQueue)
	close(dp.alertQueue)
	
	// 等待所有协程结束
	dp.wg.Wait()
	
	// 关闭所有客户端连接
	dp.clientsMux.Lock()
	for _, client := range dp.clients {
		client.Conn.Close()
		close(client.Send)
	}
	dp.clients = make(map[string]*Client)
	dp.clientsMux.Unlock()
	
	log.Printf("WebSocket数据推送器已停止")
	return nil
}

// PushDeviceData 推送设备数据
func (dp *WebSocketDataPusher) PushDeviceData(ctx context.Context, deviceData *models.DeviceDataPayload) error {
	select {
	case dp.deviceDataQueue <- deviceData:
		dp.updateStats(func(s *PusherStats) {
			s.TotalMessages++
			s.DeviceMessages++
			s.LastMessageTime = time.Now()
		})
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// 队列满，记录失败
		dp.updateStats(func(s *PusherStats) {
			s.FailedMessages++
		})
		return ErrQueueFull
	}
}

// PushAlert 推送告警数据
func (dp *WebSocketDataPusher) PushAlert(ctx context.Context, alertData *models.AlertPayload) error {
	select {
	case dp.alertQueue <- alertData:
		dp.updateStats(func(s *PusherStats) {
			s.TotalMessages++
			s.AlertMessages++
			s.LastMessageTime = time.Now()
		})
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// 队列满，记录失败
		dp.updateStats(func(s *PusherStats) {
			s.FailedMessages++
		})
		return ErrQueueFull
	}
}

// AddClient 添加客户端
func (dp *WebSocketDataPusher) AddClient(client *Client) error {
	dp.clientsMux.Lock()
	defer dp.clientsMux.Unlock()
	
	dp.clients[client.ID] = client
	client.LastSeen = time.Now()
	
	dp.updateStats(func(s *PusherStats) {
		s.ActiveClients = int64(len(dp.clients))
		s.TotalClients++
	})
	
	log.Printf("WebSocket客户端已连接: %s, 当前客户端数: %d", client.ID, len(dp.clients))
	return nil
}

// RemoveClient 移除客户端
func (dp *WebSocketDataPusher) RemoveClient(clientID string) error {
	dp.clientsMux.Lock()
	defer dp.clientsMux.Unlock()
	
	if client, exists := dp.clients[clientID]; exists {
		client.Conn.Close()
		close(client.Send)
		delete(dp.clients, clientID)
		
		dp.updateStats(func(s *PusherStats) {
			s.ActiveClients = int64(len(dp.clients))
		})
		
		log.Printf("WebSocket客户端已断开: %s, 当前客户端数: %d", clientID, len(dp.clients))
	}
	
	return nil
}

// GetClientCount 获取客户端数量
func (dp *WebSocketDataPusher) GetClientCount() int {
	dp.clientsMux.RLock()
	defer dp.clientsMux.RUnlock()
	return len(dp.clients)
}

// deviceDataWorker 设备数据工作协程
func (dp *WebSocketDataPusher) deviceDataWorker() {
	defer dp.wg.Done()
	
	for {
		select {
		case deviceData, ok := <-dp.deviceDataQueue:
			if !ok {
				return
			}
			dp.broadcastDeviceData(deviceData)
		case <-dp.ctx.Done():
			return
		}
	}
}

// alertWorker 告警数据工作协程
func (dp *WebSocketDataPusher) alertWorker() {
	defer dp.wg.Done()
	
	for {
		select {
		case alertData, ok := <-dp.alertQueue:
			if !ok {
				return
			}
			dp.broadcastAlert(alertData)
		case <-dp.ctx.Done():
			return
		}
	}
}

// clientManager 客户端管理协程
func (dp *WebSocketDataPusher) clientManager() {
	defer dp.wg.Done()
	
	ticker := time.NewTicker(dp.config.PingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			dp.pingClients()
			dp.cleanupInactiveClients()
		case <-dp.ctx.Done():
			return
		}
	}
}

// broadcastDeviceData 广播设备数据
func (dp *WebSocketDataPusher) broadcastDeviceData(deviceData *models.DeviceDataPayload) {
	message := &WebSocketMessage{
		Type:      "device_data",
		Timestamp: time.Now(),
		Data:      deviceData,
	}
	
	dp.broadcastToClients(message, func(client *Client, data *models.DeviceDataPayload) bool {
		return dp.matchesDeviceFilter(client.Filter, data)
	})
}

// broadcastAlert 广播告警数据
func (dp *WebSocketDataPusher) broadcastAlert(alertData *models.AlertPayload) {
	message := &WebSocketMessage{
		Type:      "alert",
		Timestamp: time.Now(),
		Data:      alertData,
	}
	
	dp.broadcastToClients(message, func(client *Client, data *models.AlertPayload) bool {
		return dp.matchesAlertFilter(client.Filter, data)
	})
}

// broadcastToClients 向客户端广播消息
func (dp *WebSocketDataPusher) broadcastToClients(message *WebSocketMessage, filter interface{}) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化WebSocket消息失败: %v", err)
		return
	}
	
	dp.clientsMux.RLock()
	clients := make([]*Client, 0, len(dp.clients))
	for _, client := range dp.clients {
		clients = append(clients, client)
	}
	dp.clientsMux.RUnlock()
	
	for _, client := range clients {
		// 应用过滤器
		shouldSend := true
		if filter != nil {
			switch f := filter.(type) {
			case func(*Client, *models.DeviceDataPayload) bool:
				if data, ok := message.Data.(*models.DeviceDataPayload); ok {
					shouldSend = f(client, data)
				}
			case func(*Client, *models.AlertPayload) bool:
				if data, ok := message.Data.(*models.AlertPayload); ok {
					shouldSend = f(client, data)
				}
			}
		}
		
		if shouldSend {
			select {
			case client.Send <- messageBytes:
				client.LastSeen = time.Now()
			default:
				// 客户端发送队列满，移除客户端
				go dp.RemoveClient(client.ID)
			}
		}
	}
}

// matchesDeviceFilter 检查设备数据是否匹配过滤器
func (dp *WebSocketDataPusher) matchesDeviceFilter(filter *ClientFilter, data *models.DeviceDataPayload) bool {
	if filter == nil || data.Device == nil {
		return true
	}
	
	// 检查设备ID过滤器
	if len(filter.DeviceIDs) > 0 {
		if !filter.DeviceIDs[data.Device.DeviceID] {
			return false
		}
	}
	
	// 检查设备类型过滤器
	if len(filter.DeviceTypes) > 0 {
		if !filter.DeviceTypes[data.Device.DeviceType] {
			return false
		}
	}
	
	// 检查位置过滤器
	if len(filter.Locations) > 0 {
		// LocationInfo是结构体，不是指针，使用Building字段作为位置标识
		if !filter.Locations[data.Device.Location.Building] {
			return false
		}
	}
	
	return true
}

// matchesAlertFilter 检查告警数据是否匹配过滤器
func (dp *WebSocketDataPusher) matchesAlertFilter(filter *ClientFilter, data *models.AlertPayload) bool {
	if filter == nil || data.Alert == nil {
		return true
	}
	
	// 检查设备ID过滤器
	if len(filter.DeviceIDs) > 0 {
		if !filter.DeviceIDs[data.DeviceID] {
			return false
		}
	}
	
	// 检查告警级别过滤器
	if len(filter.AlertLevels) > 0 {
		if !filter.AlertLevels[string(data.Alert.Severity)] {
			return false
		}
	}
	
	return true
}

// pingClients 向所有客户端发送ping消息
func (dp *WebSocketDataPusher) pingClients() {
	dp.clientsMux.RLock()
	clients := make([]*Client, 0, len(dp.clients))
	for _, client := range dp.clients {
		clients = append(clients, client)
	}
	dp.clientsMux.RUnlock()
	
	for _, client := range clients {
		client.Conn.SetWriteDeadline(time.Now().Add(dp.config.WriteTimeout))
		if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			go dp.RemoveClient(client.ID)
		}
	}
}

// cleanupInactiveClients 清理不活跃的客户端
func (dp *WebSocketDataPusher) cleanupInactiveClients() {
	now := time.Now()
	inactiveClients := make([]string, 0)
	
	dp.clientsMux.RLock()
	for clientID, client := range dp.clients {
		if now.Sub(client.LastSeen) > dp.config.ClientTimeout {
			inactiveClients = append(inactiveClients, clientID)
		}
	}
	dp.clientsMux.RUnlock()
	
	for _, clientID := range inactiveClients {
		dp.RemoveClient(clientID)
	}
}

// updateStats 更新统计信息
func (dp *WebSocketDataPusher) updateStats(updater func(*PusherStats)) {
	dp.stats.mutex.Lock()
	defer dp.stats.mutex.Unlock()
	updater(dp.stats)
}

// GetStats 获取统计信息
func (dp *WebSocketDataPusher) GetStats() *PusherStats {
	dp.stats.mutex.RLock()
	defer dp.stats.mutex.RUnlock()
	
	// 返回统计信息的副本
	return &PusherStats{
		TotalMessages:   dp.stats.TotalMessages,
		DeviceMessages:  dp.stats.DeviceMessages,
		AlertMessages:   dp.stats.AlertMessages,
		FailedMessages:  dp.stats.FailedMessages,
		ActiveClients:   dp.stats.ActiveClients,
		TotalClients:    dp.stats.TotalClients,
		LastMessageTime: dp.stats.LastMessageTime,
	}
}

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// 错误定义
var (
	ErrQueueFull = fmt.Errorf("消息队列已满")
)
