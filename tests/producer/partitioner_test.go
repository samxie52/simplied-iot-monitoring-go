package producer

import (
	"testing"

	"simplied-iot-monitoring-go/internal/producer"
)

func TestHashPartitioner_Creation(t *testing.T) {
	partitioner := producer.NewHashPartitioner()

	if partitioner == nil {
		t.Fatal("Failed to create hash partitioner")
	}
}

func TestHashPartitioner_Partition(t *testing.T) {
	partitioner := producer.NewHashPartitioner()
	numPartitions := int32(4)

	// 测试相同的key应该总是分配到相同的分区
	message := &producer.ProducerMessage{
		Key:   "test-key",
		Value: map[string]interface{}{"data": "test"},
		Topic: "test-topic",
	}
	
	partition1 := partitioner.Partition(message, numPartitions)
	partition2 := partitioner.Partition(message, numPartitions)

	if partition1 != partition2 {
		t.Errorf("Same key should map to same partition, got %d and %d", partition1, partition2)
	}

	// 测试分区范围
	if partition1 < 0 || partition1 >= numPartitions {
		t.Errorf("Partition should be in range [0, %d), got %d", numPartitions, partition1)
	}

	// 测试不同的key应该可能分配到不同的分区
	differentKeys := []string{"key1", "key2", "key3", "key4", "key5"}
	partitions := make(map[int32]bool)

	for _, key := range differentKeys {
		msg := &producer.ProducerMessage{
			Key:   key,
			Value: map[string]interface{}{"data": "test"},
			Topic: "test-topic",
		}
		partition := partitioner.Partition(msg, numPartitions)
		partitions[partition] = true
		
		if partition < 0 || partition >= numPartitions {
			t.Errorf("Partition should be in range [0, %d), got %d for key %s", numPartitions, partition, key)
		}
	}

	// 应该至少使用了一些不同的分区
	if len(partitions) < 2 {
		t.Errorf("Expected multiple partitions to be used, only got %d", len(partitions))
	}

	t.Logf("Hash partitioner used %d different partitions for %d keys", len(partitions), len(differentKeys))
}

func TestRoundRobinPartitioner_Creation(t *testing.T) {
	partitioner := producer.NewRoundRobinPartitioner()

	if partitioner == nil {
		t.Fatal("Failed to create round robin partitioner")
	}
}

func TestRoundRobinPartitioner_Partition(t *testing.T) {
	partitioner := producer.NewRoundRobinPartitioner()
	numPartitions := int32(3)

	// 测试轮询分配
	expectedSequence := []int32{0, 1, 2, 0, 1, 2, 0, 1, 2}
	
	for i, expected := range expectedSequence {
		message := &producer.ProducerMessage{
			Key:   "any-key",
			Value: map[string]interface{}{"data": "test"},
			Topic: "test-topic",
		}
		partition := partitioner.Partition(message, numPartitions)
		if partition != expected {
			t.Errorf("Round robin sequence broken at position %d: expected %d, got %d", i, expected, partition)
		}
	}
}

func TestRoundRobinPartitioner_ConcurrentAccess(t *testing.T) {
	partitioner := producer.NewRoundRobinPartitioner()
	numPartitions := int32(4)

	// 并发访问测试
	done := make(chan []int32, 10)

	for i := 0; i < 10; i++ {
		go func() {
			results := make([]int32, 100)
			for j := 0; j < 100; j++ {
				message := &producer.ProducerMessage{
					Key:   "concurrent-key",
					Value: map[string]interface{}{"data": "test"},
					Topic: "test-topic",
				}
				results[j] = partitioner.Partition(message, numPartitions)
			}
			done <- results
		}()
	}

	// 收集所有结果
	allResults := make([]int32, 0, 1000)
	for i := 0; i < 10; i++ {
		results := <-done
		allResults = append(allResults, results...)
	}

	// 验证所有分区都在有效范围内
	partitionCounts := make(map[int32]int)
	for _, partition := range allResults {
		if partition < 0 || partition >= numPartitions {
			t.Errorf("Invalid partition %d, should be in range [0, %d)", partition, numPartitions)
		}
		partitionCounts[partition]++
	}

	// 在并发情况下，每个分区应该都被使用
	for i := int32(0); i < numPartitions; i++ {
		if partitionCounts[i] == 0 {
			t.Errorf("Partition %d was never used", i)
		}
	}

	t.Logf("Concurrent round robin partition distribution: %v", partitionCounts)
}

