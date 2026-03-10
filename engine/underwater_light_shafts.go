package engine

import (
	"fmt"
	"math"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// UnderwaterLightShaftParams задает качество и характер god-rays под водой.
// Большинство параметров можно менять в runtime для баланса качества/производительности.
type UnderwaterLightShaftParams struct {
	Intensity               float32
	Density                 float32
	Falloff                 float32
	ScatteringStrength      float32
	UnderwaterBlendFactor   float32
	SurfaceIntensity        float32
	UnderwaterOnly          bool
	NoiseDistortionStrength float32
	NoiseDistortionSpeed    float32
	SampleCount             int32
	ResolutionScale         float32
	BlurRadius              float32
}

// DefaultUnderwaterLightShaftParams возвращает настройки «высокого» качества по умолчанию.
// UnderwaterShaftEnvironment описывает цвет/видимость среды для лучей.
type UnderwaterShaftEnvironment struct {
	FogColor           mgl32.Vec3
	DepthTint          mgl32.Vec3
	SunColor           mgl32.Vec3
	VisibilityDistance float32
}

func DefaultUnderwaterShaftEnvironment() UnderwaterShaftEnvironment {
	return UnderwaterShaftEnvironment{
		FogColor:           mgl32.Vec3{0.05, 0.24, 0.34},
		DepthTint:          mgl32.Vec3{0.08, 0.34, 0.48},
		SunColor:           mgl32.Vec3{1.0, 0.95, 0.86},
		VisibilityDistance: 90.0,
	}
}

func sanitizeUnderwaterShaftEnvironment(environment UnderwaterShaftEnvironment) UnderwaterShaftEnvironment {
	if environment.FogColor.Len() < 0.0001 {
		environment.FogColor = mgl32.Vec3{0.05, 0.24, 0.34}
	}
	if environment.DepthTint.Len() < 0.0001 {
		environment.DepthTint = mgl32.Vec3{0.08, 0.34, 0.48}
	}
	if environment.SunColor.Len() < 0.0001 {
		environment.SunColor = mgl32.Vec3{1.0, 0.95, 0.86}
	}
	if environment.VisibilityDistance < 4.0 {
		environment.VisibilityDistance = 4.0
	}
	return environment
}

func DefaultUnderwaterLightShaftParams() UnderwaterLightShaftParams {
	return UnderwaterLightShaftParams{
		Intensity:               1.95,
		Density:                 0.95,
		Falloff:                 0.050,
		ScatteringStrength:      1.25,
		UnderwaterBlendFactor:   1.90,
		SurfaceIntensity:        0.30,
		UnderwaterOnly:          true,
		NoiseDistortionStrength: 1.2,
		NoiseDistortionSpeed:    0.45,
		SampleCount:             52,
		ResolutionScale:         0.60,
		BlurRadius:              1.25,
	}
}

// UnderwaterLightShaftRenderer рендерит volumetric лучи в пониженном разрешении
// и затем размывает результат в два прохода (горизонтальный + вертикальный).
type UnderwaterLightShaftRenderer struct {
	shaftProgram uint32
	blurProgram  uint32
	quad         *FullscreenQuad

	depthUniform            int32
	noiseUniform            int32
	screenSizeUniform       int32
	nearUniform             int32
	farUniform              int32
	timeUniform             int32
	waterLevelUniform       int32
	underwaterUniform       int32
	cameraPosUniform        int32
	lightDirUniform         int32
	sunScreenUniform        int32
	sunVisibilityUniform    int32
	intensityUniform        int32
	densityUniform          int32
	falloffUniform          int32
	scatteringUniform       int32
	underwaterBoostUniform  int32
	surfaceIntensityUniform int32
	underwaterOnlyUniform   int32
	noiseStrengthUniform    int32
	noiseSpeedUniform       int32
	sampleCountUniform      int32
	invProjectionUniform    int32
	invViewUniform          int32
	fogColorUniform         int32
	depthTintUniform        int32
	sunColorUniform         int32
	visibilityUniform       int32
	blurInputUniform        int32
	blurTexelSizeUniform    int32
	blurDirectionUniform    int32
	blurRadiusUniform       int32
	noiseTex                uint32
	bufferA                 *PostColorBuffer
	bufferB                 *PostColorBuffer
	fullWidth               int32
	fullHeight              int32
	params                  UnderwaterLightShaftParams
	environment             UnderwaterShaftEnvironment
	customLightDirection    mgl32.Vec3
	useCustomLightDirection bool
}

// NewUnderwaterLightShaftRenderer инициализирует программы лучей/блюра и рабочие буферы.
func NewUnderwaterLightShaftRenderer(width, height int32) *UnderwaterLightShaftRenderer {
	shaftProgram := newFullscreenProgram(
		"assets/shaders/water/underwater_post.vert",
		"assets/shaders/water/underwater_light_shafts.frag",
		"underwater shafts",
	)
	blurProgram := newFullscreenProgram(
		"assets/shaders/water/underwater_post.vert",
		"assets/shaders/water/underwater_blur.frag",
		"underwater blur",
	)

	params := clampLightShaftParams(DefaultUnderwaterLightShaftParams())
	targetW, targetH := scaledPostSize(width, height, params.ResolutionScale)

	renderer := &UnderwaterLightShaftRenderer{
		shaftProgram: shaftProgram,
		blurProgram:  blurProgram,
		quad:         NewFullscreenQuad(),

		depthUniform:            gl.GetUniformLocation(shaftProgram, gl.Str("uDepthTex\x00")),
		noiseUniform:            gl.GetUniformLocation(shaftProgram, gl.Str("uNoiseTex\x00")),
		screenSizeUniform:       gl.GetUniformLocation(shaftProgram, gl.Str("uScreenSize\x00")),
		nearUniform:             gl.GetUniformLocation(shaftProgram, gl.Str("uNear\x00")),
		farUniform:              gl.GetUniformLocation(shaftProgram, gl.Str("uFar\x00")),
		timeUniform:             gl.GetUniformLocation(shaftProgram, gl.Str("uTime\x00")),
		waterLevelUniform:       gl.GetUniformLocation(shaftProgram, gl.Str("uWaterLevel\x00")),
		underwaterUniform:       gl.GetUniformLocation(shaftProgram, gl.Str("uUnderwaterBlend\x00")),
		cameraPosUniform:        gl.GetUniformLocation(shaftProgram, gl.Str("uCameraPos\x00")),
		lightDirUniform:         gl.GetUniformLocation(shaftProgram, gl.Str("uLightDir\x00")),
		sunScreenUniform:        gl.GetUniformLocation(shaftProgram, gl.Str("uSunScreenPos\x00")),
		sunVisibilityUniform:    gl.GetUniformLocation(shaftProgram, gl.Str("uSunVisibility\x00")),
		intensityUniform:        gl.GetUniformLocation(shaftProgram, gl.Str("uIntensity\x00")),
		densityUniform:          gl.GetUniformLocation(shaftProgram, gl.Str("uDensity\x00")),
		falloffUniform:          gl.GetUniformLocation(shaftProgram, gl.Str("uFalloff\x00")),
		scatteringUniform:       gl.GetUniformLocation(shaftProgram, gl.Str("uScatteringStrength\x00")),
		underwaterBoostUniform:  gl.GetUniformLocation(shaftProgram, gl.Str("uUnderwaterBoost\x00")),
		surfaceIntensityUniform: gl.GetUniformLocation(shaftProgram, gl.Str("uSurfaceIntensity\x00")),
		underwaterOnlyUniform:   gl.GetUniformLocation(shaftProgram, gl.Str("uUnderwaterOnly\x00")),
		noiseStrengthUniform:    gl.GetUniformLocation(shaftProgram, gl.Str("uNoiseStrength\x00")),
		noiseSpeedUniform:       gl.GetUniformLocation(shaftProgram, gl.Str("uNoiseSpeed\x00")),
		sampleCountUniform:      gl.GetUniformLocation(shaftProgram, gl.Str("uSampleCount\x00")),
		invProjectionUniform:    gl.GetUniformLocation(shaftProgram, gl.Str("uInvProjection\x00")),
		invViewUniform:          gl.GetUniformLocation(shaftProgram, gl.Str("uInvView\x00")),
		fogColorUniform:         gl.GetUniformLocation(shaftProgram, gl.Str("uFogColor\x00")),
		depthTintUniform:        gl.GetUniformLocation(shaftProgram, gl.Str("uDepthTint\x00")),
		sunColorUniform:         gl.GetUniformLocation(shaftProgram, gl.Str("uSunColor\x00")),
		visibilityUniform:       gl.GetUniformLocation(shaftProgram, gl.Str("uVisibilityDistance\x00")),

		blurInputUniform:     gl.GetUniformLocation(blurProgram, gl.Str("uInputTex\x00")),
		blurTexelSizeUniform: gl.GetUniformLocation(blurProgram, gl.Str("uTexelSize\x00")),
		blurDirectionUniform: gl.GetUniformLocation(blurProgram, gl.Str("uDirection\x00")),
		blurRadiusUniform:    gl.GetUniformLocation(blurProgram, gl.Str("uRadius\x00")),

		noiseTex:    generateCausticTexture(384, 8.3),
		bufferA:     NewPostColorBuffer(targetW, targetH),
		bufferB:     NewPostColorBuffer(targetW, targetH),
		fullWidth:   width,
		fullHeight:  height,
		params:      params,
		environment: sanitizeUnderwaterShaftEnvironment(DefaultUnderwaterShaftEnvironment()),
	}

	gl.UseProgram(renderer.shaftProgram)
	gl.Uniform1i(renderer.depthUniform, 0)
	gl.Uniform1i(renderer.noiseUniform, 1)
	gl.UseProgram(renderer.blurProgram)
	gl.Uniform1i(renderer.blurInputUniform, 0)

	return renderer
}

func (r *UnderwaterLightShaftRenderer) SetParams(params UnderwaterLightShaftParams) {
	r.params = clampLightShaftParams(params)
	if r.fullWidth > 0 && r.fullHeight > 0 {
		r.Resize(r.fullWidth, r.fullHeight)
	}
}

func (r *UnderwaterLightShaftRenderer) Params() UnderwaterLightShaftParams {
	return r.params
}

func (r *UnderwaterLightShaftRenderer) SetEnvironment(environment UnderwaterShaftEnvironment) {
	r.environment = sanitizeUnderwaterShaftEnvironment(environment)
}

func (r *UnderwaterLightShaftRenderer) Environment() UnderwaterShaftEnvironment {
	return r.environment
}

func (r *UnderwaterLightShaftRenderer) SetLightDirectionOverride(direction mgl32.Vec3) {
	if direction.Len() < 0.0001 {
		return
	}
	r.customLightDirection = direction.Normalize()
	r.useCustomLightDirection = true
}

func (r *UnderwaterLightShaftRenderer) ClearLightDirectionOverride() {
	r.useCustomLightDirection = false
}

func (r *UnderwaterLightShaftRenderer) effectiveLightDirection(defaultDirection mgl32.Vec3) mgl32.Vec3 {
	if r.useCustomLightDirection {
		return r.customLightDirection
	}
	if defaultDirection.Len() < 0.0001 {
		return mgl32.Vec3{0.25, -1.0, 0.32}
	}
	return defaultDirection.Normalize()
}

func (r *UnderwaterLightShaftRenderer) Resize(width, height int32) {
	r.fullWidth = width
	r.fullHeight = height
	targetW, targetH := scaledPostSize(width, height, r.params.ResolutionScale)
	r.bufferA.Resize(targetW, targetH)
	r.bufferB.Resize(targetW, targetH)
}

// Render вычисляет текстуру световых лучей.
// Если камера не под водой, функция возвращает пустую (черную) текстуру буфера A.
func (r *UnderwaterLightShaftRenderer) Render(
	depthTex uint32,
	screenWidth, screenHeight int32,
	nearPlane, farPlane, timeSec, waterLevel, underwaterBlend float32,
	cameraPos, lightDir mgl32.Vec3,
	viewProj, invProjection, invView mgl32.Mat4,
) uint32 {
	if screenWidth != r.fullWidth || screenHeight != r.fullHeight {
		r.Resize(screenWidth, screenHeight)
	}

	// Если отключен strictly-underwater режим, разрешаем слабые лучи и над водой.
	allowSurfaceShafts := !r.params.UnderwaterOnly && r.params.SurfaceIntensity > 0.0001
	if underwaterBlend <= 0.001 && !allowSurfaceShafts {
		r.bufferA.Bind()
		gl.ClearColor(0.0, 0.0, 0.0, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		r.bufferA.Unbind(screenWidth, screenHeight)
		return r.bufferA.ColorTex
	}

	effectiveLightDir := r.effectiveLightDirection(lightDir)
	sunUV, sunVisibility := projectDirectionalLightToScreen(viewProj, cameraPos, effectiveLightDir)
	environment := sanitizeUnderwaterShaftEnvironment(r.environment)

	r.bufferA.Bind()
	gl.Disable(gl.DEPTH_TEST)
	gl.DepthMask(false)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT)

	gl.UseProgram(r.shaftProgram)
	gl.Uniform2f(r.screenSizeUniform, float32(screenWidth), float32(screenHeight))
	gl.Uniform1f(r.nearUniform, nearPlane)
	gl.Uniform1f(r.farUniform, farPlane)
	gl.Uniform1f(r.timeUniform, timeSec)
	gl.Uniform1f(r.waterLevelUniform, waterLevel)
	gl.Uniform1f(r.underwaterUniform, underwaterBlend)
	gl.Uniform3f(r.cameraPosUniform, cameraPos.X(), cameraPos.Y(), cameraPos.Z())
	gl.Uniform3f(r.lightDirUniform, effectiveLightDir.X(), effectiveLightDir.Y(), effectiveLightDir.Z())
	gl.Uniform2f(r.sunScreenUniform, sunUV.X(), sunUV.Y())
	gl.Uniform1f(r.sunVisibilityUniform, sunVisibility)
	gl.Uniform1f(r.intensityUniform, r.params.Intensity)
	gl.Uniform1f(r.densityUniform, r.params.Density)
	gl.Uniform1f(r.falloffUniform, r.params.Falloff)
	gl.Uniform1f(r.scatteringUniform, r.params.ScatteringStrength)
	gl.Uniform1f(r.underwaterBoostUniform, r.params.UnderwaterBlendFactor)
	gl.Uniform1f(r.surfaceIntensityUniform, r.params.SurfaceIntensity)
	if r.params.UnderwaterOnly {
		gl.Uniform1f(r.underwaterOnlyUniform, 1.0)
	} else {
		gl.Uniform1f(r.underwaterOnlyUniform, 0.0)
	}
	gl.Uniform1f(r.noiseStrengthUniform, r.params.NoiseDistortionStrength)
	gl.Uniform1f(r.noiseSpeedUniform, r.params.NoiseDistortionSpeed)
	gl.Uniform1i(r.sampleCountUniform, r.params.SampleCount)
	gl.Uniform3f(r.fogColorUniform, environment.FogColor.X(), environment.FogColor.Y(), environment.FogColor.Z())
	gl.Uniform3f(r.depthTintUniform, environment.DepthTint.X(), environment.DepthTint.Y(), environment.DepthTint.Z())
	gl.Uniform3f(r.sunColorUniform, environment.SunColor.X(), environment.SunColor.Y(), environment.SunColor.Z())
	gl.Uniform1f(r.visibilityUniform, environment.VisibilityDistance)
	gl.UniformMatrix4fv(r.invProjectionUniform, 1, false, &invProjection[0])
	gl.UniformMatrix4fv(r.invViewUniform, 1, false, &invView[0])

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, depthTex)
	gl.ActiveTexture(gl.TEXTURE0 + 1)
	gl.BindTexture(gl.TEXTURE_2D, r.noiseTex)

	r.quad.Draw()

	texelSizeX := 1.0 / float32(r.bufferA.Width)
	texelSizeY := 1.0 / float32(r.bufferA.Height)

	r.bufferB.Bind()
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.UseProgram(r.blurProgram)
	gl.Uniform2f(r.blurTexelSizeUniform, texelSizeX, texelSizeY)
	gl.Uniform2f(r.blurDirectionUniform, 1.0, 0.0)
	gl.Uniform1f(r.blurRadiusUniform, r.params.BlurRadius)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, r.bufferA.ColorTex)
	r.quad.Draw()

	r.bufferA.Bind()
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.Uniform2f(r.blurDirectionUniform, 0.0, 1.0)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, r.bufferB.ColorTex)
	r.quad.Draw()

	r.bufferA.Unbind(screenWidth, screenHeight)
	gl.DepthMask(true)
	gl.Enable(gl.DEPTH_TEST)

	return r.bufferA.ColorTex
}

