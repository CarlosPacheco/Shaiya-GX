//go:build integration

package mapgeometry

import (
	"crypto/sha256"
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
)

func heldObstacleMap(tb testing.TB) *Geometry {
	tb.Helper()
	if os.Getenv("SHAIYA_DATA") == "" {
		tb.Skip("requires original map assets")
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		tb.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("world/1.wld")
	if err != nil {
		tb.Fatal(err)
	}
	layout, err := world.ParseWLD(raw)
	if err != nil {
		tb.Fatal(err)
	}
	geometry, err := Load(arc.ReadFile, layout)
	if err != nil {
		tb.Fatal(err)
	}
	return geometry
}

func TestOriginalIndexHeldMovementTrace(t *testing.T) {
	geometry := heldObstacleMap(t)
	trace := sha256.New()
	for _, start := range []Position{{1518.6023, 17.382334, 923.50507}, {995.8, 13, 797.3}} {
		for _, delta := range []Position{{-.004272655, 0, .07988545}, {.08, 0, 0}, {0, 0, -.08}} {
			position := start
			for range 360 {
				position = geometry.GroundMove(position, delta, false)
				fmt.Fprintf(trace, "%08x/%08x/%08x;", math.Float32bits(position[0]), math.Float32bits(position[1]), math.Float32bits(position[2]))
			}
		}
	}
	// Captured using the original, unfiltered cell scan. Compare float bits,
	// including sliding, stepping and support, rather than approximate positions.
	got := fmt.Sprintf("%x", trace.Sum(nil))
	const original = "160ff96a46bc31bcb62b0b22a031e9d9f86e2d3554f0ce4df6296d29ccbde5af"
	if got != original {
		t.Fatalf("original movement trace changed: %s", got)
	}
	t.Logf("2160 original-map movement frames: %s", got)
}

func BenchmarkOriginalIndexHeldObstacle(b *testing.B) {
	geometry := heldObstacleMap(b)
	for n, start := range []Position{{1518.6023, 17.382334, 923.50507}, {995.8, 13, 797.3}} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			position := start
			delta := Position{-.004272655, 0, .07988545}
			b.Logf("held position=%v next=%v delta=%v", position, geometry.GroundMove(position, delta, false), delta)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = geometry.GroundMove(position, delta, false)
			}
		})
	}
}
