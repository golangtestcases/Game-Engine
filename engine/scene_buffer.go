package engine

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// SceneBuffer хранит offscreen-рендер сцены: HDR-цвет + depth.
// Используется как источник для водных/пост-эффектов.
type SceneBuffer struct {
	FBO      uint32
	ColorTex uint32
	DepthTex uint32
	Width    int32
	Height   int32
}

func NewSceneBuffer(width, height int32) *SceneBuffer {
	sb := &SceneBuffer{
		Width:  width,
		Height: height,
	}
	sb.allocate()
	return sb
}

func (s *SceneBuffer) allocate() {
	gl.GenFramebuffers(1, &s.FBO)
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.FBO)

	gl.GenTextures(1, &s.ColorTex)
	gl.BindTexture(gl.TEXTURE_2D, s.ColorTex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, s.Width, s.Height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, s.ColorTex, 0)

	gl.GenTextures(1, &s.DepthTex)
	gl.BindTexture(gl.TEXTURE_2D, s.DepthTex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT24, s.Width, s.Height, 0, gl.DEPTH_COMPONENT, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, s.DepthTex, 0)

	attachments := []uint32{gl.COLOR_ATTACHMENT0}
	gl.DrawBuffers(1, &attachments[0])

	if status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
		panic(fmt.Sprintf("scene buffer incomplete: 0x%x", status))
	}

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

func (s *SceneBuffer) Resize(width, height int32) {
	if width == s.Width && height == s.Height {
		return
	}
	s.Delete()
	s.Width = width
	s.Height = height
	s.allocate()
}

func (s *SceneBuffer) Bind() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.FBO)
	gl.Viewport(0, 0, s.Width, s.Height)
}

func (s *SceneBuffer) Unbind(screenWidth, screenHeight int32) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, screenWidth, screenHeight)
}

// BindTextures привязывает color/depth к заданным texture unit.
func (s *SceneBuffer) BindTextures(colorUnit, depthUnit uint32) {
	gl.ActiveTexture(gl.TEXTURE0 + colorUnit)
	gl.BindTexture(gl.TEXTURE_2D, s.ColorTex)

	gl.ActiveTexture(gl.TEXTURE0 + depthUnit)
	gl.BindTexture(gl.TEXTURE_2D, s.DepthTex)
}

func (s *SceneBuffer) BindColorTexture(colorUnit uint32) {
	gl.ActiveTexture(gl.TEXTURE0 + colorUnit)
	gl.BindTexture(gl.TEXTURE_2D, s.ColorTex)
}

func (s *SceneBuffer) BindDepthTexture(depthUnit uint32) {
	gl.ActiveTexture(gl.TEXTURE0 + depthUnit)
	gl.BindTexture(gl.TEXTURE_2D, s.DepthTex)
}

// BlitToBuffer копирует содержимое текущего SceneBuffer в другой буфер (или в экран).
func (s *SceneBuffer) BlitToBuffer(dst *SceneBuffer, copyDepth bool) {
	drawFBO := uint32(0)
	drawWidth := s.Width
	drawHeight := s.Height
	if dst != nil {
		drawFBO = dst.FBO
		drawWidth = dst.Width
		drawHeight = dst.Height
	}

	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, s.FBO)
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, drawFBO)

	mask := uint32(gl.COLOR_BUFFER_BIT)
	if copyDepth {
		mask |= gl.DEPTH_BUFFER_BIT
	}

	gl.BlitFramebuffer(
		0, 0, s.Width, s.Height,
		0, 0, drawWidth, drawHeight,
		mask,
		gl.NEAREST,
	)

	gl.BindFramebuffer(gl.FRAMEBUFFER, drawFBO)
}

func (s *SceneBuffer) BlitToScreen(screenWidth, screenHeight int32, copyDepth bool) {
	s.BlitToBuffer(nil, copyDepth)
	gl.Viewport(0, 0, screenWidth, screenHeight)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

func (s *SceneBuffer) Delete() {
	if s.ColorTex != 0 {
		gl.DeleteTextures(1, &s.ColorTex)
		s.ColorTex = 0
	}
	if s.DepthTex != 0 {
		gl.DeleteTextures(1, &s.DepthTex)
		s.DepthTex = 0
	}
	if s.FBO != 0 {
		gl.DeleteFramebuffers(1, &s.FBO)
		s.FBO = 0
	}
}
