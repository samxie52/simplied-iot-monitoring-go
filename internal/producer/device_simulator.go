package producer

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/models"
)

// DeviceSimulator 设备数据模拟器
type DeviceSimulator struct {
	config     *config.DeviceSimulator
	devices    map[string]*SimulatedDevice
	producer   *KafkaProducer
	workerPool *WorkerPool
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	isRunning  bool
	mutex      sync.RWMutex
}

// SimulatedDevice 模拟设备
type SimulatedDevice struct {
	DeviceID   string
	DeviceType string
	Location   models.LocationInfo
	Sensors    map[string]*SensorSimulator
	SmartSensors map[string]*SmartDataGenerator // 新增：智能传感器
	LastUpdate time.Time
	IsActive   bool
}

// SensorSimulator 传感器模拟器（已弃用，使用SmartDataGenerator）
type SensorSimulator struct {
	SensorType     string
	BaseValue      float64
	Variance       float64
	TrendFactor    float64
	AnomalyRate    float64
	LastValue      float64
	TrendDirection int
	Random         *rand.Rand
	Unit           string
	MinValue       float64
	MaxValue       float64
	
	// 新增：智能数据生成器
	SmartGenerator *SmartDataGenerator
}

// WorkerPool 工作协程池
type WorkerPool struct {
	WorkerCount int
	TaskQueue   chan Task
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	isRunning   bool
	mutex       sync.RWMutex
}

// Task 任务接口
type Task interface {
	Execute() error
}

// DataGenerationTask 数据生成任务
type DataGenerationTask struct {
	Device   *SimulatedDevice
	Producer *KafkaProducer
}

// NewDeviceSimulator 创建设备模拟器
func NewDeviceSimulator(config *config.DeviceSimulator, producer *KafkaProducer) *DeviceSimulator {
	ctx, cancel := context.WithCancel(context.Background())

	return &DeviceSimulator{
		config:     config,
		devices:    make(map[string]*SimulatedDevice),
		producer:   producer,
		workerPool: NewWorkerPool(config.WorkerPoolSize, config.QueueBufferSize),
		ctx:        ctx,
		cancel:     cancel,
		isRunning:  false,
	}
}

// NewWorkerPool 创建工作协程池
func NewWorkerPool(workerCount, queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		WorkerCount: workerCount,
		TaskQueue:   make(chan Task, queueSize),
		ctx:         ctx,
		cancel:      cancel,
		isRunning:   false,
	}
}

// Start 启动设备模拟器
func (ds *DeviceSimulator) Start() error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.isRunning {
		return fmt.Errorf("device simulator is already running")
	}

	// 初始化设备
	if err := ds.initializeDevices(); err != nil {
		return fmt.Errorf("failed to initialize devices: %w", err)
	}

	// 启动工作池
	if err := ds.workerPool.Start(); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	ds.isRunning = true

	// 启动数据生成协程
	ds.wg.Add(1)
	go ds.dataGenerationLoop()

	return nil
}

// initializeDevices 初始化设备
func (ds *DeviceSimulator) initializeDevices() error {
	deviceTypes := []string{"temperature", "humidity", "pressure", "switch", "current"}

	for i := 0; i < ds.config.DeviceCount; i++ {
		deviceID := fmt.Sprintf("device_%04d", i+1)
		deviceType := deviceTypes[i%len(deviceTypes)]

		device := &SimulatedDevice{
			DeviceID:   deviceID,
			DeviceType: deviceType,
			Location: models.LocationInfo{
				Building: fmt.Sprintf("Building_%d", (i/100)+1),
				Floor:    (i / 20) + 1,
				Room:     fmt.Sprintf("Room_%d", (i/5)+1),
				Zone:     fmt.Sprintf("Zone_%d", i%4+1),
			},
			Sensors:    make(map[string]*SensorSimulator),
			LastUpdate: time.Now(),
			IsActive:   true,
		}

		// 根据设备类型创建传感器
		ds.createSensorsForDevice(device)

		ds.devices[deviceID] = device
	}

	return nil
}

