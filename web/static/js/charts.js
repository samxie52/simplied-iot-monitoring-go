// 实时图表管理类
// Industrial IoT Real-time Charts

class RealtimeChart {
    constructor(canvasId, options = {}) {
        this.canvasId = canvasId;
        this.canvas = document.getElementById(canvasId);
        this.ctx = this.canvas.getContext('2d');
        
        this.options = {
            maxDataPoints: 50,
            updateInterval: 1000,
            animationDuration: 300,
            maintainAspectRatio: false,
            responsive: true,
            ...options
        };
        
        this.chart = null;
        this.dataBuffer = [];
        this.isUpdating = false;
        
        this.initChart();
    }
    
    initChart() {
        const defaultConfig = {
            type: 'line',
            data: {
                labels: [],
                datasets: []
            },
            options: {
                responsive: this.options.responsive,
                maintainAspectRatio: this.options.maintainAspectRatio,
                animation: {
                    duration: this.options.animationDuration
                },
                scales: {
                    x: {
                        type: 'time',
                        time: {
                            displayFormats: {
                                minute: 'HH:mm',
                                hour: 'HH:mm'
                            }
                        },
                        title: {
                            display: true,
                            text: '时间'
                        }
                    },
                    y: {
                        beginAtZero: false,
                        title: {
                            display: true,
                            text: '数值'
                        }
                    }
                },
                plugins: {
                    legend: {
                        display: true,
                        position: 'top'
                    },
                    tooltip: {
                        mode: 'index',
                        intersect: false,
                        callbacks: {
                            title: function(tooltipItems) {
                                return new Date(tooltipItems[0].parsed.x).toLocaleString();
                            }
                        }
                    }
                },
                interaction: {
                    mode: 'nearest',
                    axis: 'x',
                    intersect: false
                }
            }
        };
        
        // 合并用户配置
        const config = this.mergeConfig(defaultConfig, this.options.chartConfig || {});
        
        this.chart = new Chart(this.ctx, config);
    }
    
    // 添加数据点
    addDataPoint(deviceId, value, timestamp = new Date()) {
        if (!this.chart) return;
        
        const datasetIndex = this.findOrCreateDataset(deviceId);
        const dataset = this.chart.data.datasets[datasetIndex];
        
        // 添加数据点
        dataset.data.push({
            x: timestamp,
            y: value
        });
        
        // 限制数据点数量
        if (dataset.data.length > this.options.maxDataPoints) {
            dataset.data.shift();
        }
        
        // 更新图表
        this.updateChart();
    }
    
    // 批量更新数据
    updateData(dataPoints) {
        if (!this.chart || !Array.isArray(dataPoints)) return;
        
        dataPoints.forEach(point => {
            this.addDataPoint(point.deviceId, point.value, point.timestamp);
        });
    }
    
    // 设置阈值线
    setThreshold(value, label = '阈值', color = 'rgba(255, 99, 132, 0.8)') {
        if (!this.chart) return;
        
        // 移除现有阈值线
        this.removeThreshold();
        
        // 添加新的阈值线
        this.chart.data.datasets.push({
            label: label,
            data: Array(this.options.maxDataPoints).fill(value),
            borderColor: color,
            backgroundColor: 'transparent',
            borderDash: [5, 5],
            pointRadius: 0,
            tension: 0,
            order: 999 // 确保阈值线在最后绘制
        });
        
        this.updateChart();
    }
    
    // 移除阈值线
    removeThreshold() {
        if (!this.chart) return;
        
        this.chart.data.datasets = this.chart.data.datasets.filter(
            dataset => !dataset.label.includes('阈值')
        );
    }
    
    // 清空数据
    clearData() {
        if (!this.chart) return;
        
        this.chart.data.datasets.forEach(dataset => {
            dataset.data = [];
        });
        
        this.updateChart();
    }
    
    // 更新图表
    updateChart() {
        if (!this.chart || this.isUpdating) return;
        
        this.isUpdating = true;
        
        requestAnimationFrame(() => {
            this.chart.update('none'); // 使用 'none' 模式避免动画
            this.isUpdating = false;
        });
    }
    
    // 调整大小
    resize() {
        if (this.chart) {
            this.chart.resize();
        }
    }
    
