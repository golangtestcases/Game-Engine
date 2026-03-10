package objects

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/go-gl/mathgl/mgl32"
)

type TerrainBrushTool uint8

const (
	TerrainBrushRaise TerrainBrushTool = iota
	TerrainBrushLower
	TerrainBrushSmooth
	TerrainBrushFlatten
)

func (t TerrainBrushTool) Label() string {
	switch t {
	case TerrainBrushLower:
		return "LOWER"
	case TerrainBrushSmooth:
		return "SMOOTH"
	case TerrainBrushFlatten:
		return "FLATTEN"
	default:
		return "RAISE"
	}
}

type EditableTerrain struct {
	GridSize   int
	CellSize   float32
	HalfExtent float32
	Heights    []float32
}

type editableTerrainFile struct {
	Version  int       `json:"version"`
	GridSize int       `json:"grid_size"`
	CellSize float32   `json:"cell_size"`
	Heights  []float32 `json:"heights"`
}

const editableTerrainFileVersion = 1

func NewDefaultEditableTerrain() *EditableTerrain {
	return NewEditableTerrain(
		DefaultGroundGridSize,
		DefaultGroundCellSize,
		DefaultGroundBaseY,
		DefaultGroundAmplitude,
	)
}

func NewEditableTerrain(gridSize int, cellSize, baseY, amplitude float32) *EditableTerrain {
	if gridSize < 1 {
		gridSize = 1
	}
	if cellSize <= 0 {
		cellSize = 1
	}

	pointCount := gridSize + 1
	half := float32(gridSize) * cellSize * 0.5
	heights := make([]float32, pointCount*pointCount)

	for z := 0; z < pointCount; z++ {
		for x := 0; x < pointCount; x++ {
			wx := float32(x)*cellSize - half
			wz := float32(z)*cellSize - half
			heights[z*pointCount+x] = groundHeight(wx, wz, baseY, amplitude)
		}
	}

	return &EditableTerrain{
		GridSize:   gridSize,
		CellSize:   cellSize,
		HalfExtent: half,
		Heights:    heights,
	}
}

func NewEditableTerrainFromHeights(gridSize int, cellSize float32, heights []float32) (*EditableTerrain, error) {
	if gridSize < 1 {
		return nil, errors.New("terrain: grid size must be >= 1")
	}
	if cellSize <= 0 {
		return nil, errors.New("terrain: cell size must be > 0")
	}

	pointCount := gridSize + 1
	expected := pointCount * pointCount
	if len(heights) != expected {
		return nil, fmt.Errorf("terrain: invalid heights length %d, expected %d", len(heights), expected)
	}

	heightCopy := make([]float32, len(heights))
	copy(heightCopy, heights)
	return &EditableTerrain{
		GridSize:   gridSize,
		CellSize:   cellSize,
		HalfExtent: float32(gridSize) * cellSize * 0.5,
		Heights:    heightCopy,
	}, nil
}

func LoadEditableTerrainJSON(path string) (*EditableTerrain, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var payload editableTerrainFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Version != editableTerrainFileVersion {
		return nil, fmt.Errorf("terrain: unsupported file version %d", payload.Version)
	}

	return NewEditableTerrainFromHeights(payload.GridSize, payload.CellSize, payload.Heights)
}

