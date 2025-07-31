# Step 1.1: 项目初始化和基础架构 - GitHub作品集文档创建指令

## 📋 任务背景
- **项目**: 工业IoT实时数据监控系统 (Go + WebSocket + Kafka架构)
- **当前阶段**: Step 1.1 - 项目初始化和基础架构
- **前置条件**: 无，项目起始阶段
- **目标受众**: GitHub招聘者、技术面试官、开源社区开发者
- **作品集定位**: 展示Go语言全栈开发能力、企业级架构设计能力、工程化开发实践

## 🎯 文档创建要求

请创建一份名为"Step 1.1: 项目初始化和基础架构 - 企业级工业IoT监控系统"的技术文档，内容结构必须与以下模板完全一致：

### **第一部分：技术亮点展示** (占文档35%)

1. **项目技术亮点**
   - 明确的技术成就与KPI指标(支持10,000+设备并发监控、100,000+消息/秒处理能力、<50ms数据延迟)
   - 核心技术栈展示(Go 1.24+、Apache Kafka、WebSocket、Chart.js、Docker容器化)
   - 企业级系统特征展示(微服务架构、高并发处理、实时数据流、可视化监控、弹性扩展)
   - 工业IoT领域价值展示(实时监控、智能告警、数据可视化、运维自动化)

2. **技术选型与架构设计**
   - 后端技术栈对比分析表格(Go vs Java vs Python vs Node.js)
   - 包含并发性能、内存占用、开发效率、生态成熟度、部署便利性、推荐指数
   - 消息队列对比分析(Kafka vs RabbitMQ vs Redis vs NATS vs Apache Pulsar)
   - 实时通信方案对比(WebSocket vs Server-Sent Events vs gRPC Streaming vs Socket.IO)
   - 前端技术选择(原生JavaScript + Chart.js vs React vs Vue.js)
   - 容器化方案选择(Docker vs Podman vs containerd)

3. **核心架构设计**
   - 系统整体架构图(设备层→消息队列层→应用服务层→存储层→前端展示层→监控层)
   - 数据流架构图(数据源→设备网关→消息队列→实时处理→数据存储→实时推送→前端展示)
   - 微服务架构图(Producer服务→Consumer服务→WebSocket服务→Web服务)
   - 项目目录结构架构图(cmd/→internal/→web/→scripts/→deployments/→tests/→docs/)

### **第二部分：核心实现展示** (占文档45%)

4. **开发实施计划** (总计3天)
   - **第一阶段**(第1天): 项目基础搭建
     - Step 1.1.1: Go模块初始化和依赖管理配置
     - Step 1.1.2: 项目目录结构设计和创建
   - **第二阶段**(第2天): 开发环境配置
     - Step 1.1.3: Git仓库配置和版本控制策略
     - Step 1.1.4: Makefile构建系统和自动化脚本
   - **第三阶段**(第3天): 文档和规范建立
     - Step 1.1.5: README.md文档结构和开发指南
     - Step 1.1.6: 项目规范和代码标准建立

5. **核心架构设计规范**
   - 项目目录结构架构设计和组织规范
   - Go模块依赖管理架构设计和版本控制策略
   - Makefile构建系统架构设计和任务自动化规范
   - Git版本控制架构设计和分支管理策略
   - 代码风格和开发规范架构设计
   - 文档结构和维护规范架构设计
   - 测试策略和质量保证规范架构设计
   - 关键目录和文件的职责定义（不包含具体实现内容）

6. **技术特性演示**
   - 完整项目目录结构展示(包含所有必要目录和文件)
   - Go模块配置和依赖管理展示
   - Makefile任务自动化和构建流程展示
   - Git配置和.gitignore规则展示
   - README.md结构和文档规范展示

### **第三部分：运维与部署** (占文档10% - 精简版)

7. **项目管理策略**
   - 简洁的版本控制和分支管理策略
   - 代码审查和质量保证流程
   - 持续集成和自动化构建准备

8. **开发环境**
   - 本地开发环境搭建指南
   - 开发工具和IDE配置建议
   - 代码格式化和静态检查工具配置

### **第四部分：项目成果** (占文档10% - 精简版)

