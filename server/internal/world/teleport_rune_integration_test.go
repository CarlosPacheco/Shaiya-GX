//go:build integration

package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
)

func TestClassicTeleportRuneDestinations(t *testing.T) {
	logger, err := logx.New("gx_teleport_rune_test")
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	oldShards, oldGrids, oldConfig, oldRunes := realm.shards, realm.grids, mapConfig, teleportRunes
	oldWeatherStarted := weatherStarted
	realm.shards, realm.grids = map[uint16]*mapShard{}, map[uint16]mapGrid{}
	t.Cleanup(func() {
		realm.shards, realm.grids, mapConfig, teleportRunes = oldShards, oldGrids, oldConfig, oldRunes
		weatherStarted = oldWeatherStarted
	})
	if err := loadMapProperties(logger); err != nil {
		t.Fatal(err)
	}
	if err := LoadSvmaps(logger); err != nil {
		t.Fatal(err)
	}
	if err := loadTeleportRunes(); err != nil {
		t.Fatal(err)
	}
	ch := &db.Character{ID: 4294967294, Level: 30}
	s := &Session{char: ch}
	defer delete(guildOfChar, ch.ID)
	for faction, maps := range [2][4]uint16{{35, 42, 40, 51}, {36, 42, 40, 52}} {
		s.faction = uint8(faction)
		g := &guild{ID: 42, Country: s.faction, Rank: 1, BuyHouse: 1,
			members: map[uint32]*guildMember{ch.ID: {}}}
		guildOfChar[ch.ID] = g
		if _, _, _, reason := s.resolveTeleportRune(4); reason != "Your guild house is not active." {
			t.Fatalf("faction %d: inactive house returned %q", faction, reason)
		}
		g.UseHouse = 1
		for i, want := range maps {
			got, _, guildID, reason := s.resolveTeleportRune(int16(i + 1))
			if reason != "" || got != want || (i == 3 && guildID != g.ID) || (i != 3 && guildID != 0) {
				t.Errorf("faction %d rune %d: got map %d guild %d, %q, want map %d", faction, i+1, got, guildID, reason, want)
			}
		}
		if _, _, _, reason := s.resolveTeleportRune(5); reason == "" {
			t.Fatalf("faction %d: unavailable map accepted", faction)
		}
	}
	for i, typeID := range []uint8{131, 132, 133, 138, 169} {
		def := db.MustDefs().Item(100, typeID)
		if def == nil || def.Special != specialTeleportRune || def.ReqVg != int16(i+1) {
			t.Errorf("incorrect definition for item 100/%d: %+v", typeID, def)
		}
	}
}
