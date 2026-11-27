package mapnav

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/arl/go-detour/detour"
	"github.com/arl/gogeo/f32/d3"
)

// Point is a foot position, in original map coordinates.
type Point [3]float32

type Map struct {
	playerPhysics bool
	mu            sync.Mutex // Detour queries use mutable search state.
	query         *detour.NavMeshQuery
	geometry      *mapgeometry.Geometry
	terrain       *mapgeometry.Geometry
	filter        *detour.StandardQueryFilter
	corridor      []detour.PolyRef
	points        []d3.Vec3
	flags         []uint8
	refs          []detour.PolyRef
	navRead       func() ([]byte, error)
	navErr        error
}

// Load verifies the complete generation before making it available to AI.
func Load(directory string, id uint16) (result *Map, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = fmt.Errorf("invalid navigation map %d: %v", id, r)
		}
	}()
	raw, err := os.ReadFile(filepath.Join(directory, fmt.Sprintf("%d.manifest.json", id)))
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	if manifest.Version != FormatVersion || manifest.CollisionRules != CollisionRulesVersion || manifest.MapID != id {
		return nil, fmt.Errorf("navigation format or map mismatch")
	}
	read := func(extension string) ([]byte, error) {
		name := fmt.Sprintf("%d.%s", id, extension)
		p := filepath.Join(directory, name)
		info, e := os.Stat(p)
		if e != nil {
			return nil, e
		}
		if info.Size() > 256<<20 {
			return nil, fmt.Errorf("oversized navigation file %s", name)
		}
		b, e := os.ReadFile(p)
		if e == nil && digest(b) != manifest.Files[name] {
			e = fmt.Errorf("navigation checksum mismatch: %s", name)
		}
		return b, e
	}
	// The Detour mesh is decoded on first query; mobs and guards use the
	// .svmap mask, so ordinary play never needs it.
	navName := fmt.Sprintf("%d.gxnav", id)
	if info, e := os.Stat(filepath.Join(directory, navName)); e != nil {
		return nil, e
	} else if info.Size() > 256<<20 {
		return nil, fmt.Errorf("oversized navigation file %s", navName)
	}
	var g *mapgeometry.Geometry
	hasPlayerPhysics := manifest.PlayerPhysics == PlayerPhysicsVersion
	if hasPlayerPhysics {
		data, e := read("gxphysics")
		if e != nil {
			return nil, e
		}
		g, e = loadPlayerPhysics(data, id)
		if e != nil {
			return nil, e
		}
	} else {
		col, err := read("gxcol")
		if err != nil {
			return nil, err
		}
		gz, err := gzip.NewReader(bytes.NewReader(col))
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		var physics physicsFile
		if err = gob.NewDecoder(io.LimitReader(gz, 512<<20)).Decode(&physics); err != nil {
			return nil, err
		}
		if physics.Version != FormatVersion || physics.MapID != id {
			return nil, fmt.Errorf("collision format or map mismatch")
		}
		g = &mapgeometry.Geometry{Triangles: physics.Triangles, Dynamic: physics.Dynamic, MapSize: physics.MapSize, Heights: physics.Heights}
		g.Reindex()
	}
	g.SetDynamicClock(mapgeometry.PhysicsSeconds)
	var terrain *mapgeometry.Geometry
	if g.HasTerrain() {
		terrain = &mapgeometry.Geometry{MapSize: g.MapSize, Heights: g.Heights}
	}
	return &Map{playerPhysics: hasPlayerPhysics, geometry: g, terrain: terrain, navRead: func() ([]byte, error) { return read("gxnav") }}, nil
}

