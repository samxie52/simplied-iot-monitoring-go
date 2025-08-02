// 方案一：聚合统计图表
// 显示平均值、最大值、最小值而不是每个设备的单独线条

class AggregatedChart {
    constructor(canvasId, options = {}) {
        this.canvasId = canvasId;
        this.container = document.getElementById(canvasId);
        
        if (!this.container) {
            console.error('图表容器未找到:', canvasId);
            return;
        }
        
        this.options = {
            maxDataPoints: 50,
            updateInterval: 2000,
            ...options
        };
        
        this.chart = null;
        this.dataBuffer = []; // 存储原始数据用于聚合
        this.aggregatedData = {
            avg: { name: '平均值', type: 'line', data: [], smooth: true, lineStyle: { width: 3 }, itemStyle: { color: '#1890ff' }},
            max: { name: '最大值', type: 'line', data: [], smooth: true, lineStyle: { width: 2 }, itemStyle: { color: '#ff4d4f' }},
            min: { name: '最小值', type: 'line', data: [], smooth: true, lineStyle: { width: 2 }, itemStyle: { color: '#52c41a' }},
            count: { name: '设备数量', type: 'bar', data: [], yAxisIndex: 1, itemStyle: { color: '#faad14', opacity: 0.6 }}
        };
        
        this.initChart();
    }
    
    initChart() {
        if (!this.container || typeof echarts === 'undefined') {
            console.error('ECharts未加载或容器不存在');
            return;
        }
        
        // 确保容器有正确的大小
        if (this.container.offsetWidth === 0 || this.container.offsetHeight === 0) {
            this.container.style.width = '100%';
            this.container.style.height = '300px';
        }
        
        this.chart = echarts.init(this.container);
        
        const option = {
            title: {
                text: this.options.title || '聚合数据趋势',
                left: 'center',
                textStyle: { color: '#333', fontSize: 16 }
            },
            tooltip: {
                trigger: 'axis',
                axisPointer: { 
                    type: 'cross',
                    lineStyle: {
                        color: '#999',
                        width: 1,
                        type: 'dashed'
                    }
                },
                backgroundColor: 'rgba(255, 255, 255, 0.95)',
                borderColor: '#ccc',
                borderWidth: 1,
                textStyle: {
                    color: '#333',
                    fontSize: 13
                },
                formatter: function(params) {
                    if (!params || params.length === 0) return '';
                    
                    // 获取时间标签
                    const timeLabel = params[0].axisValueLabel;
                    
                    // 分类数据
                    let avgValue = null, maxValue = null, minValue = null, deviceCount = null;
                    
                    params.forEach(param => {
                        switch(param.seriesName) {
                            case '平均值':
                                avgValue = param.value[1];
                                break;
                            case '最大值':
                                maxValue = param.value[1];
                                break;
                            case '最小值':
                                minValue = param.value[1];
                                break;
                            case '设备数量':
                                deviceCount = param.value[1];
                                break;
                        }
                    });
                    
                    // 构建显示内容 - 超紧凑版本
                    let result = `<div style="background: rgba(255,255,255,0.95); border: 1px solid #ccc; border-radius: 3px; padding: 4px; box-shadow: 0 1px 4px rgba(0,0,0,0.1); font-size: 11px; line-height: 1.1; max-width: 180px;">`;
                    result += `<div style="height: 20px; font-weight: bold; margin-bottom: 2px; color: #333; font-size: 12px;">${timeLabel}</div>`;
                    result += `<div style="height: 20px; margin: 0; color: #666;">📊 ${deviceCount}个设备</div>`;
                    result += `<div style="height: 20px; margin: 0; color: #666;">📈 ${avgValue.toFixed(1)}${this.options.unit}</div>`;
                    result += `<div style="height: 20px; margin: 0; color: #666;">⬆️ ${maxValue.toFixed(1)}${this.options.unit} ⬇️ ${minValue.toFixed(1)}${this.options.unit}</div>`;
                    
                    if (maxValue !== minValue) {
                        const range = (maxValue - minValue).toFixed(1);
                        result += `<div style="margin: 0; color: #666; font-size: 10px;">📉 范围${range}${this.options.unit}</div>`;
                    }
                    
                    result += `</div></div>`;
                    return result;
                }.bind(this)
            },
            legend: {
                data: ['平均值', '最大值', '最小值', '设备数量'],
                top: 30
            },
            grid: {
                left: '3%',
                right: '4%',
                bottom: '3%',
                containLabel: true
            },
            xAxis: {
                type: 'time',
                splitLine: { show: false }
            },
            yAxis: [
                {
                    type: 'value',
                    name: this.options.yAxisName || '数值',
                    position: 'left',
                    splitLine: { lineStyle: { color: '#f0f0f0' }}
                },
                {
                    type: 'value',
                    name: '设备数量',
                    position: 'right',
                    splitLine: { show: false }
                }
            ],
            series: [
                this.aggregatedData.avg,
                this.aggregatedData.max,
                this.aggregatedData.min,
                this.aggregatedData.count
            ]
        };
        
        this.chart.setOption(option);
        
        // 延迟调整大小
        setTimeout(() => {
            if (this.chart) {
                this.chart.resize();
            }
        }, 100);
    }
    
