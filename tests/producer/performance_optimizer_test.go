package producer

import (
	"testing"
	"unsafe"

	"simplied-iot-monitoring-go/internal/producer"
)

func TestMemoryPool_Creation(t *testing.T) {
	pool := producer.NewMemoryPool()

	if pool == nil {
		t.Fatal("Failed to create memory pool")
	}
}

func TestMemoryPool_GetAndPut(t *testing.T) {
	pool := producer.NewMemoryPool()

	// 测试获取不同大小的缓冲区
	sizes := []int{64, 128, 256, 512, 1024, 2048, 4096}

	for _, size := range sizes {
		buf := pool.Get(size)
		if buf == nil {
			t.Errorf("Failed to get buffer of size %d", size)
			continue
		}

		if len(buf) != size {
			t.Errorf("Expected buffer size %d, got %d", size, len(buf))
		}

		// 写入一些数据
		for i := 0; i < len(buf); i++ {
			buf[i] = byte(i % 256)
		}

		// 放回池中
		pool.Put(buf)
	}

	t.Log("Memory pool get/put operations completed successfully")
}

func TestMemoryPool_CustomSize(t *testing.T) {
	pool := producer.NewMemoryPool()

	// 测试非标准大小
	customSizes := []int{100, 300, 700, 1500, 3000}

	for _, size := range customSizes {
		buf := pool.Get(size)
		if buf == nil {
			t.Errorf("Failed to get buffer of custom size %d", size)
			continue
		}

		if len(buf) != size {
			t.Errorf("Expected buffer size %d, got %d", size, len(buf))
		}

		pool.Put(buf)
	}

	t.Log("Custom size buffer operations completed successfully")
}

func TestZeroCopyBuffer_Creation(t *testing.T) {
	data := []byte("Hello, Zero Copy World!")
	zcb := producer.NewZeroCopyBuffer(data)

	if zcb == nil {
		t.Fatal("Failed to create zero copy buffer")
	}

	if zcb.Len() != len(data) {
		t.Errorf("Expected length %d, got %d", len(data), zcb.Len())
	}
}

func TestZeroCopyBuffer_Bytes(t *testing.T) {
	original := []byte("Test data for zero copy")
	zcb := producer.NewZeroCopyBuffer(original)

	result := zcb.Bytes()

	// 验证数据相同
	if len(result) != len(original) {
		t.Errorf("Expected length %d, got %d", len(original), len(result))
	}

	for i, b := range result {
		if b != original[i] {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, original[i], b)
		}
	}

	// 验证是否为零拷贝（相同的底层数组）
	if uintptr(unsafe.Pointer(&result[0])) != uintptr(unsafe.Pointer(&original[0])) {
		t.Error("Expected zero copy, but data was copied")
	}

	t.Log("Zero copy buffer bytes operation verified")
}

func TestZeroCopyBuffer_String(t *testing.T) {
	data := []byte("Zero copy string test")
	zcb := producer.NewZeroCopyBuffer(data)

	str := zcb.String()
	expected := string(data)

	if str != expected {
		t.Errorf("Expected string %q, got %q", expected, str)
	}

	t.Log("Zero copy buffer string operation completed")
}

func TestZeroCopyBuffer_Slice(t *testing.T) {
	data := []byte("0123456789")
	zcb := producer.NewZeroCopyBuffer(data)

	// 测试正常切片
	slice := zcb.Slice(2, 7)
	if slice == nil {
		t.Fatal("Failed to create slice")
	}

	if slice.Len() != 5 {
		t.Errorf("Expected slice length 5, got %d", slice.Len())
	}

	expected := "23456"
	if slice.String() != expected {
		t.Errorf("Expected slice %q, got %q", expected, slice.String())
	}

	// 测试边界情况
	if zcb.Slice(-1, 5) != nil {
		t.Error("Expected nil for negative start index")
	}

	if zcb.Slice(5, 15) != nil {
		t.Error("Expected nil for end index beyond length")
	}

	if zcb.Slice(5, 3) != nil {
		t.Error("Expected nil for start > end")
	}

	t.Log("Zero copy buffer slice operations completed")
}

func TestMessageBuffer_Creation(t *testing.T) {
	pool := producer.NewMemoryPool()
	msgBuf := producer.NewMessageBuffer(1024, pool)

	if msgBuf == nil {
		t.Fatal("Failed to create message buffer")
	}

	// 测试写入数据
	data := []byte("Test message data")
	n := msgBuf.Write(data)

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// 获取数据
	result := msgBuf.Bytes()
	for i, b := range data {
		if result[i] != b {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, b, result[i])
		}
	}

	// 释放缓冲区
	msgBuf.Release()

	t.Log("Message buffer operations completed successfully")
}

func TestBatchBuffer_Operations(t *testing.T) {
	capacity := 10
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
		data := []byte("message " + string(rune('0'+i)))
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

	// 尝试添加更多消息（应该失败）
	extraMsg := producer.NewMessageBuffer(64, pool)
	if batchBuf.Add(extraMsg) {
		t.Error("Expected to fail adding message to full buffer")
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

	t.Log("Batch buffer operations completed successfully")
}

func TestPerformanceOptimizer_Creation(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "optimizer")
	optimizer := producer.NewPerformanceOptimizer(100, 5, metrics)

	if optimizer == nil {
		t.Fatal("Failed to create performance optimizer")
	}

	stats := optimizer.GetStats()
	if stats == nil {
		t.Error("Failed to get optimizer stats")
	}

	t.Logf("Optimizer stats: %+v", stats)
}

