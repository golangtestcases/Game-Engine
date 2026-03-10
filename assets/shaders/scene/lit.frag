#version 410

#define MAX_LIGHTS 8
#define LIGHT_TYPE_DIRECTIONAL 0
#define LIGHT_TYPE_POINT 1
#define LIGHT_TYPE_SPOT 2

struct Material {
    vec3 ambient;
    vec3 diffuse;
    vec3 specular;
    vec3 emission;
    float shininess;
    float specularStrength;
};

struct AmbientLight {
    vec3 color;
    float intensity;
};

struct Light {
    int type;
    vec3 position;
    vec3 direction;
    vec3 color;
    float intensity;
    float constant;
    float linear;
    float quadratic;
    float cutOff;
    float outerCutOff;
};

uniform vec3 objectColor;
uniform vec3 lowColor;
uniform vec3 highColor;
uniform vec2 heightRange;
uniform float heightTint;

uniform vec3 fogColor;
uniform vec2 fogRange;
uniform float fogStrength;
uniform float fogAmount;
uniform float underwaterBlend;
uniform float underwaterFogDensity;
uniform vec3 depthTint;
uniform float sunlightAttenuation;
uniform float visibilityDistance;
uniform float underwaterDepthScale;

uniform float time;
uniform float causticsSpeed;
uniform float causticsScale;
uniform float causticsIntensity;
uniform float causticsContrast;
uniform float causticsDepthFade;
uniform float waterLevel;

uniform vec3 viewPos;
uniform Material material;
uniform AmbientLight ambientLight;
uniform int numLights;
uniform int shadowLightIndex;
uniform Light lights[MAX_LIGHTS];

uniform sampler2D shadowMap;
uniform float shadowBiasMin;
uniform float shadowBiasSlope;
uniform float shadowStrength;
uniform sampler2D causticTex0;
uniform sampler2D causticTex1;

in float vHeight;
in vec3 fragPos;
in vec3 normal;
in vec4 fragPosLightSpace;

out vec4 frag_colour;

float saturate(float v) {
    return clamp(v, 0.0, 1.0);
}

float hash(vec2 p) {
    return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453);
}

float noise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    f = f * f * (3.0 - 2.0 * f);
    return mix(
        mix(hash(i), hash(i + vec2(1.0, 0.0)), f.x),
        mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), f.x),
        f.y
    );
}

float shadowCalculation(vec4 fragPosLight, vec3 norm, vec3 lightDir) {
    vec3 projCoords = fragPosLight.xyz / fragPosLight.w;
    projCoords = projCoords * 0.5 + 0.5;
    if (projCoords.z <= 0.0 || projCoords.z > 1.0 ||
        projCoords.x < 0.0 || projCoords.x > 1.0 ||
        projCoords.y < 0.0 || projCoords.y > 1.0) {
        return 0.0;
    }

    float currentDepth = projCoords.z;
    float ndotl = max(dot(normalize(norm), normalize(lightDir)), 0.0);
    float bias = max(shadowBiasSlope * (1.0 - ndotl), shadowBiasMin);

    float shadow = 0.0;
    vec2 texelSize = 1.0 / vec2(textureSize(shadowMap, 0));
    for (int x = -1; x <= 1; ++x) {
        for (int y = -1; y <= 1; ++y) {
            float pcfDepth = texture(shadowMap, projCoords.xy + vec2(x, y) * texelSize).r;
            shadow += (currentDepth - bias > pcfDepth) ? 1.0 : 0.0;
        }
    }
    shadow = (shadow / 9.0) * clamp(shadowStrength, 0.0, 1.0);
    return shadow;
}

float computeFogFactor(
    float viewDistance,
    float depthValue,
    float underwaterDepth,
    float cameraDepth,
    float underwaterMask
) {
    float amount = saturate(fogAmount);
    if (amount <= 0.0001) {
        return 0.0;
    }

    float baseDensity = max(fogStrength, 0.0001);
    float fogFactor = smoothstep(fogRange.x, fogRange.y, depthValue) * baseDensity;

    // fogRange.y > 1.5 трактуем как "новый" режим диапазона в метрах (distance fog).
    if (fogRange.y > 1.5) {
        float fogNear = max(fogRange.x, 0.0);
        float fogFar = max(fogRange.y, fogNear + 0.01);
        float fogWindow = max(fogFar - fogNear, 0.0001);
        float normalizedDistance = max(viewDistance - fogNear, 0.0) / fogWindow;

        float linearFog = saturate(normalizedDistance);
        float exponentialFog = 1.0 - exp(-normalizedDistance * baseDensity * 3.2);
        fogFactor = mix(linearFog, exponentialFog, 0.65);

        if (underwaterMask > 0.001) {
            float visible = max(visibilityDistance, 1.0);
            float uwDensity = max(underwaterFogDensity, 0.0) * baseDensity;
            float uwDistanceFog = 1.0 - exp(-(viewDistance / visible) * uwDensity);
            float depthPath = underwaterDepth * 0.85 + cameraDepth * 0.30;
            float uwDepthFog = 1.0 - exp(-depthPath * uwDensity * 0.032);
            float underwaterFog = saturate(max(uwDistanceFog, uwDistanceFog * 0.55 + uwDepthFog * 0.45));
            fogFactor = mix(fogFactor, max(fogFactor, underwaterFog), underwaterMask);
        }
    }

    return saturate(fogFactor * amount);
}

