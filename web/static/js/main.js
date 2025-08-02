// Industrial IoT Monitoring System - Main Entry Point
// 工业IoT监控系统主入口

// 全局变量
let dashboard = null;

// 初始化图表的函数
function initializeCharts() {
    console.log('初始化聚合统计图表...');
    if (dashboard && typeof AggregatedChart !== 'undefined' && typeof echarts !== 'undefined') {
        try {
            // 初始化聚合图表
            dashboard.temperatureChart = new AggregatedChart('temperature-chart', {
                title: '温度聚合趋势',
                yAxisName: '温度',
                unit: '°C'
            });
            dashboard.humidityChart = new AggregatedChart('humidity-chart', {
                title: '湿度聚合趋势',
                yAxisName: '湿度',
                unit: '%'
            });
            dashboard.pressureChart = new AggregatedChart('pressure-chart', {
                title: '压力聚合趋势',
                yAxisName: '压力',
                unit: 'hPa'
            });
            
            console.log('聚合统计图表初始化成功!');
        } catch (error) {
            console.error('聚合统计图表初始化失败:', error);
        }
    } else {
        console.warn('聚合统计图表初始化条件不满足:', {
            dashboard: !!dashboard,
            AggregatedChart: typeof AggregatedChart !== 'undefined',
            echarts: typeof echarts !== 'undefined'
        });
    }
}

// 主初始化函数页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    console.log('Industrial IoT Monitoring Dashboard 正在启动...');
    
    // 延迟检查ECharts加载情况，给脚本加载时间
    setTimeout(function() {
        console.log('检查ECharts状态...');
        console.log('typeof echarts:', typeof echarts);
        console.log('window.echarts:', window.echarts);
        
        if (typeof echarts === 'undefined') {
            console.warn('ECharts 未加载，图表功能将被禁用');
            console.log('当前已加载的脚本:');
            document.querySelectorAll('script').forEach((script, index) => {
                console.log(`Script ${index}: ${script.src || 'inline'}`);
            });
        } else {
            console.log('ECharts 加载成功, 版本:', echarts.version);
            // 初始化图表
            initializeCharts();
        }
    }, 500); // 等待500ms
    
    // 检查其他依赖，但不阻止初始化
    if (typeof WebSocketClient === 'undefined') {
        console.error('WebSocketClient 未加载');
        return;
    }
    
    if (typeof AggregatedChart === 'undefined') {
        console.warn('AggregatedChart 未加载，图表功能将被禁用');
    }
    
    if (typeof Dashboard === 'undefined') {
        console.error('Dashboard 未加载');
        return;
    }
    
    try {
        // 创建仪表板实例
        dashboard = new Dashboard();
        
        console.log('Industrial IoT Monitoring Dashboard 启动成功');
        
        // 显示启动成功消息
        showStartupMessage();
        
    } catch (error) {
        console.error('仪表板初始化失败:', error);
        showErrorMessage('仪表板初始化失败: ' + error.message);
    }
});

// 显示启动消息
function showStartupMessage() {
    console.log('🚀 Industrial IoT Monitoring Dashboard');
    console.log('📊 实时数据可视化系统已就绪');
    console.log('🔌 点击连接按钮开始监控设备数据');
}

// 显示错误消息
function showErrorMessage(message) {
    const errorDiv = document.createElement('div');
    errorDiv.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: #ef4444;
        color: white;
        padding: 1rem;
        border-radius: 0.5rem;
        z-index: 9999;
        max-width: 400px;
    `;
    errorDiv.textContent = message;
    
    document.body.appendChild(errorDiv);
    
    // 5秒后自动移除
    setTimeout(() => {
        errorDiv.remove();
    }, 5000);
}

// 页面卸载时清理资源
window.addEventListener('beforeunload', function() {
    if (dashboard && dashboard.wsClient) {
        dashboard.wsClient.disconnect();
    }
    
    if (dashboard && dashboard.chartManager) {
        dashboard.chartManager.destroyAll();
    }
});

// 导出全局访问
window.dashboard = dashboard;

