package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/google/uuid"
)

// WebSocketServer WebSocket服务器
type WebSocketServer struct {
	// 数据推送器
	dataPusher DataPusher
	
	// WebSocket升级器
	upgrader websocket.Upgrader
	
	// 服务器配置
	config *ServerConfig
	
	// HTTP服务器
	httpServer *http.Server
	
	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	// 统计
	stats *ServerStats
}

// ServerConfig WebSocket服务器配置
type ServerConfig struct {
	Host              string        `yaml:"host"`
	Port              int           `yaml:"port"`
	Path              string        `yaml:"path"`
	ReadBufferSize    int           `yaml:"read_buffer_size"`
	WriteBufferSize   int           `yaml:"write_buffer_size"`
	HandshakeTimeout  time.Duration `yaml:"handshake_timeout"`
	CheckOrigin       bool          `yaml:"check_origin"`
	EnableCompression bool          `yaml:"enable_compression"`
	MaxMessageSize    int64         `yaml:"max_message_size"`
}

// ServerStats 服务器统计信息
type ServerStats struct {
	TotalConnections    int64     `json:"total_connections"`
	ActiveConnections   int64     `json:"active_connections"`
	FailedConnections   int64     `json:"failed_connections"`
	TotalMessages       int64     `json:"total_messages"`
	LastConnectionTime  time.Time `json:"last_connection_time"`
	ServerStartTime     time.Time `json:"server_start_time"`
	mutex               sync.RWMutex
}

// NewWebSocketServer 创建WebSocket服务器
func NewWebSocketServer(config *ServerConfig, dataPusher DataPusher) *WebSocketServer {
	if config == nil {
		config = &ServerConfig{
			Host:              "0.0.0.0",
			Port:              8080,
			Path:              "/ws",
			ReadBufferSize:    1024,
			WriteBufferSize:   1024,
			HandshakeTimeout:  10 * time.Second,
			CheckOrigin:       false,
			EnableCompression: true,
			MaxMessageSize:    1024 * 1024, // 1MB
		}
	}
	
	upgrader := websocket.Upgrader{
		ReadBufferSize:   config.ReadBufferSize,
		WriteBufferSize:  config.WriteBufferSize,
		HandshakeTimeout: config.HandshakeTimeout,
		EnableCompression: config.EnableCompression,
		CheckOrigin: func(r *http.Request) bool {
			if config.CheckOrigin {
				// 实现自定义的Origin检查逻辑
				return true // 简化实现，实际应该检查Origin
			}
			return true
		},
	}
	
	return &WebSocketServer{
		dataPusher: dataPusher,
		upgrader:   upgrader,
		config:     config,
		stats: &ServerStats{
			ServerStartTime: time.Now(),
		},
	}
}

// Start 启动WebSocket服务器
func (ws *WebSocketServer) Start(ctx context.Context) error {
	ws.ctx, ws.cancel = context.WithCancel(ctx)
	
	// 启动数据推送器
	if err := ws.dataPusher.Start(ws.ctx); err != nil {
		return fmt.Errorf("启动数据推送器失败: %w", err)
	}
	
	// 设置HTTP路由
	mux := http.NewServeMux()
	mux.HandleFunc(ws.config.Path, ws.handleWebSocket)
	mux.HandleFunc("/health", ws.handleHealth)
	mux.HandleFunc("/stats", ws.handleStats)
	
	// 创建HTTP服务器
	addr := fmt.Sprintf("%s:%d", ws.config.Host, ws.config.Port)
	ws.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	
	// 启动HTTP服务器
	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		log.Printf("WebSocket服务器启动在 %s%s", addr, ws.config.Path)
		if err := ws.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket服务器启动失败: %v", err)
		}
	}()
	
	return nil
}

// Stop 停止WebSocket服务器
func (ws *WebSocketServer) Stop() error {
	if ws.cancel != nil {
		ws.cancel()
	}
	
	// 停止数据推送器
	if err := ws.dataPusher.Stop(); err != nil {
		log.Printf("停止数据推送器失败: %v", err)
	}
	
	// 停止HTTP服务器
	if ws.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := ws.httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("WebSocket服务器关闭失败: %v", err)
		}
	}
	
	// 等待所有协程结束
	ws.wg.Wait()
	
	log.Printf("WebSocket服务器已停止")
	return nil
}

// handleWebSocket 处理WebSocket连接
func (ws *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket连接
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		ws.updateStats(func(s *ServerStats) {
			s.FailedConnections++
		})
		return
	}
	
	// 生成客户端ID
	clientID := uuid.New().String()
	
	// 创建客户端
	client := &Client{
		ID:       clientID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Filter:   &ClientFilter{},
		LastSeen: time.Now(),
	}
	
	// 设置连接参数
	conn.SetReadLimit(ws.config.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		client.LastSeen = time.Now()
		return nil
	})
	
	// 添加客户端到数据推送器
	if err := ws.dataPusher.AddClient(client); err != nil {
		log.Printf("添加WebSocket客户端失败: %v", err)
		conn.Close()
		return
	}
	
	// 更新统计信息
	ws.updateStats(func(s *ServerStats) {
		s.TotalConnections++
		s.ActiveConnections++
		s.LastConnectionTime = time.Now()
	})
	
	// 启动客户端读写协程
	ws.wg.Add(2)
	go ws.clientReader(client)
	go ws.clientWriter(client)
}

