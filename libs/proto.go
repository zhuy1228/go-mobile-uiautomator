package libs

// ---------- 协议常量 ----------

// ScrollSteps 默认滚动步数
// 每步约 5ms，55 步约 275ms
const ScrollSteps = 55

// HTTPTimeout 默认 HTTP 请求超时时间（秒）
const HTTPTimeout = 300.0

// DeviceServerPort UIAutomator2 服务默认端口
const DeviceServerPort = 9008

// DefaultADBAddr 默认 ADB 服务器地址
const DefaultADBAddr = "127.0.0.1:5037"

// ---------- 方向枚举 ----------

// Direction 表示滑动/滚动方向
type Direction string

const (
	// DirectionLeft 向左
	DirectionLeft Direction = "left"
	// DirectionRight 向右
	DirectionRight Direction = "right"
	// DirectionUp 向上
	DirectionUp Direction = "up"
	// DirectionDown 向下
	DirectionDown Direction = "down"
	// DirectionForward 向前（等同于向下）
	DirectionForward Direction = "forward"
	// DirectionBackward 向后（等同于向上）
	DirectionBackward Direction = "backward"
)

// ---------- 触摸事件常量 ----------

const (
	// ActionDown 手指按下事件
	ActionDown = 0
	// ActionUp 手指抬起事件
	ActionUp = 1
	// ActionMove 手指移动事件
	ActionMove = 2
)

// ---------- 设备方向映射 ----------

// OrientationInfo 设备方向信息
type OrientationInfo struct {
	Value    int    // displayRotation 值
	Name     string // 方向名称
	Short    string // 简写
	Rotation int    // 旋转角度
}

// Orientations 所有设备方向定义
var Orientations = []OrientationInfo{
	{Value: 0, Name: "natural", Short: "n", Rotation: 0},
	{Value: 1, Name: "left", Short: "l", Rotation: 90},
	{Value: 2, Name: "upsidedown", Short: "u", Rotation: 180},
	{Value: 3, Name: "right", Short: "r", Rotation: 270},
}