// createSensorsForDevice 为设备创建传感器
func (ds *DeviceSimulator) createSensorsForDevice(device *SimulatedDevice) {
	switch device.DeviceType {
	case "temperature":
		device.Sensors["temperature"] = &SensorSimulator{
			SensorType:     "temperature",
			BaseValue:      22.0,
			Variance:       2.0,
			TrendFactor:    ds.config.TrendStrength,
			AnomalyRate:    ds.config.AnomalyRate,
			TrendDirection: 1,
			Random:         rand.New(rand.NewSource(time.Now().UnixNano())),
			Unit:           "°C",
			MinValue:       -40.0,
			MaxValue:       80.0,
		}

	case "humidity":
		device.Sensors["humidity"] = &SensorSimulator{
			SensorType:     "humidity",
			BaseValue:      50.0,
			Variance:       10.0,
			TrendFactor:    ds.config.TrendStrength,
			AnomalyRate:    ds.config.AnomalyRate,
			TrendDirection: -1,
			Random:         rand.New(rand.NewSource(time.Now().UnixNano())),
			Unit:           "%",
			MinValue:       0.0,
			MaxValue:       100.0,
		}

	case "pressure":
		device.Sensors["pressure"] = &SensorSimulator{
			SensorType:     "pressure",
			BaseValue:      1013.25,
			Variance:       50.0,
			TrendFactor:    ds.config.TrendStrength,
			AnomalyRate:    ds.config.AnomalyRate,
			TrendDirection: 1,
			Random:         rand.New(rand.NewSource(time.Now().UnixNano())),
			Unit:           "hPa",
			MinValue:       800.0,
			MaxValue:       1200.0,
		}

	case "switch":
		device.Sensors["switch"] = &SensorSimulator{
			SensorType:     "switch",
			BaseValue:      0.0,
			Variance:       0.0,
			TrendFactor:    0.0,
			AnomalyRate:    ds.config.AnomalyRate * 0.1, // 开关异常率较低
			TrendDirection: 0,
			Random:         rand.New(rand.NewSource(time.Now().UnixNano())),
			Unit:           "",
			MinValue:       0.0,
			MaxValue:       1.0,
		}

	case "current":
		device.Sensors["current"] = &SensorSimulator{
			SensorType:     "current",
			BaseValue:      5.0,
			Variance:       1.0,
			TrendFactor:    ds.config.TrendStrength,
			AnomalyRate:    ds.config.AnomalyRate,
			TrendDirection: 1,
			Random:         rand.New(rand.NewSource(time.Now().UnixNano())),
			Unit:           "A",
			MinValue:       0.0,
			MaxValue:       20.0,
		}
	}
}

// dataGenerationLoop 数据生成循环
func (ds *DeviceSimulator) dataGenerationLoop() {
	defer ds.wg.Done()

	ticker := time.NewTicker(ds.config.SampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ds.generateAndSendData()

		case <-ds.ctx.Done():
			return
		}
	}
}

// generateAndSendData 生成并发送数据
func (ds *DeviceSimulator) generateAndSendData() {
	ds.mutex.RLock()
	devices := make([]*SimulatedDevice, 0, len(ds.devices))
	for _, device := range ds.devices {
		if device.IsActive {
			devices = append(devices, device)
		}
	}
	ds.mutex.RUnlock()

	// 将设备分配给工作协程
	for _, device := range devices {
		task := &DataGenerationTask{
			Device:   device,
			Producer: ds.producer,
		}

		select {
		case ds.workerPool.TaskQueue <- task:
			// 任务已提交
		default:
			// 任务队列已满，跳过此设备
		}
	}
}

