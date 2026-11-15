package scene

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestInnateGlowFollowsWeaponButFireKeepsWorldTrail(t *testing.T) {
	// Native 0x55d270 stores local birth coordinates. 0x55de50 transforms
	// them immediately for fire. Move and rotate after birth, without a tick.
	d := &effect.Emitter{Loop: 1, InitialDelay: 1, Unk13: 2, Unk12: 60, Unk14: 60, Unk16: 1, Position: [3]float32{0, .65, 0}}
	for _, local := range []bool{false, true} {
		x := &Effect{localParticles: local, emitters: []effectEmitter{{definition: d}}}
		m := render.Identity()
		m[12] = 3
		x.setItemGlowTransform(m)
		x.Update(1.0 / 60)
		if len(x.emitters[0].particles) != 1 {
			t.Fatal("fixture birth missing")
		}
		v := x.emitters[0].particles[0]
		want := render.V3{0, .65, 0}
		if !local {
			want[0] = 3
		}
		if v.pos != want {
			t.Fatalf("local=%v birth=%v want=%v", local, v.pos, want)
		}
		// A quarter turn maps local Y to -X; the new hand is at X=8.
		m = render.M4{0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 1, 0, 8, 0, 0, 1}
		x.setItemGlowTransform(m)
		want = render.V3{3, .65, 0}
		if local {
			want = render.V3{8 - .65, 0, 0}
		}
		if got := x.particlePosition(v, d); got != want {
			t.Fatalf("local=%v moved=%v want=%v", local, got, want)
		}
	}
}
