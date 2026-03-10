package objects

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
	"subnautica-lite/utils"
)

// CreatureBehaviorStyle задает общий стиль поведения вида.
type CreatureBehaviorStyle uint8

const (
	CreatureStyleAmbient CreatureBehaviorStyle = iota
	CreatureStylePredator
	CreatureStylePassive
)

// CreatureBehaviorState — текущее состояние агента в FSM.
type CreatureBehaviorState uint8

const (
	CreatureStatePatrol CreatureBehaviorState = iota
	CreatureStateAlert
	CreatureStateEvade
)

func (s CreatureBehaviorState) Label() string {
	switch s {
	case CreatureStateAlert:
		return "ALERT"
	case CreatureStateEvade:
		return "EVADE"
	default:
		return "PATROL"
	}
}

// CreatureMovementSettings объединяет параметры кинематики и «интеллекта» существа.
// Значения подбираются так, чтобы получить правдоподобное подводное движение.
type CreatureMovementSettings struct {
	CruiseSpeed    float32
	AlertSpeed     float32
	FleeSpeed      float32
	SpeedSharpness float32

	TurnRateDeg  float32
	PitchRateDeg float32
	MaxPitchDeg  float32
	RollResponse float32

	HabitatRadius        float32
	MinDepthBelowSurface float32
	MaxDepthBelowSurface float32
	SurfaceClearance     float32
	FloorClearance       float32

	DetectionRadius float32
	TooCloseRadius  float32

	WanderIntervalMin float32
	WanderIntervalMax float32
	ArrivalDistance   float32

	VerticalDriftAmplitude float32
	VerticalDriftFrequency float32
}

// CreatureVisual хранит параметры рендера существа.
type CreatureVisual struct {
	ModelID        string
	Color          mgl32.Vec3
	Material       engine.Material
	YawOffsetDeg   float32
	BoundingRadius float32
}

// CreatureAgent — runtime-состояние AI-агента в ECS.
type CreatureAgent struct {
	Species string
	Style   CreatureBehaviorStyle

	Movement CreatureMovementSettings
	State    CreatureBehaviorState

	HabitatCenter mgl32.Vec3
	TargetPos     mgl32.Vec3
	TargetValid   bool

	Forward        mgl32.Vec3
	PitchDeg       float32
	CurrentSpeed   float32
	WanderCooldown float32
	OrbitSign      float32
	DriftPhase     float32
}

// CreatureSpawnConfig описывает параметры спавна существа.
type CreatureSpawnConfig struct {
	Species string
	Style   CreatureBehaviorStyle

	Position      mgl32.Vec3
	YawDeg        float32
	Scale         float32
	HabitatCenter mgl32.Vec3

	Movement CreatureMovementSettings
	Visual   CreatureVisual
}

// CreatureUpdateParams передается в апдейт всех существ на кадр.
type CreatureUpdateParams struct {
	DeltaTime     float32
	Time          float32
	PlayerPos     mgl32.Vec3
	WaterLevel    float32
	GroundHeightY func(x, z float32) float32
}

// CreatureManager — тонкая обертка над ECS-функциями спавна/апдейта.
type CreatureManager struct {
	world *engine.World
}

func NewCreatureManager(world *engine.World) *CreatureManager {
	return &CreatureManager{world: world}
}

func (m *CreatureManager) Spawn(cfg CreatureSpawnConfig) engine.Entity {
	return SpawnCreature(m.world, cfg)
}

func (m *CreatureManager) Update(params CreatureUpdateParams) {
	UpdateCreatures(m.world, params)
}

func DefaultCreatureMovementSettings() CreatureMovementSettings {
	return CreatureMovementSettings{
		CruiseSpeed:    1.4,
		AlertSpeed:     1.9,
		FleeSpeed:      2.3,
		SpeedSharpness: 3.2,

		TurnRateDeg:  34,
		PitchRateDeg: 20,
		MaxPitchDeg:  18,
		RollResponse: 5.5,

		HabitatRadius:        16,
		MinDepthBelowSurface: 1.5,
		MaxDepthBelowSurface: 6.5,
		SurfaceClearance:     0.8,
		FloorClearance:       0.8,

		DetectionRadius: 8.0,
		TooCloseRadius:  2.7,

		WanderIntervalMin: 2.4,
		WanderIntervalMax: 5.2,
		ArrivalDistance:   1.3,

		VerticalDriftAmplitude: 0.22,
		VerticalDriftFrequency: 0.35,
	}
}

