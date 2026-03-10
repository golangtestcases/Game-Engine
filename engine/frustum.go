package engine

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// Frustum хранит 6 плоскостей отсечения, извлечённых из view-projection матрицы.
// Плоскости нормализуются, чтобы корректно проверять сферы по радиусу.
type Frustum struct {
	planes [6]mgl32.Vec4
}

// NewFrustum извлекает плоскости в порядке: left, right, bottom, top, near, far.
func NewFrustum(viewProj mgl32.Mat4) Frustum {
	row0 := mgl32.Vec4{viewProj[0], viewProj[4], viewProj[8], viewProj[12]}
	row1 := mgl32.Vec4{viewProj[1], viewProj[5], viewProj[9], viewProj[13]}
	row2 := mgl32.Vec4{viewProj[2], viewProj[6], viewProj[10], viewProj[14]}
	row3 := mgl32.Vec4{viewProj[3], viewProj[7], viewProj[11], viewProj[15]}

	return Frustum{
		planes: [6]mgl32.Vec4{
			normalizePlane(row3.Add(row0)), // left
			normalizePlane(row3.Sub(row0)), // right
			normalizePlane(row3.Add(row1)), // bottom
			normalizePlane(row3.Sub(row1)), // top
			normalizePlane(row3.Add(row2)), // near
			normalizePlane(row3.Sub(row2)), // far
		},
	}
}

// ContainsSphere проверяет пересечение сферы с фрустумом.
func (f Frustum) ContainsSphere(center mgl32.Vec3, radius float32) bool {
	return f.ContainsSphereXYZ(center.X(), center.Y(), center.Z(), radius)
}

// ContainsSphereXYZ — версия без выделений и без временных векторов,
// используется в горячих местах при culling.
func (f Frustum) ContainsSphereXYZ(x, y, z, radius float32) bool {
	for i := 0; i < len(f.planes); i++ {
		p := f.planes[i]
		if p.X()*x+p.Y()*y+p.Z()*z+p.W() < -radius {
			return false
		}
	}
	return true
}

func normalizePlane(plane mgl32.Vec4) mgl32.Vec4 {
	length := float32(math.Sqrt(float64(plane.X()*plane.X() + plane.Y()*plane.Y() + plane.Z()*plane.Z())))
	if length < 0.000001 {
		return plane
	}
	inv := 1.0 / length
	return mgl32.Vec4{plane.X() * inv, plane.Y() * inv, plane.Z() * inv, plane.W() * inv}
}
