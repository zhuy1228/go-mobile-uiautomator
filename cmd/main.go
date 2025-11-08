package main

import (
	"encoding/json"
	"fmt"
	"go-mobile-uiautomator/adb"
	"go-mobile-uiautomator/services"
	"time"
)

func main() {
	addr := "127.0.0.1:5037"
	serial := "emulator-5556"
	FilePushInstall()
	go adb.LaunchUiautomator(addr, serial)
	select {}
}

// 文件推送加安装
func FilePushInstall() {
	addr := "127.0.0.1:5037"
	serial := "emulator-5556"
	payload, _ := adb.ListDevicesRaw(addr, 15*time.Second)
	m := adb.ParseDevicesPayload(payload)
	b2, _ := json.MarshalIndent(m, "", "  ")
	fmt.Println(string(b2))

	services.InstallServiceJar(addr, serial)

	// services.InstallServiceApk(addr, serial)
}

// 文件推送验证
func FilePush() {
	// edit these for your environment
	addr := "127.0.0.1:5037"
	local := "C:/Users/01/Desktop/aaa.PNG"
	remote := "/sdcard/ccc.PNG"
	mode := 0644
	targetProduct := "23113RKC6C"

	serial, err := adb.FindSerialByProduct(addr, targetProduct)
	if err != nil {
		fmt.Println("find device error:", err)
		return
	}
	fmt.Println("found serial:", serial)

	// Now open a new connection and transport to the found device for further ops
	conn, err := adb.DialADB(addr, 15*time.Second)
	if err != nil {
		fmt.Println("dial error:", err)
		return
	}
	defer conn.Close()
	adb.TransportTo(conn, serial)

	sync := adb.InitSync(conn)
	n, err := sync.SyncPushFile(local, remote, mode, true)
	if err != nil {
		fmt.Println("Push 失败:", err)
	} else {
		fmt.Printf("Push 成功, 共写入 %d 字节\n", n)
	}
}

// 连接验证
func Connect() {
	addr := "127.0.0.1:5037"
	targetProduct := "23113RKC6C"

	serial, err := adb.FindSerialByProduct(addr, targetProduct)
	if err != nil {
		fmt.Println("find device error:", err)
		return
	}
	fmt.Println("found serial:", serial)

	// Now open a new connection and transport to the found device for further ops
	conn, err := adb.DialADB(addr, 15*time.Second)
	if err != nil {
		fmt.Println("dial error:", err)
		return
	}
	defer conn.Close()
	adb.TransportTo(conn, serial)
	out, err := adb.ExecShell(conn, "getprop")
	if err != nil {
		fmt.Println("shell error:", err)
	} else {
		fmt.Printf("shell out: %q\n", string(out))
	}
}
