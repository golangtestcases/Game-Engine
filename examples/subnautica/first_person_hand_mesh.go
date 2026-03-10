package main

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// firstPersonHandMeshData — плоские массивы, готовые для загрузки в GPU.
type firstPersonHandMeshData struct {
	vertices []float32
	normals  []float32
}

// fpDigitSpec описывает процедуру построения одного «пальца»/сегмента.
type fpDigitSpec struct {
	base        mgl32.Vec3
	direction   mgl32.Vec3
	length      float32
	baseRadiusX float32
	baseRadiusY float32
	tipRadiusX  float32
	tipRadiusY  float32
	bend        float32
	rootInset   float32
	bendAxis    mgl32.Vec3
	segments    int
	sides       int
}

type fpHandMeshBuilder struct {
	positions []mgl32.Vec3
	indices   []uint32
}

// buildProceduralFirstPersonForearmMesh создает низкополигональное предплечье из колец.
func buildProceduralFirstPersonForearmMesh() firstPersonHandMeshData {
	builder := &fpHandMeshBuilder{}

	ring0 := builder.addRingXY(mgl32.Vec3{0.000, -0.012, 0.118}, 0.040, 0.033, 12, 0.00)
	ring1 := builder.addRingXY(mgl32.Vec3{0.000, -0.010, 0.078}, 0.037, 0.030, 12, 0.08)
	ring2 := builder.addRingXY(mgl32.Vec3{0.000, -0.007, 0.040}, 0.033, 0.027, 12, 0.12)
	ring3 := builder.addRingXY(mgl32.Vec3{0.000, -0.005, 0.012}, 0.028, 0.023, 12, 0.15)

	builder.bridgeRings(ring0, ring1)
	builder.bridgeRings(ring1, ring2)
	builder.bridgeRings(ring2, ring3)
	builder.capRing(ring0)
	builder.capRing(ring3)

	return builder.build()
}

// buildProceduralFirstPersonHandMesh строит кисть; mirrorX используется для левой руки.
func buildProceduralFirstPersonHandMesh(mirrorX bool) firstPersonHandMeshData {
	builder := &fpHandMeshBuilder{}

	wrist := builder.addPalmRing(mgl32.Vec3{0.000, -0.007, 0.024}, 0.026, 0.020, 0.18, 0.12, 12)
	midPalm := builder.addPalmRing(mgl32.Vec3{0.000, -0.002, -0.020}, 0.040, 0.024, 0.29, 0.18, 12)
	knuckles := builder.addPalmRing(mgl32.Vec3{0.000, 0.004, -0.074}, 0.045, 0.021, 0.24, 0.19, 12)
	handTipBase := builder.addPalmRing(mgl32.Vec3{-0.001, 0.006, -0.100}, 0.034, 0.015, 0.10, 0.10, 12)
	handTip := builder.addPalmRing(mgl32.Vec3{-0.001, 0.004, -0.117}, 0.018, 0.010, 0.06, 0.06, 12)

	builder.bridgeRings(wrist, midPalm)
	builder.bridgeRings(midPalm, knuckles)
	builder.bridgeRings(knuckles, handTipBase)
	builder.bridgeRings(handTipBase, handTip)
	builder.capRing(wrist)
	builder.capRing(handTip)

	if mirrorX {
		builder.mirrorX()
	}

	return builder.build()
}

