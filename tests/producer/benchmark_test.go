package main

import (
	"testing"
	"unsafe"
)

// 核心组件实现（用于基准测试）

type PrometheusMetrics struct {
	counters   map[string]float64
	histograms map[string][]float64
	gauges     map[string]float64
}

func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		counters:   make(map[string]float64),
		histograms: make(map[string][]float64),
		gauges:     make(map[string]float64),
	}
}

func (pm *PrometheusMetrics) IncrementCounter(name string, labels map[string]string) {
	pm.counters[name]++
}

func (pm *PrometheusMetrics) RecordHistogram(name string, value float64, labels map[string]string) {
	pm.histograms[name] = append(pm.histograms[name], value)
}

func (pm *PrometheusMetrics) SetGauge(name string, value float64, labels map[string]string) {
	pm.gauges[name] = value
}

type MemoryPool struct {
	buffers chan []byte
	size    int
}

func NewMemoryPool(bufferSize, poolSize int) *MemoryPool {
	pool := &MemoryPool{
		buffers: make(chan []byte, poolSize),
		size:    bufferSize,
	}
	
	for i := 0; i < poolSize; i++ {
		pool.buffers <- make([]byte, bufferSize)
	}
	
	return pool
}

func (mp *MemoryPool) Get() []byte {
	select {
	case buf := <-mp.buffers:
		return buf
	default:
		return make([]byte, mp.size)
	}
}

func (mp *MemoryPool) Put(buf []byte) {
	if len(buf) != mp.size {
		return
	}
	
	select {
	case mp.buffers <- buf:
	default:
	}
}

func (mp *MemoryPool) Close() {
	close(mp.buffers)
}

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
	return zcb.data[start:end]
}

func StringToBytesZeroCopy(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

func BytesToStringZeroCopy(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// 基准测试

func BenchmarkPrometheusMetrics_IncrementCounter(b *testing.B) {
	metrics := NewPrometheusMetrics()
	labels := map[string]string{"label": "value"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.IncrementCounter("test_counter", labels)
	}
}

func BenchmarkPrometheusMetrics_RecordHistogram(b *testing.B) {
	metrics := NewPrometheusMetrics()
	labels := map[string]string{"label": "value"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordHistogram("test_histogram", 1.5, labels)
	}
}

func BenchmarkPrometheusMetrics_SetGauge(b *testing.B) {
	metrics := NewPrometheusMetrics()
	labels := map[string]string{"label": "value"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.SetGauge("test_gauge", 42.0, labels)
	}
}

func BenchmarkMemoryPool_Get(b *testing.B) {
	pool := NewMemoryPool(1024, 100)
	defer pool.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get()
		pool.Put(buf)
	}
}

func BenchmarkMemoryPool_GetPut(b *testing.B) {
	pool := NewMemoryPool(1024, 100)
	defer pool.Close()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get()
			pool.Put(buf)
		}
	})
}

func BenchmarkZeroCopyBuffer_Creation(b *testing.B) {
	data := make([]byte, 1024)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewZeroCopyBuffer(data)
	}
}

func BenchmarkZeroCopyBuffer_BytesAccess(b *testing.B) {
	data := make([]byte, 1024)
	buf := NewZeroCopyBuffer(data)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.Bytes()
	}
}

func BenchmarkZeroCopyBuffer_StringConversion(b *testing.B) {
	data := make([]byte, 1024)
	buf := NewZeroCopyBuffer(data)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.String()
	}
}

func BenchmarkZeroCopyBuffer_Slice(b *testing.B) {
	data := make([]byte, 1024)
	buf := NewZeroCopyBuffer(data)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.Slice(0, 100)
	}
}

func BenchmarkStringToBytes_ZeroCopy(b *testing.B) {
	s := "This is a test string for zero copy conversion benchmark"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StringToBytesZeroCopy(s)
	}
}

func BenchmarkBytesToString_ZeroCopy(b *testing.B) {
	data := []byte("This is a test string for zero copy conversion benchmark")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BytesToStringZeroCopy(data)
	}
}

func BenchmarkStringToBytes_Standard(b *testing.B) {
	s := "This is a test string for standard conversion benchmark"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = []byte(s)
	}
}

func BenchmarkBytesToString_Standard(b *testing.B) {
	data := []byte("This is a test string for standard conversion benchmark")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = string(data)
	}
}

func BenchmarkConcurrentMemoryPool(b *testing.B) {
	pool := NewMemoryPool(1024, 100)
	defer pool.Close()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get()
			// 模拟一些工作
			for i := 0; i < 10; i++ {
				buf[i] = byte(i)
			}
			pool.Put(buf)
		}
	})
}

func BenchmarkIntegratedWorkflow(b *testing.B) {
	metrics := NewPrometheusMetrics()
	pool := NewMemoryPool(1024, 50)
	defer pool.Close()
	
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i % 256)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 从内存池获取缓冲区
		buf := pool.Get()
		
		// 创建零拷贝缓冲区
		zcBuf := NewZeroCopyBuffer(buf[:100])
		
		// 执行零拷贝操作
		_ = zcBuf.String()
		_ = zcBuf.Bytes()
		
		// 记录指标
		metrics.IncrementCounter("operations", nil)
		
		// 归还缓冲区
		pool.Put(buf)
	}
}
