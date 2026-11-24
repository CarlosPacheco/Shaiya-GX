package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
	"time"
)

func TestAbsolutePoolsNeverRegenerateEquipmentBonus(t *testing.T) {
	ch := &db.Character{Level: 1, CurrentHP: 2700, CurrentMP: 2300, CurrentSP: 2400}
	e := EquipmentStats{MaxHP: 3000, MaxMP: 3000, MaxSP: 3000, Rec: 20, Wis: 20, Dex: 20}
	for _, hp := range []uint32{2700, 2000, 1000, 1, 0} {
		ch.CurrentHP, ch.CurrentMP, ch.CurrentSP = hp, hp, hp
		for range 3 {
			st := StatsFor(ch, e)
			if st.CurrentHP != hp || st.CurrentMP != hp || st.CurrentSP != hp {
				t.Fatalf("bonus refilled pools at %d: %+v", hp, st)
			}
		}
	}
	s, _ := forgeRuleSession(t)
	s.char.CurrentHP, s.char.CurrentMP, s.char.CurrentSP = 20, 20, 20
	s.spendDelta(20, 20)
	if s.stats().CurrentMP != 0 || s.stats().CurrentSP != 0 {
		t.Fatal("spending did not reach zero")
	}
	hp, mp, sp := s.healDelta(10, 10, 10)
	if hp != 10 || mp != 10 || sp != 10 || s.char.CurrentHP != 30 {
		t.Fatal("healing was not an absolute delta")
	}
}

func TestRecoveryPacketIsAbsoluteAtClient(t *testing.T) {
	state := game.New()
	state.SelfID = 42
	state.Pools = proto.Pools{HP: 1000, MP: 2000, SP: 3000}
	// A standing tick restores four, but the native 0505 carries totals.
	p := buildCharacterReviveHitpoints(42, 1004, 2004, 3004)[2:]
	if _, err := state.Apply(p); err != nil {
		t.Fatal(err)
	}
	if state.Pools.HP != 1004 || state.Pools.MP != 2004 || state.Pools.SP != 3004 {
		t.Fatal(state.Pools)
	}
}

func TestBuffConsumableAnnouncesSkillAndStopsCombo(t *testing.T) {
	s, add := forgeRuleSession(t)
	defs, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	s.itemCooldowns = map[uint8]time.Time{}
	for _, id := range []uint8{75, 77, 174} {
		if d := defs.Item(100, id); d != nil && itemBuffSkill(d) != nil {
			t.Fatalf("retired consumable %d still grants a buff", id)
		}
	}
	for _, family := range []struct {
		itemType, first, count uint8
		skill                  uint16
		level                  uint8
	}{
		{44, 99, 3, 276, 1}, {44, 102, 3, 277, 1}, {44, 105, 3, 278, 1}, {44, 108, 3, 279, 1},
		{44, 223, 1, 270, 2}, {44, 224, 1, 269, 2}, {44, 225, 1, 268, 2},
		{100, 92, 3, 257, 1}, {100, 95, 3, 256, 1}, {100, 98, 3, 258, 1},
		{100, 101, 3, 259, 1}, {100, 104, 3, 260, 1},
		{100, 145, 3, 280, 1}, {100, 148, 3, 281, 1}, {100, 151, 3, 282, 1},
		{100, 154, 3, 283, 1}, {100, 157, 3, 284, 1}, {100, 160, 3, 285, 1},
	} {
		for offset := uint8(0); offset < family.count; offset++ {
			item := defs.Item(family.itemType, family.first+offset)
			if item == nil {
				t.Fatalf("missing restored item %d/%d", family.itemType, family.first+offset)
			}
			if skill := itemBuffSkill(item); skill == nil || skill.SkillID != family.skill || skill.SkillLevel != family.level+offset {
				t.Fatalf("invalid restored buff on item %d/%d: %+v", item.Type, item.TypeID, skill)
			}
		}
	}
	for _, id := range []uint8{44, 52, 53, 55, 56, 57, 67, 81, 87, 110, 112, 127, 129, 130, 163, 164, 165, 173} {
		if defs.Item(100, id) != nil {
			t.Fatalf("retired item 100/%d still in catalog", id)
		}
	}
	if box := defs.Item(100, 79); box == nil || box.Special != 0 || box.ReqVg != 0 || box.Range != 0 || box.AttackTime != 0 {
		t.Fatalf("Hard Item Pocket is not inert: %+v", box)
	}
	d := defs.Item(100, 54) // Special 35 also follows UseItemSkill, not just Special 0.
	if itemBuffSkill(d) == nil {
		t.Fatal("warehouse rune skill missing")
	}
	add(db.Item{Bag: 1, Slot: 0, Type: d.Type, TypeID: d.TypeID, Count: 2})
	wire, packets := gmWireSession(t, *s.char, false)
	wire.items = s.items
	wire.itemCooldowns = s.itemCooldowns
	wire.handleUseItem(proto.UseItemPacket(1, 0))
	if wire.items[0].Count != 1 || !wire.attackMoveUntil.After(time.Now()) {
		t.Fatal("item not consumed or cast did not reserve action")
	}
	deadline := time.After(time.Second)
	for {
		select {
		case p := <-packets:
			if proto.Op(p) != proto.SkillUseNoTarget {
				continue
			}
			result, err := proto.DecodeSkillResult(p)
			if err != nil {
				t.Fatal(err)
			}
			if result.ID != uint16(d.Range) {
				t.Fatal(result)
			}
			return
		case <-deadline:
			t.Fatal("buff applied without native spell presentation")
		}
	}
}

