package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

func (g *subnauticaGame) initTerrain(ctx *engine.Context) {
	var terrain *objects.EditableTerrain
	terrain, err := objects.LoadEditableTerrainJSON(terrainSavePath)
	switch {
	case err == nil:
		g.terrainLoadMessage = "terrain: loaded " + terrainSavePath
	case errors.Is(err, os.ErrNotExist):
		terrain = objects.NewDefaultEditableTerrain()
		g.terrainLoadMessage = "terrain: default generated"
	default:
		terrain = objects.NewDefaultEditableTerrain()
		g.terrainLoadMessage = fmt.Sprintf("terrain load failed, using default: %v", err)
	}

	g.terrain = terrain
	g.groundModel = mgl32.Ident4()
	g.rebuildTerrainMesh(ctx)
}

func (g *subnauticaGame) rebuildTerrainMesh(ctx *engine.Context) {
	if g == nil || g.terrain == nil {
		return
	}

	half := g.terrain.HalfExtent
	g.rebuildTerrainMeshInRange(-half, half, -half, half)
}

func (g *subnauticaGame) rebuildTerrainMeshInRange(minX, maxX, minZ, maxZ float32) {
	if g == nil || g.terrain == nil {
		return
	}

	if minX > maxX {
		minX, maxX = maxX, minX
	}
	if minZ > maxZ {
		minZ, maxZ = maxZ, minZ
	}

	minY, maxY := g.terrain.HeightRange()
	g.groundMinY = minY
	g.groundMaxY = maxY

	if g.worldStreaming != nil {
		g.worldStreaming.RebuildLoadedTerrainMeshesInWorldRange(minX, maxX, minZ, maxZ)
	}
}

func (g *subnauticaGame) groundHeightAt(x, z float32) float32 {
	if g != nil && g.terrain != nil {
		return g.terrain.HeightAt(x, z)
	}
	return objects.GroundHeightAt(x, z, objects.DefaultGroundBaseY, objects.DefaultGroundAmplitude)
}

func (g *subnauticaGame) saveTerrain() error {
	if g == nil || g.terrain == nil {
		return errors.New("terrain is not initialized")
	}
	return g.terrain.SaveJSON(terrainSavePath)
}

func (g *subnauticaGame) saveEditorWorld() error {
	if err := g.saveTerrain(); err != nil {
		return fmt.Errorf("save terrain: %w", err)
	}
	if err := saveScenePointLights(sceneLightsSavePath, g.sceneLighting.PointLights); err != nil {
		return fmt.Errorf("save scene lights: %w", err)
	}
	return nil
}

func (g *subnauticaGame) snapGlowingPlantsToTerrain() {
	if g == nil || g.world == nil || g.terrain == nil {
		return
	}

	engine.Each2(g.world, func(_ engine.Entity, tr *objects.Transform, _ *objects.GlowPlant) {
		tr.Position[1] = g.groundHeightAt(tr.Position.X(), tr.Position.Z())
	})
	g.buildGlowPlantInstances()
	if g.worldStreaming != nil {
		g.worldStreaming.AssignGlowPlants(g.glowPlantInstances)
	}
}
