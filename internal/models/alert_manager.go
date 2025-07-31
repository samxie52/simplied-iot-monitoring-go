package models

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AlertManagerImpl 告警管理器实现
type AlertManagerImpl struct {
	alerts        map[string]*Alert           // 活跃告警
	aggregations  map[string]*AlertAggregation // 告警聚合
	rules         map[string]*AlertRule       // 告警规则
	history       []*AlertHistory             // 告警历史
	mutex         sync.RWMutex                // 并发安全
	
	// 配置参数
	maxHistorySize    int           // 最大历史记录数
	aggregationWindow time.Duration // 聚合时间窗口
	cleanupInterval   time.Duration // 清理间隔
}

// AlertStateMachine 告警状态机
type AlertStateMachine struct {
	currentState AlertStatus
	transitions  map[AlertStatus][]AlertStatus
}

// AlertAggregator 告警聚合器
type AlertAggregator struct {
	groupBy       []string                    // 聚合字段
	aggregations  map[string]*AlertAggregation // 聚合结果
	window        time.Duration               // 时间窗口
	mutex         sync.RWMutex                // 并发安全
}

// AlertMetrics 告警统计指标
type AlertMetrics struct {
	TotalAlerts       int64                    `json:"total_alerts"`
	ActiveAlerts      int64                    `json:"active_alerts"`
	ResolvedAlerts    int64                    `json:"resolved_alerts"`
	AlertsByType      map[AlertType]int64      `json:"alerts_by_type"`
	AlertsBySeverity  map[AlertSeverity]int64  `json:"alerts_by_severity"`
	AlertsByStatus    map[AlertStatus]int64    `json:"alerts_by_status"`
	AverageResolutionTime time.Duration        `json:"average_resolution_time"`
	LastUpdated       time.Time                `json:"last_updated"`
}

// NewAlertManager 创建告警管理器
func NewAlertManager() *AlertManagerImpl {
	return &AlertManagerImpl{
		alerts:            make(map[string]*Alert),
		aggregations:      make(map[string]*AlertAggregation),
		rules:             make(map[string]*AlertRule),
		history:           make([]*AlertHistory, 0),
		maxHistorySize:    10000,
		aggregationWindow: 5 * time.Minute,
		cleanupInterval:   1 * time.Hour,
	}
}

// NewAlertStateMachine 创建告警状态机
func NewAlertStateMachine() *AlertStateMachine {
	transitions := map[AlertStatus][]AlertStatus{
		AlertStatusActive: {
			AlertStatusAcknowledged,
			AlertStatusResolved,
			AlertStatusClosed,
		},
		AlertStatusAcknowledged: {
			AlertStatusResolved,
			AlertStatusClosed,
		},
		AlertStatusResolved: {
			AlertStatusClosed,
			AlertStatusActive, // 可能重新激活
		},
		AlertStatusClosed: {
			// 关闭状态不能转换到其他状态
		},
	}
	
	return &AlertStateMachine{
		currentState: AlertStatusActive,
		transitions:  transitions,
	}
}

// NewAlertAggregator 创建告警聚合器
func NewAlertAggregator(groupBy []string, window time.Duration) *AlertAggregator {
	return &AlertAggregator{
		groupBy:      groupBy,
		aggregations: make(map[string]*AlertAggregation),
		window:       window,
	}
}

// AddAlert 添加告警
func (am *AlertManagerImpl) AddAlert(alert *Alert) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	if alert.AlertID == "" {
		return fmt.Errorf("alert ID is required")
	}
	
	// 检查是否需要聚合
	if aggregated := am.tryAggregate(alert); aggregated {
		return nil
	}
	
	// 添加新告警
	am.alerts[alert.AlertID] = alert
	
	// 记录历史
	history := &AlertHistory{
		HistoryID: uuid.New().String(),
		AlertID:   alert.AlertID,
		Action:    "created",
		Timestamp: time.Now().Unix(),
		UserID:    "system",
		NewStatus: alert.Status,
	}
	am.addHistory(history)
	
	return nil
}

// GetAlert 获取告警
func (am *AlertManagerImpl) GetAlert(alertID string) (*Alert, bool) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	alert, exists := am.alerts[alertID]
	return alert, exists
}

// UpdateAlertStatus 更新告警状态
func (am *AlertManagerImpl) UpdateAlertStatus(alertID string, newStatus AlertStatus, userID string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}
	
	// 验证状态转换
	stateMachine := NewAlertStateMachine()
	stateMachine.currentState = alert.Status
	if !stateMachine.CanTransition(newStatus) {
		return fmt.Errorf("invalid state transition from %s to %s", alert.Status, newStatus)
	}
	
	oldStatus := alert.Status
	alert.Status = newStatus
	alert.UpdatedAt = time.Now()
	
	// 根据状态更新相关字段
	switch newStatus {
	case AlertStatusAcknowledged:
		now := time.Now()
		alert.AcknowledgedBy = userID
		alert.AcknowledgedAt = &now
	case AlertStatusResolved:
		now := time.Now()
		alert.ResolvedBy = userID
		alert.ResolvedAt = &now
	case AlertStatusClosed:
		// 从活跃告警中移除
		delete(am.alerts, alertID)
	}
	
	// 记录历史
	history := &AlertHistory{
		HistoryID: uuid.New().String(),
		AlertID:   alertID,
		Action:    "status_changed",
		Timestamp: time.Now().Unix(),
		UserID:    userID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
	}
	am.addHistory(history)
	
	return nil
}

