package world

import (
	"sync"
	"time"

	Log "VoxelRPG/logging"
)

//"github.com/go-gl/gl/v4.6-core/gl"

func IsBlockFull(x, y, z int) bool {

	if z == 0 {
		if y == 0 {
			if z == 0 {
				Log.NewLog("New block at:", x, y, z)
				return true
			}
		}
	}

	return false //rand.Float32() > 0.55

}

func NewChunk(WorldPosition Vec3) *Chunk {

	outputChunk := Chunk{
		Position: WorldPosition,
		Voxels:   [FULL_CHUNK_SIZE]bool{},
		SSBO:     0,
	}

	StartedChunkGen := time.Now()

	BatchSizes := FULL_CHUNK_SIZE / CHUNK_WORKERS

	type ChunkOutput struct {
		index int
		value bool
	}

	var resultOutput = make(chan ChunkOutput, FULL_CHUNK_SIZE)

	var chunkAwait sync.WaitGroup
	chunkAwait.Add(CHUNK_WORKERS)

	for workerIndex := 0; workerIndex < CHUNK_WORKERS; workerIndex++ {

		start := workerIndex * BatchSizes
		end := start + BatchSizes

		if workerIndex == CHUNK_WORKERS-1 {
			end = FULL_CHUNK_SIZE
		}

		go func(start, end int) {

			defer chunkAwait.Done()

			for voxelIndex := start; voxelIndex < end; voxelIndex++ {

				x := voxelIndex % CHUNK_SIZE
				y := (voxelIndex / CHUNK_SIZE) % CHUNK_SIZE
				z := voxelIndex / (CHUNK_SIZE * CHUNK_SIZE)

				isFull := IsBlockFull(x, y, z)

				resultOutput <- ChunkOutput{
					index: voxelIndex,
					value: isFull,
				}

			}

		}(start, end)

	}

	go func() {
		chunkAwait.Wait()
		close(resultOutput)
	}()

	for val := range resultOutput {
		outputChunk.Voxels[val.index] = val.value
	}

	//Log.NewLog(outputChunk.Voxels)

	Log.NewLog("Length:", len(outputChunk.Voxels))

	Log.NewLog("Chunk generation took:", time.Since(StartedChunkGen))

	return &outputChunk

}

func (chunk *Chunk) Upload(shaderProgram uint32) {

	grid, _ := chunk.BuildNestedGrid()

	chunk.SetChunkPositionUniform(shaderProgram)

	chunk.UploadOctreeSSBO(grid)
}
