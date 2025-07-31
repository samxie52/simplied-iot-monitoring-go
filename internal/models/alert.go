package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AlertSeverity 告警严重级别
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus 告警状态
type AlertStatus string

const (
	AlertStatusActive       AlertStatus = "active"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
	AlertStatusClosed       AlertStatus = "closed"
)

// AlertType 告警类型
type AlertType string

const (
	AlertTypeThreshold    AlertType = "threshold"
	AlertTypeAnomaly      AlertType = "anomaly"
	AlertTypeConnectivity AlertType = "connectivity"
	AlertTypeSystem       AlertType = "system"
	AlertTypeBattery      AlertType = "battery"
	AlertTypeCustom       AlertType = "custom"
)

// Alert 告警数据模型 - 完整告警信息
type Alert struct {
	AlertID     string        `json:"alert_id" validate:"required,min=1,max=64"`
	DeviceID    string        `json:"device_id" validate:"required,min=1,max=64"`
	AlertType   AlertType     `json:"alert_type" validate:"required"`
	Severity    AlertSeverity `json:"severity" validate:"required"`
	
	// 告警内容
	Title       string        `json:"title" validate:"required,min=1,max=128"`
	Message     string        `json:"message" validate:"required,min=1,max=512"`
	Description string        `json:"description,omitempty"`
	
	// 时间信息
	Timestamp   int64         `json:"timestamp" validate:"required,min=0"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	ResolvedAt  *time.Time    `json:"resolved_at,omitempty"`
	
	// 告警状态
	Status      AlertStatus   `json:"status" validate:"required"`
	
	// 阈值信息
	Threshold   *AlertThreshold `json:"threshold,omitempty"`
	CurrentValue interface{}    `json:"current_value,omitempty"`
	
	// 关联信息
	RuleID      string        `json:"rule_id,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	
	// 处理信息
	AcknowledgedBy string     `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	ResolvedBy     string     `json:"resolved_by,omitempty"`
	
	// 统计信息
	OccurrenceCount int       `json:"occurrence_count,omitempty"`
	FirstOccurrence int64     `json:"first_occurrence,omitempty"`
	LastOccurrence  int64     `json:"last_occurrence,omitempty"`
}

// AlertThreshold 告警阈值定义
type AlertThreshold struct {
	Field       string      `json:"field" validate:"required"`
	Operator    string      `json:"operator" validate:"required,oneof=gt gte lt lte eq ne"`
	Value       interface{} `json:"value" validate:"required"`
	Unit        string      `json:"unit,omitempty"`
	Duration    int64       `json:"duration,omitempty"` // 持续时间(秒)
}

// AlertRule 告警规则定义
type AlertRule struct {
	RuleID      string        `json:"rule_id" validate:"required,min=1,max=64"`
	Name        string        `json:"name" validate:"required,min=1,max=128"`
	Description string        `json:"description,omitempty"`
	
	// 规则条件
	Conditions  []AlertCondition `json:"conditions" validate:"required,min=1"`
	LogicOp     string           `json:"logic_op" validate:"required,oneof=AND OR"`
	
	// 告警配置
	Severity    AlertSeverity    `json:"severity" validate:"required"`
	AlertType   AlertType        `json:"alert_type" validate:"required"`
	
	// 目标设备
	DeviceIDs   []string         `json:"device_ids,omitempty"`
	DeviceTypes []string         `json:"device_types,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	
	// 规则状态
	Enabled     bool             `json:"enabled"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	CreatedBy   string           `json:"created_by,omitempty"`
	
	// 冷却时间
	Cooldown    int64            `json:"cooldown,omitempty"` // 冷却时间(秒)
	
	// 通知配置
	NotificationChannels []string `json:"notification_channels,omitempty"`
}

// AlertCondition 告警条件
type AlertCondition struct {
	Field       string      `json:"field" validate:"required"`
	Operator    string      `json:"operator" validate:"required,oneof=gt gte lt lte eq ne contains"`
	Value       interface{} `json:"value" validate:"required"`
	Duration    int64       `json:"duration,omitempty"` // 持续时间(秒)
}

// AlertAggregation 告警聚合信息
type AlertAggregation struct {
	AggregationID string        `json:"aggregation_id" validate:"required"`
	GroupBy       []string      `json:"group_by" validate:"required"`
	AlertIDs      []string      `json:"alert_ids" validate:"required,min=1"`
	Count         int           `json:"count" validate:"min=1"`
	Severity      AlertSeverity `json:"severity" validate:"required"`
	
	// 时间窗口
	WindowStart   int64         `json:"window_start" validate:"required,min=0"`
	WindowEnd     int64         `json:"window_end" validate:"required,min=0"`
	
	// 聚合统计
	FirstAlert    int64         `json:"first_alert" validate:"required,min=0"`
	LastAlert     int64         `json:"last_alert" validate:"required,min=0"`
	
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// AlertHistory 告警历史记录
type AlertHistory struct {
	HistoryID   string      `json:"history_id" validate:"required"`
	AlertID     string      `json:"alert_id" validate:"required"`
	Action      string      `json:"action" validate:"required,oneof=created acknowledged resolved closed"`
	Timestamp   int64       `json:"timestamp" validate:"required,min=0"`
	UserID      string      `json:"user_id,omitempty"`
	Comment     string      `json:"comment,omitempty"`
	OldStatus   AlertStatus `json:"old_status,omitempty"`
	NewStatus   AlertStatus `json:"new_status,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewAlert 创建新告警
func NewAlert(deviceID string, alertType AlertType, severity AlertSeverity, title, message string) *Alert {
	now := time.Now()
	return &Alert{
		AlertID:         uuid.New().String(),
		DeviceID:        deviceID,
		AlertType:       alertType,
		Severity:        severity,
		Title:           title,
		Message:         message,
		Timestamp:       now.Unix(),
		CreatedAt:       now,
		UpdatedAt:       now,
		Status:          AlertStatusActive,
		OccurrenceCount: 1,
		FirstOccurrence: now.Unix(),
		LastOccurrence:  now.Unix(),
		Tags:            make([]string, 0),
		Metadata:        make(map[string]interface{}),
	}
}

// NewAlertRule 创建新告警规则
func NewAlertRule(name string, conditions []AlertCondition, severity AlertSeverity, alertType AlertType) *AlertRule {
	now := time.Now()
	return &AlertRule{
		RuleID:      uuid.New().String(),
		Name:        name,
		Conditions:  conditions,
		LogicOp:     "AND",
		Severity:    severity,
		AlertType:   alertType,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
		DeviceIDs:   make([]string, 0),
		DeviceTypes: make([]string, 0),
		Tags:        make([]string, 0),
		NotificationChannels: make([]string, 0),
	}
}

// Acknowledge 确认告警
func (a *Alert) Acknowledge(userID string) {
	now := time.Now()
	a.Status = AlertStatusAcknowledged
	a.AcknowledgedBy = userID
	a.AcknowledgedAt = &now
	a.UpdatedAt = now
}

// Resolve 解决告警
func (a *Alert) Resolve(userID string) {
	now := time.Now()
	a.Status = AlertStatusResolved
	a.ResolvedBy = userID
	a.ResolvedAt = &now
	a.UpdatedAt = now
}

// Close 关闭告警
func (a *Alert) Close() {
	a.Status = AlertStatusClosed
	a.UpdatedAt = time.Now()
}

// IncrementOccurrence 增加发生次数
func (a *Alert) IncrementOccurrence() {
	a.OccurrenceCount++
	a.LastOccurrence = time.Now().Unix()
	a.UpdatedAt = time.Now()
}

// SetThreshold 设置阈值
func (a *Alert) SetThreshold(field, operator string, value interface{}, unit string) {
	a.Threshold = &AlertThreshold{
		Field:    field,
		Operator: operator,
		Value:    value,
		Unit:     unit,
	}
}

// SetCurrentValue 设置当前值
func (a *Alert) SetCurrentValue(value interface{}) {
	a.CurrentValue = value
}

// AddTag 添加标签
func (a *Alert) AddTag(tag string) {
	if a.Tags == nil {
		a.Tags = make([]string, 0)
	}
	// 检查是否已存在
	for _, existingTag := range a.Tags {
		if existingTag == tag {
			return
		}
	}
	a.Tags = append(a.Tags, tag)
}

// RemoveTag 移除标签
func (a *Alert) RemoveTag(tag string) {
	if a.Tags == nil {
		return
	}
	for i, existingTag := range a.Tags {
		if existingTag == tag {
			a.Tags = append(a.Tags[:i], a.Tags[i+1:]...)
			return
		}
	}
}

// SetMetadata 设置元数据
func (a *Alert) SetMetadata(key string, value interface{}) {
	if a.Metadata == nil {
		a.Metadata = make(map[string]interface{})
	}
	a.Metadata[key] = value
}

// GetMetadata 获取元数据
func (a *Alert) GetMetadata(key string) (interface{}, bool) {
	if a.Metadata == nil {
		return nil, false
	}
	value, exists := a.Metadata[key]
	return value, exists
}

// IsActive 检查告警是否激活
func (a *Alert) IsActive() bool {
	return a.Status == AlertStatusActive
}

// IsResolved 检查告警是否已解决
func (a *Alert) IsResolved() bool {
	return a.Status == AlertStatusResolved || a.Status == AlertStatusClosed
}

// GetAge 获取告警年龄(秒)
func (a *Alert) GetAge() int64 {
	return time.Now().Unix() - a.Timestamp
}

// ToJSON 转换为JSON
func (a *Alert) ToJSON() ([]byte, error) {
	return json.Marshal(a)
}

// FromJSON 从JSON解析
func (a *Alert) FromJSON(data []byte) error {
	return json.Unmarshal(data, a)
}

// Validate 验证告警数据
func (a *Alert) Validate() error {
	if a.AlertID == "" {
		return fmt.Errorf("alert_id is required")
	}
	if a.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if a.AlertType == "" {
		return fmt.Errorf("alert_type is required")
	}
	if a.Severity == "" {
		return fmt.Errorf("severity is required")
	}
	if a.Title == "" {
		return fmt.Errorf("title is required")
	}
	if a.Message == "" {
		return fmt.Errorf("message is required")
	}
	if a.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}
	if a.Status == "" {
		return fmt.Errorf("status is required")
	}
	
	// 验证严重级别
	switch a.Severity {
	case AlertSeverityInfo, AlertSeverityWarning, AlertSeverityError, AlertSeverityCritical:
		// 有效
	default:
		return fmt.Errorf("invalid severity: %s", a.Severity)
	}
	
	// 验证状态
	switch a.Status {
	case AlertStatusActive, AlertStatusAcknowledged, AlertStatusResolved, AlertStatusClosed:
		// 有效
	default:
		return fmt.Errorf("invalid status: %s", a.Status)
	}
	
	return nil
}

