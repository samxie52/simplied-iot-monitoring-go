package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simplied-iot-monitoring-go/internal/models"
)

func TestNewDevice(t *testing.T) {
	deviceID := "test_device_001"
	deviceType := "sensor"

	device := models.NewDevice(deviceID, deviceType)

	assert.Equal(t, deviceID, device.DeviceID)
	assert.Equal(t, deviceType, device.DeviceType)
	assert.Equal(t, models.DeviceStatusOffline, device.Status)
	assert.True(t, device.Timestamp > 0)
	assert.True(t, device.LastSeen > 0)
	assert.False(t, device.CreatedAt.IsZero())
	assert.False(t, device.UpdatedAt.IsZero())
}

func TestDeviceStatusMethods(t *testing.T) {
	device := models.NewDevice("test_device", "sensor")

	// 测试设置在线状态
	device.SetOnline()
	assert.Equal(t, models.DeviceStatusOnline, device.Status)
	assert.True(t, device.IsOnline())
	assert.False(t, device.IsOffline())
	assert.False(t, device.HasError())

	// 测试设置离线状态
	device.SetOffline()
	assert.Equal(t, models.DeviceStatusOffline, device.Status)
	assert.False(t, device.IsOnline())
	assert.True(t, device.IsOffline())
	assert.False(t, device.HasError())

	// 测试设置错误状态
	device.SetError()
	assert.Equal(t, models.DeviceStatusError, device.Status)
	assert.False(t, device.IsOnline())
	assert.False(t, device.IsOffline())
	assert.True(t, device.HasError())
}

func TestDeviceAge(t *testing.T) {
	device := models.NewDevice("test_device", "sensor")

	// 等待一小段时间
	time.Sleep(10 * time.Millisecond)

	age := device.GetAge()
	assert.True(t, age > 0)
	assert.True(t, age < 1000) // 应该小于1秒

	// 测试数据是否过期
	assert.False(t, device.IsStale(1000)) // 1秒内不过期
	assert.True(t, device.IsStale(1))     // 1毫秒内过期
}

func TestDeviceValidation(t *testing.T) {
	tests := []struct {
		name          string
		setupDevice   func() *models.Device
		expectValid   bool
		expectedError string
	}{
		{
			name: "Valid device",
			setupDevice: func() *models.Device {
				device := createValidDevice()
				return device
			},
			expectValid: true,
		},
		{
			name: "Empty device ID",
			setupDevice: func() *models.Device {
				device := createValidDevice()
				device.DeviceID = ""
				return device
			},
			expectValid:   false,
			expectedError: "device_id",
		},
		{
			name: "Invalid device type",
			setupDevice: func() *models.Device {
				device := createValidDevice()
				device.DeviceType = "invalid"
				return device
			},
			expectValid:   false,
			expectedError: "DeviceType",
		},
		{
			name: "Future timestamp",
			setupDevice: func() *models.Device {
				device := createValidDevice()
				device.Timestamp = time.Now().UnixMilli() + 120000 // 2分钟后
				return device
			},
			expectValid:   false,
			expectedError: "timestamp",
		},
		{
			name: "Invalid battery level",
			setupDevice: func() *models.Device {
				device := createValidDevice()
				device.DeviceInfo.BatteryLevel = 150
				return device
			},
			expectValid:   false,
			expectedError: "battery_level",
		},
		{
			name: "Invalid signal strength",
			setupDevice: func() *models.Device {
				device := createValidDevice()
				device.DeviceInfo.SignalStrength = 10
				return device
			},
			expectValid:   false,
			expectedError: "signal_strength",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := tt.setupDevice()
			result := device.Validate()

			if tt.expectValid {
				assert.True(t, result.IsValid, "Expected device to be valid")
				assert.Empty(t, result.Errors, "Expected no validation errors")
			} else {
				assert.False(t, result.IsValid, "Expected device to be invalid")
				assert.NotEmpty(t, result.Errors, "Expected validation errors")

				if tt.expectedError != "" {
					found := false
					for _, err := range result.Errors {
						if err.Field == tt.expectedError ||
							err.Field == "Device."+tt.expectedError ||
							err.Message == tt.expectedError ||
							strings.Contains(err.Field, tt.expectedError) ||
							strings.Contains(err.Message, tt.expectedError) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error field '%s' not found in errors: %+v", tt.expectedError, result.Errors)
				}
			}
		})
	}
}

