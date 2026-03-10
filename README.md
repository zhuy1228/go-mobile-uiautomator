# go-mobile-uiautomator

一个用 Go 语言实现的 Android 移动设备 UI 自动化工具，通过 ADB 协议与 Android 设备通信，实现应用安装、文件传输和 UI 自动化测试功能。

## 特性

- ✅ **纯 Go 实现** - 不依赖外部 ADB 命令行工具
- ✅ **跨平台** - 支持 Windows/Linux/macOS
- ✅ **轻量级** - 无需安装 Android SDK
- ✅ **低延迟** - 直接 TCP 连接，无中间层
- ✅ **功能完整** - 支持设备发现、文件传输、APK 安装、Shell 执行
- ✅ **UIAutomator2** - 内置 UIAutomator2 服务支持

## 环境要求

- Go 1.25.1+
- 已启动的 ADB 服务器（Android SDK 自带）
- Android 设备已开启 USB 调试

## 安装

### 作为项目引用

```bash
go get github.com/zhuy1228/go-mobile-uiautomator@latest
```

### 从源码构建

```bash
git clone https://github.com/zhuy1228/go-mobile-uiautomator.git
cd go-mobile-uiautomator
go mod download
```

## 快速开始

### 1. 基础设置

确保 ADB 服务器正在运行：

```bash
adb devices
```

### 2. 运行示例

修改 `cmd/main.go` 中的设备配置：

```go
import "github.com/zhuy1228/go-mobile-uiautomator/adb"

const serial = "your-device-serial"  // 你的设备序列号
const addr = "127.0.0.1:5037"        // ADB 服务器地址
```

运行程序：

```bash
go run cmd/main.go
```

### 3. 启动 UIAutomator2 服务

程序会自动：
1. 列出已连接的设备
2. 推送 `u2.jar` 到设备
3. 启动 UIAutomator2 服务
4. 输出服务日志

## 核心功能

### 设备管理

```go
// 列出所有设备
payload, _ := adb.ListDevicesRaw(addr, 15*time.Second)
devices := adb.ParseDevicesPayload(payload)

// 通过 Product 名称查找设备
serial, err := adb.FindSerialByProduct(addr, "23113RKC6C")

// 建立设备连接
conn, err := adb.DialADB(addr, 15*time.Second)
defer conn.Close()
adb.TransportTo(conn, serial)
```

### Shell 命令执行

```go
// 执行 Shell 命令
conn, _ := adb.DialADB(addr, 15*time.Second)
adb.TransportTo(conn, serial)
output, err := adb.ExecShell(conn, "getprop")
fmt.Println(string(output))
```

### 文件传输

```go
// 推送文件到设备
sync := adb.InitSync(conn)
n, err := sync.SyncPushFile(
    "local/path/file.png",     // 本地文件路径
    "/sdcard/remote.png",       // 设备目标路径
    0644,                       // 文件权限
    true,                       // 调试模式
)
```

### APK 安装

```go
// 安装 APK
services.InstallServiceApk(addr, serial)

// 或手动安装
remotePath := "/data/local/tmp/app.apk"
output, err := adb.InstallApkOnDevice(addr, serial, remotePath, "-r", true)
```

### UI 选择器

```go
// 创建选择器
selector := libs.MustNew(map[string]interface{}{
    "className": "android.widget.TextView",
    "text": "登录",
})

// 添加子元素选择
selector.Child(map[string]interface{}{
    "resourceId": "com.example:id/button",
    "instance": 0,
})

// 序列化为 JSON
jsonData, _ := selector.ToJSON()
```

## 项目结构

```
go-mobile-uiautomator/
├── adb/                    # ADB 协议实现
│   ├── connect.go         # 连接管理、命令发送/接收
│   ├── device.go          # 设备发现、APK 安装
│   └── sync.go            # 文件同步协议
├── assets/                 # 资源文件
│   ├── u2.jar             # UIAutomator2 服务端
│   ├── app-uiautomator.apk
│   └── sync.sh
├── cmd/                    # 主程序
│   └── main.go
├── config/                 # 配置管理
│   └── index.go
├── libs/                   # 工具库
│   ├── request.go         # HTTP over ADB
│   └── selector.go        # UI 选择器
├── services/               # 服务模块
│   └── install_service.go # UIAutomator2 安装
├── test/
│   └── test.py
├── config.yaml             # 配置文件
├── go.mod
└── README.md
```

## ADB 协议实现

本项目实现了以下 ADB 协议功能：

| 功能 | 命令 | 说明 |
|------|------|------|
| 设备列表 | `host:devices-l` | 列出所有连接的设备 |
| 设备路由 | `host:transport:<serial>` | 切换到指定设备 |
| Shell 执行 | `shell:<command>` | 执行 Shell 命令 |
| 文件同步 | `sync:` | 启动文件传输协议 |
| 文件推送 | `SEND/DATA/DONE` | Sync 协议传输文件 |

## 配置说明

`config.yaml` 配置文件：

```yaml
appName: "Go Desk"
port: 6997                              # 本地监听端口
wsUrl: "106.12.33.188:6996"            # WebSocket 地址
stunUrl: "stun:106.12.33.188:3478"     # STUN 服务器
apiUrl: "http://106.12.33.188:6996"    # API 地址
siteFileDir: "www"                      # 网页文件目录
```

加载配置：

```go
cfg, err := config.LoadConfig()
if err != nil {
    log.Fatal(err)
}
fmt.Println(cfg.Port, cfg.WsUrl)
```

## 常见问题

### Q: 如何获取设备序列号？

A: 运行 `adb devices` 或使用代码：

```go
payload, _ := adb.ListDevicesRaw("127.0.0.1:5037", 15*time.Second)
devices := adb.ParseDevicesPayload(payload)
for _, dev := range devices {
    fmt.Println("Serial:", dev.Serial)
}
```

### Q: 文件推送失败怎么办？

A: 检查以下几点：
1. 设备已正确连接且授权
2. 目标路径有写入权限（如 `/sdcard/` 需要存储权限）
3. 文件路径使用绝对路径
4. 启用调试模式查看详细日志

### Q: UIAutomator2 服务启动失败？

A: 确保：
1. `assets/u2.jar` 文件存在
2. 设备已获取 root 权限或使用 `/data/local/tmp/` 路径
3. CLASSPATH 环境变量正确设置

## 开发计划

- [ ] 添加完整的错误处理和重连机制
- [ ] 实现 WebSocket 远程控制
- [ ] 支持批量设备管理
- [ ] 添加 UI 自动化测试框架
- [ ] 完善 HTTP over ADB 功能集成
- [ ] 添加设备截图和录屏功能
- [ ] 实现日志记录系统

## 依赖

```go
module github.com/zhuy1228/go-mobile-uiautomator

require (
    gopkg.in/yaml.v3 v3.0.1
)
```

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 作者

zhuy1228

## 致谢

本项目参考了以下开源项目：
- [UIAutomator2](https://github.com/openatx/uiautomator2)
- [Android ADB Protocol](https://android.googlesource.com/platform/packages/modules/adb/)