// DefaultPredatorMovementSettings усиливает скорости и радиусы обнаружения
// относительно базового профиля.
func DefaultPredatorMovementSettings() CreatureMovementSettings {
	settings := DefaultCreatureMovementSettings()
	settings.CruiseSpeed = 2.1
	settings.AlertSpeed = 2.7
	settings.FleeSpeed = 3.4
	settings.SpeedSharpness = 2.6
	settings.TurnRateDeg = 26
	settings.PitchRateDeg = 16
	settings.MaxPitchDeg = 14
	settings.RollResponse = 4.3
	settings.HabitatRadius = 28
	settings.MinDepthBelowSurface = 2.2
	settings.MaxDepthBelowSurface = 8.8
	settings.SurfaceClearance = 1.2
	settings.FloorClearance = 1.2
	settings.DetectionRadius = 14
	settings.TooCloseRadius = 4.2
	settings.WanderIntervalMin = 3.5
	settings.WanderIntervalMax = 7.5
	settings.ArrivalDistance = 2.0
	settings.VerticalDriftAmplitude = 0.34
	settings.VerticalDriftFrequency = 0.24
	return settings
}

// SpawnCreature создает сущность и добавляет компоненты Transform/Visual/Agent.
// Параметры перед этим нормализуются, чтобы избежать невалидных конфигураций.
func SpawnCreature(world *engine.World, cfg CreatureSpawnConfig) engine.Entity {
	if world == nil {
		panic("spawn creature: world is nil")
	}

	movement := sanitizeCreatureMovement(cfg.Movement)
	visual := sanitizeCreatureVisual(cfg.Visual)

	scale := cfg.Scale
	if scale <= 0 {
		scale = 1
	}

	habitatCenter := cfg.HabitatCenter
	if habitatCenter == (mgl32.Vec3{}) {
		habitatCenter = cfg.Position
	}

	forward := forwardFromYawPitchDeg(cfg.YawDeg, 0)
	if forward.Len() < 0.0001 {
		forward = mgl32.Vec3{0, 0, 1}
	}

	orbitSign := float32(1)
	if utils.RandomRange(0, 1) < 0.5 {
		orbitSign = -1
	}

	entity := world.CreateEntity()
	engine.AddComponent(world, entity, Transform{
		Position: cfg.Position,
		Yaw:      cfg.YawDeg,
		Scale:    scale,
	})
	engine.AddComponent(world, entity, CreatureVisual{
		ModelID:        visual.ModelID,
		Color:          visual.Color,
		Material:       visual.Material,
		YawOffsetDeg:   visual.YawOffsetDeg,
		BoundingRadius: visual.BoundingRadius,
	})
	engine.AddComponent(world, entity, CreatureAgent{
		Species: cfg.Species,
		Style:   cfg.Style,

		Movement: movement,
		State:    CreatureStatePatrol,

		HabitatCenter: habitatCenter,
		TargetPos:     cfg.Position,
		TargetValid:   false,

		Forward:      forward,
		PitchDeg:     0,
		CurrentSpeed: movement.CruiseSpeed,
		OrbitSign:    orbitSign,
		DriftPhase:   utils.RandomRange(0, float32(math.Pi*2)),
	})

	return entity
}

// UpdateCreatures обновляет движение и FSM всех существ в мире.
func UpdateCreatures(world *engine.World, params CreatureUpdateParams) {
	if world == nil || params.DeltaTime <= 0 {
		return
	}

	engine.Each2(world, func(_ engine.Entity, tr *Transform, creature *CreatureAgent) {
		updateCreature(tr, creature, params)
	})
}

