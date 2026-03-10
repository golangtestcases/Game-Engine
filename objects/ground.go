package objects

import "math"

const (
	// Параметры базового рельефа морского дна.
	DefaultGroundGridSize  = 256
	DefaultGroundCellSize  = float32(4.5)
	DefaultGroundBaseY     = float32(-7.4)
	DefaultGroundAmplitude = float32(1.75)

	// Половина размера стандартной сетки дна в world-space.
	DefaultGroundHalfExtent = float32(DefaultGroundGridSize) * DefaultGroundCellSize * 0.5

	// Радиус, в котором плотно размещаются объекты экосистемы.
	DefaultGroundSpawnRadius = float32(210.0)

	groundMacroFreq      = float32(0.0048)
	groundMacroAmp       = float32(1.35)
	groundDuneFreqX      = float32(0.014)
	groundDuneFreqZ      = float32(0.012)
	groundDuneAmp        = float32(0.82)
	groundRidgeFreq      = float32(0.020)
	groundRidgeAmp       = float32(0.44)
	groundDetailFreq     = float32(0.058)
	groundDetailAmp      = float32(0.14)
	groundDepressionFreq = float32(0.010)
	groundDepressionAmp  = float32(0.65)
	groundSlopeX         = float32(-0.0016)
	groundSlopeZ         = float32(0.0009)

	groundEdgeFalloffStart = float32(0.78)
	groundEdgeFalloffEnd   = float32(1.0)
	groundEdgeDropScale    = float32(4.8)
)

func GroundSpawnRadius() float32 {
	maxRadius := DefaultGroundHalfExtent - DefaultGroundCellSize*6
	minRadius := DefaultGroundCellSize * 8
	if maxRadius < minRadius {
		maxRadius = minRadius
	}
	if DefaultGroundSpawnRadius > maxRadius {
		return maxRadius
	}
	return DefaultGroundSpawnRadius
}

// GenerateGroundVertices строит триангулированную сетку дна
// с процедурной высотой в каждой вершине.
func GenerateGroundVertices(gridSize int, cellSize, baseY, amplitude float32) []float32 {
	if gridSize < 1 {
		return nil
	}

	vertices := make([]float32, 0, gridSize*gridSize*6*3)
	half := float32(gridSize) * cellSize * 0.5

	for gz := 0; gz < gridSize; gz++ {
		for gx := 0; gx < gridSize; gx++ {
			x0 := float32(gx)*cellSize - half
			z0 := float32(gz)*cellSize - half
			x1 := x0 + cellSize
			z1 := z0 + cellSize

			y00 := groundHeight(x0, z0, baseY, amplitude)
			y10 := groundHeight(x1, z0, baseY, amplitude)
			y11 := groundHeight(x1, z1, baseY, amplitude)
			y01 := groundHeight(x0, z1, baseY, amplitude)

			// Triangle 1.
			vertices = append(vertices, x0, y00, z0)
			vertices = append(vertices, x1, y10, z0)
			vertices = append(vertices, x1, y11, z1)

			// Triangle 2.
			vertices = append(vertices, x1, y11, z1)
			vertices = append(vertices, x0, y01, z1)
			vertices = append(vertices, x0, y00, z0)
		}
	}

	return vertices
}

func GenerateGroundNormalsForParams(vertices []float32, baseY, amplitude, sampleStep float32) []float32 {
	if sampleStep <= 0 {
		sampleStep = 0.5
	}
	normals := make([]float32, len(vertices))
	for i := 0; i+2 < len(vertices); i += 3 {
		nx, ny, nz := groundNormal(vertices[i], vertices[i+2], baseY, amplitude, sampleStep)
		normals[i] = nx
		normals[i+1] = ny
		normals[i+2] = nz
	}
	return normals
}

func GenerateGroundNormals(vertices []float32) []float32 {
	sampleStep := DefaultGroundCellSize * 0.5
	if sampleStep < 0.5 {
		sampleStep = 0.5
	}
	return GenerateGroundNormalsForParams(vertices, DefaultGroundBaseY, DefaultGroundAmplitude, sampleStep)
}

// GroundHeightAt возвращает высоту дна в точке (x, z) по той же формуле,
// которая используется при генерации сетки.
func GroundHeightAt(x, z, baseY, amplitude float32) float32 {
	return groundHeight(x, z, baseY, amplitude)
}

