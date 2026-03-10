package libs

import (
	"encoding/json"
	"errors"
	"fmt"
)

// FieldMeta 定义选择器字段的掩码位和默认值
// Bit 为该字段对应的掩码位，Default 为默认值（nil 表示无默认值）
type FieldMeta struct {
	Bit     uint32
	Default interface{}
}

// Selector 表示一个 UiSelector 的构造器
// 用于构建 Android UI 元素的查询条件
type Selector struct {
	// 存放已设置的字段及其值
	fields map[string]interface{}

	// 掩码值（通过设置/删除字段自动维护）
	mask uint32

	// 子/兄弟关系列表，元素为 "child" 或 "sibling"
	childOrSibling []string

	// 对应的嵌套 Selector 列表，长度与 childOrSibling 相同
	childOrSiblingSelector []*Selector
}

// fieldDefs 定义所有支持的选择器字段及其元数据（与 Python 版 uiautomator2 一致）
var fieldDefs = map[string]FieldMeta{
	"text":                  {Bit: 0x01, Default: nil},
	"textContains":          {Bit: 0x02, Default: nil},
	"textMatches":           {Bit: 0x04, Default: nil},
	"textStartsWith":        {Bit: 0x08, Default: nil},
	"className":             {Bit: 0x10, Default: nil},
	"classNameMatches":      {Bit: 0x20, Default: nil},
	"description":           {Bit: 0x40, Default: nil},
	"descriptionContains":   {Bit: 0x80, Default: nil},
	"descriptionMatches":    {Bit: 0x0100, Default: nil},
	"descriptionStartsWith": {Bit: 0x0200, Default: nil},
	"checkable":             {Bit: 0x0400, Default: false},
	"checked":               {Bit: 0x0800, Default: false},
	"clickable":             {Bit: 0x1000, Default: false},
	"longClickable":         {Bit: 0x2000, Default: false},
	"scrollable":            {Bit: 0x4000, Default: false},
	"enabled":               {Bit: 0x8000, Default: false},
	"focusable":             {Bit: 0x010000, Default: false},
	"focused":               {Bit: 0x020000, Default: false},
	"selected":              {Bit: 0x040000, Default: false},
	"packageName":           {Bit: 0x080000, Default: nil},
	"packageNameMatches":    {Bit: 0x100000, Default: nil},
	"resourceId":            {Bit: 0x200000, Default: nil},
	"resourceIdMatches":     {Bit: 0x400000, Default: nil},
	"index":                 {Bit: 0x800000, Default: 0},
	"instance":              {Bit: 0x01000000, Default: 0},
}

