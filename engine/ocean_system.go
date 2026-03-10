package engine

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// OceanSystem объединяет все стадии water pipeline:
// offscreen-сцена, planar reflection, рендер океана, подводный пост-эффект и light shafts.
type OceanSystem struct {
	shader           *OceanShader
	skyRenderer      *SkyRenderer
	underwaterPost   *UnderwaterPostProcessor
	lightShafts      *UnderwaterLightShaftRenderer
	sceneBuffer      *SceneBuffer
	reflectionBuffer *SceneBuffer
	compositeBuffer  *SceneBuffer
	lods             []oceanLOD
	activeLODs       []oceanLOD
	lodTemplates     [][]oceanPatchOffset
	instanceBuffer   []oceanPatchInstance
	quality          OceanQualitySettings
	lastBuildStats   oceanInstanceBuildStats

	baseTileSize   float32
	waterLevel     float32
	surfaceVisible bool
	screenWidth    int32
	screenHeight   int32
	edgeFadeNear   float32
	edgeFadeFar    float32

	shallowColor mgl32.Vec3
	deepColor    mgl32.Vec3
	foamColor    mgl32.Vec3
	scatterColor mgl32.Vec3
	skyParams    SkyAtmosphereParams

	absorption         float32
	refractionStrength float32
	reflectionStrength float32
	roughness          float32
	foamIntensity      float32
	waveTimeScale      float32

	shaftEnvironment         UnderwaterShaftEnvironment
	shaftLightDirOverride    mgl32.Vec3
	hasShaftLightDirOverride bool
}

// OceanDebugState содержит runtime-информацию по текущей camera-centered поверхности океана.
// Используется для оверлея и быстрой валидации без профайлера.
type OceanDebugState struct {
	CameraCentered bool
	Origin         mgl32.Vec2
	CoverageRadius float32
	EdgeFadeNear   float32
	EdgeFadeFar    float32
	InstanceCount  int
	ActiveLODs     int
	PerLOD         [maxOceanDebugLODs]int
}

// NewOceanSystem создает буферы, шейдеры и LOD-сетки океана.
// Параметр renderer используется для генерации mesh-данных (VAO/VBO).
func NewOceanSystem(renderer *Renderer, width, height int32) *OceanSystem {
	quality := DefaultOceanQualitySettings()
	sceneW, sceneH := scaledPostSize(width, height, quality.SceneResolutionScale)

	system := &OceanSystem{
		shader:         NewOceanShader(),
		skyRenderer:    NewSkyRenderer(),
		underwaterPost: NewUnderwaterPostProcessor(),
		lightShafts:    NewUnderwaterLightShaftRenderer(sceneW, sceneH),
		sceneBuffer:    NewSceneBuffer(sceneW, sceneH),
		baseTileSize:   36.0,
		waterLevel:     4.8,
		surfaceVisible: true,
		quality:        clampOceanQualitySettings(quality, 5),

		shallowColor: mgl32.Vec3{0.14, 0.49, 0.66},
		deepColor:    mgl32.Vec3{0.01, 0.06, 0.16},
		foamColor:    mgl32.Vec3{0.9, 0.97, 1.0},
		scatterColor: mgl32.Vec3{0.08, 0.26, 0.33},
		skyParams:    DefaultSkyAtmosphereParams(),

		absorption:         0.072,
		refractionStrength: 1.18,
		reflectionStrength: 1.16,
		roughness:          0.055,
		foamIntensity:      1.1,
		waveTimeScale:      0.42,
		screenWidth:        width,
		screenHeight:       height,
		edgeFadeNear:       320.0,
		edgeFadeFar:        420.0,
		shaftEnvironment:   sanitizeUnderwaterShaftEnvironment(DefaultUnderwaterShaftEnvironment()),
	}

	reflW, reflH := system.reflectionSize(sceneW, sceneH)
	system.reflectionBuffer = NewSceneBuffer(reflW, reflH)
	system.compositeBuffer = NewSceneBuffer(sceneW, sceneH)

	system.lods = []oceanLOD{
		// Более легкие сетки: сохраняем LOD-структуру, но резко снижаем стоимость вершинного этапа.
		newOceanLOD(renderer, 56, 1.0, 2),
		newOceanLOD(renderer, 40, 1.8, 3),
		newOceanLOD(renderer, 28, 3.2, 4),
		newOceanLOD(renderer, 20, 5.2, 5),
		newOceanLOD(renderer, 14, 8.0, 6),
	}
	system.rebuildActiveLODs()
	system.instanceBuffer = make([]oceanPatchInstance, 0, estimateOceanInstanceCapacity(system.lodTemplates))
	if system.lightShafts != nil {
		system.lightShafts.SetEnvironment(system.shaftEnvironment)
	}

	return system
}

