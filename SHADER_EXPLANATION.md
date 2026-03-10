# Water Shaders - Technical Explanation

## Vertex Shader Breakdown

### Input/Output
```glsl
// Input from mesh
layout (location = 0) in vec3 vp;  // Vertex position
layout (location = 1) in vec3 vn;  // Vertex normal

// Output to fragment shader
out vec3 fragPos;      // World space position
out vec3 normal;       // Transformed normal
out float waveHeight;  // Y displacement from waves
```

### Uniforms
```glsl
uniform mat4 MVP;           // Combined transformation
uniform mat4 model;         // Model matrix
uniform mat4 view;          // View matrix
uniform mat4 projection;    // Projection matrix
uniform float time;         // Animation time
uniform float waveAmplitude;// Wave height multiplier
uniform float waveFrequency;// Wave spacing
uniform float waveSpeed;    // Animation speed
```

### Gerstner Wave Function

```glsl
vec3 gerstnerWave(vec3 pos, vec2 direction, float amplitude, 
                  float frequency, float speed, float steepness) {
    // Wave number (2π / wavelength)
    float k = 2.0 * 3.14159 / frequency;
    
    // Wave speed from dispersion relation: c = sqrt(g/k)
    // where g = 9.8 m/s² (gravity)
    float c = sqrt(9.8 / k);
    
    // Normalize wave direction
    vec2 d = normalize(direction);
    
    // Phase: k * (d·p - c*speed*t)
    float f = k * (dot(d, pos.xz) - c * speed * time);
    
    // Amplitude factor with steepness
    float a = steepness / k;
    
    // Return 3D displacement
    return vec3(
        d.x * a * cos(f),  // Horizontal X displacement
        a * sin(f),        // Vertical Y displacement
        d.y * a * cos(f)   // Horizontal Z displacement
    );
}
```

**Why Gerstner Waves?**
- Physically accurate ocean wave simulation
- Creates realistic crests (peaks) and troughs (valleys)
- Particles move in circular/elliptical paths (like real water)
- Better than simple sine waves which only displace vertically

**Parameters Explained**:
- `direction`: Wave travel direction (2D vector)
- `amplitude`: Wave height (0.08-0.15 for realistic ocean)
- `frequency`: Distance between wave peaks (6-12 for variety)
- `speed`: How fast waves move (0.6-1.2 for natural motion)
- `steepness`: How sharp the peaks are (0.25-0.4 for realism)

### Gerstner Normal Function

```glsl
vec3 gerstnerNormal(vec3 pos, vec2 direction, float amplitude,
                    float frequency, float speed, float steepness) {
    float k = 2.0 * 3.14159 / frequency;
    float c = sqrt(9.8 / k);
    vec2 d = normalize(direction);
    float f = k * (dot(d, pos.xz) - c * speed * time);
    float wa = k * amplitude;
    
    // Partial derivatives for normal calculation
    return vec3(
        -d.x * wa * cos(f),              // ∂/∂x
        1.0 - steepness * wa * sin(f),   // ∂/∂y (base + wave)
        -d.y * wa * cos(f)               // ∂/∂z
    );
}
```

**Why Calculate Normals?**
- Lighting requires surface normals
- Gerstner waves change surface orientation
- Must recalculate normals for each wave
- Sum all wave normals for final surface normal

### Main Vertex Shader Logic

