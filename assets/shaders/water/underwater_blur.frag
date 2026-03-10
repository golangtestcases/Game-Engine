#version 410

in vec2 vUV;

uniform sampler2D uInputTex;
uniform vec2 uTexelSize;
uniform vec2 uDirection;
uniform float uRadius;

out vec4 fragColor;

void main() {
    vec2 stepDir = uTexelSize * uDirection * uRadius;

    vec3 color = texture(uInputTex, vUV).rgb * 0.2270270270;
    color += texture(uInputTex, vUV + stepDir * 1.3846153846).rgb * 0.3162162162;
    color += texture(uInputTex, vUV - stepDir * 1.3846153846).rgb * 0.3162162162;
    color += texture(uInputTex, vUV + stepDir * 3.2307692308).rgb * 0.0702702703;
    color += texture(uInputTex, vUV - stepDir * 3.2307692308).rgb * 0.0702702703;

    fragColor = vec4(color, 1.0);
}
