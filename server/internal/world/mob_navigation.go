package world

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/mapnav"
	"github.com/Spelunkern/Shaiya-GX/internal/movespeed"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
)

type mobNavigationCache struct {
	sync.Mutex
	maps      map[uint16]*mapnav.Map
	directory string
	used      map[uint16]time.Time
	missing   map[uint16]bool
}

var mobNavigation, dungeonMobNavigation mobNavigationCache

func loadMobNavigation(mapID uint16) (*mapnav.Map, error) {
	return mobNavigation.load(realMap(mapID), filepath.Join(config.DataDir(), "geometry"), true)
}

func loadDungeonMobNavigation(id uint16) (*mapnav.Map, error) {
	return dungeonMobNavigation.load(id, filepath.Join(config.DataDir(), "geometry", mapnav.MobDungeonDirectory), false)
}

func (cache *mobNavigationCache) load(id uint16, directory string, playerLadders bool) (*mapnav.Map, error) {
	cache.Lock()
	defer cache.Unlock()
	if directory != cache.directory {
		cache.directory = directory
		cache.maps = make(map[uint16]*mapnav.Map)
		cache.used = make(map[uint16]time.Time)
		cache.missing = make(map[uint16]bool)
	}
	if nav, ok := cache.maps[id]; ok {
		cache.used[id] = time.Now()
		return nav, nil
	}
	if cache.missing[id] {
		return nil, nil
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
		if !playerLadders {
			cache.missing[id] = true
		}
		return nil, nil
	}
	nav, err := mapnav.Load(directory, id)
	if err != nil {
		return nil, fmt.Errorf("map %d navigation: %w", id, err)
	}
	if g, ok := nav.PlayerGeometry(); ok && playerLadders {
		g.Ladders = mapgeometry.NativeLadders(id)
	}
	if len(cache.maps) >= 4 {
		var oldest uint16
		var at time.Time
		for key, used := range cache.used {
			if at.IsZero() || used.Before(at) {
				oldest, at = key, used
			}
		}
		delete(cache.maps, oldest)
		delete(cache.used, oldest)
	}
	cache.maps[id] = nav
	cache.used[id] = time.Now()
	return nav, nil
}
func (m *MapMob) point() mapnav.Point     { return mapnav.Point{m.PosX, m.PosY, m.PosZ} }
func (m *MapMob) setPoint(p mapnav.Point) { m.PosX, m.PosY, m.PosZ = p[0], p[1], p[2] }

func usesDungeonMobNavigation(nav *mapnav.Map, id uint16) bool {
	return mapConfig.Zones[id].Dungeon || nav.Terrain() == nil
}

