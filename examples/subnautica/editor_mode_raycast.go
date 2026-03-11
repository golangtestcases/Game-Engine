package main

import (
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

func mouseButtonPressedOnce(window *glfw.Window, button glfw.MouseButton, wasDown *bool) bool {
	down := window.GetMouseButton(button) == glfw.Press
	pressed := down && !*wasDown
	*wasDown = down
	return pressed
}

func screenPointToWorldRay(
	mouseX, mouseY float64,
	screenW, screenH int,
	view, projection mgl32.Mat4,
) (mgl32.Vec3, mgl32.Vec3, bool) {
	if screenW <= 0 || screenH <= 0 {
		return mgl32.Vec3{}, mgl32.Vec3{}, false
	}

	x := float32((2.0*mouseX)/float64(screenW) - 1.0)
	y := float32(1.0 - (2.0*mouseY)/float64(screenH))

	invViewProj := projection.Mul4(view).Inv()
	nearClip := mgl32.Vec4{x, y, -1.0, 1.0}
	farClip := mgl32.Vec4{x, y, 1.0, 1.0}

	nearWorld, ok := vec4ProjectToVec3(invViewProj.Mul4x1(nearClip))
	if !ok {
		return mgl32.Vec3{}, mgl32.Vec3{}, false
	}
	farWorld, ok := vec4ProjectToVec3(invViewProj.Mul4x1(farClip))
	if !ok {
		return mgl32.Vec3{}, mgl32.Vec3{}, false
	}

	dir := farWorld.Sub(nearWorld)
	if dir.Len() < 0.0001 {
		return mgl32.Vec3{}, mgl32.Vec3{}, false
	}
	return nearWorld, dir.Normalize(), true
}

func vec4ProjectToVec3(v mgl32.Vec4) (mgl32.Vec3, bool) {
	if math.Abs(float64(v.W())) < 0.000001 {
		return mgl32.Vec3{}, false
	}
	invW := 1 / v.W()
	return mgl32.Vec3{v.X() * invW, v.Y() * invW, v.Z() * invW}, true
}
