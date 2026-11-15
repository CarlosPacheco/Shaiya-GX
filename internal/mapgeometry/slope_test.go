package mapgeometry

import (
	"encoding/json"
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"math"
	"os"
	"testing"
)

func TestNativeTerrainNormalOracle(t *testing.T) {
	raw, err := os.ReadFile("testdata/native_terrain_normals.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases struct {
		Fixtures []struct {
			Heights []uint16
			Side    int
			X, Z    float32
			Normal  collision.Vec
		}
	}
	if err = json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for i, c := range cases.Fixtures {
		g := &Geometry{MapSize: uint32((c.Side - 1) * 2), Heights: c.Heights}
		got := g.TerrainNormal(c.X, c.Z)
		if got.Sub(c.Normal).Len() > .0001 {
			t.Errorf("case %d: %v want %v", i, got, c.Normal)
		}
	}
}
func TestNativeSlopeOracle(t *testing.T) {
	raw, err := os.ReadFile("testdata/native_slope.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases struct {
		Fixtures []struct {
			Normal        collision.Vec
			Delta, Result Position
			Dt, Counter   float32
			Previous      bool
			Ground        *bool
		}
	}
	if err = json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for i, c := range cases.Fixtures {
		frames := c.Counter
		if c.Previous {
			frames++
		}
		got := c.Delta
		if c.Ground == nil || *c.Ground {
			d := NativeSlopeDelta(c.Normal, c.Dt, frames)
			for a := range 3 {
				got[a] += d[a]
			}
		}
		if cv(got).Sub(cv(c.Result)).Len() > .00001 {
			t.Errorf("case %d: %v want %v", i, got, c.Result)
		}
	}
}
func TestTerrainDoesNotBecomeInvisibleWall(t *testing.T) {
	for _, slope := range []float32{.4, .9, 2} {
		g := &Geometry{MapSize: 32, Heights: make([]uint16, 17*17)}
		g.Reindex()
		for z := 0; z < 17; z++ {
			for x := 0; x < 17; x++ {
				g.Heights[z*17+x] = uint16(10000 + float32(x*2)*slope*50)
			}
		}
		p := Position{8, g.Height(8, 8), 8}
		start := p
		p = g.GroundMove(p, Position{.1, 0, 0}, false)
		if p[0] < start[0]+.09 || math.Abs(float64(p[1]-g.Height(p[0], p[2]))) > .01 {
			t.Fatal("terrain created a hard wall", slope, p, start)
		}
	}
}

func TestTerrainBackedModelStepAndWall(t *testing.T) {
	g := &Geometry{MapSize: 32, Heights: make([]uint16, 17*17)}
	for i := range g.Heights {
		g.Heights[i] = 10000
	}
	g.Reindex()
	for _, tri := range collision.BlockerBox(collision.Vec{5, 0, 0}, collision.Vec{10, .3, 20}, 0) {
		g.Add(tri)
	}
	p := Position{4, 0, 5}
	for range 30 {
		p = g.GroundMove(p, Position{.1, 0, 0}, false)
	}
	if p[0] < 6 || p[1] < .3 {
		t.Fatal("authored step on terrain became a wall", p)
	}
	for _, tri := range collision.BlockerBox(collision.Vec{12, 0, 0}, collision.Vec{13, 10, 20}, 0) {
		g.Add(tri)
	}
	for range 200 {
		p = g.GroundMove(p, Position{.1, 0, 0}, false)
	}
	if p[0] > 11.8 || p[1] > .45 {
		t.Fatal("model contact climbed tall wall", p)
	}
}
func TestSlopeAttemptGraduallyLosesToNativePressure(t *testing.T) {
	g := &Geometry{MapSize: 128, Heights: make([]uint16, 65*65)}
	g.Reindex()
	for z := 0; z < 65; z++ {
		for x := 0; x < 65; x++ {
			g.Heights[z*65+x] = uint16(10000 + x*200)
		}
	}
	p := Position{30, g.Height(30, 30), 30}
	start := p
	var slide SlopeMotion
	for i := 0; i < 180; i++ {
		p = g.GroundMove(p, Position{5.0 / 60, 0, 0}, false)
		delta := slide.Step(g, p, 1.0/60)
		p = g.GroundMove(p, delta, false)
		if i == 0 && p[0] <= start[0] {
			t.Fatal("uphill attempt was cut off immediately", p)
		}
	}
	if p[0] > start[0] {
		t.Fatal("continued uphill input beat native slide", p, start)
	}
}

func TestTerrainSlideDoesNotDependOnRenderingFPS(t *testing.T) {
	g := &Geometry{MapSize: 128, Heights: make([]uint16, 65*65)}
	g.Reindex()
	for z := 0; z < 65; z++ {
		for x := 0; x < 65; x++ {
			g.Heights[z*65+x] = uint16(10000 + x*200)
		}
	}
	var ends []Position
	for _, fps := range []int{30, 60, 120, 144} {
		p := Position{50, 100, 50}
		var motion SlopeMotion
		for i := 0; i < fps; i++ {
			p = g.GroundMove(p, motion.Step(g, p, 1/float32(fps)), false)
		}
		ends = append(ends, p)
	}
	for _, p := range ends[1:] {
		if cv(p).Sub(cv(ends[0])).Len() > .12 {
			t.Fatal("terrain slide depends on rendering FPS", ends)
		}
	}
}
