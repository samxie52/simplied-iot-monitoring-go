package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

type DeviceData struct {
	DeviceID   string    `json:"device_id"`
	DeviceType string    `json:"device_type"`
	Timestamp  int64     `json:"timestamp"`
	Location   Location  `json:"location"`
	Sensors    Sensors   `json:"sensors"`
	Status     string    `json:"status"`
}

type Location struct {
	Building string `json:"building"`
	Floor    int    `json:"floor"`
	Room     string `json:"room"`
}

type Sensors struct {
	Temperature *SensorValue `json:"temperature,omitempty"`
	Humidity    *SensorValue `json:"humidity,omitempty"`
	Pressure    *SensorValue `json:"pressure,omitempty"`
	Switch      *SwitchValue `json:"switch_status,omitempty"`
	Current     *SensorValue `json:"current,omitempty"`
}

type SensorValue struct {
	Value  float64 `json:"value"`
	Unit   string  `json:"unit"`
	Status string  `json:"status"`
}

type SwitchValue struct {
	Value  bool   `json:"value"`
	Status string `json:"status"`
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	
	fmt.Println("演示WebSocket服务器启动在端口 8080")
	fmt.Println("WebSocket地址: ws://localhost:8080/ws")
	
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("收到WebSocket升级请求: %s %s", r.Method, r.URL.Path)
	log.Printf("请求头: Origin=%s, Upgrade=%s, Connection=%s", 
		r.Header.Get("Origin"), r.Header.Get("Upgrade"), r.Header.Get("Connection"))
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()
	
	log.Printf("新的WebSocket连接成功: %s", r.RemoteAddr)
	
	// 启动数据发送协程
	go sendDemoData(conn)
	
	// 处理客户端消息
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("读取消息失败: %v", err)
			break
		}
		
		if messageType == websocket.TextMessage {
			log.Printf("收到消息: %s", string(message))
			
			// 处理ping消息
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err == nil {
				if msg["type"] == "ping" {
					pong := map[string]interface{}{
						"type":      "pong",
						"timestamp": msg["timestamp"],
					}
					if data, err := json.Marshal(pong); err == nil {
						conn.WriteMessage(websocket.TextMessage, data)
					}
				}
			}
		}
	}
	
	log.Printf("WebSocket连接关闭: %s", r.RemoteAddr)
}

func sendDemoData(conn *websocket.Conn) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	deviceTypes := []string{"temperature", "humidity", "pressure", "switch", "current"}
	buildings := []string{"Building_1", "Building_2"}
	
	for {
		select {
		case <-ticker.C:
			// 生成随机设备数据
			deviceType := deviceTypes[rand.Intn(len(deviceTypes))]
			deviceID := fmt.Sprintf("device_%04d", rand.Intn(100))
			
			data := DeviceData{
				DeviceID:   deviceID,
				DeviceType: deviceType,
				Timestamp:  time.Now().UnixMilli(),
				Location: Location{
					Building: buildings[rand.Intn(len(buildings))],
					Floor:    rand.Intn(3) + 1,
					Room:     fmt.Sprintf("Room_%d", rand.Intn(10)+1),
				},
				Status: "online",
			}
			
			// 根据设备类型生成传感器数据
			switch deviceType {
			case "temperature":
				data.Sensors.Temperature = &SensorValue{
					Value:  20 + rand.Float64()*15, // 20-35°C
					Unit:   "°C",
					Status: "normal",
				}
			case "humidity":
				data.Sensors.Humidity = &SensorValue{
					Value:  30 + rand.Float64()*40, // 30-70%
					Unit:   "%",
					Status: "normal",
				}
			case "pressure":
				data.Sensors.Pressure = &SensorValue{
					Value:  1000 + rand.Float64()*50, // 1000-1050 hPa
					Unit:   "hPa",
					Status: "normal",
				}
			case "switch":
				data.Sensors.Switch = &SwitchValue{
					Value:  rand.Float64() > 0.5,
					Status: "normal",
				}
			case "current":
				data.Sensors.Current = &SensorValue{
					Value:  rand.Float64() * 10, // 0-10A
					Unit:   "A",
					Status: "normal",
				}
			}
			
			// 发送设备数据消息
			message := Message{
				Type: "device_data",
				Data: data,
			}
			
			jsonData, err := json.Marshal(message)
			if err != nil {
				log.Printf("JSON序列化失败: %v", err)
				continue
			}
			
			if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
				log.Printf("发送消息失败: %v", err)
				return
			}
			log.Printf("发送设备数据: %s, 类型: %s", deviceID, deviceType)
			
			// 偶尔发送告警消息
			if rand.Float64() < 0.1 { // 10%概率
				alert := map[string]interface{}{
					"alert_id":  fmt.Sprintf("alert_%d", time.Now().Unix()),
					"device_id": deviceID,
					"severity":  []string{"info", "warning", "critical"}[rand.Intn(3)],
					"message":   "设备状态异常",
					"timestamp": time.Now().UnixMilli(),
				}
				
				alertMessage := Message{
					Type: "alert",
					Data: alert,
				}
				
				if alertData, err := json.Marshal(alertMessage); err == nil {
					conn.WriteMessage(websocket.TextMessage, alertData)
				}
			}
		}
	}
}
