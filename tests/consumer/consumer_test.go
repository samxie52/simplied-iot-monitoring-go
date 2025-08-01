package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/models"
	"simplied-iot-monitoring-go/internal/services/consumer"
)

// MockMessageProcessor 模拟消息处理器
type MockMessageProcessor struct {
	mock.Mock
}

func (m *MockMessageProcessor) ProcessMessage(ctx context.Context, message *models.KafkaMessage) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockMessageProcessor) GetStats() consumer.ProcessorStats {
	args := m.Called()
	return args.Get(0).(consumer.ProcessorStats)
}

// MockErrorHandler 模拟错误处理器
type MockErrorHandler struct {
	mock.Mock
}

func (m *MockErrorHandler) HandleError(ctx context.Context, err error, message *sarama.ConsumerMessage) bool {
	args := m.Called(ctx, err, message)
	return args.Bool(0)
}

func (m *MockErrorHandler) GetErrorStats() consumer.ErrorStats {
	args := m.Called()
	return args.Get(0).(consumer.ErrorStats)
}

// TestKafkaConsumerCreation 测试Kafka消费者创建
func TestKafkaConsumerCreation(t *testing.T) {
	tests := []struct {
		name          string
		brokers       []string
		topics        []string
		config        *config.KafkaConsumer
		expectError   bool
		errorContains string
	}{
		{
			name:    "valid_config",
			brokers: []string{"localhost:9092"},
			topics:  []string{"test-topic"},
			config: &config.KafkaConsumer{
				GroupID:          "test-group",
				AutoOffsetReset:  "earliest",
				EnableAutoCommit: true,
				SessionTimeout:   30 * time.Second,
				MaxPollRecords:   100,
			},
			expectError: false,
		},
		{
			name:          "empty_brokers",
			brokers:       []string{},
			topics:        []string{"test-topic"},
			config:        &config.KafkaConsumer{},
			expectError:   true,
			errorContains: "brokers cannot be empty",
		},
		{
			name:          "empty_topics",
			brokers:       []string{"localhost:9092"},
			topics:        []string{},
			config:        &config.KafkaConsumer{},
			expectError:   true,
			errorContains: "topics cannot be empty",
		},
		{
			name:          "nil_config",
			brokers:       []string{"localhost:9092"},
			topics:        []string{"test-topic"},
			config:        nil,
			expectError:   true,
			errorContains: "consumer config cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProcessor := &MockMessageProcessor{}
			mockErrorHandler := &MockErrorHandler{}

			kafkaConsumer, err := consumer.NewKafkaConsumer(
				tt.brokers,
				tt.topics,
				tt.config,
				mockProcessor,
				mockErrorHandler,
			)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, kafkaConsumer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, kafkaConsumer)
				assert.False(t, kafkaConsumer.IsRunning())
			}
		})
	}
}

// TestMessageProcessor 测试消息处理器
func TestMessageProcessor(t *testing.T) {
	config := consumer.ProcessorConfig{
		MaxRetries:           3,
		RetryDelay:           100 * time.Millisecond,
		ProcessingTimeout:    5 * time.Second,
		EnableValidation:     true,
		EnableTransformation: true,
		EnableCaching:        false,
	}

	processor := consumer.NewDefaultMessageProcessor(config)
	require.NotNil(t, processor)

	// 添加验证器和转换器
	processor.AddValidator(&consumer.BasicMessageValidator{})
	processor.AddTransformer(&consumer.TimestampTransformer{})

	// 创建测试消息
	testMessage := &models.KafkaMessage{
		MessageID:   "test-message-1",
		MessageType: models.MessageTypeDeviceData,
		Timestamp:   time.Now().Unix(),
		Source:      "test-source",
		Payload: &models.DeviceDataPayload{
			Device: &models.Device{
				DeviceID: "device-001",
				Name:     "Test Device",
				Type:     "sensor",
				Status:   "active",
			},
		},
		Priority: models.PriorityNormal,
	}

	ctx := context.Background()

	// 测试消息处理
	err := processor.ProcessMessage(ctx, testMessage)
	assert.Error(t, err) // 应该失败，因为没有注册处理器

	// 注册设备数据处理器
	processor.RegisterHandler(consumer.NewDeviceDataHandler(nil))

	// 再次测试消息处理
	err = processor.ProcessMessage(ctx, testMessage)
	assert.NoError(t, err)

	// 检查统计信息
	stats := processor.GetStats()
	assert.Equal(t, int64(1), stats.ProcessedCount)
	assert.Equal(t, int64(0), stats.ErrorCount)
	assert.Greater(t, stats.AvgLatencyMs, 0.0)
}