// clampLightShaftParams защищает систему от невалидных значений из консоли/конфига.
func clampLightShaftParams(params UnderwaterLightShaftParams) UnderwaterLightShaftParams {
	if params.Intensity < 0 {
		params.Intensity = 0
	}
	if params.Density < 0.01 {
		params.Density = 0.01
	}
	if params.Density > 5.0 {
		params.Density = 5.0
	}
	if params.ScatteringStrength < 0 {
		params.ScatteringStrength = 0
	}
	if params.Falloff < 0.001 {
		params.Falloff = 0.001
	}
	if params.Falloff > 0.3 {
		params.Falloff = 0.3
	}
	if params.UnderwaterBlendFactor < 0 {
		params.UnderwaterBlendFactor = 0
	}
	if params.UnderwaterBlendFactor > 4.0 {
		params.UnderwaterBlendFactor = 4.0
	}
	if params.SurfaceIntensity < 0 {
		params.SurfaceIntensity = 0
	}
	if params.SurfaceIntensity > 1.0 {
		params.SurfaceIntensity = 1.0
	}
	if params.NoiseDistortionStrength < 0 {
		params.NoiseDistortionStrength = 0
	}
	if params.NoiseDistortionSpeed < 0 {
		params.NoiseDistortionSpeed = 0
	}
	if params.SampleCount < 8 {
		params.SampleCount = 8
	}
	if params.SampleCount > 96 {
		params.SampleCount = 96
	}
	if params.ResolutionScale < 0.2 {
		params.ResolutionScale = 0.2
	}
	if params.ResolutionScale > 1.0 {
		params.ResolutionScale = 1.0
	}
	if params.BlurRadius < 0.25 {
		params.BlurRadius = 0.25
	}
	if params.BlurRadius > 3.0 {
		params.BlurRadius = 3.0
	}
	return params
}

