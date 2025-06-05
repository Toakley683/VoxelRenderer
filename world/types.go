package world

import (
	Log "VoxelRPG/logging"
	"errors"
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	CHUNK_WORKERS int = 8
	WORLD_WORKERS int = 32

	CHUNK_SIZE          int = 32
	VERTICAL_CHUNK_SIZE int = 32

	CHUNK_SCALE float32 = 1.0 / 32.0 // How many units across is a chunk

	VOXEL_SIZE int = 1
	GRID_SIZES int = 6

	FULL_CHUNK_SIZE int = (int(CHUNK_SIZE) * int(VERTICAL_CHUNK_SIZE) * int(CHUNK_SIZE))

	MaxUINT32 uint32 = 0xFFFFFFFF

	DEBUG_MODE = false
)

var (
	RENDER_DISTANCE         int  = 4
	RENDER_DISTANCE_POINTER *int = &RENDER_DISTANCE
)

type Vec3 struct{ X, Y, Z int32 }

func (v Vec3) Add(o Vec3) Vec3 {
	return Vec3{X: v.X + o.X, Y: v.Y + o.Y, Z: v.Z + o.Z}
}
func (v Vec3) MulScalar(o int32) Vec3 {
	return Vec3{X: v.X * o, Y: v.Y * o, Z: v.Z * o}
}

type Chunk struct {
	Position Vec3
	Voxels   []uint8

	OctreeNodes  []GridNodeFlatGPU
	OctreeOffset uint32
}

type World struct {
	RenderCenter    *Vec3
	RenderDistance  *int
	LastCameraChunk Vec3

	Chunks []*Chunk

	CombinedSSBO         uint32
	WorldInfoSSBO        uint32
	WorldInfoOffsetsSSBO uint32
	DebugResultSSBO      uint32
}

/* -- [[ Camera Chunking ]] -- */

func GetCameraChunk(pos mgl32.Vec3) Vec3 {
	fChunk := float64(CHUNK_SIZE)

	return Vec3{
		X: int32(math.Floor(float64(pos.X())/fChunk) * fChunk),
		Y: int32(math.Floor(float64(pos.Y())/fChunk) * fChunk),
		Z: int32(math.Floor(float64(pos.Z())/fChunk) * fChunk),
	}
}

/* -- [[ Voxel Bit Packing ]] -- */

func chunkSetVoxelBit(chunkData []uint8, idx int, value bool) {
	byteIndex := idx / 8
	bitIndex := uint(idx % 8)

	if value {
		// Set the bit
		chunkData[byteIndex] |= (1 << bitIndex)
	} else {
		// Clear the bit
		chunkData[byteIndex] &^= (1 << bitIndex) // &^ is bit clear (AND NOT) in Go
	}
}

func chunkGetVoxelBit(chunkData []uint8, idx int) bool {
	byteIndex := idx / 8
	bitIndex := uint(idx % 8)
	return (chunkData[byteIndex] & (1 << bitIndex)) != 0
}

/* -- [[ Converting Index to 3D coordinates in a chunk, and inverse ]] -- */

func IndexToCoords(idx, size int) (x, y, z int) {
	x = idx % size
	y = (idx / size) % size
	z = idx / (size * size)
	return
}

func CoordsToIndex(x, y, z, size int) int {
	return x + y*size + z*size*size
}

/* -- [[ ALL OCTREE TYPES ]] -- */

const (
	FlagOccupied uint32 = 1 << 0
	FlagLeaf     uint32 = 1 << 1
)

var (
	GridSizes         []int32
	LevelStartIndices []int32
	NodesRequired     int
)

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

/* -- [[ Grid Structs for sending to GPU ]] -- */

type GridMetadata struct {
	R uint32
	G uint32
	B uint32
	_ uint32
}

type GridNodeFlatGPU struct {
	Children [8]uint32
	Flags    uint32
	Size     int32
	_        [2]uint32
	Metadata GridMetadata
}
type ChunkInfo struct {
	Key        Vec3
	RootOffset uint32
}

