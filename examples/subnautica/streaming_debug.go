package main

import "fmt"

// renderStreamingDebugHUD показывает runtime-статистику cell-стриминга и LOD.
func (g *subnauticaGame) renderStreamingDebugHUD(screenW, screenH int) {
	if g == nil || g.worldStreaming == nil || g.console == nil || g.console.renderer == nil {
		return
	}
	if !g.streamingConfig.DebugOverlay {
		return
	}
	if screenW <= 0 || screenH <= 0 {
		return
	}

	stats := g.worldStreaming.Stats()
	lines := []string{
		fmt.Sprintf("CELL: X=%d Z=%d", stats.CurrentCell.X, stats.CurrentCell.Z),
		fmt.Sprintf("LOADED: %d", stats.LoadedCells),
		fmt.Sprintf("ACTIVE: %d", stats.ActiveCells),
		fmt.Sprintf("VISIBLE: %d", stats.VisibleCells),
		fmt.Sprintf(
			"CFG: size=%.1f a=%d v=%d u=%d",
			g.streamingConfig.CellSize,
			g.streamingConfig.ActiveRadius,
			g.streamingConfig.VisibleRadius,
			g.streamingConfig.UnloadRadius,
		),
		fmt.Sprintf(
			"FOG: start=%.0f end=%.0f dens=%.2f",
			g.streamingConfig.FogStart,
			g.streamingConfig.FogEnd,
			g.streamingConfig.FogDensity,
		),
	}
	if g.streamingConfig.DebugLODOverlay {
		lines = append(lines,
			fmt.Sprintf(
				"TERRAIN LOD: N%d M%d F%d SUP%d",
				stats.TerrainLODVisible[cellLODNear],
				stats.TerrainLODVisible[cellLODMid],
				stats.TerrainLODVisible[cellLODFar],
				stats.TerrainSuppressed,
			),
			fmt.Sprintf(
				"DECOR LOD CELLS: N%d M%d F%d SUP%d",
				stats.DecorLODVisible[cellLODNear],
				stats.DecorLODVisible[cellLODMid],
				stats.DecorLODVisible[cellLODFar],
				stats.DecorSuppressed,
			),
			fmt.Sprintf(
				"DECOR INST: N%d M%d F%d",
				stats.DecorInstancesVisible[cellLODNear],
				stats.DecorInstancesVisible[cellLODMid],
				stats.DecorInstancesVisible[cellLODFar],
			),
			fmt.Sprintf(
				"TERRAIN CFG: near %.0f mid %.0f h %.1f step %d/%d",
				g.streamingConfig.TerrainLODNearDistance,
				g.streamingConfig.TerrainLODMidDistance,
				g.streamingConfig.TerrainLODHysteresis,
				g.streamingConfig.TerrainLODMidStep,
				g.streamingConfig.TerrainLODFarStep,
			),
			fmt.Sprintf(
				"DECOR CFG: near %.0f mid %.0f dens %.2f/%.2f",
				g.streamingConfig.DecorLODNearDistance,
				g.streamingConfig.DecorLODMidDistance,
				g.streamingConfig.DecorLODMidDensity,
				g.streamingConfig.DecorLODFarDensity,
			),
		)
	}
	if g.streamingConfig.DebugCellBounds {
		lines = append(lines, "CELL BOUNDS: отрисовка не реализована")
	}
	if g.streamingConfig.DebugLODColors {
		lines = append(lines, "LOD COLORS: ON")
	}

	if g.oceanSystem != nil {
		ocean := g.oceanSystem.DebugState()
		centered := "OFF"
		if ocean.CameraCentered {
			centered = "ON"
		}
		lines = append(lines,
			fmt.Sprintf("OCEAN CENTERED: %s", centered),
			fmt.Sprintf("OCEAN ORIGIN: X %.0f Z %.0f", ocean.Origin.X(), ocean.Origin.Y()),
			fmt.Sprintf("OCEAN TILES: %d LODS %d", ocean.InstanceCount, ocean.ActiveLODs),
			fmt.Sprintf("OCEAN COVER: %.0f FADE %.0f %.0f", ocean.CoverageRadius, ocean.EdgeFadeNear, ocean.EdgeFadeFar),
		)

		lodLine := "OCEAN LOD:"
		maxLOD := ocean.ActiveLODs
		if maxLOD > len(ocean.PerLOD) {
			maxLOD = len(ocean.PerLOD)
		}
		for i := 0; i < maxLOD; i++ {
			lodLine += fmt.Sprintf(" L%d %d", i, ocean.PerLOD[i])
		}
		lines = append(lines, lodLine)
	}

	scale := float32(2.0)
	margin := float32(14.0)
	paddingX := float32(8.0)
	paddingY := float32(6.0)
	lineHeight := 8 * scale

	maxTextWidth := float32(0)
	for i := range lines {
		w := measureOverlayTextWidth(lines[i], scale)
		if w > maxTextWidth {
			maxTextWidth = w
		}
	}

	panelW := maxTextWidth + paddingX*2
	panelH := float32(len(lines))*lineHeight + paddingY*2
	x := float32(screenW) - margin - panelW
	y := margin + 36
	if y+panelH > float32(screenH)-margin {
		y = float32(screenH) - margin - panelH
	}
	if y < margin {
		y = margin
	}
	if x < margin {
		x = margin
	}

	g.console.renderer.Begin(int32(screenW), int32(screenH))
	g.console.renderer.DrawRect(x, y, panelW, panelH, [4]float32{0.02, 0.03, 0.05, 0.70})
	g.console.renderer.DrawRect(x, y, panelW, 1, [4]float32{0.66, 0.79, 0.94, 0.72})
	textY := y + paddingY
	for i := range lines {
		color := [4]float32{0.88, 0.95, 1.0, 1.0}
		if i == 0 {
			color = [4]float32{0.97, 1.0, 0.90, 1.0}
		}
		g.console.renderer.DrawText(lines[i], x+paddingX, textY, scale, color)
		textY += lineHeight
	}
	g.console.renderer.End()
}
