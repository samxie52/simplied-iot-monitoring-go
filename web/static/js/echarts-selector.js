// 方案二：设备选择器图表
// 用户可以选择要显示的设备，最多显示5个设备

class SelectableDeviceChart {
    constructor(canvasId, options = {}) {
        this.canvasId = canvasId;
        this.container = document.getElementById(canvasId);
        
        if (!this.container) {
            console.error('图表容器未找到:', canvasId);
            return;
        }
        
        this.options = {
            maxDataPoints: 50,
            maxSelectedDevices: 5,
            updateInterval: 2000,
            ...options
        };
        
        this.chart = null;
        this.allDevices = new Map(); // 存储所有设备数据
        this.selectedDevices = new Set(); // 当前选中的设备
        this.deviceColors = new Map(); // 设备颜色映射
        this.colorPalette = [
            '#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1',
            '#13c2c2', '#eb2f96', '#fa8c16', '#a0d911', '#2f54eb'
        ];
        
        this.createDeviceSelector();
        this.initChart();
    }
    
    // 创建设备选择器UI
    createDeviceSelector() {
        // 创建选择器容器
        const selectorContainer = document.createElement('div');
        selectorContainer.className = 'device-selector';
        selectorContainer.style.cssText = `
            margin-bottom: 10px;
            padding: 10px;
            background: #f8f9fa;
            border-radius: 4px;
            border: 1px solid #e9ecef;
        `;
        
        // 创建标题
        const title = document.createElement('div');
        title.textContent = '选择设备 (最多5个):';
        title.style.cssText = `
            font-weight: bold;
            margin-bottom: 8px;
            color: #333;
            font-size: 14px;
        `;
        
        // 创建设备列表容器
        this.deviceListContainer = document.createElement('div');
        this.deviceListContainer.className = 'device-list';
        this.deviceListContainer.style.cssText = `
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
        `;
        
        // 创建"全选"和"清空"按钮
        const buttonContainer = document.createElement('div');
        buttonContainer.style.cssText = `
            margin-top: 8px;
            display: flex;
            gap: 8px;
        `;
        
        const selectAllBtn = document.createElement('button');
        selectAllBtn.textContent = '选择前5个';
        selectAllBtn.style.cssText = `
            padding: 4px 8px;
            background: #1890ff;
            color: white;
            border: none;
            border-radius: 3px;
            cursor: pointer;
            font-size: 12px;
        `;
        selectAllBtn.onclick = () => this.selectTopDevices();
        
        const clearBtn = document.createElement('button');
        clearBtn.textContent = '清空选择';
        clearBtn.style.cssText = `
            padding: 4px 8px;
            background: #f5222d;
            color: white;
            border: none;
            border-radius: 3px;
            cursor: pointer;
            font-size: 12px;
        `;
        clearBtn.onclick = () => this.clearSelection();
        
        buttonContainer.appendChild(selectAllBtn);
        buttonContainer.appendChild(clearBtn);
        
        selectorContainer.appendChild(title);
        selectorContainer.appendChild(this.deviceListContainer);
        selectorContainer.appendChild(buttonContainer);
        
        // 插入到图表容器前面
        this.container.parentNode.insertBefore(selectorContainer, this.container);
    }
    
    // 更新设备选择器
    updateDeviceSelector() {
        // 清空现有选项
        this.deviceListContainer.innerHTML = '';
        
        // 按设备ID排序
        const sortedDevices = Array.from(this.allDevices.keys()).sort();
        
        sortedDevices.forEach(deviceId => {
            const checkbox = document.createElement('input');
            checkbox.type = 'checkbox';
            checkbox.id = `device-${deviceId}`;
            checkbox.checked = this.selectedDevices.has(deviceId);
            checkbox.onchange = (e) => this.toggleDevice(deviceId, e.target.checked);
            
            const label = document.createElement('label');
            label.htmlFor = `device-${deviceId}`;
            label.textContent = deviceId;
            label.style.cssText = `
                display: flex;
                align-items: center;
                gap: 4px;
                padding: 4px 8px;
                background: ${this.selectedDevices.has(deviceId) ? '#e6f7ff' : '#fff'};
                border: 1px solid ${this.selectedDevices.has(deviceId) ? '#1890ff' : '#d9d9d9'};
                border-radius: 3px;
                cursor: pointer;
                font-size: 12px;
                user-select: none;
            `;
            
            // 添加颜色指示器
            if (this.selectedDevices.has(deviceId)) {
                const colorIndicator = document.createElement('span');
                colorIndicator.style.cssText = `
                    width: 12px;
                    height: 12px;
                    background: ${this.deviceColors.get(deviceId)};
                    border-radius: 50%;
                    display: inline-block;
                `;
                label.appendChild(colorIndicator);
            }
            
            label.appendChild(checkbox);
            this.deviceListContainer.appendChild(label);
        });
    }
    