// TestErrorHandler 测试错误处理器
func TestErrorHandler(t *testing.T) {
	config := consumer.ErrorHandlerConfig{
		MaxRetries:        3,
		RetryDelay:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxRetryDelay:     5 * time.Second,
		LogErrors:         true,
	}

	errorHandler := consumer.NewDefaultErrorHandler(config)
	require.NotNil(t, errorHandler)

	ctx := context.Background()
	testMessage := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    123,
		Key:       []byte("test-key"),
		Value:     []byte("test-value"),
	}

	// 测试可重试错误
	retryableErr := assert.AnError
	shouldRetry := errorHandler.HandleError(ctx, retryableErr, testMessage)
	assert.True(t, shouldRetry)

	// 测试不可重试错误（验证错误）
	validationErr := &consumer.ValidationError{Message: "validation failed"}
	shouldRetry = errorHandler.HandleError(ctx, validationErr, testMessage)
	assert.False(t, shouldRetry)

	// 检查错误统计
	stats := errorHandler.GetErrorStats()
	assert.Equal(t, int64(2), stats.TotalErrors)
	assert.Equal(t, int64(1), stats.RetryableErrors)
	assert.Equal(t, int64(1), stats.FatalErrors)
}

// TestConsumerService 测试消费者服务
func TestConsumerService(t *testing.T) {
	// 创建测试配置
	cfg := &config.AppConfig{
		App: config.AppSection{
			Name:        "test-app",
			Version:     "1.0.0",
			Environment: "test",
			Debug:       true,
			LogLevel:    "debug",
		},
		Kafka: config.KafkaSection{
			Brokers: []string{"localhost:9092"},
			Topics: config.TopicConfig{
				DeviceData: "device-data",
				Alerts:     "alerts",
			},
			Consumer: config.KafkaConsumer{
				GroupID:          "test-consumer-group",
				AutoOffsetReset:  "earliest",
				EnableAutoCommit: true,
				SessionTimeout:   30 * time.Second,
				MaxPollRecords:   100,
			},
		},
	}

	// 创建消费者服务
	service, err := consumer.NewConsumerService(cfg)
	require.NoError(t, err)
	require.NotNil(t, service)

	// 测试初始状态
	assert.False(t, service.IsRunning())
	assert.Equal(t, cfg, service.GetConfig())

	// 测试获取组件
	assert.NotNil(t, service.GetKafkaConsumer())
	assert.NotNil(t, service.GetMessageProcessor())
	assert.NotNil(t, service.GetErrorHandler())

	// 测试健康状态
	health := service.GetHealth()
	assert.NotNil(t, health)
	assert.False(t, health.Healthy) // 服务未启动

	// 测试指标
	metrics := service.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "consumer")
	assert.Contains(t, metrics, "processor")
	assert.Contains(t, metrics, "error_handler")
}

// TestConfigManager 测试配置管理器
func TestConfigManager(t *testing.T) {
	// 创建模拟配置加载器
	mockLoader := &MockConfigLoader{}
	mockValidator := &MockConfigValidator{}

	configManager := consumer.NewConsumerConfigManager(mockLoader, mockValidator)
	require.NotNil(t, configManager)

	// 创建测试配置
	testConfig := &config.AppConfig{
		Kafka: config.KafkaSection{
			Consumer: config.KafkaConsumer{
				GroupID:        "test-group",
				SessionTimeout: 30 * time.Second,
				MaxPollRecords: 100,
			},
		},
	}

	// 设置模拟期望
	mockLoader.On("LoadConfig", mock.Anything).Return(testConfig, nil)
	mockLoader.On("GetConfigSource").Return("test://config")
	mockValidator.On("ValidateConfig", testConfig).Return(nil)

	// 测试加载配置
	err := configManager.LoadConfig()
	assert.NoError(t, err)

	// 验证配置
	loadedConfig := configManager.GetConfig()
	assert.Equal(t, testConfig, loadedConfig)

	consumerConfig := configManager.GetConsumerConfig()
	assert.Equal(t, &testConfig.Kafka.Consumer, consumerConfig)

	// 验证模拟调用
	mockLoader.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
}

