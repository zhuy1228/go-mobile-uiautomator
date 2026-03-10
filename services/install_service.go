package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/zhuy1228/go-mobile-uiautomator/adb"
	"github.com/zhuy1228/go-mobile-uiautomator/assets"
)

// 设备端路径常量
const (
	// DefaultJarRemote 默认设备端 JAR 路径
	DefaultJarRemote = "/data/local/tmp/u2.jar"
	// DefaultApkRemote 默认设备端 APK 临时路径
	DefaultApkRemote = "/data/local/tmp/app-uiautomator.apk"
)

// fileExistsOnDevice 通过 ADB shell 检查设备端文件是否已存在
// 返回 true 表示文件已存在，不需要重新推送
func fileExistsOnDevice(addr, serial, remotePath string) bool {
	conn, err := adb.ConnectToDevice(addr, serial, 10*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// 使用 ls 检查文件是否存在
	out, err := adb.ExecShell(conn, fmt.Sprintf("ls %s 2>/dev/null", remotePath))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == remotePath
}

// InstallServiceJar 将内嵌的 u2.jar 推送到设备的 /data/local/tmp/ 目录
// JAR 文件通过 go:embed 编译到二进制中，使用者无需关心文件路径
//
// addr 为 ADB 服务器地址，serial 为设备序列号
// force 为 true 时跳过存在性检查，强制推送
func InstallServiceJar(addr, serial string, force bool) error {
	// 非强制模式下，检查设备端文件是否已存在
	if !force && fileExistsOnDevice(addr, serial, DefaultJarRemote) {
		return nil // 文件已存在，跳过推送
	}

	_, err := adb.PushData(addr, serial, assets.JarData, DefaultJarRemote, 0644, false)
	if err != nil {
		return fmt.Errorf("推送 u2.jar 失败: %w", err)
	}
	return nil
}

// InstallServiceApk 将内嵌的 UIAutomator2 APK 推送到设备并安装
// APK 文件通过 go:embed 编译到二进制中，使用者无需关心文件路径
//
// addr 为 ADB 服务器地址，serial 为设备序列号
// force 为 true 时跳过存在性检查，强制推送
func InstallServiceApk(addr, serial string, force bool) error {
	// 非强制模式下，检查设备端文件是否已存在
	if !force && fileExistsOnDevice(addr, serial, DefaultApkRemote) {
		// 文件已存在，跳过推送，但仍需确保已安装
	} else {
		_, err := adb.PushData(addr, serial, assets.ApkData, DefaultApkRemote, 0644, false)
		if err != nil {
			return fmt.Errorf("推送 APK 失败: %w", err)
		}
	}

	// 在设备上安装 APK（覆盖安装）
	_, err := adb.InstallApkOnDevice(addr, serial, DefaultApkRemote, "-r", false)
	if err != nil {
		return fmt.Errorf("安装 APK 失败: %w", err)
	}
	return nil
}
