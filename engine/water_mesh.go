package engine

import (
	"github.com/go-gl/gl/v4.1-core/gl"
)

// NewMeshWithNormals для WaterRenderer создает mesh с позициями и нормалями.
// Отдельная реализация оставлена для симметрии API с базовым Renderer.
func (w *WaterRenderer) NewMeshWithNormals(vertices, normals []float32) Mesh {
	var vao, vbo, nbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

	gl.GenBuffers(1, &nbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, nbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(normals), gl.Ptr(normals), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 0, nil)

	return Mesh{VAO: vao, VBO: vbo, NBO: nbo, VertexCount: int32(len(vertices) / 3), HasNormals: true}
}
