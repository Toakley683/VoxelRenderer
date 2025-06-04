#version 330 core

out vec2 TexCoords;
layout(location = 1) in vec2 vertTexCoord;

const vec2 verts[6] = vec2[](
    vec2(-1.0, -1.0),
    vec2( 1.0, -1.0),
    vec2(-1.0,  1.0),
    vec2(-1.0,  1.0),
    vec2( 1.0, -1.0),
    vec2( 1.0,  1.0)
);

void main() {
    gl_Position = vec4(verts[gl_VertexID], 0.0, 1.0);
    TexCoords = verts[gl_VertexID] * 0.5 + 0.5;
}
