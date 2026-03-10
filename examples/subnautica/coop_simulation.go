package main

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

func (g *subnauticaGame) initCoopReadyState(spawnPos, lookDir mgl32.Vec3) {
	g.authoritative = newAuthoritativeState(spawnPos, lookDir)
	g.localControl = localControlState{}
	g.localPresentation = localPresentationState{}
	g.runtimeEntities = newRuntimeEntityBindings()
	g.pendingCommands = make([]simulationCommand, 0, 256)
	g.frameEvents = make([]simulationEvent, 0, 128)
}

func (g *subnauticaGame) queueInitialGameplaySpawns() {
	player := g.localPlayerState()
	if player == nil {
		return
	}

	g.queueCommand(simulationCommand{
		Kind: simulationCommandSpawnPlayerAvatar,
		SpawnPlayerAvatar: spawnPlayerAvatarCommand{
			PlayerID: player.PlayerID,
			Position: player.Position,
			YawDeg:   0,
			Scale:    1,
			Health:   player.Health,
			Oxygen:   player.Oxygen,
		},
	})

	g.queueCommand(simulationCommand{
		Kind: simulationCommandSpawnRandomGlowPlants,
		SpawnRandomGlowPlants: spawnRandomGlowPlantsCommand{
			Count:  numPlants,
			Radius: objects.GroundSpawnRadius(),
		},
	})
	g.queueCommand(simulationCommand{
		Kind: simulationCommandSpawnGlowFlower,
		SpawnGlowFlower: spawnGlowFlowerCommand{
			Position:   simVec3{X: 5, Y: g.groundHeightAt(5, 5), Z: 5},
			Color:      simVec3FromMgl(objects.GlowNeonPink),
			Intensity:  3.0,
			PulseSpeed: 2.5,
		},
	})
	g.queueCommand(simulationCommand{
		Kind: simulationCommandSpawnGlowFlower,
		SpawnGlowFlower: spawnGlowFlowerCommand{
			Position:   simVec3{X: -7, Y: g.groundHeightAt(-7, 3), Z: 3},
			Color:      simVec3FromMgl(objects.GlowNeonYellow),
			Intensity:  2.8,
			PulseSpeed: 3.0,
		},
	})
	g.queueCommand(simulationCommand{
		Kind: simulationCommandSpawnGlowFlower,
		SpawnGlowFlower: spawnGlowFlowerCommand{
			Position:   simVec3{X: 3, Y: g.groundHeightAt(3, -6), Z: -6},
			Color:      simVec3FromMgl(objects.GlowNeonCyan),
			Intensity:  3.2,
			PulseSpeed: 1.8,
		},
	})
	g.queueCommand(simulationCommand{
		Kind: simulationCommandSpawnGlowFlower,
		SpawnGlowFlower: spawnGlowFlowerCommand{
			Position:   simVec3{X: -4, Y: g.groundHeightAt(-4, -4), Z: -4},
			Color:      simVec3FromMgl(objects.GlowNeonPurple),
			Intensity:  2.5,
			PulseSpeed: 2.2,
		},
	})
	g.queueCommand(simulationCommand{
		Kind: simulationCommandSpawnGlowFlower,
		SpawnGlowFlower: spawnGlowFlowerCommand{
			Position:   simVec3{X: 8, Y: g.groundHeightAt(8, -2), Z: -2},
			Color:      simVec3FromMgl(objects.GlowNeonOrange),
			Intensity:  3.5,
			PulseSpeed: 2.8,
		},
	})

	g.queueCommand(simulationCommand{
		Kind: simulationCommandSpawnCreature,
		SpawnCreature: spawnCreatureCommand{
			TemplateID: creatureTemplateLeviathan,
			Position:   simVec3{X: 14, Y: -2.8, Z: -12},
			YawDeg:     -28,
			Scale:      3.2,
		},
	})
}

// runSimulationFrame — единая точка authoritative апдейта gameplay.
// Любые мутации мира должны проходить через команды и этот метод.
func (g *subnauticaGame) runSimulationFrame(ctx *engine.Context) {
	if ctx == nil {
		return
	}

	g.resetFrameEvents()
	g.authoritative.Tick++
	g.authoritative.TimeSec = ctx.Time
	g.authoritative.WaterLevel = waterSurfaceY + float32(math.Sin(float64(ctx.Time*waterLevelWaveSpeed)))*waterLevelWaveAmplitude

	dt := ctx.DeltaTime
	if dt < 0 {
		dt = 0
	}

	for i := range g.pendingCommands {
		g.applySimulationCommand(g.pendingCommands[i], dt)
	}
	g.pendingCommands = g.pendingCommands[:0]

	g.refreshDerivedPlayerState()
	g.syncPlayerAvatarTransform()
}

