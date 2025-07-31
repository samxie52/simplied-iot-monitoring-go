package models

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/go-playground/validator/v10"
)

// SensorData 传感器数据容器 - 支持多种传感器类型
type SensorData struct {
	Temperature *TemperatureSensor `json:"temperature,omitempty"`
	Humidity    *HumiditySensor    `json:"humidity,omitempty"`
	Pressure    *PressureSensor    `json:"pressure,omitempty"`
	Switch      *SwitchSensor      `json:"switch_status,omitempty"`
	Current     *CurrentSensor     `json:"current,omitempty"`

	// 扩展字段
	Custom     map[string]interface{} `json:"custom,omitempty"`
	LastUpdate int64                  `json:"last_update" validate:"required"`
}

// TemperatureSensor 温度传感器 - 精度0.1°C
type TemperatureSensor struct {
	Value  float64      `json:"value" validate:"required"`
	Unit   string       `json:"unit" validate:"required,eq=°C"`
	Status SensorStatus `json:"status" validate:"required,oneof=normal warning error offline"`
	Min    float64      `json:"min,omitempty"`
	Max    float64      `json:"max,omitempty"`
}

// HumiditySensor 湿度传感器 - 精度0.1%
type HumiditySensor struct {
	Value  float64      `json:"value" validate:"required"`
	Unit   string       `json:"unit" validate:"required,eq=%"`
	Status SensorStatus `json:"status" validate:"required,oneof=normal warning error offline"`
	Min    float64      `json:"min,omitempty"`
	Max    float64      `json:"max,omitempty"`
}

// PressureSensor 压力传感器 - 精度0.01hPa
type PressureSensor struct {
	Value  float64      `json:"value" validate:"required"`
	Unit   string       `json:"unit" validate:"required,eq=hPa"`
	Status SensorStatus `json:"status" validate:"required,oneof=normal warning error offline"`
	Min    float64      `json:"min,omitempty"`
	Max    float64      `json:"max,omitempty"`
}

// SwitchSensor 开关传感器 - 布尔状态
type SwitchSensor struct {
	Value  bool         `json:"value"`
	Status SwitchStatus `json:"status" validate:"required,oneof=on off error"`
}

// CurrentSensor 电流传感器 - 精度0.01A
type CurrentSensor struct {
	Value  float64      `json:"value" validate:"required"`
	Unit   string       `json:"unit" validate:"required,eq=A"`
	Status SensorStatus `json:"status" validate:"required,oneof=normal warning error offline"`
	Min    float64      `json:"min,omitempty"`
	Max    float64      `json:"max,omitempty"`
}

// NewSensorData 创建新的传感器数据容器
func NewSensorData() *SensorData {
	return &SensorData{
		Custom:     make(map[string]interface{}),
		LastUpdate: time.Now().UnixMilli(),
	}
}

// UpdateLastUpdate 更新最后更新时间
func (sd *SensorData) UpdateLastUpdate() {
	sd.LastUpdate = time.Now().UnixMilli()
}

// HasTemperature 检查是否有温度传感器数据
func (sd *SensorData) HasTemperature() bool {
	return sd.Temperature != nil
}

// HasHumidity 检查是否有湿度传感器数据
func (sd *SensorData) HasHumidity() bool {
	return sd.Humidity != nil
}

// HasPressure 检查是否有压力传感器数据
func (sd *SensorData) HasPressure() bool {
	return sd.Pressure != nil
}

// HasSwitch 检查是否有开关传感器数据
func (sd *SensorData) HasSwitch() bool {
	return sd.Switch != nil
}

// HasCurrent 检查是否有电流传感器数据
func (sd *SensorData) HasCurrent() bool {
	return sd.Current != nil
}

// GetSensorCount 获取传感器数量
func (sd *SensorData) GetSensorCount() int {
	count := 0
	if sd.HasTemperature() {
		count++
	}
	if sd.HasHumidity() {
		count++
	}
	if sd.HasPressure() {
		count++
	}
	if sd.HasSwitch() {
		count++
	}
	if sd.HasCurrent() {
		count++
	}
	return count
}

// Validate 验证传感器数据
func (sd *SensorData) Validate() *ValidationResult {
	result := NewValidationResult()
	validator := validator.New()

	// 结构体验证
	if err := validator.Struct(sd); err != nil {
		validationErr := NewValidationError(
			"struct",
			"sensor_data",
			fmt.Sprintf("Validation failed: %s", err.Error()),
			sd,
		)
		result.AddError(*validationErr)
	}

	// 验证各个传感器
	if sd.Temperature != nil {
		sd.Temperature.validate(result, "temperature")
	}
	if sd.Humidity != nil {
		sd.Humidity.validate(result, "humidity")
	}
	if sd.Pressure != nil {
		sd.Pressure.validate(result, "pressure")
	}
	if sd.Switch != nil {
		sd.Switch.validate(result, "switch")
	}
	if sd.Current != nil {
		sd.Current.validate(result, "current")
	}

	// 业务验证
	sd.validateBusinessRules(result)

	result.ProcessedAt = time.Now().UnixMilli()
	return result
}

