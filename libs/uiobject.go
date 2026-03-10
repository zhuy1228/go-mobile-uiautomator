package libs

import (
	"encoding/json"
	"fmt"
	"time"
)

// UiObject 表示一个 Android UI 控件对象
// 通过 Selector 定位，支持点击、输入、滑动等操作
// 对应 Python 版本的 UiObject 类
type UiObject struct {
	device   *Device
	selector *Selector
	jsonrpc  *JsonRpcWrapper
}

// NewUiObject 创建一个新的 UiObject
func NewUiObject(device *Device, selector *Selector) *UiObject {
	return &UiObject{
		device:   device,
		selector: selector,
		jsonrpc:  device.JsonRpc(),
	}
}

// Selector 返回当前 UiObject 的选择器
func (u *UiObject) Selector() *Selector {
	return u.selector
}

// ---------- 等待和存在性 ----------

// Exists 检查 UI 元素是否存在于当前窗口
func (u *UiObject) Exists() (bool, error) {
	raw, err := u.jsonrpc.Call("objInfo", []interface{}{u.selector.ToMap()}, 10)
	if err != nil {
		// UiObjectNotFoundError 意味着不存在
		if _, ok := err.(*UiObjectNotFoundError); ok {
			return false, nil
		}
		return false, err
	}
	return raw != nil, nil
}

// Wait 等待 UI 元素出现或消失
// exists: true 等待出现，false 等待消失
// timeout: 超时时间（秒），0 使用默认值
//
// 通过 JSON-RPC 调用服务端的 waitForExists/waitUntilGone 实现
// 使用 adb forward 端口转发，连接稳定，与 Python 版本行为一致
func (u *UiObject) Wait(exists bool, timeout float64) (bool, error) {
	if timeout <= 0 {
		timeout = u.device.WaitTimeout()
	}
	if timeout <= 0 {
		timeout = 10.0
	}
	httpWait := timeout + 10

	if exists {
		raw, err := u.jsonrpc.Call("waitForExists", []interface{}{u.selector.ToMap(), int(timeout * 1000)}, httpWait)
		if err != nil {
			// HTTP 超时时回退到 Exists 检查
			if _, ok := err.(*HTTPError); ok {
				ex, _ := u.Exists()
				return ex, nil
			}
			return false, err
		}
		var result bool
		json.Unmarshal(raw, &result)
		return result, nil
	}

	// 等待消失
	raw, err := u.jsonrpc.Call("waitUntilGone", []interface{}{u.selector.ToMap(), int(timeout * 1000)}, httpWait)
	if err != nil {
		if _, ok := err.(*HTTPError); ok {
			ex, _ := u.Exists()
			return !ex, nil
		}
		return false, err
	}
	var result bool
	json.Unmarshal(raw, &result)
	return result, nil
}

// WaitGone 等待 UI 元素消失
func (u *UiObject) WaitGone(timeout float64) (bool, error) {
	return u.Wait(false, timeout)
}

// MustWait 等待元素出现，不存在则返回 UiObjectNotFoundError
func (u *UiObject) MustWait(timeout float64) error {
	found, err := u.Wait(true, timeout)
	if err != nil {
		return err
	}
	if !found {
		return &UiObjectNotFoundError{
			Code:    -32002,
			Message: fmt.Sprintf("等待超时: %s", u.selector.String()),
			Params:  u.selector.ToMap(),
		}
	}
	return nil
}

// ---------- 元素信息 ----------

