package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"testing"
)

func TestMonsterScalePreservesWorldPlacement(t *testing.T) {
	for _, scale := range []float32{.4, .6, 1, 1.3, 1.6, 2, 5} {
		m := &CreatureActor{Character: &Character{}, visualScale: scale, visual: model.Creature{Height: 3}}
		for i := range 10 {
			pos := render.V3{float32(i), 20, 30}
			m.SetTransform(render.Basis(pos, render.V3{1, 0, 1}, render.V3{0, 1, 0}))
			if m.World[12] != pos[0] || m.World[13] != pos[1] || m.World[14] != pos[2] {
				t.Fatal("scaled world position")
			}
			for row := range 3 {
				got := render.V3{m.World[row*4], m.World[row*4+1], m.World[row*4+2]}.Len()
				if math.Abs(float64(got-scale)) > 1e-5 {
					t.Fatalf("scale accumulated or lost: %v, want %v", got, scale)
				}
			}
			if math.Abs(float64(m.NumberHeight()-(3*scale+.2))) > 1e-5 {
				t.Fatal("unscaled number anchor")
			}
		}
	}
}
