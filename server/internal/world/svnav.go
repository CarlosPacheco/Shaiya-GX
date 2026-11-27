package world

import (
	"math"
	"os"
	"strings"
)

// Original server navigation for mobs and guards. ps_game.exe never loads
// map geometry: CMap keeps only the .svmap walk mask (one bit per world unit,
// index z*size+x, least significant bit first, 1 = blocked) and moves
// creatures over it with CMap::EnableLine 0042ab80/0042ade0, AStar::FindPath
// 0048eda0 and ChasePathList 0048f660. Field masks are almost entirely clear;
// dungeon masks hold the wall limits.

// svmapNavigationEnabled selects mask navigation. GX_MOB_NAV=geometry restores
// the previous navmesh movement.
var svmapNavigationEnabled = !strings.EqualFold(os.Getenv("GX_MOB_NAV"), "geometry")

type svTile struct{ x, z int32 }

type svGrid struct {
	size int32
	mask []byte
}

func gridFor(mapID uint16) (svGrid, bool) {
	realm.gridsMu.RLock()
	g, ok := realm.grids[mapID]
	realm.gridsMu.RUnlock()
	if !ok || g.size == 0 || len(g.mask) == 0 || g.size > math.MaxInt32 {
		return svGrid{}, false
	}
	return svGrid{int32(g.size), g.mask}, true
}

// tileOf truncates like _ftol2.
func tileOf(x, z float32) svTile {
	t := func(v float32) int32 {
		f := math.Trunc(float64(v))
		if math.IsNaN(f) || f < math.MinInt32 || f > math.MaxInt32 {
			return math.MinInt32
		}
		return int32(f)
	}
	return svTile{t(x), t(z)}
}

func (g svGrid) inside(t svTile) bool { return t.x >= 0 && t.z >= 0 && t.x < g.size && t.z < g.size }

// blocked is CMap::IsBlockSafe 0041b420: outside tiles are blocked.
func (g svGrid) blocked(t svTile) bool {
	if !g.inside(t) {
		return true
	}
	i := uint64(t.z)*uint64(g.size) + uint64(t.x)
	if i/8 >= uint64(len(g.mask)) {
		return true
	}
	return g.mask[i/8]>>(i%8)&1 != 0
}

// enableLine samples the tiles after start with the original float
// accumulator. On a blocked tile previous receives the last clear sample;
// a clear line leaves it at start.
func (g svGrid) enableLine(start, end svTile, previous *svTile) bool {
	if !g.inside(start) || !g.inside(end) {
		return false
	}
	if previous != nil {
		*previous = start
	}
	dx, dz := absI32(end.x-start.x), absI32(end.z-start.z)
	sx, sz := int32(1), int32(1)
	if end.x < start.x {
		sx = -1
	}
	if end.z < start.z {
		sz = -1
	}
	if dx == 0 && dz == 0 {
		return true
	}
	if dx > dz {
		step := float32(float64(float32(dz)) / float64(float32(dx)) * float64(sz))
		x, z := start.x+sx, float32(float64(float32(start.z))+float64(step))
		for range dx {
			if g.blocked(svTile{x, int32(z)}) {
				if previous != nil {
					*previous = svTile{x - sx, int32(float64(z) - float64(step))}
				}
				return false
			}
			x += sx
			z = float32(float64(z) + float64(step))
		}
		return true
	}
	step := float32(float64(float32(dx)) / float64(float32(dz)) * float64(sx))
	z, x := start.z+sz, float32(float64(float32(start.x))+float64(step))
	for range dz {
		if g.blocked(svTile{int32(x), z}) {
			if previous != nil {
				*previous = svTile{int32(float64(x) - float64(step)), z - sz}
			}
			return false
		}
		z += sz
		x = float32(float64(x) + float64(step))
	}
	return true
}

func absI32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// --- AStar::FindPath 0048eda0 -------------------------------------------

type astarNode struct {
	pos                     svTile
	key                     int32
	g, h, total             float32
	parent, child           *astarNode
	treeParent, left, right *astarNode
}

// svAStar keeps the original node trees: open ordered by total cost, closed
// by packed key, insertion ties to the right, squared costs and a search
// window of half the span (at most 20 tiles) around both endpoints.
type svAStar struct {
	pool         []*astarNode
	open, closed *astarNode
	start, goal  *astarNode
	successors   []*astarNode
	pending      []*astarNode
	// openKeys indexes the open tree. Each key is open at most once, so the
	// lookup returns the node the original full-tree scan would find.
	openKeys map[int32]*astarNode
}

