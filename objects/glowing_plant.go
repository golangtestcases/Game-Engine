package objects

import (
	"github.com/go-gl/mathgl/mgl32"
	"subnautica-lite/engine"
	"subnautica-lite/utils"
)

// Transform — базовый компонент пространственного положения сущности.
// Используется как для растений, так и для существ.
type Transform struct {
	Position mgl32.Vec3
	Yaw      float32
	Pitch    float32
	Roll     float32
	Scale    float32
}

func (t Transform) ModelMatrix() mgl32.Mat4 {
	translate := mgl32.Translate3D(t.Position.X(), t.Position.Y(), t.Position.Z())
	rotateYaw := mgl32.HomogRotate3DY(mgl32.DegToRad(t.Yaw))
	rotatePitch := mgl32.HomogRotate3DX(mgl32.DegToRad(t.Pitch))
	rotateRoll := mgl32.HomogRotate3DZ(mgl32.DegToRad(t.Roll))
	scale := mgl32.Scale3D(t.Scale, t.Scale, t.Scale)
	return translate.Mul4(rotateYaw).Mul4(rotatePitch).Mul4(rotateRoll).Mul4(scale)
}

// GlowPlant — ECS-компонент параметров свечения растения.
type GlowPlant struct {
	Type          PlantType
	GlowColor     mgl32.Vec3
	GlowIntensity float32
	PulseSpeed    float32
}

// Набор базовых неоновых цветов для процедурного спавна.
var (
	GlowNeonGreen  = mgl32.Vec3{0.2, 1.0, 0.3}
	GlowNeonBlue   = mgl32.Vec3{0.2, 0.5, 1.0}
	GlowNeonPurple = mgl32.Vec3{0.8, 0.2, 1.0}
	GlowNeonPink   = mgl32.Vec3{1.0, 0.2, 0.6}
	GlowNeonOrange = mgl32.Vec3{1.0, 0.5, 0.1}
	GlowNeonCyan   = mgl32.Vec3{0.1, 0.9, 0.9}
	GlowNeonYellow = mgl32.Vec3{1.0, 0.95, 0.2}
	GlowNeonRed    = mgl32.Vec3{1.0, 0.1, 0.2}
)

// SpawnGlowingFlower — специализированный helper для цветка.
func SpawnGlowingFlower(world *engine.World, position mgl32.Vec3, color mgl32.Vec3, intensity, pulseSpeed float32) engine.Entity {
	return SpawnGlowingPlantAdvanced(world, position, color, intensity, pulseSpeed, PlantFlower, 1.0, 0)
}

// SpawnGlowingPlant — короткий helper с типом и параметрами по умолчанию.
func SpawnGlowingPlant(world *engine.World, position mgl32.Vec3, glowColor mgl32.Vec3, intensity float32) engine.Entity {
	return SpawnGlowingPlantAdvanced(world, position, glowColor, intensity, 2.0, PlantKelp, 1.0, 0)
}

// SpawnGlowingPlantAdvanced создает ECS-сущность и добавляет Transform + GlowPlant.
func SpawnGlowingPlantAdvanced(world *engine.World, position mgl32.Vec3, glowColor mgl32.Vec3, intensity, pulseSpeed float32, plantType PlantType, scale, yaw float32) engine.Entity {
	e := world.CreateEntity()
	engine.AddComponent(world, e, Transform{
		Position: position,
		Yaw:      yaw,
		Scale:    scale,
	})
	engine.AddComponent(world, e, GlowPlant{
		Type:          plantType,
		GlowColor:     glowColor,
		GlowIntensity: intensity,
		PulseSpeed:    pulseSpeed,
	})
	return e
}

// SpawnRandomGlowingPlants случайно расставляет светящиеся растения внутри заданного радиуса.
func SpawnRandomGlowingPlants(world *engine.World, count int, radius float32) {
	SpawnRandomGlowingPlantsWithHeightFunc(world, count, radius, func(x, z float32) float32 {
		return GroundHeightAt(x, z, DefaultGroundBaseY, DefaultGroundAmplitude)
	})
}

func SpawnRandomGlowingPlantsWithHeightFunc(world *engine.World, count int, radius float32, heightAt func(x, z float32) float32) {
	colors := []mgl32.Vec3{GlowNeonGreen, GlowNeonBlue, GlowNeonPurple, GlowNeonPink, GlowNeonOrange, GlowNeonCyan}
	types := []PlantType{PlantKelp, PlantBush, PlantCoral, PlantFlower}
	if heightAt == nil {
		heightAt = func(x, z float32) float32 {
			return GroundHeightAt(x, z, DefaultGroundBaseY, DefaultGroundAmplitude)
		}
	}

	for i := 0; i < count; i++ {
		x := utils.RandomRange(-radius, radius)
		z := utils.RandomRange(-radius, radius)
		y := heightAt(x, z)

		color := colors[int(utils.RandomRange(0, float32(len(colors))))]
		plantType := types[int(utils.RandomRange(0, float32(len(types))))]
		intensity := utils.RandomRange(1.5, 3.5)
		pulseSpeed := utils.RandomRange(0, 3.0)
		scale := utils.RandomRange(0.7, 1.4)
		yaw := utils.RandomRange(0, 360)

		SpawnGlowingPlantAdvanced(world, mgl32.Vec3{x, y, z}, color, intensity, pulseSpeed, plantType, scale, yaw)
	}
}
