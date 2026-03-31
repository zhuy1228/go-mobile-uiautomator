package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zhuy1228/go-mobile-uiautomator/adb"
	"github.com/zhuy1228/go-mobile-uiautomator/libs"
)

// 设备配置，根据实际环境修改
const (
	serial = "emulator-5554" // 设备序列号
)

func main() {
	// launchUiautomator()
	ctx := context.Background()
	go func() {
		for ev := range adb.TrackDevices(ctx, libs.DefaultADBAddr, 15*time.Second) {
			if ev.Err != nil {
				fmt.Println("track-devices 结束:", ev.Err)
				return
			}
			if ev.Payload == "" {
				continue
			}
			fmt.Println(ev.Payload)
			info, err := adb.GetProp(libs.DefaultADBAddr, serial)
			if err != nil {
				fmt.Println("getprop 失败:", err)
				return
			}
			fmt.Println("Android版本:", info["ro.build.version.release"])
			fmt.Println("设备型号:", info["ro.product.model"])
			fmt.Println("设备代号:", info["ro.product.device"])
			fmt.Println("设备品牌:", info["ro.product.brand"])
			fmt.Println("设备架构:", info["ro.product.cpu.abi"])
			fmt.Println("设备序列号:", info["ro.serialno"])
			fmt.Println("设备SDK:", info["ro.build.version.sdk"])
			fmt.Println("设备品牌:", info["ro.product.brand"])

		}
	}()
	select {}
}

// launchUiautomator 推送服务文件并启动 UIAutomator2
func launchUiautomator() {
	// 列出设备
	payload, _ := adb.ListDevicesRaw(libs.DefaultADBAddr, 15*time.Second)
	devices := adb.ParseDevicesPayload(payload)
	b, _ := json.MarshalIndent(devices, "", "  ")
	fmt.Println(string(b))

	// 使用 libs.NewDevice 启动 UIAutomator2 服务（不传 addr 使用默认地址）
	d, err := libs.NewDevice(serial)
	if err != nil {
		fmt.Println("启动失败:", err)
		return
	}
	defer d.Close() // 退出时停止 UIAutomator2

	// 获取设备信息
	info, err := d.Info()
	if err != nil {
		fmt.Println("获取设备信息失败:", err)
		return
	}
	fmt.Printf("设备信息: %v\n", info)

	// 启动 Chrome
	d.AppStart("com.android.chrome", "", true)

	// 设置隐式等待 10 秒
	d.ImplicitlyWait(10)

	// 开启调试模式，查看 JSON-RPC 请求和响应
	d.SetDebug(true)

	// 通过文本查找元素并点击
	if err = d.ByText("在裝置上新增帳戶").Click(); err != nil {
		fmt.Println("点击失败:", err)
	} else {
		fmt.Println("点击成功")
	}

	select {} // 阻塞等待
}
