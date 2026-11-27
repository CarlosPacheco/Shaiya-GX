package scene

import (
	"math"
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

func TestCreatureEffectMatchesModelSpace(t *testing.T) {
	for _, yaw := range []float64{0, 1, 2.5, -2} {
		sin, cos := math.Sincos(yaw)
		w := render.Basis(render.V3{3, 1, 4}, render.V3{-float32(sin), 0, -float32(cos)}, render.V3{0, 1, 0})
		head := render.V3{1, 10, -6}
		model := render.V3{head[0]*w[0] + head[1]*w[4] + head[2]*w[8] + w[12], head[0]*w[1] + head[1]*w[5] + head[2]*w[9] + w[13], head[0]*w[2] + head[1]*w[6] + head[2]*w[10] + w[14]}
		x := &Effect{}
		x.SetScaledTransform(creatureEffectWorld(w), 1)
		if got := effectPoint(head, x.basis).Add(x.Origin); got.Sub(model).Len() > 1e-4 {
			t.Fatalf("yaw %v: effect %v, model %v", yaw, got, model)
		}
	}
}
