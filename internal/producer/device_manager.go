package producer

import (
	"fmt"
	"sync"
	"time"

	"simplied-iot-monitoring-go/internal/models"
)

// DeviceManager 设备管理器
type DeviceManager struct {
	registry *DeviceRegistry
	factory  *DeviceFactory
	mutex    sync.RWMutex
}

// DeviceRegistry 设备注册表
type DeviceRegistry struct {
	devices map[string]*RegisteredDevice
	mutex   sync.RWMutex
}

// DeviceFactory 设备工厂
type DeviceFactory struct {
	templates map[string]*DeviceTemplate
	mutex     sync.RWMutex
}

// RegisteredDevice 注册的设备
type RegisteredDevice struct {
	Device      *SimulatedDevice
	Template    *DeviceTemplate
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Status      DeviceStatus
	Metadata    map[string]interface{}
}

// DeviceTemplate 设备模板
type DeviceTemplate struct {
	DeviceType    string
	SensorTypes   []string
	DefaultConfig map[string]interface{}
	Constraints   map[string]interface{}
}

// DeviceStatus 设备状态
type DeviceStatus string

const (
	DeviceStatusActive   DeviceStatus = "active"
	DeviceStatusInactive DeviceStatus = "inactive"
	DeviceStatusError    DeviceStatus = "error"
	DeviceStatusMaintenance DeviceStatus = "maintenance"
)

// NewDeviceManager 创建设备管理器
func NewDeviceManager() *DeviceManager {
	return &DeviceManager{
		registry: NewDeviceRegistry(),
		factory:  NewDeviceFactory(),
	}
}

// NewDeviceRegistry 创建设备注册表
func NewDeviceRegistry() *DeviceRegistry {
	return &DeviceRegistry{
		devices: make(map[string]*RegisteredDevice),
	}
}

// NewDeviceFactory 创建设备工厂
func NewDeviceFactory() *DeviceFactory {
	factory := &DeviceFactory{
		templates: make(map[string]*DeviceTemplate),
	}
	
	// 初始化默认设备模板
	factory.initializeDefaultTemplates()
	
	return factory
}

// initializeDefaultTemplates 初始化默认设备模板
func (df *DeviceFactory) initializeDefaultTemplates() {
	// 温度传感器设备模板
	df.templates["temperature"] = &DeviceTemplate{
		DeviceType:  "temperature",
		SensorTypes: []string{"temperature"},
		DefaultConfig: map[string]interface{}{
			"base_value":      22.0,
			"variance":        2.0,
			"min_value":       -40.0,
			"max_value":       80.0,
			"unit":           "°C",
			"sample_rate":    5.0,
		},
		Constraints: map[string]interface{}{
			"min_sample_rate": 1.0,
			"max_sample_rate": 60.0,
		},
	}

	// 湿度传感器设备模板
	df.templates["humidity"] = &DeviceTemplate{
		DeviceType:  "humidity",
		SensorTypes: []string{"humidity"},
		DefaultConfig: map[string]interface{}{
			"base_value":      50.0,
			"variance":        10.0,
			"min_value":       0.0,
			"max_value":       100.0,
			"unit":           "%",
			"sample_rate":    5.0,
		},
		Constraints: map[string]interface{}{
			"min_sample_rate": 1.0,
			"max_sample_rate": 60.0,
		},
	}

	// 压力传感器设备模板
	df.templates["pressure"] = &DeviceTemplate{
		DeviceType:  "pressure",
		SensorTypes: []string{"pressure"},
		DefaultConfig: map[string]interface{}{
			"base_value":      1013.25,
			"variance":        50.0,
			"min_value":       800.0,
			"max_value":       1200.0,
			"unit":           "hPa",
			"sample_rate":    5.0,
		},
		Constraints: map[string]interface{}{
			"min_sample_rate": 1.0,
			"max_sample_rate": 60.0,
		},
	}

	// 开关设备模板
	df.templates["switch"] = &DeviceTemplate{
		DeviceType:  "switch",
		SensorTypes: []string{"switch"},
		DefaultConfig: map[string]interface{}{
			"base_value":      0.0,
			"variance":        0.0,
			"min_value":       0.0,
			"max_value":       1.0,
			"unit":           "",
			"sample_rate":    10.0,
		},
		Constraints: map[string]interface{}{
			"min_sample_rate": 1.0,
			"max_sample_rate": 300.0,
		},
	}

	// 电流传感器设备模板
	df.templates["current"] = &DeviceTemplate{
		DeviceType:  "current",
		SensorTypes: []string{"current"},
		DefaultConfig: map[string]interface{}{
			"base_value":      5.0,
			"variance":        1.0,
			"min_value":       0.0,
			"max_value":       20.0,
			"unit":           "A",
			"sample_rate":    5.0,
		},
		Constraints: map[string]interface{}{
			"min_sample_rate": 1.0,
			"max_sample_rate": 60.0,
		},
	}

	// 多传感器设备模板
	df.templates["multi_sensor"] = &DeviceTemplate{
		DeviceType:  "multi_sensor",
		SensorTypes: []string{"temperature", "humidity", "pressure"},
		DefaultConfig: map[string]interface{}{
			"sample_rate": 5.0,
		},
		Constraints: map[string]interface{}{
			"min_sample_rate": 1.0,
			"max_sample_rate": 60.0,
		},
	}
}

