package mapgeometry

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"math"
	"os"
)

// MaxGroundSlope is the steepest face that can support a character. The
// original has no such limit for models: the vertical contact of its grounded
// fall step supports any upward face, and steepness only shows through the
// ellipsoid slide. The earlier assisted rules cap it at 50 degrees.
var MaxGroundSlope = func() float64 {
	if os.Getenv("GX_NATIVE_CONTACT") == "off" {
		return 50.0
	}
	return 85.0
}()

const MaxGroundStep = float32(.5)
const FootSkin = float32(.045)

type Position [3]float32

func cv(p Position) collision.Vec { return collision.Vec{float64(p[0]), float64(p[1]), float64(p[2])} }
func pv(p collision.Vec) Position { return Position{float32(p[0]), float32(p[1]), float32(p[2])} }

// TerrainVisit emits only the native fans intersecting a query, including their
// actual normals. Rendering residency and texture quality never affect physics.
func (g *Geometry) TerrainVisit(b collision.Box, emit func(collision.Triangle)) {
	if !g.HasTerrain() {
		return
	}
	step := float32(8)
	if (g.MapSize/2)%4 != 0 {
		step = 2
	}
	x0 := max(float32(0), float32(math.Floor(b.Min[0]/float64(step)))*step)
	z0 := max(float32(0), float32(math.Floor(b.Min[2]/float64(step)))*step)
	vertex := func(x, z float32) collision.Vec {
		return collision.Vec{float64(x), float64(g.Height(x, z)), float64(z)}
	}
	for z := z0; z <= min(float32(g.MapSize)-step, float32(b.Max[2])); z += step {
		for x := x0; x <= min(float32(g.MapSize)-step, float32(b.Max[0])); x += step {
			a, c, d, e := vertex(x, z), vertex(x+step, z), vertex(x, z+step), vertex(x+step, z+step)
			if step == 8 {
				m := vertex(x+step/2, z+step/2)
				for _, t := range []collision.Triangle{{a, m, c}, {c, m, e}, {e, m, d}, {d, m, a}} {
					if t.Bounds().Overlaps(b) {
						emit(t)
					}
				}
			} else {
				for _, t := range []collision.Triangle{{a, d, c}, {e, c, d}} {
					if t.Bounds().Overlaps(b) {
						emit(t)
					}
				}
			}
		}
	}
}
func (g *Geometry) Visit(b collision.Box, emit func(collision.Triangle)) {
	g.Index.Visit(b, emit)
	// Authored rotating objects use the synchronized physics clock.
	for _, dynamic := range g.Dynamic {
		elapsed := float64(0)
		if g.clock != nil {
			elapsed = g.clock()
		}
		dynamic.Visit(b, elapsed, emit)
	}
}
func (g *Geometry) MoveFeet(p, delta Position) (Position, bool) {
	c := cv(p)
	c[1] += collision.PlayerRadii[1]
	next, hit := collision.Move(c, cv(delta), g.Visit)
	next[1] -= collision.PlayerRadii[1]
	return pv(next), hit
}

// Floor selects a nearby walkable surface, with an explicit upward-normal
// threshold. A wall contact is never evidence that another jump is allowed.
func (g *Geometry) Floor(p Position, tolerance float32, below bool) (float32, bool) {
	y, _, ok := g.floorSurface(p, tolerance, below)
	return y, ok
}

