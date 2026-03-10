package adb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// mu 用于保护 ADB 连接的互斥锁，避免并发连接冲突
var mu sync.Mutex

// DialADB 建立到 ADB 服务器的 TCP 连接
// addr 格式为 "host:port"，例如 "127.0.0.1:5037"
// timeout 为连接超时时间
func DialADB(addr string, timeout time.Duration) (net.Conn, error) {
	mu.Lock()
	defer mu.Unlock()
	d := net.Dialer{Timeout: timeout}
	return d.Dial("tcp", addr)
}

// WriteAdbCmd 向 ADB 连接发送命令
// 协议格式：4 位十六进制长度前缀 + 命令内容
func WriteAdbCmd(conn net.Conn, cmd string) error {
	header := fmt.Sprintf("%04x", len(cmd))
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err := conn.Write([]byte(header + cmd))
	return err
}

// readN 从连接中精确读取 n 个字节
// 会持续读取直到收集到足够的字节数，或超时/出错
func readN(conn net.Conn, n int, timeout time.Duration) ([]byte, error) {
	buf := make([]byte, n)
	total := 0
	for total < n {
		conn.SetReadDeadline(time.Now().Add(timeout))
		nr, err := conn.Read(buf[total:])
		if err != nil {
			return nil, err
		}
		total += nr
	}
	conn.SetReadDeadline(time.Time{})
	return buf, nil
}

