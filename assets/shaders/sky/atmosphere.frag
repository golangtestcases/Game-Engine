#version 410

in vec2 vUV;

uniform mat4 uInvProjection;
uniform mat4 uInvView;
uniform vec3 uSunDirection;
uniform vec3 uSunColor;
uniform float uSunIntensity;
uniform float uSunDiscSize;
uniform float uSunHaloIntensity;
uniform vec3 uHorizonColor;
uniform vec3 uZenithColor;
uniform float uAtmosphereBlend;

out vec4 fragColor;

float saturate(float v) {
    return clamp(v, 0.0, 1.0);
}

vec3 viewRayWorld(vec2 uv) {
    vec4 clip = vec4(uv * 2.0 - 1.0, 1.0, 1.0);
    vec4 view = uInvProjection * clip;
    vec3 dirView = normalize(view.xyz / max(view.w, 0.0001));
    return normalize((uInvView * vec4(dirView, 0.0)).xyz);
}

void main() {
    vec3 ray = viewRayWorld(vUV);
    vec3 sunDir = normalize(-uSunDirection);

    float h = saturate(ray.y * 0.5 + 0.5);
    vec3 sky = mix(uHorizonColor, uZenithColor, pow(h, 0.62));

    float sunDot = clamp(dot(ray, sunDir), -1.0, 1.0);
    float sunAngle = acos(sunDot);

    float discInner = max(uSunDiscSize * 0.35, 0.0005);
    float disc = 1.0 - smoothstep(discInner, uSunDiscSize, sunAngle);

    float haloWidth = max(uSunDiscSize * 18.0, 0.015);
    float halo = exp(-sunAngle / haloWidth) * uSunHaloIntensity;

    float mie = pow(max(sunDot, 0.0), 9.0) * (0.5 + 0.5 * uAtmosphereBlend);
    float horizonScatter = pow(1.0 - h, 2.5) * (0.35 + 0.65 * uAtmosphereBlend);
    float sunHeight = saturate(sunDir.y * 0.5 + 0.5);
    vec3 warmHorizon = mix(vec3(1.0, 0.62, 0.34), uSunColor, 0.55) * horizonScatter * (1.0 - sunHeight);

    vec3 sunGlow = uSunColor * uSunIntensity * (disc * 6.5 + halo * 1.1 + mie * 0.55);
    vec3 finalColor = sky + warmHorizon + sunGlow;

    // Мягкий тонемап сохраняет яркое ядро солнца и не «ломает» плавность градиента.
    finalColor = 1.0 - exp(-finalColor);

    fragColor = vec4(finalColor, 1.0);
}
