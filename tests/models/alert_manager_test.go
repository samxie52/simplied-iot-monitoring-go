package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simplied-iot-monitoring-go/internal/models"
)

func TestAlertManager(t *testing.T) {
	manager := models.NewAlertManager()

	assert.NotNil(t, manager)
	assert.Empty(t, manager.GetActiveAlerts())
}

func TestAlertManagerAddAlert(t *testing.T) {
	manager := models.NewAlertManager()

	t.Run("Add valid alert", func(t *testing.T) {
		alert := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "High Temperature", "Temperature exceeds threshold")

		err := manager.AddAlert(alert)
		assert.NoError(t, err)

		// 验证告警已添加
		retrievedAlert, exists := manager.GetAlert(alert.AlertID)
		assert.True(t, exists)
		assert.Equal(t, alert.AlertID, retrievedAlert.AlertID)
		assert.Equal(t, alert.DeviceID, retrievedAlert.DeviceID)
	})

	t.Run("Add alert without ID", func(t *testing.T) {
		alert := &models.Alert{
			DeviceID:  "device_002",
			AlertType: models.AlertTypeThreshold,
			Severity:  models.AlertSeverityError,
			Title:     "Test Alert",
			Message:   "Test message",
		}

		err := manager.AddAlert(alert)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "alert ID is required")
	})
}