// TestConsumerGroupHandler 测试消费者组处理器
func TestConsumerGroupHandler(t *testing.T) {
	// 创建模拟消费者
	mockProcessor := &MockMessageProcessor{}
	mockErrorHandler := &MockErrorHandler{}

	kafkaConsumer, err := consumer.NewKafkaConsumer(
		[]string{"localhost:9092"},
		[]string{"test-topic"},
		&config.KafkaConsumer{
			GroupID:        "test-group",
			SessionTimeout: 30 * time.Second,
		},
		mockProcessor,
		mockErrorHandler,
	)
	require.NoError(t, err)

	handler := consumer.NewConsumerGroupHandler(kafkaConsumer)
	require.NotNil(t, handler)

	// 创建模拟会话
	mockSession := &MockConsumerGroupSession{}
	mockClaim := &MockConsumerGroupClaim{}

	// 设置模拟期望
	mockSession.On("MemberID").Return("test-member")
	mockSession.On("GenerationID").Return(int32(1))
	mockSession.On("Claims").Return(map[string][]int32{
		"test-topic": {0, 1},
	})

	// 测试Setup
	err = handler.Setup(mockSession)
	assert.NoError(t, err)

	// 测试Cleanup
	err = handler.Cleanup(mockSession)
	assert.NoError(t, err)

	// 验证模拟调用
	mockSession.AssertExpectations(t)
}

// TestMessageValidation 测试消息验证
func TestMessageValidation(t *testing.T) {
	validator := &consumer.BasicMessageValidator{}

	tests := []struct {
		name        string
		message     *models.KafkaMessage
		expectError bool
	}{
		{
			name: "valid_message",
			message: &models.KafkaMessage{
				MessageID:   "test-id",
				MessageType: models.MessageTypeDeviceData,
				Timestamp:   time.Now().Unix(),
				Source:      "test-source",
				Payload:     map[string]interface{}{"test": "data"},
				Priority:    models.PriorityNormal,
			},
			expectError: false,
		},
		{
			name: "missing_message_id",
			message: &models.KafkaMessage{
				MessageType: models.MessageTypeDeviceData,
				Timestamp:   time.Now().Unix(),
				Source:      "test-source",
				Payload:     map[string]interface{}{"test": "data"},
				Priority:    models.PriorityNormal,
			},
			expectError: true,
		},
		{
			name: "invalid_priority",
			message: &models.KafkaMessage{
				MessageID:   "test-id",
				MessageType: models.MessageTypeDeviceData,
				Timestamp:   time.Now().Unix(),
				Source:      "test-source",
				Payload:     map[string]interface{}{"test": "data"},
				Priority:    models.MessagePriority(15), // 无效优先级
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(context.Background(), tt.message)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMessageTransformation 测试消息转换
func TestMessageTransformation(t *testing.T) {
	transformer := &consumer.TimestampTransformer{}

	// 测试秒级时间戳转换
	message := &models.KafkaMessage{
		MessageID:   "test-id",
		MessageType: models.MessageTypeDeviceData,
		Timestamp:   1609459200, // 2021-01-01 00:00:00 (秒级)
		Source:      "test-source",
		Payload:     map[string]interface{}{"test": "data"},
		Priority:    models.PriorityNormal,
	}

	transformedMessage, err := transformer.Transform(context.Background(), message)
	assert.NoError(t, err)
	assert.Equal(t, int64(1609459200000), transformedMessage.Timestamp) // 应该转换为毫秒级

	// 测试毫秒级时间戳（不应该改变）
	message.Timestamp = 1609459200000 // 毫秒级
	transformedMessage, err = transformer.Transform(context.Background(), message)
	assert.NoError(t, err)
	assert.Equal(t, int64(1609459200000), transformedMessage.Timestamp) // 应该保持不变
}

// 辅助类型和函数

// ValidationError 验证错误类型
type ValidationError struct {
	Message string
}

func (ve *ValidationError) Error() string {
	return ve.Message
}

// MockConfigLoader 模拟配置加载器
type MockConfigLoader struct {
	mock.Mock
}

func (m *MockConfigLoader) LoadConfig(ctx context.Context) (*config.AppConfig, error) {
	args := m.Called(ctx)
	return args.Get(0).(*config.AppConfig), args.Error(1)
}

func (m *MockConfigLoader) GetConfigSource() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockConfigLoader) SupportsHotReload() bool {
	args := m.Called()
	return args.Bool(0)
}

// MockConfigValidator 模拟配置验证器
type MockConfigValidator struct {
	mock.Mock
}

func (m *MockConfigValidator) ValidateConfig(cfg *config.AppConfig) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *MockConfigValidator) ValidateConsumerConfig(cfg *config.KafkaConsumer) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *MockConfigValidator) GetValidationRules() []consumer.ValidationRule {
	args := m.Called()
	return args.Get(0).([]consumer.ValidationRule)
}

// MockConsumerGroupSession 模拟消费者组会话
type MockConsumerGroupSession struct {
	mock.Mock
}

func (m *MockConsumerGroupSession) Claims() map[string][]int32 {
	args := m.Called()
	return args.Get(0).(map[string][]int32)
}

func (m *MockConsumerGroupSession) MemberID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockConsumerGroupSession) GenerationID() int32 {
	args := m.Called()
	return args.Get(0).(int32)
}

