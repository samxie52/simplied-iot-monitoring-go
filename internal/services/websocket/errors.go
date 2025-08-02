package websocket

import "errors"

// WebSocket相关错误定义
var (
	// 连接相关错误
	ErrMaxConnectionsReached = errors.New("maximum connections reached")
	ErrClientAlreadyExists   = errors.New("client ID already exists")
	ErrClientNotFound        = errors.New("client not found")
	ErrClientDisconnected    = errors.New("client is disconnected")
	
	// 通道相关错误
	ErrSendChannelFull       = errors.New("send channel is full")
	ErrRegisterChannelFull   = errors.New("register channel is full")
	ErrUnregisterChannelFull = errors.New("unregister channel is full")
	ErrBroadcastChannelFull  = errors.New("broadcast channel is full")
	
	// 消息相关错误
	ErrInvalidMessageFormat  = errors.New("invalid message format")
	ErrMissingMessageType    = errors.New("missing message type")
	ErrUnsupportedMessageType = errors.New("unsupported message type")
	ErrMessageTooLarge       = errors.New("message too large")
	
	// 订阅相关错误
	ErrInvalidSubscription   = errors.New("invalid subscription")
	ErrSubscriptionNotFound  = errors.New("subscription not found")
	ErrDuplicateSubscription = errors.New("duplicate subscription")
	
	// 服务器相关错误
	ErrServerNotStarted      = errors.New("server not started")
	ErrServerAlreadyStarted  = errors.New("server already started")
	ErrServerShuttingDown    = errors.New("server is shutting down")
	
	// 配置相关错误
	ErrInvalidConfig         = errors.New("invalid configuration")
	ErrMissingConfig         = errors.New("missing configuration")
)
