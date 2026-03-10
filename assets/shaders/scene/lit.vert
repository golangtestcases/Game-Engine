#version 410

uniform mat4 MVP;
uniform mat4 model;
uniform mat3 normalMatrix;
uniform mat4 lightSpaceMatrix;
uniform vec4 clipPlane;
uniform float clipEnabled;
uniform float time;
uniform float waterWaves;

layout (location = 0) in vec3 vp;
layout (location = 1) in vec3 vn;

out float vHeight;
out vec3 fragPos;
out vec3 normal;
out vec4 fragPosLightSpace;

void main() {
    vec3 pos = vp;
    vec3 norm = vn;

    // Волновая деформация нужна только для специальных поверхностей воды.
    if (waterWaves > 0.5) {
        float wave1 = sin(pos.x * 0.8 + time * 1.2) * cos(pos.z * 0.6 + time * 0.9) * 0.08;
        float wave2 = sin(pos.x * 1.3 - time * 0.7) * cos(pos.z * 1.1 - time * 1.1) * 0.05;
        pos.y += wave1 + wave2;

        float dx = cos(pos.x * 0.8 + time * 1.2) * 0.064 * 0.8;
        float dz = -sin(pos.z * 0.6 + time * 0.9) * 0.072 * 0.6;
        norm = normalize(vec3(-dx, 1.0, -dz));
    } else if (length(norm) < 0.0001) {
        // Защитный дефолт на случай отсутствующих нормалей.
        norm = vec3(0.0, 1.0, 0.0);
    }

    vHeight = pos.y;

    vec4 worldPos = model * vec4(pos, 1.0);
    fragPos = vec3(worldPos);
    normal = normalize(normalMatrix * norm);
    fragPosLightSpace = lightSpaceMatrix * worldPos;

    gl_ClipDistance[0] = (clipEnabled > 0.5) ? dot(worldPos, clipPlane) : 1.0;
    gl_Position = MVP * vec4(pos, 1.0);
}
