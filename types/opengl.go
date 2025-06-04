package types

import (
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	Client "VoxelRPG/client"
	Log "VoxelRPG/logging"
	World "VoxelRPG/world"
)

var (
	screenVAO           uint32
	shaderProgram       uint32
	screenShaderProgram uint32

	fbo        uint32
	fboTexture uint32

	FOV   float32
	ZNear float32
	ZFar  float32

	Scaledown float32
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

func setupShaders() {

	/* --[[ Main Screen Shader ]] */

	vertex_shader, err := NewShader("octree_traverse.vert", gl.VERTEX_SHADER)
	CheckError(err)

	fragment_shader, err := NewShader("octree_traverse.frag", gl.FRAGMENT_SHADER)
	CheckError(err)

	shaders := []uint32{
		vertex_shader,
		fragment_shader,
	}

	shaderProgram, err = NewShaderProgram(shaders)
	CheckError(err)

	/* --[[ Upscaled Texture Shader ]] */

	vertex_shader, err = NewShader("screen.vert", gl.VERTEX_SHADER)
	CheckError(err)

	fragment_shader, err = NewShader("screen.frag", gl.FRAGMENT_SHADER)
	CheckError(err)

	shaders = []uint32{
		vertex_shader,
		fragment_shader,
	}

	screenShaderProgram, err = NewShaderProgram(shaders)
	CheckError(err)

}

func setupBuffers(window *WindowBuilder) {

	/* --[[ Create Shaders ]] */

	setupShaders()

	/* --[[ Create VAO (Vertex Array Object) ]] */

	gl.UseProgram(shaderProgram)

	screenVAO = NewVertexArray(1)

	NewBufferObject(1, gl.ARRAY_BUFFER, func(_ uint32) {
		gl.BufferData(gl.ARRAY_BUFFER, len(fullscreenQuadVertices)*4, gl.Ptr(fullscreenQuadVertices), gl.STATIC_DRAW)
	})

	resUniform := gl.GetUniformLocation(shaderProgram, gl.Str("iResolution\x00"))
	gl.Uniform2f(resUniform, float32(window.Width), float32(window.Height))

	vertAttrib := uint32(gl.GetAttribLocation(shaderProgram, gl.Str("vert\x00")))
	gl.VertexAttribPointerWithOffset(vertAttrib, 2, gl.FLOAT, false, 4*4, 0)
	gl.EnableVertexAttribArray(vertAttrib)

	// Set offset from our vertex buffer object to allow it to read verticies correctly

	texCoordAttrib := uint32(gl.GetAttribLocation(shaderProgram, gl.Str("vertTexCoord\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointerWithOffset(texCoordAttrib, 2, gl.FLOAT, false, 4*4, uintptr(2*4))

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)

	/* --[[ Setup Octtree ]] */

	World.MainWorld.Populate(shaderProgram)
	World.MainWorld.Update(shaderProgram)

	Log.NewLog("Total Voxel Count:", ((World.RENDER_DISTANCE * World.RENDER_DISTANCE * World.RENDER_DISTANCE) * (World.CHUNK_SIZE * World.CHUNK_SIZE * World.CHUNK_SIZE)))

	/* --[[ Frame Buffer Object for rendering world at low resolutions ]] */

	Scaledown = 1.5

	gl.UseProgram(screenShaderProgram)

	gl.GenFramebuffers(1, &fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)

	gl.GenTextures(1, &fboTexture)
	gl.BindTexture(gl.TEXTURE_2D, fboTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(float32(window.Width)/Scaledown), int32(float32(window.Height)/Scaledown), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fboTexture, 0)

	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		Log.NewLog("ERROR: Framebuffer is not complete")
	}

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

}

func OpenGLSetup(window *WindowBuilder, client *Client.ClientContext) {

	Log.NewLog("OpenGL - Setup...")

	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0.3, 0.3, 0.3, 1.0)

	setupBuffers(window)

	/* --[[ Setup Perspective ]] */

	setupPerspectives(window, client)
	OnWindowResize(glfw.GetCurrentContext(), window.Width, window.Height, window)

	Log.NewLog("OpenGL Setup - Complete\n")

}

func ResizeFramebuffer(fbo uint32, texture *uint32, width, height int) {
	scaledWidth := int32(float32(width) / Scaledown)
	scaledHeight := int32(float32(height) / Scaledown)

	// Reallocate texture storage
	gl.BindTexture(gl.TEXTURE_2D, *texture)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA8,
		scaledWidth,
		scaledHeight,
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		nil,
	)

	// Reattach texture to framebuffer
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.FramebufferTexture2D(
		gl.FRAMEBUFFER,
		gl.COLOR_ATTACHMENT0,
		gl.TEXTURE_2D,
		*texture,
		0,
	)

	// Check FBO completeness
	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		Log.NewLog("ERROR: Framebuffer is not complete after resize!")
	}

	// Unbind to avoid side effects
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

