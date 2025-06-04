package world

import "github.com/go-gl/mathgl/mgl32"

// Plane represents a plane equation Ax + By + Cz + D = 0
type Plane struct {
	Normal mgl32.Vec3
	D      float32
}

// ExtractFrustumPlanes extracts the 6 planes (left, right, top, bottom, near, far) from a viewProjection matrix
func ExtractFrustumPlanes(m mgl32.Mat4) [6]Plane {
	var planes [6]Plane

	// Left
	planes[0].Normal = m.Row(3).Vec3().Add(m.Row(0).Vec3())
	planes[0].D = m.At(3, 3) + m.At(0, 3)

	// Right
	planes[1].Normal = m.Row(3).Vec3().Sub(m.Row(0).Vec3())
	planes[1].D = m.At(3, 3) - m.At(0, 3)

	// Bottom
	planes[2].Normal = m.Row(3).Vec3().Add(m.Row(1).Vec3())
	planes[2].D = m.At(3, 3) + m.At(1, 3)

	// Top
	planes[3].Normal = m.Row(3).Vec3().Sub(m.Row(1).Vec3())
	planes[3].D = m.At(3, 3) - m.At(1, 3)

	// Near
	planes[4].Normal = m.Row(3).Vec3().Add(m.Row(2).Vec3())
	planes[4].D = m.At(3, 3) + m.At(2, 3)

	// Far
	planes[5].Normal = m.Row(3).Vec3().Sub(m.Row(2).Vec3())
	planes[5].D = m.At(3, 3) - m.At(2, 3)

	// Normalize planes
	for i := range planes {
		n := planes[i].Normal
		length := n.Len()
		planes[i].Normal = n.Mul(1.0 / length)
		planes[i].D /= length
	}

	return planes
}

// aabbIntersectsPlane returns true if the AABB intersects or is in front of the plane
func aabbIntersectsPlane(min, max mgl32.Vec3, plane Plane) bool {
	// Calculate positive vertex (farthest point in direction of plane normal)
	var pVertex mgl32.Vec3
	if plane.Normal.X() >= 0 {
		pVertex[0] = max[0]
	} else {
		pVertex[0] = min[0]
	}

	if plane.Normal.Y() >= 0 {
		pVertex[1] = max[1]
	} else {
		pVertex[1] = min[1]
	}

	if plane.Normal.Z() >= 0 {
		pVertex[2] = max[2]
	} else {
		pVertex[2] = min[2]
	}

	// Distance from plane to positive vertex
	distance := plane.Normal.Dot(pVertex) + plane.D

	// If positive vertex is behind plane, the AABB is fully outside
	return distance >= 0
}
