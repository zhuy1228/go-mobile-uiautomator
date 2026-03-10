// Package services 提供了 UIAutomator2 服务的安装和部署功能。
//
// 本包负责将 UIAutomator2 相关的资源文件推送到 Android 设备并完成安装：
//   - InstallServiceJar：推送 u2.jar 到设备（自动检查设备端是否已存在，避免重复推送）
//   - InstallServiceApk：推送并安装 UIAutomator2 APK
//
// 通常无需直接调用本包，libs.NewDevice() 会自动完成服务安装和启动。
package services
