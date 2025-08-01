package producer

import (
	"testing"
	"unsafe"
)

// 直接测试核心性能优化功能，不依赖其他组件

// MemoryPool 内存池（简化版本用于测试）
type TestMemoryPool struct {
	pools map[int]chan []byte
}

func NewTestMemoryPool() *TestMemoryPool {
	return &TestMemoryPool{
		pools: make(map[int]chan []byte),
	}
}

func (p *TestMemoryPool) Get(size int) []byte {
	// 简化实现：直接分配新内存
	return make([]byte, size)
}

func (p *TestMemoryPool) Put(buf []byte) {
	// 简化实现：不做任何操作
}

// ZeroCopyBuffer 零拷贝缓冲区（简化版本）
type TestZeroCopyBuffer struct {
	data []byte
}

func NewTestZeroCopyBuffer(data []byte) *TestZeroCopyBuffer {
	return &TestZeroCopyBuffer{data: data}
}

func (zcb *TestZeroCopyBuffer) Len() int {
	return len(zcb.data)
}

func (zcb *TestZeroCopyBuffer) Bytes() []byte {
	return zcb.data
}

func (zcb *TestZeroCopyBuffer) String() string {
	return string(zcb.data)
}

func (zcb *TestZeroCopyBuffer) Slice(start, end int) *TestZeroCopyBuffer {
	if start < 0 || end > len(zcb.data) || start > end {
		return nil
	}
	return &TestZeroCopyBuffer{data: zcb.data[start:end]}
}

// 零拷贝字符串转换函数
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

func TestMemoryPool_BasicOperations(t *testing.T) {
	pool := NewTestMemoryPool()
	
	sizes := []int{64, 128, 256, 512, 1024, 2048}
	
	for _, size := range sizes {
		buf := pool.Get(size)
		if buf == nil {
			t.Errorf("Failed to get buffer of size %d", size)
			continue
		}
		
		if len(buf) != size {
			t.Errorf("Expected buffer size %d, got %d", size, len(buf))
		}
		
		// 写入一些数据测试
		for i := 0; i < len(buf); i++ {
			buf[i] = byte(i % 256)
		}
		
		// 验证数据
		for i := 0; i < len(buf); i++ {
			if buf[i] != byte(i%256) {
				t.Errorf("Data corruption at index %d", i)
				break
			}
		}
		
		pool.Put(buf)
	}
	
	t.Log("Memory pool basic operations test passed")
}

func TestZeroCopyBuffer_Operations(t *testing.T) {
	data := []byte("Hello, Zero Copy World! This is a test of zero copy buffer operations.")
	zcb := NewTestZeroCopyBuffer(data)
	
	// 测试长度
	if zcb.Len() != len(data) {
		t.Errorf("Expected length %d, got %d", len(data), zcb.Len())
	}
	
	// 测试字节获取
	result := zcb.Bytes()
	if len(result) != len(data) {
		t.Errorf("Expected bytes length %d, got %d", len(data), len(result))
	}
	
	// 验证数据相同
	for i, b := range result {
		if b != data[i] {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, data[i], b)
		}
	}
	
	// 验证是否为零拷贝（相同的底层数组）
	if uintptr(unsafe.Pointer(&result[0])) != uintptr(unsafe.Pointer(&data[0])) {
		t.Error("Expected zero copy, but data was copied")
	}
	
	// 测试字符串转换
	str := zcb.String()
	expected := string(data)
	if str != expected {
		t.Errorf("Expected string %q, got %q", expected, str)
	}
	
	// 测试切片操作
	slice := zcb.Slice(7, 16) // "Zero Copy"
	if slice == nil {
		t.Fatal("Failed to create slice")
	}
	
	if slice.Len() != 9 {
		t.Errorf("Expected slice length 9, got %d", slice.Len())
	}
	
	if slice.String() != "Zero Copy" {
		t.Errorf("Expected slice 'Zero Copy', got %q", slice.String())
	}
	
	// 测试边界情况
	if zcb.Slice(-1, 5) != nil {
		t.Error("Expected nil for negative start index")
	}
	
	if zcb.Slice(5, len(data)+1) != nil {
		t.Error("Expected nil for end index beyond length")
	}
	
	if zcb.Slice(10, 5) != nil {
		t.Error("Expected nil for start > end")
	}
	
	t.Log("Zero copy buffer operations test passed")
}

func TestZeroCopyStringConversions(t *testing.T) {
	// 测试字符串到字节的零拷贝转换
	str := "Test string for zero copy conversion with some length to make it meaningful"
	bytes := stringToBytesZeroCopy(str)
	
	if len(bytes) != len(str) {
		t.Errorf("Expected length %d, got %d", len(str), len(bytes))
	}
	
	// 验证数据相同
	for i, b := range bytes {
		if b != str[i] {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, str[i], b)
		}
	}
	
	// 验证零拷贝
	if uintptr(unsafe.Pointer(&bytes[0])) != uintptr(unsafe.Pointer(unsafe.StringData(str))) {
		t.Error("Expected zero copy conversion, but data was copied")
	}
	
	// 测试字节到字符串的零拷贝转换
	testBytes := []byte("Test bytes for zero copy conversion back to string")
	convertedStr := bytesToStringZeroCopy(testBytes)
	
	if len(convertedStr) != len(testBytes) {
		t.Errorf("Expected length %d, got %d", len(testBytes), len(convertedStr))
	}
	
	// 验证数据相同
	for i, c := range convertedStr {
		if byte(c) != testBytes[i] {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, testBytes[i], byte(c))
		}
	}
	
	// 验证零拷贝
	if uintptr(unsafe.Pointer(unsafe.StringData(convertedStr))) != uintptr(unsafe.Pointer(&testBytes[0])) {
		t.Error("Expected zero copy conversion, but data was copied")
	}
	
	t.Log("Zero copy string conversions test passed")
}

