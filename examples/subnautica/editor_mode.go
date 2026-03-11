package main

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

type editorPrimaryTool uint8

const (
	editorPrimaryToolTerrain editorPrimaryTool = iota
	editorPrimaryToolLights
)

func (t editorPrimaryTool) Label() string {
	switch t {
	case editorPrimaryToolLights:
		return "LIGHTS"
	default:
		return "TERRAIN"
	}
}

type editorModeState struct {
	Enabled             bool
	PrimaryTool         editorPrimaryTool
	Tool                objects.TerrainBrushTool
	BrushRadius         float32
	BrushStrength       float32
	flattenTargetHeight float32
	flattenTargetLocked bool
	LightPreset         localLightPreset
	WaterVisible        bool

	SelectedLightIndex int

	HasCursorRay  bool
	CursorOrigin  mgl32.Vec3
	CursorDir     mgl32.Vec3
	HasTerrainHit bool
	HitPosition   mgl32.Vec3

	toggleKeyDown          bool
	terrainToolKeyDown     bool
	lightToolKeyDown       bool
	raiseKeyDown           bool
	lowerKeyDown           bool
	smoothKeyDown          bool
	flattenKeyDown         bool
	waterToggleKeyDown     bool
	focusCameraKeyDown     bool
	removeLightKeyDown     bool
	terrainLeftClickDown   bool
	leftClickDown          bool
	saveChordDown          bool
	RuntimePointLightCount int
	SkippedPointLightCount int
	statusText             string
	statusUntilSec         float32
}

func (g *subnauticaGame) initEditorMode(ctx *engine.Context) {
	g.editor = editorModeState{
		Enabled:            false,
		PrimaryTool:        editorPrimaryToolTerrain,
		Tool:               objects.TerrainBrushRaise,
		BrushRadius:        editorBrushRadiusDefault,
		BrushStrength:      editorBrushStrengthDefault,
		LightPreset:        localLightPresetMedium,
		WaterVisible:       true,
		SelectedLightIndex: -1,
	}

	seedVerts := []float32{0, 0, 0}
	seedNorms := []float32{0, 1, 0}
	g.editorBrushMesh = ctx.Renderer.NewMeshWithNormals(seedVerts, seedNorms)
	ctx.Renderer.UpdateMeshWithNormals(&g.editorBrushMesh, nil, nil)

	markerVerts, markerNorms := buildEditorLightMarkerMesh()
	g.editorLightMarkerMesh = ctx.Renderer.NewMeshWithNormals(markerVerts, markerNorms)
}

func (g *subnauticaGame) setEditorStatus(text string, until float32) {
	g.editor.statusText = text
	g.editor.statusUntilSec = until
}

func (g *subnauticaGame) syncEditorCapture(ctx *engine.Context, inputBlocked bool) {
	if ctx == nil || ctx.Window == nil {
		return
	}
	if inputBlocked {
		return
	}

	if !g.editor.Enabled {
		if ctx.Window.GetInputMode(glfw.CursorMode) != glfw.CursorDisabled {
			ctx.Window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		}
		ctx.Camera.SetMouseLookEnabled(true)
		return
	}

	rightDown := ctx.Window.GetMouseButton(glfw.MouseButtonRight) == glfw.Press
	if rightDown {
		if ctx.Window.GetInputMode(glfw.CursorMode) != glfw.CursorDisabled {
			ctx.Window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		}
		ctx.Camera.SetMouseLookEnabled(true)
		return
	}

	if ctx.Window.GetInputMode(glfw.CursorMode) != glfw.CursorNormal {
		ctx.Window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	}
	ctx.Camera.SetMouseLookEnabled(false)
}

func (g *subnauticaGame) updateEditorMode(ctx *engine.Context, inputBlocked bool) bool {
	if ctx == nil || ctx.Window == nil {
		return false
	}

	if !inputBlocked && keyPressedOnce(ctx.Window, glfw.KeyTab, &g.editor.toggleKeyDown) {
		g.editor.Enabled = !g.editor.Enabled
		g.resetEditorToolDragState()
		g.clearEditorRayAndHit()

		player := g.localPlayerState()
		if player != nil {
			player.VerticalVelocity = 0
			player.OnGround = false
		}
		g.localControl.SpaceWasDown = false
		g.localPresentation.ProceduralYOffset = 0
		g.localPresentation.UnderwaterIdleMix = 0

		g.editor.WaterVisible = true
		if g.editor.Enabled {
			g.applyEditorOverviewCamera(ctx, false)
			g.setEditorStatus("EDITOR ON", ctx.Time+1.8)
			if g.console != nil {
				g.console.Log("editor: ON")
			}
		} else {
			g.setEditorStatus("EDITOR OFF", ctx.Time+1.8)
			if g.console != nil {
				g.console.Log("editor: OFF")
			}
		}
	}

	if !g.editor.Enabled {
		g.clearEditorRayAndHit()
		g.resetEditorTerrainDragState()
		g.updateEditorBrushPreviewMesh(ctx)
		return false
	}

	g.handleEditorToolHotkeys(ctx, inputBlocked)
	g.handleEditorBrushParamHotkeys(ctx, inputBlocked)
	g.handleEditorLightDeleteHotkey(ctx, inputBlocked)
	g.handleEditorSaveHotkey(ctx, inputBlocked)
	g.updateEditorCamera(ctx, inputBlocked)
	g.updateEditorTerrainHit(ctx, inputBlocked)
	g.applyEditorBrush(ctx, inputBlocked)
	g.applyEditorLightTool(ctx, inputBlocked)
	g.updateEditorBrushPreviewMesh(ctx)
	return true
}

func (g *subnauticaGame) clearEditorRayAndHit() {
	g.editor.HasCursorRay = false
	g.editor.HasTerrainHit = false
}

func (g *subnauticaGame) resetEditorTerrainDragState() {
	g.editor.terrainLeftClickDown = false
	g.editor.flattenTargetLocked = false
}

func (g *subnauticaGame) resetEditorToolDragState() {
	g.resetEditorTerrainDragState()
	g.editor.leftClickDown = false
}
