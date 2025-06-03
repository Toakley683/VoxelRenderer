package world

import (
	"math/rand"
	"sync"
	"time"

	Log "VoxelRPG/logging"

	"github.com/go-gl/gl/v4.6-core/gl"
)

//"github.com/go-gl/gl/v4.6-core/gl"

func IsBlockFull(x, y, z int) bool {

	return rand.Float32() > 0.95

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

func (chunk *Chunk) SetChunkPositionUniform(program uint32) {

	loc := gl.GetUniformLocation(program, gl.Str("chunkPos\x00"))
	gl.Uniform3i(loc, int32(chunk.Position.X), int32(chunk.Position.Y), int32(chunk.Position.Z))

	lsi := gl.GetUniformLocation(program, gl.Str("LevelStartIndices\x00"))
	gl.Uniform1iv(lsi, int32(len(GridSizes)), &LevelStartIndices[0])

	gsi := gl.GetUniformLocation(program, gl.Str("GridSizes\x00"))
	gl.Uniform1iv(gsi, int32(len(GridSizes)), &GridSizes[0])

}
