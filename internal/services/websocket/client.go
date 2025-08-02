package websocket

import (
	"fmt"
	"sync"
	"time"

	"simplied-iot-monitoring-go/internal/models"

	"github.com/gorilla/websocket"
)

// Client WebSocket客户端连接
// 表示单个WebSocket连接及其相关状态
type Client struct {
	// 客户端唯一标识
	ID string

	// WebSocket连接
	Conn *websocket.Conn

	// 发送消息通道
	Send chan []byte

	// 所属Hub
	Hub *Hub

	// 最后ping时间
	LastPing time.Time

	// 最后活动时间
	LastSeen time.Time

	// 订阅信息
	Subscriptions map[string]*SubscriptionFilter

	// 客户端过滤器
	Filter *ClientFilter

	// 客户端元数据
	Metadata *ClientMetadata

	// 并发控制
	mutex sync.RWMutex

	// 连接状态
	connected bool

	// 统计信息
	stats *ClientStats
}

// ClientMetadata 客户端元数据
type ClientMetadata struct {
	UserAgent   string            `json:"user_agent"`
	RemoteAddr  string            `json:"remote_addr"`
	ConnectedAt time.Time         `json:"connected_at"`
	UserID      string            `json:"user_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Version     string            `json:"version,omitempty"`
}

// ClientStats 客户端统计信息
type ClientStats struct {
	MessagesSent     int64     `json:"messages_sent"`
	MessagesReceived int64     `json:"messages_received"`
	BytesSent        int64     `json:"bytes_sent"`
	BytesReceived    int64     `json:"bytes_received"`
	LastMessageTime  time.Time `json:"last_message_time"`
	ErrorCount       int64     `json:"error_count"`
	mutex            sync.RWMutex
}

// ClientFilter 客户端过滤器
// 用于确定哪些消息应该发送给特定客户端
type ClientFilter struct {
	// 设备ID过滤
	DeviceIDs map[string]bool `json:"device_ids,omitempty"`

	// 设备类型过滤
	DeviceTypes map[string]bool `json:"device_types,omitempty"`

	// 位置过滤
	Locations map[string]bool `json:"locations,omitempty"`

	// 告警级别过滤
	AlertLevels map[string]bool `json:"alert_levels,omitempty"`

	// 传感器类型过滤
	SensorTypes map[string]bool `json:"sensor_types,omitempty"`

	// 组ID过滤
	GroupIDs map[string]bool `json:"group_ids,omitempty"`

	// 自定义标签过滤
	Tags map[string]string `json:"tags,omitempty"`

	// 时间范围过滤
	TimeRange *TimeRange `json:"time_range,omitempty"`

	// 是否启用过滤
	Enabled bool `json:"enabled"`
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// SubscriptionFilter 订阅过滤器
type SubscriptionFilter struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // device_data, alert, system_status
	DeviceIDs   []string          `json:"device_ids,omitempty"`
	DeviceTypes []string          `json:"device_types,omitempty"`
	Locations   []string          `json:"locations,omitempty"`
	AlertLevels []string          `json:"alert_levels,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// NewClient 创建新的客户端实例
func NewClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	now := time.Now()

	return &Client{
		ID:            id,
		Conn:          conn,
		Send:          make(chan []byte, 256),
		Hub:           hub,
		LastPing:      now,
		LastSeen:      now,
		Subscriptions: make(map[string]*SubscriptionFilter),
		Filter:        &ClientFilter{Enabled: false},
		Metadata: &ClientMetadata{
			ConnectedAt: now,
			Tags:        make(map[string]string),
		},
		connected: true,
		stats: &ClientStats{
			LastMessageTime: now,
		},
	}
}

// UpdateLastSeen 更新最后活动时间
func (c *Client) UpdateLastSeen() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.LastSeen = time.Now()
}

// GetLastSeen 获取最后活动时间
func (c *Client) GetLastSeen() time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.LastSeen
}

