package models

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
)

// ValidationErrorType 验证错误类型
type ValidationErrorType string

const (
	ValidationErrorTypeStructural ValidationErrorType = "structural" // 结构验证错误
	ValidationErrorTypeBusiness   ValidationErrorType = "business"   // 业务验证错误
	ValidationErrorTypeSystem     ValidationErrorType = "system"     // 系统验证错误
)

// ValidationSeverity 验证严重性
type ValidationSeverity string

const (
	ValidationSeverityLow      ValidationSeverity = "low"      // 低严重性
	ValidationSeverityMedium   ValidationSeverity = "medium"   // 中等严重性
	ValidationSeverityHigh     ValidationSeverity = "high"     // 高严重性
	ValidationSeverityCritical ValidationSeverity = "critical" // 严重
)

// RecoveryStrategy 恢复策略
type RecoveryStrategy string

const (
	RecoveryStrategyAutoFix      RecoveryStrategy = "auto_fix"      // 自动修复
	RecoveryStrategyDefaultValue RecoveryStrategy = "default_value" // 使用默认值
	RecoveryStrategyDiscard      RecoveryStrategy = "discard"       // 丢弃数据
	RecoveryStrategyManual       RecoveryStrategy = "manual"        // 手动处理
)

// EnhancedValidationError 增强的验证错误
type EnhancedValidationError struct {
	Field       string              `json:"field"`
	Message     string              `json:"message"`
	Value       interface{}         `json:"value,omitempty"`
	Type        ValidationErrorType `json:"type"`
	Severity    ValidationSeverity  `json:"severity"`
	Code        string              `json:"code"`
	Timestamp   time.Time           `json:"timestamp"`
	Recovery    RecoveryStrategy    `json:"recovery_strategy"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EnhancedValidationResult 增强的验证结果
type EnhancedValidationResult struct {
	IsValid     bool                      `json:"is_valid"`
	Errors      []EnhancedValidationError `json:"errors"`
	Warnings    []EnhancedValidationError `json:"warnings"`
	Fixed       []EnhancedValidationError `json:"fixed,omitempty"`
	ProcessTime time.Duration             `json:"process_time"`
	Timestamp   time.Time                 `json:"timestamp"`
	mutex       sync.RWMutex              // 并发安全
}

// ValidationEngine 验证引擎
type ValidationEngine struct {
	validator     *validator.Validate
	customRules   map[string]ValidationRule
	recoveryHandlers map[string]RecoveryHandler
	cache         *ValidationCache
	metrics       *ValidationMetrics
	workerPool    *ValidationWorkerPool
	mutex         sync.RWMutex
}

// ValidationRule 自定义验证规则
type ValidationRule func(interface{}) *EnhancedValidationError

// RecoveryHandler 恢复处理器
type RecoveryHandler func(interface{}, *EnhancedValidationError) (interface{}, error)

// ValidationCache 验证缓存
type ValidationCache struct {
	cache   map[string]*EnhancedValidationResult
	maxSize int
	ttl     time.Duration
	mutex   sync.RWMutex
}

// ValidationMetrics 验证指标
type ValidationMetrics struct {
	TotalValidations    int64                            `json:"total_validations"`
	SuccessfulValidations int64                          `json:"successful_validations"`
	FailedValidations   int64                            `json:"failed_validations"`
	ErrorsByType        map[ValidationErrorType]int64    `json:"errors_by_type"`
	ErrorsBySeverity    map[ValidationSeverity]int64     `json:"errors_by_severity"`
	RecoveryActions     map[RecoveryStrategy]int64       `json:"recovery_actions"`
	AverageProcessTime  time.Duration                    `json:"average_process_time"`
	CacheHitRate        float64                          `json:"cache_hit_rate"`
	mutex               sync.RWMutex
}

// ValidationWorkerPool 验证工作池
type ValidationWorkerPool struct {
	workers   int
	taskQueue chan ValidationTask
	results   chan ValidationTaskResult
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// ValidationTask 验证任务
type ValidationTask struct {
	ID     string
	Data   interface{}
	Rules  []string
	Result chan ValidationTaskResult
}

// ValidationTaskResult 验证任务结果
type ValidationTaskResult struct {
	ID     string
	Result *EnhancedValidationResult
	Error  error
}

// NewValidationEngine 创建验证引擎
func NewValidationEngine() *ValidationEngine {
	engine := &ValidationEngine{
		validator:        validator.New(),
		customRules:      make(map[string]ValidationRule),
		recoveryHandlers: make(map[string]RecoveryHandler),
		cache:           NewValidationCache(1000, 5*time.Minute),
		metrics:         NewValidationMetrics(),
		workerPool:      NewValidationWorkerPool(10),
	}
	
	// 注册默认恢复处理器
	engine.registerDefaultRecoveryHandlers()
	
	return engine
}

// NewEnhancedValidationResult 创建验证结果
func NewEnhancedValidationResult() *EnhancedValidationResult {
	return &EnhancedValidationResult{
		IsValid:   true,
		Errors:    make([]EnhancedValidationError, 0),
		Warnings:  make([]EnhancedValidationError, 0),
		Fixed:     make([]EnhancedValidationError, 0),
		Timestamp: time.Now(),
	}
}

// NewValidationCache 创建验证缓存
func NewValidationCache(maxSize int, ttl time.Duration) *ValidationCache {
	return &ValidationCache{
		cache:   make(map[string]*EnhancedValidationResult),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// NewValidationMetrics 创建验证指标
func NewValidationMetrics() *ValidationMetrics {
	return &ValidationMetrics{
		ErrorsByType:     make(map[ValidationErrorType]int64),
		ErrorsBySeverity: make(map[ValidationSeverity]int64),
		RecoveryActions:  make(map[RecoveryStrategy]int64),
	}
}

// NewValidationWorkerPool 创建验证工作池
func NewValidationWorkerPool(workers int) *ValidationWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &ValidationWorkerPool{
		workers:   workers,
		taskQueue: make(chan ValidationTask, workers*2),
		results:   make(chan ValidationTaskResult, workers*2),
		ctx:       ctx,
		cancel:    cancel,
	}
	
	// 启动工作协程
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}
	
	return pool
}

// AddError 添加错误
func (r *EnhancedValidationResult) AddError(err EnhancedValidationError) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.Errors = append(r.Errors, err)
	r.IsValid = false
}

// AddWarning 添加警告
func (r *EnhancedValidationResult) AddWarning(warning EnhancedValidationError) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.Warnings = append(r.Warnings, warning)
}

// AddFixed 添加修复记录
func (r *EnhancedValidationResult) AddFixed(fixed EnhancedValidationError) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.Fixed = append(r.Fixed, fixed)
}

// GetErrors 获取错误列表
func (r *EnhancedValidationResult) GetErrors() []EnhancedValidationError {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	errors := make([]EnhancedValidationError, len(r.Errors))
	copy(errors, r.Errors)
	return errors
}

// GetWarnings 获取警告列表
func (r *EnhancedValidationResult) GetWarnings() []EnhancedValidationError {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	warnings := make([]EnhancedValidationError, len(r.Warnings))
	copy(warnings, r.Warnings)
	return warnings
}

// Validate 验证数据
func (e *ValidationEngine) Validate(data interface{}) *EnhancedValidationResult {
	start := time.Now()
	result := NewEnhancedValidationResult()
	
	// 结构验证
	if err := e.validator.Struct(data); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			for _, validationError := range validationErrors {
				enhancedError := EnhancedValidationError{
					Field:     validationError.Field(),
					Message:   validationError.Error(),
					Value:     validationError.Value(),
					Type:      ValidationErrorTypeStructural,
					Severity:  ValidationSeverityMedium,
					Code:      validationError.Tag(),
					Timestamp: time.Now(),
					Recovery:  RecoveryStrategyDefaultValue,
					Metadata:  make(map[string]interface{}),
				}
				result.AddError(enhancedError)
			}
		}
	}
	
	// 自定义规则验证
	e.mutex.RLock()
	for ruleName, rule := range e.customRules {
		if err := rule(data); err != nil {
			err.Code = ruleName
			result.AddError(*err)
		}
	}
	e.mutex.RUnlock()
	
	// 尝试恢复错误
	e.attemptRecovery(data, result)
	
	result.ProcessTime = time.Since(start)
	
	// 更新指标
	e.updateMetrics(result)
	
	return result
}

// ValidateAsync 异步验证
func (e *ValidationEngine) ValidateAsync(data interface{}) <-chan *EnhancedValidationResult {
	resultChan := make(chan *EnhancedValidationResult, 1)
	
	go func() {
		defer close(resultChan)
		result := e.Validate(data)
		resultChan <- result
	}()
	
	return resultChan
}

// ValidateConcurrent 并发验证
func (e *ValidationEngine) ValidateConcurrent(dataList []interface{}) []*EnhancedValidationResult {
	results := make([]*EnhancedValidationResult, len(dataList))
	var wg sync.WaitGroup
	
	for i, data := range dataList {
		wg.Add(1)
		go func(index int, d interface{}) {
			defer wg.Done()
			results[index] = e.Validate(d)
		}(i, data)
	}
	
	wg.Wait()
	return results
}

// RegisterCustomRule 注册自定义规则
func (e *ValidationEngine) RegisterCustomRule(name string, rule ValidationRule) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.customRules[name] = rule
}

// RegisterRecoveryHandler 注册恢复处理器
func (e *ValidationEngine) RegisterRecoveryHandler(errorCode string, handler RecoveryHandler) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.recoveryHandlers[errorCode] = handler
}

// attemptRecovery 尝试恢复错误
func (e *ValidationEngine) attemptRecovery(data interface{}, result *EnhancedValidationResult) {
	errors := result.GetErrors()
	for _, err := range errors {
		if handler, exists := e.recoveryHandlers[err.Code]; exists {
			if recoveredData, recoveryErr := handler(data, &err); recoveryErr == nil {
				// 记录修复
				fixedError := err
				fixedError.Message = fmt.Sprintf("Auto-fixed: %s", err.Message)
				result.AddFixed(fixedError)
				
				// 更新数据（这里需要根据具体实现来处理）
				_ = recoveredData
			}
		}
	}
}

// updateMetrics 更新指标
func (e *ValidationEngine) updateMetrics(result *EnhancedValidationResult) {
	e.metrics.mutex.Lock()
	defer e.metrics.mutex.Unlock()
	
	e.metrics.TotalValidations++
	if result.IsValid {
		e.metrics.SuccessfulValidations++
	} else {
		e.metrics.FailedValidations++
	}
	
	for _, err := range result.Errors {
		e.metrics.ErrorsByType[err.Type]++
		e.metrics.ErrorsBySeverity[err.Severity]++
	}
	
	for _, fixed := range result.Fixed {
		e.metrics.RecoveryActions[fixed.Recovery]++
	}
	
	// 更新平均处理时间
	totalTime := time.Duration(e.metrics.TotalValidations) * e.metrics.AverageProcessTime
	e.metrics.AverageProcessTime = (totalTime + result.ProcessTime) / time.Duration(e.metrics.TotalValidations)
}

// registerDefaultRecoveryHandlers 注册默认恢复处理器
func (e *ValidationEngine) registerDefaultRecoveryHandlers() {
	// 必填字段恢复处理器
	e.RegisterRecoveryHandler("required", func(data interface{}, error *EnhancedValidationError) (interface{}, error) {
		// 根据字段类型设置默认值
		return data, nil
	})
	
	// 数值范围恢复处理器
	e.RegisterRecoveryHandler("min", func(data interface{}, error *EnhancedValidationError) (interface{}, error) {
		// 设置为最小值
		return data, nil
	})
	
	e.RegisterRecoveryHandler("max", func(data interface{}, error *EnhancedValidationError) (interface{}, error) {
		// 设置为最大值
		return data, nil
	})
}

// Get 从缓存获取验证结果
func (c *ValidationCache) Get(key string) (*EnhancedValidationResult, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	result, exists := c.cache[key]
	if !exists {
		return nil, false
	}
	
	// 检查TTL
	if time.Since(result.Timestamp) > c.ttl {
		delete(c.cache, key)
		return nil, false
	}
	
	return result, true
}

// Set 设置缓存
func (c *ValidationCache) Set(key string, result *EnhancedValidationResult) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// 检查缓存大小
	if len(c.cache) >= c.maxSize {
		// 删除最旧的条目
		var oldestKey string
		var oldestTime time.Time
		for k, v := range c.cache {
			if oldestKey == "" || v.Timestamp.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.Timestamp
			}
		}
		delete(c.cache, oldestKey)
	}
	
	c.cache[key] = result
}

// Clear 清空缓存
func (c *ValidationCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache = make(map[string]*EnhancedValidationResult)
}

// Size 获取缓存大小
func (c *ValidationCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.cache)
}

// GetMetrics 获取指标
func (e *ValidationEngine) GetMetrics() *ValidationMetrics {
	e.metrics.mutex.RLock()
	defer e.metrics.mutex.RUnlock()
	
	// 创建副本
	metrics := &ValidationMetrics{
		TotalValidations:      e.metrics.TotalValidations,
		SuccessfulValidations: e.metrics.SuccessfulValidations,
		FailedValidations:     e.metrics.FailedValidations,
		ErrorsByType:          make(map[ValidationErrorType]int64),
		ErrorsBySeverity:      make(map[ValidationSeverity]int64),
		RecoveryActions:       make(map[RecoveryStrategy]int64),
		AverageProcessTime:    e.metrics.AverageProcessTime,
		CacheHitRate:          e.metrics.CacheHitRate,
	}
	
	for k, v := range e.metrics.ErrorsByType {
		metrics.ErrorsByType[k] = v
	}
	for k, v := range e.metrics.ErrorsBySeverity {
		metrics.ErrorsBySeverity[k] = v
	}
	for k, v := range e.metrics.RecoveryActions {
		metrics.RecoveryActions[k] = v
	}
	
	return metrics
}

// worker 工作协程
func (p *ValidationWorkerPool) worker() {
	defer p.wg.Done()
	
	for {
		select {
		case task := <-p.taskQueue:
			// 处理验证任务
			engine := NewValidationEngine()
			result := engine.Validate(task.Data)
			
			taskResult := ValidationTaskResult{
				ID:     task.ID,
				Result: result,
			}
			
			select {
			case task.Result <- taskResult:
			case <-p.ctx.Done():
				return
			}
			
		case <-p.ctx.Done():
			return
		}
	}
}

// SubmitTask 提交验证任务
func (p *ValidationWorkerPool) SubmitTask(task ValidationTask) {
	select {
	case p.taskQueue <- task:
	case <-p.ctx.Done():
	}
}

// Close 关闭工作池
func (p *ValidationWorkerPool) Close() {
	p.cancel()
	p.wg.Wait()
	close(p.taskQueue)
	close(p.results)
}