    // 添加数据点
    addDataPoint(deviceId, value, timestamp = new Date()) {
        // 将数据添加到缓冲区
        this.dataBuffer.push({
            deviceId,
            value,
            timestamp: timestamp.getTime()
        });
        
        // 清理过期数据（保留最近5分钟的数据用于聚合）
        const fiveMinutesAgo = Date.now() - 5 * 60 * 1000;
        this.dataBuffer = this.dataBuffer.filter(item => item.timestamp > fiveMinutesAgo);
        
        // 每2秒聚合一次数据
        this.aggregateData(timestamp);
    }
    
    // 聚合数据
    aggregateData(timestamp) {
        if (this.dataBuffer.length === 0) return;
        
        // 按时间窗口聚合（30秒窗口）
        const windowSize = 30 * 1000;
        const currentWindow = Math.floor(timestamp.getTime() / windowSize) * windowSize;
        
        // 获取当前时间窗口的数据
        const windowData = this.dataBuffer.filter(item => 
            Math.floor(item.timestamp / windowSize) * windowSize === currentWindow
        );
        
        if (windowData.length === 0) return;
        
        // 计算统计值
        const values = windowData.map(item => item.value);
        const avg = values.reduce((sum, val) => sum + val, 0) / values.length;
        const max = Math.max(...values);
        const min = Math.min(...values);
        const count = new Set(windowData.map(item => item.deviceId)).size;
        
        // 更新聚合数据
        const timePoint = new Date(currentWindow);
        
        this.aggregatedData.avg.data.push([timePoint, avg]);
        this.aggregatedData.max.data.push([timePoint, max]);
        this.aggregatedData.min.data.push([timePoint, min]);
        this.aggregatedData.count.data.push([timePoint, count]);
        
        // 限制数据点数量
        Object.values(this.aggregatedData).forEach(series => {
            if (series.data.length > this.options.maxDataPoints) {
                series.data.shift();
            }
        });
        
        // 更新图表
        this.updateChart();
    }
    
    // 更新图表
    updateChart() {
        if (!this.chart) return;
        
        this.chart.setOption({
            series: [
                this.aggregatedData.avg,
                this.aggregatedData.max,
                this.aggregatedData.min,
                this.aggregatedData.count
            ]
        });
    }
    
    // 清空数据
    clearData() {
        this.dataBuffer = [];
        Object.values(this.aggregatedData).forEach(series => {
            series.data = [];
        });
        if (this.chart) {
            this.updateChart();
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
}

// 简单的状态饼图类
class StatusPieChart {
    constructor(containerId, options = {}) {
        this.containerId = containerId;
        this.options = {
            title: options.title || '设备状态分布',
            ...options
        };
        this.chart = null;
        this.initChart();
    }
    
    initChart() {
        const container = document.getElementById(this.containerId);
        if (!container) {
            console.warn(`容器 ${this.containerId} 不存在`);
            return;
        }
        
        this.chart = echarts.init(container);
        
        const option = {
            title: {
                text: this.options.title,
                left: 'center',
                top: 20,
                textStyle: {
                    fontSize: 16,
                    fontWeight: 'bold'
                }
            },
            tooltip: {
                trigger: 'item',
                formatter: '{a} <br/>{b}: {c} ({d}%)'
            },
            legend: {
                orient: 'vertical',
                left: 'left',
                top: 'middle'
            },
            series: [{
                name: '设备状态',
                type: 'pie',
                radius: ['40%', '70%'],
                center: ['60%', '60%'],
                avoidLabelOverlap: false,
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
                data: [
                    {value: 0, name: '在线', itemStyle: {color: '#52c41a'}},
                    {value: 0, name: '离线', itemStyle: {color: '#ff4d4f'}}
                ]
            }]
        };
        
        this.chart.setOption(option);
        
        // 延迟调整大小
        setTimeout(() => {
            if (this.chart) {
                this.chart.resize();
            }
        }, 100);
    }
    
    updateData(data) {
        if (!this.chart) return;
        
        const seriesData = [
            {value: data['在线'] || 0, name: '在线', itemStyle: {color: '#52c41a'}},
            {value: data['离线'] || 0, name: '离线', itemStyle: {color: '#ff4d4f'}}
        ];
        
        this.chart.setOption({
            series: [{
                data: seriesData
            }]
        });
    }
    
    resize() {
        if (this.chart) {
            this.chart.resize();
        }
    }
    
    destroy() {
        if (this.chart) {
            this.chart.dispose();
            this.chart = null;
        }
    }
}

// 导出类
window.AggregatedChart = AggregatedChart;
window.StatusPieChart = StatusPieChart;
