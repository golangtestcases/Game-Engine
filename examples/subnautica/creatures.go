package main

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

const (
	creatureModelLeviathan    = "leviathan"
	creatureTemplateLeviathan = "template.leviathan"
	leviathanModelPath        = "assets/models/fish/Leviathan.glb"
)

// initCreatures загружает модели и подготавливает manager.
// Фактический спавн проходит через simulation-команды.
func (g *subnauticaGame) initCreatures(ctx *engine.Context) error {
	if g.world == nil {
		return nil
	}

	g.creatureManager = objects.NewCreatureManager(g.world)
	g.creatureModels = make(map[string]*engine.StaticModel, 1)

	leviathanModel, err := engine.LoadGLBModel(ctx.Renderer, leviathanModelPath)
	if err != nil {
		return fmt.Errorf("load leviathan model: %w", err)
	}
	g.creatureModels[creatureModelLeviathan] = leviathanModel

	return nil
}

// updateCreatures передает runtime-параметры в менеджер поведения существ.
func (g *subnauticaGame) updateCreatures(ctx *engine.Context) {
	if g.creatureManager == nil {
		return
	}

	g.creatureManager.Update(objects.CreatureUpdateParams{
		DeltaTime:  ctx.DeltaTime,
		Time:       ctx.Time,
		PlayerPos:  g.localPlayerPosition(),
		WaterLevel: g.authoritative.WaterLevel,
		GroundHeightY: func(x, z float32) float32 {
			return g.groundHeightAt(x, z)
		},
	})
}

// renderCreatureShadows рендерит только глубину моделей существ в shadow map.
func (g *subnauticaGame) renderCreatureShadows(shader *engine.ShadowShader, lastVAO *uint32) {
	if g.world == nil || len(g.creatureModels) == 0 || shader == nil || lastVAO == nil {
		return
	}

	engine.Each2(g.world, func(_ engine.Entity, tr *objects.Transform, visual *objects.CreatureVisual) {
		model, ok := g.creatureModels[visual.ModelID]
		if !ok {
			return
		}

		baseModel := creatureModelMatrix(*tr, *visual)
		for i := range model.Meshes {
			part := model.Meshes[i]
			shader.SetModel(baseModel.Mul4(part.LocalTransform))
			drawMeshWithCachedVAO(part.Mesh, lastVAO)
		}
	})
}

func (g *subnauticaGame) renderCreatures(
	ctx *engine.Context,
	viewProj mgl32.Mat4,
	viewPos mgl32.Vec3,
	frustum engine.Frustum,
) {
	if g.world == nil || len(g.creatureModels) == 0 {
		return
	}

	maxDistanceSq := float32(0)
	switch g.qualityPreset {
	case engine.GraphicsQualityLow:
		maxDistanceSq = 95 * 95
	case engine.GraphicsQualityMedium:
		maxDistanceSq = 130 * 130
	}

	lastVAO := uint32(0)
	engine.Each2(g.world, func(_ engine.Entity, tr *objects.Transform, visual *objects.CreatureVisual) {
		model, ok := g.creatureModels[visual.ModelID]
		if !ok {
			return
		}

		radius := visual.BoundingRadius * tr.Scale
		if radius <= 0 {
			radius = 1
		}
		if !frustum.ContainsSphere(tr.Position, radius) {
			return
		}
		if maxDistanceSq > 0 {
			delta := tr.Position.Sub(viewPos)
			if delta.Dot(delta) > maxDistanceSq {
				return
			}
		}

		creatureModel := creatureModelMatrix(*tr, *visual)
		ctx.Renderer.SetMaterial(visual.Material)
		ctx.Renderer.SetObjectColor(visual.Color)
		ctx.Renderer.SetHeightGradient(visual.Color, visual.Color, 0, 1, 0)

		for i := range model.Meshes {
			part := model.Meshes[i]
			modelMatrix := creatureModel.Mul4(part.LocalTransform)
			ctx.Renderer.SetModel(modelMatrix)
			ctx.Renderer.SetMVP(viewProj.Mul4(modelMatrix))
			drawMeshWithCachedVAO(part.Mesh, &lastVAO)
		}
	})
}

// creatureModelMatrix возвращает итоговую матрицу модели существа
// с дополнительным yaw-offset для корректной ориентации импортированных ассетов.
func creatureModelMatrix(tr objects.Transform, visual objects.CreatureVisual) mgl32.Mat4 {
	model := tr.ModelMatrix()
	if absf(visual.YawOffsetDeg) > 0.0001 {
		model = model.Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(visual.YawOffsetDeg)))
	}
	return model
}
