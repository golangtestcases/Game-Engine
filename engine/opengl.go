package engine

import (
	"fmt"
	"math"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// Mesh stores OpenGL buffers and the vertex count for draw calls.
type Mesh struct {
	VAO         uint32
	VBO         uint32
	NBO         uint32
	VertexCount int32
	HasNormals  bool
}

// UnderwaterCausticsParams задаёт художественные параметры подводной каустики
// в базовом lit-пайплайне (без отдельного рендер-прохода).
type UnderwaterCausticsParams struct {
	Speed     float32
	Scale     float32
	Intensity float32
	Contrast  float32
	DepthFade float32
}

func DefaultUnderwaterCausticsParams() UnderwaterCausticsParams {
	return UnderwaterCausticsParams{
		Speed:     0.24,
		Scale:     0.09,
		Intensity: 0.28,
		Contrast:  1.35,
		DepthFade: 0.08,
	}
}

// ClampUnderwaterCausticsParams нормализует параметры каустики для единого lit-пайплайна.
func ClampUnderwaterCausticsParams(params UnderwaterCausticsParams) UnderwaterCausticsParams {
	if params.Speed < 0 {
		params.Speed = 0
	}
	if params.Scale < 0.001 {
		params.Scale = 0.001
	}
	if params.Intensity < 0 {
		params.Intensity = 0
	}
	if params.Contrast < 0.1 {
		params.Contrast = 0.1
	}
	if params.Contrast > 4.0 {
		params.Contrast = 4.0
	}
	if params.DepthFade < 0 {
		params.DepthFade = 0
	}
	if params.DepthFade > 2.5 {
		params.DepthFade = 2.5
	}
	return params
}

// lightUniformSet РєСЌС€РёСЂСѓРµС‚ Р»РѕРєР°С†РёРё uniforms РґР»СЏ РѕРґРЅРѕРіРѕ СЌР»РµРјРµРЅС‚Р° РјР°СЃСЃРёРІР° lights[i].
type lightUniformSet struct {
	typeUniform        int32
	positionUniform    int32
	directionUniform   int32
	colorUniform       int32
	intensityUniform   int32
	constantUniform    int32
	linearUniform      int32
	quadraticUniform   int32
	cutOffUniform      int32
	outerCutOffUniform int32
}

// Renderer encapsulates a default shader and its uniform locations.
type Renderer struct {
	program uint32

	mvpUniform                  int32
	modelUniform                int32
	normalMatrixUniform         int32
	colorUniform                int32
	lowColorUniform             int32
	highColorUniform            int32
	heightRangeUniform          int32
	heightTintUniform           int32
	fogColorUniform             int32
	fogRangeUniform             int32
	fogStrengthUniform          int32
	fogAmountUniform            int32
	underwaterBlendUniform      int32
	underwaterFogDensityUniform int32
	depthTintUniform            int32
	sunAttenuationUniform       int32
	visibilityDistanceUniform   int32
	underwaterDepthScaleUniform int32
	timeUniform                 int32
	causticsSpeedUniform        int32
	causticsScaleUniform        int32
	causticsIntensityUniform    int32
	causticsContrastUniform     int32
	causticsDepthFadeUniform    int32
	waterLevelUniform           int32
	waterWavesUniform           int32
	viewPosUniform              int32
	ambientColorUniform         int32
	ambientIntensityUniform     int32
	matAmbientUniform           int32
	matDiffuseUniform           int32
	matSpecularUniform          int32
	matEmissionUniform          int32
	matShininessUniform         int32
	matSpecularStrengthUniform  int32
	numLightsUniform            int32
	shadowLightIndexUniform     int32
	lightSpaceUniform           int32
	shadowMapUniform            int32
	shadowBiasMinUniform        int32
	shadowBiasSlopeUniform      int32
	shadowStrengthUniform       int32
	clipPlaneUniform            int32
	clipEnabledUniform          int32
	causticTex0Uniform          int32
	causticTex1Uniform          int32

	causticTex0 uint32
	causticTex1 uint32

	lightUniforms [MaxLights]lightUniformSet
}

func NewRenderer() *Renderer {
	vertexShaderSource := loadShaderSourceFile("assets/shaders/scene/lit.vert")
	fragmentShaderSource := loadShaderSourceFile("assets/shaders/scene/lit.frag")

	vertexShader := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	fragmentShader := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var linkStatus int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &linkStatus)
	if linkStatus == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		logText := make([]byte, logLength+1)
		gl.GetProgramInfoLog(program, logLength, nil, &logText[0])
		panic(fmt.Sprintf("shader link failed: %s", string(logText)))
	}

	renderer := &Renderer{
		program:                     program,
		mvpUniform:                  gl.GetUniformLocation(program, gl.Str("MVP\x00")),
		modelUniform:                gl.GetUniformLocation(program, gl.Str("model\x00")),
		normalMatrixUniform:         gl.GetUniformLocation(program, gl.Str("normalMatrix\x00")),
		colorUniform:                gl.GetUniformLocation(program, gl.Str("objectColor\x00")),
		lowColorUniform:             gl.GetUniformLocation(program, gl.Str("lowColor\x00")),
		highColorUniform:            gl.GetUniformLocation(program, gl.Str("highColor\x00")),
		heightRangeUniform:          gl.GetUniformLocation(program, gl.Str("heightRange\x00")),
		heightTintUniform:           gl.GetUniformLocation(program, gl.Str("heightTint\x00")),
		fogColorUniform:             gl.GetUniformLocation(program, gl.Str("fogColor\x00")),
		fogRangeUniform:             gl.GetUniformLocation(program, gl.Str("fogRange\x00")),
		fogStrengthUniform:          gl.GetUniformLocation(program, gl.Str("fogStrength\x00")),
		fogAmountUniform:            gl.GetUniformLocation(program, gl.Str("fogAmount\x00")),
		underwaterBlendUniform:      gl.GetUniformLocation(program, gl.Str("underwaterBlend\x00")),
		underwaterFogDensityUniform: gl.GetUniformLocation(program, gl.Str("underwaterFogDensity\x00")),
		depthTintUniform:            gl.GetUniformLocation(program, gl.Str("depthTint\x00")),
		sunAttenuationUniform:       gl.GetUniformLocation(program, gl.Str("sunlightAttenuation\x00")),
		visibilityDistanceUniform:   gl.GetUniformLocation(program, gl.Str("visibilityDistance\x00")),
		underwaterDepthScaleUniform: gl.GetUniformLocation(program, gl.Str("underwaterDepthScale\x00")),
		timeUniform:                 gl.GetUniformLocation(program, gl.Str("time\x00")),
		causticsSpeedUniform:        gl.GetUniformLocation(program, gl.Str("causticsSpeed\x00")),
		causticsScaleUniform:        gl.GetUniformLocation(program, gl.Str("causticsScale\x00")),
		causticsIntensityUniform:    gl.GetUniformLocation(program, gl.Str("causticsIntensity\x00")),
		causticsContrastUniform:     gl.GetUniformLocation(program, gl.Str("causticsContrast\x00")),
		causticsDepthFadeUniform:    gl.GetUniformLocation(program, gl.Str("causticsDepthFade\x00")),
		waterLevelUniform:           gl.GetUniformLocation(program, gl.Str("waterLevel\x00")),
		waterWavesUniform:           gl.GetUniformLocation(program, gl.Str("waterWaves\x00")),
		viewPosUniform:              gl.GetUniformLocation(program, gl.Str("viewPos\x00")),
		ambientColorUniform:         gl.GetUniformLocation(program, gl.Str("ambientLight.color\x00")),
		ambientIntensityUniform:     gl.GetUniformLocation(program, gl.Str("ambientLight.intensity\x00")),
		matAmbientUniform:           gl.GetUniformLocation(program, gl.Str("material.ambient\x00")),
		matDiffuseUniform:           gl.GetUniformLocation(program, gl.Str("material.diffuse\x00")),
		matSpecularUniform:          gl.GetUniformLocation(program, gl.Str("material.specular\x00")),
		matEmissionUniform:          gl.GetUniformLocation(program, gl.Str("material.emission\x00")),
		matShininessUniform:         gl.GetUniformLocation(program, gl.Str("material.shininess\x00")),
		matSpecularStrengthUniform:  gl.GetUniformLocation(program, gl.Str("material.specularStrength\x00")),
		numLightsUniform:            gl.GetUniformLocation(program, gl.Str("numLights\x00")),
		shadowLightIndexUniform:     gl.GetUniformLocation(program, gl.Str("shadowLightIndex\x00")),
		lightSpaceUniform:           gl.GetUniformLocation(program, gl.Str("lightSpaceMatrix\x00")),
		shadowMapUniform:            gl.GetUniformLocation(program, gl.Str("shadowMap\x00")),
		shadowBiasMinUniform:        gl.GetUniformLocation(program, gl.Str("shadowBiasMin\x00")),
		shadowBiasSlopeUniform:      gl.GetUniformLocation(program, gl.Str("shadowBiasSlope\x00")),
		shadowStrengthUniform:       gl.GetUniformLocation(program, gl.Str("shadowStrength\x00")),
		clipPlaneUniform:            gl.GetUniformLocation(program, gl.Str("clipPlane\x00")),
		clipEnabledUniform:          gl.GetUniformLocation(program, gl.Str("clipEnabled\x00")),
		causticTex0Uniform:          gl.GetUniformLocation(program, gl.Str("causticTex0\x00")),
		causticTex1Uniform:          gl.GetUniformLocation(program, gl.Str("causticTex1\x00")),
	}

	for i := 0; i < MaxLights; i++ {
		prefix := fmt.Sprintf("lights[%d].", i)
		renderer.lightUniforms[i] = lightUniformSet{
			typeUniform:        gl.GetUniformLocation(program, gl.Str(prefix+"type\x00")),
			positionUniform:    gl.GetUniformLocation(program, gl.Str(prefix+"position\x00")),
			directionUniform:   gl.GetUniformLocation(program, gl.Str(prefix+"direction\x00")),
			colorUniform:       gl.GetUniformLocation(program, gl.Str(prefix+"color\x00")),
			intensityUniform:   gl.GetUniformLocation(program, gl.Str(prefix+"intensity\x00")),
			constantUniform:    gl.GetUniformLocation(program, gl.Str(prefix+"constant\x00")),
			linearUniform:      gl.GetUniformLocation(program, gl.Str(prefix+"linear\x00")),
			quadraticUniform:   gl.GetUniformLocation(program, gl.Str(prefix+"quadratic\x00")),
			cutOffUniform:      gl.GetUniformLocation(program, gl.Str(prefix+"cutOff\x00")),
			outerCutOffUniform: gl.GetUniformLocation(program, gl.Str(prefix+"outerCutOff\x00")),
		}
	}

	renderer.causticTex0 = generateCausticTexture(512, 0.0)
	renderer.causticTex1 = generateCausticTexture(512, 1.7)

	gl.UseProgram(renderer.program)
	gl.Uniform1i(renderer.shadowMapUniform, 1)
	gl.Uniform1i(renderer.causticTex0Uniform, 7)
	gl.Uniform1i(renderer.causticTex1Uniform, 8)
	gl.Uniform4f(renderer.clipPlaneUniform, 0.0, 1.0, 0.0, 0.0)
	gl.Uniform1f(renderer.clipEnabledUniform, 0.0)
	gl.Uniform1i(renderer.shadowLightIndexUniform, -1)
	gl.Uniform1f(renderer.underwaterBlendUniform, 0.0)
	gl.Uniform1f(renderer.underwaterFogDensityUniform, 1.0)
	gl.Uniform3f(renderer.depthTintUniform, 0.08, 0.30, 0.44)
	gl.Uniform1f(renderer.sunAttenuationUniform, 0.12)
	gl.Uniform1f(renderer.visibilityDistanceUniform, 90.0)
	gl.Uniform1f(renderer.underwaterDepthScaleUniform, 1.0)
	renderer.SetWaterEffects(0, DefaultUnderwaterCausticsParams())
	renderer.SetShadowSampling(DefaultShadowSamplingSettings())

	renderer.SetModel(mgl32.Ident4())
	renderer.SetMaterial(DefaultMaterial())
	renderer.SetLighting(DefaultLightingState())

	return renderer
}
func (r *Renderer) Program() uint32 {
	return r.program
}

