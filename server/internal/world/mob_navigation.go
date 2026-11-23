package world

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/movespeed"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/mapnav"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
)

var mobNavigation = struct {
	sync.Mutex
	maps      map[uint16]*mapnav.Map
	directory string
	used      map[uint16]time.Time
}{maps: make(map[uint16]*mapnav.Map)}

func loadMobNavigation(mapID uint16) (*mapnav.Map, error) {
	id := realMap(mapID)
	directory := filepath.Join(config.DataDir(), "geometry")
	mobNavigation.Lock()
	defer mobNavigation.Unlock()
	if directory != mobNavigation.directory {
		mobNavigation.directory = directory
		mobNavigation.maps = make(map[uint16]*mapnav.Map)
		mobNavigation.used = make(map[uint16]time.Time)
	}
	if nav, ok := mobNavigation.maps[id]; ok {
		mobNavigation.used[id] = time.Now()
		return nav, nil
	}
	if _, err := os.Stat(filepath.Join(directory, fmt.Sprintf("%d.manifest.json", id))); os.IsNotExist(err) {
		if catalog, e := mapnav.ReadCatalog(directory); e == nil {
			for _, required := range catalog.Maps {
				if required == id {
					return nil, fmt.Errorf("required navigation map %d is missing", id)
				}
			}
		} else if !os.IsNotExist(e) {
			return nil, e
		}
		return nil, nil
	}
	nav, err := mapnav.Load(directory, id)
	if err != nil {
		return nil, fmt.Errorf("map %d navigation: %w", id, err)
	}
	if g, ok := nav.PlayerGeometry(); ok {
		g.Ladders = mapgeometry.NativeLadders(id)
	}
	if len(mobNavigation.maps) >= 4 {
		var oldest uint16
		var at time.Time
		for key, used := range mobNavigation.used {
			if at.IsZero() || used.Before(at) {
				oldest, at = key, used
			}
		}
		delete(mobNavigation.maps, oldest)
		delete(mobNavigation.used, oldest)
	}
	mobNavigation.maps[id] = nav
	mobNavigation.used[id] = time.Now()
	return nav, nil
}
func (m *MapMob) point() mapnav.Point     { return mapnav.Point{m.PosX, m.PosY, m.PosZ} }
func (m *MapMob) setPoint(p mapnav.Point) { m.PosX, m.PosY, m.PosZ = p[0], p[1], p[2] }

func (m *MapMob) placeOnNavigation(nav *mapnav.Map, mapID uint16) bool {
	if nav == nil {
		m.nav = nil
	} else {
		m.nav = &mobNav{id: realMap(mapID)}
	}
	if nav == nil {
		return true
	}
	hint := m.point()
	// SVMap box height selects the floor. Never search for the highest floor.
	tolerance := float32(math.MaxFloat32 / 4)
	if m.spawnHeightKnown {
		hint[1] = (m.spawnMinY + m.spawnMaxY) / 2
	}
	// Grounded polygons can still be isolated from the surrounding terrain.
	// Prefer a spawn with room to leave when the mob is meant to move.
	checkExit := m.Area != nil && m.def != nil && !m.def.Obelisk && (m.def.NormalStep != 0 || m.def.ChaseStep != 0)
	var spawn mapnav.Point
	found := false
	attempts := 1
	if m.Area != nil {
		attempts += 64 // areaCandidate repeats after one 8x8 grid.
	}
	for attempt := range attempts {
		if attempt > 0 {
			a := m.Area.normalized()
			p := areaCandidate(vec3{a.LowerX, hint[1], a.LowerZ}, vec3{a.UpperX, hint[1], a.UpperZ}, m.ID+uint32(attempt)-1)
			hint[0], hint[2] = p.x, p.z
		}
		p, ok := nav.Spawn(hint, .75, tolerance)
		if !ok || !IsMapWalkable(mapID, p[0], p[2]) {
			continue
		}
		if m.Area != nil {
			a := m.Area.normalized()
			if p[0] < a.LowerX || p[0] > a.UpperX || p[2] < a.LowerZ || p[2] > a.UpperZ {
				continue
			}
		}
		if !found {
			spawn, found = p, true
		}
		if !checkExit || mobSpawnHasExit(nav, mapID, p) {
			spawn = p
			break
		}
	}
	if !found {
		return false
	}
	m.setPoint(spawn)
	m.cell, m.hasCell = cellFor(spawn[0], spawn[2]), true
	m.homeX, m.homeY, m.homeZ = spawn[0], spawn[1], spawn[2]
	m.navPath = nil
	return true
}

