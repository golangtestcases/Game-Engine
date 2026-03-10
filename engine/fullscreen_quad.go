package engine

import "github.com/go-gl/gl/v4.1-core/gl"

// FullscreenQuad — статическая геометрия из двух треугольников на весь экран.
// Используется для полноэкранных post-process pass'ов.
type FullscreenQuad struct {
	vao uint32
	vbo uint32
}

// NewFullscreenQuad создает VAO/VBO с позициями и UV.
func NewFullscreenQuad() *FullscreenQuad {
	verts := []float32{
		-1.0, -1.0, 0.0, 0.0,
		1.0, -1.0, 1.0, 0.0,
		1.0, 1.0, 1.0, 1.0,
		-1.0, -1.0, 0.0, 0.0,
		1.0, 1.0, 1.0, 1.0,
		-1.0, 1.0, 0.0, 1.0,
	}

	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)

	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(verts), gl.Ptr(verts), gl.STATIC_DRAW)

	const stride = 4 * 4
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, nil)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(2*4))

	gl.BindVertexArray(0)
	return &FullscreenQuad{vao: vao, vbo: vbo}
}

// Draw рисует quad в текущий framebuffer.
func (q *FullscreenQuad) Draw() {
	gl.BindVertexArray(q.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, 6)
	gl.BindVertexArray(0)
}
