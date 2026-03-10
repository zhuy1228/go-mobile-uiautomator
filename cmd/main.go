package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zhuy1228/go-mobile-uiautomator/adb"
	"github.com/zhuy1228/go-mobile-uiautomator/services"
)

// 设备配置，根据实际环境修改
const (
	serial = "emulator-5554"  // 设备序列号
	addr   = "127.0.0.1:5037" // ADB 服务器地址
)

func main() {
	filePushInstall()
}

// launchUiautomator 推送服务文件并启动 UIAutomator2
func launchUiautomator() {
	filePushInstall()
	go adb.LaunchUiautomator(addr, serial)
	select {} // 阻塞等待，持续输出日志
}

// filePushInstall 列出设备并推送 u2.jar 到设备
func filePushInstall() {
	payload, _ := adb.ListDevicesRaw(addr, 15*time.Second)
	m := adb.ParseDevicesPayload(payload)
	b2, _ := json.MarshalIndent(m, "", "  ")
	fmt.Println(string(b2))

	services.InstallServiceJar(addr, serial)
}

// filePush 文件推送验证示例
func filePush() {
	// 根据实际环境修改以下参数
	local := "C:/Users/01/Desktop/aaa.PNG"
	remote := "/sdcard/ccc.PNG"
	mode := 0644
	targetProduct := "23113RKC6C"

	devSerial, err := adb.FindSerialByProduct(addr, targetProduct)
	if err != nil {
		fmt.Println("查找设备失败:", err)
		return
	}
	fmt.Println("找到设备:", devSerial)

	conn, err := adb.DialADB(addr, 15*time.Second)
	if err != nil {
		fmt.Println("连接失败:", err)
		return
	}
	defer conn.Close()
	adb.TransportTo(conn, devSerial)

	sync := adb.InitSync(conn)
	n, err := sync.SyncPushFile(local, remote, mode, true)
	if err != nil {
		fmt.Println("推送失败:", err)
	} else {
		fmt.Printf("推送成功, 共写入 %d 字节\n", n)
	}
}

// connect 连接验证示例
func connect() {
	targetProduct := "23113RKC6C"

	devSerial, err := adb.FindSerialByProduct(addr, targetProduct)
	if err != nil {
		fmt.Println("查找设备失败:", err)
		return
	}
	fmt.Println("找到设备:", devSerial)

	conn, err := adb.DialADB(addr, 15*time.Second)
	if err != nil {
		fmt.Println("连接失败:", err)
		return
	}
	defer conn.Close()
	adb.TransportTo(conn, devSerial)

	out, err := adb.ExecShell(conn, "getprop")
	if err != nil {
		fmt.Println("Shell 执行失败:", err)
	} else {
		fmt.Printf("Shell 输出: %q\n", string(out))
	}
}
