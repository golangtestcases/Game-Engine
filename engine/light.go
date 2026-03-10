package engine

import "github.com/go-gl/mathgl/mgl32"

// LightType определяет модель источника света в шейдере.
type LightType int

const (
	LightDirectional LightType = iota
	LightPoint
	LightSpot
)

// MaxLights ограничивает количество источников, передаваемых в GLSL.
const MaxLights = 8

const (
	defaultPointConstant  = 1.0
	defaultPointLinear    = 0.09
	defaultPointQuadratic = 0.032
)

// AmbientLight задает фоновое (ненаправленное) освещение сцены.
type AmbientLight struct {
	Color     mgl32.Vec3
	Intensity float32
}

// Light описывает параметры одного источника света.
type Light struct {
	Type      LightType
	Position  mgl32.Vec3
	Direction mgl32.Vec3
	Color     mgl32.Vec3
	Intensity float32

	// Параметры затухания для point/spot источников.
	Constant  float32
	Linear    float32
	Quadratic float32

	// Угол внутреннего и внешнего конуса для spot-light.
	CutOff      float32
	OuterCutOff float32

	Enabled bool
}

// LightingState — единая структура для передачи света из Go в shader layer.
type LightingState struct {
	Ambient          AmbientLight
	Lights           []Light
	ShadowLightIndex int
}

// DefaultAmbientLight возвращает безопасный дефолт, чтобы сцена не была черной без явной конфигурации.
func DefaultAmbientLight() AmbientLight {
	return AmbientLight{
		Color:     mgl32.Vec3{1.0, 1.0, 1.0},
		Intensity: 0.30,
	}
}

// NewAmbientLight создает ambient-источник с базовой валидацией.
func NewAmbientLight(color mgl32.Vec3, intensity float32) AmbientLight {
	if color.Len() < 0.0001 {
		color = mgl32.Vec3{1.0, 1.0, 1.0}
	}
	if intensity < 0 {
		intensity = 0
	}
	return AmbientLight{
		Color:     color,
		Intensity: intensity,
	}
}

// DefaultSunLight предоставляет fallback направленного света.
func DefaultSunLight() Light {
	return NewDirectionalLight(
		mgl32.Vec3{0.28, -1.0, 0.22},
		mgl32.Vec3{1.0, 0.97, 0.92},
		1.0,
	)
}

// DefaultLightingState возвращает fallback-состояние для рендера без scene-конфига.
func DefaultLightingState() LightingState {
	sun := DefaultSunLight()
	return LightingState{
		Ambient:          DefaultAmbientLight(),
		Lights:           []Light{sun},
		ShadowLightIndex: 0,
	}
}

// NewDirectionalLight создает направленный свет (например, солнце).
func NewDirectionalLight(direction, color mgl32.Vec3, intensity float32) Light {
	if direction.Len() < 0.0001 {
		direction = mgl32.Vec3{0.28, -1.0, 0.22}
	}
	if color.Len() < 0.0001 {
		color = mgl32.Vec3{1.0, 1.0, 1.0}
	}
	if intensity < 0 {
		intensity = 0
	}
	return Light{
		Type:      LightDirectional,
		Direction: direction.Normalize(),
		Color:     color,
		Intensity: intensity,
		Enabled:   true,
	}
}

// NewPointLight создает точечный источник с дефолтным затуханием.
func NewPointLight(position, color mgl32.Vec3, intensity float32) Light {
	if color.Len() < 0.0001 {
		color = mgl32.Vec3{1.0, 1.0, 1.0}
	}
	if intensity < 0 {
		intensity = 0
	}
	return Light{
		Type:      LightPoint,
		Position:  position,
		Color:     color,
		Intensity: intensity,
		Constant:  defaultPointConstant,
		Linear:    defaultPointLinear,
		Quadratic: defaultPointQuadratic,
		Enabled:   true,
	}
}

// NewPointLightWithRange создает point-light с затуханием, подобранным под рабочий радиус.
func NewPointLightWithRange(position, color mgl32.Vec3, intensity, rangeMeters float32) Light {
	light := NewPointLight(position, color, intensity)
	light.Constant, light.Linear, light.Quadratic = PointLightAttenuation(rangeMeters)
	return light
}

// NewSpotLight создает прожектор.
func NewSpotLight(position, direction, color mgl32.Vec3, intensity, cutOff, outerCutOff float32) Light {
	if direction.Len() < 0.0001 {
		direction = mgl32.Vec3{0, -1, 0}
	}
	if color.Len() < 0.0001 {
		color = mgl32.Vec3{1.0, 1.0, 1.0}
	}
	if intensity < 0 {
		intensity = 0
	}
	return Light{
		Type:        LightSpot,
		Position:    position,
		Direction:   direction.Normalize(),
		Color:       color,
		Intensity:   intensity,
		Constant:    defaultPointConstant,
		Linear:      defaultPointLinear,
		Quadratic:   defaultPointQuadratic,
		CutOff:      cutOff,
		OuterCutOff: outerCutOff,
		Enabled:     true,
	}
}