func TestExpWheelOriginalRequirements(t *testing.T) {
	s, _ := forgeRuleSession(t)
	defs := db.MustDefs()
	defs.Exp = map[uint16]uint32{1: 1000}
	defs.LevelCap = 2
	s.char.MapID = 18
	s.char.Exp = 499
	if s.canUseExpReset() {
		t.Fatal("wheel below half level")
	}
	s.char.Exp = 500
	if !s.canUseExpReset() {
		t.Fatal("valid arena wheel denied")
	}
	s.char.MapID = 0
	if s.canUseExpReset() {
		t.Fatal("wheel outside arena")
	}
}

func TestMobDamageCrossesEquipmentBonusAndPersistsZero(t *testing.T) {
	s, _ := forgeRuleSession(t)
	d := db.MustDefs().Item(16, 1)
	d.ConstHP = 2000
	d.ConstMP = 2000
	d.ConstSP = 2000
	s.items = []db.Item{{Bag: 0, Slot: 0, Type: 16, TypeID: 1, Count: 1}}
	s.refillPools()
	full := s.stats().MaxHP
	if full <= 2000 {
		t.Fatal("missing equipment bonus")
	}
	normalize(s.char)
	if s.char.CurrentHP != full {
		t.Fatal("reconnect clamped to base pool")
	}
	state := game.New()
	state.SelfID = s.char.ID
	for _, damage := range []uint32{full - 1500, 1000, 499, 1} {
		st := s.stats()
		killed := s.takeMobDamage(pendingDamage{MobGlobalID: 1}, damage, &st)
		after := s.stats()
		if _, err := state.Apply(buildCurrentHitpoints(&after)[2:]); err != nil {
			t.Fatal(err)
		}
		if state.Pools.HP != s.char.CurrentHP || killed != (state.Pools.HP == 0) {
			t.Fatal("server/client HP mismatch", state.Pools, s.char.CurrentHP, killed)
		}
	}
	if s.char.CurrentHP != 0 {
		t.Fatal("equipment established an HP floor")
	}
	pool, _ := db.Pool(db.GameData)
	var stored uint32
	err := pool.QueryRow("SELECT HP FROM PS_GameData_Chars WHERE CharID=?", s.char.ID).Scan(&stored)
	if err != nil || stored != 0 {
		t.Fatal("death not persisted", stored, err)
	}
}

