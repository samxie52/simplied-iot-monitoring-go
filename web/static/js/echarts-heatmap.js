// 方案三：热力图 + 表格组合
// 热力图显示所有设备状态概览，表格显示详细数据

class HeatmapTableChart {
    constructor(canvasId, options = {}) {
        this.canvasId = canvasId;
        this.container = document.getElementById(canvasId);
        
        if (!this.container) {
            console.error('图表容器未找到:', canvasId);
            return;
        }
        
        this.options = {
            maxTimeSlots: 20, // 最多显示20个时间段
            timeSlotDuration: 30000, // 30秒一个时间段
            ...options
        };
        
        this.chart = null;
        this.deviceData = new Map(); // 存储设备数据
        this.timeSlots = []; // 时间段数组
        this.heatmapData = []; // 热力图数据
        
        this.createLayout();
        this.initChart();
        this.initTable();
    }
    
    // 创建布局
    createLayout() {
        // 创建主容器
        const mainContainer = document.createElement('div');
        mainContainer.style.cssText = `
            display: flex;
            flex-direction: column;
            height: 100%;
        `;
        
        // 创建热力图容器
        this.heatmapContainer = document.createElement('div');
        this.heatmapContainer.id = `${this.canvasId}-heatmap`;
        this.heatmapContainer.style.cssText = `
            height: 60%;
            min-height: 200px;
        `;
        
        // 创建表格容器
        this.tableContainer = document.createElement('div');
        this.tableContainer.style.cssText = `
            height: 40%;
            overflow-y: auto;
            margin-top: 10px;
            border: 1px solid #e9ecef;
            border-radius: 4px;
        `;
        
        mainContainer.appendChild(this.heatmapContainer);
        mainContainer.appendChild(this.tableContainer);
        
        // 替换原容器内容
        this.container.innerHTML = '';
        this.container.appendChild(mainContainer);
    }
    