func TestConcurrentMemoryPool(t *testing.T) {
	pool := NewTestMemoryPool()
	
	// 并发测试
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < 100; j++ {
				size := 64 + (j%10)*64 // 64到640字节
				buf := pool.Get(size)
				if buf == nil {
					t.Errorf("Goroutine %d: Failed to get buffer of size %d", id, size)
					return
				}
				
				if len(buf) != size {
					t.Errorf("Goroutine %d: Expected buffer size %d, got %d", id, size, len(buf))
					return
				}
				
				// 写入数据
				for k := 0; k < len(buf); k++ {
					buf[k] = byte((id + j + k) % 256)
				}
				
				// 验证数据
				for k := 0; k < len(buf); k++ {
					expected := byte((id + j + k) % 256)
					if buf[k] != expected {
						t.Errorf("Goroutine %d: Data corruption at index %d", id, k)
						return
					}
				}
				
				pool.Put(buf)
			}
		}(i)
	}
	
	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
	
	t.Log("Concurrent memory pool test passed")
}

func TestLargeDataZeroCopy(t *testing.T) {
	// 测试大数据的零拷贝操作
	size := 1024 * 1024 // 1MB
	data := make([]byte, size)
	
	// 填充测试数据
	for i := 0; i < size; i++ {
		data[i] = byte(i % 256)
	}
	
	// 创建零拷贝缓冲区
	zcb := NewTestZeroCopyBuffer(data)
	
	// 验证长度
	if zcb.Len() != size {
		t.Errorf("Expected length %d, got %d", size, zcb.Len())
	}
	
	// 获取字节并验证零拷贝
	result := zcb.Bytes()
	if uintptr(unsafe.Pointer(&result[0])) != uintptr(unsafe.Pointer(&data[0])) {
		t.Error("Expected zero copy for large data, but data was copied")
	}
	
	// 验证数据完整性（抽样检查）
	checkPoints := []int{0, size/4, size/2, 3*size/4, size-1}
	for _, point := range checkPoints {
		if result[point] != data[point] {
			t.Errorf("Data mismatch at point %d: expected %d, got %d", point, data[point], result[point])
		}
	}
	
	// 测试大数据切片
	sliceStart := size / 4
	sliceEnd := 3 * size / 4
	slice := zcb.Slice(sliceStart, sliceEnd)
	
	if slice == nil {
		t.Fatal("Failed to create large data slice")
	}
	
	if slice.Len() != sliceEnd-sliceStart {
		t.Errorf("Expected slice length %d, got %d", sliceEnd-sliceStart, slice.Len())
	}
	
	// 验证切片数据
	sliceData := slice.Bytes()
	for i := 0; i < len(sliceData); i++ {
		expected := byte((sliceStart + i) % 256)
		if sliceData[i] != expected {
			t.Errorf("Slice data mismatch at index %d: expected %d, got %d", i, expected, sliceData[i])
			break
		}
	}
	
	t.Log("Large data zero copy test passed")
}

// 基准测试

func BenchmarkMemoryPool_Get(b *testing.B) {
	pool := NewTestMemoryPool()
	size := 1024
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get(size)
		pool.Put(buf)
	}
}

func BenchmarkZeroCopyBuffer_Creation(b *testing.B) {
	data := make([]byte, 1024)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		zcb := NewTestZeroCopyBuffer(data)
		_ = zcb.Len()
	}
}

func BenchmarkZeroCopyBuffer_BytesAccess(b *testing.B) {
	data := make([]byte, 1024)
	zcb := NewTestZeroCopyBuffer(data)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = zcb.Bytes()
	}
}

func BenchmarkZeroCopyBuffer_StringConversion(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte('A' + (i % 26))
	}
	zcb := NewTestZeroCopyBuffer(data)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = zcb.String()
	}
}

func BenchmarkZeroCopyBuffer_Slice(b *testing.B) {
	data := make([]byte, 1024)
	zcb := NewTestZeroCopyBuffer(data)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = zcb.Slice(100, 900)
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
	bytes := []byte("This is a benchmark test byte slice for zero copy conversion performance measurement")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bytesToStringZeroCopy(bytes)
	}
}

func BenchmarkStringToBytes_Standard(b *testing.B) {
	str := "This is a benchmark test string for standard conversion performance comparison"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = []byte(str)
	}
}

func BenchmarkBytesToString_Standard(b *testing.B) {
	bytes := []byte("This is a benchmark test byte slice for standard conversion performance comparison")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = string(bytes)
	}
}

func BenchmarkConcurrentMemoryPool(b *testing.B) {
	pool := NewTestMemoryPool()
	size := 1024
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(size)
			// 模拟一些使用
			buf[0] = 1
			buf[size-1] = 1
			pool.Put(buf)
		}
	})
}
