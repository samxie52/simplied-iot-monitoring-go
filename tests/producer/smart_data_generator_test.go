package producer

import (
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/producer"
)

func TestSmartDataGenerator_Creation(t *testing.T) {
	config := producer.SensorGeneratorConfig{
		BaseValue:       25.0,
		Variance:        2.0,
		TrendFactor:     0.1,
		SeasonalFactor:  0.5,
		AnomalyRate:     0.01,
		NoiseLevel:      0.1,
		HistorySize:     100,
		MinValue:        -10.0,
		MaxValue:        50.0,
		Unit:            "°C",
		Precision:       2,
		DriftRate:       0.001,
		DegradationRate: 0.0001,
	}

	generator := producer.NewSmartDataGenerator("temperature", config)

	if generator == nil {
		t.Fatal("Failed to create smart data generator")
	}

	if generator.GetSensorType() != "temperature" {
		t.Errorf("Expected sensor type 'temperature', got '%s'", generator.GetSensorType())
	}

	if generator.GetCurrentValue() != 25.0 {
		t.Errorf("Expected initial value 25.0, got %f", generator.GetCurrentValue())
	}
}

func TestSmartDataGenerator_ValueGeneration(t *testing.T) {
	config := producer.SensorGeneratorConfig{
		BaseValue:       20.0,
		Variance:        1.0,
		TrendFactor:     0.05,
		SeasonalFactor:  0.2,
		AnomalyRate:     0.0, // 禁用异常以便测试
		NoiseLevel:      0.05,
		HistorySize:     50,
		MinValue:        0.0,
		MaxValue:        40.0,
		Unit:            "°C",
		Precision:       1,
		DriftRate:       0.0,
		DegradationRate: 0.0,
	}

	generator := producer.NewSmartDataGenerator("temperature", config)

	// 生成一系列值
	values := make([]float64, 100)
	for i := 0; i < 100; i++ {
		values[i] = generator.GenerateValue()
		time.Sleep(time.Millisecond) // 模拟时间流逝
	}

	// 检查值是否在合理范围内
	for i, value := range values {
		if value < config.MinValue || value > config.MaxValue {
			t.Errorf("Value %d (%f) is out of range [%f, %f]", i, value, config.MinValue, config.MaxValue)
		}
	}

	// 检查精度
	for i, value := range values {
		rounded := float64(int(value*10)) / 10 // 1位小数精度
		if value != rounded {
			t.Errorf("Value %d (%f) doesn't match precision requirement", i, value)
		}
	}
}

func TestSmartDataGenerator_Statistics(t *testing.T) {
	config := producer.SensorGeneratorConfig{
		BaseValue:       25.0,
		Variance:        2.0,
		TrendFactor:     0.0,
		SeasonalFactor:  0.0,
		AnomalyRate:     0.0,
		NoiseLevel:      0.1,
		HistorySize:     20,
		MinValue:        0.0,
		MaxValue:        50.0,
		Unit:            "°C",
		Precision:       2,
		DriftRate:       0.0,
		DegradationRate: 0.0,
	}

	generator := producer.NewSmartDataGenerator("temperature", config)

	// 生成一些值
	for i := 0; i < 15; i++ {
		generator.GenerateValue()
		time.Sleep(time.Millisecond)
	}

	stats := generator.GetStatistics()

	if stats.Count != 15 {
		t.Errorf("Expected count 15, got %d", stats.Count)
	}

	if stats.Mean < 20.0 || stats.Mean > 30.0 {
		t.Errorf("Mean %f is not in expected range [20.0, 30.0]", stats.Mean)
	}

	if stats.Min > stats.Max {
		t.Errorf("Min (%f) should not be greater than Max (%f)", stats.Min, stats.Max)
	}

	if stats.StdDev < 0 {
		t.Errorf("Standard deviation should not be negative: %f", stats.StdDev)
	}
}

func TestSmartDataGenerator_HealthCheck(t *testing.T) {
	config := producer.SensorGeneratorConfig{
		BaseValue:       25.0,
		Variance:        1.0,
		TrendFactor:     0.0,
		SeasonalFactor:  0.0,
		AnomalyRate:     0.0,
		NoiseLevel:      0.1,
		HistorySize:     10,
		MinValue:        0.0,
		MaxValue:        50.0,
		Unit:            "°C",
		Precision:       2,
		DriftRate:       0.0,
		DegradationRate: 0.0,
	}

	generator := producer.NewSmartDataGenerator("temperature", config)

	// 生成正常值
	for i := 0; i < 10; i++ {
		generator.GenerateValue()
		time.Sleep(time.Millisecond)
	}

	if !generator.IsHealthy() {
		t.Error("Generator should be healthy with normal values")
	}
}

func TestSmartDataGenerator_Calibration(t *testing.T) {
	config := producer.SensorGeneratorConfig{
		BaseValue:       25.0,
		Variance:        2.0,
		TrendFactor:     0.2,
		SeasonalFactor:  0.3,
		AnomalyRate:     0.0,
		NoiseLevel:      0.1,
		HistorySize:     10,
		MinValue:        0.0,
		MaxValue:        50.0,
		Unit:            "°C",
		Precision:       2,
		DriftRate:       0.001,
		DegradationRate: 0.0001,
	}

	generator := producer.NewSmartDataGenerator("temperature", config)

	// 生成一些值以建立趋势和季节性
	for i := 0; i < 20; i++ {
		generator.GenerateValue()
		time.Sleep(time.Millisecond)
	}

	statsBefore := generator.GetStatistics()

	// 校准传感器
	generator.Calibrate()

	// 校准后趋势和季节性应该重置
	statsAfter := generator.GetStatistics()

	if statsAfter.Trend != 0.0 {
		t.Errorf("Trend should be reset to 0 after calibration, got %f", statsAfter.Trend)
	}

	if statsAfter.Seasonal != 0.0 {
		t.Errorf("Seasonal phase should be reset to 0 after calibration, got %f", statsAfter.Seasonal)
	}

	// 历史记录应该保持
	if statsAfter.Count != statsBefore.Count {
		t.Errorf("History count should be preserved after calibration")
	}
}

