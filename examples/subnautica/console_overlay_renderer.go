package main

import "subnautica-lite/engine"

// consoleOverlayRenderer reuses an engine-level debug overlay renderer.
type consoleOverlayRenderer = engine.DebugOverlayTextRenderer

func newConsoleOverlayRenderer() *consoleOverlayRenderer {
	return engine.NewDebugOverlayTextRenderer()
}
