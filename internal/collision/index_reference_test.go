package collision

// referenceIndex retains the original cell scan for behavioral and cost comparisons.
type referenceIndex struct {
	triangles []Triangle
	cells     map[[2]int][]int
	global    []int
}

func newReferenceIndex() *referenceIndex { return &referenceIndex{cells: map[[2]int][]int{}} }
func (i *referenceIndex) Len() int {
	if i == nil {
		return 0
	}
	return len(i.triangles)
}
func (i *referenceIndex) Add(t Triangle) {
	if !t.Valid() {
		return
	}
	id := len(i.triangles)
	i.triangles = append(i.triangles, t)
	b := t.Bounds()
	if !gridBox(b) {
		i.global = append(i.global, id)
		return
	}
	x0, x1, z0, z1 := cells(b)
	if (x1-x0+1)*(z1-z0+1) > 64 {
		i.global = append(i.global, id)
		return
	}
	for z := z0; z <= z1; z++ {
		for x := x0; x <= x1; x++ {
			key := [2]int{x, z}
			i.cells[key] = append(i.cells[key], id)
		}
	}
}
func (i *referenceIndex) Visit(b Box, visit func(Triangle)) {
	if i == nil {
		return
	}
	seen := map[int]struct{}{}
	emit := func(id int) {
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		t := i.triangles[id]
		if b.Overlaps(t.Bounds()) {
			visit(t)
		}
	}
	for _, id := range i.global {
		emit(id)
	}
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
			for _, id := range i.cells[[2]int{x, z}] {
				emit(id)
			}
		}
	}
}
