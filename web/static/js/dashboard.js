// 仪表板主控制器
// Industrial IoT Dashboard Controller

class Dashboard {
    constructor() {
        this.wsClient = null;
        this.messageHandler = null;
        this.chartManager = null;
        
        // 数据存储
        this.deviceData = new Map();
        this.alertData = [];
        this.systemStats = {
            deviceCount: 0,
            messageCount: 0,
            alertCount: 0,
            onlineDevices: 0,
            offlineDevices: 0,
            dataRate: 0,
            avgLatency: 0
        };
        
        // 过滤器状态
        this.currentFilters = {
            deviceTypes: new Set(['temperature', 'humidity', 'pressure', 'switch', 'current']),
            locations: new Set(['Building_1', 'Building_2']),
            floors: new Set(['1', '2', '3']),
            alertLevels: new Set(['critical', 'warning', 'info'])
        };
        
        // 订阅ID
        this.subscriptionId = null;
        
        // 统计计算
        this.messageRateCalculator = new MessageRateCalculator();
        
        this.init();
    }
    
    // 初始化仪表板
    init() {
        console.log('初始化IoT监控仪表板...');
        
        // 初始化聚合图表（如果可用）
        if (typeof AggregatedChart !== 'undefined' && typeof echarts !== 'undefined') {
            try {
                // 初始化聚合图表
                this.temperatureChart = new AggregatedChart('temperature-chart', {
                    title: '温度聚合趋势',
                    yAxisName: '温度',
                    unit: '°C'
                });
                this.humidityChart = new AggregatedChart('humidity-chart', {
                    title: '湿度聚合趋势',
                    yAxisName: '湿度',
                    unit: '%'
                });
                this.pressureChart = new AggregatedChart('pressure-chart', {
                    title: '压力聚合趋势',
                    yAxisName: '压力',
                    unit: 'hPa'
                });
                
                // 初始化状态饼图
                if (typeof StatusPieChart !== 'undefined') {
                    this.statusChart = new StatusPieChart('status-chart', {
                        title: '设备状态分布'
                    });
                }
                
                console.log('聚合图表初始化成功');
            } catch (error) {
                console.warn('聚合图表初始化失败:', error);
                this.temperatureChart = null;
                this.humidityChart = null;
                this.pressureChart = null;
                this.statusChart = null;
            }
        } else {
            console.warn('图表功能不可用，ECharts或AggregatedChart未加载');
            this.temperatureChart = null;
            this.humidityChart = null;
            this.pressureChart = null;
            this.statusChart = null;
        }
        
        // 初始化WebSocket客户端
        this.initWebSocket();
        
        // 初始化UI事件
        this.initUIEvents();
        
        // 初始化定时器
        this.initTimers();
        
        console.log('仪表板初始化完成');
    }
    
    // 初始化WebSocket连接
    initWebSocket() {
        // 连接到WebSocket服务器
        const wsUrl = `ws://${window.location.host}/ws`;
        console.log('WebSocket URL:', wsUrl);
        
        this.wsClient = new WebSocketClient(wsUrl, {
            reconnectInterval: 3000,
            maxReconnectAttempts: 10,
            heartbeatInterval: 30000
        });
        
        this.messageHandler = new MessageHandler();
        
        // 注册事件处理器
        this.wsClient.onConnect(() => {
            this.onWebSocketConnect();
        });
        
        this.wsClient.onDisconnect(() => {
            this.onWebSocketDisconnect();
        });
        
        this.wsClient.onError((error) => {
            this.onWebSocketError(error);
        });
        
        this.wsClient.onMessage((message) => {
            this.messageHandler.handleMessage(message);
        });
        
        // 注册消息处理器
        this.messageHandler.onDeviceData((data) => {
            this.handleDeviceData(data);
        });
        
        this.messageHandler.onAlert((alert) => {
            this.handleAlert(alert);
        });
        
        this.messageHandler.onSystemStatus((status) => {
            this.handleSystemStatus(status);
        });
    }
    
