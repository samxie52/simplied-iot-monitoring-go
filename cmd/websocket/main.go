package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// 版本信息（构建时注入）
var (
	Version   = "dev"
	BuildTime = "unknown"
	GoVersion = "unknown"
)

// IoTDevice 设备数据结构
type IoTDevice struct {
	DeviceID     string    `json:"device_id"`
	DeviceType   string    `json:"device_type"`
	Location     string    `json:"location"`
	Timestamp    time.Time `json:"timestamp"`
	Temperature  float64   `json:"temperature"`
	Humidity     float64   `json:"humidity"`
	Pressure     float64   `json:"pressure"`
	Status       string    `json:"status"`
	BatteryLevel float64   `json:"battery_level"`
}

// Client WebSocket客户端
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	id     string
	filter map[string]bool // 设备过滤器
}

// Hub WebSocket连接管理中心
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

// NewHub 创建新的Hub实例
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run 运行Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("WebSocket客户端连接: %s, 当前连接数: %d", client.id, len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mutex.Unlock()
			log.Printf("WebSocket客户端断开: %s, 当前连接数: %d", client.id, len(h.clients))

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					delete(h.clients, client)
					close(client.send)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastToFiltered 向符合过滤条件的客户端广播消息
func (h *Hub) BroadcastToFiltered(device IoTDevice) {
	message, err := json.Marshal(device)
	if err != nil {
		log.Printf("序列化设备数据失败: %v", err)
		return
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		// 检查客户端过滤器
		if len(client.filter) > 0 {
			if !client.filter[device.DeviceID] && !client.filter[device.Location] {
				continue
			}
		}

		select {
		case client.send <- message:
		default:
			delete(h.clients, client)
			close(client.send)
		}
	}
}

// readPump 读取WebSocket消息
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket错误: %v", err)
			}
			break
		}

		// 处理客户端发送的过滤器设置
		var filterMsg map[string]interface{}
		if err := json.Unmarshal(message, &filterMsg); err == nil {
			if devices, ok := filterMsg["devices"].([]interface{}); ok {
				c.filter = make(map[string]bool)
				for _, device := range devices {
					if deviceStr, ok := device.(string); ok {
						c.filter[deviceStr] = true
					}
				}
				log.Printf("客户端 %s 设置过滤器: %v", c.id, c.filter)
			}
		}
	}
}

// writePump 写入WebSocket消息
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量发送队列中的其他消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// WebSocketServer WebSocket服务器
type WebSocketServer struct {
	hub      *Hub
	upgrader websocket.Upgrader
}

// NewWebSocketServer 创建新的WebSocket服务器
func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		hub: NewHub(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 在生产环境中应该进行适当的Origin检查
			},
		},
	}
}

// ServeWS 处理WebSocket连接
func (ws *WebSocketServer) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}

	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		hub:    ws.hub,
		id:     clientID,
		filter: make(map[string]bool),
	}

	client.hub.register <- client

	// 启动读写协程
	go client.writePump()
	go client.readPump()
}

// KafkaConsumer Kafka消费者
type KafkaConsumer struct {
	hub   *Hub
	ready chan bool
}

// NewKafkaConsumer 创建新的Kafka消费者
func NewKafkaConsumer(hub *Hub) *KafkaConsumer {
	return &KafkaConsumer{
		hub:   hub,
		ready: make(chan bool),
	}
}

// Setup 消费者组设置
func (kc *KafkaConsumer) Setup(sarama.ConsumerGroupSession) error {
	close(kc.ready)
	return nil
}

// Cleanup 消费者组清理
func (kc *KafkaConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 消费消息
func (kc *KafkaConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			var device IoTDevice
			if err := json.Unmarshal(message.Value, &device); err != nil {
				log.Printf("反序列化消息失败: %v", err)
				continue
			}

			// 广播到WebSocket客户端
			kc.hub.BroadcastToFiltered(device)

			// 标记消息已处理
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

// runWebSocketServer 运行WebSocket服务器
func runWebSocketServer(cmd *cobra.Command, args []string) error {
	// 读取配置
	port := viper.GetString("websocket.port")
	brokers := viper.GetStringSlice("kafka.brokers")
	topics := viper.GetStringSlice("kafka.topics")
	groupID := viper.GetString("consumer.group_id")

	log.Printf("启动WebSocket实时通信服务...")
	log.Printf("服务端口: %s", port)
	log.Printf("Kafka代理: %v", brokers)
	log.Printf("订阅主题: %v", topics)

	// 创建WebSocket服务器
	wsServer := NewWebSocketServer()

	// 启动Hub
	go wsServer.hub.Run()

	// 创建Kafka消费者
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Group.Session.Timeout = 10 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 3 * time.Second

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return fmt.Errorf("创建消费者组失败: %w", err)
	}
	defer consumerGroup.Close()

	kafkaConsumer := NewKafkaConsumer(wsServer.hub)

	// 启动Kafka消费者
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			if err := consumerGroup.Consume(ctx, topics, kafkaConsumer); err != nil {
				log.Printf("Kafka消费者错误: %v", err)
			}

			if ctx.Err() != nil {
				return
			}

			kafkaConsumer.ready = make(chan bool)
		}
	}()

	// 等待Kafka消费者准备就绪
	<-kafkaConsumer.ready
	log.Printf("Kafka消费者已启动")

	// 设置HTTP路由
	http.HandleFunc("/ws", wsServer.ServeWS)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// 启动HTTP服务器
	go func() {
		log.Printf("WebSocket服务器启动在端口 %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP服务器启动失败: %v", err)
		}
	}()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("WebSocket实时通信服务已启动，按 Ctrl+C 停止")

	// 等待停止信号
	<-sigChan
	log.Printf("收到停止信号，正在关闭WebSocket服务...")

	// 优雅关闭
	cancel()
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP服务器关闭失败: %v", err)
	}

	return nil
}

// initConfig 初始化配置
func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("websocket.port", "8080")
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.topics", []string{"iot-sensor-data"})
	viper.SetDefault("consumer.group_id", "websocket-consumer-group")

	// 支持环境变量
	viper.SetEnvPrefix("IOT")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("配置文件读取失败，使用默认配置: %v", err)
	}
}

func main() {
	// 初始化配置
	initConfig()

	// 创建根命令
	var rootCmd = &cobra.Command{
		Use:   "websocket",
		Short: "WebSocket实时通信服务",
		Long: `工业IoT监控系统WebSocket实时通信服务
		
该服务提供WebSocket连接，将Kafka中的IoT数据实时推送到前端客户端。
支持客户端过滤和高并发连接管理。`,
		Version: fmt.Sprintf("%s (构建时间: %s, Go版本: %s)", Version, BuildTime, GoVersion),
		RunE:    runWebSocketServer,
	}

	// 添加命令行参数
	rootCmd.Flags().StringP("port", "p", "8080", "WebSocket服务端口")
	rootCmd.Flags().StringSliceP("brokers", "b", []string{"localhost:9092"}, "Kafka代理地址列表")
	rootCmd.Flags().StringSliceP("topics", "t", []string{"iot-sensor-data"}, "订阅的Kafka主题列表")
	rootCmd.Flags().StringP("group", "g", "websocket-consumer-group", "消费者组ID")

	// 绑定命令行参数到配置
	viper.BindPFlag("websocket.port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("kafka.brokers", rootCmd.Flags().Lookup("brokers"))
	viper.BindPFlag("kafka.topics", rootCmd.Flags().Lookup("topics"))
	viper.BindPFlag("consumer.group_id", rootCmd.Flags().Lookup("group"))

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("程序执行失败: %v", err)
	}
}