// newOceanLOD собирает одну LOD-сетку океанского патча.
func newOceanLOD(renderer *Renderer, resolution int, scale float32, radius int) oceanLOD {
	verts, norms := generateOceanPatch(resolution)
	return oceanLOD{
		Mesh:   renderer.NewMeshWithNormals(verts, norms),
		Scale:  scale,
		Radius: radius,
	}
}

func (o *OceanSystem) SetWaterLevel(level float32) {
	o.waterLevel = level
}

func (o *OceanSystem) SetSkyAtmosphere(params SkyAtmosphereParams) {
	o.skyParams = clampSkyAtmosphereParams(params)
}

func (o *OceanSystem) SkyAtmosphere() SkyAtmosphereParams {
	return o.skyParams
}

func (o *OceanSystem) WaterLevel() float32 {
	return o.waterLevel
}

func (o *OceanSystem) SetSurfaceVisible(visible bool) {
	o.surfaceVisible = visible
}

func (o *OceanSystem) SurfaceVisible() bool {
	return o.surfaceVisible
}

func (o *OceanSystem) SetWaveTimeScale(scale float32) {
	if scale < 0.01 {
		scale = 0.01
	}
	o.waveTimeScale = scale
}

// UnderwaterBlend возвращает плавный коэффициент перехода \"над водой -> под водой\".
// Используется сразу в нескольких pass'ах (fog, shafts, color grading).
func (o *OceanSystem) UnderwaterBlend(cameraY float32) float32 {
	return clamp01((o.waterLevel - cameraY + 0.45) / 1.75)
}

func (o *OceanSystem) SetQualitySettings(settings OceanQualitySettings) {
	o.quality = clampOceanQualitySettings(settings, len(o.lods))
	o.rebuildActiveLODs()
	if o.screenWidth > 0 && o.screenHeight > 0 {
		o.Resize(o.screenWidth, o.screenHeight)
	}
}

func (o *OceanSystem) QualitySettings() OceanQualitySettings {
	return o.quality
}

func (o *OceanSystem) Resize(width, height int32) {
	o.screenWidth = width
	o.screenHeight = height

	sceneW, sceneH := scaledPostSize(width, height, o.quality.SceneResolutionScale)
	o.sceneBuffer.Resize(sceneW, sceneH)
	o.compositeBuffer.Resize(sceneW, sceneH)
	reflW, reflH := o.reflectionSize(sceneW, sceneH)
	o.reflectionBuffer.Resize(reflW, reflH)
	if o.lightShafts != nil {
		o.lightShafts.Resize(sceneW, sceneH)
	}
}

func (o *OceanSystem) SetLightShaftParams(params UnderwaterLightShaftParams) {
	if o.lightShafts != nil {
		o.lightShafts.SetParams(params)
	}
}

func (o *OceanSystem) LightShaftParams() UnderwaterLightShaftParams {
	if o.lightShafts != nil {
		return o.lightShafts.Params()
	}
	return DefaultUnderwaterLightShaftParams()
}

// SetLightShaftEnvironment синхронизирует среду лучей с текущим fog/подводным тоном сцены.
func (o *OceanSystem) SetLightShaftEnvironment(environment UnderwaterShaftEnvironment) {
	o.shaftEnvironment = sanitizeUnderwaterShaftEnvironment(environment)
	if o.lightShafts != nil {
		o.lightShafts.SetEnvironment(o.shaftEnvironment)
	}
}

func (o *OceanSystem) LightShaftEnvironment() UnderwaterShaftEnvironment {
	return o.shaftEnvironment
}

// DebugState возвращает актуальные параметры океанской поверхности для debug-overlay.
func (o *OceanSystem) DebugState() OceanDebugState {
	return OceanDebugState{
		CameraCentered: true,
		Origin:         mgl32.Vec2{o.lastBuildStats.CenterX, o.lastBuildStats.CenterZ},
		CoverageRadius: o.lastBuildStats.MaxRadius,
		EdgeFadeNear:   o.edgeFadeNear,
		EdgeFadeFar:    o.edgeFadeFar,
		InstanceCount:  o.lastBuildStats.Total,
		ActiveLODs:     len(o.activeLODs),
		PerLOD:         o.lastBuildStats.PerLOD,
	}
}

