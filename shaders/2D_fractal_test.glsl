#version 330 core

#ifdef GL_ES
precision highp float;
#endif

out vec4 FragColor;

in vec2 fragTexCoord;

uniform vec2 iResolution;
uniform float iTime;
uniform vec3 camPos;        // camera position (optional)
uniform mat4 camera;        // Camera/view matrix
uniform mat4 view;          // view matrix (optional)
uniform mat4 projection;    // projection matrix (optional)

// Mandelbulb Distance Estimator (DE)
float mandelbulbDE(vec3 pos) {
    vec3 z = pos;
    float dr = 1.0;
    float r = 0.0;

    const int ITERATIONS = 10;

    // Animate power folding between 6 and 8 (folding in and out)
    float power = 6.0 + 2.0 * sin(iTime * 0.5);

    for (int i = 0; i < ITERATIONS; i++) {
        r = length(z);
        if (r > 2.0) break;

        // Spherical coords
        float theta = acos(z.z / r);
        float phi = atan(z.y, z.x);
        dr = pow(r, power - 1.0) * power * dr + 1.0;

        // Fold angles with animation for extra movement
        theta = theta * power + 0.3 * sin(iTime + float(i));
        phi = phi * power + 0.3 * cos(iTime + float(i));

        // Convert back to Cartesian coords
        float zr = pow(r, power);
        z = zr * vec3(
            sin(theta) * cos(phi),
            sin(phi) * sin(theta),
            cos(theta)
        ) + pos;
    }

    return 0.5 * log(r) * r / dr;
}

// Ray marching function
float rayMarch(vec3 ro, vec3 rd) {
    float totalDist = 0.0;
    const float MAX_DIST = 100.0;
    const float EPSILON = 0.001;
    const int MAX_STEPS = 100;

    for (int i = 0; i < MAX_STEPS; i++) {
        vec3 p = ro + totalDist * rd;
        float dist = mandelbulbDE(p);

        if (dist < EPSILON) return totalDist;
        totalDist += dist;
        if (totalDist > MAX_DIST) break;
    }
    return -1.0;
}

// Estimate normal by gradient of DE function
vec3 estimateNormal(vec3 p) {
    const float eps = 0.001;
    return normalize(vec3(
        mandelbulbDE(p + vec3(eps, 0.0, 0.0)) - mandelbulbDE(p - vec3(eps, 0.0, 0.0)),
        mandelbulbDE(p + vec3(0.0, eps, 0.0)) - mandelbulbDE(p - vec3(0.0, eps, 0.0)),
        mandelbulbDE(p + vec3(0.0, 0.0, eps)) - mandelbulbDE(p - vec3(0.0, 0.0, eps))
    ));
}

void main() {
    
    vec2 uv = fragTexCoord * 2.0 - 1.0;
    uv.x *= iResolution.x / iResolution.y;
    
    vec3 ro = camPos;

    vec4 rayClip = vec4(uv, -1.0, 1.0);

    vec3 rd_view = normalize(vec3(uv, -1.0));  // ray direction in view space
    vec3 rd = normalize((inverse(camera) * vec4(rd_view, 0.0)).xyz);

    // If you have view/projection matrices, transform rd accordingly
    // e.g. rd = normalize((inverse(view) * vec4(rd,0.0)).xyz);

    float dist = rayMarch(ro, rd);

    vec3 col = vec3(0.0);
    if (dist > 0.0) {
        vec3 p = ro + dist * rd;
        vec3 normal = estimateNormal(p);

        // Simple lighting
        vec3 lightDir = normalize(vec3(1.0, 1.0, -1.0));
        float diff = clamp(dot(normal, lightDir), 0.0, 1.0);
        col = normal;
    }

    FragColor = vec4(col, 1.0);
}
