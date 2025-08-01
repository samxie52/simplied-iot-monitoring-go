package main

import (
	"log"
	"time"

	"github.com/IBM/sarama"
)

func main() {
	log.Println("开始测试Kafka连接...")

	brokers := []string{"192.168.5.16:9092"}
	
	// 创建Sarama配置
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.ClientID = "debug-client"
	
	// 设置网络超时
	config.Net.DialTimeout = 10 * time.Second
	config.Net.ReadTimeout = 10 * time.Second
	config.Net.WriteTimeout = 10 * time.Second
	
	// 生产者配置
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	config.Producer.Retry.Backoff = 100 * time.Millisecond
	config.Producer.Flush.Frequency = 500 * time.Millisecond
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.MaxMessageBytes = 1000000

	log.Printf("Kafka配置: Brokers=%v, Version=%v", brokers, config.Version)

	// 1. 测试基本连接
	log.Println("1. 测试基本连接...")
	done := make(chan error, 1)
	go func() {
		client, err := sarama.NewClient(brokers, config)
		if err != nil {
			done <- err
			return
		}
		defer client.Close()
		
		// 获取broker信息
		brokerList := client.Brokers()
		log.Printf("连接成功，发现%d个broker", len(brokerList))
		for _, broker := range brokerList {
			log.Printf("Broker: %s", broker.Addr())
		}
		done <- nil
	}()
	
	select {
	case err := <-done:
		if err != nil {
			log.Printf("基本连接失败: %v", err)
			return
		}
		log.Println("基本连接成功!")
	case <-time.After(15 * time.Second):
		log.Println("基本连接超时!")
		return
	}

	// 2. 测试同步生产者
	log.Println("2. 测试同步生产者...")
	done = make(chan error, 1)
	go func() {
		producer, err := sarama.NewSyncProducer(brokers, config)
		if err != nil {
			done <- err
			return
		}
		defer producer.Close()
		
		log.Println("同步生产者创建成功")
		done <- nil
	}()
	
	select {
	case err := <-done:
		if err != nil {
			log.Printf("同步生产者创建失败: %v", err)
		} else {
			log.Println("同步生产者创建成功!")
		}
	case <-time.After(15 * time.Second):
		log.Println("同步生产者创建超时!")
	}

	// 3. 测试异步生产者
	log.Println("3. 测试异步生产者...")
	done = make(chan error, 1)
	go func() {
		producer, err := sarama.NewAsyncProducer(brokers, config)
		if err != nil {
			done <- err
			return
		}
		defer producer.Close()
		
		log.Println("异步生产者创建成功")
		done <- nil
	}()
	
	select {
	case err := <-done:
		if err != nil {
			log.Printf("异步生产者创建失败: %v", err)
		} else {
			log.Println("异步生产者创建成功!")
		}
	case <-time.After(15 * time.Second):
		log.Println("异步生产者创建超时!")
	}

	log.Println("Kafka连接测试完成")
}
