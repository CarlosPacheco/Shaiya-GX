package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"math"
	"testing"
)

func TestCombatHeightUsesMONAndNativeFallback(t *testing.T) {
	for _, tc := range []struct{ height, want float32 }{{5.4, 5.6}, {0, .2}, {-1, -.8}, {-2, 1.2}} {
		m := &CreatureActor{visual: model.Creature{Height: tc.height}}
		if math.Abs(float64(m.NumberHeight()-tc.want)) > .0001 {
			t.Fatal(tc, m.NumberHeight())
		}
	}
}
