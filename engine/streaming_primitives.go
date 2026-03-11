package engine

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// StreamingCellCoord identifies one world-streaming cell on the XZ plane.
type StreamingCellCoord struct {
	X int
	Z int
}

// StreamingCellBounds stores world-space bounds and a culling sphere for one cell.
type StreamingCellBounds struct {
	Min            mgl32.Vec3
	Max            mgl32.Vec3
	Center         mgl32.Vec3
	BoundingRadius float32
}

// StreamingCellState tracks loaded/active/visible state for a cell.
type StreamingCellState struct {
	Loaded  bool
	Active  bool
	Visible bool
}

// CellCoordDistanceSq returns squared grid distance between two cells.
func CellCoordDistanceSq(a, b StreamingCellCoord) int {
	dx := a.X - b.X
	dz := a.Z - b.Z
	return dx*dx + dz*dz
}

// CellCoordFromWorldXZ converts world-space XZ into a cell coordinate.
func CellCoordFromWorldXZ(x, z, cellSize float32) StreamingCellCoord {
	if cellSize <= 0.0001 {
		cellSize = 1
	}
	return StreamingCellCoord{
		X: int(math.Floor(float64(x / cellSize))),
		Z: int(math.Floor(float64(z / cellSize))),
	}
}

// ClampCellCoord limits a coordinate into a terrain-aligned cell range.
func ClampCellCoord(coord StreamingCellCoord, minX, maxX, minZ, maxZ int) StreamingCellCoord {
	if coord.X < minX {
		coord.X = minX
	}
	if coord.X > maxX {
		coord.X = maxX
	}
	if coord.Z < minZ {
		coord.Z = minZ
	}
	if coord.Z > maxZ {
		coord.Z = maxZ
	}
	return coord
}

// SelectStreamingLODWithHysteresis returns 0/1/2 (near/mid/far) based on distance.
func SelectStreamingLODWithHysteresis(
	current int,
	distance float32,
	nearDistance float32,
	midDistance float32,
	hysteresis float32,
) int {
	if nearDistance < 0 {
		nearDistance = 0
	}
	if midDistance <= nearDistance+0.001 {
		midDistance = nearDistance + 0.001
	}
	if hysteresis < 0 {
		hysteresis = 0
	}
	if current < 0 {
		current = 0
	}
	if current > 2 {
		current = 2
	}

	switch current {
	case 1:
		if distance < nearDistance-hysteresis {
			return 0
		}
		if distance > midDistance+hysteresis {
			return 2
		}
		return 1
	case 2:
		if distance < nearDistance-hysteresis {
			return 0
		}
		if distance < midDistance-hysteresis {
			return 1
		}
		return 2
	default:
		if distance > midDistance+hysteresis {
			return 2
		}
		if distance > nearDistance+hysteresis {
			return 1
		}
		return 0
	}
}

// DistanceToggleWithHysteresis toggles a boolean state around one threshold with hysteresis.
func DistanceToggleWithHysteresis(current bool, distance, threshold, hysteresis float32) bool {
	if threshold <= 0 {
		return false
	}
	if hysteresis < 0 {
		hysteresis = 0
	}
	if current {
		return distance > threshold-hysteresis
	}
	return distance > threshold+hysteresis
}

// HorizontalDistanceXZ returns planar distance between points (ignores Y).
func HorizontalDistanceXZ(a, b mgl32.Vec3) float32 {
	dx := a.X() - b.X()
	dz := a.Z() - b.Z()
	return float32(math.Sqrt(float64(dx*dx + dz*dz)))
}
