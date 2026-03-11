# Water Runtime Quick Reference (Current)

## Primary Entry Points
- Init: `examples/subnautica/environment_runtime.go:initEnvironmentRuntime`
- Frame execution: `examples/subnautica/environment_runtime.go:executeEnvironmentRenderGraph`
- Geometry callback: `examples/subnautica/render.go:renderGeometry`

## Render Pass Contracts
Defined in `engine/render_passes.go`.

- `ShadowPass`
  - Inputs: `RenderResourceLightManager`
  - Outputs: `RenderResourceShadowMap`, `RenderResourceLightSpaceMatrix`

- `GeometryPass`
  - Inputs: `RenderResourceShadowMap`, `RenderResourceLightSpaceMatrix`, `RenderResourceLightManager`
  - Outputs: `RenderResourceSceneColor`

## Resource Keys
Centralized in `engine/render_resource_keys.go`.

## Ocean Runtime
`engine.OceanSystem` owns:
- reflection/refraction flow
- sky rendering
- underwater post
- underwater light shafts

`examples/subnautica/render.go` provides scene callback into that runtime.

## Editor Note
`V` in editor mode toggles surface visibility only; it does not switch to a separate water path.

## Removed API
Do not use:
- `ctx.WaterRenderer.*`
- `engine/water_renderer.go`
- `engine/water_mesh.go`
