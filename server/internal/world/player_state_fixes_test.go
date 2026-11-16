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
	// Every buff reference in the original item definitions must resolve.
	count := 0
	for _, d := range defs.Items {
		if !isConsumableType(d.Type) || d.Range == 0 || d.AttackTime == 0 {
			continue
		}
		sk := itemBuffSkill(d)
		if sk == nil {
			continue
		}
		count++
		if sk.TypeEffect != typeEffectBuff {
			t.Fatal("non-buff accepted")
		}
	}
	if count < 190 {
		t.Fatalf("incomplete consumable audit: %d", count)
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
	defs.Exp = map[db.ExpKey]uint32{{Level: 1, Grow: 1}: 1000}
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
	s.setPools(false)
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
