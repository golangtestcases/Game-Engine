# Ocean Water System - Architecture Diagram

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Game Engine                              │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Window     │  │   Camera     │  │   Renderer   │          │
│  │   System     │  │   System     │  │   (Main)     │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              WaterRenderer (NEW)                         │   │
│  │  ┌────────────────────────────────────────────────────┐  │   │
│  │  │  Vertex Shader                                     │  │   │
│  │  │  • 4 Gerstner Waves                               │  │   │
│  │  │  • Wave Normal Calculation                        │  │   │
│  │  │  • Position Transformation                        │  │   │
│  │  └────────────────────────────────────────────────────┘  │   │
│  │  ┌────────────────────────────────────────────────────┐  │   │
│  │  │  Fragment Shader                                   │  │   │
│  │  │  • Animated Normal Maps                           │  │   │
│  │  │  • Fresnel Reflection                             │  │   │
│  │  │  • Depth-Based Coloring                           │  │   │
│  │  │  • Lighting (Diffuse + Specular)                  │  │   │
│  │  │  • Underwater Fog                                 │  │   │
│  │  │  • Alpha Blending                                 │  │   │
│  │  └────────────────────────────────────────────────────┘  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Lighting   │  │   Shadow     │  │   Audio      │          │
│  │   System     │  │   System     │  │   System     │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
```

## Rendering Pipeline

```
Game Loop
    │
    ├─> Update Phase
    │   ├─> Camera Movement
    │   ├─> Physics
    │   └─> Game Logic
    │
    └─> Render Phase
        │
        ├─> Shadow Pass
        │   ├─> Render Ground (shadow)
        │   ├─> Render Water (shadow) ◄── Uses simple waves
        │   └─> Render Objects (shadow)
        │
        └─> Geometry Pass
            ├─> Clear Buffers
            │
            ├─> Render Opaque Objects
            │   ├─> Ground (with shadows)
            │   └─> Plants (with shadows)
            │
            ├─> Render Water ◄────────────── WaterRenderer
            │   │
            │   ├─> Enable Blending
            │   ├─> Use Water Shader
            │   ├─> Set Uniforms
            │   │   ├─> Matrices (MVP, Model, View, Projection)
            │   │   ├─> Camera Position
            │   │   ├─> Time
            │   │   ├─> Colors (Surface, Deep)
            │   │   ├─> Wave Parameters
            │   │   ├─> Fresnel Strength
            │   │   ├─> Light Direction & Color
            │   │   ├─> Fog Density
            │   │   └─> Underwater Flag
            │   ├─> Draw Water Mesh
            │   └─> Disable Blending
            │
            └─> Swap Buffers
```

## Data Flow

```
┌─────────────┐
│   Game      │
│   Init()    │
└──────┬──────┘
       │
       ├─> Create Water Mesh
       │   └─> objects.WaterPlaneVertices(sizeX, sizeZ)
       │       └─> Returns: vertices[], normals[]
       │
       └─> Store Mesh
           └─> ctx.WaterRenderer.NewMeshWithNormals(verts, norms)
               └─> Creates: VAO, VBO, NBO

┌─────────────┐
│   Game      │
│   Render()  │
└──────┬──────┘
       │
       ├─> Activate Water Shader
       │   └─> ctx.WaterRenderer.Use()
       │
       ├─> Set Transformation Matrices
       │   ├─> SetMVP(mvp)
       │   ├─> SetModel(model)
       │   ├─> SetView(view)
       │   └─> SetProjection(projection)
       │
       ├─> Set Camera & Time
       │   ├─> SetViewPos(cameraPos)
       │   └─> SetTime(currentTime)
       │
       ├─> Configure Appearance
       │   ├─> SetWaterColor(surfaceColor)
       │   ├─> SetDepthColor(deepColor)
       │   ├─> SetWaveParams(amp, freq, speed)
       │   └─> SetFresnelStrength(strength)
       │
       ├─> Set Lighting
       │   └─> SetLight(direction, color)
       │
       ├─> Configure Fog
       │   ├─> SetUnderwater(isUnderwater)
       │   └─> SetFogDensity(density)
       │
       ├─> Draw Water
       │   └─> Draw(waterMesh)
       │       ├─> Enable GL_BLEND
       │       ├─> Bind VAO
       │       ├─> DrawArrays(TRIANGLES)
       │       └─> Disable GL_BLEND
       │
       └─> Restore Main Renderer
           └─> ctx.Renderer.Use()
