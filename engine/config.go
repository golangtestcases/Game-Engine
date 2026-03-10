package engine

import (
	"encoding/json"
	"os"
)

// Config описывает базовые runtime-настройки движка,
// которые читаются из `config.json`.
type Config struct {
	Window   WindowConfig   `json:"window"`
	Graphics GraphicsConfig `json:"graphics"`
}

// WindowConfig содержит параметры окна и swap поведения.
type WindowConfig struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Title      string `json:"title"`
	Fullscreen bool   `json:"fullscreen"`
	VSync      bool   `json:"vsync"`
}

// GraphicsConfig содержит ключевые параметры камеры/рендера.
type GraphicsConfig struct {
	Quality   string          `json:"quality"`
	FOV       float32         `json:"fov"`
	Near      float32         `json:"near"`
	Far       float32         `json:"far"`
	Streaming StreamingConfig `json:"streaming"`
}

// StreamingConfig определяет параметры cell/chunk-стриминга мира.
type StreamingConfig struct {
	CellSize                  float32 `json:"cell_size"`
	ActiveRadius              int     `json:"active_radius"`
	VisibleRadius             int     `json:"visible_radius"`
	UnloadRadius              int     `json:"unload_radius"`
	FogStart                  float32 `json:"fog_start"`
	FogEnd                    float32 `json:"fog_end"`
	FogDensity                float32 `json:"fog_density"`
	TerrainLODNearDistance    float32 `json:"terrain_lod_near_distance"`
	TerrainLODMidDistance     float32 `json:"terrain_lod_mid_distance"`
	TerrainLODHysteresis      float32 `json:"terrain_lod_hysteresis"`
	TerrainLODMidStep         int     `json:"terrain_lod_mid_step"`
	TerrainLODFarStep         int     `json:"terrain_lod_far_step"`
	TerrainLODFarCullDistance float32 `json:"terrain_lod_far_cull_distance"`
	DecorLODNearDistance      float32 `json:"decor_lod_near_distance"`
	DecorLODMidDistance       float32 `json:"decor_lod_mid_distance"`
	DecorLODHysteresis        float32 `json:"decor_lod_hysteresis"`
	DecorLODMidDensity        float32 `json:"decor_lod_mid_density"`
	DecorLODFarDensity        float32 `json:"decor_lod_far_density"`
	DecorLODMidMinRadius      float32 `json:"decor_lod_mid_min_radius"`
	DecorLODFarMinRadius      float32 `json:"decor_lod_far_min_radius"`
	DecorLODFarCullDistance   float32 `json:"decor_lod_far_cull_distance"`
	DebugOverlay              bool    `json:"debug_overlay"`
	DebugCellBounds           bool    `json:"debug_cell_bounds"`
	DebugLODOverlay           bool    `json:"debug_lod_overlay"`
	DebugLODColors            bool    `json:"debug_lod_colors"`
}

