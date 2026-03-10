package adb

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

// DeviceInfo 包含从 ADB 服务器设备列表和设备端 getprop 收集到的设备信息
type DeviceInfo struct {
	Serial      string            // 设备序列号
	State       string            // 设备状态（device/offline/unauthorized）
	Product     string            // 产品名称
	Model       string            // 设备型号
	Device      string            // 设备代号
	TransportID string            // ADB 传输 ID
	Props       map[string]string // 额外属性
}

// ListDevicesRaw 向 ADB 服务器请求设备列表并返回原始文本
// addr 为 ADB 服务器地址，timeout 为超时时间
func ListDevicesRaw(addr string, timeout time.Duration) (string, error) {
	conn, err := DialADB(addr, timeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if err := WriteAdbCmd(conn, "host:devices-l"); err != nil {
		return "", err
	}
	status, err := ReadStatus(conn)
	if err != nil {
		return "", err
	}
	if status == "FAIL" {
		msg, _ := ReadLenFrame(conn)
		return "", fmt.Errorf("ADB 返回失败: %s", string(msg))
	}
	if status != "OKAY" {
		return "", fmt.Errorf("意外的状态码: %s", status)
	}

	var parts []string
	for {
		data, err := ReadLenFrame(conn)
		if err != nil {
			// 超时视为读取结束
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				break
			}
			if err == io.EOF {
				break
			}
			return "", err
		}
		if len(data) == 0 {
			break
		}
		parts = append(parts, string(data))
	}
	log.Println(parts)
	return strings.Join(parts, ""), nil
}

// ParseDevicesPayload 解析 host:devices-l 命令返回的文本
// 提取设备的序列号、状态、product/model/device/transport_id 等信息
func ParseDevicesPayload(payload string) []DeviceInfo {
	out := []DeviceInfo{}
	lines := strings.Split(payload, "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "List of devices attached") {
			continue
		}
		// adb -l 格式：serial <state> key:val key:val ...
		fields := strings.Fields(ln)
		if len(fields) < 2 {
			continue
		}
		dev := DeviceInfo{
			Serial: fields[0],
			State:  fields[1],
			Props:  map[string]string{},
		}
		// 解析剩余的 key:val 键值对
		for _, kv := range fields[2:] {
			if strings.Contains(kv, ":") {
				parts := strings.SplitN(kv, ":", 2)
				k := parts[0]
				v := parts[1]
				switch k {
				case "product":
					dev.Product = v
				case "model":
					dev.Model = v
				case "device":
					dev.Device = v
				case "transport_id":
					dev.TransportID = v
				default:
					// 将未知字段存入 Props，添加 "short." 前缀
					dev.Props["short."+k] = v
				}
			}
		}
		out = append(out, dev)
	}
	return out
}

// parseGetprop 解析 getprop 命令的输出
// 输出格式为 "[key]: [value]"，解析为 map
func parseGetprop(raw []byte) map[string]string {
	m := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		// getprop 输出格式：[ro.build.version.release]: [10]
		parts := strings.SplitN(line, "]: [", 2)
		if len(parts) == 2 {
			k := strings.TrimPrefix(parts[0], "[")
			v := strings.TrimSuffix(parts[1], "]")
			m[k] = v
		} else {
			// 回退：尝试按 ": " 分割
			kv := strings.SplitN(line, ": ", 2)
			if len(kv) == 2 {
				m[strings.Trim(kv[0], "[]")] = strings.Trim(kv[1], "[]")
			}
		}
	}
	return m
}

// FindSerialByProduct 根据产品名称查找设备序列号
// 优先从设备列表中匹配，如果列表中没有 product 字段，则回退到逐设备查询 getprop
func FindSerialByProduct(addr, targetProduct string) (string, error) {
	payload, err := ListDevicesRaw(addr, 3*time.Second)
	if err != nil {
		return "", err
	}
	devices := ParseDevicesPayload(payload)
	for _, dev := range devices {
		if dev.Product == targetProduct {
			return dev.Serial, nil
		}
	}
	// 回退：逐设备查询 getprop ro.product.model
	lines := strings.Split(payload, "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "List of devices attached") {
			continue
		}
		fields := strings.Fields(ln)
		if len(fields) < 1 {
			continue
		}
		serial := fields[0]
		conn, err := ConnectToDevice(addr, serial, 2*time.Second)
		if err != nil {
			continue
		}
		out, err := ExecShell(conn, "getprop ro.product.model")
		conn.Close()
		if err == nil {
			if strings.TrimSpace(string(out)) == targetProduct {
				return serial, nil
			}
		}
	}
	return "", fmt.Errorf("未找到产品名为 %s 的设备", targetProduct)
}

// InstallApkOnDevice 安装 APK 到设备
// addr 为 ADB 服务器地址，serial 为设备序列号
// remoteTmp 为 APK 在设备上的临时路径
// pmArgs 为 pm install 的额外参数（如 "-r" 表示覆盖安装），默认为 "-r"
// debug 为 true 时输出调试信息
func InstallApkOnDevice(addr, serial string, remoteTmp string, pmArgs string, debug bool) (string, error) {
	conn, err := ConnectToDevice(addr, serial, 15*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if pmArgs == "" {
		pmArgs = "-r"
	}

	// 执行 pm install 命令
	installCmd := "pm install " + pmArgs + " " + remoteTmp
	if debug {
		fmt.Printf("[调试] 执行命令: %s\n", installCmd)
	}
	outBuf, err := ExecShell(conn, installCmd)
	if err != nil {
		log.Println("[错误]", err)
		return "", err
	}

	outStr := string(outBuf)
	if debug {
		fmt.Printf("[调试] pm install 输出:\n%s\n", outStr)
	}

	// 安装完成后删除临时文件
	ExecShell(conn, "rm -f "+remoteTmp)

	// 根据输出判断是否安装成功
	if strings.Contains(strings.ToLower(outStr), "success") {
		return outStr, nil
	}
	return outStr, fmt.Errorf("安装失败: %s", outStr)
}
