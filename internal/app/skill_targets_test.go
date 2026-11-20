package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestTargetedSkillsRejectMissingStaleDeadAndNPC(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	k := proto.LearnedSkill{ID: 30, Level: 1, Number: 7}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 30, Level: 1}: {Target: 1, Effect: 1}}
	for _, tc := range []struct {
		name   string
		target uint32
		entity *game.Entity
	}{
		{"none", 0, nil}, {"stale", 99, nil}, {"npc", 99, &game.Entity{Kind: game.KindNpc}},
		{"dead", 99, &game.Entity{Kind: game.KindMob, Dead: true}},
		{"zero health", 99, &game.Entity{Kind: game.KindMob, HealthKnown: true}},
		{"self", 1, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a.state.Target = tc.target
			delete(a.state.Entities, 99)
			if tc.entity != nil {
				a.state.Entities[99] = tc.entity
			}
			if p := a.learnedSkillPacket(k); p != nil {
				t.Fatalf("invalid target produced request %x", p)
			}
		})
	}
	a.state.Target = 99
	a.state.Entities[99] = &game.Entity{Kind: game.KindMob}
	if a.learnedSkillPacket(k) == nil {
		t.Fatal("living mob rejected")
	}
	for _, target := range []uint8{2, 4, 5, 6} {
		a.skillDefs[sdata.SkillKey{ID: 30, Level: 1}] = sdata.Skill{Target: target, Effect: 2}
		a.state.Target = 0
		if a.learnedSkillPacket(k) == nil {
			t.Fatalf("caster/area spell %d rejected", target)
		}
	}
}

func TestNPCSkillNamespaceNeverFallsBackToPlayerIDs(t *testing.T) {
	a := inputApp()
	key := sdata.SkillKey{ID: 31, Level: 1}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{key: {Name: "Player", Icon: 1001, VisualCode: 12}}
	a.npcSkillDefs = map[sdata.SkillKey]sdata.Skill{key: {Name: "Monster", Icon: 2017, VisualCode: 84}}
	for _, tc := range []struct {
		level        uint8
		name         string
		icon, visual uint16
	}{{1, "Player", 1001, 12}, {100, "Monster", 2017, 84}} {
		d, ok := a.skillDefinition(31, tc.level)
		if !ok || d.Name != tc.name || d.Icon != tc.icon || d.VisualCode != tc.visual {
			t.Fatalf("wrong namespace: %+v", d)
		}
	}
	delete(a.npcSkillDefs, key)
	if _, ok := a.skillDefinition(31, 100); ok {
		t.Fatal("NPC ID incorrectly resolved to player definition")
	}
}

func TestSkillFallbackSelfAndNearestLivingEnemy(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 42
	k := proto.LearnedSkill{ID: 30, Level: 1, Number: 7}
	key := sdata.SkillKey{ID: 30, Level: 1}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{key: {Target: 1, Effect: 4}}
	if p := a.learnedSkillPacket(k); len(p) != 7 || proto.Op(p) != proto.SkillUseNoTarget || proto.NewReader(p).U8() != 7 {
		t.Fatalf("self buff request %x", p)
	}
	if got := a.resolveSkillTarget(a.skillDefs[key]); got != 42 {
		t.Fatal("friendly spell did not default to self", got)
	}
	a.skillDefs[key] = sdata.Skill{Target: 1, Effect: 1}
	a.wv.pos[0], a.wv.pos[2] = 10, 10
	a.state.Entities = map[uint32]*game.Entity{
		1: {ID: 1, Kind: game.KindNpc, X: 10, Z: 10}, 2: {ID: 2, Kind: game.KindMob, Dead: true, X: 10, Z: 10},
		3: {ID: 3, Kind: game.KindUser, X: 10, Z: 10}, 4: {ID: 4, Kind: game.KindMob, X: 20, Z: 10},
		5: {ID: 5, Kind: game.KindMob, X: 12, Z: 10}, 6: {ID: 6, Kind: game.KindMob, X: 12, Z: 10},
	}
	p := a.learnedSkillPacket(k)
	if len(p) != 7 || a.state.Target != 5 {
		t.Fatalf("nearest enemy %x target %d", p, a.state.Target)
	}
	a.state.Target = 4
	if got := a.resolveSkillTarget(a.skillDefs[key]); got != 4 {
		t.Fatal("explicit target replaced", got)
	}
	a.state.Target = 0
	if got := a.resolveSkillTarget(sdata.Skill{Target: 6, Effect: 1}); got != 0 || a.state.Target != 0 {
		t.Fatal("caster area auto-selected a victim")
	}
}

