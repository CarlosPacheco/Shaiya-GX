//go:build integration

package world

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
)

func TestObeliskOriginalWorldBoot(t *testing.T) {
	oldDefs := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(oldDefs) })
	if _, err := db.ReloadDefsForTest(); err != nil {
		t.Fatal(err)
	}
	logger, err := logx.New("gx_obelisk_boot_test")
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	oldShards, oldGrids := realm.shards, realm.grids
	realm.shards = map[uint16]*mapShard{}
	realm.grids = map[uint16]mapGrid{}
	oldTiers, oldWorld := kcTiers, kcWorld
	oldLoaded, oldConfig, oldMaps, oldBosses, oldPortals, oldScore, oldReset := relics.loaded, relics.config, relics.byMap, relics.bosses, relics.portals, relics.score, relics.resetPending
	relics.loaded = false
	relics.byMap = map[uint16][]*relic{}
	relics.bosses = map[uint16][]*linkedBoss{}
	relics.portals = nil
	t.Cleanup(func() {
		realm.shards, realm.grids = oldShards, oldGrids
		kcTiers, kcWorld = oldTiers, oldWorld
		relics.loaded, relics.config, relics.byMap, relics.bosses, relics.portals, relics.score, relics.resetPending = oldLoaded, oldConfig, oldMaps, oldBosses, oldPortals, oldScore, oldReset
	})
	if err := loadKillStatus(logger); err != nil {
		t.Fatal(err)
	}
	if err := LoadSvmaps(logger); err != nil {
		t.Fatal(err)
	}
	if err := LoadRelics(logger); err != nil {
		t.Fatal(err)
	}
	total, guards := 0, 0
	for mapID, list := range relics.byMap {
		s := shard(mapID)
		for _, r := range list {
			total++
			if !r.standing || r.principal == nil || s.entities.Mobs[r.principal.slot].CurrentHP == 0 {
				t.Fatalf("missing original relic %#x", r.key)
			}
			for _, g := range r.guards {
				for _, a := range g.actors {
					if s.entities.Mobs[a.slot].CurrentHP > 0 {
						guards++
					}
				}
			}
		}
	}
	if total != 9 {
		t.Fatalf("boot loaded %d relics", total)
	}
	mobile, stationary := 0, 0
	for mapID, sh := range realm.shards {
		for i := range sh.entities.Mobs {
			m := &sh.entities.Mobs[i]
			if m.event == nil || m.CurrentHP == 0 {
				continue
			}
			if m.stationary() {
				stationary++
				if m.Area != nil {
					t.Fatal("stationary event actor can wander", m.MobID)
				}
				continue
			}
			mobile++
			if m.Area == nil || m.nav == nil {
				t.Fatalf("mobile event actor %d on %d lacks ordinary navigation", m.MobID, mapID)
			}
			nav, err := loadMobNavigation(mapID)
			if err != nil || nav == nil {
				t.Fatal(err)
			}
			// Verify physical support at the spawn point. A second navmesh
			// projection can select a different polygon at a shared edge.
			geometry, _ := nav.PlayerGeometry()
			if _, supported := geometry.Support(m.PosX, m.PosY, m.PosZ, .05); !supported {
				t.Fatalf("floating actor %d on %d at %v", m.MobID, mapID, m.point())
			}
		}
	}
	if mobile == 0 || stationary == 0 {
		t.Fatal("original populations missing", mobile, stationary)
	}
	t.Logf("Grounded ordinary mobile event actors=%d; stationary=%d", mobile, stationary)
	keeps, err := db.ReadKeeps()
	if err != nil || len(keeps) < 9 {
		t.Fatalf("persistent keeps=%d error=%v", len(keeps), err)
	}
	t.Logf("Original world boot: %d standing relics, %d active guards, %d boss-map populations, %d persistent keeps", total, guards, len(relics.bosses), len(keeps))

	t.Run("Fantasma treasure", func(t *testing.T) {
		const mapID, dragonID, chestID = 17, 902, 1971
		var boss *linkedBoss
		for _, candidate := range relics.bosses[mapID] {
			if candidate.mobs[0] == dragonID {
				boss = candidate
				break
			}
		}
		if boss == nil {
			t.Fatal("missing Fantasma Dragon event")
		}
		s := shard(mapID)
		dragon := s.entities.Mobs[boss.principal.slot]
		hit, ok := DamageMob(mapID, dragon.ID, dragon.CurrentHP)
		if !ok || !hit.Killed {
			t.Fatal("could not kill Fantasma Dragon")
		}
		eventMobDestroyed(nil, mapID, &hit.Mob, logger)
		chests := 0
		for _, g := range boss.guards {
			for _, a := range g.actors {
				m := &s.entities.Mobs[a.slot]
				if m.MobID != chestID || m.CurrentHP == 0 {
					continue
				}
				chests++
				for _, fresh := range []bool{false, true} {
					spawn, err := proto.DecodeMobEnter(buildMobEnter(m, fresh)[2:])
					if err != nil || !spawn.HeightKnown || spawn.Y != m.PosY || spawn.New != fresh {
						t.Fatalf("Fantasma chest at %v: invalid spawn %+v (fresh=%t): %v", m.point(), spawn, fresh, err)
					}
				}
				move, err := proto.DecodeMobMove(buildMobMove(m, 0)[2:])
				if err != nil || !move.HeightKnown || move.Y != m.PosY {
					t.Fatalf("Fantasma chest at %v: invalid position update %+v: %v", m.point(), move, err)
				}
				if math.Abs(float64(m.PosY)-6.490642) > .01 {
					t.Fatalf("Fantasma chest off the arena floor: %v", m.point())
				}
			}
		}
		if chests != 11 {
			t.Fatalf("Fantasma Dragon spawned %d chests, want 11", chests)
		}
	})
}
