package mapgeometry

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"math"
	"testing"
)

func ladderFixture() *Geometry {
	g := &Geometry{Ladders: []Ladder{{Bottom: Position{5, 0, 5}, Top: Position{5, 5, 5}, Normal: Position{0, 0, 1}, Width: .7}}}
	g.Reindex()
	for _, tri := range []collision.Triangle{{{0, 0, 0}, {0, 0, 10}, {10, 0, 0}}, {{10, 0, 10}, {10, 0, 0}, {0, 0, 10}}, {{3, 5, 2}, {3, 5, 5}, {7, 5, 2}}, {{7, 5, 5}, {7, 5, 2}, {3, 5, 5}}} {
		g.Add(tri)
	}
	return g
}
func TestLadderClimbHoldDownAndLanding(t *testing.T) {
	g := ladderFixture()
	m, p, ok := g.AttachLadder(Position{5, .045, 5.55}, Position{0, 0, -1}, 0)
	if !ok {
		t.Fatal("cannot attach")
	}
	m.Mode = LadderUp
	for range 60 {
		p = m.Step(g, p, 1.0/120)
	}
	if math.Abs(float64(p[1]-1.295)) > .01 {
		t.Fatal("wrong climb speed", p)
	}
	m.Mode = LadderHold
	old := p
	for range 60 {
		p = m.Step(g, p, 1.0/120)
	}
	if p != old {
		t.Fatal("holding moved", p, old)
	}
	m.Mode = LadderDown
	for range 24 {
		p = m.Step(g, p, 1.0/120)
	}
	if math.Abs(float64(p[1]-.795)) > .01 {
		t.Fatal("cannot descend", p)
	}
	m.Mode = LadderUp
	for range 300 {
		p = m.Step(g, p, 1.0/120)
	}
	if m.Active || !g.Supported(p) || p[1] < 4.9 {
		t.Fatal("did not dismount onto upper landing", p, m)
	}
}
func TestLadderRejectsRemoteFacingWallAndCeiling(t *testing.T) {
	g := ladderFixture()
	for _, tc := range []struct {
		p, f  Position
		index int
	}{{Position{20, 0, 20}, Position{0, 0, -1}, 0}, {Position{5, .045, 5.55}, Position{0, 0, 1}, 0}, {Position{5, 10, 5.55}, Position{0, 0, -1}, 0}, {Position{5, .045, 5.55}, Position{0, 0, -1}, 42}} {
		if _, _, ok := g.AttachLadder(tc.p, tc.f, tc.index); ok {
			t.Fatal("invalid attachment allowed", tc)
		}
	}
	m, p, ok := g.AttachLadder(Position{5, .045, 5.55}, Position{0, 0, -1}, 0)
	if !ok {
		t.Fatal("fixture attachment")
	}
	// An authored downward-facing ceiling must block ascent.
	g.Add(collision.Triangle{{3, 3, 4}, {7, 3, 4}, {3, 3, 7}})
	g.Add(collision.Triangle{{7, 3, 7}, {3, 3, 7}, {7, 3, 4}})
	m.Mode = LadderUp
	for range 300 {
		p = m.Step(g, p, 1.0/120)
	}
	if p[1] > 2 {
		t.Fatal("climbed through ceiling", p)
	}
}
func TestNativeLadderCatalog(t *testing.T) {
	count := 0
	for _, ladders := range nativeLadders {
		if err := ValidateLadders(ladders); err != nil {
			t.Fatal(err)
		}
		count += len(ladders)
	}
	if count != 71 {
		t.Fatal("native placement catalog incomplete", count)
	}
	ladders := NativeLadders(1)
	ladders[0].Bottom[1] = -999
	if NativeLadders(1)[0].Bottom[1] == -999 {
		t.Fatal("caller modified shared catalog")
	}
}

func TestLadderDescendsFromUpperLanding(t *testing.T) {
	g := ladderFixture()
	m, p, ok := g.AttachLadder(Position{5, 5.045, 4.45}, Position{0, 0, 1}, 0)
	if !ok {
		t.Fatal("cannot board from upper landing")
	}
	m.Mode = LadderDown
	for range 300 {
		p = m.Step(g, p, 1.0/120)
	}
	if m.Active || p[1] > .1 || !g.Supported(p) {
		t.Fatal("upper landing prevented descent", m, p)
	}
}

func TestLadderNarrowLandingAboveVisualRungs(t *testing.T) {
	g := ladderFixture()
	g.Triangles = g.Triangles[:2]
	g.Reindex()
	g.Add(collision.Triangle{{4, 5.35, 4.88}, {4, 5.35, 5.12}, {6, 5.35, 4.88}})
	g.Add(collision.Triangle{{6, 5.35, 5.12}, {6, 5.35, 4.88}, {4, 5.35, 5.12}})
	m, p, ok := g.AttachLadder(Position{5, .045, 5.55}, Position{0, 0, -1}, 0)
	if !ok {
		t.Fatal("cannot attach")
	}
	m.Mode = LadderUp
	for range 400 {
		p = m.Step(g, p, 1.0/120)
	}
	if m.Active || !g.Supported(p) || p[1] < 5.35 {
		t.Fatal("could not board narrow raised landing", p, m)
	}
}