func TestDeviceJSONSerialization(t *testing.T) {
	device := createValidDevice()

	// 测试序列化
	jsonData, err := device.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// 验证JSON结构
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err)

	assert.Equal(t, device.DeviceID, jsonMap["device_id"])
	assert.Equal(t, device.DeviceType, jsonMap["device_type"])
	assert.Equal(t, string(device.Status), jsonMap["status"])

	// 测试反序列化
	newDevice := &models.Device{}
	err = newDevice.FromJSON(jsonData)
	require.NoError(t, err)

	assert.Equal(t, device.DeviceID, newDevice.DeviceID)
	assert.Equal(t, device.DeviceType, newDevice.DeviceType)
	assert.Equal(t, device.Status, newDevice.Status)
	assert.Equal(t, device.DeviceInfo.Model, newDevice.DeviceInfo.Model)
	assert.Equal(t, device.Location.Building, newDevice.Location.Building)
}

func TestDeviceClone(t *testing.T) {
	device := createValidDevice()
	clone := device.Clone()

	// 验证克隆的内容相同
	assert.Equal(t, device.DeviceID, clone.DeviceID)
	assert.Equal(t, device.DeviceType, clone.DeviceType)
	assert.Equal(t, device.Status, clone.Status)

	// 验证是不同的对象
	assert.NotSame(t, device, clone)

	// 修改原对象不应该影响克隆
	device.DeviceID = "modified"
	assert.NotEqual(t, device.DeviceID, clone.DeviceID)
}

func TestDeviceStringMethods(t *testing.T) {
	device := createValidDevice()

	// 测试位置字符串
	locationStr := device.GetLocationString()
	assert.Contains(t, locationStr, device.Location.Building)
	assert.Contains(t, locationStr, device.Location.Room)

	// 测试设备信息字符串
	deviceInfoStr := device.GetDeviceInfoString()
	assert.Contains(t, deviceInfoStr, device.DeviceInfo.Manufacturer)
	assert.Contains(t, deviceInfoStr, device.DeviceInfo.Model)

	// 测试电池状态
	batteryStatus := device.GetBatteryStatus()
	assert.NotEmpty(t, batteryStatus)
	assert.Contains(t, []string{"excellent", "good", "fair", "low", "critical"}, batteryStatus)

	// 测试信号质量
	signalQuality := device.GetSignalQuality()
	assert.NotEmpty(t, signalQuality)
	assert.Contains(t, []string{"excellent", "good", "fair", "poor", "very_poor"}, signalQuality)
}

func TestDeviceUpdateTimestamp(t *testing.T) {
	device := createValidDevice()
	originalTimestamp := device.Timestamp
	originalLastSeen := device.LastSeen

	// 等待一小段时间
	time.Sleep(10 * time.Millisecond)

	device.UpdateTimestamp()

	assert.True(t, device.Timestamp > originalTimestamp)
	assert.True(t, device.LastSeen > originalLastSeen)
	assert.True(t, device.UpdatedAt.After(device.CreatedAt))
}

// 辅助函数：创建有效的设备对象
func createValidDevice() *models.Device {
	device := models.NewDevice("test_device_001", "sensor")

	device.DeviceInfo = models.DeviceInfo{
		Model:           "IOT-SENSOR-V2",
		Manufacturer:    "TechCorp",
		FirmwareVersion: "1.2.3",
		HardwareVersion: "2.1.0",
		SerialNumber:    "SN123456789",
		BatteryLevel:    85,
		SignalStrength:  -45,
		NetworkType:     models.NetworkTypeWiFi,
	}

	device.Location = models.LocationInfo{
		Building:  "A栋",
		Floor:     3,
		Room:      "301",
		Zone:      "生产区域",
		Latitude:  39.9042,
		Longitude: 116.4074,
	}

	device.SensorData = models.SensorData{
		Temperature: models.NewTemperatureSensor(25.6),
		Humidity:    models.NewHumiditySensor(65.2),
		Pressure:    models.NewPressureSensor(1013.25),
		Switch:      models.NewSwitchSensor(true),
		Current:     models.NewCurrentSensor(2.5),
		LastUpdate:  time.Now().UnixMilli(),
	}

	device.SetOnline()
	return device
}

// 基准测试
func BenchmarkDeviceValidation(b *testing.B) {
	device := createValidDevice()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := device.Validate()
		_ = result
	}
}

func BenchmarkDeviceJSONSerialization(b *testing.B) {
	device := createValidDevice()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := device.ToJSON()
		if err != nil {
			b.Fatal(err)
		}
		_ = data
	}
}

func BenchmarkDeviceJSONDeserialization(b *testing.B) {
	device := createValidDevice()
	data, err := device.ToJSON()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newDevice := &models.Device{}
		err := newDevice.FromJSON(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
