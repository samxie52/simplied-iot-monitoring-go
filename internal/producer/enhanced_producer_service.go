package producer

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"simplied-iot-monitoring-go/internal/config"
)

// EnhancedProducerService 增强的生产者服务
type EnhancedProducerService struct {
	// 核心组件
	kafkaProducer    *KafkaProducer
	batchProcessor   *BatchProcessor
	deviceSimulator  *DeviceSimulator
	connectionPool   *ConnectionPool
	workerPool       *BatchWorkerPool
	
	// 第三阶段新增组件
	metrics          *PrometheusMetrics
	healthMonitor    *HealthMonitor
	configWatcher    *ConfigWatcher
	configReloader   *ConfigReloader
	perfOptimizer    *PerformanceOptimizer
	
	// 配置
	config           *config.Config
	
	// 控制
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	mutex            sync.RWMutex
	isRunning        bool
	
	// HTTP服务器（用于指标和健康检查）
	httpServer       *http.Server
	metricsPort      int
}

// NewEnhancedProducerService 创建增强的生产者服务
func NewEnhancedProducerService(cfg *config.Config) (*EnhancedProducerService, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// 创建Prometheus指标收集器
	metrics := NewPrometheusMetrics("iot", "kafka_producer")
	
	// 创建性能优化器
	perfOptimizer := NewPerformanceOptimizer(
		cfg.Producer.BatchSize,
		10, // 缓冲区池大小
		metrics,
	)
	
	// 创建健康监控器
	healthMonitor := NewHealthMonitor(
		30*time.Second, // 检查间隔
		5*time.Second,  // 超时时间
		metrics,
	)
	
	// 创建配置监控器
	configWatcher, err := NewConfigWatcher(
		1*time.Second, // 防抖时间
		metrics,
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create config watcher: %w", err)
	}
	
	// 创建配置重载器
	configReloader := NewConfigReloader(configWatcher)
	
	service := &EnhancedProducerService{
		config:         cfg,
		metrics:        metrics,
		healthMonitor:  healthMonitor,
		configWatcher:  configWatcher,
		configReloader: configReloader,
		perfOptimizer:  perfOptimizer,
		ctx:            ctx,
		cancel:         cancel,
		isRunning:      false,
		metricsPort:    cfg.Web.MetricsPort,
	}
	
	// 初始化核心组件
	if err := service.initializeComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}
	
	// 注册健康检查器
	service.registerHealthCheckers()
	
	// 注册配置处理器
	service.registerConfigHandlers()
	
	return service, nil
}

