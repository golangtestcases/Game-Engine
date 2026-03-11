package main

import (
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
)

func (g *subnauticaGame) applyEditorLightTool(ctx *engine.Context, inputBlocked bool) {
	if g.editor.PrimaryTool != editorPrimaryToolLights {
		g.editor.leftClickDown = false
		return
	}
	if inputBlocked || ctx == nil || ctx.Window == nil {
		g.editor.leftClickDown = false
		return
	}
	if ctx.Window.GetInputMode(glfw.CursorMode) == glfw.CursorDisabled {
		g.editor.leftClickDown = false
		return
	}

	g.ensureEditorLightSelectionValid()

	if ctx.Window.GetMouseButton(glfw.MouseButtonRight) == glfw.Press {
		g.editor.leftClickDown = ctx.Window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press
		return
	}
	if !mouseButtonPressedOnce(ctx.Window, glfw.MouseButtonLeft, &g.editor.leftClickDown) {
		return
	}
	if !g.editor.HasCursorRay {
		return
	}

	if hitIndex, _, ok := g.pickEditorScenePointLight(g.editor.CursorOrigin, g.editor.CursorDir); ok {
		g.editor.SelectedLightIndex = hitIndex
		g.setEditorStatus("LIGHT SELECTED", ctx.Time+1.2)
		return
	}
	if !g.editor.HasTerrainHit {
		return
	}

	if editorControlDown(ctx.Window) && g.editor.SelectedLightIndex >= 0 {
		g.queueCommand(simulationCommand{
			Kind: simulationCommandMoveScenePointLight,
			MoveScenePointLight: moveScenePointLightCommand{
				LightIndex: g.editor.SelectedLightIndex,
				Position:   simVec3FromMgl(g.editor.HitPosition),
			},
		})
		g.setEditorStatus("LIGHT MOVED", ctx.Time+1.2)
		return
	}

	enabledBefore := g.enabledScenePointLightsCount()
	g.queueCommand(simulationCommand{
		Kind: simulationCommandPlaceScenePointLight,
		PlaceScenePointLight: placeScenePointLightCommand{
			Position: simVec3FromMgl(g.editor.HitPosition),
			Preset:   g.editor.LightPreset,
		},
	})

	g.editor.SelectedLightIndex = len(g.sceneLighting.PointLights)
	if enabledBefore >= engine.MaxLights-1 {
		g.setEditorStatus("LIGHT PLACED (OVER RUNTIME LIMIT)", ctx.Time+2.4)
	} else {
		g.setEditorStatus("LIGHT PLACED", ctx.Time+1.2)
	}
}

func buildEditorLightMarkerMesh() ([]float32, []float32) {
	appendTri := func(
		vertices, normals []float32,
		a, b, c mgl32.Vec3,
	) ([]float32, []float32) {
		ab := b.Sub(a)
		ac := c.Sub(a)
		n := ab.Cross(ac)
		if n.Len() < 0.0001 {
			n = mgl32.Vec3{0, 1, 0}
		} else {
			n = n.Normalize()
		}

		vertices = append(vertices,
			a.X(), a.Y(), a.Z(),
			b.X(), b.Y(), b.Z(),
			c.X(), c.Y(), c.Z(),
		)
		for i := 0; i < 3; i++ {
			normals = append(normals, n.X(), n.Y(), n.Z())
		}
		return vertices, normals
	}

	top := mgl32.Vec3{0, 0.6, 0}
	bottom := mgl32.Vec3{0, -0.6, 0}
	xp := mgl32.Vec3{0.38, 0, 0}
	xn := mgl32.Vec3{-0.38, 0, 0}
	zp := mgl32.Vec3{0, 0, 0.38}
	zn := mgl32.Vec3{0, 0, -0.38}

	vertices := make([]float32, 0, 8*9)
	normals := make([]float32, 0, 8*9)

	vertices, normals = appendTri(vertices, normals, top, xp, zp)
	vertices, normals = appendTri(vertices, normals, top, zp, xn)
	vertices, normals = appendTri(vertices, normals, top, xn, zn)
	vertices, normals = appendTri(vertices, normals, top, zn, xp)
	vertices, normals = appendTri(vertices, normals, bottom, zp, xp)
	vertices, normals = appendTri(vertices, normals, bottom, xn, zp)
	vertices, normals = appendTri(vertices, normals, bottom, zn, xn)
	vertices, normals = appendTri(vertices, normals, bottom, xp, zn)

	return vertices, normals
}

