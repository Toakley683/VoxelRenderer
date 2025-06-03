package client

import (
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type Camera struct {
	Pos   mgl32.Vec3
	Pitch float64
	Yaw   float64
	Roll  float64
	Front mgl32.Vec3
	Speed float32
}

type Cursor struct {
	X          float64
	Y          float64
	FirstMouse bool
}

type ClientContext struct {
	InputMap      map[glfw.Key]glfw.Action
	OnKeyMap      map[glfw.Key]func(glfw.Action)
	Camera        *Camera
	Cursor        *Cursor
	Sensitivity   float64
	SetupKeybinds func()
}

func NewClient() (*ClientContext, error) {

	InputMap := make(map[glfw.Key]glfw.Action)
	OnKeyMap := make(map[glfw.Key]func(glfw.Action))

	Camera := Camera{}

	Camera.Speed = 2

	Camera.Pos = mgl32.Vec3{}

	Camera.Pos[0] = 0
	Camera.Pos[1] = 0
	Camera.Pos[2] = -3

	Camera.Yaw = 90

	radYaw := Camera.Yaw * math.Pi / 180
	radPitch := Camera.Pitch * math.Pi / 180

	Camera.Front = mgl32.Vec3{
		float32(math.Cos(radYaw) * math.Cos(radPitch)),
		float32(math.Sin(radPitch)),
		float32(math.Sin(radYaw) * math.Cos(radPitch)),
	}.Normalize()

	Cursor := Cursor{}
	Cursor.FirstMouse = true

	CContext := &ClientContext{
		InputMap:    InputMap,
		OnKeyMap:    OnKeyMap,
		Camera:      &Camera,
		Cursor:      &Cursor,
		Sensitivity: float64(0.14),
	}

	CContext.SetupKeybinds = func() {
		ClientSetupKeybinds(CContext)
	}

	return CContext, nil

}

func CBOnKeyChange(clientContext *ClientContext, key glfw.Key, cb func(action glfw.Action)) {

	clientContext.OnKeyMap[key] = cb

}

func ClientCheckMovement(context *ClientContext, deltaTime float32) {

	if context.InputMap[glfw.KeyW] != glfw.Release {

		Dir := context.Camera.Front.Mul(context.Camera.Speed).Mul(deltaTime)
		context.Camera.Pos = context.Camera.Pos.Add(Dir)

	}

	if context.InputMap[glfw.KeyS] != glfw.Release {

		Dir := context.Camera.Front.Mul(context.Camera.Speed).Mul(deltaTime)
		context.Camera.Pos = context.Camera.Pos.Sub(Dir)

	}

	if context.InputMap[glfw.KeyD] != glfw.Release {

		Dir := context.Camera.Front.Mul(context.Camera.Speed).Mul(deltaTime)
		NDir := mgl32.Vec3{Dir[2], 0, -Dir[0]}

		context.Camera.Pos = context.Camera.Pos.Sub(NDir)

	}

	if context.InputMap[glfw.KeyA] != glfw.Release {

		Dir := context.Camera.Front.Mul(context.Camera.Speed).Mul(deltaTime)
		NDir := mgl32.Vec3{Dir[2], 0, -Dir[0]}

		context.Camera.Pos = context.Camera.Pos.Add(NDir)

	}

	if context.InputMap[glfw.KeySpace] != glfw.Release {

		Dir := mgl32.Vec3{0, 1, 0}.Mul(context.Camera.Speed).Mul(deltaTime)

		context.Camera.Pos = context.Camera.Pos.Add(Dir)

	}

	if context.InputMap[glfw.KeyLeftControl] != glfw.Release {

		Dir := mgl32.Vec3{0, -1, 0}.Mul(context.Camera.Speed).Mul(deltaTime)

		context.Camera.Pos = context.Camera.Pos.Add(Dir)

	}

	if context.InputMap[glfw.KeyLeftShift] != glfw.Release {

		context.Camera.Speed = 10

	}

	if context.InputMap[glfw.KeyLeftShift] == glfw.Release {

		context.Camera.Speed = 3

	}

}

func ClientSetupKeybinds(context *ClientContext) {

}