    // 销毁图表
    destroy() {
        if (this.chart) {
            this.chart.destroy();
            this.chart = null;
        }
    }
    
    // 导出数据
    exportData() {
        if (!this.chart) return null;
        
        return {
            labels: this.chart.data.labels,
            datasets: this.chart.data.datasets.map(dataset => ({
                label: dataset.label,
                data: dataset.data,
                backgroundColor: dataset.backgroundColor,
                borderColor: dataset.borderColor
            }))
        };
    }
    
    // 私有方法
    
    findOrCreateDataset(deviceId) {
        const existingIndex = this.chart.data.datasets.findIndex(
            dataset => dataset.label === deviceId
        );
        
        if (existingIndex !== -1) {
            return existingIndex;
        }
        
        // 创建新的数据集
        const colors = this.getDeviceColor(deviceId);
        const newDataset = {
            label: deviceId,
            data: [],
            borderColor: colors.border,
            backgroundColor: colors.background,
            tension: 0.4,
            pointRadius: 2,
            pointHoverRadius: 4,
            borderWidth: 2
        };
        
        this.chart.data.datasets.push(newDataset);
        return this.chart.data.datasets.length - 1;
    }
    
    getDeviceColor(deviceId) {
        const colors = [
            { border: 'rgb(75, 192, 192)', background: 'rgba(75, 192, 192, 0.2)' },
            { border: 'rgb(255, 99, 132)', background: 'rgba(255, 99, 132, 0.2)' },
            { border: 'rgb(54, 162, 235)', background: 'rgba(54, 162, 235, 0.2)' },
            { border: 'rgb(255, 205, 86)', background: 'rgba(255, 205, 86, 0.2)' },
            { border: 'rgb(153, 102, 255)', background: 'rgba(153, 102, 255, 0.2)' },
            { border: 'rgb(255, 159, 64)', background: 'rgba(255, 159, 64, 0.2)' }
        ];
        
        // 基于设备ID生成颜色索引
        const hash = deviceId.split('').reduce((a, b) => {
            a = ((a << 5) - a) + b.charCodeAt(0);
            return a & a;
        }, 0);
        
        return colors[Math.abs(hash) % colors.length];
    }
    
    mergeConfig(target, source) {
        const result = { ...target };
        
        for (const key in source) {
            if (source[key] && typeof source[key] === 'object' && !Array.isArray(source[key])) {
                result[key] = this.mergeConfig(target[key] || {}, source[key]);
            } else {
                result[key] = source[key];
            }
        }
        
        return result;
    }
}

// 饼图类
class StatusPieChart {
    constructor(canvasId, options = {}) {
        this.canvasId = canvasId;
        this.canvas = document.getElementById(canvasId);
        this.ctx = this.canvas.getContext('2d');
        
        this.options = {
            maintainAspectRatio: false,
            responsive: true,
            ...options
        };
        
        this.chart = null;
        this.statusData = new Map();
        
        this.initChart();
    }
    
    initChart() {
        const config = {
            type: 'doughnut',
            data: {
                labels: [],
                datasets: [{
                    data: [],
                    backgroundColor: [
                        'rgba(16, 185, 129, 0.8)', // 在线 - 绿色
                        'rgba(239, 68, 68, 0.8)',  // 离线 - 红色
                        'rgba(245, 158, 11, 0.8)', // 错误 - 橙色
                        'rgba(107, 114, 128, 0.8)' // 维护 - 灰色
                    ],
                    borderColor: [
                        'rgb(16, 185, 129)',
                        'rgb(239, 68, 68)',
                        'rgb(245, 158, 11)',
                        'rgb(107, 114, 128)'
                    ],
                    borderWidth: 2
                }]
            },
            options: {
                responsive: this.options.responsive,
                maintainAspectRatio: this.options.maintainAspectRatio,
                plugins: {
                    legend: {
                        position: 'bottom'
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const label = context.label || '';
                                const value = context.parsed;
                                const total = context.dataset.data.reduce((a, b) => a + b, 0);
                                const percentage = ((value / total) * 100).toFixed(1);
                                return `${label}: ${value} (${percentage}%)`;
                            }
                        }
                    }
                }
            }
        };
        
        this.chart = new Chart(this.ctx, config);
    }
    
    // 更新状态数据
    updateStatus(status, count) {
        this.statusData.set(status, count);
        this.refreshChart();
    }
    
    // 批量更新状态
    updateAllStatus(statusMap) {
        this.statusData.clear();
        for (const [status, count] of Object.entries(statusMap)) {
            this.statusData.set(status, count);
        }
        this.refreshChart();
    }
    
    // 刷新图表
    refreshChart() {
        if (!this.chart) return;
        
        const labels = Array.from(this.statusData.keys());
        const data = Array.from(this.statusData.values());
        
        this.chart.data.labels = labels;
        this.chart.data.datasets[0].data = data;
        
        this.chart.update();
    }
    
    // 销毁图表
    destroy() {
        if (this.chart) {
            this.chart.destroy();
            this.chart = null;
        }
    }
}

