package engine

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// GlowRenderer рисует эмиссивные (светящиеся) объекты отдельным шейдером.
// Используется для неоновых растений и других self-lit элементов сцены.
type GlowRenderer struct {
	program              uint32
	mvpUniform           int32
	modelUniform         int32
	clipPlaneUniform     int32
	clipEnabledUniform   int32
	glowColorUniform     int32
	glowIntensityUniform int32
	pulseSpeedUniform    int32
	timeUniform          int32
	viewPosUniform       int32
}

// NewGlowRenderer компилирует GLSL-программу свечения и кэширует uniform-локации.
func NewGlowRenderer() *GlowRenderer {
	vertexShaderSource := `
#version 410
uniform mat4 MVP;
uniform mat4 model;
uniform vec4 clipPlane;
uniform float clipEnabled;
layout (location = 0) in vec3 vp;
layout (location = 1) in vec3 vn;
out vec3 fragPos;
out vec3 normal;
void main() {
    vec4 worldPos = model * vec4(vp, 1.0);
    fragPos = vec3(worldPos);
    normal = mat3(transpose(inverse(model))) * vn;
    gl_ClipDistance[0] = (clipEnabled > 0.5) ? dot(worldPos, clipPlane) : 1.0;
    gl_Position = MVP * vec4(vp, 1.0);
}
` + "\x00"

	fragmentShaderSource := `
#version 410
uniform vec3 glowColor;
uniform float glowIntensity;
uniform float pulseSpeed;
uniform float time;
uniform vec3 viewPos;
in vec3 fragPos;
in vec3 normal;
out vec4 frag_colour;
void main() {
    float pulse = 1.0;
    if (pulseSpeed > 0.01) {
        pulse = 0.7 + 0.3 * sin(time * pulseSpeed);
    }
    vec3 norm = normalize(normal);
    vec3 viewDir = normalize(viewPos - fragPos);
    float rim = 1.0 - max(dot(norm, viewDir), 0.0);
    rim = pow(rim, 2.0);
    vec3 emissive = glowColor * glowIntensity * pulse * (0.8 + 0.2 * rim);
    frag_colour = vec4(emissive, 1.0);
}
` + "\x00"

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
		panic(fmt.Sprintf("glow shader link failed: %s", string(logText)))
	}

	renderer := &GlowRenderer{
		program:              program,
		mvpUniform:           gl.GetUniformLocation(program, gl.Str("MVP\x00")),
		modelUniform:         gl.GetUniformLocation(program, gl.Str("model\x00")),
		clipPlaneUniform:     gl.GetUniformLocation(program, gl.Str("clipPlane\x00")),
		clipEnabledUniform:   gl.GetUniformLocation(program, gl.Str("clipEnabled\x00")),
		glowColorUniform:     gl.GetUniformLocation(program, gl.Str("glowColor\x00")),
		glowIntensityUniform: gl.GetUniformLocation(program, gl.Str("glowIntensity\x00")),
		pulseSpeedUniform:    gl.GetUniformLocation(program, gl.Str("pulseSpeed\x00")),
		timeUniform:          gl.GetUniformLocation(program, gl.Str("time\x00")),
		viewPosUniform:       gl.GetUniformLocation(program, gl.Str("viewPos\x00")),
	}

	gl.UseProgram(program)
	gl.Uniform4f(renderer.clipPlaneUniform, 0.0, 1.0, 0.0, 0.0)
	gl.Uniform1f(renderer.clipEnabledUniform, 0.0)

	return renderer
}

// Use активирует программу свечения в OpenGL state.
func (r *GlowRenderer) Use() {
	gl.UseProgram(r.program)
}

// NewMeshWithNormals создает mesh для glow-рендера.
// Нормали используются для rim-подсветки в фрагментном шейдере.
func (r *GlowRenderer) NewMeshWithNormals(vertices, normals []float32) Mesh {
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

func (r *GlowRenderer) SetMVP(mvp mgl32.Mat4) {
	gl.UniformMatrix4fv(r.mvpUniform, 1, false, &mvp[0])
}

func (r *GlowRenderer) SetModel(model mgl32.Mat4) {
	gl.UniformMatrix4fv(r.modelUniform, 1, false, &model[0])
}

func (r *GlowRenderer) SetClipPlane(plane mgl32.Vec4) {
	gl.Uniform4f(r.clipPlaneUniform, plane.X(), plane.Y(), plane.Z(), plane.W())
}

// SetClipPlaneEnabled включает отсечение геометрии (например, в reflection pass).
func (r *GlowRenderer) SetClipPlaneEnabled(enabled bool) {
	if enabled {
		gl.Uniform1f(r.clipEnabledUniform, 1.0)
		return
	}
	gl.Uniform1f(r.clipEnabledUniform, 0.0)
}

func (r *GlowRenderer) SetGlowColor(color mgl32.Vec3) {
	gl.Uniform3f(r.glowColorUniform, color.X(), color.Y(), color.Z())
}

func (r *GlowRenderer) SetGlowIntensity(intensity float32) {
	gl.Uniform1f(r.glowIntensityUniform, intensity)
}

func (r *GlowRenderer) SetPulseSpeed(speed float32) {
	gl.Uniform1f(r.pulseSpeedUniform, speed)
}

func (r *GlowRenderer) SetTime(time float32) {
	gl.Uniform1f(r.timeUniform, time)
}

func (r *GlowRenderer) SetViewPos(pos mgl32.Vec3) {
	gl.Uniform3f(r.viewPosUniform, pos.X(), pos.Y(), pos.Z())
}

func (r *GlowRenderer) Draw(mesh Mesh) {
	gl.BindVertexArray(mesh.VAO)
	gl.DrawArrays(gl.TRIANGLES, 0, mesh.VertexCount)
}
