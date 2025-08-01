package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"simplied-iot-monitoring-go/internal/models"

	"github.com/IBM/sarama"
)

// ConsumerGroupHandler 消费者组处理器
type ConsumerGroupHandler struct {
	consumer *KafkaConsumer
}

// NewConsumerGroupHandler 创建新的消费者组处理器
func NewConsumerGroupHandler(consumer *KafkaConsumer) *ConsumerGroupHandler {
	return &ConsumerGroupHandler{
		consumer: consumer,
	}
}

// Setup 在新会话开始时调用，在ConsumeClaim之前
func (h *ConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	log.Printf("消费者组会话开始 - MemberID: %s, GenerationID: %d",
		session.MemberID(), session.GenerationID())

	// 更新分区信息
	claims := session.Claims()
	totalPartitions := int64(0)

	for topic, partitions := range claims {
		log.Printf("分配到主题 %s 的分区: %v", topic, partitions)
		h.consumer.updateActivePartitions(topic, partitions)
		totalPartitions += int64(len(partitions))
	}

	h.consumer.updatePartitionCount(totalPartitions)
	h.consumer.incrementRebalanceCount()

	return nil
}

// Cleanup 在会话结束时调用，在所有ConsumeClaim协程退出后
func (h *ConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	log.Printf("消费者组会话结束 - MemberID: %s", session.MemberID())
	return nil
}

// ConsumeClaim 处理分区消息
func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Printf("开始消费分区 %s:%d", claim.Topic(), claim.Partition())

	// 处理分区中的消息
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				log.Printf("分区 %s:%d 消息通道已关闭", claim.Topic(), claim.Partition())
				return nil
			}

			// 处理消息
			if err := h.processMessage(session, message); err != nil {
				log.Printf("处理消息失败: %v", err)

				// 调用错误处理器
				if h.consumer.errorHandler != nil {
					shouldRetry := h.consumer.errorHandler.HandleError(session.Context(), err, message)
					if !shouldRetry {
						// 如果不需要重试，标记消息已处理
						session.MarkMessage(message, "")
					}
				} else {
					// 没有错误处理器，直接标记消息已处理
					session.MarkMessage(message, "")
				}

				h.consumer.incrementProcessingErrors()
			} else {
				// 处理成功，标记消息已处理
				session.MarkMessage(message, "")
				h.consumer.incrementMessagesProcessed()
			}

			h.consumer.incrementMessagesConsumed()

		case <-session.Context().Done():
			log.Printf("分区 %s:%d 会话上下文已取消", claim.Topic(), claim.Partition())
			return nil
		}
	}
}

// processMessage 处理单个消息
func (h *ConsumerGroupHandler) processMessage(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) error {
	startTime := time.Now()

	log.Printf("处理消息 - Topic: %s, Partition: %d, Offset: %d, Key: %s",
		message.Topic, message.Partition, message.Offset, string(message.Key))

	// 解析Kafka消息
	kafkaMessage, err := h.parseKafkaMessage(message)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	// 验证消息
	if err := kafkaMessage.Validate(); err != nil {
		return fmt.Errorf("消息验证失败: %w", err)
	}

	// 调用消息处理器
	if h.consumer.messageProcessor != nil {
		if err := h.consumer.messageProcessor.ProcessMessage(session.Context(), kafkaMessage); err != nil {
			return fmt.Errorf("消息处理器处理失败: %w", err)
		}
	}

	// 更新处理延迟指标
	processingTime := time.Since(startTime)
	h.consumer.updateProcessingLatency(processingTime.Milliseconds())

	log.Printf("消息处理完成 - 耗时: %v", processingTime)
	return nil
}

