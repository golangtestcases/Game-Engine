package main

import (
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
)

func defaultSubnauticaSkyParams() engine.SkyAtmosphereParams {
	params := engine.DefaultSkyAtmosphereParams()
	params.SunDiscSize = 0.038
	params.SunHaloIntensity = 1.45
	params.HorizonColor = mgl32.Vec3{0.70, 0.86, 0.99}
	params.ZenithColor = mgl32.Vec3{0.10, 0.40, 0.76}
	params.AtmosphereBlend = 1.08
	params.FogSunInfluence = 0.5
	return params
}

// initEnvironmentRuntime wires the canonical environment path:
// shadow map -> geometry pass -> OceanSystem (sky/ocean/post/shafts).
func (g *subnauticaGame) initEnvironmentRuntime(ctx *engine.Context) error {
	g.skyParams = defaultSubnauticaSkyParams()
	g.initSceneLighting()

	shadowSize := g.qualitySettings.Shadows.MapSize
	if shadowSize < 256 {
		shadowSize = 256
	}
	g.shadowMap = engine.NewShadowMap(shadowSize, shadowSize)
	g.shadowShader = engine.NewShadowShader()

	g.renderGraph = engine.NewRenderGraph()
	g.renderContext = engine.NewRenderContext(ctx)

	g.shadowPass = engine.NewShadowPass(g.shadowMap, g.shadowShader, g.renderShadows)
	g.shadowPass.SetSceneBounds(g.shadowSceneCenter, g.qualitySettings.Shadows.SceneRadius)
	geometryPass := engine.NewGeometryPass([4]float32{0.0, 0.2, 0.5, 1.0}, g.renderGeometry)
	g.renderGraph.AddPass(g.shadowPass)
	g.renderGraph.AddPass(geometryPass)
	if err := g.renderGraph.Build(); err != nil {
		return err
	}
	g.renderGraph.PrintExecutionOrder()

	w, h := ctx.Window.GetSize()
	g.screenWidth = w
	g.screenHeight = h
	g.oceanSystem = engine.NewOceanSystem(ctx.Renderer, int32(w), int32(h))
	g.oceanSystem.SetWaterLevel(g.authoritative.WaterLevel)
	g.oceanSystem.SetSkyAtmosphere(g.skyParams)
	return nil
}

func (g *subnauticaGame) executeEnvironmentRenderGraph(ctx *engine.Context) error {
	if g.renderGraph == nil || g.renderContext == nil {
		return nil
	}
	g.renderContext.Context = ctx
	g.renderContext.SetResource("lightManager", g.lightManager)
	return g.renderGraph.Execute(g.renderContext)
}

func (g *subnauticaGame) syncEnvironmentRuntimeState() {
	if g.oceanSystem != nil {
		g.oceanSystem.SetWaterLevel(g.authoritative.WaterLevel)
	}
	g.syncSunLighting()
}

func (g *subnauticaGame) buildEnvironmentLightingFrame(
	ctx *engine.Context,
	shadowMap *engine.ShadowMap,
	lightSpace mgl32.Mat4,
	editorWaterVisible bool,
) (engine.LightingFrame, engine.LightingState) {
	depth := ctx.Camera.Position.Y()
	underwaterBlend := g.oceanSystem.UnderwaterBlend(ctx.Camera.Position.Y())
	if !editorWaterVisible {
		underwaterBlend = 0
	}

	underwaterAtmosphere := engine.SanitizeUnderwaterAtmosphere(g.sceneLighting.Underwater)
	fogColor := g.computeFogColor(depth, underwaterBlend)
	lightingState := engine.DefaultLightingState()
	if g.lightManager != nil {
		lightingState = g.lightManager.LightingState()
	}

	fog := g.qualitySettings.Fog
	fogRangeNear, fogRangeFar := g.fogRangeForFrame(fog, underwaterBlend)
	fogDensity := 1 + (g.streamingConfig.FogDensity-1)*underwaterBlend
	fogStrength := (fog.BaseStrength + fog.UnderwaterBoost*underwaterBlend) * fogDensity
	caustics := g.qualitySettings.Caustics
	causticsParams := engine.ClampUnderwaterCausticsParams(g.sceneLighting.Caustics)
	causticsQualityScale := caustics.Base + caustics.UnderwaterBoost*underwaterBlend
	if causticsQualityScale < 0 {
		causticsQualityScale = 0
	}
	causticsParams.Intensity *= causticsQualityScale
	if !editorWaterVisible {
		causticsParams.Intensity = 0
	}

	frame := engine.LightingFrame{
		Lighting: lightingState,
		Atmosphere: engine.AtmosphereState{
			Fog: engine.FogSettings{
				Color:     fogColor,
				RangeNear: fogRangeNear,
				RangeFar:  fogRangeFar,
				Strength:  fogStrength,
				Amount:    fog.Amount,
			},
			Underwater: engine.UnderwaterAtmosphere{
				Blend:               underwaterBlend,
				FogDensity:          underwaterAtmosphere.FogDensity,
				FogColor:            underwaterAtmosphere.FogColor,
				DepthTint:           underwaterAtmosphere.DepthTint,
				SunlightAttenuation: underwaterAtmosphere.SunlightAttenuation,
				VisibilityDistance:  underwaterAtmosphere.VisibilityDistance,
				DepthScale:          1.0,
			},
			Caustics:   causticsParams,
			WaterLevel: g.oceanSystem.WaterLevel(),
			TimeSec:    ctx.Time,
		},
		Shadow: engine.ShadowState{
			Map:              shadowMap,
			LightSpaceMatrix: lightSpace,
			TextureUnit:      1,
		},
	}
	return frame, lightingState
}
