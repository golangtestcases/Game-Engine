#version 410

in VS_OUT {
    vec3 worldPos;
    vec3 worldNormal;
    vec3 viewPos;
    float crest;
    float displacement;
} fs_in;

uniform vec3 uCameraPos;
uniform vec2 uScreenSize;
uniform float uTime;
uniform float uNear;
uniform float uFar;
uniform float uWaterLevel;
uniform float uUnderwaterBlend;
uniform int uSSRMaxSteps;
uniform int uUsePlanarReflection;
uniform float uCoverageFadeNear;
uniform float uCoverageFadeFar;

uniform mat4 uProjection;
uniform mat4 uInvProjection;
uniform mat4 uView;

uniform vec3 uLightDir;
uniform vec3 uLightColor;
uniform float uSunIntensity;
uniform vec3 uShallowColor;
uniform vec3 uDeepColor;
uniform vec3 uFoamColor;
uniform vec3 uScatterColor;
uniform vec3 uSkyHorizonColor;
uniform vec3 uSkyZenithColor;
uniform float uAbsorption;
uniform float uRefractionStrength;
uniform float uReflectionStrength;
uniform float uRoughness;
uniform float uFoamIntensity;

uniform sampler2D uSceneColor;
uniform sampler2D uSceneDepth;
uniform sampler2D uReflectionTex;
uniform sampler2D uNormalTex0;
uniform sampler2D uNormalTex1;
uniform sampler2D uNormalTex2;
uniform sampler2D uFoamNoiseTex;

out vec4 fragColor;

float saturate(float v) {
    return clamp(v, 0.0, 1.0);
}

float linearizeDepth(float depth) {
    float z = depth * 2.0 - 1.0;
    return (2.0 * uNear * uFar) / max(uFar + uNear - z * (uFar - uNear), 0.0001);
}

vec3 reconstructViewPos(vec2 uv, float depth) {
    vec4 clip = vec4(uv * 2.0 - 1.0, depth * 2.0 - 1.0, 1.0);
    vec4 view = uInvProjection * clip;
    return view.xyz / max(view.w, 0.0001);
}

vec3 fresnelSchlick(float cosTheta, vec3 F0) {
    return F0 + (1.0 - F0) * pow(1.0 - cosTheta, 5.0);
}

float distributionGGX(float NdotH, float alpha) {
    float a2 = alpha * alpha;
    float d = (NdotH * NdotH) * (a2 - 1.0) + 1.0;
    return a2 / max(3.14159265 * d * d, 0.0001);
}

float geometrySchlickGGX(float NdotX, float k) {
    return NdotX / max(NdotX * (1.0 - k) + k, 0.0001);
}

float geometrySmith(float NdotV, float NdotL, float k) {
    return geometrySchlickGGX(NdotV, k) * geometrySchlickGGX(NdotL, k);
}

vec3 skyColor(vec3 dir, vec3 sunDir) {
    float h = saturate(dir.y * 0.5 + 0.5);
    vec3 horizon = uSkyHorizonColor;
    vec3 zenith = uSkyZenithColor;
    vec3 sky = mix(horizon, zenith, pow(h, 0.58));

    float sunCos = max(dot(dir, sunDir), 0.0);
    float sunDisk = pow(sunCos, 1200.0);
    float mie = pow(sunCos, 10.0);
    vec3 warm = mix(vec3(1.0, 0.72, 0.42), uLightColor, 0.64);
    sky += uLightColor * sunDisk * (12.0 + uSunIntensity * 5.5);
    sky += warm * mie * (0.45 + uSunIntensity * 0.28);
    return sky;
}

