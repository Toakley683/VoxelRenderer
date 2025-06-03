#version 460 core

layout(location = 0) in vec2 vert; // vec2 for fullscreen quad

in vec2 vertTexCoord;

out vec2 fragTexCoord;

void main() {
    fragTexCoord = vertTexCoord;
    gl_Position = vec4(vert, 0.0, 1.0);
}