func (g *Geometry) floorSurface(p Position, tolerance float32, below bool) (float32, collision.Vec, bool) {
	best := float64(tolerance) + 1
	out := float32(0)
	normal := collision.Vec{}
	found := false
	b := collision.Box{Min: collision.Vec{float64(p[0]) - .001, float64(p[1] - tolerance), float64(p[2]) - .001}, Max: collision.Vec{float64(p[0]) + .001, float64(p[1] + tolerance), float64(p[2]) + .001}}
	g.Visit(b, func(t collision.Triangle) {
		a, c := t[1].Sub(t[0]), t[2].Sub(t[0])
		n := a.Cross(c).Norm()
		if n[1] < math.Cos(MaxGroundSlope*math.Pi/180) {
			return
		}
		determinant := a[0]*c[2] - a[2]*c[0]
		if math.Abs(determinant) < 1e-12 {
			return
		}
		dx, dz := float64(p[0])-t[0][0], float64(p[2])-t[0][2]
		u, v := (dx*c[2]-dz*c[0])/determinant, (a[0]*dz-a[2]*dx)/determinant
		if u < -1e-5 || v < -1e-5 || u+v > 1.00001 {
			return
		}
		y := t[0][1] + a[1]*u + c[1]*v
		if below && y > float64(p[1])+.001 {
			return
		}
		distance := math.Abs(y - float64(p[1]))
		if distance <= float64(tolerance) && distance < best {
			best, out, found = distance, float32(y), true
			normal = n
		}
	})
	if g.HasTerrain() && p[0] >= 0 && p[2] >= 0 && p[0] <= float32(g.MapSize) && p[2] <= float32(g.MapSize) {
		y := g.Height(p[0], p[2])
		distance := math.Abs(float64(y - p[1]))
		if (!below || y <= p[1]+.001) && distance <= float64(tolerance) && distance < best {
			out, normal, found = y, collision.Vec{0, 1, 0}, true
		}
	}
	return out, normal, found
}

func (g *Geometry) DeepWater(p Position) bool {
	return g.Water && g.HasTerrain() && p[0] >= 0 && p[2] >= 0 && p[0] <= float32(g.MapSize) && p[2] <= float32(g.MapSize) && g.Height(p[0], p[2]) < -1.6
}

// GroundMove retains native horizontal edge sliding and adds a bounded swept
// step only when it improves blocked ground movement onto a real support face.
// NativeContact selects the original controller for walking: the ellipsoid
// slide of 0x4383d0 followed by the grounded fall step, with no support,
// step-height or clearance veto on top of it (0x442d9a..0x44305f).
// GX_NATIVE_CONTACT=off restores the earlier assisted rules in client and
// server. See docs/native-contact-research.md.
var NativeContact = os.Getenv("GX_NATIVE_CONTACT") != "off"

// NativeGroundFall is the grounded fall speed (-2.98 units/s). Every
// horizontal sub-step is followed by that speed times its own duration, so
// pushing against a ledge cannot ride its edge upward the way a sweep without
// gravity would, and the result depends on the frame step like the original.
const NativeGroundFall = float32(2.98)

// GroundMove is GroundMoveDT for callers that do not know their duration: it
// assumes the usual walking speed of about 3 units per second.
func (g *Geometry) GroundMove(p, delta Position, airborne bool) Position {
	dist := float32(math.Hypot(float64(delta[0]), float64(delta[2])))
	return g.GroundMoveDT(p, delta, min(max(dist/3, 1.0/240), 1.0/20), airborne)
}

// GroundMoveDT walks delta over dt seconds.
func (g *Geometry) GroundMoveDT(p, delta Position, dt float32, airborne bool) Position {
	if !NativeContact {
		return g.groundMoveAssisted(p, delta, airborne)
	}
	if delta[0] == 0 && delta[2] == 0 {
		return p
	}
	start := p
	count := HorizontalStepCount(delta)
	settle := NativeGroundFall * dt / float32(count)
	for i := 0; i < count; i++ {
		move := HorizontalStep(start, delta, i, count)
		next := Position{p[0] + move[0], p[1], p[2] + move[2]}
		if g.HasTerrain() && (next[0] < 0 || next[2] < 0 || next[0] > float32(g.MapSize) || next[2] > float32(g.MapSize)) {
			break
		}
		moved, _ := g.MoveFeet(p, move)
		if !airborne {
			moved, _ = g.MoveFeet(moved, Position{0, -settle, 0})
		}
		if !airborne && g.HasTerrain() {
			h := g.Height(moved[0], moved[2])
			if moved[1] < h {
				moved[1] = h
			}
			if g.OnTerrain(p) {
				if support, ok := g.modelSupport(moved, .13); !ok || support.feet < h {
					moved[1] = h
				}
			}
		}
		p = moved
	}
	return p
}