// Use Р°РєС‚РёРІРёСЂСѓРµС‚ Р±Р°Р·РѕРІСѓСЋ РїСЂРѕРіСЂР°РјРјСѓ СЂРµРЅРґРµСЂР° Рё Р±РёРЅРґРёРЅРі РїСЂРѕС†РµРґСѓСЂРЅС‹С… С‚РµРєСЃС‚СѓСЂ РєР°СѓСЃС‚РёРєРё.
func (r *Renderer) Use() {
	gl.UseProgram(r.program)

	gl.ActiveTexture(gl.TEXTURE0 + 7)
	gl.BindTexture(gl.TEXTURE_2D, r.causticTex0)
	gl.ActiveTexture(gl.TEXTURE0 + 8)
	gl.BindTexture(gl.TEXTURE_2D, r.causticTex1)
}

func (r *Renderer) NewMesh(vertices []float32) Mesh {
	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

	return Mesh{VAO: vao, VBO: vbo, VertexCount: int32(len(vertices) / 3), HasNormals: false}
}

// NewMeshWithNormals СЃРѕР·РґР°РµС‚ mesh СЃ РґРІСѓРјСЏ Р°С‚СЂРёР±СѓС‚Р°РјРё: РїРѕР·РёС†РёСЏ (location=0) Рё РЅРѕСЂРјР°Р»СЊ (location=1).
func (r *Renderer) NewMeshWithNormals(vertices, normals []float32) Mesh {
	var vao, vbo, nbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

	gl.GenBuffers(1, &nbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, nbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(normals), gl.Ptr(normals), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 0, nil)

	return Mesh{VAO: vao, VBO: vbo, NBO: nbo, VertexCount: int32(len(vertices) / 3), HasNormals: true}
}

