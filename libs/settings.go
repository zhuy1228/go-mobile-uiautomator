package libs

import (
	"fmt"
	"sync"
)

// Settings 管理设备的各项配置参数
// 支持类型安全的读写操作
type Settings struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// 默认配置值
var defaultSettings = map[string]interface{}{
	// 等待元素出现的超时时间（秒）
	"wait_timeout": 20.0,
	// 操作前后的延迟 [前延迟, 后延迟]（秒）
	"operation_delay": [2]float64{0, 0},
	// 需要应用操作延迟的方法列表
	"operation_delay_methods": []string{"click", "swipe", "drag", "press"},
	// dump_hierarchy 的最大深度
	"max_depth": 50,
}

// NewSettings 创建一个新的 Settings 实例，使用默认配置
func NewSettings() *Settings {
	s := &Settings{
		data: make(map[string]interface{}),
	}
	// 复制默认配置
	for k, v := range defaultSettings {
		s.data[k] = v
	}
	return s
}

// Get 获取配置项的值
// 如果配置项不存在，返回 nil
func (s *Settings) Get(key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[key]
}

// GetFloat64 获取 float64 类型的配置值
func (s *Settings) GetFloat64(key string) float64 {
	v := s.Get(key)
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	default:
		return 0
	}
}

// GetInt 获取 int 类型的配置值
func (s *Settings) GetInt(key string) int {
	v := s.Get(key)
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	default:
		return 0
	}
}

// GetStringSlice 获取 []string 类型的配置值
func (s *Settings) GetStringSlice(key string) []string {
	v := s.Get(key)
	if v == nil {
		return nil
	}
	if val, ok := v.([]string); ok {
		return val
	}
	return nil
}

// GetOperationDelay 获取操作延迟配置 [前延迟, 后延迟]
func (s *Settings) GetOperationDelay() (float64, float64) {
	v := s.Get("operation_delay")
	if v == nil {
		return 0, 0
	}
	if val, ok := v.([2]float64); ok {
		return val[0], val[1]
	}
	return 0, 0
}

// Set 设置配置项的值，包含类型校验
func (s *Settings) Set(key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 类型校验
	if existing, ok := defaultSettings[key]; ok {
		if err := validateSettingType(key, existing, value); err != nil {
			return err
		}
	}
	s.data[key] = value
	return nil
}

// validateSettingType 校验设置值的类型是否与默认值匹配
func validateSettingType(key string, defaultVal, newVal interface{}) error {
	switch defaultVal.(type) {
	case float64:
		switch newVal.(type) {
		case float64, float32, int, int64:
			return nil
		default:
			return fmt.Errorf("配置 %s 必须是数值类型", key)
		}
	case int:
		switch newVal.(type) {
		case int, int64, float64:
			return nil
		default:
			return fmt.Errorf("配置 %s 必须是整数类型", key)
		}
	}
	return nil
}