// New 创建一个新的 Selector，可选传入初始字段
func New(initial map[string]interface{}) (*Selector, error) {
	s := &Selector{
		fields:                 make(map[string]interface{}),
		childOrSibling:         []string{},
		childOrSiblingSelector: []*Selector{},
		mask:                   0,
	}
	for k, v := range initial {
		if err := s.Set(k, v); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// MustNew 与 New 相同，但在出错时 panic，适合简洁的初始化场景
func MustNew(initial map[string]interface{}) *Selector {
	s, err := New(initial)
	if err != nil {
		panic(err)
	}
	return s
}

// validateValue 对给定字段和值进行类型校验
// 布尔字段要求值为 bool 类型，整数字段要求值为整数类型
func validateValue(key string, val interface{}) error {
	meta, ok := fieldDefs[key]
	if !ok {
		return fmt.Errorf("不支持的字段: %s", key)
	}
	if meta.Default == false {
		// 布尔字段校验
		_, ok := val.(bool)
		if !ok {
			return fmt.Errorf("%s 必须是 bool 类型", key)
		}
		return nil
	}
	// 整数字段校验
	switch d := meta.Default.(type) {
	case int:
		switch val.(type) {
		case int, int8, int16, int32, int64:
			return nil
		case uint, uint8, uint16, uint32, uint64:
			return nil
		default:
			return fmt.Errorf("%s 必须是整数类型, 默认值=%v", key, d)
		}
	default:
		// 其他字段无特殊类型要求
		return nil
	}
}

// Set 设置字段值并更新掩码
// 如果字段名非法或类型不匹配则返回错误
func (s *Selector) Set(key string, val interface{}) error {
	if _, ok := fieldDefs[key]; !ok {
		return fmt.Errorf("不支持的字段: %s", key)
	}
	if err := validateValue(key, val); err != nil {
		return err
	}
	s.fields[key] = val
	s.mask = s.mask | fieldDefs[key].Bit
	return nil
}

// Delete 删除字段并更新掩码
// 删除不存在的字段不会报错（幂等操作）
func (s *Selector) Delete(key string) error {
	if _, ok := fieldDefs[key]; !ok {
		return fmt.Errorf("不支持的字段: %s", key)
	}
	if _, present := s.fields[key]; present {
		delete(s.fields, key)
		s.mask = s.mask & ^fieldDefs[key].Bit
	}
	return nil
}

// Mask 返回当前掩码值（只读）
func (s *Selector) Mask() uint32 {
	return s.mask
}

// Child 添加一个子元素选择器
func (s *Selector) Child(initial map[string]interface{}) (*Selector, error) {
	child, err := New(initial)
	if err != nil {
		return nil, err
	}
	s.childOrSibling = append(s.childOrSibling, "child")
	s.childOrSiblingSelector = append(s.childOrSiblingSelector, child)
	return s, nil
}

// Sibling 添加一个兄弟元素选择器
func (s *Selector) Sibling(initial map[string]interface{}) (*Selector, error) {
	child, err := New(initial)
	if err != nil {
		return nil, err
	}
	s.childOrSibling = append(s.childOrSibling, "sibling")
	s.childOrSiblingSelector = append(s.childOrSiblingSelector, child)
	return s, nil
}

// UpdateInstance 更新最后一个子/兄弟选择器的 instance 字段
// 如果没有子/兄弟选择器，则更新根选择器的 instance
func (s *Selector) UpdateInstance(i int) error {
	n := len(s.childOrSiblingSelector)
	if n > 0 {
		return s.childOrSiblingSelector[n-1].Set("instance", i)
	}
	return s.Set("instance", i)
}

// Clone 深拷贝当前 Selector，包括所有子/兄弟选择器
func (s *Selector) Clone() *Selector {
	clone := &Selector{
		fields:                 make(map[string]interface{}, len(s.fields)),
		mask:                   s.mask,
		childOrSibling:         append([]string{}, s.childOrSibling...),
		childOrSiblingSelector: make([]*Selector, 0, len(s.childOrSiblingSelector)),
	}
	for k, v := range s.fields {
		clone.fields[k] = v
	}
	for _, c := range s.childOrSiblingSelector {
		clone.childOrSiblingSelector = append(clone.childOrSiblingSelector, c.Clone())
	}
	return clone
}

// ToMap 将选择器序列化为 map，便于 JSON 编码或 RPC 调用
// 始终包含 childOrSibling 和 childOrSiblingSelector 字段（即使为空），
// 与 Python 版本保持一致，确保 UIAutomator2 服务端能正确解析
func (s *Selector) ToMap() map[string]interface{} {
	out := make(map[string]interface{}, len(s.fields)+3)
	for k, v := range s.fields {
		out[k] = v
	}
	out["mask"] = s.mask
	out["childOrSibling"] = append([]string{}, s.childOrSibling...)
	cs := make([]map[string]interface{}, 0, len(s.childOrSiblingSelector))
	for _, c := range s.childOrSiblingSelector {
		cs = append(cs, c.ToMap())
	}
	out["childOrSiblingSelector"] = cs
	return out
}

// ToJSON 返回选择器的 JSON 编码
func (s *Selector) ToJSON() ([]byte, error) {
	return json.Marshal(s.ToMap())
}

// FromMap 从 map 恢复 Selector 实例
// 自动解析已知字段、掩码和子/兄弟选择器
func FromMap(data map[string]interface{}) (*Selector, error) {
	root := &Selector{
		fields:                 make(map[string]interface{}),
		childOrSibling:         []string{},
		childOrSiblingSelector: []*Selector{},
		mask:                   0,
	}
	// 恢复已知字段
	for k := range fieldDefs {
		if v, ok := data[k]; ok {
			if err := root.Set(k, v); err != nil {
				return nil, err
			}
		}
	}
	// 恢复掩码值
	if m, ok := data["mask"]; ok {
		switch mv := m.(type) {
		case float64:
			root.mask = uint32(mv)
		case uint32:
			root.mask = mv
		case int:
			root.mask = uint32(mv)
		case int64:
			root.mask = uint32(mv)
		}
	}
	// 恢复子/兄弟关系列表
	if cs, ok := data["childOrSibling"]; ok {
		if arr, ok := cs.([]interface{}); ok {
			for _, e := range arr {
				if sname, ok := e.(string); ok {
					root.childOrSibling = append(root.childOrSibling, sname)
				}
			}
		}
	}
	// 恢复子/兄弟选择器
	if css, ok := data["childOrSiblingSelector"]; ok {
		if arr, ok := css.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					c, err := FromMap(m)
					if err != nil {
						return nil, err
					}
					root.childOrSiblingSelector = append(root.childOrSiblingSelector, c)
				}
			}
		}
	}
	return root, nil
}

// UpdateAtPath 在指定路径上更新字段
// path 为逐级索引，例如 [0,2] 表示 childOrSiblingSelector[0].childOrSiblingSelector[2]
func (s *Selector) UpdateAtPath(path []int, updates map[string]interface{}) error {
	node := s
	for _, idx := range path {
		if idx < 0 || idx >= len(node.childOrSiblingSelector) {
			return errors.New("路径索引越界")
		}
		node = node.childOrSiblingSelector[idx]
	}
	for k, v := range updates {
		if err := node.Set(k, v); err != nil {
			return err
		}
	}
	return nil
}

// String 实现 fmt.Stringer 接口，输出可读的选择器表示
func (s *Selector) String() string {
	m := s.ToMap()
	if _, ok := m["childOrSibling"]; !ok {
		delete(m, "childOrSibling")
		delete(m, "childOrSiblingSelector")
	}
	b, _ := json.Marshal(m)
	return "Selector " + string(b)
}
