package main

import (
	"fmt"
	"math"

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
		g.editor.leftClickDown = false
		g.editor.terrainLeftClickDown = false
		g.editor.flattenTargetLocked = false
		g.editor.HasCursorRay = false
		g.editor.HasTerrainHit = false
		player := g.localPlayerState()
		if player != nil {
			player.VerticalVelocity = 0
			player.OnGround = false
		}
		g.localControl.SpaceWasDown = false
		g.localPresentation.ProceduralYOffset = 0
		g.localPresentation.UnderwaterIdleMix = 0
		if g.editor.Enabled {
			g.editor.WaterVisible = true
			g.applyEditorOverviewCamera(ctx, false)
			g.setEditorStatus("EDITOR ON", ctx.Time+1.8)
			if g.console != nil {
				g.console.Log("editor: ON")
			}
		} else {
			g.editor.WaterVisible = true
			g.setEditorStatus("EDITOR OFF", ctx.Time+1.8)
			if g.console != nil {
				g.console.Log("editor: OFF")
			}
		}
	}

	if !g.editor.Enabled {
		g.editor.HasCursorRay = false
		g.editor.HasTerrainHit = false
		g.editor.terrainLeftClickDown = false
		g.editor.flattenTargetLocked = false
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

func (g *subnauticaGame) handleEditorToolHotkeys(ctx *engine.Context, inputBlocked bool) {
	if inputBlocked {
		return
	}

	if keyPressedOnce(ctx.Window, glfw.KeyT, &g.editor.terrainToolKeyDown) {
		g.editor.PrimaryTool = editorPrimaryToolTerrain
		g.setEditorStatus("TOOL: TERRAIN", ctx.Time+1.2)
	}
	if keyPressedOnce(ctx.Window, glfw.KeyL, &g.editor.lightToolKeyDown) {
		g.editor.PrimaryTool = editorPrimaryToolLights
		g.editor.terrainLeftClickDown = false
		g.editor.flattenTargetLocked = false
		g.setEditorStatus("TOOL: LIGHTS", ctx.Time+1.2)
	}

	if keyPressedOnce(ctx.Window, glfw.Key1, &g.editor.raiseKeyDown) {
		if g.editor.PrimaryTool == editorPrimaryToolLights {
			g.editor.LightPreset = localLightPresetSmall
			g.setEditorStatus("PRESET: SMALL", ctx.Time+1.2)
		} else {
			g.editor.Tool = objects.TerrainBrushRaise
			g.editor.flattenTargetLocked = false
			g.setEditorStatus("BRUSH: RAISE", ctx.Time+1.2)
		}
	}
	if keyPressedOnce(ctx.Window, glfw.Key2, &g.editor.lowerKeyDown) {
		if g.editor.PrimaryTool == editorPrimaryToolLights {
			g.editor.LightPreset = localLightPresetMedium
			g.setEditorStatus("PRESET: MEDIUM", ctx.Time+1.2)
		} else {
			g.editor.Tool = objects.TerrainBrushLower
			g.editor.flattenTargetLocked = false
			g.setEditorStatus("BRUSH: LOWER", ctx.Time+1.2)
		}
	}
	if keyPressedOnce(ctx.Window, glfw.Key3, &g.editor.smoothKeyDown) {
		if g.editor.PrimaryTool == editorPrimaryToolLights {
			g.editor.LightPreset = localLightPresetLarge
			g.setEditorStatus("PRESET: LARGE", ctx.Time+1.2)
		} else {
			g.editor.Tool = objects.TerrainBrushSmooth
			g.editor.flattenTargetLocked = false
			g.setEditorStatus("BRUSH: SMOOTH", ctx.Time+1.2)
		}
	}
	if keyPressedOnce(ctx.Window, glfw.Key4, &g.editor.flattenKeyDown) {
		if g.editor.PrimaryTool == editorPrimaryToolTerrain {
			g.editor.Tool = objects.TerrainBrushFlatten
			g.editor.flattenTargetLocked = false
			g.setEditorStatus("BRUSH: FLATTEN", ctx.Time+1.2)
		}
	}

	if keyPressedOnce(ctx.Window, glfw.KeyV, &g.editor.waterToggleKeyDown) {
		g.editor.WaterVisible = !g.editor.WaterVisible
		if g.editor.WaterVisible {
			g.setEditorStatus("WATER: ON", ctx.Time+1.2)
			if g.console != nil {
				g.console.Log("editor water: ON")
			}
		} else {
			g.setEditorStatus("WATER: OFF", ctx.Time+1.2)
			if g.console != nil {
				g.console.Log("editor water: OFF")
			}
		}
	}

	if keyPressedOnce(ctx.Window, glfw.KeyF, &g.editor.focusCameraKeyDown) {
		g.applyEditorOverviewCamera(ctx, true)
	}
}

func (g *subnauticaGame) handleEditorBrushParamHotkeys(ctx *engine.Context, inputBlocked bool) {
	if inputBlocked || g.editor.PrimaryTool != editorPrimaryToolTerrain {
		return
	}

	radiusDelta := float32(0)
	if ctx.Window.GetKey(glfw.KeyLeftBracket) == glfw.Press {
		radiusDelta -= editorBrushRadiusAdjustSpeed * ctx.DeltaTime
	}
	if ctx.Window.GetKey(glfw.KeyRightBracket) == glfw.Press {
		radiusDelta += editorBrushRadiusAdjustSpeed * ctx.DeltaTime
	}
	g.editor.BrushRadius = clampFloat(g.editor.BrushRadius+radiusDelta, editorBrushRadiusMin, editorBrushRadiusMax)

	strengthDelta := float32(0)
	if ctx.Window.GetKey(glfw.KeyMinus) == glfw.Press {
		strengthDelta -= editorBrushStrengthAdjustSpeed * ctx.DeltaTime
	}
	if ctx.Window.GetKey(glfw.KeyEqual) == glfw.Press {
		strengthDelta += editorBrushStrengthAdjustSpeed * ctx.DeltaTime
	}
	g.editor.BrushStrength = clampFloat(g.editor.BrushStrength+strengthDelta, editorBrushStrengthMin, editorBrushStrengthMax)
}

func (g *subnauticaGame) handleEditorLightDeleteHotkey(ctx *engine.Context, inputBlocked bool) {
	if inputBlocked || g.editor.PrimaryTool != editorPrimaryToolLights {
		g.editor.removeLightKeyDown = false
		return
	}

	if !keyPressedOnce(ctx.Window, glfw.KeyDelete, &g.editor.removeLightKeyDown) {
		return
	}

	selection := g.editor.SelectedLightIndex
	if selection < 0 || selection >= len(g.sceneLighting.PointLights) {
		g.setEditorStatus("NO LIGHT SELECTED", ctx.Time+1.3)
		return
	}

	g.queueCommand(simulationCommand{
		Kind: simulationCommandRemoveScenePointLight,
		RemoveScenePointLight: removeScenePointLightCommand{
			LightIndex: selection,
		},
	})
	g.editor.SelectedLightIndex = -1
	g.setEditorStatus("LIGHT REMOVED", ctx.Time+1.3)
}

func (g *subnauticaGame) handleEditorSaveHotkey(ctx *engine.Context, inputBlocked bool) {
	if inputBlocked {
		g.editor.saveChordDown = false
		return
	}

	ctrlDown := ctx.Window.GetKey(glfw.KeyLeftControl) == glfw.Press || ctx.Window.GetKey(glfw.KeyRightControl) == glfw.Press
	saveDown := ctrlDown && ctx.Window.GetKey(glfw.KeyS) == glfw.Press
	if saveDown && !g.editor.saveChordDown {
		if err := g.saveEditorWorld(); err != nil {
			g.setEditorStatus("SAVE FAILED", ctx.Time+2.4)
			if g.console != nil {
				g.console.Log("editor save failed: " + err.Error())
			}
		} else {
			g.setEditorStatus("WORLD SAVED", ctx.Time+2.4)
			if g.console != nil {
				g.console.Log("terrain saved: " + terrainSavePath)
				g.console.Log("scene lights saved: " + sceneLightsSavePath)
			}
		}
	}
	g.editor.saveChordDown = saveDown
}

func (g *subnauticaGame) updateEditorCamera(ctx *engine.Context, inputBlocked bool) {
	if inputBlocked {
		return
	}

	speed := float32(editorCameraSpeed)
	if ctx.Window.GetKey(glfw.KeyLeftShift) == glfw.Press {
		speed *= float32(editorCameraSprintMul)
	}
	ctx.Camera.Speed = speed
	ctx.Camera.ProcessGroundInput(ctx.Window, ctx.DeltaTime)

	step := speed * ctx.DeltaTime
	if ctx.Window.GetKey(glfw.KeySpace) == glfw.Press {
		ctx.Camera.Position = ctx.Camera.Position.Add(ctx.Camera.Up.Mul(step))
	}
	if ctx.Window.GetKey(glfw.KeyLeftControl) == glfw.Press {
		ctx.Camera.Position = ctx.Camera.Position.Sub(ctx.Camera.Up.Mul(step))
	}
}

func (g *subnauticaGame) applyEditorOverviewCamera(ctx *engine.Context, withStatus bool) {
	if ctx == nil || ctx.Camera == nil {
		return
	}

	focus := g.editorFocusPoint(ctx.Camera.Position)
	planarForward := mgl32.Vec3{ctx.Camera.Front.X(), 0, ctx.Camera.Front.Z()}
	if planarForward.Len() < 0.0001 {
		planarForward = mgl32.Vec3{0, 0, -1}
	} else {
		planarForward = planarForward.Normalize()
	}

	pos := focus.Sub(planarForward.Mul(editorCameraOverviewBackDistance))
	pos[1] += editorCameraOverviewLift
	if g.terrain != nil {
		// Русский комментарий: фиксируем минимальную высоту обзорной камеры над рельефом,
		// чтобы при входе в редактор взгляд не уходил под террейн на крутых склонах.
		minY := g.groundHeightAt(pos.X(), pos.Z()) + editorCameraGroundClearance
		if pos.Y() < minY {
			pos[1] = minY
		}
	}

	yawDeg := float32(math.Atan2(float64(planarForward.Z()), float64(planarForward.X())) * 180.0 / math.Pi)
	ctx.Camera.Position = pos
	ctx.Camera.Yaw = yawDeg
	ctx.Camera.Pitch = editorCameraOverviewPitchDeg
	updateCameraDirectionFromYawPitch(ctx.Camera)
	ctx.Camera.FirstMouse = true

	if withStatus {
		g.setEditorStatus("CAMERA: OVERVIEW", ctx.Time+1.2)
	}
}

func (g *subnauticaGame) editorFocusPoint(fallback mgl32.Vec3) mgl32.Vec3 {
	if g.editor.SelectedLightIndex >= 0 && g.editor.SelectedLightIndex < len(g.sceneLighting.PointLights) {
		light := sanitizeScenePointLight(g.sceneLighting.PointLights[g.editor.SelectedLightIndex])
		return light.Position
	}
	if g.editor.HasTerrainHit {
		return g.editor.HitPosition
	}
	if player := g.localPlayerState(); player != nil {
		return player.Position.Mgl()
	}
	return fallback
}

func updateCameraDirectionFromYawPitch(camera *engine.Camera) {
	if camera == nil {
		return
	}

	yaw := mgl32.DegToRad(camera.Yaw)
	pitch := mgl32.DegToRad(camera.Pitch)
	front := mgl32.Vec3{
		float32(math.Cos(float64(yaw)) * math.Cos(float64(pitch))),
		float32(math.Sin(float64(pitch))),
		float32(math.Sin(float64(yaw)) * math.Cos(float64(pitch))),
	}
	if front.Len() < 0.0001 {
		front = mgl32.Vec3{0, 0, -1}
	} else {
		front = front.Normalize()
	}
	camera.Front = front
}

func (g *subnauticaGame) updateEditorTerrainHit(ctx *engine.Context, inputBlocked bool) {
	g.editor.HasCursorRay = false
	g.editor.HasTerrainHit = false
	if inputBlocked || g.terrain == nil {
		return
	}

	if ctx.Window.GetInputMode(glfw.CursorMode) == glfw.CursorDisabled {
		return
	}

	width, height := ctx.Window.GetSize()
	if width <= 0 || height <= 0 {
		return
	}

	cursorX, cursorY := ctx.Window.GetCursorPos()
	rayOrigin, rayDir, ok := screenPointToWorldRay(
		cursorX,
		cursorY,
		width,
		height,
		ctx.Camera.ViewMatrix(),
		ctx.Projection,
	)
	if !ok {
		return
	}
	g.editor.HasCursorRay = true
	g.editor.CursorOrigin = rayOrigin
	g.editor.CursorDir = rayDir

	hit, ok := g.terrain.Raycast(rayOrigin, rayDir, editorRaycastMaxDistance)
	if !ok {
		return
	}
	g.editor.HasTerrainHit = true
	g.editor.HitPosition = hit
}

func (g *subnauticaGame) applyEditorBrush(ctx *engine.Context, inputBlocked bool) {
	if g.editor.PrimaryTool != editorPrimaryToolTerrain {
		g.editor.terrainLeftClickDown = false
		g.editor.flattenTargetLocked = false
		return
	}
	if ctx == nil || ctx.Window == nil {
		g.editor.terrainLeftClickDown = false
		g.editor.flattenTargetLocked = false
		return
	}
	leftDown := ctx.Window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press
	pressedOnce := leftDown && !g.editor.terrainLeftClickDown
	g.editor.terrainLeftClickDown = leftDown
	if !leftDown {
		g.editor.flattenTargetLocked = false
	}

	if inputBlocked || g.terrain == nil || !g.editor.HasTerrainHit {
		return
	}
	if ctx.Window.GetMouseButton(glfw.MouseButtonRight) == glfw.Press {
		return
	}
	if !leftDown {
		return
	}

	flattenTarget := g.editor.HitPosition.Y()
	if g.editor.Tool == objects.TerrainBrushFlatten {
		// Русский комментарий: target height фиксируем в момент начала удержания ЛКМ.
		// Это делает непрерывный flatten предсказуемым: высота цели не "плавает" от кадра к кадру.
		if pressedOnce || !g.editor.flattenTargetLocked {
			g.editor.flattenTargetHeight = g.editor.HitPosition.Y()
			g.editor.flattenTargetLocked = true
			g.setEditorStatus(fmt.Sprintf("FLATTEN TARGET: %.2f", g.editor.flattenTargetHeight), ctx.Time+1.2)
		}
		flattenTarget = g.editor.flattenTargetHeight
	}

	g.queueCommand(simulationCommand{
		Kind: simulationCommandApplyTerrainBrush,
		TerrainBrush: terrainBrushCommand{
			Center:        simVec3FromMgl(g.editor.HitPosition),
			Radius:        g.editor.BrushRadius,
			Strength:      g.editor.BrushStrength,
			DeltaTime:     ctx.DeltaTime,
			Tool:          g.editor.Tool,
			FlattenTarget: flattenTarget,
		},
	})
}

func (g *subnauticaGame) applyEditorLightTool(ctx *engine.Context, inputBlocked bool) {
	if g.editor.PrimaryTool != editorPrimaryToolLights {
		g.editor.leftClickDown = false
		return
	}
	if inputBlocked || ctx == nil || ctx.Window == nil {
		g.editor.leftClickDown = false
		return
	}
	if ctx.Window.GetInputMode(glfw.CursorMode) == glfw.CursorDisabled {
		g.editor.leftClickDown = false
		return
	}

	g.ensureEditorLightSelectionValid()

	if ctx.Window.GetMouseButton(glfw.MouseButtonRight) == glfw.Press {
		g.editor.leftClickDown = ctx.Window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press
		return
	}
	if !mouseButtonPressedOnce(ctx.Window, glfw.MouseButtonLeft, &g.editor.leftClickDown) {
		return
	}
	if !g.editor.HasCursorRay {
		return
	}

	if hitIndex, _, ok := g.pickEditorScenePointLight(g.editor.CursorOrigin, g.editor.CursorDir); ok {
		g.editor.SelectedLightIndex = hitIndex
		g.setEditorStatus("LIGHT SELECTED", ctx.Time+1.2)
		return
	}
	if !g.editor.HasTerrainHit {
		return
	}

	ctrlDown := ctx.Window.GetKey(glfw.KeyLeftControl) == glfw.Press || ctx.Window.GetKey(glfw.KeyRightControl) == glfw.Press
	if ctrlDown && g.editor.SelectedLightIndex >= 0 {
		g.queueCommand(simulationCommand{
			Kind: simulationCommandMoveScenePointLight,
			MoveScenePointLight: moveScenePointLightCommand{
				LightIndex: g.editor.SelectedLightIndex,
				Position:   simVec3FromMgl(g.editor.HitPosition),
			},
		})
		g.setEditorStatus("LIGHT MOVED", ctx.Time+1.2)
		return
	}

	enabledBefore := g.enabledScenePointLightsCount()
	g.queueCommand(simulationCommand{
		Kind: simulationCommandPlaceScenePointLight,
		PlaceScenePointLight: placeScenePointLightCommand{
			Position: simVec3FromMgl(g.editor.HitPosition),
			Preset:   g.editor.LightPreset,
		},
	})

	g.editor.SelectedLightIndex = len(g.sceneLighting.PointLights)
	if enabledBefore >= engine.MaxLights-1 {
		g.setEditorStatus("LIGHT PLACED (OVER RUNTIME LIMIT)", ctx.Time+2.4)
	} else {
		g.setEditorStatus("LIGHT PLACED", ctx.Time+1.2)
	}
}

func (g *subnauticaGame) updateEditorBrushPreviewMesh(ctx *engine.Context) {
	if ctx == nil || g.editorBrushMesh.VAO == 0 {
		return
	}
	if !g.editor.Enabled || g.editor.PrimaryTool != editorPrimaryToolTerrain || !g.editor.HasTerrainHit || g.terrain == nil {
		ctx.Renderer.UpdateMeshWithNormals(&g.editorBrushMesh, nil, nil)
		return
	}

	vertices, normals := buildBrushPreviewMesh(
		g.terrain,
		g.editor.HitPosition.X(),
		g.editor.HitPosition.Z(),
		g.editor.BrushRadius,
		g.editor.BrushStrength,
		g.editor.Tool,
		g.editor.flattenTargetHeight,
		g.editor.flattenTargetLocked,
	)
	ctx.Renderer.UpdateMeshWithNormals(&g.editorBrushMesh, vertices, normals)
}

func buildBrushPreviewMesh(
	terrain *objects.EditableTerrain,
	centerX, centerZ, radius, strength float32,
	tool objects.TerrainBrushTool,
	flattenTarget float32,
	flattenTargetLocked bool,
) ([]float32, []float32) {
	if terrain == nil || radius <= 0 {
		return nil, nil
	}

	appendProjectedRing := func(vertices, normals []float32, outerRadius, thickness float32, segments int) ([]float32, []float32) {
		if outerRadius <= 0 || thickness <= 0 || segments < 3 {
			return vertices, normals
		}

		innerRadius := outerRadius - thickness
		if innerRadius < 0.02 {
			innerRadius = outerRadius * 0.6
		}
		if innerRadius <= 0 {
			return vertices, normals
		}

		for i := 0; i < segments; i++ {
			a0 := float32(i) / float32(segments) * 2 * math.Pi
			a1 := float32(i+1) / float32(segments) * 2 * math.Pi

			c0 := float32(math.Cos(float64(a0)))
			s0 := float32(math.Sin(float64(a0)))
			c1 := float32(math.Cos(float64(a1)))
			s1 := float32(math.Sin(float64(a1)))

			o0x, o0z := centerX+c0*outerRadius, centerZ+s0*outerRadius
			o1x, o1z := centerX+c1*outerRadius, centerZ+s1*outerRadius
			i0x, i0z := centerX+c0*innerRadius, centerZ+s0*innerRadius
			i1x, i1z := centerX+c1*innerRadius, centerZ+s1*innerRadius

			o0y := terrain.HeightAt(o0x, o0z) + editorBrushPreviewYOffset
			o1y := terrain.HeightAt(o1x, o1z) + editorBrushPreviewYOffset
			i0y := terrain.HeightAt(i0x, i0z) + editorBrushPreviewYOffset
			i1y := terrain.HeightAt(i1x, i1z) + editorBrushPreviewYOffset

			vertices = append(vertices,
				o0x, o0y, o0z,
				o1x, o1y, o1z,
				i1x, i1y, i1z,

				o0x, o0y, o0z,
				i1x, i1y, i1z,
				i0x, i0y, i0z,
			)

			for n := 0; n < 6; n++ {
				normals = append(normals, 0, 1, 0)
			}
		}

		return vertices, normals
	}

	appendFlatRing := func(vertices, normals []float32, y, outerRadius, thickness float32, segments int) ([]float32, []float32) {
		if outerRadius <= 0 || thickness <= 0 || segments < 3 {
			return vertices, normals
		}
		innerRadius := outerRadius - thickness
		if innerRadius <= 0 {
			return vertices, normals
		}

		for i := 0; i < segments; i++ {
			a0 := float32(i) / float32(segments) * 2 * math.Pi
			a1 := float32(i+1) / float32(segments) * 2 * math.Pi
			c0 := float32(math.Cos(float64(a0)))
			s0 := float32(math.Sin(float64(a0)))
			c1 := float32(math.Cos(float64(a1)))
			s1 := float32(math.Sin(float64(a1)))

			o0x, o0z := centerX+c0*outerRadius, centerZ+s0*outerRadius
			o1x, o1z := centerX+c1*outerRadius, centerZ+s1*outerRadius
			i0x, i0z := centerX+c0*innerRadius, centerZ+s0*innerRadius
			i1x, i1z := centerX+c1*innerRadius, centerZ+s1*innerRadius

			vertices = append(vertices,
				o0x, y, o0z,
				o1x, y, o1z,
				i1x, y, i1z,

				o0x, y, o0z,
				i1x, y, i1z,
				i0x, y, i0z,
			)
			for n := 0; n < 6; n++ {
				normals = append(normals, 0, 1, 0)
			}
		}
		return vertices, normals
	}

	strengthNorm := clamp01((strength - editorBrushStrengthMin) / (editorBrushStrengthMax - editorBrushStrengthMin))
	mainThickness := radius * (0.055 + 0.055*strengthNorm)
	if mainThickness < 0.08 {
		mainThickness = 0.08
	}
	if mainThickness > radius*0.32 {
		mainThickness = radius * 0.32
	}

	centerRadius := radius * 0.15
	if centerRadius < 0.14 {
		centerRadius = 0.14
	}
	centerThickness := centerRadius * 0.45

	midRadius := radius * 0.66
	if midRadius < centerRadius*1.35 {
		midRadius = centerRadius * 1.35
	}
	innerRadius := radius * 0.34
	if innerRadius < centerRadius*1.05 {
		innerRadius = centerRadius * 1.05
	}

	midThickness := mainThickness * 0.68
	innerThickness := mainThickness * 0.56
	vertices := make([]float32, 0, 56*6*3+42*6*3+30*6*3+20*6*3)
	normals := make([]float32, 0, 56*6*3+42*6*3+30*6*3+20*6*3)
	vertices, normals = appendProjectedRing(vertices, normals, radius, mainThickness, 56)
	vertices, normals = appendProjectedRing(vertices, normals, midRadius, midThickness, 42)
	vertices, normals = appendProjectedRing(vertices, normals, innerRadius, innerThickness, 30)
	vertices, normals = appendProjectedRing(vertices, normals, centerRadius, centerThickness, 20)

	if tool == objects.TerrainBrushFlatten && flattenTargetLocked {
		targetY := flattenTarget + editorBrushPreviewYOffset*1.8
		flatRadius := radius * 0.86
		if flatRadius < 0.24 {
			flatRadius = 0.24
		}
		flatThickness := mainThickness * 0.5
		if flatThickness < 0.05 {
			flatThickness = 0.05
		}
		vertices, normals = appendFlatRing(vertices, normals, targetY, flatRadius, flatThickness, 56)
	}
	return vertices, normals
}

func buildEditorLightMarkerMesh() ([]float32, []float32) {
	appendTri := func(
		vertices, normals []float32,
		a, b, c mgl32.Vec3,
	) ([]float32, []float32) {
		ab := b.Sub(a)
		ac := c.Sub(a)
		n := ab.Cross(ac)
		if n.Len() < 0.0001 {
			n = mgl32.Vec3{0, 1, 0}
		} else {
			n = n.Normalize()
		}

		vertices = append(vertices,
			a.X(), a.Y(), a.Z(),
			b.X(), b.Y(), b.Z(),
			c.X(), c.Y(), c.Z(),
		)
		for i := 0; i < 3; i++ {
			normals = append(normals, n.X(), n.Y(), n.Z())
		}
		return vertices, normals
	}

	top := mgl32.Vec3{0, 0.6, 0}
	bottom := mgl32.Vec3{0, -0.6, 0}
	xp := mgl32.Vec3{0.38, 0, 0}
	xn := mgl32.Vec3{-0.38, 0, 0}
	zp := mgl32.Vec3{0, 0, 0.38}
	zn := mgl32.Vec3{0, 0, -0.38}

	vertices := make([]float32, 0, 8*9)
	normals := make([]float32, 0, 8*9)

	vertices, normals = appendTri(vertices, normals, top, xp, zp)
	vertices, normals = appendTri(vertices, normals, top, zp, xn)
	vertices, normals = appendTri(vertices, normals, top, xn, zn)
	vertices, normals = appendTri(vertices, normals, top, zn, xp)
	vertices, normals = appendTri(vertices, normals, bottom, zp, xp)
	vertices, normals = appendTri(vertices, normals, bottom, xn, zp)
	vertices, normals = appendTri(vertices, normals, bottom, zn, xn)
	vertices, normals = appendTri(vertices, normals, bottom, xp, zn)

	return vertices, normals
}

func (g *subnauticaGame) renderEditorLightMarkers(ctx *engine.Context, viewProj mgl32.Mat4, viewPos mgl32.Vec3) {
	if !g.editor.Enabled || g.editorLightMarkerMesh.VertexCount <= 0 {
		return
	}
	if !approxVec3(viewPos, ctx.Camera.Position, 0.0001) {
		return
	}
	if len(g.sceneLighting.PointLights) == 0 {
		return
	}

	lastVAO := uint32(0)
	ctx.Renderer.Use()
	ctx.Renderer.SetViewPos(viewPos)
	ctx.Renderer.SetMaterial(engine.Material{
		Ambient:          mgl32.Vec3{0.90, 0.90, 0.90},
		Diffuse:          mgl32.Vec3{0.25, 0.25, 0.25},
		Specular:         mgl32.Vec3{0.0, 0.0, 0.0},
		Emission:         mgl32.Vec3{0.28, 0.28, 0.28},
		Shininess:        1.0,
		SpecularStrength: 0.0,
	})
	ctx.Renderer.SetHeightGradient(mgl32.Vec3{1, 1, 1}, mgl32.Vec3{1, 1, 1}, 0, 1, 0)

	for i := range g.sceneLighting.PointLights {
		light := sanitizeScenePointLight(g.sceneLighting.PointLights[i])
		color := editorMarkerColorForLight(light)
		if i == g.editor.SelectedLightIndex {
			color = mgl32.Vec3{0.98, 0.92, 0.32}
		}
		if !light.Enabled || light.Intensity <= 0 {
			color = color.Mul(0.4)
		}

		scale := editorMarkerScaleForLight(light)
		yLift := scale * editorLightMarkerLiftMul
		model := mgl32.Translate3D(
			light.Position.X(),
			light.Position.Y()+yLift,
			light.Position.Z(),
		).Mul4(mgl32.Scale3D(scale, scale, scale))

		ctx.Renderer.SetObjectColor(color)
		ctx.Renderer.SetModel(model)
		ctx.Renderer.SetMVP(viewProj.Mul4(model))
		drawMeshWithCachedVAO(g.editorLightMarkerMesh, &lastVAO)
	}
}

func (g *subnauticaGame) renderEditorBrushPreview(ctx *engine.Context, viewProj mgl32.Mat4, viewPos mgl32.Vec3) {
	if !g.editor.Enabled || g.editor.PrimaryTool != editorPrimaryToolTerrain || !g.editor.HasTerrainHit || g.editorBrushMesh.VertexCount <= 0 {
		return
	}
	if !approxVec3(viewPos, ctx.Camera.Position, 0.0001) {
		return
	}

	color := editorToolColor(g.editor.Tool)
	strengthNorm := clamp01((g.editor.BrushStrength - editorBrushStrengthMin) / (editorBrushStrengthMax - editorBrushStrengthMin))
	color = color.Mul(0.72 + 0.28*strengthNorm)
	lastVAO := uint32(0)

	ctx.Renderer.Use()
	ctx.Renderer.SetViewPos(viewPos)
	ctx.Renderer.SetMaterial(engine.Material{
		Ambient:          mgl32.Vec3{0.92, 0.92, 0.92},
		Diffuse:          mgl32.Vec3{0.30, 0.30, 0.30},
		Specular:         mgl32.Vec3{0.0, 0.0, 0.0},
		Shininess:        1.0,
		SpecularStrength: 0.0,
	})
	ctx.Renderer.SetObjectColor(color)
	ctx.Renderer.SetHeightGradient(color, color, 0, 1, 0)
	ctx.Renderer.SetModel(mgl32.Ident4())
	ctx.Renderer.SetMVP(viewProj)
	drawMeshWithCachedVAO(g.editorBrushMesh, &lastVAO)
}

func editorToolColor(tool objects.TerrainBrushTool) mgl32.Vec3 {
	switch tool {
	case objects.TerrainBrushLower:
		return mgl32.Vec3{0.95, 0.34, 0.30}
	case objects.TerrainBrushSmooth:
		return mgl32.Vec3{0.35, 0.76, 0.98}
	case objects.TerrainBrushFlatten:
		return mgl32.Vec3{0.98, 0.82, 0.34}
	default:
		return mgl32.Vec3{0.34, 0.96, 0.50}
	}
}

func (g *subnauticaGame) renderEditorHUD(screenW, screenH int, timeSec float32) {
	if g.console == nil || g.console.renderer == nil {
		return
	}
	if !g.editor.Enabled && timeSec > g.editor.statusUntilSec {
		return
	}

	lines := make([]string, 0, 10)
	if g.editor.Enabled {
		g.ensureEditorLightSelectionValid()

		lines = append(lines, "EDITOR MODE: ON")
		lines = append(lines, "TOOL: "+g.editor.PrimaryTool.Label())
		if g.editor.WaterVisible {
			lines = append(lines, "WATER: ON")
		} else {
			lines = append(lines, "WATER: OFF")
		}
		if g.editor.HasTerrainHit {
			lines = append(lines, fmt.Sprintf(
				"HIT: %.1f %.1f %.1f",
				g.editor.HitPosition.X(),
				g.editor.HitPosition.Y(),
				g.editor.HitPosition.Z(),
			))
		} else {
			lines = append(lines, "HIT: NONE")
		}

		if g.editor.PrimaryTool == editorPrimaryToolLights {
			lines = append(lines, "PRESET: "+g.editor.LightPreset.Label())
			lines = append(lines, fmt.Sprintf(
				"RUNTIME POINT LIGHTS: %d/%d",
				g.editor.RuntimePointLightCount,
				engine.MaxLights-1,
			))
			if g.editor.SkippedPointLightCount > 0 {
				lines = append(lines, fmt.Sprintf(
					"OVER LIMIT (INACTIVE): %d",
					g.editor.SkippedPointLightCount,
				))
			}

			if g.editor.SelectedLightIndex >= 0 && g.editor.SelectedLightIndex < len(g.sceneLighting.PointLights) {
				selected := sanitizeScenePointLight(g.sceneLighting.PointLights[g.editor.SelectedLightIndex])
				lines = append(lines, fmt.Sprintf("SELECTED: #%d", g.editor.SelectedLightIndex))
				lines = append(lines, fmt.Sprintf("SEL RANGE: %.1f", selected.Range))
				lines = append(lines, fmt.Sprintf("SEL INTENSITY: %.2f", selected.Intensity))
			} else {
				lines = append(lines, "SELECTED: NONE")
			}

			lines = append(lines, "TAB TOGGLE  T TERRAIN  L LIGHTS")
			lines = append(lines, "V WATER TOGGLE  F OVERVIEW")
			lines = append(lines, "1 SMALL 2 MEDIUM 3 LARGE")
			lines = append(lines, "LMB SELECT/PLACE  CTRL+LMB MOVE")
			lines = append(lines, "DELETE REMOVE SELECTED")
			lines = append(lines, "CTRL S SAVE")
		} else {
			lines = append(lines, "BRUSH: "+g.editor.Tool.Label())
			lines = append(lines, fmt.Sprintf("RADIUS: %.2f", g.editor.BrushRadius))
			lines = append(lines, fmt.Sprintf("STRENGTH: %.2f", g.editor.BrushStrength))
			if g.editor.Tool == objects.TerrainBrushFlatten {
				if g.editor.flattenTargetLocked {
					lines = append(lines, fmt.Sprintf("FLATTEN TARGET: %.2f (LOCKED)", g.editor.flattenTargetHeight))
				} else {
					lines = append(lines, "FLATTEN TARGET: CLICK TO SAMPLE")
				}
			}
			lines = append(lines, "TAB TOGGLE  T TERRAIN  L LIGHTS")
			lines = append(lines, "V WATER TOGGLE  F OVERVIEW")
			lines = append(lines, "LMB APPLY  RMB LOOK")
			lines = append(lines, "1 RAISE 2 LOWER 3 SMOOTH 4 FLATTEN")
			lines = append(lines, "LBRACKET RBRACKET RADIUS")
			lines = append(lines, "MINUS EQUAL STRENGTH")
			lines = append(lines, "FLATTEN TARGET LOCKS ON LMB PRESS")
			lines = append(lines, "CTRL S SAVE")
		}
	} else {
		lines = append(lines, "EDITOR MODE: OFF")
	}

	if timeSec <= g.editor.statusUntilSec && g.editor.statusText != "" {
		lines = append(lines, g.editor.statusText)
	}

	scale := float32(2.0)
	margin := float32(14)
	paddingX := float32(9)
	paddingY := float32(8)
	lineHeight := 8 * scale

	maxTextWidth := float32(0)
	for i := range lines {
		w := measureOverlayTextWidth(lines[i], scale)
		if w > maxTextWidth {
			maxTextWidth = w
		}
	}

	panelW := maxTextWidth + paddingX*2
	panelH := float32(len(lines))*lineHeight + paddingY*2
	if panelW < 240 {
		panelW = 240
	}

	g.console.renderer.Begin(int32(screenW), int32(screenH))
	g.console.renderer.DrawRect(margin, margin, panelW, panelH, [4]float32{0.02, 0.03, 0.05, 0.76})
	g.console.renderer.DrawRect(margin, margin, panelW, 1, [4]float32{0.64, 0.74, 0.86, 0.72})
	y := margin + paddingY
	for i := range lines {
		color := [4]float32{0.90, 0.96, 1.0, 1.0}
		if i == 0 {
			color = [4]float32{0.96, 1.0, 0.92, 1.0}
		}
		g.console.renderer.DrawText(lines[i], margin+paddingX, y, scale, color)
		y += lineHeight
	}
	g.console.renderer.End()
}

func (g *subnauticaGame) enabledScenePointLightsCount() int {
	count := 0
	for i := range g.sceneLighting.PointLights {
		light := sanitizeScenePointLight(g.sceneLighting.PointLights[i])
		if light.Enabled && light.Intensity > 0 {
			count++
		}
	}
	return count
}

func (g *subnauticaGame) ensureEditorLightSelectionValid() {
	if g.editor.SelectedLightIndex < 0 {
		return
	}
	if g.editor.SelectedLightIndex >= len(g.sceneLighting.PointLights) {
		g.editor.SelectedLightIndex = -1
	}
}

func (g *subnauticaGame) pickEditorScenePointLight(rayOrigin, rayDir mgl32.Vec3) (int, float32, bool) {
	bestIndex := -1
	bestDistance := float32(editorRaycastMaxDistance)
	for i := range g.sceneLighting.PointLights {
		light := sanitizeScenePointLight(g.sceneLighting.PointLights[i])
		// Р СѓСЃСЃРєРёР№ РєРѕРјРјРµРЅС‚Р°СЂРёР№: РІС‹Р±РёСЂР°РµРј РјР°СЂРєРµСЂ РїРѕ Р±Р»РёР¶Р°Р№С€РµРјСѓ РїРµСЂРµСЃРµС‡РµРЅРёСЋ Р»СѓС‡Р° СЃРѕ СЃС„РµСЂРѕР№ РІРѕРєСЂСѓРі РёРЅРґРёРєР°С‚РѕСЂР°,
		// С‡С‚РѕР±С‹ selection РѕСЃС‚Р°РІР°Р»СЃСЏ СЃС‚Р°Р±РёР»СЊРЅС‹Рј РЅРµР·Р°РІРёСЃРёРјРѕ РѕС‚ РјР°СЃС€С‚Р°Р±Р° world-РіРµРѕРјРµС‚СЂРёРё.
		markerScale := editorMarkerScaleForLight(light)
		radius := markerScale * editorLightPickRadiusMul
		center := mgl32.Vec3{
			light.Position.X(),
			light.Position.Y() + markerScale*editorLightMarkerLiftMul,
			light.Position.Z(),
		}

		distance, ok := raySphereHitDistance(rayOrigin, rayDir, center, radius)
		if !ok || distance < 0 || distance > bestDistance {
			continue
		}

		bestDistance = distance
		bestIndex = i
	}

	if bestIndex < 0 {
		return -1, 0, false
	}
	return bestIndex, bestDistance, true
}

func raySphereHitDistance(rayOrigin, rayDir, sphereCenter mgl32.Vec3, radius float32) (float32, bool) {
	if radius <= 0 {
		return 0, false
	}
	oc := rayOrigin.Sub(sphereCenter)
	b := oc.Dot(rayDir)
	c := oc.Dot(oc) - radius*radius
	discriminant := b*b - c
	if discriminant < 0 {
		return 0, false
	}

	root := float32(math.Sqrt(float64(discriminant)))
	nearT := -b - root
	if nearT >= 0 {
		return nearT, true
	}
	farT := -b + root
	if farT >= 0 {
		return farT, true
	}
	return 0, false
}

func mouseButtonPressedOnce(window *glfw.Window, button glfw.MouseButton, wasDown *bool) bool {
	down := window.GetMouseButton(button) == glfw.Press
	pressed := down && !*wasDown
	*wasDown = down
	return pressed
}

func screenPointToWorldRay(
	mouseX, mouseY float64,
	screenW, screenH int,
	view, projection mgl32.Mat4,
) (mgl32.Vec3, mgl32.Vec3, bool) {
	if screenW <= 0 || screenH <= 0 {
		return mgl32.Vec3{}, mgl32.Vec3{}, false
	}

	x := float32((2.0*mouseX)/float64(screenW) - 1.0)
	y := float32(1.0 - (2.0*mouseY)/float64(screenH))

	invViewProj := projection.Mul4(view).Inv()
	nearClip := mgl32.Vec4{x, y, -1.0, 1.0}
	farClip := mgl32.Vec4{x, y, 1.0, 1.0}

	nearWorld, ok := vec4ProjectToVec3(invViewProj.Mul4x1(nearClip))
	if !ok {
		return mgl32.Vec3{}, mgl32.Vec3{}, false
	}
	farWorld, ok := vec4ProjectToVec3(invViewProj.Mul4x1(farClip))
	if !ok {
		return mgl32.Vec3{}, mgl32.Vec3{}, false
	}

	dir := farWorld.Sub(nearWorld)
	if dir.Len() < 0.0001 {
		return mgl32.Vec3{}, mgl32.Vec3{}, false
	}
	return nearWorld, dir.Normalize(), true
}

func vec4ProjectToVec3(v mgl32.Vec4) (mgl32.Vec3, bool) {
	if math.Abs(float64(v.W())) < 0.000001 {
		return mgl32.Vec3{}, false
	}
	invW := 1 / v.W()
	return mgl32.Vec3{v.X() * invW, v.Y() * invW, v.Z() * invW}, true
}
