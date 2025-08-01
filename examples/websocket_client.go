package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// ClientFilter 客户端过滤器
type ClientFilter struct {
	DeviceIDs   []string `json:"device_ids,omitempty"`
	DeviceTypes []string `json:"device_types,omitempty"`
	Locations   []string `json:"locations,omitempty"`
	AlertLevels []string `json:"alert_levels,omitempty"`
}

func main() {
	var addr = flag.String("addr", "localhost:8081", "WebSocket服务器地址")
	var path = flag.String("path", "/ws", "WebSocket路径")
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: *path}
	log.Printf("连接到 %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	// 启动消息读取协程
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("读取消息失败:", err)
				return
			}
			
			// 解析消息
			var wsMsg WebSocketMessage
			if err := json.Unmarshal(message, &wsMsg); err != nil {
				log.Printf("解析消息失败: %v", err)
				continue
			}
			
			// 处理不同类型的消息
			switch wsMsg.Type {
			case "device_data":
				handleDeviceData(&wsMsg)
			case "alert":
				handleAlert(&wsMsg)
			case "pong":
				log.Printf("收到pong响应: %s", wsMsg.Timestamp.Format("15:04:05"))
			default:
				log.Printf("未知消息类型: %s", wsMsg.Type)
			}
		}
	}()

	// 发送过滤器设置
	filter := map[string]interface{}{
		"type": "filter",
		"data": ClientFilter{
			DeviceIDs:   []string{"device-001", "device-002"},
			DeviceTypes: []string{"sensor"},
			Locations:   []string{"Building-A"},
			AlertLevels: []string{"warning", "critical"},
		},
	}
	
	if err := c.WriteJSON(filter); err != nil {
		log.Println("发送过滤器失败:", err)
		return
	}
	log.Println("已发送过滤器设置")

	// 定期发送ping消息
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			ping := map[string]interface{}{
				"type": "ping",
			}
			if err := c.WriteJSON(ping); err != nil {
				log.Println("发送ping失败:", err)
				return
			}
		case <-interrupt:
			log.Println("收到中断信号")

			// 优雅关闭WebSocket连接
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("写入关闭消息失败:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

// handleDeviceData 处理设备数据消息
func handleDeviceData(msg *WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		log.Printf("无效的设备数据格式")
		return
	}
	
	device, ok := data["device"].(map[string]interface{})
	if !ok {
		log.Printf("无效的设备信息格式")
		return
	}
	
	deviceID, _ := device["device_id"].(string)
	deviceType, _ := device["device_type"].(string)
	status, _ := device["status"].(string)
	
	// 提取传感器数据
	sensors, ok := device["sensors"].(map[string]interface{})
	if ok {
		var sensorInfo []string
		
		if temp, ok := sensors["temperature"].(map[string]interface{}); ok {
			if value, ok := temp["value"].(float64); ok {
				if unit, ok := temp["unit"].(string); ok {
					sensorInfo = append(sensorInfo, fmt.Sprintf("温度: %.1f%s", value, unit))
				}
			}
		}
		
		if humidity, ok := sensors["humidity"].(map[string]interface{}); ok {
			if value, ok := humidity["value"].(float64); ok {
				if unit, ok := humidity["unit"].(string); ok {
					sensorInfo = append(sensorInfo, fmt.Sprintf("湿度: %.1f%s", value, unit))
				}
			}
		}
		
		if pressure, ok := sensors["pressure"].(map[string]interface{}); ok {
			if value, ok := pressure["value"].(float64); ok {
				if unit, ok := pressure["unit"].(string); ok {
					sensorInfo = append(sensorInfo, fmt.Sprintf("压力: %.1f%s", value, unit))
				}
			}
		}
		
		log.Printf("📊 设备数据 [%s] %s (%s) - %s", 
			msg.Timestamp.Format("15:04:05"), 
			deviceID, 
			deviceType, 
			fmt.Sprintf("状态: %s", status))
		
		for _, info := range sensorInfo {
			log.Printf("   %s", info)
		}
	} else {
		log.Printf("📊 设备数据 [%s] %s (%s) - 状态: %s", 
			msg.Timestamp.Format("15:04:05"), 
			deviceID, 
			deviceType, 
			status)
	}
}

// handleAlert 处理告警消息
func handleAlert(msg *WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		log.Printf("无效的告警数据格式")
		return
	}
	
	alert, ok := data["alert"].(map[string]interface{})
	if !ok {
		log.Printf("无效的告警信息格式")
		return
	}
	
	alertID, _ := alert["alert_id"].(string)
	deviceID, _ := data["device_id"].(string)
	severity, _ := alert["severity"].(string)
	title, _ := alert["title"].(string)
	description, _ := alert["description"].(string)
	
	// 根据严重级别选择不同的图标
	var icon string
	switch severity {
	case "info":
		icon = "ℹ️"
	case "warning":
		icon = "⚠️"
	case "error":
		icon = "❌"
	case "critical":
		icon = "🚨"
	default:
		icon = "📢"
	}
	
	log.Printf("%s 告警 [%s] %s - %s", 
		icon,
		msg.Timestamp.Format("15:04:05"), 
		alertID, 
		severity)
	log.Printf("   设备: %s", deviceID)
	log.Printf("   标题: %s", title)
	if description != "" {
		log.Printf("   描述: %s", description)
	}
}
