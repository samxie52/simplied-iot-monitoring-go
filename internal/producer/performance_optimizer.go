package producer

import (
	"sync"
	"unsafe"
)

// MemoryPool 内存池实现
type MemoryPool struct {
	pools map[int]*sync.Pool
	mutex sync.RWMutex
}

// NewMemoryPool 创建新的内存池
func NewMemoryPool() *MemoryPool {
	mp := &MemoryPool{
		pools: make(map[int]*sync.Pool),
	}
	
	// 预创建常用大小的内存池
	commonSizes := []int{64, 128, 256, 512, 1024, 2048, 4096, 8192}
	for _, size := range commonSizes {
		mp.createPool(size)
	}
	
	return mp
}

// createPool 创建指定大小的内存池
func (mp *MemoryPool) createPool(size int) {
	mp.pools[size] = &sync.Pool{
		New: func() interface{} {
			return make([]byte, size)
		},
	}
}

// Get 从内存池获取指定大小的字节切片
func (mp *MemoryPool) Get(size int) []byte {
	mp.mutex.RLock()
	pool, exists := mp.pools[size]
	mp.mutex.RUnlock()
	
	if !exists {
		// 如果没有对应大小的池，创建一个
		mp.mutex.Lock()
		if _, exists := mp.pools[size]; !exists {
			mp.createPool(size)
		}
		pool = mp.pools[size]
		mp.mutex.Unlock()
	}
	
	buf := pool.Get().([]byte)
	return buf[:size] // 确保返回正确大小的切片
}

// Put 将字节切片放回内存池
func (mp *MemoryPool) Put(buf []byte) {
	if buf == nil {
		return
	}
	
	size := cap(buf)
	mp.mutex.RLock()
	pool, exists := mp.pools[size]
	mp.mutex.RUnlock()
	
	if exists {
		// 重置切片内容（可选，用于安全性）
		for i := range buf {
			buf[i] = 0
		}
		pool.Put(buf)
	}
}

// ZeroCopyBuffer 零拷贝缓冲区
type ZeroCopyBuffer struct {
	data   []byte
	offset int
	length int
}

// NewZeroCopyBuffer 创建零拷贝缓冲区
func NewZeroCopyBuffer(data []byte) *ZeroCopyBuffer {
	return &ZeroCopyBuffer{
		data:   data,
		offset: 0,
		length: len(data),
	}
}

// Bytes 返回缓冲区数据（零拷贝）
func (zcb *ZeroCopyBuffer) Bytes() []byte {
	return zcb.data[zcb.offset : zcb.offset+zcb.length]
}

// String 返回字符串表示（零拷贝）
func (zcb *ZeroCopyBuffer) String() string {
	return *(*string)(unsafe.Pointer(&zcb.data))
}

// Slice 创建子切片（零拷贝）
func (zcb *ZeroCopyBuffer) Slice(start, end int) *ZeroCopyBuffer {
	if start < 0 || end > zcb.length || start > end {
		return nil
	}
	
	return &ZeroCopyBuffer{
		data:   zcb.data,
		offset: zcb.offset + start,
		length: end - start,
	}
}

// Len 返回缓冲区长度
func (zcb *ZeroCopyBuffer) Len() int {
	return zcb.length
}

// MessageBuffer 消息缓冲区，支持零拷贝操作
type MessageBuffer struct {
	buffer []byte
	pool   *MemoryPool
	size   int
}

// NewMessageBuffer 创建新的消息缓冲区
func NewMessageBuffer(size int, pool *MemoryPool) *MessageBuffer {
	return &MessageBuffer{
		buffer: pool.Get(size),
		pool:   pool,
		size:   size,
	}
}

// Write 写入数据
func (mb *MessageBuffer) Write(data []byte) int {
	n := copy(mb.buffer, data)
	return n
}

// Bytes 获取缓冲区数据
func (mb *MessageBuffer) Bytes() []byte {
	return mb.buffer
}

// Release 释放缓冲区回池
func (mb *MessageBuffer) Release() {
	if mb.pool != nil && mb.buffer != nil {
		mb.pool.Put(mb.buffer)
		mb.buffer = nil
	}
}

// BatchBuffer 批量消息缓冲区
type BatchBuffer struct {
	messages []*MessageBuffer
	capacity int
	count    int
	mutex    sync.Mutex
}

// NewBatchBuffer 创建批量缓冲区
func NewBatchBuffer(capacity int) *BatchBuffer {
	return &BatchBuffer{
		messages: make([]*MessageBuffer, capacity),
		capacity: capacity,
		count:    0,
	}
}

// Add 添加消息到批次
func (bb *BatchBuffer) Add(msg *MessageBuffer) bool {
	bb.mutex.Lock()
	defer bb.mutex.Unlock()
	
	if bb.count >= bb.capacity {
		return false
	}
	
	bb.messages[bb.count] = msg
	bb.count++
	return true
}

// GetMessages 获取所有消息
func (bb *BatchBuffer) GetMessages() []*MessageBuffer {
	bb.mutex.Lock()
	defer bb.mutex.Unlock()
	
	result := make([]*MessageBuffer, bb.count)
	copy(result, bb.messages[:bb.count])
	return result
}