    // 切换设备选择状态
    toggleDevice(deviceId, selected) {
        if (selected) {
            if (this.selectedDevices.size >= this.options.maxSelectedDevices) {
                alert(`最多只能选择 ${this.options.maxSelectedDevices} 个设备`);
                return;
            }
            this.selectedDevices.add(deviceId);
            // 分配颜色
            if (!this.deviceColors.has(deviceId)) {
                const colorIndex = this.deviceColors.size % this.colorPalette.length;
                this.deviceColors.set(deviceId, this.colorPalette[colorIndex]);
            }
        } else {
            this.selectedDevices.delete(deviceId);
        }
        
        this.updateDeviceSelector();
        this.updateChart();
    }
    
    // 选择前5个设备
    selectTopDevices() {
        this.clearSelection();
        const devices = Array.from(this.allDevices.keys()).sort().slice(0, this.options.maxSelectedDevices);
        devices.forEach(deviceId => {
            this.selectedDevices.add(deviceId);
            if (!this.deviceColors.has(deviceId)) {
                const colorIndex = this.deviceColors.size % this.colorPalette.length;
                this.deviceColors.set(deviceId, this.colorPalette[colorIndex]);
            }
        });
        this.updateDeviceSelector();
        this.updateChart();
    }
    
    // 清空选择
    clearSelection() {
        this.selectedDevices.clear();
        this.updateDeviceSelector();
        this.updateChart();
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
                text: this.options.title || '选定设备趋势',
                left: 'center',
                textStyle: { color: '#333', fontSize: 16 }
            },
            tooltip: {
                trigger: 'axis',
                axisPointer: { type: 'cross' }
            },
            legend: {
                data: [],
                top: 30,
                type: 'scroll'
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
            yAxis: {
                type: 'value',
                name: this.options.yAxisName || '数值',
                splitLine: { lineStyle: { color: '#f0f0f0' }}
            },
            series: []
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
        // 初始化设备数据
        if (!this.allDevices.has(deviceId)) {
            this.allDevices.set(deviceId, []);
            this.updateDeviceSelector();
        }
        
        const deviceData = this.allDevices.get(deviceId);
        deviceData.push([timestamp, value]);
        
        // 限制数据点数量
        if (deviceData.length > this.options.maxDataPoints) {
            deviceData.shift();
        }
        
        // 如果设备被选中，更新图表
        if (this.selectedDevices.has(deviceId)) {
            this.updateChart();
        }
    }
    
    // 更新图表
    updateChart() {
        if (!this.chart) return;
        
        const series = [];
        const legendData = [];
        
        this.selectedDevices.forEach(deviceId => {
            if (this.allDevices.has(deviceId)) {
                const deviceData = this.allDevices.get(deviceId);
                series.push({
                    name: deviceId,
                    type: 'line',
                    data: deviceData,
                    smooth: true,
                    lineStyle: { width: 2 },
                    itemStyle: { color: this.deviceColors.get(deviceId) }
                });
                legendData.push(deviceId);
            }
        });
        
        this.chart.setOption({
            legend: { data: legendData },
            series: series
        });
    }
    
    // 清空数据
    clearData() {
        this.allDevices.clear();
        this.selectedDevices.clear();
        this.deviceColors.clear();
        this.updateDeviceSelector();
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

// 导出类
window.SelectableDeviceChart = SelectableDeviceChart;