// initializeComponents 初始化核心组件
func (eps *EnhancedProducerService) initializeComponents() error {
	var err error
	
	// 创建连接池
	eps.connectionPool, err = NewConnectionPool(&ConnectionPoolConfig{
		MaxConnections:    eps.config.Producer.MaxConnections,
		MinConnections:    eps.config.Producer.MinConnections,
		MaxIdleTime:       eps.config.Producer.MaxIdleTime,
		ConnectionTimeout: eps.config.Producer.ConnectionTimeout,
		HealthCheckInterval: 30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	
	// 创建Kafka生产者
	eps.kafkaProducer, err = NewKafkaProducer(
		eps.config.Kafka.Brokers,
		eps.config.Producer.Topic,
		&eps.config.Producer,
	)
	if err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	
	// 创建批处理器
	eps.batchProcessor = NewBatchProcessor(
		&eps.config.Producer,
		eps.kafkaProducer.producer,
		eps.config.Producer.PartitionCount,
	)
	
	// 创建工作池
	eps.workerPool = NewBatchWorkerPool(
		eps.config.Producer.WorkerPoolSize,
		eps.config.Producer.QueueSize,
		eps.config.Producer.WorkerTimeout,
	)
	
	// 创建设备模拟器
	eps.deviceSimulator, err = NewDeviceSimulator(
		&eps.config.Device,
		eps.kafkaProducer,
		eps.workerPool,
	)
	if err != nil {
		return fmt.Errorf("failed to create device simulator: %w", err)
	}
	
	return nil
}

// registerHealthCheckers 注册健康检查器
func (eps *EnhancedProducerService) registerHealthCheckers() {
	// Kafka生产者健康检查
	kafkaChecker := NewKafkaHealthChecker(eps.kafkaProducer)
	eps.healthMonitor.RegisterChecker(kafkaChecker)
	
	// 连接池健康检查
	poolChecker := NewConnectionPoolHealthChecker(eps.connectionPool)
	eps.healthMonitor.RegisterChecker(poolChecker)
	
	// 系统资源健康检查
	resourceChecker := NewSystemResourceHealthChecker()
	eps.healthMonitor.RegisterChecker(resourceChecker)
	
	// 设置告警阈值
	eps.healthMonitor.SetAlertThreshold("kafka_producer", 10)
	eps.healthMonitor.SetAlertThreshold("connection_pool", 5)
	
	// 注册告警回调
	eps.healthMonitor.RegisterAlertCallback(eps.handleHealthAlert)
}

// registerConfigHandlers 注册配置处理器
func (eps *EnhancedProducerService) registerConfigHandlers() {
	// Kafka生产者配置处理器
	kafkaHandler := NewKafkaProducerConfigHandler(eps.kafkaProducer)
	eps.configReloader.RegisterComponent("kafka_producer", kafkaHandler)
	
	// 设备模拟器配置处理器
	deviceHandler := NewDeviceSimulatorConfigHandler(eps.deviceSimulator)
	eps.configReloader.RegisterComponent("device_simulator", deviceHandler)
}

// handleHealthAlert 处理健康告警
func (eps *EnhancedProducerService) handleHealthAlert(component string, health *ComponentHealth) {
	fmt.Printf("ALERT: Component %s status changed to %s: %s\n",
		component, health.Status.String(), health.Message)
	
	// 这里可以添加更复杂的告警处理逻辑
	// 例如：发送邮件、Slack通知、自动恢复等
}

// Start 启动服务
func (eps *EnhancedProducerService) Start() error {
	eps.mutex.Lock()
	defer eps.mutex.Unlock()
	
	if eps.isRunning {
		return fmt.Errorf("service is already running")
	}
	
	// 启动HTTP服务器（指标和健康检查）
	if err := eps.startHTTPServer(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	
	// 启动健康监控
	if err := eps.healthMonitor.Start(); err != nil {
		return fmt.Errorf("failed to start health monitor: %w", err)
	}
	
	// 启动配置监控
	if err := eps.configWatcher.Start(); err != nil {
		return fmt.Errorf("failed to start config watcher: %w", err)
	}
	
	// 启动连接池
	if err := eps.connectionPool.Start(); err != nil {
		return fmt.Errorf("failed to start connection pool: %w", err)
	}
	
	// 启动Kafka生产者
	if err := eps.kafkaProducer.Start(); err != nil {
		return fmt.Errorf("failed to start Kafka producer: %w", err)
	}
	
	// 启动批处理器
	if err := eps.batchProcessor.Start(); err != nil {
		return fmt.Errorf("failed to start batch processor: %w", err)
	}
	
	// 启动工作池
	if err := eps.workerPool.Start(); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}
	
	// 启动设备模拟器
	if err := eps.deviceSimulator.Start(); err != nil {
		return fmt.Errorf("failed to start device simulator: %w", err)
	}
	
	// 启动指标更新协程
	eps.wg.Add(1)
	go eps.metricsUpdateLoop()
	
	eps.isRunning = true
	fmt.Println("Enhanced Producer Service started successfully")
	
	return nil
}

// Stop 停止服务
func (eps *EnhancedProducerService) Stop() error {
	eps.mutex.Lock()
	defer eps.mutex.Unlock()
	
	if !eps.isRunning {
		return nil
	}
	
	fmt.Println("Stopping Enhanced Producer Service...")
	
	// 取消上下文
	eps.cancel()
	
	// 停止设备模拟器
	if eps.deviceSimulator != nil {
		eps.deviceSimulator.Stop()
	}
	
	// 停止工作池
	if eps.workerPool != nil {
		eps.workerPool.Stop()
	}
	
	// 停止批处理器
	if eps.batchProcessor != nil {
		eps.batchProcessor.Stop()
	}
	
	// 停止Kafka生产者
	if eps.kafkaProducer != nil {
		eps.kafkaProducer.Stop()
	}
	
	// 停止连接池
	if eps.connectionPool != nil {
		eps.connectionPool.Stop()
	}
	
	// 停止配置监控
	if eps.configWatcher != nil {
		eps.configWatcher.Stop()
	}
	
	// 停止健康监控
	if eps.healthMonitor != nil {
		eps.healthMonitor.Stop()
	}
	
	// 停止HTTP服务器
	if eps.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		eps.httpServer.Shutdown(ctx)
	}
	
	// 等待所有协程结束
	eps.wg.Wait()
	
	eps.isRunning = false
	fmt.Println("Enhanced Producer Service stopped")
	
	return nil
}

// startHTTPServer 启动HTTP服务器
func (eps *EnhancedProducerService) startHTTPServer() error {
	mux := http.NewServeMux()
	
	// Prometheus指标端点
	mux.Handle("/metrics", promhttp.Handler())
	
	// 健康检查端点
	mux.HandleFunc("/health", eps.handleHealthCheck)
	mux.HandleFunc("/health/", eps.handleComponentHealth)
	
	// 配置状态端点
	mux.HandleFunc("/config/status", eps.handleConfigStatus)
	mux.HandleFunc("/config/reload", eps.handleConfigReload)
	
	// 性能统计端点
	mux.HandleFunc("/stats", eps.handleStats)
	
	eps.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", eps.metricsPort),
		Handler: mux,
	}
	
	go func() {
		if err := eps.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()
	
	return nil
}

// handleHealthCheck 处理健康检查请求
func (eps *EnhancedProducerService) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	health := eps.healthMonitor.GetHealth()
	
	w.Header().Set("Content-Type", "application/json")
	if health.OverallStatus == HealthStatusHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	// 返回健康状态JSON
	fmt.Fprintf(w, `{
		"status": "%s",
		"timestamp": "%s",
		"uptime": "%s",
		"version": "%s",
		"components": %d
	}`,
		health.OverallStatus.String(),
		health.Timestamp.Format(time.RFC3339),
		health.Uptime.String(),
		health.Version,
		len(health.Components),
	)
}

