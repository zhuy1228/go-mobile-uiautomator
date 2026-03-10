package libs

import "fmt"

// SwipeExt 提供扩展的滑动操作
// 支持按方向、比例、区域进行滑动
// 对应 Python 版本的 SwipeExt 类
type SwipeExt struct {
	device *Device
}

// NewSwipeExt 创建扩展滑动操作实例
func NewSwipeExt(device *Device) *SwipeExt {
	return &SwipeExt{device: device}
}

// SwipeDirection 按方向滑动
// direction: 滑动方向（DirectionLeft/Right/Up/Down）
// scale: 滑动比例（0-1.0），默认 0.9
// box: 滑动区域 [left, top, right, bottom]，nil 表示全屏
// steps: 滑动步数
func (s *SwipeExt) SwipeDirection(direction Direction, scale float64, box *[4]int, steps int) error {
	if scale <= 0 || scale > 1.0 {
		scale = 0.9
	}
	if steps <= 0 {
		steps = ScrollSteps
	}

	var lx, ly, rx, ry int
	if box != nil {
		lx, ly, rx, ry = box[0], box[1], box[2], box[3]
	} else {
		w, h, err := s.device.WindowSize()
		if err != nil {
			return err
		}
		lx, ly = 0, 0
		rx, ry = w, h
	}

	width := rx - lx
	height := ry - ly

	hOffset := int(float64(width) * (1 - scale) / 2)
	vOffset := int(float64(height) * (1 - scale) / 2)

	center := [2]int{lx + width/2, ly + height/2}
	left := [2]int{lx + hOffset, ly + height/2}
	up := [2]int{lx + width/2, ly + vOffset}
	right := [2]int{rx - hOffset, ly + height/2}
	bottom := [2]int{lx + width/2, ry - vOffset}

	switch direction {
	case DirectionLeft:
		return s.device.Swipe(right[0], right[1], left[0], left[1], steps)
	case DirectionRight:
		return s.device.Swipe(left[0], left[1], right[0], right[1], steps)
	case DirectionUp:
		return s.device.Swipe(center[0], center[1], up[0], up[1], steps)
	case DirectionDown:
		return s.device.Swipe(center[0], center[1], bottom[0], bottom[1], steps)
	default:
		return fmt.Errorf("不支持的方向: %s", string(direction))
	}
}

// Left 向左滑动
func (s *SwipeExt) Left(scale float64, steps int) error {
	return s.SwipeDirection(DirectionLeft, scale, nil, steps)
}

// Right 向右滑动
func (s *SwipeExt) Right(scale float64, steps int) error {
	return s.SwipeDirection(DirectionRight, scale, nil, steps)
}

// Up 向上滑动
func (s *SwipeExt) Up(scale float64, steps int) error {
	return s.SwipeDirection(DirectionUp, scale, nil, steps)
}

// Down 向下滑动
func (s *SwipeExt) Down(scale float64, steps int) error {
	return s.SwipeDirection(DirectionDown, scale, nil, steps)
}
