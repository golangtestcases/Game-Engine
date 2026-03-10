#version 410

const int MAX_SHAFT_SAMPLES = 96;

in vec2 vUV;

uniform sampler2D uDepthTex;
uniform sampler2D uNoiseTex;

uniform vec2 uScreenSize;
uniform float uNear;
uniform float uFar;
uniform float uTime;
uniform float uWaterLevel;
uniform float uUnderwaterBlend;
uniform vec3 uCameraPos;
uniform vec3 uLightDir;
uniform vec2 uSunScreenPos;
uniform float uSunVisibility;

uniform float uIntensity;
uniform float uDensity;
uniform float uFalloff;
uniform float uScatteringStrength;
uniform float uUnderwaterBoost;
uniform float uSurfaceIntensity;
uniform float uUnderwaterOnly;
uniform float uNoiseStrength;
uniform float uNoiseSpeed;
uniform int uSampleCount;

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

float underwaterSegmentLength(vec3 startPos, vec3 endPos) {
    vec3 delta = endPos - startPos;
    float rayLen = length(delta);
    if (rayLen <= 0.0001) {
        return 0.0;
    }

    bool startUnder = startPos.y < uWaterLevel;
    bool endUnder = endPos.y < uWaterLevel;
    if (startUnder && endUnder) {
        return rayLen;
    }
    if (!startUnder && !endUnder) {
        return 0.0;
    }

    if (abs(delta.y) <= 0.0001) {
        return startUnder ? rayLen : 0.0;
    }

    float tSurface = clamp((uWaterLevel - startPos.y) / delta.y, 0.0, 1.0);
    if (startUnder) {
        return rayLen * tSurface;
    }
    return rayLen * (1.0 - tSurface);
}

