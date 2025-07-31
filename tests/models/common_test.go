package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"simplied-iot-monitoring-go/internal/models"
)

func TestTimestamp(t *testing.T) {
	now := time.Now()
	timestamp := models.NewTimestamp(now)

	assert.Equal(t, now.UnixMilli(), timestamp.Unix)
	assert.Equal(t, now.Format(time.RFC3339), timestamp.RFC3339)
	assert.Equal(t, now.Location().String(), timestamp.Timezone)
	assert.Equal(t, now.Unix(), timestamp.LocalTime.Unix()) // 比较到秒级精度

	// 测试转换回time.Time
	convertedTime := timestamp.ToTime()
	assert.Equal(t, now.Unix(), convertedTime.Unix()) // 比较到秒级精度

	// 测试有效性验证
	assert.True(t, timestamp.IsValid())
}

func TestTimestampNow(t *testing.T) {
	timestamp := models.NewTimestampNow()

	assert.True(t, timestamp.Unix > 0)
	assert.NotEmpty(t, timestamp.RFC3339)
	assert.NotEmpty(t, timestamp.Timezone)
	assert.False(t, timestamp.LocalTime.IsZero())
	assert.True(t, timestamp.IsValid())
}

func TestTimestampValidation(t *testing.T) {
	tests := []struct {
		name      string
		timestamp *models.Timestamp
		expected  bool
	}{
		{
			name:      "Valid timestamp",
			timestamp: models.NewTimestampNow(),
			expected:  true,
		},
		{
			name: "Invalid unix timestamp (zero)",
			timestamp: &models.Timestamp{
				Unix:    0,
				RFC3339: time.Now().Format(time.RFC3339),
			},
			expected: false,
		},
		{
			name: "Invalid unix timestamp (negative)",
			timestamp: &models.Timestamp{
				Unix:    -1,
				RFC3339: time.Now().Format(time.RFC3339),
			},
			expected: false,
		},
		{
			name: "Empty RFC3339",
			timestamp: &models.Timestamp{
				Unix:    time.Now().UnixMilli(),
				RFC3339: "",
			},
			expected: false,
		},
		{
			name: "Invalid RFC3339 format",
			timestamp: &models.Timestamp{
				Unix:    time.Now().UnixMilli(),
				RFC3339: "invalid-format",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.timestamp.IsValid())
		})
	}
}

func TestValidationError(t *testing.T) {
	errorType := "business"
	field := "device_id"
	message := "Device ID cannot be empty"
	value := ""

	validationErr := models.NewValidationError(errorType, field, message, value)

	assert.Equal(t, "VALIDATION_business_device_id", validationErr.ErrorCode)
	assert.Equal(t, errorType, validationErr.ErrorType)
	assert.Equal(t, field, validationErr.Field)
	assert.Equal(t, message, validationErr.Message)
	assert.Equal(t, value, validationErr.Value)
	assert.True(t, validationErr.Timestamp > 0)
}

func TestValidationResult(t *testing.T) {
	result := models.NewValidationResult()

	// 初始状态
	assert.True(t, result.IsValid)
	assert.Empty(t, result.Errors)
	assert.Empty(t, result.Warnings)
	assert.False(t, result.HasErrors())
	assert.False(t, result.HasWarnings())
	assert.Equal(t, 0, result.ErrorCount())
	assert.Equal(t, 0, result.WarningCount())
	assert.True(t, result.ProcessedAt > 0)

	// 添加错误
	err1 := *models.NewValidationError("struct", "field1", "Error 1", nil)
	result.AddError(err1)

	assert.False(t, result.IsValid)
	assert.True(t, result.HasErrors())
	assert.Equal(t, 1, result.ErrorCount())
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, err1, result.Errors[0])

	// 添加警告
	warning1 := *models.NewValidationError("business", "field2", "Warning 1", nil)
	result.AddWarning(warning1)

	assert.True(t, result.HasWarnings())
	assert.Equal(t, 1, result.WarningCount())
	assert.Len(t, result.Warnings, 1)
	assert.Equal(t, warning1, result.Warnings[0])

	// 添加更多错误和警告
	err2 := *models.NewValidationError("system", "field3", "Error 2", nil)
	warning2 := *models.NewValidationError("business", "field4", "Warning 2", nil)

	result.AddError(err2)
	result.AddWarning(warning2)

	assert.Equal(t, 2, result.ErrorCount())
	assert.Equal(t, 2, result.WarningCount())
	assert.Len(t, result.Errors, 2)
	assert.Len(t, result.Warnings, 2)
}

