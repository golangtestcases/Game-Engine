package main

import (
	"fmt"

	"subnautica-lite/engine"
	"subnautica-lite/objects"
)

func (g *subnauticaGame) renderEditorHUD(screenW, screenH int, timeSec float32) {
	if g.console == nil || g.console.renderer == nil {
		return
	}
	if !g.editor.Enabled && timeSec > g.editor.statusUntilSec {
		return
	}

	lines := make([]string, 0, 10)
	if g.editor.Enabled {
		g.ensureEditorLightSelectionValid()

		lines = append(lines, "EDITOR MODE: ON")
		lines = append(lines, "TOOL: "+g.editor.PrimaryTool.Label())
		if g.editor.WaterVisible {
			lines = append(lines, "WATER: ON")
		} else {
			lines = append(lines, "WATER: OFF")
		}
		if g.editor.HasTerrainHit {
			lines = append(lines, fmt.Sprintf(
				"HIT: %.1f %.1f %.1f",
				g.editor.HitPosition.X(),
				g.editor.HitPosition.Y(),
				g.editor.HitPosition.Z(),
			))
		} else {
			lines = append(lines, "HIT: NONE")
		}

		if g.editor.PrimaryTool == editorPrimaryToolLights {
			lines = append(lines, "PRESET: "+g.editor.LightPreset.Label())
			lines = append(lines, fmt.Sprintf(
				"RUNTIME POINT LIGHTS: %d/%d",
				g.editor.RuntimePointLightCount,
				engine.MaxLights-1,
			))
			if g.editor.SkippedPointLightCount > 0 {
				lines = append(lines, fmt.Sprintf(
					"OVER LIMIT (INACTIVE): %d",
					g.editor.SkippedPointLightCount,
				))
			}

			if g.editor.SelectedLightIndex >= 0 && g.editor.SelectedLightIndex < len(g.sceneLighting.PointLights) {
				selected := sanitizeScenePointLight(g.sceneLighting.PointLights[g.editor.SelectedLightIndex])
				lines = append(lines, fmt.Sprintf("SELECTED: #%d", g.editor.SelectedLightIndex))
				lines = append(lines, fmt.Sprintf("SEL RANGE: %.1f", selected.Range))
				lines = append(lines, fmt.Sprintf("SEL INTENSITY: %.2f", selected.Intensity))
			} else {
				lines = append(lines, "SELECTED: NONE")
			}

			lines = append(lines, "TAB TOGGLE  T TERRAIN  L LIGHTS")
			lines = append(lines, "V WATER TOGGLE  F OVERVIEW")
			lines = append(lines, "1 SMALL 2 MEDIUM 3 LARGE")
			lines = append(lines, "LMB SELECT/PLACE  CTRL+LMB MOVE")
			lines = append(lines, "DELETE REMOVE SELECTED")
			lines = append(lines, "CTRL S SAVE")
		} else {
			lines = append(lines, "BRUSH: "+g.editor.Tool.Label())
			lines = append(lines, fmt.Sprintf("RADIUS: %.2f", g.editor.BrushRadius))
			lines = append(lines, fmt.Sprintf("STRENGTH: %.2f", g.editor.BrushStrength))
			if g.editor.Tool == objects.TerrainBrushFlatten {
				if g.editor.flattenTargetLocked {
					lines = append(lines, fmt.Sprintf("FLATTEN TARGET: %.2f (LOCKED)", g.editor.flattenTargetHeight))
				} else {
					lines = append(lines, "FLATTEN TARGET: CLICK TO SAMPLE")
				}
			}
			lines = append(lines, "TAB TOGGLE  T TERRAIN  L LIGHTS")
			lines = append(lines, "V WATER TOGGLE  F OVERVIEW")
			lines = append(lines, "LMB APPLY  RMB LOOK")
			lines = append(lines, "1 RAISE 2 LOWER 3 SMOOTH 4 FLATTEN")
			lines = append(lines, "LBRACKET RBRACKET RADIUS")
			lines = append(lines, "MINUS EQUAL STRENGTH")
			lines = append(lines, "FLATTEN TARGET LOCKS ON LMB PRESS")
			lines = append(lines, "CTRL S SAVE")
		}
	} else {
		lines = append(lines, "EDITOR MODE: OFF")
	}

	if timeSec <= g.editor.statusUntilSec && g.editor.statusText != "" {
		lines = append(lines, g.editor.statusText)
	}

	scale := float32(2.0)
	margin := float32(14)
	paddingX := float32(9)
	paddingY := float32(8)
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
	if panelW < 240 {
		panelW = 240
	}

	g.console.renderer.Begin(int32(screenW), int32(screenH))
	g.console.renderer.DrawRect(margin, margin, panelW, panelH, [4]float32{0.02, 0.03, 0.05, 0.76})
	g.console.renderer.DrawRect(margin, margin, panelW, 1, [4]float32{0.64, 0.74, 0.86, 0.72})
	y := margin + paddingY
	for i := range lines {
		color := [4]float32{0.90, 0.96, 1.0, 1.0}
		if i == 0 {
			color = [4]float32{0.96, 1.0, 0.92, 1.0}
		}
		g.console.renderer.DrawText(lines[i], margin+paddingX, y, scale, color)
		y += lineHeight
	}
	g.console.renderer.End()
}
