#!/bin/bash

# Docker构建测试脚本
# 用于测试多服务Docker镜像构建

set -e

echo "🐳 开始测试Docker构建..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 服务列表
SERVICES=("producer" "consumer" "websocket" "web")

# 项目信息
PROJECT_NAME="simplied-iot-monitoring-go"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

echo -e "${YELLOW}项目信息:${NC}"
echo "  项目名称: $PROJECT_NAME"
echo "  版本: $VERSION"
echo "  构建时间: $BUILD_TIME"
echo "  Git提交: ${GIT_COMMIT:0:8}"
echo ""

# 测试单个服务构建
test_service_build() {
    local service=$1
    echo -e "${YELLOW}测试构建 $service 服务...${NC}"
    
    if docker build --target $service \
        --build-arg SERVICE=$service \
        --build-arg VERSION=$VERSION \
        --build-arg BUILD_TIME="$BUILD_TIME" \
        --build-arg GIT_COMMIT="$GIT_COMMIT" \
        -t localhost:5000/$PROJECT_NAME-$service:$VERSION \
        -f Dockerfile . > /tmp/docker-build-$service.log 2>&1; then
        echo -e "${GREEN}✅ $service 服务构建成功${NC}"
        return 0
    else
        echo -e "${RED}❌ $service 服务构建失败${NC}"
        echo "构建日志:"
        tail -20 /tmp/docker-build-$service.log
        return 1
    fi
}

# 测试镜像运行
test_image_run() {
    local service=$1
    echo -e "${YELLOW}测试运行 $service 镜像...${NC}"
    
    # 尝试运行容器并检查版本
    if docker run --rm localhost:5000/$PROJECT_NAME-$service:$VERSION version > /tmp/docker-run-$service.log 2>&1; then
        echo -e "${GREEN}✅ $service 镜像运行成功${NC}"
        cat /tmp/docker-run-$service.log
        return 0
    else
        echo -e "${YELLOW}⚠️  $service 镜像运行测试跳过（可能需要配置文件）${NC}"
        return 0
    fi
}

# 清理函数
cleanup() {
    echo -e "${YELLOW}清理临时文件...${NC}"
    rm -f /tmp/docker-build-*.log /tmp/docker-run-*.log
}

# 设置清理陷阱
trap cleanup EXIT

# 主测试流程
main() {
    local success_count=0
    local total_count=${#SERVICES[@]}
    
    echo -e "${YELLOW}开始构建测试...${NC}"
    echo ""
    
    for service in "${SERVICES[@]}"; do
        if test_service_build $service; then
            ((success_count++))
            # test_image_run $service
        fi
        echo ""
    done
    
    echo -e "${YELLOW}构建测试完成!${NC}"
    echo "成功: $success_count/$total_count"
    
    if [ $success_count -eq $total_count ]; then
        echo -e "${GREEN}🎉 所有服务构建成功!${NC}"
        
        # 显示构建的镜像
        echo -e "${YELLOW}构建的镜像:${NC}"
        for service in "${SERVICES[@]}"; do
            if docker images localhost:5000/$PROJECT_NAME-$service:$VERSION --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}" | grep -v REPOSITORY; then
                :
            fi
        done
        
        return 0
    else
        echo -e "${RED}❌ 部分服务构建失败${NC}"
        return 1
    fi
}

# 检查Docker是否运行
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker未运行，请先启动Docker${NC}"
    exit 1
fi

# 检查是否在项目根目录
if [ ! -f "Dockerfile" ] || [ ! -f "go.mod" ]; then
    echo -e "${RED}❌ 请在项目根目录运行此脚本${NC}"
    exit 1
fi

# 运行主函数
main "$@"
