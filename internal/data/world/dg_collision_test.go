package world

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

// dgBytes builds a one-node DG: header box, no names, an unused i32, a root
// marker, then the node with one display group of a single triangle.
func dgBytes(authored bool) []byte {
	var b bytes.Buffer
	i32 := func(v int32) { binary.Write(&b, binary.LittleEndian, v) }
	f32 := func(v float32) { binary.Write(&b, binary.LittleEndian, math.Float32bits(v)) }
	for range 6 {
		f32(0) // header box
	}
	i32(0) // names
	i32(0)
	i32(1)                // root present
	for range 3 + 6 + 6 { // vec3 and two boxes
		f32(0)
	}
	i32(1) // groups
	i32(7)
	i32(1) // sub parts
	i32(9)
	i32(3) // display vertices, 44 bytes each
	for _, v := range [][3]float32{{0, 5, 0}, {4, 5, 0}, {0, 5, 4}} {
		f32(v[0])
		f32(v[1])
		f32(v[2])
		for range 8 {
			f32(0)
		}
	}
	i32(1) // display faces
	for _, idx := range []uint16{0, 1, 2} {
		binary.Write(&b, binary.LittleEndian, idx)
	}
	if authored {
		i32(1)
		i32(3)
		for _, v := range [][3]float32{{0, 1, 0}, {2, 1, 0}, {0, 1, 2}} {
			f32(v[0])
			f32(v[1])
			f32(v[2])
		}
		i32(1)
		for _, idx := range []uint16{0, 1, 2} {
			binary.Write(&b, binary.LittleEndian, idx)
		}
	} else {
		i32(0)
	}
	for range 8 {
		i32(0) // children
	}
	return b.Bytes()
}

// Game.exe 0x5a4990-0x5a4a80: a node with authored collision keeps it, and one
// without borrows the triangles of its display groups.
func TestDGNodeWithoutCollisionUsesDisplayTriangles(t *testing.T) {
	meshes, err := ParseDGCollisions(dgBytes(false))
	if err != nil || len(meshes) != 1 || len(meshes[0].Faces) != 1 || meshes[0].Vertices[0][1] != 5 {
		t.Fatalf("display fallback: %v %+v", err, meshes)
	}
	if meshes[0].Bounds.Max[0] != 4 || meshes[0].Bounds.Max[2] != 4 {
		t.Fatalf("bounds %+v", meshes[0].Bounds)
	}
	meshes, err = ParseDGCollisions(dgBytes(true))
	if err != nil || len(meshes) != 1 || meshes[0].Vertices[0][1] != 1 {
		t.Fatalf("authored collision replaced: %v %+v", err, meshes)
	}
}