// validateBusinessRules 业务规则验证
func (sd *SensorData) validateBusinessRules(result *ValidationResult) {
	// 验证至少有一个传感器数据
	if sd.GetSensorCount() == 0 {
		result.AddError(*NewValidationError("business", "sensors", "At least one sensor data is required", nil))
	}

	// 验证最后更新时间
	now := time.Now().UnixMilli()
	if sd.LastUpdate > now+60000 { // 不能超过当前时间1分钟
		result.AddError(*NewValidationError("business", "last_update", "LastUpdate cannot be in the future", sd.LastUpdate))
	}
}

// ToJSON 转换为JSON字符串
func (sd *SensorData) ToJSON() ([]byte, error) {
	return json.Marshal(sd)
}

// FromJSON 从JSON字符串解析
func (sd *SensorData) FromJSON(data []byte) error {
	return json.Unmarshal(data, sd)
}

// TemperatureSensor 方法

// NewTemperatureSensor 创建温度传感器
func NewTemperatureSensor(value float64) *TemperatureSensor {
	ts := &TemperatureSensor{
		Unit: "°C",
	}
	ts.SetValue(value) // 使用SetValue来正确设置值和状态
	return ts
}

// SetValue 设置温度值
func (ts *TemperatureSensor) SetValue(value float64) {
	ts.Value = math.Round(value*10) / 10 // 保留1位小数
	ts.updateStatus()
}

// updateStatus 根据数值更新状态
func (ts *TemperatureSensor) updateStatus() {
	switch {
	case ts.Value < -40 || ts.Value > 120:
		ts.Status = SensorStatusError
	case ts.Value < -20 || ts.Value > 80:
		ts.Status = SensorStatusWarning
	default:
		ts.Status = SensorStatusNormal
	}
}

// IsNormal 检查状态是否正常
func (ts *TemperatureSensor) IsNormal() bool {
	return ts.Status == SensorStatusNormal
}

// validate 验证温度传感器
func (ts *TemperatureSensor) validate(result *ValidationResult, prefix string) {
	validator := validator.New()
	if err := validator.Struct(ts); err != nil {
		validationErr := NewValidationError(
			"struct",
			fmt.Sprintf("%s.temperature", prefix),
			fmt.Sprintf("Validation failed: %s", err.Error()),
			ts,
		)
		result.AddError(*validationErr)
	}

	// 业务验证
	if ts.Min != 0 && ts.Max != 0 && ts.Min >= ts.Max {
		result.AddError(*NewValidationError("business", fmt.Sprintf("%s.range", prefix), "Min value must be less than Max value", nil))
	}
}

// HumiditySensor 方法

// NewHumiditySensor 创建湿度传感器
func NewHumiditySensor(value float64) *HumiditySensor {
	hs := &HumiditySensor{
		Unit: "%",
	}
	hs.SetValue(value) // 使用SetValue来正确设置值和状态
	return hs
}

// SetValue 设置湿度值
func (hs *HumiditySensor) SetValue(value float64) {
	hs.Value = math.Round(value*10) / 10 // 保留1位小数
	hs.updateStatus()
}

// updateStatus 根据数值更新状态
func (hs *HumiditySensor) updateStatus() {
	switch {
	case hs.Value < 0 || hs.Value > 100:
		hs.Status = SensorStatusError
	case hs.Value < 10 || hs.Value > 90:
		hs.Status = SensorStatusWarning
	default:
		hs.Status = SensorStatusNormal
	}
}

// IsNormal 检查状态是否正常
func (hs *HumiditySensor) IsNormal() bool {
	return hs.Status == SensorStatusNormal
}

// validate 验证湿度传感器
func (hs *HumiditySensor) validate(result *ValidationResult, prefix string) {
	validator := validator.New()
	if err := validator.Struct(hs); err != nil {
		validationErr := NewValidationError(
			"struct",
			fmt.Sprintf("%s.humidity", prefix),
			fmt.Sprintf("Validation failed: %s", err.Error()),
			hs,
		)
		result.AddError(*validationErr)
	}

	// 业务验证
	if hs.Min != 0 && hs.Max != 0 && hs.Min >= hs.Max {
		result.AddError(*NewValidationError("business", fmt.Sprintf("%s.range", prefix), "Min value must be less than Max value", nil))
	}
}

// PressureSensor 方法

// NewPressureSensor 创建压力传感器
func NewPressureSensor(value float64) *PressureSensor {
	ps := &PressureSensor{
		Unit: "hPa",
	}
	ps.SetValue(value) // 使用SetValue来正确设置值和状态
	return ps
}

