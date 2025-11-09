package selector

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// FieldMeta 保存每个字段对应的 mask 位和默认值（nil 表示无默认值）
type FieldMeta struct {
	Bit     uint32
	Default interface{}
}

// Selector 表示一个 UiSelector 的构造器
type Selector struct {
	// 存放字段及其值（只包含显式设置的字段）
	fields map[string]interface{}

	// mask 值（通过设置/删除字段自动维护）
	mask uint32

	// childOrSibling 顺序列表，元素为 "child" 或 "sibling"
	childOrSibling []string

	// 对应的嵌套 Selector 列表，长度与 childOrSibling 相同
	childOrSiblingSelector []*Selector
}

// 字段元数据（与 Python 版本一致）
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

// New creates a Selector and可选传入初始字段
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

// MustNew 跟 New 相同，但在错误时 panic，便于简洁示例
func MustNew(initial map[string]interface{}) *Selector {
	s, err := New(initial)
	if err != nil {
		panic(err)
	}
	return s
}

// validateValue 对给定字段和值做类型校验（布尔字段与整数字段）
func validateValue(key string, val interface{}) error {
	meta, ok := fieldDefs[key]
	if !ok {
		return fmt.Errorf("field %s is not allowed", key)
	}
	if meta.Default == false {
		// 期望 bool
		_, ok := val.(bool)
		if !ok {
			return fmt.Errorf("%s must be bool", key)
		}
		return nil
	}
	// 对整数字段（Default 为 int 类型）要求 int
	switch d := meta.Default.(type) {
	case int:
		// 支持 int 和可被转为 int 的数值（如 int64）
		switch val.(type) {
		case int, int8, int16, int32, int64:
			return nil
		case uint, uint8, uint16, uint32, uint64:
			return nil
		default:
			return fmt.Errorf("%s must be integer type, default=%v", key, d)
		}
	default:
		// 其它字段没有特别要求
		return nil
	}
}

// Set 设置字段并更新 mask；若字段非法或类型不对则返回错误
func (s *Selector) Set(key string, val interface{}) error {
	if _, ok := fieldDefs[key]; !ok {
		return fmt.Errorf("%s is not allowed", key)
	}
	if err := validateValue(key, val); err != nil {
		return err
	}
	s.fields[key] = val
	s.mask = s.mask | fieldDefs[key].Bit
	return nil
}

// Delete 删除字段并更新 mask；幂等（删除不存在字段不报错）
func (s *Selector) Delete(key string) error {
	if _, ok := fieldDefs[key]; !ok {
		return fmt.Errorf("%s is not allowed", key)
	}
	if _, present := s.fields[key]; present {
		delete(s.fields, key)
		s.mask = s.mask & ^fieldDefs[key].Bit
	}
	return nil
}

// Mask 返回当前 mask（只读）
func (s *Selector) Mask() uint32 {
	return s.mask
}

// Child 在末尾添加 child
func (s *Selector) Child(initial map[string]interface{}) (*Selector, error) {
	child, err := New(initial)
	if err != nil {
		return nil, err
	}
	s.childOrSibling = append(s.childOrSibling, "child")
	s.childOrSiblingSelector = append(s.childOrSiblingSelector, child)
	return s, nil
}

// Sibling 在末尾添加 sibling
func (s *Selector) Sibling(initial map[string]interface{}) (*Selector, error) {
	child, err := New(initial)
	if err != nil {
		return nil, err
	}
	s.childOrSibling = append(s.childOrSibling, "sibling")
	s.childOrSiblingSelector = append(s.childOrSiblingSelector, child)
	return s, nil
}

// UpdateInstance 更新最后一个 childOrSiblingSelector 的 instance 字段（或根 selector）
func (s *Selector) UpdateInstance(i int) error {
	n := len(s.childOrSiblingSelector)
	if n > 0 {
		return s.childOrSiblingSelector[n-1].Set("instance", i)
	}
	return s.Set("instance", i)
}

// Clone 深拷贝 Selector，包括子/兄弟
func (s *Selector) Clone() *Selector {
	clone := &Selector{
		fields:                 make(map[string]interface{}, len(s.fields)),
		mask:                   s.mask,
		childOrSibling:         append([]string{}, s.childOrSibling...),
		childOrSiblingSelector: make([]*Selector, 0, len(s.childOrSiblingSelector)),
	}
	for k, v := range s.fields {
		// 简单深拷贝：对于常见类型（string,bool,int）直接赋值即可。
		// 若值为复杂结构，调用方应使用 ToMap/ToJSON 再 Parse 得到深拷贝。
		clone.fields[k] = v
	}
	for _, c := range s.childOrSiblingSelector {
		clone.childOrSiblingSelector = append(clone.childOrSiblingSelector, c.Clone())
	}
	return clone
}

