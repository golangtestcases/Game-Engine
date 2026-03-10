package engine

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// ShadowPass строит карту теней directional-источника.
type ShadowPass struct {
	shadowMap          *ShadowMap
	shadowShader       *ShadowShader
	renderFunc         func(shader *ShadowShader, ctx *Context)
	shadowCenter       mgl32.Vec3
	shadowRadius       float32
	cameraConfig       ShadowCameraConfig
	followCamera       bool
	cameraFollowOffset mgl32.Vec3
}

func NewShadowPass(shadowMap *ShadowMap, shadowShader *ShadowShader, renderFunc func(*ShadowShader, *Context)) *ShadowPass {
	return &ShadowPass{
		shadowMap:          shadowMap,
		shadowShader:       shadowShader,
		renderFunc:         renderFunc,
		shadowCenter:       mgl32.Vec3{0, -3, 0},
		shadowRadius:       80,
		cameraConfig:       DefaultShadowCameraConfig(),
		followCamera:       true,
		cameraFollowOffset: mgl32.Vec3{0, -3, 0},
	}
}

func (p *ShadowPass) SetShadowMap(shadowMap *ShadowMap) {
	if shadowMap == nil {
		return
	}
	p.shadowMap = shadowMap
}

// SetSceneBounds задает размер области, которую покрывает directional shadow map.
func (p *ShadowPass) SetSceneBounds(center mgl32.Vec3, radius float32) {
	if radius < 1 {
		radius = 1
	}
	p.shadowCenter = center
	p.shadowRadius = radius
}

// SetFollowCamera включает режим, при котором shadow-объём следует за камерой.
func (p *ShadowPass) SetFollowCamera(enabled bool) {
	p.followCamera = enabled
}

func (p *ShadowPass) SetFollowOffset(offset mgl32.Vec3) {
	p.cameraFollowOffset = offset
}

func (p *ShadowPass) SetCameraConfig(config ShadowCameraConfig) {
	p.cameraConfig = sanitizeShadowCameraConfig(config)
}

func (p *ShadowPass) Name() string         { return "ShadowPass" }
func (p *ShadowPass) Type() RenderPassType { return PassShadow }
func (p *ShadowPass) Inputs() []string     { return []string{"lightManager"} }
func (p *ShadowPass) Outputs() []string    { return []string{"shadowMap", "lightSpaceMatrix"} }

func (p *ShadowPass) Execute(ctx *RenderContext) error {
	if p.shadowMap == nil || p.shadowShader == nil {
		return nil
	}

	lightMgr, ok := ctx.GetResource("lightManager")
	if !ok {
		return nil
	}

	shadowCenter := p.shadowCenter
	if p.followCamera && ctx.Context != nil && ctx.Context.Camera != nil {
		shadowCenter = ctx.Context.Camera.Position.Add(p.cameraFollowOffset)
	}

	lm := lightMgr.(*LightManager)
	lightingState := lm.LightingState()
	if lightingState.ShadowLightIndex >= 0 && lightingState.ShadowLightIndex < len(lightingState.Lights) {
		dirLight := lightingState.Lights[lightingState.ShadowLightIndex]
		if dirLight.Type == LightDirectional {
			p.shadowMap.CalculateDirectionalLightSpaceMatrix(
				dirLight.Direction,
				shadowCenter,
				p.shadowRadius,
				p.cameraConfig,
			)
		}
	}

	p.shadowMap.Bind()
	gl.Enable(gl.POLYGON_OFFSET_FILL)
	gl.PolygonOffset(p.cameraConfig.DepthBiasSlope, p.cameraConfig.DepthBiasConstant)
	p.shadowShader.Use()
	p.shadowShader.SetLightSpaceMatrix(p.shadowMap.LightSpaceMatrix)
	p.shadowShader.SetTime(ctx.Context.Time)

	if p.renderFunc != nil {
		p.renderFunc(p.shadowShader, ctx.Context)
	}

	gl.Disable(gl.POLYGON_OFFSET_FILL)

	w, h := ctx.Context.Window.GetSize()
	p.shadowMap.Unbind(int32(w), int32(h))

	ctx.SetResource("shadowMap", p.shadowMap)
	ctx.SetResource("lightSpaceMatrix", p.shadowMap.LightSpaceMatrix)
	return nil
}

// GeometryPass рисует основную геометрию сцены.
type GeometryPass struct {
	clearColor [4]float32
	renderFunc func(ctx *Context, shadowMap *ShadowMap, lightSpace mgl32.Mat4)
}

func NewGeometryPass(clearColor [4]float32, renderFunc func(*Context, *ShadowMap, mgl32.Mat4)) *GeometryPass {
	return &GeometryPass{
		clearColor: clearColor,
		renderFunc: renderFunc,
	}
}

func (p *GeometryPass) Name() string         { return "GeometryPass" }
func (p *GeometryPass) Type() RenderPassType { return PassGeometry }
func (p *GeometryPass) Inputs() []string {
	return []string{"shadowMap", "lightSpaceMatrix", "lightManager"}
}
func (p *GeometryPass) Outputs() []string { return []string{"sceneColor"} }

func (p *GeometryPass) Execute(ctx *RenderContext) error {
	gl.ClearColor(p.clearColor[0], p.clearColor[1], p.clearColor[2], p.clearColor[3])
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	var shadowMap *ShadowMap
	var lightSpace mgl32.Mat4

	if sm, ok := ctx.GetResource("shadowMap"); ok {
		shadowMap = sm.(*ShadowMap)
	}
	if ls, ok := ctx.GetResource("lightSpaceMatrix"); ok {
		lightSpace = ls.(mgl32.Mat4)
	}

	if p.renderFunc != nil {
		p.renderFunc(ctx.Context, shadowMap, lightSpace)
	}

	ctx.SetResource("sceneColor", true)
	return nil
}

// PostProcessPass представляет один этап пост-обработки.
type PostProcessPass struct {
	name       string
	renderFunc func(ctx *Context)
}

func NewPostProcessPass(name string, renderFunc func(*Context)) *PostProcessPass {
	return &PostProcessPass{
		name:       name,
		renderFunc: renderFunc,
	}
}

func (p *PostProcessPass) Name() string         { return p.name }
func (p *PostProcessPass) Type() RenderPassType { return PassPostProcess }
func (p *PostProcessPass) Inputs() []string     { return []string{"sceneColor"} }
func (p *PostProcessPass) Outputs() []string    { return []string{"finalColor"} }

func (p *PostProcessPass) Execute(ctx *RenderContext) error {
	if p.renderFunc != nil {
		p.renderFunc(ctx.Context)
	}
	ctx.SetResource("finalColor", true)
	return nil
}

// UIPass рисует поверх финального изображения HUD/оверлеи.
type UIPass struct {
	renderFunc func(ctx *Context)
}

func NewUIPass(renderFunc func(*Context)) *UIPass {
	return &UIPass{renderFunc: renderFunc}
}

func (p *UIPass) Name() string         { return "UIPass" }
func (p *UIPass) Type() RenderPassType { return PassUI }
func (p *UIPass) Inputs() []string     { return []string{"finalColor"} }
func (p *UIPass) Outputs() []string    { return []string{} }

func (p *UIPass) Execute(ctx *RenderContext) error {
	if p.renderFunc != nil {
		p.renderFunc(ctx.Context)
	}
	return nil
}
