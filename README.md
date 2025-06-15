# Voxel Renderer

This is my first test at a voxel renderer made in GoLang with **GoGL** and **GoGLFW** libraries (Using CGO) to render.

<br>

I plan to remake this project instead in C++ at a later date due to the limitations of GoLang's garbage collector.

## Settings
<br>

- The settings are all over the place, again due to this being a test environment and just understanding in what ways I could be able to render voxels at high speeds. (Will be solved in new version)
- You can look in chunk.go to see change the "IsBlockFull(x,y,z int)" function, this determines if a block is spawned at a world coordinate (X,Y,Z) is it full or not? By default it's just randomly selected to show off the performance
- types.go has the other functions such as render distance, chunk_sizes, and how many thread works are used for generating chunks and generating individual voxels.
- You can also change the scale of each voxel with the CHUNK_SIZE ( (VoxelSize) / 32f ) 

## KNOWN ISSUES

- At specific render distances some chunks may not load at all, this is likely due to the chunk hashing not generating good enough random numbers.
- Low performance with VERY low voxel counts inside each chunk. (Will be solved)

### Any problems? Contact me!

- If there are any other problems, feel free to open an issue tab and tell me about them!
- After all this was a test environment and there will be problems, please keep that in mind. These problems will be solved in my future renderer :)
