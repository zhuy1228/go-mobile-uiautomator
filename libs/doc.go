// Package libs 提供了与 UIAutomator2 服务交互的工具库。
//
// 本包包含以下核心组件：
//   - AdbHTTPConnection：通过 ADB 隧道发送 HTTP 请求到设备端 UIAutomator2 服务
//   - Selector：UI 元素选择器构造器，支持文本、类名、资源 ID 等多种查询条件
//   - HTTPResponse：HTTP 响应封装
//
// 使用示例：
//
//	// 创建 UI 选择器
//	selector := libs.MustNew(map[string]interface{}{
//	    "text": "登录",
//	    "className": "android.widget.Button",
//	})
//	jsonData, _ := selector.ToJSON()
package libs