vec3 calcLight(
    Light light,
    vec3 norm,
    vec3 viewDir,
    vec3 baseColor,
    float shadow,
    float underwaterDepth,
    float cameraDepth
) {
    vec3 lightDir;
    float attenuation = 1.0;

    if (light.type == LIGHT_TYPE_DIRECTIONAL) {
        lightDir = normalize(-light.direction);

        // Под водой направленный свет (солнце) теряет энергию по мере роста глубины.
        float submersion = smoothstep(0.0, 0.25, underwaterDepth + cameraDepth * 0.85) * saturate(underwaterBlend);
        if (submersion > 0.001) {
            float sunDepth = underwaterDepth + cameraDepth * 0.65;
            float sunFade = exp(-max(sunlightAttenuation, 0.0) * sunDepth);
            attenuation *= mix(1.0, sunFade, submersion);
        }
    } else {
        vec3 toLight = light.position - fragPos;
        float dist = length(toLight);
        if (dist <= 0.0001) {
            return vec3(0.0);
        }
        lightDir = toLight / dist;

        float denom = light.constant + light.linear * dist + light.quadratic * dist * dist;
        attenuation = (denom > 0.0001) ? (1.0 / denom) : 0.0;

        if (light.type == LIGHT_TYPE_SPOT) {
            float theta = dot(lightDir, normalize(-light.direction));
            float epsilon = max(light.cutOff - light.outerCutOff, 0.0001);
            float cone = clamp((theta - light.outerCutOff) / epsilon, 0.0, 1.0);
            attenuation *= cone;
        }
    }

    float diff = max(dot(norm, lightDir), 0.0);
    vec3 diffuse = diff * light.color * light.intensity * material.diffuse * baseColor * attenuation * (1.0 - shadow);

    float specularStrength = max(material.specularStrength, 0.0);
    float spec = 0.0;
    if (diff > 0.0 && specularStrength > 0.0001) {
        vec3 halfDir = normalize(lightDir + viewDir);
        float shininess = max(material.shininess, 1.0);
        spec = pow(max(dot(norm, halfDir), 0.0), shininess);
    }
    vec3 specular = spec * light.color * light.intensity * material.specular * specularStrength * attenuation * (1.0 - shadow);

    return diffuse + specular;
}

