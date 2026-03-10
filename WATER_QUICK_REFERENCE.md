# Ocean Water System - Quick Reference

## Quick Start

### 1. Create Water Mesh (in Init)
```go
verts, norms := objects.WaterPlaneVertices(100.0, 100.0)
waterMesh := ctx.WaterRenderer.NewMeshWithNormals(verts, norms)
```

### 2. Render Water (in Render)
```go
ctx.WaterRenderer.Use()

// Matrices
waterModel := mgl32.Translate3D(0, waterY, 0)
waterMVP := ctx.Projection.Mul4(view).Mul4(waterModel)
ctx.WaterRenderer.SetMVP(waterMVP)
ctx.WaterRenderer.SetModel(waterModel)
ctx.WaterRenderer.SetView(view)
ctx.WaterRenderer.SetProjection(ctx.Projection)
ctx.WaterRenderer.SetViewPos(ctx.Camera.Position)
ctx.WaterRenderer.SetTime(ctx.Time)

// Colors
ctx.WaterRenderer.SetWaterColor(mgl32.Vec3{0.1, 0.4, 0.7})
ctx.WaterRenderer.SetDepthColor(mgl32.Vec3{0.0, 0.1, 0.3})

// Waves
ctx.WaterRenderer.SetWaveParams(1.0, 8.0, 1.0)
ctx.WaterRenderer.SetFresnelStrength(0.8)

// Light
ctx.WaterRenderer.SetLight(lightDir, lightColor)

// Fog
isUnderwater := ctx.Camera.Position.Y() < waterY
ctx.WaterRenderer.SetUnderwater(isUnderwater)
ctx.WaterRenderer.SetFogDensity(0.15)

// Draw
ctx.WaterRenderer.Draw(waterMesh)

ctx.Renderer.Use() // Switch back
```

## Parameter Presets

### Calm Tropical Ocean
```go
SetWaterColor(Vec3{0.1, 0.5, 0.8})    // Bright turquoise
SetDepthColor(Vec3{0.0, 0.15, 0.4})   // Deep blue
SetWaveParams(0.6, 10.0, 0.7)         // Small gentle waves
SetFresnelStrength(0.9)                // Clear water
SetFogDensity(0.10)                    // Good visibility
```

### Stormy Ocean
```go
SetWaterColor(Vec3{0.2, 0.3, 0.4})    // Gray-blue
SetDepthColor(Vec3{0.0, 0.05, 0.1})   // Very dark
SetWaveParams(2.0, 6.0, 1.5)          // Large fast waves
SetFresnelStrength(0.6)                // Rough surface
SetFogDensity(0.20)                    // Poor visibility
```

### Clear Lake
```go
SetWaterColor(Vec3{0.2, 0.6, 0.7})    // Clear blue-green
SetDepthColor(Vec3{0.0, 0.2, 0.3})    // Medium blue
SetWaveParams(0.4, 12.0, 0.5)         // Tiny ripples
SetFresnelStrength(0.95)               // Very clear
SetFogDensity(0.08)                    // Excellent visibility
```

### Murky Swamp
```go
SetWaterColor(Vec3{0.3, 0.4, 0.3})    // Greenish
SetDepthColor(Vec3{0.1, 0.15, 0.1})   // Dark green
SetWaveParams(0.3, 8.0, 0.4)          // Minimal waves
SetFresnelStrength(0.5)                // Murky
SetFogDensity(0.25)                    // Very poor visibility
```

## API Reference

### Setup Methods
| Method | Parameters | Description |
|--------|-----------|-------------|
| `Use()` | - | Activate water shader |
| `NewMeshWithNormals()` | vertices, normals | Create water mesh |

### Transform Methods
| Method | Parameters | Description |
|--------|-----------|-------------|
| `SetMVP()` | mat4 | Model-View-Projection matrix |
| `SetModel()` | mat4 | Model matrix |
| `SetView()` | mat4 | View matrix |
| `SetProjection()` | mat4 | Projection matrix |
| `SetViewPos()` | vec3 | Camera position |

### Appearance Methods
| Method | Parameters | Description |
|--------|-----------|-------------|
| `SetWaterColor()` | vec3 | Shallow water color (RGB) |
| `SetDepthColor()` | vec3 | Deep water color (RGB) |
| `SetFresnelStrength()` | float | Reflection intensity (0-1) |

### Wave Methods
| Method | Parameters | Description |
|--------|-----------|-------------|
| `SetWaveParams()` | amplitude, frequency, speed | Wave configuration |
| `SetTime()` | float | Animation time (seconds) |

### Lighting Methods
| Method | Parameters | Description |
|--------|-----------|-------------|
| `SetLight()` | direction, color | Main light source |

