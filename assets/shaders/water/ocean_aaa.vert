#version 410

const int WAVE_COUNT = 12;

layout (location = 0) in vec3 vp;
layout (location = 1) in vec3 vn;

uniform mat4 uModel;
uniform mat4 uView;
uniform mat4 uViewProj;
uniform float uTime;
uniform float uWaveTimeScale;
uniform vec4 uWaveData0[WAVE_COUNT]; // dir.xy, амплитуда, длина волны
uniform vec4 uWaveData1[WAVE_COUNT]; // скорость, крутизна, фаза, padding

out VS_OUT {
    vec3 worldPos;
    vec3 worldNormal;
    vec3 viewPos;
    float crest;
    float displacement;
} vs_out;

void main() {
    vec3 worldBase = (uModel * vec4(vp, 1.0)).xyz;
    vec3 displaced = worldBase;
    vec3 dPdx = vec3(1.0, 0.0, 0.0);
    vec3 dPdz = vec3(0.0, 0.0, 1.0);
    float crestAccum = 0.0;
    vec2 baseXZ = worldBase.xz;

    for (int i = 0; i < WAVE_COUNT; ++i) {
        vec2 dir = normalize(uWaveData0[i].xy);
        float amplitude = uWaveData0[i].z;
        float wavelength = max(uWaveData0[i].w, 0.001);
        float speed = uWaveData1[i].x;
        float steepness = uWaveData1[i].y;
        float phaseOffset = uWaveData1[i].z;

        float k = 6.28318530718 / wavelength;
        float c = sqrt(9.81 / k) * speed;
        float phase = k * dot(dir, baseXZ) - c * (uTime * uWaveTimeScale) + phaseOffset;
        float s = sin(phase);
        float cph = cos(phase);

        float qa = steepness * amplitude;
        displaced.x += dir.x * qa * cph;
        displaced.y += amplitude * s;
        displaced.z += dir.y * qa * cph;

        float wa = k * amplitude;
        dPdx += vec3(
            -dir.x * dir.x * steepness * wa * s,
            dir.x * wa * cph,
            -dir.x * dir.y * steepness * wa * s
        );
        dPdz += vec3(
            -dir.x * dir.y * steepness * wa * s,
            dir.y * wa * cph,
            -dir.y * dir.y * steepness * wa * s
        );

        crestAccum += max(0.0, wa * abs(cph) - 0.33);
    }

    vec3 worldNormal = normalize(cross(dPdz, dPdx));
    vec4 world = vec4(displaced, 1.0);
    vec4 view = uView * world;

    float slope = 1.0 - clamp(worldNormal.y, 0.0, 1.0);
    vs_out.crest = clamp(slope * 1.65 + crestAccum * 0.11, 0.0, 1.0);
    vs_out.worldPos = world.xyz;
    vs_out.worldNormal = worldNormal;
    vs_out.viewPos = view.xyz;
    vs_out.displacement = displaced.y;

    gl_Position = uViewProj * world;
}
