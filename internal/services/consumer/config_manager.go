package consumer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"simplied-iot-monitoring-go/internal/config"
)

// ConsumerConfigManager 消费者配置管理器
type ConsumerConfigManager struct {
	// 当前配置
	currentConfig *config.AppConfig
	mutex         sync.RWMutex

	// 配置加载器
	configLoader ConfigLoader

	// 配置验证器
	validator ConfigValidator

	// 配置监听器
	watchers []ConfigWatcher

	// 热重载配置
	hotReloadEnabled bool
	reloadInterval   time.Duration
	
	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ConfigLoader 配置加载器接口
type ConfigLoader interface {
	LoadConfig(ctx context.Context) (*config.AppConfig, error)
	GetConfigSource() string
	SupportsHotReload() bool
}

// ConfigValidator 配置验证器接口
type ConfigValidator interface {
	ValidateConfig(cfg *config.AppConfig) error
	ValidateConsumerConfig(cfg *config.KafkaConsumer) error
	GetValidationRules() []ValidationRule
}

// ConfigWatcher 配置监听器接口
type ConfigWatcher interface {
	OnConfigChanged(oldConfig, newConfig *config.AppConfig) error
	GetWatcherName() string
}

// ValidationRule 验证规则
type ValidationRule struct {
	Name        string
	Description string
	Validator   func(*config.AppConfig) error
}

// NewConsumerConfigManager 创建消费者配置管理器
func NewConsumerConfigManager(loader ConfigLoader, validator ConfigValidator) *ConsumerConfigManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ConsumerConfigManager{
		configLoader:     loader,
		validator:        validator,
		watchers:         make([]ConfigWatcher, 0),
		hotReloadEnabled: false,
		reloadInterval:   30 * time.Second,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// LoadConfig 加载配置
func (ccm *ConsumerConfigManager) LoadConfig() error {
	ccm.mutex.Lock()
	defer ccm.mutex.Unlock()

	log.Println("正在加载消费者配置...")

	// 加载配置
	cfg, err := ccm.configLoader.LoadConfig(ccm.ctx)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 验证配置
	if ccm.validator != nil {
		if err := ccm.validator.ValidateConfig(cfg); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}
	}

	// 保存配置
	oldConfig := ccm.currentConfig
	ccm.currentConfig = cfg

	log.Printf("配置加载成功，来源: %s", ccm.configLoader.GetConfigSource())

	// 通知配置变更
	if oldConfig != nil {
		ccm.notifyConfigChanged(oldConfig, cfg)
	}

	return nil
}

// GetConfig 获取当前配置
func (ccm *ConsumerConfigManager) GetConfig() *config.AppConfig {
	ccm.mutex.RLock()
	defer ccm.mutex.RUnlock()
	return ccm.currentConfig
}

// GetConsumerConfig 获取消费者配置
func (ccm *ConsumerConfigManager) GetConsumerConfig() *config.KafkaConsumer {
	ccm.mutex.RLock()
	defer ccm.mutex.RUnlock()
	
	if ccm.currentConfig == nil {
		return nil
	}
	
	return &ccm.currentConfig.Kafka.Consumer
}

// UpdateConfig 更新配置
func (ccm *ConsumerConfigManager) UpdateConfig(newConfig *config.AppConfig) error {
	ccm.mutex.Lock()
	defer ccm.mutex.Unlock()

	// 验证新配置
	if ccm.validator != nil {
		if err := ccm.validator.ValidateConfig(newConfig); err != nil {
			return fmt.Errorf("new config validation failed: %w", err)
		}
	}

	oldConfig := ccm.currentConfig
	ccm.currentConfig = newConfig

	log.Println("配置已更新")

	// 通知配置变更
	if oldConfig != nil {
		ccm.notifyConfigChanged(oldConfig, newConfig)
	}

	return nil
}

// EnableHotReload 启用热重载
func (ccm *ConsumerConfigManager) EnableHotReload(interval time.Duration) error {
	ccm.mutex.Lock()
	defer ccm.mutex.Unlock()

	if !ccm.configLoader.SupportsHotReload() {
		return fmt.Errorf("config loader does not support hot reload")
	}

	ccm.hotReloadEnabled = true
	ccm.reloadInterval = interval

	// 启动热重载协程
	ccm.wg.Add(1)
	go ccm.runHotReload()

	log.Printf("热重载已启用，检查间隔: %v", interval)
	return nil
}

// DisableHotReload 禁用热重载
func (ccm *ConsumerConfigManager) DisableHotReload() {
	ccm.mutex.Lock()
	defer ccm.mutex.Unlock()

	ccm.hotReloadEnabled = false
	log.Println("热重载已禁用")
}

// AddWatcher 添加配置监听器
func (ccm *ConsumerConfigManager) AddWatcher(watcher ConfigWatcher) {
	ccm.mutex.Lock()
	defer ccm.mutex.Unlock()
	
	ccm.watchers = append(ccm.watchers, watcher)
	log.Printf("已添加配置监听器: %s", watcher.GetWatcherName())
}

// RemoveWatcher 移除配置监听器
func (ccm *ConsumerConfigManager) RemoveWatcher(watcherName string) {
	ccm.mutex.Lock()
	defer ccm.mutex.Unlock()
	
	for i, watcher := range ccm.watchers {
		if watcher.GetWatcherName() == watcherName {
			ccm.watchers = append(ccm.watchers[:i], ccm.watchers[i+1:]...)
			log.Printf("已移除配置监听器: %s", watcherName)
			return
		}
	}
}

// Stop 停止配置管理器
func (ccm *ConsumerConfigManager) Stop() {
	ccm.cancel()
	ccm.wg.Wait()
	log.Println("配置管理器已停止")
}

// runHotReload 运行热重载
func (ccm *ConsumerConfigManager) runHotReload() {
	defer ccm.wg.Done()
	
	ticker := time.NewTicker(ccm.reloadInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if ccm.hotReloadEnabled {
				ccm.checkAndReloadConfig()
			}
		case <-ccm.ctx.Done():
			return
		}
	}
}