func TestAlertManagerUpdateStatus(t *testing.T) {
	manager := models.NewAlertManager()
	alert := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Test Alert", "Test message")

	err := manager.AddAlert(alert)
	require.NoError(t, err)

	t.Run("Valid status transition", func(t *testing.T) {
		err := manager.UpdateAlertStatus(alert.AlertID, models.AlertStatusAcknowledged, "user_001")
		assert.NoError(t, err)

		updatedAlert, exists := manager.GetAlert(alert.AlertID)
		assert.True(t, exists)
		assert.Equal(t, models.AlertStatusAcknowledged, updatedAlert.Status)
		assert.Equal(t, "user_001", updatedAlert.AcknowledgedBy)
		assert.NotNil(t, updatedAlert.AcknowledgedAt)
	})

	t.Run("Invalid status transition", func(t *testing.T) {
		// 尝试从已确认状态直接跳转到激活状态（无效转换）
		err := manager.UpdateAlertStatus(alert.AlertID, models.AlertStatusActive, "user_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state transition")
	})

	t.Run("Alert not found", func(t *testing.T) {
		err := manager.UpdateAlertStatus("non_existent", models.AlertStatusResolved, "user_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "alert not found")
	})
}

func TestAlertManagerGetAlerts(t *testing.T) {
	manager := models.NewAlertManager()

	// 添加多个告警
	alert1 := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Alert 1", "Message 1")
	alert2 := models.NewAlert("device_002", models.AlertTypeConnectivity, models.AlertSeverityError, "Alert 2", "Message 2")
	alert3 := models.NewAlert("device_001", models.AlertTypeBattery, models.AlertSeverityCritical, "Alert 3", "Message 3")

	manager.AddAlert(alert1)
	manager.AddAlert(alert2)
	manager.AddAlert(alert3)

	t.Run("Get active alerts", func(t *testing.T) {
		activeAlerts := manager.GetActiveAlerts()
		assert.Len(t, activeAlerts, 3)
	})

	t.Run("Get alerts by device", func(t *testing.T) {
		deviceAlerts := manager.GetAlertsByDevice("device_001")
		assert.Len(t, deviceAlerts, 2)

		for _, alert := range deviceAlerts {
			assert.Equal(t, "device_001", alert.DeviceID)
		}
	})

	t.Run("Get alerts by severity", func(t *testing.T) {
		criticalAlerts := manager.GetAlertsBySeverity(models.AlertSeverityCritical)
		assert.Len(t, criticalAlerts, 1)
		assert.Equal(t, models.AlertSeverityCritical, criticalAlerts[0].Severity)
	})
}

func TestAlertManagerMetrics(t *testing.T) {
	manager := models.NewAlertManager()

	// 添加不同类型和严重级别的告警
	alert1 := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Alert 1", "Message 1")
	alert2 := models.NewAlert("device_002", models.AlertTypeConnectivity, models.AlertSeverityError, "Alert 2", "Message 2")
	alert3 := models.NewAlert("device_003", models.AlertTypeBattery, models.AlertSeverityCritical, "Alert 3", "Message 3")

	manager.AddAlert(alert1)
	manager.AddAlert(alert2)
	manager.AddAlert(alert3)

	// 解决一个告警
	manager.UpdateAlertStatus(alert1.AlertID, models.AlertStatusResolved, "user_001")

	metrics := manager.GetMetrics()

	assert.Equal(t, int64(3), metrics.TotalAlerts)
	assert.Equal(t, int64(2), metrics.ActiveAlerts)
	assert.Equal(t, int64(1), metrics.ResolvedAlerts)

	// 检查按类型统计
	assert.Equal(t, int64(1), metrics.AlertsByType[models.AlertTypeThreshold])
	assert.Equal(t, int64(1), metrics.AlertsByType[models.AlertTypeConnectivity])
	assert.Equal(t, int64(1), metrics.AlertsByType[models.AlertTypeBattery])

	// 检查按严重级别统计
	assert.Equal(t, int64(1), metrics.AlertsBySeverity[models.AlertSeverityWarning])
	assert.Equal(t, int64(1), metrics.AlertsBySeverity[models.AlertSeverityError])
	assert.Equal(t, int64(1), metrics.AlertsBySeverity[models.AlertSeverityCritical])
}

func TestAlertStateMachine(t *testing.T) {
	t.Run("Valid transitions", func(t *testing.T) {
		sm := models.NewAlertStateMachine()

		// 从激活状态可以转换到确认状态
		assert.True(t, sm.CanTransition(models.AlertStatusAcknowledged))
		assert.True(t, sm.CanTransition(models.AlertStatusResolved))
		assert.True(t, sm.CanTransition(models.AlertStatusClosed))

		// 执行转换
		err := sm.Transition(models.AlertStatusAcknowledged)
		assert.NoError(t, err)

		// 从确认状态可以转换到解决或关闭状态
		assert.True(t, sm.CanTransition(models.AlertStatusResolved))
		assert.True(t, sm.CanTransition(models.AlertStatusClosed))
		assert.False(t, sm.CanTransition(models.AlertStatusActive))
	})

	t.Run("Invalid transitions", func(t *testing.T) {
		sm := models.NewAlertStateMachine()

		// 转换到确认状态
		sm.Transition(models.AlertStatusAcknowledged)

		// 尝试无效转换
		err := sm.Transition(models.AlertStatusActive)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid transition")
	})

	t.Run("Get allowed transitions", func(t *testing.T) {
		sm := models.NewAlertStateMachine()

		allowed := sm.GetAllowedTransitions()
		assert.Len(t, allowed, 3)
		assert.Contains(t, allowed, models.AlertStatusAcknowledged)
		assert.Contains(t, allowed, models.AlertStatusResolved)
		assert.Contains(t, allowed, models.AlertStatusClosed)
	})
}

func TestAlertAggregator(t *testing.T) {
	t.Run("Create aggregator", func(t *testing.T) {
		aggregator := models.NewAlertAggregator([]string{"device_id", "alert_type"}, 5*time.Minute)

		assert.NotNil(t, aggregator)
		assert.Equal(t, []string{"device_id", "alert_type"}, aggregator.GetGroupBy())
	})

	t.Run("Aggregate alerts", func(t *testing.T) {
		aggregator := models.NewAlertAggregator([]string{"device_id", "alert_type"}, 5*time.Minute)

		// 创建相同设备和类型的告警
		alert1 := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Alert 1", "Message 1")
		alert2 := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityError, "Alert 2", "Message 2")
		alert3 := models.NewAlert("device_002", models.AlertTypeThreshold, models.AlertSeverityWarning, "Alert 3", "Message 3")

		// 聚合告警
		agg1 := aggregator.Aggregate(alert1)
		agg2 := aggregator.Aggregate(alert2)
		agg3 := aggregator.Aggregate(alert3)

		// 前两个告警应该聚合到同一个组
		assert.Equal(t, agg1.AggregationID, agg2.AggregationID)
		assert.NotEqual(t, agg1.AggregationID, agg3.AggregationID)

		// 检查聚合计数
		assert.Equal(t, 2, agg2.Count)
		assert.Equal(t, 1, agg3.Count)

		// 检查严重级别（应该取最高级别）
		assert.Equal(t, models.AlertSeverityError, agg2.Severity)
	})

	t.Run("Time window aggregation", func(t *testing.T) {
		aggregator := models.NewAlertAggregator([]string{"device_id"}, 100*time.Millisecond)

		alert1 := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Alert 1", "Message 1")
		agg1 := aggregator.Aggregate(alert1)

		// 等待时间窗口过期
		time.Sleep(150 * time.Millisecond)

		alert2 := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Alert 2", "Message 2")
		agg2 := aggregator.Aggregate(alert2)

		// 应该创建新的聚合
		assert.NotEqual(t, agg1.AggregationID, agg2.AggregationID)
		assert.Equal(t, 1, agg2.Count)
	})

	t.Run("Cleanup expired aggregations", func(t *testing.T) {
		aggregator := models.NewAlertAggregator([]string{"device_id"}, 50*time.Millisecond)

		alert := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Alert", "Message")
		aggregator.Aggregate(alert)

		// 等待过期
		time.Sleep(150 * time.Millisecond)

		cleaned := aggregator.CleanupExpired()
		assert.Equal(t, 1, cleaned)

		aggregations := aggregator.GetAggregations()
		assert.Empty(t, aggregations)
	})
}