// CreateDevice 创建设备
func (dm *DeviceManager) CreateDevice(deviceID, deviceType string, location models.LocationInfo, config map[string]interface{}) (*SimulatedDevice, error) {
	device, err := dm.factory.CreateDevice(deviceID, deviceType, location, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	// 注册设备
	if err := dm.registry.RegisterDevice(device, dm.factory.GetTemplate(deviceType)); err != nil {
		return nil, fmt.Errorf("failed to register device: %w", err)
	}

	return device, nil
}

// CreateDevice 工厂创建设备
func (df *DeviceFactory) CreateDevice(deviceID, deviceType string, location models.LocationInfo, config map[string]interface{}) (*SimulatedDevice, error) {
	df.mutex.RLock()
	template, exists := df.templates[deviceType]
	df.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown device type: %s", deviceType)
	}

	// 合并配置
	finalConfig := make(map[string]interface{})
	for k, v := range template.DefaultConfig {
		finalConfig[k] = v
	}
	for k, v := range config {
		finalConfig[k] = v
	}

	// 验证配置
	if err := df.validateConfig(template, finalConfig); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// 创建设备
	device := &SimulatedDevice{
		DeviceID:   deviceID,
		DeviceType: deviceType,
		Location:   location,
		Sensors:    make(map[string]*SensorSimulator),
		LastUpdate: time.Now(),
		IsActive:   true,
	}

	// 为设备创建传感器
	for _, sensorType := range template.SensorTypes {
		sensor := df.createSensor(sensorType, finalConfig)
		device.Sensors[sensorType] = sensor
	}

	return device, nil
}

// createSensor 创建传感器
func (df *DeviceFactory) createSensor(sensorType string, config map[string]interface{}) *SensorSimulator {
	baseValue, _ := config["base_value"].(float64)
	variance, _ := config["variance"].(float64)
	minValue, _ := config["min_value"].(float64)
	maxValue, _ := config["max_value"].(float64)
	unit, _ := config["unit"].(string)

	return &SensorSimulator{
		SensorType:     sensorType,
		BaseValue:      baseValue,
		Variance:       variance,
		TrendFactor:    0.1, // 默认趋势因子
		AnomalyRate:    0.01, // 默认异常率
		LastValue:      baseValue,
		TrendDirection: 1,
		Random:         nil, // 将在使用时初始化
		Unit:           unit,
		MinValue:       minValue,
		MaxValue:       maxValue,
	}
}

// validateConfig 验证配置
func (df *DeviceFactory) validateConfig(template *DeviceTemplate, config map[string]interface{}) error {
	// 检查采样率约束
	if sampleRate, ok := config["sample_rate"].(float64); ok {
		if minRate, ok := template.Constraints["min_sample_rate"].(float64); ok && sampleRate < minRate {
			return fmt.Errorf("sample_rate %f is below minimum %f", sampleRate, minRate)
		}
		if maxRate, ok := template.Constraints["max_sample_rate"].(float64); ok && sampleRate > maxRate {
			return fmt.Errorf("sample_rate %f is above maximum %f", sampleRate, maxRate)
		}
	}

	return nil
}

// RegisterDevice 注册设备
func (dr *DeviceRegistry) RegisterDevice(device *SimulatedDevice, template *DeviceTemplate) error {
	dr.mutex.Lock()
	defer dr.mutex.Unlock()

	if _, exists := dr.devices[device.DeviceID]; exists {
		return fmt.Errorf("device %s already registered", device.DeviceID)
	}

	registeredDevice := &RegisteredDevice{
		Device:    device,
		Template:  template,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    DeviceStatusActive,
		Metadata:  make(map[string]interface{}),
	}

	dr.devices[device.DeviceID] = registeredDevice
	return nil
}

// GetDevice 获取设备
func (dm *DeviceManager) GetDevice(deviceID string) (*SimulatedDevice, error) {
	return dm.registry.GetDevice(deviceID)
}

// GetDevice 从注册表获取设备
func (dr *DeviceRegistry) GetDevice(deviceID string) (*SimulatedDevice, error) {
	dr.mutex.RLock()
	defer dr.mutex.RUnlock()

	registeredDevice, exists := dr.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device %s not found", deviceID)
	}

	return registeredDevice.Device, nil
}

// ListDevices 列出所有设备
func (dm *DeviceManager) ListDevices() map[string]*SimulatedDevice {
	return dm.registry.ListDevices()
}

