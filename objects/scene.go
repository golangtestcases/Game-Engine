package objects

import (
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/utils"
)

// Plant — экземпляр декоративного растения на сцене.
type Plant struct {
	Type     PlantType
	Position mgl32.Vec3
	Yaw      float32
	Scale    float32
	Color    mgl32.Vec3
}

// ModelMatrix формирует world-transform растения.
func (p Plant) ModelMatrix() mgl32.Mat4 {
	translate := mgl32.Translate3D(p.Position.X(), p.Position.Y(), p.Position.Z())
	rotate := mgl32.HomogRotate3DY(mgl32.DegToRad(p.Yaw))
	scale := mgl32.Scale3D(p.Scale, p.Scale, p.Scale)
	return translate.Mul4(rotate).Mul4(scale)
}

// Scene содержит процедурно сгенерированные объекты окружения.
type Scene struct {
	Plants []Plant
}

// NewScene заполняет сцену случайно распределенными растениями над грунтом.
func NewScene(numPlants int) Scene {
	plants := make([]Plant, numPlants)

	spawnRadius := GroundSpawnRadius()
	xMin, xMax := -spawnRadius, spawnRadius
	zMin, zMax := -spawnRadius, spawnRadius

	for i := 0; i < numPlants; i++ {
		x := utils.RandomRange(xMin, xMax)
		z := utils.RandomRange(zMin, zMax)
		y := GroundHeightAt(x, z, DefaultGroundBaseY, DefaultGroundAmplitude)

		kindRoll := utils.RandomRange(0, 1)
		plantType := PlantKelp
		scale := utils.RandomRange(0.85, 1.45)
		color := mgl32.Vec3{utils.RandomRange(0.12, 0.22), utils.RandomRange(0.55, 0.76), utils.RandomRange(0.24, 0.36)}

		switch {
		case kindRoll < 0.50:
			plantType = PlantKelp
			scale = utils.RandomRange(0.95, 1.65)
			color = mgl32.Vec3{utils.RandomRange(0.10, 0.18), utils.RandomRange(0.58, 0.80), utils.RandomRange(0.22, 0.34)}
		case kindRoll < 0.82:
			plantType = PlantBush
			scale = utils.RandomRange(0.75, 1.25)
			color = mgl32.Vec3{utils.RandomRange(0.14, 0.24), utils.RandomRange(0.50, 0.68), utils.RandomRange(0.18, 0.30)}
		default:
			plantType = PlantCoral
			scale = utils.RandomRange(0.70, 1.15)
			color = mgl32.Vec3{utils.RandomRange(0.50, 0.72), utils.RandomRange(0.28, 0.44), utils.RandomRange(0.42, 0.62)}
		}

		plants[i] = Plant{
			Type:     plantType,
			Position: mgl32.Vec3{x, y + 0.03, z},
			Yaw:      utils.RandomRange(0, 360),
			Scale:    scale,
			Color:    color,
		}
	}

	return Scene{Plants: plants}
}
