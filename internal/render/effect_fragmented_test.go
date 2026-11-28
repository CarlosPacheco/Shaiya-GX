package render

import "testing"

func TestEffectRunsFragmentedThreshold(t *testing.T) {
	for _, c := range []struct {
		runs, commands int
		want           bool
	}{
		{16, 20, false},  // few runs: keep the multi-draw
		{17, 136, false}, // eight commands per run on average
		{17, 135, true},  // just under eight per run
		{100, 120, true}, // hundreds of tiny runs
		{3, 300, false},
	} {
		if got := effectRunsFragmented(c.runs, c.commands); got != c.want {
			t.Fatalf("runs=%d commands=%d: %v", c.runs, c.commands, got)
		}
	}
}
