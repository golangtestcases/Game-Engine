package engine

import "strings"

// GraphicsQualityPreset — унифицированные уровни качества,
// используемые и в конфиге, и в runtime-переключении.
type GraphicsQualityPreset string

const (
	GraphicsQualityLow    GraphicsQualityPreset = "low"
	GraphicsQualityMedium GraphicsQualityPreset = "medium"
	GraphicsQualityHigh   GraphicsQualityPreset = "high"
)

// ParseGraphicsQualityPreset нормализует пользовательский ввод.
// Любое неизвестное значение трактуется как `high`.
func ParseGraphicsQualityPreset(raw string) GraphicsQualityPreset {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(GraphicsQualityLow):
		return GraphicsQualityLow
	case string(GraphicsQualityMedium):
		return GraphicsQualityMedium
	default:
		return GraphicsQualityHigh
	}
}

// Label возвращает человекочитаемый текст для UI/логов.
func (p GraphicsQualityPreset) Label() string {
	switch p {
	case GraphicsQualityLow:
		return "Low"
	case GraphicsQualityMedium:
		return "Medium"
	default:
		return "High"
	}
}
