package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestNPCQuestRewardChoiceRequiresPressAndRelease(t *testing.T) {
	v := &NPCQuest{held: -1, RewardSelected: -1, RewardSelectable: true, Rewards: make([]NPCQuestReward, 3)}
	x, y := NPCQuestOrigin(1024, 768)
	second := y + npcQuestRewardTop + npcQuestRewardHeight + 5
	p := v.Update(&platform.Input{MouseX: x + 30, MouseY: second, Pressed: true, MouseDown: true}, 1024, 768, 2, 0, true, true)
	if p.Reward != -1 {
		t.Fatal("press chose reward")
	}
	p = v.Update(&platform.Input{MouseX: x + 30, MouseY: second, Clicked: true}, 1024, 768, 2, 0, true, true)
	if p.Reward != 1 || v.RewardSelected != 1 {
		t.Fatalf("choice row=%d", p.Reward)
	}
	p = v.Update(&platform.Input{MouseX: x + 30, MouseY: y + npcQuestRewardTop + 5, Clicked: true}, 1024, 768, 2, 0, true, true)
	if p.Reward != -1 || v.RewardSelected != 1 {
		t.Fatal("release without press chose reward")
	}
	v.RewardSelectable = false
	v.Update(&platform.Input{MouseX: x + 30, MouseY: second, Pressed: true, MouseDown: true}, 1024, 768, 2, 0, true, false)
	if p = v.Update(&platform.Input{MouseX: x + 30, MouseY: second, Clicked: true}, 1024, 768, 2, 0, true, false); p.Reward != -1 {
		t.Fatal("display-only rewards were selectable")
	}
}

func TestNPCQuestDescriptionScrollPreservesListAndRewards(t *testing.T) {
	v := &NPCQuest{held: -1, Rewards: make([]NPCQuestReward, 2)}
	x, y := NPCQuestOrigin(1024, 768)
	a := v.Update(&platform.Input{MouseX: x + 30, MouseY: y + 150, Wheel: -120}, 1024, 768, 10, 2, true, true)
	if v.textOffset != 3 || a.Offset != 2 || a.Reward != -1 {
		t.Fatal("description wheel affected list or reward selection", v.textOffset, a)
	}
	v.Update(&platform.Input{MouseX: x + 30, MouseY: y + 350, Wheel: -120}, 1024, 768, 10, 2, true, true)
	if v.textOffset != 3 {
		t.Fatal("reward area moved description")
	}
	v.ResetDescriptionScroll()
	if v.textOffset != 0 {
		t.Fatal("selection did not reset description")
	}
}

func TestGenericNPCNativeOkayClosesConversation(t *testing.T) {
	v, err := NewNPCQuest(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	x, y := v.Origin(1024, 768)
	in := platform.Input{MouseX: x + 58, MouseY: y + 342, Pressed: true, MouseDown: true}
	if got := v.Update(&in, 1024, 768, 0, 0, false, false); got.Close {
		t.Fatal("button fired on press")
	}
	in.Pressed, in.MouseDown, in.Clicked = false, false, true
	if got := v.Update(&in, 1024, 768, 0, 0, false, false); !got.Close {
		t.Fatal("native okay did not close dialogue")
	}
	v.ActionDisabled = true
	in = platform.Input{MouseX: x + 58, MouseY: y + 472, Pressed: true, MouseDown: true}
	v.Update(&in, 1024, 768, 1, 0, true, true)
	in.Pressed, in.MouseDown, in.Clicked = false, false, true
	if got := v.Update(&in, 1024, 768, 1, 0, true, true); got.Accept {
		t.Fatal("incomplete quest submitted reward request")
	}
}