func TestRandomPartitioner_Creation(t *testing.T) {
	partitioner := producer.NewRandomPartitioner()

	if partitioner == nil {
		t.Fatal("Failed to create random partitioner")
	}
}

func TestRandomPartitioner_Partition(t *testing.T) {
	partitioner := producer.NewRandomPartitioner()
	numPartitions := int32(4)

	// 测试随机分配
	partitions := make(map[int32]int)
	numTests := 1000

	for i := 0; i < numTests; i++ {
		message := &producer.ProducerMessage{
			Key:   "random-key",
			Value: map[string]interface{}{"data": "test"},
			Topic: "test-topic",
		}
		partition := partitioner.Partition(message, numPartitions)
		
		if partition < 0 || partition >= numPartitions {
			t.Errorf("Partition should be in range [0, %d), got %d", numPartitions, partition)
		}
		
		partitions[partition]++
	}

	// 所有分区都应该被使用
	for i := int32(0); i < numPartitions; i++ {
		if partitions[i] == 0 {
			t.Errorf("Partition %d was never used in %d random selections", i, numTests)
		}
	}

	// 分布应该相对均匀（允许一些偏差）
	expectedCount := numTests / int(numPartitions)
	tolerance := expectedCount / 2 // 50% 容忍度

	for i := int32(0); i < numPartitions; i++ {
		count := partitions[i]
		if count < expectedCount-tolerance || count > expectedCount+tolerance {
			t.Logf("Warning: Partition %d has %d assignments, expected around %d (±%d)", 
				i, count, expectedCount, tolerance)
		}
	}

	t.Logf("Random partitioner distribution over %d tests: %v", numTests, partitions)
}

func TestStickyPartitioner_Creation(t *testing.T) {
	partitioner := producer.NewStickyPartitioner(100) // batch size of 100

	if partitioner == nil {
		t.Fatal("Failed to create sticky partitioner")
	}
}

func TestStickyPartitioner_Partition(t *testing.T) {
	batchSize := int32(10)
	partitioner := producer.NewStickyPartitioner(batchSize)
	numPartitions := int32(3)

	// 测试粘性分配 - 在批次大小内应该使用相同分区
	message := &producer.ProducerMessage{
		Key:   "sticky-key",
		Value: map[string]interface{}{"data": "test"},
		Topic: "test-topic",
	}
	
	firstPartition := partitioner.Partition(message, numPartitions)
	
	// 在批次大小内应该返回相同的分区
	for i := 1; i < int(batchSize); i++ {
		partition := partitioner.Partition(message, numPartitions)
		if partition != firstPartition {
			t.Errorf("Sticky partitioner should return same partition within batch, got %d, expected %d at position %d", 
				partition, firstPartition, i)
		}
	}

	// 达到批次大小后，可能会选择新分区
	nextPartition := partitioner.Partition(message, numPartitions)
	
	// 验证分区范围
	if firstPartition < 0 || firstPartition >= numPartitions {
		t.Errorf("First partition should be in range [0, %d), got %d", numPartitions, firstPartition)
	}
	if nextPartition < 0 || nextPartition >= numPartitions {
		t.Errorf("Next partition should be in range [0, %d), got %d", numPartitions, nextPartition)
	}

	t.Logf("Sticky partitioner: first batch -> partition %d, next batch -> partition %d", 
		firstPartition, nextPartition)
}

func TestStickyPartitioner_LoadBalancing(t *testing.T) {
	batchSize := int32(5)
	partitioner := producer.NewStickyPartitioner(batchSize)
	numPartitions := int32(4)

	// 测试多个批次的负载均衡
	partitionCounts := make(map[int32]int)
	totalMessages := 100

	for i := 0; i < totalMessages; i++ {
		message := &producer.ProducerMessage{
			Key:   "load-balance-key",
			Value: map[string]interface{}{"data": i},
			Topic: "test-topic",
		}
		partition := partitioner.Partition(message, numPartitions)
		partitionCounts[partition]++
	}

	// 验证负载分布
	t.Logf("Sticky partitioner load distribution over %d messages: %v", totalMessages, partitionCounts)
	
	// 所有分区都应该被使用（对于足够多的消息）
	for i := int32(0); i < numPartitions; i++ {
		if partitionCounts[i] == 0 {
			t.Errorf("Partition %d was never used", i)
		}
	}
}

