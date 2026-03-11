package main

import (
	"math"
	"sort"

	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

const cellLODVariants = 3

type cellLODLevel uint8

const (
	cellLODNear cellLODLevel = iota
	cellLODMid
	cellLODFar
)

func (l cellLODLevel) label() string {
	switch l {
	case cellLODMid:
		return "MID"
	case cellLODFar:
		return "FAR"
	default:
		return "NEAR"
	}
}

// CellCoord задает дискретную координату world-клетки на плоскости XZ.
type CellCoord = engine.StreamingCellCoord
type CellBounds = engine.StreamingCellBounds
type CellState = engine.StreamingCellState

// WorldCell объединяет ownership данных, принадлежащих конкретной клетке.
type WorldCell struct {
	Coord  CellCoord
	Bounds CellBounds
	State  CellState

	TerrainMinY float32
	TerrainMaxY float32

	// TerrainMeshes хранит кэш GPU-мешей одного и того же участка дна в разных LOD.
	TerrainMeshes     [cellLODVariants]engine.Mesh
	TerrainMeshReady  [cellLODVariants]bool
	TerrainLOD        cellLODLevel
	TerrainSuppressed bool

	// Небольшой deterministic сдвиг дистанции, чтобы разбить ровные кольца LOD.
	LodDistanceBias float32

	// Глобальные объекты/декорации клетки для разных LOD.
	GlowPlants      []glowPlantInstance
	GlowPlantsMid   []glowPlantInstance
	GlowPlantsFar   []glowPlantInstance
	DecorLOD        cellLODLevel
	DecorSuppressed bool

	// Резерв под будущую per-cell систему сущностей/ресурсов.
	DynamicEntityIDs []EntityID
	ResourceIDs      []string
}

func (c *WorldCell) TerrainMeshForCurrentLOD() engine.Mesh {
	idx := int(c.TerrainLOD)
	if idx < 0 || idx >= len(c.TerrainMeshes) {
		return engine.Mesh{}
	}
	return c.TerrainMeshes[idx]
}

func (c *WorldCell) GlowPlantsForCurrentLOD() []glowPlantInstance {
	switch c.DecorLOD {
	case cellLODMid:
		return c.GlowPlantsMid
	case cellLODFar:
		return c.GlowPlantsFar
	default:
		return c.GlowPlants
	}
}

// worldStreamingStats используется в debug-оверлее.
type worldStreamingStats struct {
	CurrentCell           CellCoord
	LoadedCells           int
	ActiveCells           int
	VisibleCells          int
	TerrainLODVisible     [cellLODVariants]int
	DecorLODVisible       [cellLODVariants]int
	DecorInstancesVisible [cellLODVariants]int
	TerrainSuppressed     int
	DecorSuppressed       int
}

// worldStreamingManager управляет загрузкой/выгрузкой клеток и их LOD.
type worldStreamingManager struct {
	renderer *engine.Renderer
	terrain  *objects.EditableTerrain
	cfg      engine.StreamingConfig

	minTerrainCellX int
	maxTerrainCellX int
	minTerrainCellZ int
	maxTerrainCellZ int

	cells        map[CellCoord]*WorldCell
	plantBuckets map[CellCoord][]glowPlantInstance

	currentCell    CellCoord
	hasCurrentCell bool

	activeCells  []*WorldCell
	visibleCells []*WorldCell
	unloadQueue  []CellCoord

	lastStats worldStreamingStats
}

func newWorldStreamingManager(renderer *engine.Renderer, terrain *objects.EditableTerrain, cfg engine.StreamingConfig) *worldStreamingManager {
	cfg = engine.SanitizeStreamingConfig(cfg)
	m := &worldStreamingManager{
		renderer:     renderer,
		terrain:      terrain,
		cfg:          cfg,
		cells:        make(map[CellCoord]*WorldCell, 64),
		plantBuckets: make(map[CellCoord][]glowPlantInstance, 64),
		activeCells:  make([]*WorldCell, 0, 64),
		visibleCells: make([]*WorldCell, 0, 64),
		unloadQueue:  make([]CellCoord, 0, 32),
	}
	m.refreshTerrainCellRange()
	return m
}

func (m *worldStreamingManager) refreshTerrainCellRange() {
	m.minTerrainCellX = 0
	m.maxTerrainCellX = -1
	m.minTerrainCellZ = 0
	m.maxTerrainCellZ = -1
	if m == nil || m.terrain == nil {
		return
	}

	half := m.terrain.HalfExtent
	m.minTerrainCellX = int(math.Floor(float64(-half / m.cfg.CellSize)))
	m.maxTerrainCellX = int(math.Ceil(float64(half/m.cfg.CellSize))) - 1
	m.minTerrainCellZ = int(math.Floor(float64(-half / m.cfg.CellSize)))
	m.maxTerrainCellZ = int(math.Ceil(float64(half/m.cfg.CellSize))) - 1
}

func (m *worldStreamingManager) isTerrainCoord(coord CellCoord) bool {
	return coord.X >= m.minTerrainCellX &&
		coord.X <= m.maxTerrainCellX &&
		coord.Z >= m.minTerrainCellZ &&
		coord.Z <= m.maxTerrainCellZ
}

func (m *worldStreamingManager) clampCoordToTerrain(coord CellCoord) CellCoord {
	return engine.ClampCellCoord(coord, m.minTerrainCellX, m.maxTerrainCellX, m.minTerrainCellZ, m.maxTerrainCellZ)
}

func (m *worldStreamingManager) coordFromWorldXZ(x, z float32) CellCoord {
	coord := m.rawCoordFromWorldXZ(x, z)
	if m.terrain != nil {
		coord = m.clampCoordToTerrain(coord)
	}
	return coord
}

func (m *worldStreamingManager) rawCoordFromWorldXZ(x, z float32) CellCoord {
	return engine.CellCoordFromWorldXZ(x, z, m.cfg.CellSize)
}

func cellDistanceSq(a, b CellCoord) int {
	return engine.CellCoordDistanceSq(a, b)
}

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

func (m *worldStreamingManager) AssignGlowPlants(instances []glowPlantInstance) {
	if m == nil {
		return
	}

	for coord := range m.plantBuckets {
		m.plantBuckets[coord] = m.plantBuckets[coord][:0]
	}

	for i := range instances {
		instance := instances[i]
		coord := m.rawCoordFromWorldXZ(instance.position.X(), instance.position.Z())
		if m.terrain != nil && !m.isTerrainCoord(coord) {
			continue
		}
		bucket := append(m.plantBuckets[coord], instance)
		m.plantBuckets[coord] = bucket
	}

	for coord := range m.plantBuckets {
		bucket := m.plantBuckets[coord]
		sort.Slice(bucket, func(i, j int) bool {
			return bucket[i].mesh.VAO < bucket[j].mesh.VAO
		})
		m.plantBuckets[coord] = bucket
	}

	for _, cell := range m.cells {
		m.syncCellPlants(cell)
	}
}

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

	// Русский комментарий: обновляем только уже загруженные cell, чтобы не создавать
	// новые чанки во время мазка и не менять поведение streaming-системы.
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

func (m *worldStreamingManager) syncCellPlants(cell *WorldCell) {
	if cell == nil {
		return
	}

	cell.GlowPlants = cell.GlowPlants[:0]
	cell.GlowPlantsMid = cell.GlowPlantsMid[:0]
	cell.GlowPlantsFar = cell.GlowPlantsFar[:0]

	bucket, ok := m.plantBuckets[cell.Coord]
	if !ok || len(bucket) == 0 {
		return
	}

	cell.GlowPlants = append(cell.GlowPlants, bucket...)
	for i := range bucket {
		instance := bucket[i]

		if instance.boundingRadius >= m.cfg.DecorLODMidMinRadius &&
			keepByDensity(instance.stableSeed, m.cfg.DecorLODMidDensity) {
			cell.GlowPlantsMid = append(cell.GlowPlantsMid, instance)
		}

		if instance.boundingRadius >= m.cfg.DecorLODFarMinRadius &&
			keepByDensity(instance.stableSeed, m.cfg.DecorLODFarDensity) {
			cell.GlowPlantsFar = append(cell.GlowPlantsFar, instance)
		}
	}

	// Оставляем минимум один экземпляр, чтобы дальняя зона не "обрывалась" целыми клетками.
	if len(cell.GlowPlantsMid) == 0 && m.cfg.DecorLODMidDensity > 0 {
		if fallback, ok := pickFallbackPlant(cell.GlowPlants, m.cfg.DecorLODMidMinRadius); ok {
			cell.GlowPlantsMid = append(cell.GlowPlantsMid, fallback)
		}
	}
	if len(cell.GlowPlantsFar) == 0 && m.cfg.DecorLODFarDensity > 0 {
		if fallback, ok := pickFallbackPlant(cell.GlowPlants, m.cfg.DecorLODFarMinRadius); ok {
			cell.GlowPlantsFar = append(cell.GlowPlantsFar, fallback)
		}
	}
}

func keepByDensity(seed uint32, density float32) bool {
	if density >= 0.9999 {
		return true
	}
	if density <= 0 {
		return false
	}
	return seedToUnitInterval(seed) <= clamp01(density)
}

func pickFallbackPlant(source []glowPlantInstance, minRadius float32) (glowPlantInstance, bool) {
	var chosen glowPlantInstance
	var chosenSeed uint32
	found := false

	for i := range source {
		instance := source[i]
		if instance.boundingRadius < minRadius {
			continue
		}
		if !found || instance.stableSeed < chosenSeed {
			chosen = instance
			chosenSeed = instance.stableSeed
			found = true
		}
	}
	return chosen, found
}

func seedToUnitInterval(seed uint32) float32 {
	const max24 = float32(0x00ffffff)
	return float32(seed&0x00ffffff) / max24
}

func cellLODDistanceBias(coord CellCoord, cellSize float32) float32 {
	seed := cellCoordSeed(coord)
	unit := seedToUnitInterval(seed)
	// Диапазон [-1..1] * часть cell-size дает мягкое распределение границ.
	return (unit*2 - 1) * cellSize * 0.22
}

func cellCoordSeed(coord CellCoord) uint32 {
	x := uint32(int32(coord.X))
	z := uint32(int32(coord.Z))
	seed := x*0x9e3779b1 ^ z*0x85ebca77 ^ 0xc2b2ae3d
	return mixSeed(seed)
}

func mixSeed(v uint32) uint32 {
	v ^= v >> 16
	v *= 0x7feb352d
	v ^= v >> 15
	v *= 0x846ca68b
	v ^= v >> 16
	return v
}

func (m *worldStreamingManager) unloadCell(coord CellCoord) {
	cell, ok := m.cells[coord]
	if !ok {
		return
	}
	m.invalidateCellTerrainMeshes(cell, true)
	delete(m.cells, coord)
}
