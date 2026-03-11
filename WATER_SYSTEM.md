# Water System (Canonical Runtime)

This document describes the active water/environment architecture used by `examples/subnautica`.

## Runtime Ownership
- Engine-level rendering primitives:
  - `engine/ocean_system.go`
  - `engine/sky_renderer.go`
  - `engine/underwater_post.go`
  - `engine/underwater_light_shafts.go`
- Game-level orchestration:
  - `examples/subnautica/environment_runtime.go`
  - `examples/subnautica/render.go`

`examples/subnautica` owns content decisions (terrain, creatures, editor water visibility toggle).
`engine` owns reusable rendering mechanisms.

## Startup Flow
1. `initEnvironmentRuntime(ctx)` in `examples/subnautica/environment_runtime.go`:
- initializes scene lighting (`g.initSceneLighting()`)
- creates shadow resources (`ShadowMap`, `ShadowShader`)
- builds render graph with:
  - `ShadowPass`
  - `GeometryPass`
- creates `OceanSystem` with current window size
- syncs water level and sky atmosphere into ocean runtime

## Per-Frame Flow
1. `syncEnvironmentRuntimeState()` updates water level + sun-driven sky state.
2. `executeEnvironmentRenderGraph(ctx)` sets render-graph resource:
- `engine.RenderResourceLightManager`
3. `RenderGraph.Execute(...)` runs:
- `ShadowPass` -> publishes:
  - `engine.RenderResourceShadowMap`
  - `engine.RenderResourceLightSpaceMatrix`
- `GeometryPass` -> calls `renderGeometry(...)`
4. `renderGeometry(...)`:
- builds `LightingFrame` with fog/underwater/caustics/shadow state
- applies frame to renderer
- calls `g.oceanSystem.Render(...)` and passes callback for opaque scene
5. `renderOpaqueScene(...)` draws terrain, editor overlays, creatures, glow plants.

## Editor Interaction
Editor water toggle (`V`) modifies only visibility/presentation:
- `g.oceanSystem.SetSurfaceVisible(editorWaterVisible)`
- underwater blend/caustics are suppressed in editor when water is hidden
- no alternate render pipeline is created

## Render-Graph Resource Keys
Keys are centralized in `engine/render_resource_keys.go`:
- `RenderResourceLightManager`
- `RenderResourceShadowMap`
- `RenderResourceLightSpaceMatrix`
- `RenderResourceSceneColor`
- `RenderResourceFinalColor`

## Removed/Obsolete Path
The following path is removed and should not be used in new docs/code:
- `ctx.WaterRenderer` API
- `engine/water_renderer.go`
- `engine/water_mesh.go`
- separate parallel water-render integration in game code
