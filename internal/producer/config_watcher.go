package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"simplied-iot-monitoring-go/internal/config"
)

// ConfigChangeEvent 配置变更事件
type ConfigChangeEvent struct {
	Type      string      `json:"type"`      // 变更类型：create, write, remove, rename
	Path      string      `json:"path"`      // 文件路径
	Timestamp time.Time   `json:"timestamp"` // 变更时间
	OldConfig interface{} `json:"old_config,omitempty"`
	NewConfig interface{} `json:"new_config,omitempty"`
}

// ConfigChangeHandler 配置变更处理器接口
type ConfigChangeHandler interface {
	OnConfigChange(event *ConfigChangeEvent) error
	GetConfigType() string
}

// ConfigWatcher 配置文件监控器
type ConfigWatcher struct {
	watcher      *fsnotify.Watcher
	configPaths  map[string]string // 配置类型 -> 文件路径
	handlers     map[string][]ConfigChangeHandler
	currentConfigs map[string]interface{}
	debounceTime time.Duration
	lastChange   map[string]time.Time
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mutex        sync.RWMutex
	metrics      *PrometheusMetrics
}

// NewConfigWatcher 创建配置监控器
func NewConfigWatcher(debounceTime time.Duration, metrics *PrometheusMetrics) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ConfigWatcher{
		watcher:        watcher,
		configPaths:    make(map[string]string),
		handlers:       make(map[string][]ConfigChangeHandler),
		currentConfigs: make(map[string]interface{}),
		debounceTime:   debounceTime,
		lastChange:     make(map[string]time.Time),
		ctx:            ctx,
		cancel:         cancel,
		metrics:        metrics,
	}, nil
}

// RegisterHandler 注册配置变更处理器
func (cw *ConfigWatcher) RegisterHandler(configType string, handler ConfigChangeHandler) {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	if cw.handlers[configType] == nil {
		cw.handlers[configType] = make([]ConfigChangeHandler, 0)
	}
	cw.handlers[configType] = append(cw.handlers[configType], handler)
}

// WatchConfig 监控配置文件
func (cw *ConfigWatcher) WatchConfig(configType, filePath string) error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", filePath)
	}

	// 添加到监控列表
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	cw.configPaths[configType] = absPath

	// 加载初始配置
	if err := cw.loadConfig(configType, absPath); err != nil {
		return fmt.Errorf("failed to load initial config: %w", err)
	}

	// 添加文件监控
	if err := cw.watcher.Add(absPath); err != nil {
		return fmt.Errorf("failed to add file to watcher: %w", err)
	}

	return nil
}

// Start 启动配置监控
func (cw *ConfigWatcher) Start() error {
	cw.wg.Add(1)
	go cw.watchLoop()
	return nil
}

// Stop 停止配置监控
func (cw *ConfigWatcher) Stop() error {
	cw.cancel()
	cw.wg.Wait()
	return cw.watcher.Close()
}

// watchLoop 监控循环
func (cw *ConfigWatcher) watchLoop() {
	defer cw.wg.Done()

	for {
		select {
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}
			cw.handleFileEvent(event)

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Config watcher error: %v\n", err)

		case <-cw.ctx.Done():
			return
		}
	}
}

// handleFileEvent 处理文件事件
func (cw *ConfigWatcher) handleFileEvent(event fsnotify.Event) {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	// 查找对应的配置类型
	var configType string
	for ct, path := range cw.configPaths {
		if path == event.Name {
			configType = ct
			break
		}
	}

	if configType == "" {
		return // 不是我们监控的文件
	}

	// 防抖处理
	now := time.Now()
	if lastChange, exists := cw.lastChange[configType]; exists {
		if now.Sub(lastChange) < cw.debounceTime {
			return
		}
	}
	cw.lastChange[configType] = now

	// 处理不同类型的事件
	switch {
	case event.Op&fsnotify.Write == fsnotify.Write:
		cw.handleConfigChange(configType, event.Name, "write")
	case event.Op&fsnotify.Create == fsnotify.Create:
		cw.handleConfigChange(configType, event.Name, "create")
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		cw.handleConfigChange(configType, event.Name, "remove")
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		cw.handleConfigChange(configType, event.Name, "rename")
	}
}