func (g *subnauticaGame) refreshDerivedPlayerState() {
	player := g.localPlayerState()
	if player == nil {
		return
	}
	player.WaterMode = classifyWaterModeByLevel(player.Position.Y, g.authoritative.WaterLevel)
}

func (g *subnauticaGame) applySimulationCommand(command simulationCommand, dt float32) {
	switch command.Kind {
	case simulationCommandMoveIntent:
		g.applyMoveIntentCommand(command.MoveIntent, dt)
	case simulationCommandToggleNoclip:
		g.applyToggleNoclipCommand(command.ToggleNoclip)
	case simulationCommandSpawnPlayerAvatar:
		g.applySpawnPlayerAvatarCommand(command.SpawnPlayerAvatar)
	case simulationCommandSpawnRandomGlowPlants:
		g.applySpawnRandomGlowPlantsCommand(command.SpawnRandomGlowPlants)
	case simulationCommandSpawnGlowFlower:
		g.applySpawnGlowFlowerCommand(command.SpawnGlowFlower)
	case simulationCommandSpawnCreature:
		g.applySpawnCreatureCommand(command.SpawnCreature)
	case simulationCommandDespawnEntity:
		g.applyDespawnEntityCommand(command.DespawnEntity)
	case simulationCommandApplyTerrainBrush:
		g.applyTerrainBrushCommand(command.TerrainBrush)
	case simulationCommandSyncPlayerFromCamera:
		g.applySyncPlayerFromCameraCommand(command.SyncPlayerFromCamera)
	case simulationCommandPlaceScenePointLight:
		g.applyPlaceScenePointLightCommand(command.PlaceScenePointLight)
	case simulationCommandRemoveScenePointLight:
		g.applyRemoveScenePointLightCommand(command.RemoveScenePointLight)
	case simulationCommandMoveScenePointLight:
		g.applyMoveScenePointLightCommand(command.MoveScenePointLight)
	}
}

func (g *subnauticaGame) applyMoveIntentCommand(command moveIntentCommand, dt float32) {
	player, ok := g.authoritative.Players[command.PlayerID]
	if !ok || player == nil || dt <= 0 {
		return
	}

	player.LookDir = command.LookDir.NormalizedOr(player.LookDir)
	look := player.LookDir.Mgl()
	up := mgl32.Vec3{0, 1, 0}
	right := look.Cross(up)
	if right.Len() < 0.0001 {
		right = mgl32.Vec3{1, 0, 0}
	} else {
		right = right.Normalize()
	}

	forwardAxis := clampFloat(command.ForwardAxis, -1, 1)
	strafeAxis := clampFloat(command.StrafeAxis, -1, 1)
	verticalAxis := clampFloat(command.VerticalAxis, -1, 1)

	if player.Noclip {
		speed := float32(noclipMoveSpeed)
		if command.Sprint {
			speed *= float32(noclipSprintMul)
		}

		move := look.Mul(forwardAxis).Add(right.Mul(strafeAxis))
		if move.Len() > 1 {
			move = move.Normalize()
		}
		move = move.Add(up.Mul(verticalAxis))
		if move.Len() > 1 {
			move = move.Normalize()
		}

		nextPos := player.Position.Mgl().Add(move.Mul(speed * dt))
		player.Position = simVec3FromMgl(nextPos)
		player.OnGround = false
		player.VerticalVelocity = 0
		player.WaterMode = classifyWaterModeByLevel(nextPos.Y(), g.authoritative.WaterLevel)
		return
	}

	speed := float32(baseMoveSpeed)
	if command.Sprint {
		speed *= sprintFactor
	}

	position := player.Position.Mgl()

	if player.OnGround {
		flatForward := mgl32.Vec3{look.X(), 0, look.Z()}
		if flatForward.Len() < 0.0001 {
			flatForward = mgl32.Vec3{0, 0, -1}
		} else {
			flatForward = flatForward.Normalize()
		}
		flatRight := flatForward.Cross(up)
		if flatRight.Len() < 0.0001 {
			flatRight = mgl32.Vec3{1, 0, 0}
		} else {
			flatRight = flatRight.Normalize()
		}

		move := flatForward.Mul(forwardAxis).Add(flatRight.Mul(strafeAxis))
		if move.Len() > 1 {
			move = move.Normalize()
		}
		position = position.Add(move.Mul(speed * dt))

		if command.JumpPressed {
			player.VerticalVelocity = jumpImpulse
			player.OnGround = false
		} else {
			groundY := g.groundHeightAt(position.X(), position.Z())
			position[1] = groundY + playerEyeHeight
			player.VerticalVelocity = 0
		}
	} else {
		move := look.Mul(forwardAxis).Add(right.Mul(strafeAxis))
		if move.Len() > 1 {
			move = move.Normalize()
		}
		position = position.Add(move.Mul(speed * dt))
		player.Position = simVec3FromMgl(position)
		g.updateSwimVertical(player, verticalAxis, dt)
		position = player.Position.Mgl()

		groundY := g.groundHeightAt(position.X(), position.Z())
		targetY := groundY + playerEyeHeight
		if position.Y() <= targetY && player.VerticalVelocity <= 0 {
			position[1] = targetY
			player.VerticalVelocity = 0
			player.OnGround = true
		}
	}

	player.Position = simVec3FromMgl(position)
	player.WaterMode = classifyWaterModeByLevel(position.Y(), g.authoritative.WaterLevel)
}

