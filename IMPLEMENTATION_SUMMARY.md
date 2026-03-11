# Water/Environment Implementation Summary (Current)

This summary reflects the active implementation in the repository after architecture consolidation.

## Active Runtime
- Water/environment rendering is executed through `engine.OceanSystem`.
- Orchestration is owned by `examples/subnautica/environment_runtime.go`.
- World rendering is executed inside the ocean-system callback in `examples/subnautica/render.go`.

## Render Graph Integration
- Graph construction: `initEnvironmentRuntime(...)`
- Frame execution: `executeEnvironmentRenderGraph(...)`
- Pass chain:
  1. `ShadowPass`
  2. `GeometryPass`

Resource keys are centralized in `engine/render_resource_keys.go`.

## Lighting And Underwater
`buildEnvironmentLightingFrame(...)` composes:
- scene lighting (`LightManager`)
- fog range and fog color
- underwater blend/fog/tint
- caustics intensity
- shadow map/light-space matrix

`renderGeometry(...)` applies the frame to the renderer and then delegates to `OceanSystem.Render(...)`.

## Editor Behavior
Editor keeps the same runtime path.
- `V` toggles water surface visibility in-editor.
- Underwater presentation terms are suppressed when water is hidden.

## Historical Context
The older implementation summary described `ctx.WaterRenderer` and related files.
That path is no longer part of active runtime architecture.
