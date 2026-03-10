package main

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

// glowingPlantsDemo — изолированный пример подсистемы светящихся растений.
type glowingPlantsDemo struct {
	world           *engine.World
	glowPlantMeshes map[objects.PlantType]engine.Mesh
	groundMesh      engine.Mesh
	groundMinY      float32
	groundMaxY      float32
}

func (g *glowingPlantsDemo) Init(ctx *engine.Context) error {
	ctx.Camera.Speed = 5.0
	g.world = engine.NewWorld()

	// Готовим меши для всех типов растений, которые могут спавниться в демо.
	kelpMesh := ctx.GlowRenderer.NewMeshWithNormals(objects.KelpVertices, objects.KelpNormals)
	bushMesh := ctx.GlowRenderer.NewMeshWithNormals(objects.BushVertices, objects.BushNormals)
	coralMesh := ctx.GlowRenderer.NewMeshWithNormals(objects.CoralVertices, objects.CoralNormals)
	g.glowPlantMeshes = map[objects.PlantType]engine.Mesh{
		objects.PlantKelp:  kelpMesh,
		objects.PlantBush:  bushMesh,
		objects.PlantCoral: coralMesh,
	}

	groundVertices := objects.GenerateGroundVertices(
		objects.DefaultGroundGridSize,
		objects.DefaultGroundCellSize,
		objects.DefaultGroundBaseY,
		objects.DefaultGroundAmplitude,
	)
	groundNormals := objects.GenerateGroundNormals(groundVertices)
	g.groundMesh = ctx.Renderer.NewMeshWithNormals(groundVertices, groundNormals)
	g.groundMinY, g.groundMaxY = minMaxY(groundVertices)

	ctx.Camera.Position = mgl32.Vec3{0, 2, 10}

	// Несколько фиксированных экземпляров для наглядной проверки цветов/пульсации.
	objects.SpawnGlowingPlantAdvanced(g.world, mgl32.Vec3{-5, g.groundMinY, 0}, objects.GlowNeonGreen, 2.0, 1.5, objects.PlantKelp, 1.2, 0)
	objects.SpawnGlowingPlantAdvanced(g.world, mgl32.Vec3{-2, g.groundMinY, -3}, objects.GlowNeonBlue, 2.5, 3.0, objects.PlantBush, 1.0, 45)
	objects.SpawnGlowingPlantAdvanced(g.world, mgl32.Vec3{2, g.groundMinY, -2}, objects.GlowNeonPurple, 3.0, 0, objects.PlantCoral, 1.3, 90)
	objects.SpawnGlowingPlantAdvanced(g.world, mgl32.Vec3{5, g.groundMinY, 1}, objects.GlowNeonPink, 2.2, 2.0, objects.PlantKelp, 1.1, 180)
	objects.SpawnGlowingPlantAdvanced(g.world, mgl32.Vec3{0, g.groundMinY, 5}, objects.GlowNeonOrange, 2.8, 2.5, objects.PlantBush, 0.9, 270)
	objects.SpawnGlowingPlantAdvanced(g.world, mgl32.Vec3{-3, g.groundMinY, 4}, objects.GlowNeonCyan, 2.4, 1.0, objects.PlantCoral, 1.0, 135)

	// Массовый случайный спавн для стресс-проверки glow-pass.
	objects.SpawnRandomGlowingPlants(g.world, 50, 40.0)

	return nil
}

func (g *glowingPlantsDemo) Update(ctx *engine.Context) error {
	ctx.Camera.ProcessInput(ctx.Window, ctx.DeltaTime)
	return nil
}

func (g *glowingPlantsDemo) Render(ctx *engine.Context) error {
	view := ctx.Camera.ViewMatrix()

	// 1) Рисуем грунт обычным рендерером.
	ctx.Renderer.Use()
	groundModel := mgl32.Ident4()
	groundMVP := ctx.Projection.Mul4(view).Mul4(groundModel)
	ctx.Renderer.SetMVP(groundMVP)
	ctx.Renderer.SetModel(groundModel)
	ctx.Renderer.SetMaterial(engine.MaterialSand)
	ctx.Renderer.SetObjectColor(mgl32.Vec3{0.3, 0.3, 0.35})
	ctx.Renderer.SetHeightGradient(mgl32.Vec3{0.2, 0.2, 0.25}, mgl32.Vec3{0.4, 0.4, 0.45}, g.groundMinY, g.groundMaxY, 1.0)
	ctx.Renderer.SetFog(mgl32.Vec3{0.05, 0.05, 0.1}, 0.9, 1.0, 0.3, 1.0)
	caustics := engine.DefaultUnderwaterCausticsParams()
	caustics.Intensity = 0
	ctx.Renderer.SetWaterEffects(ctx.Time, caustics)
	ctx.Renderer.SetViewPos(ctx.Camera.Position)
	ctx.Renderer.SetLights([]engine.Light{})
	ctx.Renderer.Draw(g.groundMesh)

	// 2) Поверх рисуем glow-проход растений.
	ctx.GlowRenderer.Use()
	ctx.GlowRenderer.SetTime(ctx.Time)
	ctx.GlowRenderer.SetViewPos(ctx.Camera.Position)

	engine.Each2(g.world, func(_ engine.Entity, tr *objects.Transform, gp *objects.GlowPlant) {
		mesh, ok := g.glowPlantMeshes[gp.Type]
		if !ok {
			return
		}
		plantModel := tr.ModelMatrix()
		plantMVP := ctx.Projection.Mul4(view).Mul4(plantModel)
		ctx.GlowRenderer.SetMVP(plantMVP)
		ctx.GlowRenderer.SetModel(plantModel)
		ctx.GlowRenderer.SetGlowColor(gp.GlowColor)
		ctx.GlowRenderer.SetGlowIntensity(gp.GlowIntensity)
		ctx.GlowRenderer.SetPulseSpeed(gp.PulseSpeed)
		ctx.GlowRenderer.Draw(mesh)
	})

	return nil
}

func main() {
	cfg, err := engine.LoadConfig("config.json")
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	cfg.Window.Title = "Glowing Plants Demo"

	game := &glowingPlantsDemo{}
	if err := engine.Run(cfg, game); err != nil {
		panic(fmt.Errorf("engine run failed: %w", err))
	}
}

// minMaxY возвращает диапазон высот в массиве вершин (x,y,z,...).
func minMaxY(vertices []float32) (float32, float32) {
	if len(vertices) < 3 {
		return 0, 0
	}
	minY := vertices[1]
	maxY := vertices[1]
	for i := 1; i < len(vertices); i += 3 {
		y := vertices[i]
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}
	return minY, maxY
}
