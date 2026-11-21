//go:build integration

package world

import (
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func TestHouseShopBuyInsideInventoryOperation(t *testing.T) {
	if _, err := os.Stat("data/game"); err != nil {
		t.Chdir("../..")
	}
	config.LoadRuntimeEnv()
	if err := db.Open(db.UserData, db.GameData, db.GameDefs); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ReloadDefsForTest(); err != nil {
		t.Fatal(err)
	}
	logger, _ := logx.New("gx_world_test")
	if err := LoadGameConfig(logger); err != nil {
		t.Fatal(err)
	}
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	go io.Copy(io.Discard, b)
	ch := &db.Character{ID: 990701, Name: "HouseGM", Level: 40, MapID: 53, PosX: 89, PosY: 8, PosZ: 115}
	s := &Session{char: ch, log: logger.Tagged("h"), logger: logger, faction: FactionLight, out: &connWriter{conn: a},
		skillCooldowns: map[uint16]time.Time{}, itemCooldowns: map[uint8]time.Time{}, isAdmin: true}
	s.player = RegisterPlayer(*ch, nil, false, FactionLight, s.out, logger)
	defer s.player.Unregister()
	g := &guild{ID: 990700, Name: "HouseGuild", Master: ch.Name, Country: 0, Rank: 1, gmHouse: true,
		members: map[uint32]*guildMember{}, applicants: map[uint32]*guildMember{}}
	g.forceGMHouse()
	guildMu.Lock()
	g.members[ch.ID] = &guildMember{CharID: ch.ID, Name: ch.Name, Rank: guildRankMaster, player: s.player}
	guildsByID[990700] = g
	s.player.guild.guild = g
	guildMu.Unlock()
	defer func() { guildMu.Lock(); delete(guildsByID, 990700); guildMu.Unlock() }()
	zone, err := createInstance(51, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer destroyInstance(zone)
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.applyTeleport(zone, 60, 5, 60, "house")
	var shop MapNpc
	for _, n := range EntitiesOnMap(zone).NPCs {
		if n.Type == npcTypeShop {
			if def := findNpcDef(n.Type, n.TypeID); def != nil && def.ShopType >= 1 && def.ShopType <= 4 && len(def.ShopItems) > 0 {
				shop = n
				break
			}
		}
	}
	if shop.ID == 0 {
		t.Skip("no house shop NPC")
	}
	w := packet.New(packet.ShopBuy)
	w.U32(shop.ID)
	w.U8(0)
	w.Count(1)
	s.handleShopBuy(w.MustFinish()[2:])
	if !guildMu.TryLock() {
		t.Fatal("guildMu left locked")
	}
	guildMu.Unlock()
}
