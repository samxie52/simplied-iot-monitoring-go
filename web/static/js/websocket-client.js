// WebSocket客户端管理类
// Industrial IoT WebSocket Client

class WebSocketClient {
    constructor(url, options = {}) {
        this.url = url;
        this.options = {
            reconnectInterval: 3000,
            maxReconnectAttempts: 5,
            heartbeatInterval: 30000,
            ...options
        };
        
        this.ws = null;
        this.isConnected = false;
        this.reconnectAttempts = 0;
        this.heartbeatTimer = null;
        this.reconnectTimer = null;
        
        // 事件回调
        this.onConnectCallbacks = [];
        this.onDisconnectCallbacks = [];
        this.onMessageCallbacks = [];
        this.onErrorCallbacks = [];
        
        // 订阅管理
        this.subscriptions = new Map();
        this.messageQueue = [];
        
        // 统计信息
        this.stats = {
            messagesReceived: 0,
            messagesSent: 0,
            reconnectCount: 0,
            lastMessageTime: null,
            connectionStartTime: null
        };
    }
    
    // 连接WebSocket
    connect() {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            console.log('WebSocket已经连接');
            return Promise.resolve();
        }
        
        return new Promise((resolve, reject) => {
            try {
                console.log(`正在连接WebSocket: ${this.url}`);
                this.ws = new WebSocket(this.url);
                
                this.ws.onopen = (event) => {
                    this.isConnected = true;
                    this.reconnectAttempts = 0;
                    this.stats.connectionStartTime = new Date();
                    
                    console.log('WebSocket连接成功');
                    this.startHeartbeat();
                    this.processMessageQueue();
                    
                    // 触发连接回调
                    this.onConnectCallbacks.forEach(callback => {
                        try {
                            callback(event);
                        } catch (error) {
                            console.error('连接回调错误:', error);
                        }
                    });
                    
                    resolve();
                };
                
                this.ws.onmessage = (event) => {
                    this.stats.messagesReceived++;
                    this.stats.lastMessageTime = new Date();
                    
                    try {
                        const data = JSON.parse(event.data);
                        this.handleMessage(data);
                    } catch (error) {
                        console.error('消息解析错误:', error, event.data);
                    }
                };
                
                this.ws.onclose = (event) => {
                    this.isConnected = false;
                    this.stopHeartbeat();
                    
                    console.log('WebSocket连接关闭:', event.code, event.reason);
                    
                    // 触发断开回调
                    this.onDisconnectCallbacks.forEach(callback => {
                        try {
                            callback(event);
                        } catch (error) {
                            console.error('断开回调错误:', error);
                        }
                    });
                    
                    // 自动重连
                    if (!event.wasClean && this.reconnectAttempts < this.options.maxReconnectAttempts) {
                        this.scheduleReconnect();
                    }
                };
                
                this.ws.onerror = (error) => {
                    console.error('WebSocket错误:', error);
                    
                    // 触发错误回调
                    this.onErrorCallbacks.forEach(callback => {
                        try {
                            callback(error);
                        } catch (err) {
                            console.error('错误回调错误:', err);
                        }
                    });
                    
                    reject(error);
                };
                
            } catch (error) {
                console.error('创建WebSocket连接失败:', error);
                reject(error);
            }
        });
    }
    
    // 断开连接
    disconnect() {
        this.stopHeartbeat();
        this.clearReconnectTimer();
        
        if (this.ws) {
            this.ws.close(1000, '用户主动断开');
            this.ws = null;
        }
        
        this.isConnected = false;
        console.log('WebSocket连接已断开');
    }
    
    // 发送消息
    send(message) {
        if (!this.isConnected || !this.ws) {
            console.warn('WebSocket未连接，消息已加入队列');
            this.messageQueue.push(message);
            return false;
        }
        
        try {
            const messageStr = typeof message === 'string' ? message : JSON.stringify(message);
            this.ws.send(messageStr);
            this.stats.messagesSent++;
            return true;
        } catch (error) {
            console.error('发送消息失败:', error);
            return false;
        }
    }
    
    // 订阅数据
    subscribe(filter) {
        const subscriptionId = this.generateSubscriptionId();
        this.subscriptions.set(subscriptionId, filter);
        
        const subscribeMessage = {
            type: 'subscribe',
            subscription_id: subscriptionId,
            filter: filter
        };
        
        this.send(subscribeMessage);
        return subscriptionId;
    }
    
    // 取消订阅
    unsubscribe(subscriptionId) {
        if (!this.subscriptions.has(subscriptionId)) {
            console.warn('订阅ID不存在:', subscriptionId);
            return false;
        }
        
        this.subscriptions.delete(subscriptionId);
        
        const unsubscribeMessage = {
            type: 'unsubscribe',
            subscription_id: subscriptionId
        };
        
        this.send(unsubscribeMessage);
        return true;
    }
    
    // 更新过滤器
    updateFilter(subscriptionId, newFilter) {
        if (!this.subscriptions.has(subscriptionId)) {
            console.warn('订阅ID不存在:', subscriptionId);
            return false;
        }
        
        this.subscriptions.set(subscriptionId, newFilter);
        
        const updateMessage = {
            type: 'update_filter',
            subscription_id: subscriptionId,
            filter: newFilter
        };
        
        this.send(updateMessage);
        return true;
    }
    
    // 事件监听器
    onConnect(callback) {
        this.onConnectCallbacks.push(callback);
    }
    
    onDisconnect(callback) {
        this.onDisconnectCallbacks.push(callback);
    }
    
    onMessage(callback) {
        this.onMessageCallbacks.push(callback);
    }
    
    onError(callback) {
        this.onErrorCallbacks.push(callback);
    }
    
    // 获取连接状态
    getConnectionState() {
        if (!this.ws) return 'CLOSED';
        
        switch (this.ws.readyState) {
            case WebSocket.CONNECTING: return 'CONNECTING';
            case WebSocket.OPEN: return 'OPEN';
            case WebSocket.CLOSING: return 'CLOSING';
            case WebSocket.CLOSED: return 'CLOSED';
            default: return 'UNKNOWN';
        }
    }
    
    // 获取统计信息
    getStats() {
        return {
            ...this.stats,
            isConnected: this.isConnected,
            connectionState: this.getConnectionState(),
            subscriptionCount: this.subscriptions.size,
            queuedMessages: this.messageQueue.length,
            reconnectAttempts: this.reconnectAttempts
        };
    }
    
    // 私有方法
    
    // 处理接收到的消息
    handleMessage(data) {
        // 触发消息回调
        this.onMessageCallbacks.forEach(callback => {
            try {
                callback(data);
            } catch (error) {
                console.error('消息回调错误:', error);
            }
        });
    }
    
    // 开始心跳
    startHeartbeat() {
        this.stopHeartbeat();
        
        this.heartbeatTimer = setInterval(() => {
            if (this.isConnected) {
                this.send({ type: 'ping', timestamp: Date.now() });
            }
        }, this.options.heartbeatInterval);
    }
    
    // 停止心跳
    stopHeartbeat() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
    }
    
    // 安排重连
    scheduleReconnect() {
        this.clearReconnectTimer();
        
        this.reconnectAttempts++;
        this.stats.reconnectCount++;
        
        const delay = this.options.reconnectInterval * Math.pow(2, this.reconnectAttempts - 1);
        console.log(`${delay}ms后尝试第${this.reconnectAttempts}次重连`);
        
        this.reconnectTimer = setTimeout(() => {
            this.connect().catch(error => {
                console.error('重连失败:', error);
            });
        }, delay);
    }
    
    // 清除重连定时器
    clearReconnectTimer() {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
    }
    
    // 处理消息队列
    processMessageQueue() {
        while (this.messageQueue.length > 0 && this.isConnected) {
            const message = this.messageQueue.shift();
            this.send(message);
        }
    }
    
    // 生成订阅ID
    generateSubscriptionId() {
        return 'sub_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
    }
}

