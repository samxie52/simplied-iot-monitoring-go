// Industrial IoT Monitoring System - Frontend JavaScript
// 工业IoT监控系统前端JavaScript

let ws = null;
let isConnected = false;
let deviceData = new Map();
let messageCount = 0;
let alertCount = 0;

function toggleConnection() {
    if (isConnected) {
        disconnect();
    } else {
        connect();
    }
}

function connect() {
    const wsUrl = 'ws://localhost:8080/ws';
    ws = new WebSocket(wsUrl);

    ws.onopen = function() {
        isConnected = true;
        updateConnectionStatus();
        console.log('WebSocket连接已建立');
    };

    ws.onmessage = function(event) {
        try {
            const device = JSON.parse(event.data);
            updateDeviceData(device);
            messageCount++;
            updateStats();
        } catch (error) {
            console.error('解析消息失败:', error);
        }
    };

    ws.onclose = function() {
        isConnected = false;
        updateConnectionStatus();
        console.log('WebSocket连接已关闭');
    };

    ws.onerror = function(error) {
        console.error('WebSocket错误:', error);
        isConnected = false;
        updateConnectionStatus();
    };
}

function disconnect() {
    if (ws) {
        ws.close();
    }
}

function updateConnectionStatus() {
    const statusElement = document.getElementById('ws-status');
    const buttonElement = document.getElementById('connect-btn');
    
    if (isConnected) {
        statusElement.textContent = '已连接';
        statusElement.className = 'connected';
        buttonElement.textContent = '断开连接';
    } else {
        statusElement.textContent = '未连接';
        statusElement.className = 'disconnected';
        buttonElement.textContent = '连接WebSocket';
    }
}

function updateDeviceData(device) {
    deviceData.set(device.device_id, device);
    
    // 检查告警条件
    if (device.temperature > 45 || device.temperature < 5 || 
        device.humidity > 80 || device.humidity < 20 || 
        device.battery_level < 20 || device.status === '故障') {
        alertCount++;
    }
    
    renderDeviceList();
}

function renderDeviceList() {
    const container = document.getElementById('device-container');
    
    if (deviceData.size === 0) {
        container.innerHTML = '<p style="text-align: center; color: #7f8c8d;">等待设备数据...</p>';
        return;
    }

    let html = '';
    deviceData.forEach((device, deviceId) => {
        const statusClass = getStatusClass(device.status);
        const timeAgo = getTimeAgo(new Date(device.timestamp));
        
        html += `
            <div class="device-item">
                <div class="device-info">
                    <div class="device-name">${device.device_id} - ${device.device_type}</div>
                    <div class="device-location">📍 ${device.location} | 🕒 ${timeAgo}</div>
                    <div style="margin-top: 5px; font-size: 0.9em;">
                        🌡️ ${device.temperature.toFixed(1)}°C | 
                        💧 ${device.humidity.toFixed(1)}% | 
                        🔋 ${device.battery_level.toFixed(1)}%
                    </div>
                </div>
                <div class="device-status ${statusClass}">${device.status}</div>
            </div>
        `;
    });
    
    container.innerHTML = html;
}

function getStatusClass(status) {
    switch (status) {
        case '正常': return 'status-normal';
        case '警告': return 'status-warning';
        case '故障': return 'status-error';
        default: return 'status-normal';
    }
}

function getTimeAgo(date) {
    const now = new Date();
    const diffMs = now - date;
    const diffSecs = Math.floor(diffMs / 1000);
    
    if (diffSecs < 60) return `${diffSecs}秒前`;
    if (diffSecs < 3600) return `${Math.floor(diffSecs / 60)}分钟前`;
    return `${Math.floor(diffSecs / 3600)}小时前`;
}

function updateStats() {
    document.getElementById('online-devices').textContent = deviceData.size;
    document.getElementById('messages-per-sec').textContent = Math.floor(messageCount / 60); // 简化计算
    document.getElementById('alert-count').textContent = alertCount;
}

// 定期更新统计信息
setInterval(updateStats, 1000);

// 页面加载完成后自动连接
window.addEventListener('load', function() {
    setTimeout(connect, 1000);
});
