package main

import (
	"log"
	"time"

	"github.com/IBM/sarama"
)

func main() {
	log.Println("开始简单Kafka连接测试...")

	brokers := []string{"192.168.5.16:9092"}
	
	// 尝试不同的Kafka版本
	versions := []sarama.KafkaVersion{
		sarama.V1_0_0_0,
		sarama.V2_0_0_0,
		sarama.V2_6_0_0,
		sarama.V3_0_0_0,
	}

	for _, version := range versions {
		log.Printf("测试Kafka版本: %v", version)
		
		config := sarama.NewConfig()
		config.Version = version
		config.ClientID = "simple-debug-client"
		
		// 设置更短的超时时间
		config.Net.DialTimeout = 5 * time.Second
		config.Net.ReadTimeout = 5 * time.Second
		config.Net.WriteTimeout = 5 * time.Second
		
		// 简化配置
		config.Producer.Return.Successes = true
		config.Producer.RequiredAcks = sarama.RequiredAcks(1)
		config.Producer.Retry.Max = 1
		
		done := make(chan error, 1)
		go func() {
			client, err := sarama.NewClient(brokers, config)
			if err != nil {
				done <- err
				return
			}
			defer client.Close()
			
			log.Printf("版本 %v 连接成功!", version)
			done <- nil
		}()
		
		select {
		case err := <-done:
			if err != nil {
				log.Printf("版本 %v 连接失败: %v", version, err)
			} else {
				log.Printf("版本 %v 连接成功，尝试创建生产者...", version)
				
				// 尝试创建异步生产者
				prodDone := make(chan error, 1)
				go func() {
					producer, err := sarama.NewAsyncProducer(brokers, config)
					if err != nil {
						prodDone <- err
						return
					}
					defer producer.Close()
					prodDone <- nil
				}()
				
				select {
				case err := <-prodDone:
					if err != nil {
						log.Printf("版本 %v 异步生产者创建失败: %v", version, err)
					} else {
						log.Printf("版本 %v 异步生产者创建成功!", version)
						return // 找到可用版本，退出
					}
				case <-time.After(10 * time.Second):
					log.Printf("版本 %v 异步生产者创建超时", version)
				}
			}
		case <-time.After(10 * time.Second):
			log.Printf("版本 %v 连接超时", version)
		}
		
		time.Sleep(1 * time.Second) // 短暂等待
	}

	log.Println("简单Kafka连接测试完成")
}