### Fog Methods
| Method | Parameters | Description |
|--------|-----------|-------------|
| `SetUnderwater()` | bool | Enable underwater mode |
| `SetFogDensity()` | float | Fog thickness (0.05-0.3) |

### Rendering Methods
| Method | Parameters | Description |
|--------|-----------|-------------|
| `Draw()` | mesh | Render water mesh |

## Parameter Ranges

| Parameter | Min | Max | Default | Description |
|-----------|-----|-----|---------|-------------|
| Wave Amplitude | 0.3 | 3.0 | 1.0 | Wave height |
| Wave Frequency | 4.0 | 16.0 | 8.0 | Wave spacing |
| Wave Speed | 0.3 | 2.0 | 1.0 | Animation speed |
| Fresnel Strength | 0.0 | 1.0 | 0.8 | Reflection amount |
| Fog Density | 0.05 | 0.3 | 0.15 | Underwater visibility |

## Color Palettes

### Water Colors (Shallow)
- **Tropical**: `Vec3{0.1, 0.5, 0.8}` - Bright turquoise
- **Ocean**: `Vec3{0.1, 0.4, 0.7}` - Standard blue
- **Lake**: `Vec3{0.2, 0.6, 0.7}` - Clear blue-green
- **Arctic**: `Vec3{0.3, 0.5, 0.6}` - Cold blue-gray
- **Swamp**: `Vec3{0.3, 0.4, 0.3}` - Murky green

### Depth Colors (Deep)
- **Tropical**: `Vec3{0.0, 0.15, 0.4}` - Deep blue
- **Ocean**: `Vec3{0.0, 0.1, 0.3}` - Dark blue
- **Lake**: `Vec3{0.0, 0.2, 0.3}` - Medium blue
- **Arctic**: `Vec3{0.1, 0.2, 0.3}` - Dark gray-blue
- **Swamp**: `Vec3{0.1, 0.15, 0.1}` - Dark green

## Common Issues

### Water looks flat
```go
// Increase wave amplitude
SetWaveParams(1.5, 8.0, 1.0)  // Higher amplitude
```

### Water too choppy
```go
// Decrease amplitude, increase frequency
SetWaveParams(0.6, 12.0, 0.8)  // Smoother waves
```

### Can't see underwater
```go
// Reduce fog density
SetFogDensity(0.08)  // Better visibility
```

### Water too transparent
```go
// Reduce Fresnel strength
SetFresnelStrength(0.5)  // More opaque
```

### No reflections
```go
// Increase Fresnel strength
SetFresnelStrength(0.9)  // More reflective
```

### Waves too fast/slow
```go
// Adjust speed parameter
SetWaveParams(1.0, 8.0, 0.5)  // Slower
SetWaveParams(1.0, 8.0, 1.5)  // Faster
```

## Performance Tips

### High Performance (60+ FPS)
```go
// Use default settings
WaterPlaneVertices(100.0, 100.0)  // 128x128 grid
SetWaveParams(1.0, 8.0, 1.0)
```

### Medium Performance (30-60 FPS)
```go
// Reduce mesh resolution in water.go
gridSize := 64  // Instead of 128
```

### Low Performance (<30 FPS)
```go
// Further reduce resolution
gridSize := 32
// Simplify waves (modify shader to use 2 waves instead of 4)
```

## Integration Examples

### With Existing Lighting
```go
if len(lightManager.GetLights()) > 0 {
    light := lightManager.GetLights()[0]
    ctx.WaterRenderer.SetLight(light.Direction, light.Color)
}
```

### With Camera System
```go
isUnderwater := ctx.Camera.Position.Y() < waterSurfaceY
ctx.WaterRenderer.SetUnderwater(isUnderwater)
```

### With Time System
```go
ctx.WaterRenderer.SetTime(ctx.Time)
```

## Debugging

### Check if water is rendering
```go
// Add before Draw()
fmt.Printf("Drawing water at Y=%.2f, camera Y=%.2f\n", waterY, ctx.Camera.Position.Y())
```

### Verify shader is active
```go
ctx.WaterRenderer.Use()
// Check OpenGL errors
var err uint32
if err = gl.GetError(); err != gl.NO_ERROR {
    fmt.Printf("OpenGL error: %d\n", err)
}
```

### Test without transparency
```go
// Temporarily disable blending to see if it's a transparency issue
// Comment out gl.Enable(gl.BLEND) in water_renderer.go Draw()
```

## Files Reference

- **Water Renderer**: `engine/water_renderer.go`
- **Water Mesh Helper**: `engine/water_mesh.go`
- **Water Geometry**: `objects/water.go`
- **Engine Context**: `engine/app.go`
- **Example Usage**: `examples/subnautica/main.go`
- **Full Documentation**: `WATER_SYSTEM.md`
- **Implementation Details**: `IMPLEMENTATION_SUMMARY.md`
