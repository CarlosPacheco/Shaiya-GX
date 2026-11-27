package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

type clothLink struct {
	a, b  int
	rest  float32
	cache int // positive writes a spring force; negative reuses its reverse
}
type clothPhysics struct {
	mode                        int
	positions, velocity, forces []render.V3
	rest                        []render.V3
	verticalLimit               []float32
	mass                        []float32
	pins                        []bool
	links                       []clothLink
	springForces                []render.V3
	columns, rows               int
}

// Native mode zero (0x580e30/0x581190): asymmetric corner masses and
// the original ordered directed spring graph, including its duplicate links.
func newClothPhysics(rest []render.V3, columns, rows int, mode int) *clothPhysics {
	c := &clothPhysics{mode: mode, rest: append([]render.V3(nil), rest...), positions: append([]render.V3(nil), rest...), columns: columns, rows: rows}
	c.velocity = make([]render.V3, len(rest))
	c.forces = make([]render.V3, len(rest))
	c.mass = make([]float32, len(rest))
	c.pins = make([]bool, len(rest))
	c.verticalLimit = make([]float32, len(rest))
	limitScale := float32(1.5)
	if mode == 1 {
		limitScale = 50
	}
	edges := make(map[[2]int]int, len(rest)*4)
	link := func(a, b int) {
		if reverse, ok := edges[[2]int{b, a}]; ok {
			original := &c.links[reverse]
			c.springForces = append(c.springForces, render.V3{})
			original.cache = len(c.springForces)
			// Same force with endpoints reversed. Preserve the native
			// accumulation order, including both directed contributions.
			c.links = append(c.links, clothLink{a: b, b: a, cache: -original.cache})
			return
		}
		edges[[2]int{a, b}] = len(c.links)
		c.links = append(c.links, clothLink{a: a, b: b, rest: rest[a].Sub(rest[b]).Len()})
	}
	for r := range rows {
		for col := range columns {
			i := r*columns + col
			if r > 0 {
				c.verticalLimit[i] = rest[i-columns].Sub(rest[i]).Len() * limitScale
			}
			factor := float32(6)
			if r == 0 || r == rows-1 {
				factor = 3
			}
			if r == 0 && col == 0 || r == rows-1 && col == columns-1 {
				factor = 1
			}
			if r == 0 && col == columns-1 || r == rows-1 && col == 0 {
				factor = 2
			}
			base := float32(.5 * .025)
			if mode == 1 {
				base = .5 / float32((columns-1)*(rows-1)*2)
			}
			c.mass[i] = base * factor * (1.0 / 3)
			if col < columns-1 {
				link(i, i+1)
			}
			if col > 1 {
				link(i, i-1)
			}
			if r < rows-1 {
				link(i, i+columns)
			}
			if r > 1 {
				link(i, i-columns)
				if col > 1 {
					link(i, i-columns-1)
				}
			}
			if r < rows-1 && col > 1 {
				link(i, i+columns-1)
			}
			if r > 1 && col < columns-1 {
				link(i, i-columns+1)
			}
			if r < rows-1 && col < columns-1 {
				link(i, i+columns+1)
			}
		}
	}
	return c
}

// force reports false for a zero-length spring (0x582d90 returns -1).
func (c *clothPhysics) force(wind render.V3) bool {
	ok := true
	for i, v := range c.velocity {
		c.forces[i] = render.V3{}
		if !c.pins[i] {
			c.forces[i] = render.V3{0, -100 * c.mass[i], 0}.Sub(v.Scale(v.Len() * .01)).Add(wind)
		}
	}
	stiffness, damping := float32(600), float32(2)
	if c.mode == 1 {
		stiffness, damping = 500, 1
	}
	for _, link := range c.links {
		if c.pins[link.a] && c.pins[link.b] {
			continue
		}
		var force render.V3
		if link.cache < 0 {
			force = c.springForces[-link.cache-1]
		} else {
			delta := c.positions[link.a].Sub(c.positions[link.b])
			distance := delta.Len()
			if distance == 0 {
				ok = false
			}
			if distance >= .000001 {
				normal := delta.Scale(1 / distance)
				force = normal.Scale(-((distance-link.rest)*stiffness + c.velocity[link.a].Sub(c.velocity[link.b]).Dot(normal)*damping))
			}
			if link.cache > 0 {
				c.springForces[link.cache-1] = force
			}
		}
		if !c.pins[link.a] {
			c.forces[link.a] = c.forces[link.a].Add(force)
		}
		if !c.pins[link.b] {
			c.forces[link.b] = c.forces[link.b].Sub(force)
		}
	}
	return ok
}
func (c *clothPhysics) step(dt float32) bool { return c.stepWithWind(dt, render.V3{}) }

// stepWithWind advances one sub-step; false asks for the native reset.
func (c *clothPhysics) stepWithWind(dt float32, wind render.V3) bool {
	if !c.force(wind) {
		return false
	}
	if c.mode == 1 {
		for i := c.columns; i < len(c.positions); i++ {
			if c.pins[i] {
				continue
			}
			delta := c.positions[i-c.columns].Sub(c.positions[i])
			if delta.Len() <= c.verticalLimit[i] {
				c.forces[i] = c.forces[i].Add(delta.Scale(.2))
			} else {
				c.forces[i] = render.V3{}
				c.positions[i] = c.positions[i].Add(delta.Scale(.5))
				c.velocity[i] = render.V3{}
			}
		}
	}
	// Native 0x581770 uses semi-implicit Euler, not a Verlet approximation.
	for i := range c.positions {
		if !c.pins[i] {
			c.velocity[i] = c.velocity[i].Add(c.forces[i].Scale(dt / c.mass[i]))
			c.positions[i] = c.positions[i].Add(c.velocity[i].Scale(dt))
		}
	}
	if c.mode == 0 {
		// Native 0x5849a0: each segment has an independent 1.5x vertical limit.
		for i := c.columns; i < len(c.positions); i++ {
			if c.pins[i] {
				continue
			}
			d := c.positions[i-c.columns].Sub(c.positions[i])
			limit := c.verticalLimit[i]
			if length := d.Len(); length > limit {
				c.positions[i] = c.positions[i].Add(d.Scale((length - limit) / length))
			}
		}
	}
	return true
}

// reset is 0x5819d0: the rest shape moved to the character position
// (without its rotation), at rest.
func (c *clothPhysics) reset(origin render.V3) {
	for i, v := range c.rest {
		if !c.pins[i] {
			c.positions[i] = v.Add(origin)
		}
		c.velocity[i] = render.V3{}
		c.forces[i] = render.V3{}
	}
}
