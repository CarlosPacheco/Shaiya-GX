package scene

import (
	"math"
	"runtime"
	"sync"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func parallelMap(count int, work func(start, end int)) {
	workers := max(1, min(runtime.GOMAXPROCS(0), count/128, 8))
	chunk := (count + workers - 1) / workers
	var group sync.WaitGroup
	for worker := 1; worker < workers; worker++ {
		lo, hi := worker*chunk, min((worker+1)*chunk, count)
		group.Go(func() { work(lo, hi) })
	}
	work(0, min(count, chunk))
	group.Wait()
}

type mapBounds struct {
	lo, hi render.V3
	valid  bool
}

func staticMeshBounds(m *render.Mesh, transform render.M4) mapBounds {
	lo, hi, ok := m.Bounds()
	if !ok {
		return mapBounds{}
	}
	b := mapBounds{valid: true}
	for i := range 8 {
		p := lo
		for axis := range 3 {
			if i&(1<<axis) != 0 {
				p[axis] = hi[axis]
			}
		}
		p = transformPoint(p, transform)
		if i == 0 {
			b.lo, b.hi = p, p
		} else {
			b.add(p, p)
		}
	}
	return b
}

func (b *mapBounds) add(lo, hi render.V3) {
	for axis := range 3 {
		b.lo[axis], b.hi[axis] = min(b.lo[axis], lo[axis]), max(b.hi[axis], hi[axis])
	}
}

func (b mapBounds) visible(f render.Frustum, clip [4]float32, o *render.TerrainOcclusion) bool {
	return !b.valid || f.BoxVisible(b.lo, b.hi) && render.ClipBoxVisible(clip, b.lo, b.hi) && !o.BoxHidden(b.lo, b.hi)
}

type dungeonDrawEntry struct {
	mesh    *render.Mesh
	texture *render.Texture
	id      uint32
	sector  int
	bounds  mapBounds
	mani    bool
}

type dungeonDrawPlan struct {
	entries       []dungeonDrawEntry
	sectors       []mapBounds
	sectorVisible []bool
	partVisible   []bool
	dgVisible     []bool
}

func (w *World) prepareMapBatch() {
	b := &w.mapBatch
	b.Transforms = append(b.Transforms, render.Identity())
	for _, obj := range w.Objects {
		b.Transforms = append(b.Transforms, obj.World)
	}
	if w.stream != nil {
		for _, obj := range w.stream.objects {
			b.Transforms = append(b.Transforms, obj.world)
		}
	}

	p := &w.dungeonPlan
	byCell := make(map[[3]int]int)
	for _, mani := range []bool{true, false} {
		for id, obj := range w.Objects {
			if obj.Mani != mani || obj.Model == nil {
				continue
			}
			cell := [3]int{}
			for axis := range cell {
				cell[axis] = int(math.Floor(float64(obj.World[12+axis] / 32)))
			}
			for _, part := range obj.Model.Parts {
				bounds := staticMeshBounds(part.Mesh, obj.World)
				sector := -1
				if bounds.valid {
					var found bool
					sector, found = byCell[cell]
					if !found {
						sector = len(p.sectors)
						byCell[cell] = sector
						p.sectors = append(p.sectors, bounds)
					} else {
						p.sectors[sector].add(bounds.lo, bounds.hi)
					}
				}
				p.entries = append(p.entries, dungeonDrawEntry{part.Mesh, part.Tex, uint32(id + 1), sector, bounds, mani})
			}
		}
	}
	p.partVisible = make([]bool, len(p.entries))
	p.sectorVisible = make([]bool, len(p.sectors))
	p.dgVisible = make([]bool, len(w.dg))
}

func (w *World) drawDG(s *render.Scene3D, f render.Frustum) {
	if len(w.dg) == 0 {
		return
	}
	o, clip := s.ActiveOcclusion(), s.ClipPlane
	p := &w.dungeonPlan
	parallelMap(len(w.dg), func(lo, hi int) {
		for i := lo; i < hi; i++ {
			g := &w.dg[i]
			p.dgVisible[i] = (mapBounds{g.minimum, g.maximum, true}).visible(f, clip, o)
		}
	})
	first := w.mapBatch.Placements([]uint32{0})
	for i, g := range w.dg {
		if p.dgVisible[i] {
			w.mapBatch.Add(g.mesh, g.tex, g.lightmap, render.NativeWorldAlphaRef(true), first, 1)
		}
	}
	w.mapBatch.Draw(s)
}
