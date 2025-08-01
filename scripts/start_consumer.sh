#!/bin/bash

echo "🚀 启动Kafka消费者服务..."
echo "配置文件: configs/development.yaml"
echo "按 Ctrl+C 停止服务"
echo "================================"

# 启动消费者
./bin/consumer -c configs/development.yaml -v
