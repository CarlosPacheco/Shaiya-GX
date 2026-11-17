//go:build integration

package world

import (
	"testing"

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
			// Project limits each coordinate to .05, not the Euclidean
			// distance. A valid diagonal offset can exceed .06 (up to sqrt(3)*.05).
			p, ok := nav.Project(m.point(), .05, .05)
			if !ok {
				t.Fatalf("floating actor %d: %v != %v", m.MobID, m.point(), p)
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
}
