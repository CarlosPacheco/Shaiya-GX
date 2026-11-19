package scene

import (
	"math"
	"runtime"
	"sync"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// PrepareCharacterCloth advances visible capes before their draw calls.
// Characters must be distinct and belong to the current camera pass.
// Workers touch actor-owned poses and cloth state. Draw still owns all GL work,
// including uploads into the shared cloak mesh.
func PrepareCharacterCloth(characters []*Character) {
	prepareCharacterCloth(characters, nil)
}

// PrepareCrowdCloth reduces simulation cadence for small projected capes.
// Clipped and shadow captures retain the ordinary cloth path.
func PrepareCrowdCloth(characters []*Character, s *render.Scene3D) {
	prepareCharacterCloth(characters, s)
}

func prepareCharacterCloth(characters []*Character, s *render.Scene3D) {
	workers := min(4, runtime.GOMAXPROCS(0), len(characters)/12)
	if len(characters) < MinCrowdActors {
		return
	}
	workers = max(1, workers)
	prepare := func(offset int) {
		for i := offset; i < len(characters); i += workers {
			c := characters[i]
			if c.cloak != nil && c.clip != nil && !c.Hidden && c.CorpseFade < 1 {
				c.prepareCrowdCloak(s)
				c.cloak.prepared = true
			}
		}
	}
	var group sync.WaitGroup
	for offset := 1; offset < workers; offset++ {
		group.Go(func() { prepare(offset) })
	}
	prepare(0)
	group.Wait()
}

func (c *Character) prepareCrowdCloak(s *render.Scene3D) {
	cloak := c.cloak
	if s == nil || s.Shadow || s.ClipPlane != [4]float32{} || s.ViewportH <= 0 {
		cloak.update(c.World, c.finals())
		return
	}
	origin := render.V3{c.World[12], c.World[13], c.World[14]}
	depth := transformPoint(origin, s.View)[2]
	height := float32(s.ViewportH) * s.Proj[5] * render.V3{c.World[4], c.World[5], c.World[6]}.Len() / max(depth, .01)
	if height >= 72 || !cloak.initialized || origin.Sub(render.V3{cloak.world[12], cloak.world[13], cloak.world[14]}).Len() > .8 {
		cloak.lodInterval = 0
		cloak.update(c.World, c.finals())
		return
	}
	interval := 1.0 / 30
	budget := 5
	if height < 38 {
		interval = 1.0 / 20
	}
	if height < 24 {
		interval, budget = 1.0/12, 3
	}
	if cloak.lodInterval != interval {
		// Spread refreshes over successive frames in a dense formation.
		key := math.Float32bits(origin[0])*0x9e3779b1 ^ math.Float32bits(origin[2])*0x85ebca77
		phase := float64(key>>16) / 65536
		cloak.lodInterval = interval
		cloak.lodAt = c.enchantTime + phase*interval
	}
	if c.enchantTime < cloak.lodAt {
		cloak.reused = true
		return
	}
	cloak.lodAt += interval
	if cloak.lodAt <= c.enchantTime {
		cloak.lodAt = c.enchantTime + interval
	}
	cloak.updateWithBudget(c.World, c.finals(), budget)
}
