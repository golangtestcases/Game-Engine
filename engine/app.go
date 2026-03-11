package engine

import (
	"fmt"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

func init() {
	// GLFW/OpenGL-контекст должен жить в фиксированном OS-потоке.
	runtime.LockOSThread()
}

// Context передается в колбэки жизненного цикла игры.
// Он содержит общие подсистемы и данные текущего кадра.
type Context struct {
	Window       *glfw.Window
	Camera       *Camera
	Renderer     *Renderer
	GlowRenderer *GlowRenderer
	Projection   mgl32.Mat4

	// Time — абсолютное время от старта приложения.
	// DeltaTime — длительность предыдущего кадра, используется для frame-rate independent логики.
	Time      float32
	DeltaTime float32
}

// Game — минимальный контракт игры, запускаемой движком.
// Порядок вызова: Init один раз, затем в цикле Update -> Render.
type Game interface {
	Init(ctx *Context) error
	Update(ctx *Context) error
	Render(ctx *Context) error
}

// Run инициализирует движок, создает окно и запускает главный цикл.
func Run(cfg Config, game Game) error {
	cfg = cfg.withDefaults()

	window, err := NewWindow(cfg.Window.Width, cfg.Window.Height, cfg.Window.Title, cfg.Window.Fullscreen)
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}
	defer glfw.Terminate()

	// Переключаем синхронизацию кадров с частотой монитора.
	if cfg.Window.VSync {
		glfw.SwapInterval(1)
	} else {
		glfw.SwapInterval(0)
	}

	// В полноэкранном режиме фактический размер может отличаться от requested.
	width, height := window.GetSize()

	camera := NewCamera(mgl32.Vec3{0, 2, 5}, float64(width)/2, float64(height)/2)
	window.SetCursorPosCallback(camera.MouseCallback)

	renderer := NewRenderer()
	glowRenderer := NewGlowRenderer()
	projection := mgl32.Perspective(mgl32.DegToRad(cfg.Graphics.FOV), float32(width)/float32(height), cfg.Graphics.Near, cfg.Graphics.Far)

	ctx := &Context{
		Window:       window,
		Camera:       camera,
		Renderer:     renderer,
		GlowRenderer: glowRenderer,
		Projection:   projection,
	}

	if err := game.Init(ctx); err != nil {
		return fmt.Errorf("game init: %w", err)
	}

	// Главный игровой цикл.
	var lastFrame float32
	for !window.ShouldClose() {
		currentFrame := float32(glfw.GetTime())
		deltaTime := currentFrame - lastFrame
		lastFrame = currentFrame

		ctx.Time = currentFrame
		ctx.DeltaTime = deltaTime

		if window.GetKey(glfw.KeyEscape) == glfw.Press {
			window.SetShouldClose(true)
		}

		if err := game.Update(ctx); err != nil {
			return fmt.Errorf("game update: %w", err)
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		if err := game.Render(ctx); err != nil {
			return fmt.Errorf("game render: %w", err)
		}

		window.SwapBuffers()
		glfw.PollEvents()
	}

	return nil
}
