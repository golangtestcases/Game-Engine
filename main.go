package main

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"

	"subnautica-lite/engine"
)

// starterGame — минимальный шаблон игры на движке.
// Его удобно использовать как стартовую точку для новых прототипов.
type starterGame struct{}

// Init вызывается один раз перед входом в главный цикл.
// Здесь можно выставить стартовые параметры камеры и загрузить ресурсы.
func (g *starterGame) Init(ctx *engine.Context) error {
	ctx.Camera.Speed = 3.0
	return nil
}

// Update вызывается каждый кадр до рендера.
// В шаблоне оставлено только базовое управление камерой.
func (g *starterGame) Update(ctx *engine.Context) error {
	ctx.Camera.ProcessInput(ctx.Window, ctx.DeltaTime)
	return nil
}

// Render вызывается каждый кадр после Update.
// Шаблон только задает цвет очистки экрана.
func (g *starterGame) Render(_ *engine.Context) error {
	gl.ClearColor(0.07, 0.09, 0.14, 1.0)
	return nil
}

func main() {
	// Загружаем конфигурацию из локального JSON.
	// Если файла нет, движок создаст его со значениями по умолчанию.
	cfg, err := engine.LoadConfig("config.json")
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	// Переопределяем заголовок окна, чтобы было видно, что запущен шаблон.
	cfg.Window.Title = "Go Engine - Starter"

	if err := engine.Run(cfg, &starterGame{}); err != nil {
		panic(fmt.Errorf("engine run failed: %w", err))
	}
}
