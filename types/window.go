package types

import (
	Log "VoxelRPG/logging"

	"github.com/go-gl/glfw/v3.3/glfw"
)

type WindowBuilder struct {
	Width  int
	Height int
	Title  string
}

func CreateWindow(builder *WindowBuilder) (*glfw.Window, error) {

	Log.NewLog("Window - Creating..")

	profiler := ProfilerStart("glfw_window")

	// Initialize the GLFW context

	err := glfw.Init()

	if err != nil {
		Log.NewLog("Could not initialize GLFW")
		return nil, err
	}

	// Setup Window Settings

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.ScaleToMonitor, glfw.True)

	window, err := glfw.CreateWindow(builder.Width, builder.Height, builder.Title, nil, nil)

	if err != nil {
		Log.NewLog("Could not create window")
		glfw.Terminate()
		return nil, err

	}

	TimeTook := profiler.EndProfiler(profiler)

	Log.NewLog("Window - Created ( Time:", TimeTook, ")\n")

	return window, nil

}
