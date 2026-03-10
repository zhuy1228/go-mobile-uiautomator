// Package libs 提供了完整的 Android UIAutomator2 自动化框架。
//
// 本包是 Python uiautomator2 的 Go 语言实现，通过 ADB 隧道协议与运行在 Android 设备上的
// UIAutomator2 HTTP 服务直连，提供设备控制、UI 操作、文本输入等功能。
// 零外部依赖，仅使用 Go 标准库。
//
// 核心组件：
//   - Device：设备客户端，管理 UIAutomator2 服务生命周期，提供所有设备操作
//   - UiObject：UI 控件对象，支持点击、输入、滑动、等待、滚动等操作
//   - Selector：UI 元素选择器，支持文本、类名、资源 ID、描述等多种查询条件
//   - JsonRpcWrapper：JSON-RPC 2.0 调用封装，自动错误映射和重试
//   - InputMethod：通过 AdbKeyboard 输入法实现快速文本输入（支持中文）
//   - SwipeExt：扩展滑动操作（按方向、比例滑动）
//   - WatchContext：弹窗/对话框自动监控和处理
//   - Session：应用会话管理，自动检测应用存活状态
//   - Settings：线程安全的设备配置管理（等待超时、操作延迟等）
//
// 通信层：
//   - AdbTunnelDevice：通过 ADB 隧道直连设备，与 Python uiautomator2 方案完全一致
//   - http.Transport：自定义 DialContext 将 HTTP 连接替换为 ADB 隧道
//   - HttpRequest：高层 HTTP 请求封装
//   - JsonRpcCall：JSON-RPC 2.0 请求/响应处理
//
// 使用示例：
//
//	// 创建设备连接（自动推送 u2.jar、启动服务）
//	d, err := libs.NewDevice("emulator-5554")
//	// 自定义 ADB 地址: libs.NewDevice("emulator-5554", "192.168.1.100:5037")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer d.Close()
//
//	// 便捷选择器，链式调用
//	d.ByText("登录").Click()
//	d.ByResourceId("com.example:id/input").SetText("admin")
//
//	// 多条件组合查找
//	d.By(libs.P{"className": "android.widget.Button", "text": "确定"}).Click()
//
//	// 使用 AdbKeyboard 输入中文
//	im := libs.NewInputMethod(d)
//	im.SendKeys("你好世界")
//
//	// 使用 Watcher 自动处理弹窗
//	w := libs.NewWatchContext(d, true)
//	w.WhenText("允许").Click()
//	w.Start()
//	defer w.Stop()
package libs