func (m *MapMob) placeOnNavigation(nav *mapnav.Map, mapID uint16) bool {
	if nav == nil {
		m.nav = nil
	} else {
		m.nav = &mobNav{id: realMap(mapID)}
	}
	if nav == nil {
		return true
	}
	id := m.nav.id
	if usesDungeonMobNavigation(nav, id) {
		// Startup validates the companion generation. Existing installations
		// without it retain the original full-mesh dungeon navigation.
		structural, err := loadDungeonMobNavigation(id)
		if err != nil {
			return false
		}
		if structural != nil {
			nav = structural
			m.nav.dungeon = true
		}
	} else {
		m.nav.terrain = nav.Terrain()
	}
	if terrain := m.nav.terrain; terrain != nil {
		p := m.point()
		if !m.nav.allowsXZ(mapID, p[0], p[2]) {
			return false
		}
		p[1] = terrain.Height(p[0], p[2])
		m.setMobGround(p)
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
		if !ok || !m.nav.allowsXZ(mapID, p[0], p[2]) {
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
		if !checkExit || m.mobSpawnHasExit(nav, mapID, p) {
			spawn = p
			break
		}
	}
	if !found {
		return false
	}
	m.setMobGround(spawn)
	return true
}

func (m *MapMob) setMobGround(p mapnav.Point) {
	m.setPoint(p)
	m.cell, m.hasCell = cellFor(p[0], p[2]), true
	m.homeX, m.homeY, m.homeZ = p[0], p[1], p[2]
	m.navPath = nil
}

func (m *MapMob) mobSpawnHasExit(nav *mapnav.Map, mapID uint16, spawn mapnav.Point) bool {
	const distance = 6
	for _, offset := range [][2]float32{{distance, 0}, {-distance, 0}, {0, distance}, {0, -distance}} {
		goal := mapnav.Point{spawn[0] + offset[0], spawn[1], spawn[2] + offset[1]}
		path, ok := nav.Path(spawn, goal)
		if !ok || len(path) < 2 {
			continue
		}
		end := path[len(path)-1]
		if m.nav.allowsXZ(mapID, end[0], end[2]) {
			return true
		}
	}
	return false
}

func (m *MapMob) navigate(now time.Time, mapID uint16, goal mapnav.Point, step float32) (mapnav.Point, bool) {
	if terrain := m.nav.terrain; terrain != nil {
		p := m.point()
		d := DistanceXZ(p[0], p[2], goal[0], goal[2])
		if d < .015 || !m.nav.allowsXZ(mapID, goal[0], goal[2]) {
			return p, false
		}
		f := min(step/d, 1)
		p[0] += (goal[0] - p[0]) * f
		p[2] += (goal[2] - p[2]) * f
		p[1] = terrain.Height(p[0], p[2])
		return p, true
	}
	// Reuse a corridor while the target stays nearby, with bounded refresh.
	if len(m.navPath) == 0 || now.After(m.navRefresh) || mapnav.Distance(goal, m.navGoal) > 1 {
		path, ok := m.nav.Path(m.point(), goal)
		if !ok {
			m.navPath = nil
			return mapnav.Point{}, false
		}
		m.navPath = path
		m.navGoal = goal
		m.navRefresh = now.Add(time.Second)
	}
	for len(m.navPath) > 0 && mapnav.Distance(m.point(), m.navPath[0]) < .05 {
		m.navPath = m.navPath[1:]
	}
	p, _, consumed, ok := m.nav.Step(m.point(), m.navPath, step, func(x, z float32) bool { return m.nav.allowsXZ(mapID, x, z) })
	if !ok {
		m.navPath = nil
		return mapnav.Point{}, false
	}
	m.navPath = m.navPath[consumed:]
	return p, true
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
type mobNav struct {
	id      uint16
	terrain *mapgeometry.Geometry
	dungeon bool
}

func (n *mobNav) allowsXZ(mapID uint16, x, z float32) bool {
	if n != nil {
		if n.terrain != nil {
			return x >= 0 && z >= 0 && x <= float32(n.terrain.MapSize) && z <= float32(n.terrain.MapSize)
		}
		if n.dungeon {
			return true
		} // The structural mesh constrains movement.
	}
	return IsMapWalkable(mapID, x, z)
}

func (n *mobNav) movementMap() (*mapnav.Map, error) {
	if n.dungeon {
		return loadDungeonMobNavigation(n.id)
	}
	return loadMobNavigation(n.id)
}

func (n *mobNav) Path(a, b mapnav.Point) ([]mapnav.Point, bool) {
	m, err := n.movementMap()
	if err != nil || m == nil {
		return nil, false
	}
	return m.Path(a, b)
}
func (n *mobNav) Step(a mapnav.Point, path []mapnav.Point, d float32, walkable func(float32, float32) bool) (mapnav.Point, float32, int, bool) {
	m, err := n.movementMap()
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

func (m *MapMob) movingAt(now time.Time) bool {
	return m.moveDuration > 0 && now.Before(m.moveStart.Add(m.moveDuration))
}

func (m *MapMob) positionAt(now time.Time) mapnav.Point {
	p := m.point()
	if !m.movingAt(now) {
		return p
	}
	t := float32(max(0, now.Sub(m.moveStart))) / float32(m.moveDuration)
	for i := range p {
		p[i] = m.moveFrom[i] + (p[i]-m.moveFrom[i])*t
	}
	if m.nav != nil {
		if terrain := m.nav.terrain; terrain != nil {
			p[1] = terrain.Height(p[0], p[2])
			return p
		}
		if nav, err := m.nav.movementMap(); err == nil && nav != nil {
			if g, ok := nav.PlayerGeometry(); ok {
				if y, found := g.Support(p[0], p[1], p[2], .5); found {
					p[1] = y
				}
			}
		}
	}
	return p
}

func (m *MapMob) moveTo(mapID uint16, now time.Time, to mapnav.Point, motion uint8) entityMove {
	from := m.positionAt(now)
	tier := movespeed.MobMoveTier(m.mods().moveSpeedIndex())
	speed := 2.
	if m.def != nil {
		step, ms := m.def.NormalStep, m.def.NormalTimeMS
		if motion == mobMotionRun || step == 0 || ms == 0 {
			step, ms = m.def.ChaseStep, m.def.ChaseTimeMS
		}
		if v := movespeed.MobUnitsPerSecond(int(step), ms, tier); v > 0 {
			speed = v
		}
	}
	travel := DistanceXZ(from[0], from[2], to[0], to[2])
	m.moveFrom, m.moveStart, m.moveMotion, m.moveSpeedTier = from, now, motion, tier
	// Positive durations use the packet's millisecond resolution, including
	// segments that finish before the next AI sweep.
	m.moveDuration = max(time.Millisecond, time.Duration(math.Ceil(float64(travel)*1000/speed))*time.Millisecond)
	if !m.wireHeightKnown() && travel >= 20 {
		m.moveDuration = 0 // Native 0x0603 jumps at twenty horizontal units.
	}
	m.setPoint(to)
	oldCell, fresh := m.cell, !m.hasCell
	m.cell, m.hasCell = cellAfterMove(m.cell, m.hasCell, to[0], to[2]), true
	entering := *m
	entering.setPoint(from)
	return entityMove{MapID: mapID, Kind: viewMob, ID: m.ID, From: oldCell, To: m.cell, Fresh: fresh,
		Changed: fresh || oldCell != m.cell, Move: buildMobMove(m, motion), Enter: buildMobEnter(&entering, false)}
}

// The original client reads the current speed index every frame. Update the
// active segment from its current position so World and the client keep the
// same destination and arrival time when a speed effect changes.
func (m *MapMob) refreshMoveSpeed(mapID uint16, now time.Time, tick *MobCombatTick) {
	if m.movingAt(now) && movespeed.MobMoveTier(m.mods().moveSpeedIndex()) != m.moveSpeedTier {
		tick.Moves = append(tick.Moves, m.moveTo(mapID, now, m.point(), m.moveMotion))
	}
}

func (m *MapMob) appearance(now time.Time, fresh bool) (enter, move []byte) {
	if !m.movingAt(now) {
		return buildMobEnter(m, fresh), nil
	}
	snapshot := *m
	snapshot.setPoint(m.positionAt(now))
	remaining := *m
	remaining.moveDuration = max(time.Millisecond, m.moveStart.Add(m.moveDuration).Sub(now))
	return buildMobEnter(&snapshot, fresh), buildMobMove(&remaining, m.moveMotion)
}