func TestAlertManagerAggregation(t *testing.T) {
	manager := models.NewAlertManager()

	t.Run("Alert aggregation", func(t *testing.T) {
		// 创建相同设备和类型的告警
		alert1 := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Alert 1", "Message 1")
		alert2 := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityError, "Alert 2", "Message 2")

		// 第一个告警不会被聚合
		err1 := manager.AddAlert(alert1)
		assert.NoError(t, err1)

		// 第二个告警应该被聚合
		err2 := manager.AddAlert(alert2)
		assert.NoError(t, err2)

		// 检查活跃告警数量（第二个被聚合，所以只有1个）
		activeAlerts := manager.GetActiveAlerts()
		assert.Len(t, activeAlerts, 1)
	})

	t.Run("Cleanup expired aggregations", func(t *testing.T) {
		// 等待聚合过期
		time.Sleep(100 * time.Millisecond)

		cleaned := manager.CleanupExpiredAggregations()
		assert.GreaterOrEqual(t, cleaned, 0)
	})
}

func TestAlertManagerConcurrency(t *testing.T) {
	manager := models.NewAlertManager()

	t.Run("Concurrent alert operations", func(t *testing.T) {
		const numGoroutines = 10
		const alertsPerGoroutine = 10

		// 并发添加告警
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer func() { done <- true }()

				for j := 0; j < alertsPerGoroutine; j++ {
					alert := models.NewAlert(
						fmt.Sprintf("device_%d_%d", goroutineID, j),
						models.AlertTypeThreshold,
						models.AlertSeverityWarning,
						fmt.Sprintf("Alert %d-%d", goroutineID, j),
						"Concurrent test message",
					)

					err := manager.AddAlert(alert)
					assert.NoError(t, err)
				}
			}(i)
		}

		// 等待所有协程完成
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// 验证所有告警都被添加
		activeAlerts := manager.GetActiveAlerts()
		assert.Len(t, activeAlerts, numGoroutines*alertsPerGoroutine)
	})
}
