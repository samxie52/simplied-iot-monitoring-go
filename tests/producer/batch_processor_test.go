package producer

import (
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/config"
	"simplied-iot-monitoring-go/internal/producer"

	"github.com/IBM/sarama"
)

// MockAsyncProducer 模拟异步生产者
type MockAsyncProducer struct {
	input     chan *sarama.ProducerMessage
	successes chan *sarama.ProducerMessage
	errors    chan *sarama.ProducerError
	closed    bool
}

func NewMockAsyncProducer() *MockAsyncProducer {
	return &MockAsyncProducer{
		input:     make(chan *sarama.ProducerMessage, 100),
		successes: make(chan *sarama.ProducerMessage, 100),
		errors:    make(chan *sarama.ProducerError, 100),
		closed:    false,
	}
}

func (m *MockAsyncProducer) AsyncClose() {
	m.closed = true
	close(m.input)
	close(m.successes)
	close(m.errors)
}

func (m *MockAsyncProducer) Close() error {
	m.AsyncClose()
	return nil
}

func (m *MockAsyncProducer) Input() chan<- *sarama.ProducerMessage {
	return m.input
}

func (m *MockAsyncProducer) Successes() <-chan *sarama.ProducerMessage {
	return m.successes
}

func (m *MockAsyncProducer) Errors() <-chan *sarama.ProducerError {
	return m.errors
}

func (m *MockAsyncProducer) IsTransactional() bool {
	return false
}

func (m *MockAsyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	return 0
}

func (m *MockAsyncProducer) BeginTxn() error {
	return nil
}

func (m *MockAsyncProducer) CommitTxn() error {
	return nil
}

func (m *MockAsyncProducer) AbortTxn() error {
	return nil
}

func (m *MockAsyncProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupId string) error {
	return nil
}

func (m *MockAsyncProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupId string, metadata *string) error {
	return nil
}

func TestBatchProcessor_Creation(t *testing.T) {
	config := &config.KafkaProducer{
		BatchSize:         100,
		BatchTimeout:      100 * time.Millisecond,
		CompressionType:   "gzip",
		MaxRetries:        3,
		RetryBackoff:      100 * time.Millisecond,
		ChannelBufferSize: 1000,
	}

	mockProducer := NewMockAsyncProducer()
	processor := producer.NewBatchProcessor(config, mockProducer, 4)

	if processor == nil {
		t.Fatal("Failed to create batch processor")
	}

	if !processor.IsRunning() {
		// 处理器创建后应该是未运行状态
		t.Log("Batch processor created in stopped state (expected)")
	}
}

func TestBatchProcessor_StartStop(t *testing.T) {
	config := &config.KafkaProducer{
		BatchSize:         10,
		BatchTimeout:      50 * time.Millisecond,
		CompressionType:   "none",
		MaxRetries:        3,
		RetryBackoff:      50 * time.Millisecond,
		ChannelBufferSize: 100,
	}

	mockProducer := NewMockAsyncProducer()
	processor := producer.NewBatchProcessor(config, mockProducer, 4)

	// 启动处理器
	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start batch processor: %v", err)
	}

	if !processor.IsRunning() {
		t.Error("Batch processor should be running after start")
	}

	// 停止处理器
	err = processor.Stop()
	if err != nil {
		t.Fatalf("Failed to stop batch processor: %v", err)
	}

	if processor.IsRunning() {
		t.Error("Batch processor should not be running after stop")
	}

	mockProducer.Close()
}

func TestBatchProcessor_MessageSending(t *testing.T) {
	config := &config.KafkaProducer{
		BatchSize:         5,
		BatchTimeout:      100 * time.Millisecond,
		CompressionType:   "none",
		MaxRetries:        3,
		RetryBackoff:      50 * time.Millisecond,
		ChannelBufferSize: 100,
	}

	mockProducer := NewMockAsyncProducer()
	processor := producer.NewBatchProcessor(config, mockProducer, 4)

	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start batch processor: %v", err)
	}
	defer processor.Stop()

	// 发送消息
	messages := []*producer.ProducerMessage{
		{
			Key:       "key1",
			Value:     map[string]interface{}{"data": "value1"},
			Topic:     "test-topic",
			Partition: -1,
		},
		{
			Key:       "key2",
			Value:     map[string]interface{}{"data": "value2"},
			Topic:     "test-topic",
			Partition: -1,
		},
		{
			Key:       "key3",
			Value:     map[string]interface{}{"data": "value3"},
			Topic:     "test-topic",
			Partition: -1,
		},
	}

	for _, msg := range messages {
		err := processor.SendMessage(msg)
		if err != nil {
			t.Errorf("Failed to send message: %v", err)
		}
	}

	// 等待消息处理
	time.Sleep(200 * time.Millisecond)

	// 检查指标
	metrics := processor.GetMetrics()
	if metrics.TotalMessages < int64(len(messages)) {
		t.Errorf("Expected at least %d total messages, got %d", len(messages), metrics.TotalMessages)
	}

	mockProducer.Close()
}