func TestPerformanceOptimizer_MessageBuffer(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "optimizer")
	optimizer := producer.NewPerformanceOptimizer(100, 5, metrics)

	// 测试获取消息缓冲区
	sizes := []int{64, 128, 256, 512, 1024}

	for _, size := range sizes {
		msgBuf := optimizer.GetMessageBuffer(size)
		if msgBuf == nil {
			t.Errorf("Failed to get message buffer of size %d", size)
			continue
		}

		// 写入数据
		data := make([]byte, size/2)
		for i := range data {
			data[i] = byte(i % 256)
		}

		n := msgBuf.Write(data)
		if n != len(data) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
		}

		// 释放缓冲区
		msgBuf.Release()
	}

	t.Log("Message buffer operations completed successfully")
}

func TestPerformanceOptimizer_BatchBuffer(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "optimizer")
	optimizer := producer.NewPerformanceOptimizer(100, 5, metrics)

	// 获取批次缓冲区
	batchBuf := optimizer.GetBatchBuffer()
	if batchBuf == nil {
		t.Fatal("Failed to get batch buffer")
	}

	// 添加一些消息
	for i := 0; i < 10; i++ {
		msgBuf := optimizer.GetMessageBuffer(64)
		data := []byte("test message")
		msgBuf.Write(data)
		batchBuf.Add(msgBuf)
	}

	if batchBuf.Count() != 10 {
		t.Errorf("Expected 10 messages in batch, got %d", batchBuf.Count())
	}

	// 释放批次缓冲区
	optimizer.ReleaseBatchBuffer(batchBuf)

	if batchBuf.Count() != 0 {
		t.Errorf("Expected 0 messages after release, got %d", batchBuf.Count())
	}

	t.Log("Batch buffer operations completed successfully")
}

func TestPerformanceOptimizer_ZeroCopyBuffer(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "optimizer")
	optimizer := producer.NewPerformanceOptimizer(100, 5, metrics)

	data := []byte("Zero copy test data")
	zcb := optimizer.CreateZeroCopyBuffer(data)

	if zcb == nil {
		t.Fatal("Failed to create zero copy buffer")
	}

	if zcb.Len() != len(data) {
		t.Errorf("Expected length %d, got %d", len(data), zcb.Len())
	}

	result := zcb.Bytes()
	for i, b := range data {
		if result[i] != b {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, b, result[i])
		}
	}

	t.Log("Zero copy buffer creation completed successfully")
}

func TestPerformanceOptimizer_OptimizeMessage(t *testing.T) {
	metrics := producer.NewPrometheusMetrics("test", "optimizer")
	optimizer := producer.NewPerformanceOptimizer(100, 5, metrics)

	testData := [][]byte{
		[]byte("small"),
		[]byte("medium size message data"),
		make([]byte, 1000), // large message
		make([]byte, 5000), // very large message
	}

	for i, data := range testData {
		msgBuf, err := optimizer.OptimizeMessage(data)
		if err != nil {
			t.Errorf("Failed to optimize message %d: %v", i, err)
			continue
		}

		if msgBuf == nil {
			t.Errorf("Got nil message buffer for message %d", i)
			continue
		}

		// 验证数据
		result := msgBuf.Bytes()
		for j := 0; j < len(data); j++ {
			if result[j] != data[j] {
				t.Errorf("Data mismatch in message %d at index %d", i, j)
				break
			}
		}

		msgBuf.Release()
	}

	t.Log("Message optimization completed successfully")
}

func TestStringToBytes_ZeroCopy(t *testing.T) {
	str := "Test string for zero copy conversion"
	bytes := producer.StringToBytes(str)

	if len(bytes) != len(str) {
		t.Errorf("Expected length %d, got %d", len(str), len(bytes))
	}

	for i, b := range bytes {
		if b != str[i] {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, str[i], b)
		}
	}

	// 验证零拷贝
	if uintptr(unsafe.Pointer(&bytes[0])) != uintptr(unsafe.Pointer(unsafe.StringData(str))) {
		t.Error("Expected zero copy conversion, but data was copied")
	}

	t.Log("String to bytes zero copy conversion verified")
}

func TestBytesToString_ZeroCopy(t *testing.T) {
	bytes := []byte("Test bytes for zero copy conversion")
	str := producer.BytesToString(bytes)

	if len(str) != len(bytes) {
		t.Errorf("Expected length %d, got %d", len(bytes), len(str))
	}

	for i, c := range str {
		if byte(c) != bytes[i] {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, bytes[i], byte(c))
		}
	}

	// 验证零拷贝
	if uintptr(unsafe.Pointer(unsafe.StringData(str))) != uintptr(unsafe.Pointer(&bytes[0])) {
		t.Error("Expected zero copy conversion, but data was copied")
	}

	t.Log("Bytes to string zero copy conversion verified")
}

func BenchmarkMemoryPool_Get(b *testing.B) {
	pool := producer.NewMemoryPool()
	size := 1024

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get(size)
		pool.Put(buf)
	}
}

func BenchmarkPerformanceOptimizer_GetMessageBuffer(b *testing.B) {
	metrics := producer.NewPrometheusMetrics("test", "optimizer")
	optimizer := producer.NewPerformanceOptimizer(100, 5, metrics)
	size := 1024

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msgBuf := optimizer.GetMessageBuffer(size)
		msgBuf.Release()
	}
}

func BenchmarkZeroCopyBuffer_Operations(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		zcb := producer.NewZeroCopyBuffer(data)
		_ = zcb.Bytes()
		_ = zcb.String()
		_ = zcb.Slice(10, 100)
	}
}

func BenchmarkStringToBytes_ZeroCopy(b *testing.B) {
	str := "This is a test string for zero copy benchmark"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = producer.StringToBytes(str)
	}
}

func BenchmarkBytesToString_ZeroCopy(b *testing.B) {
	bytes := []byte("This is a test byte slice for zero copy benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = producer.BytesToString(bytes)
	}
}
