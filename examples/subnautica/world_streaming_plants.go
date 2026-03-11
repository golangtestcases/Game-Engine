package main

import "sort"

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