void main() {
    float depthRaw = texture(uDepthTex, vUV).r;
    bool hasGeometry = depthRaw < 0.99995;
    float fallbackDepth = min(uFar * 0.86, max(uVisibilityDistance * 1.25, 40.0));
    float linearDepth = hasGeometry ? linearizeDepth(depthRaw) : fallbackDepth;
    vec3 worldPos = hasGeometry ? reconstructWorldPos(vUV, depthRaw) : uCameraPos + worldViewDir(vUV) * linearDepth;

    float mediumPath = underwaterSegmentLength(uCameraPos, worldPos);
    if (mediumPath <= 0.0001 || uSunVisibility <= 0.0001) {
        fragColor = vec4(0.0, 0.0, 0.0, 1.0);
        return;
    }

    float cameraDepth = max(uWaterLevel - uCameraPos.y, 0.0);
    // Русский комментарий: повышаем базовый вклад среды, чтобы лучи читались как
    // объемные столбы света уже при небольшой глубине камеры под водой.
    float underwaterWeight = saturate(uUnderwaterBlend * uUnderwaterBoost * (0.56 + cameraDepth * 0.13));
    underwaterWeight *= 0.78 + 0.22 * saturate(mediumPath / max(uVisibilityDistance * 0.38, 2.0));

    float surfaceWeight = (1.0 - step(0.5, uUnderwaterOnly)) * saturate(uSurfaceIntensity);
    surfaceWeight *= saturate(1.0 - uUnderwaterBlend * 0.92);
    surfaceWeight *= saturate(mediumPath / max(uVisibilityDistance * 0.45, 2.0));

    float mediumWeight = max(underwaterWeight, surfaceWeight);
    if (mediumWeight <= 0.0001) {
        fragColor = vec4(0.0, 0.0, 0.0, 1.0);
        return;
    }

    vec3 rayDirWorld = normalize(worldPos - uCameraPos);
    vec3 mediumPoint = uCameraPos + rayDirWorld * (mediumPath * 0.55);
    float depthBelowSurface = max(uWaterLevel - mediumPoint.y, 0.0);

    vec3 sunDir = normalize(-uLightDir);
    float cosTheta = dot(rayDirWorld, sunDir);
    float g = 0.72;
    float denom = pow(max(1.0 + g * g - 2.0 * g * cosTheta, 0.02), 1.5);
    float phase = ((1.0 - g * g) / (4.0 * 3.14159265 * denom)) * 7.6;

    vec2 toSun = uSunScreenPos - vUV;
    float sunDistance = length(toSun);
    vec2 rayDir = toSun / max(sunDistance, 0.0001);
    float centerMask = 1.0 / (1.0 + sunDistance * (3.6 + uScatteringStrength * 2.6));
    centerMask = pow(centerMask, 0.34);

    int steps = clamp(uSampleCount, 8, MAX_SHAFT_SAMPLES);
    float jitter = texture(uNoiseTex, vUV * 4.5 + vec2(uTime * 0.11, -uTime * 0.09)).r;
    float accumulated = 0.0;
    float transmittance = 1.0;

    for (int i = 0; i < MAX_SHAFT_SAMPLES; ++i) {
        if (i >= steps) {
            break;
        }

        float t = (float(i) + jitter) / float(steps);
        vec2 sampleUV = vUV + rayDir * t * sunDistance;

        vec2 flow = vec2(uTime * uNoiseSpeed * 0.08, -uTime * uNoiseSpeed * 0.06);
        float n0 = texture(uNoiseTex, sampleUV * 3.6 + flow).r;
        float n1 = texture(uNoiseTex, sampleUV * 7.1 - flow * 1.3).r;
        vec2 distortion = vec2(n0 - 0.5, n1 - 0.5) * (uNoiseStrength * (0.18 + 0.82 * t)) / uScreenSize;
        sampleUV = clamp(sampleUV + distortion, vec2(0.001), vec2(0.999));

        float sampleDepthRaw = texture(uDepthTex, sampleUV).r;
        float sampleDepth = sampleDepthRaw < 0.99995 ? linearizeDepth(sampleDepthRaw) : fallbackDepth;
        float depthDelta = linearDepth - sampleDepth;
        float blocker = smoothstep(0.30, 6.8, depthDelta);
        blocker *= (1.0 - step(0.99995, sampleDepthRaw));

        float travel = t * max(mediumPath, linearDepth * 0.55);
        float mediumDecay = exp(-travel * (0.0038 + uDensity * 0.0072));
        accumulated += transmittance * mediumDecay * (1.0 - blocker);
        transmittance *= mix(1.0, 0.58, blocker);
        transmittance *= 0.994;
    }

    accumulated /= float(steps);
    float verticalFade = exp(-depthBelowSurface * uFalloff * 0.34);
    float distanceFade = exp(-linearDepth * (uFalloff * 0.09 + 0.00095));
    float murkyFade = exp(-mediumPath * (0.0012 + uDensity * 0.0026));
    float visibilityNorm = saturate(mediumPath / max(uVisibilityDistance, 1.0));
    float visibilityFade = 1.0 - visibilityNorm * 0.12;
    // Русский комментарий: мягкий буст для этапа верификации видимости лучей.
    float nearSurfaceFocus = 0.82 + 0.28 * saturate(1.0 - depthBelowSurface / max(uVisibilityDistance * 0.42, 4.0));
    // Русский комментарий: аккуратный буст для читаемости лучей в суммарном fog/water
    // без отделения эффекта в отдельный декоративный overlay.
    float readabilityBoost = 1.42;

    float shaftStrength = accumulated *
        phase *
        centerMask *
        verticalFade *
        distanceFade *
        murkyFade *
        visibilityFade *
        nearSurfaceFocus *
        mediumWeight *
        uIntensity *
        uScatteringStrength *
        uSunVisibility *
        readabilityBoost;

    shaftStrength = saturate(shaftStrength);

    vec3 deepTint = mix(uFogColor, uDepthTint, 0.62);
    vec3 sunTint = mix(deepTint * 1.08, uSunColor, 0.72);
    float warmMix = saturate(phase * 0.85 + (1.0 - verticalFade) * 0.30 + surfaceWeight * 0.35);
    vec3 shaftColor = mix(deepTint, sunTint, warmMix);
    shaftColor *= 0.72 + 0.38 * (1.0 - saturate(depthBelowSurface / max(uVisibilityDistance * 0.35, 4.0)));

    fragColor = vec4(shaftColor * shaftStrength, 1.0);
}
