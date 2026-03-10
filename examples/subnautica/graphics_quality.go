package main

import (
	"fmt"
	"strings"

	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
)

// fogQualitySettings управляет плотностью и диапазоном тумана.
type fogQualitySettings struct {
	RangeNear       float32
	RangeFar        float32
	BaseStrength    float32
	UnderwaterBoost float32
	Amount          float32
}

type causticsQualitySettings struct {
	Base            float32
	UnderwaterBoost float32
}

type vegetationQualitySettings struct {
	DrawStride  int
	MaxDistance float32
}

type shadowQualitySettings struct {
	MapSize      int32
	SceneRadius  float32
	FollowCamera bool
	FollowOffset mgl32.Vec3
	Camera       engine.ShadowCameraConfig
	Sampling     engine.ShadowSamplingSettings
}

type subnauticaGraphicsQuality struct {
	Preset       engine.GraphicsQualityPreset
	FarClipScale float32
	MinFarClip   float32
	Fog          fogQualitySettings
	Caustics     causticsQualitySettings
	Vegetation   vegetationQualitySettings
	Shadows      shadowQualitySettings
	Ocean        engine.OceanQualitySettings
	LightShafts  engine.UnderwaterLightShaftParams
}

// subnauticaHighLightShafts — профиль параметров лучей для high-предустановки.
func subnauticaHighLightShafts() engine.UnderwaterLightShaftParams {
	// Русский комментарий: профиль нарочно усилен, чтобы в debug-проверке лучи
	// были заметны сразу и их можно было оценить без "пиксель-хантинга".
	return engine.UnderwaterLightShaftParams{
		Intensity:               2.95,
		Density:                 0.78,
		Falloff:                 0.040,
		ScatteringStrength:      1.48,
		UnderwaterBlendFactor:   2.30,
		SurfaceIntensity:        0.30,
		UnderwaterOnly:          false,
		NoiseDistortionStrength: 1.12,
		NoiseDistortionSpeed:    0.46,
		SampleCount:             64,
		ResolutionScale:         0.74,
		BlurRadius:              1.18,
	}
}

// qualitySettingsForPreset возвращает полный набор связанных настроек для выбранного preset.
// Важно, что параметры подбираются пакетом (туман, тени, океан, LOD), а не по отдельности.
func qualitySettingsForPreset(preset engine.GraphicsQualityPreset) subnauticaGraphicsQuality {
	highShafts := subnauticaHighLightShafts()
	high := subnauticaGraphicsQuality{
		Preset:       engine.GraphicsQualityHigh,
		FarClipScale: 1.0,
		MinFarClip:   1200,
		Fog: fogQualitySettings{
			RangeNear:       0.82,
			RangeFar:        1.0,
			BaseStrength:    0.08,
			UnderwaterBoost: 0.82,
			Amount:          1.0,
		},
		Caustics: causticsQualitySettings{
			Base:            0.15,
			UnderwaterBoost: 0.25,
		},
		Vegetation: vegetationQualitySettings{
			DrawStride:  1,
			MaxDistance: 0,
		},
		Shadows: shadowQualitySettings{
			MapSize:      engine.ShadowMapSize,
			SceneRadius:  82,
			FollowCamera: true,
			FollowOffset: mgl32.Vec3{0, -6, 0},
			Camera: engine.ShadowCameraConfig{
				NearPlane:         0.7,
				FarPlaneScale:     4.2,
				Stabilize:         true,
				DepthBiasSlope:    2.3,
				DepthBiasConstant: 4.0,
			},
			Sampling: engine.ShadowSamplingSettings{
				BiasMin:   0.00075,
				BiasSlope: 0.0065,
				Strength:  1.0,
			},
		},
		Ocean:       engine.DefaultOceanQualitySettings(),
		LightShafts: highShafts,
	}

	switch preset {
	case engine.GraphicsQualityLow:
		return subnauticaGraphicsQuality{
			Preset:       engine.GraphicsQualityLow,
			FarClipScale: 0.68,
			MinFarClip:   520,
			Fog: fogQualitySettings{
				RangeNear:       0.74,
				RangeFar:        0.99,
				BaseStrength:    0.14,
				UnderwaterBoost: 1.05,
				Amount:          1.0,
			},
			Caustics: causticsQualitySettings{
				Base:            0.07,
				UnderwaterBoost: 0.08,
			},
			Vegetation: vegetationQualitySettings{
				DrawStride:  3,
				MaxDistance: 50,
			},
			Shadows: shadowQualitySettings{
				MapSize:      1024,
				SceneRadius:  56,
				FollowCamera: true,
				FollowOffset: mgl32.Vec3{0, -5, 0},
				Camera: engine.ShadowCameraConfig{
					NearPlane:         0.9,
					FarPlaneScale:     3.6,
					Stabilize:         true,
					DepthBiasSlope:    2.8,
					DepthBiasConstant: 5.0,
				},
				Sampling: engine.ShadowSamplingSettings{
					BiasMin:   0.0011,
					BiasSlope: 0.0095,
					Strength:  0.92,
				},
			},
			Ocean: engine.OceanQualitySettings{
				SceneResolutionScale:      0.68,
				ReflectionResolutionScale: 0.72,
				ReflectionPassEnabled:     false,
				LightShaftsEnabled:        false,
				SSRMaxSteps:               8,
				LODCount:                  5,
				LODRadiusScale:            1.0,
				CullRadiusScale:           1.85,
			},
			LightShafts: engine.UnderwaterLightShaftParams{
				Intensity:               1.35,
				Density:                 1.18,
				Falloff:                 0.085,
				ScatteringStrength:      1.00,
				UnderwaterBlendFactor:   1.35,
				SurfaceIntensity:        0.12,
				UnderwaterOnly:          true,
				NoiseDistortionStrength: 0.75,
				NoiseDistortionSpeed:    0.38,
				SampleCount:             24,
				ResolutionScale:         0.42,
				BlurRadius:              1.00,
			},
		}
	case engine.GraphicsQualityMedium:
		return subnauticaGraphicsQuality{
			Preset:       engine.GraphicsQualityMedium,
			FarClipScale: 0.86,
			MinFarClip:   820,
			Fog: fogQualitySettings{
				RangeNear:       0.80,
				RangeFar:        1.0,
				BaseStrength:    0.10,
				UnderwaterBoost: 0.90,
				Amount:          1.0,
			},
			Caustics: causticsQualitySettings{
				Base:            0.12,
				UnderwaterBoost: 0.18,
			},
			Vegetation: vegetationQualitySettings{
				DrawStride:  2,
				MaxDistance: 72,
			},
			Shadows: shadowQualitySettings{
				MapSize:      1536,
				SceneRadius:  70,
				FollowCamera: true,
				FollowOffset: mgl32.Vec3{0, -5.5, 0},
				Camera: engine.ShadowCameraConfig{
					NearPlane:         0.8,
					FarPlaneScale:     3.9,
					Stabilize:         true,
					DepthBiasSlope:    2.5,
					DepthBiasConstant: 4.5,
				},
				Sampling: engine.ShadowSamplingSettings{
					BiasMin:   0.0009,
					BiasSlope: 0.0075,
					Strength:  0.97,
				},
			},
			Ocean: engine.OceanQualitySettings{
				SceneResolutionScale:      0.85,
				ReflectionResolutionScale: 1.0,
				ReflectionPassEnabled:     true,
				LightShaftsEnabled:        true,
				SSRMaxSteps:               20,
				LODCount:                  4,
				LODRadiusScale:            0.84,
				CullRadiusScale:           1.15,
			},
			LightShafts: engine.UnderwaterLightShaftParams{
				Intensity:               2.25,
				Density:                 0.88,
				Falloff:                 0.050,
				ScatteringStrength:      1.34,
				UnderwaterBlendFactor:   2.00,
				SurfaceIntensity:        0.20,
				UnderwaterOnly:          true,
				NoiseDistortionStrength: 1.0,
				NoiseDistortionSpeed:    0.45,
				SampleCount:             48,
				ResolutionScale:         0.62,
				BlurRadius:              1.08,
			},
		}
	default:
		return high
	}
}

