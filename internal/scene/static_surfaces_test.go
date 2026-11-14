package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"testing"
)

func sheetMesh(backDepth float32) world.SMODMesh {
	m := world.SMODMesh{Texture: "authored.dds"}
	for _, z := range []float32{0, backDepth} {
		for _, xy := range [][2]float32{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
			m.Vertices = append(m.Vertices, world.Vertex{Pos: world.Vec3{xy[0], xy[1], z}, UV: xy})
		}
	}
	// Opposite sheets use different diagonals; exact triangle dedup cannot find them.
	m.Faces = []world.Face{{0, 1, 2}, {0, 2, 3}, {5, 4, 7}, {5, 7, 6}}
	return m
}

func TestStaticSurfacePairsFindOppositeRetriangulatedSheets(t *testing.T) {
	m := sheetMesh(0)
	flags := coincidentStaticFaces(m)
	for i, p := range flags {
		if !p {
			t.Fatal("missed coplanar opposite face", i)
		}
	}
	m.Faces = m.Faces[:2]
	for _, p := range coincidentStaticFaces(m) {
		if p {
			t.Fatal("unpaired foliage sheet classified as solid")
		}
	}
	m = sheetMesh(.001)
	for _, p := range coincidentStaticFaces(m) {
		if p {
			t.Fatal("separate parallel walls classified as coincident")
		}
	}
	m = sheetMesh(0)
	m.Faces[2], m.Faces[3] = world.Face{4, 5, 7}, world.Face{5, 6, 7}
	for _, p := range coincidentStaticFaces(m) {
		if p {
			t.Fatal("same-facing overlay was removed")
		}
	}
}

func TestStaticSurfaceSplitPreservesSourceUVsNormalsAndCollision(t *testing.T) {
	m := sheetMesh(0)
	m.Vertices = append(m.Vertices, world.Vertex{Pos: world.Vec3{3, 0, 0}}, world.Vertex{Pos: world.Vec3{4, 0, 0}}, world.Vertex{Pos: world.Vec3{3, 1, 0}})
	m.Faces = append(m.Faces, world.Face{8, 9, 10})
	source := &world.SMOD{Meshes: []world.SMODMesh{m}, Radius: 5, Colliders: []world.CollisionMesh{{Faces: []world.Face{{0, 1, 2}}}}}
	prepared, cull := prepareStaticSurfaces(source)
	if len(prepared.Meshes) != 2 || cull[0] || !cull[1] || len(prepared.Meshes[0].Vertices) != 3 || len(prepared.Meshes[1].Vertices) != 8 {
		t.Fatal("bad mixed-part split", prepared.Meshes, cull)
	}
	if len(source.Meshes[0].Faces) != 5 || len(source.Meshes[0].Vertices) != 11 || prepared.Radius != source.Radius || &prepared.Colliders[0] != &source.Colliders[0] {
		t.Fatal("source/collision data changed")
	}
	counts := map[[3]world.Vertex]int{}
	for _, face := range m.Faces {
		counts[[3]world.Vertex{m.Vertices[face[0]], m.Vertices[face[1]], m.Vertices[face[2]]}]++
	}
	for _, part := range prepared.Meshes {
		if part.Texture != m.Texture {
			t.Fatal("material changed")
		}
		for _, face := range part.Faces {
			key := [3]world.Vertex{part.Vertices[face[0]], part.Vertices[face[1]], part.Vertices[face[2]]}
			counts[key]--
		}
	}
	for _, n := range counts {
		if n != 0 {
			t.Fatal("lost or changed authored face attributes")
		}
	}
}
