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

const maxChunk = 64 * 1024

type Sync struct {
	Conn net.Conn
}

func InitSync(conn net.Conn) *Sync {
	return &Sync{
		Conn: conn,
	}
}

// SyncPushFile 将本地文件推送到设备
// localPath: 本地文件路径
// remotePath: 设备目标路径
// mode: 文件权限 (如 0644)
// debug: 是否打印调试信息
func (s *Sync) SyncPushFile(localPath, remotePath string, mode int, debug bool) (int64, error) {
	// 打开文件
	f, err := os.Open(localPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	fi, _ := f.Stat()
	if debug {
		fmt.Printf("[DEBUG] local filesize=%d\n", fi.Size())
	}
	s.StartSync()
	// 构造 SEND payload
	modeStr := strconv.Itoa(syscall.S_IFREG | mode)
	sendPayload := []byte(remotePath + "," + modeStr)

	// 写入 "SEND" + 长度 + payload
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
		fmt.Printf("[DEBUG] Wrote SEND payload len=%d path=%s mode=%s\n", len(sendPayload), remotePath, modeStr)
	}

	// 写入 DATA 块
	var total int64
	buf := make([]byte, maxChunk)
	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			hdr := make([]byte, 8)
			copy(hdr[:4], []byte("DATA"))
			binary.LittleEndian.PutUint32(hdr[4:], uint32(n))
			if _, err := s.Conn.Write(hdr); err != nil {
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

	// 发送 DONE
	done := make([]byte, 8)
	copy(done[:4], []byte("DONE"))
	mtime := uint32(fi.ModTime().Unix())
	binary.LittleEndian.PutUint32(done[4:], mtime)
	if _, err := s.Conn.Write(done); err != nil {
		return total, err
	}
	if debug {
		fmt.Println("[DEBUG] Wrote DONE, waiting response")
	}

	// 读取最终响应
	resp, msg, err := ReadSyncStatus(s.Conn)
	if err != nil {
		return total, err
	}
	if resp != "OKAY" {
		if len(msg) > 0 {
			return total, fmt.Errorf("sync failed: %s", string(msg))
		}
		return total, fmt.Errorf("sync failed: %s", resp)
	}
	return total, nil
}

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
			return fmt.Errorf("sync open failed: %s", string(msg))
		}
		return fmt.Errorf("sync open failed: %q", string(tok))
	}
	return nil
}

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
		// 非预期状态，直接返回原始字符串，便于上层报错
		return status, "", nil
	}
}
