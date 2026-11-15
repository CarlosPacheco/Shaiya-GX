package mapgeometry

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"testing"
)

func staircase(t *testing.T, rise, tread float32, terrain bool) *Geometry {
	t.Helper()
	g := &Geometry{}
	g.Reindex()
	if terrain {
		g.MapSize = 32
		g.Heights = make([]uint16, 17*17)
		for i := range g.Heights {
			g.Heights[i] = 10000
		}
	} else {
		testFloor(g, 0, -2, 30, -2, 10)
	}
	for i := 0; i < 10; i++ {
		for _, tri := range collision.BlockerBox(collision.Vec{float64(2 + float32(i)*tread), 0, 0}, collision.Vec{float64(2 + float32(i+1)*tread), float64(float32(i+1) * rise), 8}, 0) {
			g.Add(tri)
		}
	}
	testFloor(g, float64(10*rise), float64(2+10*tread), 30, 0, 8)
	return g
}
func TestConsecutiveModelStairsWithGravity(t *testing.T) {
	for _, terrain := range []bool{false, true} {
		for _, shape := range [][2]float32{{.1, .2}, {.15, .3}, {.3, .5}, {.45, .8}} {
			for _, fps := range []int{30, 60, 144} {
				t.Run(fmt.Sprint(terrain, shape, fps), func(t *testing.T) {
					g := staircase(t, shape[0], shape[1], terrain)
					p := Position{1, FootSkin, 4}
					for i := 0; i < fps*4; i++ {
						p = g.GroundMove(p, Position{5 / float32(fps), 0, 0}, false)
						p, _ = g.MoveFeet(p, Position{0, -3 / float32(fps), 0})
						p = g.StandingPosition(p)
					}
					if p[0] < 2+10*shape[1] || p[1] < 10*shape[0]-.1 {
						t.Fatalf("stairs blocked at %v; rise=%g tread=%g", p, shape[0], shape[1])
					}
				})
			}
		}
	}
}
func TestJumpAcrossSmallPlatformEdge(t *testing.T) {
	for _, height := range []float32{.15, .3, .6, 1, 1.2} {
		for _, start := range []float32{3.5, 3.8, 4.2, 4.6} {
			t.Run(fmt.Sprint(height, start), func(t *testing.T) {
				g := &Geometry{}
				g.Reindex()
				testFloor(g, 0, -5, 30, -5, 10)
				for _, tri := range collision.BlockerBox(collision.Vec{5, 0, 0}, collision.Vec{9, float64(height), 8}, 0) {
					g.Add(tri)
				}
				p := Position{start, FootSkin, 4}
				fall := float32(-2.98)
				jumping := true
				for i := 0; i < 150; i++ {
					dx := float32(0)
					if p[0] < 7 {
						dx = 5.0 / 60
					}
					p = g.GroundMove(p, Position{dx, 0, 0}, jumping)
					dy := float32(-3.0 / 60)
					if jumping {
						fall = max(-20, fall*(1+2.8/60))
						dy = (9 + fall) / 60
					}
					next, hit := g.MoveFeet(p, Position{0, dy, 0})
					p = next
					if hit {
						if dy < 0 {
							jumping = false
						} else {
							fall = min(fall, -9)
						}
					}
					if !jumping {
						p = g.StandingPosition(p)
					}
				}
				if p[0] < 6 || p[1] < height || jumping {
					t.Fatal("could not jump onto and stand on small asset", p, jumping)
				}
			})
		}
	}
}
func TestStepAssistDoesNotClimbTallWallsOrCrossCeilings(t *testing.T) {
	for _, height := range []float64{.75, 1, 10} {
		g := &Geometry{}
		g.Reindex()
		testFloor(g, 0, -5, 30, -5, 10)
		for _, tri := range collision.BlockerBox(collision.Vec{5, 0, 0}, collision.Vec{9, height, 8}, 0) {
			g.Add(tri)
		}
		p := Position{3, FootSkin, 4}
		for range 600 {
			p = g.GroundMove(p, Position{.1, 0, 0}, false)
		}
		if p[0] > 4.9 || p[1] > .13 {
			t.Fatal("step assistance climbed a high wall", height, p)
		}
	}
	g := staircase(t, .3, .5, false)
	g.Add(collision.Triangle{{-5, 1.95, 0}, {30, 1.95, 0}, {30, 1.95, 8}})
	g.Add(collision.Triangle{{-5, 1.95, 0}, {30, 1.95, 8}, {-5, 1.95, 8}})
	p := Position{1, FootSkin, 4}
	for range 120 {
		p = g.GroundMove(p, Position{.1, 0, 0}, false)
	}
	if p[0] > 2.1 || p[1]+1.8 > 1.96 {
		t.Fatal("step assistance passed through a ceiling", p)
	}
}
