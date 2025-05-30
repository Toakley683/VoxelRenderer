package types

import (
	"github.com/go-gl/gl/v4.6-core/gl"
)

func NewGLContext() error {

	profiler := ProfilerStart("opengl")

	NewLog("OpenGL(Glow) Context Creating..")

	err := gl.Init()

	if err != nil {
		NewLog("Could not initialize OpenGL")
		return err
	}

	TimeTook := profiler.EndProfiler(profiler)

	NewLog("OpenGL Context Created - Time:", TimeTook, "\n")

	version := gl.GoStr(gl.GetString(gl.VERSION))

	NewLog("Loaded OpenGL, version:", version, "\n")

	return nil

}
