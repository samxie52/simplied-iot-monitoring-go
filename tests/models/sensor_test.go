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

func TestNewSensorData(t *testing.T) {
	sensorData := models.NewSensorData()

	assert.NotNil(t, sensorData)
	assert.NotNil(t, sensorData.Custom)
	assert.True(t, sensorData.LastUpdate > 0)
	assert.Equal(t, 0, sensorData.GetSensorCount())
}

func TestSensorDataMethods(t *testing.T) {
	sensorData := models.NewSensorData()

	// 添加传感器数据
	sensorData.Temperature = models.NewTemperatureSensor(25.6)
	sensorData.Humidity = models.NewHumiditySensor(65.2)
	sensorData.Pressure = models.NewPressureSensor(1013.25)
	sensorData.Switch = models.NewSwitchSensor(true)
	sensorData.Current = models.NewCurrentSensor(2.5)

	// 测试检查方法
	assert.True(t, sensorData.HasTemperature())
	assert.True(t, sensorData.HasHumidity())
	assert.True(t, sensorData.HasPressure())
	assert.True(t, sensorData.HasSwitch())
	assert.True(t, sensorData.HasCurrent())
	assert.Equal(t, 5, sensorData.GetSensorCount())

	// 测试更新时间
	originalTime := sensorData.LastUpdate
	time.Sleep(10 * time.Millisecond)
	sensorData.UpdateLastUpdate()
	assert.True(t, sensorData.LastUpdate > originalTime)
}

func TestTemperatureSensor(t *testing.T) {
	tests := []struct {
		name           string
		value          float64
		expectedValue  float64
		expectedStatus models.SensorStatus
	}{
		{"Normal temperature", 25.6, 25.6, models.SensorStatusNormal},
		{"High temperature warning", 85.0, 85.0, models.SensorStatusWarning},
		{"Very high temperature error", 125.0, 125.0, models.SensorStatusError},
		{"Low temperature warning", -25.0, -25.0, models.SensorStatusWarning},
		{"Very low temperature error", -45.0, -45.0, models.SensorStatusError},
		{"Precision test", 25.67, 25.7, models.SensorStatusNormal}, // 测试精度
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sensor := models.NewTemperatureSensor(tt.value)

			assert.Equal(t, tt.expectedValue, sensor.Value)
			assert.Equal(t, "°C", sensor.Unit)
			assert.Equal(t, tt.expectedStatus, sensor.Status)
			assert.Equal(t, tt.expectedStatus == models.SensorStatusNormal, sensor.IsNormal())
		})
	}
}

func TestHumiditySensor(t *testing.T) {
	tests := []struct {
		name           string
		value          float64
		expectedValue  float64
		expectedStatus models.SensorStatus
	}{
		{"Normal humidity", 65.2, 65.2, models.SensorStatusNormal},
		{"High humidity warning", 95.0, 95.0, models.SensorStatusWarning},
		{"Low humidity warning", 5.0, 5.0, models.SensorStatusWarning},
		{"Invalid high humidity", 105.0, 105.0, models.SensorStatusError},
		{"Invalid low humidity", -5.0, -5.0, models.SensorStatusError},
		{"Precision test", 65.27, 65.3, models.SensorStatusNormal}, // 测试精度
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sensor := models.NewHumiditySensor(tt.value)

			assert.Equal(t, tt.expectedValue, sensor.Value)
			assert.Equal(t, "%", sensor.Unit)
			assert.Equal(t, tt.expectedStatus, sensor.Status)
			assert.Equal(t, tt.expectedStatus == models.SensorStatusNormal, sensor.IsNormal())
		})
	}
}

func TestPressureSensor(t *testing.T) {
	tests := []struct {
		name           string
		value          float64
		expectedValue  float64
		expectedStatus models.SensorStatus
	}{
		{"Normal pressure", 1013.25, 1013.25, models.SensorStatusNormal},
		{"High pressure warning", 1150.0, 1150.0, models.SensorStatusWarning},
		{"Low pressure warning", 850.0, 850.0, models.SensorStatusWarning},
		{"Invalid high pressure", 1250.0, 1250.0, models.SensorStatusError},
		{"Invalid low pressure", 750.0, 750.0, models.SensorStatusError},
		{"Precision test", 1013.256, 1013.26, models.SensorStatusNormal}, // 测试精度
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sensor := models.NewPressureSensor(tt.value)

			assert.Equal(t, tt.expectedValue, sensor.Value)
			assert.Equal(t, "hPa", sensor.Unit)
			assert.Equal(t, tt.expectedStatus, sensor.Status)
			assert.Equal(t, tt.expectedStatus == models.SensorStatusNormal, sensor.IsNormal())
		})
	}
}

