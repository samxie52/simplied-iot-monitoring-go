package models

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"simplied-iot-monitoring-go/internal/models"
)

// TestValidationEngine 测试验证引擎
func TestValidationEngine(t *testing.T) {
	engine := models.NewValidationEngine()
	assert.NotNil(t, engine)
}

// TestEnhancedValidationResult 测试增强验证结果
func TestEnhancedValidationResult(t *testing.T) {
	result := models.NewEnhancedValidationResult()
	assert.NotNil(t, result)
	assert.True(t, result.IsValid)
	assert.Empty(t, result.Errors)
	assert.Empty(t, result.Warnings)
	assert.Empty(t, result.Fixed)
}

// TestValidationEngineValidate 测试验证功能
func TestValidationEngineValidate(t *testing.T) {
	engine := models.NewValidationEngine()

	// 测试有效数据
	validDevice := &models.Device{
		DeviceID:   "device-001",
		DeviceType: "sensor",
		Status:     "online",
		Timestamp:  time.Now().Unix(),
		DeviceInfo: models.DeviceInfo{
			Model:           "TempSensor-v1",
			Manufacturer:    "IoTCorp",
			FirmwareVersion: "1.0.0",
			HardwareVersion: "1.0",
			SerialNumber:    "SN123456",
			BatteryLevel:    85,
			SignalStrength:  -45,
			NetworkType:     "wifi",
		},
		Location: models.LocationInfo{
			Building:  "Building A",
			Floor:     1,
			Room:      "Room 101",
			Zone:      "Zone A",
			Latitude:  39.9042,
			Longitude: 116.4074,
			Altitude:  50.0,
		},
		SensorData: models.SensorData{
			LastUpdate: time.Now().Unix(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result := engine.Validate(validDevice)
	assert.NotNil(t, result)
	assert.True(t, result.IsValid)
	assert.Empty(t, result.Errors)
}

// TestValidationEngineInvalidData 测试无效数据验证
func TestValidationEngineInvalidData(t *testing.T) {
	engine := models.NewValidationEngine()

	// 测试无效数据
	invalidDevice := &models.Device{
		DeviceID:   "", // 缺少必填字段
		DeviceType: "sensor",
		Status:     "online",
		DeviceInfo: models.DeviceInfo{
			Model:           "",  // 缺少必填字段
			Manufacturer:    "",  // 缺少必填字段
			FirmwareVersion: "",  // 缺少必填字段
			HardwareVersion: "",  // 缺少必填字段
			SerialNumber:    "",  // 缺少必填字段
			BatteryLevel:    150, // 超出范围
			SignalStrength:  10,  // 超出范围
			NetworkType:     "wifi",
		},
		SensorData: models.SensorData{
			LastUpdate: time.Now().Unix(),
		},
	}

	result := engine.Validate(invalidDevice)
	assert.NotNil(t, result)
	assert.False(t, result.IsValid)
	assert.NotEmpty(t, result.Errors)
}

// TestValidationEngineCustomRules 测试自定义规则
func TestValidationEngineCustomRules(t *testing.T) {
	engine := models.NewValidationEngine()

	// 注册自定义规则
	engine.RegisterCustomRule("device_id_format", func(data interface{}) *models.EnhancedValidationError {
		if device, ok := data.(*models.Device); ok {
			if len(device.DeviceID) < 5 {
				return &models.EnhancedValidationError{
					Field:     "DeviceID",
					Message:   "Device ID must be at least 5 characters",
					Value:     device.DeviceID,
					Type:      models.ValidationErrorTypeBusiness,
					Severity:  models.ValidationSeverityMedium,
					Code:      "device_id_format",
					Timestamp: time.Now(),
					Recovery:  models.RecoveryStrategyManual,
					Metadata:  make(map[string]interface{}),
				}
			}
		}
		return nil
	})

	// 测试自定义规则
	device := &models.Device{
		DeviceID:   "abc", // 太短
		DeviceType: "sensor",
		Status:     "online",
		DeviceInfo: models.DeviceInfo{
			Model:           "TempSensor-v1",
			Manufacturer:    "IoTCorp",
			FirmwareVersion: "1.0.0",
			HardwareVersion: "1.0",
			SerialNumber:    "SN123456",
			BatteryLevel:    85,
			SignalStrength:  -45,
			NetworkType:     "wifi",
		},
		SensorData: models.SensorData{
			LastUpdate: time.Now().Unix(),
		},
	}

	result := engine.Validate(device)
	assert.NotNil(t, result)
	assert.False(t, result.IsValid)
	assert.NotEmpty(t, result.Errors)

	// 检查自定义规则错误
	found := false
	for _, err := range result.Errors {
		if err.Code == "device_id_format" {
			found = true
			assert.Equal(t, "DeviceID", err.Field)
			assert.Equal(t, models.ValidationErrorTypeBusiness, err.Type)
			break
		}
	}
	assert.True(t, found, "Custom rule error should be found")
}

// TestValidationEngineAsync 测试异步验证
func TestValidationEngineAsync(t *testing.T) {
	engine := models.NewValidationEngine()

	device := &models.Device{
		DeviceID:   "device-001",
		DeviceType: "sensor",
		Status:     "online",
		Timestamp:  time.Now().Unix(),
		DeviceInfo: models.DeviceInfo{
			Model:           "TempSensor-v1",
			Manufacturer:    "IoTCorp",
			FirmwareVersion: "1.0.0",
			HardwareVersion: "1.0",
			SerialNumber:    "SN123456",
			BatteryLevel:    85,
			SignalStrength:  -45,
			NetworkType:     "wifi",
		},
		SensorData: models.SensorData{
			LastUpdate: time.Now().Unix(),
		},
	}

	resultChan := engine.ValidateAsync(device)
	result := <-resultChan

	assert.NotNil(t, result)
	assert.True(t, result.IsValid)
}

// TestValidationEngineConcurrent 测试并发验证
func TestValidationEngineConcurrent(t *testing.T) {
	engine := models.NewValidationEngine()

	devices := make([]interface{}, 5)
	for i := 0; i < 5; i++ {
		devices[i] = &models.Device{
			DeviceID:   fmt.Sprintf("device-%03d", i),
			DeviceType: "sensor",
			Status:     "online",
			Timestamp:  time.Now().Unix(),
			DeviceInfo: models.DeviceInfo{
				Model:           "TempSensor-v1",
				Manufacturer:    "IoTCorp",
				FirmwareVersion: "1.0.0",
				HardwareVersion: "1.0",
				SerialNumber:    fmt.Sprintf("SN%06d", i),
				BatteryLevel:    85,
				SignalStrength:  -45,
				NetworkType:     "wifi",
			},
			SensorData: models.SensorData{
				LastUpdate: time.Now().Unix(),
			},
		}
	}

	results := engine.ValidateConcurrent(devices)
	assert.Len(t, results, 5)

	for _, result := range results {
		assert.NotNil(t, result)
		assert.True(t, result.IsValid)
	}
}

// TestValidationCache 测试验证缓存
func TestValidationCache(t *testing.T) {
	cache := models.NewValidationCache(10, 5*time.Minute)
	assert.NotNil(t, cache)

	result := models.NewEnhancedValidationResult()
	result.IsValid = false
	result.Errors = []models.EnhancedValidationError{
		{
			Field:     "test",
			Message:   "test error",
			Type:      models.ValidationErrorTypeStructural,
			Severity:  models.ValidationSeverityMedium,
			Code:      "test",
			Timestamp: time.Now(),
			Recovery:  models.RecoveryStrategyDefaultValue,
			Metadata:  make(map[string]interface{}),
		},
	}

	// 测试设置和获取
	cache.Set("test-key", result)
	cachedResult, found := cache.Get("test-key")
	assert.True(t, found)
	assert.NotNil(t, cachedResult)
	assert.False(t, cachedResult.IsValid)
	assert.Len(t, cachedResult.Errors, 1)

	// 测试不存在的键
	_, found = cache.Get("non-existent")
	assert.False(t, found)

	// 测试缓存大小
	assert.Equal(t, 1, cache.Size())

	// 测试清空缓存
	cache.Clear()
	assert.Equal(t, 0, cache.Size())
}

// TestValidationMetrics 测试验证指标
func TestValidationMetrics(t *testing.T) {
	engine := models.NewValidationEngine()

	// 执行一些验证
	validDevice := &models.Device{
		DeviceID:   "device-001",
		DeviceType: "sensor",
		Status:     "online",
		DeviceInfo: models.DeviceInfo{
			Model:           "TempSensor-v1",
			Manufacturer:    "IoTCorp",
			FirmwareVersion: "1.0.0",
			HardwareVersion: "1.0",
			SerialNumber:    "SN123456",
			BatteryLevel:    85,
			SignalStrength:  -45,
			NetworkType:     "wifi",
		},
		SensorData: models.SensorData{
			LastUpdate: time.Now().Unix(),
		},
	}

	invalidDevice := &models.Device{
		DeviceID:   "", // 无效
		DeviceType: "sensor",
		Status:     "online",
		SensorData: models.SensorData{
			LastUpdate: time.Now().Unix(),
		},
	}

	engine.Validate(validDevice)
	engine.Validate(invalidDevice)

	metrics := engine.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(2), metrics.TotalValidations)
	assert.Equal(t, int64(1), metrics.SuccessfulValidations)
	assert.Equal(t, int64(1), metrics.FailedValidations)
}

// TestEnhancedValidationResultConcurrency 测试验证结果的并发安全性
func TestEnhancedValidationResultConcurrency(t *testing.T) {
	result := models.NewEnhancedValidationResult()

	var wg sync.WaitGroup
	numGoroutines := 100

	// 并发添加错误
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			err := models.EnhancedValidationError{
				Field:     fmt.Sprintf("field-%d", index),
				Message:   fmt.Sprintf("error-%d", index),
				Type:      models.ValidationErrorTypeStructural,
				Severity:  models.ValidationSeverityMedium,
				Code:      "test",
				Timestamp: time.Now(),
				Recovery:  models.RecoveryStrategyDefaultValue,
				Metadata:  make(map[string]interface{}),
			}
			result.AddError(err)
		}(i)
	}

	wg.Wait()

	errors := result.GetErrors()
	assert.Len(t, errors, numGoroutines)
	assert.False(t, result.IsValid)
}

// TestValidationRecoveryHandlers 测试恢复处理器
func TestValidationRecoveryHandlers(t *testing.T) {
	engine := models.NewValidationEngine()

	// 注册恢复处理器
	engine.RegisterRecoveryHandler("required", func(data interface{}, error *models.EnhancedValidationError) (interface{}, error) {
		// 简单的恢复逻辑
		return data, nil
	})

	// 测试无效数据
	invalidDevice := &models.Device{
		DeviceID:   "", // 缺少必填字段
		DeviceType: "sensor",
		Status:     "online",
		SensorData: models.SensorData{
			LastUpdate: time.Now().Unix(),
		},
	}

	result := engine.Validate(invalidDevice)
	assert.NotNil(t, result)
	// 注意：由于恢复处理器的存在，可能会有修复记录
}