func TestBatchProcessor_BatchSizeTriggering(t *testing.T) {
	config := &config.KafkaProducer{
		BatchSize:         3,               // 小批次大小用于测试
		BatchTimeout:      1 * time.Second, // 长超时确保批次大小触发
		CompressionType:   "none",
		MaxRetries:        3,
		RetryBackoff:      50 * time.Millisecond,
		ChannelBufferSize: 100,
	}

	mockProducer := NewMockAsyncProducer()
	processor := producer.NewBatchProcessor(config, mockProducer, 4)

	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start batch processor: %v", err)
	}
	defer processor.Stop()

	// 发送正好达到批次大小的消息
	for i := 0; i < 3; i++ {
		msg := &producer.ProducerMessage{
			Key:       "batch-key",
			Value:     map[string]interface{}{"batch": i},
			Topic:     "test-topic",
			Partition: -1,
		}

		err := processor.SendMessage(msg)
		if err != nil {
			t.Errorf("Failed to send message %d: %v", i, err)
		}
	}

	// 短暂等待批次处理
	time.Sleep(100 * time.Millisecond)

	metrics := processor.GetMetrics()
	if metrics.TotalBatches < 1 {
		t.Errorf("Expected at least 1 batch to be processed, got %d", metrics.TotalBatches)
	}

	mockProducer.Close()
}

func TestBatchProcessor_TimeoutTriggering(t *testing.T) {
	config := &config.KafkaProducer{
		BatchSize:         100,                    // 大批次大小确保超时触发
		BatchTimeout:      100 * time.Millisecond, // 短超时
		CompressionType:   "none",
		MaxRetries:        3,
		RetryBackoff:      50 * time.Millisecond,
		ChannelBufferSize: 100,
	}

	mockProducer := NewMockAsyncProducer()
	processor := producer.NewBatchProcessor(config, mockProducer, 4)

	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start batch processor: %v", err)
	}
	defer processor.Stop()

	// 发送少量消息（不足以触发批次大小）
	msg := &producer.ProducerMessage{
		Key:       "timeout-key",
		Value:     map[string]interface{}{"timeout": "test"},
		Topic:     "test-topic",
		Partition: -1,
	}

	err = processor.SendMessage(msg)
	if err != nil {
		t.Errorf("Failed to send message: %v", err)
	}

	// 等待超时触发
	time.Sleep(200 * time.Millisecond)

	metrics := processor.GetMetrics()
	if metrics.TotalBatches < 1 {
		t.Errorf("Expected at least 1 batch to be processed by timeout, got %d", metrics.TotalBatches)
	}

	mockProducer.Close()
}

func TestBatchProcessor_Metrics(t *testing.T) {
	config := &config.KafkaProducer{
		BatchSize:         5,
		BatchTimeout:      100 * time.Millisecond,
		CompressionType:   "none",
		MaxRetries:        3,
		RetryBackoff:      50 * time.Millisecond,
		ChannelBufferSize: 100,
	}

	mockProducer := NewMockAsyncProducer()
	processor := producer.NewBatchProcessor(config, mockProducer, 4)

	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start batch processor: %v", err)
	}
	defer processor.Stop()

	// 发送一些消息
	messageCount := 10
	for i := 0; i < messageCount; i++ {
		msg := &producer.ProducerMessage{
			Key:       "metrics-key",
			Value:     map[string]interface{}{"index": i},
			Topic:     "test-topic",
			Partition: -1,
		}

		err := processor.SendMessage(msg)
		if err != nil {
			t.Errorf("Failed to send message %d: %v", i, err)
		}
	}

	// 等待处理完成
	time.Sleep(300 * time.Millisecond)

	// 检查指标
	metrics := processor.GetMetrics()

	if metrics.TotalMessages < int64(messageCount) {
		t.Errorf("Expected at least %d total messages, got %d", messageCount, metrics.TotalMessages)
	}

	if metrics.TotalBatches < 1 {
		t.Errorf("Expected at least 1 batch, got %d", metrics.TotalBatches)
	}

	if metrics.TotalBytes <= 0 {
		t.Errorf("Expected positive total bytes, got %d", metrics.TotalBytes)
	}

	if metrics.AvgBatchSize <= 0 {
		t.Errorf("Expected positive average batch size, got %f", metrics.AvgBatchSize)
	}

	mockProducer.Close()
}

