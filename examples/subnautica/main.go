package main

import (
	"fmt"

	"subnautica-lite/engine"
)

func main() {
	cfg, err := engine.LoadConfig("config.json")
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	cfg.Window.Title = "Subnautica Lite - Underwater"

	game := newSubnauticaGame(cfg.Graphics)
	if err := engine.Run(cfg, game); err != nil {
		panic(fmt.Errorf("engine run failed: %w", err))
	}

	// Закрываем аудио после выхода из цикла, чтобы корректно освободить ресурсы.
	if game.audioPlayer != nil {
		game.audioPlayer.Close()
	}
}
