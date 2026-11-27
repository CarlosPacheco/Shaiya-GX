package scene

import (
	"math/rand/v2"
	"sync/atomic"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Native cloth environment (Game.exe EP5, CCloth): body spheres and the
// terrain floor (0x583370), the per-frame wind gust (0x416220/0x54e320) and
// the reset after a degenerate spring (0x582d90/0x5819d0).

type clothSphere struct {
	bone   int
	radius float32
}

// clothBodySpheres is the 0x583370 table, indexed by the native body type
// race*4 + gender*2 + (family != f) (0x4103a0). Bones are the skeleton
// matrices at 0xc0 + index*0x118; spheres with a zero radius are omitted.
var clothBodySpheres = [16][]clothSphere{
	0: {{1, .25}, {4, .15}}, 1: {{1, .25}, {4, .15}}, 2: {{1, .25}, {4, .15}}, 3: {{1, .25}, {4, .15}},
	4: {{1, .25}, {4, .15}}, 5: {{1, .25}, {4, .15}},
	6: {{1, .22}, {4, .14}}, 7: {{1, .22}, {4, .14}},
	8:  {{1, .28}, {4, .28}},
	9:  {{1, .23}, {2, .18}, {4, .18}, {5, .15}},
	10: {{1, .23}, {4, .23}},
	11: {{2, .23}, {3, .18}, {4, .18}, {5, .15}},
	12: {{1, .2}, {3, .15}, {4, .15}, {5, .15}}, 13: {{1, .2}, {3, .15}, {4, .15}, {5, .15}},
	14: {{1, .2}, {3, .15}, {4, .15}, {5, .15}}, 15: {{1, .2}, {3, .15}, {4, .15}, {5, .15}},
}

func clothBodyType(a Appearance) int {
	t := int(a.Race%4) * 4
	if a.Gender == 1 {
		t += 2
	}
	if p := cloakPrefix(a); p[len(p)-1] != 'f' {
		t++
	}
	return t
}

const (
	clothFloorLift      = .005 // 0x6bc85c
	clothCollisionRange = 20   // character+0x2c0: camera distance
	clothWindBone       = 4    // 0x416190: skeleton matrix 0x520
	clothWindMax        = .1   // rand()/32767 * 0.1
)

// clothGround supplies the terrain height under a character (native
// 0x410580). The world installs it; without one the floor is not applied.
var clothGround atomic.Pointer[func(x, z float32) float32]

// SetClothGround installs the terrain height used by cloak collisions.
func SetClothGround(height func(x, z float32) float32) {
	if height == nil {
		clothGround.Store(nil)
		return
	}
	clothGround.Store(&height)
}

type clothEnvironment struct {
	spheres  [4]render.V3
	radii    [4]float32
	count    int
	floor    float32
	hasFloor bool
	wind     render.V3
}

// environment samples the per-frame inputs from the pose.
func (c *cloakInstance) environment(world render.M4, finals []render.M4, collide bool) clothEnvironment {
	var e clothEnvironment
	if collide {
		for _, s := range clothBodySpheres[c.bodyType&15] {
			if s.bone < len(finals) {
				m := finals[s.bone].Mul(world)
				e.spheres[e.count] = render.V3{m[12], m[13], m[14]}
				e.radii[e.count] = s.radius
				e.count++
			}
		}
		if h := clothGround.Load(); h != nil {
			e.floor, e.hasFloor = (*h)(world[12], world[14]), true
		}
	}
	if clothWindBone < len(finals) {
		m := finals[clothWindBone].Mul(world)
		axis := render.V3{m[8], m[9], m[10]}
		if l := axis.Len(); l > 0 {
			e.wind = axis.Scale(-rand.Float32() * clothWindMax / l)
		}
	}
	return e
}

// collide keeps free vertices above the floor and outside the body spheres.
func (p *clothPhysics) collide(e *clothEnvironment) {
	if e.count == 0 && !e.hasFloor {
		return
	}
	for i := range p.positions {
		if p.pins[i] {
			continue
		}
		pos := p.positions[i]
		if e.hasFloor && pos[1] < e.floor {
			pos[1] = e.floor + clothFloorLift
			if p.velocity[i][1] < 0 {
				p.velocity[i][1] = 0
			}
		}
		for s := range e.count {
			d := pos.Sub(e.spheres[s])
			l := d.Len()
			r := e.radii[s]
			if l >= r {
				continue
			}
			var normal render.V3
			if l > 1e-6 {
				normal = d.Scale(1 / l)
			} else if w := e.wind.Len(); w > 0 {
				normal = e.wind.Scale(1 / w) // native deep contact: pushed backwards
			} else {
				continue
			}
			pos = e.spheres[s].Add(normal.Scale(r))
			if v := p.velocity[i].Dot(normal); v < 0 {
				p.velocity[i] = p.velocity[i].Sub(normal.Scale(v))
			}
		}
		p.positions[i] = pos
	}
}
