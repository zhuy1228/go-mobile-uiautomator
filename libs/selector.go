package libs

import (
	"fmt"
	"sort"
	"strings"
)

const (
	MASK_TEXT                    uint32 = 0x00000001
	MASK_TEXT_CONTAINS           uint32 = 0x00000002
	MASK_TEXT_MATCHES            uint32 = 0x00000004
	MASK_TEXT_STARTS_WITH        uint32 = 0x00000008
	MASK_CLASS_NAME              uint32 = 0x00000010
	MASK_CLASS_NAME_MATCHES      uint32 = 0x00000020
	MASK_DESCRIPTION             uint32 = 0x00000040
	MASK_DESCRIPTION_CONTAINS    uint32 = 0x00000080
	MASK_DESCRIPTION_MATCHES     uint32 = 0x00000100
	MASK_DESCRIPTION_STARTS_WITH uint32 = 0x00000200
	MASK_CHECKABLE               uint32 = 0x00000400
	MASK_CHECKED                 uint32 = 0x00000800
	MASK_CLICKABLE               uint32 = 0x00001000
	MASK_LONG_CLICKABLE          uint32 = 0x00002000
	MASK_SCROLLABLE              uint32 = 0x00004000
	MASK_ENABLED                 uint32 = 0x00008000
	MASK_FOCUSABLE               uint32 = 0x00010000
	MASK_FOCUSED                 uint32 = 0x00020000
	MASK_SELECTED                uint32 = 0x00040000
	MASK_PACKAGE_NAME            uint32 = 0x00080000
	MASK_PACKAGE_NAME_MATCHES    uint32 = 0x00100000
	MASK_RESOURCE_ID             uint32 = 0x00200000
	MASK_RESOURCE_ID_MATCHES     uint32 = 0x00400000
	MASK_INDEX                   uint32 = 0x00800000
	MASK_INSTANCE                uint32 = 0x01000000
)

// Relationship type for building child/sibling chains.
type relationType string

const (
	relChild   relationType = "child"
	relSibling relationType = "sibling"
)

// Field metadata mirrors Python __fields: name -> (mask, default)
type fieldMeta struct {
	Mask    uint32
	Default any
}

var allowedFields = map[string]fieldMeta{
	"text":                  {MASK_TEXT, nil},
	"textContains":          {MASK_TEXT_CONTAINS, nil},
	"textMatches":           {MASK_TEXT_MATCHES, nil},
	"textStartsWith":        {MASK_TEXT_STARTS_WITH, nil},
	"className":             {MASK_CLASS_NAME, nil},
	"classNameMatches":      {MASK_CLASS_NAME_MATCHES, nil},
	"description":           {MASK_DESCRIPTION, nil},
	"descriptionContains":   {MASK_DESCRIPTION_CONTAINS, nil},
	"descriptionMatches":    {MASK_DESCRIPTION_MATCHES, nil},
	"descriptionStartsWith": {MASK_DESCRIPTION_STARTS_WITH, nil},
	"checkable":             {MASK_CHECKABLE, false},
	"checked":               {MASK_CHECKED, false},
	"clickable":             {MASK_CLICKABLE, false},
	"longClickable":         {MASK_LONG_CLICKABLE, false},
	"scrollable":            {MASK_SCROLLABLE, false},
	"enabled":               {MASK_ENABLED, false},
	"focusable":             {MASK_FOCUSABLE, false},
	"focused":               {MASK_FOCUSED, false},
	"selected":              {MASK_SELECTED, false},
	"packageName":           {MASK_PACKAGE_NAME, nil},
	"packageNameMatches":    {MASK_PACKAGE_NAME_MATCHES, nil},
	"resourceId":            {MASK_RESOURCE_ID, nil},
	"resourceIdMatches":     {MASK_RESOURCE_ID_MATCHES, nil},
	"index":                 {MASK_INDEX, 0},
	"instance":              {MASK_INSTANCE, 0},
}

// Selector replicates Python Selector(dict) behavior with explicit APIs.
type Selector struct {
	mask                 uint32
	fields               map[string]any // only allowedFields keys permitted
	childOrSibling       []relationType
	childOrSiblingSelect []*Selector
}

