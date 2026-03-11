package main

import (
	"fmt"
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
)

// Update вЂ” РіР»Р°РІРЅС‹Р№ per-frame Р°РїРґРµР№С‚ gameplay-Р»РѕРіРёРєРё.
// РљРѕРјР°РЅРґС‹ СЃРѕР±РёСЂР°СЋС‚СЃСЏ Р»РѕРєР°Р»СЊРЅРѕ, Р° РјСѓС‚Р°С†РёРё РїСЂРёРјРµРЅСЏСЋС‚СЃСЏ РІ authoritative simulation-СЃР»РѕРµ.
func (g *subnauticaGame) Update(ctx *engine.Context) error {
	g.engineCtx = ctx

	if g.fpsOverlay != nil {
		g.fpsOverlay.Update(ctx.DeltaTime)
	}

	g.handleQualityHotkeys(ctx)

	g.syncConsoleCapture(ctx)
	inputBlocked := g.console != nil && g.console.IsOpen()
	g.syncEditorCapture(ctx, inputBlocked)
	g.removeProceduralOffset(ctx)
	playerBaseStart := g.localPlayerPosition()

	editorActive := g.updateEditorMode(ctx, inputBlocked)
	g.syncEditorCapture(ctx, inputBlocked)
	if editorActive {
		g.localControl.SpaceWasDown = false
		g.localControl.LastVerticalAxis = 0
		g.queueCommand(simulationCommand{
			Kind: simulationCommandSyncPlayerFromCamera,
			SyncPlayerFromCamera: syncPlayerFromCameraCommand{
				PlayerID: g.authoritative.LocalPlayerID,
				Position: simVec3FromMgl(ctx.Camera.Position),
				LookDir:  simVec3FromMgl(sanitizeLookDir(ctx.Camera.Front, mgl32.Vec3{0, 0, -1})),
			},
		})
	} else {
		g.queueLocalPlayerMoveIntent(ctx, inputBlocked)
	}

	g.runSimulationFrame(ctx)

	ctx.Camera.Position = g.localPlayerPosition()
	if editorActive {
		g.localPresentation.ProceduralYOffset = 0
		g.localPresentation.UnderwaterIdleMix = 0
	} else {
		g.applyProceduralBobbing(ctx, g.localControl.LastVerticalAxis)
	}

	g.syncEnvironmentRuntimeState()
	if g.worldStreaming != nil {
		g.worldStreaming.UpdateStreaming(g.localPlayerPosition())
	}
	g.updateCreatures(ctx)

	physicalDelta := g.localPlayerPosition().Sub(playerBaseStart)
	movementSpeed := float32(0)
	if ctx.DeltaTime > 0 {
		movementSpeed = physicalDelta.Len() / ctx.DeltaTime
	}
	if g.firstPersonHands != nil {
		inWater := !g.localPlayerNoclip() && g.localPlayerWaterMode() != waterModeAboveWater
		g.firstPersonHands.Update(ctx.DeltaTime, ctx.Time, movementSpeed, inWater)
	}

	mode := "WALK"
	if g.editor.Enabled {
		mode = "EDITOR"
	} else if g.localPlayerNoclip() {
		mode = "NOCLIP"
	} else if !g.localPlayerOnGround() {
		mode = g.localPlayerWaterMode().Label()
	}
	if inputBlocked {
		mode += " + CONSOLE"
	}
	mode += " | Q:" + g.qualityPreset.Label()
	if mode != g.lastMode || ctx.Time >= g.nextTitleUpdateTime {
		mem := engine.GetMemStats()
		title := fmt.Sprintf("Subnautica Lite [%s] - %s", mode, mem.String())
		if title != g.lastWindowTitle {
			ctx.Window.SetTitle(title)
			g.lastWindowTitle = title
		}
		g.lastMode = mode
		g.nextTitleUpdateTime = ctx.Time + titleUpdateInterval
	}

	return nil
}