// ReadStatus 读取 ADB 协议的 4 字节状态码（如 "OKAY" 或 "FAIL"）
func ReadStatus(conn net.Conn) (string, error) {
	b, err := readN(conn, 4, 3*time.Second)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ReadLenFrame 读取一个带长度前缀的数据帧
// 先读取 4 字节十六进制长度前缀，再读取相应长度的数据
func ReadLenFrame(conn net.Conn) ([]byte, error) {
	hdr, err := readN(conn, 4, 3*time.Second)
	if err != nil {
		return nil, err
	}
	l, err := strconv.ParseInt(string(hdr), 16, 32)
	if err != nil {
		return nil, err
	}
	if l == 0 {
		return []byte{}, nil
	}
	return readN(conn, int(l), 10*time.Second)
}

// TransportTo 指示 ADB 服务器将后续请求路由到指定设备
// serial 为设备序列号，例如 "emulator-5556"
func TransportTo(conn net.Conn, serial string) error {
	if err := WriteAdbCmd(conn, "host:transport:"+serial); err != nil {
		return err
	}
	status, err := ReadStatus(conn)
	if err != nil {
		return err
	}
	if status == "FAIL" {
		msg, _ := ReadLenFrame(conn)
		return fmt.Errorf("传输失败: %s", string(msg))
	}
	if status != "OKAY" {
		return fmt.Errorf("意外的传输状态: %s", status)
	}
	conn.SetWriteDeadline(time.Time{})
	return nil
}

// ExecShell 在设备上执行 Shell 命令并返回输出结果
// shellCmd 为要执行的 Shell 命令字符串
func ExecShell(conn net.Conn, shellCmd string) ([]byte, error) {
	if err := WriteAdbCmd(conn, "shell:"+shellCmd); err != nil {
		return nil, err
	}
	st, err := ReadStatus(conn)
	if err != nil {
		return nil, err
	}
	if st != "OKAY" {
		msg, _ := ReadLenFrame(conn)
		return nil, fmt.Errorf("Shell 执行失败: %s", string(msg))
	}

	var buf bytes.Buffer
	tmp := make([]byte, 4096)
	for {
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, err := conn.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				break
			}
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// ReadResponse 读取 ADB 协议响应
// 返回状态码（"OKAY"/"FAIL"/其他）、消息内容和错误
// 支持小端 uint32 和 ASCII 十六进制两种长度编码格式
// debug 为 true 时会将调试信息输出到 stderr
func ReadResponse(conn net.Conn, debug bool) (string, []byte, error) {
	stb, err := readN(conn, 4, 10*time.Second)
	if err != nil {
		return "", nil, err
	}
	st := string(stb)
	if debug {
		fmt.Fprintf(os.Stderr, "ReadResponse: 状态原始 hex=%x ascii=%q\n", stb, st)
	}
	if st == "OKAY" {
		return st, nil, nil
	}
	if st == "FAIL" {
		// 读取后续 4 字节（可能是小端 uint32 长度或 ASCII 十六进制长度）
		hdr, err := readN(conn, 4, 10*time.Second)
		if err != nil {
			if debug {
				fmt.Fprintf(os.Stderr, "ReadResponse: FAIL 后无长度头: %v\n", err)
			}
			return st, nil, nil
		}
		if debug {
			fmt.Fprintf(os.Stderr, "ReadResponse: 长度头原始 hex=%x ascii=%q\n", hdr, string(hdr))
		}

		// 优先尝试小端 uint32 解析
		l := int(binary.LittleEndian.Uint32(hdr))
		if l > 0 {
			msg, err := readN(conn, l, 10*time.Second)
			if err != nil {
				if debug {
					fmt.Fprintf(os.Stderr, "ReadResponse: 读取 %d 字节消息失败: %v\n", l, err)
				}
				return st, nil, nil
			}
			if debug {
				fmt.Fprintf(os.Stderr, "ReadResponse: 消息 hex=%x ascii=%q\n", msg, string(msg))
			}
			return st, msg, nil
		}

		// 回退：尝试 ASCII 十六进制解析（向后兼容）
		if n, perr := strconv.ParseInt(string(hdr), 16, 32); perr == nil && n > 0 {
			msg, err := readN(conn, int(n), 10*time.Second)
			if err != nil {
				if debug {
					fmt.Fprintf(os.Stderr, "ReadResponse: 读取 ASCII 十六进制消息（长度 %d）失败: %v\n", n, err)
				}
				return st, nil, nil
			}
			if debug {
				fmt.Fprintf(os.Stderr, "ReadResponse: ASCII 十六进制消息 hex=%x ascii=%q\n", msg, string(msg))
			}
			return st, msg, nil
		}

		// 两种方式都未解析出消息
		return st, nil, nil
	}

	// 非预期的状态码
	return st, nil, nil
}

// ConnectToDevice 建立到指定设备的 ADB 连接并完成路由
// 返回已路由到目标设备的连接，调用者负责关闭
// 这是一个便捷函数，封装了 DialADB + TransportTo 的常见组合
func ConnectToDevice(addr, serial string, timeout time.Duration) (net.Conn, error) {
	conn, err := DialADB(addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("连接 ADB 服务器失败: %w", err)
	}
	if err := TransportTo(conn, serial); err != nil {
		conn.Close()
		return nil, fmt.Errorf("设备路由失败: %w", err)
	}
	return conn, nil
}

// PushFile 通过 ADB 推送本地文件到设备
// 这是一个独立函数，不依赖 Device 实例，适合初始化阶段使用
// addr 为 ADB 服务器地址，serial 为设备序列号
// localPath 为本地文件路径，remotePath 为设备目标路径
// mode 为文件权限（如 0644），debug 为 true 时输出调试信息
func PushFile(addr, serial, localPath, remotePath string, mode int, debug bool) (int64, error) {
	conn, err := ConnectToDevice(addr, serial, 15*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	s := InitSync(conn)
	return s.SyncPushFile(localPath, remotePath, mode, debug)
}

// PushData 通过 ADB 推送内存数据到设备
// 与 PushFile 功能相同，但数据来源是 []byte 而非本地文件
// 适用于通过 go:embed 嵌入的资源文件
func PushData(addr, serial string, data []byte, remotePath string, mode int, debug bool) (int64, error) {
	conn, err := ConnectToDevice(addr, serial, 15*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	s := InitSync(conn)
	return s.SyncPushData(data, remotePath, mode, debug)
}

// ---------- ADB 隧道（与 Python uiautomator2 完全一致） ----------

// CreateTunnel 建立到设备指定端口的 ADB 隧道
// 等同于 Python adbutils 的 device.create_connection(Network.TCP, port)
//
// 工作原理：
//  1. 连接 ADB 服务器
//  2. 通过 transport 选择目标设备
//  3. 发送 tcp:<port> 命令建立隧道
//  4. 返回的连接是一条直通设备端口的原始 TCP 管道
//
// 返回的连接由调用者负责关闭
func CreateTunnel(addr, serial string, port int) (net.Conn, error) {
	conn, err := DialADB(addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("连接 ADB 服务器失败: %w", err)
	}
	if err := TransportTo(conn, serial); err != nil {
		conn.Close()
		return nil, fmt.Errorf("设备路由失败: %w", err)
	}
	if err := WriteAdbCmd(conn, fmt.Sprintf("tcp:%d", port)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("发送隧道命令失败: %w", err)
	}
	status, err := ReadStatus(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("读取隧道状态失败: %w", err)
	}
	if status != "OKAY" {
		msg, _ := ReadLenFrame(conn)
		conn.Close()
		return nil, fmt.Errorf("ADB 隧道建立失败: %s %s", status, string(msg))
	}
	return conn, nil
}
