// 基于ECharts的实时图表管理类
// Industrial IoT Real-time Charts with ECharts

class RealtimeChart {
    constructor(canvasId, options = {}) {
        this.canvasId = canvasId;
        this.container = document.getElementById(canvasId);
        
        if (!this.container) {
            console.error('图表容器未找到:', canvasId);
            return;
        }
        
        this.options = {
            maxDataPoints: 50,
            updateInterval: 1000,
            ...options
        };
        
        this.chart = null;
        this.deviceData = new Map(); // 存储每个设备的数据
        this.isUpdating = false;
        
        this.initChart();
    }
    
    initChart() {
        if (!this.container || typeof echarts === 'undefined') {
            console.error('ECharts未加载或容器不存在');
            return;
        }
        
        // 确保容器有正确的大小
        if (this.container.offsetWidth === 0 || this.container.offsetHeight === 0) {
            console.warn('图表容器大小为0，尝试设置默认大小:', this.canvasId);
            this.container.style.width = '100%';
            this.container.style.height = '300px';
        }
        
        // 初始化ECharts实例
        this.chart = echarts.init(this.container);
        
        // 立即调整大小
        setTimeout(() => {
            if (this.chart) {
                this.chart.resize();
            }
        }, 100);
        
        const defaultOption = {
            title: {
                text: this.options.title || '实时数据',
                left: 'center',
                textStyle: {
                    color: '#333',
                    fontSize: 16
                }
            },
            tooltip: {
                trigger: 'axis',
                axisPointer: {
                    type: 'cross'
                },
                formatter: function(params) {
                    let result = params[0].axisValueLabel + '<br/>';
                    params.forEach(param => {
                        result += `${param.seriesName}: ${param.value[1]}<br/>`;
                    });
                    return result;
                }
            },
            legend: {
                top: 30,
                textStyle: {
                    color: '#666'
                }
            },
            grid: {
                left: '3%',
                right: '4%',
                bottom: '3%',
                top: '15%',
                containLabel: true
            },
            xAxis: {
                type: 'time',
                splitLine: {
                    show: false
                },
                axisLabel: {
                    formatter: function(value) {
                        return new Date(value).toLocaleTimeString();
                    }
                }
            },
            yAxis: {
                type: 'value',
                name: this.options.yAxisName || '数值',
                splitLine: {
                    lineStyle: {
                        color: '#f0f0f0'
                    }
                }
            },
            series: []
        };
        
        // 合并用户配置
        const option = this.mergeConfig(defaultOption, this.options.chartConfig || {});
        this.chart.setOption(option);
        
        // 响应式处理
        window.addEventListener('resize', () => {
            if (this.chart) {
                this.chart.resize();
            }
        });
    }
    
    // 添加数据点
    addDataPoint(deviceId, value, timestamp = new Date()) {
        if (!this.chart) return;
        
        // 获取或创建设备数据系列
        if (!this.deviceData.has(deviceId)) {
            this.deviceData.set(deviceId, {
                name: deviceId,
                type: 'line',
                smooth: true,
                data: [],
                lineStyle: {
                    width: 2
                },
                itemStyle: {
                    color: this.getDeviceColor(deviceId)
                }
            });
        }
        
        const deviceSeries = this.deviceData.get(deviceId);
        
        // 添加数据点 [时间戳, 数值]
        deviceSeries.data.push([timestamp.getTime(), value]);
        
        // 限制数据点数量
        if (deviceSeries.data.length > this.options.maxDataPoints) {
            deviceSeries.data.shift();
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
    
    // 更新图表
    updateChart() {
        if (!this.chart || this.isUpdating) return;
        
        this.isUpdating = true;
        
        const option = {
            series: Array.from(this.deviceData.values())
        };
        
        this.chart.setOption(option);
        this.isUpdating = false;
    }
    
    // 清空数据
    clearData() {
        this.deviceData.clear();
        if (this.chart) {
            this.chart.setOption({
                series: []
            });
        }
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
            this.chart.dispose();
            this.chart = null;
        }
    }
    
    // 获取设备颜色
    getDeviceColor(deviceId) {
        const colors = [
            '#5470c6', '#91cc75', '#fac858', '#ee6666', '#73c0de',
            '#3ba272', '#fc8452', '#9a60b4', '#ea7ccc', '#5470c6'
        ];
        
        // 根据设备ID生成一致的颜色
        let hash = 0;
        for (let i = 0; i < deviceId.length; i++) {
            hash = deviceId.charCodeAt(i) + ((hash << 5) - hash);
        }
        return colors[Math.abs(hash) % colors.length];
    }
    
    // 合并配置
    mergeConfig(target, source) {
        const result = { ...target };
        for (const key in source) {
            if (source.hasOwnProperty(key)) {
                if (typeof source[key] === 'object' && !Array.isArray(source[key])) {
                    result[key] = this.mergeConfig(result[key] || {}, source[key]);
                } else {
                    result[key] = source[key];
                }
            }
        }
        return result;
    }
}

// 饼图类
class StatusPieChart {
    constructor(canvasId, options = {}) {
        this.canvasId = canvasId;
        this.container = document.getElementById(canvasId);
        
        if (!this.container) {
            console.error('饼图容器未找到:', canvasId);
            return;
        }
        
        this.options = {
            ...options
        };
        
        this.chart = null;
        this.statusData = {};
        
        this.initChart();
    }
    
