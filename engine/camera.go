package engine

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/utils"
)

// Camera хранит состояние FPS-камеры и параметры управления.
type Camera struct {
	Position mgl32.Vec3
	Front    mgl32.Vec3
	Up       mgl32.Vec3

	Yaw   float32
	Pitch float32

	LastX float64
	LastY float64

	FirstMouse bool

	Speed       float32
	Sensitivity float32

	// MouseLookEnabled позволяет временно отключать поворот камерой мышью,
	// например при открытии консоли или UI.
	MouseLookEnabled bool
}

// NewCamera создает камеру с базовой ориентацией вдоль -Z,
// что соответствует типичному OpenGL-view пространству.
func NewCamera(position mgl32.Vec3, lastX, lastY float64) *Camera {
	return &Camera{
		Position:         position,
		Front:            mgl32.Vec3{0, 0, -1},
		Up:               mgl32.Vec3{0, 1, 0},
		Yaw:              -90,
		Pitch:            0,
		LastX:            lastX,
		LastY:            lastY,
		FirstMouse:       true,
		Speed:            3.0,
		Sensitivity:      0.1,
		MouseLookEnabled: true,
	}
}

// ProcessInput применяет режим "полёта":
// горизонтальное движение плюс отдельные клавиши подъёма/спуска.
func (c *Camera) ProcessInput(window *glfw.Window, deltaTime float32) {
	c.ProcessSwimInput(window, deltaTime)

	step := c.Speed * deltaTime
	if window.GetKey(glfw.KeySpace) == glfw.Press {
		c.Position = c.Position.Add(c.Up.Mul(step))
	}
	if window.GetKey(glfw.KeyLeftControl) == glfw.Press {
		c.Position = c.Position.Sub(c.Up.Mul(step))
	}
}

// ProcessSwimInput двигает камеру в направлении взгляда и по стрейфу,
// без принудительной фиксации на горизонтальной плоскости.
func (c *Camera) ProcessSwimInput(window *glfw.Window, deltaTime float32) {
	step := c.Speed * deltaTime
	if window.GetKey(glfw.KeyW) == glfw.Press {
		c.Position = c.Position.Add(c.Front.Mul(step))
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		c.Position = c.Position.Sub(c.Front.Mul(step))
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		c.Position = c.Position.Sub(c.Front.Cross(c.Up).Normalize().Mul(step))
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		c.Position = c.Position.Add(c.Front.Cross(c.Up).Normalize().Mul(step))
	}
}

// ProcessGroundInput ограничивает перемещение плоскостью XZ,
// что удобно для "ходьбы" по поверхности.
func (c *Camera) ProcessGroundInput(window *glfw.Window, deltaTime float32) {
	step := c.Speed * deltaTime

	forward := mgl32.Vec3{c.Front.X(), 0, c.Front.Z()}
	if forward.Len() < 0.0001 {
		forward = mgl32.Vec3{0, 0, -1}
	} else {
		forward = forward.Normalize()
	}
	right := forward.Cross(c.Up).Normalize()

	if window.GetKey(glfw.KeyW) == glfw.Press {
		c.Position = c.Position.Add(forward.Mul(step))
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		c.Position = c.Position.Sub(forward.Mul(step))
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		c.Position = c.Position.Sub(right.Mul(step))
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		c.Position = c.Position.Add(right.Mul(step))
	}
}

// MouseCallback обновляет yaw/pitch и вектор Front.
// Pitch ограничен, чтобы избежать сингулярности и инверсии управления.
func (c *Camera) MouseCallback(_ *glfw.Window, xpos float64, ypos float64) {
	if !c.MouseLookEnabled {
		c.FirstMouse = true
		c.LastX = xpos
		c.LastY = ypos
		return
	}

	if c.FirstMouse {
		c.LastX = xpos
		c.LastY = ypos
		c.FirstMouse = false
	}

	xoffset := float32(xpos - c.LastX)
	yoffset := float32(c.LastY - ypos)
	c.LastX = xpos
	c.LastY = ypos

	xoffset *= c.Sensitivity
	yoffset *= c.Sensitivity

	c.Yaw += xoffset
	c.Pitch += yoffset

	if c.Pitch > 89.0 {
		c.Pitch = 89.0
	}
	if c.Pitch < -89.0 {
		c.Pitch = -89.0
	}

	direction := mgl32.Vec3{
		utils.CosDeg(c.Yaw) * utils.CosDeg(c.Pitch),
		utils.SinDeg(c.Pitch),
		utils.SinDeg(c.Yaw) * utils.CosDeg(c.Pitch),
	}
	c.Front = direction.Normalize()
}

// SetMouseLookEnabled переключает режим обзора мышью.
// При смене режима сбрасывается FirstMouse, чтобы не было рывка на первом кадре.
func (c *Camera) SetMouseLookEnabled(enabled bool) {
	if c.MouseLookEnabled == enabled {
		return
	}
	c.MouseLookEnabled = enabled
	c.FirstMouse = true
}

// ViewMatrix формирует стандартную матрицу вида для текущего положения и направления.
func (c *Camera) ViewMatrix() mgl32.Mat4 {
	return mgl32.LookAtV(c.Position, c.Position.Add(c.Front), c.Up)
}