// checkAndReloadConfig 检查并重载配置
func (ccm *ConsumerConfigManager) checkAndReloadConfig() {
	// 尝试加载新配置
	newConfig, err := ccm.configLoader.LoadConfig(ccm.ctx)
	if err != nil {
		log.Printf("热重载配置失败: %v", err)
		return
	}

	// 验证新配置
	if ccm.validator != nil {
		if err := ccm.validator.ValidateConfig(newConfig); err != nil {
			log.Printf("热重载配置验证失败: %v", err)
			return
		}
	}

	// 检查配置是否有变化
	ccm.mutex.Lock()
	oldConfig := ccm.currentConfig
	configChanged := ccm.hasConfigChanged(oldConfig, newConfig)
	
	if configChanged {
		ccm.currentConfig = newConfig
		ccm.mutex.Unlock()
		
		log.Println("检测到配置变化，正在热重载...")
		ccm.notifyConfigChanged(oldConfig, newConfig)
	} else {
		ccm.mutex.Unlock()
	}
}

// hasConfigChanged 检查配置是否有变化
func (ccm *ConsumerConfigManager) hasConfigChanged(oldConfig, newConfig *config.AppConfig) bool {
	if oldConfig == nil || newConfig == nil {
		return true
	}
	
	// 简化的配置比较，实际应该进行深度比较
	return oldConfig.Kafka.Consumer.GroupID != newConfig.Kafka.Consumer.GroupID ||
		   oldConfig.Kafka.Consumer.SessionTimeout != newConfig.Kafka.Consumer.SessionTimeout ||
		   oldConfig.Kafka.Consumer.MaxPollRecords != newConfig.Kafka.Consumer.MaxPollRecords ||
		   len(oldConfig.Kafka.Brokers) != len(newConfig.Kafka.Brokers)
}

// notifyConfigChanged 通知配置变更
func (ccm *ConsumerConfigManager) notifyConfigChanged(oldConfig, newConfig *config.AppConfig) {
	for _, watcher := range ccm.watchers {
		if err := watcher.OnConfigChanged(oldConfig, newConfig); err != nil {
			log.Printf("配置监听器 %s 处理配置变更失败: %v", watcher.GetWatcherName(), err)
		}
	}
}

// FileConfigLoader 文件配置加载器
type FileConfigLoader struct {
	configPath string
	manager    config.ConfigManager
}

// NewFileConfigLoader 创建文件配置加载器
func NewFileConfigLoader(configPath string) *FileConfigLoader {
	return &FileConfigLoader{
		configPath: configPath,
		manager:    config.NewConfigManager(),
	}
}

// LoadConfig 加载配置
func (fcl *FileConfigLoader) LoadConfig(ctx context.Context) (*config.AppConfig, error) {
	if err := fcl.manager.Load(fcl.configPath, config.Development); err != nil {
		return nil, err
	}
	return fcl.manager.GetConfig(), nil
}

// GetConfigSource 获取配置源
func (fcl *FileConfigLoader) GetConfigSource() string {
	return fmt.Sprintf("file://%s", fcl.configPath)
}

// SupportsHotReload 是否支持热重载
func (fcl *FileConfigLoader) SupportsHotReload() bool {
	return true
}

// DefaultConfigValidator 默认配置验证器
type DefaultConfigValidator struct {
	rules []ValidationRule
}

// NewDefaultConfigValidator 创建默认配置验证器
func NewDefaultConfigValidator() *DefaultConfigValidator {
	validator := &DefaultConfigValidator{
		rules: make([]ValidationRule, 0),
	}
	
	// 添加默认验证规则
	validator.addDefaultRules()
	
	return validator
}

