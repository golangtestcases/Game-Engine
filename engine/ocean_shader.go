package engine

import (
	"fmt"
	"math"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// oceanWaveCount — число волн в спектре Герстнера, отправляемом в GLSL.
const oceanWaveCount = 12

// OceanShader инкапсулирует шейдер океана и связанные процедурные текстуры.
type OceanShader struct {
	program uint32

	modelUniform               int32
	viewProjUniform            int32
	viewUniform                int32
	projUniform                int32
	invProjUniform             int32
	timeUniform                int32
	waveTimeScaleUniform       int32
	cameraPosUniform           int32
	screenSizeUniform          int32
	nearUniform                int32
	farUniform                 int32
	waterLevelUniform          int32
	underwaterUniform          int32
	lightDirUniform            int32
	lightColorUniform          int32
	sunIntensityUniform        int32
	shallowColorUniform        int32
	deepColorUniform           int32
	foamColorUniform           int32
	scatterColorUniform        int32
	skyHorizonUniform          int32
	skyZenithUniform           int32
	absorptionUniform          int32
	refractionUniform          int32
	reflectionUniform          int32
	roughnessUniform           int32
	foamIntensityUniform       int32
	waveData0Uniform           int32
	waveData1Uniform           int32
	usePlanarReflectionUniform int32
	coverageFadeNearUniform    int32
	coverageFadeFarUniform     int32

	sceneColorUniform        int32
	sceneDepthUniform        int32
	reflectionUniformSampler int32
	normal0Uniform           int32
	normal1Uniform           int32
	normal2Uniform           int32
	foamNoiseUniform         int32
	ssrStepsUniform          int32

	normal0Tex uint32
	normal1Tex uint32
	normal2Tex uint32
	foamNoise  uint32

	waves0 [oceanWaveCount * 4]float32
	waves1 [oceanWaveCount * 4]float32
}

// NewOceanShader собирает программу рендера океана и подготавливает all uniforms/textures.
func NewOceanShader() *OceanShader {
	vertexSource := loadShaderSourceFile("assets/shaders/water/ocean_aaa.vert")
	fragmentSource := loadShaderSourceFile("assets/shaders/water/ocean_aaa.frag")

	vs := compileShader(vertexSource, gl.VERTEX_SHADER)
	fs := compileShader(fragmentSource, gl.FRAGMENT_SHADER)

	program := gl.CreateProgram()
	gl.AttachShader(program, vs)
	gl.AttachShader(program, fs)
	gl.LinkProgram(program)

	var linkStatus int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &linkStatus)
	if linkStatus == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		logText := make([]byte, logLength+1)
		gl.GetProgramInfoLog(program, logLength, nil, &logText[0])
		panic(fmt.Sprintf("ocean shader link failed: %s", string(logText)))
	}

	shader := &OceanShader{
		program: program,

		modelUniform:               gl.GetUniformLocation(program, gl.Str("uModel\x00")),
		viewProjUniform:            gl.GetUniformLocation(program, gl.Str("uViewProj\x00")),
		viewUniform:                gl.GetUniformLocation(program, gl.Str("uView\x00")),
		projUniform:                gl.GetUniformLocation(program, gl.Str("uProjection\x00")),
		invProjUniform:             gl.GetUniformLocation(program, gl.Str("uInvProjection\x00")),
		timeUniform:                gl.GetUniformLocation(program, gl.Str("uTime\x00")),
		waveTimeScaleUniform:       gl.GetUniformLocation(program, gl.Str("uWaveTimeScale\x00")),
		cameraPosUniform:           gl.GetUniformLocation(program, gl.Str("uCameraPos\x00")),
		screenSizeUniform:          gl.GetUniformLocation(program, gl.Str("uScreenSize\x00")),
		nearUniform:                gl.GetUniformLocation(program, gl.Str("uNear\x00")),
		farUniform:                 gl.GetUniformLocation(program, gl.Str("uFar\x00")),
		waterLevelUniform:          gl.GetUniformLocation(program, gl.Str("uWaterLevel\x00")),
		underwaterUniform:          gl.GetUniformLocation(program, gl.Str("uUnderwaterBlend\x00")),
		lightDirUniform:            gl.GetUniformLocation(program, gl.Str("uLightDir\x00")),
		lightColorUniform:          gl.GetUniformLocation(program, gl.Str("uLightColor\x00")),
		sunIntensityUniform:        gl.GetUniformLocation(program, gl.Str("uSunIntensity\x00")),
		shallowColorUniform:        gl.GetUniformLocation(program, gl.Str("uShallowColor\x00")),
		deepColorUniform:           gl.GetUniformLocation(program, gl.Str("uDeepColor\x00")),
		foamColorUniform:           gl.GetUniformLocation(program, gl.Str("uFoamColor\x00")),
		scatterColorUniform:        gl.GetUniformLocation(program, gl.Str("uScatterColor\x00")),
		skyHorizonUniform:          gl.GetUniformLocation(program, gl.Str("uSkyHorizonColor\x00")),
		skyZenithUniform:           gl.GetUniformLocation(program, gl.Str("uSkyZenithColor\x00")),
		absorptionUniform:          gl.GetUniformLocation(program, gl.Str("uAbsorption\x00")),
		refractionUniform:          gl.GetUniformLocation(program, gl.Str("uRefractionStrength\x00")),
		reflectionUniform:          gl.GetUniformLocation(program, gl.Str("uReflectionStrength\x00")),
		roughnessUniform:           gl.GetUniformLocation(program, gl.Str("uRoughness\x00")),
		foamIntensityUniform:       gl.GetUniformLocation(program, gl.Str("uFoamIntensity\x00")),
		waveData0Uniform:           gl.GetUniformLocation(program, gl.Str("uWaveData0[0]\x00")),
		waveData1Uniform:           gl.GetUniformLocation(program, gl.Str("uWaveData1[0]\x00")),
		usePlanarReflectionUniform: gl.GetUniformLocation(program, gl.Str("uUsePlanarReflection\x00")),
		coverageFadeNearUniform:    gl.GetUniformLocation(program, gl.Str("uCoverageFadeNear\x00")),
		coverageFadeFarUniform:     gl.GetUniformLocation(program, gl.Str("uCoverageFadeFar\x00")),

		sceneColorUniform:        gl.GetUniformLocation(program, gl.Str("uSceneColor\x00")),
		sceneDepthUniform:        gl.GetUniformLocation(program, gl.Str("uSceneDepth\x00")),
		reflectionUniformSampler: gl.GetUniformLocation(program, gl.Str("uReflectionTex\x00")),
		normal0Uniform:           gl.GetUniformLocation(program, gl.Str("uNormalTex0\x00")),
		normal1Uniform:           gl.GetUniformLocation(program, gl.Str("uNormalTex1\x00")),
		normal2Uniform:           gl.GetUniformLocation(program, gl.Str("uNormalTex2\x00")),
		foamNoiseUniform:         gl.GetUniformLocation(program, gl.Str("uFoamNoiseTex\x00")),
		ssrStepsUniform:          gl.GetUniformLocation(program, gl.Str("uSSRMaxSteps\x00")),

		normal0Tex: generateOceanNormalTexture(1024, 0.1, 13.0),
		normal1Tex: generateOceanNormalTexture(1024, 1.7, 17.0),
		normal2Tex: generateOceanNormalTexture(1024, 3.2, 23.0),
		foamNoise:  generateFoamNoiseTexture(1024, 2.4),
	}

	shader.buildWaveSpectrum()
	shader.Use()
	gl.Uniform1i(shader.sceneColorUniform, 0)
	gl.Uniform1i(shader.sceneDepthUniform, 1)
	gl.Uniform1i(shader.reflectionUniformSampler, 2)
	gl.Uniform1i(shader.normal0Uniform, 3)
	gl.Uniform1i(shader.normal1Uniform, 4)
	gl.Uniform1i(shader.normal2Uniform, 5)
	gl.Uniform1i(shader.foamNoiseUniform, 6)
	shader.uploadWaveSpectrum()
	shader.SetSSRMaxSteps(32)
	shader.SetUsePlanarReflection(true)
	shader.SetCoverageFade(320.0, 420.0)

	return shader
}