func (r *Renderer) UpdateMeshWithNormals(mesh *Mesh, vertices, normals []float32) {
	if mesh == nil || mesh.VAO == 0 || mesh.VBO == 0 {
		return
	}

	var vertexPtr unsafe.Pointer
	if len(vertices) > 0 {
		vertexPtr = gl.Ptr(vertices)
	}

	gl.BindVertexArray(mesh.VAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, mesh.VBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, vertexPtr, gl.DYNAMIC_DRAW)

	if mesh.HasNormals && mesh.NBO != 0 {
		var normalPtr unsafe.Pointer
		if len(normals) > 0 {
			normalPtr = gl.Ptr(normals)
		}
		gl.BindBuffer(gl.ARRAY_BUFFER, mesh.NBO)
		gl.BufferData(gl.ARRAY_BUFFER, len(normals)*4, normalPtr, gl.DYNAMIC_DRAW)
	}

	mesh.VertexCount = int32(len(vertices) / 3)
}

// DeleteMesh РѕСЃРІРѕР±РѕР¶РґР°РµС‚ OpenGL-Р±СѓС„РµСЂС‹ mesh Рё СЃР±СЂР°СЃС‹РІР°РµС‚ РµРіРѕ РїРѕР»СЏ.
func (r *Renderer) DeleteMesh(mesh *Mesh) {
	if mesh == nil {
		return
	}

	if mesh.NBO != 0 {
		gl.DeleteBuffers(1, &mesh.NBO)
		mesh.NBO = 0
	}
	if mesh.VBO != 0 {
		gl.DeleteBuffers(1, &mesh.VBO)
		mesh.VBO = 0
	}
	if mesh.VAO != 0 {
		gl.DeleteVertexArrays(1, &mesh.VAO)
		mesh.VAO = 0
	}

	mesh.VertexCount = 0
	mesh.HasNormals = false
}