// ValidateConfig 验证配置
func (dcv *DefaultConfigValidator) ValidateConfig(cfg *config.AppConfig) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// 执行所有验证规则
	for _, rule := range dcv.rules {
		if err := rule.Validator(cfg); err != nil {
			return fmt.Errorf("validation rule '%s' failed: %w", rule.Name, err)
		}
	}

	// 验证消费者配置
	return dcv.ValidateConsumerConfig(&cfg.Kafka.Consumer)
}

// ValidateConsumerConfig 验证消费者配置
func (dcv *DefaultConfigValidator) ValidateConsumerConfig(cfg *config.KafkaConsumer) error {
	if cfg == nil {
		return fmt.Errorf("consumer config cannot be nil")
	}

	if cfg.GroupID == "" {
		return fmt.Errorf("consumer group_id cannot be empty")
	}

	if cfg.SessionTimeout <= 0 {
		return fmt.Errorf("consumer session_timeout must be positive")
	}

	if cfg.MaxPollRecords <= 0 {
		return fmt.Errorf("consumer max_poll_records must be positive")
	}

	return nil
}

// GetValidationRules 获取验证规则
func (dcv *DefaultConfigValidator) GetValidationRules() []ValidationRule {
	return dcv.rules
}

// addDefaultRules 添加默认验证规则
func (dcv *DefaultConfigValidator) addDefaultRules() {
	dcv.rules = append(dcv.rules, ValidationRule{
		Name:        "kafka_brokers_not_empty",
		Description: "Kafka brokers must not be empty",
		Validator: func(cfg *config.AppConfig) error {
			if len(cfg.Kafka.Brokers) == 0 {
				return fmt.Errorf("kafka brokers cannot be empty")
			}
			return nil
		},
	})

	dcv.rules = append(dcv.rules, ValidationRule{
		Name:        "kafka_topics_not_empty",
		Description: "Kafka topics must not be empty",
		Validator: func(cfg *config.AppConfig) error {
			if cfg.Kafka.Topics.DeviceData == "" || cfg.Kafka.Topics.Alerts == "" {
				return fmt.Errorf("kafka topics cannot be empty")
			}
			return nil
		},
	})

	dcv.rules = append(dcv.rules, ValidationRule{
		Name:        "consumer_group_id_valid",
		Description: "Consumer group ID must be valid",
		Validator: func(cfg *config.AppConfig) error {
			if cfg.Kafka.Consumer.GroupID == "" {
				return fmt.Errorf("consumer group_id cannot be empty")
			}
			if len(cfg.Kafka.Consumer.GroupID) > 255 {
				return fmt.Errorf("consumer group_id too long (max 255 characters)")
			}
			return nil
		},
	})
}

// ConsumerServiceWatcher 消费者服务配置监听器
type ConsumerServiceWatcher struct {
	service *ConsumerService
}

// NewConsumerServiceWatcher 创建消费者服务配置监听器
func NewConsumerServiceWatcher(service *ConsumerService) *ConsumerServiceWatcher {
	return &ConsumerServiceWatcher{
		service: service,
	}
}

// OnConfigChanged 配置变更处理
func (csw *ConsumerServiceWatcher) OnConfigChanged(oldConfig, newConfig *config.AppConfig) error {
	log.Println("消费者服务检测到配置变更")
	
	// 检查是否需要重启服务
	needRestart := csw.needsRestart(oldConfig, newConfig)
	
	if needRestart {
		log.Println("配置变更需要重启消费者服务")
		
		// 停止服务
		if err := csw.service.Stop(); err != nil {
			return fmt.Errorf("failed to stop consumer service: %w", err)
		}
		
		// 更新配置
		if err := csw.service.UpdateConfig(newConfig); err != nil {
			return fmt.Errorf("failed to update config: %w", err)
		}
		
		// 重启服务
		if err := csw.service.Start(); err != nil {
			return fmt.Errorf("failed to restart consumer service: %w", err)
		}
		
		log.Println("消费者服务已重启")
	} else {
		// 只更新配置，不重启
		if err := csw.service.UpdateConfig(newConfig); err != nil {
			return fmt.Errorf("failed to update config: %w", err)
		}
		log.Println("消费者服务配置已更新")
	}
	
	return nil
}

// needsRestart 检查是否需要重启
func (csw *ConsumerServiceWatcher) needsRestart(oldConfig, newConfig *config.AppConfig) bool {
	if oldConfig == nil || newConfig == nil {
		return true
	}
	
	// 检查关键配置是否变化
	return oldConfig.Kafka.Consumer.GroupID != newConfig.Kafka.Consumer.GroupID ||
		   len(oldConfig.Kafka.Brokers) != len(newConfig.Kafka.Brokers) ||
		   oldConfig.Kafka.Topics.DeviceData != newConfig.Kafka.Topics.DeviceData ||
		   oldConfig.Kafka.Topics.Alerts != newConfig.Kafka.Topics.Alerts
}

// GetWatcherName 获取监听器名称
func (csw *ConsumerServiceWatcher) GetWatcherName() string {
	return "ConsumerServiceWatcher"
}
