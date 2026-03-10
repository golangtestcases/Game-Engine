package engine

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/go-gl/mathgl/mgl32"
)

// Константы формата GLB/GLTF, используемые в минимальном загрузчике модели.
const (
	glbMagic               = 0x46546C67
	glbVersion             = 2
	glbChunkTypeJSON       = 0x4E4F534A
	glbChunkTypeBIN        = 0x004E4942
	gltfPrimitiveTriangles = 4
	gltfComponentTypeU8    = 5121
	gltfComponentTypeU16   = 5123
	gltfComponentTypeU32   = 5125
	gltfComponentTypeFloat = 5126
	gltfAccessorTypeScalar = "SCALAR"
	gltfAccessorTypeVec3   = "VEC3"
)

// StaticModel is a GPU-ready static mesh model built from GLB data.
type StaticModel struct {
	Meshes []StaticModelMesh
}

// StaticModelMesh stores one primitive mesh and its local node transform.
type StaticModelMesh struct {
	Mesh           Mesh
	LocalTransform mgl32.Mat4
}

type gltfDocument struct {
	Accessors   []gltfAccessor   `json:"accessors"`
	BufferViews []gltfBufferView `json:"bufferViews"`
	Buffers     []gltfBuffer     `json:"buffers"`
	Meshes      []gltfMesh       `json:"meshes"`
	Nodes       []gltfNode       `json:"nodes"`
	Scenes      []gltfScene      `json:"scenes"`
	Scene       int              `json:"scene"`
}

// gltfAccessor описывает, как читать элементарные данные (позиции/нормали/индексы)
// из бинарного буфера GLTF.
type gltfAccessor struct {
	BufferView    *int   `json:"bufferView"`
	ByteOffset    int    `json:"byteOffset"`
	ComponentType int    `json:"componentType"`
	Count         int    `json:"count"`
	Type          string `json:"type"`
}

type gltfBufferView struct {
	Buffer     int `json:"buffer"`
	ByteOffset int `json:"byteOffset"`
	ByteLength int `json:"byteLength"`
	ByteStride int `json:"byteStride"`
}

type gltfBuffer struct {
	ByteLength int `json:"byteLength"`
}

type gltfMesh struct {
	Primitives []gltfPrimitive `json:"primitives"`
}

type gltfPrimitive struct {
	Attributes map[string]int `json:"attributes"`
	Indices    *int           `json:"indices"`
	Mode       *int           `json:"mode"`
}

type gltfNode struct {
	Mesh        *int      `json:"mesh"`
	Children    []int     `json:"children"`
	Matrix      []float32 `json:"matrix"`
	Translation []float32 `json:"translation"`
	Rotation    []float32 `json:"rotation"`
	Scale       []float32 `json:"scale"`
}

type gltfScene struct {
	Nodes []int `json:"nodes"`
}

// LoadGLBModel loads a static GLB model and uploads mesh data through the current renderer.
func LoadGLBModel(renderer *Renderer, path string) (*StaticModel, error) {
	if renderer == nil {
		return nil, fmt.Errorf("load glb model %q: renderer is nil", path)
	}

	fileData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load glb model %q: %w", path, err)
	}

	jsonChunk, binChunk, err := parseGLB(fileData)
	if err != nil {
		return nil, fmt.Errorf("load glb model %q: %w", path, err)
	}

	var doc gltfDocument
	if err := json.Unmarshal(jsonChunk, &doc); err != nil {
		return nil, fmt.Errorf("load glb model %q: decode gltf json: %w", path, err)
	}

	if len(doc.Buffers) == 0 {
		return nil, fmt.Errorf("load glb model %q: gltf has no buffers", path)
	}
	if len(binChunk) == 0 {
		return nil, fmt.Errorf("load glb model %q: glb has no BIN chunk", path)
	}
	if doc.Buffers[0].ByteLength > len(binChunk) {
		return nil, fmt.Errorf(
			"load glb model %q: BIN chunk too small (%d < %d)",
			path,
			len(binChunk),
			doc.Buffers[0].ByteLength,
		)
	}

	model := &StaticModel{
		Meshes: make([]StaticModelMesh, 0, 4),
	}

	rootNodes := rootNodeIndices(doc)
	for _, nodeIndex := range rootNodes {
		if err := collectModelNode(renderer, model, &doc, binChunk, nodeIndex, mgl32.Ident4()); err != nil {
			return nil, fmt.Errorf("load glb model %q: %w", path, err)
		}
	}

	if len(model.Meshes) == 0 {
		return nil, fmt.Errorf("load glb model %q: no triangle mesh primitives", path)
	}

	return model, nil
}

