package world

const (
	CHUNK_WORKERS int = 8
	WORLD_WORKERS int = 32

	CHUNK_SIZE          int = 32
	VERTICAL_CHUNK_SIZE int = 32

	VOXEL_SIZE int = 1
	GRID_SIZES int = 6

	FULL_CHUNK_SIZE int = (int(CHUNK_SIZE) * int(VERTICAL_CHUNK_SIZE) * int(CHUNK_SIZE))
)

var (
	RENDER_DISTANCE         int  = 1
	RENDER_DISTANCE_POINTER *int = &RENDER_DISTANCE
)

type Vec3 struct{ X, Y, Z uint32 }

func (v2 Vec3) Mul(v1 Vec3) Vec3 {
	return Vec3{v1.X + v2.X, v1.Y + v2.Y, v1.Z + v2.Z}
}
func (v Vec3) MulScalar(s uint32) Vec3 {
	return Vec3{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

func (v Vec3) Add(o Vec3) Vec3 {
	return Vec3{X: v.X + o.X, Y: v.Y + o.Y, Z: v.Z + o.Z}
}

type Chunk struct {
	Position Vec3
	Voxels   [FULL_CHUNK_SIZE]bool
	SSBO     uint32
}

type World struct {
	RenderCenter   *Vec3
	RenderDistance *int
	Chunks         []*Chunk
}