// handleComponentHealth 处理组件健康检查请求
func (eps *EnhancedProducerService) handleComponentHealth(w http.ResponseWriter, r *http.Request) {
	component := r.URL.Path[len("/health/"):]
	health := eps.healthMonitor.GetComponentHealth(component)
	
	w.Header().Set("Content-Type", "application/json")
	if health == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"error": "Component not found: %s"}`, component)
		return
	}
	
	if health.Status == HealthStatusHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	fmt.Fprintf(w, `{
		"name": "%s",
		"status": "%s",
		"message": "%s",
		"last_check": "%s",
		"check_count": %d,
		"error_count": %d
	}`,
		health.Name,
		health.Status.String(),
		health.Message,
		health.LastCheck.Format(time.RFC3339),
		health.CheckCount,
		health.ErrorCount,
	)
}

// handleConfigStatus 处理配置状态请求
func (eps *EnhancedProducerService) handleConfigStatus(w http.ResponseWriter, r *http.Request) {
	status := eps.configReloader.GetConfigStatus()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	fmt.Fprintf(w, `{"config_status": %v}`, status)
}

// handleConfigReload 处理配置重载请求
func (eps *EnhancedProducerService) handleConfigReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	
	configType := r.URL.Query().Get("type")
	if configType == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "Missing config type parameter"}`)
		return
	}
	
	if err := eps.configReloader.ReloadConfig(configType); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "%s"}`, err.Error())
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message": "Config reloaded successfully", "type": "%s"}`, configType)
}

// handleStats 处理统计信息请求
func (eps *EnhancedProducerService) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"kafka_producer":    eps.kafkaProducer.GetMetrics(),
		"connection_pool":   eps.connectionPool.GetStats(),
		"worker_pool":       eps.workerPool.GetMetrics(),
		"device_simulator":  eps.deviceSimulator.GetStats(),
		"performance_optimizer": eps.perfOptimizer.GetStats(),
		"uptime":           eps.metrics.GetUptime().String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	fmt.Fprintf(w, `{"stats": %v}`, stats)
}

// metricsUpdateLoop 指标更新循环
func (eps *EnhancedProducerService) metricsUpdateLoop() {
	defer eps.wg.Done()
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			eps.updateMetrics()
		case <-eps.ctx.Done():
			return
		}
	}
}

// updateMetrics 更新指标
func (eps *EnhancedProducerService) updateMetrics() {
	// 更新资源指标
	eps.metrics.UpdateResourceMetrics()
	
	// 更新业务指标
	if eps.deviceSimulator != nil {
		stats := eps.deviceSimulator.GetStats()
		if deviceCount, ok := stats["device_count"].(int); ok {
			eps.metrics.SetDeviceCount(deviceCount)
		}
	}
	
	if eps.connectionPool != nil {
		stats := eps.connectionPool.GetStats()
		if activeConns, ok := stats["active_connections"].(int); ok {
			eps.metrics.SetActiveConnections(activeConns)
		}
	}
	
	if eps.kafkaProducer != nil {
		producerMetrics := eps.kafkaProducer.GetMetrics()
		eps.metrics.SetQueueDepth(int(producerMetrics.QueueDepth))
	}
}

// IsRunning 检查服务是否运行中
func (eps *EnhancedProducerService) IsRunning() bool {
	eps.mutex.RLock()
	defer eps.mutex.RUnlock()
	return eps.isRunning
}

// GetMetrics 获取指标
func (eps *EnhancedProducerService) GetMetrics() *PrometheusMetrics {
	return eps.metrics
}

// GetHealthMonitor 获取健康监控器
func (eps *EnhancedProducerService) GetHealthMonitor() *HealthMonitor {
	return eps.healthMonitor
}

// GetConfigReloader 获取配置重载器
func (eps *EnhancedProducerService) GetConfigReloader() *ConfigReloader {
	return eps.configReloader
}
