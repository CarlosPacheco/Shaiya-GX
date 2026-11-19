package scene

import (
	"runtime"
	"sync"
)

// PrepareCharacterCloth advances visible capes before their draw calls.
// Characters must be distinct and belong to the current camera pass.
// Workers touch actor-owned poses and cloth state. Draw still owns all GL work,
// including uploads into the shared cloak mesh.
func PrepareCharacterCloth(characters []*Character) {
	workers := min(4, runtime.GOMAXPROCS(0), len(characters)/12)
	if workers < 2 {
		return
	}
	prepare := func(offset int) {
		for i := offset; i < len(characters); i += workers {
			c := characters[i]
			if c.cloak != nil && c.clip != nil && !c.Hidden && c.CorpseFade < 1 {
				c.cloak.update(c.World, c.finals())
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