```

## Shader Pipeline

```
Vertex Shader Input
    │
    ├─> Vertex Position (vp)
    └─> Vertex Normal (vn)
    │
    ├─> Apply Gerstner Wave 1 ──┐
    ├─> Apply Gerstner Wave 2 ──┤
    ├─> Apply Gerstner Wave 3 ──┼─> Sum Displacements
    └─> Apply Gerstner Wave 4 ──┘
    │
    ├─> Calculate Wave Normal 1 ──┐
    ├─> Calculate Wave Normal 2 ──┤
    ├─> Calculate Wave Normal 3 ──┼─> Sum & Normalize
    └─> Calculate Wave Normal 4 ──┘
    │
    ├─> Transform to World Space
    ├─> Transform to Clip Space
    │
    └─> Output to Fragment Shader
        ├─> fragPos (world position)
        ├─> normal (transformed normal)
        └─> waveHeight (Y displacement)

Fragment Shader Input
    │
    ├─> fragPos
    ├─> normal
    └─> waveHeight
    │
    ├─> Normalize Normal
    │
    ├─> Add Animated Detail Normal
    │   ├─> Generate UV1 (scrolling)
    │   ├─> Generate UV2 (scrolling)
    │   ├─> Compute Procedural Noise
    │   └─> Perturb Normal
    │
    ├─> Calculate Fresnel
    │   └─> pow(1 - dot(view, normal), 5)
    │
    ├─> Interpolate Color by Depth
    │   └─> mix(waterColor, depthColor, depth)
    │
    ├─> Calculate Lighting
    │   ├─> Diffuse (Lambertian)
    │   │   └─> dot(normal, lightDir)
    │   │
    │   └─> Specular (Blinn-Phong)
    │       └─> pow(dot(normal, halfDir), 128)
    │
    ├─> Add Sky Reflection
    │   └─> skyColor * fresnelFactor
    │
    ├─> Apply Underwater Fog (if underwater)
    │   └─> mix(color, fogColor, fogFactor)
    │
    ├─> Calculate Alpha
    │   └─> 0.85 + fresnelFactor * 0.15
    │
    └─> Output Final Color
        └─> vec4(finalColor, alpha)
```

## Component Interaction

```
┌──────────────────────────────────────────────────────────────┐
│                        Context                                │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────────┐ │
│  │  Window    │  │  Camera    │  │  WaterRenderer         │ │
│  │            │  │            │  │                        │ │
│  │  • Size    │  │  • Pos     │  │  • Shader Program     │ │
│  │  • Input   │  │  • Dir     │  │  • Uniforms           │ │
│  └────────────┘  │  • Speed   │  │  • Draw Method        │ │
│                  └────────────┘  └────────────────────────┘ │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────────┐ │
│  │ Renderer   │  │ Projection │  │  Time & DeltaTime      │ │
│  │ (Main)     │  │  Matrix    │  │                        │ │
│  └────────────┘  └────────────┘  └────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
                            │
                            │ Passed to
                            ▼
┌──────────────────────────────────────────────────────────────┐
│                         Game                                  │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Init(ctx)                                             │  │
│  │    • Create water mesh using ctx.WaterRenderer        │  │
│  │    • Store mesh reference                             │  │
│  └────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Update(ctx)                                           │  │
│  │    • Update camera position                           │  │
│  │    • Check if underwater                              │  │
│  └────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Render(ctx)                                           │  │
│  │    • Render ground & objects                          │  │
│  │    • Configure & render water using ctx.WaterRenderer │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## File Structure

