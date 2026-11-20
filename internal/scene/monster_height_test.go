package scene

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestCombatHeightUsesMONAndNativeFallback(t *testing.T) {
	for _, tc := range []struct{ height, want float32 }{{5.4, 5.6}, {0, .2}, {-1, -.8}, {-2, 1.2}} {
		m := &CreatureActor{visual: model.Creature{Height: tc.height}}
		if math.Abs(float64(m.NumberHeight()-tc.want)) > .0001 {
			t.Fatal(tc, m.NumberHeight())
		}
	}
}

func TestCreaturePlateAnchorsUseNativeWorldHeights(t *testing.T) {
	m := &CreatureActor{
		Character: &Character{World: render.Basis(render.V3{10, 20, 30}, render.V3{0, 0, 1}, render.V3{0, 1, 0})},
		visual:    model.Creature{Height: 4}, visualScale: 1.6,
	}
	for _, tc := range []struct {
		anchor render.V3
		y      float32
	}{
		{m.MonsterPlateAnchor(), 26.6},
		{m.NPCPlateAnchor(1, 1), 24.3},
		{m.NPCPlateAnchor(7, 541), 20.86},
		{m.NPCPlateAnchor(12, 1), 22.15},
	} {
		if tc.anchor[0] != 10 || tc.anchor[2] != 30 || math.Abs(float64(tc.anchor[1]-tc.y)) > .0001 {
			t.Fatal(tc)
		}
	}
	m.visual.Height = -2
	if got := m.MonsterPlateAnchor()[1]; math.Abs(float64(got-17)) > .0001 {
		t.Fatalf("authored negative height became the combat-number fallback: %f", got)
	}
}
