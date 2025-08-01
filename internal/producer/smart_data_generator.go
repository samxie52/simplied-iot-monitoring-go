package producer

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

// SmartDataGenerator 智能数据生成器
type SmartDataGenerator struct {
	sensorType     string
	baseValue      float64
	variance       float64
	trendFactor    float64
	seasonalFactor float64
	anomalyRate    float64
	noiseLevel     float64

	// 时间序列状态
	currentTrend   float64
	trendDirection int
	seasonalPhase  float64
	lastValue      float64

	// 统计信息
	valueHistory []float64
	historySize  int

	// 随机数生成器
	random *rand.Rand
	mutex  sync.RWMutex

	// 传感器特定参数
	minValue  float64
	maxValue  float64
	unit      string
	precision int

	// 高级特性
	driftRate       float64   // 传感器漂移率
	calibrationTime time.Time // 上次校准时间
	degradationRate float64   // 设备老化率
}

// NewSmartDataGenerator 创建智能数据生成器
func NewSmartDataGenerator(sensorType string, config SensorGeneratorConfig) *SmartDataGenerator {
	source := rand.NewSource(time.Now().UnixNano())

	generator := &SmartDataGenerator{
		sensorType:     sensorType,
		baseValue:      config.BaseValue,
		variance:       config.Variance,
		trendFactor:    config.TrendFactor,
		seasonalFactor: config.SeasonalFactor,
		anomalyRate:    config.AnomalyRate,
		noiseLevel:     config.NoiseLevel,

		currentTrend:   0.0,
		trendDirection: 1,
		seasonalPhase:  0.0,
		lastValue:      config.BaseValue,

		valueHistory: make([]float64, 0, config.HistorySize),
		historySize:  config.HistorySize,

		random: rand.New(source),

		minValue:  config.MinValue,
		maxValue:  config.MaxValue,
		unit:      config.Unit,
		precision: config.Precision,

		driftRate:       config.DriftRate,
		calibrationTime: time.Now(),
		degradationRate: config.DegradationRate,
	}

	return generator
}

// SensorGeneratorConfig 传感器生成器配置
type SensorGeneratorConfig struct {
	BaseValue       float64
	Variance        float64
	TrendFactor     float64
	SeasonalFactor  float64
	AnomalyRate     float64
	NoiseLevel      float64
	HistorySize     int
	MinValue        float64
	MaxValue        float64
	Unit            string
	Precision       int
	DriftRate       float64
	DegradationRate float64
}

// GenerateValue 生成智能传感器数值
func (g *SmartDataGenerator) GenerateValue() float64 {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	now := time.Now()

	// 基础值计算
	baseValue := g.calculateBaseValue(now)

	// 添加趋势
	trendValue := g.calculateTrend(now)

	// 添加季节性变化
	seasonalValue := g.calculateSeasonal(now)

	// 添加噪声
	noiseValue := g.calculateNoise()

	// 添加异常
	anomalyValue := g.calculateAnomaly()

	// 添加传感器漂移
	driftValue := g.calculateDrift(now)

	// 添加设备老化影响
	degradationValue := g.calculateDegradation(now)

	// 组合所有因素
	rawValue := baseValue + trendValue + seasonalValue + noiseValue + anomalyValue + driftValue + degradationValue

	// 应用平滑过渡
	smoothedValue := g.applySmoothTransition(rawValue)

	// 限制在合理范围内
	finalValue := g.applyConstraints(smoothedValue)

	// 更新历史记录
	g.updateHistory(finalValue)

	// 更新状态
	g.lastValue = finalValue

	return g.applyPrecision(finalValue)
}

// calculateBaseValue 计算基础值
func (g *SmartDataGenerator) calculateBaseValue(now time.Time) float64 {
	switch g.sensorType {
	case "temperature":
		// 温度传感器：基于时间的日夜变化
		hour := float64(now.Hour())
		dailyCycle := math.Sin((hour-6)*math.Pi/12) * 5.0 // 6点最低，18点最高，变化幅度5度
		return g.baseValue + dailyCycle

	case "humidity":
		// 湿度传感器：与温度相反的变化模式
		hour := float64(now.Hour())
		dailyCycle := -math.Sin((hour-6)*math.Pi/12) * 10.0 // 与温度相反
		return g.baseValue + dailyCycle

	case "pressure":
		// 气压传感器：较为稳定，微小变化
		hour := float64(now.Hour())
		dailyCycle := math.Sin((hour-12)*math.Pi/12) * 2.0
		return g.baseValue + dailyCycle

	default:
		return g.baseValue
	}
}

// calculateTrend 计算趋势值
func (g *SmartDataGenerator) calculateTrend(now time.Time) float64 {
	if g.trendFactor == 0 {
		return 0
	}

	// 随机改变趋势方向
	if g.random.Float64() < 0.01 { // 1%概率改变趋势
		g.trendDirection *= -1
	}

	// 更新趋势值
	trendChange := g.trendFactor * float64(g.trendDirection) * g.random.NormFloat64() * 0.1
	g.currentTrend += trendChange

	// 限制趋势幅度
	maxTrend := g.variance * 2.0
	if math.Abs(g.currentTrend) > maxTrend {
		g.currentTrend = math.Copysign(maxTrend, g.currentTrend)
	}

	return g.currentTrend
}

// calculateSeasonal 计算季节性变化
func (g *SmartDataGenerator) calculateSeasonal(now time.Time) float64 {
	if g.seasonalFactor == 0 {
		return 0
	}

	// 更新季节相位
	g.seasonalPhase += 0.001 // 缓慢变化
	if g.seasonalPhase > 2*math.Pi {
		g.seasonalPhase -= 2 * math.Pi
	}

	return g.seasonalFactor * math.Sin(g.seasonalPhase)
}

