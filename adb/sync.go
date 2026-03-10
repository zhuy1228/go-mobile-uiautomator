package adb

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"
)

// maxChunk 定义单次数据块传输的最大字节数（64KB）
const maxChunk = 64 * 1024

// Sync 封装了 ADB 文件同步协议的操作
type Sync struct {
	Conn net.Conn // 已建立的 ADB 连接
}

// InitSync 创建一个文件同步操作实例
// conn 必须是已经通过 TransportTo 路由到目标设备的连接
func InitSync(conn net.Conn) *Sync {
	return &Sync{
		Conn: conn,
	}
}

// SyncPushFile 将本地文件推送到设备
// localPath: 本地文件路径
// remotePath: 设备端目标路径
// mode: 文件权限（如 0644）
// debug: 是否输出调试信息
// 返回写入的字节数和错误信息
func (s *Sync) SyncPushFile(localPath, remotePath string, mode int, debug bool) (int64, error) {
	// 打开本地文件
	f, err := os.Open(localPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	fi, _ := f.Stat()
	if debug {
		fmt.Printf("[调试] 本地文件大小=%d\n", fi.Size())
	}

	// 初始化同步模式
	if err := s.StartSync(); err != nil {
		return 0, err
	}

	// 构造 SEND 请求：remotePath + "," + 文件权限
	modeStr := strconv.Itoa(syscall.S_IFREG | mode)
	sendPayload := []byte(remotePath + "," + modeStr)

	// 写入 "SEND" 命令头 + 数据长度 + 请求内容
	hdr := make([]byte, 8)
	copy(hdr[:4], []byte("SEND"))
	binary.LittleEndian.PutUint32(hdr[4:], uint32(len(sendPayload)))
	if _, err := s.Conn.Write(hdr); err != nil {
		return 0, err
	}
	if _, err := s.Conn.Write(sendPayload); err != nil {
		return 0, err
	}
	if debug {
		fmt.Printf("[调试] 发送 SEND 请求: 长度=%d 路径=%s 权限=%s\n", len(sendPayload), remotePath, modeStr)
	}

	// 分块写入文件数据（DATA 命令）
	var total int64
	buf := make([]byte, maxChunk)
	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			dataHdr := make([]byte, 8)
			copy(dataHdr[:4], []byte("DATA"))
			binary.LittleEndian.PutUint32(dataHdr[4:], uint32(n))
			if _, err := s.Conn.Write(dataHdr); err != nil {
				return total, err
			}
			if _, err := s.Conn.Write(buf[:n]); err != nil {
				return total, err
			}
			total += int64(n)
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			return total, rerr
		}
	}

	// 发送 DONE 命令，携带文件修改时间戳
	done := make([]byte, 8)
	copy(done[:4], []byte("DONE"))
	mtime := uint32(fi.ModTime().Unix())
	binary.LittleEndian.PutUint32(done[4:], mtime)
	if _, err := s.Conn.Write(done); err != nil {
		return total, err
	}
	if debug {
		fmt.Println("[调试] 发送 DONE，等待响应")
	}

	// 读取最终响应
	resp, msg, err := ReadSyncStatus(s.Conn)
	if err != nil {
		return total, err
	}
	if resp != "OKAY" {
		if len(msg) > 0 {
			return total, fmt.Errorf("同步失败: %s", string(msg))
		}
		return total, fmt.Errorf("同步失败: %s", resp)
	}
	return total, nil
}

// StartSync 启动 ADB 同步模式
// 发送 "sync:" 命令并等待 "OKAY" 响应
func (s *Sync) StartSync() error {
	if err := WriteAdbCmd(s.Conn, "sync:"); err != nil {
		return err
	}
	tok, err := readN(s.Conn, 4, 10*time.Second)
	if err != nil {
		return err
	}
	if string(tok) != "OKAY" {
		_, msg, _ := ReadSyncStatus(s.Conn)
		if len(msg) > 0 {
			return fmt.Errorf("同步模式启动失败: %s", string(msg))
		}
		return fmt.Errorf("同步模式启动失败: %q", string(tok))
	}
	return nil
}

// ReadSyncStatus 读取同步协议的状态响应
// 返回状态码（"OKAY"/"FAIL"/其他）、失败时的消息内容和错误
func ReadSyncStatus(r io.Reader) (string, string, error) {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return "", "", err
	}
	status := string(hdr)
	switch status {
	case "OKAY":
		return "OKAY", "", nil
	case "FAIL":
		// 读取 4 字节小端长度 + 对应长度的错误消息
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			return "FAIL", "", err
		}
		l := binary.LittleEndian.Uint32(lenBuf)
		msg := make([]byte, l)
		if _, err := io.ReadFull(r, msg); err != nil {
			return "FAIL", "", err
		}
		return "FAIL", string(msg), nil
	default:
		// 非预期状态，直接返回原始字符串，便于上层排查
		return status, "", nil
	}
}
