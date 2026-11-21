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
)

func TestGMGuildRankingParticipation(t *testing.T) {
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
	logger, err := logx.New("gx_world_test")
	if err != nil {
		t.Fatal(err)
	}
	if err := LoadGameConfig(logger); err != nil {
		t.Fatal(err)
	}
	for _, m := range []uint16{53, 54} {
		if !hasGrid(m) {
			if l, err := loadSvmapFile(m, mapDataDir()+"/"+itoa(m)+".svmap", db.MustDefs()); err == nil {
				RegisterMapGrid(m, l.mapSize, l.mask)
			}
		}
	}
	const guildID = 990600
	defer func() {
		pool, _ := db.Pool(db.GameData)
		pool.Exec(`DELETE FROM PS_GameData_GuildDetails WHERE GuildID = ?1`, int32(guildID))
		guildMu.Lock()
		delete(guildsByID, guildID)
		guildMu.Unlock()
		grbNow = time.Now
	}()
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	go io.Copy(io.Discard, b)
	ch := &db.Character{ID: 990601, Name: "GrbGM", Level: 40, MapID: 53, PosX: 89, PosY: 8, PosZ: 115}
	s := &Session{char: ch, log: logger.Tagged("grbgm"), logger: logger, faction: FactionLight, out: &connWriter{conn: a},
		skillCooldowns: map[uint16]time.Time{}, isAdmin: true}
	s.player = RegisterPlayer(*ch, nil, false, FactionLight, s.out, logger)
	defer func() { s.player.Unregister() }()
	g := &guild{ID: guildID, Name: "GrbGMGuild", Master: ch.Name, Country: 0, Rank: grbUnranked,
		members: map[uint32]*guildMember{}, applicants: map[uint32]*guildMember{}}
	guildMu.Lock()
	g.members[ch.ID] = &guildMember{CharID: ch.ID, Name: ch.Name, Rank: guildRankMaster, player: s.player}
	guildsByID[guildID] = g
	s.player.guild.guild = g
	guildMu.Unlock()

	closed := time.Date(2026, 11, 8, 20, 30, 0, 0, time.Local) // before the Sunday slot
	grbNow = func() time.Time { return closed }
	s.applyTeleport(50, 507.78, 3.08, 98.41, "gm entry")
	zone := ch.MapID
	if !isInstanceZone(zone) || realMap(zone) != 50 {
		t.Fatalf("GM could not enter the closed battle: zone %d", zone)
	}
	t.Cleanup(func() { destroyInstance(zone) })
	TickGuildRanking(closed, logger) // the closed schedule must not evict the GM
	if tp := s.player.TakeTeleport(); tp != nil {
		t.Fatalf("closed battle moved the GM out: %+v gm=%t owner=%d guild=%d", tp, g.gmGRB, instanceOwnerOf(zone), g.ID)
	}
	var mob MapMob
	for _, m := range EntitiesOnMap(zone).Mobs {
		if m.MoneyMin == 0 && m.MoneyMax > 1 {
			mob = m
			break
		}
	}
	if mob.ID == 0 {
		t.Fatal("no battle mob")
	}
	if !grbKill(zone, &mob, ch.ID) || g.grbPoints != mob.MoneyMax {
		t.Fatalf("closed-battle points %d", g.grbPoints)
	}
	s.applyTeleport(53, 89, 8, 115, "gm leave")
	if g.Points != mob.MoneyMax || g.grbPoints != 0 || g.gmGRB {
		t.Fatalf("points not credited on leaving: points=%d pending=%d gm=%t", g.Points, g.grbPoints, g.gmGRB)
	}
}