    initChart() {
        if (!this.container || typeof echarts === 'undefined') {
            console.error('ECharts未加载或容器不存在');
            return;
        }
        
        // 确保容器有正确的大小
        if (this.container.offsetWidth === 0 || this.container.offsetHeight === 0) {
            console.warn('饼图容器大小为0，尝试设置默认大小:', this.canvasId);
            this.container.style.width = '100%';
            this.container.style.height = '300px';
        }
        
        this.chart = echarts.init(this.container);
        
        // 立即调整大小
        setTimeout(() => {
            if (this.chart) {
                this.chart.resize();
            }
        }, 100);
        
        const option = {
            title: {
                text: '设备状态分布',
                left: 'center',
                textStyle: {
                    color: '#333',
                    fontSize: 16
                }
            },
            tooltip: {
                trigger: 'item',
                formatter: '{a} <br/>{b}: {c} ({d}%)'
            },
            legend: {
                orient: 'vertical',
                left: 'left',
                top: 'middle',
                textStyle: {
                    color: '#666'
                }
            },
            series: [{
                name: '设备状态',
                type: 'pie',
                radius: ['40%', '70%'],
                center: ['60%', '50%'],
                avoidLabelOverlap: false,
                itemStyle: {
                    borderRadius: 10,
                    borderColor: '#fff',
                    borderWidth: 2
                },
                label: {
                    show: false,
                    position: 'center'
                },
                emphasis: {
                    label: {
                        show: true,
                        fontSize: '18',
                        fontWeight: 'bold'
                    }
                },
                labelLine: {
                    show: false
                },
                data: []
            }]
        };
        
        this.chart.setOption(option);
        
        // 响应式处理
        window.addEventListener('resize', () => {
            if (this.chart) {
                this.chart.resize();
            }
        });
    }
    
    // 更新状态数据
    updateStatus(status, count) {
        this.statusData[status] = count;
        this.refreshChart();
    }
    
    // 批量更新状态
    updateAllStatus(statusMap) {
        this.statusData = { ...statusMap };
        this.refreshChart();
    }
    
    // 刷新图表
    refreshChart() {
        if (!this.chart) return;
        
        const data = Object.entries(this.statusData).map(([status, count]) => ({
            name: status,
            value: count,
            itemStyle: {
                color: this.getStatusColor(status)
            }
        }));
        
        this.chart.setOption({
            series: [{
                data: data
            }]
        });
    }
    
    // 获取状态颜色
    getStatusColor(status) {
        const statusColors = {
            'online': '#52c41a',
            'offline': '#ff4d4f',
            'warning': '#faad14',
            'normal': '#1890ff'
        };
        return statusColors[status] || '#d9d9d9';
    }
    
    // 销毁图表
    destroy() {
        if (this.chart) {
            this.chart.dispose();
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
            title: '温度趋势',
            yAxisName: '温度 (°C)',
            chartConfig: {
                yAxis: {
                    min: function(value) {
                        return Math.floor(value.min - 5);
                    },
                    max: function(value) {
                        return Math.ceil(value.max + 5);
                    }
                }
            }
        }));
        
        // 湿度图表
        this.charts.set('humidity', new RealtimeChart('humidity-chart', {
            title: '湿度趋势',
            yAxisName: '湿度 (%)',
            chartConfig: {
                yAxis: {
                    min: 0,
                    max: 100
                }
            }
        }));
        
        // 压力图表
        this.charts.set('pressure', new RealtimeChart('pressure-chart', {
            title: '压力趋势',
            yAxisName: '压力 (hPa)',
            chartConfig: {
                yAxis: {
                    min: function(value) {
                        return Math.floor(value.min - 10);
                    },
                    max: function(value) {
                        return Math.ceil(value.max + 10);
                    }
                }
            }
        }));
        
        // 设备状态饼图
        this.charts.set('status', new StatusPieChart('status-chart'));
        
        this.isInitialized = true;
        
        // 添加全局resize事件监听
        window.addEventListener('resize', () => {
            this.resizeAll();
        });
        
        // 初始化后立即调整所有图表大小
        setTimeout(() => {
            this.resizeAll();
        }, 200);
        
        console.log('ECharts图表管理器初始化完成');
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
