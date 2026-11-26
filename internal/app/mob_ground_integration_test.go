//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func mobGroundMap(tb testing.TB, id uint16) *scene.World {
	tb.Helper()
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" || os.Getenv("SHAIYA_DATA") == "" {
		tb.Skip("requires original dungeon assets and GL")
	}
	runtime.LockOSThread()
	tb.Cleanup(runtime.UnlockOSThread)
	win, err := platform.Open(platform.Config{Width: 64, Height: 64, Hidden: true})
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(win.Close)
	if err = render.Init(true); err != nil {
		tb.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { arc.Close() })
	assets := scene.NewAssets(arc)
	w, err := scene.LoadMap(assets, id)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(w.Close)
	return w
}

func TestMobGroundCachePreservesDungeonHeight(t *testing.T) {
	w := mobGroundMap(t, 15)
	var cache mobGroundCache
	// Include each axis, both height modes, explicit same-position respawns and
	// a map change. Input Y is part of the key to preserve different floors.
	cases := []struct {
		p     render.V3
		known bool
		reset uint64
		world *scene.World
	}{
		{render.V3{136.27, 6.6, 229.12186}, true, 0, w},
		{render.V3{136.27, 6.6, 229.12186}, true, 0, w},
		{render.V3{136.28, 6.6, 229.12186}, true, 0, w},
		{render.V3{136.28, 6.7, 229.12186}, true, 0, w},
		{render.V3{136.28, 6.7, 235.59166}, true, 0, w},
		{render.V3{136.28, 6.7, 235.59166}, false, 0, w},
		{render.V3{136.28, 6.7, 235.59166}, true, 1, w},
		{render.V3{136.28, 20, 235.59166}, true, 1, w},
		{render.V3{136.28, 20, 235.59166}, true, 1, nil},
	}
	for _, c := range cases {
		want := c.p
		if c.known {
			want = c.world.MobGroundPosition(c.p)
		} else if c.world != nil && c.world.HasTerrain() {
			want[1] = c.world.Height(c.p[0], c.p[2])
		}
		for range 60 {
			if got := cache.resolve(c.world, c.p, c.known, c.reset); got != want {
				t.Fatalf("%+v: got %v, want %v", c, got, want)
			}
		}
	}
	if got := w.MobGroundPosition(render.V3{136.27, 6.6, 229.12186}); got[1] == 6.6 {
		t.Fatal("fixture did not exercise floor correction")
	}
}

func BenchmarkStationaryDungeonMobGround(b *testing.B) {
	w := mobGroundMap(b, 15)
	positions := make([]render.V3, 256)
	caches := make([]mobGroundCache, len(positions))
	for n := range positions {
		positions[n] = render.V3{136.27 + float32(n%8)*.1, 6.6, 229.12186 + float32(n/8)*.1}
		caches[n].resolve(w, positions[n], true, 0)
	}
	b.Run("every-frame", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			for _, p := range positions {
				_ = w.MobGroundPosition(p)
			}
		}
	})
	b.Run("cached", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			for n, p := range positions {
				_ = caches[n].resolve(w, p, true, 0)
			}
		}
	})
}

func TestMobGroundCachePreservesTerrainHeight(t *testing.T) {
	w := mobGroundMap(t, 1)
	var cache mobGroundCache
	for _, p := range []render.V3{{995.8, 0, 797.3}, {995.9, 0, 797.3}, {995.9, 0, 797.4}, {995.9, 13, 797.4}} {
		for _, known := range []bool{false, true} {
			expected := p
			if known {
				expected = w.MobGroundPosition(p)
			} else {
				expected[1] = w.Height(p[0], p[2])
			}
			for range 60 {
				if got := cache.resolve(w, p, known, 0); got != expected {
					t.Fatalf("known=%v input=%v got=%v want=%v", known, p, got, expected)
				}
			}
		}
	}
	if p := cache.resolve(w, render.V3{995.8, 0, 797.3}, false, 0); p[1] == 0 {
		t.Fatal("fixture did not exercise terrain grounding")
	}
}

// Exercise the movement actor itself, including the midpoint of a network
// segment. The full collision surface remains available to other callers.
func TestMovingMobIgnoresPlacedCollisionWhenRendered(t *testing.T) {
	t.Run("field", func(t *testing.T) {
		w := mobGroundMap(t, 19)
		// This segment crosses a real collider in the field's full geometry.
		p, q := render.V3{1570.3013, 0, 1102.9619}, render.V3{1576.3013, 0, 1102.9619}
		p[1], q[1] = w.Height(p[0], p[2]), w.Height(q[0], q[2])
		testVisualMobSegment(t, w, p, q, func(mid render.V3) render.V3 {
			mid[1] = w.Height(mid[0], mid[2])
			return mid
		})
	})
	t.Run("dungeon", func(t *testing.T) {
		w := mobGroundMap(t, 15)
		p := render.V3{135.27, 6.6, 229.12186}
		q := render.V3{137.27, 6.6, 229.12186}
		mid := p.Add(q.Sub(p).Scale(.5))
		if w.MobGroundPosition(mid) == mid {
			t.Fatal("fixture does not distinguish full collision snapping")
		}
		testVisualMobSegment(t, w, p, q, func(mid render.V3) render.V3 { return mid })
	})
}

func testVisualMobSegment(t *testing.T, w *scene.World, from, to render.V3, wantMid func(render.V3) render.V3) {
	t.Helper()
	a := &App{wv: &worldView{world: w}}
	e := &game.Entity{Kind: game.KindMob, MobHeightKnown: true, X: from[0], Y: from[1], Z: from[2]}
	m := &mobActor{entity: e}
	a.updateMobPosition(m)
	e.X, e.Y, e.Z = to[0], to[1], to[2]
	e.MoveSerial, e.MobMoveDurationMS = 1, 1000
	a.updateMobPosition(m)
	a.clock = .5
	a.updateMobPosition(m)
	want := wantMid(from.Add(to.Sub(from).Scale(.5)))
	if m.pos != want {
		t.Fatalf("mid-segment mob position %v, want %v", m.pos, want)
	}
}