func (g *subnauticaGame) queueLocalPlayerMoveIntent(ctx *engine.Context, inputBlocked bool) {
	if ctx == nil || ctx.Window == nil {
		return
	}

	if inputBlocked {
		g.localControl.SpaceWasDown = false
		g.localControl.LastVerticalAxis = 0
		return
	}

	spaceDown := ctx.Window.GetKey(glfw.KeySpace) == glfw.Press
	ctrlDown := ctx.Window.GetKey(glfw.KeyLeftControl) == glfw.Press
	sprintDown := ctx.Window.GetKey(glfw.KeyLeftShift) == glfw.Press
	forwardAxis := movementAxis(
		ctx.Window.GetKey(glfw.KeyW) == glfw.Press,
		ctx.Window.GetKey(glfw.KeyS) == glfw.Press,
	)
	strafeAxis := movementAxis(
		ctx.Window.GetKey(glfw.KeyD) == glfw.Press,
		ctx.Window.GetKey(glfw.KeyA) == glfw.Press,
	)
	verticalAxis := movementAxis(spaceDown, ctrlDown)
	jumpPressed := spaceDown && !g.localControl.SpaceWasDown
	g.localControl.SpaceWasDown = spaceDown
	g.localControl.LastVerticalAxis = verticalAxis

	g.queueCommand(simulationCommand{
		Kind: simulationCommandMoveIntent,
		MoveIntent: moveIntentCommand{
			PlayerID:     g.authoritative.LocalPlayerID,
			ForwardAxis:  forwardAxis,
			StrafeAxis:   strafeAxis,
			VerticalAxis: verticalAxis,
			Sprint:       sprintDown,
			JumpPressed:  jumpPressed,
			LookDir:      simVec3FromMgl(sanitizeLookDir(ctx.Camera.Front, mgl32.Vec3{0, 0, -1})),
		},
	})
}

// classifyWaterMode РѕРїСЂРµРґРµР»СЏРµС‚ СЂРµР¶РёРј РёРіСЂРѕРєР° РѕС‚РЅРѕСЃРёС‚РµР»СЊРЅРѕ С‚РµРєСѓС‰РµРіРѕ СѓСЂРѕРІРЅСЏ РІРѕРґС‹.
func (g *subnauticaGame) classifyWaterMode(cameraY float32) waterMovementMode {
	return classifyWaterModeByLevel(cameraY, g.authoritative.WaterLevel)
}

func (g *subnauticaGame) removeProceduralOffset(ctx *engine.Context) {
	if g.localPresentation.ProceduralYOffset == 0 {
		return
	}
	ctx.Camera.Position[1] -= g.localPresentation.ProceduralYOffset
}

// applyProceduralBobbing РґРѕР±Р°РІР»СЏРµС‚ РјСЏРіРєРѕРµ РїСЂРѕС†РµРґСѓСЂРЅРѕРµ СЃРјРµС‰РµРЅРёРµ РєР°РјРµСЂС‹:
// РѕРґРёРЅ РїСЂРѕС„РёР»СЊ Сѓ РїРѕРІРµСЂС…РЅРѕСЃС‚Рё Рё РґСЂСѓРіРѕР№ вЂ” РІ РїРѕРґРІРѕРґРЅРѕРј idle.
func (g *subnauticaGame) applyProceduralBobbing(ctx *engine.Context, verticalInput float32) {
	player := g.localPlayerState()
	if player == nil {
		return
	}

	targetOffset := float32(0)

	if !player.OnGround && player.WaterMode == waterModeSurfaceFloating {
		inputFade := float32(1.0)
		if absf(verticalInput) > swimVerticalInputDeadzone {
			inputFade = 0.55
		}
		targetOffset = layeredBob(
			ctx.Time,
			surfaceBobAmplitude,
			surfaceBobSpeed,
			surfaceBobSecondaryAmplitude,
			surfaceBobSecondarySpeed,
		) * inputFade
		g.localPresentation.UnderwaterIdleMix = expSmoothing(
			g.localPresentation.UnderwaterIdleMix,
			0,
			underwaterIdleBlendSpeed,
			ctx.DeltaTime,
		)
	} else if !player.OnGround && player.WaterMode == waterModeUnderwaterSwimming {
		idleTarget := float32(1.0)
		if absf(verticalInput) > swimVerticalInputDeadzone {
			idleTarget = 0
		}
		velocityFade := clamp01(1.0 - absf(player.VerticalVelocity)/underwaterIdleVelocityThreshold)
		idleTarget *= velocityFade
		g.localPresentation.UnderwaterIdleMix = expSmoothing(
			g.localPresentation.UnderwaterIdleMix,
			idleTarget,
			underwaterIdleBlendSpeed,
			ctx.DeltaTime,
		)

		targetOffset = layeredBob(
			ctx.Time+31.7,
			underwaterIdleBobAmplitude,
			underwaterIdleBobSpeed,
			underwaterIdleBobSecondaryAmplitude,
			underwaterIdleBobSecondarySpeed,
		) * g.localPresentation.UnderwaterIdleMix
	} else {
		g.localPresentation.UnderwaterIdleMix = expSmoothing(
			g.localPresentation.UnderwaterIdleMix,
			0,
			underwaterIdleBlendSpeed,
			ctx.DeltaTime,
		)
	}

	g.localPresentation.ProceduralYOffset = expSmoothing(
		g.localPresentation.ProceduralYOffset,
		targetOffset,
		proceduralBobSmoothing,
		ctx.DeltaTime,
	)
	if absf(g.localPresentation.ProceduralYOffset) < 0.00001 {
		g.localPresentation.ProceduralYOffset = 0
	}
	ctx.Camera.Position[1] += g.localPresentation.ProceduralYOffset
}

