package services

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/zhuy1228/go-mobile-uiautomator/adb"
)

// InstallServiceJar 将 u2.jar 推送到设备的 /data/local/tmp/ 目录
// addr 为 ADB 服务器地址，serial 为设备序列号
func InstallServiceJar(addr, serial string) {
	conn, err := adb.DialADB(addr, 15*time.Second)
	if err != nil {
		fmt.Println("连接失败:", err)
		return
	}
	defer conn.Close()

	if err := adb.TransportTo(conn, serial); err != nil {
		fmt.Println("设备路由失败:", err)
		return
	}

	targetPath := "/data/local/tmp/u2.jar"
	sync := adb.InitSync(conn)
	n, err := sync.SyncPushFile("./assets/u2.jar", targetPath, 0644, true)
	if err != nil {
		fmt.Println("推送失败:", err)
	} else {
		fmt.Printf("推送成功, 共写入 %d 字节\n", n)
	}
}

// InstallServiceApk 将 UIAutomator2 APK 推送到设备并安装
// addr 为 ADB 服务器地址，serial 为设备序列号
func InstallServiceApk(addr, serial string) {
	conn, err := adb.DialADB(addr, 15*time.Second)
	if err != nil {
		fmt.Println("连接失败:", err)
		return
	}

	if err := adb.TransportTo(conn, serial); err != nil {
		fmt.Println("设备路由失败:", err)
		conn.Close()
		return
	}

	targetPath := "/data/local/tmp/app-uiautomator.apk"
	sync := adb.InitSync(conn)
	abs, _ := filepath.Abs("./assets/app-uiautomator.apk")
	n, err := sync.SyncPushFile(abs, targetPath, 0644, true)
	if err != nil {
		fmt.Println("推送失败:", err)
	} else {
		fmt.Printf("推送成功, 共写入 %d 字节\n", n)
	}
	conn.Close()

	// 在设备上安装 APK（覆盖安装）
	adb.InstallApkOnDevice(addr, serial, targetPath, "-r", true)
}
