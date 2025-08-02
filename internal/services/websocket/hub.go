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

// Hub WebSocket连接管理中心
// 负责管理所有WebSocket连接的生命周期和消息分发
type Hub struct {
	// 客户端连接池
	clients map[string]*Client
	
	// 通道管理
	register   chan *Client    // 客户端注册通道
	unregister chan *Client    // 客户端注销通道
	broadcast  chan []byte     // 广播消息通道
	
	// 消息路由
	deviceDataChan chan *models.DeviceDataPayload // 设备数据通道
	alertChan      chan *models.AlertPayload      // 告警数据通道
	
	// 并发控制
	mutex sync.RWMutex
	
	// 上下文控制
	ctx    context.Context
	cancel context.CancelFunc
	
	// 统计信息
	stats *HubStats
	
	// 配置
	config *HubConfig
}

// HubConfig Hub配置
type HubConfig struct {
	MaxConnections    int           `yaml:"max_connections"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	CleanupInterval   time.Duration `yaml:"cleanup_interval"`
	MessageBufferSize int           `yaml:"message_buffer_size"`
	BroadcastTimeout  time.Duration `yaml:"broadcast_timeout"`
}

// HubStats Hub统计信息
type HubStats struct {
	ConnectedClients    int64     `json:"connected_clients"`
	TotalConnections    int64     `json:"total_connections"`
	TotalDisconnections int64     `json:"total_disconnections"`
	MessagesSent        int64     `json:"messages_sent"`
	MessagesReceived    int64     `json:"messages_received"`
	BroadcastsSent      int64     `json:"broadcasts_sent"`
	LastActivity        time.Time `json:"last_activity"`
	StartTime           time.Time `json:"start_time"`
	mutex               sync.RWMutex
}

// NewHub 创建新的Hub实例
func NewHub(config *HubConfig) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	
	if config == nil {
		config = &HubConfig{
			MaxConnections:    1000,
			HeartbeatInterval: 15 * time.Second,
			CleanupInterval:   60 * time.Second,
			MessageBufferSize: 256,
			BroadcastTimeout:  5 * time.Second,
		}
	}
	
	return &Hub{
		clients:        make(map[string]*Client),
		register:       make(chan *Client, config.MessageBufferSize),
		unregister:     make(chan *Client, config.MessageBufferSize),
		broadcast:      make(chan []byte, config.MessageBufferSize),
		deviceDataChan: make(chan *models.DeviceDataPayload, config.MessageBufferSize),
		alertChan:      make(chan *models.AlertPayload, config.MessageBufferSize),
		ctx:            ctx,
		cancel:         cancel,
		stats: &HubStats{
			StartTime: time.Now(),
		},
		config: config,
	}
}

// Start 启动Hub
func (h *Hub) Start() error {
	log.Printf("Starting WebSocket Hub with max connections: %d", h.config.MaxConnections)
	
	// 启动主要的事件循环
	go h.run()
	
	// 启动心跳检查
	go h.heartbeatLoop()
	
	// 启动清理任务
	go h.cleanupLoop()
	
	return nil
}

// Stop 停止Hub
func (h *Hub) Stop() error {
	log.Println("Stopping WebSocket Hub...")
	
	h.cancel()
	
	// 关闭所有客户端连接
	h.mutex.Lock()
	for _, client := range h.clients {
		close(client.Send)
		client.Conn.Close()
	}
	h.mutex.Unlock()
	
	log.Println("WebSocket Hub stopped")
	return nil
}

// RegisterClient 注册新客户端
func (h *Hub) RegisterClient(client *Client) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	// 检查连接数限制
	if len(h.clients) >= h.config.MaxConnections {
		return fmt.Errorf("maximum connections reached: %d", h.config.MaxConnections)
	}
	
	// 检查客户端ID是否已存在
	if _, exists := h.clients[client.ID]; exists {
		return fmt.Errorf("client ID already exists: %s", client.ID)
	}
	
	select {
	case h.register <- client:
		return nil
	case <-h.ctx.Done():
		return fmt.Errorf("hub is shutting down")
	default:
		return fmt.Errorf("register channel is full")
	}
}

// UnregisterClient 注销客户端
func (h *Hub) UnregisterClient(client *Client) error {
	select {
	case h.unregister <- client:
		return nil
	case <-h.ctx.Done():
		return fmt.Errorf("hub is shutting down")
	default:
		return fmt.Errorf("unregister channel is full")
	}
}

// BroadcastToAll 向所有客户端广播消息
func (h *Hub) BroadcastToAll(message []byte) error {
	select {
	case h.broadcast <- message:
		h.updateStats(func(stats *HubStats) {
			stats.BroadcastsSent++
			stats.LastActivity = time.Now()
		})
		return nil
	case <-h.ctx.Done():
		return fmt.Errorf("hub is shutting down")
	default:
		return fmt.Errorf("broadcast channel is full")
	}
}

// BroadcastToGroup 向指定组广播消息
func (h *Hub) BroadcastToGroup(groupID string, message []byte) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	sentCount := 0
	for _, client := range h.clients {
		if client.Filter != nil && client.Filter.MatchesGroup(groupID) {
			select {
			case client.Send <- message:
				sentCount++
			default:
				log.Printf("Failed to send message to client %s: channel full", client.ID)
			}
		}
	}
	
	if sentCount > 0 {
		h.updateStats(func(stats *HubStats) {
			stats.MessagesSent += int64(sentCount)
			stats.LastActivity = time.Now()
		})
	}
	
	return nil
}

// PushDeviceData 推送设备数据
func (h *Hub) PushDeviceData(data *models.DeviceDataPayload) error {
	select {
	case h.deviceDataChan <- data:
		return nil
	case <-h.ctx.Done():
		return fmt.Errorf("hub is shutting down")
	default:
		return fmt.Errorf("device data channel is full")
	}
}

// PushAlert 推送告警数据
func (h *Hub) PushAlert(alert *models.AlertPayload) error {
	select {
	case h.alertChan <- alert:
		return nil
	case <-h.ctx.Done():
		return fmt.Errorf("hub is shutting down")
	default:
		return fmt.Errorf("alert channel is full")
	}
}

// GetStats 获取统计信息
func (h *Hub) GetStats() *HubStats {
	h.stats.mutex.RLock()
	defer h.stats.mutex.RUnlock()
	
	// 返回统计信息的副本
	statsCopy := *h.stats
	statsCopy.ConnectedClients = int64(len(h.clients))
	
	return &statsCopy
}

// GetConnections 获取当前连接数
func (h *Hub) GetConnections() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// GetClient 获取指定客户端
func (h *Hub) GetClient(clientID string) *Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.clients[clientID]
}

// run Hub主事件循环
func (h *Hub) run() {
	defer log.Println("Hub event loop stopped")
	
	for {
		select {
		case client := <-h.register:
			h.handleClientRegister(client)
			
		case client := <-h.unregister:
			h.handleClientUnregister(client)
			
		case message := <-h.broadcast:
			h.handleBroadcast(message)
			
		case deviceData := <-h.deviceDataChan:
			h.handleDeviceData(deviceData)
			
		case alert := <-h.alertChan:
			h.handleAlert(alert)
			
		case <-h.ctx.Done():
			return
		}
	}
}

// handleClientRegister 处理客户端注册
func (h *Hub) handleClientRegister(client *Client) {
	h.mutex.Lock()
	h.clients[client.ID] = client
	h.mutex.Unlock()
	
	log.Printf("Client registered: %s (total: %d)", client.ID, len(h.clients))
	
	h.updateStats(func(stats *HubStats) {
		stats.TotalConnections++
		stats.LastActivity = time.Now()
	})
	
	// 启动客户端处理协程
	go h.handleClient(client)
}

// handleClientUnregister 处理客户端注销
func (h *Hub) handleClientUnregister(client *Client) {
	h.mutex.Lock()
	if _, ok := h.clients[client.ID]; ok {
		delete(h.clients, client.ID)
		close(client.Send)
		client.Conn.Close()
	}
	h.mutex.Unlock()
	
	log.Printf("Client unregistered: %s (total: %d)", client.ID, len(h.clients))
	
	h.updateStats(func(stats *HubStats) {
		stats.TotalDisconnections++
		stats.LastActivity = time.Now()
	})
}

// handleBroadcast 处理广播消息
func (h *Hub) handleBroadcast(message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	sentCount := 0
	for _, client := range h.clients {
		select {
		case client.Send <- message:
			sentCount++
		default:
			log.Printf("Failed to send broadcast to client %s: channel full", client.ID)
		}
	}
	
	if sentCount > 0 {
		h.updateStats(func(stats *HubStats) {
			stats.MessagesSent += int64(sentCount)
		})
	}
}

// handleDeviceData 处理设备数据
func (h *Hub) handleDeviceData(data *models.DeviceDataPayload) {
	message, err := json.Marshal(map[string]interface{}{
		"type":      "device_data",
		"timestamp": time.Now().Unix(),
		"data":      data,
	})
	
	if err != nil {
		log.Printf("Failed to marshal device data: %v", err)
		return
	}
	
	// 根据过滤器发送给匹配的客户端
	h.sendToFilteredClients(message, func(client *Client) bool {
		return client.Filter == nil || client.Filter.MatchesDeviceData(data)
	})
}

// handleAlert 处理告警数据
func (h *Hub) handleAlert(alert *models.AlertPayload) {
	message, err := json.Marshal(map[string]interface{}{
		"type":      "alert",
		"timestamp": time.Now().Unix(),
		"data":      alert,
	})
	
	if err != nil {
		log.Printf("Failed to marshal alert: %v", err)
		return
	}
	
	// 根据过滤器发送给匹配的客户端
	h.sendToFilteredClients(message, func(client *Client) bool {
		return client.Filter == nil || client.Filter.MatchesAlert(alert)
	})
}

// sendToFilteredClients 发送消息给过滤匹配的客户端
func (h *Hub) sendToFilteredClients(message []byte, filter func(*Client) bool) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	sentCount := 0
	for _, client := range h.clients {
		if filter(client) {
			select {
			case client.Send <- message:
				sentCount++
			default:
				log.Printf("Failed to send filtered message to client %s: channel full", client.ID)
			}
		}
	}
	
	if sentCount > 0 {
		h.updateStats(func(stats *HubStats) {
			stats.MessagesSent += int64(sentCount)
		})
	}
}

// handleClient 处理单个客户端连接
func (h *Hub) handleClient(client *Client) {
	defer func() {
		h.UnregisterClient(client)
	}()
	
	// 设置读取超时和消息大小限制
	client.Conn.SetReadLimit(1024)
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		client.UpdateLastSeen()
		return nil
	})
	
	// 启动写入协程
	go h.clientWriter(client)
	
	// 读取客户端消息
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for client %s: %v", client.ID, err)
			}
			break
		}
		
		// 处理客户端消息
		h.handleClientMessage(client, message)
		
		h.updateStats(func(stats *HubStats) {
			stats.MessagesReceived++
			stats.LastActivity = time.Now()
		})
	}
}

// clientWriter 客户端写入协程
func (h *Hub) clientWriter(client *Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Failed to write message to client %s: %v", client.ID, err)
				return
			}
			
		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleClientMessage 处理客户端消息
func (h *Hub) handleClientMessage(client *Client, message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Invalid message from client %s: %v", client.ID, err)
		return
	}
	
	msgType, ok := msg["type"].(string)
	if !ok {
		log.Printf("Missing message type from client %s", client.ID)
		return
	}
	
	switch msgType {
	case "subscribe":
		h.handleSubscription(client, msg)
	case "unsubscribe":
		h.handleUnsubscription(client, msg)
	case "ping":
		h.handlePing(client)
	default:
		log.Printf("Unknown message type from client %s: %s", client.ID, msgType)
	}
}

// handleSubscription 处理订阅请求
func (h *Hub) handleSubscription(client *Client, msg map[string]interface{}) {
	// 这里可以实现订阅逻辑
	log.Printf("Subscription request from client %s: %v", client.ID, msg)
}

// handleUnsubscription 处理取消订阅请求
func (h *Hub) handleUnsubscription(client *Client, msg map[string]interface{}) {
	// 这里可以实现取消订阅逻辑
	log.Printf("Unsubscription request from client %s: %v", client.ID, msg)
}

// handlePing 处理ping消息
func (h *Hub) handlePing(client *Client) {
	client.UpdateLastSeen()
	
	response, _ := json.Marshal(map[string]interface{}{
		"type":      "pong",
		"timestamp": time.Now().Unix(),
	})
	
	select {
	case client.Send <- response:
	default:
		log.Printf("Failed to send pong to client %s: channel full", client.ID)
	}
}

// heartbeatLoop 心跳检查循环
func (h *Hub) heartbeatLoop() {
	ticker := time.NewTicker(h.config.HeartbeatInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			h.checkHeartbeats()
		case <-h.ctx.Done():
			return
		}
	}
}

// checkHeartbeats 检查客户端心跳
func (h *Hub) checkHeartbeats() {
	h.mutex.RLock()
	var staleClients []*Client
	timeout := time.Now().Add(-2 * h.config.HeartbeatInterval)
	
	for _, client := range h.clients {
		if client.GetLastSeen().Before(timeout) {
			staleClients = append(staleClients, client)
		}
	}
	h.mutex.RUnlock()
	
	// 移除过期的客户端
	for _, client := range staleClients {
		log.Printf("Removing stale client: %s", client.ID)
		h.UnregisterClient(client)
	}
}

// cleanupLoop 清理任务循环
func (h *Hub) cleanupLoop() {
	ticker := time.NewTicker(h.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			h.performCleanup()
		case <-h.ctx.Done():
			return
		}
	}
}

// performCleanup 执行清理任务
func (h *Hub) performCleanup() {
	// 这里可以实现内存清理、日志轮转等任务
	log.Printf("Performing cleanup - Current connections: %d", len(h.clients))
}

// updateStats 更新统计信息
func (h *Hub) updateStats(updater func(*HubStats)) {
	h.stats.mutex.Lock()
	defer h.stats.mutex.Unlock()
	updater(h.stats)
}
