package main

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

func (m *worldStreamingManager) RebuildLoadedTerrainMeshes() {
	if m == nil || m.terrain == nil {
		return
	}

	half := m.terrain.HalfExtent
	m.RebuildLoadedTerrainMeshesInWorldRange(-half, half, -half, half)
}

func (m *worldStreamingManager) RebuildLoadedTerrainMeshesInWorldRange(minX, maxX, minZ, maxZ float32) {
	if m == nil || m.terrain == nil {
		return
	}
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	if minZ > maxZ {
		minZ, maxZ = maxZ, minZ
	}

	half := m.terrain.HalfExtent
	if maxX < -half || minX > half || maxZ < -half || minZ > half {
		return
	}
	if minX < -half {
		minX = -half
	}
	if maxX > half {
		maxX = half
	}
	if minZ < -half {
		minZ = -half
	}
	if maxZ > half {
		maxZ = half
	}

	// Обновляем только уже загруженные cell, чтобы не создавать новые чанки во время мазка
	// и не менять поведение streaming-системы.
	for _, cell := range m.cells {
		cellMinX := float32(cell.Coord.X) * m.cfg.CellSize
		cellMaxX := cellMinX + m.cfg.CellSize
		cellMinZ := float32(cell.Coord.Z) * m.cfg.CellSize
		cellMaxZ := cellMinZ + m.cfg.CellSize
		if cellMaxX < minX || cellMinX > maxX || cellMaxZ < minZ || cellMinZ > maxZ {
			continue
		}

		if !m.updateCellTerrainBounds(cell) {
			continue
		}
		m.invalidateCellTerrainMeshes(cell, false)
		if cell.State.Active {
			m.ensureTerrainLODMesh(cell, cell.TerrainLOD)
		}
	}
}

func (m *worldStreamingManager) terrainStepForLOD(lod cellLODLevel) int {
	switch lod {
	case cellLODMid:
		if m.cfg.TerrainLODMidStep < 1 {
			return 1
		}
		return m.cfg.TerrainLODMidStep
	case cellLODFar:
		if m.cfg.TerrainLODFarStep < 1 {
			return 1
		}
		return m.cfg.TerrainLODFarStep
	default:
		return 1
	}
}

func (m *worldStreamingManager) ensureTerrainLODMesh(cell *WorldCell, lod cellLODLevel) bool {
	if cell == nil {
		return false
	}
	lodIdx := int(lod)
	if lodIdx < 0 || lodIdx >= len(cell.TerrainMeshes) {
		return false
	}
	if !cell.TerrainMeshReady[lodIdx] {
		if !m.buildCellTerrainLODMesh(cell, lod) {
			return false
		}
	}
	return cell.TerrainMeshes[lodIdx].VertexCount > 0
}

func (m *worldStreamingManager) buildCellTerrainLODMesh(cell *WorldCell, lod cellLODLevel) bool {
	if m == nil || m.terrain == nil || cell == nil {
		return false
	}

	minGX, minGZ, maxGX, maxGZ := m.terrainRangeForCell(cell.Coord)
	step := m.terrainStepForLOD(lod)
	vertices, normals, _, _ := m.terrain.BuildMeshDataForRangeLOD(minGX, minGZ, maxGX, maxGZ, step)

	lodIdx := int(lod)
	mesh := &cell.TerrainMeshes[lodIdx]
	if len(vertices) == 0 || len(normals) == 0 {
		if mesh.VAO != 0 {
			m.renderer.UpdateMeshWithNormals(mesh, nil, nil)
		}
		cell.TerrainMeshReady[lodIdx] = true
		return false
	}

	if mesh.VAO == 0 {
		*mesh = m.renderer.NewMeshWithNormals(vertices, normals)
	} else {
		m.renderer.UpdateMeshWithNormals(mesh, vertices, normals)
	}
	cell.TerrainMeshReady[lodIdx] = true
	return mesh.VertexCount > 0
}

func (m *worldStreamingManager) invalidateCellTerrainMeshes(cell *WorldCell, releaseGPU bool) {
	if cell == nil {
		return
	}
	for i := 0; i < len(cell.TerrainMeshes); i++ {
		cell.TerrainMeshReady[i] = false
		if releaseGPU && cell.TerrainMeshes[i].VAO != 0 {
			m.renderer.DeleteMesh(&cell.TerrainMeshes[i])
		}
	}
}

