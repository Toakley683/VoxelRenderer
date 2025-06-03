package world

import (
	Log "VoxelRPG/logging"
	"math/rand"
	"unsafe"

	"github.com/go-gl/gl/v4.6-core/gl"
)

var (
	GridSizes         []int32
	LevelStartIndices []int32
	NodesRequired     int

	FlagOccupied uint32 = 1 << 0
	FlagLeaf     uint32 = 1 << 1

	MaxUINT32 uint32 = 0xFFFFFFFF
)

type GridMetadata struct {
	R uint32
	G uint32
	B uint32
	_ uint32
}

type GridNodeFlat struct {
	Flags    uint32
	Children [8]uint32 // index of first child, or -1 if leaf
	Size     int32
	Metadata GridMetadata
}

type GridNodeFlatGPU struct {
	Flags    uint32
	Children [8]uint32
	Size     int32
	Metadata GridMetadata
}

func init() {

	GridSizes = GenerateGridSizes(CHUNK_SIZE, GRID_SIZES)
	Log.NewLog("Grid sizes per level", GridSizes)

	NodesRequired = CalculateTotalNodes(GridSizes, GRID_SIZES)
	Log.NewLog("Nodes required", NodesRequired)

	LevelStartIndices = make([]int32, len(GridSizes))

	for i := 0; i < len(GridSizes); i++ {

		LevelStartIndices[i] = int32(CalculateTotalNodes(GridSizes, int(i-1)))

	}

	Log.NewLog("Indicies:", LevelStartIndices)

}

func GenerateGridSizes(maxSize, levels int) []int32 {
	sizes := make([]int32, levels)
	size := maxSize
	for i := levels - 1; i >= 0; i-- {
		Log.NewLog(i)
		sizes[i] = int32(size)
		if size > 1 {
			size = size / 2
		}
	}
	return sizes
}

func CalculateTotalNodes(sizes []int32, maxLevel int) int {
	total := 0
	for level, size := range sizes {
		if level > maxLevel {
			return total
		}

		total += int(size * size * size)
	}
	return total
}

func GetChildIndex(parentIndex int, parentLevel int, childNum int) int {
	if parentLevel < 0 || parentLevel >= len(GridSizes)-1 {
		return -1
	}

	childLevel := parentLevel + 1
	relParentIdx := parentIndex - int(LevelStartIndices[parentLevel])
	childBase := int(LevelStartIndices[childLevel]) + (relParentIdx * 8)
	return childBase + childNum
}

func VoxelMetadata(idx int) GridMetadata {

	return GridMetadata{
		R: rand.Uint32() % 256,
		G: rand.Uint32() % 256,
		B: rand.Uint32() % 256,
	}

}

func IndexToCoords(idx, size int) (x, y, z int) {
	x = idx % size
	y = (idx / size) % size
	z = idx / (size * size)
	return
}

func CoordsToIndex(x, y, z, size int) int {
	return x + y*size + z*size*size
}

type FlagBits struct {
	Occupied bool
	Leaf     bool
}

func DecodeFlags(flags uint32) FlagBits {
	return FlagBits{
		Occupied: flags&FlagOccupied != 0,
		Leaf:     flags&FlagLeaf != 0,
	}
}

func (chunk *Chunk) NewLevel(gridID int, nodeList []GridNodeFlat) ([]GridNodeFlat, int) {

	S := int32(GridSizes[(GRID_SIZES-1)-gridID])
	Size := int32(GridSizes[gridID])
	idxMax := Size * Size * Size

	startIndex := CalculateTotalNodes(GridSizes, gridID-1)

	for idx := 0; idx < int(idxMax); idx++ {

		parentGlobalIdx := startIndex + idx
		cIndex := int32(GetChildIndex(parentGlobalIdx, gridID, 0))

		children := [8]uint32{
			MaxUINT32, MaxUINT32, MaxUINT32, MaxUINT32,
			MaxUINT32, MaxUINT32, MaxUINT32, MaxUINT32,
		}

		flags := uint32(0)

		if cIndex == -1 {

			if chunk.Voxels[idx] {
				flags |= FlagOccupied
			}
			flags |= FlagLeaf

		} else {

			childSize := GridSizes[gridID+1]

			x, y, z := IndexToCoords(idx, int(Size))

			baseX, baseY, baseZ := x*2, y*2, z*2

			for i := uint32(0); i < 8; i++ {

				Pos := Vec3{X: uint32(baseX), Y: uint32(baseY), Z: uint32(baseZ)}.Add(Vec3{
					X: i & 1,
					Y: (i >> 1) & 1,
					Z: (i >> 2) & 1,
				})

				if int32(Pos.X) >= childSize || int32(Pos.Y) >= childSize || int32(Pos.Z) >= childSize {
					// Out of bounds - skip
					continue
				}
				childIdxLocal := CoordsToIndex(int(Pos.X), int(Pos.Y), int(Pos.Z), int(childSize))
				childGlobalIdx := int(LevelStartIndices[gridID+1]) + childIdxLocal

				childNode := nodeList[childGlobalIdx]

				children[i] = uint32(childGlobalIdx)

				f := DecodeFlags(childNode.Flags)

				if f.Occupied {
					flags |= FlagOccupied
				}

			}

		}

		nodeList[parentGlobalIdx] = GridNodeFlat{
			Flags:    flags,
			Children: children,
			Size:     S,
			Metadata: VoxelMetadata(idx),
		}

	}

	return nodeList, startIndex

}

func (chunk *Chunk) BuildNestedGrid() ([]GridNodeFlat, int) {

	var Nodes []GridNodeFlat = make([]GridNodeFlat, NodesRequired)

	for i := GRID_SIZES - 1; i >= 0; i-- {

		Nodes, _ = chunk.NewLevel(i, Nodes)

	}

	return Nodes, 0

}

func ConvertToGPU(nodes []GridNodeFlat) []GridNodeFlatGPU {
	gpuNodes := make([]GridNodeFlatGPU, len(nodes))
	for i, n := range nodes {
		gpuNodes[i] = GridNodeFlatGPU(n)
	}
	return gpuNodes
}

func (chunk *Chunk) UploadOctreeSSBO(nodes []GridNodeFlat) {
	gpuNodes := ConvertToGPU(nodes)

	if chunk.SSBO == 0 {
		gl.GenBuffers(1, &chunk.SSBO)
	}
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, chunk.SSBO)
	gl.BufferData(gl.SHADER_STORAGE_BUFFER, len(gpuNodes)*int(unsafe.Sizeof(gpuNodes[0])), gl.Ptr(gpuNodes), gl.DYNAMIC_DRAW)
	// Bind the buffer to binding point 0 (match GLSL layout(binding = 0))
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 0, chunk.SSBO)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, 0)
}
