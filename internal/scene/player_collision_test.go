package scene

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// A visible wall with an optional separately authored collision triangle.
func collisionSMOD(solid bool) []byte {
	var b bytes.Buffer
	write := func(v any) { _ = binary.Write(&b, binary.LittleEndian, v) }
	write([10]float32{}) // sphere + view bounds
	write(int32(1))
	write(int32(1))
	b.WriteByte(0) // material name
	write(int32(3))
	points := [][3]float32{{0, -10, -10}, {0, 0, 10}, {0, 10, -10}}
	for _, p := range points {
		write(p)
		write([6]float32{})
	}
	write(int32(1))
	write([3]uint16{0, 1, 2})
	write([6]float32{}) // Collision box must never become a solid substitute.
	if solid {
		write(int32(1))
		write(int32(3))
		for _, p := range points {
			write(p)
		}
		write(int32(1))
		write([3]uint16{0, 1, 2})
	} else {
		write(int32(0))
	}
	return b.Bytes()
}
func collisionVANI() []byte {
	var b bytes.Buffer
	write := func(v any) { _ = binary.Write(&b, binary.LittleEndian, v) }
	write([10]float32{})
	write(int32(1))
	write(int32(1))
	write(int32(33))
	write(int32(1))
	b.WriteByte(0)
	write(int32(1))
	write([3]uint16{0, 1, 2})
	write(int32(3))
	for _, p := range [][3]float32{{0, -10, -10}, {0, 0, 10}, {0, 10, -10}} {
		write(p)
		write([6]float32{})
	}
	write([6]float32{-.1, -10, -10, .1, 10, 10})
	write(int32(0))
	return b.Bytes()
}
func TestOriginalCollisionGroupConstructionPolicies(t *testing.T) {
	root := t.TempDir()
	put := func(name string, data []byte) {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	put("Entity/Shape/solid.smod", collisionSMOD(true))
	put("Entity/Shape/decoration.smod", collisionSMOD(false))
	put("Entity/Grass/solid.smod", collisionSMOD(true))
	put("Entity/VAni/wall.vani", collisionVANI())
	arc, err := archive.OpenDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	instance := func(id int32, x float32) world.Instance {
		return world.Instance{Asset: id, Pos: world.Vec3{x, 0, 0}, Fwd: world.Vec3{0, 0, 1}, Up: world.Vec3{0, 1, 0}}
	}
	w := &World{WLD: &world.WLD{
		Shapes: world.Group{Assets: []string{"solid.smod", "decoration.smod"}, Instances: []world.Instance{instance(0, 0), instance(1, 20)}},
		Grass:  world.Group{Assets: []string{"solid.smod"}, Instances: []world.Instance{instance(0, 10)}},
		VaniA:  world.Group{Assets: []string{"wall.vani"}, Instances: []world.Instance{instance(0, 30)}},
		VaniB:  world.Group{Assets: []string{"wall.vani"}, Instances: []world.Instance{instance(0, 40)}},
	}}
	if err := w.prepareCollisions(NewAssets(arc)); err != nil {
		t.Fatal(err)
	}
	if w.collision.Len() != 14 {
		t.Fatalf("unexpected solid geometry: %d triangles", w.collision.Len())
	}
	for _, tc := range []struct {
		x     float32
		solid bool
	}{{0, true}, {10, false}, {20, true}, {30, true}, {40, false}} {
		p, hit := w.MovePlayer(render.V3{tc.x - 3, 0, 0}, render.V3{6, 0, 0})
		if hit != tc.solid || tc.solid && p[0] >= tc.x || !tc.solid && p[0] != tc.x+3 {
			t.Errorf("x=%v: position %v hit=%v; solid=%v", tc.x, p, hit, tc.solid)
		}
	}
	// A renderer can evict everything without changing any query result.
	w.Objects = nil
	w.stream = nil
	p, hit := w.MovePlayer(render.V3{-3, 0, 0}, render.V3{60, 0, 0})
	if !hit || p[0] >= 0 {
		t.Fatal("fast motion tunneled through an unloaded wall", p)
	}
	count := 0
	w.CollisionTriangles(render.V3{}, 5, func(collision.Triangle) { count++ })
	if count != 1 {
		t.Fatal("viewer does not share the movement geometry", count)
	}
}

func TestMalformedCollisionDoesNotBecomeSilentNoclip(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "Entity/Shape")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	data := collisionSMOD(true)
	binary.LittleEndian.PutUint16(data[len(data)-2:], 500)
	if err := os.WriteFile(filepath.Join(directory, "broken.smod"), data, 0600); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	w := &World{WLD: &world.WLD{Shapes: world.Group{Assets: []string{"broken.smod"}, Instances: []world.Instance{{Fwd: world.Vec3{0, 0, 1}, Up: world.Vec3{0, 1, 0}}}}}}
	if err := w.prepareCollisions(NewAssets(arc)); err == nil {
		t.Fatal("invalid collision was silently accepted")
	}
	if w.HasPlayerCollision() {
		t.Fatal("partially constructed collision world was published")
	}
}