// Use активирует программу океанского шейдера.
func (s *OceanShader) Use() {
	gl.UseProgram(s.program)
}

func (s *OceanShader) SetModel(model mgl32.Mat4) {
	gl.UniformMatrix4fv(s.modelUniform, 1, false, &model[0])
}

func (s *OceanShader) SetWaveTimeScale(scale float32) {
	gl.Uniform1f(s.waveTimeScaleUniform, scale)
}

func (s *OceanShader) SetSSRMaxSteps(steps int32) {
	if steps < 0 {
		steps = 0
	}
	if steps > 32 {
		steps = 32
	}
	gl.Uniform1i(s.ssrStepsUniform, steps)
}

func (s *OceanShader) SetUsePlanarReflection(enabled bool) {
	if enabled {
		gl.Uniform1i(s.usePlanarReflectionUniform, 1)
		return
	}
	gl.Uniform1i(s.usePlanarReflectionUniform, 0)
}

func (s *OceanShader) SetCoverageFade(nearDist, farDist float32) {
	if nearDist < 0 {
		nearDist = 0
	}
	if farDist < nearDist+0.001 {
		farDist = nearDist + 0.001
	}
	gl.Uniform1f(s.coverageFadeNearUniform, nearDist)
	gl.Uniform1f(s.coverageFadeFarUniform, farDist)
}

