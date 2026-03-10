// Package libs 提供了完整的 Android UIAutomator2 自动化框架。
//
// 本包是 Python uiautomator2 的 Go 语言实现，通过 ADB 协议与运行在 Android 设备上的
// UIAutomator2 HTTP 服务通信，提供设备控制、UI 操作、文本输入等功能。
//
// 核心组件：
//   - Device：设备客户端，管理 UIAutomator2 服务生命周期，提供所有设备操作
//   - UiObject：UI 控件对象，支持点击、输入、滑动、等待等操作
//   - Selector：UI 元素选择器，支持文本、类名、资源 ID 等多种查询条件
//   - JsonRpcWrapper：JSON-RPC 2.0 调用封装
//   - InputMethod：通过 AdbKeyboard 输入法实现快速文本输入
//   - SwipeExt：扩展滑动操作（按方向、比例滑动）
//   - WatchContext/Watcher：弹窗/对话框自动监控和处理
//   - Session：应用会话管理，自动检测应用状态
//   - Settings：设备配置管理（等待超时、操作延迟等）
//
// 通信层：
//   - AdbHTTPConnection：通过 ADB 隧道发送 HTTP 请求
//   - HttpRequest：高层 HTTP 请求封装
//   - JsonRpcCall：JSON-RPC 2.0 请求/响应处理
//
// 使用示例：
//
//	// 创建设备连接
//	device, err := libs.NewDevice("emulator-5554")
//	// 自定义 ADB 地址: libs.NewDevice("emulator-5554", "192.168.1.100:5037")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer device.StopUiautomator()
//
//	// 查找并点击按钮
//	btn, _ := device.FindElement(map[string]interface{}{"text": "登录"})
//	btn.Click()
//
//	// 输入文本
//	input, _ := device.FindElement(map[string]interface{}{"resourceId": "com.example:id/username"})
//	input.SetText("admin")
//
//	// 使用 Watcher 自动处理弹窗
//	watcher := libs.NewWatcher(device)
//	watcher.WhenText("同意").Click()
//	watcher.Start(2.0)
//	defer watcher.Stop()
package libs
