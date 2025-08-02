# 多阶段构建 Dockerfile for Industrial IoT Monitoring System
# 支持构建多个服务：producer, consumer, web, websocket

# 构建参数
ARG SERVICE=producer
ARG VERSION=latest
ARG BUILD_TIME
ARG GIT_COMMIT

# 第一阶段：构建阶段
FROM golang:1.24.4-alpine AS builder

# 构建参数传递
ARG SERVICE
ARG VERSION
ARG BUILD_TIME
ARG GIT_COMMIT

# 设置工作目录
WORKDIR /app

# 安装必要的系统依赖
RUN apk add --no-cache git ca-certificates tzdata make

# 设置Go代理和环境变量
ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.google.cn
ENV GO111MODULE=on
ENV CGO_ENABLED=0

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖（带重试机制）
RUN go mod download || \
    (echo "第一次下载失败，重试..." && sleep 5 && go mod download) || \
    (echo "第二次下载失败，使用直连模式..." && GOPROXY=direct go mod download)

# 复制源代码
COPY . .

# 构建指定服务
RUN GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}" \
    -a -installsuffix cgo \
    -o iot-${SERVICE} \
    ./cmd/${SERVICE}

# 第二阶段：运行阶段
FROM alpine:3.18 AS runtime

# 构建参数传递
ARG SERVICE

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata curl

# 创建非root用户
RUN addgroup -g 1001 -S iot && \
    adduser -u 1001 -S iot -G iot

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/iot-${SERVICE} /app/iot-${SERVICE}

# 复制配置文件和静态资源
COPY --from=builder /app/configs /app/configs
COPY --from=builder /app/web /app/web

# 设置权限和创建启动脚本
RUN chmod +x /app/iot-${SERVICE} && \
    chown -R iot:iot /app && \
    echo '#!/bin/sh' > /app/start.sh && \
    echo 'SERVICE_NAME=${SERVICE}' >> /app/start.sh && \
    echo 'if [ "$SERVICE_NAME" = "producer" ] || [ "$SERVICE_NAME" = "consumer" ]; then' >> /app/start.sh && \
    echo '  exec /app/iot-'${SERVICE}' start --config=/app/configs/development.yaml "$@"' >> /app/start.sh && \
    echo 'elif [ "$SERVICE_NAME" = "web" ]; then' >> /app/start.sh && \
    echo '  exec /app/iot-'${SERVICE}' --port=8082 "$@"' >> /app/start.sh && \
    echo 'elif [ "$SERVICE_NAME" = "websocket" ]; then' >> /app/start.sh && \
    echo '  exec /app/iot-'${SERVICE}' --config=/app/configs/development.yaml "$@"' >> /app/start.sh && \
    echo 'else' >> /app/start.sh && \
    echo '  exec /app/iot-'${SERVICE}' "$@"' >> /app/start.sh && \
    echo 'fi' >> /app/start.sh && \
    chmod +x /app/start.sh

# 设置环境变量
ENV TZ=Asia/Shanghai
ENV SERVICE_NAME=${SERVICE}
ENV CONFIG_PATH=/app/configs/development.yaml
ENV LOG_LEVEL=info

# 根据服务类型暴露不同端口
EXPOSE 8080 8081 8082 8083

# 健康检查（根据服务类型调整）
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD /app/iot-${SERVICE} version || exit 1

# 切换到非root用户
USER iot:iot

# 启动应用程序
ENTRYPOINT ["/app/start.sh"]

# 最终阶段：服务特定配置
FROM runtime AS producer
ENV SERVICE_TYPE=producer
ENV METRICS_PORT=8080
EXPOSE 8080

FROM runtime AS consumer  
ENV SERVICE_TYPE=consumer
ENV METRICS_PORT=8081
EXPOSE 8081

FROM runtime AS web
ENV SERVICE_TYPE=web
ENV WEB_PORT=8082
EXPOSE 8082

FROM runtime AS websocket
ENV SERVICE_TYPE=websocket
ENV WEBSOCKET_PORT=8083
EXPOSE 8083