func (m *worldStreamingManager) updateCellTerrainBounds(cell *WorldCell) bool {
	if m == nil || m.terrain == nil || cell == nil {
		return false
	}

	minGX, minGZ, maxGX, maxGZ := m.terrainRangeForCell(cell.Coord)
	minY, maxY, ok := m.terrainHeightRangeForCell(minGX, minGZ, maxGX, maxGZ)
	if !ok {
		cell.State.Loaded = false
		cell.State.Visible = false
		return false
	}

	cell.TerrainMinY = minY
	cell.TerrainMaxY = maxY
	cell.State.Loaded = true

	cellMinX := float32(cell.Coord.X) * m.cfg.CellSize
	cellMinZ := float32(cell.Coord.Z) * m.cfg.CellSize
	cellMaxX := cellMinX + m.cfg.CellSize
	cellMaxZ := cellMinZ + m.cfg.CellSize
	centerY := (minY + maxY) * 0.5

	halfX := (cellMaxX - cellMinX) * 0.5
	halfZ := (cellMaxZ - cellMinZ) * 0.5
	halfY := (maxY - minY) * 0.5
	if halfY < 1.2 {
		halfY = 1.2
	}
	radius := float32(math.Sqrt(float64(halfX*halfX + halfY*halfY + halfZ*halfZ)))

	cell.Bounds = CellBounds{
		Min:            mgl32.Vec3{cellMinX, minY, cellMinZ},
		Max:            mgl32.Vec3{cellMaxX, maxY, cellMaxZ},
		Center:         mgl32.Vec3{cellMinX + halfX, centerY, cellMinZ + halfZ},
		BoundingRadius: radius,
	}
	return true
}

func (m *worldStreamingManager) terrainHeightRangeForCell(minGX, minGZ, maxGX, maxGZ int) (float32, float32, bool) {
	if m == nil || m.terrain == nil {
		return 0, 0, false
	}
	if minGX < 0 {
		minGX = 0
	}
	if minGZ < 0 {
		minGZ = 0
	}
	if maxGX > m.terrain.GridSize {
		maxGX = m.terrain.GridSize
	}
	if maxGZ > m.terrain.GridSize {
		maxGZ = m.terrain.GridSize
	}
	if minGX > maxGX || minGZ > maxGZ {
		return 0, 0, false
	}

	pointCount := m.terrain.GridSize + 1
	minY := float32(0)
	maxY := float32(0)
	first := true

	for gz := minGZ; gz <= maxGZ; gz++ {
		row := gz * pointCount
		for gx := minGX; gx <= maxGX; gx++ {
			h := m.terrain.Heights[row+gx]
			if first {
				minY = h
				maxY = h
				first = false
				continue
			}
			if h < minY {
				minY = h
			}
			if h > maxY {
				maxY = h
			}
		}
	}
	return minY, maxY, !first
}

func (m *worldStreamingManager) terrainRangeForCell(coord CellCoord) (int, int, int, int) {
	minX := float32(coord.X) * m.cfg.CellSize
	minZ := float32(coord.Z) * m.cfg.CellSize
	maxX := minX + m.cfg.CellSize
	maxZ := minZ + m.cfg.CellSize

	half := m.terrain.HalfExtent
	toGridMin := func(v float32) int {
		return int(math.Floor(float64((v + half) / m.terrain.CellSize)))
	}
	toGridMax := func(v float32) int {
		return int(math.Ceil(float64((v + half) / m.terrain.CellSize)))
	}

	minGX := toGridMin(minX)
	minGZ := toGridMin(minZ)
	maxGX := toGridMax(maxX)
	maxGZ := toGridMax(maxZ)

	if minGX < 0 {
		minGX = 0
	}
	if minGZ < 0 {
		minGZ = 0
	}
	if maxGX > m.terrain.GridSize {
		maxGX = m.terrain.GridSize
	}
	if maxGZ > m.terrain.GridSize {
		maxGZ = m.terrain.GridSize
	}
	if minGX > maxGX {
		minGX = maxGX
	}
	if minGZ > maxGZ {
		minGZ = maxGZ
	}
	return minGX, minGZ, maxGX, maxGZ
}

func (m *worldStreamingManager) ensureCell(coord CellCoord) *WorldCell {
	if !m.isTerrainCoord(coord) {
		return nil
	}
	if cell, ok := m.cells[coord]; ok {
		return cell
	}

	cell := &WorldCell{
		Coord: coord,
		State: CellState{
			Loaded: true,
		},
		TerrainLOD:       cellLODNear,
		DecorLOD:         cellLODNear,
		LodDistanceBias:  cellLODDistanceBias(coord, m.cfg.CellSize),
		GlowPlants:       make([]glowPlantInstance, 0, 8),
		GlowPlantsMid:    make([]glowPlantInstance, 0, 6),
		GlowPlantsFar:    make([]glowPlantInstance, 0, 4),
		DynamicEntityIDs: make([]EntityID, 0),
		ResourceIDs:      make([]string, 0),
	}
	m.cells[coord] = cell

	if !m.updateCellTerrainBounds(cell) {
		delete(m.cells, coord)
		return nil
	}
	if !m.ensureTerrainLODMesh(cell, cell.TerrainLOD) {
		delete(m.cells, coord)
		return nil
	}
	m.syncCellPlants(cell)
	return cell
}

func (m *worldStreamingManager) unloadCell(coord CellCoord) {
	cell, ok := m.cells[coord]
	if !ok {
		return
	}
	m.invalidateCellTerrainMeshes(cell, true)
	delete(m.cells, coord)
}
