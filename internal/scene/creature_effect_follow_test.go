package scene

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestScaledEffectFollowsAndScales(t *testing.T) {
	def := &effect.Emitter{Sub3: []effect.Sub3Frame{{Unk1: 1, Unk2: 2}}}
	x := &Effect{emitters: []effectEmitter{{definition: def}}, basis: render.Identity()}
	x.ScaleParticleSizes(2)
	if def.Sub3[0].Unk1 != 1 || x.emitters[0].definition.Sub3[0] != (effect.Sub3Frame{Unk1: 2, Unk2: 4}) {
		t.Fatal("sizes not scaled on a private copy")
	}
	world := render.Basis(render.V3{5, 0, 7}, render.V3{0, 0, -1}, render.V3{0, 1, 0})
	x.SetScaledTransform(world, 1.6)
	if x.Origin != (render.V3{5, 0, 7}) {
		t.Fatal("origin", x.Origin)
	}
	if l := (render.V3{x.basis[0], x.basis[1], x.basis[2]}).Len(); l < 1.599 || l > 1.601 {
		t.Fatal("basis scale", l)
	}
	x.SetScaledTransform(render.Basis(render.V3{9, 0, 1}, render.V3{1, 0, 0}, render.V3{0, 1, 0}), 1)
	if x.Origin != (render.V3{9, 0, 1}) {
		t.Fatal("did not follow", x.Origin)
	}
}
