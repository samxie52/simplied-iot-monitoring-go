package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// Device 设备数据模型 - 核心结构体
type Device struct {
	// 设备标识信息
	DeviceID   string `json:"device_id" validate:"required"`
	DeviceType string `json:"device_type" validate:"required,oneof=sensor gateway controller"`

	// 时间戳信息
	Timestamp int64     `json:"timestamp" validate:"required"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 嵌套结构引用
	DeviceInfo DeviceInfo   `json:"device_info" validate:"required"`
	Location   LocationInfo `json:"location" validate:"required"`
	SensorData SensorData   `json:"sensors" validate:"required"`

	// 设备状态
	Status   DeviceStatus `json:"status" validate:"required,oneof=online offline error maintenance"`
	LastSeen int64        `json:"last_seen" validate:"min=0"`
}

// DeviceInfo 设备详细信息 - 硬件和固件信息
type DeviceInfo struct {
	Model           string `json:"model" validate:"required"`
	Manufacturer    string `json:"manufacturer" validate:"required"`
	FirmwareVersion string `json:"firmware" validate:"required"`
	HardwareVersion string `json:"hardware" validate:"required"`
	SerialNumber    string `json:"serial_number" validate:"required"`

	// 设备能力信息
	BatteryLevel   int         `json:"battery" validate:"min=0,max=100"`
	SignalStrength int         `json:"signal_strength" validate:"min=-120,max=0"`
	NetworkType    NetworkType `json:"network_type" validate:"required,oneof=wifi ethernet lora nb-iot"`
}

// LocationInfo 层次化位置信息 - 建筑物定位
type LocationInfo struct {
	Building string `json:"building" validate:"required"`
	Floor    int    `json:"floor"`
	Room     string `json:"room" validate:"required"`
	Zone     string `json:"zone,omitempty" validate:"max=32"`

	// 地理坐标(可选)
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Altitude  float64 `json:"altitude,omitempty"`
}

// NewDevice 创建新设备
func NewDevice(deviceID, deviceType string) *Device {
	now := time.Now()
	return &Device{
		DeviceID:   deviceID,
		DeviceType: deviceType,
		Timestamp:  now.UnixMilli(),
		CreatedAt:  now,
		UpdatedAt:  now,
		Status:     DeviceStatusOffline,
		LastSeen:   now.UnixMilli(),
	}
}

// UpdateTimestamp 更新时间戳
func (d *Device) UpdateTimestamp() {
	now := time.Now()
	d.Timestamp = now.UnixMilli()
	d.UpdatedAt = now
	d.LastSeen = now.UnixMilli()
}

// SetOnline 设置设备在线
func (d *Device) SetOnline() {
	d.Status = DeviceStatusOnline
	d.UpdateTimestamp()
}

// SetOffline 设置设备离线
func (d *Device) SetOffline() {
	d.Status = DeviceStatusOffline
	d.UpdateTimestamp()
}

// SetError 设置设备错误状态
func (d *Device) SetError() {
	d.Status = DeviceStatusError
	d.UpdateTimestamp()
}

// IsOnline 检查设备是否在线
func (d *Device) IsOnline() bool {
	return d.Status == DeviceStatusOnline
}

// IsOffline 检查设备是否离线
func (d *Device) IsOffline() bool {
	return d.Status == DeviceStatusOffline
}

// HasError 检查设备是否有错误
func (d *Device) HasError() bool {
	return d.Status == DeviceStatusError
}

// GetAge 获取设备数据年龄(毫秒)
func (d *Device) GetAge() int64 {
	return time.Now().UnixMilli() - d.Timestamp
}

// IsStale 检查数据是否过期(超过指定毫秒数)
func (d *Device) IsStale(maxAgeMs int64) bool {
	return d.GetAge() > maxAgeMs
}

// ToJSON 转换为JSON字符串
func (d *Device) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}

// FromJSON 从JSON字符串解析
func (d *Device) FromJSON(data []byte) error {
	return json.Unmarshal(data, d)
}

// Validate 验证设备数据
func (d *Device) Validate() *ValidationResult {
	result := NewValidationResult()
	validator := validator.New()

	// 结构体验证
	if err := validator.Struct(d); err != nil {
		validationErr := NewValidationError(
			"struct",
			"device",
			fmt.Sprintf("Validation failed: %s", err.Error()),
			d,
		)
		result.AddError(*validationErr)
	}

	// 业务验证
	d.validateBusinessRules(result)

	// 设置处理完成时间
	result.ProcessedAt = time.Now().UnixMilli()
	return result
}

// validateBusinessRules 业务规则验证
func (d *Device) validateBusinessRules(result *ValidationResult) {
	// 验证设备ID不能为空
	if d.DeviceID == "" {
		result.AddError(*NewValidationError("business", "device_id", "Device ID cannot be empty", d.DeviceID))
	}

	// 验证时间戳合理性
	now := time.Now().UnixMilli()
	if d.Timestamp > now+60000 { // 不能超过当前时间1分钟
		result.AddError(*NewValidationError("business", "timestamp", "Timestamp cannot be in the future", d.Timestamp))
	}

	// 验证LastSeen不能晚于Timestamp
	if d.LastSeen > d.Timestamp {
		result.AddError(*NewValidationError("business", "last_seen", "LastSeen cannot be later than Timestamp", d.LastSeen))
	}

	// 验证设备信息
	if d.DeviceInfo.BatteryLevel < 0 || d.DeviceInfo.BatteryLevel > 100 {
		result.AddError(*NewValidationError("business", "battery_level", "Battery level must be between 0 and 100", d.DeviceInfo.BatteryLevel))
	}

	// 验证信号强度
	if d.DeviceInfo.SignalStrength < -120 || d.DeviceInfo.SignalStrength > 0 {
		result.AddError(*NewValidationError("business", "signal_strength", "Signal strength must be between -120 and 0 dBm", d.DeviceInfo.SignalStrength))
	}

	// 验证位置信息
	if d.Location.Floor < -10 || d.Location.Floor > 100 {
		result.AddError(*NewValidationError("business", "floor", "Floor must be between -10 and 100", d.Location.Floor))
	}

	// 验证地理坐标
	if d.Location.Latitude != 0 && (d.Location.Latitude < -90 || d.Location.Latitude > 90) {
		result.AddError(*NewValidationError("business", "latitude", "Latitude must be between -90 and 90", d.Location.Latitude))
	}

	if d.Location.Longitude != 0 && (d.Location.Longitude < -180 || d.Location.Longitude > 180) {
		result.AddError(*NewValidationError("business", "longitude", "Longitude must be between -180 and 180", d.Location.Longitude))
	}
}

// Clone 克隆设备对象
func (d *Device) Clone() *Device {
	clone := *d
	return &clone
}

// GetLocationString 获取位置字符串描述
func (d *Device) GetLocationString() string {
	if d.Location.Zone != "" {
		return fmt.Sprintf("%s-%d楼-%s-%s", d.Location.Building, d.Location.Floor, d.Location.Room, d.Location.Zone)
	}
	return fmt.Sprintf("%s-%d楼-%s", d.Location.Building, d.Location.Floor, d.Location.Room)
}

// GetDeviceInfoString 获取设备信息字符串描述
func (d *Device) GetDeviceInfoString() string {
	return fmt.Sprintf("%s %s (FW:%s, HW:%s, SN:%s)",
		d.DeviceInfo.Manufacturer,
		d.DeviceInfo.Model,
		d.DeviceInfo.FirmwareVersion,
		d.DeviceInfo.HardwareVersion,
		d.DeviceInfo.SerialNumber)
}

// GetBatteryStatus 获取电池状态描述
func (d *Device) GetBatteryStatus() string {
	battery := d.DeviceInfo.BatteryLevel
	switch {
	case battery >= 80:
		return "excellent"
	case battery >= 60:
		return "good"
	case battery >= 40:
		return "fair"
	case battery >= 20:
		return "low"
	default:
		return "critical"
	}
}

// GetSignalQuality 获取信号质量描述
func (d *Device) GetSignalQuality() string {
	signal := d.DeviceInfo.SignalStrength
	switch {
	case signal >= -30:
		return "excellent"
	case signal >= -50:
		return "good"
	case signal >= -70:
		return "fair"
	case signal >= -90:
		return "poor"
	default:
		return "very_poor"
	}
}