// ListDevices 从注册表列出设备
func (dr *DeviceRegistry) ListDevices() map[string]*SimulatedDevice {
	dr.mutex.RLock()
	defer dr.mutex.RUnlock()

	devices := make(map[string]*SimulatedDevice)
	for deviceID, registeredDevice := range dr.devices {
		devices[deviceID] = registeredDevice.Device
	}

	return devices
}

// UpdateDeviceStatus 更新设备状态
func (dm *DeviceManager) UpdateDeviceStatus(deviceID string, status DeviceStatus) error {
	return dm.registry.UpdateDeviceStatus(deviceID, status)
}

// UpdateDeviceStatus 在注册表中更新设备状态
func (dr *DeviceRegistry) UpdateDeviceStatus(deviceID string, status DeviceStatus) error {
	dr.mutex.Lock()
	defer dr.mutex.Unlock()

	registeredDevice, exists := dr.devices[deviceID]
	if !exists {
		return fmt.Errorf("device %s not found", deviceID)
	}

	registeredDevice.Status = status
	registeredDevice.UpdatedAt = time.Now()

	// 同步更新设备的活跃状态
	registeredDevice.Device.IsActive = (status == DeviceStatusActive)

	return nil
}

// RemoveDevice 移除设备
func (dm *DeviceManager) RemoveDevice(deviceID string) error {
	return dm.registry.RemoveDevice(deviceID)
}

// RemoveDevice 从注册表移除设备
func (dr *DeviceRegistry) RemoveDevice(deviceID string) error {
	dr.mutex.Lock()
	defer dr.mutex.Unlock()

	if _, exists := dr.devices[deviceID]; !exists {
		return fmt.Errorf("device %s not found", deviceID)
	}

	delete(dr.devices, deviceID)
	return nil
}

// GetDeviceCount 获取设备总数
func (dm *DeviceManager) GetDeviceCount() int {
	return dm.registry.GetDeviceCount()
}

// GetDeviceCount 从注册表获取设备总数
func (dr *DeviceRegistry) GetDeviceCount() int {
	dr.mutex.RLock()
	defer dr.mutex.RUnlock()
	return len(dr.devices)
}

// GetActiveDeviceCount 获取活跃设备数量
func (dm *DeviceManager) GetActiveDeviceCount() int {
	return dm.registry.GetActiveDeviceCount()
}

// GetActiveDeviceCount 从注册表获取活跃设备数量
func (dr *DeviceRegistry) GetActiveDeviceCount() int {
	dr.mutex.RLock()
	defer dr.mutex.RUnlock()

	count := 0
	for _, registeredDevice := range dr.devices {
		if registeredDevice.Status == DeviceStatusActive {
			count++
		}
	}
	return count
}

// GetTemplate 获取设备模板
func (df *DeviceFactory) GetTemplate(deviceType string) *DeviceTemplate {
	df.mutex.RLock()
	defer df.mutex.RUnlock()
	return df.templates[deviceType]
}

// AddTemplate 添加设备模板
func (df *DeviceFactory) AddTemplate(template *DeviceTemplate) {
	df.mutex.Lock()
	defer df.mutex.Unlock()
	df.templates[template.DeviceType] = template
}

// ListTemplates 列出所有模板
func (df *DeviceFactory) ListTemplates() map[string]*DeviceTemplate {
	df.mutex.RLock()
	defer df.mutex.RUnlock()

	templates := make(map[string]*DeviceTemplate)
	for deviceType, template := range df.templates {
		templates[deviceType] = template
	}
	return templates
}

// GetDevicesByType 根据类型获取设备
func (dm *DeviceManager) GetDevicesByType(deviceType string) []*SimulatedDevice {
	return dm.registry.GetDevicesByType(deviceType)
}

// GetDevicesByType 从注册表根据类型获取设备
func (dr *DeviceRegistry) GetDevicesByType(deviceType string) []*SimulatedDevice {
	dr.mutex.RLock()
	defer dr.mutex.RUnlock()

	var devices []*SimulatedDevice
	for _, registeredDevice := range dr.devices {
		if registeredDevice.Device.DeviceType == deviceType {
			devices = append(devices, registeredDevice.Device)
		}
	}
	return devices
}

// GetDevicesByStatus 根据状态获取设备
func (dm *DeviceManager) GetDevicesByStatus(status DeviceStatus) []*SimulatedDevice {
	return dm.registry.GetDevicesByStatus(status)
}

// GetDevicesByStatus 从注册表根据状态获取设备
func (dr *DeviceRegistry) GetDevicesByStatus(status DeviceStatus) []*SimulatedDevice {
	dr.mutex.RLock()
	defer dr.mutex.RUnlock()

	var devices []*SimulatedDevice
	for _, registeredDevice := range dr.devices {
		if registeredDevice.Status == status {
			devices = append(devices, registeredDevice.Device)
		}
	}
	return devices
}
