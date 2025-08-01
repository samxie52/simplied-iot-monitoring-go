package producer

import (
	"testing"
	"time"
	"unsafe"
	"sync"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// 简化的PrometheusMetrics实现用于测试
type PrometheusMetrics struct {
	messagesProduced prometheus.Counter
	messagesSent     prometheus.Counter
	messagesErrors   prometheus.Counter
	batchSize        prometheus.Histogram
	latency          prometheus.Histogram
	mutex            sync.RWMutex
}

func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		messagesProduced: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "kafka_messages_produced_total",
			Help: "Total number of messages produced",
		}),
		messagesSent: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "kafka_messages_sent_total",
			Help: "Total number of messages sent",
		}),
		messagesErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "kafka_messages_errors_total",
			Help: "Total number of message errors",
		}),
		batchSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "kafka_batch_size",
			Help:    "Size of message batches",
			Buckets: prometheus.LinearBuckets(1, 10, 10),
		}),
		latency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "kafka_message_latency_seconds",
			Help:    "Message processing latency",
			Buckets: prometheus.DefBuckets,
		}),
	}
}

func (pm *PrometheusMetrics) IncrementMessagesProduced() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.messagesProduced.Inc()
}

func (pm *PrometheusMetrics) IncrementMessagesSent() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.messagesSent.Inc()
}

func (pm *PrometheusMetrics) IncrementMessagesErrors() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.messagesErrors.Inc()
}

func (pm *PrometheusMetrics) RecordBatchSize(size float64) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.batchSize.Observe(size)
}

func (pm *PrometheusMetrics) RecordLatency(duration time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.latency.Observe(duration.Seconds())
}

// 简化的MemoryPool实现
type MemoryPool struct {
	pool sync.Pool
}

func NewMemoryPool(size int) *MemoryPool {
	return &MemoryPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

func (mp *MemoryPool) Get() []byte {
	return mp.pool.Get().([]byte)
}

func (mp *MemoryPool) Put(buf []byte) {
	mp.pool.Put(buf)
}

// 简化的ZeroCopyBuffer实现
type ZeroCopyBuffer struct {
	data []byte
}

func NewZeroCopyBuffer(data []byte) *ZeroCopyBuffer {
	return &ZeroCopyBuffer{data: data}
}

func (zcb *ZeroCopyBuffer) Bytes() []byte {
	return zcb.data
}

func (zcb *ZeroCopyBuffer) String() string {
	return *(*string)(unsafe.Pointer(&zcb.data))
}

func (zcb *ZeroCopyBuffer) Slice(start, end int) []byte {
	if start < 0 || end > len(zcb.data) || start > end {
		return nil
	}
	return zcb.data[start:end]
}

// 零拷贝转换函数
func stringToBytesZeroCopy(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&struct {
		string
		int
	}{s, len(s)}))
}