// parseGLB разбирает бинарный контейнер GLB и извлекает JSON + первый BIN chunk.
// Минимально поддерживается версия 2.0.
func parseGLB(data []byte) ([]byte, []byte, error) {
	if len(data) < 20 {
		return nil, nil, fmt.Errorf("glb file is too small")
	}

	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != glbMagic {
		return nil, nil, fmt.Errorf("invalid glb magic: 0x%08x", magic)
	}

	version := binary.LittleEndian.Uint32(data[4:8])
	if version != glbVersion {
		return nil, nil, fmt.Errorf("unsupported glb version: %d", version)
	}

	totalLength := int(binary.LittleEndian.Uint32(data[8:12]))
	if totalLength > len(data) {
		return nil, nil, fmt.Errorf("glb length header %d exceeds file size %d", totalLength, len(data))
	}
	if totalLength < 20 {
		return nil, nil, fmt.Errorf("invalid glb length: %d", totalLength)
	}

	offset := 12
	var jsonChunk []byte
	var binChunk []byte

	for offset+8 <= totalLength {
		chunkLength := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		chunkType := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8

		if chunkLength < 0 || offset+chunkLength > totalLength {
			return nil, nil, fmt.Errorf("invalid glb chunk length %d", chunkLength)
		}

		chunkData := data[offset : offset+chunkLength]
		offset += chunkLength

		switch chunkType {
		case glbChunkTypeJSON:
			jsonChunk = append([]byte(nil), chunkData...)
		case glbChunkTypeBIN:
			// GLB may contain multiple BIN chunks; use the first one.
			if len(binChunk) == 0 {
				binChunk = append([]byte(nil), chunkData...)
			}
		}
	}

	if len(jsonChunk) == 0 {
		return nil, nil, fmt.Errorf("missing JSON chunk")
	}

	return jsonChunk, binChunk, nil
}

// rootNodeIndices возвращает корневые узлы активной сцены.
// Если сцены не заданы явно, функция пытается вычислить корни по child-связям.
func rootNodeIndices(doc gltfDocument) []int {
	if len(doc.Scenes) > 0 {
		sceneIndex := doc.Scene
		if sceneIndex < 0 || sceneIndex >= len(doc.Scenes) {
			sceneIndex = 0
		}
		if len(doc.Scenes[sceneIndex].Nodes) > 0 {
			return append([]int(nil), doc.Scenes[sceneIndex].Nodes...)
		}
	}

	if len(doc.Nodes) == 0 {
		return nil
	}

	hasParent := make([]bool, len(doc.Nodes))
	for _, node := range doc.Nodes {
		for _, child := range node.Children {
			if child >= 0 && child < len(hasParent) {
				hasParent[child] = true
			}
		}
	}

	roots := make([]int, 0, len(doc.Nodes))
	for i := range doc.Nodes {
		if !hasParent[i] {
			roots = append(roots, i)
		}
	}
	if len(roots) > 0 {
		return roots
	}

	roots = roots[:0]
	for i := range doc.Nodes {
		roots = append(roots, i)
	}
	return roots
}

// collectModelNode рекурсивно обходит node-tree и накапливает примитивы мешей
// с уже умноженной матрицей трансформации.
func collectModelNode(
	renderer *Renderer,
	model *StaticModel,
	doc *gltfDocument,
	binChunk []byte,
	nodeIndex int,
	parentTransform mgl32.Mat4,
) error {
	if nodeIndex < 0 || nodeIndex >= len(doc.Nodes) {
		return fmt.Errorf("node index %d out of range", nodeIndex)
	}

	node := doc.Nodes[nodeIndex]
	localTransform := nodeLocalTransform(node)
	worldTransform := parentTransform.Mul4(localTransform)

	if node.Mesh != nil {
		if err := appendNodeMeshPrimitives(renderer, model, doc, binChunk, *node.Mesh, worldTransform); err != nil {
			return fmt.Errorf("node %d mesh %d: %w", nodeIndex, *node.Mesh, err)
		}
	}

	for _, childIndex := range node.Children {
		if err := collectModelNode(renderer, model, doc, binChunk, childIndex, worldTransform); err != nil {
			return err
		}
	}

	return nil
}

