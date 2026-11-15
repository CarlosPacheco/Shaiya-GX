package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"testing"
)

func TestEarlierComboAcknowledgmentDoesNotRewindLatestAnimation(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.wv.self = &scene.Character{Time: .35}
	first := &requestedSkillPresentation{key: sdata.SkillKey{ID: 100, Level: 1}, target: 2, actor: a.wv.self}
	latest := &requestedSkillPresentation{key: sdata.SkillKey{ID: 101, Level: 1}, target: 2, actor: a.wv.self}
	a.wv.requestedSkills = []*requestedSkillPresentation{first, latest}
	a.wv.requestedSkill = latest
	for _, id := range []uint16{100, 101} {
		a.state.LastSkill = proto.SkillResult{Source: 1, Target: 2, ID: id, Level: 1}
		a.confirmedAnimation(proto.NewWriter(proto.SkillUse).Bytes())
		if a.wv.self.Time != .35 {
			t.Fatal("acknowledgment rewound combo animation")
		}
		if id == 100 && a.wv.requestedSkill != latest {
			t.Fatal("older ack discarded latest prediction")
		}
	}
	if a.wv.requestedSkill != nil || len(a.wv.requestedSkills) != 0 {
		t.Fatal("acknowledged predictions leaked")
	}
}

func TestOlderAreaReplyPreservesNewerComboPrediction(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.wv.self = &scene.Character{Time: .35}
	latest := &requestedSkillPresentation{key: sdata.SkillKey{ID: 101, Level: 1}, target: 2, actor: a.wv.self}
	a.wv.requestedSkill = latest
	a.state.LastSkill = proto.SkillResult{Source: 1, Target: 2, ID: 100, Level: 1, Result: 4}
	a.confirmedAnimation(proto.NewWriter(proto.SkillUseNoTarget).Bytes())
	if a.wv.requestedSkill != latest || a.wv.self.Time != .35 {
		t.Fatal("late area reply interrupted newer prediction")
	}
}