// Execute 执行数据生成任务
func (task *DataGenerationTask) Execute() error {
	// 为每个传感器生成数据
	for sensorType, sensor := range task.Device.Sensors {
		value := sensor.GenerateValue()

		// 生成传感器数据
		sensorData := models.NewSensorData()

		// 根据传感器类型设置相应的传感器数据
		switch sensorType {
		case "temperature":
			sensorData.Temperature = &models.TemperatureSensor{
				Value:  value,
				Unit:   "°C",
				Status: models.SensorStatusNormal,
			}
		case "humidity":
			sensorData.Humidity = &models.HumiditySensor{
				Value:  value,
				Unit:   "%",
				Status: models.SensorStatusNormal,
			}
		case "pressure":
			sensorData.Pressure = &models.PressureSensor{
				Value:  value,
				Unit:   "hPa",
				Status: models.SensorStatusNormal,
			}
		case "current":
			sensorData.Current = &models.CurrentSensor{
				Value:  value,
				Unit:   "A",
				Status: models.SensorStatusNormal,
			}
		case "switch":
			sensorData.Switch = &models.SwitchSensor{
				Value:  value > 0.5, // 转换为布尔值
				Status: models.SwitchStatusOn,
			}
			if value <= 0.5 {
				sensorData.Switch.Status = models.SwitchStatusOff
			}
		}

		// 创建设备信息
		deviceInfo := models.DeviceInfo{
			Model:           "IoT-Sensor-v1.0",
			Manufacturer:    "Industrial IoT Corp",
			FirmwareVersion: "1.2.3",
			HardwareVersion: "2.1.0",
			SerialNumber:    fmt.Sprintf("SN%s", task.Device.DeviceID),
			BatteryLevel:    int(85 + rand.Float64()*15),  // 85-100%
			SignalStrength:  int(-30 - rand.Float64()*40), // -30 to -70 dBm
			NetworkType:     models.NetworkTypeWiFi,
		}

		deviceData := &models.Device{
			DeviceID:   task.Device.DeviceID,
			DeviceType: task.Device.DeviceType,
			Timestamp:  time.Now().UnixMilli(),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			DeviceInfo: deviceInfo,
			Location:   task.Device.Location,
			SensorData: *sensorData,
			Status:     models.DeviceStatusOnline,
			LastSeen:   time.Now().UnixMilli(),
		}

		// 发送到Kafka
		key := fmt.Sprintf("%s_%s", task.Device.DeviceID, sensorType)
		if err := task.Producer.SendMessage(key, deviceData); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	task.Device.LastUpdate = time.Now()
	return nil
}

// GenerateValue 生成传感器数值
func (s *SensorSimulator) GenerateValue() float64 {
	var value float64

	switch s.SensorType {
	case "switch":
		// 开关类型：随机生成0或1
		if s.Random.Float64() < 0.1 { // 10%概率切换状态
			s.LastValue = 1.0 - s.LastValue
		}
		value = s.LastValue

	default:
		// 连续值传感器：使用正态分布
		normalValue := s.Random.NormFloat64()*s.Variance + s.BaseValue

		// 添加趋势因子
		if s.TrendFactor > 0 {
			trendValue := normalValue + (s.TrendFactor * float64(s.TrendDirection))

			// 定期改变趋势方向
			if s.Random.Float64() < 0.01 { // 1%概率改变趋势
				s.TrendDirection *= -1
			}

			normalValue = trendValue
		}

		// 异常注入
		if s.Random.Float64() < s.AnomalyRate {
			anomalyMultiplier := 2.0 + s.Random.Float64()*3.0
			if s.Random.Float64() < 0.5 {
				anomalyMultiplier = -anomalyMultiplier
			}
			normalValue *= anomalyMultiplier
		}

		// 平滑过渡
		smoothedValue := s.LastValue*0.7 + normalValue*0.3

		// 限制在合理范围内
		value = math.Max(s.MinValue, math.Min(s.MaxValue, smoothedValue))
		s.LastValue = value
	}

	return value
}

// Start 启动工作协程池
func (wp *WorkerPool) Start() error {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	if wp.isRunning {
		return fmt.Errorf("worker pool is already running")
	}

	wp.isRunning = true

	// 启动工作协程
	for i := 0; i < wp.WorkerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	return nil
}

// worker 工作协程
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case task := <-wp.TaskQueue:
			if err := task.Execute(); err != nil {
				fmt.Printf("Worker %d: task execution failed: %v\n", id, err)
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

// Stop 停止工作协程池
func (wp *WorkerPool) Stop() error {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	if !wp.isRunning {
		return nil
	}

	wp.cancel()
	wp.wg.Wait()

	wp.isRunning = false
	return nil
}

// Stop 停止设备模拟器
func (ds *DeviceSimulator) Stop() error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if !ds.isRunning {
		return nil
	}

	ds.cancel()
	ds.wg.Wait()

	if err := ds.workerPool.Stop(); err != nil {
		return fmt.Errorf("failed to stop worker pool: %w", err)
	}

	ds.isRunning = false
	return nil
}

// GetDeviceCount 获取设备数量
func (ds *DeviceSimulator) GetDeviceCount() int {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	return len(ds.devices)
}

// GetActiveDeviceCount 获取活跃设备数量
func (ds *DeviceSimulator) GetActiveDeviceCount() int {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	count := 0
	for _, device := range ds.devices {
		if device.IsActive {
			count++
		}
	}
	return count
}

// GetDeviceById 根据ID获取设备
func (ds *DeviceSimulator) GetDeviceById(deviceID string) (*SimulatedDevice, bool) {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	device, exists := ds.devices[deviceID]
	return device, exists
}

// IsRunning 检查模拟器是否运行中
func (ds *DeviceSimulator) IsRunning() bool {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	return ds.isRunning
}

// GetStats 获取设备模拟器统计信息
func (ds *DeviceSimulator) GetStats() map[string]interface{} {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	activeCount := 0
	inactiveCount := 0
	lastUpdateTimes := make([]time.Time, 0, len(ds.devices))

	for _, device := range ds.devices {
		if device.IsActive {
			activeCount++
		} else {
			inactiveCount++
		}
		lastUpdateTimes = append(lastUpdateTimes, device.LastUpdate)
	}

	// 计算平均更新间隔
	var avgUpdateInterval time.Duration
	if len(lastUpdateTimes) > 1 {
		totalInterval := time.Duration(0)
		for i := 1; i < len(lastUpdateTimes); i++ {
			interval := lastUpdateTimes[i].Sub(lastUpdateTimes[i-1])
			if interval > 0 {
				totalInterval += interval
			}
		}
		if totalInterval > 0 {
			avgUpdateInterval = totalInterval / time.Duration(len(lastUpdateTimes)-1)
		}
	}

	return map[string]interface{}{
		"total_devices":        len(ds.devices),
		"active_devices":       activeCount,
		"inactive_devices":     inactiveCount,
		"is_running":           ds.isRunning,
		"worker_pool_size":     ds.config.WorkerPoolSize,
		"queue_buffer_size":    ds.config.QueueBufferSize,
		"sample_interval_ms":   float64(ds.config.SampleInterval) / float64(time.Millisecond),
		"data_variation":       ds.config.DataVariation,
		"anomaly_rate":         ds.config.AnomalyRate,
		"trend_enabled":        ds.config.TrendEnabled,
		"trend_strength":       ds.config.TrendStrength,
		"avg_update_interval_ms": float64(avgUpdateInterval) / float64(time.Millisecond),
	}
}