func (r *Renderer) SetMVP(mvp mgl32.Mat4) {
	gl.UniformMatrix4fv(r.mvpUniform, 1, false, &mvp[0])
}

func (r *Renderer) SetModel(model mgl32.Mat4) {
	gl.UniformMatrix4fv(r.modelUniform, 1, false, &model[0])
	normalMatrix := normalMatrixFromModel(model)
	gl.UniformMatrix3fv(r.normalMatrixUniform, 1, false, &normalMatrix[0])
}

func (r *Renderer) SetLights(lights []Light) {
	r.setLightsInternal(lights, -1)
}

func (r *Renderer) setLightsInternal(lights []Light, preferredShadowIndex int) {
	lightCount := 0
	shadowLightIndex := int32(-1)
	for i := 0; i < len(lights) && lightCount < MaxLights; i++ {
		light := sanitizeLight(lights[i])
		if !isLightActive(light) {
			continue
		}

		uniforms := r.lightUniforms[lightCount]
		gl.Uniform1i(uniforms.typeUniform, int32(light.Type))
		gl.Uniform3f(uniforms.positionUniform, light.Position.X(), light.Position.Y(), light.Position.Z())
		gl.Uniform3f(uniforms.directionUniform, light.Direction.X(), light.Direction.Y(), light.Direction.Z())
		gl.Uniform3f(uniforms.colorUniform, light.Color.X(), light.Color.Y(), light.Color.Z())
		gl.Uniform1f(uniforms.intensityUniform, light.Intensity)
		gl.Uniform1f(uniforms.constantUniform, light.Constant)
		gl.Uniform1f(uniforms.linearUniform, light.Linear)
		gl.Uniform1f(uniforms.quadraticUniform, light.Quadratic)
		gl.Uniform1f(uniforms.cutOffUniform, light.CutOff)
		gl.Uniform1f(uniforms.outerCutOffUniform, light.OuterCutOff)

		if i == preferredShadowIndex && light.Type == LightDirectional {
			shadowLightIndex = int32(lightCount)
		} else if shadowLightIndex < 0 && light.Type == LightDirectional {
			shadowLightIndex = int32(lightCount)
		}
		lightCount++
	}

	gl.Uniform1i(r.numLightsUniform, int32(lightCount))
	gl.Uniform1i(r.shadowLightIndexUniform, shadowLightIndex)
}

