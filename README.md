# Go Mobile UIAutomator

<p align="center">
    <a href="https://github.com/zhuy1228/go-mobile-uiautomator">
        <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version">
    </a>
    <a href="LICENSE">
        <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
    </a>
    <a href="#">
        <img src="https://img.shields.io/badge/dependencies-0-brightgreen?style=flat-square" alt="Zero Dependencies">
    </a>
</p>

Go 语言实现的 Android UIAutomator2 客户端库。纯 Go 实现，零外部依赖，通过 ADB 隧道协议与 Android 设备直连，提供完整的 UI 自动化测试能力。

参考 Python [openatx/uiautomator2](https://github.com/openatx/uiautomator2) 项目实现，连接方案与 Python 版本完全一致。

## 功能特性

- ✅ **ADB 隧道** - 纯 Go 实现 ADB TCP 协议，通过 ADB 隧道直连设备，无需端口转发
- ✅ **资源内嵌** - 通过 `go:embed` 将 u2.jar/APK 编译进二进制，`go get` 即用，无需额外文件
- ✅ **设备管理** - 设备发现、连接、信息查询
- ✅ **UIAutomator2 服务** - 自动启动和管理 UIAutomator2 服务端
- ✅ **UI 元素操作** - 点击、滑动、输入、拖拽、手势等完整 UI 操作
- ✅ **元素定位** - 支持 text、resourceId、className、description 等多种定位方式
- ✅ **便捷选择器** - `d.ByText("登录").Click()` Python 风格链式调用
- ✅ **输入法集成** - AdbKeyboard 输入法支持，支持中文输入
- ✅ **Watcher 监控** - 自动监控并处理弹窗（权限提示、广告等）
- ✅ **文件传输** - 通过 ADB Sync 协议推送文件
- ✅ **应用管理** - 启动、停止、卸载、清除应用数据
- ✅ **截图功能** - 设备屏幕截图
- ✅ **剪贴板** - 读写设备剪贴板
- ✅ **Session 会话** - 应用会话管理，自动检测应用存活状态
- ✅ **配置管理** - 线程安全的设备参数配置
- ✅ **错误体系** - 完整的错误类型定义和自动重试机制
- ✅ **JSON-RPC** - 完整的 JSON-RPC 2.0 通信层

## 安装

```bash
go get github.com/zhuy1228/go-mobile-uiautomator
```

## 快速开始

### 连接设备

```go
package main

import (
    "fmt"
    "github.com/zhuy1228/go-mobile-uiautomator/libs"
)

func main() {
    // 创建设备连接（自动推送 u2.jar、启动 UIAutomator2 服务）
    d, err := libs.NewDevice("设备序列号")
    if err != nil {
        panic(err)
    }
    defer d.Close()

    // 自定义 ADB 地址
    // d, err := libs.NewDevice("设备序列号", "192.168.1.100:5037")

    // 获取设备信息
    info, err := d.Info()
    if err != nil {
        panic(err)
    }
    fmt.Printf("屏幕尺寸: %v x %v\n", info["displayWidth"], info["displayHeight"])
}
```

### 设备发现

```go
import (
    "fmt"
    "time"
    "github.com/zhuy1228/go-mobile-uiautomator/adb"
    "github.com/zhuy1228/go-mobile-uiautomator/libs"
)

// 列出所有连接的设备
payload, err := adb.ListDevicesRaw(libs.DefaultADBAddr, 15*time.Second)
devices := adb.ParseDevicesPayload(payload)
for _, dev := range devices {
    fmt.Printf("设备: %s 状态: %s\n", dev.Serial, dev.State)
}
```

### 应用管理

```go
// 启动应用（stop=true 先停止再启动）
d.AppStart("com.example.app", "", true)

// 启动应用并指定 Activity
d.AppStart("com.example.app", ".MainActivity", false)

// 获取当前运行的应用信息
info, _ := d.AppCurrent()
fmt.Printf("当前应用: %s PID: %s\n", info["package"], info["pid"])

// 等待应用启动（超时 20 秒，front=true 等待应用到前台）
pid, err := d.AppWait("com.example.app", 20.0, true)

// 停止应用
d.AppStop("com.example.app")

// 清除应用数据
d.AppClear("com.example.app")

// 卸载应用
d.AppUninstall("com.example.app")
```

### UI 元素查找

```go
// ===== 便捷方法（推荐） =====

// 通过 text 查找并操作
d.ByText("登录").Click()

// 通过 textContains 模糊查找
d.ByTextContains("登").Click()

// 通过 resourceId 查找
d.ByResourceId("com.example:id/btn_login").Click()

// 通过 description 查找
d.ByDescription("搜索").Click()

// 通过 className 查找
d.ByClassName("android.widget.EditText").SetText("Hello")

// 多条件组合查找
el := d.By(libs.P{
    "className": "android.widget.TextView",
    "text":      "确定",
})
el.Click()

// ===== FindElement 方法 =====

el, err := d.FindElement(map[string]interface{}{
    "resourceId": "com.example:id/btn_login",
})
if err != nil {
    panic(err)
}
el.Click()

// 检查元素是否存在
exists, _ := d.Exists(map[string]interface{}{"text": "登录"})
```

### 等待操作

```go
el := d.ByText("登录")

// 等待元素出现（超时 10 秒）
found, _ := el.Wait(true, 10.0)

// 等待元素消失
gone, _ := el.WaitGone(10.0)

// 等待元素出现，不存在则返回错误
err := el.MustWait(10.0)

// 设置全局隐式等待超时
d.ImplicitlyWait(20.0)
```

### 点击操作

```go
// 坐标点击
d.Click(500, 800)

// 双击
d.DoubleClick(500, 800, 0)

// 长按坐标（2 秒）
d.LongClick(500, 800, 2.0)

// 元素点击
d.ByText("确定").Click()

// 带偏移的点击（相对元素宽高的比例，0.5 = 中心）
d.ByText("确定").ClickWithOffset(0.5, 0.5)

// 点击元素（如果存在），返回是否成功
clicked := d.ByText("确定").ClickExists(5.0)

// 持续点击直到元素消失（最多重试 5 次，间隔 1 秒）
d.ByText("确定").ClickGone(5, 1.0)

// 元素长按（1.5 秒）
d.ByText("确定").LongClick(1.5)
```

### 文本输入

```go
el := d.ByResourceId("com.example:id/input")

// 获取元素文本
text, _ := el.GetText()

// 设置文本（先清空再输入）
el.SetText("Hello World")

// 清除文本
el.ClearText()

// 追加文本
el.SendKeys("additional text")

// 使用 AdbKeyboard 输入法（支持中文）
im := libs.NewInputMethod(d)
im.SendKeys("你好世界")
im.ClearText()        // 清除文本
im.SendAction("done") // 模拟输入法动作
```

### 滑动操作

```go
// 坐标滑动（steps 控制速度，每步约 5ms）
d.Swipe(500, 1500, 500, 500, 55)

// 按持续时间滑动（0.5 秒）
d.SwipeWithDuration(500, 1500, 500, 500, 0.5)

// 多点连续滑动
d.SwipePoints([][2]int{{500, 1500}, {500, 1000}, {500, 500}}, 0.2)

// 使用 SwipeExt 扩展滑动
se := libs.NewSwipeExt(d)
se.Up(0.8, 55)    // 向上滑动 80% 屏幕
se.Down(0.5, 55)  // 向下滑动 50% 屏幕
se.Left(0.6, 55)  // 向左滑动 60% 屏幕
se.Right(0.4, 55) // 向右滑动 40% 屏幕

// 元素内滑动
d.ByClassName("android.widget.ListView").UiSwipe("up", 55)
d.ByClassName("android.widget.ListView").UiSwipe("down", 55)

// 拖拽
d.Drag(100, 200, 500, 600, 0.5)
d.ByText("拖拽我").DragTo(500, 600, 0.5)
```

### 滚动查找

```go
list := d.ByClassName("android.widget.ListView")

// 向前/向后滚动
list.ScrollForward(true, 55)   // 垂直向前
list.ScrollBackward(true, 55)  // 垂直向后

// 滚动到顶部/底部
list.ScrollToBeginning(true, 10, 55) // 最多 10 次
list.ScrollToEnd(true, 10, 55)

// 滚动直到目标元素可见
list.ScrollTo(map[string]interface{}{"text": "目标文本"}, true)

// 快速滑动（Fling）
list.FlingForward(true)
list.FlingBackward(true)
list.FlingToBeginning(true, 10)
list.FlingToEnd(true, 10)
```

### 手势操作

```go
el := d.ByResourceId("com.example:id/image")

// 捏合（缩小）
el.PinchIn(50, 10)

// 捏合（放大）
el.PinchOut(50, 10)
```

### 子元素和兄弟元素

```go
// 查找子元素
parent := d.ByClassName("android.widget.LinearLayout")
child, _ := parent.Child(map[string]interface{}{"className": "android.widget.Button"})
child.Click()

// 查找兄弟元素
sibling, _ := d.ByText("标题").Sibling(map[string]interface{}{"className": "android.widget.Button"})
sibling.Click()

// 通过文本在子元素中搜索
item, _ := parent.ChildByText("目标文本", map[string]interface{}{"className": "android.widget.TextView"})

// 获取匹配元素数量
count, _ := d.ByClassName("android.widget.Button").Count()

// 获取指定索引的元素
d.ByClassName("android.widget.Button").Instance(0).Click()  // 第一个
d.ByClassName("android.widget.Button").Instance(-1).Click() // 最后一个
```

### Watcher 弹窗监控

```go
// 创建 Watcher 上下文（builtin=true 自动添加常见弹窗规则）
w := libs.NewWatchContext(d, true)

// 添加监控规则：点击"同意"按钮
w.WhenText("同意").Click()

// 添加监控规则：点击"允许"按钮
w.WhenText("ALLOW").Click()

// 添加监控规则：按下返回键关闭弹窗
w.WhenText("取消").Press("back")

// 添加监控规则：自定义回调
w.WhenDescription("广告").Call(func(d *libs.Device) error {
    fmt.Println("检测到广告弹窗")
    return d.Press("back")
})

// 启动监控（后台运行）
w.Start()

// 等待页面稳定（5 秒内无弹窗，超时 30 秒）
w.WaitStable(5.0, 30.0)

// 停止监控
w.Stop()
```

### Session 会话管理

```go
// 创建应用会话（attach=false 启动新实例）
session, err := libs.NewSession(d, "com.example.app", false)
if err != nil {
    panic(err)
}

// 检查应用是否仍在运行
if session.Running() {
    fmt.Println("应用运行中, PID:", session.PID())
}

// 会话中的 JSON-RPC 调用会自动检查应用状态
// 如果应用退出，会返回 SessionBrokenError

// 重启应用
session.Restart()

// 关闭会话
session.Close()
```

### 截图

```go
// 获取截图（JPEG 格式字节数据）
imgData, err := d.Screenshot()
if err != nil {
    panic(err)
}

// 保存到文件
os.WriteFile("screenshot.png", imgData, 0644)
```

### 设备控制

```go
// 亮屏/息屏
d.ScreenOn()
d.ScreenOff()

// 解锁屏幕
d.Unlock()

// 按键操作
d.Press("home")
d.Press("back")
d.Press("recent")
d.Press("volume_up")
d.Press("volume_down")
d.Press("power")
d.Press("enter")

// 发送按键码
d.PressKeyCode(4, 0)  // KEYCODE_BACK

// 长按按键
d.LongPress("power")

// 发送按键事件
d.Keyevent("KEYCODE_MENU")

// 屏幕方向
orientation, _ := d.Orientation()     // 获取当前方向
d.SetOrientation("natural")           // 竖屏
d.SetOrientation("left")              // 左横屏
d.FreezeRotation(true)                // 冻结旋转

// 打开通知栏/快速设置
d.OpenNotification()
d.OpenQuickSettings()

// 打开 URL
d.OpenURL("https://www.example.com")

// 剪贴板
d.SetClipboard("复制的文本")
text, _ := d.GetClipboard()

// Toast
d.MakeToast("提示消息", 2000)  // 2000 毫秒
toast, _ := d.GetLastToast()
d.ClearToast()

// 获取页面层级结构
hierarchy, _ := d.DumpHierarchy(false, 50)

// 执行 Shell 命令
result, _ := d.Shell("ls", "/sdcard/")
fmt.Println(result.Output)

// 获取窗口大小
w, h, _ := d.WindowSize()
fmt.Printf("窗口大小: %d x %d\n", w, h)

// 获取设备 WLAN IP
ip, _ := d.WlanIP()

// 推送文件到设备
d.Push("local/file.txt", "/sdcard/file.txt")
```

### 配置管理

```go
// 设置等待超时（秒）
d.Settings().Set("wait_timeout", 30.0)

// 获取配置
timeout := d.Settings().GetFloat64("wait_timeout")

// 设置操作延迟
d.Settings().Set("operation_delay", []float64{0.5, 1.0})  // [操作前延迟, 操作后延迟]

// 设置需要延迟的操作方法
d.Settings().Set("operation_delay_methods", []string{"click", "swipe"})

// 设置隐式等待超时
d.ImplicitlyWait(20.0)
```

### UI 选择器

```go
// 创建选择器
selector := libs.MustNew(map[string]interface{}{
    "className": "android.widget.TextView",
    "text":      "登录",
})

// 添加子元素选择
selector.Child(map[string]interface{}{
    "resourceId": "com.example:id/button",
    "instance":   0,
})

// 序列化为 JSON
jsonData, _ := selector.ToJSON()
```

### UIAutomator2 服务部署

```go
import "github.com/zhuy1228/go-mobile-uiautomator/services"

// 安装 UIAutomator2 JAR（自动从内嵌资源推送，检查设备端是否已存在）
err := services.InstallServiceJar(addr, serial, false)

// 安装 UIAutomator2 APK（自动从内嵌资源推送）
err = services.InstallServiceApk(addr, serial, false)

// 强制重新安装（force=true 跳过存在检查）
err = services.InstallServiceJar(addr, serial, true)
```

> 资源文件通过 `go:embed` 编译进二进制，无需在本地维护 JAR/APK 文件。

## 项目结构

```
go-mobile-uiautomator/
├── adb/                        # ADB 协议实现
│   ├── connect.go             # 连接管理、ADB 隧道、命令发送/接收
│   ├── device.go              # 设备发现、APK 安装
│   ├── sync.go                # 文件同步协议（Sync Push）
│   └── doc.go                 # 包文档
├── libs/                       # UIAutomator2 客户端核心库
│   ├── device.go              # 核心设备客户端（连接、操作、应用管理）
│   ├── uiobject.go            # UI 元素操作（点击、输入、滑动、滚动）
│   ├── jsonrpc.go             # JSON-RPC 2.0 通信层
│   ├── selector.go            # UI 选择器（定位条件构建）
│   ├── request.go             # ADB 隧道 + 自定义 HTTP Transport
│   ├── input.go               # AdbKeyboard 输入法集成
│   ├── watcher.go             # Watcher 弹窗自动监控
│   ├── session.go             # 应用会话管理
│   ├── swipe_ext.go           # 扩展滑动操作
│   ├── settings.go            # 线程安全配置管理
│   ├── errors.go              # 完整错误类型定义
│   ├── proto.go               # 协议常量和枚举
│   └── doc.go                 # 包文档
├── services/                   # 服务模块
│   ├── install_service.go     # UIAutomator2 安装服务
│   └── doc.go                 # 包文档
├── assets/                     # 资源文件（通过 go:embed 编译进二进制）
│   ├── embed.go               # go:embed 指令，导出 JarData/ApkData/ApkTestData
│   ├── u2.jar                 # UIAutomator2 服务端 JAR
│   ├── app-uiautomator.apk   # UIAutomator2 APK
│   ├── app-uiautomator-test.apk # UIAutomator2 测试 APK
│   └── sync.sh               # 资源同步脚本
├── cmd/                        # 主程序入口
│   └── main.go
├── go.mod                      # Go 模块定义（零外部依赖）
└── README.md
```

## 架构说明

```
┌─────────────────────────────────────────────────┐
│                 用户代码                          │
├─────────────────────────────────────────────────┤
│  libs.Device / libs.UiObject / libs.Session     │
│  (设备操作 / UI 元素 / 会话管理)                   │
├─────────────────────────────────────────────────┤
│  libs.JsonRpcWrapper                            │
│  (JSON-RPC 2.0 通信层)                           │
├─────────────────────────────────────────────────┤
│  libs.AdbTunnelDevice + http.Transport          │
│  (ADB 隧道 + 标准 HTTP 客户端)                    │
├─────────────────────────────────────────────────┤
│  adb.CreateTunnel (纯 Go ADB 协议)               │
│  (TCP → ADB Server → transport → tcp:9008)     │
├─────────────────────────────────────────────────┤
│  assets.JarData / assets.ApkData (go:embed)     │
│  (资源文件编译进二进制，零配置部署)                  │
├─────────────────────────────────────────────────┤
│  Android 设备                                    │
│  UIAutomator2 HTTP 服务 (端口 9008)              │
└─────────────────────────────────────────────────┘
```

### 连接原理

与 Python uiautomator2 采用完全相同的 ADB 隧道方案：

1. **每次 HTTP 请求**通过 `adb.CreateTunnel()` 建立一条 ADB 隧道到设备端口 9008
2. `http.Transport` 的 `DialContext` 被替换为 ADB 隧道连接
3. Go 标准 `http.Client` 在隧道上发送完整的 HTTP/1.1 请求
4. 请求完成后隧道自动关闭，**无需 `adb forward`，不占用本地端口**

```
Go http.Client → http.Transport.DialContext → adb.CreateTunnel()
    ↓                                              ↓
HTTP/1.1 请求  ←→  ADB Server  ←→  设备:9008 (UIAutomator2)
```

## ADB 协议实现

本项目实现了以下 ADB 协议功能：

| 功能 | 命令 | 说明 |
|------|------|------|
| 设备列表 | `host:devices-l` | 列出所有连接的设备 |
| 设备路由 | `host:transport:<serial>` | 切换到指定设备 |
| Shell 执行 | `shell:<command>` | 执行 Shell 命令 |
| ADB 隧道 | `tcp:<port>` | 建立到设备端口的直连隧道 |
| 文件同步 | `sync:` | 启动文件传输协议 |
| 文件推送 | `SEND/DATA/DONE` | Sync 协议传输文件 |

## 错误类型

本库定义了完整的错误类型层次，便于精确错误处理：

| 错误类型 | 说明 |
|---------|------|
| `DeviceError` | 设备通用错误 |
| `ConnectError` | 连接错误 |
| `HTTPError` | HTTP 请求错误 |
| `HTTPTimeoutError` | HTTP 超时错误 |
| `AdbShellError` | ADB Shell 执行错误 |
| `RPCError` | JSON-RPC 调用错误 |
| `RPCUnknownError` | 未知 RPC 错误 |
| `RPCInvalidError` | 无效 RPC 响应 |
| `RPCStackOverflowError` | Java 端栈溢出 |
| `UiObjectNotFoundError` | UI 元素未找到 |
| `UiAutomationNotConnectedError` | UIAutomation 服务断开 |
| `HierarchyEmptyError` | UI 层级为空 |
| `LaunchUiAutomationError` | UIAutomator2 启动失败 |
| `SessionBrokenError` | 应用会话中断 |
| `AppNotFoundError` | 应用未安装 |
| `InputIMEError` | 输入法错误 |

## 常见问题

### Q: 如何获取设备序列号？

运行 `adb devices` 或使用代码：

```go
payload, _ := adb.ListDevicesRaw(libs.DefaultADBAddr, 15*time.Second)
devices := adb.ParseDevicesPayload(payload)
for _, dev := range devices {
    fmt.Println("Serial:", dev.Serial)
}
```

### Q: UIAutomator2 服务启动失败？

确保：
1. 设备已正确连接且已授权 USB 调试
2. ADB Server 正在运行（默认 `127.0.0.1:5037`）
3. 使用 `NewDevice()` 会自动从内嵌资源推送 JAR/APK 并启动服务，无需手动管理

> u2.jar 和 APK 通过 `go:embed` 编译进二进制，`go get` 后即可直接使用，无需额外下载资源文件。

### Q: 如何处理弹窗干扰自动化？

使用 Watcher 机制自动处理弹窗：

```go
w := libs.NewWatchContext(d, true) // builtin=true 自带常见弹窗规则
w.WhenText("允许").Click()
w.Start()
defer w.Stop()
```

### Q: 如何输入中文？

使用 AdbKeyboard 输入法：

```go
im := libs.NewInputMethod(d)
im.SendKeys("你好世界")
```

### Q: 连接已有服务（不启动新服务）？

```go
// 适用于 UIAutomator2 已经在设备上运行的场景
d := libs.NewDeviceWithoutStart("emulator-5554")
d.ByText("Hello").Click()
```

## 开发计划

- [x] 纯 Go ADB 协议实现
- [x] go:embed 资源内嵌（零配置部署）
- [x] ADB 隧道直连（与 Python 一致）
- [x] 设备发现和连接管理
- [x] JSON-RPC 2.0 通信层
- [x] 完整 UI 元素操作（点击、滑动、输入、拖拽）
- [x] 便捷选择器（ByText、ByResourceId 等）
- [x] Watcher 弹窗自动监控
- [x] Session 会话管理
- [x] AdbKeyboard 输入法集成
- [x] 截图功能
- [x] 应用生命周期管理
- [x] 完整错误类型体系
- [x] 线程安全配置管理
- [ ] XPath 选择器支持
- [ ] 设备录屏功能
- [ ] 批量设备管理
- [ ] 完整单元测试

## 依赖

**零外部依赖** — 仅使用 Go 标准库。

```go
module github.com/zhuy1228/go-mobile-uiautomator

go 1.25.1
```

## 许可证

MIT License

## 致谢

本项目参考了以下开源项目：
- [openatx/uiautomator2](https://github.com/openatx/uiautomator2) — Python Android 自动化框架
- [Android ADB Protocol](https://android.googlesource.com/platform/packages/modules/adb/) — ADB 协议规范