// parseKafkaMessage 解析Kafka消息为内部消息格式
func (h *ConsumerGroupHandler) parseKafkaMessage(message *sarama.ConsumerMessage) (*models.KafkaMessage, error) {
	// 首先解析为通用map以处理类型不匹配问题
	var rawMessage map[string]interface{}
	if err := json.Unmarshal(message.Value, &rawMessage); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}
	
	// 调试日志：输出原始消息结构
	log.Printf("原始消息结构: %+v", rawMessage)

	// 创建KafkaMessage并手动设置字段
	kafkaMessage := &models.KafkaMessage{}
	
	// 检查是否为完整的KafkaMessage格式（包含message_id字段）
	if _, hasMessageID := rawMessage["message_id"]; !hasMessageID {
		// 如果是简单的设备数据格式，将其包装为KafkaMessage
		return h.wrapSimpleDeviceData(rawMessage, message)
	}

	// 处理message_id
	if messageID, ok := rawMessage["message_id"].(string); ok {
		kafkaMessage.MessageID = messageID
	}

	// 处理message_type
	if messageType, ok := rawMessage["message_type"].(string); ok {
		kafkaMessage.MessageType = models.MessageType(messageType)
	}

	// 处理timestamp - 支持字符串和数字格式
	if timestampRaw, exists := rawMessage["timestamp"]; exists {
		switch v := timestampRaw.(type) {
		case string:
			// 尝试解析字符串时间戳
			if timestamp, err := strconv.ParseInt(v, 10, 64); err == nil {
				kafkaMessage.Timestamp = timestamp
			} else {
				// 如果不是数字字符串，使用当前时间
				kafkaMessage.Timestamp = time.Now().Unix()
			}
		case float64:
			kafkaMessage.Timestamp = int64(v)
		case int64:
			kafkaMessage.Timestamp = v
		default:
			kafkaMessage.Timestamp = time.Now().Unix()
		}
	} else {
		kafkaMessage.Timestamp = time.Now().Unix()
	}

	// 处理source
	if source, ok := rawMessage["source"].(string); ok {
		kafkaMessage.Source = source
	}

	// 处理payload
	if payload, exists := rawMessage["payload"]; exists {
		kafkaMessage.Payload = payload
	}

	// 处理headers
	if headersRaw, exists := rawMessage["headers"]; exists {
		if headersMap, ok := headersRaw.(map[string]interface{}); ok {
			headers := make(map[string]string)
			for k, v := range headersMap {
				if strVal, ok := v.(string); ok {
					headers[k] = strVal
				}
			}
			kafkaMessage.Headers = headers
		}
	}

	// 处理priority - 支持字符串和数字格式
	if priorityRaw, exists := rawMessage["priority"]; exists {
		switch v := priorityRaw.(type) {
		case string:
			// 将字符串优先级转换为数字
			switch v {
			case "low":
				kafkaMessage.Priority = models.PriorityLow
			case "normal":
				kafkaMessage.Priority = models.PriorityNormal
			case "high":
				kafkaMessage.Priority = models.PriorityHigh
			case "critical":
				kafkaMessage.Priority = models.PriorityCritical
			default:
				kafkaMessage.Priority = models.PriorityNormal
			}
		case float64:
			kafkaMessage.Priority = models.MessagePriority(v)
		case int:
			kafkaMessage.Priority = models.MessagePriority(v)
		default:
			kafkaMessage.Priority = models.PriorityNormal
		}
	} else {
		kafkaMessage.Priority = models.PriorityNormal
	}

	// 处理TTL
	if ttlRaw, exists := rawMessage["ttl"]; exists {
		switch v := ttlRaw.(type) {
		case float64:
			kafkaMessage.TTL = int64(v)
		case int64:
			kafkaMessage.TTL = v
		case string:
			if ttl, err := strconv.ParseInt(v, 10, 64); err == nil {
				kafkaMessage.TTL = ttl
			}
		}
	}

	// 设置Kafka特定信息
	kafkaMessage.Topic = message.Topic
	kafkaMessage.Partition = message.Partition
	kafkaMessage.Key = string(message.Key)

	// 解析消息头
	if kafkaMessage.Headers == nil {
		kafkaMessage.Headers = make(map[string]string)
	}

	for _, header := range message.Headers {
		kafkaMessage.Headers[string(header.Key)] = string(header.Value)
	}

	return kafkaMessage, nil
}

