package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simplied-iot-monitoring-go/internal/models"
)

func TestNewAlert(t *testing.T) {
	t.Run("Create new alert", func(t *testing.T) {
		deviceID := "device_001"
		alertType := models.AlertTypeThreshold
		severity := models.AlertSeverityWarning
		title := "High Temperature"
		message := "Temperature exceeds threshold"

		alert := models.NewAlert(deviceID, alertType, severity, title, message)

		assert.NotEmpty(t, alert.AlertID)
		assert.Equal(t, deviceID, alert.DeviceID)
		assert.Equal(t, alertType, alert.AlertType)
		assert.Equal(t, severity, alert.Severity)
		assert.Equal(t, title, alert.Title)
		assert.Equal(t, message, alert.Message)
		assert.Equal(t, models.AlertStatusActive, alert.Status)
		assert.Equal(t, 1, alert.OccurrenceCount)
		assert.True(t, alert.Timestamp > 0)
		assert.True(t, alert.FirstOccurrence > 0)
		assert.True(t, alert.LastOccurrence > 0)
		assert.NotNil(t, alert.Tags)
		assert.NotNil(t, alert.Metadata)
	})
}

func TestAlertStatusMethods(t *testing.T) {
	alert := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Test", "Test message")

	t.Run("Initial status", func(t *testing.T) {
		assert.True(t, alert.IsActive())
		assert.False(t, alert.IsResolved())
		assert.Equal(t, models.AlertStatusActive, alert.Status)
	})

	t.Run("Acknowledge alert", func(t *testing.T) {
		userID := "user123"
		alert.Acknowledge(userID)

		assert.Equal(t, models.AlertStatusAcknowledged, alert.Status)
		assert.Equal(t, userID, alert.AcknowledgedBy)
		assert.NotNil(t, alert.AcknowledgedAt)
		assert.False(t, alert.IsActive())
		assert.False(t, alert.IsResolved())
	})

	t.Run("Resolve alert", func(t *testing.T) {
		userID := "user456"
		alert.Resolve(userID)

		assert.Equal(t, models.AlertStatusResolved, alert.Status)
		assert.Equal(t, userID, alert.ResolvedBy)
		assert.NotNil(t, alert.ResolvedAt)
		assert.False(t, alert.IsActive())
		assert.True(t, alert.IsResolved())
	})

	t.Run("Close alert", func(t *testing.T) {
		alert.Close()

		assert.Equal(t, models.AlertStatusClosed, alert.Status)
		assert.True(t, alert.IsResolved())
	})
}

func TestAlertOccurrence(t *testing.T) {
	alert := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Test", "Test message")

	initialCount := alert.OccurrenceCount
	initialLastOccurrence := alert.LastOccurrence

	time.Sleep(1100 * time.Millisecond) // 确保时间戳不同（秒级精度）

	alert.IncrementOccurrence()

	assert.Equal(t, initialCount+1, alert.OccurrenceCount)
	assert.True(t, alert.LastOccurrence > initialLastOccurrence)
}

func TestAlertThreshold(t *testing.T) {
	alert := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Test", "Test message")

	t.Run("Set threshold", func(t *testing.T) {
		alert.SetThreshold("temperature", "gt", 75.0, "°C")

		require.NotNil(t, alert.Threshold)
		assert.Equal(t, "temperature", alert.Threshold.Field)
		assert.Equal(t, "gt", alert.Threshold.Operator)
		assert.Equal(t, 75.0, alert.Threshold.Value)
		assert.Equal(t, "°C", alert.Threshold.Unit)
	})

	t.Run("Set current value", func(t *testing.T) {
		alert.SetCurrentValue(78.5)
		assert.Equal(t, 78.5, alert.CurrentValue)
	})
}