func TestPartitioner_EdgeCases(t *testing.T) {
	partitioners := []struct {
		name        string
		partitioner interface {
			Partition(*producer.ProducerMessage, int32) int32
		}
	}{
		{"Hash", producer.NewHashPartitioner()},
		{"RoundRobin", producer.NewRoundRobinPartitioner()},
		{"Random", producer.NewRandomPartitioner()},
		{"Sticky", producer.NewStickyPartitioner(10)},
	}

	for _, p := range partitioners {
		name := p.name
		partitioner := p.partitioner
		
		message := &producer.ProducerMessage{
			Key:   "edge-case-key",
			Value: map[string]interface{}{"data": "test"},
			Topic: "test-topic",
		}
		
		// 测试单分区情况
		partition := partitioner.Partition(message, 1)
		if partition != 0 {
			t.Errorf("%s partitioner should return 0 for single partition, got %d", name, partition)
		}

		// 测试空key
		emptyKeyMessage := &producer.ProducerMessage{
			Key:   "",
			Value: map[string]interface{}{"data": "test"},
			Topic: "test-topic",
		}
		partition = partitioner.Partition(emptyKeyMessage, 4)
		if partition < 0 || partition >= 4 {
			t.Errorf("%s partitioner should handle empty key, got partition %d", name, partition)
		}

		// 测试大分区数
		partition = partitioner.Partition(message, 1000)
		if partition < 0 || partition >= 1000 {
			t.Errorf("%s partitioner should handle large partition count, got partition %d", name, partition)
		}
	}
}

func BenchmarkHashPartitioner(b *testing.B) {
	partitioner := producer.NewHashPartitioner()
	numPartitions := int32(32)
	message := &producer.ProducerMessage{
		Key:   "benchmark-key",
		Value: map[string]interface{}{"data": "benchmark"},
		Topic: "benchmark-topic",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		partitioner.Partition(message, numPartitions)
	}
}

func BenchmarkRoundRobinPartitioner(b *testing.B) {
	partitioner := producer.NewRoundRobinPartitioner()
	numPartitions := int32(32)
	message := &producer.ProducerMessage{
		Key:   "benchmark-key",
		Value: map[string]interface{}{"data": "benchmark"},
		Topic: "benchmark-topic",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		partitioner.Partition(message, numPartitions)
	}
}

func BenchmarkRandomPartitioner(b *testing.B) {
	partitioner := producer.NewRandomPartitioner()
	numPartitions := int32(32)
	message := &producer.ProducerMessage{
		Key:   "benchmark-key",
		Value: map[string]interface{}{"data": "benchmark"},
		Topic: "benchmark-topic",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		partitioner.Partition(message, numPartitions)
	}
}

func BenchmarkStickyPartitioner(b *testing.B) {
	partitioner := producer.NewStickyPartitioner(100)
	numPartitions := int32(32)
	message := &producer.ProducerMessage{
		Key:   "benchmark-key",
		Value: map[string]interface{}{"data": "benchmark"},
		Topic: "benchmark-topic",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		partitioner.Partition(message, numPartitions)
	}
}

func BenchmarkPartitioners_Concurrent(b *testing.B) {
	partitioners := []struct {
		name        string
		partitioner interface {
			Partition(*producer.ProducerMessage, int32) int32
		}
	}{
		{"Hash", producer.NewHashPartitioner()},
		{"RoundRobin", producer.NewRoundRobinPartitioner()},
		{"Random", producer.NewRandomPartitioner()},
		{"Sticky", producer.NewStickyPartitioner(100)},
	}

	for _, p := range partitioners {
		b.Run(p.name, func(b *testing.B) {
			numPartitions := int32(32)
			message := &producer.ProducerMessage{
				Key:   "concurrent-key",
				Value: map[string]interface{}{"data": "concurrent"},
				Topic: "concurrent-topic",
			}
			
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					p.partitioner.Partition(message, numPartitions)
				}
			})
		})
	}
}