// UpdateLastPing 更新最后ping时间
func (c *Client) UpdateLastPing() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.LastPing = time.Now()
}

// GetLastPing 获取最后ping时间
func (c *Client) GetLastPing() time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.LastPing
}

// IsConnected 检查连接状态
func (c *Client) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

// SetConnected 设置连接状态
func (c *Client) SetConnected(connected bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.connected = connected
}

// AddSubscription 添加订阅
func (c *Client) AddSubscription(filter *SubscriptionFilter) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	filter.CreatedAt = time.Now()
	filter.UpdatedAt = time.Now()
	c.Subscriptions[filter.ID] = filter
}

// RemoveSubscription 移除订阅
func (c *Client) RemoveSubscription(filterID string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.Subscriptions[filterID]; exists {
		delete(c.Subscriptions, filterID)
		return true
	}
	return false
}

// GetSubscription 获取订阅
func (c *Client) GetSubscription(filterID string) *SubscriptionFilter {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Subscriptions[filterID]
}

// GetAllSubscriptions 获取所有订阅
func (c *Client) GetAllSubscriptions() []*SubscriptionFilter {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	filters := make([]*SubscriptionFilter, 0, len(c.Subscriptions))
	for _, filter := range c.Subscriptions {
		filters = append(filters, filter)
	}
	return filters
}

// UpdateFilter 更新客户端过滤器
func (c *Client) UpdateFilter(filter *ClientFilter) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Filter = filter
}

// GetFilter 获取客户端过滤器
func (c *Client) GetFilter() *ClientFilter {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Filter
}

// UpdateMetadata 更新客户端元数据
func (c *Client) UpdateMetadata(metadata *ClientMetadata) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Metadata = metadata
}

// GetMetadata 获取客户端元数据
func (c *Client) GetMetadata() *ClientMetadata {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Metadata
}

// GetStats 获取客户端统计信息
func (c *Client) GetStats() *ClientStats {
	c.stats.mutex.RLock()
	defer c.stats.mutex.RUnlock()

	// 返回统计信息的副本
	statsCopy := *c.stats
	return &statsCopy
}

// UpdateStats 更新统计信息
func (c *Client) UpdateStats(updater func(*ClientStats)) {
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()
	updater(c.stats)
}

// SendMessage 发送消息给客户端
func (c *Client) SendMessage(message []byte) error {
	if !c.IsConnected() {
		return ErrClientDisconnected
	}

	select {
	case c.Send <- message:
		c.UpdateStats(func(stats *ClientStats) {
			stats.MessagesSent++
			stats.BytesSent += int64(len(message))
			stats.LastMessageTime = time.Now()
		})
		return nil
	default:
		c.UpdateStats(func(stats *ClientStats) {
			stats.ErrorCount++
		})
		return ErrSendChannelFull
	}
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	c.SetConnected(false)

	// 关闭发送通道
	close(c.Send)

	// 关闭WebSocket连接
	return c.Conn.Close()
}

