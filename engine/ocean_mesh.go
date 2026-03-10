package engine

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// oceanLOD описывает одну сетку LOD и масштаб её тайлов в world-space.
type oceanLOD struct {
	Mesh   Mesh
	Scale  float32
	Radius int
}

// oceanPatchInstance — конкретный тайл, который будет нарисован в кадре.
type oceanPatchInstance struct {
	LOD      int
	Position mgl32.Vec2
	Scale    float32
}

// oceanPatchOffset хранит относительное смещение тайла от центра камеры для шаблона LOD-кольца.
type oceanPatchOffset struct {
	dx int16
	dz int16
}

const maxOceanDebugLODs = 8

// oceanInstanceBuildStats хранит отладочную статистику последней сборки инстансов океана.
// Структура фиксированного размера, чтобы не создавать лишние аллокации в кадре.
type oceanInstanceBuildStats struct {
	CenterX   float32
	CenterZ   float32
	MaxRadius float32
	Total     int
	PerLOD    [maxOceanDebugLODs]int
}

// generateOceanPatch создает регулярный квадратный патч в локальных координатах [-0.5..0.5].
// Итоговая плотность определяется `resolution`.
func generateOceanPatch(resolution int) ([]float32, []float32) {
	if resolution < 2 {
		resolution = 2
	}

	cell := 1.0 / float32(resolution)
	verts := make([]float32, 0, resolution*resolution*6*3)
	normals := make([]float32, 0, resolution*resolution*6*3)

	for z := 0; z < resolution; z++ {
		for x := 0; x < resolution; x++ {
			x0 := -0.5 + float32(x)*cell
			x1 := x0 + cell
			z0 := -0.5 + float32(z)*cell
			z1 := z0 + cell

			verts = append(verts,
				x0, 0, z0,
				x1, 0, z1,
				x1, 0, z0,
				x0, 0, z0,
				x0, 0, z1,
				x1, 0, z1,
			)

			for i := 0; i < 6; i++ {
				normals = append(normals, 0, 1, 0)
			}
		}
	}

	return verts, normals
}

// buildOceanLODTemplates подготавливает «кольца» тайлов для каждого LOD.
// Внутренняя часть более грубого LOD вырезается, чтобы избежать перекрытия с более детальным.
func buildOceanLODTemplates(lods []oceanLOD) [][]oceanPatchOffset {
	templates := make([][]oceanPatchOffset, len(lods))
	prevOuterCoverage := float32(0)
	hasPrev := false

	for i, lod := range lods {
		scale := lod.Scale
		if scale < 0.01 {
			scale = 0.01
		}

		// Work in world-space coverage (scaled tile units), not raw tile indices.
		// This keeps adjacent LOD rings continuous even when tile scales differ.
		outerCoverage := (float32(lod.Radius) + 0.55) * scale
		innerCoverage := float32(-1)
		if hasPrev {
			innerCoverage = prevOuterCoverage - 0.55*scale
			if innerCoverage < 0 {
				innerCoverage = 0
			}
		}

		offsets := make([]oceanPatchOffset, 0, (lod.Radius*2+1)*(lod.Radius*2+1))
		for dz := -lod.Radius; dz <= lod.Radius; dz++ {
			for dx := -lod.Radius; dx <= lod.Radius; dx++ {
				dist := float32(math.Sqrt(float64(dx*dx+dz*dz))) * scale
				if dist > outerCoverage {
					continue
				}
				if hasPrev && dist <= innerCoverage {
					continue
				}

				offsets = append(offsets, oceanPatchOffset{dx: int16(dx), dz: int16(dz)})
			}
		}

		templates[i] = offsets
		prevOuterCoverage = outerCoverage
		hasPrev = true
	}
	return templates
}

func estimateOceanInstanceCapacity(templates [][]oceanPatchOffset) int {
	capacity := 0
	for i := range templates {
		capacity += len(templates[i])
	}
	return capacity
}

// buildOceanInstances строит список тайлов под текущую камеру:
// снап по сетке + culling по фрустуму + ограничение радиуса.
func buildOceanInstances(
	dst []oceanPatchInstance,
	cameraPos mgl32.Vec3,
	waterLevel,
	baseTileSize float32,
	lods []oceanLOD,
	templates [][]oceanPatchOffset,
	frustum Frustum,
	cullRadiusScale float32,
) ([]oceanPatchInstance, oceanInstanceBuildStats) {
	stats := oceanInstanceBuildStats{}
	dst = dst[:0]
	if cullRadiusScale < 1.0 {
		cullRadiusScale = 1.0
	}

	for lodIndex, lod := range lods {
		tileSize := baseTileSize * lod.Scale
		centerX := float32(math.Floor(float64(cameraPos.X()/tileSize))) * tileSize
		centerZ := float32(math.Floor(float64(cameraPos.Z()/tileSize))) * tileSize
		radius := tileSize * 0.86 * cullRadiusScale
		if lodIndex == 0 {
			// Берем центр самого детального LOD как текущий «origin» океана.
			stats.CenterX = centerX
			stats.CenterZ = centerZ
		}
		// Консервативно оцениваем внешний радиус покрытия этого LOD.
		lodCoverage := (float32(lod.Radius) + 1.0) * tileSize
		if lodCoverage > stats.MaxRadius {
			stats.MaxRadius = lodCoverage
		}

		for _, offset := range templates[lodIndex] {
			worldX := centerX + float32(offset.dx)*tileSize
			worldZ := centerZ + float32(offset.dz)*tileSize
			if !frustum.ContainsSphereXYZ(worldX, waterLevel, worldZ, radius) {
				continue
			}

			dst = append(dst, oceanPatchInstance{
				LOD:      lodIndex,
				Position: mgl32.Vec2{worldX, worldZ},
				Scale:    tileSize,
			})
			stats.Total++
			if lodIndex >= 0 && lodIndex < maxOceanDebugLODs {
				stats.PerLOD[lodIndex]++
			}
		}
	}

	if stats.MaxRadius < baseTileSize {
		stats.MaxRadius = baseTileSize
	}

	return dst, stats
}

// oceanModelMatrix формирует affine-матрицу для одного океанского тайла.
func oceanModelMatrix(x, y, z, scale float32) mgl32.Mat4 {
	return mgl32.Mat4{
		scale, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, scale, 0,
		x, y, z, 1,
	}
}