// calculateNoise 计算噪声
func (g *SmartDataGenerator) calculateNoise() float64 {
	if g.noiseLevel == 0 {
		return 0
	}

	return g.random.NormFloat64() * g.noiseLevel
}

// calculateAnomaly 计算异常值
func (g *SmartDataGenerator) calculateAnomaly() float64 {
	if g.random.Float64() > g.anomalyRate {
		return 0
	}

	// 生成异常值
	anomalyTypes := []string{"spike", "drop", "drift"}
	anomalyType := anomalyTypes[g.random.Intn(len(anomalyTypes))]

	switch anomalyType {
	case "spike":
		return g.variance * (2.0 + g.random.Float64()*3.0) // 2-5倍方差的正向异常
	case "drop":
		return -g.variance * (2.0 + g.random.Float64()*3.0) // 2-5倍方差的负向异常
	case "drift":
		return g.variance * g.random.NormFloat64() * 1.5 // 1.5倍方差的随机漂移
	default:
		return 0
	}
}

// calculateDrift 计算传感器漂移
func (g *SmartDataGenerator) calculateDrift(now time.Time) float64 {
	if g.driftRate == 0 {
		return 0
	}

	// 计算自上次校准以来的时间
	timeSinceCalibration := now.Sub(g.calibrationTime).Hours()

	// 线性漂移
	drift := g.driftRate * timeSinceCalibration / 24.0 // 每天的漂移量

	return drift
}

// calculateDegradation 计算设备老化影响
func (g *SmartDataGenerator) calculateDegradation(now time.Time) float64 {
	if g.degradationRate == 0 {
		return 0
	}

	// 计算设备使用时间（假设从校准时间开始）
	usageHours := now.Sub(g.calibrationTime).Hours()

	// 非线性老化影响
	degradation := g.degradationRate * math.Sqrt(usageHours/24.0) * g.random.NormFloat64() * 0.1

	return degradation
}

// applySmoothTransition 应用平滑过渡
func (g *SmartDataGenerator) applySmoothTransition(newValue float64) float64 {
	// 使用指数移动平均进行平滑
	alpha := 0.3 // 平滑因子
	return alpha*newValue + (1-alpha)*g.lastValue
}

// applyConstraints 应用约束条件
func (g *SmartDataGenerator) applyConstraints(value float64) float64 {
	if value < g.minValue {
		return g.minValue
	}
	if value > g.maxValue {
		return g.maxValue
	}
	return value
}

// applyPrecision 应用精度设置
func (g *SmartDataGenerator) applyPrecision(value float64) float64 {
	if g.precision <= 0 {
		return value
	}

	multiplier := math.Pow(10, float64(g.precision))
	return math.Round(value*multiplier) / multiplier
}

// updateHistory 更新历史记录
func (g *SmartDataGenerator) updateHistory(value float64) {
	g.valueHistory = append(g.valueHistory, value)

	if len(g.valueHistory) > g.historySize {
		g.valueHistory = g.valueHistory[1:]
	}
}

// GetStatistics 获取统计信息
func (g *SmartDataGenerator) GetStatistics() SensorStatistics {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if len(g.valueHistory) == 0 {
		return SensorStatistics{}
	}

	// 计算统计指标
	sum := 0.0
	min := g.valueHistory[0]
	max := g.valueHistory[0]

	for _, value := range g.valueHistory {
		sum += value
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}

	mean := sum / float64(len(g.valueHistory))

	// 计算标准差
	variance := 0.0
	for _, value := range g.valueHistory {
		variance += math.Pow(value-mean, 2)
	}
	variance /= float64(len(g.valueHistory))
	stdDev := math.Sqrt(variance)

	return SensorStatistics{
		Mean:     mean,
		Min:      min,
		Max:      max,
		StdDev:   stdDev,
		Count:    len(g.valueHistory),
		Current:  g.lastValue,
		Trend:    g.currentTrend,
		Seasonal: g.seasonalPhase,
	}
}

// SensorStatistics 传感器统计信息
type SensorStatistics struct {
	Mean     float64 `json:"mean"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	StdDev   float64 `json:"std_dev"`
	Count    int     `json:"count"`
	Current  float64 `json:"current"`
	Trend    float64 `json:"trend"`
	Seasonal float64 `json:"seasonal"`
}

// Calibrate 校准传感器
func (g *SmartDataGenerator) Calibrate() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.calibrationTime = time.Now()
	g.currentTrend = 0.0
	g.seasonalPhase = 0.0
}

// Reset 重置生成器状态
func (g *SmartDataGenerator) Reset() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.currentTrend = 0.0
	g.trendDirection = 1
	g.seasonalPhase = 0.0
	g.lastValue = g.baseValue
	g.valueHistory = g.valueHistory[:0]
	g.calibrationTime = time.Now()
}

// GetSensorType 获取传感器类型
func (g *SmartDataGenerator) GetSensorType() string {
	return g.sensorType
}

// GetCurrentValue 获取当前值
func (g *SmartDataGenerator) GetCurrentValue() float64 {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.lastValue
}

// IsHealthy 检查传感器健康状态
func (g *SmartDataGenerator) IsHealthy() bool {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	// 简单的健康检查逻辑
	if len(g.valueHistory) < 5 {
		return true
	}

	// 检查是否有异常的值变化
	recent := g.valueHistory[len(g.valueHistory)-5:]
	for i := 1; i < len(recent); i++ {
		change := math.Abs(recent[i] - recent[i-1])
		if change > g.variance*5 { // 变化超过5倍方差认为异常
			return false
		}
	}

	return true
}