// clientReader 客户端读取协程
func (ws *WebSocketServer) clientReader(client *Client) {
	defer func() {
		ws.wg.Done()
		ws.dataPusher.RemoveClient(client.ID)
		client.Conn.Close()
		ws.updateStats(func(s *ServerStats) {
			s.ActiveConnections--
		})
	}()
	
	for {
		select {
		case <-ws.ctx.Done():
			return
		default:
			_, message, err := client.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket读取错误: %v", err)
				}
				return
			}
			
			// 处理客户端消息
			ws.handleClientMessage(client, message)
			
			// 更新统计信息
			ws.updateStats(func(s *ServerStats) {
				s.TotalMessages++
			})
		}
	}
}

// clientWriter 客户端写入协程
func (ws *WebSocketServer) clientWriter(client *Client) {
	defer func() {
		ws.wg.Done()
		client.Conn.Close()
	}()
	
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket写入错误: %v", err)
				return
			}
			
		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			
		case <-ws.ctx.Done():
			return
		}
	}
}

// handleClientMessage 处理客户端消息
func (ws *WebSocketServer) handleClientMessage(client *Client, message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("解析客户端消息失败: %v", err)
		return
	}
	
	// 处理不同类型的消息
	switch msgType, ok := msg["type"].(string); {
	case ok && msgType == "filter":
		ws.handleFilterMessage(client, msg)
	case ok && msgType == "ping":
		ws.handlePingMessage(client)
	default:
		log.Printf("未知的客户端消息类型: %v", msgType)
	}
}

// handleFilterMessage 处理过滤器消息
func (ws *WebSocketServer) handleFilterMessage(client *Client, msg map[string]interface{}) {
	filterData, ok := msg["data"].(map[string]interface{})
	if !ok {
		log.Printf("无效的过滤器数据格式")
		return
	}
	
	// 解析过滤器
	filter := &ClientFilter{}
	
	if deviceIDs, ok := filterData["device_ids"].([]interface{}); ok {
		filter.DeviceIDs = make(map[string]bool)
		for _, id := range deviceIDs {
			if idStr, ok := id.(string); ok {
				filter.DeviceIDs[idStr] = true
			}
		}
	}
	
	if deviceTypes, ok := filterData["device_types"].([]interface{}); ok {
		filter.DeviceTypes = make(map[string]bool)
		for _, t := range deviceTypes {
			if tStr, ok := t.(string); ok {
				filter.DeviceTypes[tStr] = true
			}
		}
	}
	
	if locations, ok := filterData["locations"].([]interface{}); ok {
		filter.Locations = make(map[string]bool)
		for _, loc := range locations {
			if locStr, ok := loc.(string); ok {
				filter.Locations[locStr] = true
			}
		}
	}
	
	if alertLevels, ok := filterData["alert_levels"].([]interface{}); ok {
		filter.AlertLevels = make(map[string]bool)
		for _, level := range alertLevels {
			if levelStr, ok := level.(string); ok {
				filter.AlertLevels[levelStr] = true
			}
		}
	}
	
	// 更新客户端过滤器
	client.mutex.Lock()
	client.Filter = filter
	client.mutex.Unlock()
	
	log.Printf("客户端 %s 过滤器已更新", client.ID)
}

// handlePingMessage 处理ping消息
func (ws *WebSocketServer) handlePingMessage(client *Client) {
	// 发送pong响应
	pongMsg := map[string]interface{}{
		"type":      "pong",
		"timestamp": time.Now(),
	}
	
	if msgBytes, err := json.Marshal(pongMsg); err == nil {
		select {
		case client.Send <- msgBytes:
		default:
			log.Printf("客户端 %s 发送队列满，无法发送pong", client.ID)
		}
	}
}

// handleHealth 处理健康检查
func (ws *WebSocketServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":           "healthy",
		"timestamp":        time.Now(),
		"active_clients":   ws.dataPusher.GetClientCount(),
		"server_uptime":    time.Since(ws.stats.ServerStartTime).String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleStats 处理统计信息
func (ws *WebSocketServer) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := ws.GetStats()
	pusherStats := ws.dataPusher.(*WebSocketDataPusher).GetStats()
	
	response := map[string]interface{}{
		"server": stats,
		"pusher": pusherStats,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetStats 获取服务器统计信息
func (ws *WebSocketServer) GetStats() *ServerStats {
	ws.stats.mutex.RLock()
	defer ws.stats.mutex.RUnlock()
	
	return &ServerStats{
		TotalConnections:   ws.stats.TotalConnections,
		ActiveConnections:  ws.stats.ActiveConnections,
		FailedConnections:  ws.stats.FailedConnections,
		TotalMessages:      ws.stats.TotalMessages,
		LastConnectionTime: ws.stats.LastConnectionTime,
		ServerStartTime:    ws.stats.ServerStartTime,
	}
}

// updateStats 更新统计信息
func (ws *WebSocketServer) updateStats(updater func(*ServerStats)) {
	ws.stats.mutex.Lock()
	defer ws.stats.mutex.Unlock()
	updater(ws.stats)
}

// GetAddress 获取服务器地址
func (ws *WebSocketServer) GetAddress() string {
	return fmt.Sprintf("%s:%d", ws.config.Host, ws.config.Port)
}

// GetPath 获取WebSocket路径
func (ws *WebSocketServer) GetPath() string {
	return ws.config.Path
}

// GetClientCount 获取客户端数量
func (ws *WebSocketServer) GetClientCount() int {
	return ws.dataPusher.GetClientCount()
}