// ToMap 序列化为 map，便于 RPC 调用或 JSON 编码
func (s *Selector) ToMap() map[string]interface{} {
	out := make(map[string]interface{}, len(s.fields)+3)
	for k, v := range s.fields {
		out[k] = v
	}
	out["mask"] = s.mask
	if len(s.childOrSibling) > 0 {
		out["childOrSibling"] = append([]string{}, s.childOrSibling...)
		cs := make([]map[string]interface{}, 0, len(s.childOrSiblingSelector))
		for _, c := range s.childOrSiblingSelector {
			cs = append(cs, c.ToMap())
		}
		out["childOrSiblingSelector"] = cs
	}
	return out
}

// ToJSON 返回 ToMap 的 JSON 编码
func (s *Selector) ToJSON() ([]byte, error) {
	return json.Marshal(s.ToMap())
}

// FromMap 从 map 恢复 Selector（简单实现，忽略非法字段）
func FromMap(data map[string]interface{}) (*Selector, error) {
	// 提取根字段
	root := &Selector{
		fields:                 make(map[string]interface{}),
		childOrSibling:         []string{},
		childOrSiblingSelector: []*Selector{},
		mask:                   0,
	}
	// 读取已知字段
	for k, meta := range fieldDefs {
		if v, ok := data[k]; ok {
			// 尝试 Set 以做类型校验并设置 mask
			if err := root.Set(k, v); err != nil {
				return nil, err
			}
			// 注意：Set 已经更新了 mask
			_ = meta
		}
	}
	// 恢复 mask（如果提供了 mask，并且为数值）
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
		default:
			// 忽略不能解析的 mask
		}
	}
	// 恢复 childOrSibling 列表和对应 selector（期望 childOrSiblingSelector 为 []map[string]interface{}）
	if cs, ok := data["childOrSibling"]; ok {
		if arr, ok := cs.([]interface{}); ok {
			for _, e := range arr {
				if sname, ok := e.(string); ok {
					root.childOrSibling = append(root.childOrSibling, sname)
				}
			}
		}
	}
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

// UpdateAtPath 在指定路径（child 索引链）上更新字段
// path: 逐级索引，例如 [0,2] 表示 childOrSiblingSelector[0].childOrSiblingSelector[2]
func (s *Selector) UpdateAtPath(path []int, updates map[string]interface{}) error {
	node := s
	for _, idx := range path {
		if idx < 0 || idx >= len(node.childOrSiblingSelector) {
			return errors.New("path out of range")
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

// String 实现 fmt.Stringer，输出友好可读的 Selector 表示（类似 Python 的 __str__）
func (s *Selector) String() string {
	m := s.ToMap()
	// 删除空的 childOrSibling 字段以保持简洁
	if _, ok := m["childOrSibling"]; !ok {
		delete(m, "childOrSibling")
		delete(m, "childOrSiblingSelector")
	}
	b, _ := json.Marshal(m)
	return "Selector " + string(b)
}

// Example 用法示例（不是正式测试，仅供快速手动运行）
func Example() {
	// 初始化根 selector
	root := MustNew(map[string]interface{}{
		"className": "android.widget.LinearLayout",
	})

	// 添加 child
	root.Child(map[string]interface{}{
		"text":     "下一步",
		"instance": 0,
	})

	// 更新最后一个 child 的 instance
	_ = root.UpdateInstance(2)

	// 深拷贝
	cpy := root.Clone()

	// 序列化到 JSON
	j, _ := cpy.ToJSON()
	fmt.Println(string(j))
}

// 简单测试函数（你可在 package 内使用 testing 包将其改写成真正的单元测试）
func SimpleTests() {
	// set & delete
	s := MustNew(map[string]interface{}{"text": "hello"})
	fmt.Println("mask after set:", strconv.FormatUint(uint64(s.Mask()), 10))
	_ = s.Delete("text")
	fmt.Println("mask after delete:", strconv.FormatUint(uint64(s.Mask()), 10))

	// bool 类型校验
	_, err := New(map[string]interface{}{"checkable": "yes"})
	fmt.Println("expected error for bad bool:", err != nil)

	// clone 深拷贝检查
	s2 := MustNew(map[string]interface{}{"text": "a"})
	s2.Child(map[string]interface{}{"text": "b", "instance": 1})
	c := s2.Clone()
	c.childOrSibling[0] = "sibling"
	fmt.Println("original childOrSibling:", s2.childOrSibling[0], "clone childOrSibling:", c.childOrSibling[0])
}
