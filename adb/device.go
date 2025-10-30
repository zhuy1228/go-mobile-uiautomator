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

// DeviceInfo 包含从 adb server 列表和设备端 getprop 收集到的信息
type DeviceInfo struct {
	Serial      string
	State       string
	Product     string
	Model       string
	Device      string
	TransportID string
	Props       map[string]string
}

// listDevicesRaw: 请求 host:devices 并返回原始 payload
func listDevicesRaw(addr string, timeout time.Duration) (string, error) {
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
		return "", fmt.Errorf("adb FAIL: %s", string(msg))
	}
	if status != "OKAY" {
		return "", fmt.Errorf("unexpected status: %s", status)
	}

	var parts []string
	for {
		data, err := ReadLenFrame(conn)
		if err != nil {
			// treat short read timeout as finish
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

// parseDevicesPayload: 解析 host:devices 返回的 payload，提取可能的 product/model/device/transport_id
func parseDevicesPayload(payload string) []DeviceInfo {
	out := []DeviceInfo{}
	lines := strings.Split(payload, "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "List of devices attached") {
			continue
		}
		// adb -l 格式通常：serial <state> key:val key:val ...
		fields := strings.Fields(ln)
		if len(fields) < 2 {
			continue
		}
		dev := DeviceInfo{
			Serial: fields[0],
			State:  fields[1],
			Props:  map[string]string{},
		}
		// parse remaining key:val pairs
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
					// store any extra short fields into Props under prefixed key
					dev.Props["short."+k] = v
				}
			}
		}
		out = append(out, dev)
	}
	return out
}

// parseGetprop parses getprop output "key]: [value" lines into map
func parseGetprop(raw []byte) map[string]string {
	m := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		// getprop lines are like: [ro.build.version.release]: [10]
		if line == "" {
			continue
		}
		// find first ']:' separator
		// safe parse: extract between first '[' and first ']: [' pattern
		// simpler: split by "]: [" into two parts after trimming surrounding brackets
		parts := strings.SplitN(line, "]: [", 2)
		if len(parts) == 2 {
			k := strings.TrimPrefix(parts[0], "[")
			v := strings.TrimSuffix(parts[1], "]")
			m[k] = v
		} else {
			// fallback: try split by ": "
			kv := strings.SplitN(line, ": ", 2)
			if len(kv) == 2 {
				m[strings.Trim(kv[0], "[]")] = strings.Trim(kv[1], "[]")
			}
		}
	}
	return m
}

func parseDevicesMap(payload string) (map[string]string, []string) {
	m := make(map[string]string)
	lines := strings.Split(payload, "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "List of devices attached") {
			continue
		}
		fields := strings.Fields(ln)
		if len(fields) < 2 {
			continue
		}
		serial := fields[0]
		product := ""
		for _, kv := range fields[2:] {
			if strings.HasPrefix(kv, "product:") {
				product = strings.TrimPrefix(kv, "product:")
				break
			}
		}
		m[serial] = product
	}
	return m, lines
}

// Find device serial by product value and return serial (first match)
func FindSerialByProduct(addr, targetProduct string) (string, error) {
	payload, err := listDevicesRaw(addr, 3*time.Second)
	if err != nil {
		return "", err
	}
	m, _ := parseDevicesMap(payload)
	for serial, product := range m {
		if product == targetProduct {
			return serial, nil
		}
	}
	// fallback: if no product fields in devices-l, try per-device getprop
	payloadBasic, _ := listDevicesRaw(addr, 3*time.Second) // reuse; could be host:devices if preferred
	lines := strings.Split(payloadBasic, "\n")
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
		// query getprop ro.product.model for this serial
		conn, err := DialADB(addr, 2*time.Second)
		if err != nil {
			continue
		}
		// ensure close
		defer conn.Close()
		if err := TransportTo(conn, serial); err != nil {
			continue
		}
		out, err := ExecShell(conn, "getprop ro.product.model")
		if err == nil {
			if strings.TrimSpace(string(out)) == targetProduct {
				return serial, nil
			}
		}
	}
	return "", fmt.Errorf("no device with product=%s found", targetProduct)
}
