# 多阶段构建 Dockerfile for Industrial IoT Kafka Producer

# 第一阶段：构建阶段
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的系统依赖
RUN apk add --no-cache git ca-certificates tzdata

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用程序
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o iot-producer \
    ./cmd/producer

# 第二阶段：运行阶段
FROM scratch

# 从构建阶段复制必要文件
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/iot-producer /iot-producer

# 复制配置文件
COPY --from=builder /app/configs /configs

# 设置环境变量
ENV TZ=Asia/Shanghai
ENV CONFIG_PATH=/configs/production.yaml
ENV LOG_LEVEL=info
ENV METRICS_PORT=8080

# 暴露端口
EXPOSE 8080 8081

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/iot-producer", "health-check"]

# 设置用户（安全最佳实践）
USER 65534:65534

# 启动应用程序
ENTRYPOINT ["/iot-producer"]
CMD ["start"]
