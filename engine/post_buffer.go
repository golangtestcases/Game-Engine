package engine

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// PostColorBuffer — упрощенный offscreen-буфер только с цветовым attachment.
// Подходит для промежуточных пост-процесс проходов, где depth не нужен.
type PostColorBuffer struct {
	FBO      uint32
	ColorTex uint32
	Width    int32
	Height   int32
}

func NewPostColorBuffer(width, height int32) *PostColorBuffer {
	buffer := &PostColorBuffer{
		Width:  width,
		Height: height,
	}
	buffer.allocate()
	return buffer
}

func (b *PostColorBuffer) allocate() {
	gl.GenFramebuffers(1, &b.FBO)
	gl.BindFramebuffer(gl.FRAMEBUFFER, b.FBO)

	gl.GenTextures(1, &b.ColorTex)
	gl.BindTexture(gl.TEXTURE_2D, b.ColorTex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, b.Width, b.Height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, b.ColorTex, 0)

	attachments := []uint32{gl.COLOR_ATTACHMENT0}
	gl.DrawBuffers(1, &attachments[0])

	if status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
		panic(fmt.Sprintf("post color buffer incomplete: 0x%x", status))
	}

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

func (b *PostColorBuffer) Resize(width, height int32) {
	if width == b.Width && height == b.Height {
		return
	}
	b.Delete()
	b.Width = width
	b.Height = height
	b.allocate()
}

func (b *PostColorBuffer) Bind() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, b.FBO)
	gl.Viewport(0, 0, b.Width, b.Height)
}

func (b *PostColorBuffer) Unbind(screenWidth, screenHeight int32) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, screenWidth, screenHeight)
}

func (b *PostColorBuffer) Delete() {
	if b.ColorTex != 0 {
		gl.DeleteTextures(1, &b.ColorTex)
		b.ColorTex = 0
	}
	if b.FBO != 0 {
		gl.DeleteFramebuffers(1, &b.FBO)
		b.FBO = 0
	}
}