func (g *subnauticaGame) updateSwimVertical(player *playerAuthoritativeState, verticalInput, dt float32) {
	if player == nil || dt <= 0 {
		return
	}

	y := player.Position.Y
	waterLevel := g.authoritative.WaterLevel
	inputActive := absf(verticalInput) > swimVerticalInputDeadzone

	accel := float32(-underwaterG)
	accel -= player.VerticalVelocity * waterDrag

	inputAccel := verticalInput * swimVerticalAccel
	if verticalInput > swimVerticalInputDeadzone {
		resistance := smoothStep(waterLevel-surfaceBandThickness, waterLevel+maxRiseAboveSurface, y)
		inputAccel *= 1.0 - surfaceUpInputResistance*resistance
	}
	accel += inputAccel

	if y < waterLevel-surfaceBandThickness && !inputActive {
		accel += underwaterNeutralBuoyancy
		accel -= player.VerticalVelocity * underwaterIdleVelocityDamping
	}

	surfaceInfluenceStart := waterLevel - surfaceBandThickness - surfaceInfluenceExtra
	if y >= surfaceInfluenceStart {
		targetSurfaceY := waterLevel - surfaceFloatDepth
		errorToSurface := targetSurfaceY - y
		influence := smoothStep(surfaceInfluenceStart, waterLevel+maxRiseAboveSurface, y)
		scaledInfluence := 0.35 + 0.65*influence
		accel += errorToSurface * buoyancyStrength * scaledInfluence
		accel -= player.VerticalVelocity * surfaceBuoyancyDamp * scaledInfluence
	}

	upperSoftLimit := waterLevel + maxRiseAboveSurface
	if y > upperSoftLimit {
		overshoot := y - upperSoftLimit
		accel -= overshoot * buoyancyStrength * surfaceOvershootPullFactor
	}

	player.VerticalVelocity += accel * dt
	player.VerticalVelocity = clampFloat(player.VerticalVelocity, -swimVerticalMaxDiveSpeed, swimVerticalMaxRiseSpeed)
	player.Position.Y += player.VerticalVelocity * dt
}

func (g *subnauticaGame) applyToggleNoclipCommand(command toggleNoclipCommand) {
	player, ok := g.authoritative.Players[command.PlayerID]
	if !ok || player == nil {
		return
	}

	player.Noclip = command.Enabled
	player.OnGround = false
	player.VerticalVelocity = 0
	player.WaterMode = classifyWaterModeByLevel(player.Position.Y, g.authoritative.WaterLevel)

	g.localControl.SpaceWasDown = false
	g.localPresentation.UnderwaterIdleMix = 0
	g.localPresentation.ProceduralYOffset = 0

	g.emitEvent(simulationEvent{
		Kind:     simulationEventPlayerModeChanged,
		PlayerID: command.PlayerID,
	})
}

