package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/producer"
)

// 模拟传感器数据
type SensorData struct {
	DeviceID    string    `json:"device_id"`
	Timestamp   time.Time `json:"timestamp"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	Battery     float64   `json:"battery"`
}

func main() {
	fmt.Println("Testing Producer Message Sending...")

	// 创建配置
	cfg := &config.AppConfig{
		Kafka: config.KafkaSection{
			Brokers: []string{"192.168.5.16:9092"},
			Topics: config.TopicConfig{
				DeviceData: "sensor-data",
				Alerts:     "alerts",
			},
			Producer: config.KafkaProducer{
				ClientID:          "test-producer-send",
				BatchSize:         10,
				BatchTimeout:      time.Millisecond * 100,
				CompressionType:   "gzip",
				MaxRetries:        3,
				RetryBackoff:      time.Millisecond * 100,
				RequiredAcks:      1,
				FlushFrequency:    time.Millisecond * 100,
				ChannelBufferSize: 1000,
				Timeout:           time.Second * 30,
			},
			Timeout: time.Second * 30,
		},
		Producer: config.ProducerSection{
			// 基本producer配置
		},
		Device: config.DeviceSection{
			// 基本设备配置  
		},
	}

	// 创建producer服务
	producerService, err := producer.NewProducerService(cfg)
	if err != nil {
		log.Fatalf("Failed to create producer service: %v", err)
	}

	fmt.Println("Producer service created successfully")

	// 启动producer服务
	err = producerService.Start()
	if err != nil {
		log.Fatalf("Failed to start producer service: %v", err)
	}

	fmt.Println("Producer service started successfully")

	// 等待连接建立
	time.Sleep(2 * time.Second)

	// 检查健康状态
	healthChecker := producerService.GetHealthChecker()
	if healthChecker != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		health := healthChecker.Check(ctx)
		fmt.Printf("Health check result: Status=%s, Message=%s\n", health.Status, health.Message)
		fmt.Printf("Metadata: %+v\n", health.Metadata)
	}

	// 创建测试消息
	testMessages := []SensorData{
		{
			DeviceID:    "device-001",
			Timestamp:   time.Now(),
			Temperature: 23.5,
			Humidity:    65.2,
			Battery:     87.3,
		},
		{
			DeviceID:    "device-002", 
			Timestamp:   time.Now(),
			Temperature: 24.1,
			Humidity:    62.8,
			Battery:     91.5,
		},
		{
			DeviceID:    "device-003",
			Timestamp:   time.Now(),
			Temperature: 22.8,
			Humidity:    68.1,
			Battery:     76.2,
		},
	}

	fmt.Printf("Sending %d test messages...\n", len(testMessages))

	// 发送测试消息
	for i, msg := range testMessages {
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal message %d: %v", i+1, err)
			continue
		}

		fmt.Printf("Message %d: %s\n", i+1, string(msgBytes))
		
		// 使用producer服务的SendMessage方法
		err = producerService.SendMessage(msg.DeviceID, msg)
		if err != nil {
			log.Printf("Failed to send message %d: %v", i+1, err)
			continue
		}
		
		fmt.Printf("Message %d sent successfully!\n", i+1)
		
		// 等待一点时间再发送下一条
		time.Sleep(500 * time.Millisecond)
	}

	// 等待消息处理
	fmt.Println("Waiting for message processing...")
	time.Sleep(5 * time.Second)

	// 再次检查健康状态
	if healthChecker != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		health := healthChecker.Check(ctx)
		fmt.Printf("Final health check: Status=%s\n", health.Status)
		fmt.Printf("Final metadata: %+v\n", health.Metadata)
	}

	// 停止服务
	fmt.Println("Stopping producer service...")
	err = producerService.Stop()
	if err != nil {
		log.Printf("Error stopping producer service: %v", err)
	}

	fmt.Println("Producer send test completed")
}
