package libs

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// ---------- 外部接口和类型定义 ----------

// AdbDevice 定义了通过 ADB 隧道创建设备连接的接口
type AdbDevice interface {
	// CreateConnection 建立到设备的 TCP 连接
	// network 通常为 "tcp"，port 为设备上服务监听端口
	CreateConnection(network string, port int) (net.Conn, error)
}

// HTTPResponse 封装 HTTP 响应数据
type HTTPResponse struct {
	Content []byte // 响应体内容
	Status  int    // HTTP 状态码
	Reason  string // 状态描述
}

// JSON 将响应体解析为指定的结构体
func (r *HTTPResponse) JSON(v interface{}) error {
	return json.Unmarshal(r.Content, v)
}

// Text 返回响应体的文本内容
func (r *HTTPResponse) Text() string {
	return string(r.Content)
}

// ---------- 自定义错误类型 ----------

var (
	// ErrHTTPTimeout 表示 HTTP 请求超时
	ErrHTTPTimeout = errors.New("HTTP 请求超时")
	// ErrHTTPFailed 表示 HTTP 请求失败
	ErrHTTPFailed = errors.New("HTTP 请求失败")
)

// ---------- AdbHTTPConnection：通过 ADB 隧道发送 HTTP 请求 ----------

// AdbHTTPConnection 基于 net.Conn 实现的 HTTP 连接
// 通过 ADB 端口转发直接与设备端 UIAutomator2 服务通信
type AdbHTTPConnection struct {
	Conn net.Conn
}

// NewAdbHTTPConnection 创建一个新的 ADB HTTP 连接
// dev 为设备接口，port 为设备端服务端口，timeout 为连接超时
func NewAdbHTTPConnection(dev AdbDevice, port int, timeout time.Duration) (*AdbHTTPConnection, error) {
	conn, err := dev.CreateConnection("tcp", port)
	if err != nil {
		return nil, fmt.Errorf("无法连接到 UIAutomator2 服务: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	return &AdbHTTPConnection{Conn: conn}, nil
}

// Close 关闭底层连接
func (c *AdbHTTPConnection) Close() error {
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}

// sendRequest 将 HTTP 请求写入连接并读取响应
// 通过原始 TCP 连接发送 HTTP 报文，避免依赖标准 http.Client
func (c *AdbHTTPConnection) sendRequest(req *http.Request, timeout time.Duration) (*http.Response, error) {
	// 设置读写截止时间
	if timeout > 0 {
		_ = c.Conn.SetDeadline(time.Now().Add(timeout))
	} else {
		_ = c.Conn.SetDeadline(time.Time{})
	}

	// 序列化 HTTP 请求为原始报文
	var buf bytes.Buffer

	// 请求行：METHOD PATH HTTP/1.1
	path := req.URL.RequestURI()
	if path == "" {
		path = "/"
	}
	fmt.Fprintf(&buf, "%s %s HTTP/1.1\r\n", req.Method, path)
	fmt.Fprintf(&buf, "Host: localhost\r\n")

	// 设置默认请求头
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "uiautomator2")
	}
	if req.Header.Get("Accept-Encoding") == "" {
		req.Header.Set("Accept-Encoding", "")
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 写入请求头
	for k, vals := range req.Header {
		for _, v := range vals {
			fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
		}
	}

	// 处理请求体
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("读取请求体失败: %w", err)
		}
		fmt.Fprintf(&buf, "Content-Length: %d\r\n", len(bodyBytes))
	} else {
		fmt.Fprintf(&buf, "Content-Length: 0\r\n")
	}

	// 请求头与请求体之间的空行
	buf.WriteString("\r\n")

	// 发送请求头
	if _, err := c.Conn.Write(buf.Bytes()); err != nil {
		return nil, fmt.Errorf("发送请求头失败: %w", err)
	}
	// 发送请求体
	if len(bodyBytes) > 0 {
		if _, err := c.Conn.Write(bodyBytes); err != nil {
			return nil, fmt.Errorf("发送请求体失败: %w", err)
		}
	}

	// 使用标准库解析 HTTP 响应
	reader := bufio.NewReader(c.Conn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		return nil, fmt.Errorf("读取 HTTP 响应失败: %w", err)
	}
	return resp, nil
}

// ---------- HttpRequest：高层 HTTP 请求封装 ----------

// HttpRequest 向设备端 UIAutomator2 服务发送 HTTP 请求
// ctx 为上下文控制，dev 为设备接口，devicePort 为设备端服务端口
// method 为 HTTP 方法，path 为请求路径
// data 为请求体数据（会被 JSON 编码），timeoutSecs 为超时秒数
// printRequest 为 true 时输出 curl 风格的调试信息
func HttpRequest(ctx context.Context, dev AdbDevice, devicePort int, method, path string, data map[string]interface{}, timeoutSecs float64, printRequest bool) (*HTTPResponse, error) {
	// 默认超时 10 秒
	if timeoutSecs <= 0 {
		timeoutSecs = 10.0
	}
	timeout := time.Duration(timeoutSecs * float64(time.Second))

	// 调试模式：打印 curl 风格的请求信息
	if printRequest {
		now := time.Now().Format("15:04:05.000")
		url := fmt.Sprintf("http://127.0.0.1:%d%s", devicePort, path)
		if data != nil {
			b, _ := json.Marshal(data)
			fmt.Printf("# HTTP 超时=%.3f\n%s $ curl -X %s %s -d '%s'\n", timeoutSecs, now, method, url, string(b))
		} else {
			fmt.Printf("# HTTP 超时=%.3f\n%s $ curl -X %s %s\n", timeoutSecs, now, method, url)
		}
	}

	// 构造 HTTP 请求
	var body io.Reader
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("JSON 编码失败: %w", err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://localhost"+path, body)
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "uiautomator2")
	req.Header.Set("Accept-Encoding", "")
	req.Header.Set("Content-Type", "application/json")

	// 建立到设备的连接
	connWrapper, err := NewAdbHTTPConnection(dev, devicePort, timeout)
	if err != nil {
		return nil, err
	}
	defer connWrapper.Close()

	// 发送请求并读取响应
	resp, err := connWrapper.sendRequest(req, timeout)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, fmt.Errorf("%w: %v", ErrHTTPTimeout, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrHTTPFailed, err)
	}
	defer resp.Body.Close()

	// 读取响应体
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 请求失败: %d %s", resp.StatusCode, resp.Status)
	}

	response := &HTTPResponse{
		Content: content,
		Status:  resp.StatusCode,
		Reason:  resp.Status,
	}

	if printRequest {
		now := time.Now().Format("15:04:05.000")
		fmt.Printf("%s 响应 >>>\n%s\n<<< 结束\n\n", now, strings.TrimRight(response.Text(), "\n"))
	}

	return response, nil
}
