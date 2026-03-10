package libs

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ---------- WatchContext：简化版弹窗监控 ----------

// WatchCondition 定义一个监控条件和对应的操作
type WatchCondition struct {
	// Selectors 匹配条件列表（xpath 或文本），全部匹配时触发
	Selectors []map[string]interface{}
	// Callback 匹配后的回调操作
	Callback func(d *Device) error
}

// WatchContext 提供 UI 弹窗/对话框的自动监控和处理
// 对应 Python 版本的 WatchContext 和 Watcher
type WatchContext struct {
	device *Device

	// 监控条件列表
	conditions []WatchCondition

	// 当前正在构建的条件
	pendingSelectors []map[string]interface{}

	// 状态管理
	mu          sync.Mutex
	stopCh      chan struct{}
	stopped     chan struct{}
	started     bool
	triggerTime time.Time

	// 检查间隔（秒）
	interval float64
}

// NewWatchContext 创建一个新的监控上下文
// builtin: 是否添加内置的中文弹窗处理规则
func NewWatchContext(device *Device, builtin bool) *WatchContext {
	wc := &WatchContext{
		device:      device,
		conditions:  []WatchCondition{},
		interval:    2.0,
		triggerTime: time.Now(),
	}

	if builtin {
		// 添加常见的中文弹窗自动处理规则
		wc.WhenText("继续使用").Click()
		wc.WhenText("同意").Click()
		wc.WhenText("确定").Click()
		wc.WhenText("好的").Click()
		wc.WhenText("继续安装").Click()
		wc.WhenText("安装").Click()
		wc.WhenText("Agree").Click()
		wc.WhenText("ALLOW").Click()
	}

	return wc
}

// WhenText 添加按文本匹配的监控条件（支持链式调用）
func (wc *WatchContext) WhenText(text string) *WatchContext {
	wc.pendingSelectors = append(wc.pendingSelectors, map[string]interface{}{
		"text": text,
	})
	return wc
}

// WhenDescription 添加按描述匹配的监控条件
func (wc *WatchContext) WhenDescription(desc string) *WatchContext {
	wc.pendingSelectors = append(wc.pendingSelectors, map[string]interface{}{
		"description": desc,
	})
	return wc
}

// WhenResourceID 添加按资源 ID 匹配的监控条件
func (wc *WatchContext) WhenResourceID(id string) *WatchContext {
	wc.pendingSelectors = append(wc.pendingSelectors, map[string]interface{}{
		"resourceId": id,
	})
	return wc
}

// Click 为当前待处理的条件设置点击操作
func (wc *WatchContext) Click() {
	if len(wc.pendingSelectors) == 0 {
		return
	}

	selectors := make([]map[string]interface{}, len(wc.pendingSelectors))
	copy(selectors, wc.pendingSelectors)
	wc.pendingSelectors = nil

	wc.conditions = append(wc.conditions, WatchCondition{
		Selectors: selectors,
		Callback: func(d *Device) error {
			// 点击最后一个匹配的选择器
			lastSel := selectors[len(selectors)-1]
			obj, err := d.FindElement(lastSel)
			if err != nil {
				return err
			}
			return obj.Click(0)
		},
	})
}

// Press 为当前待处理的条件设置按键操作
func (wc *WatchContext) Press(key string) {
	if len(wc.pendingSelectors) == 0 {
		return
	}

	selectors := make([]map[string]interface{}, len(wc.pendingSelectors))
	copy(selectors, wc.pendingSelectors)
	wc.pendingSelectors = nil

	wc.conditions = append(wc.conditions, WatchCondition{
		Selectors: selectors,
		Callback: func(d *Device) error {
			return d.Press(key)
		},
	})
}

// Call 为当前待处理的条件设置自定义回调
func (wc *WatchContext) Call(fn func(d *Device) error) {
	if len(wc.pendingSelectors) == 0 {
		return
	}

	selectors := make([]map[string]interface{}, len(wc.pendingSelectors))
	copy(selectors, wc.pendingSelectors)
	wc.pendingSelectors = nil

	wc.conditions = append(wc.conditions, WatchCondition{
		Selectors: selectors,
		Callback:  fn,
	})
}

// ---------- 运行控制 ----------

// Start 开始后台监控
func (wc *WatchContext) Start() {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if wc.started {
		return
	}
	wc.started = true
	wc.stopCh = make(chan struct{})
	wc.stopped = make(chan struct{})

	go wc.runForever()
}

// Stop 停止监控
func (wc *WatchContext) Stop() {
	wc.mu.Lock()
	if !wc.started {
		wc.mu.Unlock()
		return
	}
	close(wc.stopCh)
	wc.mu.Unlock()

	// 等待停止
	select {
	case <-wc.stopped:
	case <-time.After(10 * time.Second):
	}

	wc.mu.Lock()
	wc.started = false
	wc.mu.Unlock()
}

// Running 检查是否正在运行
func (wc *WatchContext) Running() bool {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	return wc.started
}

// runForever 持续监控循环
func (wc *WatchContext) runForever() {
	defer close(wc.stopped)

	ticker := time.NewTicker(time.Duration(wc.interval * float64(time.Second)))
	defer ticker.Stop()

	for {
		select {
		case <-wc.stopCh:
			return
		case <-ticker.C:
			wc.runOnce()
		}
	}
}

// runOnce 执行一次监控检查
func (wc *WatchContext) runOnce() bool {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	for _, cond := range wc.conditions {
		allMatched := true
		for _, sel := range cond.Selectors {
			exists, err := wc.device.Exists(sel)
			if err != nil || !exists {
				allMatched = false
				break
			}
		}

		if allMatched {
			log.Printf("[Watcher] 条件匹配，执行回调")
			if err := cond.Callback(wc.device); err != nil {
				log.Printf("[Watcher] 回调执行失败: %v", err)
			}
			wc.triggerTime = time.Now()
			return true
		}
	}
	return false
}

// WaitStable 等待直到监控不再触发（稳定状态）
// stableSeconds: 稳定时间（秒）
// timeout: 超时时间（秒）
func (wc *WatchContext) WaitStable(stableSeconds, timeout float64) error {
	if stableSeconds <= 0 {
		stableSeconds = 5.0
	}
	if timeout <= 0 {
		timeout = 60.0
	}

	if !wc.started {
		wc.Start()
	}

	deadline := time.Now().Add(time.Duration(timeout * float64(time.Second)))
	for time.Now().Before(deadline) {
		wc.mu.Lock()
		stable := time.Since(wc.triggerTime).Seconds() > stableSeconds
		wc.mu.Unlock()

		if stable {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("等待稳定超时")
}

// Reset 停止并移除所有监控条件
func (wc *WatchContext) Reset() {
	if wc.started {
		wc.Stop()
	}
	wc.conditions = nil
}

// Remove 移除所有监控条件
func (wc *WatchContext) Remove() {
	wc.conditions = nil
}
