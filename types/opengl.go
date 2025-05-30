package types

import (
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	Client "VoxelRPG/client"
	Log "VoxelRPG/logging"
)

var (
	vao           uint32
	shaderProgram uint32
	texture       uint32

	FOV   float32
	ZNear float32
	ZFar  float32
)

func NewGLContext() error {

	profiler := ProfilerStart("opengl")

	Log.NewLog("OpenGL(Glow) Context - Creating..")

	err := gl.Init()

	if err != nil {
		Log.NewLog("Could not initialize OpenGL")
		return err
	}

	TimeTook := profiler.EndProfiler(profiler)

	Log.NewLog("OpenGL(Glow) Context - Created ( Time:", TimeTook, ")\n")

	return nil

}

func setupPerspectives(window *WindowBuilder, client *Client.ClientContext) {

	FOV = 90
	ZNear = 0.1
	ZFar = 1000.0

	projection := mgl32.Perspective(mgl32.DegToRad(FOV), float32(window.Width)/float32(window.Height), ZNear, ZFar)
	projectionUniform := gl.GetUniformLocation(shaderProgram, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projectionUniform, 1, false, &projection[0])

	cam := client.Camera

	camera := mgl32.LookAtV(
		mgl32.Vec3{cam.Pos[0], cam.Pos[1], cam.Pos[2]},
		mgl32.Vec3{cam.Pos[0] + cam.Front[0], cam.Pos[1] + cam.Front[1], cam.Pos[2] + cam.Front[2]},
		mgl32.Vec3{0, 1, 0},
	)

	cameraUniform := gl.GetUniformLocation(shaderProgram, gl.Str("camera\x00"))
	gl.UniformMatrix4fv(cameraUniform, 1, false, &camera[0])
	model := mgl32.Ident4()
	modelUniform := gl.GetUniformLocation(shaderProgram, gl.Str("model\x00"))
	gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])

}

func setupBuffers() {

	var err error

	/* --[[ Create Shaders ]] */

	// Create Vertex Shader

	vertex_shader, err := NewShader("vertex_shader.glsl", gl.VERTEX_SHADER)
	CheckError(err)

	// Create Fragment Shader

	fragment_shader, err := NewShader("fragment_shader.glsl", gl.FRAGMENT_SHADER)
	CheckError(err)

	shaders := []uint32{
		vertex_shader,
		fragment_shader,
	}

	shaderProgram, err = NewShaderProgram(shaders)
	CheckError(err)

	gl.UseProgram(shaderProgram)

	textureUniform := gl.GetUniformLocation(shaderProgram, gl.Str("tex\x00"))
	gl.Uniform1i(textureUniform, 0)

	gl.BindFragDataLocation(shaderProgram, 0, gl.Str("outputColor\x00"))

	texture, err = NewTexture("square.png")
	CheckError(err)

	/* --[[ Create VAO (Vertex Array Object) ]] */

	vao = NewVertexArray(1)

	/* --[[ Create Buffers ]] */

	// Make static buffer object with cube verticies

	NewBufferObject(1, gl.ARRAY_BUFFER, func() {
		gl.BufferData(gl.ARRAY_BUFFER, len(cubeVertices)*4, gl.Ptr(cubeVertices), gl.STATIC_DRAW)
	})

	// Set offset from our vertex buffer object to allow it to read verticies correctly

	vertAttrib := uint32(gl.GetAttribLocation(shaderProgram, gl.Str("vert\x00")))
	gl.VertexAttribPointerWithOffset(vertAttrib, 3, gl.FLOAT, false, 5*4, 0)
	gl.EnableVertexAttribArray(vertAttrib)

	texCoordAttrib := uint32(gl.GetAttribLocation(shaderProgram, gl.Str("vertTexCoord\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointerWithOffset(texCoordAttrib, 2, gl.FLOAT, false, 5*4, 3*4)

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)

}

func OpenGLSetup(window *WindowBuilder, client *Client.ClientContext) {

	Log.NewLog("OpenGL - Setup...")

	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0.3, 0.3, 0.3, 1.0)

	setupBuffers()

	/* --[[ Setup Perspective ]] */

	setupPerspectives(window, client)

	gl.Viewport(0, 0, int32(window.Width), int32(window.Height))

	Log.NewLog("OpenGL Setup - Complete\n")

}

func OnWindowResize(w *glfw.Window, width int, height int, wBuild *WindowBuilder) {

	// Change OpenGL viewport to the new window size

	Log.NewLog("New Size - W:", width, "H:", height)

	gl.Viewport(0, 0, int32(width), int32(height))

	wBuild.Width = width
	wBuild.Height = height

	projection := mgl32.Perspective(mgl32.DegToRad(FOV), float32(width)/float32(height), ZNear, ZFar)
	projectionUniform := gl.GetUniformLocation(shaderProgram, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projectionUniform, 1, false, &projection[0])

}

func OpenGLFixedUpdate(window *glfw.Window, windowBuilder *WindowBuilder) {

	// Always runs every 1/60 of a second

	W, H := window.GetSize()

	if windowBuilder.Width != W || windowBuilder.Height != H {

		OnWindowResize(window, W, H, windowBuilder)

	}

}

func OpenGLUpdate(cam *Client.Camera) {

	// Runs every frame

	gl.UseProgram(shaderProgram)
	gl.BindVertexArray(vao)

	// Update Camera Matrix

	view := mgl32.LookAtV(
		mgl32.Vec3{cam.Pos[0], cam.Pos[1], cam.Pos[2]},
		mgl32.Vec3{cam.Pos[0] + cam.Front[0], cam.Pos[1] + cam.Front[1], cam.Pos[2] + cam.Front[2]},
		mgl32.Vec3{0, 1, 0},
	)

	cameraUniform := gl.GetUniformLocation(shaderProgram, gl.Str("camera\x00"))
	gl.UniformMatrix4fv(cameraUniform, 1, false, &view[0])

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)

	gl.DrawArrays(gl.TRIANGLES, 0, 6*2*3)

}