func TestDeviceStatusConstants(t *testing.T) {
	assert.Equal(t, models.DeviceStatus("online"), models.DeviceStatusOnline)
	assert.Equal(t, models.DeviceStatus("offline"), models.DeviceStatusOffline)
	assert.Equal(t, models.DeviceStatus("error"), models.DeviceStatusError)
	assert.Equal(t, models.DeviceStatus("maintenance"), models.DeviceStatusMaint)
}

func TestSensorStatusConstants(t *testing.T) {
	assert.Equal(t, models.SensorStatus("normal"), models.SensorStatusNormal)
	assert.Equal(t, models.SensorStatus("warning"), models.SensorStatusWarning)
	assert.Equal(t, models.SensorStatus("error"), models.SensorStatusError)
	assert.Equal(t, models.SensorStatus("offline"), models.SensorStatusOffline)
}

func TestSwitchStatusConstants(t *testing.T) {
	assert.Equal(t, models.SwitchStatus("on"), models.SwitchStatusOn)
	assert.Equal(t, models.SwitchStatus("off"), models.SwitchStatusOff)
	assert.Equal(t, models.SwitchStatus("error"), models.SwitchStatusError)
}

func TestNetworkTypeConstants(t *testing.T) {
	assert.Equal(t, models.NetworkType("wifi"), models.NetworkTypeWiFi)
	assert.Equal(t, models.NetworkType("ethernet"), models.NetworkTypeEthernet)
	assert.Equal(t, models.NetworkType("lora"), models.NetworkTypeLoRa)
	assert.Equal(t, models.NetworkType("nb-iot"), models.NetworkTypeNBIoT)
}

func TestValidationResultConcurrency(t *testing.T) {
	result := models.NewValidationResult()

	// 并发添加错误和警告
	done := make(chan bool, 100)

	// 启动多个goroutine添加错误
	for i := 0; i < 50; i++ {
		go func(i int) {
			err := *models.NewValidationError("test", "field", "message", i)
			result.AddError(err)
			done <- true
		}(i)
	}

	// 启动多个goroutine添加警告
	for i := 0; i < 50; i++ {
		go func(i int) {
			warning := *models.NewValidationError("test", "field", "message", i)
			result.AddWarning(warning)
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 100; i++ {
		<-done
	}

	// 验证结果
	assert.Equal(t, 50, result.ErrorCount())
	assert.Equal(t, 50, result.WarningCount())
	assert.False(t, result.IsValid)
	assert.True(t, result.HasErrors())
	assert.True(t, result.HasWarnings())
}

// 基准测试
func BenchmarkNewTimestamp(b *testing.B) {
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timestamp := models.NewTimestamp(now)
		_ = timestamp
	}
}

func BenchmarkTimestampValidation(b *testing.B) {
	timestamp := models.NewTimestampNow()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valid := timestamp.IsValid()
		_ = valid
	}
}

func BenchmarkNewValidationError(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := models.NewValidationError("business", "field", "message", "value")
		_ = err
	}
}

func BenchmarkValidationResultAddError(b *testing.B) {
	result := models.NewValidationResult()
	err := *models.NewValidationError("test", "field", "message", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result.AddError(err)
	}
}

func BenchmarkValidationResultOperations(b *testing.B) {
	result := models.NewValidationResult()
	err := *models.NewValidationError("test", "field", "message", nil)
	warning := *models.NewValidationError("test", "field", "warning", nil)

	result.AddError(err)
	result.AddWarning(warning)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.HasErrors()
		_ = result.HasWarnings()
		_ = result.ErrorCount()
		_ = result.WarningCount()
		_ = result.IsValid
	}
}
