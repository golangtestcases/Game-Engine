package engine

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// DebugOverlayRenderer draws simple 2D debug panels and bitmap text above the 3D frame.
type DebugOverlayRenderer struct {
	program uint32
	vao     uint32
	vbo     uint32

	screenSizeUniform int32
	colorUniform      int32
}

// NewDebugOverlayRenderer builds a minimal shader pipeline for screen-space overlays.
func NewDebugOverlayRenderer() *DebugOverlayRenderer {
	vertexShaderSource := `
#version 410
layout (location = 0) in vec2 vp;
uniform vec2 uScreenSize;

void main() {
    vec2 ndc = vec2(
        (vp.x / uScreenSize.x) * 2.0 - 1.0,
        1.0 - (vp.y / uScreenSize.y) * 2.0
    );
    gl_Position = vec4(ndc, 0.0, 1.0);
}
` + "\x00"

	fragmentShaderSource := `
#version 410
uniform vec4 uColor;
out vec4 fragColor;

void main() {
    fragColor = uColor;
}
` + "\x00"

	vertexShader := compileDebugOverlayShader(vertexShaderSource, gl.VERTEX_SHADER)
	fragmentShader := compileDebugOverlayShader(fragmentShaderSource, gl.FRAGMENT_SHADER)

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var linkStatus int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &linkStatus)
	if linkStatus == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		logText := make([]byte, logLength+1)
		gl.GetProgramInfoLog(program, logLength, nil, &logText[0])
		panic(fmt.Sprintf("debug overlay shader link failed: %s", string(logText)))
	}

	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4, nil, gl.DYNAMIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 0, nil)
	gl.BindVertexArray(0)

	return &DebugOverlayRenderer{
		program:           program,
		vao:               vao,
		vbo:               vbo,
		screenSizeUniform: gl.GetUniformLocation(program, gl.Str("uScreenSize\x00")),
		colorUniform:      gl.GetUniformLocation(program, gl.Str("uColor\x00")),
	}
}

// Begin prepares OpenGL state for 2D overlay draw calls.
func (r *DebugOverlayRenderer) Begin(screenW, screenH int32) {
	gl.UseProgram(r.program)
	gl.Uniform2f(r.screenSizeUniform, float32(screenW), float32(screenH))

	gl.BindVertexArray(r.vao)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.DEPTH_TEST)
}

// End restores default state after overlay rendering.
func (r *DebugOverlayRenderer) End() {
	gl.BindVertexArray(0)
	gl.UseProgram(0)
	gl.Disable(gl.BLEND)
	gl.Enable(gl.DEPTH_TEST)
}

// DrawRect draws one axis-aligned screen-space rectangle.
func (r *DebugOverlayRenderer) DrawRect(x, y, w, h float32, color [4]float32) {
	if w <= 0 || h <= 0 {
		return
	}
	vertices := appendOverlayRectVertices(nil, x, y, w, h)
	r.DrawTriangles(vertices, color)
}

// DrawTriangles draws packed xy vertices as triangles in screen-space.
func (r *DebugOverlayRenderer) DrawTriangles(vertices []float32, color [4]float32) {
	if len(vertices) == 0 {
		return
	}

	gl.Uniform4f(r.colorUniform, color[0], color[1], color[2], color[3])
	gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.DYNAMIC_DRAW)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(vertices)/2))
}

func appendOverlayRectVertices(dst []float32, x, y, w, h float32) []float32 {
	return append(dst,
		x, y,
		x+w, y,
		x+w, y+h,
		x, y,
		x+w, y+h,
		x, y+h,
	)
}

func compileDebugOverlayShader(source string, shaderType uint32) uint32 {
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		logText := make([]byte, logLength+1)
		gl.GetShaderInfoLog(shader, logLength, nil, &logText[0])
		panic(fmt.Sprintf("debug overlay shader compile failed: %s", string(logText)))
	}

	return shader
}

// DebugOverlayTextRenderer is a lightweight built-in bitmap text helper for debug overlays.
type DebugOverlayTextRenderer struct {
	overlay *DebugOverlayRenderer
}

// NewDebugOverlayTextRenderer builds a text renderer on top of the core overlay primitive renderer.
func NewDebugOverlayTextRenderer() *DebugOverlayTextRenderer {
	return &DebugOverlayTextRenderer{
		overlay: NewDebugOverlayRenderer(),
	}
}

func (r *DebugOverlayTextRenderer) Begin(screenW, screenH int32) {
	r.overlay.Begin(screenW, screenH)
}

func (r *DebugOverlayTextRenderer) End() {
	r.overlay.End()
}

func (r *DebugOverlayTextRenderer) DrawRect(x, y, w, h float32, color [4]float32) {
	r.overlay.DrawRect(x, y, w, h, color)
}