// groundHeight — основная функция рельефа:
// macro-шум + дюны + гребни + детали + депрессии + уклон + просадка у краев.
func groundHeight(x, z, baseY, amplitude float32) float32 {
	macro := fbm2(x*groundMacroFreq, z*groundMacroFreq, 4, 2.0, 0.5) * groundMacroAmp
	warp := fbm2(x*0.009+37.4, z*0.009-11.3, 3, 2.1, 0.56)

	dunes := float32(math.Sin(float64((x+warp*16.0)*groundDuneFreqX))*0.65+
		math.Cos(float64((z-warp*11.0)*groundDuneFreqZ))*0.55) * groundDuneAmp

	ridgeWave := float32(math.Sin(float64((x-z)*groundRidgeFreq + warp*1.6)))
	ridges := (1.0-float32(math.Abs(float64(ridgeWave))))*2.0 - 1.0
	ridges *= groundRidgeAmp

	detail := fbm2(x*groundDetailFreq+83.1, z*groundDetailFreq-59.7, 2, 2.2, 0.5) * groundDetailAmp

	depressionMask := fbm2(x*groundDepressionFreq-23.8, z*groundDepressionFreq+51.2, 3, 2.1, 0.56)
	depressions := groundMaxf(depressionMask-0.18, 0)
	depressions = -depressions * depressions * groundDepressionAmp * 2.1

	slope := x*groundSlopeX + z*groundSlopeZ
	edgeDrop := groundEdgeDrop(x, z)

	shape := macro + dunes + ridges + detail + depressions + slope - edgeDrop*edgeDrop*groundEdgeDropScale
	return baseY + amplitude*shape
}

// groundNormal оценивает нормаль через центральные разности по высоте.
func groundNormal(x, z, baseY, amplitude, sampleStep float32) (float32, float32, float32) {
	hL := groundHeight(x-sampleStep, z, baseY, amplitude)
	hR := groundHeight(x+sampleStep, z, baseY, amplitude)
	hD := groundHeight(x, z-sampleStep, baseY, amplitude)
	hU := groundHeight(x, z+sampleStep, baseY, amplitude)

	ddx := (hR - hL) / (2.0 * sampleStep)
	ddz := (hU - hD) / (2.0 * sampleStep)

	nx, ny, nz := -ddx, float32(1.0), -ddz
	length := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
	if length <= 0.0001 {
		return 0, 1, 0
	}
	return nx / length, ny / length, nz / length
}

func groundEdgeDrop(x, z float32) float32 {
	if DefaultGroundHalfExtent <= 0 {
		return 0
	}
	edgeDistance := groundMaxf(groundAbsf(x), groundAbsf(z)) / DefaultGroundHalfExtent
	return groundSmoothStep(groundEdgeFalloffStart, groundEdgeFalloffEnd, edgeDistance)
}

func fbm2(x, z float32, octaves int, lacunarity, gain float32) float32 {
	if octaves <= 0 {
		return 0
	}
	sum := float32(0)
	amp := float32(1)
	freq := float32(1)
	ampSum := float32(0)

	for i := 0; i < octaves; i++ {
		sum += valueNoise2D(x*freq, z*freq) * amp
		ampSum += amp
		amp *= gain
		freq *= lacunarity
	}
	if ampSum <= 0 {
		return 0
	}
	return sum / ampSum
}

func valueNoise2D(x, z float32) float32 {
	x0 := float32(math.Floor(float64(x)))
	z0 := float32(math.Floor(float64(z)))
	x1 := x0 + 1
	z1 := z0 + 1

	tx := x - x0
	tz := z - z0
	sx := tx * tx * (3.0 - 2.0*tx)
	sz := tz * tz * (3.0 - 2.0*tz)

	n00 := hash2D(x0, z0)
	n10 := hash2D(x1, z0)
	n01 := hash2D(x0, z1)
	n11 := hash2D(x1, z1)

	nx0 := groundLerp(n00, n10, sx)
	nx1 := groundLerp(n01, n11, sx)
	return groundLerp(nx0, nx1, sz)*2.0 - 1.0
}

func hash2D(x, z float32) float32 {
	n := float64(x*127.1 + z*311.7)
	s := math.Sin(n) * 43758.5453123
	return float32(s - math.Floor(s))
}

func groundLerp(a, b, t float32) float32 {
	return a + (b-a)*t
}

func groundSmoothStep(edge0, edge1, x float32) float32 {
	if edge0 == edge1 {
		if x >= edge1 {
			return 1
		}
		return 0
	}
	t := groundClamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3.0 - 2.0*t)
}

func groundClamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func groundMaxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func groundAbsf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
