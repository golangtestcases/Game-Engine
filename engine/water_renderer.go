package engine

import (
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// WaterRenderer отвечает за отдельный water-pass:
// загрузку шейдеров воды, параметры оптики и вспомогательные normal-map текстуры.
type WaterRenderer struct {
	program uint32

	normalMap0 uint32
	normalMap1 uint32

	mvpUniform             int32
	modelUniform           int32
	viewPosUniform         int32
	timeUniform            int32
	waterColorUniform      int32
	shallowColorUniform    int32
	depthColorUniform      int32
	foamColorUniform       int32
	lightDirUniform        int32
	lightColorUniform      int32
	fresnelStrengthUniform int32
	foamIntensityUniform   int32
	absorptionUniform      int32
	roughnessUniform       int32
	waveAmplitudeUniform   int32
	waveFrequencyUniform   int32
	waveSpeedUniform       int32
	underwaterUniform      int32
	screenSizeUniform      int32
	cameraNearUniform      int32
	cameraFarUniform       int32
	sceneColorTexUniform   int32
	sceneDepthTexUniform   int32
	reflectionTexUniform   int32
	normalMap0Uniform      int32
	normalMap1Uniform      int32
	refractionScaleUniform int32
	reflectionScaleUniform int32
}

// NewWaterRenderer инициализирует water shader и uniform-кэш.
func NewWaterRenderer() *WaterRenderer {
	vertexShader := loadShaderSource("assets/shaders/water/ocean.vert")
	fragmentShader := loadShaderSource("assets/shaders/water/ocean.frag")

	vs := compileShader(vertexShader, gl.VERTEX_SHADER)
	fs := compileShader(fragmentShader, gl.FRAGMENT_SHADER)

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
		panic("water shader link failed: " + string(logText))
	}

	return &WaterRenderer{
		program:                program,
		normalMap0:             generateNormalMapTexture(1024, 17.0, 23.0),
		normalMap1:             generateNormalMapTexture(1024, 9.0, 31.0),
		mvpUniform:             gl.GetUniformLocation(program, gl.Str("MVP\x00")),
		modelUniform:           gl.GetUniformLocation(program, gl.Str("model\x00")),
		viewPosUniform:         gl.GetUniformLocation(program, gl.Str("viewPos\x00")),
		timeUniform:            gl.GetUniformLocation(program, gl.Str("time\x00")),
		waterColorUniform:      gl.GetUniformLocation(program, gl.Str("waterColor\x00")),
		shallowColorUniform:    gl.GetUniformLocation(program, gl.Str("shallowColor\x00")),
		depthColorUniform:      gl.GetUniformLocation(program, gl.Str("depthColor\x00")),
		foamColorUniform:       gl.GetUniformLocation(program, gl.Str("foamColor\x00")),
		lightDirUniform:        gl.GetUniformLocation(program, gl.Str("lightDir\x00")),
		lightColorUniform:      gl.GetUniformLocation(program, gl.Str("lightColor\x00")),
		fresnelStrengthUniform: gl.GetUniformLocation(program, gl.Str("fresnelStrength\x00")),
		foamIntensityUniform:   gl.GetUniformLocation(program, gl.Str("foamIntensity\x00")),
		absorptionUniform:      gl.GetUniformLocation(program, gl.Str("absorption\x00")),
		roughnessUniform:       gl.GetUniformLocation(program, gl.Str("roughness\x00")),
		waveAmplitudeUniform:   gl.GetUniformLocation(program, gl.Str("waveAmplitude\x00")),
		waveFrequencyUniform:   gl.GetUniformLocation(program, gl.Str("waveFrequency\x00")),
		waveSpeedUniform:       gl.GetUniformLocation(program, gl.Str("waveSpeed\x00")),
		underwaterUniform:      gl.GetUniformLocation(program, gl.Str("underwater\x00")),
		screenSizeUniform:      gl.GetUniformLocation(program, gl.Str("screenSize\x00")),
		cameraNearUniform:      gl.GetUniformLocation(program, gl.Str("cameraNear\x00")),
		cameraFarUniform:       gl.GetUniformLocation(program, gl.Str("cameraFar\x00")),
		sceneColorTexUniform:   gl.GetUniformLocation(program, gl.Str("sceneColorTex\x00")),
		sceneDepthTexUniform:   gl.GetUniformLocation(program, gl.Str("sceneDepthTex\x00")),
		reflectionTexUniform:   gl.GetUniformLocation(program, gl.Str("reflectionTex\x00")),
		normalMap0Uniform:      gl.GetUniformLocation(program, gl.Str("normalMap0\x00")),
		normalMap1Uniform:      gl.GetUniformLocation(program, gl.Str("normalMap1\x00")),
		refractionScaleUniform: gl.GetUniformLocation(program, gl.Str("refractionScale\x00")),
		reflectionScaleUniform: gl.GetUniformLocation(program, gl.Str("reflectionScale\x00")),
	}
}

