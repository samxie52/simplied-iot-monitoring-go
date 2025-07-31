package config

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// ConfigWatcher 配置监听器
type ConfigWatcher struct {
	viper    *viper.Viper
	callback ConfigWatchCallback
	stopCh   chan struct{}
	mu       sync.RWMutex
	running  bool
}

// NewConfigWatcher 创建新的配置监听器
func NewConfigWatcher(v *viper.Viper, callback ConfigWatchCallback) *ConfigWatcher {
	return &ConfigWatcher{
		viper:    v,
		callback: callback,
		stopCh:   make(chan struct{}),
		running:  false,
	}
}

// Start 开始监听配置变化
func (cw *ConfigWatcher) Start() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.running {
		return fmt.Errorf("配置监听器已在运行")
	}

	// 启用Viper的配置文件监听
	cw.viper.WatchConfig()

	// 设置配置变化回调
	cw.viper.OnConfigChange(func(e fsnotify.Event) {
		// 重新解析配置
		var newConfig AppConfig
		if err := cw.viper.Unmarshal(&newConfig); err != nil {
			// 记录错误但不停止监听
			return
		}

		// 调用用户回调
		if cw.callback != nil {
			if err := cw.callback(&newConfig); err != nil {
				// 记录回调错误
				return
			}
		}
	})

	cw.running = true
	return nil
}

// Stop 停止监听配置变化
func (cw *ConfigWatcher) Stop() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if !cw.running {
		return nil
	}

	close(cw.stopCh)
	cw.running = false
	return nil
}

// IsRunning 检查监听器是否在运行
func (cw *ConfigWatcher) IsRunning() bool {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.running
}
