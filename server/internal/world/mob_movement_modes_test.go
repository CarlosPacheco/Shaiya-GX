package world

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/mapnav"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestFieldMobMovesInXZAndFollowsTerrain(t *testing.T) {
	const id = 903
	RegisterMapGrid(id, 4, []byte{0xff, 0xff})
	m := MapMob{nav: &mobNav{terrain: &mapgeometry.Geometry{
		MapSize: 4,
		Heights: []uint16{10000, 10500, 11000, 10000, 10500, 11000, 10000, 10500, 11000},
	}}}
	for _, y := range []float32{-100, 100} {
		got, ok := m.navigate(time.Time{}, id, mapnav.Point{4, y, 0}, 2)
		if !ok || got != (mapnav.Point{2, 10, 0}) {
			t.Fatalf("step toward Y=%g: got %v, %v", y, got, ok)
		}
	}
	m.moveTo(id, time.Unix(1, 0), mapnav.Point{2, 10, 0}, mobMotionRun)
	if got := m.positionAt(m.moveStart.Add(m.moveDuration / 2)); got != (mapnav.Point{1, 5, 0}) {
		t.Fatalf("interpolated position: %v", got)
	}
}

func TestMobMovementModesOnRealMaps(t *testing.T) {
	data := os.Getenv("GX_TEST_MOB_MOVEMENT_DATA")
	if data == "" {
		t.Skip("requires full geometry and structure-only dungeon navigation")
	}
	t.Setenv("RAPTURE_DATA_DIR", data)
	for _, tc := range []struct {
		name       string
		id         uint16
		start, end mapnav.Point
		moves      bool
	}{
		{"field prop", 1, mapnav.Point{1517.0399, 16.851233, 926.8113}, mapnav.Point{1529.0399, 18.017159, 926.8113}, true},
		{"field climb", 1, mapnav.Point{1117.6882, -2.4364212, 429.29654}, mapnav.Point{1117.6882, 4.586901, 445.29654}, true},
		{"dungeon prop", 3, mapnav.Point{405.85962, 13.241182, 255.15433}, mapnav.Point{409.85962, 13.241182, 255.15433}, true},
		{"dungeon prop elevated target", 3, mapnav.Point{405.85962, 13.241182, 255.15433}, mapnav.Point{409.85962, 40, 255.15433}, true},
		{"dungeon wall", 3, mapnav.Point{320.8829, 13.241196, 287.5975}, mapnav.Point{323.7109, 12.92228, 290.4255}, true},
		{"connected floors", 3, mapnav.Point{134.67891, 18.190376, 27.616495}, mapnav.Point{126.67891, 20.4074, 27.616495}, true},
		{"separate floors", 3, mapnav.Point{146.5, 19.76707, 44.5}, mapnav.Point{146.5, 17.663082, 44.5}, false},
		{"separate floors reverse", 3, mapnav.Point{146.5, 17.663082, 44.5}, mapnav.Point{146.5, 19.76707, 44.5}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nav, err := loadMobNavigation(tc.id)
			if err != nil || nav == nil {
				t.Fatalf("load map %d: %v", tc.id, err)
			}
			m := MapMob{AggroTarget: 1, def: &db.MobDef{ChaseStep: 2, ChaseTimeMS: 500, Size: 1}}
			m.setPoint(tc.start)
			m.spawnHeightKnown, m.spawnMinY, m.spawnMaxY = true, tc.start[1]+.5, tc.start[1]+.5
			if !m.placeOnNavigation(nav, tc.id) || (tc.id == 3 && !m.nav.dungeon) || (tc.id == 1 && m.nav.terrain == nil) {
				t.Fatalf("map %d did not use the expected movement mode (data %s)", tc.id, filepath.Clean(data))
			}
			// Exercise the exact feet of each case after selecting the map movement mode.
			m.setPoint(tc.start)
			target := PlayerCombatSnapshot{CharacterID: 1, PosX: tc.end[0], PosY: tc.end[1], PosZ: tc.end[2], Alive: true}
			mods, tick, now := m.mods(), MobCombatTick{}, time.Unix(1, 0)
			for range 1200 {
				if m.AggroTarget == 0 || (len(tick.Moves) > 0 && m.targetDistance(&target) < 1.4) {
					break
				}
				m.stepChase(now, tc.id, &target, m.targetDistance(&target), &mods, &tick)
				now = now.Add(mobNavigationTick)
			}
			if tc.moves {
				if len(tick.Moves) == 0 || m.targetDistance(&target) >= 1.4 {
					t.Fatalf("chase ended at %v after %d moves", m.point(), len(tick.Moves))
				}
			} else if len(tick.Moves) != 0 || m.AggroTarget != target.CharacterID {
				t.Fatalf("target already in XZ range produced %d moves, aggro=%d", len(tick.Moves), m.AggroTarget)
			}
		})
	}
}
