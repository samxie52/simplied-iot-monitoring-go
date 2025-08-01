package main

import (
	"log"
	"time"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/producer"
)

func main() {
	log.Println("开始调试Producer服务启动...")

	// 加载配置
	manager := config.NewConfigManager()
	if err := manager.Load("configs/development.yaml", config.Development); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	cfg := manager.GetConfig()

	log.Printf("配置加载成功: Kafka Brokers=%v", cfg.Kafka.Brokers)

	// 创建生产者服务
	log.Println("创建ProducerService...")
	service, err := producer.NewProducerService(cfg)
	if err != nil {
		log.Fatalf("创建生产者服务失败: %v", err)
	}
	log.Println("ProducerService创建成功")

	// 启动服务，设置超时
	log.Println("开始启动服务...")
	done := make(chan error, 1)
	go func() {
		done <- service.Start()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Fatalf("启动服务失败: %v", err)
		}
		log.Println("服务启动成功!")
	case <-time.After(30 * time.Second):
		log.Println("服务启动超时!")
		return
	}

	// 测试健康检查
	log.Println("测试健康检查...")
	healthChecker := service.GetHealthChecker()
	if healthChecker != nil {
		log.Println("健康检查器可用")
	}

	// 停止服务
	log.Println("停止服务...")
	if err := service.Stop(); err != nil {
		log.Printf("停止服务失败: %v", err)
	} else {
		log.Println("服务停止成功")
	}
}
