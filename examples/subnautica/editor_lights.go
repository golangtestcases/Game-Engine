package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-gl/mathgl/mgl32"
)

type localLightPreset uint8

const (
	localLightPresetSmall localLightPreset = iota
	localLightPresetMedium
	localLightPresetLarge
)

type localLightPresetConfig struct {
	Range       float32
	Intensity   float32
	Color       mgl32.Vec3
	MarkerScale float32
	MarkerColor mgl32.Vec3
}

var localLightPresets = map[localLightPreset]localLightPresetConfig{
	localLightPresetSmall: {
		Range:       180.0,
		Intensity:   1.70,
		Color:       mgl32.Vec3{0.16, 0.78, 0.96},
		MarkerScale: 0.32,
		MarkerColor: mgl32.Vec3{0.35, 0.92, 0.98},
	},
	localLightPresetMedium: {
		Range:       340.0,
		Intensity:   2.70,
		Color:       mgl32.Vec3{0.20, 0.88, 0.74},
		MarkerScale: 0.42,
		MarkerColor: mgl32.Vec3{0.42, 0.95, 0.78},
	},
	localLightPresetLarge: {
		Range:       520.0,
		Intensity:   3.90,
		Color:       mgl32.Vec3{0.28, 0.86, 0.62},
		MarkerScale: 0.56,
		MarkerColor: mgl32.Vec3{0.54, 0.96, 0.70},
	},
}

func (p localLightPreset) Label() string {
	switch p {
	case localLightPresetSmall:
		return "SMALL"
	case localLightPresetLarge:
		return "LARGE"
	default:
		return "MEDIUM"
	}
}

func (p localLightPreset) storageLabel() string {
	switch p {
	case localLightPresetSmall:
		return "small"
	case localLightPresetLarge:
		return "large"
	default:
		return "medium"
	}
}

func parseLocalLightPreset(value string) (localLightPreset, bool) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "small":
		return localLightPresetSmall, true
	case "large":
		return localLightPresetLarge, true
	case "medium":
		return localLightPresetMedium, true
	default:
		return localLightPresetMedium, false
	}
}

func localLightPresetConfigFor(preset localLightPreset) localLightPresetConfig {
	cfg, ok := localLightPresets[preset]
	if !ok {
		return localLightPresets[localLightPresetMedium]
	}
	return cfg
}

func inferLocalLightPresetByRange(lightRange float32) localLightPreset {
	bestPreset := localLightPresetMedium
	bestDistance := float32(1e9)
	for preset, cfg := range localLightPresets {
		distance := absf(cfg.Range - lightRange)
		if distance < bestDistance {
			bestDistance = distance
			bestPreset = preset
		}
	}
	return bestPreset
}

func scenePointLightFromPreset(position mgl32.Vec3, preset localLightPreset) scenePointLight {
	cfg := localLightPresetConfigFor(preset)
	return scenePointLight{
		Position:  position,
		Color:     cfg.Color,
		Intensity: cfg.Intensity,
		Range:     cfg.Range,
		Enabled:   true,
		Preset:    preset,
	}
}

func sanitizeScenePointLight(light scenePointLight) scenePointLight {
	if light.Range < 0.5 {
		light.Range = 0.5
	}
	if light.Intensity < 0 {
		light.Intensity = 0
	}

	cfg, hasPreset := localLightPresets[light.Preset]
	if !hasPreset {
		light.Preset = inferLocalLightPresetByRange(light.Range)
		cfg = localLightPresetConfigFor(light.Preset)
	}
	if light.Color.Len() < 0.0001 {
		light.Color = cfg.Color
	}

	return light
}

func editorMarkerScaleForLight(light scenePointLight) float32 {
	scale := localLightPresetConfigFor(light.Preset).MarkerScale
	if scale < 0.18 {
		scale = 0.18
	}
	return scale
}

func editorMarkerColorForLight(light scenePointLight) mgl32.Vec3 {
	return localLightPresetConfigFor(light.Preset).MarkerColor
}

type scenePointLightsFile struct {
	Version     int                   `json:"version"`
	PointLights []scenePointLightFile `json:"point_lights"`
}

type scenePointLightFile struct {
	Position  simVec3 `json:"position"`
	Color     simVec3 `json:"color"`
	Intensity float32 `json:"intensity"`
	Range     float32 `json:"range"`
	Enabled   bool    `json:"enabled"`
	Preset    string  `json:"preset,omitempty"`
}

const scenePointLightsFileVersion = 1

func loadScenePointLights(path string) ([]scenePointLight, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var payload scenePointLightsFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Version != scenePointLightsFileVersion {
		return nil, fmt.Errorf("scene lights: unsupported file version %d", payload.Version)
	}

	lights := make([]scenePointLight, 0, len(payload.PointLights))
	for i := range payload.PointLights {
		serialized := payload.PointLights[i]
		preset, ok := parseLocalLightPreset(serialized.Preset)
		if !ok {
			preset = inferLocalLightPresetByRange(serialized.Range)
		}
		light := sanitizeScenePointLight(scenePointLight{
			Position:  serialized.Position.Mgl(),
			Color:     serialized.Color.Mgl(),
			Intensity: serialized.Intensity,
			Range:     serialized.Range,
			Enabled:   serialized.Enabled,
			Preset:    preset,
		})
		lights = append(lights, light)
	}

	return lights, nil
}

func saveScenePointLights(path string, lights []scenePointLight) error {
	if len(path) == 0 {
		return errors.New("scene lights: empty save path")
	}

	payload := scenePointLightsFile{
		Version:     scenePointLightsFileVersion,
		PointLights: make([]scenePointLightFile, 0, len(lights)),
	}
	for i := range lights {
		light := sanitizeScenePointLight(lights[i])
		payload.PointLights = append(payload.PointLights, scenePointLightFile{
			Position:  simVec3FromMgl(light.Position),
			Color:     simVec3FromMgl(light.Color),
			Intensity: light.Intensity,
			Range:     light.Range,
			Enabled:   light.Enabled,
			Preset:    light.Preset.storageLabel(),
		})
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
