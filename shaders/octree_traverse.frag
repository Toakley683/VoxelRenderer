#version 450 core

/* -- [[ Shader Inputs ]] -- */

in vec2 fragTexCoord;

/* -- [[ Shader Outputs ]] -- */

out vec4 FragColor;

/* -- [[ Octree Traversal Grid SSBO ]] -- */

struct GridMetadata {
    uint R;
    uint G;
    uint B;
    uint padding;
};

struct GridNodeFlat {
    uint[8] children;
    uint flags;
    int size;
    uint padding[2];
    GridMetadata metadata;
};


layout(std430, binding = 0) buffer NodeBuffer {
    GridNodeFlat nodes[];
};

/* -- [[ Chunk Info & Hashmap Offsets SSBO ]] -- */

struct ChunkInfo {
    ivec3 Position;
    uint offset;
};

layout(std430, binding = 1) buffer ChunkInfoBuffer {
    ChunkInfo chunkInfo[];
};

layout(std430, binding = 2) buffer Offsets {
    uint displacements[];
};

/*layout(std430, binding = 3) buffer DebugResult {
    vec3 debugOutput;
};*/

/* -- [[ World Variables ]] -- */

uniform uint numChunks;
uniform float chunkSize;
uniform float chunkScale;

/* -- [[ Shader Variables ]] -- */

uniform vec2 iResolution;
uniform float fov;
uniform vec3 camPos;
uniform mat4 invView;

float resolutionScale = iResolution.y / tan(fov / 2.0);

/* -- [[ Hashmap Variables ]] -- */

uniform int numBuckets;

/* -- [[ Grid Map Variables ]] -- */

uint MAX_STEPS = 24;

struct FaceHit {
    vec3 normal;
    vec3 position;
};

/* -- [[ Global Variables ]] -- */

uint MaxUINT32 = 0xFFFFFFFFu;
float EPSILON = 1e-4;

/* -- [[ Hashmap Functions ]] -- */

uint hash_u32(uint x) {
    x ^= x >> 16u;
    x *= 0x7feb352du;
    x ^= x >> 15u;
    x *= 0x846ca68bu;
    x ^= x >> 16u;
    return x;
}

uint hash3D(ivec3 v) {
    // Convert signed to unsigned (e.g. offset to positive range)
    uvec3 uv = uvec3(v) + uvec3(0x80000000u);

    // Mix the three components into one uint hash
    uint h = hash_u32(uv.x);
    h ^= hash_u32(uv.y) + 0x9e3779b9u + (h << 6u) + (h >> 2u);
    h ^= hash_u32(uv.z) + 0x9e3779b9u + (h << 6u) + (h >> 2u);

    return h;
}

// Example hash functions - MUST match your CPU ones exactly
uint hash1(ivec3 pos) {
    return hash3D(pos);
}

uint hash2(ivec3 pos) {
    return hash3D(pos * ivec3(0x27d4eb2du, 0x165667b1u, 0x1b873593u));
}



ChunkInfo lookupRootOffset(ivec3 chunkPos) {
    int N = int(chunkInfo.length());

    uint h1Val = hash1(chunkPos) % uint(N);
    uint h2Val = hash2(chunkPos) % uint(N);

    uint idx = (h1Val + displacements[h2Val]) % uint(N);

    ChunkInfo result;
    result.offset = 0xFFFFFFFFu;  // MaxUInt32 default for not found
    result.Position = ivec3(0);

    ChunkInfo entry = chunkInfo[idx];

    if (entry.offset == 0xFFFFFFFFu) {
        return result; // Not found
    }

    // Verify position to avoid false positives
    if (entry.Position != chunkPos) {
        return result; // No match
    }

    result.Position = chunkPos;
    result.offset = entry.offset;

    return result;
}

/* -- [[ Octree Traversal ]] -- */

/* -- [[ Ray Intersection With AABB ]] -- */