// Clear 清空批次
func (bb *BatchBuffer) Clear() {
	bb.mutex.Lock()
	defer bb.mutex.Unlock()
	
	for i := 0; i < bb.count; i++ {
		bb.messages[i] = nil
	}
	bb.count = 0
}

// Count 获取当前消息数量
func (bb *BatchBuffer) Count() int {
	bb.mutex.Lock()
	defer bb.mutex.Unlock()
	return bb.count
}

// IsFull 检查批次是否已满
func (bb *BatchBuffer) IsFull() bool {
	bb.mutex.Lock()
	defer bb.mutex.Unlock()
	return bb.count >= bb.capacity
}

// PerformanceOptimizer 性能优化器
type PerformanceOptimizer struct {
	memoryPool      *MemoryPool
	batchBuffers    []*BatchBuffer
	bufferIndex     int
	bufferMutex     sync.Mutex
	metrics         *PrometheusMetrics
	
	// 配置参数
	batchSize       int
	bufferPoolSize  int
	enableZeroCopy  bool
	enableMemPool   bool
}

// NewPerformanceOptimizer 创建性能优化器
func NewPerformanceOptimizer(batchSize, bufferPoolSize int, metrics *PrometheusMetrics) *PerformanceOptimizer {
	po := &PerformanceOptimizer{
		memoryPool:     NewMemoryPool(),
		batchBuffers:   make([]*BatchBuffer, bufferPoolSize),
		bufferIndex:    0,
		metrics:        metrics,
		batchSize:      batchSize,
		bufferPoolSize: bufferPoolSize,
		enableZeroCopy: true,
		enableMemPool:  true,
	}
	
	// 初始化批次缓冲区
	for i := 0; i < bufferPoolSize; i++ {
		po.batchBuffers[i] = NewBatchBuffer(batchSize)
	}
	
	return po
}

// GetMessageBuffer 获取消息缓冲区
func (po *PerformanceOptimizer) GetMessageBuffer(size int) *MessageBuffer {
	if po.enableMemPool {
		po.metrics.IncrementMemoryPoolHits()
		return NewMessageBuffer(size, po.memoryPool)
	} else {
		po.metrics.IncrementMemoryPoolMisses()
		return &MessageBuffer{
			buffer: make([]byte, size),
			size:   size,
		}
	}
}

// GetBatchBuffer 获取批次缓冲区
func (po *PerformanceOptimizer) GetBatchBuffer() *BatchBuffer {
	po.bufferMutex.Lock()
	defer po.bufferMutex.Unlock()
	
	// 轮询选择缓冲区
	buffer := po.batchBuffers[po.bufferIndex]
	po.bufferIndex = (po.bufferIndex + 1) % po.bufferPoolSize
	
	return buffer
}

// CreateZeroCopyBuffer 创建零拷贝缓冲区
func (po *PerformanceOptimizer) CreateZeroCopyBuffer(data []byte) *ZeroCopyBuffer {
	if po.enableZeroCopy {
		po.metrics.IncrementZeroCopyOperations()
		return NewZeroCopyBuffer(data)
	}
	
	// 如果不启用零拷贝，创建数据副本
	copied := make([]byte, len(data))
	copy(copied, data)
	return NewZeroCopyBuffer(copied)
}

// OptimizeMessage 优化消息处理
func (po *PerformanceOptimizer) OptimizeMessage(data []byte) (*MessageBuffer, error) {
	// 获取合适大小的缓冲区
	size := len(data)
	if size < 64 {
		size = 64 // 最小缓冲区大小
	} else {
		// 向上取整到2的幂
		size = nextPowerOfTwo(size)
	}
	
	msgBuf := po.GetMessageBuffer(size)
	msgBuf.Write(data)
	
	return msgBuf, nil
}

// ReleaseBatchBuffer 释放批次缓冲区
func (po *PerformanceOptimizer) ReleaseBatchBuffer(buffer *BatchBuffer) {
	// 释放所有消息缓冲区
	messages := buffer.GetMessages()
	for _, msg := range messages {
		if msg != nil {
			msg.Release()
		}
	}
	
	// 清空批次缓冲区
	buffer.Clear()
}

// GetStats 获取性能统计
func (po *PerformanceOptimizer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"memory_pool_enabled":  po.enableMemPool,
		"zero_copy_enabled":    po.enableZeroCopy,
		"batch_size":          po.batchSize,
		"buffer_pool_size":    po.bufferPoolSize,
		"active_buffers":      len(po.batchBuffers),
	}
}

// nextPowerOfTwo 计算下一个2的幂
func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 1
	}
	
	// 如果已经是2的幂，直接返回
	if n&(n-1) == 0 {
		return n
	}
	
	// 找到下一个2的幂
	power := 1
	for power < n {
		power <<= 1
	}
	
	return power
}

// StringToBytes 零拷贝字符串转字节切片
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&struct {
		string
		int
	}{s, len(s)}))
}

// BytesToString 零拷贝字节切片转字符串
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