// NewSelector initializes with kwargs like Python's Selector(**kwargs).
func NewSelector(kwargs map[string]any) (*Selector, error) {
	s := &Selector{
		mask:                 0,
		fields:               make(map[string]any),
		childOrSibling:       make([]relationType, 0),
		childOrSiblingSelect: make([]*Selector, 0),
	}
	for k, v := range kwargs {
		if err := s.Set(k, v); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Set assigns a field and updates the bitmask. Only allowed fields permitted.
func (s *Selector) Set(key string, val any) error {
	meta, ok := allowedFields[key]
	if !ok {
		return fmt.Errorf("%s is not allowed", key)
	}
	s.fields[key] = val
	s.mask |= meta.Mask
	return nil
}

// Delete removes a field and clears its bit in mask.
func (s *Selector) Delete(key string) error {
	meta, ok := allowedFields[key]
	if !ok {
		return fmt.Errorf("%s is not allowed", key)
	}
	delete(s.fields, key)
	s.mask &^= meta.Mask // AND NOT
	return nil
}

// Get retrieves a field (nil if absent).
func (s *Selector) Get(key string) (any, bool) {
	v, ok := s.fields[key]
	return v, ok
}

// Mask returns current bit mask.
func (s *Selector) Mask() uint32 { return s.mask }

// Child appends a child selector (in-place) and returns the receiver.
func (s *Selector) Child(kwargs map[string]any) (*Selector, error) {
	sub, err := NewSelector(kwargs)
	if err != nil {
		return nil, err
	}
	s.childOrSibling = append(s.childOrSibling, relChild)
	s.childOrSiblingSelect = append(s.childOrSiblingSelect, sub)
	return s, nil
}

// Sibling appends a sibling selector (in-place) and returns the receiver.
func (s *Selector) Sibling(kwargs map[string]any) (*Selector, error) {
	sub, err := NewSelector(kwargs)
	if err != nil {
		return nil, err
	}
	s.childOrSibling = append(s.childOrSibling, relSibling)
	s.childOrSiblingSelect = append(s.childOrSiblingSelect, sub)
	return s, nil
}

// Clone deep-copies selector, including child/sibling chains.
func (s *Selector) Clone() *Selector {
	cp := &Selector{
		mask:                 s.mask,
		fields:               make(map[string]any, len(s.fields)),
		childOrSibling:       make([]relationType, len(s.childOrSibling)),
		childOrSiblingSelect: make([]*Selector, 0, len(s.childOrSiblingSelect)),
	}
	for k, v := range s.fields {
		cp.fields[k] = v
	}
	copy(cp.childOrSibling, s.childOrSibling)
	for _, sub := range s.childOrSiblingSelect {
		cp.childOrSiblingSelect = append(cp.childOrSiblingSelect, sub.Clone())
	}
	return cp
}

// UpdateInstance updates 'instance' on the last child/sibling selector if present;
// otherwise updates the current selector's 'instance'.
func (s *Selector) UpdateInstance(i int) error {
	if len(s.childOrSiblingSelect) > 0 {
		last := s.childOrSiblingSelect[len(s.childOrSiblingSelect)-1]
		return last.Set("instance", i)
	}
	return s.Set("instance", i)
}

// String prints Selector [k='v', ...] while skipping mask and empty child/sibling groups.
// Deterministic order for readability.
func (s *Selector) String() string {
	parts := make([]string, 0, len(s.fields)+2)

	// stable sort keys for consistent output
	keys := make([]string, 0, len(s.fields))
	for k := range s.fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// repr-like rendering
		parts = append(parts, fmt.Sprintf("%s=%q", k, fmtAny(s.fields[k])))
	}

	// child/sibling only if present
	if len(s.childOrSibling) > 0 && len(s.childOrSiblingSelect) > 0 {
		// render as chain: rel(type)->subselector
		chunks := make([]string, 0, len(s.childOrSibling))
		for i, rel := range s.childOrSibling {
			chunks = append(chunks, fmt.Sprintf("%s(%s)", rel, s.childOrSiblingSelect[i].String()))
		}
		parts = append(parts, "chain=["+strings.Join(chunks, ", ")+"]")
	}

	return "Selector [" + strings.Join(parts, ", ") + "]"
}

func fmtAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
