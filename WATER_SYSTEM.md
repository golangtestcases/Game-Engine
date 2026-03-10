# Ocean Water System Documentation

## Overview

The ocean water system implements realistic water rendering with Gerstner waves, Fresnel reflection, depth-based coloring, and underwater fog effects.

## Architecture

### Components

1. **WaterRenderer** (`engine/water_renderer.go`)
   - Dedicated shader system for water rendering
   - Handles all water-specific rendering logic
   - Separate from main renderer for performance

2. **Water Mesh** (`objects/water.go`)
   - High-resolution grid (128x128) for smooth wave deformation
   - Generated with normals for lighting calculations

3. **Integration** (`engine/app.go`)
   - WaterRenderer added to engine Context
   - Available to all games using the engine

## Features Implemented

### 1. Gerstner Waves (Vertex Shader)

Multiple overlapping Gerstner waves create realistic ocean motion:

```glsl
// 4 waves with different directions, frequencies, and amplitudes
wave1: direction(1.0, 0.0), amplitude=0.15, frequency=8.0
wave2: direction(0.7, 0.7), amplitude=0.12, frequency=6.0
wave3: direction(0.0, 1.0), amplitude=0.10, frequency=10.0
wave4: direction(-0.5, 0.8), amplitude=0.08, frequency=12.0
```

**Benefits:**
- Physically-based wave simulation
- Natural-looking crests and troughs
- Proper wave normals for lighting

### 2. Animated Normal Maps (Fragment Shader)

Small-scale wave detail added via procedural normal animation:

```glsl
vec2 uv1 = uv * 2.0 + vec2(time * 0.05, time * 0.03);
vec2 uv2 = uv * 3.0 - vec2(time * 0.04, time * 0.06);
```

**Benefits:**
- Fine surface detail without additional textures
- Animated ripples and small waves
- Enhances realism at close range

### 3. Depth-Based Water Color

Water color transitions from shallow to deep:

```go
surfaceColor := Vec3{0.1, 0.4, 0.7}  // Bright blue
deepColor := Vec3{0.0, 0.1, 0.3}     // Dark blue
```

**Benefits:**
- Realistic depth perception
- Visual feedback for water depth
- Enhances underwater atmosphere

### 4. Fresnel Reflection

View-angle dependent reflection using Fresnel equation:

```glsl
float fresnel = pow(1.0 - max(dot(viewDir, normal), 0.0), 5.0)
```

**Benefits:**
- Realistic water surface appearance
- More reflective at grazing angles
- Less reflective when looking straight down

### 5. Water Refraction (Simulated)

Transparency with depth-based alpha blending:

```glsl
float alpha = 0.85 + fresnelFactor * 0.15;
```

**Benefits:**
- See-through water surface
- Depth perception
- Realistic water transparency

### 6. Specular Highlights

Sun reflection on water surface:

```glsl
float spec = pow(max(dot(normal, halfDir), 0.0), 128.0);
vec3 specular = spec * lightColor * 1.5;
```

**Benefits:**
- Sparkling water surface
- Dynamic sun reflections
- Enhanced realism

### 7. Underwater Fog

Exponential fog when camera is below water surface:

```glsl
float fogFactor = 1.0 - exp(-fogDensity * dist);
finalColor = mix(finalColor, depthColor, fogFactor);
```

**Benefits:**
- Limited underwater visibility
- Atmospheric depth
- Performance optimization (distant objects fade)

## Usage

### Basic Setup

```go
// In game Init()
waterVerts, waterNorms := objects.WaterPlaneVertices(100.0, 100.0)
waterMesh := ctx.WaterRenderer.NewMeshWithNormals(waterVerts, waterNorms)
```

### Rendering

```go
// In game Render()
ctx.WaterRenderer.Use()
ctx.WaterRenderer.SetMVP(mvp)
ctx.WaterRenderer.SetModel(model)
ctx.WaterRenderer.SetView(view)
ctx.WaterRenderer.SetProjection(projection)
ctx.WaterRenderer.SetViewPos(cameraPos)
ctx.WaterRenderer.SetTime(time)

// Colors
ctx.WaterRenderer.SetWaterColor(mgl32.Vec3{0.1, 0.4, 0.7})
ctx.WaterRenderer.SetDepthColor(mgl32.Vec3{0.0, 0.1, 0.3})

// Wave parameters (amplitude, frequency, speed)
ctx.WaterRenderer.SetWaveParams(1.0, 8.0, 1.0)
ctx.WaterRenderer.SetFresnelStrength(0.8)

// Lighting
ctx.WaterRenderer.SetLight(lightDir, lightColor)

// Underwater fog
isUnderwater := cameraY < waterY
ctx.WaterRenderer.SetUnderwater(isUnderwater)
ctx.WaterRenderer.SetFogDensity(0.15)

ctx.WaterRenderer.Draw(waterMesh)
```