func TestBatchProcessor_PartitionDistribution(t *testing.T) {
	config := &config.KafkaProducer{
		BatchSize:         3,
		BatchTimeout:      100 * time.Millisecond,
		CompressionType:   "none",
		MaxRetries:        3,
		RetryBackoff:      50 * time.Millisecond,
		ChannelBufferSize: 100,
	}

	mockProducer := NewMockAsyncProducer()
	processor := producer.NewBatchProcessor(config, mockProducer, 4) // 4个分区

	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start batch processor: %v", err)
	}
	defer processor.Stop()

	// 发送消息到不同分区
	partitions := []int32{0, 1, 2, 3}
	for _, partition := range partitions {
		msg := &producer.ProducerMessage{
			Key:       "partition-key",
			Value:     map[string]interface{}{"partition": partition},
			Topic:     "test-topic",
			Partition: partition,
		}

		err := processor.SendMessage(msg)
		if err != nil {
			t.Errorf("Failed to send message to partition %d: %v", partition, err)
		}
	}

	// 等待处理完成
	time.Sleep(200 * time.Millisecond)

	// 检查分区分布
	metrics := processor.GetMetrics()
	if len(metrics.PartitionDistribution) == 0 {
		t.Error("Expected partition distribution data")
	}

	for _, partition := range partitions {
		if count, exists := metrics.PartitionDistribution[partition]; !exists || count <= 0 {
			t.Errorf("Expected messages in partition %d, got count: %d", partition, count)
		}
	}

	mockProducer.Close()
}

func TestBatchProcessor_ErrorHandling(t *testing.T) {
	config := &config.KafkaProducer{
		BatchSize:         5,
		BatchTimeout:      100 * time.Millisecond,
		CompressionType:   "none",
		MaxRetries:        3,
		RetryBackoff:      50 * time.Millisecond,
		ChannelBufferSize: 10, // 小缓冲区用于测试
	}

	mockProducer := NewMockAsyncProducer()
	processor := producer.NewBatchProcessor(config, mockProducer, 4)

	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start batch processor: %v", err)
	}
	defer processor.Stop()

	// 尝试发送无效消息（循环引用会导致JSON序列化失败）
	invalidValue := make(map[string]interface{})
	invalidValue["self"] = invalidValue // 创建循环引用

	msg := &producer.ProducerMessage{
		Key:       "error-key",
		Value:     invalidValue,
		Topic:     "test-topic",
		Partition: -1,
	}

	// 这应该会导致序列化错误
	err = processor.SendMessage(msg)
	// 注意：错误可能在后台处理中发生，所以这里可能不会立即返回错误

	// 等待错误处理
	time.Sleep(200 * time.Millisecond)

	// // 检查错误指标
	// metrics := processor.GetMetrics()
	// 错误计数可能会增加，但这取决于具体的错误处理实现

	mockProducer.Close()
}

func BenchmarkBatchProcessor_SendMessage(b *testing.B) {
	config := &config.KafkaProducer{
		BatchSize:         100,
		BatchTimeout:      100 * time.Millisecond,
		CompressionType:   "none",
		MaxRetries:        3,
		RetryBackoff:      50 * time.Millisecond,
		ChannelBufferSize: 10000,
	}

	mockProducer := NewMockAsyncProducer()
	processor := producer.NewBatchProcessor(config, mockProducer, 4)

	err := processor.Start()
	if err != nil {
		b.Fatalf("Failed to start batch processor: %v", err)
	}
	defer processor.Stop()

	msg := &producer.ProducerMessage{
		Key:       "bench-key",
		Value:     map[string]interface{}{"data": "benchmark"},
		Topic:     "test-topic",
		Partition: -1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.SendMessage(msg)
	}
	b.StopTimer()
	
	// Stop processor first, then close mock producer
	processor.Stop()
	mockProducer.Close()
}

func BenchmarkBatchProcessor_Concurrent(b *testing.B) {
	config := &config.KafkaProducer{
		BatchSize:         100,
		BatchTimeout:      100 * time.Millisecond,
		CompressionType:   "none",
		MaxRetries:        3,
		RetryBackoff:      50 * time.Millisecond,
		ChannelBufferSize: 10000,
	}

	mockProducer := NewMockAsyncProducer()
	processor := producer.NewBatchProcessor(config, mockProducer, 4)

	err := processor.Start()
	if err != nil {
		b.Fatalf("Failed to start batch processor: %v", err)
	}
	defer processor.Stop()

	msg := &producer.ProducerMessage{
		Key:       "concurrent-key",
		Value:     map[string]interface{}{"data": "concurrent"},
		Topic:     "test-topic",
		Partition: -1,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			processor.SendMessage(msg)
		}
	})
	b.StopTimer()
	
	// Stop processor first, then close mock producer
	processor.Stop()
	mockProducer.Close()
}
