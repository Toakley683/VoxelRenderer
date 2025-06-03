package world

import (
	Log "VoxelRPG/logging"
	"strconv"
	"sync"
	"time"
)

var (
	MainWorld *World
)

func init() {

	MainWorld = &World{
		RenderDistance: RENDER_DISTANCE_POINTER,
	}

}

func (w *World) Update() {

}

func (w *World) Populate(shaderProgram uint32) {

	ChunksLength := (*w.RenderDistance * *w.RenderDistance)

	w.Chunks = make([]*Chunk, ChunksLength)

	Log.NewLog("Loading '" + strconv.Itoa(ChunksLength) + "' chunks")

	StartedWorldGen := time.Now()

	type WorldOutput struct {
		index int
		value *Chunk
	}

	var resultOutput = make(chan WorldOutput, FULL_CHUNK_SIZE)

	batchSize := ChunksLength / WORLD_WORKERS

	var worldAwait sync.WaitGroup
	worldAwait.Add(WORLD_WORKERS)

	for workerIndex := 0; workerIndex < WORLD_WORKERS; workerIndex++ {

		start := workerIndex * batchSize
		end := start + batchSize

		if workerIndex == WORLD_WORKERS-1 {
			end = ChunksLength
		}

		go func(start, end int) {

			defer worldAwait.Done()

			for chunkIndex := start; chunkIndex < end; chunkIndex++ {

				x := chunkIndex % CHUNK_SIZE
				y := (chunkIndex / CHUNK_SIZE) % CHUNK_SIZE

				resultOutput <- WorldOutput{
					index: chunkIndex,
					value: NewChunk(Vec3{uint32(x), 0, uint32(y)}),
				}

			}

		}(start, end)

	}

	worldAwait.Wait()
	close(resultOutput)

	for val := range resultOutput {

		val.value.Upload(shaderProgram)

		w.Chunks[val.index] = val.value

	}

	Log.NewLog("World generation took:", time.Since(StartedWorldGen))

}
