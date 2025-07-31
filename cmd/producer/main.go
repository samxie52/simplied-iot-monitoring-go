package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
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
	DeviceID    string    `json:"device_id"`
	DeviceType  string    `json:"device_type"`
	Location    string    `json:"location"`
	Timestamp   time.Time `json:"timestamp"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	Pressure    float64   `json:"pressure"`
	Status      string    `json:"status"`
	BatteryLevel float64  `json:"battery_level"`
}

// Producer Kafka生产者配置
type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

// NewProducer 创建新的生产者实例
func NewProducer(brokers []string, topic string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true
	config.Producer.Compression = sarama.CompressionSnappy

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("创建Kafka生产者失败: %w", err)
	}

	return &Producer{
		producer: producer,
		topic:    topic,
	}, nil
}

// SendMessage 发送消息到Kafka
func (p *Producer) SendMessage(device IoTDevice) error {
	data, err := json.Marshal(device)
	if err != nil {
		return fmt.Errorf("序列化设备数据失败: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(device.DeviceID),
		Value: sarama.ByteEncoder(data),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}

	log.Printf("消息发送成功 - 分区: %d, 偏移量: %d, 设备: %s", 
		partition, offset, device.DeviceID)
	return nil
}

// Close 关闭生产者
func (p *Producer) Close() error {
	return p.producer.Close()
}

// generateDeviceData 生成模拟设备数据
func generateDeviceData(deviceID string) IoTDevice {
	deviceTypes := []string{"温度传感器", "湿度传感器", "压力传感器", "综合传感器"}
	locations := []string{"车间A", "车间B", "车间C", "仓库1", "仓库2", "办公区"}
	statuses := []string{"正常", "警告", "故障"}

	return IoTDevice{
		DeviceID:     deviceID,
		DeviceType:   deviceTypes[rand.Intn(len(deviceTypes))],
		Location:     locations[rand.Intn(len(locations))],
		Timestamp:    time.Now(),
		Temperature:  20.0 + rand.Float64()*30.0, // 20-50°C
		Humidity:     30.0 + rand.Float64()*40.0, // 30-70%
		Pressure:     1000.0 + rand.Float64()*100.0, // 1000-1100 hPa
		Status:       statuses[rand.Intn(len(statuses))],
		BatteryLevel: rand.Float64() * 100.0, // 0-100%
	}
}

// runProducer 运行生产者服务
func runProducer(cmd *cobra.Command, args []string) error {
	// 读取配置
	brokers := viper.GetStringSlice("kafka.brokers")
	topic := viper.GetString("kafka.topic")
	deviceCount := viper.GetInt("producer.device_count")
	interval := viper.GetDuration("producer.interval")

	log.Printf("启动IoT数据生产者服务...")
	log.Printf("Kafka代理: %v", brokers)
	log.Printf("主题: %s", topic)
	log.Printf("设备数量: %d", deviceCount)
	log.Printf("发送间隔: %v", interval)

	// 创建生产者
	producer, err := NewProducer(brokers, topic)
	if err != nil {
		return fmt.Errorf("创建生产者失败: %w", err)
	}
	defer producer.Close()

	// 创建上下文和信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动数据生成协程
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 为每个设备生成并发送数据
				for i := 0; i < deviceCount; i++ {
					deviceID := fmt.Sprintf("device_%03d", i+1)
					device := generateDeviceData(deviceID)
					
					if err := producer.SendMessage(device); err != nil {
						log.Printf("发送设备数据失败 [%s]: %v", deviceID, err)
					}
				}
			}
		}
	}()

	log.Printf("IoT数据生产者服务已启动，按 Ctrl+C 停止")

	// 等待停止信号
	<-sigChan
	log.Printf("收到停止信号，正在关闭生产者服务...")
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
	viper.SetDefault("kafka.topic", "iot-sensor-data")
	viper.SetDefault("producer.device_count", 100)
	viper.SetDefault("producer.interval", "1s")

	// 支持环境变量
	viper.SetEnvPrefix("IOT")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("配置文件读取失败，使用默认配置: %v", err)
	}
}

func main() {
	// 初始化随机种子
	rand.Seed(time.Now().UnixNano())

	// 初始化配置
	initConfig()

	// 创建根命令
	var rootCmd = &cobra.Command{
		Use:   "producer",
		Short: "IoT数据生产者服务",
		Long: `工业IoT监控系统数据生产者服务
		
该服务模拟工业设备传感器，生成实时IoT数据并发送到Kafka消息队列。
支持多种传感器类型和配置参数。`,
		Version: fmt.Sprintf("%s (构建时间: %s, Go版本: %s)", Version, BuildTime, GoVersion),
		RunE:    runProducer,
	}

	// 添加命令行参数
	rootCmd.Flags().StringSliceP("brokers", "b", []string{"localhost:9092"}, "Kafka代理地址列表")
	rootCmd.Flags().StringP("topic", "t", "iot-sensor-data", "Kafka主题名称")
	rootCmd.Flags().IntP("devices", "d", 100, "模拟设备数量")
	rootCmd.Flags().DurationP("interval", "i", time.Second, "数据发送间隔")

	// 绑定命令行参数到配置
	viper.BindPFlag("kafka.brokers", rootCmd.Flags().Lookup("brokers"))
	viper.BindPFlag("kafka.topic", rootCmd.Flags().Lookup("topic"))
	viper.BindPFlag("producer.device_count", rootCmd.Flags().Lookup("devices"))
	viper.BindPFlag("producer.interval", rootCmd.Flags().Lookup("interval"))

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("程序执行失败: %v", err)
	}
}
