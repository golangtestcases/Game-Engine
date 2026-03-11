package main

import (
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

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
		g.resetEditorTerrainDragState()
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

	saveDown := editorControlDown(ctx.Window) && ctx.Window.GetKey(glfw.KeyS) == glfw.Press
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
		// Fix minimum overview camera height above terrain to avoid spawning below steep slopes.
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
	g.clearEditorRayAndHit()
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

func editorControlDown(window *glfw.Window) bool {
	return window.GetKey(glfw.KeyLeftControl) == glfw.Press || window.GetKey(glfw.KeyRightControl) == glfw.Press
}