// updateCreature реализует один шаг поведения:
// выбор состояния -> выбор цели -> поворот/скорость -> интеграция позиции.
func updateCreature(tr *Transform, creature *CreatureAgent, params CreatureUpdateParams) {
	settings := creature.Movement

	distanceToPlayer := params.PlayerPos.Sub(tr.Position).Len()
	nextState := creatureStateByDistance(distanceToPlayer, settings)
	if nextState != creature.State {
		creature.State = nextState
		creature.TargetValid = false
		creature.WanderCooldown = 0
	}

	creature.WanderCooldown -= params.DeltaTime
	if !creature.TargetValid || creature.WanderCooldown <= 0 || tr.Position.Sub(creature.TargetPos).Len() <= settings.ArrivalDistance {
		creature.TargetPos = creatureNextTarget(*tr, creature, params)
		creature.TargetValid = true
		creature.WanderCooldown = utils.RandomRange(settings.WanderIntervalMin, settings.WanderIntervalMax)
	}

	targetPos := creature.TargetPos
	targetPos[1] += float32(math.Sin(float64(params.Time*settings.VerticalDriftFrequency+creature.DriftPhase))) * settings.VerticalDriftAmplitude
	targetPos = clampCreatureTarget(targetPos, creature.HabitatCenter, settings, params)

	toTarget := targetPos.Sub(tr.Position)
	if toTarget.Len() < 0.0001 {
		toTarget = creature.Forward
	}
	desiredDir := toTarget.Normalize()

	desiredYaw := directionToYawDeg(desiredDir)
	desiredPitch := clampFloat(directionToPitchDeg(desiredDir), -settings.MaxPitchDeg, settings.MaxPitchDeg)

	tr.Yaw = rotateTowardsAngleDeg(tr.Yaw, desiredYaw, settings.TurnRateDeg*params.DeltaTime)
	creature.PitchDeg = rotateTowardsAngleDeg(creature.PitchDeg, desiredPitch, settings.PitchRateDeg*params.DeltaTime)
	creature.PitchDeg = clampFloat(creature.PitchDeg, -settings.MaxPitchDeg, settings.MaxPitchDeg)

	tr.Pitch = creature.PitchDeg
	turnDelta := normalizeAngleDeg(desiredYaw - tr.Yaw)
	targetRoll := clampFloat(-turnDelta*0.35, -18, 18)
	tr.Roll = expSmoothing(tr.Roll, targetRoll, settings.RollResponse, params.DeltaTime)

	creature.Forward = forwardFromYawPitchDeg(tr.Yaw, creature.PitchDeg)
	targetSpeed := creatureTargetSpeed(creature.State, settings)
	creature.CurrentSpeed = expSmoothing(creature.CurrentSpeed, targetSpeed, settings.SpeedSharpness, params.DeltaTime)

	tr.Position = tr.Position.Add(creature.Forward.Mul(creature.CurrentSpeed * params.DeltaTime))
	tr.Position = clampCreatureTarget(tr.Position, creature.HabitatCenter, settings, params)

	if horizontalDistance(tr.Position, creature.HabitatCenter) >= settings.HabitatRadius*0.98 {
		creature.TargetValid = false
	}
}

// creatureNextTarget выбирает стратегию цели в зависимости от текущего состояния FSM.
func creatureNextTarget(tr Transform, creature *CreatureAgent, params CreatureUpdateParams) mgl32.Vec3 {
	switch creature.State {
	case CreatureStateAlert:
		return creatureAlertTarget(tr, creature, params)
	case CreatureStateEvade:
		return creatureEvadeTarget(tr, creature, params)
	default:
		return creaturePatrolTarget(creature, params)
	}
}

// creaturePatrolTarget генерирует случайную цель в пределах habitat-радиуса.
func creaturePatrolTarget(creature *CreatureAgent, params CreatureUpdateParams) mgl32.Vec3 {
	angle := utils.RandomRange(0, float32(math.Pi*2))
	radius := float32(math.Sqrt(float64(utils.RandomRange(0, 1)))) * creature.Movement.HabitatRadius
	target := mgl32.Vec3{
		creature.HabitatCenter.X() + float32(math.Sin(float64(angle)))*radius,
		creature.HabitatCenter.Y(),
		creature.HabitatCenter.Z() + float32(math.Cos(float64(angle)))*radius,
	}
	target[1] = creatureDepthSample(target.X(), target.Z(), creature.HabitatCenter.Y(), creature.Movement, params)
	return clampCreatureTarget(target, creature.HabitatCenter, creature.Movement, params)
}