/* -- [[ Flags for Octree Nodes ]] -- */

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

/* -- [[ Generate Octree Grid Sizes / Total Nodes Count ]] -- */

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

/* -- [[ Generate Hashmap for GLSL shader to lookup (For chunk lookup) ]] -- */

type MapEntry struct {
	Position   Vec3
	RootOffset uint32
}

func hashUint32(x uint32) uint32 {
	x ^= x >> 16
	x *= 0x7feb352d
	x ^= x >> 15
	x *= 0x846ca68b
	x ^= x >> 16
	return x
}

func hash3D(pos [3]int32) uint32 {
	// Convert signed to unsigned by offsetting
	ux := uint32(int64(pos[0]) + 0x80000000)
	uy := uint32(int64(pos[1]) + 0x80000000)
	uz := uint32(int64(pos[2]) + 0x80000000)

	h := hashUint32(ux)
	h ^= hashUint32(uy) + 0x9e3779b9 + (h << 6) + (h >> 2)
	h ^= hashUint32(uz) + 0x9e3779b9 + (h << 6) + (h >> 2)
	return h
}

func hash1(pos [3]int32) uint32 {
	return hash3D(pos)
}

func hash2(pos [3]int32) uint32 {
	// multiply components by constants same as GLSL
	return hash3D([3]int32{
		pos[0] * int32(0x27d4eb2d),
		pos[1] * int32(0x165667b1),
		pos[2] * int32(0x1b873593),
	})
}

func BuildPerfectHashTable(chunks []*Chunk) ([]uint32, []MapEntry, error) {
	N := len(chunks)
	if N == 0 {
		return nil, nil, errors.New("empty chunk slice")
	}

	// Precompute keys as hash3D of chunk positions
	keys := make([][3]int32, N)
	for i, c := range chunks {
		keys[i] = [3]int32{c.Position.X, c.Position.Y, c.Position.Z}
	}

	// Define slot type with index and position key (3D)
	type slot struct {
		idx int
		key [3]int32
	}

	buckets := make([][]slot, N)
	for i, pos := range keys {
		h := hash2(pos) % uint32(N)
		buckets[h] = append(buckets[h], slot{i, pos})
	}

	// Sort buckets descending by size (largest first)
	for i := 0; i < N-1; i++ {
		for j := 0; j < N-1-i; j++ {
			if len(buckets[j]) < len(buckets[j+1]) {
				buckets[j], buckets[j+1] = buckets[j+1], buckets[j]
			}
		}
	}

	displacements := make([]uint32, N)
	assigned := make([]int, N)
	for i := range assigned {
		assigned[i] = -1
	}

	usedSlots := make([]bool, N)

	// Assign displacements to avoid collisions
	for _, bucket := range buckets {
		if len(bucket) == 0 {
			continue
		}
		var d uint32
		found := false
		for d = 0; d < uint32(N); d++ {
			collision := false
			for _, s := range bucket {
				pos := (hash1(s.key) + d) % uint32(N)
				if usedSlots[pos] {
					collision = true
					break
				}
			}
			if !collision {
				found = true
				break
			}
		}
		if !found {
			return nil, nil, errors.New("failed to find displacement for a bucket")
		}

		h := hash2(bucket[0].key) % uint32(N)
		displacements[h] = d

		for _, s := range bucket {
			pos := (hash1(s.key) + d) % uint32(N)
			usedSlots[pos] = true
			assigned[pos] = s.idx
		}
	}

	// Build the table output
	table := make([]MapEntry, N)
	for i := 0; i < N; i++ {
		if assigned[i] == -1 {
			table[i] = MapEntry{
				Position:   Vec3{},
				RootOffset: math.MaxUint32,
			}
		} else {
			c := chunks[assigned[i]]
			table[i] = MapEntry{
				Position:   c.Position,
				RootOffset: c.OctreeOffset,
			}
		}
	}

	return displacements, table, nil
}
