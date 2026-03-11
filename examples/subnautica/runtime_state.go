package main

import (
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

// worldRuntimeState owns ECS/world data and content streaming state.
type worldRuntimeState struct {
	world              *engine.World
	terrain            *objects.EditableTerrain
	worldStreaming     *worldStreamingManager
	groundModel        mgl32.Mat4
	groundMinY         float32
	groundMaxY         float32
	terrainLoadMessage string

	creatureManager *objects.CreatureManager
	creatureModels  map[string]*engine.StaticModel

	glowPlantInstances []glowPlantInstance
	glowPlantMeshes    map[objects.PlantType]engine.Mesh

	firstPersonHands *firstPersonHands
}

// environmentRuntimeState owns render/environment orchestration.
type environmentRuntimeState struct {
	oceanSystem            *engine.OceanSystem
	lightManager           *engine.LightManager
	shadowMap              *engine.ShadowMap
	shadowShader           *engine.ShadowShader
	shadowPass             *engine.ShadowPass
	renderGraph            *engine.RenderGraph
	renderContext          *engine.RenderContext
	skyParams              engine.SkyAtmosphereParams
	sceneLighting          subnauticaSceneLighting
	sceneLightsLoadMessage string
	shadowSceneCenter      mgl32.Vec3
	screenWidth            int
	screenHeight           int
}

// simulationRuntimeState owns authoritative and local simulation layers.
type simulationRuntimeState struct {
	authoritative     authoritativeState
	localControl      localControlState
	localPresentation localPresentationState

	runtimeEntities runtimeEntityBindings
	pendingCommands []simulationCommand
	frameEvents     []simulationEvent
}

// editorRuntimeState owns editor tool state and meshes.
type editorRuntimeState struct {
	editor                editorModeState
	editorBrushMesh       engine.Mesh
	editorLightMarkerMesh engine.Mesh
}

// uiRuntimeState owns HUD/debug overlays and input-capture state.
type uiRuntimeState struct {
	console            *devConsole
	fpsOverlay         *fpsOverlay
	consoleInputActive bool

	lastMode            string
	lastWindowTitle     string
	nextTitleUpdateTime float32
}

// qualityRuntimeState owns graphics quality + streaming knobs.
type qualityRuntimeState struct {
	baseFOV            float32
	baseNear           float32
	baseFar            float32
	qualityPreset      engine.GraphicsQualityPreset
	qualitySettings    subnauticaGraphicsQuality
	streamingConfig    engine.StreamingConfig
	currentFarClip     float32
	qualityLowKeyDown  bool
	qualityMedKeyDown  bool
	qualityHighKeyDown bool
}

// appRuntimeState owns cross-cutting app runtime state.
type appRuntimeState struct {
	engineCtx   *engine.Context
	audioPlayer *engine.AudioPlayer
}