func TestSwitchSensor(t *testing.T) {
	tests := []struct {
		name           string
		value          bool
		expectedStatus models.SwitchStatus
	}{
		{"Switch on", true, models.SwitchStatusOn},
		{"Switch off", false, models.SwitchStatusOff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sensor := models.NewSwitchSensor(tt.value)

			assert.Equal(t, tt.value, sensor.Value)
			assert.Equal(t, tt.expectedStatus, sensor.Status)

			if tt.value {
				assert.True(t, sensor.IsOn())
				assert.False(t, sensor.IsOff())
			} else {
				assert.False(t, sensor.IsOn())
				assert.True(t, sensor.IsOff())
			}
			assert.False(t, sensor.HasError())
		})
	}

	// 测试设置值
	sensor := models.NewSwitchSensor(false)
	sensor.SetValue(true)
	assert.True(t, sensor.Value)
	assert.Equal(t, models.SwitchStatusOn, sensor.Status)
	assert.True(t, sensor.IsOn())
}

func TestCurrentSensor(t *testing.T) {
	tests := []struct {
		name           string
		value          float64
		expectedValue  float64
		expectedStatus models.SensorStatus
	}{
		{"Normal current", 2.5, 2.5, models.SensorStatusNormal},
		{"High current warning", 85.0, 85.0, models.SensorStatusWarning},
		{"Invalid high current", 105.0, 105.0, models.SensorStatusError},
		{"Invalid negative current", -5.0, -5.0, models.SensorStatusError},
		{"Precision test", 2.567, 2.57, models.SensorStatusNormal}, // 测试精度
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sensor := models.NewCurrentSensor(tt.value)

			assert.Equal(t, tt.expectedValue, sensor.Value)
			assert.Equal(t, "A", sensor.Unit)
			assert.Equal(t, tt.expectedStatus, sensor.Status)
			assert.Equal(t, tt.expectedStatus == models.SensorStatusNormal, sensor.IsNormal())
		})
	}
}