func (a *svAStar) alloc(p svTile) *astarNode {
	var n *astarNode
	if k := len(a.pool); k > 0 {
		n, a.pool = a.pool[k-1], a.pool[:k-1]
		*n = astarNode{}
	} else {
		n = &astarNode{}
	}
	n.pos, n.key = p, int32(uint32(p.x)<<16+uint32(p.z))
	return n
}

func (a *svAStar) release(n *astarNode) { a.pool = append(a.pool, n) }

func astarPush(root **astarNode, n *astarNode, priority bool) {
	if *root == nil {
		*root, n.treeParent = n, nil
		return
	}
	cur := *root
	for {
		next := &cur.right
		if priority && n.total < cur.total || !priority && n.key < cur.key {
			next = &cur.left
		}
		if *next == nil {
			*next, n.treeParent = n, cur
			return
		}
		cur = *next
	}
}

func astarPop(root **astarNode) *astarNode {
	n := *root
	if n == nil {
		return nil
	}
	for n.left != nil {
		n = n.left
	}
	if n.treeParent == nil {
		*root = n.right
	} else {
		n.treeParent.left = n.right
	}
	if n.right != nil {
		n.right.treeParent = n.treeParent
	}
	return n
}

func (a *svAStar) find(root *astarNode, key int32, priority bool) *astarNode {
	if priority {
		return a.openKeys[key]
	}
	for root != nil && root.key != key {
		if key < root.key {
			root = root.left
		} else {
			root = root.right
		}
	}
	return root
}

func astarRemove(root **astarNode, n *astarNode, priority bool) {
	switch {
	case n.treeParent == nil:
		*root = nil
	case n.treeParent.left == n:
		n.treeParent.left = nil
	default:
		n.treeParent.right = nil
	}
	if n.left != nil {
		astarPush(root, n.left, priority)
	}
	if n.right != nil {
		astarPush(root, n.right, priority)
	}
}

func (a *svAStar) clearTree(root *astarNode, keepPath bool) {
	a.pending = a.pending[:0]
	if root != nil {
		a.pending = append(a.pending, root)
	}
	for len(a.pending) > 0 {
		n := a.pending[len(a.pending)-1]
		a.pending = a.pending[:len(a.pending)-1]
		if n.left != nil {
			a.pending = append(a.pending, n.left)
		}
		if n.right != nil {
			a.pending = append(a.pending, n.right)
		}
		if !keepPath || n.child == nil {
			a.release(n)
		}
	}
}

func astarCost(a, b svTile) float32 {
	x, z := float64(float32(a.x-b.x)), float64(float32(a.z-b.z))
	return float32(x*x + z*z)
}

func astarMargin(a, b int32) int32 { return min(absI32(a-b)/2, 20) }

var astarOffsets = [8]svTile{{-1, -1}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}}

// path returns the compressed waypoints after from, ending at to.
func (a *svAStar) path(g svGrid, from, to svTile, out []svTile) ([]svTile, bool) {
	out = out[:0]
	if from == to {
		return out, true
	}
	mx, mz := astarMargin(to.x, from.x), astarMargin(to.z, from.z)
	minX, minZ := max(min(from.x, to.x)-mx, 0), max(min(from.z, to.z)-mz, 0)
	maxX, maxZ := min(max(from.x, to.x)+mx, g.size-1), min(max(from.z, to.z)+mz, g.size-1)
	a.open, a.closed = nil, nil
	a.start, a.goal = a.alloc(from), a.alloc(to)
	a.start.h = astarCost(from, to)
	a.start.total = a.start.h
	if a.openKeys == nil {
		a.openKeys = make(map[int32]*astarNode, 256)
	}
	clear(a.openKeys)
	astarPush(&a.open, a.start, true)
	a.openKeys[a.start.key] = a.start
	for {
		cur := astarPop(&a.open)
		if cur == nil {
			break
		}
		delete(a.openKeys, cur.key)
		if cur.pos == to {
			a.goal.parent = cur.parent
			for n := a.goal; n != a.start; n = n.parent {
				n.parent.child = n
			}
			a.release(cur)
			a.clearTree(a.open, true)
			a.clearTree(a.closed, true)
			a.open, a.closed = nil, nil
			a.compress()
			for n := a.start.child; n != nil; n = n.child {
				out = append(out, n.pos)
			}
			for n := a.start; n != nil; {
				next := n.child
				a.release(n)
				n = next
			}
			a.start, a.goal = nil, nil
			return out, true
		}
		at := cur.pos
		if at.x >= minX && at.x <= maxX && at.z >= minZ && at.z <= maxZ {
			// GetSuccessorList allocates every candidate before inspecting any.
			a.successors = a.successors[:0]
			for _, d := range astarOffsets {
				p := svTile{at.x + d.x, at.z + d.z}
				if !g.blocked(p) {
					a.successors = append(a.successors, a.alloc(p))
				}
			}
			for _, c := range a.successors {
				cost := float32(float64(astarCost(at, c.pos)) + float64(cur.g))
				if e := a.find(a.open, c.key, true); e != nil {
					if e.g <= cost {
						a.release(c)
						continue
					}
					astarRemove(&a.open, e, true)
					delete(a.openKeys, e.key)
					a.release(e)
				}
				if e := a.find(a.closed, c.key, false); e != nil {
					if e.g <= cost {
						a.release(c)
						continue
					}
					astarRemove(&a.closed, e, false)
					a.release(e)
				}
				c.g, c.h = cost, astarCost(c.pos, to)
				c.total = float32(float64(c.h) + float64(cost))
				c.parent = cur
				astarPush(&a.open, c, true)
				a.openKeys[c.key] = c
			}
		}
		cur.treeParent, cur.left, cur.right = nil, nil, nil
		astarPush(&a.closed, cur, false)
	}
	a.release(a.goal)
	a.clearTree(a.open, false)
	a.clearTree(a.closed, false)
	a.open, a.closed, a.start, a.goal = nil, nil, nil, nil
	return out, false
}