func TestAlertTags(t *testing.T) {
	alert := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Test", "Test message")

	t.Run("Add tags", func(t *testing.T) {
		alert.AddTag("production")
		alert.AddTag("critical-system")
		alert.AddTag("production") // 重复标签

		assert.Len(t, alert.Tags, 2)
		assert.Contains(t, alert.Tags, "production")
		assert.Contains(t, alert.Tags, "critical-system")
	})

	t.Run("Remove tag", func(t *testing.T) {
		alert.RemoveTag("production")

		assert.Len(t, alert.Tags, 1)
		assert.NotContains(t, alert.Tags, "production")
		assert.Contains(t, alert.Tags, "critical-system")
	})

	t.Run("Remove non-existent tag", func(t *testing.T) {
		initialLen := len(alert.Tags)
		alert.RemoveTag("non-existent")

		assert.Len(t, alert.Tags, initialLen)
	})
}

func TestAlertMetadata(t *testing.T) {
	alert := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Test", "Test message")

	t.Run("Set and get metadata", func(t *testing.T) {
		alert.SetMetadata("location", "Building A")
		alert.SetMetadata("priority", 8)
		alert.SetMetadata("auto_resolve", true)

		location, exists := alert.GetMetadata("location")
		assert.True(t, exists)
		assert.Equal(t, "Building A", location)

		priority, exists := alert.GetMetadata("priority")
		assert.True(t, exists)
		assert.Equal(t, 8, priority)

		autoResolve, exists := alert.GetMetadata("auto_resolve")
		assert.True(t, exists)
		assert.Equal(t, true, autoResolve)

		_, exists = alert.GetMetadata("non-existent")
		assert.False(t, exists)
	})
}

func TestAlertAge(t *testing.T) {
	alert := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "Test", "Test message")

	time.Sleep(1 * time.Second)

	age := alert.GetAge()
	assert.True(t, age >= 1)
	assert.True(t, age <= 2) // 应该在1-2秒之间
}

func TestAlertSerialization(t *testing.T) {
	t.Run("JSON serialization", func(t *testing.T) {
		alert := models.NewAlert("device_001", models.AlertTypeThreshold, models.AlertSeverityWarning, "High Temperature", "Temperature exceeds threshold")
		alert.SetThreshold("temperature", "gt", 75.0, "°C")
		alert.SetCurrentValue(78.5)
		alert.AddTag("production")
		alert.SetMetadata("location", "Building A")

		// 序列化
		data, err := alert.ToJSON()
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// 反序列化
		var deserializedAlert models.Alert
		err = deserializedAlert.FromJSON(data)
		require.NoError(t, err)

		assert.Equal(t, alert.AlertID, deserializedAlert.AlertID)
		assert.Equal(t, alert.DeviceID, deserializedAlert.DeviceID)
		assert.Equal(t, alert.AlertType, deserializedAlert.AlertType)
		assert.Equal(t, alert.Severity, deserializedAlert.Severity)
		assert.Equal(t, alert.Title, deserializedAlert.Title)
		assert.Equal(t, alert.Message, deserializedAlert.Message)
		assert.Equal(t, alert.Status, deserializedAlert.Status)
		assert.Equal(t, alert.CurrentValue, deserializedAlert.CurrentValue)

		// 检查阈值
		require.NotNil(t, deserializedAlert.Threshold)
		assert.Equal(t, alert.Threshold.Field, deserializedAlert.Threshold.Field)
		assert.Equal(t, alert.Threshold.Operator, deserializedAlert.Threshold.Operator)
		assert.Equal(t, alert.Threshold.Value, deserializedAlert.Threshold.Value)
		assert.Equal(t, alert.Threshold.Unit, deserializedAlert.Threshold.Unit)

		// 检查标签和元数据
		assert.Equal(t, alert.Tags, deserializedAlert.Tags)
		location, exists := deserializedAlert.GetMetadata("location")
		assert.True(t, exists)
		assert.Equal(t, "Building A", location)
	})
}