// wrapSimpleDeviceData 将简单的设备数据包装为KafkaMessage格式
func (h *ConsumerGroupHandler) wrapSimpleDeviceData(rawData map[string]interface{}, message *sarama.ConsumerMessage) (*models.KafkaMessage, error) {
	// 从原始数据中提取设备信息
	deviceID, _ := rawData["device_id"].(string)
	deviceType, _ := rawData["device_type"].(string)
	location, _ := rawData["location"].(string)
	status, _ := rawData["status"].(string)
	
	// 提取传感器数据
	temperature, _ := rawData["temperature"].(float64)
	humidity, _ := rawData["humidity"].(float64)
	pressure, _ := rawData["pressure"].(float64)
	batteryLevel, _ := rawData["battery_level"].(float64)
	
	// 处理时间戳
	timestamp := time.Now().Unix()
	if timestampRaw, exists := rawData["timestamp"]; exists {
		if timestampStr, ok := timestampRaw.(string); ok {
			// 尝试解析ISO时间格式
			if parsedTime, err := time.Parse(time.RFC3339, timestampStr); err == nil {
				timestamp = parsedTime.Unix()
			}
		}
	}
	
	// 创建Device对象
	device := &models.Device{
		DeviceID:   deviceID,
		DeviceType: deviceType,
		Timestamp:  timestamp,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		DeviceInfo: models.DeviceInfo{
			Model:           deviceType,
			Manufacturer:    "Unknown",
			FirmwareVersion: "1.0",
			HardwareVersion: "1.0",
			SerialNumber:    deviceID,
			BatteryLevel:    int(batteryLevel),
			SignalStrength:  -50,
			NetworkType:     models.NetworkTypeWiFi,
		},
		Location: models.LocationInfo{
			Building:  location,
			Floor:     1,
			Room:      "Unknown",
			Zone:      "Zone1",
			Latitude:  0.0,
			Longitude: 0.0,
		},
		SensorData: models.SensorData{
			Temperature: &models.TemperatureSensor{
				Value:  temperature,
				Unit:   "°C",
				Status: models.SensorStatusNormal,
			},
			Humidity: &models.HumiditySensor{
				Value:  humidity,
				Unit:   "%",
				Status: models.SensorStatusNormal,
			},
			Pressure: &models.PressureSensor{
				Value:  pressure,
				Unit:   "hPa",
				Status: models.SensorStatusNormal,
			},
			LastUpdate: timestamp,
		},
		Status:   models.DeviceStatus(status),
		LastSeen: timestamp,
	}
	
	// 创建DeviceDataPayload
	payload := &models.DeviceDataPayload{
		Device:    device,
		BatchSize: 1,
		BatchID:   fmt.Sprintf("batch-%d-%d", message.Partition, message.Offset),
	}
	
	// 生成KafkaMessage
	kafkaMessage := &models.KafkaMessage{
		MessageID:   fmt.Sprintf("auto-%d-%d", message.Partition, message.Offset),
		MessageType: models.MessageTypeDeviceData,
		Timestamp:   timestamp,
		Source:      "legacy-producer",
		Payload:     payload,
		Headers:     make(map[string]string),
		Priority:    models.PriorityNormal,
		TTL:         3600,
		Topic:       message.Topic,
		Partition:   message.Partition,
		Key:         string(message.Key),
	}
	
	// 复制Kafka消息头
	for _, header := range message.Headers {
		kafkaMessage.Headers[string(header.Key)] = string(header.Value)
	}
	
	log.Printf("将简单设备数据包装为KafkaMessage: %s", kafkaMessage.MessageID)
	return kafkaMessage, nil
}

// PartitionConsumerHandler 分区消费者处理器
type PartitionConsumerHandler struct {
	topic     string
	partition int32
	consumer  *KafkaConsumer
}

// NewPartitionConsumerHandler 创建分区消费者处理器
func NewPartitionConsumerHandler(topic string, partition int32, consumer *KafkaConsumer) *PartitionConsumerHandler {
	return &PartitionConsumerHandler{
		topic:     topic,
		partition: partition,
		consumer:  consumer,
	}
}