func (r *DebugOverlayTextRenderer) DrawText(text string, x, y, scale float32, color [4]float32) {
	if scale <= 0 || len(text) == 0 {
		return
	}

	cursorX := x
	cursorY := y
	advance := 6 * scale
	lineAdvance := 8 * scale

	vertices := make([]float32, 0, len(text)*7*5*12)
	for _, raw := range text {
		if raw == '\n' {
			cursorX = x
			cursorY += lineAdvance
			continue
		}

		ch := normalizeOverlayGlyphRune(raw)
		glyph, ok := debugOverlayFont5x7[ch]
		if !ok {
			glyph = debugOverlayFont5x7['?']
		}

		for row, rowBits := range glyph {
			for col := 0; col < 5; col++ {
				mask := uint8(1 << uint(4-col))
				if rowBits&mask == 0 {
					continue
				}
				px := cursorX + float32(col)*scale
				py := cursorY + float32(row)*scale
				vertices = appendOverlayRectVertices(vertices, px, py, scale, scale)
			}
		}

		cursorX += advance
	}

	r.overlay.DrawTriangles(vertices, color)
}

func normalizeOverlayGlyphRune(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - ('a' - 'A')
	}
	return r
}

var debugOverlayFont5x7 = map[rune][7]uint8{
	' ':  {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000},
	'\'': {0b00100, 0b00100, 0b00010, 0b00000, 0b00000, 0b00000, 0b00000},
	'(':  {0b00010, 0b00100, 0b01000, 0b01000, 0b01000, 0b00100, 0b00010},
	')':  {0b01000, 0b00100, 0b00010, 0b00010, 0b00010, 0b00100, 0b01000},
	'-':  {0b00000, 0b00000, 0b00000, 0b11111, 0b00000, 0b00000, 0b00000},
	'.':  {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00100, 0b00100},
	'/':  {0b00001, 0b00010, 0b00100, 0b01000, 0b10000, 0b00000, 0b00000},
	':':  {0b00000, 0b00100, 0b00100, 0b00000, 0b00100, 0b00100, 0b00000},
	'>':  {0b10000, 0b01000, 0b00100, 0b00010, 0b00100, 0b01000, 0b10000},
	'?':  {0b01110, 0b10001, 0b00001, 0b00010, 0b00100, 0b00000, 0b00100},
	'_':  {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b11111},
	'0':  {0b01110, 0b10001, 0b10011, 0b10101, 0b11001, 0b10001, 0b01110},
	'1':  {0b00100, 0b01100, 0b00100, 0b00100, 0b00100, 0b00100, 0b01110},
	'2':  {0b01110, 0b10001, 0b00001, 0b00010, 0b00100, 0b01000, 0b11111},
	'3':  {0b11110, 0b00001, 0b00001, 0b01110, 0b00001, 0b00001, 0b11110},
	'4':  {0b00010, 0b00110, 0b01010, 0b10010, 0b11111, 0b00010, 0b00010},
	'5':  {0b11111, 0b10000, 0b10000, 0b11110, 0b00001, 0b00001, 0b11110},
	'6':  {0b01110, 0b10000, 0b10000, 0b11110, 0b10001, 0b10001, 0b01110},
	'7':  {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b01000, 0b01000},
	'8':  {0b01110, 0b10001, 0b10001, 0b01110, 0b10001, 0b10001, 0b01110},
	'9':  {0b01110, 0b10001, 0b10001, 0b01111, 0b00001, 0b00001, 0b01110},
	'A':  {0b01110, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'B':  {0b11110, 0b10001, 0b10001, 0b11110, 0b10001, 0b10001, 0b11110},
	'C':  {0b01111, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b01111},
	'D':  {0b11110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b11110},
	'E':  {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b11111},
	'F':  {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b10000},
	'G':  {0b01111, 0b10000, 0b10000, 0b10111, 0b10001, 0b10001, 0b01110},
	'H':  {0b10001, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'I':  {0b11111, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b11111},
	'J':  {0b00001, 0b00001, 0b00001, 0b00001, 0b10001, 0b10001, 0b01110},
	'K':  {0b10001, 0b10010, 0b10100, 0b11000, 0b10100, 0b10010, 0b10001},
	'L':  {0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b11111},
	'M':  {0b10001, 0b11011, 0b10101, 0b10101, 0b10001, 0b10001, 0b10001},
	'N':  {0b10001, 0b11001, 0b10101, 0b10011, 0b10001, 0b10001, 0b10001},
	'O':  {0b01110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'P':  {0b11110, 0b10001, 0b10001, 0b11110, 0b10000, 0b10000, 0b10000},
	'Q':  {0b01110, 0b10001, 0b10001, 0b10001, 0b10101, 0b10010, 0b01101},
	'R':  {0b11110, 0b10001, 0b10001, 0b11110, 0b10100, 0b10010, 0b10001},
	'S':  {0b01111, 0b10000, 0b10000, 0b01110, 0b00001, 0b00001, 0b11110},
	'T':  {0b11111, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100},
	'U':  {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'V':  {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01010, 0b00100},
	'W':  {0b10001, 0b10001, 0b10001, 0b10101, 0b10101, 0b10101, 0b01010},
	'X':  {0b10001, 0b10001, 0b01010, 0b00100, 0b01010, 0b10001, 0b10001},
	'Y':  {0b10001, 0b10001, 0b01010, 0b00100, 0b00100, 0b00100, 0b00100},
	'Z':  {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b10000, 0b11111},
}