func (g *subnauticaGame) applySpawnPlayerAvatarCommand(command spawnPlayerAvatarCommand) {
	if g.world == nil {
		return
	}

	player, ok := g.authoritative.Players[command.PlayerID]
	if !ok || player == nil {
		return
	}

	runtimeEntity := objects.SpawnPlayerAvatar(g.world, objects.PlayerAvatarSpawnConfig{
		PlayerID: uint32(command.PlayerID),
		Position: command.Position.Mgl(),
		YawDeg:   command.YawDeg,
		Scale:    command.Scale,
		Health:   command.Health,
		Oxygen:   command.Oxygen,
	})
	id := g.ensureEntityMeta(runtimeEntity, entityKindPlayerAvatar, command.PlayerID, true, replicationScopeGlobal)
	player.ControlledEntityID = id
	player.Position = command.Position
}

func (g *subnauticaGame) applySpawnRandomGlowPlantsCommand(command spawnRandomGlowPlantsCommand) {
	if g.world == nil || command.Count <= 0 || command.Radius <= 0 {
		return
	}

	objects.SpawnRandomGlowingPlantsWithHeightFunc(g.world, command.Count, command.Radius, g.groundHeightAt)
	g.buildGlowPlantInstances()
	if g.worldStreaming != nil {
		g.worldStreaming.AssignGlowPlants(g.glowPlantInstances)
	}
}

func (g *subnauticaGame) applySpawnGlowFlowerCommand(command spawnGlowFlowerCommand) {
	if g.world == nil {
		return
	}

	runtimeEntity := objects.SpawnGlowingFlower(
		g.world,
		command.Position.Mgl(),
		command.Color.Mgl(),
		command.Intensity,
		command.PulseSpeed,
	)
	g.ensureEntityMeta(runtimeEntity, entityKindGlowPlant, playerIDInvalid, false, replicationScopeSpatial)
	g.buildGlowPlantInstances()
	if g.worldStreaming != nil {
		g.worldStreaming.AssignGlowPlants(g.glowPlantInstances)
	}
}

func (g *subnauticaGame) applySpawnCreatureCommand(command spawnCreatureCommand) {
	if g.world == nil || g.creatureManager == nil {
		return
	}

	switch command.TemplateID {
	case creatureTemplateLeviathan:
		movement := objects.DefaultPredatorMovementSettings()
		movement.HabitatRadius = 36
		movement.MinDepthBelowSurface = 2.1
		movement.MaxDepthBelowSurface = 10.8
		movement.SurfaceClearance = 1.4
		movement.FloorClearance = 1.5
		movement.DetectionRadius = 18
		movement.TooCloseRadius = 6
		movement.VerticalDriftAmplitude = 0.42
		movement.VerticalDriftFrequency = 0.22

		spawn := command.Position.Mgl()
		groundY := g.groundHeightAt(spawn.X(), spawn.Z())
		minY := groundY + movement.FloorClearance
		maxY := g.authoritative.WaterLevel - movement.MinDepthBelowSurface
		spawn[1] = clampFloat(spawn.Y(), minY, maxY)

		runtimeEntity := g.creatureManager.Spawn(objects.CreatureSpawnConfig{
			Species:       "Leviathan",
			Style:         objects.CreatureStylePredator,
			Position:      spawn,
			YawDeg:        command.YawDeg,
			Scale:         command.Scale,
			HabitatCenter: mgl32.Vec3{0, spawn.Y(), 0},
			Movement:      movement,
			Visual: objects.CreatureVisual{
				ModelID:        creatureModelLeviathan,
				Color:          mgl32.Vec3{0.84, 0.89, 0.96},
				Material:       engine.MaterialPlastic,
				YawOffsetDeg:   0,
				BoundingRadius: 5.2,
			},
		})
		g.ensureEntityMeta(runtimeEntity, entityKindCreature, playerIDInvalid, true, replicationScopeSpatial)
	}
}

func (g *subnauticaGame) applyDespawnEntityCommand(command despawnEntityCommand) {
	if command.EntityID == 0 {
		return
	}

	if runtimeEntity, ok := g.runtimeEntityByID(command.EntityID); ok {
		if g.world != nil {
			g.world.DestroyEntity(runtimeEntity)
		}
		delete(g.runtimeEntities.byRuntime, runtimeEntity)
	}
	delete(g.runtimeEntities.byID, command.EntityID)
	delete(g.authoritative.Entities, command.EntityID)

	player := g.localPlayerState()
	if player != nil && player.ControlledEntityID == command.EntityID {
		player.ControlledEntityID = 0
	}

	g.emitEvent(simulationEvent{
		Kind:     simulationEventEntityDespawned,
		EntityID: command.EntityID,
	})
}