func TestExpWheelCastsAndSynchronizesAuthoritativeExperience(t *testing.T) {
	s, add := forgeRuleSession(t)
	defs, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	s.char.Level, s.char.MapID = 10, 18
	prev, next := expBounds(s.char)
	s.char.Exp = prev + (next-prev)/2
	d := defs.Item(44, 213)
	add(db.Item{Bag: 1, Slot: 0, Type: d.Type, TypeID: d.TypeID, Count: 2})
	wire, packets := gmWireSession(t, *s.char, false)
	wire.items = s.items
	wire.itemCooldowns = map[uint8]time.Time{}
	wire.handleUseItem(proto.UseItemPacket(1, 0))
	if wire.char.Exp != prev || wire.items[0].Count != 1 {
		t.Fatal("wheel did not reset EXP and consume once", wire.char.Exp, prev, wire.items)
	}
	state := game.New()
	state.SelfID = s.char.ID
	state.Self.Dead = true
	seenSpell := false
	deadline := time.After(time.Second)
	for {
		select {
		case p := <-packets:
			if proto.Op(p) == proto.SkillUseNoTarget {
				seenSpell = true
			}
			if proto.Op(p) != proto.CharacterDetails {
				continue
			}
			if _, err := state.Apply(p); err != nil {
				t.Fatal(err)
			}
			if !seenSpell || state.Details.Exp != prev || !state.Self.Dead {
				t.Fatal("snapshot lost EXP or entity state", state.Details)
			}
			return
		case <-deadline:
			t.Fatal("missing wheel spell or EXP snapshot")
		}
	}
}

func TestGMExperienceStoneRestoresHistoricalBuff(t *testing.T) {
	s, add := forgeRuleSession(t)
	defs, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	item := defs.Item(44, 242)
	if item == nil || item.Range != 250 || item.AttackTime != 2 || item.Special != 0 {
		t.Fatalf("GM EXP Stone reference: %+v", item)
	}
	rankOne := defs.Skill(250, 1)
	if rankOne == nil || rankOne.KeepSeconds != 3600 || rankOne.AbilityValue[0] != 50 {
		t.Fatalf("regular Mini Sun EXP Stone changed: %+v", rankOne)
	}
	snowball := defs.Item(44, 147)
	if snowball == nil || itemBuffSkill(snowball) != nil {
		t.Fatal("targeted Snowball was treated as a buff")
	}
	sk := itemBuffSkill(item)
	if sk == nil || sk.SkillLevel != 2 || sk.KeepSeconds != 86400 || !sk.KeepOnDeath || sk.TypeShow != 0 {
		t.Fatalf("restored item skill: %+v", sk)
	}

	add(db.Item{Bag: 1, Slot: 0, Type: 44, TypeID: 242, Count: 2})
	wire, packets := gmWireSession(t, *s.char, false)
	wire.items = s.items
	wire.itemCooldowns = map[uint8]time.Time{}
	wire.handleUseItem(proto.UseItemPacket(1, 0))
	if wire.items[0].Count != 1 || wire.experienceBonus() != 50 || len(wire.buffs) != 1 {
		t.Fatalf("use failed to consume once and grant 50%% EXP: items=%v buffs=%v", wire.items, wire.buffs)
	}
	deadline := time.After(time.Second)
	for {
		select {
		case p := <-packets:
			if proto.Op(p) != proto.SkillUseNoTarget {
				continue
			}
			result, err := proto.DecodeSkillResult(p)
			if err != nil || result.ID != 250 || result.Level != 2 {
				t.Fatalf("item cast result=%+v error=%v", result, err)
			}
			return
		case <-deadline:
			t.Fatal("item cast did not announce the restored rank")
		}
	}
}
