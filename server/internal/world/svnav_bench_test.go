package world

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/mapnav"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// BenchmarkMobChase compares full chases under both navigation modes on the
// installed maps. GX_TEST_MOB_MOVEMENT_DATA names the server data directory.
func BenchmarkMobChase(b *testing.B) {
	data := os.Getenv("GX_TEST_MOB_MOVEMENT_DATA")
	if data == "" {
		b.Skip("requires installed svmap and geometry data")
	}
	b.Setenv("RAPTURE_DATA_DIR", data)
	defer func(v bool) { svmapNavigationEnabled = v }(svmapNavigationEnabled)
	for _, id := range []uint16{1, 3, 9, 47} {
		load, err := loadSvmapFile(id, filepath.Join(data, "maps", fmt.Sprint(id)+".svmap"), &db.Defs{Mobs: map[uint16]*db.MobDef{}})
		if err != nil {
			b.Fatal(err)
		}
		RegisterMapGrid(id, load.mapSize, load.mask)
		nav, err := loadMobNavigation(id)
		if err != nil || nav == nil {
			b.Fatal(err)
		}
		// Spawn and target pairs that both modes accept, 8-40 units apart.
		type pair struct {
			geo, grid MapMob
			target    PlayerCombatSnapshot
		}
		var pairs []pair
		rng := rand.New(rand.NewPCG(uint64(id), 7))
		size := float32(load.mapSize)
		for tries := 0; len(pairs) < 200 && tries < 200000; tries++ {
			x, z := rng.Float32()*size, rng.Float32()*size
			a := float32(rng.Float64() * 6.283)
			r := 8 + rng.Float32()*32
			tx, tz := x+r*float32(sinf(a)), z+r*float32(cosf(a))
			if !IsMapWalkable(id, x, z) || !IsMapWalkable(id, tx, tz) {
				continue
			}
			var p pair
			ok := true
			for i, mode := range []bool{false, true} {
				svmapNavigationEnabled = mode
				m := MapMob{AggroTarget: 1, def: &db.MobDef{ChaseStep: 2, ChaseTimeMS: 500, Size: 1}}
				m.setPoint(mapnav.Point{x, 0, z})
				m.spawnHeightKnown, m.spawnMinY, m.spawnMaxY = true, 400, 400
				if !m.placeOnNavigation(nav, id) {
					ok = false
					break
				}
				if i == 0 {
					p.geo = m
				} else {
					p.grid = m
				}
			}
			if !ok {
				continue
			}
			ty := p.geo.PosY
			if t := nav.Terrain(); t != nil {
				ty = t.Height(tx, tz)
			}
			p.target = PlayerCombatSnapshot{CharacterID: 1, PosX: tx, PosY: ty, PosZ: tz, Alive: true}
			pairs = append(pairs, p)
		}
		for _, mode := range []struct {
			name string
			grid bool
		}{{"geometry", false}, {"svmap", true}} {
			b.Run(fmt.Sprintf("map%d/%s", id, mode.name), func(b *testing.B) {
				svmapNavigationEnabled = mode.grid
				steps, arrived := 0, 0
				for b.Loop() {
					steps, arrived = 0, 0
					for _, p := range pairs {
						m := p.geo
						if mode.grid {
							m = p.grid
						}
						m.navPath, m.svRoute = nil, svRoute{}
						target := p.target
						mods, tick, now := m.mods(), MobCombatTick{}, time.Unix(1, 0)
						for range 400 {
							if m.AggroTarget == 0 || m.targetDistance(&target) < 1.4 {
								break
							}
							m.stepChase(now, id, &target, m.targetDistance(&target), &mods, &tick)
							now = now.Add(mobNavigationTick)
						}
						steps += len(tick.Moves)
						if m.targetDistance(&target) < 1.4 {
							arrived++
						}
					}
				}
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(max(steps, 1)), "ns/step")
				b.ReportMetric(float64(arrived)/float64(len(pairs))*100, "%arrived")
				b.ReportMetric(float64(len(pairs)), "pairs")
			})
		}
	}
}

func sinf(a float32) float64 { return math.Sin(float64(a)) }
func cosf(a float32) float64 { return math.Cos(float64(a)) }