// appendNodeMeshPrimitives извлекает triangle-примитивы узла, разворачивает индексы
// и формирует GPU-ready mesh для рендера без index-buffer.
func appendNodeMeshPrimitives(
	renderer *Renderer,
	model *StaticModel,
	doc *gltfDocument,
	binChunk []byte,
	meshIndex int,
	localTransform mgl32.Mat4,
) error {
	if meshIndex < 0 || meshIndex >= len(doc.Meshes) {
		return fmt.Errorf("mesh index %d out of range", meshIndex)
	}

	mesh := doc.Meshes[meshIndex]
	for primitiveIndex, primitive := range mesh.Primitives {
		mode := gltfPrimitiveTriangles
		if primitive.Mode != nil {
			mode = *primitive.Mode
		}
		if mode != gltfPrimitiveTriangles {
			continue
		}

		positionAccessor, ok := primitive.Attributes["POSITION"]
		if !ok {
			continue
		}

		positions, err := readFloatAccessor(doc, binChunk, positionAccessor, gltfAccessorTypeVec3)
		if err != nil {
			return fmt.Errorf("primitive %d POSITION: %w", primitiveIndex, err)
		}

		var indices []uint32
		if primitive.Indices != nil {
			indices, err = readIndicesAccessor(doc, binChunk, *primitive.Indices)
			if err != nil {
				return fmt.Errorf("primitive %d indices: %w", primitiveIndex, err)
			}
		} else {
			indices = sequentialIndices(len(positions) / 3)
		}

		expandedPositions, err := expandIndexedVec3(positions, indices)
		if err != nil {
			return fmt.Errorf("primitive %d positions: %w", primitiveIndex, err)
		}

		var expandedNormals []float32
		normalAccessor, hasNormals := primitive.Attributes["NORMAL"]
		if hasNormals {
			normals, err := readFloatAccessor(doc, binChunk, normalAccessor, gltfAccessorTypeVec3)
			if err != nil {
				return fmt.Errorf("primitive %d NORMAL: %w", primitiveIndex, err)
			}
			expandedNormals, err = expandIndexedVec3(normals, indices)
			if err != nil {
				return fmt.Errorf("primitive %d normals: %w", primitiveIndex, err)
			}
		} else {
			expandedNormals = generateFlatNormals(expandedPositions)
		}

		gpuMesh := renderer.NewMeshWithNormals(expandedPositions, expandedNormals)
		model.Meshes = append(model.Meshes, StaticModelMesh{
			Mesh:           gpuMesh,
			LocalTransform: localTransform,
		})
	}

	return nil
}

// nodeLocalTransform собирает local transform узла:
// приоритет имеет явная матрица `matrix`, иначе TRS.
func nodeLocalTransform(node gltfNode) mgl32.Mat4 {
	if len(node.Matrix) == 16 {
		var mat mgl32.Mat4
		copy(mat[:], node.Matrix)
		return mat
	}

	translate := mgl32.Ident4()
	if len(node.Translation) == 3 {
		translate = mgl32.Translate3D(node.Translation[0], node.Translation[1], node.Translation[2])
	}

	rotate := mgl32.Ident4()
	if len(node.Rotation) == 4 {
		q := mgl32.Quat{
			W: node.Rotation[3],
			V: mgl32.Vec3{node.Rotation[0], node.Rotation[1], node.Rotation[2]},
		}.Normalize()
		rotate = q.Mat4()
	}

	scale := mgl32.Scale3D(1, 1, 1)
	if len(node.Scale) == 3 {
		scale = mgl32.Scale3D(node.Scale[0], node.Scale[1], node.Scale[2])
	}

	return translate.Mul4(rotate).Mul4(scale)
}

// readFloatAccessor читает float-аксессор (например POSITION/NORMAL) в плоский []float32.
func readFloatAccessor(doc *gltfDocument, binChunk []byte, accessorIndex int, expectedType string) ([]float32, error) {
	accessor, view, start, stride, comps, err := accessorLayout(doc, accessorIndex)
	if err != nil {
		return nil, err
	}

	if expectedType != "" && accessor.Type != expectedType {
		return nil, fmt.Errorf("expected accessor type %s, got %s", expectedType, accessor.Type)
	}
	if accessor.ComponentType != gltfComponentTypeFloat {
		return nil, fmt.Errorf("expected float accessor, got component type %d", accessor.ComponentType)
	}
	if view.Buffer != 0 {
		return nil, fmt.Errorf("buffer index %d is not supported", view.Buffer)
	}

	packedStride := comps * 4
	if stride < packedStride {
		return nil, fmt.Errorf("invalid stride %d for %d components", stride, comps)
	}
	if accessor.Count == 0 {
		return []float32{}, nil
	}
	end := start + (accessor.Count-1)*stride + packedStride
	if end > len(binChunk) {
		return nil, fmt.Errorf("accessor %d exceeds BIN chunk bounds (%d > %d)", accessorIndex, end, len(binChunk))
	}

	out := make([]float32, accessor.Count*comps)
	for i := 0; i < accessor.Count; i++ {
		base := start + i*stride
		for c := 0; c < comps; c++ {
			offset := base + c*4
			bits := binary.LittleEndian.Uint32(binChunk[offset : offset+4])
			out[i*comps+c] = math.Float32frombits(bits)
		}
	}
	return out, nil
}

