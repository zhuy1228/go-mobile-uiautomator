package services

import (
	"fmt"
	"go-mobile-uiautomator/adb"
	"path/filepath"
	"time"
)

func InstallServiceJar(addr, serial string) {
	conn, err := adb.DialADB(addr, 15*time.Second)
	if err != nil {
		fmt.Println("dial error:", err)
		return
	}
	defer conn.Close()
	adb.TransportTo(conn, serial)
	target_path := "/data/local/tmp/u2.jar"
	sync := adb.InitSync(conn)
	n, err := sync.SyncPushFile("./assets/u2.jar", target_path, 0644, true)
	if err != nil {
		fmt.Println("Push 失败:", err)
	} else {
		fmt.Printf("Push 成功, 共写入 %d 字节\n", n)
	}
}

func InstallServiceApk(addr, serial string) {
	conn, err := adb.DialADB(addr, 15*time.Second)
	if err != nil {
		fmt.Println("dial error:", err)
		return
	}
	adb.TransportTo(conn, serial)
	target_path := "/data/local/tmp/app-uiautomator.apk"
	sync := adb.InitSync(conn)
	abs, _ := filepath.Abs("./assets/app-uiautomator.apk")
	n, err := sync.SyncPushFile(abs, target_path, 0644, true)
	if err != nil {
		fmt.Println("Push 失败:", err)
	} else {
		fmt.Printf("Push 成功, 共写入 %d 字节\n", n)
	}
	conn.Close()
	adb.InstallApkOnDevice(addr, serial, target_path, "-r", true)
}
