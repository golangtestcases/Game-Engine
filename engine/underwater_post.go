package engine

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// UnderwaterPostProcessor — финальный полноэкранный pass для эффекта подводной среды.
// Использует цвет/глубину сцены, каустику и опциональную текстуру volumetric лучей.
type UnderwaterPostProcessor struct {
	program uint32
	quad    *FullscreenQuad

	colorUniform          int32
	depthUniform          int32
	causticUniform        int32
	shaftUniform          int32
	shaftEnabledUniform   int32
	shaftUnderOnlyUniform int32
	shaftSurfaceUniform   int32
	screenSizeUniform     int32
	nearUniform           int32
	farUniform            int32
	timeUniform           int32
	waterLevelUniform     int32
	underwaterUniform     int32
	cameraPosUniform      int32
	lightDirUniform       int32
	fogColorUniform       int32
	depthTintUniform      int32
	sunColorUniform       int32
	visibilityUniform     int32
	invProjectionUniform  int32
	invViewUniform        int32

	causticTex uint32
}

// NewUnderwaterPostProcessor создает шейдер и подготавливает fullscreen quad.
func NewUnderwaterPostProcessor() *UnderwaterPostProcessor {
	vertexSource := loadShaderSourceFile("assets/shaders/water/underwater_post.vert")
	fragmentSource := loadShaderSourceFile("assets/shaders/water/underwater_post.frag")

	vs := compileShader(vertexSource, gl.VERTEX_SHADER)
	fs := compileShader(fragmentSource, gl.FRAGMENT_SHADER)

	program := gl.CreateProgram()
	gl.AttachShader(program, vs)
	gl.AttachShader(program, fs)
	gl.LinkProgram(program)

	var linkStatus int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &linkStatus)
	if linkStatus == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		logText := make([]byte, logLength+1)
		gl.GetProgramInfoLog(program, logLength, nil, &logText[0])
		panic(fmt.Sprintf("underwater post shader link failed: %s", string(logText)))
	}

	post := &UnderwaterPostProcessor{
		program: program,
		quad:    NewFullscreenQuad(),

		colorUniform:          gl.GetUniformLocation(program, gl.Str("uColorTex\x00")),
		depthUniform:          gl.GetUniformLocation(program, gl.Str("uDepthTex\x00")),
		causticUniform:        gl.GetUniformLocation(program, gl.Str("uCausticTex\x00")),
		shaftUniform:          gl.GetUniformLocation(program, gl.Str("uShaftTex\x00")),
		shaftEnabledUniform:   gl.GetUniformLocation(program, gl.Str("uShaftEnabled\x00")),
		shaftUnderOnlyUniform: gl.GetUniformLocation(program, gl.Str("uShaftUnderwaterOnly\x00")),
		shaftSurfaceUniform:   gl.GetUniformLocation(program, gl.Str("uShaftSurfaceIntensity\x00")),
		screenSizeUniform:     gl.GetUniformLocation(program, gl.Str("uScreenSize\x00")),
		nearUniform:           gl.GetUniformLocation(program, gl.Str("uNear\x00")),
		farUniform:            gl.GetUniformLocation(program, gl.Str("uFar\x00")),
		timeUniform:           gl.GetUniformLocation(program, gl.Str("uTime\x00")),
		waterLevelUniform:     gl.GetUniformLocation(program, gl.Str("uWaterLevel\x00")),
		underwaterUniform:     gl.GetUniformLocation(program, gl.Str("uUnderwaterBlend\x00")),
		cameraPosUniform:      gl.GetUniformLocation(program, gl.Str("uCameraPos\x00")),
		lightDirUniform:       gl.GetUniformLocation(program, gl.Str("uLightDir\x00")),
		fogColorUniform:       gl.GetUniformLocation(program, gl.Str("uFogColor\x00")),
		depthTintUniform:      gl.GetUniformLocation(program, gl.Str("uDepthTint\x00")),
		sunColorUniform:       gl.GetUniformLocation(program, gl.Str("uSunColor\x00")),
		visibilityUniform:     gl.GetUniformLocation(program, gl.Str("uVisibilityDistance\x00")),
		invProjectionUniform:  gl.GetUniformLocation(program, gl.Str("uInvProjection\x00")),
		invViewUniform:        gl.GetUniformLocation(program, gl.Str("uInvView\x00")),

		causticTex: generateCausticTexture(512, 4.2),
	}

	gl.UseProgram(program)
	gl.Uniform1i(post.colorUniform, 0)
	gl.Uniform1i(post.depthUniform, 1)
	gl.Uniform1i(post.causticUniform, 2)
	gl.Uniform1i(post.shaftUniform, 3)

	return post
}