// SetFrameUniforms задает параметры кадра, используемые при рефракции/SSR/глубине.
func (s *OceanShader) SetFrameUniforms(viewProj, view, proj, invProj mgl32.Mat4, cameraPos mgl32.Vec3, screenW, screenH, nearPlane, farPlane, timeSec, waterLevel, underwaterBlend float32) {
	gl.UniformMatrix4fv(s.viewProjUniform, 1, false, &viewProj[0])
	gl.UniformMatrix4fv(s.viewUniform, 1, false, &view[0])
	gl.UniformMatrix4fv(s.projUniform, 1, false, &proj[0])
	gl.UniformMatrix4fv(s.invProjUniform, 1, false, &invProj[0])
	gl.Uniform3f(s.cameraPosUniform, cameraPos.X(), cameraPos.Y(), cameraPos.Z())
	gl.Uniform2f(s.screenSizeUniform, screenW, screenH)
	gl.Uniform1f(s.nearUniform, nearPlane)
	gl.Uniform1f(s.farUniform, farPlane)
	gl.Uniform1f(s.timeUniform, timeSec)
	gl.Uniform1f(s.waterLevelUniform, waterLevel)
	gl.Uniform1f(s.underwaterUniform, underwaterBlend)
}

func (s *OceanShader) SetLighting(direction, color mgl32.Vec3) {
	gl.Uniform3f(s.lightDirUniform, direction.X(), direction.Y(), direction.Z())
	gl.Uniform3f(s.lightColorUniform, color.X(), color.Y(), color.Z())
}

func (s *OceanShader) SetSkyLighting(horizon, zenith mgl32.Vec3, sunIntensity float32) {
	gl.Uniform3f(s.skyHorizonUniform, horizon.X(), horizon.Y(), horizon.Z())
	gl.Uniform3f(s.skyZenithUniform, zenith.X(), zenith.Y(), zenith.Z())
	gl.Uniform1f(s.sunIntensityUniform, sunIntensity)
}

func (s *OceanShader) SetColors(shallow, deep, foam, scatter mgl32.Vec3) {
	gl.Uniform3f(s.shallowColorUniform, shallow.X(), shallow.Y(), shallow.Z())
	gl.Uniform3f(s.deepColorUniform, deep.X(), deep.Y(), deep.Z())
	gl.Uniform3f(s.foamColorUniform, foam.X(), foam.Y(), foam.Z())
	gl.Uniform3f(s.scatterColorUniform, scatter.X(), scatter.Y(), scatter.Z())
}

func (s *OceanShader) SetOptics(absorption, refractionStrength, reflectionStrength, roughness, foamIntensity float32) {
	gl.Uniform1f(s.absorptionUniform, absorption)
	gl.Uniform1f(s.refractionUniform, refractionStrength)
	gl.Uniform1f(s.reflectionUniform, reflectionStrength)
	gl.Uniform1f(s.roughnessUniform, roughness)
	gl.Uniform1f(s.foamIntensityUniform, foamIntensity)
}

func (s *OceanShader) BindInputTextures(sceneColor, sceneDepth, reflection uint32) {
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, sceneColor)
	gl.ActiveTexture(gl.TEXTURE0 + 1)
	gl.BindTexture(gl.TEXTURE_2D, sceneDepth)
	gl.ActiveTexture(gl.TEXTURE0 + 2)
	gl.BindTexture(gl.TEXTURE_2D, reflection)

	gl.ActiveTexture(gl.TEXTURE0 + 3)
	gl.BindTexture(gl.TEXTURE_2D, s.normal0Tex)
	gl.ActiveTexture(gl.TEXTURE0 + 4)
	gl.BindTexture(gl.TEXTURE_2D, s.normal1Tex)
	gl.ActiveTexture(gl.TEXTURE0 + 5)
	gl.BindTexture(gl.TEXTURE_2D, s.normal2Tex)
	gl.ActiveTexture(gl.TEXTURE0 + 6)
	gl.BindTexture(gl.TEXTURE_2D, s.foamNoise)
}

func (s *OceanShader) Draw(mesh Mesh) {
	gl.BindVertexArray(mesh.VAO)
	gl.DrawArrays(gl.TRIANGLES, 0, mesh.VertexCount)
}