bool traceSSR(vec3 viewPos, vec3 rayDir, int maxSteps, out vec3 hitColor, out float hitStrength) {
    if (maxSteps <= 0) {
        return false;
    }

    vec3 ray = viewPos + rayDir * 0.35;
    float stride = 0.72;

    for (int i = 0; i < 32; ++i) {
        if (i >= maxSteps) {
            break;
        }

        ray += rayDir * stride;
        vec4 clip = uProjection * vec4(ray, 1.0);
        if (clip.w <= 0.0) {
            break;
        }

        vec2 uv = clip.xy / clip.w * 0.5 + 0.5;
        if (uv.x <= 0.001 || uv.x >= 0.999 || uv.y <= 0.001 || uv.y >= 0.999) {
            break;
        }

        float depthRaw = texture(uSceneDepth, uv).r;
        if (depthRaw >= 0.99999) {
            stride *= 1.05;
            continue;
        }

        vec3 sceneView = reconstructViewPos(uv, depthRaw);
        float rayDepth = -ray.z;
        float sceneDepth = -sceneView.z;
        float delta = rayDepth - sceneDepth;
        float thickness = 0.45 + float(i) * 0.06;

        if (delta > 0.0 && delta < thickness) {
            hitColor = texture(uSceneColor, uv).rgb;
            hitStrength = 1.0 - float(i) / float(maxSteps);
            return true;
        }

        stride *= 1.06;
    }

    return false;
}