// creatureAlertTarget удерживает существо вблизи игрока по орбитальной траектории.
func creatureAlertTarget(tr Transform, creature *CreatureAgent, params CreatureUpdateParams) mgl32.Vec3 {
	toPlayer := params.PlayerPos.Sub(tr.Position)
	playerDir := safeNormalize(toPlayer, creature.Forward)

	if utils.RandomRange(0, 1) < 0.2 {
		creature.OrbitSign = -creature.OrbitSign
	}

	side := mgl32.Vec3{-playerDir.Z(), 0, playerDir.X()}
	side = safeNormalize(side, mgl32.Vec3{1, 0, 0}).Mul(creature.OrbitSign)

	orbitDistance := creature.Movement.TooCloseRadius * 1.7
	maxOrbit := creature.Movement.DetectionRadius * 0.72
	if orbitDistance > maxOrbit && maxOrbit > 0 {
		orbitDistance = maxOrbit
	}
	if orbitDistance < creature.Movement.TooCloseRadius*1.2 {
		orbitDistance = creature.Movement.TooCloseRadius * 1.2
	}

	target := params.PlayerPos.
		Sub(playerDir.Mul(orbitDistance)).
		Add(side.Mul(orbitDistance * 0.55))

	target[1] = creatureDepthSample(
		target.X(),
		target.Z(),
		params.PlayerPos.Y(),
		creature.Movement,
		params,
	)
	return clampCreatureTarget(target, creature.HabitatCenter, creature.Movement, params)
}

// creatureEvadeTarget уводит существо от игрока с добавлением бокового смещения.
func creatureEvadeTarget(tr Transform, creature *CreatureAgent, params CreatureUpdateParams) mgl32.Vec3 {
	away := safeNormalize(tr.Position.Sub(params.PlayerPos), creature.Forward)
	side := safeNormalize(mgl32.Vec3{-away.Z(), 0, away.X()}, mgl32.Vec3{1, 0, 0}).Mul(creature.OrbitSign)

	escapeDistance := creature.Movement.HabitatRadius*0.45 + creature.Movement.TooCloseRadius
	target := tr.Position.
		Add(away.Mul(escapeDistance)).
		Add(side.Mul(escapeDistance * 0.22))

	target[1] = creatureDepthSample(
		target.X(),
		target.Z(),
		tr.Position.Y()+utils.RandomRange(-0.6, 0.6),
		creature.Movement,
		params,
	)
	return clampCreatureTarget(target, creature.HabitatCenter, creature.Movement, params)
}

func creatureStateByDistance(distance float32, settings CreatureMovementSettings) CreatureBehaviorState {
	if distance <= settings.TooCloseRadius {
		return CreatureStateEvade
	}
	if distance <= settings.DetectionRadius {
		return CreatureStateAlert
	}
	return CreatureStatePatrol
}

func creatureTargetSpeed(state CreatureBehaviorState, settings CreatureMovementSettings) float32 {
	switch state {
	case CreatureStateAlert:
		return settings.AlertSpeed
	case CreatureStateEvade:
		return settings.FleeSpeed
	default:
		return settings.CruiseSpeed
	}
}

// clampCreatureTarget ограничивает цель по радиусу обитания и допустимому диапазону глубин.
func clampCreatureTarget(
	target mgl32.Vec3,
	habitatCenter mgl32.Vec3,
	settings CreatureMovementSettings,
	params CreatureUpdateParams,
) mgl32.Vec3 {
	dx := target.X() - habitatCenter.X()
	dz := target.Z() - habitatCenter.Z()
	dist := float32(math.Sqrt(float64(dx*dx + dz*dz)))
	if dist > settings.HabitatRadius && dist > 0.0001 {
		scale := settings.HabitatRadius / dist
		target[0] = habitatCenter.X() + dx*scale
		target[2] = habitatCenter.Z() + dz*scale
	}

	minY, maxY := creatureDepthRange(target.X(), target.Z(), settings, params)
	target[1] = clampFloat(target.Y(), minY, maxY)
	return target
}