9. **功能演示与测试**
   - 项目结构完整性验证预期结果
   - Go模块和构建系统测试预期效果
   - 开发环境配置测试预期数据

10. **GitHub展示要点**
    - 项目亮点总结(工业IoT系统、微服务架构、高并发处理、实时监控)
    - 技术能力展示(Go语言工程化实践、系统架构设计、项目管理能力)
    - 预期项目规模和复杂度展示

## 🔧 技术规格要求

### **核心技术栈**
- Go 1.24+, Go Modules, Go工具链
- Git 2.0+, Makefile, Shell脚本
- 项目管理: GitHub, README.md, .gitignore
- 构建工具: make, go build, go mod

### **完整项目架构指导要求**
- 必须提供详细的项目目录结构设计，让AI能完整创建
- 必须提供完整的Go模块配置和依赖管理规范，让AI能完整实现
- 所有设计必须基于README.md中描述的完整项目结构
- 架构设计必须为后续Step 1.2配置管理和Step 1.3数据模型做好准备
- 包含完整的文件结构、构建规范、开发流程，供AI生成项目时参考

### **项目结构规格** (基于README.md)
- **cmd/**: 应用程序入口目录(producer/consumer/websocket/web主程序)
- **internal/**: 内部业务逻辑目录(config/models/services子目录)
- **web/**: 前端资源目录(static/templates子目录)
- **scripts/**: 部署和工具脚本目录
- **deployments/**: 部署配置目录(docker子目录)
- **tests/**: 测试代码目录(unit/integration/performance子目录)
- **docs/**: 项目文档目录

### **Go模块管理规格**
- **依赖管理**: github.com/spf13/viper, github.com/gorilla/websocket, github.com/Shopify/sarama
- **版本控制**: Go 1.24+, 语义化版本控制
- **模块配置**: go.mod和go.sum文件管理
- **构建标签**: 支持不同环境和构建模式

### **Makefile任务规格**
- **构建任务**: build, build-all, build-producer, build-consumer, build-websocket, build-web
- **测试任务**: test, test-unit, test-integration, test-performance, test-coverage
- **开发任务**: dev, dev-producer, dev-consumer, dev-websocket, dev-web
- **部署任务**: docker-build, docker-up, docker-down, deploy
- **工具任务**: lint, format, clean, deps, help

### **Git管理规格**
- **分支策略**: main(生产), develop(开发), feature/(功能), hotfix/(热修复)
- **忽略规则**: Go编译产物、IDE配置、日志文件、环境变量文件、临时文件
- **提交规范**: feat/fix/docs/style/refactor/test/chore前缀规范
- **标签管理**: 版本标签和发布管理

### **性能目标**
- 项目构建时间 < 30秒
- 单元测试执行时间 < 5秒
- 代码格式化时间 < 3秒
- 支持并行构建和测试

### **展示重点**
- 工程化: Go语言项目工程化最佳实践
- 规范化: 代码结构清晰、文档完整、规范统一
- 自动化: 构建自动化、测试自动化、部署自动化
- 扩展性: 易于扩展的项目架构和模块设计

## 📝 GitHub作品集文档风格要求

### **技术展示性**
- 突出Go语言全栈开发能力
- 展示企业级项目管理和工程化实践
- 包含具体的架构设计和技术选型分析
- 体现对工业IoT领域的深度理解

### **可视化展示**  
- 丰富的系统架构图和技术栈对比图
- 清晰的项目目录结构和组织架构图
- 直观的开发流程和构建流程图
- 完整的技术选型对比和分析图表

### **简洁专业**
- 重点突出项目架构设计能力
- 运维部分简化，聚焦开发环境配置
- 测试部分精简，突出质量保证策略
- 突出实际可用的企业级项目框架

### **招聘导向**
- 体现全栈开发能力和架构设计能力
- 展示工业IoT项目经验和领域知识
- 突出Go语言工程化开发实践
- 体现企业级项目管理和团队协作能力

## 🎯 特别强调

1. **架构设计**: 必须突出企业级系统架构设计能力
2. **工程化**: 重点展示Go语言项目工程化最佳实践
3. **规范化**: 详细说明代码规范、文档规范、开发流程规范
4. **扩展性**: 说明为后续功能开发预留的架构空间
5. **专业性**: 体现对工业IoT监控系统的专业理解

## 📋 输出格式要求

请创建一个完整的Markdown文档，文档长度应在4000-6000字之间，确保内容的技术深度和展示效果平衡。文档结构要清晰，多用表格、架构图、对比图等格式丰富内容呈现。

### **📁 项目结构架构规范**
文档必须包含以下详细的架构设计，供AI实现时参考：

1. **完整项目目录结构设计** (基于README.md)
   ```
   industrial-iot-monitor/                   # 项目根目录架构设计
   ├── cmd/                                 # 应用程序入口目录设计
   │   ├── producer/                        # 设备数据生成器目录
   │   │   └── main.go                      # 生产者主程序设计
   │   ├── consumer/                        # 数据消费服务目录
   │   │   └── main.go                      # 消费者主程序设计
   │   ├── websocket/                       # WebSocket服务目录
   │   │   └── main.go                      # WebSocket服务器设计
   │   └── web/                             # Web服务器目录
   │       └── main.go                      # Web服务器设计
   ├── internal/                            # 内部业务逻辑目录设计
   │   ├── config/                          # 配置管理目录
   │   │   └── config.go                    # 配置文件解析设计
   │   ├── models/                          # 数据模型目录
   │   │   ├── device.go                    # 设备数据模型设计
   │   │   └── message.go                   # 消息模型设计
   │   └── services/                        # 业务服务目录
   │       ├── producer/                    # 生产者服务目录
   │       ├── consumer/                    # 消费者服务目录
   │       └── websocket/                   # WebSocket处理器目录
   ├── web/                                 # 前端资源目录设计
   │   ├── static/                          # 静态资源目录
   │   └── templates/                       # HTML模板目录
   ├── scripts/                             # 部署和工具脚本目录设计
   ├── deployments/                         # 部署配置目录设计
   ├── tests/                               # 测试代码目录设计
   ├── docs/                                # 项目文档目录设计
   ├── go.mod                               # Go模块依赖设计
   ├── go.sum                               # 依赖版本锁定设计
   ├── Makefile                             # 构建和任务自动化设计
   ├── README.md                            # 项目说明文档设计
   ├── LICENSE                              # 项目许可证设计
   └── .env.example                         # 环境变量示例设计
   ```

2. **Go模块配置架构设计**
   ```go
   // go.mod 模块配置设计规范
   module industrial-iot-monitor
   
   go 1.24
   
   require (
       // 核心依赖设计
       github.com/spf13/viper v1.17.0          // 配置管理
       github.com/gorilla/websocket v1.5.0     // WebSocket通信
       github.com/Shopify/sarama v1.41.0       // Kafka客户端
       github.com/go-playground/validator/v10 v10.15.0  // 数据验证
       // 其他依赖...
   )
   ```

3. **Makefile任务架构设计**
   ```makefile
   # Makefile 构建任务设计规范
   .PHONY: help build test clean docker-up
   
   # 构建任务设计
   build: build-producer build-consumer build-websocket build-web
   build-producer:    # 构建生产者服务
   build-consumer:    # 构建消费者服务
   build-websocket:   # 构建WebSocket服务
   build-web:         # 构建Web服务
   
   # 测试任务设计
   test: test-unit test-integration
   test-unit:         # 单元测试
   test-integration:  # 集成测试
   test-coverage:     # 测试覆盖率
   
   # 开发任务设计
   dev: dev-env       # 启动开发环境
   deps:              # 安装依赖
   lint:              # 代码检查
   format:            # 代码格式化
   
   # 部署任务设计
   docker-build:      # 构建Docker镜像
   docker-up:         # 启动Docker服务
   docker-down:       # 停止Docker服务
   ```

4. **Git配置架构设计**
   ```gitignore
   # .gitignore 规则设计
   # Go编译产物
   *.exe
   *.exe~
   *.dll
   *.so
   *.dylib
   *.test
   *.out
   /bin/
   /dist/
   
   # IDE配置
   .vscode/
   .idea/
   *.swp
   *.swo
   
   # 环境配置
   .env
   .env.local
   .env.*.local
   
   # 日志文件
   *.log
   /logs/
   
   # 临时文件
   /tmp/
   *.tmp
   ```

5. **README.md架构设计**
   ```markdown
   # README.md 文档结构设计
   - 项目描述和核心价值
   - 技术栈展示
   - 系统架构图
   - 快速开始指南
   - 项目目录结构
   - 核心功能特点
   - 性能指标
   - 开发指南
   - API文档
   - 部署说明
   - 测试指南
   - 贡献指南
   ```

### **架构设计详细程度要求**
- 提供详细的目录结构创建规范和文件职责定义
- 提供完整的Go模块配置和依赖管理策略
- 提供详细的Makefile任务定义和构建流程设计
- 提供Git配置和版本控制的详细策略设计
- 提供README.md和文档的详细结构设计
- **不包含具体的代码实现，只包含架构设计和组织规范**

### **项目管理设计标准** (基于README.md要求)
```yaml
# 项目管理配置示例
project:
  name: "industrial-iot-monitor"
  version: "v1.0.0"
  description: "高性能工业设备实时数据监控系统"
  
build:
  go_version: "1.24+"
  target_platforms: ["linux/amd64", "darwin/amd64", "windows/amd64"]
  
development:
  code_style: "gofmt"
  linter: "golangci-lint"
  test_framework: "go test"
  
deployment:
  containerization: "docker"
  orchestration: "docker-compose"
  registry: "docker.io"
```

重点包含：
- 系统整体架构图和技术选型对比图
- **详细的项目结构设计和组织规范** (供AI实现项目时参考)
- Go模块管理和构建系统预期展示
- 开发环境配置和工具链预期展示
- 项目技术亮点总结

最后，在文档末尾添加：
- GitHub项目展示要点总结
- Go语言工程化技术栈掌握程度说明
- **完整的项目架构设计总结** (供AI生成项目时参考)
- **项目初始化清单和规范** (每个组件的创建要求)
- 下一步开发方向(Step 1.2预览)

### **🎯 文档与实现分离要求**
生成的1.1.md架构文档必须确保：
1. ✅ **只包含架构设计，不包含具体代码和文件内容**
2. ✅ 包含足够详细的结构规范，让AI能根据文档完整创建项目
3. ✅ 所有设计基于README.md中定义的完整项目结构
4. ✅ 架构设计为Step 1.2配置管理正确预留接口
5. ✅ 包含完整的目录结构和文件职责定义
6. ✅ 提供清晰的创建指导，但不提供具体文件内容

### **📋 后续项目生成说明**
基于此1.1.md架构文档，AI应该能够：
- 根据目录设计创建完整的项目目录结构
- 根据模块设计生成完整的go.mod和go.sum文件
- 根据构建设计生成完整的Makefile文件
- 根据版本控制设计生成完整的.gitignore文件
- 根据文档设计生成完整的README.md基础结构

### **🏗️ 项目初始化架构层次**
```
企业级项目初始化架构:
┌─────────────────────────────────────────┐
│            项目管理层 (Project Layer)      │
├─────────────────────────────────────────┤
│ ┌─────────────┐ ┌─────────────────────┐ │
│ │   版本控制   │ │      构建系统        │ │
│ │    (Git)    │ │    (Makefile)       │ │
│ └─────────────┘ └─────────────────────┘ │
├─────────────────────────────────────────┤
│            模块管理层 (Module Layer)       │
│ ┌─────────────────────────────────────┐ │
│ │          Go Modules                 │ │
│ │      (依赖管理和版本控制)             │ │
│ └─────────────────────────────────────┘ │
├─────────────────────────────────────────┤
│           目录结构层 (Structure Layer)    │
│ ┌────────┬────────┬────────┬─────────┐  │
│ │  cmd/  │internal│  web/  │scripts/ │  │
│ │ 入口   │ 业务   │ 前端   │ 脚本    │  │
│ └────────┴────────┴────────┴─────────┘  │
└─────────────────────────────────────────┘
```

现在请开始创建这份面向GitHub作品集的Step 1.1**架构设计文档**，确保AI后续能根据此文档完整初始化整个工业IoT监控系统项目和企业级Go项目架构。