func TestFriendlySpellOnEnemyUsesSelfWithoutLosingSelection(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 42
	a.state.Faction.Faction = 0
	a.state.Entities[9] = &game.Entity{ID: 9, Kind: game.KindMob}
	a.state.Entities[10] = &game.Entity{ID: 10, Kind: game.KindUser, Shape: &proto.UserShape{Faction: 1}}
	a.state.Entities[11] = &game.Entity{ID: 11, Kind: game.KindUser, Shape: &proto.UserShape{Faction: 0}}
	for _, effect := range []uint8{2, 4} {
		for _, id := range []uint32{9, 10, 11} {
			a.state.Target = id
			want := uint32(42)
			if id == 11 {
				want = 11
			}
			if got := a.resolveSkillTarget(sdata.Skill{Target: 3, Effect: effect}); got != want || a.state.Target != id {
				t.Fatal("wrong recipient or combat selection lost", effect, id, got, a.state.Target)
			}
		}
	}
	a.state.Target = 9
	if got := a.resolveSkillTarget(sdata.Skill{Target: 3, Effect: 3}); got != 9 {
		t.Fatal("debuff redirected", got)
	}
}

func TestRaidSkillCastKeepsCombatSelection(t *testing.T) {
	a := inputApp()
	a.state.SelfID, a.state.Target = 42, 99
	a.state.Entities[99] = &game.Entity{Kind: game.KindMob}
	k := proto.LearnedSkill{ID: 124, Level: 1, Number: 7}
	key := sdata.SkillKey{ID: k.ID, Level: k.Level}
	a.skillDefs = make(map[sdata.SkillKey]sdata.Skill)
	for _, d := range []sdata.Skill{
		{Effect: 2, Detail: 41}, {Effect: 4, Detail: 81}, {Effect: 4, Detail: 82}, {Effect: 1, Detail: 21}, {Effect: 5, Detail: 101},
	} {
		d.Target = proto.SkillTargetRaid
		a.skillDefs[key] = d
		p := a.learnedSkillPacket(k)
		if d.Effect == 2 || d.Effect == 4 {
			r := proto.NewReader(p)
			if proto.Op(p) != proto.SkillUseNoTarget || r.U8() != k.Number || r.U32() != a.state.SelfID || r.Err() != nil {
				t.Fatalf("raid skill %d sent wrong cast request: %x", d.Detail, p)
			}
		} else if p != nil {
			t.Fatalf("effect %d sent an unsupported raid cast: %x", d.Effect, p)
		}
		if a.state.Target != 99 {
			t.Fatal("raid cast changed the selected enemy")
		}
	}
}

