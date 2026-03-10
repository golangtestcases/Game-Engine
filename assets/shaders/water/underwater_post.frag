#version 410

in vec2 vUV;

uniform sampler2D uColorTex;
uniform sampler2D uDepthTex;
uniform sampler2D uCausticTex;
uniform sampler2D uShaftTex;

uniform vec2 uScreenSize;
uniform float uNear;
uniform float uFar;
uniform float uTime;
uniform float uWaterLevel;
uniform float uUnderwaterBlend;
uniform float uShaftEnabled;
uniform float uShaftUnderwaterOnly;
uniform float uShaftSurfaceIntensity;
uniform vec3 uCameraPos;
uniform vec3 uLightDir;
uniform vec3 uFogColor;
uniform vec3 uDepthTint;
uniform vec3 uSunColor;
uniform float uVisibilityDistance;
uniform mat4 uInvProjection;
uniform mat4 uInvView;

out vec4 fragColor;

float saturate(float v) {
    return clamp(v, 0.0, 1.0);
}

float linearizeDepth(float depth) {
    float z = depth * 2.0 - 1.0;
    return (2.0 * uNear * uFar) / max(uFar + uNear - z * (uFar - uNear), 0.0001);
}

vec3 reconstructWorldPos(vec2 uv, float depth) {
    vec4 clip = vec4(uv * 2.0 - 1.0, depth * 2.0 - 1.0, 1.0);
    vec4 view = uInvProjection * clip;
    view /= max(view.w, 0.0001);
    vec4 world = uInvView * view;
    return world.xyz;
}

vec3 worldViewDir(vec2 uv) {
    vec4 clip = vec4(uv * 2.0 - 1.0, 1.0, 1.0);
    vec4 view = uInvProjection * clip;
    vec3 viewDir = normalize(view.xyz / max(view.w, 0.0001));
    return normalize((uInvView * vec4(viewDir, 0.0)).xyz);
}