// loadQuery decodes the navigation mesh once. Callers hold m.mu.
func (m *Map) loadQuery() bool {
	if m.query != nil {
		return true
	}
	if m.navErr != nil || m.navRead == nil {
		return false
	}
	m.navErr = func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("invalid navigation mesh: %v", r)
			}
		}()
		nav, err := m.navRead()
		if err != nil {
			return err
		}
		mesh, err := detour.Decode(bytes.NewReader(nav))
		if err != nil {
			return err
		}
		status, q := detour.NewNavMeshQuery(mesh, 8192)
		if detour.StatusFailed(status) {
			return status
		}
		points := make([]d3.Vec3, 2048)
		for i := range points {
			points[i] = make(d3.Vec3, 3)
		}
		m.query, m.filter, m.points = q, detour.NewStandardQueryFilter(), points
		m.corridor, m.flags, m.refs = make([]detour.PolyRef, 2048), make([]uint8, 2048), make([]detour.PolyRef, 2048)
		return nil
	}()
	return m.navErr == nil
}

// Terrain returns the heightfield without retaining the collision index or mesh.
func (m *Map) Terrain() *mapgeometry.Geometry { return m.terrain }

func valid(p Point) bool {
	for _, v := range p {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return false
		}
	}
	return true
}
func (m *Map) nearest(p Point, horizontal, vertical float32) (detour.PolyRef, Point, bool) {
	if !m.loadQuery() {
		return 0, Point{}, false
	}
	if !valid(p) {
		return 0, Point{}, false
	}
	status, ref, at := m.query.FindNearestPoly(d3.Vec3(p[:]), d3.Vec3{horizontal, vertical, horizontal}, m.filter)
	if detour.StatusFailed(status) || ref == 0 {
		return 0, Point{}, false
	}
	result := Point{at[0], at[1], at[2]}
	if y, ok := m.geometry.Support(result[0], result[1], result[2], .65); ok {
		result[1] = y
	} else {
		return 0, Point{}, false
	}
	if abs(result[1]-p[1]) > vertical || abs(result[0]-p[0]) > horizontal || abs(result[2]-p[2]) > horizontal {
		return 0, Point{}, false
	}
	return ref, result, true
}
func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// Project preserves the hinted floor; callers choose a bounded search volume.
func (m *Map) Project(p Point, horizontal, vertical float32) (Point, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, out, ok := m.nearest(p, horizontal, vertical)
	return out, ok
}

// Spawn projects down from the authored volume, preserving the first floor hit.
func (m *Map) Spawn(p Point, horizontal, drop float32) (Point, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !valid(p) {
		return Point{}, false
	}
	if y, ok := m.geometry.Below(p[0], p[1]+.5, p[2], drop); ok {
		p[1] = y
		_, out, ok := m.nearest(p, horizontal, .65)
		return out, ok
	}
	// Outdoor svmap volumes may predate the shipped terrain heightmap. When
	// the volume is buried, the native terrain is the only physical base floor.
	if m.geometry.HasTerrain() && p[0] >= 0 && p[2] >= 0 && p[0] <= float32(m.geometry.MapSize) && p[2] <= float32(m.geometry.MapSize) {
		ground := m.geometry.Height(p[0], p[2])
		if ground > p[1] {
			p[1] = ground
			_, out, ok := m.nearest(p, horizontal, .65)
			return out, ok
		}
	}
	return Point{}, false
}

// Path returns corners on a connected surface. A partial path is never success.
func (m *Map) Path(start, end Point) ([]Point, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sr, s, ok := m.nearest(start, .6, .75)
	if !ok {
		return nil, false
	}
	er, e, ok := m.nearest(end, 1, 2)
	if !ok {
		return nil, false
	}
	corridor := m.corridor
	n, status := m.query.FindPath(sr, er, d3.Vec3(s[:]), d3.Vec3(e[:]), m.filter, corridor)
	if detour.StatusFailed(status) || status&(detour.PartialResult|detour.OutOfNodes|detour.BufferTooSmall) != 0 || n == 0 || corridor[n-1] != er {
		return nil, false
	}
	points := m.points
	n, status = m.query.FindStraightPath(d3.Vec3(s[:]), d3.Vec3(e[:]), corridor[:n], points, m.flags, m.refs, int32(detour.StraightPathAllCrossings))
	if detour.StatusFailed(status) || status&detour.BufferTooSmall != 0 || n < 1 {
		return nil, false
	}
	result := make([]Point, 0, n)
	for i, p := range points[:n] {
		// Straight-path portals interpolate polygon corner Y, which may differ
		// from detailed terrain on a large tile. Preserve the corridor's floor
		// reference while querying its detail mesh before original-surface support.
		ref := m.refs[i]
		if ref == 0 {
			ref = er
		}
		detail := make(d3.Vec3, 3)
		if status := m.query.ClosestPointOnPoly(ref, p, detail, nil); detour.StatusFailed(status) {
			return nil, false
		}
		v := Point{p[0], detail[1], p[2]}
		if y, ok := m.geometry.Support(v[0], v[1], v[2], .65); ok {
			v[1] = y
		} else {
			return nil, false
		}
		result = append(result, v)
	}
	return result, true
}

