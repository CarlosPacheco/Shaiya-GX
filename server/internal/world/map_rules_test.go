package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
)

func TestMapRulesMigrationAndAccess(t *testing.T) {
	old, started := mapConfig, weatherStarted
	t.Cleanup(func() { mapConfig, weatherStarted = old, started })
	log, err := logx.New("gx_map_rules_test")
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	if err := loadMapProperties(log); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint16{25, 33, 34, 37, 38} {
		if _, ok := mapConfig.Zones[id]; !ok {
			t.Errorf("missing canonical dungeon %d", id)
		}
	}
	if mapConfig.Zones[30].WeatherType != proto.WeatherSnow {
		t.Fatal("missing map 30 snow rule")
	}
	s := &Session{}
	for _, v := range []struct {
		mapID, level uint16
		allow        bool
	}{
		{18, 1, true}, {18, 15, true}, {18, 16, false}, {30, 19, false}, {30, 20, true}, {30, 30, true}, {30, 31, false}, {1, 100, true},
	} {
		if s.canEnterMap(v.mapID, v.level) != v.allow {
			t.Fatalf("map %d level %d", v.mapID, v.level)
		}
	}
	s.isAdmin = true
	if !s.canEnterMap(18, 100) || !s.canEnterMap(30, 1) {
		t.Fatal("GM level bypass missing")
	}
	l, lok := rebirthPosition(5, FactionLight)
	f, fok := rebirthPosition(5, FactionDark)
	if !lok || !fok || l != f || l.MapID != 1 || l.X != 236 {
		t.Fatal("legacy common bind was not split correctly")
	}
	c := &db.Character{Race: 3, MapID: 65500, Level: 1}
	id, x, y, z := s.respawnDestination(c)
	if (MapPos{id, x, y, z}) != mapConfig.Start[3] {
		t.Fatal("missing racial fallback")
	}
}

func TestWeatherScheduleRespectsServerStart(t *testing.T) {
	start := time.Unix(1700000000, 0)
	z := ZoneConfig{WeatherType: proto.WeatherRain}
	for _, tc := range []struct {
		minute int
		want   uint8
	}{{0, 0}, {59, 0}, {60, 1}, {69, 1}, {70, 0}, {120, 1}} {
		if got := weatherAt(z, start, start.Add(time.Duration(tc.minute)*time.Minute)); got != tc.want {
			t.Fatalf("minute %d got %d", tc.minute, got)
		}
	}
	z.WeatherFrequency = 15 * time.Minute
	if weatherAt(z, start, start.Add(15*time.Minute)) != proto.WeatherRain ||
		weatherAt(z, start, start.Add(25*time.Minute)) != proto.WeatherClear {
		t.Fatal("custom frequency is not applied")
	}
}

func TestMapCurseSurvivesDispelAndDeathThenLeavesWithMap(t *testing.T) {
	old := mapConfig
	defer func() { mapConfig = old }()
	mapConfig.Zones = map[uint16]ZoneConfig{18: {MapCurses: [3]uint16{71}}}
	s, _ := gmWireSession(t, db.Character{MapID: 18, Level: 5, CurrentHP: 100}, false)
	s.refreshMapCurses(false)
	if len(s.buffs) != 1 || !s.buffs[0].MapCurse || !s.buffs[0].EndsAt.IsZero() {
		t.Fatal("missing permanent curse")
	}
	s.removeEffects(func(*ActiveBuff) bool { return true }, "dispel")
	s.processBuffs()
	s.clearBuffsOnDeath()
	if len(s.buffs) != 1 || !s.buffs[0].MapCurse {
		t.Fatal("curse removed inside the map")
	}
	s.char.MapID = 1
	s.refreshMapCurses(true)
	if len(s.buffs) != 0 {
		t.Fatal("map curse remained after leaving")
	}
}

func TestMapLevelDeniesPortalsAndRecordedPoints(t *testing.T) {
	old := mapConfig
	defer func() { mapConfig = old }()
	mapConfig.Zones = map[uint16]ZoneConfig{1: {}, 18: {MinLevel: 1, MaxLevel: 15}}
	const source = 64684
	sh := shard(source)
	sh.mu.Lock()
	portals := sh.entities.Portals
	sh.entities.Portals = []MapPortal{{Index: 0, TargetMapID: 18, MaxLevel: 100}}
	sh.mu.Unlock()
	defer func() { sh.mu.Lock(); sh.entities.Portals = portals; sh.mu.Unlock() }()
	s, _ := gmWireSession(t, db.Character{MapID: source, Level: 16, CurrentHP: 100}, false)
	index := uint8(0)
	if _, ok := s.usablePortal(0, 0, 0, &index); ok || s.teleportStoneMapAllowed(18) {
		t.Fatal("portal or stone bypassed map rule")
	}
	s.applyTeleport(18, 0, 0, 0, "test")
	if s.char.MapID != source {
		t.Fatal("central teleport bypassed map rule")
	}
	s.isAdmin = true
	if _, ok := s.usablePortal(0, 0, 0, &index); !ok || !s.teleportStoneMapAllowed(18) {
		t.Fatal("GM did not bypass the level check")
	}
}

func TestClearWeatherPreviewRestoresScheduledWeather(t *testing.T) {
	old, started := mapConfig, weatherStarted
	t.Cleanup(func() { mapConfig, weatherStarted = old, started })
	mapConfig.Zones = map[uint16]ZoneConfig{18: {WeatherType: proto.WeatherRain}}
	weatherStarted = time.Now().Add(-61 * time.Minute)
	s, _ := gmWireSession(t, db.Character{MapID: 18}, true)
	if err := s.gmWeather([]string{"2"}); err != nil || s.lastWeather != proto.WeatherSnow {
		t.Fatalf("snow preview failed: %v", err)
	}
	if err := s.gmWeather([]string{"0"}); err != nil || s.lastWeather != proto.WeatherRain || !s.weatherOverrideUntil.IsZero() {
		t.Fatalf("scheduled rain was not restored: %v", err)
	}
}

func TestMap27OpenPortalsAllowBothFactions(t *testing.T) {
	for _, mapID := range []uint16{27, 26} {
		sh := shard(mapID)
		sh.mu.Lock()
		original := sh.entities.Portals
		sh.entities.Portals = []MapPortal{
			{Index: 0, TargetMapID: mapID, MaxLevel: 80, FactionOrPortalID: 2},
			{Index: 1, TargetMapID: mapID, MaxLevel: 80, FactionOrPortalID: 3},
			{Index: 2, TargetMapID: mapID, MinLevel: 80, MaxLevel: 80, FactionOrPortalID: 2},
		}
		sh.mu.Unlock()
		t.Cleanup(func() { sh.mu.Lock(); sh.entities.Portals = original; sh.mu.Unlock() })
	}
	s, _ := gmWireSession(t, db.Character{MapID: 27, Level: 60, CurrentHP: 100}, false)
	for _, faction := range []uint8{FactionLight, FactionDark} {
		s.faction = faction
		for index := range uint8(3) {
			if _, ok := s.usablePortal(0, 0, 0, &index); ok != (index == 0) {
				t.Errorf("map 27 portal %d: admission=%t for faction %d", index, ok, faction)
			}
		}
	}
	s.char.MapID, s.faction = 26, FactionLight
	index := uint8(0)
	if _, ok := s.usablePortal(0, 0, 0, &index); ok {
		t.Fatal("map 26 lost its faction restriction")
	}
}
