package world

import (
	Log "VoxelRPG/logging"
	"fmt"
	"math/rand"
	"strconv"
	"unsafe"

	"github.com/go-gl/gl/v4.6-core/gl"
)

var (
	GridSizes         []int
	LevelStartIndices []int
	NodesRequired     int

	FlagOccupied uint32 = 1 << 0
)

type GridMetadata struct {
	R uint32
	G uint32
	B uint32
	_ uint32
}

type GridNodeFlat struct {
	Flags      uint32
	ChildStart int32 // index of first child, or -1 if leaf
	Size       int32
	Metadata   GridMetadata
}

type GridNodeFlatGPU struct {
	Flags      uint32
	ChildStart int32
	Size       int32
	Metadata   GridMetadata
}

func init() {

	GridSizes = GenerateGridSizes(CHUNK_SIZE, GRID_SIZES)
	Log.NewLog("Grid sizes per level", GridSizes)

	NodesRequired = CalculateTotalNodes(GridSizes, GRID_SIZES)
	Log.NewLog("Nodes required", NodesRequired)

	LevelStartIndices = make([]int, len(GridSizes))

	for i := 0; i < len(GridSizes); i++ {

		LevelStartIndices[i] = CalculateTotalNodes(GridSizes, i-1)

	}

	Log.NewLog("Indicies:", LevelStartIndices)

}

func GenerateGridSizes(maxSize, levels int) []int {
	sizes := make([]int, levels)
	size := maxSize
	for i := levels - 1; i >= 0; i-- {
		Log.NewLog(i)
		sizes[i] = size
		if size > 1 {
			size = size / 2
		}
	}
	return sizes
}

func CalculateTotalNodes(sizes []int, maxLevel int) int {
	total := 0
	for level, size := range sizes {
		if level > maxLevel {
			return total
		}

		total += size * size * size
	}
	return total
}

func GetChildIndex(parentIndex int, parentLevel int, childNum int) int {
	if parentLevel < 0 || parentLevel >= len(GridSizes)-1 {
		return -1
	}

	childLevel := parentLevel + 1
	relParentIdx := parentIndex - LevelStartIndices[parentLevel]
	childBase := LevelStartIndices[childLevel] + (relParentIdx * 8)
	return childBase + childNum
}

/*func GetChildIndex(parentIndex int, parentLevel int, childNum int) int {
    if parentLevel < 0 || parentLevel >= len(GridSizes)-1 {
        return -1
    }

    childLevel := parentLevel + 1

    // Size of grid at parent and child level
    parentSize := GridSizes[parentLevel]
    childSize := GridSizes[childLevel]

    // Convert parentIndex to coords in parent grid
    relParentIdx := parentIndex - LevelStartIndices[parentLevel]
    px, py, pz := IndexToCoords(relParentIdx, parentSize)

    // Decode childNum into child offsets (each 0 or 1)
    // childNum bits: bit0=x, bit1=y, bit2=z
    cx := px*2 + (childNum & 1)
    cy := py*2 + ((childNum >> 1) & 1)
    cz := pz*2 + ((childNum >> 2) & 1)

    // Compute child relative index in child grid
    childRelIdx := CoordsToIndex(cx, cy, cz, childSize)

    // Child absolute index = start of child level + relative child index
    childIndex := LevelStartIndices[childLevel] + childRelIdx

    return childIndex
}*/

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

func (chunk *Chunk) NewLevel(gridID int, nodeList []GridNodeFlat) ([]GridNodeFlat, int) {

	S := int32(GridSizes[(GRID_SIZES-1)-gridID])
	Size := int32(GridSizes[gridID])
	idxMax := Size * Size * Size

	startIndex := CalculateTotalNodes(GridSizes, gridID-1)

	Log.NewLog("Start", startIndex)
	Log.NewLog("Size:", S)
	Log.NewLog("Max:", idxMax)

	HasVoxels := false

	for idx := 0; idx < int(idxMax); idx++ {

		parentGlobalIdx := startIndex + idx
		cIndex := int32(GetChildIndex(parentGlobalIdx, gridID, 0))

		flags := uint32(0)

		if cIndex == -1 {

			if chunk.Voxels[idx] {

				HasVoxels = true
				flags = FlagOccupied
			}

		} else {

			for i := int32(0); i < 8; i++ {

				nIndex := int32(GetChildIndex(parentGlobalIdx, gridID, int(i)))

				if nIndex >= 0 && int(nIndex) < len(nodeList) {

					if nodeList[nIndex].Flags == FlagOccupied {
						childSize := 32 / GridSizes[gridID+1]
						relIndex := int(nIndex) - LevelStartIndices[gridID+1]
						x, y, z := IndexToCoords(relIndex, childSize)

						Log.NewLog("RelIndex", relIndex, "GridID:", i, "Coords:", nIndex, "At:", x, y, z)
						HasVoxels = true
						flags = FlagOccupied
					}
				}

			}

		}

		nodeList[parentGlobalIdx] = GridNodeFlat{
			Flags:      flags,
			ChildStart: cIndex,
			Size:       S,
			Metadata:   VoxelMetadata(idx),
		}

	}

	Log.NewLog("Has:", HasVoxels)

	if gridID == GRID_SIZES-2 {

		Log.NewLog("Occupied:", HasVoxels)

		//log.Fatal("Test")

	}

	Log.NewLog("Grid:", strconv.Itoa(gridID-1), "StartIndex:", startIndex)

	fmt.Printf("\n")

	return nodeList, startIndex

}

func (chunk *Chunk) BuildNestedGrid() ([]GridNodeFlat, int) {

	var Nodes []GridNodeFlat = make([]GridNodeFlat, NodesRequired)

	fmt.Printf("\n")

	for i := GRID_SIZES - 1; i >= 0; i-- {

		S := GridSizes[i]

		Log.NewLog("Level:", i, "Size:", S)

		Nodes, _ = chunk.NewLevel(i, Nodes)

	}

	lastN := 0

	for i := 0; i < 6; i++ {

		C := Nodes[lastN]

		Log.NewLog("Node["+strconv.Itoa(lastN)+"].Flags =", C.Flags)
		Log.NewLog("Node["+strconv.Itoa(lastN)+"].ChildStart =", C.ChildStart)
		Log.NewLog("Node["+strconv.Itoa(lastN)+"].Size =", C.Size)

		fmt.Printf("\n")

		lastN = int(C.ChildStart)

	}

	FC := Nodes[len(Nodes)-1]

	Log.NewLog("Node["+strconv.Itoa(len(Nodes)-1)+"].Flags =", FC.Flags)
	Log.NewLog("Node["+strconv.Itoa(len(Nodes)-1)+"].ChildStart =", FC.ChildStart)
	Log.NewLog("Node["+strconv.Itoa(len(Nodes)-1)+"].Size =", FC.Size)

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

func (chunk *Chunk) SetChunkPositionUniform(program uint32) {
	loc := gl.GetUniformLocation(program, gl.Str("chunkPos\x00"))
	gl.Uniform3i(loc, int32(chunk.Position.X), int32(chunk.Position.Y), int32(chunk.Position.Z))
}