func (o *OceanSystem) SetLightShaftDirectionOverride(direction mgl32.Vec3) {
	if direction.Len() < 0.0001 {
		return
	}
	o.shaftLightDirOverride = direction.Normalize()
	o.hasShaftLightDirOverride = true
	if o.lightShafts != nil {
		o.lightShafts.SetLightDirectionOverride(direction)
	}
}

func (o *OceanSystem) ClearLightShaftDirectionOverride() {
	o.hasShaftLightDirOverride = false
	if o.lightShafts != nil {
		o.lightShafts.ClearLightDirectionOverride()
	}
}

func (o *OceanSystem) rebuildActiveLODs() {
	if len(o.lods) == 0 {
		o.activeLODs = o.activeLODs[:0]
		o.lodTemplates = o.lodTemplates[:0]
		return
	}

	o.quality = clampOceanQualitySettings(o.quality, len(o.lods))
	count := o.quality.LODCount
	if cap(o.activeLODs) < count {
		o.activeLODs = make([]oceanLOD, count)
	} else {
		o.activeLODs = o.activeLODs[:count]
	}

	for i := 0; i < count; i++ {
		lod := o.lods[i]
		scaledRadius := int(float32(lod.Radius)*o.quality.LODRadiusScale + 0.5)
		if scaledRadius < 1 {
			scaledRadius = 1
		}
		lod.Radius = scaledRadius
		o.activeLODs[i] = lod
	}

	o.lodTemplates = buildOceanLODTemplates(o.activeLODs)
	estimatedCapacity := estimateOceanInstanceCapacity(o.lodTemplates)
	if cap(o.instanceBuffer) < estimatedCapacity {
		o.instanceBuffer = make([]oceanPatchInstance, 0, estimatedCapacity)
	}
}

// OceanOpaqueRenderFunc — callback для рендера «непрозрачной» сцены
// в основном и отраженном виде (reflection pass).
type OceanOpaqueRenderFunc func(view mgl32.Mat4, viewPos mgl32.Vec3)