// 消息处理器类
class MessageHandler {
    constructor() {
        this.deviceDataHandlers = [];
        this.alertHandlers = [];
        this.systemStatusHandlers = [];
        this.customHandlers = new Map();
    }
    
    // 注册设备数据处理器
    onDeviceData(handler) {
        this.deviceDataHandlers.push(handler);
    }
    
    // 注册告警处理器
    onAlert(handler) {
        this.alertHandlers.push(handler);
    }
    
    // 注册系统状态处理器
    onSystemStatus(handler) {
        this.systemStatusHandlers.push(handler);
    }
    
    // 注册自定义消息处理器
    onCustomMessage(type, handler) {
        if (!this.customHandlers.has(type)) {
            this.customHandlers.set(type, []);
        }
        this.customHandlers.get(type).push(handler);
    }
    
    // 处理消息
    handleMessage(message) {
        try {
            switch (message.type) {
                case 'device_data':
                    this.deviceDataHandlers.forEach(handler => {
                        try {
                            handler(message.data);
                        } catch (error) {
                            console.error('设备数据处理错误:', error);
                        }
                    });
                    break;
                    
                case 'alert':
                    this.alertHandlers.forEach(handler => {
                        try {
                            handler(message.data);
                        } catch (error) {
                            console.error('告警处理错误:', error);
                        }
                    });
                    break;
                    
                case 'system_status':
                    this.systemStatusHandlers.forEach(handler => {
                        try {
                            handler(message.data);
                        } catch (error) {
                            console.error('系统状态处理错误:', error);
                        }
                    });
                    break;
                    
                case 'pong':
                    // 心跳响应，可以计算延迟
                    if (message.timestamp) {
                        const latency = Date.now() - message.timestamp;
                        console.debug('心跳延迟:', latency + 'ms');
                    }
                    break;
                    
                default:
                    // 处理自定义消息类型
                    if (this.customHandlers.has(message.type)) {
                        this.customHandlers.get(message.type).forEach(handler => {
                            try {
                                handler(message);
                            } catch (error) {
                                console.error(`自定义消息处理错误 (${message.type}):`, error);
                            }
                        });
                    } else {
                        console.warn('未知消息类型:', message.type);
                    }
                    break;
            }
        } catch (error) {
            console.error('消息处理错误:', error);
        }
    }
}

// 导出类
window.WebSocketClient = WebSocketClient;
window.MessageHandler = MessageHandler;
