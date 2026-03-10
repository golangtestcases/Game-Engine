package main

import (
	"math"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"

	"subnautica-lite/engine"
)

const (
	// Базовый transform viewmodel относительно камеры.
	firstPersonHandsBaseOffsetX  = 0.21
	firstPersonHandsBaseOffsetY  = -0.25
	firstPersonHandsBaseOffsetZ  = 0.58
	firstPersonHandsBasePitchDeg = 18.0
	firstPersonHandsBaseYawDeg   = 9.5
	firstPersonHandsBaseRollDeg  = 16.0

	// Параметры idle-анимации рук.
	firstPersonHandsIdleVerticalAmplitude = 0.014
	firstPersonHandsIdleLateralAmplitude  = 0.011
	firstPersonHandsIdleForwardAmplitude  = 0.009
	firstPersonHandsIdleRollAmplitudeDeg  = 2.6
	firstPersonHandsIdleYawAmplitudeDeg   = 1.8
	firstPersonHandsIdleSpeed             = 0.21

	// Параметры гребковой анимации при плавании.
	firstPersonHandsSwimAmplitude        = 0.17
	firstPersonHandsSwimCycleSpeed       = 0.84
	firstPersonHandsBlendSpeed           = 5.0
	firstPersonHandsMovementThreshold    = 0.25
	firstPersonHandsMovementRange        = 2.0
	firstPersonHandsStrokeWidth          = 0.115
	firstPersonHandsRecoveryStrength     = 0.12
	firstPersonHandsRightPhaseOffset     = 0.11
	firstPersonHandsViewmodelFOVDeg      = 0.0
	firstPersonHandsViewmodelNear        = 0.025
	firstPersonHandsViewmodelFar         = 6.0
	firstPersonHandsForearmOffsetY       = 0.007
	firstPersonHandsForearmOffsetZ       = 0.004
	firstPersonHandsHandOffsetY          = 0.006
	firstPersonHandsHandOffsetZ          = 0.006
	firstPersonHandsHandTiltDeg          = -2.5
	firstPersonHandsRightVerticalOffset  = -0.010
	firstPersonHandsUnderwaterDepthScale = 0.32
)

type firstPersonHandPose struct {
	offset       mgl32.Vec3
	pitch        float32
	yaw          float32
	roll         float32
	forearmPitch float32
	forearmRoll  float32
}

type firstPersonHandPart struct {
	mesh       engine.Mesh
	localModel mgl32.Mat4
	color      mgl32.Vec3
}

type firstPersonHands struct {
	forearmMesh   engine.Mesh
	rightHandMesh engine.Mesh
	leftHandMesh  engine.Mesh
	material      engine.Material
	swimBlend     float32
	swimPhase     float32
	timeSec       float32
}

// newFirstPersonHands процедурно строит меши предплечья и кистей.
func newFirstPersonHands(ctx *engine.Context) *firstPersonHands {
	if ctx == nil || ctx.Renderer == nil {
		return nil
	}

	forearmData := buildProceduralFirstPersonForearmMesh()
	rightHandData := buildProceduralFirstPersonHandMesh(false)
	leftHandData := buildProceduralFirstPersonHandMesh(true)
	material := engine.NewMaterial(
		mgl32.Vec3{0.18, 0.14, 0.11},
		mgl32.Vec3{0.62, 0.50, 0.43},
		mgl32.Vec3{0.16, 0.13, 0.11},
		engine.ShininessFromSmoothness(0.10),
	)
	material.SpecularStrength = 0.36

	return &firstPersonHands{
		forearmMesh:   ctx.Renderer.NewMeshWithNormals(forearmData.vertices, forearmData.normals),
		rightHandMesh: ctx.Renderer.NewMeshWithNormals(rightHandData.vertices, rightHandData.normals),
		leftHandMesh:  ctx.Renderer.NewMeshWithNormals(leftHandData.vertices, leftHandData.normals),
		material:      material,
	}
}

// Update обновляет blend между idle и swim-позой в зависимости от скорости движения.
func (h *firstPersonHands) Update(dt, timeSec, movementSpeed float32, inWater bool) {
	if h == nil {
		return
	}

	targetSwim := float32(0)
	if inWater {
		targetSwim = clamp01((movementSpeed - firstPersonHandsMovementThreshold) / firstPersonHandsMovementRange)
	}
	h.swimBlend = expSmoothing(h.swimBlend, targetSwim, firstPersonHandsBlendSpeed, dt)

	cycleRate := firstPersonHandsSwimCycleSpeed * (0.20 + 0.80*h.swimBlend)
	h.swimPhase = fpHandsWrap01(h.swimPhase + cycleRate*dt)
	h.timeSec = timeSec
}

