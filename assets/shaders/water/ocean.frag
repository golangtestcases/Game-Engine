#version 410
in vec3 fragPos;
in vec3 normal;
in float waveHeight;
in float waveFoam;

uniform vec3 viewPos;
uniform float time;
uniform vec3 waterColor;
uniform vec3 shallowColor;
uniform vec3 depthColor;
uniform vec3 foamColor;
uniform vec3 lightDir;
uniform vec3 lightColor;
uniform float fresnelStrength;
uniform float foamIntensity;
uniform float absorption;
uniform float roughness;
uniform float underwater;
uniform vec2 screenSize;
uniform float cameraNear;
uniform float cameraFar;
uniform sampler2D sceneColorTex;
uniform sampler2D sceneDepthTex;
uniform sampler2D reflectionTex;
uniform sampler2D normalMap0;
uniform sampler2D normalMap1;
uniform float refractionScale;
uniform float reflectionScale;

out vec4 fragColor;

float saturate(float v) {
    return clamp(v, 0.0, 1.0);
}

float linearizeDepth(float depth) {
    float z = depth * 2.0 - 1.0;
    return (2.0 * cameraNear * cameraFar) / max(cameraFar + cameraNear - z * (cameraFar - cameraNear), 0.0001);
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
    float ggx1 = geometrySchlickGGX(NdotV, k);
    float ggx2 = geometrySchlickGGX(NdotL, k);
    return ggx1 * ggx2;
}

vec3 fresnelSchlick(float cosTheta, vec3 F0) {
    return F0 + (1.0 - F0) * pow(1.0 - cosTheta, 5.0);
}

vec3 skyModel(vec3 dir, vec3 sunDir) {
    float h = saturate(dir.y * 0.5 + 0.5);
    vec3 horizon = vec3(0.56, 0.70, 0.86);
    vec3 zenith = vec3(0.07, 0.22, 0.47);
    vec3 sky = mix(horizon, zenith, pow(h, 0.60));

    float sunCos = max(dot(dir, sunDir), 0.0);
    float sunDisk = pow(sunCos, 1400.0);
    float mie = pow(sunCos, 12.0);
    sky += vec3(1.0, 0.96, 0.86) * sunDisk * 20.0;
    sky += vec3(1.0, 0.70, 0.42) * mie * 0.75;
    return sky;
}

void main() {
    vec2 uv = gl_FragCoord.xy / screenSize;
    vec3 n = normalize(normal);

    vec2 uvA = fragPos.xz * 0.047 + vec2(time * 0.032, time * 0.017);
    vec2 uvB = fragPos.xz * 0.091 + vec2(-time * 0.020, time * 0.028);
    vec3 nA = texture(normalMap0, uvA).xyz * 2.0 - 1.0;
    vec3 nB = texture(normalMap1, uvB).xyz * 2.0 - 1.0;
    vec3 detailN = normalize(vec3(nA.x + nB.x, 1.0, nA.y + nB.y));
    n = normalize(n + detailN * 0.40);

    vec3 viewDir = normalize(viewPos - fragPos);
    vec3 lightDirection = normalize(-lightDir);

    float sceneDepthRaw = texture(sceneDepthTex, uv).r;
    float waterDepthRaw = gl_FragCoord.z;
    float sceneDepth = linearizeDepth(sceneDepthRaw);
    float waterDepth = linearizeDepth(waterDepthRaw);
    float thickness = max(sceneDepth - waterDepth, 0.0);

    float shoreMask = 1.0 - saturate(thickness * 0.40);

    vec2 distortion = n.xz * (0.012 + waveFoam * 0.020);
    vec2 refractUV = clamp(uv + distortion * refractionScale, vec2(0.001), vec2(0.999));
    vec2 reflectUV = clamp(uv + vec2(n.x, -n.z) * 0.060 * reflectionScale, vec2(0.001), vec2(0.999));

    // Небольшое хроматическое расщепление делает рефракцию визуально правдоподобнее.
    float chroma = 0.0016;
    vec3 sceneRefract;
    sceneRefract.r = texture(sceneColorTex, clamp(refractUV + distortion * chroma, vec2(0.001), vec2(0.999))).r;
    sceneRefract.g = texture(sceneColorTex, refractUV).g;
    sceneRefract.b = texture(sceneColorTex, clamp(refractUV - distortion * chroma, vec2(0.001), vec2(0.999))).b;

    vec3 planarReflection = texture(reflectionTex, vec2(reflectUV.x, 1.0 - reflectUV.y)).rgb;
    vec3 skyReflection = skyModel(reflect(-viewDir, n), lightDirection);
    vec3 reflectionSource = mix(skyReflection, planarReflection, 0.78);

    float transmittance = exp(-absorption * max(thickness, 0.02));
    vec3 volumetricTint = mix(shallowColor, depthColor, saturate(thickness * 0.07));
    vec3 refractedColor = mix(volumetricTint, sceneRefract, transmittance * 0.88);
    refractedColor = mix(waterColor, refractedColor, 0.90);

    float NdotV = saturate(dot(n, viewDir));
    float NdotL = saturate(dot(n, lightDirection));
    vec3 halfDir = normalize(lightDirection + viewDir);
    float NdotH = saturate(dot(n, halfDir));
    float VdotH = saturate(dot(viewDir, halfDir));

    float alpha = max(roughness * roughness, 0.020);
    float D = distributionGGX(NdotH, alpha);
    float k = (alpha + 1.0) * (alpha + 1.0) * 0.125;
    float G = geometrySmith(NdotV, NdotL, k);
    vec3 F0 = vec3(0.02);
    vec3 F = fresnelSchlick(VdotH, F0) * fresnelStrength;
    vec3 specular = (D * G * F) / max(4.0 * NdotV * NdotL, 0.0001);

    float fresnel = pow(1.0 - NdotV, 5.0) * fresnelStrength;
    vec3 baseComposite = mix(refractedColor, reflectionSource, saturate(fresnel));

    float glint = pow(max(dot(reflect(-lightDirection, n), viewDir), 0.0), 200.0) * (0.7 + (nA.z + nB.z) * 0.5);
    vec3 glintColor = lightColor * max(glint, 0.0) * 1.6;
    vec3 litWater = baseComposite + specular * lightColor * 1.25 + glintColor;

    float microFoam = saturate((nA.z + nB.z) * 0.5);
    float crestFoam = pow(saturate(waveFoam * 1.42 + max(waveHeight, 0.0) * 3.8), 1.20);
    float foamMask = saturate(crestFoam + shoreMask * 1.35 + microFoam * 0.35) * foamIntensity;
    vec3 foam = foamColor * foamMask;

    vec3 finalColor = litWater + foam;
    if (underwater > 0.5) {
        float underwaterFog = saturate(thickness * 0.08 + 0.20);
        finalColor = mix(sceneRefract, depthColor, underwaterFog);
    }

    float alphaOut = saturate(0.58 + saturate(thickness * 0.03) * 0.26 + fresnel * 0.24 + foamMask * 0.18);
    if (underwater > 0.5) {
        alphaOut = 0.68;
    }
    fragColor = vec4(finalColor, alphaOut);
}