func OnWindowResize(w *glfw.Window, width int, height int, wBuild *WindowBuilder) {

	// Change OpenGL viewport to the new window size

	Log.NewLog("New Size - W:", width, "H:", height, "Aspect:", float32(wBuild.Width)/float32(wBuild.Height))

	gl.Viewport(0, 0, int32(width), int32(height))

	wBuild.Width = width
	wBuild.Height = height

	ResizeFramebuffer(fbo, &fboTexture, width, height)
}

func OpenGLFixedUpdate(window *glfw.Window, windowBuilder *WindowBuilder) {

	// Always runs every 1/165 of a second

	W, H := window.GetSize()

	if windowBuilder.Width != W || windowBuilder.Height != H {

		OnWindowResize(window, W, H, windowBuilder)

	}

}

func OpenGLUpdate(cam *Client.Camera, windowBuilder *WindowBuilder) {

	Log.NewLog("Camera Pos:", cam.Pos, "Chunk Pos:", World.GetCameraChunk(cam.Pos))

	// === First Pass: Render raymarcher to half-resolution FBO ===

	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.Viewport(0, 0, int32(float32(windowBuilder.Width)/Scaledown), int32(float32(windowBuilder.Height)/Scaledown))
	gl.UseProgram(shaderProgram)
	gl.BindVertexArray(screenVAO)

	// === Camera Setup ===

	view := mgl32.LookAtV(
		cam.Pos,
		mgl32.Vec3{cam.Pos[0] + cam.Front[0], cam.Pos[1] + cam.Front[1], cam.Pos[2] + cam.Front[2]},
		mgl32.Vec3{0, 1, 0},
	)
	invView := view.Inv()

	projection := mgl32.Perspective(mgl32.DegToRad(FOV), float32(windowBuilder.Width)/Scaledown/float32(windowBuilder.Height)/Scaledown, ZNear, ZFar)
	viewProjection := projection.Mul4(view)

	// === Uniform Uploads ===

	loc := gl.GetUniformLocation(shaderProgram, gl.Str("invView\x00"))

	gl.UniformMatrix4fv(loc, 1, false, &invView[0])
	gl.UniformMatrix4fv(gl.GetUniformLocation(shaderProgram, gl.Str("projection\x00")), 1, false, &projection[0])
	gl.Uniform3f(gl.GetUniformLocation(shaderProgram, gl.Str("camPos\x00")), cam.Pos[0], cam.Pos[1], cam.Pos[2])
	gl.Uniform1f(gl.GetUniformLocation(shaderProgram, gl.Str("iTime\x00")), float32(glfw.GetTime()))
	gl.Uniform2f(gl.GetUniformLocation(shaderProgram, gl.Str("iResolution\x00")), float32(windowBuilder.Width)/float32(Scaledown), float32(windowBuilder.Height)/float32(Scaledown))

	// === Bind SSBO ===

	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 0, World.MainWorld.CombinedSSBO)

	// === Update World if required ===

	World.MainWorld.UpdateIfNeeded(shaderProgram, viewProjection, cam.Pos)

	// === Draw Fullscreen Quad ===

	gl.DrawArrays(gl.TRIANGLES, 0, 6)

	// === Second Pass: Blot to full-resolution screen ===

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, int32(windowBuilder.Width), int32(windowBuilder.Height))
	gl.UseProgram(screenShaderProgram)
	gl.BindVertexArray(screenVAO)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, fboTexture)
	gl.Uniform1i(gl.GetUniformLocation(screenShaderProgram, gl.Str("tex\x00")), 0)

	gl.DrawArrays(gl.TRIANGLES, 0, 6)

}