// LoadConfig загружает JSON-конфиг из файла.
// Если файл не найден, создается новый с безопасными дефолтами.
func LoadConfig(path string) (Config, error) {
	cfg := Config{
		Window: WindowConfig{
			Width:      1280,
			Height:     720,
			Title:      "Go Game Engine",
			Fullscreen: false,
			VSync:      true,
		},
		Graphics: GraphicsConfig{
			Quality: string(GraphicsQualityHigh),
			FOV:     60,
			Near:    0.1,
			Far:     100.0,
			Streaming: StreamingConfig{
				CellSize:                  72.0,
				ActiveRadius:              3,
				VisibleRadius:             3,
				UnloadRadius:              5,
				FogStart:                  150.0,
				FogEnd:                    260.0,
				FogDensity:                1.15,
				TerrainLODNearDistance:    95.0,
				TerrainLODMidDistance:     185.0,
				TerrainLODHysteresis:      14.0,
				TerrainLODMidStep:         2,
				TerrainLODFarStep:         4,
				TerrainLODFarCullDistance: 0.0,
				DecorLODNearDistance:      80.0,
				DecorLODMidDistance:       150.0,
				DecorLODHysteresis:        12.0,
				DecorLODMidDensity:        0.55,
				DecorLODFarDensity:        0.20,
				DecorLODMidMinRadius:      1.05,
				DecorLODFarMinRadius:      1.45,
				DecorLODFarCullDistance:   210.0,
			},
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if saveErr := SaveConfig(path, cfg); saveErr != nil {
				return cfg, saveErr
			}
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// SaveConfig сохраняет конфиг в читаемом JSON-формате.
func SaveConfig(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// withDefaults дополняет пустые/некорректные поля значениями по умолчанию.
// Это защищает запуск от частично заполненного конфига.
func (c Config) withDefaults() Config {
	if c.Window.Width <= 0 {
		c.Window.Width = 1280
	}
	if c.Window.Height <= 0 {
		c.Window.Height = 720
	}
	if c.Window.Title == "" {
		c.Window.Title = "Go Game Engine"
	}

	c.Graphics.Quality = string(ParseGraphicsQualityPreset(c.Graphics.Quality))

	if c.Graphics.FOV <= 0 {
		c.Graphics.FOV = 60
	}
	if c.Graphics.Near <= 0 {
		c.Graphics.Near = 0.1
	}
	if c.Graphics.Far <= 0 {
		c.Graphics.Far = 100.0
	}

	if c.Graphics.Streaming.CellSize <= 0 {
		c.Graphics.Streaming.CellSize = 72.0
	}
	if c.Graphics.Streaming.ActiveRadius < 1 {
		c.Graphics.Streaming.ActiveRadius = 3
	}
	if c.Graphics.Streaming.VisibleRadius < 1 {
		c.Graphics.Streaming.VisibleRadius = c.Graphics.Streaming.ActiveRadius
	}
	if c.Graphics.Streaming.VisibleRadius > c.Graphics.Streaming.ActiveRadius {
		c.Graphics.Streaming.VisibleRadius = c.Graphics.Streaming.ActiveRadius
	}
	if c.Graphics.Streaming.UnloadRadius <= c.Graphics.Streaming.ActiveRadius {
		c.Graphics.Streaming.UnloadRadius = c.Graphics.Streaming.ActiveRadius + 2
	}
	if c.Graphics.Streaming.FogStart <= 0 {
		c.Graphics.Streaming.FogStart = 150.0
	}
	if c.Graphics.Streaming.FogEnd <= c.Graphics.Streaming.FogStart+1 {
		c.Graphics.Streaming.FogEnd = c.Graphics.Streaming.FogStart + 110.0
	}
	if c.Graphics.Streaming.FogDensity <= 0 {
		c.Graphics.Streaming.FogDensity = 1.15
	}
	if c.Graphics.Streaming.TerrainLODNearDistance <= 0 {
		c.Graphics.Streaming.TerrainLODNearDistance = 95.0
	}
	if c.Graphics.Streaming.TerrainLODMidDistance <= c.Graphics.Streaming.TerrainLODNearDistance+1.0 {
		c.Graphics.Streaming.TerrainLODMidDistance = c.Graphics.Streaming.TerrainLODNearDistance + 90.0
	}
	if c.Graphics.Streaming.TerrainLODHysteresis < 0 {
		c.Graphics.Streaming.TerrainLODHysteresis = 0
	}
	if c.Graphics.Streaming.TerrainLODHysteresis == 0 {
		c.Graphics.Streaming.TerrainLODHysteresis = 14.0
	}
	if c.Graphics.Streaming.TerrainLODMidStep < 1 {
		c.Graphics.Streaming.TerrainLODMidStep = 2
	}
	if c.Graphics.Streaming.TerrainLODFarStep <= c.Graphics.Streaming.TerrainLODMidStep {
		c.Graphics.Streaming.TerrainLODFarStep = c.Graphics.Streaming.TerrainLODMidStep * 2
	}
	if c.Graphics.Streaming.TerrainLODFarStep > 16 {
		c.Graphics.Streaming.TerrainLODFarStep = 16
	}
	if c.Graphics.Streaming.TerrainLODFarCullDistance > 0 &&
		c.Graphics.Streaming.TerrainLODFarCullDistance <= c.Graphics.Streaming.TerrainLODMidDistance {
		c.Graphics.Streaming.TerrainLODFarCullDistance = c.Graphics.Streaming.TerrainLODMidDistance + c.Graphics.Streaming.CellSize
	}

	if c.Graphics.Streaming.DecorLODNearDistance <= 0 {
		c.Graphics.Streaming.DecorLODNearDistance = 80.0
	}
	if c.Graphics.Streaming.DecorLODMidDistance <= c.Graphics.Streaming.DecorLODNearDistance+1.0 {
		c.Graphics.Streaming.DecorLODMidDistance = c.Graphics.Streaming.DecorLODNearDistance + 70.0
	}
	if c.Graphics.Streaming.DecorLODHysteresis < 0 {
		c.Graphics.Streaming.DecorLODHysteresis = 0
	}
	if c.Graphics.Streaming.DecorLODHysteresis == 0 {
		c.Graphics.Streaming.DecorLODHysteresis = 12.0
	}
	if c.Graphics.Streaming.DecorLODMidDensity <= 0 || c.Graphics.Streaming.DecorLODMidDensity > 1 {
		c.Graphics.Streaming.DecorLODMidDensity = 0.55
	}
	if c.Graphics.Streaming.DecorLODFarDensity < 0 || c.Graphics.Streaming.DecorLODFarDensity > 1 {
		c.Graphics.Streaming.DecorLODFarDensity = 0.20
	}
	if c.Graphics.Streaming.DecorLODFarDensity > c.Graphics.Streaming.DecorLODMidDensity {
		c.Graphics.Streaming.DecorLODFarDensity = c.Graphics.Streaming.DecorLODMidDensity
	}
	if c.Graphics.Streaming.DecorLODMidMinRadius < 0 {
		c.Graphics.Streaming.DecorLODMidMinRadius = 0
	}
	if c.Graphics.Streaming.DecorLODFarMinRadius < c.Graphics.Streaming.DecorLODMidMinRadius {
		c.Graphics.Streaming.DecorLODFarMinRadius = c.Graphics.Streaming.DecorLODMidMinRadius
	}
	if c.Graphics.Streaming.DecorLODFarCullDistance > 0 &&
		c.Graphics.Streaming.DecorLODFarCullDistance <= c.Graphics.Streaming.DecorLODMidDistance {
		c.Graphics.Streaming.DecorLODFarCullDistance = c.Graphics.Streaming.DecorLODMidDistance + c.Graphics.Streaming.CellSize*0.5
	}
	return c
}
