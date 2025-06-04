package main

import (
	"runtime"

	ClientContext "VoxelRPG/client"
	Log "VoxelRPG/logging"
	Types "VoxelRPG/types"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	_ "net/http/pprof"
)

func main() {

	runtime.LockOSThread()

	WindowBuilder := &Types.WindowBuilder{
		Width:  800,
		Height: 600,
		Title:  "Voxel RPG",
	}

	window, err := Types.CreateWindow(WindowBuilder)
	Types.CheckError(err)

	go renderLoop(window)

	for !window.ShouldClose() {
		// WaitEventsTimeout waits max 100ms or until an event happens,
		// so the UI remains responsive during resize.
		glfw.WaitEventsTimeout(0.1)
	}

	glfw.Terminate()

}

func renderLoop(window *glfw.Window) {

	runtime.LockOSThread()
	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	W, H := window.GetSize()

	WindowBuilder := &Types.WindowBuilder{
		Width:  W,
		Height: H,
		Title:  "",
	}

	err := Types.NewGLContext()
	Types.CheckError(err)

	Client, err := ClientContext.NewClient()
	Types.CheckError(err)

	Types.OpenGLSetup(WindowBuilder, Client)

	version := gl.GoStr(gl.GetString(gl.VERSION))
	Log.NewLog("Loaded OpenGL - Version:", version, "\n")

	Log.NewLog("Events - Initializing..")

	Client.SetupKeybinds()

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		Types.WindowInputCB(Client, w, key, scancode, action, mods)
	})

	window.SetCursorPosCallback(func(w *glfw.Window, xpos float64, ypos float64) {
		Types.WindowMouseCB(Client, w, xpos, ypos)
	})

	Log.NewLog("Events - Initialized\n")

	if glfw.RawMouseMotionSupported() {
		window.SetInputMode(glfw.RawMouseMotion, glfw.True)
	}

	window.SetFocusCallback(func(w *glfw.Window, focused bool) {

		if !focused {
			w.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
		}

	})

	Log.NewLog("Program startup:")

	CheckDelta := float32(1) / float32(165)
	UpdateCheck := float64(0.0)

	//LastCh := glfw.GetTime()

	for !window.ShouldClose() {

		Now := glfw.GetTime()

		//Delta := Now - LastCh
		//LastCh = Now

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		if Now > UpdateCheck {

			UpdateCheck = Now + float64(CheckDelta)

			Types.OpenGLFixedUpdate(window, WindowBuilder)
			ClientContext.ClientCheckMovement(Client, CheckDelta)

			//Log.NewLog("FPS:", 1/Delta)

		}

		Types.OpenGLUpdate(Client.Camera, WindowBuilder)

		glfw.PollEvents()
		window.SwapBuffers()

	}

}
