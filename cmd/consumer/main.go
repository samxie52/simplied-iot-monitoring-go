package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/services/consumer"
)

// 版本信息（构建时注入）
var (
	Version   = "dev"
	BuildTime = "unknown"
	GoVersion = "unknown"
)

// CLI flags
var (
	configFile string
	verbose    bool
)

// runConsumer 运行消费者服务
func runConsumer(cmd *cobra.Command, args []string) error {
	log.Printf("启动IoT数据消费者服务...")

	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建消费者服务
	consumerService, err := consumer.NewConsumerService(cfg)
	if err != nil {
		return fmt.Errorf("创建消费者服务失败: %w", err)
	}



	// 启动消费者服务
	log.Printf("Kafka代理: %v", cfg.Kafka.Brokers)
	log.Printf("订阅主题: %v", []string{cfg.Kafka.Topics.DeviceData, cfg.Kafka.Topics.Alerts})
	log.Printf("消费者组: %s", cfg.Kafka.Consumer.GroupID)

	if err := consumerService.Start(); err != nil {
		return fmt.Errorf("启动消费者服务失败: %w", err)
	}
	defer consumerService.Stop()

	log.Printf("IoT数据消费者服务已启动，按 Ctrl+C 停止")

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待停止信号
	<-sigChan
	log.Printf("收到停止信号，正在关闭消费者服务...")

	// 等待服务优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	select {
	case <-shutdownCtx.Done():
		log.Printf("服务关闭超时")
	case <-time.After(1 * time.Second):
		log.Printf("消费者服务已关闭")
	}

	return nil
}

// loadConfig 加载配置
func loadConfig() (*config.AppConfig, error) {
	if configFile == "" {
		return nil, fmt.Errorf("配置文件路径不能为空")
	}

	// 获取配置文件的绝对路径
	absPath, err := filepath.Abs(configFile)
	if err != nil {
		return nil, fmt.Errorf("获取配置文件绝对路径失败: %w", err)
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", absPath)
	}

	// 创建配置管理器
	configManager := config.NewConfigManager()

	// 加载配置
	if err := configManager.Load(absPath, config.Development); err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %w", err)
	}

	// 验证配置
	if err := configManager.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 获取配置
	cfg := configManager.GetConfig()

	if verbose {
		log.Printf("配置加载成功: %s", absPath)
		log.Printf("Kafka配置: %+v", cfg.Kafka)
		log.Printf("消费者配置: %+v", cfg.Consumer)
	}

	return cfg, nil
}

// healthCheck 健康检查命令
func healthCheck(cmd *cobra.Command, args []string) error {
	// TODO: 实现健康检查逻辑
	// 可以检查Kafka连接、配置文件等
	fmt.Println("健康检查功能待实现")
	return nil
}

// version 版本信息命令
func versionInfo(cmd *cobra.Command, args []string) error {
	fmt.Printf("IoT消费者服务\n")
	fmt.Printf("版本: %s\n", Version)
	fmt.Printf("构建时间: %s\n", BuildTime)
	fmt.Printf("Go版本: %s\n", GoVersion)
	return nil
}

func main() {
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

	// 添加全局标志
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "configs/development.yaml", "配置文件路径")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细输出")

	// 添加子命令
	healthCmd := &cobra.Command{
		Use:   "health",
		Short: "健康检查",
		Long:  "检查消费者服务的健康状态",
		RunE:  healthCheck,
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "版本信息",
		Long:  "显示消费者服务的版本信息",
		RunE:  versionInfo,
	}

	rootCmd.AddCommand(healthCmd, versionCmd)

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("程序执行失败: %v", err)
	}
}