func mobSpawnHasExit(nav *mapnav.Map, mapID uint16, spawn mapnav.Point) bool {
	const distance = 6
	for _, offset := range [][2]float32{{distance, 0}, {-distance, 0}, {0, distance}, {0, -distance}} {
		goal := mapnav.Point{spawn[0] + offset[0], spawn[1], spawn[2] + offset[1]}
		path, ok := nav.Path(spawn, goal)
		if !ok || len(path) < 2 {
			continue
		}
		end := path[len(path)-1]
		if IsMapWalkable(mapID, end[0], end[2]) {
			return true
		}
	}
	return false
}

func (m *MapMob) navigate(mapID uint16, goal mapnav.Point, step float32) (float32, bool) {
	if m.nav == nil {
		return 0, false
	}
	// Reuse a corridor while the target stays nearby, with bounded refresh.
	if len(m.navPath) == 0 || time.Now().After(m.navRefresh) || mapnav.Distance(goal, m.navGoal) > 1 {
		path, ok := m.nav.Path(m.point(), goal)
		if !ok {
			m.navPath = nil
			return 0, false
		}
		m.navPath = path
		m.navGoal = goal
		m.navRefresh = time.Now().Add(time.Second)
	}
	for len(m.navPath) > 0 && mapnav.Distance(m.point(), m.navPath[0]) < .05 {
		m.navPath = m.navPath[1:]
	}
	p, d, consumed, ok := m.nav.Step(m.point(), m.navPath, step, func(x, z float32) bool { return IsMapWalkable(mapID, x, z) })
	if !ok {
		m.navPath = nil
		return 0, false
	}
	m.navPath = m.navPath[consumed:]
	m.setPoint(p)
	return d, true
}
func (m *MapMob) targetDistance(t *PlayerCombatSnapshot) float32 {
	if m.nav == nil {
		return DistanceXZ(m.PosX, m.PosZ, t.PosX, t.PosZ)
	}
	return mapnav.Distance(m.point(), mapnav.Point{t.PosX, t.PosY, t.PosZ})
}
func (m *MapMob) canReachTarget(t *PlayerCombatSnapshot) bool {
	if m.nav == nil {
		return true
	}
	return m.nav.Visible(m.point(), mapnav.Point{t.PosX, t.PosY, t.PosZ})
}

// Mobs retain a lightweight map reference. The decoded geometry cache is bounded
// instead of keeping every world map resident merely because it has spawns.
type mobNav struct{ id uint16 }

func (n *mobNav) Path(a, b mapnav.Point) ([]mapnav.Point, bool) {
	m, err := loadMobNavigation(n.id)
	if err != nil || m == nil {
		return nil, false
	}
	return m.Path(a, b)
}
func (n *mobNav) Step(a mapnav.Point, path []mapnav.Point, d float32, walkable func(float32, float32) bool) (mapnav.Point, float32, int, bool) {
	m, err := loadMobNavigation(n.id)
	if err != nil || m == nil {
		return a, 0, 0, false
	}
	return m.Step(a, path, d, walkable)
}
func (n *mobNav) Visible(a, b mapnav.Point) bool {
	m, err := loadMobNavigation(n.id)
	return err == nil && m != nil && m.Visible(a, b)
}
func (m *MapMob) placeRespawn(mapID uint16) bool {
	nav, err := loadMobNavigation(mapID)
	return err == nil && nav != nil && m.placeOnNavigation(nav, mapID)
}

// Corner arrivals need a finer scheduler than the legacy half-second sweep.
const mobNavigationTick = 50 * time.Millisecond

func (m *MapMob) movementTiming(now time.Time, travel, step float32, interval time.Duration) {
	duration := time.Duration(float64(interval) * float64(min(1, travel/step)))
	// Match the next scheduler boundary so the client does not finish early and
	// stand idle while the next corner packet waits for a sweep.
	duration = max(mobNavigationTick, (duration+mobNavigationTick-1)/mobNavigationTick*mobNavigationTick)
	m.moveDuration = duration
	m.navHop = true
	if m.def != nil {
		mods := m.mods()
		if v := movespeed.MobUnitsPerSecond(int(m.def.ChaseStep), m.def.ChaseTimeMS, uint8(min(max(2+mods.MoveSpeed, 0), 4))); v > 0 {
			// The client draws the hop at the original constant speed.
			m.moveDuration = max(mobNavigationTick, time.Duration(float64(travel)/v*float64(time.Second)))
		}
	}
	m.nextChaseAt = now.Add(duration)
}