// creatureDepthSample выбирает «живую» высоту в допустимом depth-диапазоне.
func creatureDepthSample(
	x, z float32,
	preferredY float32,
	settings CreatureMovementSettings,
	params CreatureUpdateParams,
) float32 {
	minY, maxY := creatureDepthRange(x, z, settings, params)
	if maxY-minY <= 0.001 {
		return minY
	}
	centerWeight := utils.RandomRange(0.35, 0.65)
	randomY := minY + (maxY-minY)*centerWeight
	return clampFloat(preferredY*0.55+randomY*0.45, minY, maxY)
}

// creatureDepthRange вычисляет вертикальные границы движения:
// от поверхности (с зазором) до дна (с floor clearance).
func creatureDepthRange(
	x, z float32,
	settings CreatureMovementSettings,
	params CreatureUpdateParams,
) (float32, float32) {
	maxY := params.WaterLevel - settings.MinDepthBelowSurface
	minY := params.WaterLevel - settings.MaxDepthBelowSurface

	surfaceCap := params.WaterLevel - settings.SurfaceClearance
	if maxY > surfaceCap {
		maxY = surfaceCap
	}

	if params.GroundHeightY != nil {
		floorY := params.GroundHeightY(x, z) + settings.FloorClearance
		if minY < floorY {
			minY = floorY
		}
	}

	if minY > maxY {
		mid := (minY + maxY) * 0.5
		minY = mid
		maxY = mid
	}

	return minY, maxY
}

func forwardFromYawPitchDeg(yawDeg, pitchDeg float32) mgl32.Vec3 {
	yaw := float64(mgl32.DegToRad(yawDeg))
	pitch := float64(mgl32.DegToRad(pitchDeg))

	cp := float32(math.Cos(pitch))
	forward := mgl32.Vec3{
		float32(math.Sin(yaw)) * cp,
		float32(math.Sin(pitch)),
		float32(math.Cos(yaw)) * cp,
	}
	if forward.Len() < 0.0001 {
		return mgl32.Vec3{0, 0, 1}
	}
	return forward.Normalize()
}

func directionToYawDeg(dir mgl32.Vec3) float32 {
	return float32(math.Atan2(float64(dir.X()), float64(dir.Z())) * (180.0 / math.Pi))
}

func directionToPitchDeg(dir mgl32.Vec3) float32 {
	y := clampFloat(dir.Y(), -1, 1)
	return float32(math.Asin(float64(y)) * (180.0 / math.Pi))
}

func rotateTowardsAngleDeg(current, target, maxStep float32) float32 {
	delta := normalizeAngleDeg(target - current)
	if delta > maxStep {
		delta = maxStep
	} else if delta < -maxStep {
		delta = -maxStep
	}
	return current + delta
}

func normalizeAngleDeg(deg float32) float32 {
	for deg > 180 {
		deg -= 360
	}
	for deg < -180 {
		deg += 360
	}
	return deg
}

func safeNormalize(v mgl32.Vec3, fallback mgl32.Vec3) mgl32.Vec3 {
	if v.Len() > 0.0001 {
		return v.Normalize()
	}
	if fallback.Len() > 0.0001 {
		return fallback.Normalize()
	}
	return mgl32.Vec3{0, 0, 1}
}

func horizontalDistance(a, b mgl32.Vec3) float32 {
	dx := a.X() - b.X()
	dz := a.Z() - b.Z()
	return float32(math.Sqrt(float64(dx*dx + dz*dz)))
}

