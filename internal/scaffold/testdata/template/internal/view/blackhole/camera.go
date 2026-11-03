package blackhole

import "math"

// 摄像机与向量运算：移植自 mygo examples/blackhole（MIT）。

// vec3 场景中的一个点或方向，y 轴向上，单位为黑洞半径。
type vec3 [3]float64

func (a vec3) add(b vec3) vec3    { return vec3{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func (a vec3) mul(k float64) vec3 { return vec3{a[0] * k, a[1] * k, a[2] * k} }
func (a vec3) dot(b vec3) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func (a vec3) norm() vec3         { return a.mul(1 / math.Sqrt(a.dot(a))) }
func (a vec3) f32() [3]float32    { return [3]float32{float32(a[0]), float32(a[1]), float32(a[2])} }

func (a vec3) cross(b vec3) vec3 {
	return vec3{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

// focal 是摄像机焦距，以视口高度为单位。
const focal = 1.5

// camera 绕黑洞旋转：绕盘面的方位角（yaw）、离盘面的仰角（pitch）、
// 到黑洞的距离，以及松手后两个角度各自的惯性转速。
type camera struct {
	yaw, pitch, dist   float64
	spinYaw, spinPitch float64
}

// axes 返回摄像机的位置与它的右、上、前三轴。
func (c *camera) axes() (eye, right, up, forward vec3) {
	cp := math.Cos(c.pitch)
	eye = vec3{cp * math.Sin(c.yaw), math.Sin(c.pitch), cp * math.Cos(c.yaw)}.mul(c.dist)
	forward = eye.mul(-1).norm()
	right = forward.cross(vec3{0, 1, 0}).norm()
	up = right.cross(forward)
	return
}

// spinning 报告摄像机是否还在自行旋转（决定要不要继续请求下一帧）。
func (c *camera) spinning() bool {
	return math.Abs(c.spinYaw) > 1e-3 || math.Abs(c.spinPitch) > 1e-3
}
