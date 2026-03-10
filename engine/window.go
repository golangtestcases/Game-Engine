package engine

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// NewWindow создает и инициализирует GLFW-окно вместе с OpenGL-контекстом.
// Функция также включает depth-test и захватывает курсор под режим FPS-камеры.
func NewWindow(width, height int, title string, fullscreen bool) (*glfw.Window, error) {
	if err := glfw.Init(); err != nil {
		return nil, err
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	var window *glfw.Window
	var err error

	if fullscreen {
		// В полноэкранном режиме берем параметры главного монитора.
		monitor := glfw.GetPrimaryMonitor()
		videoMode := monitor.GetVideoMode()
		window, err = glfw.CreateWindow(videoMode.Width, videoMode.Height, title, monitor, nil)
	} else {
		window, err = glfw.CreateWindow(width, height, title, nil, nil)
	}

	if err != nil {
		glfw.Terminate()
		return nil, err
	}

	window.MakeContextCurrent()
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

	if err := gl.Init(); err != nil {
		window.Destroy()
		glfw.Terminate()
		return nil, err
	}

	gl.Enable(gl.DEPTH_TEST)
	return window, nil
}