// Render применяет подводный пост-эффект в default framebuffer (экран).
// Важно: функция сама временно отключает depth-test и затем возвращает его обратно.
func (p *UnderwaterPostProcessor) Render(
	colorTex, depthTex, shaftTex uint32,
	shaftEnabled bool,
	shaftParams UnderwaterLightShaftParams,
	environment UnderwaterShaftEnvironment,
	width, height int32,
	nearPlane, farPlane, timeSec, waterLevel, underwaterBlend float32,
	cameraPos, lightDir mgl32.Vec3,
	invProjection, invView mgl32.Mat4,
) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, width, height)
	gl.Disable(gl.DEPTH_TEST)
	gl.DepthMask(false)

	gl.UseProgram(p.program)
	environment = sanitizeUnderwaterShaftEnvironment(environment)
	gl.Uniform2f(p.screenSizeUniform, float32(width), float32(height))
	gl.Uniform1f(p.nearUniform, nearPlane)
	gl.Uniform1f(p.farUniform, farPlane)
	gl.Uniform1f(p.timeUniform, timeSec)
	gl.Uniform1f(p.waterLevelUniform, waterLevel)
	gl.Uniform1f(p.underwaterUniform, underwaterBlend)
	if shaftEnabled {
		gl.Uniform1f(p.shaftEnabledUniform, 1.0)
	} else {
		gl.Uniform1f(p.shaftEnabledUniform, 0.0)
	}
	if shaftParams.UnderwaterOnly {
		gl.Uniform1f(p.shaftUnderOnlyUniform, 1.0)
	} else {
		gl.Uniform1f(p.shaftUnderOnlyUniform, 0.0)
	}
	gl.Uniform1f(p.shaftSurfaceUniform, shaftParams.SurfaceIntensity)
	gl.Uniform3f(p.cameraPosUniform, cameraPos.X(), cameraPos.Y(), cameraPos.Z())
	gl.Uniform3f(p.lightDirUniform, lightDir.X(), lightDir.Y(), lightDir.Z())
	gl.Uniform3f(p.fogColorUniform, environment.FogColor.X(), environment.FogColor.Y(), environment.FogColor.Z())
	gl.Uniform3f(p.depthTintUniform, environment.DepthTint.X(), environment.DepthTint.Y(), environment.DepthTint.Z())
	gl.Uniform3f(p.sunColorUniform, environment.SunColor.X(), environment.SunColor.Y(), environment.SunColor.Z())
	gl.Uniform1f(p.visibilityUniform, environment.VisibilityDistance)
	gl.UniformMatrix4fv(p.invProjectionUniform, 1, false, &invProjection[0])
	gl.UniformMatrix4fv(p.invViewUniform, 1, false, &invView[0])

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, colorTex)
	gl.ActiveTexture(gl.TEXTURE0 + 1)
	gl.BindTexture(gl.TEXTURE_2D, depthTex)
	gl.ActiveTexture(gl.TEXTURE0 + 2)
	gl.BindTexture(gl.TEXTURE_2D, p.causticTex)
	gl.ActiveTexture(gl.TEXTURE0 + 3)
	gl.BindTexture(gl.TEXTURE_2D, shaftTex)

	p.quad.Draw()

	gl.DepthMask(true)
	gl.Enable(gl.DEPTH_TEST)
}