// NewSpotLightWithRange создает прожектор с параметрами затухания от целевого радиуса.
func NewSpotLightWithRange(position, direction, color mgl32.Vec3, intensity, cutOff, outerCutOff, rangeMeters float32) Light {
	light := NewSpotLight(position, direction, color, intensity, cutOff, outerCutOff)
	light.Constant, light.Linear, light.Quadratic = PointLightAttenuation(rangeMeters)
	return light
}

// PointLightAttenuation возвращает коэффициенты c/l/q для формулы 1 / (c + l*d + q*d^2).
// Русский комментарий: формула нормализована под «рабочий радиус», чтобы сценам было проще
// оперировать в метрах, а не подбирать линейный/квадратичный коэффициенты вручную.
func PointLightAttenuation(rangeMeters float32) (float32, float32, float32) {
	if rangeMeters <= 0 {
		return defaultPointConstant, defaultPointLinear, defaultPointQuadratic
	}
	linear := 1.35 / rangeMeters
	quadratic := 8.0 / (rangeMeters * rangeMeters)
	return 1.0, linear, quadratic
}

// LightManager хранит компактный массив источников для быстрой отправки в шейдер.
type LightManager struct {
	lights   [MaxLights]Light
	count    int
	ambient  AmbientLight
	sunIndex int

	activeLights     [MaxLights]Light
	activeCount      int
	shadowLightIndex int
}

func NewLightManager() *LightManager {
	return &LightManager{
		ambient:          DefaultAmbientLight(),
		sunIndex:         -1,
		shadowLightIndex: -1,
	}
}

// SetAmbientLight задает engine-уровневый ambient.
func (lm *LightManager) SetAmbientLight(ambient AmbientLight) {
	if lm == nil {
		return
	}
	lm.ambient = NewAmbientLight(ambient.Color, ambient.Intensity)
}

// SetAmbient — короткий helper без промежуточной структуры.
func (lm *LightManager) SetAmbient(color mgl32.Vec3, intensity float32) {
	lm.SetAmbientLight(NewAmbientLight(color, intensity))
}

func (lm *LightManager) AmbientLight() AmbientLight {
	if lm == nil {
		return DefaultAmbientLight()
	}
	return lm.ambient
}

// LightingState возвращает текущие ambient + активные lights для renderer.SetLighting(...).
func (lm *LightManager) LightingState() LightingState {
	if lm == nil {
		return DefaultLightingState()
	}

	lm.rebuildActiveLights()
	return LightingState{
		Ambient:          lm.ambient,
		Lights:           lm.activeLights[:lm.activeCount],
		ShadowLightIndex: lm.shadowLightIndex,
	}
}

// SetSun обновляет/добавляет directional light, который считается солнцем.
// Русский комментарий: храним индекс солнца отдельно, чтобы shadow/sky/ocean стабильно
// ссылались на один и тот же источник даже при добавлении point/spot источников.
func (lm *LightManager) SetSun(direction, color mgl32.Vec3, intensity float32) {
	if lm == nil {
		return
	}
	sun := NewDirectionalLight(direction, color, intensity)
	if lm.sunIndex >= 0 && lm.sunIndex < lm.count {
		lm.lights[lm.sunIndex] = sun
		return
	}
	idx := lm.AddLight(sun)
	if idx >= 0 {
		lm.sunIndex = idx
	}
}

// Sun возвращает directional light, назначенный как солнце.
func (lm *LightManager) Sun() *Light {
	if lm == nil {
		return nil
	}
	if lm.sunIndex >= 0 && lm.sunIndex < lm.count && lm.lights[lm.sunIndex].Type == LightDirectional {
		return &lm.lights[lm.sunIndex]
	}
	for i := 0; i < lm.count; i++ {
		if lm.lights[i].Type == LightDirectional {
			lm.sunIndex = i
			return &lm.lights[i]
		}
	}
	return nil
}

func (lm *LightManager) AddDirectionalLight(direction, color mgl32.Vec3, intensity float32) int {
	return lm.AddLight(NewDirectionalLight(direction, color, intensity))
}

func (lm *LightManager) AddPointLight(position, color mgl32.Vec3, intensity float32) int {
	return lm.AddLight(NewPointLight(position, color, intensity))
}

func (lm *LightManager) AddPointLightWithRange(position, color mgl32.Vec3, intensity, rangeMeters float32) int {
	return lm.AddLight(NewPointLightWithRange(position, color, intensity, rangeMeters))
}

func (lm *LightManager) AddSpotLight(position, direction, color mgl32.Vec3, intensity, cutOff, outerCutOff float32) int {
	return lm.AddLight(NewSpotLight(position, direction, color, intensity, cutOff, outerCutOff))
}

func (lm *LightManager) AddSpotLightWithRange(position, direction, color mgl32.Vec3, intensity, cutOff, outerCutOff, rangeMeters float32) int {
	return lm.AddLight(NewSpotLightWithRange(position, direction, color, intensity, cutOff, outerCutOff, rangeMeters))
}

