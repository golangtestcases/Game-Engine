package engine

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// SkyAtmosphereParams описывает параметры процедурного неба и солнца.
type SkyAtmosphereParams struct {
	SunDirection     mgl32.Vec3
	SunColor         mgl32.Vec3
	SunIntensity     float32
	SunDiscSize      float32
	SunHaloIntensity float32
	HorizonColor     mgl32.Vec3
	ZenithColor      mgl32.Vec3
	AtmosphereBlend  float32
	FogSunInfluence  float32
}

func DefaultSkyAtmosphereParams() SkyAtmosphereParams {
	return SkyAtmosphereParams{
		SunDirection:     mgl32.Vec3{0.32, -0.92, 0.18}.Normalize(),
		SunColor:         mgl32.Vec3{1.0, 0.93, 0.83},
		SunIntensity:     2.1,
		SunDiscSize:      0.032,
		SunHaloIntensity: 1.3,
		HorizonColor:     mgl32.Vec3{0.66, 0.84, 0.98},
		ZenithColor:      mgl32.Vec3{0.11, 0.42, 0.77},
		AtmosphereBlend:  1.0,
		FogSunInfluence:  0.42,
	}
}

// SkyRenderer рисует небо полноэкранным pass'ом на основе обратных матриц камеры.
type SkyRenderer struct {
	program uint32
	quad    *FullscreenQuad

	invProjectionUniform   int32
	invViewUniform         int32
	sunDirUniform          int32
	sunColorUniform        int32
	sunIntensityUniform    int32
	sunDiscSizeUniform     int32
	sunHaloUniform         int32
	horizonColorUniform    int32
	zenithColorUniform     int32
	atmosphereBlendUniform int32
}

func NewSkyRenderer() *SkyRenderer {
	program := newFullscreenProgram(
		"assets/shaders/water/underwater_post.vert",
		"assets/shaders/sky/atmosphere.frag",
		"sky atmosphere",
	)
	return &SkyRenderer{
		program: program,
		quad:    NewFullscreenQuad(),

		invProjectionUniform:   gl.GetUniformLocation(program, gl.Str("uInvProjection\x00")),
		invViewUniform:         gl.GetUniformLocation(program, gl.Str("uInvView\x00")),
		sunDirUniform:          gl.GetUniformLocation(program, gl.Str("uSunDirection\x00")),
		sunColorUniform:        gl.GetUniformLocation(program, gl.Str("uSunColor\x00")),
		sunIntensityUniform:    gl.GetUniformLocation(program, gl.Str("uSunIntensity\x00")),
		sunDiscSizeUniform:     gl.GetUniformLocation(program, gl.Str("uSunDiscSize\x00")),
		sunHaloUniform:         gl.GetUniformLocation(program, gl.Str("uSunHaloIntensity\x00")),
		horizonColorUniform:    gl.GetUniformLocation(program, gl.Str("uHorizonColor\x00")),
		zenithColorUniform:     gl.GetUniformLocation(program, gl.Str("uZenithColor\x00")),
		atmosphereBlendUniform: gl.GetUniformLocation(program, gl.Str("uAtmosphereBlend\x00")),
	}
}

// Render рисует фон неба. Depth временно отключается,
// чтобы pass не конфликтовал с геометрией сцены.
func (r *SkyRenderer) Render(view, projection mgl32.Mat4, params SkyAtmosphereParams) {
	params = clampSkyAtmosphereParams(params)

	gl.Disable(gl.DEPTH_TEST)
	gl.DepthMask(false)
	gl.UseProgram(r.program)

	invProjection := projection.Inv()
	invView := view.Inv()
	gl.UniformMatrix4fv(r.invProjectionUniform, 1, false, &invProjection[0])
	gl.UniformMatrix4fv(r.invViewUniform, 1, false, &invView[0])
	gl.Uniform3f(r.sunDirUniform, params.SunDirection.X(), params.SunDirection.Y(), params.SunDirection.Z())
	gl.Uniform3f(r.sunColorUniform, params.SunColor.X(), params.SunColor.Y(), params.SunColor.Z())
	gl.Uniform1f(r.sunIntensityUniform, params.SunIntensity)
	gl.Uniform1f(r.sunDiscSizeUniform, params.SunDiscSize)
	gl.Uniform1f(r.sunHaloUniform, params.SunHaloIntensity)
	gl.Uniform3f(r.horizonColorUniform, params.HorizonColor.X(), params.HorizonColor.Y(), params.HorizonColor.Z())
	gl.Uniform3f(r.zenithColorUniform, params.ZenithColor.X(), params.ZenithColor.Y(), params.ZenithColor.Z())
	gl.Uniform1f(r.atmosphereBlendUniform, params.AtmosphereBlend)

	r.quad.Draw()

	gl.DepthMask(true)
	gl.Enable(gl.DEPTH_TEST)
}

func clampSkyAtmosphereParams(params SkyAtmosphereParams) SkyAtmosphereParams {
	if params.SunDirection.Len() < 0.0001 {
		params.SunDirection = mgl32.Vec3{0.32, -0.92, 0.18}
	}
	params.SunDirection = params.SunDirection.Normalize()

	if params.SunIntensity < 0.01 {
		params.SunIntensity = 0.01
	}
	if params.SunDiscSize < 0.004 {
		params.SunDiscSize = 0.004
	}
	if params.SunDiscSize > 0.095 {
		params.SunDiscSize = 0.095
	}
	if params.SunHaloIntensity < 0 {
		params.SunHaloIntensity = 0
	}
	if params.AtmosphereBlend < 0 {
		params.AtmosphereBlend = 0
	}
	if params.FogSunInfluence < 0 {
		params.FogSunInfluence = 0
	}
	return params
}