func scaledPostSize(width, height int32, scale float32) (int32, int32) {
	targetW := int32(float32(width) * scale)
	targetH := int32(float32(height) * scale)
	if targetW < 1 {
		targetW = 1
	}
	if targetH < 1 {
		targetH = 1
	}
	return targetW, targetH
}

func projectDirectionalLightToScreen(viewProj mgl32.Mat4, cameraPos, lightDir mgl32.Vec3) (mgl32.Vec2, float32) {
	dir := lightDir
	if dir.Len() < 0.0001 {
		dir = mgl32.Vec3{0.25, -1.0, 0.32}
	}
	dir = dir.Normalize()
	sunWorld := cameraPos.Sub(dir.Mul(450.0))
	clip := viewProj.Mul4x1(mgl32.Vec4{sunWorld.X(), sunWorld.Y(), sunWorld.Z(), 1.0})

	w := clip.W()
	if w <= 0.0001 {
		return mgl32.Vec2{0.5, 1.2}, 0.0
	}

	invW := 1.0 / w
	ndcX := clip.X() * invW
	ndcY := clip.Y() * invW
	uv := mgl32.Vec2{ndcX*0.5 + 0.5, ndcY*0.5 + 0.5}

	outsideX := float32(math.Max(0.0, math.Abs(float64(ndcX))-1.0))
	outsideY := float32(math.Max(0.0, math.Abs(float64(ndcY))-1.0))
	outside := outsideX + outsideY
	visibility := clamp01(1.0 - outside*1.35)

	elevation := clamp01((-dir.Y() - 0.02) / 0.95)
	visibility *= elevation

	return uv, visibility
}

// newFullscreenProgram — общий helper для компиляции fullscreen pass-программ.
func newFullscreenProgram(vertexPath, fragmentPath, label string) uint32 {
	vertexSource := loadShaderSourceFile(vertexPath)
	fragmentSource := loadShaderSourceFile(fragmentPath)
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
		panic(fmt.Sprintf("%s shader link failed: %s", label, string(logText)))
	}
	return program
}
