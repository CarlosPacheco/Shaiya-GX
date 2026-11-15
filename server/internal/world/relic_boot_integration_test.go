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
	oldShards, oldGrids := shards, grids
	shards = map[uint16]*mapShard{}
	grids = map[uint16]mapGrid{}
	oldTiers, oldWorld := kcTiers, kcWorld
	oldLoaded, oldConfig, oldMaps, oldBosses, oldPortals, oldScore, oldReset := relics.loaded, relics.config, relics.byMap, relics.bosses, relics.portals, relics.score, relics.resetPending
	relics.loaded = false
	relics.byMap = map[uint16][]*relic{}
	relics.bosses = map[uint16][]*linkedBoss{}
	relics.portals = nil
	t.Cleanup(func() {
		shards, grids = oldShards, oldGrids
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
	keeps, err := db.ReadKeeps()
	if err != nil || len(keeps) < 9 {
		t.Fatalf("persistent keeps=%d error=%v", len(keeps), err)
	}
	t.Logf("Original world boot: %d standing relics, %d active guards, %d boss-map populations, %d persistent keeps", total, guards, len(relics.bosses), len(keeps))
}