func (s *OceanShader) buildWaveSpectrum() {
	directions := []mgl32.Vec2{
		{1.0, 0.12},
		{0.85, 0.53},
		{0.28, 0.96},
		{-0.62, 0.78},
		{-0.96, 0.25},
		{-0.71, -0.69},
		{-0.18, -0.98},
		{0.46, -0.88},
		{0.90, -0.42},
		{0.73, 0.67},
		{0.08, 0.99},
		{-0.34, 0.94},
	}

	amplitudes := []float32{1.35, 1.0, 0.82, 0.66, 0.52, 0.42, 0.34, 0.28, 0.22, 0.18, 0.14, 0.1}
	wavelengths := []float32{86, 62, 47, 34, 23, 17, 12, 9, 6.5, 4.8, 3.6, 2.8}
	speeds := []float32{0.62, 0.74, 0.9, 1.02, 1.18, 1.34, 1.55, 1.74, 1.95, 2.2, 2.45, 2.75}
	steepness := []float32{0.38, 0.35, 0.33, 0.3, 0.28, 0.25, 0.23, 0.21, 0.18, 0.15, 0.13, 0.1}
	phase := []float32{0.0, 1.2, 2.6, 0.8, 2.1, 3.0, 1.7, 0.4, 2.9, 1.1, 2.3, 0.6}

	for i := 0; i < oceanWaveCount; i++ {
		dir := directions[i].Normalize()
		base0 := i * 4
		base1 := i * 4
		s.waves0[base0+0] = dir.X()
		s.waves0[base0+1] = dir.Y()
		s.waves0[base0+2] = amplitudes[i]
		s.waves0[base0+3] = wavelengths[i]
		s.waves1[base1+0] = speeds[i]
		s.waves1[base1+1] = steepness[i]
		s.waves1[base1+2] = phase[i]
		s.waves1[base1+3] = 0.0
	}
}

// uploadWaveSpectrum отправляет precomputed волновые параметры в uniform-массивы GLSL.
func (s *OceanShader) uploadWaveSpectrum() {
	gl.Uniform4fv(s.waveData0Uniform, oceanWaveCount, &s.waves0[0])
	gl.Uniform4fv(s.waveData1Uniform, oceanWaveCount, &s.waves1[0])
}

// generateOceanNormalTexture создает бесшовную normal-map ряби.
func generateOceanNormalTexture(size int, phase, frequency float64) uint32 {
	data := make([]uint8, size*size*3)

	height := func(x, y int) float64 {
		fx := float64((x%size+size)%size) / float64(size)
		fy := float64((y%size+size)%size) / float64(size)
		v0 := math.Sin((fx*frequency+phase)*2.0*math.Pi) * 0.5
		v1 := math.Cos((fy*frequency*1.27-phase*0.4)*2.0*math.Pi) * 0.34
		v2 := math.Sin((fx+fy*0.82+phase)*2.0*math.Pi*3.3) * 0.22
		v3 := math.Cos((fx*2.1-fy*1.7+phase*0.6)*2.0*math.Pi*2.4) * 0.12
		return v0 + v1 + v2 + v3
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			hL := height(x-1, y)
			hR := height(x+1, y)
			hD := height(x, y-1)
			hU := height(x, y+1)

			nx := -(hR - hL)
			ny := -(hU - hD)
			nz := 1.0
			invLen := 1.0 / math.Sqrt(nx*nx+ny*ny+nz*nz)
			nx *= invLen
			ny *= invLen
			nz *= invLen

			idx := (y*size + x) * 3
			data[idx+0] = uint8((nx*0.5 + 0.5) * 255.0)
			data[idx+1] = uint8((ny*0.5 + 0.5) * 255.0)
			data[idx+2] = uint8((nz*0.5 + 0.5) * 255.0)
		}
	}

	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB8, int32(size), int32(size), 0, gl.RGB, gl.UNSIGNED_BYTE, gl.Ptr(data))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.GenerateMipmap(gl.TEXTURE_2D)

	return tex
}

// generateFoamNoiseTexture создает шумовую карту для пены и breakup-паттернов.
func generateFoamNoiseTexture(size int, seed float64) uint32 {
	data := make([]uint8, size*size)

	sample := func(x, y int) float64 {
		fx := float64(x) / float64(size)
		fy := float64(y) / float64(size)
		v0 := math.Sin((fx*17.0+seed)*2.0*math.Pi) * math.Cos((fy*19.0-seed*0.7)*2.0*math.Pi)
		v1 := math.Sin((fx*43.0+fy*37.0+seed*1.9)*2.0*math.Pi) * 0.5
		v2 := math.Cos((fx*31.0-fy*27.0+seed*0.4)*2.0*math.Pi) * 0.35
		value := (v0 + v1 + v2) * 0.5
		value = value*0.5 + 0.5
		if value < 0 {
			value = 0
		}
		if value > 1 {
			value = 1
		}
		return value
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			v := sample(x, y)
			data[y*size+x] = uint8(v * 255.0)
		}
	}

	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.R8, int32(size), int32(size), 0, gl.RED, gl.UNSIGNED_BYTE, gl.Ptr(data))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.GenerateMipmap(gl.TEXTURE_2D)
	return tex
}
