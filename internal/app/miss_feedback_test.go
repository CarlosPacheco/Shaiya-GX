package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestMagicSpotMissPacketReachesChatAndFloatingMiss(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindMob, X: 4}
	a.skillDefs[sdata.SkillKey{ID: 137, Level: 3}] = sdata.Skill{Name: "Magic Spot Lv3", Attack: 3, Effect: 1, Target: 3}
	a.msg = text.SysMsg{1100: "Used <s>.", 1108: "Failed <s> against <t>."}
	w := proto.NewWriter(proto.SkillUse)
	w.U8(3)
	w.U32(1)
	w.U32(2)
	w.U16(137)
	w.U8(3)
	w.U16(0)
	w.U16(0)
	w.U16(0)
	if _, err := a.state.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	a.confirmedSystemChat(w.Bytes(), 0)
	a.confirmedSkillFeedback(proto.SkillUse)
	if len(a.state.Chat) != 2 || a.state.Chat[1].MessageID != 1108 {
		t.Fatal("missing native miss chat", a.state.Chat)
	}
	if len(a.combatNumbers) != 1 || !a.combatNumbers[0].Missed {
		t.Fatal("missing floating miss", a.combatNumbers)
	}
}

func TestSkillCooldownRefusalUsesOriginalMessage(t *testing.T) {
	a := inputApp()
	a.clock = 10
	a.skills.cooldowns = map[uint16]float64{137: 11}
	a.msg = text.SysMsg{215: "Recharge must be completed before next use"}
	if p := a.learnedSkillPacket(proto.LearnedSkill{ID: 137, Level: 3}); p != nil {
		t.Fatal("sent cooling skill")
	}
	if len(a.state.Chat) != 1 || a.state.Chat[0].MessageID != 215 {
		t.Fatal("silent cooldown refusal")
	}
}

func TestBasicMissPacketsUseOriginalOutgoingAndIncomingMessages(t *testing.T) {
	for _, outgoing := range []bool{true, false} {
		a := inputApp()
		a.state.SelfID = 1
		a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindMob, X: 4}
		a.msg = text.SysMsg{1004: "Attack against <t> missed.", 1005: "<p>'s attack missed the target."}
		op, source, target, want := proto.CharacterMobAutoAttack, uint32(1), uint32(2), 1004
		if !outgoing {
			op, source, target, want = proto.MobAttackCharacter, 2, 1, 1005
		}
		w := proto.NewWriter(op)
		w.U8(3)
		w.U32(source)
		w.U32(target)
		w.U16(0)
		w.U16(0)
		w.U16(0)
		if _, err := a.state.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
		a.confirmedSystemChat(w.Bytes(), 0)
		if len(a.state.Chat) != 1 || a.state.Chat[0].MessageID != want {
			t.Fatal("missing basic miss chat", a.state.Chat)
		}
	}
}