func (m *MockConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
	m.Called(topic, partition, offset, metadata)
}

func (m *MockConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
	m.Called(topic, partition, offset, metadata)
}

func (m *MockConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
	m.Called(msg, metadata)
}

func (m *MockConsumerGroupSession) Context() context.Context {
	args := m.Called()
	return args.Get(0).(context.Context)
}

func (m *MockConsumerGroupSession) Commit() {
	m.Called()
}

// MockConsumerGroupClaim 模拟消费者组声明
type MockConsumerGroupClaim struct {
	mock.Mock
}

func (m *MockConsumerGroupClaim) Topic() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockConsumerGroupClaim) Partition() int32 {
	args := m.Called()
	return args.Get(0).(int32)
}

func (m *MockConsumerGroupClaim) InitialOffset() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

func (m *MockConsumerGroupClaim) HighWaterMarkOffset() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

func (m *MockConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	args := m.Called()
	return args.Get(0).(<-chan *sarama.ConsumerMessage)
}

// TestIntegration 集成测试
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 这里可以添加需要真实Kafka的集成测试
	// 例如启动嵌入式Kafka或连接到测试Kafka集群
	t.Log("集成测试需要真实的Kafka环境")
}

// BenchmarkMessageProcessing 消息处理性能测试
func BenchmarkMessageProcessing(b *testing.B) {
	config := consumer.ProcessorConfig{
		MaxRetries:           1,
		RetryDelay:           1 * time.Millisecond,
		ProcessingTimeout:    1 * time.Second,
		EnableValidation:     true,
		EnableTransformation: false,
		EnableCaching:        false,
	}

	processor := consumer.NewDefaultMessageProcessor(config)
	processor.AddValidator(&consumer.BasicMessageValidator{})
	processor.RegisterHandler(consumer.NewDeviceDataHandler(nil))

	message := &models.KafkaMessage{
		MessageID:   "bench-message",
		MessageType: models.MessageTypeDeviceData,
		Timestamp:   time.Now().Unix(),
		Source:      "bench-source",
		Payload: &models.DeviceDataPayload{
			Device: &models.Device{
				DeviceID: "device-001",
				Name:     "Bench Device",
				Type:     "sensor",
				Status:   "active",
			},
		},
		Priority: models.PriorityNormal,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processor.ProcessMessage(ctx, message)
	}
}
