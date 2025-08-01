package producer

import (
	"hash/fnv"
	"math/rand"
	"sync"
	"time"
)

// HashPartitioner 哈希分区器
type HashPartitioner struct {
	mutex sync.RWMutex
}

// NewHashPartitioner 创建哈希分区器
func NewHashPartitioner() *HashPartitioner {
	return &HashPartitioner{}
}

// Partition 计算分区
func (hp *HashPartitioner) Partition(message *ProducerMessage, partitionCount int32) int32 {
	if partitionCount <= 0 {
		return 0
	}
	
	// 使用消息键进行哈希
	hash := fnv.New32a()
	hash.Write([]byte(message.Key))
	partition := int32(hash.Sum32() % uint32(partitionCount))
	// 确保分区值为非负数
	if partition < 0 {
		partition = -partition
	}
	return partition
}

// RoundRobinPartitioner 轮询分区器
type RoundRobinPartitioner struct {
	counter int32
	mutex   sync.Mutex
}

// NewRoundRobinPartitioner 创建轮询分区器
func NewRoundRobinPartitioner() *RoundRobinPartitioner {
	return &RoundRobinPartitioner{}
}

// Partition 计算分区
func (rp *RoundRobinPartitioner) Partition(message *ProducerMessage, partitionCount int32) int32 {
	if partitionCount <= 0 {
		return 0
	}
	
	rp.mutex.Lock()
	defer rp.mutex.Unlock()
	
	partition := rp.counter % partitionCount
	rp.counter++
	return partition
}

// RandomPartitioner 随机分区器
type RandomPartitioner struct {
	random *rand.Rand
	mutex  sync.Mutex
}

// NewRandomPartitioner 创建随机分区器
func NewRandomPartitioner() *RandomPartitioner {
	return &RandomPartitioner{
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Partition 计算分区
func (rp *RandomPartitioner) Partition(message *ProducerMessage, partitionCount int32) int32 {
	if partitionCount <= 0 {
		return 0
	}
	
	rp.mutex.Lock()
	defer rp.mutex.Unlock()
	
	return rp.random.Int31n(partitionCount)
}

// StickyPartitioner 粘性分区器（Kafka 2.4+优化）
type StickyPartitioner struct {
	currentPartition int32
	messageCount     int32
	batchSize        int32
	partitionCount   int32
	random           *rand.Rand
	mutex            sync.Mutex
}

// NewStickyPartitioner 创建粘性分区器
func NewStickyPartitioner(batchSize int32) *StickyPartitioner {
	return &StickyPartitioner{
		currentPartition: -1,
		messageCount:     0,
		batchSize:        batchSize,
		random:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Partition 计算分区
func (sp *StickyPartitioner) Partition(message *ProducerMessage, partitionCount int32) int32 {
	if partitionCount <= 0 {
		return 0
	}
	
	sp.mutex.Lock()
	defer sp.mutex.Unlock()
	
	// 如果分区数量变化，重置状态
	if sp.partitionCount != partitionCount {
		sp.partitionCount = partitionCount
		sp.currentPartition = -1
		sp.messageCount = 0
	}
	
	// 如果是第一次或者达到批次大小，选择新分区
	if sp.currentPartition == -1 || sp.messageCount >= sp.batchSize {
		sp.currentPartition = sp.random.Int31n(partitionCount)
		sp.messageCount = 0
	}
	
	sp.messageCount++
	return sp.currentPartition
}
