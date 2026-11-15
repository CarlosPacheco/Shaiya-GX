package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"testing"
)

func TestCombatHeightUsesMONAndNativeFallback(t *testing.T) {
	for _, tc := range []struct{ height, want float32 }{{5.4, 5.4}, {0, 0}, {-1, 1.2}, {-2, 1.2}} {
		m := &CreatureActor{visual: model.Creature{Height: tc.height}}
		if m.NumberHeight() != tc.want {
			t.Fatal(tc, m.NumberHeight())
		}
	}
}
