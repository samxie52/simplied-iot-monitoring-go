package unit

import (
	"testing"
	"time"
)

// TestDeviceDataGeneration 测试设备数据生成
func TestDeviceDataGeneration(t *testing.T) {
	// 模拟设备数据结构
	type IoTDevice struct {
		DeviceID     string    `json:"device_id"`
		DeviceType   string    `json:"device_type"`
		Location     string    `json:"location"`
		Timestamp    time.Time `json:"timestamp"`
		Temperature  float64   `json:"temperature"`
		Humidity     float64   `json:"humidity"`
		Pressure     float64   `json:"pressure"`
		Status       string    `json:"status"`
		BatteryLevel float64   `json:"battery_level"`
	}

	// 测试设备数据生成
	device := IoTDevice{
		DeviceID:     "test_device_001",
		DeviceType:   "温度传感器",
		Location:     "车间A",
		Timestamp:    time.Now(),
		Temperature:  25.5,
		Humidity:     60.0,
		Pressure:     1013.25,
		Status:       "正常",
		BatteryLevel: 85.0,
	}

	// 验证设备数据
	if device.DeviceID == "" {
		t.Error("设备ID不能为空")
	}

	if device.Temperature < -50 || device.Temperature > 100 {
		t.Errorf("温度值异常: %.2f°C", device.Temperature)
	}

	if device.Humidity < 0 || device.Humidity > 100 {
		t.Errorf("湿度值异常: %.2f%%", device.Humidity)
	}

	if device.BatteryLevel < 0 || device.BatteryLevel > 100 {
		t.Errorf("电池电量异常: %.2f%%", device.BatteryLevel)
	}

	t.Logf("设备数据验证通过: %+v", device)
}

// TestDataValidation 测试数据验证逻辑
func TestDataValidation(t *testing.T) {
	testCases := []struct {
		name        string
		temperature float64
		humidity    float64
		battery     float64
		expectError bool
	}{
		{"正常数据", 25.0, 50.0, 80.0, false},
		{"温度过高", 150.0, 50.0, 80.0, true},
		{"温度过低", -100.0, 50.0, 80.0, true},
		{"湿度过高", 25.0, 150.0, 80.0, true},
		{"湿度过低", 25.0, -10.0, 80.0, true},
		{"电池电量过低", 25.0, 50.0, -5.0, true},
		{"电池电量过高", 25.0, 50.0, 150.0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasError := false

			if tc.temperature < -50 || tc.temperature > 100 {
				hasError = true
			}
			if tc.humidity < 0 || tc.humidity > 100 {
				hasError = true
			}
			if tc.battery < 0 || tc.battery > 100 {
				hasError = true
			}

			if hasError != tc.expectError {
				t.Errorf("期望错误: %v, 实际错误: %v", tc.expectError, hasError)
			}
		})
	}
}
