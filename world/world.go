package world

import (
	Log "VoxelRPG/logging"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type OctreeChanInput struct {
	Input *Chunk
	Value []GridNodeFlatGPU
}

var (
	MainWorld            *World
	CombinedOctree       map[*Chunk][]GridNodeFlatGPU = map[*Chunk][]GridNodeFlatGPU{}
	CombinedOctreeLength uint32                       = 0

	OctreeNodeByteSize int

	CombinedOctreeChan = make(chan OctreeChanInput)
)

func init() {

	var dummySize GridNodeFlatGPU
	OctreeNodeByteSize = int(unsafe.Sizeof(dummySize))

	MainWorld = &World{
		RenderDistance: RENDER_DISTANCE_POINTER,
	}

	var LastCombinedOctreeLength uint32 = 0

	go func() {
		for {
			for Data := range CombinedOctreeChan {

				CombinedOctree[Data.Input] = Data.Value
				Data.Input.OctreeOffset = uint32((len(CombinedOctree) - 1) * len(CombinedOctree[Data.Input]))
				CombinedOctreeLength += uint32(len(CombinedOctree[Data.Input]))

			}

			if CombinedOctreeLength != LastCombinedOctreeLength {

				Log.NewLog("Updated Octree, New Size:", CombinedOctreeLength)

			}
		}
	}()

}

func (w *World) Update(shaderProgram uint32) {

	w.UploadCombinedOctree(shaderProgram)

}

func (w *World) UpdateIfNeeded(shaderProgram uint32, viewProjection mgl32.Mat4, cameraPos mgl32.Vec3) {
	currentChunk := GetCameraChunk(cameraPos)

	if currentChunk == w.LastCameraChunk {
		return // Skip expensive update
	}

	Log.NewLog("New Camera ChunkPos:", currentChunk, "Pos:", cameraPos)

	w.LastCameraChunk = currentChunk
	w.Update(shaderProgram)
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
					value: NewChunk(Vec3{int32(x), int32(y), int32(z)}),
				}

			}

		}(start, end)

	}

	go func() {
		worldAwait.Wait()
		close(resultOutput)
	}()

	for val := range resultOutput {

		/* -- [[ Upload the Chunk Octree data to SSBO ]] -- */
		val.value.Upload()

		w.Chunks[val.index] = val.value

	}

	Log.NewLog("World generation took:", time.Since(StartedWorldGen))

}

func (w *World) GetRootOffsets() []uint32 {

	offsets := make([]uint32, len(w.Chunks))

	for i := 0; i < len(w.Chunks); i++ {
		offsets[i] = w.Chunks[i].OctreeOffset
	}

	return offsets

}

func (w *World) UploadCombinedOctree(shaderProgram uint32) {

	//gpuNodes := BuildCombinedOctreeData(w.Chunks)

	w.UploadCombinedOctreeSSBO(shaderProgram)

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

func (w *World) UploadCombinedOctreeSSBO(shaderProgram uint32) {

	if w.WorldInfoSSBO == 0 {
		gl.GenBuffers(1, &w.WorldInfoSSBO)
		if w.WorldInfoSSBO == 0 {
			Log.NewLog("Failed to generate World Info SSBO")
		}
	}

	if w.WorldInfoOffsetsSSBO == 0 {
		gl.GenBuffers(1, &w.WorldInfoOffsetsSSBO)
		if w.WorldInfoOffsetsSSBO == 0 {
			Log.NewLog("Failed to generate World Info Offset SSBO")
		}
	}

	w.SendGPUBuffers(shaderProgram)
}

func (w *World) SendGPUBuffers(shaderProgram uint32) {

	if w.CombinedSSBO == 0 {
		return
	}

	offsets, chunkInfo, numBuckets := w.GetChunkInfo()

	/* -- [[ Send over the amount of chunks to render ]] -- */

	gl.Uniform1ui(gl.GetUniformLocation(shaderProgram, gl.Str("numChunks\x00")), uint32(len(w.Chunks)))
	gl.Uniform1f(gl.GetUniformLocation(shaderProgram, gl.Str("chunkSize\x00")), float32(CHUNK_SIZE))
	gl.Uniform1f(gl.GetUniformLocation(shaderProgram, gl.Str("chunkScale\x00")), CHUNK_SCALE)
	gl.Uniform1i(gl.GetUniformLocation(shaderProgram, gl.Str("numBuckets\x00")), int32(numBuckets))

	/* -- [[ Send over the chunk information itself ]] -- */

	/* -- [[ Send over the Chunk Info (Chunk Key & Chunk Root Offsets) itself ]] -- */

	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, w.WorldInfoSSBO)

	gl.BufferData(
		gl.SHADER_STORAGE_BUFFER,
		len(chunkInfo)*int(unsafe.Sizeof(chunkInfo[0])),
		gl.Ptr(chunkInfo), gl.STATIC_DRAW,
	)

	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 1, w.WorldInfoSSBO)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, 0)

	/* -- [[ Send over the Chunk Info (Offsets) itself ]] -- */

	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, w.WorldInfoOffsetsSSBO)

	gl.BufferData(
		gl.SHADER_STORAGE_BUFFER,
		len(offsets)*int(unsafe.Sizeof(offsets[0])),
		gl.Ptr(offsets), gl.STATIC_DRAW,
	)

	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 2, w.WorldInfoOffsetsSSBO)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, 0)

}

func (w *World) GetChunkInfo() ([]uint32, []MapEntry, int) {

	Displacement, MapEntries, err := BuildPerfectHashTable(w.Chunks)

	if err != nil {
		panic("GetChunkInfo() error:" + err.Error())
	}

	return Displacement, MapEntries, len(w.Chunks)

}

func (chunk *Chunk) UpdateSSBO() {

}
