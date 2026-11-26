package scene

import (
	"math"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/clientlog"
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func collisionVec(v render.V3) collision.Vec {
	return collision.Vec{float64(v[0]), float64(v[1]), float64(v[2])}
}
func renderVec(v collision.Vec) render.V3 {
	return render.V3{float32(v[0]), float32(v[1]), float32(v[2])}
}

// prepareCollisions retains CPU-only geometry for the whole map. Rendering may
// unload or cull any cell without opening a hole in movement collision.
func (w *World) prepareCollisions(a *Assets) error {
	if w.collision != nil {
		return nil
	}
	started := time.Now()
	geometry, err := mapgeometry.Load(a.Arc.ReadFile, w.WLD)
	if err != nil {
		return err
	}
	geometry.Ladders, err = mapgeometry.BuildLadders(a.Arc.ReadFile, w.WLD)
	if err != nil {
		return err
	}
	w.collision = geometry.Index
	w.physics = geometry
	geometry.SetDynamicClock(w.DynamicTime)
	w.movingColliders = geometry.Dynamic
	clientlog.Debugf("collision ready: %d static triangles, %d moving instances, %d resources in %s", geometry.Index.Len(), len(geometry.Dynamic), len(geometry.Sources), time.Since(started))
	return nil
}

func (w *World) visitCollisions(bounds collision.Box, emit func(collision.Triangle)) {
	if w == nil {
		return
	}
	w.collision.Visit(bounds, emit)
	for _, instance := range w.movingColliders {
		instance.Visit(bounds, w.DynamicTime(), emit)
	}
}

func (w *World) PlayerHeightLimits() (low, high float32) {
	if w == nil {
		return (*mapgeometry.Geometry)(nil).PlayerHeightLimits()
	}
	return w.physics.PlayerHeightLimits()
}

// MovePlayer adapts GX's foot position to the original ellipsoid centre.
func (w *World) MovePlayer(feet, delta render.V3) (render.V3, bool) {
	if w == nil || w.collision == nil {
		return feet.Add(delta), false
	}
	centre := collisionVec(feet)
	centre[1] += collision.PlayerRadii[1]
	moved, hit := collision.Move(centre, collisionVec(delta), w.visitCollisions)
	moved[1] -= collision.PlayerRadii[1]
	return renderVec(moved), hit
}

// CollisionTriangles visits the exact movement surfaces, used by GM /cview.
func (w *World) CollisionTriangles(centre render.V3, radius float32, emit func(collision.Triangle)) {
	p := collisionVec(centre)
	r := float64(radius)
	w.visitCollisions(collision.Box{Min: p.Sub(collision.Vec{r, r, r}), Max: p.Add(collision.Vec{r, r, r})}, emit)
}

func (w *World) HasPlayerCollision() bool { return w != nil && w.collision != nil }

// Terrain remains the native height-fan surface, separate from model blockers.
func (w *World) TerrainCollisionTriangles(centre render.V3, radius float32, emit func(collision.Triangle)) {
	if w == nil || !w.HasTerrain() {
		return
	}
	spacing := w.step
	if w.terrainFan {
		spacing *= 4
	}
	size := float32(w.WLD.MapSize)
	x0, z0 := max(float32(0), float32(math.Floor(float64((centre[0]-radius)/spacing)))*spacing), max(float32(0), float32(math.Floor(float64((centre[2]-radius)/spacing)))*spacing)
	vertex := func(x, z float32) collision.Vec { return collisionVec(render.V3{x, w.Height(x, z), z}) }
	for z := z0; z < min(size, centre[2]+radius); z += spacing {
		for x := x0; x < min(size, centre[0]+radius); x += spacing {
			a, b, c, d := vertex(x, z), vertex(x+spacing, z), vertex(x, z+spacing), vertex(x+spacing, z+spacing)
			if w.terrainFan {
				m := vertex(x+spacing/2, z+spacing/2)
				for _, t := range []collision.Triangle{{a, m, b}, {b, m, d}, {d, m, c}, {c, m, a}} {
					emit(t)
				}
			} else {
				emit(collision.Triangle{a, c, b})
				emit(collision.Triangle{d, b, c})
			}
		}
	}
}

// MobGroundPosition refines interpolation against the nearby original surface.
// Network/entity Y stays authoritative; this only follows the same floor visually.
func (w *World) MobGroundPosition(p render.V3) render.V3 {
	if w == nil || w.collision == nil || w.WLD == nil {
		return p
	}
	g := mapgeometry.Geometry{Index: w.collision, MapSize: w.WLD.MapSize, Heights: w.WLD.Heights}
	if y, ok := g.Support(p[0], p[1], p[2], .5); ok {
		p[1] = y
	}
	return p
}

// GroundPlayer shares server slope/step rules, independent of display meshes.
func (w *World) GroundPlayer(p, delta render.V3, airborne bool) render.V3 {
	if w == nil || w.physics == nil {
		return p.Add(delta)
	}
	return render.V3(w.physics.GroundMove(mapgeometry.Position(p), mapgeometry.Position(delta), airborne))
}

// GroundPlayerDT walks delta over dt seconds (see mapgeometry.GroundMoveDT).
func (w *World) GroundPlayerDT(p, delta render.V3, dt float64, airborne bool) render.V3 {
	if w == nil || w.physics == nil {
		return p.Add(delta)
	}
	return render.V3(w.physics.GroundMoveDT(mapgeometry.Position(p), mapgeometry.Position(delta), float32(dt), airborne))
}

func (w *World) StandingPosition(p render.V3) render.V3 {
	if w == nil || w.physics == nil {
		return p
	}
	return render.V3(w.physics.StandingPosition(mapgeometry.Position(p)))
}

func (w *World) SlidePlayer(p render.V3, dt float32, m *mapgeometry.SlopeMotion) render.V3 {
	if w == nil || w.physics == nil {
		return p
	}
	return render.V3(m.Move(w.physics, mapgeometry.Position(p), dt))
}

// PlayerSupported uses the same support contacts as the authoritative server.
func (w *World) PlayerSupported(feet render.V3) bool {
	return w == nil || w.physics == nil || w.physics.Supported(mapgeometry.Position(feet))
}

// SegmentVisible tests an exact world-space ray against map geometry.
func (w *World) SegmentVisible(a, b render.V3) bool {
	return w == nil || w.physics == nil || w.physics.SegmentVisible(mapgeometry.Position(a), mapgeometry.Position(b))
}

// SpellLineOfSight shares the server's CPU geometry and actor-center ray.
func (w *World) SpellLineOfSight(a, b render.V3) bool {
	return w == nil || w.physics == nil || w.physics.LineOfSight(mapgeometry.Position(a), mapgeometry.Position(b))
}

func (w *World) PlayerLadders() []mapgeometry.Ladder {
	if w == nil || w.physics == nil {
		return nil
	}
	return w.physics.Ladders
}
func (w *World) AttachLadder(p, f render.V3, index int, mode uint8) (mapgeometry.LadderMotion, render.V3, bool) {
	if w == nil || w.physics == nil {
		return mapgeometry.LadderMotion{}, p, false
	}
	m, q, ok := w.physics.AttachLadder(mapgeometry.Position(p), mapgeometry.Position(f), index, mode)
	return m, render.V3(q), ok
}
func (w *World) StepLadder(m *mapgeometry.LadderMotion, p render.V3, dt float32) render.V3 {
	return render.V3(m.Step(w.physics, mapgeometry.Position(p), dt))
}

// LadderVisual draws a climbing character on its guide line (see
// mapgeometry.LadderVisualAnchor). A negative index searches for the guide.
func (w *World) LadderVisual(p render.V3, index int) render.V3 {
	if w == nil || w.physics == nil {
		return p
	}
	if index >= 0 {
		return render.V3(w.physics.LadderVisualAnchor(index, mapgeometry.Position(p)))
	}
	if q, ok := w.physics.NearestLadderVisualAnchor(mapgeometry.Position(p)); ok {
		return render.V3(q)
	}
	return p
}
