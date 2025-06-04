package world

import (
	Log "VoxelRPG/logging"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v4.6-core/gl"
)

var (
	MainWorld *World
)

func init() {

	MainWorld = &World{
		RenderDistance: RENDER_DISTANCE_POINTER,
	}

}

func (w *World) Update(shaderProgram uint32) {

	w.UploadCombinedOctree(shaderProgram)

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

		w.Chunks[val.index] = val.value

	}

	Log.NewLog("World generation took:", time.Since(StartedWorldGen))

	w.Update(shaderProgram)

}

func (w *World) GetRootOffsets() []uint32 {

	offsets := make([]uint32, len(w.Chunks))

	for i := 0; i < len(w.Chunks); i++ {
		offsets[i] = w.Chunks[i].OctreeOffset
	}

	return offsets

}

func (w *World) UploadCombinedOctree(shaderProgram uint32) {

	gpuNodes := BuildCombinedOctreeData(w.Chunks)

	fmt.Println(unsafe.Offsetof(GridNodeFlatGPU{}.Children)) // should be multiple of 16
	fmt.Println(unsafe.Sizeof(GridNodeFlatGPU{}))

	w.UploadCombinedOctreeSSBO(gpuNodes, shaderProgram)

	runtime.GC()

}

func (w *World) GetChunkPositions() []int32 {

	output := make([]int32, len(w.Chunks)*3)

	for i := 0; i < len(w.Chunks); i++ {

		var c Chunk = *w.Chunks[i]

		Index := (i * 3) + 3

		output[Index-3] = int32(c.Position.X)
		output[Index-2] = int32(c.Position.Y)
		output[Index-1] = int32(c.Position.Z)

	}

	return output

}

func (w *World) UploadCombinedOctreeSSBO(nodes []GridNodeFlatGPU, shaderProgram uint32) {

	if w.CombinedSSBO == 0 {
		gl.GenBuffers(1, &w.CombinedSSBO)
		if w.CombinedSSBO == 0 {
			Log.NewLog("Failed to generate Combined SSBO")
		}
	}

	if w.WorldInfoSSBO == 0 {
		gl.GenBuffers(1, &w.WorldInfoSSBO)
		if w.WorldInfoSSBO == 0 {
			Log.NewLog("Failed to generate World Info SSBO")
		}
	}

	w.SendGPUBuffers(nodes, shaderProgram)
}

func (w *World) SendGPUBuffers(nodes []GridNodeFlatGPU, shaderProgram uint32) {

	if w.CombinedSSBO == 0 {
		return
	}

	ChunkInfo, ChunkNum := w.GetChunkInfo()

	/* -- [[ Send over the amount of chunks to render ]] -- */

	chunkNumberUniform := gl.GetUniformLocation(shaderProgram, gl.Str("numChunks\x00"))
	gl.Uniform1ui(chunkNumberUniform, uint32(ChunkNum))

	/* -- [[ Send over the chunk information itself ]] -- */

	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, w.CombinedSSBO)

	gl.BufferData(
		gl.SHADER_STORAGE_BUFFER,
		len(nodes)*int(unsafe.Sizeof(nodes[0])),
		gl.Ptr(nodes), gl.STATIC_DRAW,
	)

	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 0, w.CombinedSSBO)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, 0)

	/* -- [[ Send over the Chunk Info (Index Offsets/Chunk Positions) itself ]] -- */

	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, w.WorldInfoSSBO)

	gl.BufferData(
		gl.SHADER_STORAGE_BUFFER,
		len(ChunkInfo)*int(unsafe.Sizeof(ChunkInfo[0])),
		gl.Ptr(ChunkInfo), gl.STATIC_DRAW,
	)

	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 1, w.WorldInfoSSBO)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, 0)

}

func (w *World) GetChunkInfo() ([]ChunkInfo, uint32) {

	RootOffsets := w.GetRootOffsets()
	chunkPositions := w.GetChunkPositions()
	ChunkNum := uint32(len(RootOffsets))

	chunkInformation := make([]ChunkInfo, ChunkNum)

	for i := 0; i < int(ChunkNum); i++ {

		Index := (i * 3) + 3

		chunkInformation[i].ChunkPos[0] = chunkPositions[Index-3]
		chunkInformation[i].ChunkPos[1] = chunkPositions[Index-2]
		chunkInformation[i].ChunkPos[2] = chunkPositions[Index-1]
		chunkInformation[i].RootOffset = RootOffsets[i]
	}

	return chunkInformation, ChunkNum

}

func (chunk *Chunk) UpdateSSBO() {

}