// Return true if the ray intersects an AABB, and set near/far distances
bool intersectAABB(vec3 ro, vec3 rd, vec3 bmin, vec3 bmax, out float tNear, out float tFar) {
    vec3 invDir = 1.0 / rd;
    vec3 t0 = (bmin - ro) * invDir;
    vec3 t1 = (bmax - ro) * invDir;
    vec3 tmin = min(t0, t1);
    vec3 tmax = max(t0, t1);
    tNear = max(max(tmin.x, tmin.y), tmin.z);
    tFar = min(min(tmax.x, tmax.y), tmax.z);
    return (tFar >= max(tNear, 0.0));
}

/* -- [[ Octree Leaf Decoding ]] -- */

struct FlagBits {
    bool occupied;
    bool leaf;
};

FlagBits DecodeFlags(uint flags) {
    FlagBits result;
    result.occupied = (flags & 1u) != 0u;     // bit 0
    result.leaf = (flags & 2u) != 0u;         // bit 1
    return result;
}

/* -- [[ Octree Traversal Function ]] -- */

bool raymarchOctree(vec3 ro, vec3 rd, ChunkInfo rootNode, out vec4 hitColor ) {
    
    const int MAX_STACK = 64;
    uint stack[MAX_STACK];
    vec3 stackPos[MAX_STACK];
    int stackSize = 0;
    
    int rootNodeIndex = int(rootNode.offset);

    stack[stackSize] = rootNodeIndex;
    stackPos[stackSize] = vec3(rootNode.Position) * float(chunkSize * chunkScale);
    stackSize++;

    while (stackSize > 0) {

        stackSize--;
        
        uint nodeIndex = stack[stackSize];
        vec3 nodePos = stackPos[stackSize];

        if (nodeIndex >= nodes.length()) continue;

        GridNodeFlat node = nodes[nodeIndex];
        float size = node.size;
        vec3 boxMin = nodePos;
        vec3 boxMax = nodePos + vec3(size * chunkScale);

        /* -- [[ LOD Cutoff (Projected Size) ]] -- */
        
        float voxelSize = node.size * chunkScale;
        float distance = length(camPos - ((boxMin + boxMax) / 2 ));
        float screenSpaceSize = (voxelSize / distance) * resolutionScale;

        if (screenSpaceSize < 1) {
            hitColor = vec4(vec3(node.metadata.R, node.metadata.G, node.metadata.B) / 256.0, 1.0);
            return true;
        }

        /* -- [[ Check if node has any children ]] -- */

        FlagBits flagInfo = DecodeFlags( node.flags);
        
        if (!flagInfo.occupied) { 
            continue;
        }

        if (flagInfo.leaf) {
            if (flagInfo.occupied) { 

                //float v = ( node.metadata.R + node.metadata.G + node.metadata.B ) / ( 3.0 * 256.0 )

                hitColor = vec4(vec3(node.metadata.R, node.metadata.G, node.metadata.B) / 256.0, 1.0);
                return true; // Hit found!
            }
        }

        struct ChildEntry {
            uint index;
            vec3 pos;
            float tNear;
        };
        
        ChildEntry children[8];
        int childCount = 0;

        float tClosest = 1e30;
        bool hit = false;
        int finalCIndex;
        vec3 finalChildPos;
        
        for ( int i = 0; i < 8; i++ ) {

            uint cIndex = node.children[i];

            if ( cIndex == MaxUINT32 ) continue;

            GridNodeFlat child = nodes[cIndex];
            
            FlagBits flagInfo = DecodeFlags( child.flags);


            float size = child.size * chunkScale;
            vec3 childPos = nodePos + vec3(i & 1, (i >> 1) & 1, (i >> 2) & 1) * size; 

            vec3 minSize = childPos;
            vec3 maxSize = childPos + vec3( size );

            float tNear, tFar;
            if ( intersectAABB( ro, rd, minSize, maxSize, tNear, tFar )) {
                children[childCount] = ChildEntry(cIndex, childPos, tNear);
                childCount++;
            }

        }

        for (int a = 0; a < childCount-1; a++) {
            for (int b = a+1; b < childCount; b++) {
                if (children[b].tNear < children[a].tNear) {
                    ChildEntry tmp = children[a];
                    children[a] = children[b];
                    children[b] = tmp;
                }
            }
        }

        for (int i = childCount - 1; i >= 0; i--) {
            if (stackSize >= MAX_STACK) break; // Prevent overflow

            stack[stackSize] = children[i].index;
            stackPos[stackSize] = children[i].pos;
            stackSize++;
        }

    }

    return false;
}


