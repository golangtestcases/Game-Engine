# Water Runtime Changelog

## 2026-03-11 (Post-Consolidation)
- Confirmed single canonical environment path in `examples/subnautica`:
  - render graph (`ShadowPass` -> `GeometryPass`)
  - `OceanSystem` for sky/ocean/underwater rendering
- Removed documentation references that implied active `WaterRenderer` usage.
- Standardized render-graph resource keys via constants in `engine/render_resource_keys.go`.

## Historical Note
Earlier revisions used a separate `WaterRenderer` branch (`ctx.WaterRenderer`).
That branch is not part of the current runtime and remains historical only.

For current details, use:
- `WATER_SYSTEM.md`
- `WATER_ARCHITECTURE.md`
- `WATER_QUICK_REFERENCE.md`
