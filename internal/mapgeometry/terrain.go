package mapgeometry

import "github.com/Spelunkern/Shaiya-GX/internal/collision"

func (g *Geometry) HasTerrain() bool {
	side := int(g.MapSize/2) + 1
	return g.MapSize > 0 && len(g.Heights) == side*side
}
func (g *Geometry) Height(x, z float32) float32 {
	if !g.HasTerrain() {
		return 0
	}
	side := int(g.MapSize/2) + 1
	x = min(max(x/2, 0), float32(side-1))
	z = min(max(z/2, 0), float32(side-1))
	h := func(x, z int) float32 { return (float32(g.Heights[z*side+x]) - 10000) / 50 }
	if (side-1)%4 == 0 {
		ix, iz := min(int(x)/4*4, side-5), min(int(z)/4*4, side-5)
		dx, dz := x-float32(ix), z-float32(iz)
		a, b, c, d, m := h(ix, iz), h(ix+4, iz), h(ix, iz+4), h(ix+4, iz+4), h(ix+2, iz+2)
		if dz <= dx {
			if 4-dx < dz {
				return b + (4-dx)*(m-b)/2 + (dz-(4-dx))*(d-b)/4
			}
			return a + dx*(b-a)/4 + dz*(m-(a+b)/2)/2
		}
		if 4-dx < dz {
			return c + dx*(d-c)/4 + (4-dz)*(m-(c+d)/2)/2
		}
		return a + dz*(c-a)/4 + dx*(m-(a+c)/2)/2
	}
	ix, iz := int(x), int(z)
	jx, jz := min(ix+1, side-1), min(iz+1, side-1)
	u, v := x-float32(ix), z-float32(iz)
	if u+v <= 1 {
		return h(ix, iz) + u*(h(jx, iz)-h(ix, iz)) + v*(h(ix, jz)-h(ix, iz))
	}
	return h(jx, jz) + (1-u)*(h(ix, jz)-h(jx, jz)) + (1-v)*(h(jx, iz)-h(jx, jz))
}
func (g *Geometry) Terrain(emit func(collision.Triangle)) {
	if !g.HasTerrain() {
		return
	}
	step := float32(8)
	if (g.MapSize/2)%4 != 0 {
		step = 2
	}
	p := func(x, z float32) collision.Vec {
		return collision.Vec{float64(x), float64(g.Height(x, z)), float64(z)}
	}
	for z := float32(0); z < float32(g.MapSize); z += step {
		for x := float32(0); x < float32(g.MapSize); x += step {
			a, b, c, d := p(x, z), p(x+step, z), p(x, z+step), p(x+step, z+step)
			if step == 8 {
				m := p(x+step/2, z+step/2)
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

// Support finds the exact original surface near a navmesh-projected height.
func (g *Geometry) Support(x, y, z, tolerance float32) (float32, bool) {
	return g.support(x, y, z, tolerance, false)
}

// Below selects the first supporting floor below a spawn volume, not a roof.
func (g *Geometry) Below(x, y, z, drop float32) (float32, bool) {
	return g.support(x, y, z, drop, true)
}

func (g *Geometry) support(x, y, z, tolerance float32, below bool) (float32, bool) {
	best := float64(tolerance) + 1
	result := float32(0)
	found := false
	candidate := func(height float64) {
		if below && height > float64(y)+.01 {
			return
		}
		distance := height - float64(y)
		if distance < 0 {
			distance = -distance
		}
		if distance <= float64(tolerance) && distance < best {
			best, result, found = distance, float32(height), true
		}
	}
	if g.HasTerrain() && x >= 0 && z >= 0 && x <= float32(g.MapSize) && z <= float32(g.MapSize) {
		candidate(float64(g.Height(x, z)))
	}
	g.Index.Visit(collision.Box{Min: collision.Vec{float64(x) - .001, float64(y - tolerance), float64(z) - .001}, Max: collision.Vec{float64(x) + .001, float64(y + tolerance), float64(z) + .001}}, func(t collision.Triangle) {
		a, b := t[1].Sub(t[0]), t[2].Sub(t[0])
		normal := a.Cross(b).Norm()
		if normal[1] < .5 {
			return
		}
		d := (a[0]*b[2] - a[2]*b[0])
		if d == 0 {
			return
		}
		dx, dz := float64(x)-t[0][0], float64(z)-t[0][2]
		u, v := (dx*b[2]-dz*b[0])/d, (a[0]*dz-a[2]*dx)/d
		if u >= -1e-5 && v >= -1e-5 && u+v <= 1.00001 {
			candidate(t[0][1] + a[1]*u + b[1]*v)
		}
	})
	return result, found
}
