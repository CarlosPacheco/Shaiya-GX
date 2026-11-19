package scene

import (
	"runtime"
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestCharacterClothBatchMatchesSerial(t *testing.T) {
	previous := runtime.GOMAXPROCS(4)
	defer runtime.GOMAXPROCS(previous)

	rest := []render.V3{{0, 1, 0}, {1, 1, 0}, {0, 0, 0}, {1, 0, 0}}
	clip := poseTestClip(2)
	geometry := &cloakGeometry{
		record:   model.CloakRecord{Columns: 2, Rows: 2},
		vertices: make([]render.Vertex, len(rest)),
		indices:  []uint32{0, 2, 1, 1, 2, 3},
	}
	for i, p := range rest {
		geometry.vertices[i] = render.Vertex{Pos: p, Normal: render.V3{0, 0, 1}}
	}
	makeActor := func(i int) *Character {
		world := render.Identity()
		world[12] = float32(i)
		cloak := &cloakInstance{geometry: geometry, physics: newClothPhysics(rest, 2, 2, i%2), vertices: slices.Clone(geometry.vertices)}
		cloak.physics.pins[0] = true
		return &Character{clip: clip, World: world, Time: float32(i) / 60, cloak: cloak}
	}
	batch := make([]*Character, 48)
	serial := make([]*Character, len(batch))
	for i := range batch {
		batch[i], serial[i] = makeActor(i), makeActor(i)
	}
	for frame := range 6 {
		for _, actors := range [][]*Character{batch, serial} {
			for i, c := range actors {
				c.World[14] = float32(frame+i) * .1
				c.Update(1.0 / 60)
			}
		}
		PrepareCharacterCloth(batch)
		for i, c := range serial {
			c.cloak.update(c.World, c.finals())
			got := batch[i].cloak
			if !got.prepared || !slices.Equal(got.vertices, c.cloak.vertices) ||
				!slices.Equal(got.physics.positions, c.cloak.physics.positions) ||
				!slices.Equal(got.physics.velocity, c.cloak.physics.velocity) {
				t.Fatalf("actor %d frame %d differs from serial cloth", i, frame)
			}
		}
	}
}

func TestCrowdClothQualityUsesProjectedSize(t *testing.T) {
	rest := []render.V3{{0, 1, 0}, {1, 1, 0}, {0, 0, 0}, {1, 0, 0}}
	vertices := make([]render.Vertex, len(rest))
	c := &Character{clip: poseTestClip(1), World: render.Identity(), cloak: &cloakInstance{
		geometry: &cloakGeometry{vertices: vertices},
		vertices: slices.Clone(vertices),
		physics:  newClothPhysics(rest, 2, 2, 0),
	}}
	c.World[14] = 60
	s := &render.Scene3D{View: render.Identity(), Proj: render.PerspectiveFovLH(.8, 1, .1, 120), ViewportH: 720}
	c.prepareCrowdCloak(s)
	c.cloak.lodInterval, c.cloak.lodAt = 1.0/20, 1.0/20
	c.Update(1.0 / 60)
	c.prepareCrowdCloak(s)
	if !c.cloak.reused || c.cloak.elapsed == 0 {
		t.Fatal("small cape did not reuse the last mesh")
	}
	c.Update(1.0 / 20)
	c.prepareCrowdCloak(s)
	if c.cloak.reused || c.cloak.elapsed != 0 {
		t.Fatal("small cape missed its scheduled update")
	}
	c.Update(1.0 / 60)
	s.ClipPlane = [4]float32{0, 1, 0, 0}
	c.prepareCrowdCloak(s)
	if c.cloak.reused || c.cloak.elapsed != 0 {
		t.Fatal("clipped pass did not use ordinary cloth updates")
	}
	s.ClipPlane = [4]float32{}
	c.World[14] = 5
	c.prepareCrowdCloak(s)
	c.Update(1.0 / 60)
	c.prepareCrowdCloak(s)
	if c.cloak.reused || c.cloak.elapsed != 0 {
		t.Fatal("large projected cape did not update every frame")
	}
}
