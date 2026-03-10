package engine

import "github.com/go-gl/mathgl/mgl32"

// FogSettings описывает параметры тумана в lit-пайплайне сцены.
type FogSettings struct {
	Color     mgl32.Vec3
	RangeNear float32
	RangeFar  float32
	Strength  float32
	Amount    float32
}

func DefaultFogSettings() FogSettings {
	return FogSettings{
		Color:     mgl32.Vec3{0.05, 0.23, 0.34},
		RangeNear: 0.0,
		RangeFar:  90.0,
		Strength:  1.0,
		Amount:    1.0,
	}
}

func SanitizeFogSettings(settings FogSettings) FogSettings {
	if settings.Color.Len() < 0.0001 {
		settings.Color = mgl32.Vec3{0.05, 0.23, 0.34}
	}
	if settings.RangeNear < 0 {
		settings.RangeNear = 0
	}
	if settings.RangeFar <= settings.RangeNear+0.01 {
		settings.RangeFar = settings.RangeNear + 0.01
	}
	if settings.Strength < 0 {
		settings.Strength = 0
	}
	if settings.Amount < 0 {
		settings.Amount = 0
	}
	if settings.Amount > 1 {
		settings.Amount = 1
	}
	return settings
}

// UnderwaterAtmosphere хранит единый набор подводных параметров для lit/ocean/shaft-проходов.
// Русский комментарий: структуру держим в engine-слое, чтобы scene-код задавал только художественные значения,
// а не раскладывал вручную одни и те же параметры по нескольким рендер-подсистемам.
type UnderwaterAtmosphere struct {
	Blend               float32
	FogDensity          float32
	FogColor            mgl32.Vec3
	DepthTint           mgl32.Vec3
	SunlightAttenuation float32
	VisibilityDistance  float32
	DepthScale          float32
}

func DefaultUnderwaterAtmosphere() UnderwaterAtmosphere {
	return UnderwaterAtmosphere{
		Blend:               0.0,
		FogDensity:          1.0,
		FogColor:            mgl32.Vec3{0.05, 0.24, 0.34},
		DepthTint:           mgl32.Vec3{0.08, 0.34, 0.48},
		SunlightAttenuation: 0.12,
		VisibilityDistance:  90.0,
		DepthScale:          1.0,
	}
}

func SanitizeUnderwaterAtmosphere(settings UnderwaterAtmosphere) UnderwaterAtmosphere {
	if settings.Blend < 0 {
		settings.Blend = 0
	}
	if settings.Blend > 1 {
		settings.Blend = 1
	}
	if settings.FogDensity < 0 {
		settings.FogDensity = 0
	}
	if settings.FogColor.Len() < 0.0001 {
		settings.FogColor = mgl32.Vec3{0.05, 0.24, 0.34}
	}
	if settings.DepthTint.Len() < 0.0001 {
		settings.DepthTint = mgl32.Vec3{0.08, 0.34, 0.48}
	}
	if settings.SunlightAttenuation < 0 {
		settings.SunlightAttenuation = 0
	}
	if settings.VisibilityDistance < 4 {
		settings.VisibilityDistance = 4
	}
	if settings.DepthScale < 0 {
		settings.DepthScale = 0
	}
	if settings.DepthScale > 1 {
		settings.DepthScale = 1
	}
	return settings
}

// AtmosphereState объединяет fog + underwater + каустику в одном кадре.
type AtmosphereState struct {
	Fog        FogSettings
	Underwater UnderwaterAtmosphere
	Caustics   UnderwaterCausticsParams
	WaterLevel float32
	TimeSec    float32
}

func DefaultAtmosphereState() AtmosphereState {
	return AtmosphereState{
		Fog:        DefaultFogSettings(),
		Underwater: DefaultUnderwaterAtmosphere(),
		Caustics:   DefaultUnderwaterCausticsParams(),
		WaterLevel: 0.0,
		TimeSec:    0.0,
	}
}

func sanitizeAtmosphereState(state AtmosphereState) AtmosphereState {
	state.Fog = SanitizeFogSettings(state.Fog)
	state.Underwater = SanitizeUnderwaterAtmosphere(state.Underwater)
	state.Caustics = ClampUnderwaterCausticsParams(state.Caustics)
	return state
}

