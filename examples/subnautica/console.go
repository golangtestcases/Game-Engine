package main

import (
	"fmt"
	"strings"

	"github.com/go-gl/glfw/v3.3/glfw"

	"subnautica-lite/engine"
)

const (
	consoleMaxHistoryLines = 14
	consoleMaxInputRunes   = 96
)

type devConsole struct {
	window         *glfw.Window
	renderer       *consoleOverlayRenderer
	commandHandler func(string) string

	open      bool
	input     []rune
	history   []string
	maxInput  int
	maxLog    int
	prevKey   glfw.KeyCallback
	prevChar  glfw.CharCallback
	skipKChar bool
}

// newDevConsole перехватывает key/char callbacks окна и встраивает dev-консоль.
// Предыдущие callbacks сохраняются и вызываются, когда консоль закрыта.
func newDevConsole(window *glfw.Window, commandHandler func(string) string) *devConsole {
	console := &devConsole{
		window:         window,
		renderer:       newConsoleOverlayRenderer(),
		commandHandler: commandHandler,
		maxInput:       consoleMaxInputRunes,
		maxLog:         consoleMaxHistoryLines,
	}

	console.prevKey = window.SetKeyCallback(console.onKey)
	console.prevChar = window.SetCharCallback(console.onChar)
	return console
}

func (c *devConsole) IsOpen() bool {
	return c != nil && c.open
}

func (c *devConsole) Log(line string) {
	if c == nil {
		return
	}
	c.appendHistory(line)
}

func (c *devConsole) Render(screenW, screenH int, timeSec float32) {
	if c == nil || !c.open || c.renderer == nil {
		return
	}

	margin := float32(16)
	panelWidth := float32(screenW) - margin*2
	panelHeight := clampFloat(float32(screenH)*0.34, 150, 260)
	inputHeight := float32(36)
	scale := float32(2)
	lineHeight := 8 * scale

	textX := margin + 12
	headerY := margin + 12
	historyY := headerY + lineHeight + 4
	inputY := margin + panelHeight - inputHeight + 10
	maxChars := int((panelWidth - 24) / (6 * scale))

	c.renderer.Begin(int32(screenW), int32(screenH))
	c.renderer.DrawRect(margin, margin, panelWidth, panelHeight, [4]float32{0.02, 0.03, 0.04, 0.78})
	c.renderer.DrawRect(margin, margin, panelWidth, 1, [4]float32{0.62, 0.66, 0.72, 0.75})
	c.renderer.DrawRect(margin, margin+panelHeight-inputHeight, panelWidth, inputHeight, [4]float32{0.06, 0.07, 0.10, 0.94})
	c.renderer.DrawText(truncateRunes("DEV CONSOLE (K TO CLOSE)", maxChars), textX, headerY, scale, [4]float32{0.86, 0.92, 1.0, 1.0})

	historySpace := panelHeight - inputHeight - (historyY - margin) - 8
	maxVisibleLines := int(historySpace / lineHeight)
	if maxVisibleLines < 1 {
		maxVisibleLines = 1
	}
	start := len(c.history) - maxVisibleLines
	if start < 0 {
		start = 0
	}
	y := historyY
	for _, line := range c.history[start:] {
		c.renderer.DrawText(truncateRunes(line, maxChars), textX, y, scale, [4]float32{0.93, 0.96, 1.0, 0.98})
		y += lineHeight
	}

	inputText := "> " + string(c.input)
	if int(timeSec*2)%2 == 0 {
		inputText += "_"
	}
	c.renderer.DrawText(truncateRunes(inputText, maxChars), textX, inputY, scale, [4]float32{1.0, 1.0, 1.0, 1.0})
	c.renderer.End()
}

// onKey обрабатывает toggle консоли, backspace и execute input.
// Когда консоль открыта, обычный игровой input не должен проходить дальше.
func (c *devConsole) onKey(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if key == glfw.KeyK && action == glfw.Press {
		c.open = !c.open
		c.skipKChar = c.open
		return
	}

	if c.open {
		switch key {
		case glfw.KeyBackspace:
			if action == glfw.Press || action == glfw.Repeat {
				c.backspace()
			}
		case glfw.KeyEnter, glfw.KeyKPEnter:
			if action == glfw.Press {
				c.executeInput()
			}
		}
		return
	}

	if c.prevKey != nil {
		c.prevKey(window, key, scancode, action, mods)
	}
}

func (c *devConsole) onChar(window *glfw.Window, ch rune) {
	if !c.open {
		if c.prevChar != nil {
			c.prevChar(window, ch)
		}
		return
	}

	if c.skipKChar && (ch == 'k' || ch == 'K') {
		c.skipKChar = false
		return
	}
	c.skipKChar = false

	if !isSupportedConsoleRune(ch) || len(c.input) >= c.maxInput {
		return
	}
	c.input = append(c.input, ch)
}

func (c *devConsole) backspace() {
	if len(c.input) == 0 {
		return
	}
	c.input = c.input[:len(c.input)-1]
}

func (c *devConsole) executeInput() {
	raw := strings.TrimSpace(string(c.input))
	c.input = c.input[:0]

	if raw == "" {
		return
	}

	c.appendHistory("> " + raw)
	if c.commandHandler == nil {
		return
	}

	response := strings.TrimSpace(c.commandHandler(raw))
	if response == "" {
		return
	}

	for _, line := range strings.Split(response, "\n") {
		c.appendHistory(strings.TrimSpace(line))
	}
}

func (c *devConsole) appendHistory(line string) {
	if line == "" {
		return
	}
	c.history = append(c.history, line)
	if len(c.history) > c.maxLog {
		c.history = c.history[len(c.history)-c.maxLog:]
	}
}

func isSupportedConsoleRune(ch rune) bool {
	return ch >= 32 && ch <= 126
}

func truncateRunes(text string, max int) string {
	if max <= 0 {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func (g *subnauticaGame) syncConsoleCapture(ctx *engine.Context) {
	consoleOpen := g.console != nil && g.console.IsOpen()
	if consoleOpen == g.consoleInputActive {
		return
	}

	g.consoleInputActive = consoleOpen
	if consoleOpen {
		ctx.Window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
		ctx.Camera.SetMouseLookEnabled(false)
		return
	}

	ctx.Window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	ctx.Camera.SetMouseLookEnabled(true)
}

// executeConsoleCommand — простой роутер команд от dev-консоли.
func (g *subnauticaGame) executeConsoleCommand(raw string) string {
	parts := strings.Fields(strings.ToLower(raw))
	if len(parts) == 0 {
		return ""
	}

	switch parts[0] {
	case "noclip":
		player := g.localPlayerState()
		if player == nil {
			return "noclip: player not ready"
		}
		target := !player.Noclip
		g.queueCommand(simulationCommand{
			Kind: simulationCommandToggleNoclip,
			ToggleNoclip: toggleNoclipCommand{
				PlayerID: player.PlayerID,
				Enabled:  target,
			},
		})
		if target {
			return "noclip: ON"
		}
		return "noclip: OFF"
	case "quality":
		if g.engineCtx == nil {
			return "quality: context unavailable"
		}
		if len(parts) < 2 {
			return "quality: " + g.qualityPreset.Label() + " (use: quality low|medium|high)"
		}
		preset, ok := parseQualityToken(parts[1])
		if !ok {
			return fmt.Sprintf("unknown quality: %s", parts[1])
		}
		return g.applyGraphicsPreset(g.engineCtx, preset, false)
	default:
		return fmt.Sprintf("unknown command: %s", parts[0])
	}
}