func (g *Geometry) groundMoveAssisted(p, delta Position, airborne bool) Position {
	start := p
	count := HorizontalStepCount(delta)
	for i := 0; i < count; i++ {
		move := HorizontalStep(start, delta, i, count)
		next := Position{p[0] + move[0], p[1], p[2] + move[2]}
		if g.HasTerrain() && (next[0] < 0 || next[2] < 0 || next[0] > float32(g.MapSize) || next[2] > float32(g.MapSize)) {
			break
		}
		base, grounded := g.groundSupport(p, .13)
		moved, _ := g.MoveFeet(p, move)
		if moved[1] > p[1]+.002 {
			support, ok := g.modelSupport(moved, .13)
			limit := p[1] + MaxGroundStep
			if grounded && !airborne {
				limit = base.surface + MaxGroundStep
			}
			if !ok || support.surface > limit+.01 || moved[1]-p[1] > MaxGroundStep+.01 || !g.clearBody(moved) {
				moved = p
			}
		}
		progress := func(at Position) float32 { return (at[0]-p[0])*move[0] + (at[2]-p[2])*move[2] }
		wanted := move[0]*move[0] + move[2]*move[2]
		if !airborne && grounded && wanted > 0 && progress(moved) < wanted*.98 {
			lift := max(float32(0), base.surface+MaxGroundStep+FootSkin-p[1])
			raised, _ := g.MoveFeet(p, Position{0, lift, 0})
			if raised[1] > p[1]+.001 {
				step, _ := g.MoveFeet(raised, move)
				step, _ = g.MoveFeet(step, Position{0, -(step[1] - p[1] + .13), 0})
				if support, ok := g.groundSupport(step, .13); ok && support.surface <= base.surface+MaxGroundStep+.01 && support.surface >= base.surface-MaxGroundStep && progress(step) > progress(moved)+.000001 && g.clearBody(step) {
					moved = step
				}
			}
		}
		// Terrain keeps its native height controller; authored steps above it retain
		// their actual ellipsoid contact instead of being snapped back to terrain.
		if !airborne && g.HasTerrain() {
			h := g.Height(moved[0], moved[2])
			if moved[1] < h {
				moved[1] = h
			}
			if g.OnTerrain(p) {
				if support, ok := g.modelSupport(moved, .13); !ok || support.feet < h {
					moved[1] = h
				}
			}
		}
		p = moved
	}
	if !airborne {
		p = g.StandingPosition(p)
	}
	return p
}

func HorizontalStepCount(delta Position) int {
	return max(1, int(math.Ceil(math.Hypot(float64(delta[0]), float64(delta[2]))/.15)))
}

// HorizontalStep partitions the original path into short sweeps. Sampling
// endpoints before subtracting preserves the final position on clear ground,
// even when a world-coordinate float cannot represent each equal subdivision.
func HorizontalStep(start, delta Position, index, count int) Position {
	if count == 1 {
		return Position{delta[0], 0, delta[2]}
	}
	previous, next := float64(index)/float64(count), float64(index+1)/float64(count)
	x, z := float64(start[0]), float64(start[2])
	dx, dz := float64(delta[0]), float64(delta[2])
	return Position{
		float32(x+dx*next) - float32(x+dx*previous), 0,
		float32(z+dz*next) - float32(z+dz*previous),
	}
}

// Entry fixes authored spawn-volume heights without projecting onto another
// horizontal location or trusting a character's stored airborne elevation.
func (g *Geometry) Entry(p Position) (Position, bool) {
	return g.entry(p, .05)
}

// Spawn accepts authored server spawn volumes slightly buried in their floor.
// It is never used for a position supplied by a movement packet or saved login.
func (g *Geometry) Spawn(p Position) (Position, bool) { return g.entry(p, MaxGroundStep) }

func (g *Geometry) entry(p Position, upward float32) (Position, bool) {
	if !cv(p).Finite() || g.HasTerrain() && (p[0] < 0 || p[2] < 0 || p[0] > float32(g.MapSize) || p[2] > float32(g.MapSize)) {
		return Position{}, false
	}
	if g.DeepWater(p) && p[1] <= -1.5 && p[1] >= g.Height(p[0], p[2]) {
		return p, true
	}
	hint := p
	hint[1] += upward
	if y, ok := g.Floor(hint, 1e6, true); ok {
		p[1] = y + FootSkin
		return g.StandingPosition(p), true
	}
	if g.HasTerrain() {
		p[1] = g.Height(p[0], p[2]) + FootSkin
		if g.Supported(p) {
			return p, true
		}
	}
	return Position{}, false
}
