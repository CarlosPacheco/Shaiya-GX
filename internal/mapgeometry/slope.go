package mapgeometry

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"math"
)

// TerrainNormal is the original three-probe normal (0x442996/0x5a2c70).
// Probe coordinates are in half-world units and heights are truncated to native
// uint16 sample units before conversion. This differs from a triangle normal.
func (g *Geometry) TerrainNormal(x, z float32) collision.Vec {
	sample := func(px, pz float32) collision.Vec {
		raw := math.Trunc(float64(g.Height(px, pz))*50 + 10000)
		y := float32(raw*float64(float32(.02)) - 200)
		return collision.Vec{float64(px * .5), float64(y), float64(pz * .5)}
	}
	a, b, c := sample(x-.1, z+.1), sample(x+.1, z+.1), sample(x, z-.1)
	return b.Sub(a).Cross(c.Sub(a)).Norm()
}

// NativeSlopeDelta reproduces 0x442ac3..0x442c8c for an already normalized
// terrain normal. The native slide begins at n.Y < .5, without an uphill gate.
func NativeSlopeDelta(n collision.Vec, dt, frames float32) Position {
	if n[1] >= .5 || n[1] <= 0 || dt <= 0 {
		return Position{}
	}
	ny := float32(n[1])
	factor := 1 - ny
	if ny < .1 {
		factor *= 5
	} else if ny < .3 {
		factor *= 3
	}
	x, z := float32(n[0]), float32(n[2])
	scale := dt * frames * factor
	return Position{x * ny * scale, -(x*x + z*z) * scale, z * ny * scale}
}

// SlopeMotion advances native frame pressure at a fixed 60 Hz physics rate,
// rather than letting display FPS or the number of movement packets set it.
type SlopeMotion struct{ Frames float32 }

func (m *SlopeMotion) Step(g *Geometry, p Position, dt float32) Position {
	if dt <= 0 {
		return Position{}
	}
	if !g.HasTerrain() || g.DeepWater(p) || p[1]+.9 <= 0 || p[1] > g.Height(p[0], p[2])+.13 {
		m.Frames = 0
		return Position{}
	}
	n := g.TerrainNormal(p[0], p[2])
	if n[1] >= .5 {
		m.Frames = 0
		return Position{}
	}
	// Integral of native pressure over the elapsed interval. At 60 Hz the first
	// frame has pressure 1, followed by 2, 3, ...; other FPS give the same force.
	frames := m.Frames + .5 + dt*30
	m.Frames += dt * 60
	return NativeSlopeDelta(n, dt, frames)
}

func (g *Geometry) OnTerrain(p Position) bool {
	return g.HasTerrain() && p[0] >= 0 && p[2] >= 0 && p[0] <= float32(g.MapSize) && p[2] <= float32(g.MapSize) && math.Abs(float64(p[1]-g.Height(p[0], p[2]))) <= .13
}