func movementAxis(positive, negative bool) float32 {
	axis := float32(0)
	if positive {
		axis += 1
	}
	if negative {
		axis -= 1
	}
	return axis
}

func clampFloat(v, minV, maxV float32) float32 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func smoothStep(edge0, edge1, x float32) float32 {
	if edge0 == edge1 {
		return 0
	}
	t := clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3.0 - 2.0*t)
}

func expSmoothing(current, target, sharpness, dt float32) float32 {
	if dt <= 0 {
		return current
	}
	if sharpness <= 0 {
		return target
	}
	blend := 1 - float32(math.Exp(float64(-sharpness*dt)))
	return current + (target-current)*blend
}

func layeredBob(timeSec, ampA, speedA, ampB, speedB float32) float32 {
	phaseA := float32(math.Sin(float64(timeSec*speedA*2*math.Pi + 0.37)))
	phaseB := float32(math.Sin(float64(timeSec*speedB*2*math.Pi + 1.91)))
	return phaseA*ampA + phaseB*ampB
}

// syncSunLighting РїСЂРѕС‚Р°Р»РєРёРІР°РµС‚ scene-level РЅР°СЃС‚СЂРѕР№РєРё СЃРѕР»РЅС†Р° РІ directional light
// Рё СЃРёРЅС…СЂРѕРЅРЅРѕ РѕР±РЅРѕРІР»СЏРµС‚ sky/ocean, С‡С‚РѕР±С‹ РЅРµ Р±С‹Р»Рѕ СЂР°СЃС…РѕР¶РґРµРЅРёР№ РјРµР¶РґСѓ passes.
func (g *subnauticaGame) syncSunLighting() {
	if g.lightManager == nil {
		return
	}

	g.setSunLighting(
		g.sceneLighting.SunDirection,
		g.sceneLighting.SunColor,
		g.sceneLighting.SunIntensity,
	)

	sunLight := g.lightManager.Sun()
	if sunLight == nil {
		return
	}

	if g.oceanSystem != nil {
		params := g.skyParams
		params.SunDirection = sunLight.Direction
		params.SunColor = sunLight.Color
		params.SunIntensity = sunLight.Intensity
		g.skyParams = params
		g.oceanSystem.SetSkyAtmosphere(params)
	}
}

func keyPressedOnce(window *glfw.Window, key glfw.Key, wasDown *bool) bool {
	down := window.GetKey(key) == glfw.Press
	pressed := down && !*wasDown
	*wasDown = down
	return pressed
}

// handleQualityHotkeys РїРµСЂРµРєР»СЋС‡Р°РµС‚ preset РєР°С‡РµСЃС‚РІР° РїРѕ F1/F2/F3.
func (g *subnauticaGame) handleQualityHotkeys(ctx *engine.Context) {
	if ctx == nil {
		return
	}

	if keyPressedOnce(ctx.Window, glfw.KeyF1, &g.qualityLowKeyDown) {
		g.applyGraphicsPreset(ctx, engine.GraphicsQualityLow, true)
	}
	if keyPressedOnce(ctx.Window, glfw.KeyF2, &g.qualityMedKeyDown) {
		g.applyGraphicsPreset(ctx, engine.GraphicsQualityMedium, true)
	}
	if keyPressedOnce(ctx.Window, glfw.KeyF3, &g.qualityHighKeyDown) {
		g.applyGraphicsPreset(ctx, engine.GraphicsQualityHigh, true)
	}
}
