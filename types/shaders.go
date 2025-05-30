package types

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-gl/gl/v4.6-core/gl"
)

func NewShaderData(shader_file_name string) (string, error) {

	shader_info, err := os.ReadFile("shaders/" + shader_file_name)

	if err != nil {

		return "", err

	}

	return string(shader_info) + "\x00", nil

}

func NewShader(shader_file_name string, shader_type uint32) (uint32, error) {

	shader := gl.CreateShader(shader_type)

	shaderSourceGoString, err := NewShaderData(shader_file_name)

	if err != nil {
		return 0, err
	}

	// Add source from shader file into shader object

	shaderSources, free := gl.Strs(shaderSourceGoString)
	gl.ShaderSource(shader, 1, shaderSources, nil)
	free()

	// Compile the shader

	gl.CompileShader(shader)

	// Check to make sure shader compiled correctly

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)

	if status == gl.FALSE {

		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile \n\n%v\nLog: %v", shaderSourceGoString, log)

	}

	return shader, nil

}

func NewShaderProgram(shaders []uint32) (uint32, error) {

	// Create shader program

	var shaderProgram uint32 = gl.CreateProgram()

	// Attach all shaders provided

	for i := 0; i < len(shaders); i++ {

		gl.AttachShader(shaderProgram, shaders[i])

	}

	// Link the program to context

	gl.LinkProgram(shaderProgram)

	// Check if there was an error linking to context

	var status int32
	gl.GetProgramiv(shaderProgram, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(shaderProgram, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(shaderProgram, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	// Cleanup the now unused shaders

	for i := 0; i < len(shaders); i++ {

		gl.DeleteShader(shaders[i])

	}

	return shaderProgram, nil

}