// HandlePartition 处理分区消息
func (h *PartitionConsumerHandler) HandlePartition(ctx context.Context, partitionConsumer sarama.PartitionConsumer) error {
	log.Printf("开始处理分区 %s:%d", h.topic, h.partition)

	for {
		select {
		case message := <-partitionConsumer.Messages():
			if message == nil {
				continue
			}

			// 处理消息
			if err := h.processPartitionMessage(ctx, message); err != nil {
				log.Printf("处理分区消息失败: %v", err)
				h.consumer.incrementProcessingErrors()
			} else {
				h.consumer.incrementMessagesProcessed()
			}

			h.consumer.incrementMessagesConsumed()

		case err := <-partitionConsumer.Errors():
			if err != nil {
				log.Printf("分区 %s:%d 错误: %v", h.topic, h.partition, err)
				h.consumer.incrementProcessingErrors()
			}

		case <-ctx.Done():
			log.Printf("分区 %s:%d 上下文已取消", h.topic, h.partition)
			return ctx.Err()
		}
	}
}

// processPartitionMessage 处理分区消息
func (h *PartitionConsumerHandler) processPartitionMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	startTime := time.Now()

	// 解析消息
	kafkaMessage, err := h.parseMessage(message)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	// 验证消息
	if err := kafkaMessage.Validate(); err != nil {
		return fmt.Errorf("消息验证失败: %w", err)
	}

	// 处理消息
	if h.consumer.messageProcessor != nil {
		if err := h.consumer.messageProcessor.ProcessMessage(ctx, kafkaMessage); err != nil {
			return fmt.Errorf("消息处理失败: %w", err)
		}
	}

	// 更新处理延迟
	processingTime := time.Since(startTime)
	h.consumer.updateProcessingLatency(processingTime.Milliseconds())

	return nil
}

// parseMessage 解析消息
func (h *PartitionConsumerHandler) parseMessage(message *sarama.ConsumerMessage) (*models.KafkaMessage, error) {
	var kafkaMessage models.KafkaMessage

	if err := json.Unmarshal(message.Value, &kafkaMessage); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	kafkaMessage.Topic = message.Topic
	kafkaMessage.Partition = message.Partition
	kafkaMessage.Key = string(message.Key)

	if kafkaMessage.Headers == nil {
		kafkaMessage.Headers = make(map[string]string)
	}

	for _, header := range message.Headers {
		kafkaMessage.Headers[string(header.Key)] = string(header.Value)
	}

	return &kafkaMessage, nil
}

// ConsumerGroupCoordinator 消费者组协调器
type ConsumerGroupCoordinator struct {
	groupID   string
	brokers   []string
	topics    []string
	config    *sarama.Config
	consumers map[string]*KafkaConsumer
	mutex     sync.RWMutex
}

// NewConsumerGroupCoordinator 创建消费者组协调器
func NewConsumerGroupCoordinator(groupID string, brokers []string, topics []string, config *sarama.Config) *ConsumerGroupCoordinator {
	return &ConsumerGroupCoordinator{
		groupID:   groupID,
		brokers:   brokers,
		topics:    topics,
		config:    config,
		consumers: make(map[string]*KafkaConsumer),
	}
}

// AddConsumer 添加消费者
func (cgc *ConsumerGroupCoordinator) AddConsumer(consumerID string, consumer *KafkaConsumer) {
	cgc.mutex.Lock()
	defer cgc.mutex.Unlock()
	cgc.consumers[consumerID] = consumer
}

// RemoveConsumer 移除消费者
func (cgc *ConsumerGroupCoordinator) RemoveConsumer(consumerID string) {
	cgc.mutex.Lock()
	defer cgc.mutex.Unlock()
	if consumer, exists := cgc.consumers[consumerID]; exists {
		consumer.Stop()
		delete(cgc.consumers, consumerID)
	}
}

// GetConsumers 获取所有消费者
func (cgc *ConsumerGroupCoordinator) GetConsumers() map[string]*KafkaConsumer {
	cgc.mutex.RLock()
	defer cgc.mutex.RUnlock()

	consumers := make(map[string]*KafkaConsumer)
	for id, consumer := range cgc.consumers {
		consumers[id] = consumer
	}
	return consumers
}

// GetGroupMetrics 获取组指标
func (cgc *ConsumerGroupCoordinator) GetGroupMetrics() map[string]*ConsumerMetrics {
	cgc.mutex.RLock()
	defer cgc.mutex.RUnlock()

	metrics := make(map[string]*ConsumerMetrics)
	for id, consumer := range cgc.consumers {
		metrics[id] = consumer.GetMetrics()
	}
	return metrics
}
