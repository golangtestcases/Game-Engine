package main

const (
	numPlants       = 180
	playerEyeHeight = 1.15
	waterSurfaceY   = 4.8
	baseMoveSpeed   = 3.0
	sprintFactor    = 1.9
	noclipMoveSpeed = 8.5
	noclipSprintMul = 2.1
	jumpImpulse     = 2.8
	underwaterG     = 3.2
	waterDrag       = 2.4

	waterLevelWaveAmplitude = 0.01
	waterLevelWaveSpeed     = 0.5

	swimVerticalAccel          = 7.6
	swimVerticalMaxRiseSpeed   = 3.4
	swimVerticalMaxDiveSpeed   = 4.1
	swimVerticalInputDeadzone  = 0.01
	surfaceUpInputResistance   = 0.82
	surfaceOvershootPullFactor = 2.4

	buoyancyStrength      = 8.6
	surfaceBuoyancyDamp   = 5.0
	surfaceBandThickness  = 0.8
	surfaceFloatDepth     = 0.06
	maxRiseAboveSurface   = 1.1
	surfaceInfluenceExtra = 0.35

	surfaceBobAmplitude          = 0.07
	surfaceBobSpeed              = 0.78
	surfaceBobSecondaryAmplitude = 0.028
	surfaceBobSecondarySpeed     = 1.37

	underwaterIdleBobAmplitude          = 0.08
	underwaterIdleBobSpeed              = 0.56
	underwaterIdleBobSecondaryAmplitude = 0.032
	underwaterIdleBobSecondarySpeed     = 1.06
	underwaterIdleBlendSpeed            = 4.6
	underwaterIdleVelocityThreshold     = 0.32
	underwaterNeutralBuoyancy           = 3.05
	underwaterIdleVelocityDamping       = 2.6

	proceduralBobSmoothing = 8.2
	titleUpdateInterval    = 0.5

	terrainSavePath     = "terrain_map.json"
	sceneLightsSavePath = "scene_lights.json"

	editorCameraSpeed                = 9.0
	editorCameraSprintMul            = 2.0
	editorCameraOverviewBackDistance = 14.0
	editorCameraOverviewLift         = 9.0
	editorCameraOverviewPitchDeg     = -34.0
	editorCameraGroundClearance      = 2.4

	editorBrushRadiusDefault     = 6.0
	editorBrushRadiusMin         = 0.5
	editorBrushRadiusMax         = 45.0
	editorBrushRadiusAdjustSpeed = 16.0

	editorBrushStrengthDefault     = 2.2
	editorBrushStrengthMin         = 0.1
	editorBrushStrengthMax         = 18.0
	editorBrushStrengthAdjustSpeed = 6.0

	editorBrushStrokeDeltaTimeMin = 1.0 / 480.0
	editorBrushStrokeDeltaTimeMax = 1.0 / 20.0

	editorBrushPreviewYOffset = 0.06
	editorRaycastMaxDistance  = 3000.0
	editorLightMarkerLiftMul  = 0.52
	editorLightPickRadiusMul  = 1.35
)
