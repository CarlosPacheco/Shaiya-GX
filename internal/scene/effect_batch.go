package scene

import (
	"runtime"
	"sync"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type EffectUpdate struct {
	Instance *Effect
	Delta    float32
}

// UpdateEffects advances independent instances. Each job owns its particles;
// shared geometry and emitter definitions are read-only to these workers.
func UpdateEffects(updates []EffectUpdate) {
	parallelEffects(len(updates), func(i int) {
		job := updates[i]
		job.Instance.Update(job.Delta)
	})
}

// DrawEffects prepares independent effects concurrently, then submits their
// commands in the original effect, emitter and particle order.
func DrawEffects(s *render.Scene3D, effects []*Effect) {
	if len(effects) == 0 {
		return
	}
	view := s.View
	parallelEffects(len(effects), func(i int) { effects[i].prepareDraw(view) })
	s.BeginEffects()
	for _, effect := range effects {
		effect.submitDraw(s)
	}
	s.EndEffects()
}

func parallelEffects(count int, work func(int)) {
	workers := max(1, min(runtime.GOMAXPROCS(0), count/12))
	run := func(offset int) {
		for i := offset; i < count; i += workers {
			work(i)
		}
	}
	var group sync.WaitGroup
	for i := 1; i < workers; i++ {
		group.Go(func() { run(i) })
	}
	run(0)
	group.Wait()
}