// Render рисует руки в пространстве камеры как viewmodel.
// Глубина пишется с `DepthFunc(ALWAYS)`, чтобы руки не пересекались с world-геометрией.
func (h *firstPersonHands) Render(ctx *engine.Context, view mgl32.Mat4) {
	if h == nil || h.forearmMesh.VAO == 0 || h.rightHandMesh.VAO == 0 || h.leftHandMesh.VAO == 0 ||
		ctx == nil || ctx.Camera == nil || ctx.Renderer == nil {
		return
	}

	cameraTransform := firstPersonCameraTransform(ctx.Camera)
	viewProj := h.viewProjection(ctx, view)

	ctx.Renderer.Use()
	ctx.Renderer.SetViewPos(ctx.Camera.Position)
	ctx.Renderer.SetMaterial(h.material)
	ctx.Renderer.EnableWaterWaves(false)
	// Русский комментарий: viewmodel находится в непосредственной близости от камеры,
	// поэтому глубинное подводное затухание для него ослабляем, сохраняя тот же lit-путь.
	ctx.Renderer.SetUnderwaterDepthScale(firstPersonHandsUnderwaterDepthScale)
	defer ctx.Renderer.SetUnderwaterDepthScale(1.0)

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthMask(true)
	gl.DepthFunc(gl.ALWAYS)
	defer gl.DepthFunc(gl.LESS)

	lastVAO := uint32(0)
	for _, side := range []float32{-1.0, 1.0} {
		pose := h.composePose(side)
		parts := h.buildParts(pose, side)
		for i := range parts {
			part := parts[i]
			model := cameraTransform.Mul4(part.localModel)
			mvp := viewProj.Mul4(model)
			ctx.Renderer.SetModel(model)
			ctx.Renderer.SetMVP(mvp)
			ctx.Renderer.SetObjectColor(part.color)
			ctx.Renderer.SetHeightGradient(part.color.Mul(0.84), part.color.Mul(1.08), -0.07, 0.07, 0.28)
			drawMeshWithCachedVAO(part.mesh, &lastVAO)
		}
	}
}

func (h *firstPersonHands) viewProjection(ctx *engine.Context, view mgl32.Mat4) mgl32.Mat4 {
	if firstPersonHandsViewmodelFOVDeg <= 0 {
		return ctx.Projection.Mul4(view)
	}

	w, hgt := ctx.Window.GetSize()
	if hgt <= 0 {
		hgt = 1
	}
	aspect := float32(w) / float32(hgt)
	proj := mgl32.Perspective(
		mgl32.DegToRad(firstPersonHandsViewmodelFOVDeg),
		aspect,
		firstPersonHandsViewmodelNear,
		firstPersonHandsViewmodelFar,
	)
	return proj.Mul4(view)
}

func (h *firstPersonHands) composePose(side float32) firstPersonHandPose {
	idle := h.idlePose(side)
	swim := h.swimPose(side)
	pose := fpHandsLerpPose(idle, swim, h.swimBlend)

	pose.offset = pose.offset.Add(mgl32.Vec3{
		side * firstPersonHandsBaseOffsetX,
		firstPersonHandsBaseOffsetY,
		-firstPersonHandsBaseOffsetZ,
	})
	if side > 0 {
		pose.offset[1] += firstPersonHandsRightVerticalOffset
	}

	pose.pitch += fpHandsDeg(firstPersonHandsBasePitchDeg)
	pose.yaw += fpHandsDeg(firstPersonHandsBaseYawDeg * side)
	pose.roll += fpHandsDeg(firstPersonHandsBaseRollDeg * side)

	return pose
}

