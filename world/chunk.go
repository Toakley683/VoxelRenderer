package world

import (
	"math/rand"
	"runtime"
	"sync"
	"time"

	Log "VoxelRPG/logging"

	"github.com/go-gl/mathgl/mgl32"
)

//"github.com/go-gl/gl/v4.6-core/gl"

func IsBlockFull(x, y, z int) bool {

	return rand.Float32() > 0.5

}

func VoxelMetadata(idx int) GridMetadata {

	return GridMetadata{
		R: rand.Uint32() % 256,
		G: rand.Uint32() % 256,
		B: rand.Uint32() % 256,
	}

}

func NewChunk(WorldPosition Vec3) *Chunk {

	outputChunk := Chunk{
		Position:     WorldPosition,
		Voxels:       make([]uint8, (FULL_CHUNK_SIZE+7)/8),
		OctreeOffset: MaxUINT32,
	}

	StartedChunkGen := time.Now()

	outputChunk.GenerateVoxelData()

	Log.NewLog("Length:", len(outputChunk.Voxels))

	Log.NewLog("Chunk generation took:", time.Since(StartedChunkGen))

	StartedOctreeGen := time.Now()

	outputChunk.BuildNodes()

	Log.NewLog("Octree generation took:", time.Since(StartedOctreeGen))

	return &outputChunk

}

func (chunk *Chunk) BuildNodes() {

	chunk.OctreeNodes = chunk.BuildNestedGrid()
	runtime.GC()

}

func (chunk *Chunk) IsVisible(viewProjection mgl32.Mat4) bool {
	// Calculate chunk bounding box in world space

	CSize := float32(CHUNK_SIZE)

	min := mgl32.Vec3{float32(chunk.Position.X), float32(chunk.Position.Y), float32(chunk.Position.Z)}
	max := min.Add(mgl32.Vec3{CSize, CSize, CSize})

	// Extract frustum planes from viewProjection matrix
	planes := ExtractFrustumPlanes(viewProjection)

	// Test AABB against all frustum planes
	for _, plane := range planes {
		if !aabbIntersectsPlane(min, max, plane) {
			// If chunk is fully outside any frustum plane, it's not visible
			return false
		}
	}

	// If chunk intersects all planes, it is visible
	return true
}

func (chunk *Chunk) GenerateVoxelData() {

	ByteNumber := FULL_CHUNK_SIZE //(FULL_CHUNK_SIZE + 7) / 8
	BatchSizes := ByteNumber / CHUNK_WORKERS

	var chunkAwait sync.WaitGroup
	chunkAwait.Add(CHUNK_WORKERS)

	for workerIndex := 0; workerIndex < CHUNK_WORKERS; workerIndex++ {

		start := workerIndex * BatchSizes
		end := start + BatchSizes

		if workerIndex == CHUNK_WORKERS-1 {
			end = ByteNumber
		}

		go func(start, end int) {

			defer chunkAwait.Done()

			for voxelIndex := start; voxelIndex < end; voxelIndex++ {

				m := chunk.Position.MulScalar(int32(CHUNK_SIZE))

				x := voxelIndex % CHUNK_SIZE
				y := (voxelIndex / CHUNK_SIZE) % CHUNK_SIZE
				z := voxelIndex / (CHUNK_SIZE * CHUNK_SIZE)

				isFull := IsBlockFull(int(m.X)+x, int(m.Y)+y, int(m.Z)+z)

				chunkSetVoxelBit(chunk.Voxels, voxelIndex, isFull)

			}

		}(start, end)

	}

	chunkAwait.Wait()

}