// AlertRule方法

// AddCondition 添加条件
func (ar *AlertRule) AddCondition(condition AlertCondition) {
	ar.Conditions = append(ar.Conditions, condition)
	ar.UpdatedAt = time.Now()
}

// RemoveCondition 移除条件
func (ar *AlertRule) RemoveCondition(index int) error {
	if index < 0 || index >= len(ar.Conditions) {
		return fmt.Errorf("invalid condition index: %d", index)
	}
	ar.Conditions = append(ar.Conditions[:index], ar.Conditions[index+1:]...)
	ar.UpdatedAt = time.Now()
	return nil
}

// Enable 启用规则
func (ar *AlertRule) Enable() {
	ar.Enabled = true
	ar.UpdatedAt = time.Now()
}

// Disable 禁用规则
func (ar *AlertRule) Disable() {
	ar.Enabled = false
	ar.UpdatedAt = time.Now()
}

// AddDeviceID 添加设备ID
func (ar *AlertRule) AddDeviceID(deviceID string) {
	if ar.DeviceIDs == nil {
		ar.DeviceIDs = make([]string, 0)
	}
	// 检查是否已存在
	for _, id := range ar.DeviceIDs {
		if id == deviceID {
			return
		}
	}
	ar.DeviceIDs = append(ar.DeviceIDs, deviceID)
	ar.UpdatedAt = time.Now()
}