func (h *firstPersonHands) idlePose(side float32) firstPersonHandPose {
	mainPhase := float32(h.timeSec)*firstPersonHandsIdleSpeed*2*math.Pi + side*0.73
	secondaryPhase := float32(h.timeSec)*firstPersonHandsIdleSpeed*2*math.Pi*0.62 + side*1.83

	offset := mgl32.Vec3{
		side * firstPersonHandsIdleLateralAmplitude * float32(math.Sin(float64(secondaryPhase))),
		firstPersonHandsIdleVerticalAmplitude * float32(math.Sin(float64(mainPhase))),
		firstPersonHandsIdleForwardAmplitude * float32(math.Sin(float64(mainPhase*0.47+1.17))),
	}

	return firstPersonHandPose{
		offset:       offset,
		pitch:        fpHandsDeg(1.8 * float32(math.Sin(float64(secondaryPhase*0.73+0.31)))),
		yaw:          fpHandsDeg(side * firstPersonHandsIdleYawAmplitudeDeg * float32(math.Sin(float64(mainPhase*0.59+1.37)))),
		roll:         fpHandsDeg(side * firstPersonHandsIdleRollAmplitudeDeg * float32(math.Sin(float64(mainPhase*0.81+2.11)))),
		forearmPitch: fpHandsDeg(1.2 * float32(math.Sin(float64(mainPhase*0.64+0.44)))),
		forearmRoll:  fpHandsDeg(side * 0.9 * float32(math.Sin(float64(secondaryPhase*0.72+0.88)))),
	}
}

func (h *firstPersonHands) swimPose(side float32) firstPersonHandPose {
	phase := h.swimPhase
	amplitudeScale := float32(1.0)
	if side > 0 {
		phase = fpHandsWrap01(phase + firstPersonHandsRightPhaseOffset)
		amplitudeScale = 0.94
	}

	phase = fpHandsWrap01(phase)
	amplitude := firstPersonHandsSwimAmplitude * (0.62 + 0.38*h.swimBlend) * amplitudeScale
	strokeWidth := firstPersonHandsStrokeWidth * (0.78 + 0.22*h.swimBlend)
	if side < 0 {
		strokeWidth *= 0.96
	}
	recoil := firstPersonHandsRecoveryStrength * (0.74 + 0.26*h.swimBlend)

	pose := sampleSwimCyclePose(phase, amplitude, strokeWidth, recoil)
	pose.offset[0] *= side
	pose.yaw *= side
	pose.roll *= side
	pose.forearmRoll *= side
	return pose
}

func (h *firstPersonHands) buildParts(pose firstPersonHandPose, side float32) [2]firstPersonHandPart {
	root := mgl32.Translate3D(pose.offset.X(), pose.offset.Y(), pose.offset.Z()).
		Mul4(mgl32.HomogRotate3DZ(pose.roll)).
		Mul4(mgl32.HomogRotate3DY(pose.yaw)).
		Mul4(mgl32.HomogRotate3DX(pose.pitch))

	forearm := root.
		Mul4(mgl32.Translate3D(0, -firstPersonHandsForearmOffsetY, firstPersonHandsForearmOffsetZ)).
		Mul4(mgl32.HomogRotate3DX(pose.forearmPitch)).
		Mul4(mgl32.HomogRotate3DZ(pose.forearmRoll))

	handMesh := h.rightHandMesh
	if side < 0 {
		handMesh = h.leftHandMesh
	}
	hand := root.
		Mul4(mgl32.Translate3D(0, -firstPersonHandsHandOffsetY, -firstPersonHandsHandOffsetZ)).
		Mul4(mgl32.HomogRotate3DZ(fpHandsDeg(firstPersonHandsHandTiltDeg * side)))

	return [2]firstPersonHandPart{
		{
			mesh:       h.forearmMesh,
			localModel: forearm,
			color:      mgl32.Vec3{0.64, 0.51, 0.45},
		},
		{
			mesh:       handMesh,
			localModel: hand,
			color:      mgl32.Vec3{0.80, 0.65, 0.57},
		},
	}
}

