package main

import "subnautica-lite/objects"

type simulationCommandKind uint8

const (
	simulationCommandMoveIntent simulationCommandKind = iota
	simulationCommandToggleNoclip
	simulationCommandSpawnPlayerAvatar
	simulationCommandSpawnRandomGlowPlants
	simulationCommandSpawnGlowFlower
	simulationCommandSpawnCreature
	simulationCommandDespawnEntity
	simulationCommandApplyTerrainBrush
	simulationCommandSyncPlayerFromCamera
	simulationCommandPlaceScenePointLight
	simulationCommandRemoveScenePointLight
	simulationCommandMoveScenePointLight
)

// moveIntentCommand — входной intent игрока.
// Команда не мутирует мир сама по себе: мутация происходит только в симуляции.
type moveIntentCommand struct {
	PlayerID     PlayerID
	ForwardAxis  float32
	StrafeAxis   float32
	VerticalAxis float32
	Sprint       bool
	JumpPressed  bool
	LookDir      simVec3
}

type toggleNoclipCommand struct {
	PlayerID PlayerID
	Enabled  bool
}

type spawnPlayerAvatarCommand struct {
	PlayerID PlayerID
	Position simVec3
	YawDeg   float32
	Scale    float32
	Health   float32
	Oxygen   float32
}

type spawnRandomGlowPlantsCommand struct {
	Count  int
	Radius float32
}

type spawnGlowFlowerCommand struct {
	Position   simVec3
	Color      simVec3
	Intensity  float32
	PulseSpeed float32
}

type spawnCreatureCommand struct {
	TemplateID string
	Position   simVec3
	YawDeg     float32
	Scale      float32
}

type despawnEntityCommand struct {
	EntityID EntityID
}

type terrainBrushCommand struct {
	Center    simVec3
	Radius    float32
	Strength  float32
	DeltaTime float32
	Tool      objects.TerrainBrushTool
	// FlattenTarget используется только в режиме TerrainBrushFlatten.
	FlattenTarget float32
}

type syncPlayerFromCameraCommand struct {
	PlayerID PlayerID
	Position simVec3
	LookDir  simVec3
}

type placeScenePointLightCommand struct {
	Position simVec3
	Preset   localLightPreset
}

type removeScenePointLightCommand struct {
	LightIndex int
}

type moveScenePointLightCommand struct {
	LightIndex int
	Position   simVec3
}

type simulationCommand struct {
	Kind simulationCommandKind

	MoveIntent            moveIntentCommand
	ToggleNoclip          toggleNoclipCommand
	SpawnPlayerAvatar     spawnPlayerAvatarCommand
	SpawnRandomGlowPlants spawnRandomGlowPlantsCommand
	SpawnGlowFlower       spawnGlowFlowerCommand
	SpawnCreature         spawnCreatureCommand
	DespawnEntity         despawnEntityCommand
	TerrainBrush          terrainBrushCommand
	SyncPlayerFromCamera  syncPlayerFromCameraCommand
	PlaceScenePointLight  placeScenePointLightCommand
	RemoveScenePointLight removeScenePointLightCommand
	MoveScenePointLight   moveScenePointLightCommand
}

type simulationEventKind uint8

const (
	simulationEventEntitySpawned simulationEventKind = iota
	simulationEventEntityDespawned
	simulationEventPlayerModeChanged
	simulationEventTerrainChanged
	simulationEventSceneLightingChanged
)

type simulationEvent struct {
	Kind     simulationEventKind
	PlayerID PlayerID
	EntityID EntityID
	Message  string
}

func (g *subnauticaGame) queueCommand(command simulationCommand) {
	g.pendingCommands = append(g.pendingCommands, command)
}

func (g *subnauticaGame) resetFrameEvents() {
	g.frameEvents = g.frameEvents[:0]
}

func (g *subnauticaGame) emitEvent(event simulationEvent) {
	g.frameEvents = append(g.frameEvents, event)
}