// readIndicesAccessor читает индексный accessor в uint32.
// Поддерживаются U8/U16/U32.
func readIndicesAccessor(doc *gltfDocument, binChunk []byte, accessorIndex int) ([]uint32, error) {
	accessor, view, start, stride, comps, err := accessorLayout(doc, accessorIndex)
	if err != nil {
		return nil, err
	}
	if accessor.Type != gltfAccessorTypeScalar {
		return nil, fmt.Errorf("index accessor must be SCALAR, got %s", accessor.Type)
	}
	if comps != 1 {
		return nil, fmt.Errorf("index accessor has invalid component count %d", comps)
	}
	if view.Buffer != 0 {
		return nil, fmt.Errorf("buffer index %d is not supported", view.Buffer)
	}

	componentSize := gltfComponentByteSize(accessor.ComponentType)
	if componentSize <= 0 {
		return nil, fmt.Errorf("unsupported index component type %d", accessor.ComponentType)
	}
	if stride < componentSize {
		return nil, fmt.Errorf("invalid index stride %d", stride)
	}
	if accessor.Count == 0 {
		return []uint32{}, nil
	}
	end := start + (accessor.Count-1)*stride + componentSize
	if end > len(binChunk) {
		return nil, fmt.Errorf("accessor %d exceeds BIN chunk bounds (%d > %d)", accessorIndex, end, len(binChunk))
	}

	indices := make([]uint32, accessor.Count)
	for i := 0; i < accessor.Count; i++ {
		base := start + i*stride
		switch accessor.ComponentType {
		case gltfComponentTypeU8:
			indices[i] = uint32(binChunk[base])
		case gltfComponentTypeU16:
			indices[i] = uint32(binary.LittleEndian.Uint16(binChunk[base : base+2]))
		case gltfComponentTypeU32:
			indices[i] = binary.LittleEndian.Uint32(binChunk[base : base+4])
		default:
			return nil, fmt.Errorf("unsupported index component type %d", accessor.ComponentType)
		}
	}
	return indices, nil
}

