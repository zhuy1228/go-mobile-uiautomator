package adb

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
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

// SyncPushTryVariants 按顺序尝试多种 SEND payload 风格，直到成功或尝试完毕。
// addr: adb server (e.g., "127.0.0.1:5037")
// serial: device serial
// localPath: 本地文件路径
// remotePath: 目标完整路径（必须包含文件名）
// mode: unix permission like 0644
// debug: 打印调试信息
func SyncPushTryVariants(addr, serial, localPath, remotePath string, mode int, debug bool) (int64, error) {
	type sendOption struct {
		name       string
		withNUL    bool
		modeFormat string // "hex" (0x8000|mode) or "dec" (decimal S_IFREG|mode)
	}
	opts := []sendOption{
		{"send-with-nul-hex", true, "hex"},
		{"send-without-nul-hex", false, "hex"},
		{"send-without-nul-dec", false, "dec"},
	}

	var lastErr error
	for _, opt := range opts {
		if debug {
			fmt.Printf("[try] option=%s\n", opt.name)
		}
		n, err := syncPushOne(addr, serial, localPath, remotePath, mode, opt.withNUL, opt.modeFormat, debug)
		if err == nil {
			if debug {
				fmt.Printf("[ok] option=%s pushed=%d\n", opt.name, n)
			}
			return n, nil
		}
		lastErr = fmt.Errorf("%s: %w", opt.name, err)
		if debug {
			fmt.Printf("[fail] option=%s err=%v\n", opt.name, err)
		}
		// small pause between tries
		time.Sleep(150 * time.Millisecond)
	}
	return 0, lastErr
}

// syncPushOne 在单个连接上按给定选项完成 sync push（一次性连接）
// withNUL: 是否在 SEND payload 后追加 NUL byte
// modeFormat: "hex" 表示 use 0x8000|mode in decimal string (common), "dec" 表示 decimal S_IFREG|mode
func syncPushOne(addr, serial, localPath, remotePath string, mode int, withNUL bool, modeFormat string, debug bool) (int64, error) {
	// open file
	f, err := os.Open(localPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	fi, _ := f.Stat()
	if debug {
		fmt.Printf("local filesize=%d\n", fi.Size())
	}

	// dial adb
	d := net.Dialer{Timeout: 8 * time.Second}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetNoDelay(true)
	}

	writeAdbCmd := func(cmd string) error {
		hdr := fmt.Sprintf("%04x", len(cmd))
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		_, err := conn.Write([]byte(hdr + cmd))
		return err
	}
	readN := func(n int) ([]byte, error) {
		buf := make([]byte, n)
		total := 0
		for total < n {
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			nr, err := conn.Read(buf[total:])
			if err != nil {
				return nil, err
			}
			total += nr
		}
		return buf, nil
	}
	read4 := func() ([]byte, error) { return readN(4) }

	// readResponse that returns status and optional msg (LE length)
	readResponse := func() (string, []byte, error) {
		stb, err := read4()
		if err != nil {
			return "", nil, err
		}
		st := string(stb)
		// if OKAY quick return
		if st == "OKAY" {
			return st, nil, nil
		}
		if st == "FAIL" {
			// try LE length
			hdr, err := read4()
			if err != nil {
				return st, nil, nil
			}
			l := int(binary.LittleEndian.Uint32(hdr))
			if l > 0 {
				msg, err := readN(l)
				if err != nil {
					return st, nil, nil
				}
				return st, msg, nil
			}
			// fallback ascii-hex
			if n, perr := strconv.ParseInt(string(hdr), 16, 32); perr == nil && n > 0 {
				msg, err := readN(int(n))
				if err != nil {
					return st, nil, nil
				}
				return st, msg, nil
			}
			return st, nil, nil
		}
		return st, nil, nil
	}

	// transport
	if err := writeAdbCmd("host:transport:" + serial); err != nil {
		return 0, err
	}
	tok, err := read4()
	if err != nil {
		return 0, err
	}
	if string(tok) != "OKAY" {
		return 0, fmt.Errorf("transport failed: %q", string(tok))
	}

	// open sync
	if err := writeAdbCmd("sync:"); err != nil {
		return 0, err
	}
	tok, err = read4()
	if err != nil {
		return 0, err
	}
	if string(tok) != "OKAY" {
		_, msg, _ := readResponse()
		if len(msg) > 0 {
			return 0, fmt.Errorf("sync open failed: %s", string(msg))
		}
		return 0, fmt.Errorf("sync open failed: %q", string(tok))
	}

	// build SEND payload according options
	var modeStr string
	if modeFormat == "hex" {
		// common implementations expect decimal of (S_IFREG|mode) where S_IFREG is 0100000 (octal) but using 0x8000 is fine as decimal string
		modeStr = strconv.FormatInt(int64(0x8000|mode), 10)
	} else {
		// decimal form: simply decimal of (0x8000|mode)
		modeStr = strconv.FormatInt(int64(0x8000|mode), 10)
	}
	sendPayload := []byte(remotePath + "," + modeStr)
	if withNUL {
		sendPayload = append(sendPayload, 0)
	}

	// write SEND
	if _, err := conn.Write(append([]byte("SEND"), sendPayload...)); err != nil {
		return 0, err
	}
	if debug {
		fmt.Printf("Wrote SEND payload len=%d withNUL=%v modeFmt=%s path=%s\n", len(sendPayload), withNUL, modeFormat, remotePath)
	}

	// write DATA blocks if file has content (if zero-length, skip DATA)
	var total int64
	buf := make([]byte, maxChunk)
	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			hdr := make([]byte, 8)
			copy(hdr[:4], []byte("DATA"))
			binary.LittleEndian.PutUint32(hdr[4:], uint32(n))
			if _, err := conn.Write(hdr); err != nil {
				return total, err
			}
			if _, err := conn.Write(buf[:n]); err != nil {
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

	// send DONE
	done := make([]byte, 8)
	copy(done[:4], []byte("DONE"))
	binary.LittleEndian.PutUint32(done[4:], uint32(time.Now().Unix()))
	if _, err := conn.Write(done); err != nil {
		return total, err
	}
	if debug {
		fmt.Println("Wrote DONE, waiting response")
	}

	// read final
	resp, msg, err := readResponse()
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