```glsl
void main() {
    vec3 pos = vp;  // Start with original position
    
    // Apply 4 different Gerstner waves
    vec3 wave1 = gerstnerWave(pos, vec2(1.0, 0.0), 
                              waveAmplitude * 0.15, 8.0, waveSpeed, 0.4);
    vec3 wave2 = gerstnerWave(pos, vec2(0.7, 0.7), 
                              waveAmplitude * 0.12, 6.0, waveSpeed * 0.8, 0.3);
    vec3 wave3 = gerstnerWave(pos, vec2(0.0, 1.0), 
                              waveAmplitude * 0.10, 10.0, waveSpeed * 1.2, 0.35);
    vec3 wave4 = gerstnerWave(pos, vec2(-0.5, 0.8), 
                              waveAmplitude * 0.08, 12.0, waveSpeed * 0.6, 0.25);
    
    // Sum all wave displacements
    pos += wave1 + wave2 + wave3 + wave4;
    waveHeight = pos.y;  // Store for fragment shader
    
    // Calculate normals for each wave
    vec3 n1 = gerstnerNormal(vp, vec2(1.0, 0.0), 
                             waveAmplitude * 0.15, 8.0, waveSpeed, 0.4);
    vec3 n2 = gerstnerNormal(vp, vec2(0.7, 0.7), 
                             waveAmplitude * 0.12, 6.0, waveSpeed * 0.8, 0.3);
    vec3 n3 = gerstnerNormal(vp, vec2(0.0, 1.0), 
                             waveAmplitude * 0.10, 10.0, waveSpeed * 1.2, 0.35);
    vec3 n4 = gerstnerNormal(vp, vec2(-0.5, 0.8), 
                             waveAmplitude * 0.08, 12.0, waveSpeed * 0.6, 0.25);
    
    // Sum and normalize all normals
    vec3 waveNormal = normalize(n1 + n2 + n3 + n4);
    
    // Transform to world space
    vec4 worldPos = model * vec4(pos, 1.0);
    fragPos = vec3(worldPos);
    
    // Transform normal to world space
    normal = mat3(transpose(inverse(model))) * waveNormal;
    
    // Final clip space position
    gl_Position = MVP * vec4(pos, 1.0);
}
```

**Wave Configuration Strategy**:
- Wave 1: Main wave (largest, slowest)
- Wave 2: Secondary wave (diagonal, medium)
- Wave 3: Cross wave (perpendicular, faster)
- Wave 4: Detail wave (small, varied direction)

Different directions prevent repetitive patterns.

---

## Fragment Shader Breakdown

### Input/Output
```glsl
// Input from vertex shader
in vec3 fragPos;      // World position
in vec3 normal;       // Surface normal
in float waveHeight;  // Wave displacement

// Output
out vec4 fragColor;   // Final pixel color (RGBA)
```

### Uniforms
```glsl
uniform vec3 viewPos;          // Camera position
uniform float time;            // Animation time
uniform vec3 waterColor;       // Shallow water color
uniform vec3 depthColor;       // Deep water color
uniform float fresnelStrength; // Reflection intensity
uniform vec3 lightDir;         // Main light direction
uniform vec3 lightColor;       // Main light color
uniform float fogDensity;      // Underwater fog
uniform float underwater;      // 1.0 if camera underwater
```

### Animated Normal Map

```glsl
vec3 animatedNormal(vec2 uv, float time) {
    // Two scrolling UV layers
    vec2 uv1 = uv * 2.0 + vec2(time * 0.05, time * 0.03);
    vec2 uv2 = uv * 3.0 - vec2(time * 0.04, time * 0.06);
    
    // Procedural noise using sine/cosine
    float n1 = sin(uv1.x * 10.0) * cos(uv1.y * 10.0);
    float n2 = sin(uv2.x * 15.0) * cos(uv2.y * 15.0);
    
    // Combine into normal vector
    return normalize(vec3(n1 + n2, 1.0, n1 - n2) * 0.3);
}
```

**Purpose**:
- Adds small-scale surface detail
- Simulates ripples and small waves
- Animates over time for dynamic look
- Doesn't require texture memory

**How it works**:
1. Create two UV layers scrolling at different speeds
2. Generate noise using sine/cosine functions
3. Convert noise to normal perturbation
4. Scale by 0.3 to keep subtle

### Fresnel Effect

