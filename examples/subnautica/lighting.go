package main

import (
	"errors"
	"os"

	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
)

type scenePointLight struct {
	Position  mgl32.Vec3
	Color     mgl32.Vec3
	Intensity float32
	Range     float32
	Enabled   bool
	Preset    localLightPreset
}

// subnauticaSceneLighting хранит scene-level настройки освещения.
// Здесь только художественные параметры, а не детали shader/uniform слоя.
type subnauticaSceneLighting struct {
	Ambient      engine.AmbientLight
	SunDirection mgl32.Vec3
	SunColor     mgl32.Vec3
	SunIntensity float32
	PointLights  []scenePointLight
	Underwater   engine.UnderwaterAtmosphere
	Caustics     engine.UnderwaterCausticsParams
}

func defaultSubnauticaSceneLighting() subnauticaSceneLighting {
	return subnauticaSceneLighting{
		Ambient:      engine.NewAmbientLight(mgl32.Vec3{0.74, 0.84, 0.98}, 0.17),
		SunDirection: mgl32.Vec3{0.34, -0.91, 0.24}.Normalize(),
		SunColor:     mgl32.Vec3{1.0, 0.95, 0.86},
		SunIntensity: 1.85,
		PointLights: []scenePointLight{
			{
				Position:  mgl32.Vec3{9.0, -2.2, 7.0},
				Color:     mgl32.Vec3{0.18, 0.74, 0.98},
				Intensity: 1.45,
				Range:     24.0,
				Enabled:   true,
				Preset:    localLightPresetMedium,
			},
			{
				Position:  mgl32.Vec3{-11.0, -3.1, -5.5},
				Color:     mgl32.Vec3{0.24, 0.93, 0.70},
				Intensity: 1.25,
				Range:     21.0,
				Enabled:   true,
				Preset:    localLightPresetMedium,
			},
		},
		Underwater: engine.SanitizeUnderwaterAtmosphere(engine.UnderwaterAtmosphere{
			FogDensity:          1.18,
			FogColor:            mgl32.Vec3{0.05, 0.24, 0.34},
			DepthTint:           mgl32.Vec3{0.08, 0.34, 0.48},
			SunlightAttenuation: 0.14,
			VisibilityDistance:  96.0,
		}),
		Caustics: engine.ClampUnderwaterCausticsParams(engine.UnderwaterCausticsParams{
			Speed:     0.24,
			Scale:     0.085,
			Intensity: 0.96,
			Contrast:  1.38,
			DepthFade: 0.08,
		}),
	}
}

func (g *subnauticaGame) initSceneLighting() {
	g.sceneLighting = defaultSubnauticaSceneLighting()
	switch persistedLights, err := loadScenePointLights(sceneLightsSavePath); {
	case err == nil:
		g.sceneLighting.PointLights = persistedLights
		g.sceneLightsLoadMessage = "scene lights: loaded " + sceneLightsSavePath
	case errors.Is(err, os.ErrNotExist):
		g.sceneLightsLoadMessage = "scene lights: default setup"
	default:
		g.sceneLightsLoadMessage = "scene lights load failed: " + err.Error()
	}

	g.lightManager = engine.NewLightManager()
	g.applySceneLighting()
}

func (g *subnauticaGame) applySceneLighting() {
	if g.lightManager == nil {
		return
	}

	g.sceneLighting.Underwater = engine.SanitizeUnderwaterAtmosphere(g.sceneLighting.Underwater)
	g.sceneLighting.Caustics = engine.ClampUnderwaterCausticsParams(g.sceneLighting.Caustics)

	g.lightManager.Clear()
	g.lightManager.SetAmbientLight(g.sceneLighting.Ambient)

	// Русский комментарий: солнце добавляем первым, чтобы shadow/ocean/sky опирались
	// на единый directional-источник, а point-light слой оставался независимым.
	g.setSunLighting(
		g.sceneLighting.SunDirection,
		g.sceneLighting.SunColor,
		g.sceneLighting.SunIntensity,
	)

	runtimePointLightCount := 0
	skippedPointLights := 0
	for i := range g.sceneLighting.PointLights {
		cfg := sanitizeScenePointLight(g.sceneLighting.PointLights[i])
		g.sceneLighting.PointLights[i] = cfg
		if !cfg.Enabled || cfg.Intensity <= 0 {
			continue
		}

		light := engine.NewPointLightWithRange(cfg.Position, cfg.Color, cfg.Intensity, cfg.Range)
		light.Enabled = cfg.Enabled
		if g.lightManager.AddLight(light) < 0 {
			skippedPointLights++
			continue
		}
		runtimePointLightCount++
	}

	// Русский комментарий: храним все scene-light в данных мира, даже если runtime-лимит GLSL уже достигнут.
	// Это не ломает редактор и оставляет данные готовыми для будущего расширения лимитов.
	g.editor.RuntimePointLightCount = runtimePointLightCount
	g.editor.SkippedPointLightCount = skippedPointLights
}

func (g *subnauticaGame) setSunLighting(direction, color mgl32.Vec3, intensity float32) {
	if direction.Len() < 0.0001 {
		direction = engine.DefaultSunLight().Direction
	} else {
		direction = direction.Normalize()
	}
	if color.Len() < 0.0001 {
		color = mgl32.Vec3{1.0, 0.97, 0.92}
	}
	if intensity < 0 {
		intensity = 0
	}

	g.sceneLighting.SunDirection = direction
	g.sceneLighting.SunColor = color
	g.sceneLighting.SunIntensity = intensity

	g.skyParams.SunDirection = direction
	g.skyParams.SunColor = color
	g.skyParams.SunIntensity = intensity

	if g.lightManager != nil {
		g.lightManager.SetSun(direction, color, intensity)
	}
}
