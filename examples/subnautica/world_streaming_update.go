package main

import (
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
)

func (m *worldStreamingManager) UpdateStreaming(cameraPos mgl32.Vec3) {
	if m == nil || m.renderer == nil || m.terrain == nil {
		return
	}

	current := m.coordFromWorldXZ(cameraPos.X(), cameraPos.Z())
	m.currentCell = current
	m.hasCurrentCell = true

	activeRadiusSq := m.cfg.ActiveRadius * m.cfg.ActiveRadius
	unloadRadiusSq := m.cfg.UnloadRadius * m.cfg.UnloadRadius

	m.unloadQueue = m.unloadQueue[:0]
	for coord, cell := range m.cells {
		distSq := cellDistanceSq(coord, current)
		cell.State.Visible = false
		if distSq > unloadRadiusSq {
			m.unloadQueue = append(m.unloadQueue, coord)
			continue
		}
		cell.State.Active = distSq <= activeRadiusSq
	}
	for i := range m.unloadQueue {
		m.unloadCell(m.unloadQueue[i])
	}

	for dz := -m.cfg.ActiveRadius; dz <= m.cfg.ActiveRadius; dz++ {
		for dx := -m.cfg.ActiveRadius; dx <= m.cfg.ActiveRadius; dx++ {
			if dx*dx+dz*dz > activeRadiusSq {
				continue
			}
			coord := CellCoord{X: current.X + dx, Z: current.Z + dz}
			if !m.isTerrainCoord(coord) {
				continue
			}
			cell := m.ensureCell(coord)
			if cell == nil {
				continue
			}
			cell.State.Active = true
		}
	}

	m.collectActiveCells()
	m.updateActiveCellLODs(cameraPos)

	m.lastStats.LoadedCells = len(m.cells)
	m.lastStats.ActiveCells = len(m.activeCells)
	m.lastStats.CurrentCell = m.currentCell
}

func (m *worldStreamingManager) collectActiveCells() {
	m.activeCells = m.activeCells[:0]
	if !m.hasCurrentCell {
		return
	}

	activeRadiusSq := m.cfg.ActiveRadius * m.cfg.ActiveRadius
	for dz := -m.cfg.ActiveRadius; dz <= m.cfg.ActiveRadius; dz++ {
		for dx := -m.cfg.ActiveRadius; dx <= m.cfg.ActiveRadius; dx++ {
			if dx*dx+dz*dz > activeRadiusSq {
				continue
			}
			coord := CellCoord{X: m.currentCell.X + dx, Z: m.currentCell.Z + dz}
			cell, ok := m.cells[coord]
			if !ok || !cell.State.Loaded || !cell.State.Active {
				continue
			}
			m.activeCells = append(m.activeCells, cell)
		}
	}
}

func (m *worldStreamingManager) ActiveCells() []*WorldCell {
	return m.activeCells
}