func (g *subnauticaGame) applyTerrainBrushCommand(command terrainBrushCommand) {
	if g.terrain == nil {
		return
	}

	radius := clampFloat(command.Radius, editorBrushRadiusMin, editorBrushRadiusMax)
	strength := clampFloat(command.Strength, editorBrushStrengthMin, editorBrushStrengthMax)
	brushDeltaTime := clampFloat(command.DeltaTime, editorBrushStrokeDeltaTimeMin, editorBrushStrokeDeltaTimeMax)
	if brushDeltaTime <= 0 {
		return
	}
	// Русский комментарий: flatten всегда должен получать валидную опорную высоту.
	flattenTarget := command.FlattenTarget
	if command.Tool != objects.TerrainBrushFlatten {
		flattenTarget = command.Center.Y
	}

	changed := g.terrain.ApplyBrush(
		command.Center.X,
		command.Center.Z,
		radius,
		strength,
		brushDeltaTime,
		command.Tool,
		flattenTarget,
	)
	if !changed {
		return
	}

	dirtyPadding := g.terrain.CellSize * 2
	dirtyRadius := radius + dirtyPadding
	g.rebuildTerrainMeshInRange(
		command.Center.X-dirtyRadius,
		command.Center.X+dirtyRadius,
		command.Center.Z-dirtyRadius,
		command.Center.Z+dirtyRadius,
	)
	g.snapGlowingPlantsToTerrain()
	g.emitEvent(simulationEvent{
		Kind:    simulationEventTerrainChanged,
		Message: "terrain modified",
	})
}

func (g *subnauticaGame) applySyncPlayerFromCameraCommand(command syncPlayerFromCameraCommand) {
	player, ok := g.authoritative.Players[command.PlayerID]
	if !ok || player == nil {
		return
	}

	player.Position = command.Position
	player.LookDir = command.LookDir.NormalizedOr(player.LookDir)
	player.VerticalVelocity = 0
	player.OnGround = false
	player.WaterMode = classifyWaterModeByLevel(command.Position.Y, g.authoritative.WaterLevel)
}

func (g *subnauticaGame) applyPlaceScenePointLightCommand(command placeScenePointLightCommand) {
	// Русский комментарий: редактор отправляет только preset + позицию, а итоговые range/intensity/color
	// нормализуются в scene-слое и затем проходят через единый lightManager.
	light := scenePointLightFromPreset(command.Position.Mgl(), command.Preset)
	g.sceneLighting.PointLights = append(g.sceneLighting.PointLights, light)
	g.applySceneLighting()
	g.emitEvent(simulationEvent{
		Kind:    simulationEventSceneLightingChanged,
		Message: "scene point light placed",
	})
}

func (g *subnauticaGame) applyRemoveScenePointLightCommand(command removeScenePointLightCommand) {
	index := command.LightIndex
	if index < 0 || index >= len(g.sceneLighting.PointLights) {
		return
	}

	g.sceneLighting.PointLights = append(
		g.sceneLighting.PointLights[:index],
		g.sceneLighting.PointLights[index+1:]...,
	)
	g.applySceneLighting()
	g.emitEvent(simulationEvent{
		Kind:    simulationEventSceneLightingChanged,
		Message: "scene point light removed",
	})
}

func (g *subnauticaGame) applyMoveScenePointLightCommand(command moveScenePointLightCommand) {
	index := command.LightIndex
	if index < 0 || index >= len(g.sceneLighting.PointLights) {
		return
	}

	light := sanitizeScenePointLight(g.sceneLighting.PointLights[index])
	light.Position = command.Position.Mgl()
	g.sceneLighting.PointLights[index] = light
	g.applySceneLighting()
	g.emitEvent(simulationEvent{
		Kind:    simulationEventSceneLightingChanged,
		Message: "scene point light moved",
	})
}

func sanitizeLookDir(dir mgl32.Vec3, fallback mgl32.Vec3) mgl32.Vec3 {
	if dir.Len() > 0.0001 {
		return dir.Normalize()
	}
	if fallback.Len() > 0.0001 {
		return fallback.Normalize()
	}
	return mgl32.Vec3{0, 0, -1}
}

func classifyWaterModeByLevel(y, waterLevel float32) waterMovementMode {
	if y > waterLevel+maxRiseAboveSurface {
		return waterModeAboveWater
	}
	if y >= waterLevel-surfaceBandThickness {
		return waterModeSurfaceFloating
	}
	return waterModeUnderwaterSwimming
}
