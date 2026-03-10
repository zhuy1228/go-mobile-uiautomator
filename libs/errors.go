package libs

import "fmt"

// ---------- 基础错误类型 ----------

// DeviceError 设备层面的错误基类
type DeviceError struct {
	Message string
}

func (e *DeviceError) Error() string {
	return fmt.Sprintf("设备错误: %s", e.Message)
}

// ConnectError 设备连接失败
type ConnectError struct {
	Message string
}

func (e *ConnectError) Error() string {
	return fmt.Sprintf("连接失败: %s", e.Message)
}

// HTTPError HTTP 请求失败
type HTTPError struct {
	Message string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP 错误: %s", e.Message)
}

// HTTPTimeoutError HTTP 请求超时
type HTTPTimeoutError struct {
	Message string
}

func (e *HTTPTimeoutError) Error() string {
	return fmt.Sprintf("HTTP 超时: %s", e.Message)
}

// AdbShellError ADB Shell 执行失败
type AdbShellError struct {
	Message string
}

func (e *AdbShellError) Error() string {
	return fmt.Sprintf("ADB Shell 错误: %s", e.Message)
}

// ---------- RPC 错误类型 ----------

// RPCError JSON-RPC 调用的错误基类
type RPCError struct {
	Code    int
	Message string
	Data    string // 堆栈信息
	Params  interface{}
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC 错误 [%d]: %s", e.Code, e.Message)
}

// RPCUnknownError 未知的 RPC 错误
type RPCUnknownError struct {
	RPCError
}

// RPCInvalidError 无效的 RPC 响应
type RPCInvalidError struct {
	Message string
}

func (e *RPCInvalidError) Error() string {
	return fmt.Sprintf("RPC 无效响应: %s", e.Message)
}

// RPCStackOverflowError Java 端栈溢出错误
type RPCStackOverflowError struct {
	RPCError
}

// UiObjectNotFoundError UI 元素未找到
type UiObjectNotFoundError struct {
	Code    int
	Message string
	Params  interface{}
}

func (e *UiObjectNotFoundError) Error() string {
	return fmt.Sprintf("UiObject 未找到: %s (参数: %v)", e.Message, e.Params)
}

// UiAutomationNotConnectedError UIAutomation 服务未连接
type UiAutomationNotConnectedError struct {
	Message string
}

func (e *UiAutomationNotConnectedError) Error() string {
	return fmt.Sprintf("UIAutomation 未连接: %s", e.Message)
}

// HierarchyEmptyError dump_hierarchy 返回空结果
type HierarchyEmptyError struct {
	Message string
}

func (e *HierarchyEmptyError) Error() string {
	return fmt.Sprintf("层级为空: %s", e.Message)
}

// ---------- 应用相关错误 ----------

// LaunchUiAutomationError UIAutomator2 服务启动失败
type LaunchUiAutomationError struct {
	Message string
	Output  string
}

func (e *LaunchUiAutomationError) Error() string {
	return fmt.Sprintf("UIAutomator 启动失败: %s\n输出: %s", e.Message, e.Output)
}

// AccessibilityServiceAlreadyRegisteredError 辅助功能服务已注册
type AccessibilityServiceAlreadyRegisteredError struct {
	Output string
}

func (e *AccessibilityServiceAlreadyRegisteredError) Error() string {
	return fmt.Sprintf("辅助功能服务已注册: %s", e.Output)
}

// SessionBrokenError 应用会话中断（应用已退出或崩溃）
type SessionBrokenError struct {
	Message string
}

func (e *SessionBrokenError) Error() string {
	return fmt.Sprintf("会话中断: %s", e.Message)
}

// AppNotFoundError 应用未安装
type AppNotFoundError struct {
	PackageName string
}

func (e *AppNotFoundError) Error() string {
	return fmt.Sprintf("应用未找到: %s", e.PackageName)
}

// InputIMEError 输入法错误
type InputIMEError struct {
	Message string
}

func (e *InputIMEError) Error() string {
	return fmt.Sprintf("输入法错误: %s", e.Message)
}
