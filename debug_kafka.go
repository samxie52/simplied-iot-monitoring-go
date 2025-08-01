package main

import (
	"log"
	"time"

	"github.com/IBM/sarama"
)

func main() {
	log.Println("测试Kafka连接...")

	// 创建Sarama配置
	config := sarama.NewConfig()
	config.ClientID = "test-client"
	config.Net.DialTimeout = 10 * time.Second
	config.Net.ReadTimeout = 10 * time.Second
	config.Net.WriteTimeout = 10 * time.Second
	config.Version = sarama.V2_6_0_0

	brokers := []string{"192.168.5.16:9092"}

	log.Println("尝试创建Kafka客户端...")
	
	// 设置超时
	done := make(chan error, 1)
	go func() {
		client, err := sarama.NewClient(brokers, config)
		if err != nil {
			done <- err
			return
		}
		defer client.Close()
		
		log.Println("Kafka客户端创建成功!")
		
		// 获取broker信息
		brokers := client.Brokers()
		log.Printf("连接的Brokers数量: %d", len(brokers))
		
		for _, broker := range brokers {
			log.Printf("Broker: %s", broker.Addr())
		}
		
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Fatalf("创建Kafka客户端失败: %v", err)
		}
		log.Println("Kafka连接测试成功!")
	case <-time.After(30 * time.Second):
		log.Println("Kafka连接测试超时!")
	}
}