// ShadowState описывает входы shadow-подсистемы для кадра lit-рендера.
type ShadowState struct {
	Map              *ShadowMap
	LightSpaceMatrix mgl32.Mat4
	TextureUnit      int32
}

func DefaultShadowState() ShadowState {
	return ShadowState{
		Map:              nil,
		LightSpaceMatrix: mgl32.Ident4(),
		TextureUnit:      1,
	}
}

func sanitizeShadowState(state ShadowState) ShadowState {
	if state.LightSpaceMatrix == (mgl32.Mat4{}) {
		state.LightSpaceMatrix = mgl32.Ident4()
	}
	if state.TextureUnit < 0 {
		state.TextureUnit = 0
	}
	return state
}

// LightingFrame — канонический набор lighting-данных на кадр.
type LightingFrame struct {
	Lighting   LightingState
	Atmosphere AtmosphereState
	Shadow     ShadowState
}

func DefaultLightingFrame() LightingFrame {
	return LightingFrame{
		Lighting:   DefaultLightingState(),
		Atmosphere: DefaultAtmosphereState(),
		Shadow:     DefaultShadowState(),
	}
}

func sanitizeLightingFrame(frame LightingFrame) LightingFrame {
	if len(frame.Lighting.Lights) == 0 && frame.Lighting.Ambient.Intensity <= 0 {
		frame.Lighting = DefaultLightingState()
	}
	frame.Atmosphere = sanitizeAtmosphereState(frame.Atmosphere)
	frame.Shadow = sanitizeShadowState(frame.Shadow)
	return frame
}

// DirectionalLightInfo содержит primary directional light, выбранный для ocean/sky/shafts.
type DirectionalLightInfo struct {
	Direction mgl32.Vec3
	Color     mgl32.Vec3
	Intensity float32
	Found     bool
}

func DefaultDirectionalLightInfo() DirectionalLightInfo {
	sun := DefaultSunLight()
	return DirectionalLightInfo{
		Direction: sun.Direction,
		Color:     sun.Color,
		Intensity: sun.Intensity,
		Found:     false,
	}
}

// ResolvePrimaryDirectionalLight выбирает directional-источник в приоритете:
// 1) explicit shadow light, 2) первый активный directional, 3) fallback sun.
func ResolvePrimaryDirectionalLight(state LightingState) DirectionalLightInfo {
	info := DefaultDirectionalLightInfo()
	lightCount := len(state.Lights)
	if lightCount == 0 {
		return info
	}

	if state.ShadowLightIndex >= 0 && state.ShadowLightIndex < lightCount {
		candidate := sanitizeLight(state.Lights[state.ShadowLightIndex])
		if candidate.Type == LightDirectional && isLightActive(candidate) {
			info.Direction = candidate.Direction
			info.Color = candidate.Color
			info.Intensity = candidate.Intensity
			info.Found = true
			return info
		}
	}

	for i := 0; i < lightCount; i++ {
		candidate := sanitizeLight(state.Lights[i])
		if candidate.Type != LightDirectional || !isLightActive(candidate) {
			continue
		}
		info.Direction = candidate.Direction
		info.Color = candidate.Color
		info.Intensity = candidate.Intensity
		info.Found = true
		return info
	}

	return info
}

// BuildUnderwaterShaftEnvironment сводит fog/underwater/sun в общую среду для volumetric shafts.
func BuildUnderwaterShaftEnvironment(
	fogColor mgl32.Vec3,
	underwater UnderwaterAtmosphere,
	sunColor mgl32.Vec3,
) UnderwaterShaftEnvironment {
	underwater = SanitizeUnderwaterAtmosphere(underwater)
	if fogColor.Len() < 0.0001 {
		fogColor = underwater.FogColor
	}
	if sunColor.Len() < 0.0001 {
		sunColor = mgl32.Vec3{1.0, 0.95, 0.86}
	}

	return sanitizeUnderwaterShaftEnvironment(UnderwaterShaftEnvironment{
		FogColor:           fogColor,
		DepthTint:          underwater.DepthTint,
		SunColor:           sunColor,
		VisibilityDistance: underwater.VisibilityDistance,
	})
}
