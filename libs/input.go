package libs

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ---------- 输入法功能 ----------

// 输入法相关常量
const (
	// imeID AdbKeyboard 输入法的标识符
	imeID = "com.github.uiautomator/.AdbKeyboard"

	// broadcastResultOK 广播成功返回码
	broadcastResultOK = -1
)

// BroadcastResult 广播命令的返回结果
type BroadcastResult struct {
	Code int    // 结果码，-1 表示成功
	Data string // 返回数据
}

// InputMethod 提供输入法相关操作
// 通过 AdbKeyboard 输入法实现快速文本输入
type InputMethod struct {
	device *Device
}

// NewInputMethod 创建输入法操作实例
func NewInputMethod(device *Device) *InputMethod {
	return &InputMethod{device: device}
}

// CurrentIME 获取当前活动的输入法
func (im *InputMethod) CurrentIME() (string, error) {
	resp, err := im.device.Shell("settings", "get", "secure", "default_input_method")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Output), nil
}

// SetInputIME 启用或禁用 AdbKeyboard 输入法
func (im *InputMethod) SetInputIME(enable bool) error {
	if !enable {
		_, err := im.device.Shell("ime", "disable", imeID)
		return err
	}

	// 检查是否已经设置为当前输入法
	current, err := im.CurrentIME()
	if err == nil && current == imeID {
		return nil
	}

	// 检查是否已安装
	if !im.IsInstalled() {
		return &InputIMEError{Message: "AdbKeyboard 输入法未安装，请先安装 app-uiautomator.apk"}
	}

	// 启用并设置为默认输入法
	im.device.Shell("ime", "enable", imeID)
	im.device.Shell("ime", "set", imeID)
	im.device.Shell("settings", "put", "secure", "default_input_method", imeID)

	// 等待输入法就绪
	return im.waitReady()
}

// IsInstalled 检查 AdbKeyboard 输入法是否已安装
func (im *InputMethod) IsInstalled() bool {
	list, _ := im.getIMEList()
	for _, id := range list {
		if id == imeID {
			return true
		}
	}
	return false
}

// getIMEList 获取设备上所有输入法列表
func (im *InputMethod) getIMEList() ([]string, error) {
	resp, err := im.device.Shell("ime", "list", "-s")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(resp.Output), "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

// waitReady 等待输入法就绪
func (im *InputMethod) waitReady() error {
	for i := 0; i < 10; i++ {
		current, err := im.CurrentIME()
		if err == nil && current == imeID {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return &InputIMEError{Message: "等待输入法就绪超时"}
}

// broadcast 发送广播命令
func (im *InputMethod) broadcast(action string, extras map[string]string) (*BroadcastResult, error) {
	args := []string{"am", "broadcast", "-a", action}
	for k, v := range extras {
		args = append(args, "--es", k, v)
	}

	resp, err := im.device.Shell(args...)
	if err != nil {
		return nil, err
	}

	// 解析返回结果
	// 格式: result=-1 data="success"
	result := &BroadcastResult{Code: 0}

	reResult := regexp.MustCompile(`result=(-?\d+)`)
	reData := regexp.MustCompile(`data="([^"]+)"`)

	if m := reResult.FindStringSubmatch(resp.Output); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &result.Code)
	}
	if m := reData.FindStringSubmatch(resp.Output); len(m) > 1 {
		result.Data = m[1]
	}

	return result, nil
}

// mustBroadcast 发送广播并确保成功
func (im *InputMethod) mustBroadcast(action string, extras map[string]string) error {
	var lastErr error
	for i := 0; i < 3; i++ {
		result, err := im.broadcast(action, extras)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(1000+i*500) * time.Millisecond)
			continue
		}
		if result.Code == broadcastResultOK {
			return nil
		}
		lastErr = fmt.Errorf("广播 %s 失败: code=%d data=%s", action, result.Code, result.Data)
		time.Sleep(time.Duration(1000+i*500) * time.Millisecond)
	}
	return lastErr
}

// SendKeys 通过 AdbKeyboard 输入法输入文本
// 自动启用输入法并在输入完成后隐藏键盘
func (im *InputMethod) SendKeys(text string) error {
	if err := im.SetInputIME(true); err != nil {
		return err
	}

	// Base64 编码文本
	encoded := base64.StdEncoding.EncodeToString([]byte(text))

	// 发送文本输入广播
	if err := im.mustBroadcast("ADB_KEYBOARD_INPUT_TEXT", map[string]string{
		"text": encoded,
	}); err != nil {
		return err
	}

	// 隐藏键盘
	im.mustBroadcast("ADB_KEYBOARD_HIDE", nil)
	return nil
}

// SendAction 模拟输入法编辑器动作
// code 为动作代码：
//
//	"go"/"search"/"send"/"next"/"done"/"previous" 或数字
func (im *InputMethod) SendAction(code string) error {
	if err := im.SetInputIME(true); err != nil {
		return err
	}

	// 将名称转换为代码
	actionCodes := map[string]string{
		"go":       "2",
		"search":   "3",
		"send":     "4",
		"next":     "5",
		"done":     "6",
		"previous": "7",
	}

	codeStr := code
	if mapped, ok := actionCodes[strings.ToLower(code)]; ok {
		codeStr = mapped
	}

	return im.mustBroadcast("ADB_KEYBOARD_EDITOR_CODE", map[string]string{
		"code": codeStr,
	})
}

// ClearText 通过输入法清除文本
func (im *InputMethod) ClearText() error {
	if err := im.SetInputIME(true); err != nil {
		return err
	}
	return im.mustBroadcast("ADB_KEYBOARD_CLEAR_TEXT", nil)
}