## Parameters

### Wave Parameters

- **Amplitude** (0.5 - 2.0): Wave height multiplier
- **Frequency** (4.0 - 16.0): Wave spacing
- **Speed** (0.5 - 2.0): Animation speed

### Visual Parameters

- **Fresnel Strength** (0.0 - 1.0): Reflection intensity
- **Fog Density** (0.05 - 0.3): Underwater visibility
- **Water Color**: Shallow water color (RGB)
- **Depth Color**: Deep water color (RGB)

## Performance Considerations

1. **Mesh Resolution**: 128x128 grid = 32,768 triangles
   - Adjust in `objects/water.go` if needed
   - Lower for mobile/low-end hardware

2. **Blend Mode**: Alpha blending enabled during water rendering
   - Render water after opaque geometry
   - Disable depth writing for proper transparency

3. **Shader Complexity**: 
   - 4 Gerstner waves per vertex
   - Procedural normal generation
   - Multiple lighting calculations
   - Consider LOD system for large scenes

## Technical Details

### Vertex Shader Flow

1. Read vertex position and normal
2. Calculate 4 Gerstner wave offsets
3. Sum wave contributions to position
4. Calculate wave normals for lighting
5. Transform to clip space

### Fragment Shader Flow

1. Normalize interpolated normal
2. Add animated detail normals
3. Calculate Fresnel reflection
4. Compute depth-based color
5. Apply lighting (diffuse + specular)
6. Add sky reflection
7. Apply underwater fog if needed
8. Output final color with alpha

## Integration with Existing Systems

### Shadow Mapping

Water casts shadows using existing shadow system:

```go
// In renderShadows()
shader.SetWaterWaves(true)
shader.SetModel(waterTransform)
ctx.Renderer.Draw(waterMesh)
```

### Lighting System

Water responds to all lights in scene:

```go
if len(lightManager.GetLights()) > 0 {
    mainLight := lightManager.GetLights()[0]
    waterRenderer.SetLight(mainLight.Direction, mainLight.Color)
}
```

### Camera System

Underwater detection:

```go
isUnderwater := camera.Position.Y() < waterSurfaceY
waterRenderer.SetUnderwater(isUnderwater)
```

## Future Enhancements

1. **Foam/Whitecaps**: Add foam at wave peaks
2. **Caustics**: Underwater light patterns
3. **Reflection Texture**: Real-time reflection rendering
4. **Refraction Texture**: Scene color buffer for refraction
5. **Shore Interaction**: Waves breaking on beaches
6. **Particle Effects**: Splashes and spray
7. **Buoyancy System**: Physics for floating objects

## Troubleshooting

### Water appears flat
- Check wave amplitude parameter
- Verify time uniform is updating
- Ensure mesh has sufficient resolution

### Transparency issues
- Render water after opaque objects
- Enable blending: `gl.Enable(gl.BLEND)`
- Set blend func: `gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)`

### Performance problems
- Reduce mesh resolution in `water.go`
- Decrease number of Gerstner waves
- Simplify fragment shader calculations
- Use LOD system for distant water

### Lighting looks wrong
- Verify light direction is normalized
- Check normal calculations
- Ensure view position is correct
- Validate material properties

## Example: Subnautica-Style Ocean

```go
// Tropical ocean colors
surfaceColor := mgl32.Vec3{0.1, 0.5, 0.8}  // Bright turquoise
deepColor := mgl32.Vec3{0.0, 0.15, 0.4}    // Deep blue

// Calm tropical waves
waterRenderer.SetWaveParams(0.8, 10.0, 0.7)

// Strong Fresnel for clear water
waterRenderer.SetFresnelStrength(0.9)

// Moderate underwater fog
waterRenderer.SetFogDensity(0.12)
```

## Shader Uniforms Reference

### Vertex Shader
- `MVP`: Model-View-Projection matrix
- `model`: Model matrix
- `view`: View matrix
- `projection`: Projection matrix
- `time`: Current time in seconds
- `waveAmplitude`: Wave height multiplier
- `waveFrequency`: Wave spacing
- `waveSpeed`: Animation speed

### Fragment Shader
- `viewPos`: Camera position
- `waterColor`: Shallow water color
- `depthColor`: Deep water color
- `fresnelStrength`: Reflection intensity
- `lightDir`: Main light direction
- `lightColor`: Main light color
- `fogDensity`: Underwater fog density
- `underwater`: Boolean (1.0 or 0.0)

## Credits

Based on:
- Gerstner waves (GPU Gems)
- Fresnel-Schlick approximation
- Exponential fog model
- Physically-based rendering principles