// MatchesDeviceData 检查设备数据是否匹配过滤器
func (cf *ClientFilter) MatchesDeviceData(data *models.DeviceDataPayload) bool {
	if !cf.Enabled {
		return true
	}

	// 检查设备ID过滤
	if len(cf.DeviceIDs) > 0 {
		if data.Device == nil || !cf.DeviceIDs[data.Device.DeviceID] {
			return false
		}
	}

	// 检查设备类型过滤
	if len(cf.DeviceTypes) > 0 {
		if data.Device == nil || !cf.DeviceTypes[data.Device.DeviceType] {
			return false
		}
	}

	// 检查位置过滤
	if len(cf.Locations) > 0 {
		if data.Device == nil {
			return false
		}
		// 检查建筑物、楼层、房间是否匹配
		if !cf.Locations[data.Device.Location.Building] &&
			!cf.Locations[fmt.Sprintf("%d", data.Device.Location.Floor)] &&
			!cf.Locations[data.Device.Location.Room] {
			return false
		}
	}

	// 检查传感器类型过滤
	if len(cf.SensorTypes) > 0 && data.Device != nil {
		// 检查各种传感器类型
		if (cf.SensorTypes["temperature"] && data.Device.SensorData.Temperature != nil) ||
			(cf.SensorTypes["humidity"] && data.Device.SensorData.Humidity != nil) ||
			(cf.SensorTypes["pressure"] && data.Device.SensorData.Pressure != nil) ||
			(cf.SensorTypes["switch"] && data.Device.SensorData.Switch != nil) ||
			(cf.SensorTypes["current"] && data.Device.SensorData.Current != nil) {
			// 找到匹配的传感器类型
		} else {
			return false
		}
	}

	// 检查时间范围过滤
	if cf.TimeRange != nil {
		timestamp := time.Unix(data.Device.Timestamp/1000, 0)
		if timestamp.Before(cf.TimeRange.Start) || timestamp.After(cf.TimeRange.End) {
			return false
		}
	}

	return true
}

// MatchesAlert 检查告警是否匹配过滤器
func (cf *ClientFilter) MatchesAlert(alert *models.AlertPayload) bool {
	if !cf.Enabled {
		return true
	}

	// 检查设备ID过滤
	if len(cf.DeviceIDs) > 0 {
		if alert.Alert == nil || !cf.DeviceIDs[alert.Alert.DeviceID] {
			return false
		}
	}

	// 检查告警级别过滤
	if len(cf.AlertLevels) > 0 {
		if alert.Alert == nil || !cf.AlertLevels[string(alert.Alert.Severity)] {
			return false
		}
	}

	// 检查时间范围过滤
	if cf.TimeRange != nil {
		timestamp := time.Unix(alert.Alert.Timestamp, 0)
		if timestamp.Before(cf.TimeRange.Start) || timestamp.After(cf.TimeRange.End) {
			return false
		}
	}

	return true
}

// MatchesGroup 检查是否匹配组ID
func (cf *ClientFilter) MatchesGroup(groupID string) bool {
	if !cf.Enabled {
		return true
	}

	if len(cf.GroupIDs) == 0 {
		return true
	}

	return cf.GroupIDs[groupID]
}

// MatchesSubscription 检查订阅过滤器是否匹配设备数据
func (sf *SubscriptionFilter) MatchesDeviceData(data *models.DeviceDataPayload) bool {
	if sf.Type != "device_data" && sf.Type != "all" {
		return false
	}

	// 检查设备ID过滤
	if len(sf.DeviceIDs) > 0 {
		found := false
		for _, id := range sf.DeviceIDs {
			if data.Device != nil && data.Device.DeviceID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查设备类型过滤
	if len(sf.DeviceTypes) > 0 {
		found := false
		for _, deviceType := range sf.DeviceTypes {
			if data.Device != nil && data.Device.DeviceType == deviceType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查位置过滤
	if len(sf.Locations) > 0 {
		found := false
		for _, location := range sf.Locations {
			if data.Device != nil &&
				(data.Device.Location.Building == location ||
					fmt.Sprintf("%d", data.Device.Location.Floor) == location ||
					data.Device.Location.Room == location) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// MatchesAlert 检查订阅过滤器是否匹配告警
func (sf *SubscriptionFilter) MatchesAlert(alert *models.AlertPayload) bool {
	if sf.Type != "alert" && sf.Type != "all" {
		return false
	}

	// 检查设备ID过滤
	if len(sf.DeviceIDs) > 0 {
		found := false
		for _, id := range sf.DeviceIDs {
			if alert.Alert != nil && alert.Alert.DeviceID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查告警级别过滤
	if len(sf.AlertLevels) > 0 {
		found := false
		for _, level := range sf.AlertLevels {
			if alert.Alert != nil && string(alert.Alert.Severity) == level {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