```glsl
float fresnel(vec3 viewDir, vec3 normal, float power) {
    float facing = dot(viewDir, normal);
    return pow(1.0 - max(facing, 0.0), power);
}
```

**Fresnel-Schlick Approximation**:
- `facing = 1.0`: Looking straight down (perpendicular)
- `facing = 0.0`: Looking at grazing angle (parallel)
- `1.0 - facing`: Inverts so grazing angle = 1.0
- `pow(x, 5.0)`: Power 5 gives physically accurate water

**Result**:
- Water is more reflective at shallow viewing angles
- Water is more transparent when looking straight down
- Matches real-world water behavior

### Main Fragment Shader Logic

```glsl
void main() {
    // Normalize interpolated normal
    vec3 norm = normalize(normal);
    vec3 viewDir = normalize(viewPos - fragPos);
    
    // Add animated detail to normal
    vec3 detailNormal = animatedNormal(fragPos.xz * 0.5, time);
    norm = normalize(norm + detailNormal * 0.3);
    
    // Calculate Fresnel reflection
    float fresnelFactor = fresnel(viewDir, norm, 5.0) * fresnelStrength;
    
    // Depth-based color interpolation
    float depthFactor = clamp(abs(waveHeight) * 0.5, 0.0, 1.0);
    vec3 baseColor = mix(waterColor, depthColor, depthFactor);
    
    // === LIGHTING ===
    
    // Diffuse lighting
    vec3 lightDirection = normalize(-lightDir);
    float diff = max(dot(norm, lightDirection), 0.0);
    vec3 diffuse = diff * lightColor * 0.6;
    
    // Specular lighting (Blinn-Phong)
    vec3 halfDir = normalize(lightDirection + viewDir);
    float spec = pow(max(dot(norm, halfDir), 0.0), 128.0);
    vec3 specular = spec * lightColor * 1.5;
    
    // Sky reflection (simplified)
    vec3 skyColor = vec3(0.5, 0.7, 0.9);
    vec3 reflectionColor = skyColor * fresnelFactor;
    
    // Combine all lighting
    vec3 finalColor = baseColor * (0.3 + diffuse) + specular + reflectionColor;
    
    // === UNDERWATER FOG ===
    
    if (underwater > 0.5) {
        float dist = length(viewPos - fragPos);
        float fogFactor = 1.0 - exp(-fogDensity * dist);
        finalColor = mix(finalColor, depthColor, fogFactor);
    }
    
    // Transparency with Fresnel
    float alpha = 0.85 + fresnelFactor * 0.15;
    
    fragColor = vec4(finalColor, alpha);
}
```

### Lighting Breakdown

**Diffuse (Lambertian)**:
```glsl
float diff = max(dot(norm, lightDirection), 0.0);
vec3 diffuse = diff * lightColor * 0.6;
```
- Angle between normal and light
- 0.6 multiplier for subtle effect
- Provides base shading

**Specular (Blinn-Phong)**:
```glsl
vec3 halfDir = normalize(lightDirection + viewDir);
float spec = pow(max(dot(norm, halfDir), 0.0), 128.0);
vec3 specular = spec * lightColor * 1.5;
```
- Half-vector between light and view
- Power 128 = very tight highlight (shiny water)
- 1.5 multiplier for bright sun reflections

**Sky Reflection**:
```glsl
vec3 skyColor = vec3(0.5, 0.7, 0.9);
vec3 reflectionColor = skyColor * fresnelFactor;
```
- Simplified sky color (light blue)
- Modulated by Fresnel (more at grazing angles)
- Real implementation would use cubemap

### Fog Calculation

```glsl
float dist = length(viewPos - fragPos);
float fogFactor = 1.0 - exp(-fogDensity * dist);
finalColor = mix(finalColor, depthColor, fogFactor);
```

**Exponential Fog Formula**:
- `fogFactor = 1 - e^(-density * distance)`
- At distance = 0: fogFactor = 0 (no fog)
- At distance = ∞: fogFactor = 1 (full fog)
- Density controls how quickly fog increases

