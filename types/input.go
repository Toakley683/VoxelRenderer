package types

import (
	Client "VoxelRPG/client"
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

func WindowInputCB(clientContext *Client.ClientContext, w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {

	if clientContext.OnKeyMap[key] != nil {
		clientContext.OnKeyMap[key](action)
	}

	clientContext.InputMap[key] = action

}

func WindowMouseCB(clientContext *Client.ClientContext, w *glfw.Window, xpos float64, ypos float64) {

	if w.GetAttrib(glfw.Focused) == 0 {
		clientContext.Cursor.FirstMouse = true
		return
	}

	// Get Window and Framebuffer sizes

	width, height := w.GetSize()

	// Calculate Center point of window

	centerX := float64(width / 2)
	centerY := float64(height / 2)

	if clientContext.Cursor.FirstMouse {

		w.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

		w.SetCursorPos(centerX, centerY)
		clientContext.Cursor.X = centerX
		clientContext.Cursor.Y = centerY
		clientContext.Cursor.FirstMouse = false
		return
	}

	xOffset := xpos - centerX
	yOffset := centerY - ypos

	xOffset *= clientContext.Sensitivity
	yOffset *= clientContext.Sensitivity

	clientContext.Camera.Yaw += xOffset
	clientContext.Camera.Pitch += yOffset

	clientContext.Camera.Pitch = ClampF64(clientContext.Camera.Pitch, -89.99, 89.99)

	radYaw := clientContext.Camera.Yaw * math.Pi / 180
	radPitch := clientContext.Camera.Pitch * math.Pi / 180

	front := mgl32.Vec3{
		float32(math.Cos(radYaw) * math.Cos(radPitch)),
		float32(math.Sin(radPitch)),
		float32(math.Sin(radYaw) * math.Cos(radPitch)),
	}.Normalize()

	clientContext.Camera.Front = front

	w.SetCursorPos(centerX, centerY)
	clientContext.Cursor.FirstMouse = true

}