func TestSmartDataGenerator_Reset(t *testing.T) {
	config := producer.SensorGeneratorConfig{
		BaseValue:       25.0,
		Variance:        2.0,
		TrendFactor:     0.2,
		SeasonalFactor:  0.3,
		AnomalyRate:     0.0,
		NoiseLevel:      0.1,
		HistorySize:     10,
		MinValue:        0.0,
		MaxValue:        50.0,
		Unit:            "°C",
		Precision:       2,
		DriftRate:       0.001,
		DegradationRate: 0.0001,
	}

	generator := producer.NewSmartDataGenerator("temperature", config)

	// 生成一些值
	for i := 0; i < 15; i++ {
		generator.GenerateValue()
		time.Sleep(time.Millisecond)
	}

	// 重置生成器
	generator.Reset()

	stats := generator.GetStatistics()

	if stats.Count != 0 {
		t.Errorf("History count should be 0 after reset, got %d", stats.Count)
	}

	if generator.GetCurrentValue() != config.BaseValue {
		t.Errorf("Current value should be reset to base value (%f), got %f", 
			config.BaseValue, generator.GetCurrentValue())
	}
}

func TestSmartDataGenerator_DifferentSensorTypes(t *testing.T) {
	sensorTypes := []string{"temperature", "humidity", "pressure", "current"}
	
	for _, sensorType := range sensorTypes {
		t.Run(sensorType, func(t *testing.T) {
			config := producer.SensorGeneratorConfig{
				BaseValue:       50.0,
				Variance:        5.0,
				TrendFactor:     0.1,
				SeasonalFactor:  0.2,
				AnomalyRate:     0.0,
				NoiseLevel:      0.1,
				HistorySize:     20,
				MinValue:        0.0,
				MaxValue:        100.0,
				Unit:            "unit",
				Precision:       2,
				DriftRate:       0.0,
				DegradationRate: 0.0,
			}

			generator := producer.NewSmartDataGenerator(sensorType, config)

			if generator.GetSensorType() != sensorType {
				t.Errorf("Expected sensor type '%s', got '%s'", sensorType, generator.GetSensorType())
			}

			// 生成一些值并检查
			values := make([]float64, 10)
			for i := 0; i < 10; i++ {
				values[i] = generator.GenerateValue()
				time.Sleep(time.Millisecond)
			}

			// 检查值是否合理
			for i, value := range values {
				if value < config.MinValue || value > config.MaxValue {
					t.Errorf("Value %d (%f) is out of range for sensor type %s", i, value, sensorType)
				}
			}
		})
	}
}

func TestSmartDataGenerator_AnomalyGeneration(t *testing.T) {
	config := producer.SensorGeneratorConfig{
		BaseValue:       25.0,
		Variance:        1.0,
		TrendFactor:     0.0,
		SeasonalFactor:  0.0,
		AnomalyRate:     1.0, // 100%异常率用于测试
		NoiseLevel:      0.0,
		HistorySize:     10,
		MinValue:        -100.0,
		MaxValue:        100.0,
		Unit:            "°C",
		Precision:       2,
		DriftRate:       0.0,
		DegradationRate: 0.0,
	}

	generator := producer.NewSmartDataGenerator("temperature", config)

	// 生成值，应该包含异常
	anomalyDetected := false
	for i := 0; i < 20; i++ {
		value := generator.GenerateValue()
		// 如果值偏离基础值超过3倍方差，认为是异常
		if value < config.BaseValue-3*config.Variance || value > config.BaseValue+3*config.Variance {
			anomalyDetected = true
			break
		}
		time.Sleep(time.Millisecond)
	}

	if !anomalyDetected {
		t.Error("Expected to detect anomalies with 100% anomaly rate")
	}
}

func BenchmarkSmartDataGenerator_GenerateValue(b *testing.B) {
	config := producer.SensorGeneratorConfig{
		BaseValue:       25.0,
		Variance:        2.0,
		TrendFactor:     0.1,
		SeasonalFactor:  0.2,
		AnomalyRate:     0.01,
		NoiseLevel:      0.1,
		HistorySize:     100,
		MinValue:        0.0,
		MaxValue:        50.0,
		Unit:            "°C",
		Precision:       2,
		DriftRate:       0.001,
		DegradationRate: 0.0001,
	}

	generator := producer.NewSmartDataGenerator("temperature", config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.GenerateValue()
	}
}

func BenchmarkSmartDataGenerator_Concurrent(b *testing.B) {
	config := producer.SensorGeneratorConfig{
		BaseValue:       25.0,
		Variance:        2.0,
		TrendFactor:     0.1,
		SeasonalFactor:  0.2,
		AnomalyRate:     0.01,
		NoiseLevel:      0.1,
		HistorySize:     100,
		MinValue:        0.0,
		MaxValue:        50.0,
		Unit:            "°C",
		Precision:       2,
		DriftRate:       0.001,
		DegradationRate: 0.0001,
	}

	generator := producer.NewSmartDataGenerator("temperature", config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			generator.GenerateValue()
		}
	})
}