// GetActiveAlerts 获取活跃告警
func (am *AlertManagerImpl) GetActiveAlerts() []*Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	alerts := make([]*Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		if alert.Status == AlertStatusActive {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// GetAlertsByDevice 根据设备ID获取告警
func (am *AlertManagerImpl) GetAlertsByDevice(deviceID string) []*Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	alerts := make([]*Alert, 0)
	for _, alert := range am.alerts {
		if alert.DeviceID == deviceID {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// GetAlertsBySeverity 根据严重级别获取告警
func (am *AlertManagerImpl) GetAlertsBySeverity(severity AlertSeverity) []*Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	alerts := make([]*Alert, 0)
	for _, alert := range am.alerts {
		if alert.Severity == severity {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// GetMetrics 获取告警统计指标
func (am *AlertManagerImpl) GetMetrics() *AlertMetrics {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	metrics := &AlertMetrics{
		AlertsByType:     make(map[AlertType]int64),
		AlertsBySeverity: make(map[AlertSeverity]int64),
		AlertsByStatus:   make(map[AlertStatus]int64),
		LastUpdated:      time.Now(),
	}
	
	var totalResolutionTime time.Duration
	var resolvedCount int64
	
	for _, alert := range am.alerts {
		metrics.TotalAlerts++
		
		// 按状态统计
		metrics.AlertsByStatus[alert.Status]++
		if alert.Status == AlertStatusActive {
			metrics.ActiveAlerts++
		} else if alert.Status == AlertStatusResolved || alert.Status == AlertStatusClosed {
			metrics.ResolvedAlerts++
			if alert.ResolvedAt != nil {
				resolutionTime := alert.ResolvedAt.Sub(alert.CreatedAt)
				totalResolutionTime += resolutionTime
				resolvedCount++
			}
		}
		
		// 按类型统计
		metrics.AlertsByType[alert.AlertType]++
		
		// 按严重级别统计
		metrics.AlertsBySeverity[alert.Severity]++
	}
	
	// 计算平均解决时间
	if resolvedCount > 0 {
		metrics.AverageResolutionTime = totalResolutionTime / time.Duration(resolvedCount)
	}
	
	return metrics
}

// tryAggregate 尝试聚合告警
func (am *AlertManagerImpl) tryAggregate(alert *Alert) bool {
	// 生成聚合键
	key := am.generateAggregationKey(alert)
	
	aggregation, exists := am.aggregations[key]
	if !exists {
		// 创建新聚合
		aggregation = &AlertAggregation{
			AggregationID: uuid.New().String(),
			GroupBy:       []string{"device_id", "alert_type"},
			AlertIDs:      []string{alert.AlertID},
			Count:         1,
			Severity:      alert.Severity,
			FirstAlert:    alert.Timestamp,
			LastAlert:     alert.Timestamp,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		am.aggregations[key] = aggregation
		return false // 新聚合，不算聚合
	}
	
	// 检查时间窗口
	if time.Now().Unix()-aggregation.LastAlert > int64(am.aggregationWindow.Seconds()) {
		// 超出时间窗口，创建新聚合
		return false
	}
	
	// 更新现有聚合
	aggregation.AlertIDs = append(aggregation.AlertIDs, alert.AlertID)
	aggregation.Count++
	aggregation.LastAlert = alert.Timestamp
	aggregation.UpdatedAt = time.Now()
	
	// 更新严重级别（取最高级别）
	if am.isHigherSeverity(alert.Severity, aggregation.Severity) {
		aggregation.Severity = alert.Severity
	}
	
	return true
}

// generateAggregationKey 生成聚合键
func (am *AlertManagerImpl) generateAggregationKey(alert *Alert) string {
	return fmt.Sprintf("%s:%s", alert.DeviceID, alert.AlertType)
}

// isHigherSeverity 检查是否为更高严重级别
func (am *AlertManagerImpl) isHigherSeverity(severity1, severity2 AlertSeverity) bool {
	severityOrder := map[AlertSeverity]int{
		AlertSeverityInfo:     1,
		AlertSeverityWarning:  2,
		AlertSeverityError:    3,
		AlertSeverityCritical: 4,
	}
	
	return severityOrder[severity1] > severityOrder[severity2]
}

// addHistory 添加历史记录
func (am *AlertManagerImpl) addHistory(history *AlertHistory) {
	am.history = append(am.history, history)
	
	// 限制历史记录数量
	if len(am.history) > am.maxHistorySize {
		am.history = am.history[len(am.history)-am.maxHistorySize:]
	}
}

// CleanupExpiredAggregations 清理过期聚合
func (am *AlertManagerImpl) CleanupExpiredAggregations() int {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	now := time.Now().Unix()
	cleaned := 0
	
	for key, aggregation := range am.aggregations {
		if now-aggregation.LastAlert > int64(am.aggregationWindow.Seconds()*2) {
			delete(am.aggregations, key)
			cleaned++
		}
	}
	
	return cleaned
}

// 状态机方法

// CanTransition 检查是否可以转换到目标状态
func (sm *AlertStateMachine) CanTransition(targetState AlertStatus) bool {
	allowedStates, exists := sm.transitions[sm.currentState]
	if !exists {
		return false
	}
	
	for _, state := range allowedStates {
		if state == targetState {
			return true
		}
	}
	return false
}

// GetAllowedTransitions 获取允许的状态转换
func (sm *AlertStateMachine) GetAllowedTransitions() []AlertStatus {
	return sm.transitions[sm.currentState]
}

// Transition 执行状态转换
func (sm *AlertStateMachine) Transition(targetState AlertStatus) error {
	if !sm.CanTransition(targetState) {
		return fmt.Errorf("invalid transition from %s to %s", sm.currentState, targetState)
	}
	
	sm.currentState = targetState
	return nil
}

// 聚合器方法

// Aggregate 聚合告警
func (aa *AlertAggregator) Aggregate(alert *Alert) *AlertAggregation {
	aa.mutex.Lock()
	defer aa.mutex.Unlock()
	
	key := aa.generateKey(alert)
	
	aggregation, exists := aa.aggregations[key]
	if !exists {
		aggregation = &AlertAggregation{
			AggregationID: uuid.New().String(),
			GroupBy:       aa.groupBy,
			AlertIDs:      []string{alert.AlertID},
			Count:         1,
			Severity:      alert.Severity,
			FirstAlert:    alert.Timestamp,
			LastAlert:     alert.Timestamp,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		aa.aggregations[key] = aggregation
	} else {
		// 检查时间窗口（使用更新时间）
		if time.Since(aggregation.UpdatedAt) <= aa.window {
			aggregation.AlertIDs = append(aggregation.AlertIDs, alert.AlertID)
			aggregation.Count++
			aggregation.LastAlert = alert.Timestamp
			aggregation.UpdatedAt = time.Now()
			
			// 更新严重级别（取最高级别）
			if aa.isHigherSeverity(alert.Severity, aggregation.Severity) {
				aggregation.Severity = alert.Severity
			}
		} else {
			// 创建新聚合
			aggregation = &AlertAggregation{
				AggregationID: uuid.New().String(),
				GroupBy:       aa.groupBy,
				AlertIDs:      []string{alert.AlertID},
				Count:         1,
				Severity:      alert.Severity,
				FirstAlert:    alert.Timestamp,
				LastAlert:     alert.Timestamp,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			aa.aggregations[key] = aggregation
		}
	}
	
	return aggregation
}

// GetGroupBy 获取聚合字段
func (aa *AlertAggregator) GetGroupBy() []string {
	return aa.groupBy
}

// isHigherSeverity 检查是否为更高严重级别
func (aa *AlertAggregator) isHigherSeverity(severity1, severity2 AlertSeverity) bool {
	severityOrder := map[AlertSeverity]int{
		AlertSeverityInfo:     1,
		AlertSeverityWarning:  2,
		AlertSeverityError:    3,
		AlertSeverityCritical: 4,
	}
	
	return severityOrder[severity1] > severityOrder[severity2]
}

// generateKey 生成聚合键
func (aa *AlertAggregator) generateKey(alert *Alert) string {
	key := ""
	for i, field := range aa.groupBy {
		if i > 0 {
			key += ":"
		}
		switch field {
		case "device_id":
			key += alert.DeviceID
		case "alert_type":
			key += string(alert.AlertType)
		case "severity":
			key += string(alert.Severity)
		default:
			key += "unknown"
		}
	}
	return key
}

// GetAggregations 获取所有聚合
func (aa *AlertAggregator) GetAggregations() map[string]*AlertAggregation {
	aa.mutex.RLock()
	defer aa.mutex.RUnlock()
	
	result := make(map[string]*AlertAggregation)
	for k, v := range aa.aggregations {
		result[k] = v
	}
	return result
}

// CleanupExpired 清理过期聚合
func (aa *AlertAggregator) CleanupExpired() int {
	aa.mutex.Lock()
	defer aa.mutex.Unlock()
	
	cleaned := 0
	
	for key, aggregation := range aa.aggregations {
		// 使用更新时间来判断是否过期
		if time.Since(aggregation.UpdatedAt) > aa.window*2 {
			delete(aa.aggregations, key)
			cleaned++
		}
	}
	
	return cleaned
}
