package main

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

// PlayerID — стабильный идентификатор игрока в сессии.
// Даже в singleplayer используется явно, чтобы не зашивать "только один игрок".
type PlayerID uint32

const (
	playerIDInvalid PlayerID = 0
	playerIDLocal   PlayerID = 1
)

// EntityID — стабильный идентификатор gameplay-сущности для будущей репликации.
type EntityID uint64

type entityKind uint8

const (
	entityKindUnknown entityKind = iota
	entityKindPlayerAvatar
	entityKindCreature
	entityKindGlowPlant
)

type entityAuthority uint8

const (
	entityAuthorityHost entityAuthority = iota
	entityAuthorityOwner
)

type replicationScope uint8

const (
	replicationScopeGlobal replicationScope = iota
	replicationScopeSpatial
	replicationScopeLocalOnly
)

type sessionMode uint8

const (
	sessionModeSingleplayer sessionMode = iota
	sessionModeHost
	sessionModeClient
)

// simVec3 — сериализуемый вектор без runtime-методов.
// В authoritative слое используем его вместо прямого хранения mgl32.Vec3.
type simVec3 struct {
	X float32
	Y float32
	Z float32
}

func simVec3FromMgl(v mgl32.Vec3) simVec3 {
	return simVec3{X: v.X(), Y: v.Y(), Z: v.Z()}
}

func (v simVec3) Mgl() mgl32.Vec3 {
	return mgl32.Vec3{v.X, v.Y, v.Z}
}

func (v simVec3) Len() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}

func (v simVec3) NormalizedOr(fallback simVec3) simVec3 {
	length := v.Len()
	if length <= 0.0001 {
		return fallback
	}
	inv := 1.0 / length
	return simVec3{
		X: v.X * inv,
		Y: v.Y * inv,
		Z: v.Z * inv,
	}
}

// playerAuthoritativeState — gameplay-состояние игрока.
// Это authoritative данные, которые в будущем будут сниматься в snapshot/replication.
type playerAuthoritativeState struct {
	PlayerID           PlayerID
	ControlledEntityID EntityID

	Position         simVec3
	LookDir          simVec3
	VerticalVelocity float32
	OnGround         bool
	WaterMode        waterMovementMode
	Noclip           bool

	Health float32
	Oxygen float32
}

// entityMeta описывает сетевую/владельческую роль сущности.
// Структура не содержит GL-объектов и подходит для сериализации.
type entityMeta struct {
	ID        EntityID
	Kind      entityKind
	Owner     PlayerID
	Auth      entityAuthority
	Scope     replicationScope
	Dynamic   bool
	SpawnTick uint64
}

// authoritativeState — централизованное состояние симуляции.
// Здесь только gameplay-данные, без рендера, UI и GLFW.
type authoritativeState struct {
	Mode sessionMode
	Tick uint64

	TimeSec    float32
	WaterLevel float32

	HostPlayerID  PlayerID
	LocalPlayerID PlayerID

	Players  map[PlayerID]*playerAuthoritativeState
	Entities map[EntityID]entityMeta
}

func newAuthoritativeState(spawnPos, lookDir mgl32.Vec3) authoritativeState {
	local := &playerAuthoritativeState{
		PlayerID:           playerIDLocal,
		ControlledEntityID: 0,
		Position:           simVec3FromMgl(spawnPos),
		LookDir:            simVec3FromMgl(sanitizeLookDir(lookDir, mgl32.Vec3{0, 0, -1})),
		VerticalVelocity:   0,
		OnGround:           true,
		WaterMode:          waterModeUnderwaterSwimming,
		Noclip:             false,
		Health:             100,
		Oxygen:             100,
	}

	return authoritativeState{
		Mode:          sessionModeSingleplayer,
		Tick:          0,
		TimeSec:       0,
		WaterLevel:    waterSurfaceY,
		HostPlayerID:  playerIDLocal,
		LocalPlayerID: playerIDLocal,
		Players: map[PlayerID]*playerAuthoritativeState{
			playerIDLocal: local,
		},
		Entities: make(map[EntityID]entityMeta, 128),
	}
}

func (s *authoritativeState) localPlayer() *playerAuthoritativeState {
	if s == nil || s.Players == nil {
		return nil
	}
	return s.Players[s.LocalPlayerID]
}

// localControlState — локально-предсказанное состояние ввода.
// Это не authoritative данные, но они помогают собирать команды в frame-loop.
type localControlState struct {
	SpaceWasDown     bool
	LastVerticalAxis float32
}

// localPresentationState — чисто визуальные transient-данные.
// Их можно сбрасывать/пересчитывать без влияния на gameplay.
type localPresentationState struct {
	ProceduralYOffset float32
	UnderwaterIdleMix float32
}

