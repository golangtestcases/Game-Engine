#version 410
layout (location = 0) in vec3 vp;
layout (location = 1) in vec3 vn;

uniform mat4 MVP;
uniform mat4 model;
uniform float time;
uniform float waveAmplitude;
uniform float waveFrequency;
uniform float waveSpeed;

out vec3 fragPos;
out vec3 normal;
out float waveHeight;
out float waveFoam;

vec3 gerstnerWave(vec3 pos, vec2 direction, float amplitude, float wavelength, float speed, float steepness) {
    float k = 2.0 * 3.14159265 / max(wavelength, 0.001);
    float c = sqrt(9.81 / k);
    vec2 d = normalize(direction);
    float phase = k * (dot(d, pos.xz) - c * speed * time);
    float a = steepness / k;
    return vec3(
        d.x * a * cos(phase),
        amplitude * sin(phase),
        d.y * a * cos(phase)
    );
}

vec3 gerstnerNormal(vec3 pos, vec2 direction, float amplitude, float wavelength, float speed, float steepness) {
    float k = 2.0 * 3.14159265 / max(wavelength, 0.001);
    float c = sqrt(9.81 / k);
    vec2 d = normalize(direction);
    float phase = k * (dot(d, pos.xz) - c * speed * time);
    float wa = k * amplitude;
    return vec3(
        -d.x * wa * cos(phase),
        1.0 - steepness * wa * sin(phase),
        -d.y * wa * cos(phase)
    );
}

void main() {
    vec3 pos = vp;
    float wf = max(waveFrequency, 0.25);
    float ws = max(waveSpeed, 0.05);

    // Многодиапазонный спектр направленных волн: зыбь + средняя рябь + мелкая капиллярная.
    vec3 w1 = gerstnerWave(pos, vec2(1.00, 0.12), waveAmplitude * 0.36, 22.0 / wf, ws * 0.45, 0.62);
    vec3 w2 = gerstnerWave(pos, vec2(0.82, 0.58), waveAmplitude * 0.28, 15.0 / wf, ws * 0.65, 0.55);
    vec3 w3 = gerstnerWave(pos, vec2(-0.36, 0.93), waveAmplitude * 0.22, 10.5 / wf, ws * 0.95, 0.50);
    vec3 w4 = gerstnerWave(pos, vec2(-0.94, 0.21), waveAmplitude * 0.18, 7.8 / wf, ws * 1.20, 0.44);
    vec3 w5 = gerstnerWave(pos, vec2(0.26, -0.97), waveAmplitude * 0.15, 5.3 / wf, ws * 1.45, 0.38);
    vec3 w6 = gerstnerWave(pos, vec2(0.64, -0.77), waveAmplitude * 0.12, 3.6 / wf, ws * 1.85, 0.30);
    vec3 w7 = gerstnerWave(pos, vec2(-0.13, -0.99), waveAmplitude * 0.09, 2.8 / wf, ws * 2.25, 0.25);
    vec3 w8 = gerstnerWave(pos, vec2(-0.74, -0.67), waveAmplitude * 0.06, 2.1 / wf, ws * 2.70, 0.20);

    pos += w1 + w2 + w3 + w4 + w5 + w6 + w7 + w8;
    waveHeight = pos.y;

    vec3 n1 = gerstnerNormal(vp, vec2(1.00, 0.12), waveAmplitude * 0.36, 22.0 / wf, ws * 0.45, 0.62);
    vec3 n2 = gerstnerNormal(vp, vec2(0.82, 0.58), waveAmplitude * 0.28, 15.0 / wf, ws * 0.65, 0.55);
    vec3 n3 = gerstnerNormal(vp, vec2(-0.36, 0.93), waveAmplitude * 0.22, 10.5 / wf, ws * 0.95, 0.50);
    vec3 n4 = gerstnerNormal(vp, vec2(-0.94, 0.21), waveAmplitude * 0.18, 7.8 / wf, ws * 1.20, 0.44);
    vec3 n5 = gerstnerNormal(vp, vec2(0.26, -0.97), waveAmplitude * 0.15, 5.3 / wf, ws * 1.45, 0.38);
    vec3 n6 = gerstnerNormal(vp, vec2(0.64, -0.77), waveAmplitude * 0.12, 3.6 / wf, ws * 1.85, 0.30);
    vec3 n7 = gerstnerNormal(vp, vec2(-0.13, -0.99), waveAmplitude * 0.09, 2.8 / wf, ws * 2.25, 0.25);
    vec3 n8 = gerstnerNormal(vp, vec2(-0.74, -0.67), waveAmplitude * 0.06, 2.1 / wf, ws * 2.70, 0.20);

    vec3 waveNormal = normalize(n1 + n2 + n3 + n4 + n5 + n6 + n7 + n8);
    float slope = 1.0 - max(waveNormal.y, 0.0);
    waveFoam = clamp(pow(slope, 1.35) * 2.4 + max(pos.y, 0.0) * 3.4, 0.0, 1.0);

    vec4 worldPos = model * vec4(pos, 1.0);
    fragPos = vec3(worldPos);
    normal = normalize(mat3(transpose(inverse(model))) * waveNormal);
    gl_Position = MVP * vec4(pos, 1.0);
}