// Step consumes consecutive collinear polygon crossings in one movement.
// It stops at a genuine horizontal turn so interpolation cannot cut a wall.
// Height anchors remain in the path and every substep checks physical support.
func (m *Map) Step(start Point, path []Point, distance float32, walkable func(float32, float32) bool) (Point, float32, int, bool) {
	end := start
	travelled := float32(0)
	consumed := 0
	remaining := distance
	directionX, directionZ := float32(0), float32(0)
	for i, corner := range path {
		dx, dz := corner[0]-end[0], corner[2]-end[2]
		length := float32(math.Hypot(float64(dx), float64(dz)))
		if length < .015 {
			consumed = i + 1
			continue
		}
		ux, uz := dx/length, dz/length
		if travelled > 0 && (abs(directionX*uz-directionZ*ux) > .001 || directionX*ux+directionZ*uz < .999) {
			break
		}
		if travelled == 0 {
			directionX, directionZ = ux, uz
		}
		origin := end
		travel := min(length, remaining)
		count := max(1, int(math.Ceil(float64(travel/.2))))
		for j := 1; j <= count; j++ {
			f := travel / length * float32(j) / float32(count)
			hint := Point{origin[0] + dx*f, origin[1] + (corner[1]-origin[1])*f, origin[2] + dz*f}
			p, ok := m.Project(hint, .05, .65)
			if !ok || !walkable(p[0], p[2]) {
				return end, travelled, consumed, end != start
			}
			travelled += float32(math.Hypot(float64(p[0]-end[0]), float64(p[2]-end[2])))
			end = p
		}
		remaining -= travel
		if travel >= length-.001 {
			consumed = i + 1
		}
		if remaining < .001 {
			break
		}
	}
	return end, travelled, consumed, end != start
}
func Distance(a, b Point) float32 {
	return float32(math.Sqrt(float64((a[0]-b[0])*(a[0]-b[0]) + (a[1]-b[1])*(a[1]-b[1]) + (a[2]-b[2])*(a[2]-b[2]))))
}

// Visible rejects walls and intervening floors using the original triangles.
func (m *Map) Visible(start, end Point) bool {
	a := collision.Vec{float64(start[0]), float64(start[1] + .9), float64(start[2])}
	b := collision.Vec{float64(end[0]), float64(end[1] + .9), float64(end[2])}
	direction := b.Sub(a)
	visible := true
	bounds := collision.Box{}
	for i := range a {
		bounds.Min[i] = min(a[i], b[i]) - .001
		bounds.Max[i] = max(a[i], b[i]) + .001
	}
	m.geometry.Index.Visit(bounds, func(t collision.Triangle) {
		e1, e2 := t[1].Sub(t[0]), t[2].Sub(t[0])
		h := direction.Cross(e2)
		d := e1.Dot(h)
		if math.Abs(d) < 1e-9 {
			return
		}
		inv := 1 / d
		s := a.Sub(t[0])
		u := inv * s.Dot(h)
		if u < 0 || u > 1 {
			return
		}
		q := s.Cross(e1)
		v := inv * direction.Dot(q)
		if v < 0 || u+v > 1 {
			return
		}
		at := inv * e2.Dot(q)
		if at > .001 && at < .999 {
			visible = false
		}
	})
	return visible
}