void main() {
    vec3 color = texture(uColorTex, vUV).rgb;
    float depthRaw = texture(uDepthTex, vUV).r;

    float underwaterMask = saturate(uUnderwaterBlend);
    bool shaftEnabled = uShaftEnabled > 0.001;
    bool shaftsAboveWaterAllowed = shaftEnabled &&
        (uShaftUnderwaterOnly < 0.5) &&
        (uShaftSurfaceIntensity > 0.0001);

    if (underwaterMask <= 0.001 && !shaftsAboveWaterAllowed) {
        fragColor = vec4(color, 1.0);
        return;
    }

    bool hasGeometry = depthRaw < 0.99999;
    float fallbackDepth = min(uFar * 0.86, max(uVisibilityDistance * 1.2, 36.0));
    float linearDepth = hasGeometry ? linearizeDepth(depthRaw) : fallbackDepth;
    vec3 worldPos = hasGeometry ? reconstructWorldPos(vUV, depthRaw) : uCameraPos + worldViewDir(vUV) * linearDepth;
    vec3 viewDir = normalize(worldPos - uCameraPos);
    float depthBelowSurface = max(uWaterLevel - worldPos.y, 0.0);
    float cameraDepth = max(uWaterLevel - uCameraPos.y, 0.0);
    float warmMask = saturate((color.r + color.g * 0.85 - color.b * 1.15 - 0.12) * 1.6);
    float reverseSandDepth = saturate(cameraDepth * 0.10);
    float seabedMask = saturate((depthBelowSurface - 3.2) * 0.22);
    float sandLikeMask = saturate(max(warmMask, seabedMask * 0.9));

    float distanceFog = (1.0 - exp(-linearDepth * (0.010 + underwaterMask * 0.014))) * underwaterMask;
    float verticalFog = saturate(depthBelowSurface * 0.04) * underwaterMask;
    float cameraFog = saturate(cameraDepth * 0.028) * underwaterMask;
    float fogFactor = saturate(distanceFog * 0.72 + verticalFog * 0.24 + cameraFog * 0.18);

    float depthRetention = exp(-linearDepth * 0.012);
    float verticalRetention = exp(-depthBelowSurface * 0.035);
    float bottomRetention = saturate(0.22 + depthRetention * verticalRetention * 0.78);
    fogFactor *= (1.0 - bottomRetention * 0.38 * underwaterMask);
    fogFactor *= (1.0 - sandLikeMask * reverseSandDepth * 0.65 * underwaterMask);

    vec3 shallowFog = mix(uFogColor, uDepthTint, 0.22);
    vec3 deepFog = mix(uDepthTint, uFogColor * 0.45, 0.68);
    vec3 fogColor = mix(shallowFog, deepFog, saturate(verticalFog + cameraFog * 0.6));
    vec3 sandFogTint = mix(uFogColor, vec3(0.30, 0.26, 0.18), 0.74);
    float sandFogInfluence = saturate(bottomRetention * 0.35 + sandLikeMask * reverseSandDepth * 0.85) * underwaterMask;
    fogColor = mix(fogColor, sandFogTint, sandFogInfluence);

    vec2 causticUV0 = worldPos.xz * 0.09 + vec2(uTime * 0.07, -uTime * 0.05);
    vec2 causticUV1 = worldPos.xz * 0.14 + vec2(-uTime * 0.045, uTime * 0.058);
    float c0 = texture(uCausticTex, causticUV0).r;
    float c1 = texture(uCausticTex, causticUV1).r;

    vec3 sunDir = normalize(-uLightDir);
    float phase = pow(max(dot(-viewDir, sunDir), 0.0), 5.5);
    vec3 scatteringTint = mix(uFogColor, uSunColor, 0.28);
    vec3 scattering = scatteringTint * phase * (0.16 + 0.24 * (1.0 - fogFactor));
    scattering *= (1.0 - bottomRetention * 0.5);
    scattering *= (1.0 - sandLikeMask * reverseSandDepth * 0.75);
    scattering *= underwaterMask;

    vec3 shafts = vec3(0.0);
    if (shaftEnabled) {
        vec2 shaftDistort = vec2(c0 - 0.5, c1 - 0.5) * 0.011 * (0.55 + 0.45 * underwaterMask);
        shafts = texture(uShaftTex, clamp(vUV + shaftDistort, vec2(0.001), vec2(0.999))).rgb;

        float shaftFade = saturate((1.0 - fogFactor * 0.16) * (0.74 + cameraFog * 1.05 + (1.0 - underwaterMask) * 0.22));
        shaftFade *= exp(-depthBelowSurface * (0.006 + underwaterMask * 0.002));
        shaftFade *= mix(1.0, (1.0 - sandLikeMask * reverseSandDepth * 0.28), underwaterMask);

        float shaftContext = mix(saturate(uShaftSurfaceIntensity), 1.0, underwaterMask);
        if (uShaftUnderwaterOnly > 0.5 && underwaterMask <= 0.001) {
            shaftContext = 0.0;
        }

        vec3 shaftSceneTint = mix(vec3(1.0), mix(uFogColor, uSunColor, 0.66), 0.14 + underwaterMask * 0.18);
        shafts *= shaftFade * shaftContext * shaftSceneTint;
        shafts *= (1.28 + underwaterMask * 0.34);
    }

    vec3 fogged = mix(color, fogColor, fogFactor);
    vec3 bottomBoost = color * vec3(1.20, 1.12, 0.92) * bottomRetention * 0.22 * underwaterMask;
    vec3 graded = fogged + bottomBoost;
    graded += scattering + shafts;

    float preserve = sandLikeMask * bottomRetention * (0.55 + 0.45 * (1.0 - fogFactor));
    preserve = saturate(preserve + sandLikeMask * reverseSandDepth * 1.10);
    preserve *= underwaterMask;
    // Русский комментарий: не даем preserve-слою полностью «перетереть» объемные лучи.
    float shaftPresence = saturate(dot(shafts, vec3(0.299, 0.587, 0.114)) * 1.8);
    preserve *= (1.0 - shaftPresence * 0.55);
    vec3 preservedSand = mix(color * vec3(1.08, 1.02, 0.90), vec3(0.78, 0.68, 0.48), sandLikeMask * 0.32);
    graded = mix(graded, preservedSand, preserve * 0.90);

    float lum = dot(graded, vec3(0.299, 0.587, 0.114));
    float desaturateAmount = 0.92 + 0.08 * (1.0 - underwaterMask);
    graded = mix(vec3(lum), graded, desaturateAmount);

    fragColor = vec4(graded, 1.0);
}
