// Package adb 实现了 Android Debug Bridge (ADB) 协议的核心功能。
//
// 本包提供了纯 Go 实现的 ADB 客户端，支持以下功能：
//   - 设备发现与连接管理
//   - Shell 命令执行
//   - 文件同步传输（推送文件到设备）
//   - APK 安装
//   - UIAutomator2 服务启动
//
// 使用示例：
//
//	// 连接 ADB 服务器
//	conn, err := adb.DialADB("127.0.0.1:5037", 15*time.Second)
//	defer conn.Close()
//
//	// 路由到指定设备
//	adb.TransportTo(conn, "emulator-5556")
//
//	// 执行 Shell 命令
//	output, err := adb.ExecShell(conn, "getprop ro.product.model")
package adb
