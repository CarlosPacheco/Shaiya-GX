package mapgeometry

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"os"
	"strings"
	"testing"
)

func TestOriginalMapOneStoneFallback(t *testing.T) {
	directory := os.Getenv("SHAIYA_DATA")
	if directory == "" {
		t.Skip("requires local original map-one assets")
	}
	a, err := archive.OpenDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	raw, err := a.ReadFile("world/1.wld")
	if err != nil {
		t.Fatal(err)
	}
	w, err := world.ParseWLD(raw)
	if err != nil {
		t.Fatal(err)
	}
	g, err := Load(a.ReadFile, w)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for id, name := range w.Shapes.Assets {
		if !strings.Contains(strings.ToLower(name), "scarpstone") {
			continue
		}
		raw, err := a.ReadFile("entity/shape/" + name)
		if err != nil {
			t.Fatal(err)
		}
		model, err := world.ParseSMOD(raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(model.Colliders) > 0 {
			continue
		}
		parts, err := world.ParseSMODCollisions(raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, instance := range w.Shapes.Instances {
			if instance.Asset != int32(id) {
				continue
			}
			hitWall := false
			Visit(parts, Basis(instance.Pos, instance.Fwd, instance.Up), func(tri collision.Triangle) {
				if hitWall {
					return
				}
				n := tri[1].Sub(tri[0]).Cross(tri[2].Sub(tri[0])).Norm()
				if n[1] > .5 || n[1] < -.5 {
					return
				}
				center := tri[0].Add(tri[1]).Add(tri[2]).Scale(1.0 / 3)
				start := center.Add(n.Scale(2))
				start[1] -= collision.PlayerRadii[1]
				_, hit := g.MoveFeet(pv(start), pv(n.Scale(-4)))
				if !hit {
					t.Fatalf("original stone remains traversable: %s at %v", name, instance.Pos)
				}
				hitWall = true
			})
			if hitWall {
				checked++
				t.Logf("%s original fallback blocks at %v", name, instance.Pos)
			}
			break
		}
	}
	if checked < 3 {
		t.Fatal("too few original stone cases", checked)
	}
}

func TestOriginalMapOneSmallSurfaceSupport(t *testing.T) {
	directory := os.Getenv("SHAIYA_DATA")
	if directory == "" {
		t.Skip("requires local original map-one assets")
	}
	a, err := archive.OpenDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	raw, err := a.ReadFile("world/1.wld")
	if err != nil {
		t.Fatal(err)
	}
	w, err := world.ParseWLD(raw)
	if err != nil {
		t.Fatal(err)
	}
	g, err := Load(a.ReadFile, w)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, tri := range g.Triangles {
		if checked >= 30 {
			break
		}
		n := tri[1].Sub(tri[0]).Cross(tri[2].Sub(tri[0])).Norm()
		if n[1] < .95 {
			continue
		}
		center := tri[0].Add(tri[1]).Add(tri[2]).Scale(1.0 / 3)
		if center[0] < 850 || center[0] > 1150 || center[2] < 650 || center[2] > 950 {
			continue
		}
		ground := g.Height(float32(center[0]), float32(center[2]))
		height := float32(center[1]) - ground
		if height < .1 || height > 1.2 {
			continue
		}
		p := Position{float32(center[0]), float32(center[1]) + 2, float32(center[2])}
		if !g.clearBody(p) {
			continue
		}
		p, hit := g.MoveFeet(p, Position{0, -3, 0})
		if !hit || p[1] < float32(center[1])-.02 || p[1] > float32(center[1])+.15 {
			continue
		}
		if !g.clearBody(p) {
			continue
		}
		if !g.Supported(p) {
			t.Fatal("landing on original small surface lost support", p, center)
		}
		p = g.StandingPosition(p)
		p = g.GroundMove(p, Position{.025, 0, .025}, false)
		if !g.Supported(p) {
			t.Fatal("small movement lost original surface support", p, center)
		}
		checked++
	}
	if checked < 10 {
		t.Fatal("too few central-map small surfaces", checked)
	}
	t.Logf("landed and moved on %d low original surfaces around central map 1", checked)
}