// addDigit строит сегментированную трубчатую геометрию вдоль направления с изгибом.
func (b *fpHandMeshBuilder) addDigit(spec fpDigitSpec) {
	if spec.segments < 1 {
		spec.segments = 1
	}
	if spec.sides < 4 {
		spec.sides = 4
	}

	dir := fpSafeNormalize(spec.direction, mgl32.Vec3{0, 0, -1})
	down := mgl32.Vec3{0, -1, 0}
	bendReference := spec.bendAxis
	if bendReference.Len() < 0.0001 {
		bendReference = down
	}
	bendAxis := bendReference.Sub(dir.Mul(bendReference.Dot(dir)))
	if bendAxis.Len() < 0.0001 {
		bendAxis = down.Sub(dir.Mul(down.Dot(dir)))
	}
	bendAxis = fpSafeNormalize(bendAxis, down)
	base := spec.base
	travel := spec.length
	if spec.rootInset > 0 {
		rootInset := spec.rootInset
		maxInset := spec.length * 0.45
		if rootInset > maxInset {
			rootInset = maxInset
		}
		base = base.Sub(dir.Mul(rootInset))
		travel += rootInset
	}

	rings := make([][]uint32, spec.segments+1)
	for i := 0; i <= spec.segments; i++ {
		t := float32(i) / float32(spec.segments)
		center := base.
			Add(dir.Mul(travel * t)).
			Add(bendAxis.Mul(spec.bend * t * t))
		rx := fpLerp(spec.baseRadiusX, spec.tipRadiusX, t)
		ry := fpLerp(spec.baseRadiusY, spec.tipRadiusY, t)
		if i == 0 {
			rx *= 1.08
			ry *= 1.10
		}
		rings[i] = b.addRingOriented(center, dir, rx, ry, spec.sides)
	}

	for i := 0; i < spec.segments; i++ {
		b.bridgeRings(rings[i], rings[i+1])
	}
	b.capRing(rings[0])
	b.capRing(rings[len(rings)-1])
}

func (b *fpHandMeshBuilder) addRingXY(center mgl32.Vec3, radiusX, radiusY float32, sides int, twist float32) []uint32 {
	if sides < 3 {
		sides = 3
	}
	ring := make([]uint32, sides)
	step := float32(2 * math.Pi / float64(sides))
	for i := 0; i < sides; i++ {
		a := step*float32(i) + twist
		x := float32(math.Cos(float64(a))) * radiusX
		y := float32(math.Sin(float64(a))) * radiusY
		ring[i] = b.addVertex(center.Add(mgl32.Vec3{x, y, 0}))
	}
	return ring
}

func (b *fpHandMeshBuilder) addPalmRing(
	center mgl32.Vec3,
	radiusX, radiusY, thumbBulge, pinkyBulge float32,
	sides int,
) []uint32 {
	if sides < 3 {
		sides = 3
	}
	ring := make([]uint32, sides)
	step := float32(2 * math.Pi / float64(sides))
	for i := 0; i < sides; i++ {
		a := step * float32(i)
		c := float32(math.Cos(float64(a)))
		s := float32(math.Sin(float64(a)))

		x := c * radiusX
		y := s * radiusY
		absSin := float32(math.Abs(float64(s)))

		if c >= 0 {
			x *= 1.0 + thumbBulge*(0.40+0.60*(1.0-absSin))
		} else {
			x *= 1.0 + pinkyBulge*(0.30+0.70*(1.0-absSin))
		}

		if s < 0 {
			y *= 0.72
		} else {
			y *= 1.08
		}

		ring[i] = b.addVertex(center.Add(mgl32.Vec3{x, y, 0}))
	}
	return ring
}

func (b *fpHandMeshBuilder) addRingOriented(
	center mgl32.Vec3,
	direction mgl32.Vec3,
	radiusX, radiusY float32,
	sides int,
) []uint32 {
	if sides < 3 {
		sides = 3
	}

	dir := fpSafeNormalize(direction, mgl32.Vec3{0, 0, -1})
	side := mgl32.Vec3{0, 1, 0}.Cross(dir)
	if side.Len() < 0.0001 {
		side = mgl32.Vec3{1, 0, 0}.Cross(dir)
	}
	side = fpSafeNormalize(side, mgl32.Vec3{1, 0, 0})
	up := fpSafeNormalize(dir.Cross(side), mgl32.Vec3{0, 1, 0})

	ring := make([]uint32, sides)
	step := float32(2 * math.Pi / float64(sides))
	for i := 0; i < sides; i++ {
		a := step * float32(i)
		c := float32(math.Cos(float64(a)))
		s := float32(math.Sin(float64(a)))
		offset := side.Mul(c * radiusX).Add(up.Mul(s * radiusY))
		ring[i] = b.addVertex(center.Add(offset))
	}
	return ring
}

