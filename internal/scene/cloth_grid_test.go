package scene

import (
	"math"
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestReducedClothPreservesPinsAndResumesFullGrid(t *testing.T) {
	const columns, rows = 5, 9
	for mode := range 2 {
		record := model.CloakRecord{Columns: columns, Rows: rows, Mode: mode}
		rest := make([]render.V3, columns*rows)
		vertices := make([]render.Vertex, len(rest))
		for i := range vertices {
			col, row := i%columns, i/columns
			rest[i] = render.V3{float32(col) * .15, 2 - float32(row)*.13, float32(math.Sin(float64(i))) * .025}
			vertices[i] = render.Vertex{Pos: rest[i], UV: [2]float32{float32(col) / 4, float32(row) / 8}}
			if row == 0 {
				record.Pins = append(record.Pins, model.CloakPin{Vertex: i, Offset: rest[i]})
			}
		}
		g := &cloakGeometry{record: record, vertices: vertices, grid: makeClothGrid(record, vertices)}
		if g.grid == nil || len(g.grid.nodes) != 25 {
			t.Fatal("reduced grid lost a pin column or failed to halve the rows")
		}
		makeCloak := func() *cloakInstance {
			p := newClothPhysics(rest, columns, rows, mode)
			for _, pin := range record.Pins {
				p.pins[pin.Vertex] = true
			}
			return &cloakInstance{geometry: g, vertices: slices.Clone(vertices), physics: p}
		}
		full, moving := makeCloak(), makeCloak()
		world := render.Identity()
		finals := []render.M4{render.Identity()}
		full.update(world, finals)
		moving.updateGrid(world, finals, 5, true)
		for i, v := range full.vertices {
			if moving.vertices[i].Pos.Sub(v.Pos).Len() > .00001 || moving.vertices[i].UV != v.UV {
				t.Fatal("reduced rest shape changed authored geometry")
			}
		}
		still := makeCloak()
		still.updateGrid(world, finals, 5, true)
		for frame := range 300 {
			finals[0][14] = float32(math.Sin(float64(frame)*.01)) * .09
			moving.elapsed = 1.0 / 30
			moving.updateGrid(world, finals, 5, true)
		}
		if moving.coarse.positions[5] == still.coarse.positions[5] {
			t.Fatal("actors shared the reduced simulation state")
		}
		before := slices.Clone(moving.vertices)
		for _, pin := range record.Pins {
			if before[pin.Vertex].Pos != transformPoint(rest[pin.Vertex], finals[0]) {
				t.Fatal("reduced cloth moved a pinned vertex off its joint")
			}
		}
		moving.update(world, finals)
		if moving.usingGrid {
			t.Fatal("ordinary draw did not restore the full solver")
		}
		for i, v := range moving.vertices {
			if length := v.Pos.Sub(before[i].Pos).Len(); !(length <= .00001) {
				t.Fatalf("mode %d vertex %d jumped by %v when restoring the full grid", mode, i, length)
			}
		}
		actor := &Character{World: world, clip: poseTestClip(1), cloak: moving}
		actor.World[14] = 60
		scene := &render.Scene3D{View: world, Proj: render.PerspectiveFovLH(.8, 1, .1, 120), ViewportH: 720}
		actor.prepareCrowdCloak(scene)
		if !moving.usingGrid {
			t.Fatal("distant main camera did not use the reduced grid")
		}
		scene.View[14] = -25
		actor.prepareCrowdCloak(scene)
		if moving.usingGrid {
			t.Fatal("zoom did not restore the full grid immediately")
		}
		scene.View = world
		for range 5 {
			actor.Update(1.0 / 50)
			actor.prepareCrowdCloak(scene)
		}
		if !moving.usingGrid {
			t.Fatal("receding camera did not reuse the reduced grid")
		}
		scene.ClipPlane = [4]float32{0, 1, 0, 0}
		actor.prepareCrowdCloak(scene)
		if moving.usingGrid {
			t.Fatal("clipped capture did not use the full grid")
		}
	}
}