// SetValue 设置压力值
func (ps *PressureSensor) SetValue(value float64) {
	ps.Value = math.Round(value*100) / 100 // 保留2位小数
	ps.updateStatus()
}

// updateStatus 根据数值更新状态
func (ps *PressureSensor) updateStatus() {
	switch {
	case ps.Value < 800 || ps.Value > 1200:
		ps.Status = SensorStatusError
	case ps.Value < 900 || ps.Value > 1100:
		ps.Status = SensorStatusWarning
	default:
		ps.Status = SensorStatusNormal
	}
}

// IsNormal 检查状态是否正常
func (ps *PressureSensor) IsNormal() bool {
	return ps.Status == SensorStatusNormal
}

// validate 验证压力传感器
func (ps *PressureSensor) validate(result *ValidationResult, prefix string) {
	validator := validator.New()
	if err := validator.Struct(ps); err != nil {
		validationErr := NewValidationError(
			"struct",
			fmt.Sprintf("%s.pressure", prefix),
			fmt.Sprintf("Validation failed: %s", err.Error()),
			ps,
		)
		result.AddError(*validationErr)
	}

	// 业务验证
	if ps.Min != 0 && ps.Max != 0 && ps.Min >= ps.Max {
		result.AddError(*NewValidationError("business", fmt.Sprintf("%s.range", prefix), "Min value must be less than Max value", nil))
	}
}

// SwitchSensor 方法

// NewSwitchSensor 创建开关传感器
func NewSwitchSensor(value bool) *SwitchSensor {
	status := SwitchStatusOff
	if value {
		status = SwitchStatusOn
	}
	return &SwitchSensor{
		Value:  value,
		Status: status,
	}
}

// SetValue 设置开关值
func (ss *SwitchSensor) SetValue(value bool) {
	ss.Value = value
	if value {
		ss.Status = SwitchStatusOn
	} else {
		ss.Status = SwitchStatusOff
	}
}

// IsOn 检查开关是否打开
func (ss *SwitchSensor) IsOn() bool {
	return ss.Value && ss.Status == SwitchStatusOn
}

// IsOff 检查开关是否关闭
func (ss *SwitchSensor) IsOff() bool {
	return !ss.Value && ss.Status == SwitchStatusOff
}

// HasError 检查开关是否有错误
func (ss *SwitchSensor) HasError() bool {
	return ss.Status == SwitchStatusError
}

// validate 验证开关传感器
func (ss *SwitchSensor) validate(result *ValidationResult, prefix string) {
	validator := validator.New()
	if err := validator.Struct(ss); err != nil {
		validationErr := NewValidationError(
			"struct",
			fmt.Sprintf("%s.switch", prefix),
			fmt.Sprintf("Validation failed: %s", err.Error()),
			ss,
		)
		result.AddError(*validationErr)
	}

	// 业务验证：检查状态与值的一致性
	if ss.Value && ss.Status == SwitchStatusOff {
		result.AddError(*NewValidationError("business", fmt.Sprintf("%s.consistency", prefix), "Switch value and status are inconsistent", nil))
	}
	if !ss.Value && ss.Status == SwitchStatusOn {
		result.AddError(*NewValidationError("business", fmt.Sprintf("%s.consistency", prefix), "Switch value and status are inconsistent", nil))
	}
}

// CurrentSensor 方法

// NewCurrentSensor 创建电流传感器
func NewCurrentSensor(value float64) *CurrentSensor {
	cs := &CurrentSensor{
		Unit: "A",
	}
	cs.SetValue(value) // 使用SetValue来正确设置值和状态
	return cs
}

// SetValue 设置电流值
func (cs *CurrentSensor) SetValue(value float64) {
	cs.Value = math.Round(value*100) / 100 // 保留2位小数
	cs.updateStatus()
}

// updateStatus 根据数值更新状态
func (cs *CurrentSensor) updateStatus() {
	switch {
	case cs.Value < 0 || cs.Value > 100:
		cs.Status = SensorStatusError
	case cs.Value > 80:
		cs.Status = SensorStatusWarning
	default:
		cs.Status = SensorStatusNormal
	}
}

// IsNormal 检查状态是否正常
func (cs *CurrentSensor) IsNormal() bool {
	return cs.Status == SensorStatusNormal
}

// validate 验证电流传感器
func (cs *CurrentSensor) validate(result *ValidationResult, prefix string) {
	validator := validator.New()
	if err := validator.Struct(cs); err != nil {
		validationErr := NewValidationError(
			"struct",
			fmt.Sprintf("%s.current", prefix),
			fmt.Sprintf("Validation failed: %s", err.Error()),
			cs,
		)
		result.AddError(*validationErr)
	}

	// 业务验证
	if cs.Min != 0 && cs.Max != 0 && cs.Min >= cs.Max {
		result.AddError(*NewValidationError("business", fmt.Sprintf("%s.range", prefix), "Min value must be less than Max value", nil))
	}
}
