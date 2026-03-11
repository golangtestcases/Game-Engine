package main

import (
	"fmt"
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

func (g *subnauticaGame) applyEditorBrush(ctx *engine.Context, inputBlocked bool) {
	if g.editor.PrimaryTool != editorPrimaryToolTerrain {
		g.resetEditorTerrainDragState()
		return
	}
	if ctx == nil || ctx.Window == nil {
		g.resetEditorTerrainDragState()
		return
	}

	leftDown := ctx.Window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press
	pressedOnce := leftDown && !g.editor.terrainLeftClickDown
	g.editor.terrainLeftClickDown = leftDown
	if !leftDown {
		g.editor.flattenTargetLocked = false
	}

	if inputBlocked || g.terrain == nil || !g.editor.HasTerrainHit {
		return
	}
	if ctx.Window.GetMouseButton(glfw.MouseButtonRight) == glfw.Press {
		return
	}
	if !leftDown {
		return
	}

	flattenTarget := g.editor.HitPosition.Y()
	if g.editor.Tool == objects.TerrainBrushFlatten {
		// Lock target height when starting LMB hold to keep flatten stable across frames.
		if pressedOnce || !g.editor.flattenTargetLocked {
			g.editor.flattenTargetHeight = g.editor.HitPosition.Y()
			g.editor.flattenTargetLocked = true
			g.setEditorStatus(fmt.Sprintf("FLATTEN TARGET: %.2f", g.editor.flattenTargetHeight), ctx.Time+1.2)
		}
		flattenTarget = g.editor.flattenTargetHeight
	}

	g.queueCommand(simulationCommand{
		Kind: simulationCommandApplyTerrainBrush,
		TerrainBrush: terrainBrushCommand{
			Center:        simVec3FromMgl(g.editor.HitPosition),
			Radius:        g.editor.BrushRadius,
			Strength:      g.editor.BrushStrength,
			DeltaTime:     ctx.DeltaTime,
			Tool:          g.editor.Tool,
			FlattenTarget: flattenTarget,
		},
	})
}

func (g *subnauticaGame) updateEditorBrushPreviewMesh(ctx *engine.Context) {
	if ctx == nil || g.editorBrushMesh.VAO == 0 {
		return
	}
	if !g.editor.Enabled || g.editor.PrimaryTool != editorPrimaryToolTerrain || !g.editor.HasTerrainHit || g.terrain == nil {
		ctx.Renderer.UpdateMeshWithNormals(&g.editorBrushMesh, nil, nil)
		return
	}

	vertices, normals := buildBrushPreviewMesh(
		g.terrain,
		g.editor.HitPosition.X(),
		g.editor.HitPosition.Z(),
		g.editor.BrushRadius,
		g.editor.BrushStrength,
		g.editor.Tool,
		g.editor.flattenTargetHeight,
		g.editor.flattenTargetLocked,
	)
	ctx.Renderer.UpdateMeshWithNormals(&g.editorBrushMesh, vertices, normals)
}

func buildBrushPreviewMesh(
	terrain *objects.EditableTerrain,
	centerX, centerZ, radius, strength float32,
	tool objects.TerrainBrushTool,
	flattenTarget float32,
	flattenTargetLocked bool,
) ([]float32, []float32) {
	if terrain == nil || radius <= 0 {
		return nil, nil
	}

	appendProjectedRing := func(vertices, normals []float32, outerRadius, thickness float32, segments int) ([]float32, []float32) {
		if outerRadius <= 0 || thickness <= 0 || segments < 3 {
			return vertices, normals
		}

		innerRadius := outerRadius - thickness
		if innerRadius < 0.02 {
			innerRadius = outerRadius * 0.6
		}
		if innerRadius <= 0 {
			return vertices, normals
		}

		for i := 0; i < segments; i++ {
			a0 := float32(i) / float32(segments) * 2 * math.Pi
			a1 := float32(i+1) / float32(segments) * 2 * math.Pi

			c0 := float32(math.Cos(float64(a0)))
			s0 := float32(math.Sin(float64(a0)))
			c1 := float32(math.Cos(float64(a1)))
			s1 := float32(math.Sin(float64(a1)))

			o0x, o0z := centerX+c0*outerRadius, centerZ+s0*outerRadius
			o1x, o1z := centerX+c1*outerRadius, centerZ+s1*outerRadius
			i0x, i0z := centerX+c0*innerRadius, centerZ+s0*innerRadius
			i1x, i1z := centerX+c1*innerRadius, centerZ+s1*innerRadius

			o0y := terrain.HeightAt(o0x, o0z) + editorBrushPreviewYOffset
			o1y := terrain.HeightAt(o1x, o1z) + editorBrushPreviewYOffset
			i0y := terrain.HeightAt(i0x, i0z) + editorBrushPreviewYOffset
			i1y := terrain.HeightAt(i1x, i1z) + editorBrushPreviewYOffset

			vertices = append(vertices,
				o0x, o0y, o0z,
				o1x, o1y, o1z,
				i1x, i1y, i1z,

				o0x, o0y, o0z,
				i1x, i1y, i1z,
				i0x, i0y, i0z,
			)

			for n := 0; n < 6; n++ {
				normals = append(normals, 0, 1, 0)
			}
		}

		return vertices, normals
	}

	appendFlatRing := func(vertices, normals []float32, y, outerRadius, thickness float32, segments int) ([]float32, []float32) {
		if outerRadius <= 0 || thickness <= 0 || segments < 3 {
			return vertices, normals
		}
		innerRadius := outerRadius - thickness
		if innerRadius <= 0 {
			return vertices, normals
		}

		for i := 0; i < segments; i++ {
			a0 := float32(i) / float32(segments) * 2 * math.Pi
			a1 := float32(i+1) / float32(segments) * 2 * math.Pi
			c0 := float32(math.Cos(float64(a0)))
			s0 := float32(math.Sin(float64(a0)))
			c1 := float32(math.Cos(float64(a1)))
			s1 := float32(math.Sin(float64(a1)))

			o0x, o0z := centerX+c0*outerRadius, centerZ+s0*outerRadius
			o1x, o1z := centerX+c1*outerRadius, centerZ+s1*outerRadius
			i0x, i0z := centerX+c0*innerRadius, centerZ+s0*innerRadius
			i1x, i1z := centerX+c1*innerRadius, centerZ+s1*innerRadius

			vertices = append(vertices,
				o0x, y, o0z,
				o1x, y, o1z,
				i1x, y, i1z,

				o0x, y, o0z,
				i1x, y, i1z,
				i0x, y, i0z,
			)
			for n := 0; n < 6; n++ {
				normals = append(normals, 0, 1, 0)
			}
		}
		return vertices, normals
	}

	strengthNorm := clamp01((strength - editorBrushStrengthMin) / (editorBrushStrengthMax - editorBrushStrengthMin))
	mainThickness := radius * (0.055 + 0.055*strengthNorm)
	if mainThickness < 0.08 {
		mainThickness = 0.08
	}
	if mainThickness > radius*0.32 {
		mainThickness = radius * 0.32
	}

	centerRadius := radius * 0.15
	if centerRadius < 0.14 {
		centerRadius = 0.14
	}
	centerThickness := centerRadius * 0.45

	midRadius := radius * 0.66
	if midRadius < centerRadius*1.35 {
		midRadius = centerRadius * 1.35
	}
	innerRadius := radius * 0.34
	if innerRadius < centerRadius*1.05 {
		innerRadius = centerRadius * 1.05
	}

	midThickness := mainThickness * 0.68
	innerThickness := mainThickness * 0.56
	vertices := make([]float32, 0, 56*6*3+42*6*3+30*6*3+20*6*3)
	normals := make([]float32, 0, 56*6*3+42*6*3+30*6*3+20*6*3)
	vertices, normals = appendProjectedRing(vertices, normals, radius, mainThickness, 56)
	vertices, normals = appendProjectedRing(vertices, normals, midRadius, midThickness, 42)
	vertices, normals = appendProjectedRing(vertices, normals, innerRadius, innerThickness, 30)
	vertices, normals = appendProjectedRing(vertices, normals, centerRadius, centerThickness, 20)

	if tool == objects.TerrainBrushFlatten && flattenTargetLocked {
		targetY := flattenTarget + editorBrushPreviewYOffset*1.8
		flatRadius := radius * 0.86
		if flatRadius < 0.24 {
			flatRadius = 0.24
		}
		flatThickness := mainThickness * 0.5
		if flatThickness < 0.05 {
			flatThickness = 0.05
		}
		vertices, normals = appendFlatRing(vertices, normals, targetY, flatRadius, flatThickness, 56)
	}
	return vertices, normals
}

func (g *subnauticaGame) renderEditorBrushPreview(ctx *engine.Context, viewProj mgl32.Mat4, viewPos mgl32.Vec3) {
	if !g.editor.Enabled || g.editor.PrimaryTool != editorPrimaryToolTerrain || !g.editor.HasTerrainHit || g.editorBrushMesh.VertexCount <= 0 {
		return
	}
	if !approxVec3(viewPos, ctx.Camera.Position, 0.0001) {
		return
	}

	color := editorToolColor(g.editor.Tool)
	strengthNorm := clamp01((g.editor.BrushStrength - editorBrushStrengthMin) / (editorBrushStrengthMax - editorBrushStrengthMin))
	color = color.Mul(0.72 + 0.28*strengthNorm)
	lastVAO := uint32(0)

	ctx.Renderer.Use()
	ctx.Renderer.SetViewPos(viewPos)
	ctx.Renderer.SetMaterial(engine.Material{
		Ambient:          mgl32.Vec3{0.92, 0.92, 0.92},
		Diffuse:          mgl32.Vec3{0.30, 0.30, 0.30},
		Specular:         mgl32.Vec3{0.0, 0.0, 0.0},
		Shininess:        1.0,
		SpecularStrength: 0.0,
	})
	ctx.Renderer.SetObjectColor(color)
	ctx.Renderer.SetHeightGradient(color, color, 0, 1, 0)
	ctx.Renderer.SetModel(mgl32.Ident4())
	ctx.Renderer.SetMVP(viewProj)
	drawMeshWithCachedVAO(g.editorBrushMesh, &lastVAO)
}

func editorToolColor(tool objects.TerrainBrushTool) mgl32.Vec3 {
	switch tool {
	case objects.TerrainBrushLower:
		return mgl32.Vec3{0.95, 0.34, 0.30}
	case objects.TerrainBrushSmooth:
		return mgl32.Vec3{0.35, 0.76, 0.98}
	case objects.TerrainBrushFlatten:
		return mgl32.Vec3{0.98, 0.82, 0.34}
	default:
		return mgl32.Vec3{0.34, 0.96, 0.50}
	}
}