// accessorLayout валидирует accessor/bufferView и возвращает параметры чтения.
func accessorLayout(
	doc *gltfDocument,
	accessorIndex int,
) (gltfAccessor, gltfBufferView, int, int, int, error) {
	if accessorIndex < 0 || accessorIndex >= len(doc.Accessors) {
		return gltfAccessor{}, gltfBufferView{}, 0, 0, 0, fmt.Errorf("accessor index %d out of range", accessorIndex)
	}

	accessor := doc.Accessors[accessorIndex]
	if accessor.BufferView == nil {
		return gltfAccessor{}, gltfBufferView{}, 0, 0, 0, fmt.Errorf("accessor %d has no bufferView", accessorIndex)
	}
	if *accessor.BufferView < 0 || *accessor.BufferView >= len(doc.BufferViews) {
		return gltfAccessor{}, gltfBufferView{}, 0, 0, 0, fmt.Errorf("bufferView index %d out of range", *accessor.BufferView)
	}

	view := doc.BufferViews[*accessor.BufferView]
	if accessor.Count < 0 {
		return gltfAccessor{}, gltfBufferView{}, 0, 0, 0, fmt.Errorf("accessor %d has negative count", accessorIndex)
	}
	if view.ByteOffset < 0 || view.ByteLength < 0 {
		return gltfAccessor{}, gltfBufferView{}, 0, 0, 0, fmt.Errorf("bufferView %d has invalid range", *accessor.BufferView)
	}
	comps := gltfAccessorComponentCount(accessor.Type)
	if comps <= 0 {
		return gltfAccessor{}, gltfBufferView{}, 0, 0, 0, fmt.Errorf("unsupported accessor type %q", accessor.Type)
	}

	componentSize := gltfComponentByteSize(accessor.ComponentType)
	if componentSize <= 0 {
		return gltfAccessor{}, gltfBufferView{}, 0, 0, 0, fmt.Errorf("unsupported accessor component type %d", accessor.ComponentType)
	}

	packedStride := comps * componentSize
	stride := view.ByteStride
	if stride == 0 {
		stride = packedStride
	}
	if stride < packedStride {
		return gltfAccessor{}, gltfBufferView{}, 0, 0, 0, fmt.Errorf("stride %d is smaller than packed data %d", stride, packedStride)
	}

	start := view.ByteOffset + accessor.ByteOffset
	if start < 0 {
		return gltfAccessor{}, gltfBufferView{}, 0, 0, 0, fmt.Errorf("negative accessor offset")
	}
	if accessor.ByteOffset < 0 || accessor.ByteOffset > view.ByteLength {
		return gltfAccessor{}, gltfBufferView{}, 0, 0, 0, fmt.Errorf("accessor %d byteOffset %d outside bufferView length %d", accessorIndex, accessor.ByteOffset, view.ByteLength)
	}
	if accessor.Count > 0 {
		relativeEnd := accessor.ByteOffset + (accessor.Count-1)*stride + packedStride
		if relativeEnd > view.ByteLength {
			return gltfAccessor{}, gltfBufferView{}, 0, 0, 0, fmt.Errorf(
				"accessor %d exceeds bufferView bounds (%d > %d)",
				accessorIndex,
				relativeEnd,
				view.ByteLength,
			)
		}
	}

	return accessor, view, start, stride, comps, nil
}

func gltfAccessorComponentCount(accessorType string) int {
	switch accessorType {
	case gltfAccessorTypeScalar:
		return 1
	case "VEC2":
		return 2
	case gltfAccessorTypeVec3:
		return 3
	case "VEC4":
		return 4
	case "MAT4":
		return 16
	default:
		return 0
	}
}

func gltfComponentByteSize(componentType int) int {
	switch componentType {
	case gltfComponentTypeU8:
		return 1
	case gltfComponentTypeU16:
		return 2
	case gltfComponentTypeU32, gltfComponentTypeFloat:
		return 4
	default:
		return 0
	}
}

func sequentialIndices(count int) []uint32 {
	out := make([]uint32, count)
	for i := 0; i < count; i++ {
		out[i] = uint32(i)
	}
	return out
}

// expandIndexedVec3 преобразует indexed-данные в fully expanded треугольный буфер.
// Это упрощает дальнейший рендер (gl.DrawArrays).
func expandIndexedVec3(values []float32, indices []uint32) ([]float32, error) {
	if len(values)%3 != 0 {
		return nil, fmt.Errorf("vec3 buffer length %d is not divisible by 3", len(values))
	}

	vertexCount := len(values) / 3
	expanded := make([]float32, len(indices)*3)

	for i, index := range indices {
		if int(index) >= vertexCount {
			return nil, fmt.Errorf("index %d out of range [0, %d)", index, vertexCount)
		}
		src := int(index) * 3
		dst := i * 3
		expanded[dst] = values[src]
		expanded[dst+1] = values[src+1]
		expanded[dst+2] = values[src+2]
	}

	return expanded, nil
}

// generateFlatNormals генерирует плоские нормали по треугольникам,
// когда исходная модель не содержит NORMAL-атрибутов.
func generateFlatNormals(vertices []float32) []float32 {
	normals := make([]float32, len(vertices))
	for i := 0; i+8 < len(vertices); i += 9 {
		ax, ay, az := vertices[i], vertices[i+1], vertices[i+2]
		bx, by, bz := vertices[i+3], vertices[i+4], vertices[i+5]
		cx, cy, cz := vertices[i+6], vertices[i+7], vertices[i+8]

		abx, aby, abz := bx-ax, by-ay, bz-az
		acx, acy, acz := cx-ax, cy-ay, cz-az

		nx := aby*acz - abz*acy
		ny := abz*acx - abx*acz
		nz := abx*acy - aby*acx

		length := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
		if length > 0.000001 {
			nx /= length
			ny /= length
			nz /= length
		}

		for j := 0; j < 3; j++ {
			base := i + j*3
			normals[base] = nx
			normals[base+1] = ny
			normals[base+2] = nz
		}
	}
	return normals
}
