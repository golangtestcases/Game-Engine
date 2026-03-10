package main

import "github.com/go-gl/mathgl/mgl32"

// minMaxY возвращает минимальную и максимальную Y-координату в массиве вершин (x,y,z,...).
func minMaxY(vertices []float32) (float32, float32) {
	if len(vertices) < 3 {
		return 0, 0
	}
	minY := vertices[1]
	maxY := vertices[1]
	for i := 1; i < len(vertices); i += 3 {
		y := vertices[i]
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}
	return minY, maxY
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// clampVec3 ограничивает каждый канал цвета/вектора диапазоном [0..1].
func clampVec3(v mgl32.Vec3) mgl32.Vec3 {
	return mgl32.Vec3{
		clamp01(v.X()),
		clamp01(v.Y()),
		clamp01(v.Z()),
	}
}
