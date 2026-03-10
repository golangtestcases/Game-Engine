package objects

import (
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
)

// PlayerAvatar — компонент авторитетного состояния игрока в ECS.
// Хранит только сериализуемые gameplay-данные (без ссылок на рендер/GL).
type PlayerAvatar struct {
	PlayerID uint32
	Health   float32
	Oxygen   float32
}

// PlayerAvatarSpawnConfig описывает параметры спавна сущности игрока.
type PlayerAvatarSpawnConfig struct {
	PlayerID uint32
	Position mgl32.Vec3
	YawDeg   float32
	Scale    float32
	Health   float32
	Oxygen   float32
}

// SpawnPlayerAvatar создает ECS-сущность игрока (Transform + PlayerAvatar).
func SpawnPlayerAvatar(world *engine.World, cfg PlayerAvatarSpawnConfig) engine.Entity {
	if world == nil {
		panic("spawn player avatar: world is nil")
	}

	scale := cfg.Scale
	if scale <= 0 {
		scale = 1
	}

	health := cfg.Health
	if health <= 0 {
		health = 100
	}
	oxygen := cfg.Oxygen
	if oxygen <= 0 {
		oxygen = 100
	}

	entity := world.CreateEntity()
	engine.AddComponent(world, entity, Transform{
		Position: cfg.Position,
		Yaw:      cfg.YawDeg,
		Scale:    scale,
	})
	engine.AddComponent(world, entity, PlayerAvatar{
		PlayerID: cfg.PlayerID,
		Health:   health,
		Oxygen:   oxygen,
	})
	return entity
}