void main() {
    vec2 uv = gl_FragCoord.xy / uScreenSize;
    vec3 N = normalize(fs_in.worldNormal);

    vec2 uv0 = fs_in.worldPos.xz * 0.045 + vec2(uTime * 0.032, uTime * 0.018);
    vec2 uv1 = fs_in.worldPos.xz * 0.082 + vec2(-uTime * 0.024, uTime * 0.029);
    vec2 uv2 = fs_in.worldPos.xz * 0.154 + vec2(uTime * 0.045, -uTime * 0.037);

    vec3 n0 = texture(uNormalTex0, uv0).xyz * 2.0 - 1.0;
    vec3 n1 = texture(uNormalTex1, uv1).xyz * 2.0 - 1.0;
    vec3 n2 = texture(uNormalTex2, uv2).xyz * 2.0 - 1.0;

    vec3 micro = normalize(vec3(n0.x + n1.x * 0.85 + n2.x * 0.55, n0.z + n1.z * 0.7 + n2.z * 0.45, n0.y + n1.y * 0.85 + n2.y * 0.55));
    vec3 microWorld = normalize(vec3(micro.x, micro.z, micro.y));
    N = normalize(mix(N, normalize(N + microWorld * 0.7), 0.42));

    vec3 V = normalize(uCameraPos - fs_in.worldPos);
    float NdotV = saturate(dot(N, V));
    float edgeFade = 1.0;
    if (uCoverageFadeFar > uCoverageFadeNear + 0.0001) {
        float edgeDistance = distance(fs_in.worldPos.xz, uCameraPos.xz);
        edgeFade = 1.0 - smoothstep(uCoverageFadeNear, uCoverageFadeFar, edgeDistance);
    }

    float sceneDepthRaw = texture(uSceneDepth, uv).r;
    float waterDepthRaw = gl_FragCoord.z;
    float sceneDepth = linearizeDepth(sceneDepthRaw);
    float waterDepth = linearizeDepth(waterDepthRaw);
    float thickness = max(sceneDepth - waterDepth, 0.0);

    float depthBlend = 1.0 - exp(-thickness * 0.065);
    vec3 depthTint = mix(uShallowColor, uDeepColor, saturate(depthBlend));

    vec2 distortion = (vec2(micro.x, micro.z) * 0.015 + N.xz * 0.01) * (1.0 + fs_in.crest * 0.6);
    vec2 refractUV = clamp(uv + distortion * uRefractionStrength, vec2(0.001), vec2(0.999));

    float chroma = 0.0015;
    vec3 refractScene;
    refractScene.r = texture(uSceneColor, clamp(refractUV + distortion * chroma, vec2(0.001), vec2(0.999))).r;
    refractScene.g = texture(uSceneColor, refractUV).g;
    refractScene.b = texture(uSceneColor, clamp(refractUV - distortion * chroma, vec2(0.001), vec2(0.999))).b;

    float transmittance = exp(-uAbsorption * max(thickness, 0.02));
    vec3 refractionColor = mix(depthTint, refractScene, transmittance * 0.9);

    vec3 worldViewRay = normalize(fs_in.worldPos - uCameraPos);
    vec3 reflectedWorld = reflect(worldViewRay, N);
    vec3 sunDir = normalize(-uLightDir);
    float underwaterReflectionFade = saturate(1.0 - uUnderwaterBlend * 1.35);

    vec2 planarUV = clamp(uv + vec2(N.x, -N.z) * 0.055 * uReflectionStrength, vec2(0.001), vec2(0.999));
    vec3 skyReflection = skyColor(reflectedWorld, sunDir);
    vec3 planarReflection = skyReflection;
    float planarMix = (uUsePlanarReflection > 0) ? underwaterReflectionFade : 0.0;
    if (planarMix > 0.001) {
        planarReflection = texture(uReflectionTex, vec2(planarUV.x, 1.0 - planarUV.y)).rgb;
    }
    vec3 fallbackReflection = mix(skyReflection, planarReflection, 0.72 * planarMix);

    vec3 viewNormal = normalize(mat3(uView) * N);
    vec3 rayDir = normalize(reflect(normalize(fs_in.viewPos), viewNormal));
    vec3 ssrColor = vec3(0.0);
    float ssrStrength = 0.0;
    float ssrFade = saturate(1.0 - uUnderwaterBlend * 1.1);
    int ssrSteps = int(float(min(max(uSSRMaxSteps, 0), 32)) * ssrFade + 0.5);
    bool ssrHit = traceSSR(fs_in.viewPos, rayDir, ssrSteps, ssrColor, ssrStrength);
    float ssrMix = ssrStrength * ssrFade;
    vec3 reflectionColor = ssrHit ? mix(fallbackReflection, ssrColor, ssrMix) : fallbackReflection;

    float fresnel = pow(1.0 - NdotV, 5.0);
    vec3 baseSurface = mix(refractionColor, reflectionColor, saturate(fresnel * 1.15 + 0.08));

    vec3 L = sunDir;
    vec3 H = normalize(L + V);
    float NdotL = saturate(dot(N, L));
    float NdotH = saturate(dot(N, H));
    float VdotH = saturate(dot(V, H));
    float alpha = max(uRoughness * uRoughness, 0.02);
    float D = distributionGGX(NdotH, alpha);
    float k = (alpha + 1.0) * (alpha + 1.0) * 0.125;
    float G = geometrySmith(NdotV, NdotL, k);
    vec3 F = fresnelSchlick(VdotH, vec3(0.02));
    vec3 specular = (D * G * F) / max(4.0 * NdotV * NdotL, 0.0001);

    float glint = pow(max(dot(reflect(-L, N), V), 0.0), 160.0) * (0.6 + 0.4 * micro.y);
    vec3 litSurface = baseSurface + specular * uLightColor * 1.28 + glint * uLightColor * 0.9;

    float shoreMask = exp(-thickness * 1.8);
    float intersectionMask = exp(-thickness * 5.4);
    float noise = texture(uFoamNoiseTex, fs_in.worldPos.xz * 0.03 + vec2(uTime * 0.028, -uTime * 0.021)).r;
    float foam = saturate((fs_in.crest * 1.2 + shoreMask * 1.4 + intersectionMask * 0.9) * (0.55 + noise * 0.9)) * uFoamIntensity * edgeFade;
    vec3 foamColor = uFoamColor * foam;

    vec3 finalColor = litSurface + foamColor;
    if (uUnderwaterBlend > 0.001) {
        float underwaterFog = saturate(1.0 - exp(-thickness * 0.045));
        vec3 underwaterColor = mix(refractionColor, uScatterColor + depthTint * 0.42, underwaterFog);
        finalColor = mix(finalColor, underwaterColor, uUnderwaterBlend * 0.42);
    }

    float alphaOut = saturate(0.64 + fresnel * 0.24 + foam * 0.2);
    alphaOut = mix(alphaOut, 0.74, uUnderwaterBlend);
    alphaOut *= edgeFade;
    fragColor = vec4(finalColor, alphaOut);
}
