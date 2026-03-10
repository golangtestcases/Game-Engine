package engine

// OceanQualitySettings управляет балансом качества океана и стоимости рендера.
type OceanQualitySettings struct {
	SceneResolutionScale      float32
	ReflectionResolutionScale float32
	ReflectionPassEnabled     bool
	LightShaftsEnabled        bool
	SSRMaxSteps               int32
	LODCount                  int
	LODRadiusScale            float32
	CullRadiusScale           float32
}

func DefaultOceanQualitySettings() OceanQualitySettings {
	return OceanQualitySettings{
		SceneResolutionScale:      1.0,
		ReflectionResolutionScale: 1.35,
		ReflectionPassEnabled:     true,
		LightShaftsEnabled:        true,
		SSRMaxSteps:               32,
		LODCount:                  5,
		LODRadiusScale:            1.0,
		CullRadiusScale:           1.0,
	}
}

// clampOceanQualitySettings нормализует значения, чтобы избежать
// слишком дорогих или некорректных конфигураций.
func clampOceanQualitySettings(settings OceanQualitySettings, maxLODCount int) OceanQualitySettings {
	if settings.SceneResolutionScale < 0.5 {
		settings.SceneResolutionScale = 0.5
	}
	if settings.SceneResolutionScale > 1.0 {
		settings.SceneResolutionScale = 1.0
	}
	if settings.ReflectionResolutionScale < 0.35 {
		settings.ReflectionResolutionScale = 0.35
	}
	if settings.ReflectionResolutionScale > 1.5 {
		settings.ReflectionResolutionScale = 1.5
	}
	if settings.SSRMaxSteps < 0 {
		settings.SSRMaxSteps = 0
	}
	if settings.SSRMaxSteps > 32 {
		settings.SSRMaxSteps = 32
	}
	if maxLODCount < 1 {
		maxLODCount = 1
	}
	if settings.LODCount < 1 {
		settings.LODCount = 1
	}
	if settings.LODCount > maxLODCount {
		settings.LODCount = maxLODCount
	}
	if settings.LODRadiusScale < 0.35 {
		settings.LODRadiusScale = 0.35
	}
	if settings.LODRadiusScale > 1.35 {
		settings.LODRadiusScale = 1.35
	}
	if settings.CullRadiusScale < 1.0 {
		settings.CullRadiusScale = 1.0
	}
	if settings.CullRadiusScale > 3.0 {
		settings.CullRadiusScale = 3.0
	}
	return settings
}