func sampleSwimCyclePose(phase, amplitude, strokeWidth, recoil float32) firstPersonHandPose {
	keys := [...]struct {
		phase float32
		pose  firstPersonHandPose
	}{
		{
			phase: 0.00,
			pose:  firstPersonHandPose{},
		},
		{
			phase: 0.24,
			pose: firstPersonHandPose{
				offset:       mgl32.Vec3{-0.018, 0.044, -amplitude},
				pitch:        fpHandsDeg(-17),
				yaw:          fpHandsDeg(4),
				roll:         fpHandsDeg(-7),
				forearmPitch: fpHandsDeg(-12),
				forearmRoll:  fpHandsDeg(-2),
			},
		},
		{
			phase: 0.52,
			pose: firstPersonHandPose{
				offset:       mgl32.Vec3{strokeWidth, 0.010, -amplitude * 0.44},
				pitch:        fpHandsDeg(-26),
				yaw:          fpHandsDeg(23),
				roll:         fpHandsDeg(-22),
				forearmPitch: fpHandsDeg(8),
				forearmRoll:  fpHandsDeg(-15),
			},
		},
		{
			phase: 0.78,
			pose: firstPersonHandPose{
				offset:       mgl32.Vec3{strokeWidth * 0.22, -0.078, recoil},
				pitch:        fpHandsDeg(21),
				yaw:          fpHandsDeg(-11),
				roll:         fpHandsDeg(30),
				forearmPitch: fpHandsDeg(23),
				forearmRoll:  fpHandsDeg(15),
			},
		},
		{
			phase: 1.00,
			pose: firstPersonHandPose{
				offset:       mgl32.Vec3{0.014, -0.030, recoil * 0.28},
				pitch:        fpHandsDeg(8),
				yaw:          fpHandsDeg(-2),
				roll:         fpHandsDeg(12),
				forearmPitch: fpHandsDeg(10),
				forearmRoll:  fpHandsDeg(6),
			},
		},
	}

	phase = fpHandsWrap01(phase)
	for i := 0; i < len(keys)-1; i++ {
		start := keys[i]
		end := keys[i+1]
		if phase > end.phase {
			continue
		}
		span := end.phase - start.phase
		if span <= 0 {
			return end.pose
		}
		t := fpHandsSmooth01((phase - start.phase) / span)
		return fpHandsLerpPose(start.pose, end.pose, t)
	}

	return keys[len(keys)-1].pose
}

// firstPersonCameraTransform строит матрицу камеры как transform в world-space,
// чтобы привязать части рук к локальной системе координат игрока.
func firstPersonCameraTransform(camera *engine.Camera) mgl32.Mat4 {
	forward := camera.Front
	if forward.Len() < 0.0001 {
		forward = mgl32.Vec3{0, 0, -1}
	} else {
		forward = forward.Normalize()
	}

	right := forward.Cross(camera.Up)
	if right.Len() < 0.0001 {
		right = mgl32.Vec3{1, 0, 0}
	} else {
		right = right.Normalize()
	}

	up := right.Cross(forward)
	if up.Len() < 0.0001 {
		up = mgl32.Vec3{0, 1, 0}
	} else {
		up = up.Normalize()
	}

	cameraBasis := mgl32.Mat4{
		right.X(), right.Y(), right.Z(), 0,
		up.X(), up.Y(), up.Z(), 0,
		-forward.X(), -forward.Y(), -forward.Z(), 0,
		0, 0, 0, 1,
	}

	return mgl32.Translate3D(camera.Position.X(), camera.Position.Y(), camera.Position.Z()).
		Mul4(cameraBasis)
}

func fpHandsLerpPose(a, b firstPersonHandPose, t float32) firstPersonHandPose {
	return firstPersonHandPose{
		offset:       fpHandsLerpVec3(a.offset, b.offset, t),
		pitch:        fpHandsLerp(a.pitch, b.pitch, t),
		yaw:          fpHandsLerp(a.yaw, b.yaw, t),
		roll:         fpHandsLerp(a.roll, b.roll, t),
		forearmPitch: fpHandsLerp(a.forearmPitch, b.forearmPitch, t),
		forearmRoll:  fpHandsLerp(a.forearmRoll, b.forearmRoll, t),
	}
}

func fpHandsLerpVec3(a, b mgl32.Vec3, t float32) mgl32.Vec3 {
	return mgl32.Vec3{
		fpHandsLerp(a.X(), b.X(), t),
		fpHandsLerp(a.Y(), b.Y(), t),
		fpHandsLerp(a.Z(), b.Z(), t),
	}
}

func fpHandsLerp(a, b, t float32) float32 {
	return a + (b-a)*t
}

func fpHandsSmooth01(v float32) float32 {
	v = clamp01(v)
	return v * v * (3.0 - 2.0*v)
}

func fpHandsWrap01(v float32) float32 {
	if v >= 1.0 || v < 0.0 {
		v = float32(math.Mod(float64(v), 1.0))
		if v < 0 {
			v += 1.0
		}
	}
	return v
}

func fpHandsDeg(v float32) float32 {
	return mgl32.DegToRad(v)
}
