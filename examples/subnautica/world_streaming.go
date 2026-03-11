package main

import (
	"math"

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
