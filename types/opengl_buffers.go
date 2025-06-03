package types

import "github.com/go-gl/gl/v4.6-core/gl"

func NewBufferObject(index int32, buffer_type uint32, data_cb func(uint32)) uint32 {

	var vbo uint32

	gl.GenBuffers(index, &vbo)
	gl.BindBuffer(buffer_type, vbo)

	data_cb(vbo)

	return vbo

}

func NewVertexArray(index int32) uint32 {

	var vao uint32

	gl.GenVertexArrays(index, &vao)
	gl.BindVertexArray(vao)

	return vao

}

var fullscreenQuadVertices = []float32{ // x, y, u, v
	-1.0, -1.0, 0.0, 0.0,
	1.0, -1.0, 1.0, 0.0,
	-1.0, 1.0, 0.0, 1.0,

	1.0, -1.0, 1.0, 0.0,
	1.0, 1.0, 1.0, 1.0,
	-1.0, 1.0, 0.0, 1.0,
}