void main() {
    float t = clamp((vHeight - heightRange.x) / max(heightRange.y - heightRange.x, 0.0001), 0.0, 1.0);
    vec3 sandColor = mix(lowColor, highColor, t);

    vec3 norm = normalize(normal);
    if (length(norm) < 0.0001) {
        norm = vec3(0.0, 1.0, 0.0);
    }

    float terrainMask = clamp(heightTint, 0.0, 1.0);
    float slope = clamp(1.0 - max(dot(norm, vec3(0.0, 1.0, 0.0)), 0.0), 0.0, 1.0);
    float macroNoise = noise(fragPos.xz * 0.028 + vec2(19.7, -7.3));
    float detailNoise = noise(fragPos.xz * 0.19 + vec2(-41.2, 63.8));
    float albedoVariation = (macroNoise - 0.5) * 0.28 + (detailNoise - 0.5) * 0.12;

    vec3 terrainColor = sandColor * (1.0 + albedoVariation);
    vec3 slopeTint = mix(vec3(1.04, 1.01, 0.96), vec3(0.74, 0.79, 0.83), slope);
    terrainColor *= slopeTint;
    terrainColor = max(terrainColor, vec3(0.0));

    vec3 baseColor = mix(objectColor, terrainColor, terrainMask);
    vec3 viewDir = normalize(viewPos - fragPos);
    float viewDistance = length(viewPos - fragPos);
    float depthScale = clamp(underwaterDepthScale, 0.0, 1.0);
    float underwaterDepth = max(waterLevel - fragPos.y, 0.0) * depthScale;
    float cameraDepth = max(waterLevel - viewPos.y, 0.0) * depthScale;
    float underwaterMask = saturate(underwaterBlend) * smoothstep(0.0, 0.30, underwaterDepth + cameraDepth);

    vec3 ambient = material.ambient * baseColor * ambientLight.color * ambientLight.intensity;
    vec3 result = ambient + material.emission * baseColor;

    int lightCount = min(numLights, MAX_LIGHTS);
    int activeShadowIndex = shadowLightIndex;
    if (activeShadowIndex < 0 ||
        activeShadowIndex >= lightCount ||
        lights[activeShadowIndex].type != LIGHT_TYPE_DIRECTIONAL) {
        activeShadowIndex = -1;
    }

    float activeDirectionalShadow = 0.0;
    vec3 localLightAccum = vec3(0.0);
    for (int i = 0; i < lightCount; i++) {
        float shadow = 0.0;
        if (i == activeShadowIndex) {
            shadow = shadowCalculation(fragPosLightSpace, norm, normalize(-lights[i].direction));
            activeDirectionalShadow = shadow;
        }
        vec3 lightContribution = calcLight(lights[i], norm, viewDir, baseColor, shadow, underwaterDepth, cameraDepth);
        result += lightContribution;
        if (lights[i].type == LIGHT_TYPE_POINT || lights[i].type == LIGHT_TYPE_SPOT) {
            localLightAccum += lightContribution;
        }
    }

    int primaryDirectionalIndex = activeShadowIndex;
    if (primaryDirectionalIndex < 0) {
        for (int i = 0; i < lightCount; i++) {
            if (lights[i].type == LIGHT_TYPE_DIRECTIONAL) {
                primaryDirectionalIndex = i;
                break;
            }
        }
    }

    vec3 causticSunDir = vec3(0.0, -1.0, 0.0);
    vec3 causticSunColor = vec3(1.0, 0.95, 0.86);
    float causticSunIntensity = 0.0;
    float causticSunShadow = 0.0;
    if (primaryDirectionalIndex >= 0) {
        causticSunDir = lights[primaryDirectionalIndex].direction;
        if (length(causticSunDir) < 0.0001) {
            causticSunDir = vec3(0.0, -1.0, 0.0);
        } else {
            causticSunDir = normalize(causticSunDir);
        }
        causticSunColor = max(lights[primaryDirectionalIndex].color, vec3(0.0));
        causticSunIntensity = max(lights[primaryDirectionalIndex].intensity, 0.0);
        if (primaryDirectionalIndex == activeShadowIndex) {
            causticSunShadow = activeDirectionalShadow;
        }
    }

    if (underwaterDepth > 0.001 && causticsIntensity > 0.0001 && causticSunIntensity > 0.0001) {
        // Русский комментарий: проецируем паттерн в точку пересечения солнечного луча с поверхностью воды,
        // чтобы каустика ехала по дну и объектам согласованно с направлением солнца.
        float sunRayY = max(-causticSunDir.y, 0.08);
        float distanceToSurface = max((waterLevel - fragPos.y) / sunRayY, 0.0);
        vec3 projectedSurfacePos = fragPos - causticSunDir * distanceToSurface;

        float patternScale = max(causticsScale, 0.0001);
        float flowTime = time * max(causticsSpeed, 0.0);

        vec2 sunFlowBase = vec2(-causticSunDir.x, -causticSunDir.z);
        if (dot(sunFlowBase, sunFlowBase) < 0.000001) {
            sunFlowBase = vec2(0.002, -0.001);
        }
        vec2 sunFlow = normalize(sunFlowBase);
        vec2 crossFlow = vec2(-sunFlow.y, sunFlow.x);

        vec2 cuv0 = projectedSurfacePos.xz * (patternScale * 0.58) + sunFlow * (flowTime * 0.52);
        vec2 cuv1 = projectedSurfacePos.xz * (patternScale * 0.84) - crossFlow * (flowTime * 0.60);
        vec2 cuvMacro = projectedSurfacePos.xz * (patternScale * 0.26) +
            (sunFlow + crossFlow * 0.30) * (flowTime * 0.18);
        float c0 = texture(causticTex0, cuv0).r;
        float c1 = texture(causticTex1, cuv1).r;
        float cMacro = texture(causticTex0, cuvMacro).r;

        // Русский комментарий: смещаем рисунок к крупным плавным "волнам" света:
        // текстура задаёт органику, а синусоидальные полосы — широкую текущую форму без точечной ряби.
        vec2 waveAxisA = normalize(sunFlow * 0.82 + crossFlow * 0.18);
        vec2 waveAxisB = normalize(crossFlow * 0.74 - sunFlow * 0.26);
        vec2 wavePos = projectedSurfacePos.xz * (patternScale * 0.21);
        float wavePhaseA = dot(wavePos, waveAxisA) * 6.2831853 + flowTime * 0.42;
        float wavePhaseB = dot(wavePos, waveAxisB) * 6.2831853 - flowTime * 0.31;
        float waveBands = 0.5 + 0.5 * sin(wavePhaseA + 0.55 * sin(wavePhaseB));
        float waveMask = smoothstep(0.43, 0.90, waveBands);

        float broadPattern = mix(c0, c1, 0.50);
        float macroEnvelope = smoothstep(0.20, 0.88, cMacro);
        float causticPattern = mix(broadPattern, waveMask, 0.56);
        causticPattern *= mix(0.84, 1.18, macroEnvelope);
        float contrast = max(causticsContrast, 0.1);
        float artisticContrast = 0.62 + contrast * 0.34;
        causticPattern = saturate((causticPattern - 0.50) * artisticContrast + 0.50);
        causticPattern = pow(causticPattern, 1.12);

        float fade = max(causticsDepthFade, 0.0);
        // Русский комментарий: дополнительно удлиняем глубинный хвост,
        // чтобы каустика не исчезала до достижения морского дна в глубоких зонах.
        float depthPath = underwaterDepth + cameraDepth * 0.14;
        float depthExtinction = exp(-depthPath * fade * 0.18);
        float depthAttenuation = mix(0.46, 1.0, depthExtinction);

        // Русский комментарий: мутность должна ослаблять, но не "обнулять" паттерн на большой глубине.
        float murkyPath = underwaterDepth + cameraDepth * 0.20;
        float murkyExtinction = exp(-max(underwaterFogDensity, 0.0) * murkyPath * 0.010);
        float murkyFade = mix(0.62, 1.0, murkyExtinction);
        float sunElevation = saturate((-causticSunDir.y - 0.04) / 0.96);
        float sunVisibility = (1.0 - causticSunShadow) * (0.35 + sunElevation * 0.65);

        float lightFacing = max(dot(norm, normalize(-causticSunDir)), 0.0);
        float receiveMask = smoothstep(0.05, 0.85, lightFacing);
        float shallowBoost = 1.0 + exp(-underwaterDepth * 0.085) * 0.34;
        float deepResidualBlend = smoothstep(8.0, 95.0, underwaterDepth + cameraDepth * 0.16);
        float deepResidual = deepResidualBlend * 0.22;
        float bottomReachBoost = 1.0 + deepResidualBlend * 0.18;
        float causticVisibility = max(depthAttenuation * murkyFade, deepResidual);
        float causticMask = underwaterMask * causticVisibility * receiveMask * sunVisibility * shallowBoost * bottomReachBoost;
        causticMask = saturate(causticMask);

        vec3 causticTint = mix(vec3(0.50, 0.58, 0.64), causticSunColor, 0.58);
        result += causticTint * causticPattern * causticsIntensity * causticSunIntensity * causticMask;
    }

    // Упрощенное спектральное поглощение воды: красный канал гаснет быстрее.
    float waterPath = max(underwaterDepth + cameraDepth * 0.72 + viewDistance * 0.08, 0.0);
    float attenuationStrength = 0.25 + max(sunlightAttenuation, 0.0) * 0.55;
    vec3 spectralAbsorption = vec3(0.34, 0.12, 0.05) * attenuationStrength;
    vec3 transmittance = exp(-spectralAbsorption * waterPath);
    vec3 attenuatedResult = mix(result, result * transmittance, underwaterMask);
    vec3 localVisibleLight = localLightAccum;
    if (underwaterMask > 0.001) {
        float localWaterPath = max(underwaterDepth * 0.42 + cameraDepth * 0.20 + viewDistance * 0.05, 0.0);
        vec3 localTransmittance = exp(-spectralAbsorption * localWaterPath * 0.55);
        vec3 localRecovered = localLightAccum * localTransmittance;
        vec3 localGlobalAttenuated = localLightAccum * transmittance;
        attenuatedResult += (localRecovered - localGlobalAttenuated) * underwaterMask;
        localVisibleLight = localRecovered;
    }
    float tintFactor = underwaterMask * saturate((1.0 - transmittance.r) * 1.1);
    attenuatedResult = mix(attenuatedResult, attenuatedResult * depthTint, tintFactor);

    float fogFactor = computeFogFactor(viewDistance, gl_FragCoord.z, underwaterDepth, cameraDepth, underwaterMask);
    if (underwaterMask > 0.001) {
        float localLuminance = dot(max(localVisibleLight, vec3(0.0)), vec3(0.299, 0.587, 0.114));
        float localProximity = exp(-viewDistance * 0.045);
        float localFogRelief = saturate(localLuminance * 0.85) * localProximity;
        fogFactor *= (1.0 - localFogRelief * 0.55 * underwaterMask);
    }
    vec3 finalColor = mix(attenuatedResult, fogColor, fogFactor);
    frag_colour = vec4(finalColor, 1.0);
}