// loadShaderSource пытается найти GLSL-файл в разных рабочих директориях.
// Это позволяет запускать пример как из корня, так и из подкаталогов.
func loadShaderSource(path string) string {
	candidates := []string{
		path,
		filepath.Join("..", "..", path),
	}
	var data []byte
	var err error
	for _, candidate := range candidates {
		data, err = os.ReadFile(candidate)
		if err == nil {
			break
		}
	}
	if err != nil {
		panic("load water shader source failed: " + err.Error())
	}
	src := string(data)
	if !strings.HasSuffix(src, "\x00") {
		src += "\x00"
	}
	return src
}

// generateNormalMapTexture процедурно генерирует tileable normal-map
// для мелкой ряби поверхности воды без внешних текстур.
func generateNormalMapTexture(size int, phase, frequency float64) uint32 {
	data := make([]uint8, size*size*3)

	sample := func(x, y int) float64 {
		xx := (x%size + size) % size
		yy := (y%size + size) % size
		fx := float64(xx) / float64(size)
		fy := float64(yy) / float64(size)
		h1 := math.Sin((fx*frequency+phase)*2.0*math.Pi) * 0.42
		h2 := math.Cos((fy*frequency*0.86-phase)*2.0*math.Pi) * 0.34
		h3 := math.Sin((fx+fy+phase*0.13)*2.0*math.Pi*3.2) * 0.24
		return h1 + h2 + h3
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			hL := sample(x-1, y)
			hR := sample(x+1, y)
			hD := sample(x, y-1)
			hU := sample(x, y+1)

			nx := -(hR - hL)
			ny := -(hU - hD)
			nz := 1.0
			nLen := math.Sqrt(nx*nx + ny*ny + nz*nz)
			nx /= nLen
			ny /= nLen
			nz /= nLen

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

func (w *WaterRenderer) Use() {
	gl.UseProgram(w.program)
}

func (w *WaterRenderer) SetMVP(mvp mgl32.Mat4) {
	gl.UniformMatrix4fv(w.mvpUniform, 1, false, &mvp[0])
}

func (w *WaterRenderer) SetModel(model mgl32.Mat4) {
	gl.UniformMatrix4fv(w.modelUniform, 1, false, &model[0])
}

func (w *WaterRenderer) SetViewPos(pos mgl32.Vec3) {
	gl.Uniform3f(w.viewPosUniform, pos.X(), pos.Y(), pos.Z())
}

func (w *WaterRenderer) SetTime(time float32) {
	gl.Uniform1f(w.timeUniform, time)
}

func (w *WaterRenderer) SetWaterColor(color mgl32.Vec3) {
	gl.Uniform3f(w.waterColorUniform, color.X(), color.Y(), color.Z())
}

func (w *WaterRenderer) SetShallowColor(color mgl32.Vec3) {
	gl.Uniform3f(w.shallowColorUniform, color.X(), color.Y(), color.Z())
}

func (w *WaterRenderer) SetDepthColor(color mgl32.Vec3) {
	gl.Uniform3f(w.depthColorUniform, color.X(), color.Y(), color.Z())
}

func (w *WaterRenderer) SetLight(direction, color mgl32.Vec3) {
	gl.Uniform3f(w.lightDirUniform, direction.X(), direction.Y(), direction.Z())
	gl.Uniform3f(w.lightColorUniform, color.X(), color.Y(), color.Z())
}

func (w *WaterRenderer) SetFresnelStrength(strength float32) {
	gl.Uniform1f(w.fresnelStrengthUniform, strength)
}

func (w *WaterRenderer) SetFoam(color mgl32.Vec3, intensity float32) {
	gl.Uniform3f(w.foamColorUniform, color.X(), color.Y(), color.Z())
	gl.Uniform1f(w.foamIntensityUniform, intensity)
}

func (w *WaterRenderer) SetAbsorption(value float32) {
	gl.Uniform1f(w.absorptionUniform, value)
}

func (w *WaterRenderer) SetRoughness(value float32) {
	gl.Uniform1f(w.roughnessUniform, value)
}

func (w *WaterRenderer) SetWaveParams(amplitude, frequency, speed float32) {
	gl.Uniform1f(w.waveAmplitudeUniform, amplitude)
	gl.Uniform1f(w.waveFrequencyUniform, frequency)
	gl.Uniform1f(w.waveSpeedUniform, speed)
}

func (w *WaterRenderer) SetUnderwater(underwater bool) {
	if underwater {
		gl.Uniform1f(w.underwaterUniform, 1.0)
	} else {
		gl.Uniform1f(w.underwaterUniform, 0.0)
	}
}

func (w *WaterRenderer) SetScreenSize(width, height float32) {
	gl.Uniform2f(w.screenSizeUniform, width, height)
}

func (w *WaterRenderer) SetCameraRange(nearPlane, farPlane float32) {
	gl.Uniform1f(w.cameraNearUniform, nearPlane)
	gl.Uniform1f(w.cameraFarUniform, farPlane)
}

func (w *WaterRenderer) SetSceneTextures(colorUnit, depthUnit int32) {
	gl.Uniform1i(w.sceneColorTexUniform, colorUnit)
	gl.Uniform1i(w.sceneDepthTexUniform, depthUnit)
}

func (w *WaterRenderer) SetReflectionTexture(unit int32) {
	gl.Uniform1i(w.reflectionTexUniform, unit)
}

func (w *WaterRenderer) SetDetailNormalMaps(unit0, unit1 int32) {
	gl.Uniform1i(w.normalMap0Uniform, unit0)
	gl.Uniform1i(w.normalMap1Uniform, unit1)
}

func (w *WaterRenderer) BindDetailNormalMaps(unit0, unit1 uint32) {
	gl.ActiveTexture(gl.TEXTURE0 + unit0)
	gl.BindTexture(gl.TEXTURE_2D, w.normalMap0)
	gl.ActiveTexture(gl.TEXTURE0 + unit1)
	gl.BindTexture(gl.TEXTURE_2D, w.normalMap1)
}

func (w *WaterRenderer) SetOptics(refractionScale, reflectionScale float32) {
	gl.Uniform1f(w.refractionScaleUniform, refractionScale)
	gl.Uniform1f(w.reflectionScaleUniform, reflectionScale)
}

// Draw рисует водную поверхность с альфа-смешиванием поверх базовой сцены.
// Depth write временно отключается, чтобы вода не «запирала» последующие прозрачные эффекты.
func (w *WaterRenderer) Draw(mesh Mesh) {
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.DepthMask(false)
	gl.BindVertexArray(mesh.VAO)
	gl.DrawArrays(gl.TRIANGLES, 0, mesh.VertexCount)
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
}
