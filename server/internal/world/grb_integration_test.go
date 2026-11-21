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

func TestGuildRankingBattle(t *testing.T) {
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
	const guildID = 990500
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
	ch := &db.Character{ID: 990501, Name: "GrbTest", Level: 40, MapID: 53, PosX: 89, PosY: 8, PosZ: 115}
	s := &Session{char: ch, log: logger.Tagged("grb"), logger: logger, faction: FactionLight, out: &connWriter{conn: a},
		skillCooldowns: map[uint16]time.Time{}}
	s.player = RegisterPlayer(*ch, nil, false, FactionLight, s.out, logger)
	defer func() { s.player.Unregister() }()
	g := &guild{ID: guildID, Name: "GrbTestGuild", Country: 0, Rank: grbUnranked,
		members: map[uint32]*guildMember{}, applicants: map[uint32]*guildMember{}}
	guildMu.Lock()
	g.members[ch.ID] = &guildMember{CharID: ch.ID, Name: ch.Name, Rank: guildRankMaster, player: s.player}
	guildsByID[guildID] = g
	s.player.guild.guild = g
	guildMu.Unlock()

	sunday := func(h, m int) time.Time { return time.Date(2026, 11, 8, h, m, 0, 0, time.Local) }
	grbNow = func() time.Time { return sunday(20, 30) }
	s.applyTeleport(50, 507.78, 3.08, 98.41, "test")
	if ch.MapID != 53 {
		t.Fatal("entered the battle before it opened")
	}
	now := sunday(21, 30)
	grbNow = func() time.Time { return now }
	TickGuildRanking(sunday(21, 30), logger)
	s.applyTeleport(50, 507.78, 3.08, 98.41, "test")
	zone := ch.MapID
	if !isInstanceZone(zone) || realMap(zone) != 50 {
		t.Fatalf("battle zone %d (map %d)", zone, realMap(zone))
	}
	t.Cleanup(func() { destroyInstance(zone) })
	var mob MapMob
	for _, m := range EntitiesOnMap(zone).Mobs {
		if m.MoneyMin == 0 && m.MoneyMax > 1 {
			mob = m
			break
		}
	}
	if mob.ID == 0 {
		t.Fatal("no battle mob in the instance")
	}
	if hit, ok := DamageMobFrom(zone, mob.ID, mob.CurrentHP, ch.ID); !ok || !hit.Killed {
		t.Fatal("could not kill battle mob")
	}
	if !grbKill(zone, &mob, ch.ID) || g.grbPoints != mob.MoneyMax {
		t.Fatalf("points %d after killing a %d-point mob", g.grbPoints, mob.MoneyMax)
	}
	if grbKill(53, &mob, ch.ID) {
		t.Fatal("points outside the battle map")
	}

	s.applyTeleport(53, 89, 8, 115, "test leave")
	if ch.MapID != 53 || len(PlayersOnMap(zone)) != 0 {
		t.Fatal("battle instance did not become empty")
	}
	TickInstances(now, logger)
	TickInstances(now.Add(mapConfig.Zones[50].Expire+time.Minute), logger)
	s.applyTeleport(50, 507.78, 3.08, 98.41, "test reentry")
	if ch.MapID != zone {
		t.Fatalf("reentered zone %d, want original zone %d", ch.MapID, zone)
	}
	if dead, ok := MobByID(zone, mob.ID); !ok || dead.CurrentHP != 0 {
		t.Fatal("battle mob state was lost during reentry")
	}
	if g.grbPoints != mob.MoneyMax {
		t.Fatalf("reentry changed points to %d", g.grbPoints)
	}

	s.player.Unregister()
	if len(PlayersOnMap(zone)) != 0 {
		t.Fatal("disconnect left the instance occupied")
	}
	TickInstances(now, logger)
	TickInstances(now.Add(mapConfig.Zones[50].Expire+time.Minute), logger)
	// Login returns characters saved in instances to their bind map.
	ch.MapID, ch.PosX, ch.PosY, ch.PosZ = 53, 89, 8, 115
	s.player = RegisterPlayer(*ch, nil, false, FactionLight, s.out, logger)
	guildMu.Lock()
	s.player.guild.guild = g
	g.members[ch.ID].player = s.player
	guildMu.Unlock()
	s.applyTeleport(50, 507.78, 3.08, 98.41, "test reentry after disconnect")
	if ch.MapID != zone {
		t.Fatalf("after reconnect entered zone %d, want original zone %d", ch.MapID, zone)
	}

	now = sunday(22, 1)
	TickGuildRanking(now, logger)
	tp := s.player.TakeTeleport()
	if tp == nil || tp.MapID != 53 {
		t.Fatalf("not sent to the bind point at the close: %+v", tp)
	}
	s.applyTeleport(tp.MapID, tp.X, tp.Y, tp.Z, "test close")
	s.applyTeleport(zone, 507.78, 3.08, 98.41, "test closed reentry")
	if ch.MapID != 53 {
		t.Fatal("reentered the instance after the battle closed")
	}
	now = sunday(22, 2)
	TickGuildRanking(now, logger)
	if len(instanceZones(50)) != 0 {
		t.Fatal("empty instance kept after the close")
	}
	now = sunday(22, 32)
	TickGuildRanking(now, logger)
	want := mob.MoneyMax
	if g.Rank != 1 || g.Points != want {
		t.Fatalf("rank %d points %d, want 1 / %d", g.Rank, g.Points, want)
	}
	var rank int
	pool, _ := db.Pool(db.GameData)
	pool.QueryRow(`SELECT Rank FROM PS_GameData_GuildDetails WHERE GuildID = ?1`, int32(guildID)).Scan(&rank)
	if rank != 1 {
		t.Fatalf("stored rank %d", rank)
	}
}

func itoa(v uint16) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}