// runtimeEntityBindings связывает runtime ECS entity и стабильный EntityID.
// Это seam между "живым" миром движка и сериализуемым authoritative слоем.
type runtimeEntityBindings struct {
	nextID    EntityID
	byRuntime map[engine.Entity]EntityID
	byID      map[EntityID]engine.Entity
}

func newRuntimeEntityBindings() runtimeEntityBindings {
	return runtimeEntityBindings{
		nextID:    1,
		byRuntime: make(map[engine.Entity]EntityID, 128),
		byID:      make(map[EntityID]engine.Entity, 128),
	}
}

func (g *subnauticaGame) localPlayerState() *playerAuthoritativeState {
	return g.authoritative.localPlayer()
}

func (g *subnauticaGame) localPlayerPosition() mgl32.Vec3 {
	player := g.localPlayerState()
	if player == nil {
		return mgl32.Vec3{}
	}
	return player.Position.Mgl()
}

func (g *subnauticaGame) localPlayerNoclip() bool {
	player := g.localPlayerState()
	if player == nil {
		return false
	}
	return player.Noclip
}

func (g *subnauticaGame) localPlayerOnGround() bool {
	player := g.localPlayerState()
	if player == nil {
		return false
	}
	return player.OnGround
}

func (g *subnauticaGame) localPlayerWaterMode() waterMovementMode {
	player := g.localPlayerState()
	if player == nil {
		return waterModeUnderwaterSwimming
	}
	return player.WaterMode
}

func (g *subnauticaGame) localPlayerVerticalVelocity() float32 {
	player := g.localPlayerState()
	if player == nil {
		return 0
	}
	return player.VerticalVelocity
}

func (g *subnauticaGame) setLocalPlayerPosition(position mgl32.Vec3) {
	player := g.localPlayerState()
	if player == nil {
		return
	}
	player.Position = simVec3FromMgl(position)
}

func (g *subnauticaGame) ensureEntityMeta(
	runtimeEntity engine.Entity,
	kind entityKind,
	owner PlayerID,
	dynamic bool,
	scope replicationScope,
) EntityID {
	if id, ok := g.runtimeEntities.byRuntime[runtimeEntity]; ok {
		meta := g.authoritative.Entities[id]
		if meta.ID == 0 {
			meta = entityMeta{
				ID:        id,
				Kind:      kind,
				Owner:     owner,
				Auth:      entityAuthorityHost,
				Scope:     scope,
				Dynamic:   dynamic,
				SpawnTick: g.authoritative.Tick,
			}
		} else {
			meta.Kind = kind
			meta.Owner = owner
			meta.Scope = scope
			meta.Dynamic = dynamic
		}
		g.authoritative.Entities[id] = meta
		return id
	}

	id := g.runtimeEntities.nextID
	g.runtimeEntities.nextID++
	g.runtimeEntities.byRuntime[runtimeEntity] = id
	g.runtimeEntities.byID[id] = runtimeEntity
	g.authoritative.Entities[id] = entityMeta{
		ID:        id,
		Kind:      kind,
		Owner:     owner,
		Auth:      entityAuthorityHost,
		Scope:     scope,
		Dynamic:   dynamic,
		SpawnTick: g.authoritative.Tick,
	}

	g.emitEvent(simulationEvent{
		Kind:     simulationEventEntitySpawned,
		EntityID: id,
		PlayerID: owner,
	})
	return id
}

func (g *subnauticaGame) runtimeEntityByID(id EntityID) (engine.Entity, bool) {
	runtimeEntity, ok := g.runtimeEntities.byID[id]
	return runtimeEntity, ok
}

func (g *subnauticaGame) syncPlayerAvatarTransform() {
	player := g.localPlayerState()
	if player == nil || g.world == nil || player.ControlledEntityID == 0 {
		return
	}

	runtimeEntity, ok := g.runtimeEntityByID(player.ControlledEntityID)
	if !ok {
		return
	}

	transform, ok := engine.GetComponent[objects.Transform](g.world, runtimeEntity)
	if !ok {
		return
	}

	transform.Position = player.Position.Mgl()
	look := player.LookDir.Mgl()
	if look.Len() > 0.0001 {
		transform.Yaw = float32(math.Atan2(float64(look.X()), float64(look.Z())) * (180.0 / math.Pi))
	}
	if transform.Scale <= 0 {
		transform.Scale = 1
	}

	avatar, ok := engine.GetComponent[objects.PlayerAvatar](g.world, runtimeEntity)
	if !ok {
		return
	}
	avatar.Health = player.Health
	avatar.Oxygen = player.Oxygen
}
