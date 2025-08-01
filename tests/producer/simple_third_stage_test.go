package producer

import (
	"testing"
	"time"

	"simplied-iot-monitoring-go/internal/producer"
)

// 简化的第三阶段功能测试

func TestPrometheusMetrics_BasicFunctionality(t *testing.T) {
	// 测试Prometheus指标的基本功能
	metrics := producer.NewPrometheusMetrics("test", "kafka_producer")
	if metrics == nil {
		t.Fatal("Failed to create Prometheus metrics")
	}

	// 测试基本的指标操作
	metrics.IncrementMessages()
	metrics.IncrementBytes(1024)
	metrics.SetDeviceCount(100)
	metrics.RecordSendLatency(10 * time.Millisecond)
	metrics.UpdateResourceMetrics()

	// 测试运行时间
	uptime := metrics.GetUptime()
	if uptime <= 0 {
		t.Errorf("Expected positive uptime, got %v", uptime)
	}

	t.Log("Prometheus metrics basic functionality test passed")
}

func TestMemoryPool_BasicOperations(t *testing.T) {
	// 测试内存池的基本操作
	pool := producer.NewMemoryPool()
	if pool == nil {
		t.Fatal("Failed to create memory pool")
	}

	// 测试获取和释放缓冲区
	sizes := []int{64, 128, 256, 512, 1024}
	for _, size := range sizes {
		buf := pool.Get(size)
		if buf == nil {
			t.Errorf("Failed to get buffer of size %d", size)
			continue
		}
		if len(buf) != size {
			t.Errorf("Expected buffer size %d, got %d", size, len(buf))
		}
		pool.Put(buf)
	}

	t.Log("Memory pool basic operations test passed")
}

func TestZeroCopyBuffer_Operations(t *testing.T) {
	// 测试零拷贝缓冲区
	data := []byte("Hello, Zero Copy World!")
	zcb := producer.NewZeroCopyBuffer(data)
	if zcb == nil {
		t.Fatal("Failed to create zero copy buffer")
	}

	if zcb.Len() != len(data) {
		t.Errorf("Expected length %d, got %d", len(data), zcb.Len())
	}

	result := zcb.Bytes()
	if len(result) != len(data) {
		t.Errorf("Expected bytes length %d, got %d", len(data), len(result))
	}

	str := zcb.String()
	expected := string(data)
	if str != expected {
		t.Errorf("Expected string %q, got %q", expected, str)
	}

	// 测试切片操作
	slice := zcb.Slice(0, 5)
	if slice == nil {
		t.Error("Failed to create slice")
	} else if slice.String() != "Hello" {
		t.Errorf("Expected slice 'Hello', got %q", slice.String())
	}

	t.Log("Zero copy buffer operations test passed")
}

func TestMessageBuffer_BasicOperations(t *testing.T) {
	// 测试消息缓冲区
	pool := producer.NewMemoryPool()
	msgBuf := producer.NewMessageBuffer(1024, pool)
	if msgBuf == nil {
		t.Fatal("Failed to create message buffer")
	}

	// 写入数据
	data := []byte("Test message data")
	n := msgBuf.Write(data)
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// 获取数据
	result := msgBuf.Bytes()
	if len(result) < len(data) {
		t.Errorf("Expected at least %d bytes, got %d", len(data), len(result))
	}

	// 验证数据内容
	for i, b := range data {
		if result[i] != b {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, b, result[i])
		}
	}

	// 释放缓冲区
	msgBuf.Release()

	t.Log("Message buffer basic operations test passed")
}

func TestBatchBuffer_Operations(t *testing.T) {
	// 测试批次缓冲区
	capacity := 5
	batchBuf := producer.NewBatchBuffer(capacity)
	if batchBuf == nil {
		t.Fatal("Failed to create batch buffer")
	}

	if batchBuf.Count() != 0 {
		t.Errorf("Expected initial count 0, got %d", batchBuf.Count())
	}

	if batchBuf.IsFull() {
		t.Error("Expected batch buffer not to be full initially")
	}

	// 添加消息
	pool := producer.NewMemoryPool()
	for i := 0; i < capacity; i++ {
		msgBuf := producer.NewMessageBuffer(64, pool)
		data := []byte("message")
		msgBuf.Write(data)

		if !batchBuf.Add(msgBuf) {
			t.Errorf("Failed to add message %d", i)
		}
	}

	if batchBuf.Count() != capacity {
		t.Errorf("Expected count %d, got %d", capacity, batchBuf.Count())
	}

	if !batchBuf.IsFull() {
		t.Error("Expected batch buffer to be full")
	}

	// 获取消息
	messages := batchBuf.GetMessages()
	if len(messages) != capacity {
		t.Errorf("Expected %d messages, got %d", capacity, len(messages))
	}

	// 清空缓冲区
	batchBuf.Clear()
	if batchBuf.Count() != 0 {
		t.Errorf("Expected count 0 after clear, got %d", batchBuf.Count())
	}

	t.Log("Batch buffer operations test passed")
}

