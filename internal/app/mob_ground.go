package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// World collision geometry is immutable after loading. A stationary mob can
// retain its visual floor until its input, height authority or lifecycle changes.
type mobGroundCache struct {
	world         *scene.World
	input, output render.V3
	reset         uint64
	known, valid  bool
}

func (c *mobGroundCache) resolve(w *scene.World, p render.V3, known bool, reset uint64) render.V3 {
	if c.valid && c.world == w && c.input == p && c.known == known && c.reset == reset {
		return c.output
	}
	c.world, c.input, c.known, c.reset, c.valid = w, p, known, reset, true
	if known {
		p = w.MobGroundPosition(p)
	} else if w != nil && w.HasTerrain() {
		p[1] = w.Height(p[0], p[2])
	}
	c.output = p
	return p
}