func (r *Renderer) SetAmbientLight(ambient AmbientLight) {
	ambient = NewAmbientLight(ambient.Color, ambient.Intensity)
	gl.Uniform3f(r.ambientColorUniform, ambient.Color.X(), ambient.Color.Y(), ambient.Color.Z())
	gl.Uniform1f(r.ambientIntensityUniform, ambient.Intensity)
}

func (r *Renderer) SetLighting(state LightingState) {
	r.SetAmbientLight(state.Ambient)
	r.setLightsInternal(state.Lights, state.ShadowLightIndex)
}

func (r *Renderer) SetViewPos(pos mgl32.Vec3) {
	gl.Uniform3f(r.viewPosUniform, pos.X(), pos.Y(), pos.Z())
}

func (r *Renderer) SetMaterial(mat Material) {
	mat = SanitizeMaterial(mat)
	gl.Uniform3f(r.matAmbientUniform, mat.Ambient.X(), mat.Ambient.Y(), mat.Ambient.Z())
	gl.Uniform3f(r.matDiffuseUniform, mat.Diffuse.X(), mat.Diffuse.Y(), mat.Diffuse.Z())
	gl.Uniform3f(r.matSpecularUniform, mat.Specular.X(), mat.Specular.Y(), mat.Specular.Z())
	gl.Uniform3f(r.matEmissionUniform, mat.Emission.X(), mat.Emission.Y(), mat.Emission.Z())
	gl.Uniform1f(r.matShininessUniform, mat.Shininess)
	gl.Uniform1f(r.matSpecularStrengthUniform, mat.SpecularStrength)
}

func (r *Renderer) SetLightSpaceMatrix(mat mgl32.Mat4) {
	gl.UniformMatrix4fv(r.lightSpaceUniform, 1, false, &mat[0])
}

func (r *Renderer) SetShadowMap(unit int32) {
	gl.Uniform1i(r.shadowMapUniform, unit)
}

func (r *Renderer) SetShadowSampling(settings ShadowSamplingSettings) {
	settings = sanitizeShadowSamplingSettings(settings)
	gl.Uniform1f(r.shadowBiasMinUniform, settings.BiasMin)
	gl.Uniform1f(r.shadowBiasSlopeUniform, settings.BiasSlope)
	gl.Uniform1f(r.shadowStrengthUniform, settings.Strength)
}

// ApplyShadowState централизует light-space matrix и shadow texture binding для lit-pass.
func (r *Renderer) ApplyShadowState(state ShadowState) {
	state = sanitizeShadowState(state)
	r.SetLightSpaceMatrix(state.LightSpaceMatrix)
	if state.Map != nil {
		state.Map.BindTexture(uint32(state.TextureUnit))
		r.SetShadowMap(state.TextureUnit)
	}
}

func (r *Renderer) SetClipPlane(plane mgl32.Vec4) {
	gl.Uniform4f(r.clipPlaneUniform, plane.X(), plane.Y(), plane.Z(), plane.W())
}

func (r *Renderer) SetClipPlaneEnabled(enabled bool) {
	if enabled {
		gl.Uniform1f(r.clipEnabledUniform, 1.0)
		return
	}
	gl.Uniform1f(r.clipEnabledUniform, 0.0)
}

