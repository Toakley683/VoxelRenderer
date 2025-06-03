#version 430 core

struct GridMetadata {
    uint R;
    uint G;
    uint B;
    uint padding;
};

struct GridNodeFlat {
    uint flags;
    int childStart;
    int size;
    GridMetadata metadata;
};

layout(std430, binding = 0) buffer NodeBuffer {
    GridNodeFlat nodes[];
};

out vec4 FragColor;

in vec2 fragTexCoord;

uniform vec2 iResolution;
uniform vec3 camPos;
uniform mat4 camera;

uniform ivec3 chunkPos;

int chunkSize = 32;

// Return true if the ray intersects an AABB, and set near/far distances
bool intersectAABB(vec3 ro, vec3 rd, vec3 bmin, vec3 bmax, out float tNear, out float tFar) {
    vec3 invDir = 1.0 / rd;
    vec3 t0 = (bmin - ro) * invDir;
    vec3 t1 = (bmax - ro) * invDir;
    vec3 tmin = min(t0, t1);
    vec3 tmax = max(t0, t1);
    tNear = max(max(tmin.x, tmin.y), tmin.z);
    tFar = min(min(tmax.x, tmax.y), tmax.z);
    return tFar >= max(tNear, 0.0);
}

// Return true if a node is a leaf (no children)
bool isLeaf(GridNodeFlat node) {
    return node.childStart == -1;
}

// Compute world-space position of a node (based on flat index math)
vec3 getNodePosition(int nodeIndex, int size) {
    int gridDim = chunkSize;
    int localIdx = nodeIndex;

    int x = localIdx % gridDim;
    int y = (localIdx / gridDim) % gridDim;
    int z = localIdx / (gridDim * gridDim);

    vec3 localPos = vec3(x, y, z) * float(size);
    return vec3(chunkPos) * float(chunkSize) + localPos;
}

bool raymarchOctree(vec3 ro, vec3 rd, out vec4 hitColor ) {
    
    const int MAX_STACK = 64;
    int stack[MAX_STACK];
    vec3 stackPos[MAX_STACK];
    int stackSize = 0;

    stack[stackSize] = 0;
    stackPos[stackSize] = getNodePosition(0, nodes[0].size);
    stackSize++;

    while (stackSize > 0) {

        stackSize--;
        
        int nodeIndex = stack[stackSize];
        vec3 nodePos = stackPos[stackSize];

        if (nodeIndex < 0 || nodeIndex >= nodes.length()) continue;

        GridNodeFlat node = nodes[nodeIndex];
        int size = node.size;
        vec3 boxMin = nodePos;
        vec3 boxMax = nodePos + vec3(size);

        float tNear, tFar;
        if (!intersectAABB(ro, rd, boxMin, boxMax, tNear, tFar)) {
            hitColor = vec4(vec3(0.4),1.0);
            return true;
        }

        if (isLeaf(node)) {
            if ((node.flags & 1u) != 0u) { 

                //float v = ( node.metadata.R + node.metadata.G + node.metadata.B ) / ( 3.0 * 256.0 )

                hitColor = vec4(vec3(node.metadata.R, node.metadata.G, node.metadata.B) / 256.0, 1.0);
                return true; // Hit found!
            } else {
                continue; // Leaf but empty voxel, ignore
            }
        }

        struct ChildEntry {
            int index;
            vec3 pos;
            float tNear;
        };
        
        ChildEntry children[8];
        int childCount = 0;
        int childStart = node.childStart;

        float tClosest = 1e30;
        bool hit = false;
        int finalCIndex;
        vec3 finalChildPos;
        
        for ( int i = 0; i < 8; i++ ) {

            int cIndex = childStart + i;
            if (cIndex < 0 || cIndex >= nodes.length()) break;

            GridNodeFlat child = nodes[cIndex];

            if ( (child.flags & 1u) == 0u ) continue;

            float size = child.size;
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
    vec3 rd = normalize((inverse(camera) * vec4(rd_view, 0.0)).xyz);

    vec4 hitColor;
    bool hit = raymarchOctree(ro, rd, hitColor );

    if ( hit ) {
        FragColor = hitColor;
    } else {
        FragColor = vec4( 0.3, 0.3, 0.3, 1.0 );
    }

}