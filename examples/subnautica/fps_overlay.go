package main

import "fmt"

const (
	fpsUpdateInterval = 0.25

	fpsOverlayScale    = float32(2.0)
	fpsOverlayMargin   = float32(14.0)
	fpsOverlayPaddingX = float32(8.0)
	fpsOverlayPaddingY = float32(6.0)
)

// fpsOverlay рисует компактный индикатор текущего FPS.
type fpsOverlay struct {
	renderer *consoleOverlayRenderer

	sampleTime   float32
	sampleFrames int
	label        string
}

func newFPSOverlay(renderer *consoleOverlayRenderer) *fpsOverlay {
	return &fpsOverlay{
		renderer: renderer,
		label:    "FPS: --",
	}
}

// Update усредняет FPS на коротком интервале,
// чтобы число в оверлее не «дрожало» каждый кадр.
func (o *fpsOverlay) Update(deltaTime float32) {
	if o == nil || deltaTime <= 0 {
		return
	}

	o.sampleTime += deltaTime
	o.sampleFrames++

	if o.sampleTime < fpsUpdateInterval {
		return
	}

	fps := int(float32(o.sampleFrames)/o.sampleTime + 0.5)
	if fps < 0 {
		fps = 0
	}

	o.label = fmt.Sprintf("FPS: %d", fps)
	o.sampleTime = 0
	o.sampleFrames = 0
}

func (o *fpsOverlay) Render(screenW, screenH int) {
	if o == nil || o.renderer == nil || screenW <= 0 || screenH <= 0 {
		return
	}

	textW := measureOverlayTextWidth(o.label, fpsOverlayScale)
	textH := 7 * fpsOverlayScale

	panelW := textW + fpsOverlayPaddingX*2
	panelH := textH + fpsOverlayPaddingY*2

	x := float32(screenW) - fpsOverlayMargin - panelW
	if x < fpsOverlayMargin {
		x = fpsOverlayMargin
	}
	y := fpsOverlayMargin

	textX := x + fpsOverlayPaddingX
	textY := y + fpsOverlayPaddingY

	o.renderer.Begin(int32(screenW), int32(screenH))
	o.renderer.DrawRect(x, y, panelW, panelH, [4]float32{0.01, 0.02, 0.04, 0.62})
	o.renderer.DrawRect(x, y, panelW, 1, [4]float32{0.80, 0.86, 0.95, 0.55})
	o.renderer.DrawText(o.label, textX+1, textY+1, fpsOverlayScale, [4]float32{0, 0, 0, 0.9})
	o.renderer.DrawText(o.label, textX, textY, fpsOverlayScale, [4]float32{0.93, 0.97, 1.0, 1.0})
	o.renderer.End()
}

func measureOverlayTextWidth(text string, scale float32) float32 {
	if scale <= 0 || len(text) == 0 {
		return 0
	}
	return float32(len([]rune(text))) * 6 * scale
}
