package collision

import "math"

const (
	cellSize          = 16
	triangleBlockSize = 32
)

// triangleRefs preserves insertion order while bounding consecutive groups of
// references. Bounds are built once; queries skip distant groups before dedup.
type triangleRefs struct {
	ids    []int
	bounds []Box
}

func (r *triangleRefs) add(id int, b Box) {
	block := len(r.ids) / triangleBlockSize
	if len(r.ids)%triangleBlockSize == 0 {
		r.bounds = append(r.bounds, b)
	} else {
		for a := range 3 {
			r.bounds[block].Min[a] = min(r.bounds[block].Min[a], b.Min[a])
			r.bounds[block].Max[a] = max(r.bounds[block].Max[a], b.Max[a])
		}
	}
	r.ids = append(r.ids, id)
}

func (r triangleRefs) visit(b Box, emit func(int)) {
	for block, bounds := range r.bounds {
		if !b.Overlaps(bounds) {
			continue
		}
		start := block * triangleBlockSize
		for _, id := range r.ids[start:min(start+triangleBlockSize, len(r.ids))] {
			emit(id)
		}
	}
}

// Index is an immutable uniform-grid broad phase. Queries own their scratch
// state; a prepared index can be read concurrently without render residency.
type Index struct {
	triangles  []Triangle
	cells      map[[2]int]triangleRefs
	global     triangleRefs
	minY, maxY float64
}

func NewIndex() *Index { return &Index{cells: map[[2]int]triangleRefs{}} }
func (i *Index) Len() int {
	if i == nil {
		return 0
	}
	return len(i.triangles)
}
func (i *Index) VerticalBounds() (low, high float64, ok bool) {
	if i.Len() == 0 {
		return 0, 0, false
	}
	return i.minY, i.maxY, true
}

func (i *Index) Add(t Triangle) {
	if !t.Valid() {
		return
	}
	id := len(i.triangles)
	i.triangles = append(i.triangles, t)
	b := t.Bounds()
	if id == 0 {
		i.minY, i.maxY = b.Min[1], b.Max[1]
	} else {
		i.minY, i.maxY = min(i.minY, b.Min[1]), max(i.maxY, b.Max[1])
	}
	if !gridBox(b) {
		i.global.add(id, b)
		return
	}
	x0, x1, z0, z1 := cells(b)
	if (x1-x0+1)*(z1-z0+1) > 64 {
		i.global.add(id, b)
		return
	}
	for z := z0; z <= z1; z++ {
		for x := x0; x <= x1; x++ {
			key := [2]int{x, z}
			refs := i.cells[key]
			refs.add(id, b)
			i.cells[key] = refs
		}
	}
}
func cells(b Box) (int, int, int, int) {
	return int(math.Floor(b.Min[0] / cellSize)), int(math.Floor(b.Max[0] / cellSize)), int(math.Floor(b.Min[2] / cellSize)), int(math.Floor(b.Max[2] / cellSize))
}
func (i *Index) Visit(b Box, visit func(Triangle)) {
	if i == nil {
		return
	}
	seen := map[int]struct{}{}
	emit := func(id int) {
		t := i.triangles[id]
		if !b.Overlaps(t.Bounds()) {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		visit(t)
	}
	i.global.visit(b, emit)
	if !gridBox(b) {
		for id := range i.triangles {
			emit(id)
		}
		return
	}
	x0, x1, z0, z1 := cells(b)
	if (x1-x0+1)*(z1-z0+1) > 4096 {
		for id := range i.triangles {
			emit(id)
		}
		return
	}
	for z := z0; z <= z1; z++ {
		for x := x0; x <= x1; x++ {
			i.cells[[2]int{x, z}].visit(b, emit)
		}
	}
}

func gridBox(b Box) bool {
	return b.Min.Finite() && b.Max.Finite() && math.Abs(b.Min[0]) < 1e9 && math.Abs(b.Max[0]) < 1e9 && math.Abs(b.Min[2]) < 1e9 && math.Abs(b.Max[2]) < 1e9
}