func TestSoulBurnTargetsOnlyEnemyPlayerCorpses(t *testing.T) {
	a := inputApp()
	a.state.SelfID, a.state.Target = 42, 99
	a.state.Faction.Faction = 0
	k := proto.LearnedSkill{ID: 125, Level: 1, Number: 7}
	key := sdata.SkillKey{ID: k.ID, Level: k.Level}
	for _, effect := range []uint8{3, 6} {
		a.skillDefs = map[sdata.SkillKey]sdata.Skill{key: {Effect: effect, Detail: 122, Target: 3}}
		e := &game.Entity{ID: 99, Kind: game.KindUser, Shape: &proto.UserShape{Faction: 1}, Dead: true}
		a.state.Entities[99] = e
		p := a.learnedSkillPacket(k)
		r := proto.NewReader(p)
		if proto.Op(p) != proto.SkillUseNoTarget || r.U8() != k.Number || r.U32() != 99 || r.Err() != nil {
			t.Fatalf("Soul Burn sent the wrong corpse request: %x", p)
		}
		for _, tc := range []struct {
			name   string
			change func()
		}{
			{"living", func() { e.Dead = false }},
			{"ally", func() { e.Shape.Faction = 0 }},
			{"Ultimate", func() { e.Shape.Mode = 3 }},
			{"mob", func() { e.Kind = game.KindMob }},
			{"self", func() { a.state.Target = a.state.SelfID }},
			{"unselected", func() { a.state.Target = 0 }},
		} {
			e.Kind, e.Dead, e.Shape.Faction, e.Shape.Mode, a.state.Target = game.KindUser, true, 1, 0, 99
			tc.change()
			if p := a.learnedSkillPacket(k); p != nil {
				t.Fatalf("Soul Burn accepted %s: %x", tc.name, p)
			}
		}
		a.state.Target = 99
	}
}

func TestFriendlyMobRefusedBeforeAttackPredictionAndSkillRequest(t *testing.T) {
	a := inputApp()
	a.msg = text.SysMsg{317: "Wrong target"}
	a.state.SelfID, a.state.Target = 1, 9
	a.state.Entities[9] = &game.Entity{ID: 9, Kind: game.KindMob, MobCountryKnown: true, MobCountry: 0, X: 100}
	a.state.Faction.Faction = 0
	a.wv.approachTarget = 9
	if !a.attackTarget() || a.wv.approachTarget != 0 {
		t.Fatal("friendly attack began approaching")
	}
	if len(a.state.Chat) != 1 || a.state.Chat[0].MessageID != 317 || a.state.Chat[0].Style != 17 {
		t.Fatal(a.state.Chat)
	}
	k := proto.LearnedSkill{ID: 30, Level: 1, Number: 7}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 30, Level: 1}: {Target: 1, Effect: 1}}
	if p := a.learnedSkillPacket(k); p != nil {
		t.Fatalf("friendly skill sent %x", p)
	}
	if len(a.state.Chat) != 2 || a.state.Chat[1].MessageID != 317 {
		t.Fatal(a.state.Chat)
	}
	for _, country := range []uint8{1, 2} {
		a.state.Entities[9].MobCountry = country
		if !a.validSkillTarget(a.skillDefs[sdata.SkillKey{ID: 30, Level: 1}], 9) {
			t.Fatal("enemy/neutral mob refused", country)
		}
	}
}

// With yourself selected, an offensive skill goes for the nearest enemy.
func TestOffensiveSkillWithSelfSelectedPicksNearestEnemy(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 42
	a.wv.pos[0], a.wv.pos[2] = 10, 10
	a.state.Entities = map[uint32]*game.Entity{
		4: {ID: 4, Kind: game.KindMob, X: 20, Z: 10}, 5: {ID: 5, Kind: game.KindMob, X: 12, Z: 10},
	}
	a.state.Target = 42
	if got := a.resolveSkillTarget(sdata.Skill{Target: 1, Effect: 1}); got != 5 || a.state.Target != 5 {
		t.Fatalf("self selected: target %d, selected %d", got, a.state.Target)
	}
	// Buffs and heals still act on yourself.
	a.state.Target = 42
	if got := a.resolveSkillTarget(sdata.Skill{Target: 1, Effect: 4}); got != 42 {
		t.Fatalf("a buff left yourself: %d", got)
	}
}

func TestBasicAttackWithSelfSelectedTargetsNearestEnemy(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 42
	a.wv.pos[0], a.wv.pos[2] = 10, 10
	a.state.Entities = map[uint32]*game.Entity{
		4: {ID: 4, Kind: game.KindMob, X: 30, Z: 10}, 5: {ID: 5, Kind: game.KindMob, X: 12, Z: 10}, 6: {ID: 6, Kind: game.KindMob, Dead: true, X: 10, Z: 10},
	}
	if got := a.nearestHostileMob(); got != 5 {
		t.Fatalf("nearest enemy %d", got)
	}
}
