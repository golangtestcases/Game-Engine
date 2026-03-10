package engine

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// ShadowShader — минимальный шейдер для depth-pass теней.
// Фрагментный шейдер пустой, так как нам важна только глубина.
type ShadowShader struct {
	program           uint32
	lightSpaceUniform int32
	modelUniform      int32
	waterWavesUniform int32
	timeUniform       int32
}

func NewShadowShader() *ShadowShader {
	vertexSource := `
#version 410
uniform mat4 lightSpaceMatrix;
uniform mat4 model;
uniform float waterWaves;
uniform float time;
layout (location = 0) in vec3 vp;
void main() {
    vec3 pos = vp;
    if (waterWaves > 0.5) {
        float wave1 = sin(pos.x * 0.8 + time * 1.2) * cos(pos.z * 0.6 + time * 0.9) * 0.08;
        float wave2 = sin(pos.x * 1.3 - time * 0.7) * cos(pos.z * 1.1 - time * 1.1) * 0.05;
        pos.y += wave1 + wave2;
    }
    gl_Position = lightSpaceMatrix * model * vec4(pos, 1.0);
}
` + "\x00"

	fragmentSource := `
#version 410
void main() {
}
` + "\x00"

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
		panic(fmt.Sprintf("shadow shader link failed: %s", string(logText)))
	}

	return &ShadowShader{
		program:           program,
		lightSpaceUniform: gl.GetUniformLocation(program, gl.Str("lightSpaceMatrix\x00")),
		modelUniform:      gl.GetUniformLocation(program, gl.Str("model\x00")),
		waterWavesUniform: gl.GetUniformLocation(program, gl.Str("waterWaves\x00")),
		timeUniform:       gl.GetUniformLocation(program, gl.Str("time\x00")),
	}
}

func (s *ShadowShader) Use() {
	gl.UseProgram(s.program)
}

func (s *ShadowShader) SetLightSpaceMatrix(mat mgl32.Mat4) {
	gl.UniformMatrix4fv(s.lightSpaceUniform, 1, false, &mat[0])
}

func (s *ShadowShader) SetModel(mat mgl32.Mat4) {
	gl.UniformMatrix4fv(s.modelUniform, 1, false, &mat[0])
}

// SetWaterWaves включает деформацию воды в depth-pass,
// чтобы тени совпадали с анимированной поверхностью.
func (s *ShadowShader) SetWaterWaves(enable bool) {
	if enable {
		gl.Uniform1f(s.waterWavesUniform, 1.0)
	} else {
		gl.Uniform1f(s.waterWavesUniform, 0.0)
	}
}

func (s *ShadowShader) SetTime(time float32) {
	gl.Uniform1f(s.timeUniform, time)
}
