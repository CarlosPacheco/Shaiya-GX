package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestSkillResetStoneKeepsCooldownAfterLearningAgain(t *testing.T) {
	s, add := forgeRuleSession(t)
	const skillID = 64001
	defs := db.MustDefs()
	defs.Items[db.ItemKey{Type: 100, TypeID: 48}] = &db.ItemDef{Type: 100, TypeID: 48, Special: specialSkillReset}
	defs.Skills = map[db.SkillKey]*db.SkillDef{{SkillID: skillID, Level: 1}: {
		SkillID: skillID, SkillLevel: 1, TypeShow: 1, SkillPointCost: 3, ClassEligible: [6]bool{true},
	}}
	add(db.Item{Type: 100, TypeID: 48, Bag: 1, Slot: 1, Count: 2})
	s.char.Level, s.char.Mode = 10, growHard
	s.skills = []db.CharacterSkill{{SkillID: skillID, Level: 1, Number: 1}}
	if err := db.SaveCharacterSkill(s.char.ID, s.skills[0]); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Minute).Truncate(time.Millisecond)
	s.skillCooldowns = map[uint16]time.Time{skillID: deadline}
	s.quickSlots = []db.QuickSlot{{Bag: 100, Number: 1}, {Bag: 1, Number: 1, QuickSlot: 1}}
	if err := db.SaveQuickSlots(s.char.ID, s.quickSlots); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		store, _ := db.Pool(db.GameData)
		for _, table := range []string{"GX_ResetSkillCooldowns", "PS_GameData_CharSkills", "PS_GameData_CharQuickSlots"} {
			store.Exec("DELETE FROM "+table+" WHERE CharID=?", s.char.ID)
		}
	})
	s.handleUseItem(proto.UseItemPacket(1, 1))
	if s.char.SkillPoint != 50 || len(s.skills) != 0 || len(s.quickSlots) != 1 || s.items[0].Count != 1 {
		t.Fatal("invalid reset state")
	}
	saved, err := db.ResetSkillCooldowns(s.char.ID)
	if err != nil || !saved[skillID].Equal(deadline) {
		t.Fatal(saved, err)
	}
	s.skillCooldowns = saved
	s.handleLearnSkill([]byte{0x09, 0x02, byte(skillID & 255), byte(skillID >> 8), 1})
	if len(s.skills) != 1 || s.skillReady(skillID) || s.skills[0].Delay <= 0 {
		t.Fatal("relearning bypassed cooldown")
	}
	if s.char.SkillPoint != 47 {
		t.Fatal("wrong points after relearning")
	}
}

func TestTargetResurrectionRuneValidatesAndRestoresFullPools(t *testing.T) {
	s, add := forgeRuleSession(t)
	defs := db.MustDefs()
	defs.Items[db.ItemKey{Type: 100, TypeID: 67}] = &db.ItemDef{Type: 100, TypeID: 67, Special: specialTargetResurrection}
	add(db.Item{Type: 100, TypeID: 67, Bag: 1, Slot: 1, Count: 2})
	id, err := db.CreateCharacter(testAccount(t), db.NewCharacter{Slot: 3, Name: "RuneResProbe", HP: 100})
	if err != nil {
		t.Fatal(err)
	}
	target, _ := gmWireSession(t, db.Character{ID: id, Level: 1, MapID: s.char.MapID, PosX: 61}, false)
	target.dead = true
	if err := db.SaveCharacter(target.char); err != nil {
		t.Fatal(err)
	}
	player := RegisterPlayer(*target.char, nil, false, target.faction, target.out, target.logger, target)
	target.player = player
	player.SetAlive(false)
	t.Cleanup(func() {
		player.Unregister()
		store, _ := db.Pool(db.GameData)
		store.Exec("DELETE FROM PS_GameData_Chars WHERE CharID=?", id)
	})
	request := proto.TargetResurrectionItemPacket(1, 1, id)
	s.handleUseItem(request)
	if !target.dead || s.items[0].Count != 2 {
		t.Fatal("out-of-range rune was accepted")
	}
	target.char.PosX = 60
	target.faction = FactionDark
	s.handleUseItem(request)
	if !target.dead || s.items[0].Count != 2 {
		t.Fatal("enemy rune was accepted")
	}
	target.faction = s.faction
	s.movement.motion = 1
	s.handleUseItem(request)
	stats := target.stats()
	if target.dead || !player.IsAlive() || target.char.PosX != 60 || s.items[0].Count != 1 ||
		stats.CurrentHP != stats.MaxHP || stats.CurrentMP != stats.MaxMP || stats.CurrentSP != stats.MaxSP {
		t.Fatal("eligible ally was not revived in place with full pools")
	}
	s.handleUseItem(request)
	if s.items[0].Count != 1 {
		t.Fatal("already alive target consumed another rune")
	}
}
