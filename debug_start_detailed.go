package main

import (
	"log"
	"time"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/producer"
)

func main() {
	log.Println("开始详细调试Producer服务启动...")

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

	// 逐步启动各个组件
	log.Println("开始逐步启动各个组件...")

	// 1. 测试连接池启动
	log.Println("1. 启动连接池...")
	connectionPool := service.GetConnectionPool()
	if connectionPool != nil {
		done := make(chan error, 1)
		go func() {
			done <- connectionPool.Start()
		}()
		
		select {
		case err := <-done:
			if err != nil {
				log.Printf("连接池启动失败: %v", err)
			} else {
				log.Println("连接池启动成功!")
			}
		case <-time.After(15 * time.Second):
			log.Println("连接池启动超时!")
		}
	}

	// 2. 测试Kafka生产者启动
	log.Println("2. 启动Kafka生产者...")
	kafkaProducer := service.GetKafkaProducer()
	if kafkaProducer != nil {
		done := make(chan error, 1)
		go func() {
			done <- kafkaProducer.Start()
		}()
		
		select {
		case err := <-done:
			if err != nil {
				log.Printf("Kafka生产者启动失败: %v", err)
			} else {
				log.Println("Kafka生产者启动成功!")
			}
		case <-time.After(15 * time.Second):
			log.Println("Kafka生产者启动超时!")
		}
	}

	// 3. 测试设备模拟器启动
	log.Println("3. 启动设备模拟器...")
	deviceSimulator := service.GetDeviceSimulator()
	if deviceSimulator != nil && cfg.Device.Simulator.Enabled {
		done := make(chan error, 1)
		go func() {
			done <- deviceSimulator.Start()
		}()
		
		select {
		case err := <-done:
			if err != nil {
				log.Printf("设备模拟器启动失败: %v", err)
			} else {
				log.Println("设备模拟器启动成功!")
			}
		case <-time.After(10 * time.Second):
			log.Println("设备模拟器启动超时!")
		}
	} else {
		log.Println("设备模拟器未启用或不存在")
	}

	// 4. 测试健康检查器启动
	log.Println("4. 跳过健康检查器启动测试（接口不匹配）")
	healthChecker := service.GetHealthChecker()
	if healthChecker != nil {
		log.Println("健康检查器存在")
	} else {
		log.Println("健康检查器不存在")
	}

	log.Println("组件启动测试完成")
}