// ObjInfo 包含 UI 元素的详细信息
type ObjInfo struct {
	Text               string                 `json:"text"`
	ClassName          string                 `json:"className"`
	ContentDescription string                 `json:"contentDescription"`
	PackageName        string                 `json:"packageName"`
	ResourceName       string                 `json:"resourceName"`
	Checkable          bool                   `json:"checkable"`
	Checked            bool                   `json:"checked"`
	Clickable          bool                   `json:"clickable"`
	Enabled            bool                   `json:"enabled"`
	Focusable          bool                   `json:"focusable"`
	Focused            bool                   `json:"focused"`
	LongClickable      bool                   `json:"longClickable"`
	Scrollable         bool                   `json:"scrollable"`
	Selected           bool                   `json:"selected"`
	Bounds             map[string]int         `json:"bounds"`
	VisibleBounds      map[string]int         `json:"visibleBounds"`
	ChildCount         int                    `json:"childCount"`
	Extra              map[string]interface{} `json:"-"` // 额外字段
}

// Info 获取 UI 元素信息
func (u *UiObject) Info() (*ObjInfo, error) {
	raw, err := u.jsonrpc.Call("objInfo", []interface{}{u.selector.ToMap()})
	if err != nil {
		return nil, err
	}
	var info ObjInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// InfoRaw 获取 UI 元素的原始 map 信息
func (u *UiObject) InfoRaw() (map[string]interface{}, error) {
	raw, err := u.jsonrpc.Call("objInfo", []interface{}{u.selector.ToMap()})
	if err != nil {
		return nil, err
	}
	var info map[string]interface{}
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, err
	}
	return info, nil
}

