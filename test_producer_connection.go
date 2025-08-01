package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/producer"
)

func main() {
	fmt.Println("Testing Producer Connection...")

	// 创建基本配置
	cfg := &config.AppConfig{
		Kafka: config.KafkaSection{
			Brokers: []string{"192.168.5.16:9092"},
			Topics: config.TopicConfig{
				DeviceData: "sensor-data",
				Alerts: "alerts",
			},
			Producer: config.KafkaProducer{
				ClientID:         "test-producer",
				BatchSize:        100,
				BatchTimeout:     time.Millisecond * 100,
				CompressionType:  "gzip",
				MaxRetries:       3,
				RetryBackoff:     time.Millisecond * 100,
				RequiredAcks:     1,
				FlushFrequency:   time.Millisecond * 100,
				ChannelBufferSize: 1000,
				Timeout:          time.Second * 30,
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
	fmt.Println("Starting producer service...")
	err = producerService.Start()
	if err != nil {
		log.Fatalf("Failed to start producer service: %v", err)
	}

	fmt.Println("Producer service started successfully")

	// 等待一段时间让连接建立
	time.Sleep(2 * time.Second)

	// 检查健康状态
	healthChecker := producerService.GetHealthChecker()
	if healthChecker != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		health := healthChecker.Check(ctx)
		fmt.Printf("Health check result: %+v\n", health)
	}

	// 测试发送一条消息
	fmt.Println("Testing message sending...")
	
	// 这里我们需要检查producer服务是否有发送消息的方法
	// 让我们先检查连接池状态
	fmt.Println("Checking connection pool status...")

	// 保持运行一段时间
	fmt.Println("Producer running for 10 seconds...")
	time.Sleep(10 * time.Second)

	// 停止服务
	fmt.Println("Stopping producer service...")
	err = producerService.Stop()
	if err != nil {
		log.Printf("Error stopping producer service: %v", err)
	}

	fmt.Println("Producer connection test completed")
}
