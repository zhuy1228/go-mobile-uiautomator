// Package adb 实现了 Android Debug Bridge (ADB) 协议的核心功能。
//
// 本包提供了纯 Go 实现的 ADB 客户端，零外部依赖，支持以下功能：
//   - 设备发现与连接管理
//   - Shell 命令执行
//   - ADB 隧道（transport + tcp:<port>，与 Python adbutils 一致）
//   - 文件同步传输（Sync Push 协议）
//   - APK 安装
//
// ADB 隧道是本包的核心功能，用于替代 adb forward 端口转发。
// 每次调用 CreateTunnel 会建立一条直通设备端口的原始 TCP 管道，
// 无需占用本地端口，无需额外清理。
//
// 使用示例：
//
//	// 列出所有设备
//	payload, _ := adb.ListDevicesRaw("127.0.0.1:5037", 15*time.Second)
//	devices := adb.ParseDevicesPayload(payload)
//
//	// 建立 ADB 隧道到设备端口 9008
//	conn, err := adb.CreateTunnel("127.0.0.1:5037", "emulator-5554", 9008)
//	defer conn.Close()
//	// conn 现在是一条直通设备 9008 端口的 TCP 管道
//
//	// 推送文件到设备
//	adb.PushFile("127.0.0.1:5037", "emulator-5554", "local.txt", "/sdcard/remote.txt", 0644, false)
package adb
