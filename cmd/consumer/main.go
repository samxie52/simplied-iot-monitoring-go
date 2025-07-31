package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
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

// Consumer Kafka消费者
type Consumer struct {
	ready chan bool
}

// NewConsumer 创建新的消费者实例
func NewConsumer() *Consumer {
	return &Consumer{
		ready: make(chan bool),
	}
}

// Setup 消费者组设置
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	close(consumer.ready)
	return nil
}

// Cleanup 消费者组清理
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 消费消息
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			// 处理消息
			if err := consumer.processMessage(message); err != nil {
				log.Printf("处理消息失败: %v", err)
				continue
			}

			// 标记消息已处理
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

// processMessage 处理单条消息
func (consumer *Consumer) processMessage(message *sarama.ConsumerMessage) error {
	var device IoTDevice
	if err := json.Unmarshal(message.Value, &device); err != nil {
		return fmt.Errorf("反序列化消息失败: %w", err)
	}

	// 打印接收到的数据
	log.Printf("收到设备数据 - 设备ID: %s, 类型: %s, 位置: %s, 温度: %.2f°C, 湿度: %.2f%%, 状态: %s",
		device.DeviceID, device.DeviceType, device.Location, 
		device.Temperature, device.Humidity, device.Status)

	// 数据验证和处理逻辑
	if err := consumer.validateDevice(device); err != nil {
		log.Printf("设备数据验证失败 [%s]: %v", device.DeviceID, err)
		return nil // 不返回错误，继续处理下一条消息
	}

	// 异常检测
	if alerts := consumer.detectAnomalies(device); len(alerts) > 0 {
		log.Printf("设备异常告警 [%s]: %s", device.DeviceID, strings.Join(alerts, ", "))
	}

	// 这里可以添加数据存储逻辑
	// 例如：存储到数据库、发送到时序数据库等
	if err := consumer.storeDevice(device); err != nil {
		log.Printf("存储设备数据失败 [%s]: %v", device.DeviceID, err)
	}

	return nil
}

// validateDevice 验证设备数据
func (consumer *Consumer) validateDevice(device IoTDevice) error {
	if device.DeviceID == "" {
		return fmt.Errorf("设备ID不能为空")
	}

	if device.Temperature < -50 || device.Temperature > 100 {
		return fmt.Errorf("温度值异常: %.2f°C", device.Temperature)
	}

	if device.Humidity < 0 || device.Humidity > 100 {
		return fmt.Errorf("湿度值异常: %.2f%%", device.Humidity)
	}

	if device.BatteryLevel < 0 || device.BatteryLevel > 100 {
		return fmt.Errorf("电池电量异常: %.2f%%", device.BatteryLevel)
	}

	return nil
}

// detectAnomalies 异常检测
func (consumer *Consumer) detectAnomalies(device IoTDevice) []string {
	var alerts []string

	// 温度异常检测
	if device.Temperature > 45 {
		alerts = append(alerts, fmt.Sprintf("高温告警: %.2f°C", device.Temperature))
	} else if device.Temperature < 5 {
		alerts = append(alerts, fmt.Sprintf("低温告警: %.2f°C", device.Temperature))
	}

	// 湿度异常检测
	if device.Humidity > 80 {
		alerts = append(alerts, fmt.Sprintf("高湿度告警: %.2f%%", device.Humidity))
	} else if device.Humidity < 20 {
		alerts = append(alerts, fmt.Sprintf("低湿度告警: %.2f%%", device.Humidity))
	}

	// 电池电量检测
	if device.BatteryLevel < 20 {
		alerts = append(alerts, fmt.Sprintf("低电量告警: %.2f%%", device.BatteryLevel))
	}

	// 设备状态检测
	if device.Status == "故障" {
		alerts = append(alerts, "设备故障告警")
	}

	return alerts
}

// storeDevice 存储设备数据（模拟）
func (consumer *Consumer) storeDevice(device IoTDevice) error {
	// 这里应该实现实际的数据存储逻辑
	// 例如：存储到PostgreSQL、InfluxDB、MongoDB等
	
	// 模拟存储延迟
	time.Sleep(1 * time.Millisecond)
	
	log.Printf("设备数据已存储 [%s] - 时间戳: %s", 
		device.DeviceID, device.Timestamp.Format("2006-01-02 15:04:05"))
	
	return nil
}

// runConsumer 运行消费者服务
func runConsumer(cmd *cobra.Command, args []string) error {
	// 读取配置
	brokers := viper.GetStringSlice("kafka.brokers")
	topics := viper.GetStringSlice("kafka.topics")
	groupID := viper.GetString("consumer.group_id")

	log.Printf("启动IoT数据消费者服务...")
	log.Printf("Kafka代理: %v", brokers)
	log.Printf("订阅主题: %v", topics)
	log.Printf("消费者组: %s", groupID)

	// 创建消费者配置
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Group.Session.Timeout = 10 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 3 * time.Second

	// 创建消费者组
	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return fmt.Errorf("创建消费者组失败: %w", err)
	}
	defer consumerGroup.Close()

	// 创建消费者实例
	consumer := NewConsumer()

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动消费者协程
	go func() {
		for {
			if err := consumerGroup.Consume(ctx, topics, consumer); err != nil {
				log.Printf("消费者错误: %v", err)
			}

			if ctx.Err() != nil {
				return
			}

			consumer.ready = make(chan bool)
		}
	}()

	// 等待消费者准备就绪
	<-consumer.ready
	log.Printf("IoT数据消费者服务已启动，按 Ctrl+C 停止")

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待停止信号
	<-sigChan
	log.Printf("收到停止信号，正在关闭消费者服务...")
	cancel()

	return nil
}

// initConfig 初始化配置
func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.topics", []string{"iot-sensor-data"})
	viper.SetDefault("consumer.group_id", "iot-consumer-group")

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
		Use:   "consumer",
		Short: "IoT数据消费者服务",
		Long: `工业IoT监控系统数据消费者服务
		
该服务从Kafka消息队列消费IoT设备数据，进行数据验证、异常检测和存储处理。
支持高并发处理和容错机制。`,
		Version: fmt.Sprintf("%s (构建时间: %s, Go版本: %s)", Version, BuildTime, GoVersion),
		RunE:    runConsumer,
	}

	// 添加命令行参数
	rootCmd.Flags().StringSliceP("brokers", "b", []string{"localhost:9092"}, "Kafka代理地址列表")
	rootCmd.Flags().StringSliceP("topics", "t", []string{"iot-sensor-data"}, "订阅的Kafka主题列表")
	rootCmd.Flags().StringP("group", "g", "iot-consumer-group", "消费者组ID")

	// 绑定命令行参数到配置
	viper.BindPFlag("kafka.brokers", rootCmd.Flags().Lookup("brokers"))
	viper.BindPFlag("kafka.topics", rootCmd.Flags().Lookup("topics"))
	viper.BindPFlag("consumer.group_id", rootCmd.Flags().Lookup("group"))

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("程序执行失败: %v", err)
	}
}