func parseQualityToken(token string) (engine.GraphicsQualityPreset, bool) {
	switch strings.TrimSpace(strings.ToLower(token)) {
	case "low":
		return engine.GraphicsQualityLow, true
	case "medium":
		return engine.GraphicsQualityMedium, true
	case "high":
		return engine.GraphicsQualityHigh, true
	default:
		return engine.GraphicsQualityHigh, false
	}
}

// applyGraphicsPreset применяет профиль качества в runtime:
// shadow map размер, ocean quality, shafts и projection far clip.
func (g *subnauticaGame) applyGraphicsPreset(ctx *engine.Context, preset engine.GraphicsQualityPreset, emitLog bool) string {
	settings := qualitySettingsForPreset(preset)
	g.qualitySettings = settings
	g.qualityPreset = settings.Preset

	if g.shadowPass != nil {
		g.shadowPass.SetSceneBounds(g.shadowSceneCenter, settings.Shadows.SceneRadius)
		g.shadowPass.SetFollowCamera(settings.Shadows.FollowCamera)
		g.shadowPass.SetFollowOffset(settings.Shadows.FollowOffset)
		g.shadowPass.SetCameraConfig(settings.Shadows.Camera)
	}

	targetShadowSize := settings.Shadows.MapSize
	if targetShadowSize < 256 {
		targetShadowSize = 256
	}
	if g.shadowMap == nil || g.shadowMap.Width != targetShadowSize || g.shadowMap.Height != targetShadowSize {
		if g.shadowMap != nil {
			g.shadowMap.Delete()
		}
		g.shadowMap = engine.NewShadowMap(targetShadowSize, targetShadowSize)
		if g.shadowPass != nil {
			g.shadowPass.SetShadowMap(g.shadowMap)
		}
	}

	if g.oceanSystem != nil {
		g.oceanSystem.SetQualitySettings(settings.Ocean)
		g.oceanSystem.SetLightShaftParams(settings.LightShafts)
	}

	if ctx != nil && ctx.Renderer != nil {
		ctx.Renderer.Use()
		ctx.Renderer.SetShadowSampling(settings.Shadows.Sampling)
	}

	if ctx != nil {
		g.updateProjectionForQuality(ctx)
	}

	message := fmt.Sprintf("graphics quality: %s", settings.Preset.Label())
	if emitLog && g.console != nil {
		g.console.Log(message)
	}
	return message
}

// updateProjectionForQuality пересобирает projection matrix после смены качества.
// Это позволяет уменьшать дальность видимости на low/medium и экономить fill-rate.
func (g *subnauticaGame) updateProjectionForQuality(ctx *engine.Context) {
	if ctx == nil {
		return
	}

	w, h := ctx.Window.GetSize()
	if h <= 0 {
		h = 1
	}
	aspect := float32(w) / float32(h)
	farPlane := g.baseFar * g.qualitySettings.FarClipScale
	minFar := g.baseNear + 5.0
	if g.qualitySettings.MinFarClip > minFar {
		minFar = g.qualitySettings.MinFarClip
	}
	if farPlane < minFar {
		farPlane = minFar
	}
	g.currentFarClip = farPlane
	ctx.Projection = mgl32.Perspective(
		mgl32.DegToRad(g.baseFOV),
		aspect,
		g.baseNear,
		farPlane,
	)
}
