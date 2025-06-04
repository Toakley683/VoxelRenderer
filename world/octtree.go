package world

import (
	"runtime"
	"runtime/debug"
)

func GetChildIndex(parentIndex int, parentLevel int) int {
	if parentLevel < 0 || parentLevel >= len(GridSizes)-1 {
		return -1
	}

	childLevel := parentLevel + 1
	relParentIdx := parentIndex - int(LevelStartIndices[parentLevel])
	childBase := int(LevelStartIndices[childLevel]) + (relParentIdx * 8)
	return childBase
}

func (chunk *Chunk) NewLevel(gridID int, nodeList []GridNodeFlatGPU) ([]GridNodeFlatGPU, int) {

	S := int32(GridSizes[(GRID_SIZES-1)-gridID])
	Size := int32(GridSizes[gridID])
	idxMax := Size * Size * Size

	startIndex := CalculateTotalNodes(GridSizes, gridID-1)

	for idx := 0; idx < int(idxMax); idx++ {

		parentGlobalIdx := startIndex + idx
		cIndex := int32(GetChildIndex(parentGlobalIdx, gridID))

		children := [8]uint32{
			MaxUINT32, MaxUINT32, MaxUINT32, MaxUINT32,
			MaxUINT32, MaxUINT32, MaxUINT32, MaxUINT32,
		}

		flags := uint32(0)

		if cIndex == -1 {

			if chunkGetVoxelBit(chunk.Voxels, idx) {
				flags |= FlagOccupied
			}
			flags |= FlagLeaf

		} else {

			childSize := GridSizes[gridID+1]

			x, y, z := IndexToCoords(idx, int(Size))

			baseX, baseY, baseZ := x*2, y*2, z*2

			for i := int32(0); i < 8; i++ {

				Pos := Vec3{X: int32(baseX), Y: int32(baseY), Z: int32(baseZ)}.Add(Vec3{
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

		nodeList[parentGlobalIdx] = GridNodeFlatGPU{
			Flags:    flags,
			Children: children,
			Size:     S,
			Metadata: VoxelMetadata(idx),
		}

	}

	return nodeList, startIndex

}

func (chunk *Chunk) BuildNestedGrid() []GridNodeFlatGPU {

	var Nodes []GridNodeFlatGPU = make([]GridNodeFlatGPU, NodesRequired)

	for i := GRID_SIZES - 1; i >= 0; i-- {

		Nodes, _ = chunk.NewLevel(i, Nodes)

	}

	return Nodes

}

func BuildCombinedOctreeData(chunks []*Chunk) []GridNodeFlatGPU {

	totalNodes := 0

	for i := 0; i < len(chunks); i++ {
		chunks[i].BuildNodes()
		nodeCount := len(chunks[i].OctreeNodes)
		totalNodes += nodeCount
	}

	combined := make([]GridNodeFlatGPU, totalNodes)

	var currentOffset int = 0

	for _, chunk := range chunks {

		chunk.OctreeOffset = uint32(currentOffset)

		copy(combined[currentOffset:], chunk.OctreeNodes)

		currentOffset += len(chunk.OctreeNodes)

		chunk.OctreeNodes = nil

	}

	runtime.GC()
	debug.FreeOSMemory()

	return combined
}