// InfoList 获取所有匹配元素的信息列表
func (u *UiObject) InfoList() ([]map[string]interface{}, error) {
	raw, err := u.jsonrpc.Call("objInfoOfAllInstances", []interface{}{u.selector.ToMap()})
	if err != nil {
		return nil, err
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// ---------- 边界和坐标 ----------

// Bounds 获取元素的边界坐标 (left, top, right, bottom)
func (u *UiObject) Bounds() (int, int, int, int, error) {
	info, err := u.InfoRaw()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	// 优先使用 visibleBounds
	bounds, ok := info["visibleBounds"].(map[string]interface{})
	if !ok {
		bounds, ok = info["bounds"].(map[string]interface{})
		if !ok {
			return 0, 0, 0, 0, fmt.Errorf("无法获取元素边界")
		}
	}

	lx := int(bounds["left"].(float64))
	ly := int(bounds["top"].(float64))
	rx := int(bounds["right"].(float64))
	ry := int(bounds["bottom"].(float64))
	return lx, ly, rx, ry, nil
}

// Center 获取元素中心坐标
// offset: [xoff, yoff]，(0,0) 表示左上角，(0.5,0.5) 表示中心
func (u *UiObject) Center(offset ...float64) (int, int, error) {
	xoff, yoff := 0.5, 0.5
	if len(offset) >= 2 {
		xoff, yoff = offset[0], offset[1]
	}

	lx, ly, rx, ry, err := u.Bounds()
	if err != nil {
		return 0, 0, err
	}

	width := rx - lx
	height := ry - ly
	x := lx + int(float64(width)*xoff)
	y := ly + int(float64(height)*yoff)
	return x, y, nil
}

// ---------- 点击操作 ----------

// Click 点击 UI 元素
// timeout: 等待元素出现的超时时间（秒），0 使用默认值
func (u *UiObject) Click(timeout ...float64) error {
	t := 0.0
	if len(timeout) > 0 {
		t = timeout[0]
	}

	if err := u.MustWait(t); err != nil {
		return err
	}

	x, y, err := u.Center()
	if err != nil {
		return err
	}
	return u.device.Click(x, y)
}

// ClickWithOffset 带偏移量点击 UI 元素
func (u *UiObject) ClickWithOffset(xoff, yoff float64, timeout ...float64) error {
	t := 0.0
	if len(timeout) > 0 {
		t = timeout[0]
	}

	if err := u.MustWait(t); err != nil {
		return err
	}

	x, y, err := u.Center(xoff, yoff)
	if err != nil {
		return err
	}
	return u.device.Click(x, y)
}

// ClickExists 如果元素存在则点击，返回是否成功
func (u *UiObject) ClickExists(timeout ...float64) bool {
	t := 0.0
	if len(timeout) > 0 {
		t = timeout[0]
	}
	err := u.Click(t)
	return err == nil
}

// ClickGone 持续点击直到元素消失
// maxRetry: 最大重试次数
// interval: 重试间隔（秒）
func (u *UiObject) ClickGone(maxRetry int, interval float64) bool {
	if maxRetry <= 0 {
		maxRetry = 10
	}
	if interval <= 0 {
		interval = 1.0
	}

	u.ClickExists(0)
	for i := 0; i < maxRetry; i++ {
		time.Sleep(time.Duration(interval * float64(time.Second)))
		exists, _ := u.Exists()
		if !exists {
			return true
		}
		u.ClickExists(0)
	}
	return false
}

// LongClick 长按 UI 元素
// duration: 按住时间（秒），默认 0.5
func (u *UiObject) LongClick(duration float64, timeout ...float64) error {
	if duration <= 0 {
		duration = 0.5
	}
	t := 0.0
	if len(timeout) > 0 {
		t = timeout[0]
	}

	if err := u.MustWait(t); err != nil {
		return err
	}

	x, y, err := u.Center()
	if err != nil {
		return err
	}
	return u.device.LongClick(x, y, duration)
}

// ---------- 文本操作 ----------

// GetText 获取元素文本内容
func (u *UiObject) GetText(timeout ...float64) (string, error) {
	t := 0.0
	if len(timeout) > 0 {
		t = timeout[0]
	}
	if err := u.MustWait(t); err != nil {
		return "", err
	}

	raw, err := u.jsonrpc.Call("getText", []interface{}{u.selector.ToMap()})
	if err != nil {
		return "", err
	}
	var text string
	json.Unmarshal(raw, &text)
	return text, nil
}

// SetText 设置元素文本内容
// 如果 text 为空，则清除文本
func (u *UiObject) SetText(text string, timeout ...float64) error {
	t := 0.0
	if len(timeout) > 0 {
		t = timeout[0]
	}
	if err := u.MustWait(t); err != nil {
		return err
	}

	if text == "" {
		_, err := u.jsonrpc.Call("clearTextField", []interface{}{u.selector.ToMap()})
		return err
	}
	_, err := u.jsonrpc.Call("setText", []interface{}{u.selector.ToMap(), text})
	return err
}

// ClearText 清除元素文本
func (u *UiObject) ClearText(timeout ...float64) error {
	return u.SetText("", timeout...)
}

// SendKeys SetText 的别名
func (u *UiObject) SendKeys(text string, timeout ...float64) error {
	return u.SetText(text, timeout...)
}

// ---------- 滑动操作 ----------

// UiSwipe 在元素范围内滑动
// direction: 方向 "left"/"right"/"up"/"down"
// steps: 滑动步数
func (u *UiObject) UiSwipe(direction string, steps int) error {
	if steps <= 0 {
		steps = 10
	}

	if err := u.MustWait(0); err != nil {
		return err
	}

	lx, ly, rx, ry, err := u.Bounds()
	if err != nil {
		return err
	}

	cx := (lx + rx) / 2
	cy := (ly + ry) / 2

	switch direction {
	case "up":
		return u.device.Swipe(cx, cy, cx, ly, steps)
	case "down":
		return u.device.Swipe(cx, cy, cx, ry-1, steps)
	case "left":
		return u.device.Swipe(cx, cy, lx, cy, steps)
	case "right":
		return u.device.Swipe(cx, cy, rx-1, cy, steps)
	default:
		return fmt.Errorf("不支持的方向: %s", direction)
	}
}

// DragTo 将元素拖拽到指定坐标
func (u *UiObject) DragTo(x, y int, duration float64, timeout ...float64) error {
	if duration <= 0 {
		duration = 0.5
	}
	t := 0.0
	if len(timeout) > 0 {
		t = timeout[0]
	}
	if err := u.MustWait(t); err != nil {
		return err
	}

	steps := int(duration * 200)
	_, err := u.jsonrpc.Call("dragTo", []interface{}{u.selector.ToMap(), x, y, steps})
	return err
}

// ---------- 手势操作 ----------

// PinchIn 向内捏合（缩小）
func (u *UiObject) PinchIn(percent, steps int) error {
	if percent <= 0 {
		percent = 100
	}
	if steps <= 0 {
		steps = 50
	}
	_, err := u.jsonrpc.Call("pinchIn", []interface{}{u.selector.ToMap(), percent, steps})
	return err
}

// PinchOut 向外捏合（放大）
func (u *UiObject) PinchOut(percent, steps int) error {
	if percent <= 0 {
		percent = 100
	}
	if steps <= 0 {
		steps = 50
	}
	_, err := u.jsonrpc.Call("pinchOut", []interface{}{u.selector.ToMap(), percent, steps})
	return err
}

// ---------- 子/兄弟元素 ----------

// Child 查找子元素
func (u *UiObject) Child(params map[string]interface{}) (*UiObject, error) {
	sel := u.selector.Clone()
	if _, err := sel.Child(params); err != nil {
		return nil, err
	}
	return NewUiObject(u.device, sel), nil
}

// Sibling 查找兄弟元素
func (u *UiObject) Sibling(params map[string]interface{}) (*UiObject, error) {
	sel := u.selector.Clone()
	if _, err := sel.Sibling(params); err != nil {
		return nil, err
	}
	return NewUiObject(u.device, sel), nil
}

// ChildByText 通过文本查找子元素
func (u *UiObject) ChildByText(text string, params map[string]interface{}) (*UiObject, error) {
	childSel, err := New(params)
	if err != nil {
		return nil, err
	}
	raw, err := u.jsonrpc.Call("childByText", []interface{}{u.selector.ToMap(), childSel.ToMap(), text})
	if err != nil {
		return nil, err
	}
	var resultMap map[string]interface{}
	if err := json.Unmarshal(raw, &resultMap); err != nil {
		return nil, err
	}
	resultSel, err := FromMap(resultMap)
	if err != nil {
		return nil, err
	}
	return NewUiObject(u.device, resultSel), nil
}

// ChildByDescription 通过描述查找子元素
func (u *UiObject) ChildByDescription(desc string, params map[string]interface{}) (*UiObject, error) {
	childSel, err := New(params)
	if err != nil {
		return nil, err
	}
	raw, err := u.jsonrpc.Call("childByDescription", []interface{}{u.selector.ToMap(), childSel.ToMap(), desc})
	if err != nil {
		return nil, err
	}
	var resultMap map[string]interface{}
	if err := json.Unmarshal(raw, &resultMap); err != nil {
		return nil, err
	}
	resultSel, err := FromMap(resultMap)
	if err != nil {
		return nil, err
	}
	return NewUiObject(u.device, resultSel), nil
}

// ---------- 数量和索引 ----------

// Count 获取匹配元素的数量
func (u *UiObject) Count() (int, error) {
	raw, err := u.jsonrpc.Call("count", []interface{}{u.selector.ToMap()})
	if err != nil {
		return 0, err
	}
	var count int
	json.Unmarshal(raw, &count)
	return count, nil
}

// Instance 获取指定索引的元素
func (u *UiObject) Instance(index int) *UiObject {
	sel := u.selector.Clone()
	if index < 0 {
		// 负数索引需要先获取总数
		count, err := u.Count()
		if err == nil && index+count >= 0 {
			index = index + count
		} else {
			index = 0
		}
	}
	sel.UpdateInstance(index)
	return NewUiObject(u.device, sel)
}

// ---------- 滚动操作 ----------

// ScrollForward 向前滚动
func (u *UiObject) ScrollForward(vertical bool, steps int) (bool, error) {
	if steps <= 0 {
		steps = ScrollSteps
	}
	raw, err := u.jsonrpc.Call("scrollForward", []interface{}{u.selector.ToMap(), vertical, steps})
	if err != nil {
		return false, err
	}
	var result bool
	json.Unmarshal(raw, &result)
	return result, nil
}

// ScrollBackward 向后滚动
func (u *UiObject) ScrollBackward(vertical bool, steps int) (bool, error) {
	if steps <= 0 {
		steps = ScrollSteps
	}
	raw, err := u.jsonrpc.Call("scrollBackward", []interface{}{u.selector.ToMap(), vertical, steps})
	if err != nil {
		return false, err
	}
	var result bool
	json.Unmarshal(raw, &result)
	return result, nil
}

// ScrollToBeginning 滚动到开头
func (u *UiObject) ScrollToBeginning(vertical bool, maxSwipes, steps int) (bool, error) {
	if maxSwipes <= 0 {
		maxSwipes = 500
	}
	if steps <= 0 {
		steps = ScrollSteps
	}
	raw, err := u.jsonrpc.Call("scrollToBeginning", []interface{}{u.selector.ToMap(), vertical, maxSwipes, steps})
	if err != nil {
		return false, err
	}
	var result bool
	json.Unmarshal(raw, &result)
	return result, nil
}

// ScrollToEnd 滚动到末尾
func (u *UiObject) ScrollToEnd(vertical bool, maxSwipes, steps int) (bool, error) {
	if maxSwipes <= 0 {
		maxSwipes = 500
	}
	if steps <= 0 {
		steps = ScrollSteps
	}
	raw, err := u.jsonrpc.Call("scrollToEnd", []interface{}{u.selector.ToMap(), vertical, maxSwipes, steps})
	if err != nil {
		return false, err
	}
	var result bool
	json.Unmarshal(raw, &result)
	return result, nil
}

// ScrollTo 滚动到指定元素可见
func (u *UiObject) ScrollTo(targetParams map[string]interface{}, vertical bool) (bool, error) {
	targetSel, err := New(targetParams)
	if err != nil {
		return false, err
	}
	raw, err := u.jsonrpc.Call("scrollTo", []interface{}{u.selector.ToMap(), targetSel.ToMap(), vertical})
	if err != nil {
		return false, err
	}
	var result bool
	json.Unmarshal(raw, &result)
	return result, nil
}

// ---------- Fling 操作 ----------

// FlingForward 向前快速滑动
func (u *UiObject) FlingForward(vertical bool) (bool, error) {
	raw, err := u.jsonrpc.Call("flingForward", []interface{}{u.selector.ToMap(), vertical})
	if err != nil {
		return false, err
	}
	var result bool
	json.Unmarshal(raw, &result)
	return result, nil
}

// FlingBackward 向后快速滑动
func (u *UiObject) FlingBackward(vertical bool) (bool, error) {
	raw, err := u.jsonrpc.Call("flingBackward", []interface{}{u.selector.ToMap(), vertical})
	if err != nil {
		return false, err
	}
	var result bool
	json.Unmarshal(raw, &result)
	return result, nil
}

// FlingToBeginning 快速滑动到开头
func (u *UiObject) FlingToBeginning(vertical bool, maxSwipes int) (bool, error) {
	if maxSwipes <= 0 {
		maxSwipes = 500
	}
	raw, err := u.jsonrpc.Call("flingToBeginning", []interface{}{u.selector.ToMap(), vertical, maxSwipes})
	if err != nil {
		return false, err
	}
	var result bool
	json.Unmarshal(raw, &result)
	return result, nil
}

// FlingToEnd 快速滑动到末尾
func (u *UiObject) FlingToEnd(vertical bool, maxSwipes int) (bool, error) {
	if maxSwipes <= 0 {
		maxSwipes = 500
	}
	raw, err := u.jsonrpc.Call("flingToEnd", []interface{}{u.selector.ToMap(), vertical, maxSwipes})
	if err != nil {
		return false, err
	}
	var result bool
	json.Unmarshal(raw, &result)
	return result, nil
}
