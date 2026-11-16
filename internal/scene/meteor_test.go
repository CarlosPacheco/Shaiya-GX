package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"testing"
)

func TestMeteorBodyFollowsControllerWhileTrailKeepsWorldBirth(t *testing.T) {
	mesh := &effect.Emitter{Mesh: 0, Position: [3]float32{1, 0, 0}, InitialDelay: 1, Unk13: 1, Unk12: 60, Unk14: 60, Unk16: 1}
	trail := *mesh
	trail.Mesh = -1
	x := &Effect{emitters: []effectEmitter{{definition: mesh, geometry: &effectGeometry{source: &effect.Mesh3DE{}}}, {definition: &trail}}}
	from := render.Basis(render.V3{10, 15, 20}, render.V3{-1, 0, 0}, render.V3{0, 1, 0})
	x.SetMeteorTransform(from)
	x.Update(1.0 / 60)
	if len(x.emitters[0].particles) != 1 || len(x.emitters[1].particles) != 1 {
		t.Fatal("missing ordinary EFT births")
	}
	body := x.emitters[0].particles[0]
	smoke := x.emitters[1].particles[0]
	before := x.particlePosition(smoke, &trail)
	to := render.Basis(render.V3{10, 4, 27.5}, render.V3{0, 0, -1}, render.V3{0, 1, 0})
	x.SetMeteorTransform(to)
	if got, want := x.particlePosition(body, mesh), transformPoint(body.pos, to); got != want {
		t.Fatal("mesh stayed at birth instead of falling", got, want)
	}
	if got := x.particlePosition(smoke, &trail); got != before {
		t.Fatal("trail moved with body", got, before)
	}
	if len(x.emitters) != 2 || len(x.emitters[0].particles) != 1 || len(x.emitters[1].particles) != 1 {
		t.Fatal("placement created extra emitters/particles")
	}
}