func TestSensorDataValidation(t *testing.T) {
	tests := []struct {
		name          string
		setupSensor   func() *models.SensorData
		expectValid   bool
		expectedError string
	}{
		{
			name: "Valid sensor data",
			setupSensor: func() *models.SensorData {
				return createValidSensorData()
			},
			expectValid: true,
		},
		{
			name: "Empty sensor data",
			setupSensor: func() *models.SensorData {
				return models.NewSensorData()
			},
			expectValid:   false,
			expectedError: "sensors",
		},
		{
			name: "Future last update",
			setupSensor: func() *models.SensorData {
				sensorData := createValidSensorData()
				sensorData.LastUpdate = time.Now().UnixMilli() + 120000 // 2分钟后
				return sensorData
			},
			expectValid:   false,
			expectedError: "last_update",
		},
		{
			name: "Invalid temperature range",
			setupSensor: func() *models.SensorData {
				sensorData := createValidSensorData()
				sensorData.Temperature.Min = 30.0
				sensorData.Temperature.Max = 20.0 // Max < Min
				return sensorData
			},
			expectValid:   false,
			expectedError: "range",
		},
		{
			name: "Switch value status inconsistency",
			setupSensor: func() *models.SensorData {
				sensorData := createValidSensorData()
				sensorData.Switch.Value = true
				sensorData.Switch.Status = models.SwitchStatusOff // 不一致
				return sensorData
			},
			expectValid:   false,
			expectedError: "consistency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sensorData := tt.setupSensor()
			result := sensorData.Validate()

			if tt.expectValid {
				assert.True(t, result.IsValid, "Expected sensor data to be valid")
				assert.Empty(t, result.Errors, "Expected no validation errors")
			} else {
				assert.False(t, result.IsValid, "Expected sensor data to be invalid")
				assert.NotEmpty(t, result.Errors, "Expected validation errors")

				if tt.expectedError != "" {
					found := false
					for _, err := range result.Errors {
						if err.Field == tt.expectedError ||
							err.Message == tt.expectedError ||
							err.Field == "SensorData."+tt.expectedError ||
							strings.Contains(err.Field, tt.expectedError) {
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

func TestSensorDataJSONSerialization(t *testing.T) {
	sensorData := createValidSensorData()

	// 测试序列化
	jsonData, err := sensorData.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// 验证JSON结构
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err)

	assert.NotNil(t, jsonMap["temperature"])
	assert.NotNil(t, jsonMap["humidity"])
	assert.NotNil(t, jsonMap["pressure"])
	assert.NotNil(t, jsonMap["switch_status"])
	assert.NotNil(t, jsonMap["current"])

	// 测试反序列化
	newSensorData := &models.SensorData{}
	err = newSensorData.FromJSON(jsonData)
	require.NoError(t, err)

	assert.Equal(t, sensorData.Temperature.Value, newSensorData.Temperature.Value)
	assert.Equal(t, sensorData.Humidity.Value, newSensorData.Humidity.Value)
	assert.Equal(t, sensorData.Pressure.Value, newSensorData.Pressure.Value)
	assert.Equal(t, sensorData.Switch.Value, newSensorData.Switch.Value)
	assert.Equal(t, sensorData.Current.Value, newSensorData.Current.Value)
}

func TestSensorValueUpdates(t *testing.T) {
	// 测试温度传感器值更新
	tempSensor := models.NewTemperatureSensor(25.0)
	tempSensor.SetValue(85.0)
	assert.Equal(t, 85.0, tempSensor.Value)
	assert.Equal(t, models.SensorStatusWarning, tempSensor.Status)

	// 测试湿度传感器值更新
	humidSensor := models.NewHumiditySensor(50.0)
	humidSensor.SetValue(95.0)
	assert.Equal(t, 95.0, humidSensor.Value)
	assert.Equal(t, models.SensorStatusWarning, humidSensor.Status)

	// 测试压力传感器值更新
	pressureSensor := models.NewPressureSensor(1000.0)
	pressureSensor.SetValue(1150.0)
	assert.Equal(t, 1150.0, pressureSensor.Value)
	assert.Equal(t, models.SensorStatusWarning, pressureSensor.Status)

	// 测试电流传感器值更新
	currentSensor := models.NewCurrentSensor(5.0)
	currentSensor.SetValue(85.0)
	assert.Equal(t, 85.0, currentSensor.Value)
	assert.Equal(t, models.SensorStatusWarning, currentSensor.Status)
}

// 辅助函数：创建有效的传感器数据
func createValidSensorData() *models.SensorData {
	sensorData := models.NewSensorData()

	sensorData.Temperature = models.NewTemperatureSensor(25.6)
	sensorData.Temperature.Min = 20.0
	sensorData.Temperature.Max = 30.0

	sensorData.Humidity = models.NewHumiditySensor(65.2)
	sensorData.Humidity.Min = 40.0
	sensorData.Humidity.Max = 80.0

	sensorData.Pressure = models.NewPressureSensor(1013.25)
	sensorData.Pressure.Min = 1000.0
	sensorData.Pressure.Max = 1020.0

	sensorData.Switch = models.NewSwitchSensor(true)

	sensorData.Current = models.NewCurrentSensor(2.5)
	sensorData.Current.Min = 0.0
	sensorData.Current.Max = 10.0

	return sensorData
}

// 基准测试
func BenchmarkSensorDataValidation(b *testing.B) {
	sensorData := createValidSensorData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := sensorData.Validate()
		_ = result
	}
}

func BenchmarkTemperatureSensorCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sensor := models.NewTemperatureSensor(25.6)
		_ = sensor
	}
}

func BenchmarkSensorDataJSONSerialization(b *testing.B) {
	sensorData := createValidSensorData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := sensorData.ToJSON()
		if err != nil {
			b.Fatal(err)
		}
		_ = data
	}
}

func BenchmarkAllSensorCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sensorData := models.NewSensorData()
		sensorData.Temperature = models.NewTemperatureSensor(25.6)
		sensorData.Humidity = models.NewHumiditySensor(65.2)
		sensorData.Pressure = models.NewPressureSensor(1013.25)
		sensorData.Switch = models.NewSwitchSensor(true)
		sensorData.Current = models.NewCurrentSensor(2.5)
		_ = sensorData
	}
}
