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
	worldRuntimeState
	environmentRuntimeState
	simulationRuntimeState
	editorRuntimeState
	uiRuntimeState
	qualityRuntimeState
	appRuntimeState
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
	streaming := engine.SanitizeStreamingConfig(graphics.Streaming)
	return &subnauticaGame{
		qualityRuntimeState: qualityRuntimeState{
			baseFOV:         fov,
			baseNear:        near,
			baseFar:         far,
			qualityPreset:   preset,
			qualitySettings: qualitySettingsForPreset(preset),
			streamingConfig: streaming,
			currentFarClip:  far,
		},
		environmentRuntimeState: environmentRuntimeState{
			shadowSceneCenter: mgl32.Vec3{0, -3, 0},
		},
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

	if err := g.initEnvironmentRuntime(ctx); err != nil {
		return err
	}
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
	if err := g.executeEnvironmentRenderGraph(ctx); err != nil {
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
