package collision

import "math"

const (
	cellSize          = 16
	triangleBlockSize = 32
)

// triangleRefs preserves insertion order while bounding consecutive groups of
// references. Bounds are built once; queries skip distant groups before dedup.
type triangleRefs struct {
	ids    []int32
	bounds []Box
}

func (r *triangleRefs) add(id int32, b Box) {
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

func (r triangleRefs) visit(b Box, emit func(int32)) {
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
	boxes      []Box // per-triangle bounds, computed once
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
	i.triangles = append(i.triangles, t)
	i.insert(int32(len(i.triangles)-1), t.Bounds())
}

// AddAll indexes ts. When every triangle is valid and the index is empty,
// it keeps ts as its storage instead of copying it.
func (i *Index) AddAll(ts []Triangle) {
	shared := len(i.triangles) == 0
	for _, t := range ts {
		if !t.Valid() {
			shared = false
			break
		}
	}
	if !shared {
		for _, t := range ts {
			i.Add(t)
		}
		return
	}
	i.triangles = ts
	i.boxes = make([]Box, 0, len(ts))
	for id, t := range ts {
		i.insert(int32(id), t.Bounds())
	}
}

func (i *Index) insert(id int32, b Box) {
	i.boxes = append(i.boxes, b)
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
	// Global and cell references are disjoint. A cell triangle spanning
	// several visited cells is emitted at the first of them in z/x order,
	// which is where the former seen-set admitted it.
	i.global.visit(b, func(id int32) {
		if b.Overlaps(i.boxes[id]) {
			visit(i.triangles[id])
		}
	})
	all := !gridBox(b)
	var x0, x1, z0, z1 int
	if !all {
		x0, x1, z0, z1 = cells(b)
		all = (x1-x0+1)*(z1-z0+1) > 4096
	}
	if all {
		global := map[int32]struct{}{}
		i.global.visit(b, func(id int32) { global[id] = struct{}{} })
		for id := range i.triangles {
			if _, ok := global[int32(id)]; !ok && b.Overlaps(i.boxes[id]) {
				visit(i.triangles[id])
			}
		}
		return
	}
	for z := z0; z <= z1; z++ {
		for x := x0; x <= x1; x++ {
			i.cells[[2]int{x, z}].visit(b, func(id int32) {
				tb := i.boxes[id]
				if !b.Overlaps(tb) {
					return
				}
				tx0, _, tz0, _ := cells(tb)
				if x == max(x0, tx0) && z == max(z0, tz0) {
					visit(i.triangles[id])
				}
			})
		}
	}
}

func gridBox(b Box) bool {
	return b.Min.Finite() && b.Max.Finite() && math.Abs(b.Min[0]) < 1e9 && math.Abs(b.Max[0]) < 1e9 && math.Abs(b.Min[2]) < 1e9 && math.Abs(b.Max[2]) < 1e9
}
