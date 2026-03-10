package libs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// ---------- JSON-RPC 请求/响应结构 ----------

// JsonRpcRequest JSON-RPC 2.0 请求
type JsonRpcRequest struct {
	JsonRpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// JsonRpcResponse JSON-RPC 2.0 响应
type JsonRpcResponse struct {
	JsonRpc string           `json:"jsonrpc"`
	ID      int              `json:"id"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *JsonRpcError    `json:"error,omitempty"`
}

// JsonRpcError JSON-RPC 错误对象
type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"` // Java 堆栈信息
}

// ---------- JSON-RPC 调用函数 ----------

// JsonRpcCall 向 UIAutomator2 服务发送 JSON-RPC 调用
// dev 实现 AdbDevice 接口，devicePort 为设备端服务端口
// method 为 RPC 方法名，params 为参数
// timeout 为请求超时（秒），debug 为 true 时输出调试信息
//
// 返回值为 JSON 原始字节（由调用者根据需要解析）
//
// 可能返回的错误类型：
//   - *UiObjectNotFoundError: UI 元素未找到
//   - *UiAutomationNotConnectedError: UIAutomation 服务断开
//   - *RPCStackOverflowError: Java 端栈溢出
//   - *RPCUnknownError: 未知 RPC 错误
//   - *RPCInvalidError: 无效的 RPC 响应
func JsonRpcCall(ctx context.Context, dev AdbDevice, devicePort int, method string, params interface{}, timeout float64, debug bool) (json.RawMessage, error) {
	// 构造 JSON-RPC 请求体
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}

	// 通过 HTTP 发送到 /jsonrpc/0 端点
	resp, err := HttpRequest(ctx, dev, devicePort, "POST", "/jsonrpc/0", payload, timeout, debug)
	if err != nil {
		return nil, err
	}

	// 解析响应
	var rpcResp JsonRpcResponse
	if err := json.Unmarshal(resp.Content, &rpcResp); err != nil {
		return nil, &RPCInvalidError{Message: fmt.Sprintf("JSON 解析失败: %v", err)}
	}

	// 处理 RPC 错误
	if rpcResp.Error != nil {
		return nil, handleRpcError(rpcResp.Error, resp.Text(), params)
	}

	// 确保有结果字段
	if rpcResp.Result == nil {
		return nil, &RPCInvalidError{Message: "响应中缺少 result 字段"}
	}

	return *rpcResp.Result, nil
}

// handleRpcError 根据 JSON-RPC 错误内容映射到具体的 Go 错误类型
func handleRpcError(rpcErr *JsonRpcError, rawText string, params interface{}) error {
	code := rpcErr.Code
	message := rpcErr.Message
	data := rpcErr.Data

	if debug := false; debug {
		log.Printf("JSON-RPC 错误: code=%d message=%s", code, message)
	}

	// UIAutomation 未连接
	if strings.Contains(rawText, "UiAutomation not connected") {
		return &UiAutomationNotConnectedError{Message: "UiAutomation not connected"}
	}
	if strings.Contains(message, "android.os.DeadObjectException") {
		return &UiAutomationNotConnectedError{Message: "android.os.DeadObjectException"}
	}
	if strings.Contains(message, "android.os.DeadSystemRuntimeException") {
		return &UiAutomationNotConnectedError{Message: "android.os.DeadSystemRuntimeException"}
	}

	// UI 元素未找到
	if strings.Contains(message, "uiautomator.UiObjectNotFoundException") {
		return &UiObjectNotFoundError{
			Code:    code,
			Message: message,
			Params:  params,
		}
	}

	// 栈溢出
	if strings.Contains(message, "java.lang.StackOverflowError") {
		truncated := data
		if len(data) > 2000 {
			truncated = data[:1000] + "..." + data[len(data)-1000:]
		}
		return &RPCStackOverflowError{
			RPCError: RPCError{
				Code:    code,
				Message: fmt.Sprintf("StackOverflowError: %s", message),
				Data:    truncated,
				Params:  params,
			},
		}
	}

	// 未知 RPC 错误
	return &RPCUnknownError{
		RPCError: RPCError{
			Code:    code,
			Message: fmt.Sprintf("未知 RPC 错误: %d %s", code, message),
			Data:    data,
			Params:  params,
		},
	}
}

// ---------- JSON-RPC 动态调用包装器 ----------

// JsonRpcWrapper 提供动态方法名的 JSON-RPC 调用
// 通过记录方法名并在 Call 时发送请求，实现类似 Python 的动态属性访问
type JsonRpcWrapper struct {
	// caller 为实际发送 JSON-RPC 请求的函数
	caller func(method string, params interface{}, timeout float64) (json.RawMessage, error)
}

// NewJsonRpcWrapper 创建 JSON-RPC 动态调用包装器
func NewJsonRpcWrapper(caller func(method string, params interface{}, timeout float64) (json.RawMessage, error)) *JsonRpcWrapper {
	return &JsonRpcWrapper{caller: caller}
}

// Call 发送 JSON-RPC 调用
// method 为 RPC 方法名（如 "click"、"setText"）
// params 为方法参数，通常为 []interface{} 或 map[string]interface{}
// timeout 为超时时间（秒），0 使用默认值
func (w *JsonRpcWrapper) Call(method string, params interface{}, timeout ...float64) (json.RawMessage, error) {
	t := HTTPTimeout
	if len(timeout) > 0 && timeout[0] > 0 {
		t = timeout[0]
	}
	return w.caller(method, params, t)
}

// CallResult 发送 JSON-RPC 调用并将结果解析到指定结构体
func (w *JsonRpcWrapper) CallResult(result interface{}, method string, params interface{}, timeout ...float64) error {
	raw, err := w.Call(method, params, timeout...)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}