func (m *worldStreamingManager) VisibleCells(
	frustum engine.Frustum,
	viewPos mgl32.Vec3,
	maxDistance float32,
	recordStats bool,
) []*WorldCell {
	m.visibleCells = m.visibleCells[:0]
	if m == nil || !m.hasCurrentCell {
		return m.visibleCells
	}

	visibleRadiusSq := m.cfg.VisibleRadius * m.cfg.VisibleRadius
	maxDistanceSq := float32(0)
	if maxDistance > 0 {
		maxDistanceSq = maxDistance * maxDistance
	}

	stats := m.lastStats
	stats.VisibleCells = 0
	stats.TerrainLODVisible = [cellLODVariants]int{}
	stats.DecorLODVisible = [cellLODVariants]int{}
	stats.DecorInstancesVisible = [cellLODVariants]int{}
	stats.TerrainSuppressed = 0
	stats.DecorSuppressed = 0

	for dz := -m.cfg.VisibleRadius; dz <= m.cfg.VisibleRadius; dz++ {
		for dx := -m.cfg.VisibleRadius; dx <= m.cfg.VisibleRadius; dx++ {
			if dx*dx+dz*dz > visibleRadiusSq {
				continue
			}
			coord := CellCoord{X: m.currentCell.X + dx, Z: m.currentCell.Z + dz}
			cell, ok := m.cells[coord]
			if !ok || !cell.State.Loaded || !cell.State.Active {
				continue
			}

			if !frustum.ContainsSphere(cell.Bounds.Center, cell.Bounds.BoundingRadius) {
				cell.State.Visible = false
				continue
			}
			if maxDistanceSq > 0 {
				delta := cell.Bounds.Center.Sub(viewPos)
				if delta.Dot(delta) > maxDistanceSq {
					cell.State.Visible = false
					continue
				}
			}

			// Подстраховка: если текущий LOD еще не построен, достраиваем его при входе в видимость.
			if !m.ensureTerrainLODMesh(cell, cell.TerrainLOD) {
				cell.State.Visible = false
				continue
			}

			cell.State.Visible = true
			m.visibleCells = append(m.visibleCells, cell)
			if !recordStats {
				continue
			}

			if cell.TerrainSuppressed {
				stats.TerrainSuppressed++
			} else {
				stats.TerrainLODVisible[int(cell.TerrainLOD)]++
			}

			if cell.DecorSuppressed {
				stats.DecorSuppressed++
			} else {
				stats.DecorLODVisible[int(cell.DecorLOD)]++
				stats.DecorInstancesVisible[int(cell.DecorLOD)] += len(cell.GlowPlantsForCurrentLOD())
			}
		}
	}

	if recordStats {
		stats.VisibleCells = len(m.visibleCells)
		m.lastStats = stats
	}
	return m.visibleCells
}

func (m *worldStreamingManager) Stats() worldStreamingStats {
	stats := m.lastStats
	stats.LoadedCells = len(m.cells)
	stats.ActiveCells = len(m.activeCells)
	if m.hasCurrentCell {
		stats.CurrentCell = m.currentCell
	}
	return stats
}

func (m *worldStreamingManager) updateActiveCellLODs(cameraPos mgl32.Vec3) {
	if m == nil {
		return
	}
	for i := range m.activeCells {
		cell := m.activeCells[i]
		if cell == nil || !cell.State.Loaded {
			continue
		}

		dist := engine.HorizontalDistanceXZ(cameraPos, cell.Bounds.Center)
		terrainDist := dist + cell.LodDistanceBias
		if terrainDist < 0 {
			terrainDist = 0
		}
		decorDist := dist + cell.LodDistanceBias*0.65
		if decorDist < 0 {
			decorDist = 0
		}

		cell.TerrainLOD = cellLODLevel(engine.SelectStreamingLODWithHysteresis(
			int(cell.TerrainLOD),
			terrainDist,
			m.cfg.TerrainLODNearDistance,
			m.cfg.TerrainLODMidDistance,
			m.cfg.TerrainLODHysteresis,
		))
		cell.DecorLOD = cellLODLevel(engine.SelectStreamingLODWithHysteresis(
			int(cell.DecorLOD),
			decorDist,
			m.cfg.DecorLODNearDistance,
			m.cfg.DecorLODMidDistance,
			m.cfg.DecorLODHysteresis,
		))

		cell.TerrainSuppressed = engine.DistanceToggleWithHysteresis(
			cell.TerrainSuppressed,
			terrainDist,
			m.cfg.TerrainLODFarCullDistance,
			m.cfg.TerrainLODHysteresis,
		)
		cell.DecorSuppressed = engine.DistanceToggleWithHysteresis(
			cell.DecorSuppressed,
			decorDist,
			m.cfg.DecorLODFarCullDistance,
			m.cfg.DecorLODHysteresis,
		)

		m.ensureTerrainLODMesh(cell, cell.TerrainLOD)
	}
}
