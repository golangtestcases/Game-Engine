# Water/Environment Architecture (Current)

## High-Level Graph

```text
subnauticaGame.Init
  -> initEnvironmentRuntime
     -> RenderGraph(ShadowPass -> GeometryPass)
     -> OceanSystem

subnauticaGame.Render
  -> executeEnvironmentRenderGraph
     -> set RenderResourceLightManager
     -> ShadowPass.Execute
        -> RenderResourceShadowMap
        -> RenderResourceLightSpaceMatrix
     -> GeometryPass.Execute
        -> renderGeometry
           -> buildEnvironmentLightingFrame
           -> OceanSystem.Render
              -> renderOpaqueScene callback
```

## Layer Boundaries
- `engine/*`: reusable rendering/runtime primitives
- `examples/subnautica/*`: content, orchestration, editor decisions

No Subnautica-specific content logic should move into `engine` unless it becomes generic and reusable.

## Key Files
- `examples/subnautica/environment_runtime.go`
- `examples/subnautica/render.go`
- `engine/ocean_system.go`
- `engine/render_graph.go`
- `engine/render_passes.go`
- `engine/render_resource_keys.go`

## Notes
- The render graph still passes resources by string keys internally; keys are now centralized constants.
- Water behavior is underwater-first and driven through `OceanSystem` and lighting frame state.
- Editor water visibility is a presentation switch inside the same canonical path.

## Removed Architecture
The old `WaterRenderer` branch is removed from runtime.
Any mention of `ctx.WaterRenderer` describes historical code only.
