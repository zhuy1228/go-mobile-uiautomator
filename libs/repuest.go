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

// ---------- 用于适配的外部接口/类型 ----------
type AdbDevice interface {
	// CreateConnection 建立到设备的 TCP 连接（通常通过 adb 隧道/port-forward 实现）
	// network 常为 "tcp" 或 "tcp4"/"tcp6"， port 为设备上服务监听端口
	CreateConnection(network string, port int) (net.Conn, error)
}

// ---------- HTTPResponse 等价类型 ----------
type HTTPResponse struct {
	Content []byte
	Status  int
	Reason  string
}

func (r *HTTPResponse) JSON(v interface{}) error {
	return json.Unmarshal(r.Content, v)
}

func (r *HTTPResponse) Text() string {
	return string(r.Content)
}

// ---------- 自定义错误类型 ----------
var (
	ErrHTTPTimeout = errors.New("http request timeout")
	ErrHTTPFailed  = errors.New("http request failed")
)

// ---------- AdbHTTPConnection 核心：使用 net.Conn 写请求并用 http.ReadResponse 解析 ----------
type AdbHTTPConnection struct {
	Conn net.Conn
}

func NewAdbHTTPConnection(dev AdbDevice, port int, timeout time.Duration) (*AdbHTTPConnection, error) {
	// 这里 network 使用 "tcp"，调用方可依据实际实现调整
	conn, err := dev.CreateConnection("tcp", port)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to uiautomator2 server: %w", err)
	}
	// 设置默认 deadline，调用方可在需要时调整 Conn.SetDeadline
	_ = conn.SetDeadline(time.Now().Add(timeout))
	return &AdbHTTPConnection{Conn: conn}, nil
}

func (c *AdbHTTPConnection) Close() error {
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}

// sendRequest 写入 HTTP 请求并返回 *http.Response
func (c *AdbHTTPConnection) sendRequest(req *http.Request, timeout time.Duration) (*http.Response, error) {
	// 确保 deadline
	if timeout > 0 {
		_ = c.Conn.SetDeadline(time.Now().Add(timeout))
	} else {
		_ = c.Conn.SetDeadline(time.Time{})
	}

	// 将 http.Request 序列化为原始 HTTP 报文并写到 conn
	var buf bytes.Buffer
	// 行: METHOD PATH HTTP/1.1
	path := req.URL.RequestURI()
	if path == "" {
		path = "/"
	}
	fmt.Fprintf(&buf, "%s %s HTTP/1.1\r\n", req.Method, path)
	// Host 头；Python 里使用 "localhost" 但最终是通过 adb 隧道，Host 不重要，这里写 localhost
	fmt.Fprintf(&buf, "Host: localhost\r\n")

	// 写 headers
	// 保证 Content-Length 或 Transfer-Encoding 存在
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "uiautomator2")
	}
	if req.Header.Get("Accept-Encoding") == "" {
		// 与 Python 保持一致，明确禁用压缩
		req.Header.Set("Accept-Encoding", "")
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// copy headers into buffer
	for k, vals := range req.Header {
		for _, v := range vals {
			fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
		}
	}

	// body
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("read request body failed: %w", err)
		}
		// set Content-Length
		fmt.Fprintf(&buf, "Content-Length: %d\r\n", len(bodyBytes))
	} else {
		fmt.Fprintf(&buf, "Content-Length: 0\r\n")
	}

	// header-body separator
	buf.WriteString("\r\n")

	// write header+body to conn
	if _, err := c.Conn.Write(buf.Bytes()); err != nil {
		return nil, fmt.Errorf("write request headers failed: %w", err)
	}
	if len(bodyBytes) > 0 {
		if _, err := c.Conn.Write(bodyBytes); err != nil {
			return nil, fmt.Errorf("write request body failed: %w", err)
		}
	}

	// read response using http.ReadResponse
	reader := bufio.NewReader(c.Conn)
	// note: http.ReadResponse expects a Request to be passed for RequestURI related logic,
	// but if nil it still parses status & headers fine. Provide req for completeness.
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		return nil, fmt.Errorf("read http response failed: %w", err)
	}
	return resp, nil
}

// ---------- _http_request 的 Go 等价实现 ----------
func HttpRequest(ctx context.Context, dev AdbDevice, devicePort int, method, path string, data map[string]interface{}, timeoutSecs float64, printRequest bool) (*HTTPResponse, error) {
	// 兼容 python 默认 timeout
	if timeoutSecs <= 0 {
		timeoutSecs = 10.0
	}
	timeout := time.Duration(timeoutSecs * float64(time.Second))

	// debug 打印 curl 样式
	if printRequest {
		now := time.Now().Format("15:04:05.000")
		url := fmt.Sprintf("http://127.0.0.1:%d%s", devicePort, path)
		if data != nil {
			b, _ := json.Marshal(data)
			fmt.Printf("# http timeout=%.3f\n%s $ curl -X %s %s -d '%s'\n", timeoutSecs, now, method, url, string(b))
		} else {
			fmt.Printf("# http timeout=%.3f\n%s $ curl -X %s %s\n", timeoutSecs, now, method, url)
		}
	}

	// 构造 http.Request
	var body io.Reader
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("json marshal data failed: %w", err)
		}
		body = bytes.NewReader(b)
	}
	// URL 可以是任何合法的虚拟 URL，因为我们直接写原始请求行到 conn
	req, err := http.NewRequestWithContext(ctx, method, "http://localhost"+path, body)
	if err != nil {
		return nil, fmt.Errorf("create http request failed: %w", err)
	}
	// 设置 headers 与 Python 保持一致
	req.Header.Set("User-Agent", "uiautomator2")
	req.Header.Set("Accept-Encoding", "")
	req.Header.Set("Content-Type", "application/json")

	// 建立到设备的连接（AdbHTTPConnection.connect）
	connWrapper, err := NewAdbHTTPConnection(dev, devicePort, timeout)
	if err != nil {
		return nil, err
	}
	defer connWrapper.Close()

	// 发送请求并读取响应
	resp, err := connWrapper.sendRequest(req, timeout)
	if err != nil {
		// 判断是否为超时（net.Error with Timeout）
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, fmt.Errorf("%w: %v", ErrHTTPTimeout, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrHTTPFailed, err)
	}
	defer resp.Body.Close()

	// 按块读取响应体（与 Python 的循环等效）
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http request failed: %d %s", resp.StatusCode, resp.Status)
	}

	response := &HTTPResponse{
		Content: content,
		Status:  resp.StatusCode,
		Reason:  resp.Status,
	}

	if printRequest {
		now := time.Now().Format("15:04:05.000")
		fmt.Printf("%s Response >>>\n%s\n<<< END timed_used = %.3f\n\n", now, strings.TrimRight(response.Text(), "\n"), time.Since(time.Now().Add(-timeout)).Seconds())
	}

	return response, nil
}
