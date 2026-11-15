package world

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/mapnav"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAuthoritativeMobHeightProtocol(t *testing.T) {
	m := NewStaticMob(nil, 1, 12, 34, nil, 0)
	m.PosY = 11.5
	m.nav = &mobNav{}
	m.moveDuration = 250 * time.Millisecond
	spawn, err := proto.DecodeMobEnter(buildMobEnter(&m, false)[2:])
	if err != nil || !spawn.HeightKnown || spawn.Y != 11.5 {
		t.Fatalf("spawn height %v %v", spawn, err)
	}
	move, err := proto.DecodeMobMove(buildMobMove(&m, 1)[2:])
	if err != nil || !move.HeightKnown || move.Y != 11.5 || move.DurationMS != 250 {
		t.Fatalf("move height %v %v", move, err)
	}
	m.nav = nil
	spawn, err = proto.DecodeMobEnter(buildMobEnter(&m, false)[2:])
	if err != nil || spawn.HeightKnown {
		t.Fatal("legacy mob layout changed")
	}
}
func TestOriginalMobNavigation(t *testing.T) {
	directory := os.Getenv("GX_TEST_GEOMETRY")
	if directory == "" {
		t.Skip("requires local derived navigation data")
	}
	ids := []uint16{3, 1, 47}
	if words := os.Getenv("GX_TEST_MAPS"); words != "" {
		ids = nil
		for _, word := range strings.Split(words, ",") {
			id, e := strconv.ParseUint(word, 10, 16)
			if e != nil {
				t.Fatal(e)
			}
			ids = append(ids, uint16(id))
		}
	}
	for _, id := range ids {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			nav, err := mapnav.Load(directory, id)
			if err != nil {
				t.Fatal(err)
			}
			load, err := loadSvmapFile(id, filepath.Join(config.DataDir(), "maps", fmt.Sprint(id)+".svmap"), &db.Defs{Mobs: map[uint16]*db.MobDef{}})
			if err != nil {
				t.Fatal(err)
			}
			RegisterMapGrid(id, load.mapSize, load.mask)
			grounded, rejected, paths := 0, 0, 0
			var first *MapMob
			var routeGoal mapnav.Point
			haveGoal := false
			for i := range load.entities.Mobs {
				m := &load.entities.Mobs[i]
				if !m.placeOnNavigation(nav, id) {
					rejected++
					if rejected < 3 {
						p, ok := nav.Project(m.point(), 5, 1000)
						t.Logf("rejected %v yband=%g..%g nearest=%v %v", m.point(), m.spawnMinY, m.spawnMaxY, p, ok)
					}

					continue
				}
				grounded++
				if first == nil {
					first = m
					continue
				}
				if paths < 10 && DistanceXZ(first.PosX, first.PosZ, m.PosX, m.PosZ) < 100 {
					if path, ok := nav.Path(first.point(), m.point()); ok && len(path) > 1 {
						paths++
						if !haveGoal && mapnav.Distance(first.point(), m.point()) > 5 {
							routeGoal = m.point()
							haveGoal = true
						}
					}
				}
			}
			t.Logf("map %d: grounded=%d rejected=%d routes=%d total=%d", id, grounded, rejected, paths, len(load.entities.Mobs))
			if grounded == 0 || paths == 0 {
				t.Fatal("real map has no usable mob navigation")
			}
			if id == 3 && grounded < len(load.entities.Mobs)*9/10 {
				t.Error("too many map 3 spawns rejected")
			}
			if !haveGoal {
				t.Fatal("no real route for movement validation")
			}
			// Use the same bounded runtime cache as chase, home and respawn.
			mobNavigation.Lock()
			oldMaps, oldUsed, oldDirectory := mobNavigation.maps, mobNavigation.used, mobNavigation.directory
			mobNavigation.maps = map[uint16]*mapnav.Map{id: nav}
			mobNavigation.used = map[uint16]time.Time{id: time.Now()}
			mobNavigation.directory = filepath.Join(config.DataDir(), "geometry")
			mobNavigation.Unlock()
			t.Cleanup(func() {
				mobNavigation.Lock()
				mobNavigation.maps, mobNavigation.used, mobNavigation.directory = oldMaps, oldUsed, oldDirectory
				mobNavigation.Unlock()
			})
			origin := first.point()
			first.def = &db.MobDef{ChaseStep: 2, ChaseTimeMS: 500}
			for i := 0; i < 3000 && mapnav.Distance(first.point(), routeGoal) > .1; i++ {
				if _, ok := first.navigate(id, routeGoal, 2); !ok {
					t.Fatalf("real route stalled at %v toward %v", first.point(), routeGoal)
				}
			}
			if mapnav.Distance(first.point(), routeGoal) > .1 {
				t.Fatal("real route did not finish")
			}
			first.returningHome = true
			mods := first.mods()
			tick := MobCombatTick{}
			start := time.Now()
			now := start
			first.nextChaseAt = time.Time{}
			travelled := float32(0)
			for i := 0; i < 3000 && first.returningHome; i++ {
				before := first.point()
				first.stepHome(now, id, &mods, &tick)
				travelled += DistanceXZ(before[0], before[2], first.PosX, first.PosZ)
				now = now.Add(mobNavigationTick)
			}
			elapsed := now.Sub(start).Seconds()
			speed := float64(travelled) / elapsed
			if speed < 3.2 {
				_, backPath := nav.Path(first.point(), origin)
				t.Fatalf("map %d path reduced nominal 4 units/s to %.2f (%g units / %.2fs); current=%v home=%v reversePath=%t", id, speed, travelled, elapsed, first.point(), origin, backPath)
			}
			t.Logf("map %d simulated home speed %.2f units/s (nominal 4)", id, speed)
			if first.returningHome || mapnav.Distance(first.point(), origin) > .1 {
				t.Fatalf("home route failed: %v expected %v", first.point(), origin)
			}
			t.Logf("map %d: followed real route and returned home (%d movement packets)", id, len(tick.Moves))

		})
	}
}