/* -- [[ Grid Map Traversal ]] -- */

ivec3 getChunkPosition( vec3 Position ) {

    ivec3 val;
    float cScale = chunkSize * chunkScale;

    val.x = int(floor( float(Position.x) / cScale ));
    val.y = int(floor( float(Position.y) / cScale ) );
    val.z = int(floor( float(Position.z) / cScale ));

    return val;

}

FaceHit getVoxelFaceHit(vec3 ro, vec3 rd ) {

    float cScale = chunkSize * chunkScale;

    ivec3 voxel = ivec3(floor(ro / cScale));
    ivec3 step = ivec3(sign(rd));
    ivec3 positiveStep = ivec3(greaterThan(step, ivec3(0)));

    vec3 chunkMin = vec3(voxel) * cScale;
    vec3 chunkMax = chunkMin + vec3(cScale);

    vec3 t1 = (chunkMin - ro) / rd;
    vec3 t2 = (chunkMax - ro) / rd;

    vec3 tMin = min(t1, t2);
    vec3 tMax = max(t1, t2);

    // Avoid div-by-zero in tDelta
    vec3 tDelta = vec3(
        rd.x != 0.0 ? cScale / abs(rd.x) : 1e30,
        rd.y != 0.0 ? cScale / abs(rd.y) : 1e30,
        rd.z != 0.0 ? cScale / abs(rd.z) : 1e30
    );

    FaceHit hit;

    if (tMax.x < tMax.y && tMax.x < tMax.z) {
        hit.normal = vec3(float(step.x), 0.0, 0.0);
        hit.position = ro + rd * (tMax.x);
    } else if (tMax.y < tMax.z) {
        hit.normal = vec3(0.0, float(step.y), 0.0);
        hit.position = ro + rd * (tMax.y);
    } else {
        hit.normal = vec3(0.0, 0.0, float(step.z));
        hit.position = ro + rd * (tMax.z);
    }

    return hit;
}

bool traverseChunks( in vec3 ro, in vec3 rd, out vec4 finalColor ) { 

    float cScale = chunkSize * chunkScale;

    vec3 origin = ro;
    ivec3 currentChunk = getChunkPosition(origin);

    for (int i = 0; i < MAX_STEPS; i++) {

        ChunkInfo f = lookupRootOffset( currentChunk );

        if ( f.offset != MaxUINT32 ) {

            FlagBits flagInfo = DecodeFlags( nodes[f.offset].flags);
            
            if (flagInfo.occupied) { 
                
                bool hit = raymarchOctree(ro, rd, f, finalColor);

                if (hit == true) {
                    return true;
                }

            }
        }

        FaceHit hit = getVoxelFaceHit(origin,rd);
        currentChunk = currentChunk + ivec3(hit.normal);

        origin = hit.position + ( hit.normal * ( EPSILON * cScale ) );

    }

    return false;
}

/* -- [[ Main function ]] -- */

void main() {
    
    vec2 uv = fragTexCoord * 2.0 - 1.0;
    uv.x *= iResolution.x / iResolution.y;
    
    vec3 ro = camPos;

    vec4 rayClip = vec4(uv, -1.0, 1.0);

    vec3 rd_view = normalize(vec3(uv, -1.0));  // ray direction in view space
    vec3 rd = normalize((invView * vec4(rd_view, 0.0)).xyz);

    float closestT = 1e30;
    vec4 finalColor = vec4(0.0);

    bool hit = traverseChunks(ro, rd, finalColor );

    if (hit) {
        FragColor = finalColor;
    } else {
        FragColor = vec4( vec3(0.3), 1.0 );
    }

}