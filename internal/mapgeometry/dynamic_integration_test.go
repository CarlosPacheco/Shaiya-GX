//go:build integration

package mapgeometry

import (
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
)

func TestOriginalHeldObstacleMovementParity(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires original map assets")
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("world/1.wld")
	if err != nil {
		t.Fatal(err)
	}
	layout, err := world.ParseWLD(raw)
	if err != nil {
		t.Fatal(err)
	}
	fast, err := Load(arc.ReadFile, layout)
	if err != nil {
		t.Fatal(err)
	}
	reference := *fast
	reference.Dynamic = append([]Dynamic(nil), fast.Dynamic...)
	for i := range reference.Dynamic {
		reference.Dynamic[i].bounds = nil
	}
	elapsed := float64(0)
	fast.SetDynamicClock(func() float64 { return elapsed })
	reference.SetDynamicClock(func() float64 { return elapsed })
	starts := []Position{{1518.6023, 17.382334, 923.50507}, {995.8, 13, 797.3}}
	for _, d := range fast.Dynamic {
		if d.bounds == nil {
			t.Fatal("loaded dynamic instance lacks broad phase")
		}
		p := Point(d.bounds.Min, d.Transform(0))
		p[1] -= .9
		starts = append(starts, pv(p))
	}
	for _, start := range starts {
		for _, delta := range []Position{{-.004272655, 0, .07988545}, {.08, 0, 0}, {0, 0, -.08}} {
			got, want := start, start
			for frame := 0; frame < 180; frame++ {
				elapsed = float64(frame) / 60
				got = fast.GroundMove(got, delta, false)
				want = reference.GroundMove(want, delta, false)
				if got != want {
					t.Fatalf("frame %d start=%v delta=%v: optimized=%v reference=%v", frame, start, delta, got, want)
				}
			}
		}
	}
	t.Logf("%d held-movement frames match unculled original triangle queries exactly", len(starts)*3*180)
}
