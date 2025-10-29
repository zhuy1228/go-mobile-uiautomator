package adb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

func DialADB(addr string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	return d.Dial("tcp", addr)
}

func WriteAdbCmd(conn net.Conn, cmd string) error {
	header := fmt.Sprintf("%04x", len(cmd))
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err := conn.Write([]byte(header + cmd))
	return err
}

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
	return buf, nil
}

func ReadStatus(conn net.Conn) (string, error) {
	b, err := readN(conn, 4, 3*time.Second)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

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

func ExecOut(conn net.Conn, cmd string) ([]byte, error) {
	if err := WriteAdbCmd(conn, "exec-out:"+cmd); err != nil {
		return nil, err
	}
	status, err := ReadStatus(conn)
	if err != nil {
		return nil, err
	}
	if status == "FAIL" {
		msg, _ := ReadLenFrame(conn)
		return nil, fmt.Errorf("exec-out FAIL: %s", string(msg))
	}
	if status != "OKAY" {
		return nil, fmt.Errorf("unexpected exec-out status: %s", status)
	}
	var out []byte
	for {
		data, err := ReadLenFrame(conn)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				break
			}
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if len(data) == 0 {
			break
		}
		out = append(out, data...)
	}
	return out, nil
}

// transportTo: 指示 adb server 将后续请求路由到指定 serial
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
		return fmt.Errorf("transport FAIL: %s", string(msg))
	}
	if status != "OKAY" {
		return fmt.Errorf("unexpected transport status: %s", status)
	}
	return nil
}

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
		return nil, fmt.Errorf("shell FAIL: %s", string(msg))
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

// readResponse reads a 4-byte response like OKAY/FAIL and returns it and optional message (for FAIL)
func ReadResponse(conn net.Conn, debug bool) (string, []byte, error) {
	readN := func(n int) ([]byte, error) {
		buf := make([]byte, n)
		total := 0
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for total < n {
			nr, err := conn.Read(buf[total:])
			if err != nil {
				return nil, err
			}
			total += nr
		}
		return buf, nil
	}

	stb, err := readN(4)
	if err != nil {
		return "", nil, err
	}
	st := string(stb)
	if debug {
		fmt.Fprintf(os.Stderr, "ReadResponseFixed: status raw hex=%x ascii=%q\n", stb, st)
	}
	if st == "OKAY" {
		return st, nil, nil
	}
	if st == "FAIL" {
		// read next 4 bytes (may be little-endian uint32 length or ASCII hex)
		hdr, err := readN(4)
		if err != nil {
			if debug {
				fmt.Fprintf(os.Stderr, "ReadResponseFixed: no length header after FAIL: %v\n", err)
			}
			return st, nil, nil
		}
		if debug {
			fmt.Fprintf(os.Stderr, "ReadResponseFixed: length header raw hex=%x ascii=%q\n", hdr, string(hdr))
		}

		// try little-endian uint32 first
		l := int(binary.LittleEndian.Uint32(hdr))
		if l > 0 {
			msg, err := readN(l)
			if err != nil {
				if debug {
					fmt.Fprintf(os.Stderr, "ReadResponseFixed: failed to read %d bytes message: %v\n", l, err)
				}
				return st, nil, nil
			}
			if debug {
				fmt.Fprintf(os.Stderr, "ReadResponseFixed: message hex=%x ascii=%q\n", msg, string(msg))
			}
			return st, msg, nil
		}

		// fallback: try ASCII-hex parse (backwards compatibility)
		if n, perr := strconv.ParseInt(string(hdr), 16, 32); perr == nil && n > 0 {
			msg, err := readN(int(n))
			if err != nil {
				if debug {
					fmt.Fprintf(os.Stderr, "ReadResponseFixed: failed to read ascii-hex message of len %d: %v\n", n, err)
				}
				return st, nil, nil
			}
			if debug {
				fmt.Fprintf(os.Stderr, "ReadResponseFixed: ascii-hex message hex=%x ascii=%q\n", msg, string(msg))
			}
			return st, msg, nil
		}

		// neither produced a message
		return st, nil, nil
	}

	// unexpected token
	return st, nil, nil
}
