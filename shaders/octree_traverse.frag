#version 430 core


out vec4 FragColor;


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

struct ChunkInfo {
    ivec3 ChunkPos;
    uint RootOffset;
};

layout(std430, binding = 0) buffer NodeBuffer {
    GridNodeFlat nodes[];
};

uniform uint numChunks;
uniform float chunkSize;
uniform float chunkScale;

layout(std430, binding = 1) buffer ChunkInfoBuffer {
    ChunkInfo chunksInformation[];
};

uint MaxUINT32 = 0xFFFFFFFFu;

in vec2 fragTexCoord;

uniform vec2 iResolution;
uniform vec3 camPos;
uniform mat4 invView;

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

// Return true if a node is a leaf (no children)

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

bool raymarchOctree(vec3 ro, vec3 rd, uint chunkIndex, out vec4 hitColor ) {
    
    const int MAX_STACK = 64;
    uint stack[MAX_STACK];
    vec3 stackPos[MAX_STACK];
    int stackSize = 0;
    
    int rootNodeIndex = int(chunksInformation[chunkIndex].RootOffset);

    stack[stackSize] = rootNodeIndex;
    stackPos[stackSize] = vec3(chunksInformation[chunkIndex].ChunkPos) * float(chunkSize * chunkScale);
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

        float tNear, tFar;
        if (!intersectAABB(ro, rd, boxMin, boxMax, tNear, tFar)) {
            continue;
        }

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

            //if ( !flagInfo.occupied ) continue;

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

    return false; // background color
}

void main() {
    
    vec2 uv = fragTexCoord * 2.0 - 1.0;
    uv.x *= iResolution.x / iResolution.y;
    
    vec3 ro = camPos;

    vec4 rayClip = vec4(uv, -1.0, 1.0);

    vec3 rd_view = normalize(vec3(uv, -1.0));  // ray direction in view space
    vec3 rd = normalize((invView * vec4(rd_view, 0.0)).xyz);

    float closestT = 1e30;
    vec4 finalColor = vec4(0.0);
    bool foundHit = false;

    for (uint chunkIndex = 0u; chunkIndex < numChunks; ++chunkIndex) {
        vec3 chunkWorldPos = vec3(chunksInformation[chunkIndex].ChunkPos) * float(chunkSize*chunkScale);
        vec3 bmin = chunkWorldPos;
        vec3 bmax = chunkWorldPos + vec3(chunkSize*chunkScale);

        float tNear, tFar;
        if (!intersectAABB(ro, rd, bmin, bmax, tNear, tFar)) {
            continue;
        }

        if (tNear >= closestT) {
            continue;
        }

        vec4 hitColor;
        bool hit = raymarchOctree(ro, rd, chunkIndex, hitColor);
        if (hit && tNear < closestT) {
            closestT = tNear;
            finalColor = hitColor;
            foundHit = true;
        }
    }

    if (foundHit) {
        FragColor = finalColor;
    } else {
        FragColor = vec4( vec3(0.3), 1.0 );
    }

}