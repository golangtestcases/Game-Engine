package objects

// WaterPlaneVertices строит плотную сетку поверхности воды.
// Высокое разрешение нужно для качественной вершинной деформации волн.
func WaterPlaneVertices(sizeX, sizeZ float32) ([]float32, []float32) {
	gridSize := 384
	cellSizeX := sizeX / float32(gridSize)
	cellSizeZ := sizeZ / float32(gridSize)
	halfX := sizeX / 2
	halfZ := sizeZ / 2

	vertices := make([]float32, 0, gridSize*gridSize*6*3)
	normals := make([]float32, 0, gridSize*gridSize*6*3)

	for i := 0; i < gridSize; i++ {
		for j := 0; j < gridSize; j++ {
			x0 := -halfX + float32(i)*cellSizeX
			x1 := x0 + cellSizeX
			z0 := -halfZ + float32(j)*cellSizeZ
			z1 := z0 + cellSizeZ
			y := float32(0.0)

			vertices = append(vertices, x0, y, z0, x1, y, z1, x1, y, z0)
			vertices = append(vertices, x0, y, z0, x0, y, z1, x1, y, z1)
			normals = append(normals, 0, 1, 0, 0, 1, 0, 0, 1, 0)
			normals = append(normals, 0, 1, 0, 0, 1, 0, 0, 1, 0)
		}
	}

	return vertices, normals
}
