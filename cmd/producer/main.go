package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/producer"
)

// 版本信息（构建时注入）
var (
	Version   = "dev"
	BuildTime = "unknown"
	GoVersion = "unknown"
)

// 全局变量
var (
	configFile string
	verbose    bool
	service    *producer.ProducerService
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "producer",
	Short: "Industrial IoT Kafka Producer",
	Long: `Industrial IoT Kafka Producer - 工业物联网Kafka生产者服务

该服务负责:
- 模拟工业设备数据生成
- 高性能Kafka消息生产
- 设备状态监控和管理
- 健康检查和指标收集`,
	Version: fmt.Sprintf("%s (built at %s with %s)", Version, BuildTime, GoVersion),
}

// startCmd 启动命令
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动生产者服务",
	Long:  "启动Industrial IoT Kafka生产者服务，开始设备数据模拟和消息生产",
	RunE:  runStart,
}

// statusCmd 状态命令
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看服务状态",
	Long:  "查看生产者服务的运行状态、健康检查和性能指标",
	RunE:  runStatus,
}

// stopCmd 停止命令
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止生产者服务",
	Long:  "优雅停止生产者服务，确保所有消息都已发送完成",
	RunE:  runStop,
}

func init() {
	// 添加全局标志
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细输出")

	// 添加子命令
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(stopCmd)

	// 设置配置
	cobra.OnInitialize(initConfig)
}

// initConfig 初始化配置
func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		// 查找配置文件
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath("$HOME/.iot-producer")
		viper.AddConfigPath("/etc/iot-producer")
	}

	// 环境变量支持
	viper.AutomaticEnv()
	viper.SetEnvPrefix("IOT")

	if err := viper.ReadInConfig(); err != nil {
		if verbose {
			log.Printf("配置文件读取失败: %v", err)
			log.Println("将使用默认配置")
		}
	} else if verbose {
		log.Printf("使用配置文件: %s", viper.ConfigFileUsed())
	}
}

// runStart 运行启动命令
func runStart(cmd *cobra.Command, args []string) error {
	log.Println("启动Industrial IoT Kafka生产者服务...")

	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	if verbose {
		log.Printf("配置加载成功: Kafka Brokers=%v, Topic=%s",
			cfg.Kafka.Brokers, cfg.Kafka.Topics.DeviceData)
	}

	// 创建生产者服务
	service, err = producer.NewProducerService(cfg)
	if err != nil {
		return fmt.Errorf("创建生产者服务失败: %w", err)
	}

	// 启动服务
	if err := service.Start(); err != nil {
		return fmt.Errorf("启动服务失败: %w", err)
	}

	log.Println("生产者服务启动成功!")

	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动状态监控
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				printServiceStatus()
			case <-ctx.Done():
				return
			}
		}
	}()

	// 等待信号
	select {
	case sig := <-sigChan:
		log.Printf("收到信号 %v，开始优雅关闭...", sig)
	case <-ctx.Done():
		log.Println("上下文取消，开始关闭...")
	}

	// 停止服务
	cancel()
	if err := service.Stop(); err != nil {
		log.Printf("停止服务时出错: %v", err)
		return err
	}

	log.Println("生产者服务已停止")
	return nil
}

// runStatus 运行状态命令
func runStatus(cmd *cobra.Command, args []string) error {
	if service == nil || !service.IsRunning() {
		fmt.Println("生产者服务未运行")
		return nil
	}

	printServiceStatus()
	return nil
}

// runStop 运行停止命令
func runStop(cmd *cobra.Command, args []string) error {
	if service == nil || !service.IsRunning() {
		fmt.Println("生产者服务未运行")
		return nil
	}

	log.Println("停止生产者服务...")
	if err := service.Stop(); err != nil {
		return fmt.Errorf("停止服务失败: %w", err)
	}

	log.Println("生产者服务已停止")
	return nil
}

// loadConfig 加载配置
func loadConfig() (*config.AppConfig, error) {
	// 尝试从配置文件加载
	if configFile != "" {
		if _, err := os.Stat(configFile); err == nil {
			manager := config.NewConfigManager()
			if err := manager.Load(configFile, config.Development); err != nil {
				return nil, fmt.Errorf("加载配置文件失败: %w", err)
			}
			return manager.GetConfig(), nil
		}
	}

	// 使用默认配置
	return config.DefaultDevelopmentConfig(), nil
}

// printServiceStatus 打印服务状态
func printServiceStatus() {
	if service == nil {
		fmt.Println("服务未初始化")
		return
	}

	metrics := service.GetMetrics()
	healthStatus := service.GetHealthStatus()

	fmt.Println("\n=== 生产者服务状态 ===")
	fmt.Printf("运行状态: %v\n", service.IsRunning())
	fmt.Printf("健康状态: %v\n", healthStatus.Healthy)
	fmt.Printf("运行时间: %v\n", metrics.Uptime.Round(time.Second))
	fmt.Printf("活跃设备: %d/%d\n", metrics.DevicesActive, metrics.DevicesTotal)
	fmt.Printf("消息总数: %d\n", metrics.TotalMessages)
	fmt.Printf("错误总数: %d\n", metrics.TotalErrors)
	fmt.Printf("连接状态: %d/%d\n", metrics.ConnectionsActive, metrics.ConnectionsTotal)

	if len(healthStatus.Errors) > 0 {
		fmt.Println("\n错误信息:")
		for _, err := range healthStatus.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	if len(healthStatus.Services) > 0 {
		fmt.Println("\n服务状态:")
		for name, status := range healthStatus.Services {
			statusStr := "❌"
			if status {
				statusStr = "✅"
			}
			fmt.Printf("  %s %s\n", statusStr, name)
		}
	}

	if verbose && len(healthStatus.Metrics) > 0 {
		fmt.Println("\n详细指标:")
		for key, value := range healthStatus.Metrics {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}
	fmt.Println()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