    // 初始化UI事件
    initUIEvents() {
        // 连接按钮
        const connectBtn = document.getElementById('connect-btn');
        connectBtn?.addEventListener('click', () => {
            this.toggleConnection();
        });
        
        // 过滤器应用按钮
        const applyFiltersBtn = document.getElementById('apply-filters');
        applyFiltersBtn?.addEventListener('click', () => {
            this.applyFilters();
        });
        
        // 图表时间范围按钮
        document.querySelectorAll('.chart-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.handleTimeRangeChange(e.target);
            });
        });
        
        // 窗口大小变化
        window.addEventListener('resize', () => {
            this.chartManager?.resizeAll();
        });
        
        // 过滤器复选框
        this.initFilterCheckboxes();
    }
    
    // 初始化过滤器复选框
    initFilterCheckboxes() {
        // 设备类型过滤器
        document.querySelectorAll('.filter-group input[type="checkbox"]').forEach(checkbox => {
            checkbox.addEventListener('change', () => {
                this.updateFilterState();
            });
        });
        
        // 告警级别过滤器
        document.querySelectorAll('.alert-filters input[type="checkbox"]').forEach(checkbox => {
            checkbox.addEventListener('change', () => {
                this.updateFilterState();
            });
        });
    }
    
    // 初始化定时器
    initTimers() {
        // 每秒更新统计信息
        setInterval(() => {
            this.updateSystemStats();
        }, 1000);
        
        // 每5秒清理旧数据
        setInterval(() => {
            this.cleanupOldData();
        }, 5000);
    }
    
    // WebSocket连接事件
    onWebSocketConnect() {
        console.log('WebSocket连接成功');
        this.updateConnectionStatus(true);
        
        // 订阅数据
        this.subscribeToData();
    }
    
    onWebSocketDisconnect() {
        console.log('WebSocket连接断开');
        this.updateConnectionStatus(false);
    }
    
    onWebSocketError(error) {
        console.error('WebSocket错误:', error);
        this.updateConnectionStatus(false);
        this.showNotification('WebSocket连接错误', 'error');
    }
    
    // 订阅数据
    subscribeToData() {
        if (!this.wsClient || !this.wsClient.isConnected) return;
        
        const filter = this.buildSubscriptionFilter();
        this.subscriptionId = this.wsClient.subscribe(filter);
        
        console.log('已订阅数据，订阅ID:', this.subscriptionId);
    }
    
    // 构建订阅过滤器
    buildSubscriptionFilter() {
        return {
            type: 'all', // 订阅所有类型的消息
            device_types: Array.from(this.currentFilters.deviceTypes),
            locations: Array.from(this.currentFilters.locations),
            floors: Array.from(this.currentFilters.floors),
            alert_levels: Array.from(this.currentFilters.alertLevels)
        };
    }
    
    // 处理设备数据
    handleDeviceData(data) {
        this.messageRateCalculator.addMessage();
        this.systemStats.messageCount++;
        
        // 存储设备数据
        this.deviceData.set(data.device_id, {
            ...data,
            lastUpdate: new Date()
        });
        
        // 更新聚合图表
        this.updateAggregatedCharts(data);
        
        // 更新最新数据显示
        this.updateRecentDataDisplay(data);
        
        // 更新设备计数
        this.updateDeviceCount();
    }
    
    // 处理告警
    handleAlert(alert) {
        this.systemStats.alertCount++;
        
        // 添加到告警列表
        this.alertData.unshift({
            ...alert,
            timestamp: new Date()
        });
        
        // 限制告警数量
        if (this.alertData.length > 50) {
            this.alertData = this.alertData.slice(0, 50);
        }
        
        // 更新告警显示
        this.updateAlertsDisplay();
        
        // 显示通知
        this.showNotification(`新告警: ${alert.message}`, this.getAlertNotificationType(alert.severity));
    }
    
    // 处理系统状态
    handleSystemStatus(status) {
        Object.assign(this.systemStats, status);
        this.updateSystemStatsDisplay();
    }
    
    // 切换连接状态
    toggleConnection() {
        if (this.wsClient?.isConnected) {
            this.wsClient.disconnect();
        } else {
            this.wsClient?.connect().catch(error => {
                console.error('连接失败:', error);
                this.showNotification('连接失败', 'error');
            });
        }
    }
    
    // 应用过滤器
    applyFilters() {
        this.updateFilterState();
        
        if (this.subscriptionId && this.wsClient?.isConnected) {
            const newFilter = this.buildSubscriptionFilter();
            this.wsClient.updateFilter(this.subscriptionId, newFilter);
            console.log('过滤器已更新');
            this.showNotification('过滤器已应用', 'success');
        }
    }
    
    // 更新过滤器状态
    updateFilterState() {
        // 设备类型
        this.currentFilters.deviceTypes.clear();
        document.querySelectorAll('.filter-group input[type="checkbox"]:checked').forEach(checkbox => {
            this.currentFilters.deviceTypes.add(checkbox.value);
        });
        
        // 位置
        this.currentFilters.locations.clear();
        document.querySelectorAll('.filter-group input[value^="Building"]:checked').forEach(checkbox => {
            this.currentFilters.locations.add(checkbox.value);
        });
        
        // 楼层
        this.currentFilters.floors.clear();
        document.querySelectorAll('.filter-group input[value="1"]:checked, .filter-group input[value="2"]:checked, .filter-group input[value="3"]:checked').forEach(checkbox => {
            this.currentFilters.floors.add(checkbox.value);
        });
        
        // 告警级别
        this.currentFilters.alertLevels.clear();
        document.querySelectorAll('.alert-filters input[type="checkbox"]:checked').forEach(checkbox => {
            this.currentFilters.alertLevels.add(checkbox.value);
        });
    }
    
    // 处理时间范围变化
    handleTimeRangeChange(button) {
        // 移除其他按钮的active类
        button.parentElement.querySelectorAll('.chart-btn').forEach(btn => {
            btn.classList.remove('active');
        });
        
        // 添加active类到当前按钮
        button.classList.add('active');
        
        const range = button.dataset.range;
        console.log('时间范围变更为:', range);
        
        // 这里可以实现时间范围过滤逻辑
        // 例如：清除图表数据并重新请求指定时间范围的数据
    }
    
    // 更新连接状态显示
    updateConnectionStatus(isConnected) {
        const indicator = document.getElementById('connection-indicator');
        const text = document.getElementById('connection-text');
        const button = document.getElementById('connect-btn');
        
        if (isConnected) {
            indicator?.classList.remove('offline');
            indicator?.classList.add('online');
            if (text) text.textContent = '在线';
            if (button) {
                button.innerHTML = '<i class="fas fa-plug"></i> 断开';
                button.classList.add('connected');
            }
        } else {
            indicator?.classList.remove('online');
            indicator?.classList.add('offline');
            if (text) text.textContent = '离线';
            if (button) {
                button.innerHTML = '<i class="fas fa-plug"></i> 连接';
                button.classList.remove('connected');
            }
        }
    }
    
    // 更新系统统计显示
    updateSystemStats() {
        // 更新消息速率
        this.systemStats.dataRate = this.messageRateCalculator.getRate();
        
        // 更新显示
        this.updateSystemStatsDisplay();
    }
    
    updateSystemStatsDisplay() {
        const elements = {
            'device-count': this.systemStats.deviceCount,
            'message-count': this.systemStats.messageCount,
            'alert-count': this.systemStats.alertCount,
            'online-devices': this.systemStats.onlineDevices,
            'offline-devices': this.systemStats.offlineDevices,
            'data-rate': this.systemStats.dataRate.toFixed(1) + ' msg/s',
            'avg-latency': this.systemStats.avgLatency + ' ms'
        };
        
        Object.entries(elements).forEach(([id, value]) => {
            const element = document.getElementById(id);
            if (element) element.textContent = value;
        });
        
        // 更新设备状态图表
        if (this.statusChart) {
            this.statusChart.updateData({
                '在线': this.systemStats.onlineDevices,
                '离线': this.systemStats.offlineDevices
            });
        }
    }
    
    // 更新设备计数
    updateDeviceCount() {
        const now = Date.now();
        let onlineCount = 0;
        let offlineCount = 0;
        
        this.deviceData.forEach(device => {
            const timeSinceUpdate = now - new Date(device.lastUpdate).getTime();
            if (timeSinceUpdate < 60000) { // 1分钟内有数据认为在线
                onlineCount++;
            } else {
                offlineCount++;
            }
        });
        
        this.systemStats.deviceCount = this.deviceData.size;
        this.systemStats.onlineDevices = onlineCount;
        this.systemStats.offlineDevices = offlineCount;
    }
    
    // 更新最新数据显示
    updateRecentDataDisplay(data) {
        const container = document.getElementById('recent-data');
        if (!container) return;
        
        // 移除"暂无数据"提示
        const noData = container.querySelector('.no-data');
        if (noData) noData.remove();
        
        // 创建数据项
        const dataItem = document.createElement('div');
        dataItem.className = 'data-item';
        dataItem.innerHTML = `
            <div class="device-id">${data.device_id}</div>
            <div class="data-value">类型: ${data.device_type}</div>
            <div class="data-value">时间: ${new Date(data.timestamp).toLocaleTimeString()}</div>
        `;
        
        // 添加到顶部
        container.insertBefore(dataItem, container.firstChild);
        
        // 限制显示数量
        const items = container.querySelectorAll('.data-item');
        if (items.length > 10) {
            items[items.length - 1].remove();
        }
    }
    
    // 更新告警显示
    updateAlertsDisplay() {
        const container = document.getElementById('alerts-list');
        if (!container) return;
        
        // 清空容器
        container.innerHTML = '';
        
        if (this.alertData.length === 0) {
            container.innerHTML = '<div class="no-alerts">暂无告警</div>';
            return;
        }
        
        // 显示最新的告警
        this.alertData.slice(0, 10).forEach(alert => {
            const alertItem = document.createElement('div');
            alertItem.className = `alert-item ${alert.severity}`;
            alertItem.innerHTML = `
                <div>${alert.message}</div>
                <div class="alert-time">${alert.timestamp.toLocaleString()}</div>
            `;
            container.appendChild(alertItem);
        });
    }
    
    // 清理旧数据
    cleanupOldData() {
        const now = Date.now();
        const maxAge = 5 * 60 * 1000; // 5分钟
        
        // 清理设备数据
        for (const [deviceId, device] of this.deviceData.entries()) {
            if (now - new Date(device.lastUpdate).getTime() > maxAge) {
                this.deviceData.delete(deviceId);
            }
        }
        
        // 清理告警数据
        this.alertData = this.alertData.filter(alert => 
            now - alert.timestamp.getTime() < maxAge
        );
    }
    
    // 显示通知
    showNotification(message, type = 'info') {
        // 这里可以实现通知显示逻辑
        console.log(`[${type.toUpperCase()}] ${message}`);
    }
    
    // 获取告警通知类型
    getAlertNotificationType(severity) {
        switch (severity) {
            case 'critical': return 'error';
            case 'warning': return 'warning';
            case 'info': return 'info';
            default: return 'info';
        }
    }
    
    // 更新聚合图表
    updateAggregatedCharts(data) {
        if (!data.sensors) return;
        
        const timestamp = new Date(data.timestamp || Date.now());
        const deviceId = data.device_id;
        
        // 更新温度图表
        if (data.sensors.temperature && this.temperatureChart) {
            this.temperatureChart.addDataPoint(
                deviceId, 
                data.sensors.temperature.value, 
                timestamp
            );
        }
        
        // 更新湿度图表
        if (data.sensors.humidity && this.humidityChart) {
            this.humidityChart.addDataPoint(
                deviceId, 
                data.sensors.humidity.value, 
                timestamp
            );
        }
        
        // 更新压力图表
        if (data.sensors.pressure && this.pressureChart) {
            this.pressureChart.addDataPoint(
                deviceId, 
                data.sensors.pressure.value, 
                timestamp
            );
        }
    }
}

// 消息速率计算器
class MessageRateCalculator {
    constructor(windowSize = 60) {
        this.windowSize = windowSize; // 时间窗口大小（秒）
        this.messages = [];
    }
    
    addMessage() {
        const now = Date.now();
        this.messages.push(now);
        
        // 清理超出时间窗口的消息
        const cutoff = now - (this.windowSize * 1000);
        this.messages = this.messages.filter(timestamp => timestamp > cutoff);
    }
    
    getRate() {
        return this.messages.length / this.windowSize;
    }
}

// 导出类
window.Dashboard = Dashboard;
window.MessageRateCalculator = MessageRateCalculator;