func TestPerformanceOptimizer_BasicOperations(t *testing.T) {
	// 测试性能优化器
	metrics := producer.NewPrometheusMetrics("test", "optimizer")
	optimizer := producer.NewPerformanceOptimizer(100, 5, metrics)
	if optimizer == nil {
		t.Fatal("Failed to create performance optimizer")
	}

	// 测试获取消息缓冲区
	msgBuf := optimizer.GetMessageBuffer(1024)
	if msgBuf == nil {
		t.Error("Failed to get message buffer")
	} else {
		data := []byte("test data")
		n := msgBuf.Write(data)
		if n != len(data) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
		}
		msgBuf.Release()
	}

	// 测试获取批次缓冲区
	batchBuf := optimizer.GetBatchBuffer()
	if batchBuf == nil {
		t.Error("Failed to get batch buffer")
	} else {
		optimizer.ReleaseBatchBuffer(batchBuf)
	}

	// 测试零拷贝缓冲区创建
	data := []byte("zero copy test")
	zcb := optimizer.CreateZeroCopyBuffer(data)
	if zcb == nil {
		t.Error("Failed to create zero copy buffer")
	} else if zcb.Len() != len(data) {
		t.Errorf("Expected length %d, got %d", len(data), zcb.Len())
	}

	// 测试消息优化
	testData := []byte("optimization test data")
	optimizedBuf, err := optimizer.OptimizeMessage(testData)
	if err != nil {
		t.Errorf("Failed to optimize message: %v", err)
	} else if optimizedBuf == nil {
		t.Error("Got nil optimized buffer")
	} else {
		optimizedBuf.Release()
	}

	// 获取统计信息
	stats := optimizer.GetStats()
	if stats == nil {
		t.Error("Failed to get optimizer stats")
	}

	t.Log("Performance optimizer basic operations test passed")
}

func TestZeroCopyConversions(t *testing.T) {
	// 测试零拷贝字符串转换
	str := "Test string for zero copy"
	bytes := producer.StringToBytes(str)
	if len(bytes) != len(str) {
		t.Errorf("Expected length %d, got %d", len(str), len(bytes))
	}

	convertedStr := producer.BytesToString(bytes)
	if convertedStr != str {
		t.Errorf("Expected string %q, got %q", str, convertedStr)
	}

	t.Log("Zero copy conversions test passed")
}

// 基准测试

func BenchmarkPrometheusMetrics_IncrementMessages(b *testing.B) {
	metrics := producer.NewPrometheusMetrics("bench", "kafka_producer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.IncrementMessages()
	}
}

func BenchmarkMemoryPool_GetPut(b *testing.B) {
	pool := producer.NewMemoryPool()
	size := 1024
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get(size)
		pool.Put(buf)
	}
}

func BenchmarkZeroCopyBuffer_Operations(b *testing.B) {
	data := make([]byte, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		zcb := producer.NewZeroCopyBuffer(data)
		_ = zcb.Bytes()
		_ = zcb.String()
	}
}

func BenchmarkStringToBytes_ZeroCopy(b *testing.B) {
	str := "This is a benchmark test string for zero copy conversion"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = producer.StringToBytes(str)
	}
}

func BenchmarkPerformanceOptimizer_GetMessageBuffer(b *testing.B) {
	metrics := producer.NewPrometheusMetrics("bench", "optimizer")
	optimizer := producer.NewPerformanceOptimizer(100, 5, metrics)
	size := 1024
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msgBuf := optimizer.GetMessageBuffer(size)
		msgBuf.Release()
	}
}