// compress drops interior nodes that continue the previous direction.
func (a *svAStar) compress() {
	s := a.start
	if s == nil || s.child == nil || s.child.child == nil {
		return
	}
	n, next := s.child, s.child.child
	dx, dz := s.pos.x-n.pos.x, s.pos.z-n.pos.z
	for next != nil {
		nx, nz := n.pos.x-next.pos.x, n.pos.z-next.pos.z
		if nx == dx && nz == dz {
			n.parent.child = next
			next.parent = n.parent
			a.release(n)
		}
		dx, dz, n, next = nx, nz, next, next.child
	}
}

// --- per-creature movement ---------------------------------------------

// svRoute is a creature's cursor over its current A* route or its return
// trail, as in the original AStar/ChasePathList owners.
type svRoute struct {
	waypoints []svTile
	goal      svTile
	trail     []svTile
}

func (r *svRoute) reset() { r.waypoints = r.waypoints[:0] }

// addTrail is ChasePathList::AddPath: the head is replaced while the node
// after it still sees the new point.
func (r *svRoute) addTrail(g svGrid, p svTile) {
	if n := len(r.trail); n > 1 && g.enableLine(p, r.trail[n-2], nil) {
		r.trail[n-1] = p
		return
	}
	if n := len(r.trail); n > 0 && r.trail[n-1] == p {
		return
	}
	if len(r.trail) < 256 {
		r.trail = append(r.trail, p)
	}
}

// Pathfinders are pooled so node storage is reused across AI shards.
var svPathfinders = make(chan *svAStar, 64)

func borrowAStar() *svAStar {
	select {
	case a := <-svPathfinders:
		return a
	default:
		return &svAStar{}
	}
}

func returnAStar(a *svAStar) {
	select {
	case svPathfinders <- a:
	default:
	}
}

// svNext returns the next movement target toward goal for a creature at pos,
// limited to step and to its next route corner. ok is false when the goal
// tile is blocked or unreachable.
func svNext(g svGrid, r *svRoute, pos, goal [3]float32, step float32) ([3]float32, bool) {
	from, to := tileOf(pos[0], pos[2]), tileOf(goal[0], goal[2])
	if g.blocked(to) {
		r.reset()
		return pos, false
	}
	target := goal
	if len(r.waypoints) > 0 && r.goal != to {
		r.reset()
	}
	if len(r.waypoints) == 0 && !g.enableLine(from, to, nil) {
		a := borrowAStar()
		var ok bool
		r.waypoints, ok = a.path(g, from, to, r.waypoints)
		returnAStar(a)
		if !ok {
			r.reset()
			return pos, false
		}
		r.goal = to
	}
	for len(r.waypoints) > 0 {
		w := r.waypoints[0]
		if w == to {
			r.reset()
			break
		}
		target = [3]float32{float32(w.x) + .5, pos[1], float32(w.z) + .5}
		if w != from && DistanceXZ(pos[0], pos[2], target[0], target[2]) >= .015 {
			break
		}
		r.waypoints = r.waypoints[1:]
		target = goal
	}
	d := DistanceXZ(pos[0], pos[2], target[0], target[2])
	if d < .015 {
		return pos, false
	}
	next := target
	if d > step {
		f := step / d
		next = [3]float32{pos[0] + (target[0]-pos[0])*f, pos[1], pos[2] + (target[2]-pos[2])*f}
	} else if len(r.waypoints) > 0 {
		r.waypoints = r.waypoints[1:]
	}
	if g.blocked(tileOf(next[0], next[2])) {
		r.reset()
		return pos, false
	}
	return next, true
}
