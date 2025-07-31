package models

import (
	"fmt"
	"sync"
	"time"
)

// DeviceStatus 设备状态枚举
type DeviceStatus string

const (
	DeviceStatusOnline  DeviceStatus = "online"
	DeviceStatusOffline DeviceStatus = "offline"
	DeviceStatusError   DeviceStatus = "error"
	DeviceStatusMaint   DeviceStatus = "maintenance"
)

// SensorStatus 传感器状态枚举
type SensorStatus string

const (
	SensorStatusNormal  SensorStatus = "normal"
	SensorStatusWarning SensorStatus = "warning"
	SensorStatusError   SensorStatus = "error"
	SensorStatusOffline SensorStatus = "offline"
)

// SwitchStatus 开关状态枚举
type SwitchStatus string

const (
	SwitchStatusOn  SwitchStatus = "on"
	SwitchStatusOff SwitchStatus = "off"
	SwitchStatusError SwitchStatus = "error"
)

// NetworkType 网络类型枚举
type NetworkType string

const (
	NetworkTypeWiFi     NetworkType = "wifi"
	NetworkTypeEthernet NetworkType = "ethernet"
	NetworkTypeLoRa     NetworkType = "lora"
	NetworkTypeNBIoT    NetworkType = "nb-iot"
)

// Timestamp 时间戳标准化 - 多格式支持
type Timestamp struct {
	Unix      int64     `json:"unix" validate:"required,min=0"`          // Unix毫秒时间戳
	RFC3339   string    `json:"rfc3339" validate:"required"`             // RFC3339格式
	Timezone  string    `json:"timezone" validate:"required"`            // 时区信息
	LocalTime time.Time `json:"local_time"`                              // 本地时间
}

// NewTimestamp 创建标准化时间戳
func NewTimestamp(t time.Time) *Timestamp {
	return &Timestamp{
		Unix:      t.UnixMilli(),
		RFC3339:   t.Format(time.RFC3339),
		Timezone:  t.Location().String(),
		LocalTime: t,
	}
}

// NewTimestampNow 创建当前时间的标准化时间戳
func NewTimestampNow() *Timestamp {
	return NewTimestamp(time.Now())
}

// ToTime 转换为time.Time类型
func (ts *Timestamp) ToTime() time.Time {
	if !ts.LocalTime.IsZero() {
		return ts.LocalTime
	}
	return time.UnixMilli(ts.Unix)
}

// IsValid 验证时间戳是否有效
func (ts *Timestamp) IsValid() bool {
	if ts.Unix <= 0 {
		return false
	}
	if ts.RFC3339 == "" {
		return false
	}
	// 验证RFC3339格式是否正确
	_, err := time.Parse(time.RFC3339, ts.RFC3339)
	return err == nil
}

// ValidationError 验证错误信息 - 详细错误分类
type ValidationError struct {
	ErrorCode   string      `json:"error_code" validate:"required"`
	ErrorType   string      `json:"error_type" validate:"required,oneof=struct business system"`
	Field       string      `json:"field" validate:"required"`
	Message     string      `json:"message" validate:"required"`
	Value       interface{} `json:"value,omitempty"`
	Constraint  string      `json:"constraint,omitempty"`
	Timestamp   int64       `json:"timestamp" validate:"required,min=0"`
}

// NewValidationError 创建验证错误
func NewValidationError(errorType, field, message string, value interface{}) *ValidationError {
	return &ValidationError{
		ErrorCode: fmt.Sprintf("VALIDATION_%s_%s", errorType, field),
		ErrorType: errorType,
		Field:     field,
		Message:   message,
		Value:     value,
		Timestamp: time.Now().UnixMilli(),
	}
}

// ValidationResult 验证结果汇总
type ValidationResult struct {
	mu          sync.RWMutex       `json:"-"` // 保护并发访问
	IsValid     bool               `json:"is_valid"`
	Errors      []ValidationError  `json:"errors,omitempty"`
	Warnings    []ValidationError  `json:"warnings,omitempty"`
	ProcessedAt int64              `json:"processed_at" validate:"required,min=0"`
	Duration    int64              `json:"duration_ms" validate:"min=0"`
}

// NewValidationResult 创建验证结果
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		IsValid:     true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationError, 0),
		ProcessedAt: time.Now().UnixMilli(),
	}
}

// AddError 添加错误
func (vr *ValidationResult) AddError(err ValidationError) {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	vr.Errors = append(vr.Errors, err)
	vr.IsValid = false
}

// AddWarning 添加警告
func (vr *ValidationResult) AddWarning(warning ValidationError) {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	vr.Warnings = append(vr.Warnings, warning)
}

// HasErrors 是否有错误
func (vr *ValidationResult) HasErrors() bool {
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	return len(vr.Errors) > 0
}

// HasWarnings 是否有警告
func (vr *ValidationResult) HasWarnings() bool {
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	return len(vr.Warnings) > 0
}

// ErrorCount 错误数量
func (vr *ValidationResult) ErrorCount() int {
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	return len(vr.Errors)
}

// WarningCount 警告数量
func (vr *ValidationResult) WarningCount() int {
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	return len(vr.Warnings)
}