func (t *EditableTerrain) SaveJSON(path string) error {
	if t == nil {
		return errors.New("terrain: nil terrain")
	}

	payload := editableTerrainFile{
		Version:  editableTerrainFileVersion,
		GridSize: t.GridSize,
		CellSize: t.CellSize,
		Heights:  make([]float32, len(t.Heights)),
	}
	copy(payload.Heights, t.Heights)

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (t *EditableTerrain) pointCount() int {
	return t.GridSize + 1
}

func (t *EditableTerrain) heightIndex(x, z int) int {
	return z*t.pointCount() + x
}

func (t *EditableTerrain) clampIndex(v int) int {
	if v < 0 {
		return 0
	}
	maxV := t.GridSize
	if v > maxV {
		return maxV
	}
	return v
}

func (t *EditableTerrain) heightAtGrid(x, z int) float32 {
	x = t.clampIndex(x)
	z = t.clampIndex(z)
	return t.Heights[t.heightIndex(x, z)]
}

func (t *EditableTerrain) worldFromGrid(x, z int) (float32, float32) {
	return float32(x)*t.CellSize - t.HalfExtent, float32(z)*t.CellSize - t.HalfExtent
}

func (t *EditableTerrain) InBoundsXZ(x, z float32) bool {
	if t == nil {
		return false
	}
	return x >= -t.HalfExtent && x <= t.HalfExtent && z >= -t.HalfExtent && z <= t.HalfExtent
}

func (t *EditableTerrain) HeightAt(x, z float32) float32 {
	if t == nil || t.GridSize < 1 {
		return 0
	}

	fx := (x + t.HalfExtent) / t.CellSize
	fz := (z + t.HalfExtent) / t.CellSize

	if fx < 0 {
		fx = 0
	}
	if fz < 0 {
		fz = 0
	}
	maxF := float32(t.GridSize)
	if fx > maxF {
		fx = maxF
	}
	if fz > maxF {
		fz = maxF
	}

	x0 := int(math.Floor(float64(fx)))
	z0 := int(math.Floor(float64(fz)))
	if x0 >= t.GridSize {
		x0 = t.GridSize - 1
	}
	if z0 >= t.GridSize {
		z0 = t.GridSize - 1
	}

	x1 := x0 + 1
	z1 := z0 + 1
	tx := fx - float32(x0)
	tz := fz - float32(z0)

	h00 := t.heightAtGrid(x0, z0)
	h10 := t.heightAtGrid(x1, z0)
	h01 := t.heightAtGrid(x0, z1)
	h11 := t.heightAtGrid(x1, z1)

	hx0 := h00 + (h10-h00)*tx
	hx1 := h01 + (h11-h01)*tx
	return hx0 + (hx1-hx0)*tz
}

func (t *EditableTerrain) sampleNeighborhoodAverage(gridX, gridZ int, source []float32) float32 {
	sum := float32(0)
	count := float32(0)
	pointCount := t.pointCount()

	for oz := -1; oz <= 1; oz++ {
		for ox := -1; ox <= 1; ox++ {
			x := t.clampIndex(gridX + ox)
			z := t.clampIndex(gridZ + oz)
			sum += source[z*pointCount+x]
			count++
		}
	}
	if count <= 0 {
		return source[t.heightIndex(gridX, gridZ)]
	}
	return sum / count
}

func (t *EditableTerrain) ApplyBrush(centerX, centerZ, radius, strength, deltaTime float32, tool TerrainBrushTool, flattenTarget float32) bool {
	if t == nil || t.GridSize < 1 {
		return false
	}
	if radius <= 0 || strength <= 0 || deltaTime <= 0 {
		return false
	}
	if deltaTime > 0.1 {
		// Русский комментарий: при резком провале FPS ограничиваем dt мазка,
		// чтобы одиночный кадр не создавал слишком агрессивный "скачок" рельефа.
		deltaTime = 0.1
	}
	if math.IsNaN(float64(flattenTarget)) || math.IsInf(float64(flattenTarget), 0) {
		flattenTarget = t.HeightAt(centerX, centerZ)
	}

	minWX := centerX - radius
	maxWX := centerX + radius
	minWZ := centerZ - radius
	maxWZ := centerZ + radius

	toGrid := func(v float32) int {
		return int(math.Floor(float64((v + t.HalfExtent) / t.CellSize)))
	}

	minGX := t.clampIndex(toGrid(minWX))
	maxGX := t.clampIndex(toGrid(maxWX) + 1)
	minGZ := t.clampIndex(toGrid(minWZ))
	maxGZ := t.clampIndex(toGrid(maxWZ) + 1)

	if minGX > maxGX || minGZ > maxGZ {
		return false
	}

	changed := false
	invRadius := float32(1.0) / radius
	baseDelta := strength * deltaTime
	source := t.Heights
	if tool == TerrainBrushSmooth {
		source = make([]float32, len(t.Heights))
		copy(source, t.Heights)
	}

	for gz := minGZ; gz <= maxGZ; gz++ {
		for gx := minGX; gx <= maxGX; gx++ {
			wx, wz := t.worldFromGrid(gx, gz)
			dx := wx - centerX
			dz := wz - centerZ
			dist := float32(math.Sqrt(float64(dx*dx + dz*dz)))
			if dist > radius {
				continue
			}

			falloff := 1 - dist*invRadius
			if falloff <= 0 {
				continue
			}
			falloff = falloff * falloff * (3 - 2*falloff)

			idx := t.heightIndex(gx, gz)
			before := t.Heights[idx]
			switch tool {
			case TerrainBrushLower:
				t.Heights[idx] -= baseDelta * falloff
			case TerrainBrushSmooth:
				target := t.sampleNeighborhoodAverage(gx, gz, source)
				blend := clamp01f(baseDelta * falloff)
				t.Heights[idx] = before + (target-before)*blend
			case TerrainBrushFlatten:
				blend := clamp01f(baseDelta * falloff)
				t.Heights[idx] = before + (flattenTarget-before)*blend
			default:
				t.Heights[idx] += baseDelta * falloff
			}
			if absf32(t.Heights[idx]-before) > 0.000001 {
				changed = true
			}
		}
	}

	return changed
}

func (t *EditableTerrain) BuildMeshData() ([]float32, []float32, float32, float32) {
	return t.BuildMeshDataForRange(0, 0, t.GridSize, t.GridSize)
}

// BuildMeshDataForRange СЃС‚СЂРѕРёС‚ РјРµС€ С‚РѕР»СЊРєРѕ РґР»СЏ РїРѕРґРґРёР°РїР°Р·РѕРЅР° terrain-СЃРµС‚РєРё.
// Р”РёР°РїР°Р·РѕРЅ Р·Р°РґР°РµС‚СЃСЏ РёРЅРґРµРєСЃР°РјРё СЏС‡РµРµРє [min; max) РїРѕ X Рё Z.
func (t *EditableTerrain) BuildMeshDataForRange(minGX, minGZ, maxGX, maxGZ int) ([]float32, []float32, float32, float32) {
	return t.BuildMeshDataForRangeLOD(minGX, minGZ, maxGX, maxGZ, 1)
}

func (t *EditableTerrain) BuildMeshDataForRangeLOD(minGX, minGZ, maxGX, maxGZ, step int) ([]float32, []float32, float32, float32) {
	if t == nil || t.GridSize < 1 {
		return nil, nil, 0, 0
	}
	if step < 1 {
		step = 1
	}

	if minGX < 0 {
		minGX = 0
	}
	if minGZ < 0 {
		minGZ = 0
	}
	if maxGX > t.GridSize {
		maxGX = t.GridSize
	}
	if maxGZ > t.GridSize {
		maxGZ = t.GridSize
	}
	if minGX >= maxGX || minGZ >= maxGZ {
		return nil, nil, 0, 0
	}

	xSamples := buildTerrainSampleAxis(minGX, maxGX, step)
	zSamples := buildTerrainSampleAxis(minGZ, maxGZ, step)
	if len(xSamples) < 2 || len(zSamples) < 2 {
		return nil, nil, 0, 0
	}

	pointsX := len(xSamples)
	pointsZ := len(zSamples)
	normalGrid := make([]mgl32.Vec3, pointsX*pointsZ)
	localNormalIndex := func(ix, iz int) int {
		return iz*pointsX + ix
	}

	for iz, gz := range zSamples {
		for ix, gx := range xSamples {
			hL := t.heightAtGrid(gx-1, gz)
			hR := t.heightAtGrid(gx+1, gz)
			hD := t.heightAtGrid(gx, gz-1)
			hU := t.heightAtGrid(gx, gz+1)

			ddx := (hR - hL) / (2 * t.CellSize)
			ddz := (hU - hD) / (2 * t.CellSize)

			n := mgl32.Vec3{-ddx, 1, -ddz}
			if n.Len() < 0.0001 {
				n = mgl32.Vec3{0, 1, 0}
			} else {
				n = n.Normalize()
			}
			normalGrid[localNormalIndex(ix, iz)] = n
		}
	}

	cellsX := len(xSamples) - 1
	cellsZ := len(zSamples) - 1
	vertices := make([]float32, 0, cellsX*cellsZ*6*3)
	normals := make([]float32, 0, cellsX*cellsZ*6*3)

	minY := float32(0)
	maxY := float32(0)
	first := true

	appendVertex := func(x, y, z float32, n mgl32.Vec3) {
		vertices = append(vertices, x, y, z)
		normals = append(normals, n.X(), n.Y(), n.Z())
		if first {
			minY = y
			maxY = y
			first = false
		} else {
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}

	for iz := 0; iz < cellsZ; iz++ {
		gz0 := zSamples[iz]
		gz1 := zSamples[iz+1]
		for ix := 0; ix < cellsX; ix++ {
			gx0 := xSamples[ix]
			gx1 := xSamples[ix+1]

			x0, z0 := t.worldFromGrid(gx0, gz0)
			x1, z1 := t.worldFromGrid(gx1, gz1)

			y00 := t.heightAtGrid(gx0, gz0)
			y10 := t.heightAtGrid(gx1, gz0)
			y11 := t.heightAtGrid(gx1, gz1)
			y01 := t.heightAtGrid(gx0, gz1)

			n00 := normalGrid[localNormalIndex(ix, iz)]
			n10 := normalGrid[localNormalIndex(ix+1, iz)]
			n11 := normalGrid[localNormalIndex(ix+1, iz+1)]
			n01 := normalGrid[localNormalIndex(ix, iz+1)]

			appendVertex(x0, y00, z0, n00)
			appendVertex(x1, y10, z0, n10)
			appendVertex(x1, y11, z1, n11)

			appendVertex(x1, y11, z1, n11)
			appendVertex(x0, y01, z1, n01)
			appendVertex(x0, y00, z0, n00)
		}
	}

	return vertices, normals, minY, maxY
}

func buildTerrainSampleAxis(minG, maxG, step int) []int {
	if step < 1 {
		step = 1
	}
	if minG > maxG {
		return nil
	}

	capHint := (maxG-minG)/step + 2
	samples := make([]int, 0, capHint)
	for g := minG; g < maxG; g += step {
		samples = append(samples, g)
	}
	samples = append(samples, maxG)
	return samples
}

// HeightRange РІРѕР·РІСЂР°С‰Р°РµС‚ min/max РІС‹СЃРѕС‚Сѓ РїРѕ РІСЃРµР№ РєР°СЂС‚Рµ РІС‹СЃРѕС‚.
func (t *EditableTerrain) HeightRange() (float32, float32) {
	if t == nil || len(t.Heights) == 0 {
		return 0, 0
	}

	minY := t.Heights[0]
	maxY := t.Heights[0]
	for i := 1; i < len(t.Heights); i++ {
		h := t.Heights[i]
		if h < minY {
			minY = h
		}
		if h > maxY {
			maxY = h
		}
	}
	return minY, maxY
}

func (t *EditableTerrain) Raycast(origin, direction mgl32.Vec3, maxDistance float32) (mgl32.Vec3, bool) {
	if t == nil || t.GridSize < 1 || maxDistance <= 0 {
		return mgl32.Vec3{}, false
	}
	if direction.Len() < 0.0001 {
		return mgl32.Vec3{}, false
	}
	dir := direction.Normalize()

	step := t.CellSize * 0.5
	if step < 0.25 {
		step = 0.25
	}

	prevValid := false
	prevT := float32(0)
	prevF := float32(0)

	for currentT := float32(0); currentT <= maxDistance; currentT += step {
		p := origin.Add(dir.Mul(currentT))
		if !t.InBoundsXZ(p.X(), p.Z()) {
			prevValid = false
			continue
		}

		terrainY := t.HeightAt(p.X(), p.Z())
		currentF := p.Y() - terrainY

		if !prevValid {
			prevValid = true
			prevT = currentT
			prevF = currentF
			continue
		}

		if (prevF >= 0 && currentF <= 0) || (prevF <= 0 && currentF >= 0) {
			hitT := t.raycastRefine(origin, dir, prevT, currentT, prevF)
			hit := origin.Add(dir.Mul(hitT))
			hit[1] = t.HeightAt(hit.X(), hit.Z())
			return hit, true
		}

		prevT = currentT
		prevF = currentF
	}

	return mgl32.Vec3{}, false
}

func (t *EditableTerrain) raycastRefine(origin, dir mgl32.Vec3, startT, endT, startF float32) float32 {
	low := startT
	high := endT
	lowF := startF

	for i := 0; i < 12; i++ {
		mid := (low + high) * 0.5
		p := origin.Add(dir.Mul(mid))
		midF := p.Y() - t.HeightAt(p.X(), p.Z())
		if (lowF >= 0 && midF >= 0) || (lowF <= 0 && midF <= 0) {
			low = mid
			lowF = midF
		} else {
			high = mid
		}
	}

	return (low + high) * 0.5
}

func clamp01f(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func absf32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