// Render выполняет весь ocean frame pipeline:
// 1) base scene -> sceneBuffer
// 2) optional reflection pass -> reflectionBuffer
// 3) compositing с океаном -> compositeBuffer
// 4) optional underwater post to screen
func (o *OceanSystem) Render(ctx *Context, frame LightingFrame, renderOpaque OceanOpaqueRenderFunc) {
	frame = sanitizeLightingFrame(frame)
	atmosphere := frame.Atmosphere
	primaryLight := ResolvePrimaryDirectionalLight(frame.Lighting)
	o.SetWaterLevel(atmosphere.WaterLevel)

	windowW, windowH := ctx.Window.GetSize()
	screenW := int32(windowW)
	screenH := int32(windowH)
	if screenW != o.screenWidth || screenH != o.screenHeight {
		o.Resize(screenW, screenH)
	}

	view := ctx.Camera.ViewMatrix()
	viewProj := ctx.Projection.Mul4(view)
	frustum := NewFrustum(viewProj)
	mainViewPos := ctx.Camera.Position
	nearPlane, farPlane := projectionNearFarFromMatrix(ctx.Projection)
	underwaterBlend := atmosphere.Underwater.Blend
	shaftEnvironment := BuildUnderwaterShaftEnvironment(
		atmosphere.Fog.Color,
		atmosphere.Underwater,
		primaryLight.Color,
	)
	o.shaftEnvironment = shaftEnvironment

	if !o.surfaceVisible {
		// Русский комментарий: editor-режим "без воды" должен оставаться в том же render flow,
		// но без дорогих water/reflection/post этапов, поэтому рисуем sky+opaque напрямую.
		o.setReflectionClipState(ctx, false)
		gl.Viewport(0, 0, screenW, screenH)
		o.renderSky(view, ctx.Projection, primaryLight)
		renderOpaque(view, mainViewPos)
		return
	}

	o.setReflectionClipState(ctx, false)
	o.sceneBuffer.Bind()
	gl.ClearColor(0.08, 0.19, 0.3, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	o.renderSky(view, ctx.Projection, primaryLight)
	renderOpaque(view, mainViewPos)
	o.sceneBuffer.Unbind(screenW, screenH)

	reflectionTex := o.sceneBuffer.ColorTex
	if o.quality.ReflectionPassEnabled {
		reflectionView, reflectionViewPos := reflectedViewForWater(ctx.Camera, o.waterLevel)
		o.reflectionBuffer.Bind()
		gl.ClearColor(0.12, 0.22, 0.34, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		o.renderSky(reflectionView, ctx.Projection, primaryLight)
		o.setReflectionClipState(ctx, true)
		gl.Enable(gl.CLIP_DISTANCE0)
		renderOpaque(reflectionView, reflectionViewPos)
		gl.Disable(gl.CLIP_DISTANCE0)
		o.setReflectionClipState(ctx, false)
		o.reflectionBuffer.Unbind(screenW, screenH)
		reflectionTex = o.reflectionBuffer.ColorTex
	}

	o.sceneBuffer.BlitToBuffer(o.compositeBuffer, true)
	gl.Viewport(0, 0, o.compositeBuffer.Width, o.compositeBuffer.Height)

	o.instanceBuffer, o.lastBuildStats = buildOceanInstances(
		o.instanceBuffer,
		ctx.Camera.Position,
		o.waterLevel,
		o.baseTileSize,
		o.activeLODs,
		o.lodTemplates,
		frustum,
		o.quality.CullRadiusScale,
	)
	o.updateEdgeFadeRange(o.lastBuildStats.MaxRadius)
	invProj := ctx.Projection.Inv()
	o.shader.Use()
	o.shader.SetFrameUniforms(
		viewProj,
		view,
		ctx.Projection,
		invProj,
		ctx.Camera.Position,
		float32(o.compositeBuffer.Width),
		float32(o.compositeBuffer.Height),
		nearPlane,
		farPlane,
		atmosphere.TimeSec,
		o.waterLevel,
		underwaterBlend,
	)
	o.shader.SetSSRMaxSteps(o.quality.SSRMaxSteps)
	o.shader.SetUsePlanarReflection(o.quality.ReflectionPassEnabled)
	o.shader.SetCoverageFade(o.edgeFadeNear, o.edgeFadeFar)
	o.shader.SetColors(o.shallowColor, o.deepColor, o.foamColor, o.scatterColor)
	o.shader.SetOptics(o.absorption, o.refractionStrength, o.reflectionStrength, o.roughness, o.foamIntensity)
	o.shader.SetWaveTimeScale(o.waveTimeScale)
	skySunIntensity := o.skyParams.SunIntensity
	if skySunIntensity < primaryLight.Intensity {
		skySunIntensity = primaryLight.Intensity
	}
	o.shader.SetSkyLighting(o.skyParams.HorizonColor, o.skyParams.ZenithColor, skySunIntensity)
	o.shader.SetLighting(primaryLight.Direction, primaryLight.Color)
	o.shader.BindInputTextures(o.sceneBuffer.ColorTex, o.sceneBuffer.DepthTex, reflectionTex)

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.DepthMask(false)

	lastVAO := uint32(0)
	for _, instance := range o.instanceBuffer {
		mesh := o.activeLODs[instance.LOD].Mesh
		if mesh.VAO != lastVAO {
			gl.BindVertexArray(mesh.VAO)
			lastVAO = mesh.VAO
		}

		model := oceanModelMatrix(instance.Position.X(), o.waterLevel, instance.Position.Y(), instance.Scale)
		o.shader.SetModel(model)
		gl.DrawArrays(gl.TRIANGLES, 0, mesh.VertexCount)
	}

	gl.DepthMask(true)
	gl.Disable(gl.BLEND)

	o.compositeBuffer.Unbind(screenW, screenH)

	// Под водой применяем финальный полноэкранный пост-процесс.
	invView := view.Inv()
	lightDir := primaryLight.Direction
	if o.hasShaftLightDirOverride {
		lightDir = o.shaftLightDirOverride
	}

	shaftEnabled := o.quality.LightShaftsEnabled && o.lightShafts != nil
	shaftParams := DefaultUnderwaterLightShaftParams()
	if shaftEnabled {
		shaftParams = o.lightShafts.Params()
		o.lightShafts.SetEnvironment(shaftEnvironment)
	}

	// Один и тот же post-pass обслуживает и подводный режим, и опциональные surface-shafts.
	allowSurfaceShafts := shaftEnabled && !shaftParams.UnderwaterOnly && shaftParams.SurfaceIntensity > 0.0001
	usePostPass := underwaterBlend > 0.001 || allowSurfaceShafts
	if usePostPass {
		shaftTex := uint32(0)
		if shaftEnabled {
			shaftTex = o.lightShafts.Render(
				o.compositeBuffer.DepthTex,
				o.compositeBuffer.Width,
				o.compositeBuffer.Height,
				nearPlane,
				farPlane,
				atmosphere.TimeSec,
				o.waterLevel,
				underwaterBlend,
				ctx.Camera.Position,
				lightDir,
				viewProj,
				invProj,
				invView,
			)
		}
		o.underwaterPost.Render(
			o.compositeBuffer.ColorTex,
			o.compositeBuffer.DepthTex,
			shaftTex,
			shaftEnabled,
			shaftParams,
			shaftEnvironment,
			screenW,
			screenH,
			nearPlane,
			farPlane,
			atmosphere.TimeSec,
			o.waterLevel,
			underwaterBlend,
			ctx.Camera.Position,
			lightDir,
			invProj,
			invView,
		)
		return
	}

	o.compositeBuffer.BlitToScreen(screenW, screenH, false)
}

// renderSky рисует атмосферу и синхронизирует параметры солнца с primary directional light.
func (o *OceanSystem) renderSky(view, projection mgl32.Mat4, primaryLight DirectionalLightInfo) {
	if o.skyRenderer == nil {
		return
	}

	params := o.skyParams
	if primaryLight.Direction.Len() > 0.0001 {
		params.SunDirection = primaryLight.Direction.Normalize()
	}
	params.SunColor = primaryLight.Color
	if params.SunIntensity < primaryLight.Intensity {
		params.SunIntensity = primaryLight.Intensity
	}
	params = clampSkyAtmosphereParams(params)
	o.skyRenderer.Render(view, projection, params)
}

func (o *OceanSystem) reflectionSize(width, height int32) (int32, int32) {
	reflW := int32(float32(width) * o.quality.ReflectionResolutionScale)
	reflH := int32(float32(height) * o.quality.ReflectionResolutionScale)
	if reflW < 1 {
		reflW = 1
	}
	if reflH < 1 {
		reflH = 1
	}
	return reflW, reflH
}

func (o *OceanSystem) reflectionClipPlane() mgl32.Vec4 {
	// Keep only geometry above the water plane in planar reflection targets.
	return mgl32.Vec4{0.0, 1.0, 0.0, -(o.waterLevel + 0.03)}
}

// setReflectionClipState переключает clip-plane сразу для обоих рендереров сцены
// (базовый материал и glow), чтобы reflection pass отсекал геометрию корректно.
func (o *OceanSystem) setReflectionClipState(ctx *Context, enabled bool) {
	plane := o.reflectionClipPlane()

	ctx.Renderer.Use()
	ctx.Renderer.SetClipPlane(plane)
	ctx.Renderer.SetClipPlaneEnabled(enabled)

	ctx.GlowRenderer.Use()
	ctx.GlowRenderer.SetClipPlane(plane)
	ctx.GlowRenderer.SetClipPlaneEnabled(enabled)
}

// reflectedViewForWater отражает камеру относительно уровня воды.
// Используется для planar reflection pass.
func reflectedViewForWater(camera *Camera, planeY float32) (mgl32.Mat4, mgl32.Vec3) {
	reflectedPos := camera.Position
	reflectedPos[1] = planeY + (planeY - reflectedPos[1])

	reflectedFront := camera.Front
	reflectedFront[1] = -reflectedFront[1]

	reflectedUp := camera.Up
	reflectedUp[1] = -reflectedUp[1]

	return mgl32.LookAtV(reflectedPos, reflectedPos.Add(reflectedFront), reflectedUp), reflectedPos
}

// projectionNearFarFromMatrix извлекает near/far из perspective matrix.
// Это нужно post-эффектам для линейной реконструкции глубины.
func projectionNearFarFromMatrix(proj mgl32.Mat4) (float32, float32) {
	m22 := proj[10]
	m32 := proj[14]
	near := m32 / (m22 - 1.0)
	far := m32 / (m22 + 1.0)
	if near < 0 {
		near = -near
	}
	if far < 0 {
		far = -far
	}
	if near < 0.0001 || far <= near {
		return 0.1, 100.0
	}
	return near, far
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// updateEdgeFadeRange подбирает мягкую зону затухания у края океанского покрытия.
// Это скрывает геометрическую границу дальнего кольца и маскирует LOD-перестройки.
func (o *OceanSystem) updateEdgeFadeRange(maxRadius float32) {
	if maxRadius < o.baseTileSize*2.0 {
		maxRadius = o.baseTileSize * 2.0
	}

	near := maxRadius * 0.78
	far := maxRadius * 0.96
	minGap := o.baseTileSize * 0.8
	if far < near+minGap {
		far = near + minGap
	}

	o.edgeFadeNear = near
	o.edgeFadeFar = far
}