func bytesToStringZeroCopy(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// 测试函数
func TestPrometheusMetrics_BasicOperations(t *testing.T) {
	metrics := NewPrometheusMetrics()
	
	// 测试计数器
	metrics.IncrementMessagesProduced()
	metrics.IncrementMessagesSent()
	metrics.IncrementMessagesErrors()
	
	// 测试直方图
	metrics.RecordBatchSize(10.0)
	metrics.RecordLatency(time.Millisecond * 100)
	
	// 验证计数器值
	if testutil.ToFloat64(metrics.messagesProduced) != 1 {
		t.Errorf("Expected messagesProduced to be 1, got %f", testutil.ToFloat64(metrics.messagesProduced))
	}
	
	if testutil.ToFloat64(metrics.messagesSent) != 1 {
		t.Errorf("Expected messagesSent to be 1, got %f", testutil.ToFloat64(metrics.messagesSent))
	}
	
	if testutil.ToFloat64(metrics.messagesErrors) != 1 {
		t.Errorf("Expected messagesErrors to be 1, got %f", testutil.ToFloat64(metrics.messagesErrors))
	}
	
	t.Log("PrometheusMetrics basic operations test passed")
}

func TestMemoryPool_BasicOperations(t *testing.T) {
	pool := NewMemoryPool(1024)
	
	// 获取缓冲区
	buf1 := pool.Get()
	if len(buf1) != 1024 {
		t.Errorf("Expected buffer size 1024, got %d", len(buf1))
	}
	
	// 写入数据
	copy(buf1, []byte("test data"))
	
	// 归还缓冲区
	pool.Put(buf1)
	
	// 再次获取（应该复用）
	buf2 := pool.Get()
	if len(buf2) != 1024 {
		t.Errorf("Expected buffer size 1024, got %d", len(buf2))
	}
	
	pool.Put(buf2)
	
	t.Log("Memory pool basic operations test passed")
}

func TestZeroCopyBuffer_Operations(t *testing.T) {
	data := []byte("Hello, Zero Copy Buffer!")
	zcb := NewZeroCopyBuffer(data)
	
	// 测试字节访问
	bytes := zcb.Bytes()
	if string(bytes) != "Hello, Zero Copy Buffer!" {
		t.Errorf("Expected 'Hello, Zero Copy Buffer!', got %s", string(bytes))
	}
	
	// 测试字符串转换
	str := zcb.String()
	if str != "Hello, Zero Copy Buffer!" {
		t.Errorf("Expected 'Hello, Zero Copy Buffer!', got %s", str)
	}
	
	// 测试切片
	slice := zcb.Slice(0, 5)
	if string(slice) != "Hello" {
		t.Errorf("Expected 'Hello', got %s", string(slice))
	}
	
	t.Log("Zero copy buffer operations test passed")
}

func TestZeroCopyStringConversions(t *testing.T) {
	// 测试字符串到字节的零拷贝转换
	str := "Test string for zero copy conversion"
	bytes := stringToBytesZeroCopy(str)
	
	if len(bytes) != len(str) {
		t.Errorf("Expected length %d, got %d", len(str), len(bytes))
	}
	
	if string(bytes) != str {
		t.Errorf("Expected %s, got %s", str, string(bytes))
	}
	
	// 测试字节到字符串的零拷贝转换
	testBytes := []byte("Test bytes for zero copy conversion")
	convertedStr := bytesToStringZeroCopy(testBytes)
	
	if len(convertedStr) != len(testBytes) {
		t.Errorf("Expected length %d, got %d", len(testBytes), len(convertedStr))
	}
	
	if convertedStr != string(testBytes) {
		t.Errorf("Expected %s, got %s", string(testBytes), convertedStr)
	}
	
	t.Log("Zero copy string conversions test passed")
}

func TestConcurrentMetrics(t *testing.T) {
	metrics := NewPrometheusMetrics()
	
	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100
	
	// 并发增加计数器
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				metrics.IncrementMessagesProduced()
				metrics.RecordLatency(time.Millisecond * 10)
			}
		}()
	}
	
	wg.Wait()
	
	expectedCount := float64(numGoroutines * operationsPerGoroutine)
	actualCount := testutil.ToFloat64(metrics.messagesProduced)
	
	if actualCount != expectedCount {
		t.Errorf("Expected count %f, got %f", expectedCount, actualCount)
	}
	
	t.Log("Concurrent metrics test passed")
}

func TestConcurrentMemoryPool(t *testing.T) {
	pool := NewMemoryPool(1024)
	
	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				buf := pool.Get()
				// 模拟使用
				copy(buf[:10], []byte("test data"))
				pool.Put(buf)
			}
		}()
	}
	
	wg.Wait()
	
	t.Log("Concurrent memory pool test passed")
}

// 基准测试
func BenchmarkPrometheusMetrics_IncrementCounter(b *testing.B) {
	metrics := NewPrometheusMetrics()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.IncrementMessagesProduced()
	}
}

func BenchmarkMemoryPool_Get(b *testing.B) {
	pool := NewMemoryPool(1024)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get()
		pool.Put(buf)
	}
}

func BenchmarkZeroCopyBuffer_Creation(b *testing.B) {
	data := []byte("Test data for benchmark")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewZeroCopyBuffer(data)
	}
}

func BenchmarkStringToBytes_ZeroCopy(b *testing.B) {
	str := "Test string for zero copy conversion benchmark"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stringToBytesZeroCopy(str)
	}
}

func BenchmarkBytesToString_ZeroCopy(b *testing.B) {
	bytes := []byte("Test bytes for zero copy conversion benchmark")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bytesToStringZeroCopy(bytes)
	}
}