// AddLight добавляет источник и возвращает его индекс, либо -1 если достигнут лимит.
func (lm *LightManager) AddLight(light Light) int {
	if lm == nil {
		return -1
	}
	if lm.count >= MaxLights {
		return -1
	}

	light = sanitizeLight(light)
	lm.lights[lm.count] = light
	if light.Type == LightDirectional && lm.sunIndex < 0 {
		lm.sunIndex = lm.count
	}
	lm.count++
	return lm.count - 1
}

// SetLight заменяет параметры источника по индексу.
func (lm *LightManager) SetLight(index int, light Light) bool {
	if lm == nil || index < 0 || index >= lm.count {
		return false
	}

	light = sanitizeLight(light)
	lm.lights[index] = light
	if light.Type == LightDirectional && lm.sunIndex < 0 {
		lm.sunIndex = index
	}
	if lm.sunIndex == index && light.Type != LightDirectional {
		lm.sunIndex = -1
	}
	return true
}

// SetLightEnabled включает/выключает источник без его удаления из менеджера.
func (lm *LightManager) SetLightEnabled(index int, enabled bool) bool {
	if lm == nil || index < 0 || index >= lm.count {
		return false
	}
	lm.lights[index].Enabled = enabled
	return true
}

// RemoveLight удаляет источник по индексу.
func (lm *LightManager) RemoveLight(index int) bool {
	if lm == nil || index < 0 || index >= lm.count {
		return false
	}

	last := lm.count - 1
	removedWasSun := index == lm.sunIndex

	if index != last {
		lm.lights[index] = lm.lights[last]
		if lm.sunIndex == last {
			lm.sunIndex = index
		}
	}

	lm.lights[last] = Light{}
	lm.count--

	if removedWasSun {
		lm.sunIndex = -1
		for i := 0; i < lm.count; i++ {
			if lm.lights[i].Type == LightDirectional {
				lm.sunIndex = i
				break
			}
		}
	}
	return true
}

func (lm *LightManager) GetLight(index int) *Light {
	if lm == nil || index < 0 || index >= lm.count {
		return nil
	}
	return &lm.lights[index]
}

func (lm *LightManager) GetLights() []Light {
	if lm == nil {
		return nil
	}
	return lm.lights[:lm.count]
}

func (lm *LightManager) Clear() {
	if lm == nil {
		return
	}
	for i := 0; i < lm.count; i++ {
		lm.lights[i] = Light{}
	}
	lm.count = 0
	lm.sunIndex = -1
	lm.activeCount = 0
	lm.shadowLightIndex = -1
}

func (lm *LightManager) rebuildActiveLights() {
	lm.activeCount = 0
	lm.shadowLightIndex = -1

	for sourceIndex := 0; sourceIndex < lm.count && lm.activeCount < MaxLights; sourceIndex++ {
		light := sanitizeLight(lm.lights[sourceIndex])
		if !isLightActive(light) {
			continue
		}

		packedIndex := lm.activeCount
		lm.activeLights[packedIndex] = light
		lm.activeCount++

		if sourceIndex == lm.sunIndex && light.Type == LightDirectional {
			lm.shadowLightIndex = packedIndex
		}
	}

	if lm.shadowLightIndex >= 0 {
		return
	}

	// Если в менеджере явно назначено солнце, но оно временно выключено/ослаблено,
	// не подменяем его автоматически другим directional-источником.
	hasDesignatedSun := lm.sunIndex >= 0 && lm.sunIndex < lm.count && lm.lights[lm.sunIndex].Type == LightDirectional
	if hasDesignatedSun {
		return
	}

	for i := 0; i < lm.activeCount; i++ {
		if lm.activeLights[i].Type == LightDirectional {
			lm.shadowLightIndex = i
			return
		}
	}
}

func isLightActive(light Light) bool {
	return light.Enabled && light.Intensity > 0.0001
}

func sanitizeLight(light Light) Light {
	if light.Type != LightDirectional && light.Type != LightPoint && light.Type != LightSpot {
		light.Type = LightPoint
	}

	if light.Color.Len() < 0.0001 {
		light.Color = mgl32.Vec3{1.0, 1.0, 1.0}
	}
	if light.Intensity < 0 {
		light.Intensity = 0
	}

	if light.Type == LightDirectional || light.Type == LightSpot {
		if light.Direction.Len() < 0.0001 {
			if light.Type == LightSpot {
				light.Direction = mgl32.Vec3{0, -1, 0}
			} else {
				light.Direction = mgl32.Vec3{0.28, -1.0, 0.22}
			}
		}
		light.Direction = light.Direction.Normalize()
	}

	if light.Type == LightPoint || light.Type == LightSpot {
		if light.Constant <= 0 && light.Linear <= 0 && light.Quadratic <= 0 {
			light.Constant = defaultPointConstant
			light.Linear = defaultPointLinear
			light.Quadratic = defaultPointQuadratic
		} else {
			if light.Constant <= 0 {
				light.Constant = defaultPointConstant
			}
			if light.Linear < 0 {
				light.Linear = 0
			}
			if light.Quadratic < 0 {
				light.Quadratic = 0
			}
		}
	}

	if light.Type == LightSpot {
		if light.OuterCutOff > light.CutOff {
			light.OuterCutOff = light.CutOff
		}
	}

	return light
}