func (g *subnauticaGame) renderEditorLightMarkers(ctx *engine.Context, viewProj mgl32.Mat4, viewPos mgl32.Vec3) {
	if !g.editor.Enabled || g.editorLightMarkerMesh.VertexCount <= 0 {
		return
	}
	if !approxVec3(viewPos, ctx.Camera.Position, 0.0001) {
		return
	}
	if len(g.sceneLighting.PointLights) == 0 {
		return
	}

	lastVAO := uint32(0)
	ctx.Renderer.Use()
	ctx.Renderer.SetViewPos(viewPos)
	ctx.Renderer.SetMaterial(engine.Material{
		Ambient:          mgl32.Vec3{0.90, 0.90, 0.90},
		Diffuse:          mgl32.Vec3{0.25, 0.25, 0.25},
		Specular:         mgl32.Vec3{0.0, 0.0, 0.0},
		Emission:         mgl32.Vec3{0.28, 0.28, 0.28},
		Shininess:        1.0,
		SpecularStrength: 0.0,
	})
	ctx.Renderer.SetHeightGradient(mgl32.Vec3{1, 1, 1}, mgl32.Vec3{1, 1, 1}, 0, 1, 0)

	for i := range g.sceneLighting.PointLights {
		light := sanitizeScenePointLight(g.sceneLighting.PointLights[i])
		color := editorMarkerColorForLight(light)
		if i == g.editor.SelectedLightIndex {
			color = mgl32.Vec3{0.98, 0.92, 0.32}
		}
		if !light.Enabled || light.Intensity <= 0 {
			color = color.Mul(0.4)
		}

		scale := editorMarkerScaleForLight(light)
		yLift := scale * editorLightMarkerLiftMul
		model := mgl32.Translate3D(
			light.Position.X(),
			light.Position.Y()+yLift,
			light.Position.Z(),
		).Mul4(mgl32.Scale3D(scale, scale, scale))

		ctx.Renderer.SetObjectColor(color)
		ctx.Renderer.SetModel(model)
		ctx.Renderer.SetMVP(viewProj.Mul4(model))
		drawMeshWithCachedVAO(g.editorLightMarkerMesh, &lastVAO)
	}
}

func (g *subnauticaGame) enabledScenePointLightsCount() int {
	count := 0
	for i := range g.sceneLighting.PointLights {
		light := sanitizeScenePointLight(g.sceneLighting.PointLights[i])
		if light.Enabled && light.Intensity > 0 {
			count++
		}
	}
	return count
}

func (g *subnauticaGame) ensureEditorLightSelectionValid() {
	if g.editor.SelectedLightIndex < 0 {
		return
	}
	if g.editor.SelectedLightIndex >= len(g.sceneLighting.PointLights) {
		g.editor.SelectedLightIndex = -1
	}
}

func (g *subnauticaGame) pickEditorScenePointLight(rayOrigin, rayDir mgl32.Vec3) (int, float32, bool) {
	bestIndex := -1
	bestDistance := float32(editorRaycastMaxDistance)
	for i := range g.sceneLighting.PointLights {
		light := sanitizeScenePointLight(g.sceneLighting.PointLights[i])
		// Pick the closest marker sphere in ray direction for stable selection.
		markerScale := editorMarkerScaleForLight(light)
		radius := markerScale * editorLightPickRadiusMul
		center := mgl32.Vec3{
			light.Position.X(),
			light.Position.Y() + markerScale*editorLightMarkerLiftMul,
			light.Position.Z(),
		}

		distance, ok := raySphereHitDistance(rayOrigin, rayDir, center, radius)
		if !ok || distance < 0 || distance > bestDistance {
			continue
		}

		bestDistance = distance
		bestIndex = i
	}

	if bestIndex < 0 {
		return -1, 0, false
	}
	return bestIndex, bestDistance, true
}

func raySphereHitDistance(rayOrigin, rayDir, sphereCenter mgl32.Vec3, radius float32) (float32, bool) {
	if radius <= 0 {
		return 0, false
	}
	oc := rayOrigin.Sub(sphereCenter)
	b := oc.Dot(rayDir)
	c := oc.Dot(oc) - radius*radius
	discriminant := b*b - c
	if discriminant < 0 {
		return 0, false
	}

	root := float32(math.Sqrt(float64(discriminant)))
	nearT := -b - root
	if nearT >= 0 {
		return nearT, true
	}
	farT := -b + root
	if farT >= 0 {
		return farT, true
	}
	return 0, false
}