func (b *fpHandMeshBuilder) bridgeRings(a, c []uint32) {
	if len(a) < 3 || len(a) != len(c) {
		return
	}
	for i := 0; i < len(a); i++ {
		next := (i + 1) % len(a)
		a0 := a[i]
		a1 := a[next]
		c0 := c[i]
		c1 := c[next]
		b.addTriangle(a0, c0, c1)
		b.addTriangle(a0, c1, a1)
	}
}

func (b *fpHandMeshBuilder) capRing(ring []uint32) {
	if len(ring) < 3 {
		return
	}
	center := mgl32.Vec3{}
	for _, idx := range ring {
		center = center.Add(b.positions[idx])
	}
	center = center.Mul(1.0 / float32(len(ring)))
	centerIdx := b.addVertex(center)
	for i := 0; i < len(ring); i++ {
		next := (i + 1) % len(ring)
		b.addTriangle(centerIdx, ring[i], ring[next])
	}
}

func (b *fpHandMeshBuilder) addVertex(v mgl32.Vec3) uint32 {
	b.positions = append(b.positions, v)
	return uint32(len(b.positions) - 1)
}

func (b *fpHandMeshBuilder) addTriangle(a, c, d uint32) {
	b.indices = append(b.indices, a, c, d)
}

func (b *fpHandMeshBuilder) mirrorX() {
	for i := range b.positions {
		b.positions[i][0] = -b.positions[i][0]
	}
}

func (b *fpHandMeshBuilder) orientTrianglesOutward() {
	if len(b.indices) < 3 || len(b.positions) == 0 {
		return
	}

	meshCenter := mgl32.Vec3{}
	for i := range b.positions {
		meshCenter = meshCenter.Add(b.positions[i])
	}
	meshCenter = meshCenter.Mul(1.0 / float32(len(b.positions)))

	for i := 0; i+2 < len(b.indices); i += 3 {
		ia := b.indices[i]
		ib := b.indices[i+1]
		ic := b.indices[i+2]
		a := b.positions[ia]
		c := b.positions[ib]
		d := b.positions[ic]
		n := c.Sub(a).Cross(d.Sub(a))
		if n.Len() < 0.000001 {
			continue
		}
		centroid := a.Add(c).Add(d).Mul(1.0 / 3.0)
		if n.Dot(centroid.Sub(meshCenter)) < 0 {
			b.indices[i+1], b.indices[i+2] = b.indices[i+2], b.indices[i+1]
		}
	}
}

// build финализирует меш: ориентирует треугольники, считает сглаженные нормали
// и разворачивает индексы в плоские vertex/normal буферы.
func (b *fpHandMeshBuilder) build() firstPersonHandMeshData {
	b.orientTrianglesOutward()

	smoothed := make([]mgl32.Vec3, len(b.positions))
	for i := 0; i+2 < len(b.indices); i += 3 {
		ia := b.indices[i]
		ib := b.indices[i+1]
		ic := b.indices[i+2]
		a := b.positions[ia]
		c := b.positions[ib]
		d := b.positions[ic]
		face := c.Sub(a).Cross(d.Sub(a))
		if face.Len() < 0.000001 {
			continue
		}
		smoothed[ia] = smoothed[ia].Add(face)
		smoothed[ib] = smoothed[ib].Add(face)
		smoothed[ic] = smoothed[ic].Add(face)
	}

	for i := range smoothed {
		smoothed[i] = fpSafeNormalize(smoothed[i], mgl32.Vec3{0, 1, 0})
	}

	vertices := make([]float32, 0, len(b.indices)*3)
	normals := make([]float32, 0, len(b.indices)*3)
	for _, idx := range b.indices {
		v := b.positions[idx]
		n := smoothed[idx]
		vertices = append(vertices, v.X(), v.Y(), v.Z())
		normals = append(normals, n.X(), n.Y(), n.Z())
	}

	return firstPersonHandMeshData{
		vertices: vertices,
		normals:  normals,
	}
}

func fpSafeNormalize(v, fallback mgl32.Vec3) mgl32.Vec3 {
	if v.Len() < 0.000001 {
		if fallback.Len() < 0.000001 {
			return mgl32.Vec3{0, 1, 0}
		}
		return fallback.Normalize()
	}
	return v.Normalize()
}

func fpLerp(a, c, t float32) float32 {
	return a + (c-a)*t
}
