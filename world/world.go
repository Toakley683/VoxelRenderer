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

	rDistance := *w.RenderDistance

	ChunksLength := (rDistance * rDistance * rDistance)

	w.Chunks = make([]*Chunk, ChunksLength)

	Log.NewLog("Loading '" + strconv.Itoa(ChunksLength) + "' chunks")

	StartedWorldGen := time.Now()

	type WorldOutput struct {
		index int
		value *Chunk
	}

	var resultOutput = make(chan WorldOutput, ChunksLength)

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

				x := chunkIndex % rDistance
				y := (chunkIndex / rDistance) % rDistance
				z := chunkIndex / (rDistance * rDistance)

				resultOutput <- WorldOutput{
					index: chunkIndex,
					value: NewChunk(Vec3{uint32(x), uint32(y), uint32(z)}),
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
