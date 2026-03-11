package main

import (
	"math"
	"sort"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

// buildGlowPlantInstances строит кэш рендер-инстансов из ECS-компонентов.
// Сортировка по VAO уменьшает число лишних bind-операций.
func (g *subnauticaGame) buildGlowPlantInstances() {
	g.glowPlantInstances = g.glowPlantInstances[:0]
	engine.Each2(g.world, func(entity engine.Entity, tr *objects.Transform, gp *objects.GlowPlant) {
		mesh, ok := g.glowPlantMeshes[gp.Type]
		if !ok {
			return
		}
		entityID := g.ensureEntityMeta(entity, entityKindGlowPlant, playerIDInvalid, false, replicationScopeSpatial)
		g.glowPlantInstances = append(g.glowPlantInstances, glowPlantInstance{
			entityID:       entityID,
			mesh:           mesh,
			model:          tr.ModelMatrix(),
			position:       tr.Position,
			boundingRadius: glowPlantBoundingRadius(gp.Type, tr.Scale),
			stableSeed:     glowPlantStableSeed(entityID, tr.Position, gp.Type),
			glowColor:      gp.GlowColor,
			glowIntensity:  gp.GlowIntensity,
			pulseSpeed:     gp.PulseSpeed,
		})
	})

	sort.Slice(g.glowPlantInstances, func(i, j int) bool {
		return g.glowPlantInstances[i].mesh.VAO < g.glowPlantInstances[j].mesh.VAO
	})
}

func glowPlantBoundingRadius(plantType objects.PlantType, scale float32) float32 {
	base := float32(1.4)
	switch plantType {
	case objects.PlantKelp:
		base = 2.1
	case objects.PlantBush:
		base = 1.55
	case objects.PlantCoral:
		base = 1.7
	case objects.PlantFlower:
		base = 1.35
	}
	return base * scale
}

func glowPlantStableSeed(entityID EntityID, position mgl32.Vec3, plantType objects.PlantType) uint32 {
	// Seed зависит от gameplay EntityID и позиции, а не от GL-ресурсов (VAO),
	// чтобы оставаться стабильным для будущей репликации/снимков.
	qx := uint32(int32(math.Round(float64(position.X() * 10.0))))
	qy := uint32(int32(math.Round(float64(position.Y() * 10.0))))
	qz := uint32(int32(math.Round(float64(position.Z() * 10.0))))
	idLo := uint32(entityID)
	idHi := uint32(entityID >> 32)

	seed := idLo*0x165667b1 ^
		idHi*0x9e3779b9 ^
		qx*0x9e3779b1 ^
		qy*0x85ebca77 ^
		qz*0xc2b2ae3d ^
		uint32(plantType)*0x27d4eb2f
	return mixSeed(seed)
}

func keepByQualityStride(seed uint32, stride int) bool {
	if stride <= 1 {
		return true
	}
	return seed%uint32(stride) == 0
}

// drawMeshWithCachedVAO избегает повторного gl.BindVertexArray для одинаковых mesh подряд.
func drawMeshWithCachedVAO(mesh engine.Mesh, lastVAO *uint32) {
	if mesh.VAO != *lastVAO {
		gl.BindVertexArray(mesh.VAO)
		*lastVAO = mesh.VAO
	}
	gl.DrawArrays(gl.TRIANGLES, 0, mesh.VertexCount)
}

// renderShadows рисует depth-pass для грунта, растений и существ.
func (g *subnauticaGame) renderShadows(shader *engine.ShadowShader, ctx *engine.Context) {
	lastVAO := uint32(0)

	shader.SetWaterWaves(false)
	if g.worldStreaming != nil {
		shader.SetModel(g.groundModel)
		for _, cell := range g.worldStreaming.ActiveCells() {
			if cell == nil || !cell.State.Active || cell.TerrainSuppressed {
				continue
			}
			mesh := cell.TerrainMeshForCurrentLOD()
			if mesh.VertexCount <= 0 {
				continue
			}
			drawMeshWithCachedVAO(mesh, &lastVAO)
		}

		for _, cell := range g.worldStreaming.ActiveCells() {
			if cell == nil || !cell.State.Active || cell.DecorSuppressed {
				continue
			}
			plants := cell.GlowPlantsForCurrentLOD()
			for i := range plants {
				instance := &plants[i]
				shader.SetModel(instance.model)
				drawMeshWithCachedVAO(instance.mesh, &lastVAO)
			}
		}
	}

	g.renderCreatureShadows(shader, &lastVAO)
}

// renderGeometry — callback geometry pass'а render-graph.
// Здесь подготавливаются fog/caustics uniforms и запускается рендер океанской системой.
func (g *subnauticaGame) renderGeometry(ctx *engine.Context, shadowMap *engine.ShadowMap, lightSpace mgl32.Mat4) {
	if g.oceanSystem == nil {
		return
	}

	editorWaterVisible := !g.editor.Enabled || g.editor.WaterVisible
	lightingFrame, lightingState := g.buildEnvironmentLightingFrame(ctx, shadowMap, lightSpace, editorWaterVisible)

	ctx.Renderer.Use()
	ctx.Renderer.ApplyLightingFrame(lightingFrame)
	ctx.Renderer.EnableWaterWaves(false)

	g.oceanSystem.SetSurfaceVisible(editorWaterVisible)
	g.oceanSystem.Render(ctx, lightingFrame, func(view mgl32.Mat4, viewPos mgl32.Vec3) {
		g.renderOpaqueScene(ctx, view, viewPos, lightingState)
	})
}

// renderOpaqueScene рисует непрозрачную геометрию мира для заданной камеры (обычной или reflection).
func (g *subnauticaGame) renderOpaqueScene(
	ctx *engine.Context,
	view mgl32.Mat4,
	viewPos mgl32.Vec3,
	lightingState engine.LightingState,
) {
	viewProj := ctx.Projection.Mul4(view)
	frustum := engine.NewFrustum(viewProj)
	isMainView := approxVec3(viewPos, ctx.Camera.Position, 0.0001)

	var visibleCells []*WorldCell
	if g.worldStreaming != nil {
		visibleCells = g.worldStreaming.VisibleCells(frustum, viewPos, 0, isMainView)
	}

	ctx.Renderer.Use()
	ctx.Renderer.SetLighting(lightingState)
	ctx.Renderer.SetViewPos(viewPos)

	g.renderGround(ctx, viewProj, visibleCells)
	g.renderEditorBrushPreview(ctx, viewProj, viewPos)
	g.renderEditorLightMarkers(ctx, viewProj, viewPos)
	g.renderCreatures(ctx, viewProj, viewPos, frustum)
	g.renderGlowingPlants(ctx, viewProj, viewPos, frustum, visibleCells)

	if !g.editor.Enabled && g.firstPersonHands != nil && isMainView {
		g.firstPersonHands.Render(ctx, view)
	}
}

// computeFogColor подбирает цвет тумана:
// под водой — из depth-модели, над водой — из параметров атмосферы и солнца.
func (g *subnauticaGame) computeFogColor(cameraY, underwaterBlend float32) mgl32.Vec3 {
	sky := g.skyParams
	if g.oceanSystem != nil {
		sky = g.oceanSystem.SkyAtmosphere()
	}

	base := sky.HorizonColor.Mul(0.74).Add(sky.ZenithColor.Mul(0.26))
	sunElevation := clamp01((-sky.SunDirection.Y() - 0.08) / 0.92)
	sunTintStrength := (0.08 + 0.28*sunElevation) * sky.FogSunInfluence * (0.55 + 0.45*sky.AtmosphereBlend)
	surfaceFog := base.Add(sky.SunColor.Mul(sunTintStrength))

	waterLevel := g.authoritative.WaterLevel
	if g.oceanSystem != nil {
		waterLevel = g.oceanSystem.WaterLevel()
	}
	underwater := engine.SanitizeUnderwaterAtmosphere(g.sceneLighting.Underwater)
	depthBelowSurface := waterLevel - cameraY
	if depthBelowSurface < 0 {
		depthBelowSurface = 0
	}
	depthBlendDistance := underwater.VisibilityDistance * 0.35
	if depthBlendDistance < 1 {
		depthBlendDistance = 1
	}
	depthInfluence := clamp01(depthBelowSurface / depthBlendDistance)
	underwaterFog := underwater.FogColor.Mul(1.0 - depthInfluence*0.62).Add(underwater.DepthTint.Mul(depthInfluence * 0.62))

	fog := surfaceFog.Mul(1.0 - underwaterBlend).Add(underwaterFog.Mul(underwaterBlend))
	return clampVec3(fog)
}

func (g *subnauticaGame) renderGround(ctx *engine.Context, viewProj mgl32.Mat4, visibleCells []*WorldCell) {
	lastVAO := uint32(0)
	lowColor := mgl32.Vec3{0.32, 0.38, 0.36}
	highColor := mgl32.Vec3{0.92, 0.83, 0.62}
	baseColor := mgl32.Vec3{0.60, 0.56, 0.42}
	debugColors := g.streamingConfig.DebugLODColors

	groundMVP := viewProj.Mul4(g.groundModel)
	ctx.Renderer.SetMVP(groundMVP)
	ctx.Renderer.SetModel(g.groundModel)
	ctx.Renderer.SetMaterial(engine.Material{
		Ambient:          mgl32.Vec3{0.18, 0.18, 0.15},
		Diffuse:          mgl32.Vec3{0.74, 0.68, 0.52},
		Specular:         mgl32.Vec3{0.26, 0.24, 0.22},
		Shininess:        engine.ShininessFromSmoothness(0.06),
		SpecularStrength: 0.34,
	})

	if !debugColors {
		ctx.Renderer.SetObjectColor(baseColor)
		ctx.Renderer.SetHeightGradient(lowColor, highColor, g.groundMinY, g.groundMaxY, 0.95)
	}

	for i := range visibleCells {
		cell := visibleCells[i]
		if cell == nil || !cell.State.Visible || cell.TerrainSuppressed {
			continue
		}

		mesh := cell.TerrainMeshForCurrentLOD()
		if mesh.VertexCount <= 0 {
			continue
		}

		if debugColors {
			ctx.Renderer.SetObjectColor(terrainLODDebugColor(cell.TerrainLOD))
			ctx.Renderer.SetHeightGradient(lowColor, highColor, g.groundMinY, g.groundMaxY, 0.45)
		}

		drawMeshWithCachedVAO(mesh, &lastVAO)
	}
}

func terrainLODDebugColor(lod cellLODLevel) mgl32.Vec3 {
	switch lod {
	case cellLODMid:
		return mgl32.Vec3{0.95, 0.82, 0.22}
	case cellLODFar:
		return mgl32.Vec3{0.98, 0.35, 0.28}
	default:
		return mgl32.Vec3{0.32, 0.93, 0.44}
	}
}

// renderGlowingPlants применяет frustum/distance culling и рисует glow-растения.
func (g *subnauticaGame) renderGlowingPlants(
	ctx *engine.Context,
	viewProj mgl32.Mat4,
	viewPos mgl32.Vec3,
	frustum engine.Frustum,
	visibleCells []*WorldCell,
) {
	lastVAO := uint32(0)

	veg := g.qualitySettings.Vegetation
	stride := veg.DrawStride
	if stride < 1 {
		stride = 1
	}

	maxDistanceSq := float32(0)
	if veg.MaxDistance > 0 {
		maxDistanceSq = veg.MaxDistance * veg.MaxDistance
	}

	ctx.GlowRenderer.Use()
	ctx.GlowRenderer.SetTime(ctx.Time)
	ctx.GlowRenderer.SetViewPos(viewPos)

	for i := range visibleCells {
		cell := visibleCells[i]
		if cell == nil || !cell.State.Visible || cell.DecorSuppressed {
			continue
		}
		plants := cell.GlowPlantsForCurrentLOD()
		for j := range plants {
			instance := &plants[j]
			if !keepByQualityStride(instance.stableSeed, stride) {
				continue
			}

			if !frustum.ContainsSphere(instance.position, instance.boundingRadius) {
				continue
			}
			if maxDistanceSq > 0 {
				delta := instance.position.Sub(viewPos)
				if delta.Dot(delta) > maxDistanceSq {
					continue
				}
			}

			plantMVP := viewProj.Mul4(instance.model)
			ctx.GlowRenderer.SetMVP(plantMVP)
			ctx.GlowRenderer.SetModel(instance.model)
			ctx.GlowRenderer.SetGlowColor(instance.glowColor)
			ctx.GlowRenderer.SetGlowIntensity(instance.glowIntensity)
			ctx.GlowRenderer.SetPulseSpeed(instance.pulseSpeed)
			drawMeshWithCachedVAO(instance.mesh, &lastVAO)
		}
	}
}

func (g *subnauticaGame) fogRangeForFrame(baseFog fogQualitySettings, underwaterBlend float32) (float32, float32) {
	surfaceNear := g.streamingConfig.FogStart
	surfaceFar := g.streamingConfig.FogEnd
	if surfaceNear < 0 {
		surfaceNear = 0
	}
	if surfaceFar <= surfaceNear+1 {
		surfaceFar = surfaceNear + 1
	}

	underwater := engine.SanitizeUnderwaterAtmosphere(g.sceneLighting.Underwater)
	underwaterFarScale := baseFog.RangeFar
	if underwaterFarScale < 0.25 {
		underwaterFarScale = 0.25
	}
	underwaterFar := underwater.VisibilityDistance * underwaterFarScale
	if underwaterFar < 6 {
		underwaterFar = 6
	}

	underwaterNearScale := baseFog.RangeNear
	if underwaterNearScale < 0 {
		underwaterNearScale = 0
	}
	if underwaterNearScale > 0.95 {
		underwaterNearScale = 0.95
	}
	underwaterNear := underwaterFar * underwaterNearScale
	if underwaterNear > underwaterFar-0.5 {
		underwaterNear = underwaterFar - 0.5
	}
	if underwaterNear < 0 {
		underwaterNear = 0
	}

	blend := smoothStep(0.05, 0.95, underwaterBlend)
	nearDist := surfaceNear + (underwaterNear-surfaceNear)*blend
	farDist := surfaceFar + (underwaterFar-surfaceFar)*blend

	if nearDist < 0 {
		nearDist = 0
	}
	if farDist <= nearDist+0.5 {
		farDist = nearDist + 0.5
	}
	return nearDist, farDist
}

func approxVec3(a, b mgl32.Vec3, eps float32) bool {
	return absf(a.X()-b.X()) <= eps &&
		absf(a.Y()-b.Y()) <= eps &&
		absf(a.Z()-b.Z()) <= eps
}
