package main

import (
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

// waterMovementMode РѕРїРёСЃС‹РІР°РµС‚ С„РёР·РёС‡РµСЃРєРёР№ СЂРµР¶РёРј РёРіСЂРѕРєР° РѕС‚РЅРѕСЃРёС‚РµР»СЊРЅРѕ РїРѕРІРµСЂС…РЅРѕСЃС‚Рё РІРѕРґС‹.
type waterMovementMode int

const (
	waterModeAboveWater waterMovementMode = iota
	waterModeSurfaceFloating
	waterModeUnderwaterSwimming
)

func (m waterMovementMode) Label() string {
	switch m {
	case waterModeSurfaceFloating:
		return "SURFACE"
	case waterModeUnderwaterSwimming:
		return "UNDERWATER"
	default:
		return "ABOVE"
	}
}

// subnauticaGame РѕР±СЉРµРґРёРЅСЏРµС‚ runtime-СЃРѕСЃС‚РѕСЏРЅРёРµ РїСЂРёРјРµСЂР°:
// РјРёСЂ ECS, СЂРµРЅРґРµСЂ-РїР°Р№РїР»Р°Р№РЅ, С„РёР·РёРєСѓ РёРіСЂРѕРєР°, UI-РєРѕРЅСЃРѕР»СЊ Рё РЅР°СЃС‚СЂРѕР№РєРё РєР°С‡РµСЃС‚РІР°.
type subnauticaGame struct {
	world                  *engine.World
	creatureManager        *objects.CreatureManager
	creatureModels         map[string]*engine.StaticModel
	glowPlantInstances     []glowPlantInstance
	glowPlantMeshes        map[objects.PlantType]engine.Mesh
	firstPersonHands       *firstPersonHands
	terrain                *objects.EditableTerrain
	terrainLoadMessage     string
	sceneLightsLoadMessage string
	worldStreaming         *worldStreamingManager
	groundModel            mgl32.Mat4
	groundMinY             float32
	groundMaxY             float32
	lastMode               string
	lastWindowTitle        string
	nextTitleUpdateTime    float32
	oceanSystem            *engine.OceanSystem
	audioPlayer            *engine.AudioPlayer
	lightManager           *engine.LightManager
	shadowMap              *engine.ShadowMap
	shadowShader           *engine.ShadowShader
	shadowPass             *engine.ShadowPass
	renderGraph            *engine.RenderGraph
	renderContext          *engine.RenderContext
	skyParams              engine.SkyAtmosphereParams
	sceneLighting          subnauticaSceneLighting
	console                *devConsole
	fpsOverlay             *fpsOverlay
	consoleInputActive     bool
	screenWidth            int
	screenHeight           int
	engineCtx              *engine.Context

	baseFOV            float32
	baseNear           float32
	baseFar            float32
	qualityPreset      engine.GraphicsQualityPreset
	qualitySettings    subnauticaGraphicsQuality
	streamingConfig    engine.StreamingConfig
	currentFarClip     float32
	shadowSceneCenter  mgl32.Vec3
	qualityLowKeyDown  bool
	qualityMedKeyDown  bool
	qualityHighKeyDown bool

	// authoritative/local СЂР°Р·РґРµР»РµРЅС‹ СЏРІРЅРѕ РґР»СЏ coop-ready Р°СЂС…РёС‚РµРєС‚СѓСЂС‹.
	authoritative     authoritativeState
	localControl      localControlState
	localPresentation localPresentationState

	runtimeEntities runtimeEntityBindings
	pendingCommands []simulationCommand
	frameEvents     []simulationEvent

	editor                editorModeState
	editorBrushMesh       engine.Mesh
	editorLightMarkerMesh engine.Mesh
}

// glowPlantInstance вЂ” РєСЌС€ РґР°РЅРЅС‹С… СЂР°СЃС‚РµРЅРёСЏ РґР»СЏ Р±С‹СЃС‚СЂРѕРіРѕ СЂРµРЅРґРµСЂР° Р±РµР· РїРѕРІС‚РѕСЂРЅРѕРіРѕ РґРѕСЃС‚СѓРїР° Рє ECS.
type glowPlantInstance struct {
	entityID       EntityID
	mesh           engine.Mesh
	model          mgl32.Mat4
	position       mgl32.Vec3
	boundingRadius float32
	stableSeed     uint32
	glowColor      mgl32.Vec3
	glowIntensity  float32
	pulseSpeed     float32
}

// newSubnauticaGame РїРѕРґРіРѕС‚Р°РІР»РёРІР°РµС‚ СЃС‚Р°СЂС‚РѕРІС‹Рµ РіСЂР°С„РёС‡РµСЃРєРёРµ РїР°СЂР°РјРµС‚СЂС‹ РёР· РєРѕРЅС„РёРіР°.
func newSubnauticaGame(graphics engine.GraphicsConfig) *subnauticaGame {
	fov := graphics.FOV
	if fov <= 0 {
		fov = 60
	}
	near := graphics.Near
	if near <= 0 {
		near = 0.1
	}
	far := graphics.Far
	if far <= 0 {
		far = 100
	}
	if far <= near+1 {
		far = near + 50
	}

	preset := engine.ParseGraphicsQualityPreset(graphics.Quality)
	streaming := sanitizeStreamingConfig(graphics.Streaming)
	return &subnauticaGame{
		baseFOV:           fov,
		baseNear:          near,
		baseFar:           far,
		qualityPreset:     preset,
		qualitySettings:   qualitySettingsForPreset(preset),
		streamingConfig:   streaming,
		currentFarClip:    far,
		shadowSceneCenter: mgl32.Vec3{0, -3, 0},
	}
}

// Init РїРѕРґРЅРёРјР°РµС‚ РІСЃРµ РїРѕРґСЃРёСЃС‚РµРјС‹: РјРёСЂ, РјРµС€Рё, СЃРІРµС‚, С‚РµРЅРё, РѕРєРµР°РЅ, СЃСѓС‰РµСЃС‚РІ Рё UI-РѕРІРµСЂР»РµРё.
func (g *subnauticaGame) Init(ctx *engine.Context) error {
	g.engineCtx = ctx
	g.world = engine.NewWorld()

	if g.qualityPreset == "" {
		g.qualityPreset = engine.GraphicsQualityHigh
	}
	if g.qualitySettings.Preset == "" {
		g.qualitySettings = qualitySettingsForPreset(g.qualityPreset)
	}

	kelpMesh := ctx.GlowRenderer.NewMeshWithNormals(objects.KelpVertices, objects.KelpNormals)
	bushMesh := ctx.GlowRenderer.NewMeshWithNormals(objects.BushVertices, objects.BushNormals)
	coralMesh := ctx.GlowRenderer.NewMeshWithNormals(objects.CoralVertices, objects.CoralNormals)
	flowerMesh := ctx.GlowRenderer.NewMeshWithNormals(objects.FlowerVertices, objects.FlowerNormals)
	g.glowPlantMeshes = map[objects.PlantType]engine.Mesh{
		objects.PlantKelp:   kelpMesh,
		objects.PlantBush:   bushMesh,
		objects.PlantCoral:  coralMesh,
		objects.PlantFlower: flowerMesh,
	}

	g.initTerrain(ctx)
	g.worldStreaming = newWorldStreamingManager(ctx.Renderer, g.terrain, g.streamingConfig)
	g.initEditorMode(ctx)

	spawnX, spawnZ := float32(0), float32(0)
	spawnY := g.groundHeightAt(spawnX, spawnZ) + playerEyeHeight
	spawnPos := mgl32.Vec3{spawnX, spawnY, spawnZ}
	ctx.Camera.Position = spawnPos
	g.initCoopReadyState(spawnPos, ctx.Camera.Front)

	g.skyParams = engine.DefaultSkyAtmosphereParams()
	g.skyParams.SunDiscSize = 0.038
	g.skyParams.SunHaloIntensity = 1.45
	g.skyParams.HorizonColor = mgl32.Vec3{0.70, 0.86, 0.99}
	g.skyParams.ZenithColor = mgl32.Vec3{0.10, 0.40, 0.76}
	g.skyParams.AtmosphereBlend = 1.08
	g.skyParams.FogSunInfluence = 0.5

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
	g.firstPersonHands = newFirstPersonHands(ctx)

	if err := g.initCreatures(ctx); err != nil {
		return err
	}
	g.queueInitialGameplaySpawns()
	g.runSimulationFrame(ctx)
	g.setLocalPlayerPosition(spawnPos)
	ctx.Camera.Position = g.localPlayerPosition()

	g.applyGraphicsPreset(ctx, g.qualityPreset, false)
	if g.worldStreaming != nil {
		g.worldStreaming.UpdateStreaming(g.localPlayerPosition())
	}

	if audioPlayer, err := engine.PlayBackgroundMusic("assets/audio/ambient.mp3"); err == nil {
		g.audioPlayer = audioPlayer
		g.audioPlayer.SetVolume(0.3)
	}

	g.console = newDevConsole(ctx.Window, g.executeConsoleCommand)
	g.console.Log("console: ready")
	g.console.Log("type 'noclip' and press enter")
	g.console.Log("quality hotkeys: F1 low / F2 medium / F3 high")
	g.console.Log("console command: quality low|medium|high")
	g.console.Log("editor: TAB toggle, T terrain, L lights, V water, F overview, RMB look, 1/2/3/4 terrain modes")
	g.console.Log("active quality: " + g.qualityPreset.Label())
	if g.terrainLoadMessage != "" {
		g.console.Log(g.terrainLoadMessage)
	}
	if g.sceneLightsLoadMessage != "" {
		g.console.Log(g.sceneLightsLoadMessage)
	}

	if g.console != nil {
		g.fpsOverlay = newFPSOverlay(g.console.renderer)
	}

	return nil
}

// Render РІС‹РїРѕР»РЅСЏРµС‚ render-graph Рё РїРѕРІРµСЂС… СЂРµР·СѓР»СЊС‚Р°С‚Р° СЂРёСЃСѓРµС‚ РєРѕРЅСЃРѕР»СЊ/FPS HUD.
func (g *subnauticaGame) Render(ctx *engine.Context) error {
	g.renderContext.Context = ctx
	g.renderContext.SetResource("lightManager", g.lightManager)
	if err := g.renderGraph.Execute(g.renderContext); err != nil {
		return err
	}

	w := g.screenWidth
	h := g.screenHeight
	if w <= 0 || h <= 0 {
		w, h = ctx.Window.GetSize()
		g.screenWidth = w
		g.screenHeight = h
	}
	if g.console != nil {
		g.console.Render(w, h, ctx.Time)
	}
	if g.fpsOverlay != nil {
		g.fpsOverlay.Render(w, h)
	}
	g.renderStreamingDebugHUD(w, h)
	g.renderEditorHUD(w, h, ctx.Time)

	return nil
}