func TestAlertValidation(t *testing.T) {
	tests := []struct {
		name          string
		alert         *models.Alert
		expectedError string
	}{
		{
			name: "Valid alert",
			alert: &models.Alert{
				AlertID:   "alert-123",
				DeviceID:  "device-001",
				AlertType: models.AlertTypeThreshold,
				Severity:  models.AlertSeverityWarning,
				Title:     "Test Alert",
				Message:   "Test message",
				Timestamp: time.Now().Unix(),
				Status:    models.AlertStatusActive,
			},
			expectedError: "",
		},
		{
			name: "Missing alert ID",
			alert: &models.Alert{
				DeviceID:  "device-001",
				AlertType: models.AlertTypeThreshold,
				Severity:  models.AlertSeverityWarning,
				Title:     "Test Alert",
				Message:   "Test message",
				Timestamp: time.Now().Unix(),
				Status:    models.AlertStatusActive,
			},
			expectedError: "alert_id is required",
		},
		{
			name: "Missing device ID",
			alert: &models.Alert{
				AlertID:   "alert-123",
				AlertType: models.AlertTypeThreshold,
				Severity:  models.AlertSeverityWarning,
				Title:     "Test Alert",
				Message:   "Test message",
				Timestamp: time.Now().Unix(),
				Status:    models.AlertStatusActive,
			},
			expectedError: "device_id is required",
		},
		{
			name: "Missing alert type",
			alert: &models.Alert{
				AlertID:   "alert-123",
				DeviceID:  "device-001",
				Severity:  models.AlertSeverityWarning,
				Title:     "Test Alert",
				Message:   "Test message",
				Timestamp: time.Now().Unix(),
				Status:    models.AlertStatusActive,
			},
			expectedError: "alert_type is required",
		},
		{
			name: "Invalid severity",
			alert: &models.Alert{
				AlertID:   "alert-123",
				DeviceID:  "device-001",
				AlertType: models.AlertTypeThreshold,
				Severity:  "invalid",
				Title:     "Test Alert",
				Message:   "Test message",
				Timestamp: time.Now().Unix(),
				Status:    models.AlertStatusActive,
			},
			expectedError: "invalid severity",
		},
		{
			name: "Invalid status",
			alert: &models.Alert{
				AlertID:   "alert-123",
				DeviceID:  "device-001",
				AlertType: models.AlertTypeThreshold,
				Severity:  models.AlertSeverityWarning,
				Title:     "Test Alert",
				Message:   "Test message",
				Timestamp: time.Now().Unix(),
				Status:    "invalid",
			},
			expectedError: "invalid status",
		},
		{
			name: "Missing title",
			alert: &models.Alert{
				AlertID:   "alert-123",
				DeviceID:  "device-001",
				AlertType: models.AlertTypeThreshold,
				Severity:  models.AlertSeverityWarning,
				Message:   "Test message",
				Timestamp: time.Now().Unix(),
				Status:    models.AlertStatusActive,
			},
			expectedError: "title is required",
		},
		{
			name: "Missing message",
			alert: &models.Alert{
				AlertID:   "alert-123",
				DeviceID:  "device-001",
				AlertType: models.AlertTypeThreshold,
				Severity:  models.AlertSeverityWarning,
				Title:     "Test Alert",
				Timestamp: time.Now().Unix(),
				Status:    models.AlertStatusActive,
			},
			expectedError: "message is required",
		},
		{
			name: "Invalid timestamp",
			alert: &models.Alert{
				AlertID:   "alert-123",
				DeviceID:  "device-001",
				AlertType: models.AlertTypeThreshold,
				Severity:  models.AlertSeverityWarning,
				Title:     "Test Alert",
				Message:   "Test message",
				Timestamp: 0,
				Status:    models.AlertStatusActive,
			},
			expectedError: "timestamp must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.alert.Validate()
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestNewAlertRule(t *testing.T) {
	t.Run("Create new alert rule", func(t *testing.T) {
		conditions := []models.AlertCondition{
			{
				Field:    "temperature",
				Operator: "gt",
				Value:    75.0,
				Duration: 300, // 5分钟
			},
		}

		rule := models.NewAlertRule("High Temperature Rule", conditions, models.AlertSeverityWarning, models.AlertTypeThreshold)

		assert.NotEmpty(t, rule.RuleID)
		assert.Equal(t, "High Temperature Rule", rule.Name)
		assert.Equal(t, conditions, rule.Conditions)
		assert.Equal(t, "AND", rule.LogicOp)
		assert.Equal(t, models.AlertSeverityWarning, rule.Severity)
		assert.Equal(t, models.AlertTypeThreshold, rule.AlertType)
		assert.True(t, rule.Enabled)
		assert.NotNil(t, rule.DeviceIDs)
		assert.NotNil(t, rule.DeviceTypes)
		assert.NotNil(t, rule.Tags)
		assert.NotNil(t, rule.NotificationChannels)
	})
}

func TestAlertRuleMethods(t *testing.T) {
	conditions := []models.AlertCondition{
		{Field: "temperature", Operator: "gt", Value: 75.0},
	}
	rule := models.NewAlertRule("Test Rule", conditions, models.AlertSeverityWarning, models.AlertTypeThreshold)

	t.Run("Add and remove conditions", func(t *testing.T) {
		initialCount := len(rule.Conditions)

		newCondition := models.AlertCondition{
			Field:    "humidity",
			Operator: "lt",
			Value:    30.0,
		}

		rule.AddCondition(newCondition)
		assert.Len(t, rule.Conditions, initialCount+1)
		assert.Equal(t, newCondition, rule.Conditions[len(rule.Conditions)-1])

		// 移除条件
		err := rule.RemoveCondition(0)
		assert.NoError(t, err)
		assert.Len(t, rule.Conditions, initialCount)

		// 移除无效索引
		err = rule.RemoveCondition(100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid condition index")
	})

	t.Run("Enable and disable rule", func(t *testing.T) {
		assert.True(t, rule.Enabled)

		rule.Disable()
		assert.False(t, rule.Enabled)

		rule.Enable()
		assert.True(t, rule.Enabled)
	})

	t.Run("Add and remove device IDs", func(t *testing.T) {
		rule.AddDeviceID("device_001")
		rule.AddDeviceID("device_002")
		rule.AddDeviceID("device_001") // 重复

		assert.Len(t, rule.DeviceIDs, 2)
		assert.Contains(t, rule.DeviceIDs, "device_001")
		assert.Contains(t, rule.DeviceIDs, "device_002")

		rule.RemoveDeviceID("device_001")
		assert.Len(t, rule.DeviceIDs, 1)
		assert.NotContains(t, rule.DeviceIDs, "device_001")
		assert.Contains(t, rule.DeviceIDs, "device_002")

		rule.RemoveDeviceID("non-existent")
		assert.Len(t, rule.DeviceIDs, 1) // 没有变化
	})
}

func TestAlertRuleValidation(t *testing.T) {
	tests := []struct {
		name          string
		rule          *models.AlertRule
		expectedError string
	}{
		{
			name: "Valid rule",
			rule: &models.AlertRule{
				RuleID: "rule-123",
				Name:   "Test Rule",
				Conditions: []models.AlertCondition{
					{Field: "temperature", Operator: "gt", Value: 75.0},
				},
				LogicOp:   "AND",
				Severity:  models.AlertSeverityWarning,
				AlertType: models.AlertTypeThreshold,
			},
			expectedError: "",
		},
		{
			name: "Missing rule ID",
			rule: &models.AlertRule{
				Name: "Test Rule",
				Conditions: []models.AlertCondition{
					{Field: "temperature", Operator: "gt", Value: 75.0},
				},
				LogicOp:   "AND",
				Severity:  models.AlertSeverityWarning,
				AlertType: models.AlertTypeThreshold,
			},
			expectedError: "rule_id is required",
		},
		{
			name: "Missing name",
			rule: &models.AlertRule{
				RuleID: "rule-123",
				Conditions: []models.AlertCondition{
					{Field: "temperature", Operator: "gt", Value: 75.0},
				},
				LogicOp:   "AND",
				Severity:  models.AlertSeverityWarning,
				AlertType: models.AlertTypeThreshold,
			},
			expectedError: "name is required",
		},
		{
			name: "No conditions",
			rule: &models.AlertRule{
				RuleID:     "rule-123",
				Name:       "Test Rule",
				Conditions: []models.AlertCondition{},
				LogicOp:    "AND",
				Severity:   models.AlertSeverityWarning,
				AlertType:  models.AlertTypeThreshold,
			},
			expectedError: "at least one condition is required",
		},
		{
			name: "Invalid logic operator",
			rule: &models.AlertRule{
				RuleID: "rule-123",
				Name:   "Test Rule",
				Conditions: []models.AlertCondition{
					{Field: "temperature", Operator: "gt", Value: 75.0},
				},
				LogicOp:   "INVALID",
				Severity:  models.AlertSeverityWarning,
				AlertType: models.AlertTypeThreshold,
			},
			expectedError: "logic_op must be AND or OR",
		},
		{
			name: "Invalid condition - missing field",
			rule: &models.AlertRule{
				RuleID: "rule-123",
				Name:   "Test Rule",
				Conditions: []models.AlertCondition{
					{Operator: "gt", Value: 75.0},
				},
				LogicOp:   "AND",
				Severity:  models.AlertSeverityWarning,
				AlertType: models.AlertTypeThreshold,
			},
			expectedError: "condition[0].field is required",
		},
		{
			name: "Invalid condition - missing operator",
			rule: &models.AlertRule{
				RuleID: "rule-123",
				Name:   "Test Rule",
				Conditions: []models.AlertCondition{
					{Field: "temperature", Value: 75.0},
				},
				LogicOp:   "AND",
				Severity:  models.AlertSeverityWarning,
				AlertType: models.AlertTypeThreshold,
			},
			expectedError: "condition[0].operator is required",
		},
		{
			name: "Invalid condition - missing value",
			rule: &models.AlertRule{
				RuleID: "rule-123",
				Name:   "Test Rule",
				Conditions: []models.AlertCondition{
					{Field: "temperature", Operator: "gt"},
				},
				LogicOp:   "AND",
				Severity:  models.AlertSeverityWarning,
				AlertType: models.AlertTypeThreshold,
			},
			expectedError: "condition[0].value is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestAlertConstants(t *testing.T) {
	t.Run("Alert severity constants", func(t *testing.T) {
		assert.Equal(t, models.AlertSeverity("info"), models.AlertSeverityInfo)
		assert.Equal(t, models.AlertSeverity("warning"), models.AlertSeverityWarning)
		assert.Equal(t, models.AlertSeverity("error"), models.AlertSeverityError)
		assert.Equal(t, models.AlertSeverity("critical"), models.AlertSeverityCritical)
	})

	t.Run("Alert status constants", func(t *testing.T) {
		assert.Equal(t, models.AlertStatus("active"), models.AlertStatusActive)
		assert.Equal(t, models.AlertStatus("acknowledged"), models.AlertStatusAcknowledged)
		assert.Equal(t, models.AlertStatus("resolved"), models.AlertStatusResolved)
		assert.Equal(t, models.AlertStatus("closed"), models.AlertStatusClosed)
	})

	t.Run("Alert type constants", func(t *testing.T) {
		assert.Equal(t, models.AlertType("threshold"), models.AlertTypeThreshold)
		assert.Equal(t, models.AlertType("anomaly"), models.AlertTypeAnomaly)
		assert.Equal(t, models.AlertType("connectivity"), models.AlertTypeConnectivity)
		assert.Equal(t, models.AlertType("system"), models.AlertTypeSystem)
		assert.Equal(t, models.AlertType("battery"), models.AlertTypeBattery)
		assert.Equal(t, models.AlertType("custom"), models.AlertTypeCustom)
	})
}
