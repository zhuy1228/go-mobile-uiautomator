package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zhuy1228/go-mobile-uiautomator/adb"
)

// 默认路径常量
const (
	// DefaultJarLocal 默认本地 JAR 路径
	DefaultJarLocal = "assets/u2.jar"
	// DefaultJarRemote 默认设备端 JAR 路径
	DefaultJarRemote = "/data/local/tmp/u2.jar"
	// DefaultApkLocal 默认本地 APK 路径
	DefaultApkLocal = "assets/app-uiautomator.apk"
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

// InstallServiceJar 将 u2.jar 推送到设备的 /data/local/tmp/ 目录
// addr 为 ADB 服务器地址，serial 为设备序列号
// localPath 为本地 JAR 路径，传空字符串则使用默认路径
// force 为 true 时跳过存在性检查，强制推送
func InstallServiceJar(addr, serial, localPath string, force bool) error {
	if localPath == "" {
		localPath = DefaultJarLocal
	}

	// 检查本地文件是否存在
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("本地 JAR 文件不存在: %s", localPath)
	}

	// 非强制模式下，检查设备端文件是否已存在
	if !force && fileExistsOnDevice(addr, serial, DefaultJarRemote) {
		return nil // 文件已存在，跳过推送
	}

	_, err := adb.PushFile(addr, serial, localPath, DefaultJarRemote, 0644, false)
	if err != nil {
		return fmt.Errorf("推送 u2.jar 失败: %w", err)
	}
	return nil
}

// InstallServiceApk 将 UIAutomator2 APK 推送到设备并安装
// addr 为 ADB 服务器地址，serial 为设备序列号
// localPath 为本地 APK 路径，传空字符串则使用默认路径
// force 为 true 时跳过存在性检查，强制推送
func InstallServiceApk(addr, serial, localPath string, force bool) error {
	if localPath == "" {
		localPath = DefaultApkLocal
	}

	abs, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("解析 APK 路径失败: %w", err)
	}

	// 检查本地文件是否存在
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		return fmt.Errorf("本地 APK 文件不存在: %s", abs)
	}

	// 非强制模式下，检查设备端文件是否已存在
	if !force && fileExistsOnDevice(addr, serial, DefaultApkRemote) {
		// 文件已存在，跳过推送，但仍需确保已安装
	} else {
		_, err = adb.PushFile(addr, serial, abs, DefaultApkRemote, 0644, false)
		if err != nil {
			return fmt.Errorf("推送 APK 失败: %w", err)
		}
	}

	// 在设备上安装 APK（覆盖安装）
	_, err = adb.InstallApkOnDevice(addr, serial, DefaultApkRemote, "-r", false)
	if err != nil {
		return fmt.Errorf("安装 APK 失败: %w", err)
	}
	return nil
}