**Example**:
- Density 0.15, Distance 10: fogFactor ≈ 0.78 (78% fog)
- Density 0.15, Distance 20: fogFactor ≈ 0.95 (95% fog)

### Alpha Blending

```glsl
float alpha = 0.85 + fresnelFactor * 0.15;
```

- Base transparency: 0.85 (15% transparent)
- Fresnel adds up to 0.15 more (at grazing angles)
- Result: 0.85 to 1.0 alpha range
- More opaque at shallow angles (realistic)

---

## Shader Optimization Tips

### Vertex Shader
1. **Reduce wave count**: Use 2 waves instead of 4
2. **Simplify calculations**: Pre-compute constants
3. **LOD system**: Fewer waves for distant water

### Fragment Shader
1. **Simplify normal animation**: Use single UV layer
2. **Reduce specular power**: Lower exponent = faster
3. **Skip fog when not underwater**: Early exit
4. **Use lower precision**: `mediump` on mobile

### Example Optimized Vertex Shader
```glsl
// Only 2 waves instead of 4
vec3 wave1 = gerstnerWave(pos, vec2(1.0, 0.0), amp * 0.15, 8.0, speed, 0.4);
vec3 wave2 = gerstnerWave(pos, vec2(0.7, 0.7), amp * 0.12, 6.0, speed * 0.8, 0.3);
pos += wave1 + wave2;  // Skip wave3 and wave4
```

---

## Shader Debugging

### Visualize Normals
```glsl
// In fragment shader
fragColor = vec4(norm * 0.5 + 0.5, 1.0);  // Normal as color
```

### Visualize Wave Height
```glsl
float h = waveHeight * 0.5 + 0.5;
fragColor = vec4(h, h, h, 1.0);  // Height as grayscale
```

### Visualize Fresnel
```glsl
fragColor = vec4(vec3(fresnelFactor), 1.0);  // Fresnel as grayscale
```

### Visualize Fog
```glsl
fragColor = vec4(vec3(fogFactor), 1.0);  // Fog as grayscale
```

---

## Mathematical Foundations

### Gerstner Wave Equation
```
x(t) = x₀ + (k̂ₓ * A / k) * cos(k * (k̂ · x₀ - c * t))
y(t) = A * sin(k * (k̂ · x₀ - c * t))
z(t) = z₀ + (k̂ᵧ * A / k) * cos(k * (k̂ · x₀ - c * t))

where:
k = 2π / λ (wave number)
c = √(g / k) (wave speed)
k̂ = wave direction (unit vector)
A = amplitude
```

### Fresnel-Schlick Approximation
```
F(θ) = F₀ + (1 - F₀) * (1 - cos(θ))⁵

where:
θ = angle between view and normal
F₀ = reflectance at normal incidence (≈0.02 for water)
```

### Exponential Fog
```
f(d) = 1 - e^(-ρ * d)

where:
d = distance from camera
ρ = fog density
```

---

## Shader Variants

### High Quality (Desktop)
- 4 Gerstner waves
- Animated normals
- Full lighting
- Exponential fog

### Medium Quality (Console)
- 3 Gerstner waves
- Simplified normals
- Basic lighting
- Linear fog

### Low Quality (Mobile)
- 2 Gerstner waves
- No normal animation
- Diffuse only
- No fog

---

## Future Shader Enhancements

1. **Foam Generation**: Add whitecaps at wave peaks
2. **Caustics**: Underwater light patterns
3. **Subsurface Scattering**: Light penetration
4. **Reflection Texture**: Real-time reflections
5. **Refraction Texture**: Scene distortion
6. **Particle Integration**: Spray and splashes
7. **Weather Effects**: Rain ripples, wind
8. **Shoreline Interaction**: Wave breaking

Each enhancement requires additional shader code and uniforms.