// 图表管理器
class ChartManager {
    constructor() {
        this.charts = new Map();
        this.isInitialized = false;
    }
    
    // 初始化所有图表
    init() {
        if (this.isInitialized) return;
        
        // 温度图表
        this.charts.set('temperature', new RealtimeChart('temperature-chart', {
            chartConfig: {
                options: {
                    scales: {
                        y: {
                            title: {
                                display: true,
                                text: '温度 (°C)'
                            }
                        }
                    }
                }
            }
        }));
        
        // 湿度图表
        this.charts.set('humidity', new RealtimeChart('humidity-chart', {
            chartConfig: {
                options: {
                    scales: {
                        y: {
                            title: {
                                display: true,
                                text: '湿度 (%)'
                            },
                            min: 0,
                            max: 100
                        }
                    }
                }
            }
        }));
        
        // 压力图表
        this.charts.set('pressure', new RealtimeChart('pressure-chart', {
            chartConfig: {
                options: {
                    scales: {
                        y: {
                            title: {
                                display: true,
                                text: '压力 (hPa)'
                            }
                        }
                    }
                }
            }
        }));
        
        // 设备状态饼图
        this.charts.set('status', new StatusPieChart('status-chart'));
        
        this.isInitialized = true;
        console.log('图表管理器初始化完成');
    }
    
    // 获取图表
    getChart(name) {
        return this.charts.get(name);
    }
    
    // 更新设备数据
    updateDeviceData(deviceData) {
        if (!this.isInitialized) return;
        
        const timestamp = new Date(deviceData.timestamp);
        const deviceId = deviceData.device_id;
        
        // 更新传感器数据
        if (deviceData.sensors) {
            if (deviceData.sensors.temperature) {
                const tempChart = this.getChart('temperature');
                tempChart?.addDataPoint(deviceId, deviceData.sensors.temperature.value, timestamp);
            }
            
            if (deviceData.sensors.humidity) {
                const humidityChart = this.getChart('humidity');
                humidityChart?.addDataPoint(deviceId, deviceData.sensors.humidity.value, timestamp);
            }
            
            if (deviceData.sensors.pressure) {
                const pressureChart = this.getChart('pressure');
                pressureChart?.addDataPoint(deviceId, deviceData.sensors.pressure.value, timestamp);
            }
        }
    }
    
    // 更新设备状态统计
    updateStatusStats(statusStats) {
        const statusChart = this.getChart('status');
        statusChart?.updateAllStatus(statusStats);
    }
    
    // 设置阈值
    setThreshold(chartName, value, label, color) {
        const chart = this.getChart(chartName);
        chart?.setThreshold(value, label, color);
    }
    
    // 清空所有数据
    clearAllData() {
        this.charts.forEach(chart => {
            if (chart.clearData) {
                chart.clearData();
            }
        });
    }
    
    // 调整所有图表大小
    resizeAll() {
        this.charts.forEach(chart => {
            if (chart.resize) {
                chart.resize();
            }
        });
    }
    
    // 销毁所有图表
    destroyAll() {
        this.charts.forEach(chart => {
            chart.destroy();
        });
        this.charts.clear();
        this.isInitialized = false;
    }
}

// 导出类
window.RealtimeChart = RealtimeChart;
window.StatusPieChart = StatusPieChart;
window.ChartManager = ChartManager;