// RemoveDeviceID 移除设备ID
func (ar *AlertRule) RemoveDeviceID(deviceID string) {
	if ar.DeviceIDs == nil {
		return
	}
	for i, id := range ar.DeviceIDs {
		if id == deviceID {
			ar.DeviceIDs = append(ar.DeviceIDs[:i], ar.DeviceIDs[i+1:]...)
			ar.UpdatedAt = time.Now()
			return
		}
	}
}

// ToJSON 转换为JSON
func (ar *AlertRule) ToJSON() ([]byte, error) {
	return json.Marshal(ar)
}

// FromJSON 从JSON解析
func (ar *AlertRule) FromJSON(data []byte) error {
	return json.Unmarshal(data, ar)
}

// Validate 验证告警规则
func (ar *AlertRule) Validate() error {
	if ar.RuleID == "" {
		return fmt.Errorf("rule_id is required")
	}
	if ar.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(ar.Conditions) == 0 {
		return fmt.Errorf("at least one condition is required")
	}
	if ar.LogicOp != "AND" && ar.LogicOp != "OR" {
		return fmt.Errorf("logic_op must be AND or OR")
	}
	if ar.Severity == "" {
		return fmt.Errorf("severity is required")
	}
	if ar.AlertType == "" {
		return fmt.Errorf("alert_type is required")
	}
	
	// 验证条件
	for i, condition := range ar.Conditions {
		if condition.Field == "" {
			return fmt.Errorf("condition[%d].field is required", i)
		}
		if condition.Operator == "" {
			return fmt.Errorf("condition[%d].operator is required", i)
		}
		if condition.Value == nil {
			return fmt.Errorf("condition[%d].value is required", i)
		}
	}
	
	return nil
}