    initChart() {
        if (!this.heatmapContainer || typeof echarts === 'undefined') {
            console.error('ECharts未加载或容器不存在');
            return;
        }
        
        this.chart = echarts.init(this.heatmapContainer);
        
        const option = {
            title: {
                text: this.options.title || '设备状态热力图',
                left: 'center',
                textStyle: { color: '#333', fontSize: 16 }
            },
            tooltip: {
                position: 'top',
                formatter: function(params) {
                    const deviceId = params.data[1];
                    const timeSlot = params.data[0];
                    const value = params.data[2];
                    return `设备: ${deviceId}<br/>时间: ${this.timeSlots[timeSlot] || '未知'}<br/>数值: ${value !== null ? value.toFixed(2) : '无数据'} ${this.options.unit || ''}`;
                }.bind(this)
            },
            grid: {
                height: '70%',
                top: '15%'
            },
            xAxis: {
                type: 'category',
                data: [],
                splitArea: {
                    show: true
                },
                axisLabel: {
                    formatter: function(value, index) {
                        // 显示时间
                        const time = new Date(parseInt(value));
                        return time.getHours().toString().padStart(2, '0') + ':' + 
                               time.getMinutes().toString().padStart(2, '0');
                    }
                }
            },
            yAxis: {
                type: 'category',
                data: [],
                splitArea: {
                    show: true
                }
            },
            visualMap: {
                min: 0,
                max: 100,
                calculable: true,
                orient: 'horizontal',
                left: 'center',
                bottom: '5%',
                inRange: {
                    color: ['#313695', '#4575b4', '#74add1', '#abd9e9', '#e0f3f8', 
                           '#ffffcc', '#fee090', '#fdae61', '#f46d43', '#d73027', '#a50026']
                }
            },
            series: [{
                name: '设备数值',
                type: 'heatmap',
                data: [],
                label: {
                    show: false
                },
                emphasis: {
                    itemStyle: {
                        shadowBlur: 10,
                        shadowColor: 'rgba(0, 0, 0, 0.5)'
                    }
                }
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
    
    // 初始化表格
    initTable() {
        const table = document.createElement('table');
        table.style.cssText = `
            width: 100%;
            border-collapse: collapse;
            font-size: 12px;
        `;
        
        // 创建表头
        const thead = document.createElement('thead');
        thead.innerHTML = `
            <tr style="background: #f8f9fa;">
                <th style="padding: 8px; border: 1px solid #dee2e6; text-align: left;">设备ID</th>
                <th style="padding: 8px; border: 1px solid #dee2e6; text-align: center;">当前值</th>
                <th style="padding: 8px; border: 1px solid #dee2e6; text-align: center;">最后更新</th>
                <th style="padding: 8px; border: 1px solid #dee2e6; text-align: center;">状态</th>
                <th style="padding: 8px; border: 1px solid #dee2e6; text-align: center;">趋势</th>
            </tr>
        `;
        
        // 创建表体
        this.tbody = document.createElement('tbody');
        
        table.appendChild(thead);
        table.appendChild(this.tbody);
        this.tableContainer.appendChild(table);
    }
    
    // 添加数据点
    addDataPoint(deviceId, value, timestamp = new Date()) {
        // 初始化设备数据
        if (!this.deviceData.has(deviceId)) {
            this.deviceData.set(deviceId, {
                values: [],
                lastValue: null,
                lastUpdate: null,
                trend: 'stable'
            });
        }
        
        const device = this.deviceData.get(deviceId);
        
        // 计算趋势
        if (device.lastValue !== null) {
            const diff = value - device.lastValue;
            if (diff > 0.1) device.trend = 'up';
            else if (diff < -0.1) device.trend = 'down';
            else device.trend = 'stable';
        }
        
        device.values.push({ value, timestamp: timestamp.getTime() });
        device.lastValue = value;
        device.lastUpdate = timestamp;
        
        // 限制历史数据
        if (device.values.length > 100) {
            device.values.shift();
        }
        
        this.updateHeatmapData();
        this.updateTable();
    }
    
    // 更新热力图数据
    updateHeatmapData() {
        // 生成时间段
        const now = Date.now();
        const slotDuration = this.options.timeSlotDuration;
        
        // 更新时间段数组
        this.timeSlots = [];
        for (let i = this.options.maxTimeSlots - 1; i >= 0; i--) {
            this.timeSlots.push(new Date(now - i * slotDuration).toISOString());
        }
        
        // 获取所有设备ID
        const deviceIds = Array.from(this.deviceData.keys()).sort();
        
        // 生成热力图数据
        this.heatmapData = [];
        
        deviceIds.forEach((deviceId, deviceIndex) => {
            const device = this.deviceData.get(deviceId);
            
            this.timeSlots.forEach((timeSlot, timeIndex) => {
                const slotStart = new Date(timeSlot).getTime();
                const slotEnd = slotStart + slotDuration;
                
                // 查找该时间段内的数据
                const slotValues = device.values.filter(item => 
                    item.timestamp >= slotStart && item.timestamp < slotEnd
                );
                
                let avgValue = null;
                if (slotValues.length > 0) {
                    avgValue = slotValues.reduce((sum, item) => sum + item.value, 0) / slotValues.length;
                }
                
                this.heatmapData.push([timeIndex, deviceIndex, avgValue]);
            });
        });
        
        // 更新图表
        if (this.chart) {
            // 计算数值范围用于颜色映射
            const values = this.heatmapData.map(item => item[2]).filter(v => v !== null);
            const minValue = values.length > 0 ? Math.min(...values) : 0;
            const maxValue = values.length > 0 ? Math.max(...values) : 100;
            
            this.chart.setOption({
                xAxis: {
                    data: this.timeSlots.map(slot => new Date(slot).getTime())
                },
                yAxis: {
                    data: deviceIds
                },
                visualMap: {
                    min: minValue,
                    max: maxValue
                },
                series: [{
                    data: this.heatmapData
                }]
            });
        }
    }
    
    // 更新表格
    updateTable() {
        if (!this.tbody) return;
        
        this.tbody.innerHTML = '';
        
        // 按设备ID排序
        const sortedDevices = Array.from(this.deviceData.entries()).sort((a, b) => a[0].localeCompare(b[0]));
        
        sortedDevices.forEach(([deviceId, device]) => {
            const row = document.createElement('tr');
            row.style.cssText = 'border-bottom: 1px solid #dee2e6;';
            
            // 趋势图标
            const trendIcon = device.trend === 'up' ? '↗️' : 
                             device.trend === 'down' ? '↘️' : '➡️';
            const trendColor = device.trend === 'up' ? '#52c41a' : 
                              device.trend === 'down' ? '#f5222d' : '#1890ff';
            
            // 状态颜色
            const isOnline = device.lastUpdate && (Date.now() - device.lastUpdate.getTime()) < 60000;
            const statusColor = isOnline ? '#52c41a' : '#f5222d';
            const statusText = isOnline ? '在线' : '离线';
            
            row.innerHTML = `
                <td style="padding: 8px; border: 1px solid #dee2e6; font-weight: bold;">${deviceId}</td>
                <td style="padding: 8px; border: 1px solid #dee2e6; text-align: center;">
                    ${device.lastValue !== null ? device.lastValue.toFixed(2) : '--'} ${this.options.unit || ''}
                </td>
                <td style="padding: 8px; border: 1px solid #dee2e6; text-align: center; font-size: 11px;">
                    ${device.lastUpdate ? device.lastUpdate.toLocaleTimeString() : '--'}
                </td>
                <td style="padding: 8px; border: 1px solid #dee2e6; text-align: center;">
                    <span style="color: ${statusColor}; font-weight: bold;">${statusText}</span>
                </td>
                <td style="padding: 8px; border: 1px solid #dee2e6; text-align: center;">
                    <span style="color: ${trendColor}; font-size: 16px;">${trendIcon}</span>
                </td>
            `;
            
            this.tbody.appendChild(row);
        });
    }
    
    // 清空数据
    clearData() {
        this.deviceData.clear();
        this.timeSlots = [];
        this.heatmapData = [];
        if (this.chart) {
            this.chart.setOption({
                xAxis: { data: [] },
                yAxis: { data: [] },
                series: [{ data: [] }]
            });
        }
        if (this.tbody) {
            this.tbody.innerHTML = '';
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

// 导出类
window.HeatmapTableChart = HeatmapTableChart;
