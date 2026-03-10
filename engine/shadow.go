package engine

import (
	"math"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// ShadowMapSize — дефолтный размер теневой карты для высокого качества.
const ShadowMapSize = 2048

// ShadowMap инкапсулирует depth-only FBO для directional shadow mapping.
type ShadowMap struct {
	FBO              uint32
	DepthTexture     uint32
	Width            int32
	Height           int32
	LightSpaceMatrix mgl32.Mat4
}

// ShadowCameraConfig задает параметры directional shadow-камеры.
// Конфигурация отделена от sampling, чтобы позже можно было расширить до каскадов.
type ShadowCameraConfig struct {
	NearPlane         float32
	FarPlaneScale     float32
	Stabilize         bool
	DepthBiasSlope    float32
	DepthBiasConstant float32
}

// ShadowSamplingSettings управляет bias/интенсивностью в lit-pass.
type ShadowSamplingSettings struct {
	BiasMin   float32
	BiasSlope float32
	Strength  float32
}

func DefaultShadowCameraConfig() ShadowCameraConfig {
	return ShadowCameraConfig{
		NearPlane:         0.5,
		FarPlaneScale:     4.0,
		Stabilize:         true,
		DepthBiasSlope:    2.2,
		DepthBiasConstant: 4.0,
	}
}

func DefaultShadowSamplingSettings() ShadowSamplingSettings {
	return ShadowSamplingSettings{
		BiasMin:   0.00075,
		BiasSlope: 0.0065,
		Strength:  1.0,
	}
}

func sanitizeShadowCameraConfig(config ShadowCameraConfig) ShadowCameraConfig {
	if config.NearPlane < 0.01 {
		config.NearPlane = 0.01
	}
	if config.FarPlaneScale < 1.5 {
		config.FarPlaneScale = 1.5
	}
	if config.DepthBiasSlope < 0 {
		config.DepthBiasSlope = 0
	}
	if config.DepthBiasConstant < 0 {
		config.DepthBiasConstant = 0
	}
	return config
}

func sanitizeShadowSamplingSettings(settings ShadowSamplingSettings) ShadowSamplingSettings {
	if settings.BiasMin < 0 {
		settings.BiasMin = 0
	}
	if settings.BiasSlope < 0 {
		settings.BiasSlope = 0
	}
	if settings.Strength < 0 {
		settings.Strength = 0
	}
	if settings.Strength > 1 {
		settings.Strength = 1
	}
	return settings
}

// NewShadowMap создает depth texture и framebuffer для рендера теней.
func NewShadowMap(width, height int32) *ShadowMap {
	sm := &ShadowMap{
		Width:  width,
		Height: height,
	}

	gl.GenFramebuffers(1, &sm.FBO)

	gl.GenTextures(1, &sm.DepthTexture)
	gl.BindTexture(gl.TEXTURE_2D, sm.DepthTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT, width, height, 0, gl.DEPTH_COMPONENT, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	borderColor := []float32{1.0, 1.0, 1.0, 1.0}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])

	gl.BindFramebuffer(gl.FRAMEBUFFER, sm.FBO)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, sm.DepthTexture, 0)
	gl.DrawBuffer(gl.NONE)
	gl.ReadBuffer(gl.NONE)

	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		panic("Shadow map framebuffer is not complete")
	}

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	return sm
}

// Bind переводит рендер в shadow framebuffer и очищает depth.
func (sm *ShadowMap) Bind() {
	gl.Viewport(0, 0, sm.Width, sm.Height)
	gl.BindFramebuffer(gl.FRAMEBUFFER, sm.FBO)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
}

// Unbind возвращает рендер в экранный framebuffer.
func (sm *ShadowMap) Unbind(screenWidth, screenHeight int32) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, screenWidth, screenHeight)
}

// BindTexture биндинг depth texture в указанный texture unit.
func (sm *ShadowMap) BindTexture(unit uint32) {
	gl.ActiveTexture(gl.TEXTURE0 + unit)
	gl.BindTexture(gl.TEXTURE_2D, sm.DepthTexture)
}

// CalculateLightSpaceMatrix рассчитывает матрицу света для ортографической directional-тени.
// sceneCenter/sceneRadius определяют охватываемый объём сцены.
func (sm *ShadowMap) CalculateLightSpaceMatrix(lightDir mgl32.Vec3, sceneCenter mgl32.Vec3, sceneRadius float32) {
	sm.CalculateDirectionalLightSpaceMatrix(lightDir, sceneCenter, sceneRadius, DefaultShadowCameraConfig())
}

// CalculateDirectionalLightSpaceMatrix рассчитывает directional light-space matrix
// с опциональной стабилизацией (texel snapping).
func (sm *ShadowMap) CalculateDirectionalLightSpaceMatrix(
	lightDir mgl32.Vec3,
	sceneCenter mgl32.Vec3,
	sceneRadius float32,
	config ShadowCameraConfig,
) {
	config = sanitizeShadowCameraConfig(config)
	if sceneRadius < 1 {
		sceneRadius = 1
	}

	if lightDir.Len() < 0.0001 {
		lightDir = mgl32.Vec3{0.28, -1.0, 0.22}
	}
	lightDir = lightDir.Normalize()

	up := mgl32.Vec3{0, 1, 0}
	if absDot(lightDir, up) > 0.95 {
		up = mgl32.Vec3{0, 0, 1}
	}

	lightDistance := sceneRadius * 2.0
	lightPos := sceneCenter.Sub(lightDir.Mul(lightDistance))
	lightView := mgl32.LookAtV(lightPos, sceneCenter, up)

	left := -sceneRadius
	right := sceneRadius
	bottom := -sceneRadius
	top := sceneRadius

	// Русский комментарий: привязываем центр ortho-объёма к шагу shadow-текселя,
	// чтобы уменьшить дрожание теней при движении камеры/игрока.
	if config.Stabilize && sm.Width > 0 && sm.Height > 0 {
		centerInLight := lightView.Mul4x1(mgl32.Vec4{
			sceneCenter.X(),
			sceneCenter.Y(),
			sceneCenter.Z(),
			1.0,
		})

		texelSizeX := (right - left) / float32(sm.Width)
		texelSizeY := (top - bottom) / float32(sm.Height)

		snappedX := snapToTexel(centerInLight.X(), texelSizeX)
		snappedY := snapToTexel(centerInLight.Y(), texelSizeY)

		left = snappedX - sceneRadius
		right = snappedX + sceneRadius
		bottom = snappedY - sceneRadius
		top = snappedY + sceneRadius
	}

	farPlane := sceneRadius * config.FarPlaneScale
	if farPlane <= config.NearPlane+1 {
		farPlane = config.NearPlane + 1
	}
	lightProj := mgl32.Ortho(left, right, bottom, top, config.NearPlane, farPlane)
	sm.LightSpaceMatrix = lightProj.Mul4(lightView)
}

func (sm *ShadowMap) Delete() {
	gl.DeleteTextures(1, &sm.DepthTexture)
	gl.DeleteFramebuffers(1, &sm.FBO)
}

func absDot(a, b mgl32.Vec3) float32 {
	d := a.Dot(b)
	if d < 0 {
		return -d
	}
	return d
}

func snapToTexel(value, texelSize float32) float32 {
	if texelSize <= 0 {
		return value
	}
	return float32(math.Floor(float64(value/texelSize)+0.5)) * texelSize
}