func (r *Renderer) SetObjectColor(color mgl32.Vec3) {
	gl.Uniform3f(r.colorUniform, color.X(), color.Y(), color.Z())
}

func (r *Renderer) SetHeightGradient(lowColor, highColor mgl32.Vec3, minY, maxY, tint float32) {
	gl.Uniform3f(r.lowColorUniform, lowColor.X(), lowColor.Y(), lowColor.Z())
	gl.Uniform3f(r.highColorUniform, highColor.X(), highColor.Y(), highColor.Z())
	gl.Uniform2f(r.heightRangeUniform, minY, maxY)
	gl.Uniform1f(r.heightTintUniform, tint)
}

func (r *Renderer) SetFog(color mgl32.Vec3, near, far, strength, amount float32) {
	gl.Uniform3f(r.fogColorUniform, color.X(), color.Y(), color.Z())
	gl.Uniform2f(r.fogRangeUniform, near, far)
	gl.Uniform1f(r.fogStrengthUniform, strength)
	gl.Uniform1f(r.fogAmountUniform, amount)
}

// SetUnderwaterAtmosphere передает параметры подводного затухания/видимости в lit-шейдер.
// Параметры используются поверх текущего lighting-пайплайна и не создают отдельный рендер-путь.
func (r *Renderer) SetUnderwaterAtmosphere(
	underwaterBlend float32,
	fogDensity float32,
	depthTint mgl32.Vec3,
	sunlightAttenuation float32,
	visibilityDistance float32,
) {
	if underwaterBlend < 0 {
		underwaterBlend = 0
	}
	if underwaterBlend > 1 {
		underwaterBlend = 1
	}
	if fogDensity < 0 {
		fogDensity = 0
	}
	if sunlightAttenuation < 0 {
		sunlightAttenuation = 0
	}
	if visibilityDistance < 1 {
		visibilityDistance = 1
	}

	gl.Uniform1f(r.underwaterBlendUniform, underwaterBlend)
	gl.Uniform1f(r.underwaterFogDensityUniform, fogDensity)
	gl.Uniform3f(r.depthTintUniform, depthTint.X(), depthTint.Y(), depthTint.Z())
	gl.Uniform1f(r.sunAttenuationUniform, sunlightAttenuation)
	gl.Uniform1f(r.visibilityDistanceUniform, visibilityDistance)
}

// SetUnderwaterDepthScale регулирует глубинный вклад подводного затухания/тумана
// в каноническом lit-пути. Для world-геометрии значение обычно 1.0.
func (r *Renderer) SetUnderwaterDepthScale(scale float32) {
	if scale < 0 {
		scale = 0
	}
	if scale > 1 {
		scale = 1
	}
	gl.Uniform1f(r.underwaterDepthScaleUniform, scale)
}

func (r *Renderer) SetWaterEffects(timeSec float32, caustics UnderwaterCausticsParams) {
	caustics = ClampUnderwaterCausticsParams(caustics)
	gl.Uniform1f(r.timeUniform, timeSec)
	gl.Uniform1f(r.causticsSpeedUniform, caustics.Speed)
	gl.Uniform1f(r.causticsScaleUniform, caustics.Scale)
	gl.Uniform1f(r.causticsIntensityUniform, caustics.Intensity)
	gl.Uniform1f(r.causticsContrastUniform, caustics.Contrast)
	gl.Uniform1f(r.causticsDepthFadeUniform, caustics.DepthFade)
}

func (r *Renderer) SetWaterLevel(level float32) {
	gl.Uniform1f(r.waterLevelUniform, level)
}

// ApplyLightingFrame — единая точка применения lighting/fog/underwater/shadow/caustics состояния кадра.
// Русский комментарий: этот метод нужен, чтобы gameplay-слой передавал только конфиг кадра,
// а не раскладывал вручную одинаковые Set* вызовы между разными местами рендера.
func (r *Renderer) ApplyLightingFrame(frame LightingFrame) {
	frame = sanitizeLightingFrame(frame)
	atmosphere := frame.Atmosphere

	r.SetLighting(frame.Lighting)
	r.ApplyShadowState(frame.Shadow)
	r.SetFog(
		atmosphere.Fog.Color,
		atmosphere.Fog.RangeNear,
		atmosphere.Fog.RangeFar,
		atmosphere.Fog.Strength,
		atmosphere.Fog.Amount,
	)
	r.SetUnderwaterAtmosphere(
		atmosphere.Underwater.Blend,
		atmosphere.Underwater.FogDensity,
		atmosphere.Underwater.DepthTint,
		atmosphere.Underwater.SunlightAttenuation,
		atmosphere.Underwater.VisibilityDistance,
	)
	r.SetUnderwaterDepthScale(atmosphere.Underwater.DepthScale)
	r.SetWaterEffects(atmosphere.TimeSec, atmosphere.Caustics)
	r.SetWaterLevel(atmosphere.WaterLevel)
}

