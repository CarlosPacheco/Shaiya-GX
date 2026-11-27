package mapnav

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
)

// BenchmarkPlayerPhysics walks players over installed maps with the server's
// ground movement and reports the resident heap of each loaded map.
// GX_TEST_GEOMETRY names the installed geometry directory.
func BenchmarkPlayerPhysics(b *testing.B) {
	dir := os.Getenv("GX_TEST_GEOMETRY")
	if dir == "" {
		b.Skip("requires installed geometry")
	}
	for _, id := range []uint16{0, 1, 3, 47} {
		heap := func() uint64 {
			runtime.GC()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return m.HeapAlloc
		}
		before := heap()
		nav, err := Load(dir, id)
		if err != nil {
			b.Fatal(err)
		}
		g, _ := nav.PlayerGeometry()
		resident := heap() - before
		rng := rand.New(rand.NewPCG(uint64(id), 3))
		size := float32(g.MapSize)
		if size == 0 {
			size = 512
		}
		var starts []mapgeometry.Position
		for tries := 0; len(starts) < 64 && tries < 100000; tries++ {
			p, ok := g.Spawn(mapgeometry.Position{rng.Float32() * size, 300, rng.Float32() * size})
			if ok {
				starts = append(starts, p)
			}
		}
		b.Run(fmt.Sprintf("map%d", id), func(b *testing.B) {
			steps := 0
			for b.Loop() {
				for i, p := range starts {
					a := float64(i) * 2.4
					d := mapgeometry.Position{float32(math.Sin(a)) * .35, 0, float32(math.Cos(a)) * .35}
					for range 40 {
						p = g.GroundMove(p, d, false)
						steps++
					}
				}
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(max(steps, 1)), "ns/step")
			b.ReportMetric(float64(resident)/(1<<20), "MiB-resident")
		})
		runtime.KeepAlive(nav)
	}
}
