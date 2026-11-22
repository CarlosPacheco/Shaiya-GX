package world

import (
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/gameini"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestSafeZonesImportedCamps(t *testing.T) {
	ini, err := readIni(filepath.Join("..", "..", "data", "game", "SafeZones.INI"))
	if err != nil {
		t.Fatal(err)
	}
	zones, err := parseSafeZones(ini, map[uint16]ZoneConfig{45: {}, 46: {}, 47: {}})
	if err != nil {
		t.Fatal(err)
	}
	for mapID, want := range map[uint16]int{45: 4, 46: 2, 47: 4} {
		if got := len(zones[mapID]); got != want {
			t.Fatalf("map %d has %d combat zones, want %d", mapID, got, want)
		}
		for _, zone := range zones[mapID] {
			var center safePoint
			for _, corner := range zone.corners {
				center.x += corner.x * .25
				center.z += corner.z * .25
				if zone.contains(corner) {
					t.Fatalf("map %d includes a boundary vertex", mapID)
				}
			}
			if got := safeProtectionAt(zones[mapID], center.x, center.z); got != zone.faction {
				t.Fatalf("map %d center %+v protects %d, want %d", mapID, center, got, zone.faction)
			}
		}
	}
	for _, entry := range []string{
		"Map=99\nProtectedFaction=Light\nVertices=(0,0) (0,4) (4,4) (4,0)",
		"Map=45\nProtectedFaction=Neutral\nVertices=(0,0) (0,4) (4,4) (4,0)",
		"Map=45\nProtectedFaction=Both\nVertices=(0,0) (4,4) (0,4) (4,0)",
		"Map=45\nProtectedFaction=Both\nVertices=(NaN,0) (0,4) (4,4) (4,0)",
	} {
		ini, err := gameini.Parse([]byte("[Invalid camp]\n" + entry))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parseSafeZones(ini, map[uint16]ZoneConfig{45: {}}); err == nil {
			t.Fatalf("accepted invalid configuration %q", entry)
		}
	}
}

func TestSafeZonesWorldCombat(t *testing.T) {
	base, _ := lapisSession(t)
	const zoneID = 59971
	RegisterMapGrid(zoneID, 1024, nil)
	previous := safeZones
	safeZones = map[uint16][]safeZone{zoneID: {
		{safeLight, [4]safePoint{{490, 490}, {502, 490}, {502, 510}, {490, 510}}},
		{safeBoth, [4]safePoint{{520, 490}, {520, 510}, {530, 510}, {530, 490}}},
	}}
	t.Cleanup(func() { safeZones = previous })
	newPlayer := func(id uint32, faction uint8, x float32) (*Session, <-chan []byte) {
		s, wire := gmWireSession(t, db.Character{ID: id, MapID: zoneID, PosX: x, PosY: 30, PosZ: 500,
			Level: 40, Mode: growNormal, CurrentHP: 100, CurrentMP: 100, CurrentSP: 100}, false)
		s.faction = faction
		s.items = base.items
		s.player = RegisterPlayer(*s.char, nil, false, faction, s.out, s.logger)
		s.player.SetAlive(true)
		t.Cleanup(s.player.Unregister)
		return s, wire
	}
	light, _ := newPlayer(599711, FactionLight, 501)
	fury, wire := newPlayer(599712, FactionDark, 503)
	exposed, _ := newPlayer(599713, FactionLight, 505)
	expectFailure := func(skill, message uint16) {
		t.Helper()
		frame := waitWorldPacket(t, wire, proto.GXSkillFailure)
		got, err := proto.DecodeSkillFailure(frame)
		if err != nil || got.Skill != skill || got.Message != message {
			t.Fatalf("localized safe zone refusal %+v, %v", got, err)
		}
	}
	if !fury.safeZoneBlocksUser(light.player) || light.safeZoneBlocksUser(fury.player) {
		t.Fatal("Light protection was not asymmetric")
	}
	if !light.userAttackSwing(fury.char.ID) || len(fury.player.TakeUserHits()) != 1 {
		t.Fatal("Light could not attack Fury from inside its camp")
	}
	if fury.userAttackSwing(light.char.ID) {
		t.Fatal("auto attack continued on protected resident")
	}
	expectFailure(0, proto.MsgCannotAttack)
	furySkill := db.CharacterSkill{SkillID: 510, Level: 1, Number: 1}
	spell := &db.SkillDef{SkillID: 510, SkillLevel: 1, TypeEffect: typeEffectAttack,
		TypeDetail: detailAttack, TargetType: 3, ReadyTime: 4, AttackRange: 15, ApplyRange: 10}
	fury.beginSkill(opUserSkillPvP, furySkill, spell, light.char.ID, 0, nil)
	if fury.cast != nil {
		t.Fatal("a protected target accepted a prepared cast")
	}
	expectFailure(furySkill.SkillID, proto.MsgCannotUseSkill)
	fury.beginSkill(opUserSkillPvP, furySkill, spell, exposed.char.ID, 0, nil)
	if fury.cast == nil {
		t.Fatal("a hostile spell outside the zones could not start")
	}
	exposed.char.PosX = 500
	exposed.player.UpdateCharacter(*exposed.char)
	prepared := fury.cast
	fury.cast = nil
	beforeMP := fury.char.CurrentMP
	fury.resolveSkillDefinition(exposed.char.ID, furySkill, spell, opUserSkillPvP, prepared)
	expectFailure(furySkill.SkillID, proto.MsgCannotUseSkill)
	if fury.char.CurrentMP != beforeMP || !fury.skillReady(furySkill.SkillID) {
		t.Fatal("the refused cast spent resources or started reuse")
	}
	if hits := exposed.player.TakeUserHits(); len(hits) != 0 {
		t.Fatal("the protected target received the prepared hit")
	}
	exposed.char.PosX = 505
	exposed.player.UpdateCharacter(*exposed.char)
	fury.areaUserHits(spell, furySkill, 503, 30, 500)
	if len(light.player.TakeUserHits()) != 0 || len(exposed.player.TakeUserHits()) != 1 {
		t.Fatal("area attack did not filter the protected resident")
	}
	light.applyUserHit(pendingUserHit{AttackerID: fury.char.ID, Reflected: true, Damage: [3]uint32{90}})
	if light.char.CurrentHP != 100 {
		t.Fatal("queued damage affected a target inside its sanctuary")
	}
	fury.char.PosX = 525
	fury.player.UpdateCharacter(*fury.char)
	if !fury.safeZoneBlocksUser(exposed.player) || !exposed.safeZoneBlocksUser(fury.player) {
		t.Fatal("the mutual sanctuary did not prevent attacks in both directions")
	}
	if !fury.safeZoneBlocksSkill(spell, opUserSkillPvP, exposed.char.ID) ||
		fury.safeZoneBlocksSkill(&db.SkillDef{TypeEffect: typeEffectHeal}, opUserSkillPvP, fury.char.ID) ||
		fury.safeZoneBlocksSkill(&db.SkillDef{TypeEffect: typeEffectBuff}, opUserSkillPvP, fury.char.ID) {
		t.Fatal("the mutual sanctuary did not distinguish offensive and friendly casts")
	}
}