func sanitizeCreatureMovement(settings CreatureMovementSettings) CreatureMovementSettings {
	defaults := DefaultCreatureMovementSettings()

	if settings.CruiseSpeed <= 0 {
		settings.CruiseSpeed = defaults.CruiseSpeed
	}
	if settings.AlertSpeed <= 0 {
		settings.AlertSpeed = defaults.AlertSpeed
	}
	if settings.FleeSpeed <= 0 {
		settings.FleeSpeed = defaults.FleeSpeed
	}
	if settings.AlertSpeed < settings.CruiseSpeed {
		settings.AlertSpeed = settings.CruiseSpeed
	}
	if settings.FleeSpeed < settings.AlertSpeed {
		settings.FleeSpeed = settings.AlertSpeed
	}
	if settings.SpeedSharpness <= 0 {
		settings.SpeedSharpness = defaults.SpeedSharpness
	}

	if settings.TurnRateDeg <= 0 {
		settings.TurnRateDeg = defaults.TurnRateDeg
	}
	if settings.PitchRateDeg <= 0 {
		settings.PitchRateDeg = defaults.PitchRateDeg
	}
	if settings.MaxPitchDeg <= 0 {
		settings.MaxPitchDeg = defaults.MaxPitchDeg
	}
	if settings.RollResponse <= 0 {
		settings.RollResponse = defaults.RollResponse
	}

	if settings.HabitatRadius < 1 {
		settings.HabitatRadius = defaults.HabitatRadius
	}
	if settings.MinDepthBelowSurface <= 0 {
		settings.MinDepthBelowSurface = defaults.MinDepthBelowSurface
	}
	if settings.MaxDepthBelowSurface <= 0 {
		settings.MaxDepthBelowSurface = defaults.MaxDepthBelowSurface
	}
	if settings.MaxDepthBelowSurface < settings.MinDepthBelowSurface {
		settings.MaxDepthBelowSurface = settings.MinDepthBelowSurface + 0.5
	}
	if settings.SurfaceClearance <= 0 {
		settings.SurfaceClearance = defaults.SurfaceClearance
	}
	if settings.FloorClearance <= 0 {
		settings.FloorClearance = defaults.FloorClearance
	}

	if settings.DetectionRadius <= 0 {
		settings.DetectionRadius = defaults.DetectionRadius
	}
	if settings.TooCloseRadius <= 0 {
		settings.TooCloseRadius = defaults.TooCloseRadius
	}
	if settings.DetectionRadius < settings.TooCloseRadius+0.5 {
		settings.DetectionRadius = settings.TooCloseRadius + 0.5
	}

	if settings.WanderIntervalMin <= 0 {
		settings.WanderIntervalMin = defaults.WanderIntervalMin
	}
	if settings.WanderIntervalMax < settings.WanderIntervalMin {
		settings.WanderIntervalMax = settings.WanderIntervalMin + 0.1
	}
	if settings.ArrivalDistance <= 0 {
		settings.ArrivalDistance = defaults.ArrivalDistance
	}

	if settings.VerticalDriftAmplitude < 0 {
		settings.VerticalDriftAmplitude = defaults.VerticalDriftAmplitude
	}
	if settings.VerticalDriftFrequency <= 0 {
		settings.VerticalDriftFrequency = defaults.VerticalDriftFrequency
	}

	return settings
}

// sanitizeCreatureVisual заполняет безопасные дефолты для визуала модели.
func sanitizeCreatureVisual(visual CreatureVisual) CreatureVisual {
	if visual.ModelID == "" {
		visual.ModelID = "default"
	}
	if visual.Color == (mgl32.Vec3{}) {
		visual.Color = mgl32.Vec3{0.85, 0.88, 0.95}
	}
	if visual.Material.Shininess <= 0 {
		visual.Material = engine.MaterialPlastic
	}
	if visual.BoundingRadius <= 0 {
		visual.BoundingRadius = 1
	}
	return visual
}

func clampFloat(v, minV, maxV float32) float32 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func expSmoothing(current, target, sharpness, dt float32) float32 {
	if dt <= 0 {
		return current
	}
	if sharpness <= 0 {
		return target
	}
	blend := 1 - float32(math.Exp(float64(-sharpness*dt)))
	return current + (target-current)*blend
}