// Р’РєР»СЋС‡РёС‚СЊ Р°РЅРёРјР°С†РёСЋ РІРѕР»РЅ (РґР»СЏ РІРѕРґРЅРѕР№ РїРѕРІРµСЂС…РЅРѕСЃС‚Рё)
func (r *Renderer) EnableWaterWaves(enable bool) {
	if enable {
		gl.Uniform1f(r.waterWavesUniform, 1.0)
	} else {
		gl.Uniform1f(r.waterWavesUniform, 0.0)
	}
}

func (r *Renderer) Draw(mesh Mesh) {
	gl.BindVertexArray(mesh.VAO)
	gl.DrawArrays(gl.TRIANGLES, 0, mesh.VertexCount)
}

func normalMatrixFromModel(model mgl32.Mat4) mgl32.Mat3 {
	invTranspose := model.Inv().Transpose()
	normalMatrix := mgl32.Mat3{
		invTranspose[0], invTranspose[1], invTranspose[2],
		invTranspose[4], invTranspose[5], invTranspose[6],
		invTranspose[8], invTranspose[9], invTranspose[10],
	}

	for i := 0; i < len(normalMatrix); i++ {
		v := float64(normalMatrix[i])
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return mgl32.Mat3{
				1, 0, 0,
				0, 1, 0,
				0, 0, 1,
			}
		}
	}
	return normalMatrix
}

func generateCausticTexture(size int, phase float64) uint32 {
	data := make([]uint8, size*size)

	sample := func(fx, fy float64) float64 {
		v0 := math.Sin((fx*7.1+phase)*2.0*math.Pi) * math.Cos((fy*6.3-phase*0.7)*2.0*math.Pi)
		v1 := math.Sin((fx+fy*0.9+phase*0.3)*2.0*math.Pi*4.7) * 0.6
		v2 := math.Cos((fx*1.4-fy*1.9+phase)*2.0*math.Pi*3.2) * 0.45
		value := (v0 + v1 + v2) * 0.5
		value = math.Abs(value)
		return math.Pow(value, 2.4)
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx := float64(x) / float64(size)
			fy := float64(y) / float64(size)
			v := sample(fx, fy)
			if v > 1.0 {
				v = 1.0
			}
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

// UnderwaterColor РІРѕР·РІСЂР°С‰Р°РµС‚ РїСЂРѕСЃС‚РѕР№ С†РІРµС‚РѕРІРѕР№ РіСЂР°РґРёРµРЅС‚ РїРѕ РіР»СѓР±РёРЅРµ.
// РСЃРїРѕР»СЊР·СѓРµС‚СЃСЏ РєР°Рє Р±Р°Р·Р° РґР»СЏ РїРѕРґРІРѕРґРЅРѕРіРѕ fog/tint РІ gameplay-СЃР»РѕРµ.
func UnderwaterColor(depth float32) (float32, float32, float32, float32) {
	r := float32(0.0)
	g := float32(0.2) + 0.3*depth/5.0
	b := float32(0.5) + 0.3*depth/5.0
	return r, g, b, 1.0
}

// compileShader РєРѕРјРїРёР»РёСЂСѓРµС‚ С€РµР№РґРµСЂ Рё РІС‹Р±СЂР°СЃС‹РІР°РµС‚ panic СЃ Р»РѕРіРѕРј РїСЂРё РѕС€РёР±РєРµ.
func compileShader(source string, shaderType uint32) uint32 {
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		logText := make([]byte, logLength+1)
		gl.GetShaderInfoLog(shader, logLength, nil, &logText[0])
		panic(fmt.Sprintf("shader compile failed: %s", string(logText)))
	}

	return shader
}