// handleConfigChange 处理配置变更
func (cw *ConfigWatcher) handleConfigChange(configType, filePath, changeType string) {
	oldConfig := cw.currentConfigs[configType]
	var newConfig interface{}
	var err error

	// 对于删除和重命名事件，新配置为nil
	if changeType != "remove" && changeType != "rename" {
		err = cw.loadConfig(configType, filePath)
		if err != nil {
			fmt.Printf("Failed to load config %s: %v\n", configType, err)
			return
		}
		newConfig = cw.currentConfigs[configType]
	}

	// 创建变更事件
	event := &ConfigChangeEvent{
		Type:      changeType,
		Path:      filePath,
		Timestamp: time.Now(),
		OldConfig: oldConfig,
		NewConfig: newConfig,
	}

	// 通知所有处理器
	if handlers, exists := cw.handlers[configType]; exists {
		for _, handler := range handlers {
			go func(h ConfigChangeHandler) {
				if err := h.OnConfigChange(event); err != nil {
					fmt.Printf("Config change handler error: %v\n", err)
				}
			}(handler)
		}
	}

	// 更新指标
	if cw.metrics != nil {
		// 这里可以添加配置变更相关的指标
	}
}

// loadConfig 加载配置文件
func (cw *ConfigWatcher) loadConfig(configType, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config interface{}
	switch configType {
	case "kafka_producer":
		var kafkaConfig config.KafkaProducer
		if err := json.Unmarshal(data, &kafkaConfig); err != nil {
			return fmt.Errorf("failed to unmarshal kafka config: %w", err)
		}
		config = kafkaConfig
	case "device_simulator":
		var deviceConfig config.DeviceSimulator
		if err := json.Unmarshal(data, &deviceConfig); err != nil {
			return fmt.Errorf("failed to unmarshal device config: %w", err)
		}
		config = deviceConfig
	default:
		// 通用JSON配置
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	cw.currentConfigs[configType] = config
	return nil
}

// GetCurrentConfig 获取当前配置
func (cw *ConfigWatcher) GetCurrentConfig(configType string) interface{} {
	cw.mutex.RLock()
	defer cw.mutex.RUnlock()
	return cw.currentConfigs[configType]
}

// KafkaProducerConfigHandler Kafka生产者配置处理器
type KafkaProducerConfigHandler struct {
	producer *KafkaProducer
	mutex    sync.Mutex
}

// NewKafkaProducerConfigHandler 创建Kafka生产者配置处理器
func NewKafkaProducerConfigHandler(producer *KafkaProducer) *KafkaProducerConfigHandler {
	return &KafkaProducerConfigHandler{
		producer: producer,
	}
}

// GetConfigType 获取配置类型
func (kpch *KafkaProducerConfigHandler) GetConfigType() string {
	return "kafka_producer"
}

// OnConfigChange 处理配置变更
func (kpch *KafkaProducerConfigHandler) OnConfigChange(event *ConfigChangeEvent) error {
	kpch.mutex.Lock()
	defer kpch.mutex.Unlock()

	if event.Type == "remove" || event.Type == "rename" {
		// 配置文件被删除或重命名，使用默认配置
		fmt.Println("Kafka producer config file removed, using default config")
		return nil
	}

	newConfig, ok := event.NewConfig.(config.KafkaProducer)
	if !ok {
		return fmt.Errorf("invalid kafka producer config type")
	}

	// 验证新配置
	if err := kpch.validateConfig(&newConfig); err != nil {
		return fmt.Errorf("invalid kafka producer config: %w", err)
	}

	// 应用新配置（这里简化处理，实际应该更优雅地重新配置）
	fmt.Printf("Kafka producer config updated: batch_size=%d, batch_timeout=%v\n",
		newConfig.BatchSize, newConfig.BatchTimeout)

	// 这里可以添加更复杂的配置更新逻辑
	// 例如：重新创建生产者、更新批处理参数等

	return nil
}

// validateConfig 验证配置
func (kpch *KafkaProducerConfigHandler) validateConfig(cfg *config.KafkaProducer) error {
	if cfg.BatchSize <= 0 {
		return fmt.Errorf("batch_size must be positive")
	}
	if cfg.BatchTimeout <= 0 {
		return fmt.Errorf("batch_timeout must be positive")
	}
	if cfg.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}
	return nil
}

