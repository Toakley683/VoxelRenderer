package main

import (
	"fmt"
	"log"
	"runtime"
	"runtime/debug"

	Types "VoxelRPG/types"

	"github.com/go-gl/glfw/v3.3/glfw"
)

func CheckError(err error) {

	if err != nil {
		fmt.Println(string(debug.Stack()))
		log.Fatalln("Error has occured:", err.Error())
	}

}

func main() {

	runtime.LockOSThread()

	WindowBuilder := &Types.WindowBuilder{
		Width:  1920,
		Height: 1080,
		Title:  "Test Window",
	}

	defer glfw.Terminate()

	window, err := Types.CreateWindow(WindowBuilder)
	CheckError(err)

	err = Types.NewGLContext()
	CheckError(err)

	Types.NewLog("Initializing OpenGL Settings")

	Types.NewLog("Program startup:")

	for !window.ShouldClose() {

		glfw.PollEvents()
		window.SwapBuffers()

	}
}
