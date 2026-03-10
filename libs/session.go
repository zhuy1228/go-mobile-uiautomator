package libs

import (
	"encoding/json"
	"fmt"
)

// Session 在 Device 基础上增加应用会话监控
// 每次 JSON-RPC 调用前检查目标应用是否仍在运行
// 如果应用退出或崩溃，会返回 SessionBrokenError
//
// 对应 Python 版本的 Session 类
type Session struct {
	*Device
	packageName string // 监控的应用包名
	pid         int    // 应用启动时的 PID
}

// NewSession 创建一个新的应用会话
// 启动应用并记录其 PID
func NewSession(device *Device, packageName string, attach bool) (*Session, error) {
	// 如果不是 attach 模式，先停止再启动应用
	if !attach {
		device.AppStop(packageName)
	}
	device.AppStart(packageName, "", false)

	// 等待应用启动并获取 PID
	pid, err := device.AppWait(packageName, 20.0, false)
	if err != nil {
		return nil, err
	}
	if pid == 0 {
		return nil, &DeviceError{Message: fmt.Sprintf("应用 %s 启动失败", packageName)}
	}

	return &Session{
		Device:      device,
		packageName: packageName,
		pid:         pid,
	}, nil
}

// PackageName 返回会话监控的应用包名
func (s *Session) PackageName() string {
	return s.packageName
}

// PID 返回应用的进程 ID
func (s *Session) PID() int {
	return s.pid
}

// Running 检查应用是否仍在运行
func (s *Session) Running() bool {
	currentPid := s.pidOfApp(s.packageName)
	return currentPid == s.pid && s.pid > 0
}

// jsonrpcCall 重写 Device 的 jsonrpcCall，增加会话状态检查
func (s *Session) jsonrpcCall(method string, params interface{}, timeout float64) (json.RawMessage, error) {
	if !s.Running() {
		return nil, &SessionBrokenError{
			Message: fmt.Sprintf("应用 %s (PID: %d) 已退出", s.packageName, s.pid),
		}
	}
	return s.Device.jsonrpcCall(method, params, timeout)
}

// Restart 重启应用
func (s *Session) Restart() error {
	s.Device.AppStop(s.packageName)
	s.Device.AppStart(s.packageName, "", false)
	pid, err := s.Device.AppWait(s.packageName, 20.0, false)
	if err != nil {
		return err
	}
	s.pid = pid
	return nil
}

// Close 关闭会话（停止应用）
func (s *Session) Close() {
	s.Device.AppStop(s.packageName)
	s.pid = 0
}