// DeviceSimulatorConfigHandler 设备模拟器配置处理器
type DeviceSimulatorConfigHandler struct {
	simulator *DeviceSimulator
	mutex     sync.Mutex
}

// NewDeviceSimulatorConfigHandler 创建设备模拟器配置处理器
func NewDeviceSimulatorConfigHandler(simulator *DeviceSimulator) *DeviceSimulatorConfigHandler {
	return &DeviceSimulatorConfigHandler{
		simulator: simulator,
	}
}

// GetConfigType 获取配置类型
func (dsch *DeviceSimulatorConfigHandler) GetConfigType() string {
	return "device_simulator"
}

// OnConfigChange 处理配置变更
func (dsch *DeviceSimulatorConfigHandler) OnConfigChange(event *ConfigChangeEvent) error {
	dsch.mutex.Lock()
	defer dsch.mutex.Unlock()

	if event.Type == "remove" || event.Type == "rename" {
		fmt.Println("Device simulator config file removed, using default config")
		return nil
	}

	newConfig, ok := event.NewConfig.(config.DeviceSimulator)
	if !ok {
		return fmt.Errorf("invalid device simulator config type")
	}

	// 验证新配置
	if err := dsch.validateConfig(&newConfig); err != nil {
		return fmt.Errorf("invalid device simulator config: %w", err)
	}

	// 应用新配置
	fmt.Printf("Device simulator config updated: device_count=%d, send_interval=%v\n",
		newConfig.DeviceCount, newConfig.SendInterval)

	// 这里可以添加设备数量动态调整等逻辑

	return nil
}

// validateConfig 验证配置
func (dsch *DeviceSimulatorConfigHandler) validateConfig(cfg *config.DeviceSimulator) error {
	if cfg.DeviceCount <= 0 {
		return fmt.Errorf("device_count must be positive")
	}
	if cfg.SendInterval <= 0 {
		return fmt.Errorf("send_interval must be positive")
	}
	if cfg.WorkerPoolSize <= 0 {
		return fmt.Errorf("worker_pool_size must be positive")
	}
	return nil
}

// ConfigReloader 配置重载器
type ConfigReloader struct {
	watcher  *ConfigWatcher
	handlers map[string]ConfigChangeHandler
	mutex    sync.RWMutex
}

// NewConfigReloader 创建配置重载器
func NewConfigReloader(watcher *ConfigWatcher) *ConfigReloader {
	return &ConfigReloader{
		watcher:  watcher,
		handlers: make(map[string]ConfigChangeHandler),
	}
}

// RegisterComponent 注册组件
func (cr *ConfigReloader) RegisterComponent(configType string, handler ConfigChangeHandler) error {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	cr.handlers[configType] = handler
	cr.watcher.RegisterHandler(configType, handler)
	return nil
}

// ReloadConfig 手动重载配置
func (cr *ConfigReloader) ReloadConfig(configType string) error {
	cr.mutex.RLock()
	handler, exists := cr.handlers[configType]
	cr.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for config type: %s", configType)
	}

	// 获取当前配置
	currentConfig := cr.watcher.GetCurrentConfig(configType)
	if currentConfig == nil {
		return fmt.Errorf("no current config found for type: %s", configType)
	}

	// 创建重载事件
	event := &ConfigChangeEvent{
		Type:      "reload",
		Path:      "",
		Timestamp: time.Now(),
		OldConfig: nil,
		NewConfig: currentConfig,
	}

	return handler.OnConfigChange(event)
}

// GetConfigStatus 获取配置状态
func (cr *ConfigReloader) GetConfigStatus() map[string]interface{} {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	status := make(map[string]interface{})
	for configType := range cr.handlers {
		config := cr.watcher.GetCurrentConfig(configType)
		status[configType] = map[string]interface{}{
			"loaded":      config != nil,
			"last_update": time.Now(), // 这里应该记录实际的更新时间
		}
	}

	return status
}
