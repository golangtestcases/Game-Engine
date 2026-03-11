# Water And Environment Runtime (Current)

This repository no longer uses the old `WaterRenderer` integration path.

Current canonical runtime path in `examples/subnautica`:
1. `initEnvironmentRuntime` builds render graph + ocean system.
2. `executeEnvironmentRenderGraph` injects `lightManager` into render-graph resources.
3. `ShadowPass` builds `shadowMap` + `lightSpaceMatrix`.
4. `GeometryPass` calls `renderGeometry`.
5. `renderGeometry` builds `LightingFrame` and renders through `engine.OceanSystem`.
6. `OceanSystem.Render(...)` calls back into `renderOpaqueScene(...)` for world geometry.

## Source Of Truth Files
- `examples/subnautica/environment_runtime.go`
- `examples/subnautica/render.go`
- `engine/ocean_system.go`
- `engine/render_graph.go`
- `engine/render_passes.go`
- `engine/render_resource_keys.go`

## Deprecated Path (Removed)
The following are not part of the runtime anymore:
- `ctx.WaterRenderer.*`
- `engine/water_renderer.go`
- `engine/water_mesh.go`
- direct example-level parallel water-render pipeline

Use `WATER_SYSTEM.md` for a concrete step-by-step runtime description and `WATER_QUICK_REFERENCE.md` for file-level quick lookup.
