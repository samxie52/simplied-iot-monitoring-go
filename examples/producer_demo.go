package main

import (
	"fmt"
	"log"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/producer"
)

func main() {
	fmt.Println("=== Industrial IoT Kafka Producer Demo ===")

	// 加载默认配置
	cfg := config.DefaultDevelopmentConfig()
	
	// 禁用设备模拟器，避免Kafka连接错误
	cfg.Device.Simulator.Enabled = false
	
	fmt.Printf("配置加载成功:\n")
	fmt.Printf("  - Kafka Brokers: %v\n", cfg.Kafka.Brokers)
	fmt.Printf("  - Device Data Topic: %s\n", cfg.Kafka.Topics.DeviceData)
	fmt.Printf("  - Alerts Topic: %s\n", cfg.Kafka.Topics.Alerts)
	fmt.Printf("  - Device Simulator Enabled: %v\n", cfg.Device.Simulator.Enabled)

	// 创建生产者服务
	fmt.Println("\n创建生产者服务...")
	service, err := producer.NewProducerService(cfg)
	if err != nil {
		log.Printf("创建生产者服务失败: %v", err)
		fmt.Println("这是预期的，因为没有运行Kafka服务器")
		return
	}

	fmt.Println("生产者服务创建成功!")

	// 获取组件
	kafkaProducer := service.GetKafkaProducer()
	deviceManager := service.GetDeviceManager()
	deviceSimulator := service.GetDeviceSimulator()
	connectionPool := service.GetConnectionPool()
	healthChecker := service.GetHealthChecker()

	fmt.Printf("\n组件状态:\n")
	fmt.Printf("  - Kafka Producer: %v\n", kafkaProducer != nil)
	fmt.Printf("  - Device Manager: %v\n", deviceManager != nil)
	fmt.Printf("  - Device Simulator: %v\n", deviceSimulator != nil)
	fmt.Printf("  - Connection Pool: %v\n", connectionPool != nil)
	fmt.Printf("  - Health Checker: %v\n", healthChecker != nil)

	// 获取指标
	fmt.Printf("\n服务指标:\n")
	metrics := service.GetMetrics()
	fmt.Printf("  - 设备总数: %d\n", metrics.DevicesTotal)
	fmt.Printf("  - 活跃设备: %d\n", metrics.DevicesActive)
	fmt.Printf("  - 消息总数: %d\n", metrics.TotalMessages)
	fmt.Printf("  - 错误总数: %d\n", metrics.TotalErrors)
	fmt.Printf("  - 连接总数: %d\n", metrics.ConnectionsTotal)
	fmt.Printf("  - 活跃连接: %d\n", metrics.ConnectionsActive)

	// 获取健康状态
	fmt.Printf("\n健康状态:\n")
	healthStatus := service.GetHealthStatus()
	fmt.Printf("  - 整体健康: %v\n", healthStatus.Healthy)
	fmt.Printf("  - 检查时间: %v\n", healthStatus.Timestamp.Format("2006-01-02 15:04:05"))
	
	if len(healthStatus.Services) > 0 {
		fmt.Printf("  - 服务状态:\n")
		for name, status := range healthStatus.Services {
			statusStr := "❌"
			if status {
				statusStr = "✅"
			}
			fmt.Printf("    %s %s\n", statusStr, name)
		}
	}

	if len(healthStatus.Errors) > 0 {
		fmt.Printf("  - 错误信息:\n")
		for _, err := range healthStatus.Errors {
			fmt.Printf("    - %s\n", err)
		}
	}

	// 测试设备管理器
	fmt.Printf("\n设备管理器测试:\n")
	deviceCount := deviceManager.GetDeviceCount()
	fmt.Printf("  - 当前设备数量: %d\n", deviceCount)
	
	devices := deviceManager.ListDevices()
	fmt.Printf("  - 设备列表长度: %d\n", len(devices))

	fmt.Println("\n=== Demo 完成 ===")
	fmt.Println("注意: 由于没有运行Kafka服务器，某些功能可能无法完全演示")
	fmt.Println("要完整测试，请启动Kafka服务器并运行: go run ./cmd/producer start")
}