```
game-engine-golang-test/
│
├── engine/
│   ├── app.go                  ◄── Modified (added WaterRenderer to Context)
│   ├── water_renderer.go       ◄── NEW (main water system)
│   ├── water_mesh.go           ◄── NEW (mesh helper)
│   ├── opengl.go               (existing renderer)
│   ├── camera.go               (camera system)
│   ├── light.go                (lighting system)
│   └── shadow.go               (shadow system)
│
├── objects/
│   ├── water.go                ◄── Modified (increased resolution)
│   ├── ground.go               (terrain generation)
│   └── plant.go                (vegetation)
│
├── examples/
│   └── subnautica/
│       └── main.go             ◄── Modified (uses WaterRenderer)
│
├── WATER_SYSTEM.md             ◄── NEW (full documentation)
├── IMPLEMENTATION_SUMMARY.md   ◄── NEW (technical details)
├── WATER_QUICK_REFERENCE.md    ◄── NEW (quick guide)
├── SHADER_EXPLANATION.md       ◄── NEW (shader breakdown)
├── README_WATER.md             ◄── NEW (overview)
└── WATER_ARCHITECTURE.md       ◄── NEW (this file)
```

## Memory Layout

```
Water Mesh (128x128 grid)
├── Vertices: 16,384 quads × 2 triangles × 3 vertices × 3 floats
│   = 294,912 floats = 1,179,648 bytes (~1.1 MB)
│
└── Normals: 16,384 quads × 2 triangles × 3 vertices × 3 floats
    = 294,912 floats = 1,179,648 bytes (~1.1 MB)

Total: ~2.2 MB per water mesh

GPU Memory:
├── VAO: 1 × 4 bytes = 4 bytes
├── VBO (vertices): 1,179,648 bytes
├── NBO (normals): 1,179,648 bytes
└── Shader Program: ~10 KB

Total GPU Memory: ~2.4 MB per water surface
```

## Performance Profile

```
Per Frame (60 FPS):
│
├── Vertex Processing
│   ├── Vertices: 98,304 (32,768 triangles × 3)
│   ├── Gerstner Waves: 4 per vertex
│   ├── Normal Calculations: 4 per vertex
│   └── Transformations: 2 per vertex (world + clip)
│   Total: ~1.2M operations
│
├── Fragment Processing
│   ├── Fragments: ~500K (1920×1080 screen, ~50% water coverage)
│   ├── Normal Animation: 2 UV layers per fragment
│   ├── Fresnel: 1 calculation per fragment
│   ├── Lighting: 2 calculations per fragment (diffuse + specular)
│   └── Fog: 1 calculation per fragment (if underwater)
│   Total: ~3M operations
│
└── Memory Bandwidth
    ├── Vertex Data Read: 2.4 MB
    ├── Uniform Updates: ~1 KB
    └── Framebuffer Write: ~8 MB (RGBA, 1920×1080)
    Total: ~10.4 MB per frame = 624 MB/s @ 60 FPS
```

## Optimization Strategies

```
Level 1: High Quality (Desktop)
├── Grid: 128×128
├── Waves: 4 Gerstner waves
├── Normals: Animated detail
└── Target: 60 FPS

Level 2: Medium Quality (Console)
├── Grid: 64×64
├── Waves: 3 Gerstner waves
├── Normals: Static detail
└── Target: 30-60 FPS

Level 3: Low Quality (Mobile)
├── Grid: 32×32
├── Waves: 2 Gerstner waves
├── Normals: None
└── Target: 30 FPS

LOD System (Future):
├── Near: 128×128, 4 waves
├── Medium: 64×64, 3 waves
├── Far: 32×32, 2 waves
└── Very Far: 16×16, 1 wave
```

## Extension Points

```
Current System
    │
    ├─> Add Foam System
    │   └─> Detect wave peaks
    │       └─> Render foam particles
    │
    ├─> Add Reflection Texture
    │   └─> Render scene to texture
    │       └─> Sample in fragment shader
    │
    ├─> Add Refraction Texture
    │   └─> Capture scene color buffer
    │       └─> Distort UVs in fragment shader
    │
    ├─> Add Caustics
    │   └─> Project caustic texture
    │       └─> Animate over time
    │
    ├─> Add Shore Interaction
    │   └─> Detect terrain proximity
    │       └─> Modify wave behavior
    │
    └─> Add Buoyancy Physics
        └─> Calculate water height at position
            └─> Apply upward force to objects
```

## Summary

The water system is:
- **Modular**: Separate from main renderer
- **Efficient**: Optimized shaders and rendering
- **Extensible**: Easy to add new features
- **Well-integrated**: Works with existing systems
- **Documented**: Complete guides and examples

All components work together to create realistic ocean water rendering.
