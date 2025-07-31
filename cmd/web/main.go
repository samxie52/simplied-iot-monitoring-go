package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// 版本信息（构建时注入）
var (
	Version   = "dev"
	BuildTime = "unknown"
	GoVersion = "unknown"
)

// WebServer Web服务器
type WebServer struct {
	router    *mux.Router
	templates *template.Template
}

// NewWebServer 创建新的Web服务器
func NewWebServer() *WebServer {
	ws := &WebServer{
		router: mux.NewRouter(),
	}

	// 加载模板
	ws.loadTemplates()

	// 设置路由
	ws.setupRoutes()

	return ws
}

// loadTemplates 加载HTML模板
func (ws *WebServer) loadTemplates() {
	templateDir := "web/templates"
	
	// 检查模板目录是否存在
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		log.Printf("模板目录不存在，将创建默认模板: %s", templateDir)
		ws.createDefaultTemplates()
	}

	// 加载所有模板文件
	templatePattern := filepath.Join(templateDir, "*.html")
	var err error
	ws.templates, err = template.ParseGlob(templatePattern)
	if err != nil {
		log.Printf("加载模板失败，使用内置模板: %v", err)
		ws.createInlineTemplates()
	}
}

// createDefaultTemplates 创建默认模板文件
func (ws *WebServer) createDefaultTemplates() {
	// 这个方法将在后续实现中创建实际的模板文件
	log.Printf("使用内置模板")
}

// createInlineTemplates 创建内联模板
func (ws *WebServer) createInlineTemplates() {
	indexTemplate := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>工业IoT监控系统</title>
    <link rel="stylesheet" href="/static/css/main.css">
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🏭 工业IoT实时监控系统</h1>
            <p style="text-align: center; margin: 10px 0 0 0; color: #7f8c8d;">
                版本: {{.Version}} | 构建时间: {{.BuildTime}}
            </p>
        </div>

        <div class="stats">
            <div class="stat-card">
                <h3>📊 在线设备</h3>
                <div class="stat-value" id="online-devices">0</div>
            </div>
            <div class="stat-card">
                <h3>📈 消息/秒</h3>
                <div class="stat-value" id="messages-per-sec">0</div>
            </div>
            <div class="stat-card">
                <h3>⚠️ 告警数量</h3>
                <div class="stat-value" id="alert-count">0</div>
            </div>
            <div class="stat-card">
                <h3>🔄 系统状态</h3>
                <div class="stat-value" style="font-size: 1.2em;">运行中</div>
            </div>
        </div>

        <div class="realtime-data">
            <h3>🔗 实时数据连接</h3>
            <div class="connection-status">
                <span id="ws-status" class="disconnected">未连接</span>
            </div>
            <button onclick="toggleConnection()" id="connect-btn">连接WebSocket</button>
        </div>

        <div class="device-list">
            <h3>📱 设备列表</h3>
            <div id="device-container">
                <p style="text-align: center; color: #7f8c8d;">等待设备数据...</p>
            </div>
        </div>
    </div>

    <script src="/static/js/main.js"></script>
</body>
</html>
`

	var err error
	ws.templates, err = template.New("index").Parse(indexTemplate)
	if err != nil {
		log.Fatalf("创建内联模板失败: %v", err)
	}
}

// setupRoutes 设置路由
func (ws *WebServer) setupRoutes() {
	// 主页
	ws.router.HandleFunc("/", ws.handleIndex).Methods("GET")
	
	// 健康检查
	ws.router.HandleFunc("/health", ws.handleHealth).Methods("GET")
	
	// API路由
	api := ws.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/status", ws.handleAPIStatus).Methods("GET")
	
	// 静态文件服务
	staticDir := "web/static/"
	if _, err := os.Stat(staticDir); !os.IsNotExist(err) {
		ws.router.PathPrefix("/static/").Handler(
			http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	}
}

// handleIndex 处理主页请求
func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Version   string
		BuildTime string
		GoVersion string
	}{
		Version:   Version,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	if err := ws.templates.ExecuteTemplate(w, "index", data); err != nil {
		log.Printf("模板执行失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}
}

// handleHealth 处理健康检查请求
func (ws *WebServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}

// handleAPIStatus 处理API状态请求
func (ws *WebServer) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":    "IoT Monitoring Web Server",
		"version":    Version,
		"build_time": BuildTime,
		"go_version": GoVersion,
		"status":     "running",
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	// 简单的JSON序列化
	jsonStr := fmt.Sprintf(`{
		"service": "%s",
		"version": "%s",
		"build_time": "%s",
		"go_version": "%s",
		"status": "%s",
		"timestamp": "%s"
	}`, status["service"], status["version"], status["build_time"], 
		status["go_version"], status["status"], status["timestamp"])
	
	w.Write([]byte(jsonStr))
}

// runWebServer 运行Web服务器
func runWebServer(cmd *cobra.Command, args []string) error {
	// 读取配置
	port := viper.GetString("web.port")

	log.Printf("启动Web管理界面服务...")
	log.Printf("服务端口: %s", port)

	// 创建Web服务器
	webServer := NewWebServer()

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      webServer.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动HTTP服务器
	go func() {
		log.Printf("Web服务器启动在端口 %s", port)
		log.Printf("访问地址: http://localhost:%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP服务器启动失败: %v", err)
		}
	}()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Web管理界面服务已启动，按 Ctrl+C 停止")

	// 等待停止信号
	<-sigChan
	log.Printf("收到停止信号，正在关闭Web服务...")

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP服务器关闭失败: %v", err)
	}

	return nil
}

// initConfig 初始化配置
func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("web.port", "8090")

	// 支持环境变量
	viper.SetEnvPrefix("IOT")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("配置文件读取失败，使用默认配置: %v", err)
	}
}

func main() {
	// 初始化配置
	initConfig()

	// 创建根命令
	var rootCmd = &cobra.Command{
		Use:   "web",
		Short: "Web管理界面服务",
		Long: `工业IoT监控系统Web管理界面服务
		
该服务提供Web界面，用于实时监控IoT设备状态、查看数据和管理系统。
集成WebSocket实时数据展示功能。`,
		Version: fmt.Sprintf("%s (构建时间: %s, Go版本: %s)", Version, BuildTime, GoVersion),
		RunE:    runWebServer,
	}

	// 添加命令行参数
	rootCmd.Flags().StringP("port", "p", "8090", "Web服务端口")

	// 绑定命令行参数到配置
	viper.BindPFlag("web.port", rootCmd.Flags().Lookup("port"))

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("程序执行失败: %v", err)
	}
}
