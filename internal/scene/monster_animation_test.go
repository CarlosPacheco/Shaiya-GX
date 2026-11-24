package scene

import (
	"fmt"
	"testing"
)

func TestMonsterIdleVariationYieldsToMovementAndAttack(t *testing.T) {
	assets := &Assets{clips: map[string]*Clip{}, Missing: map[string]bool{}}
	mob := &CreatureActor{Character: &Character{assets: assets}, folder: "Monster", action: -1}
	for action := range mob.visual.Animations {
		name := fmt.Sprintf("%d.ani", action)
		mob.visual.Animations[action] = name
		clip := poseTestClip(2)
		clip.ani.End = 60
		assets.clips["ani:monster/ani/"+name] = clip
	}
	setAction := func(action int) {
		t.Helper()
		if err := mob.SetAction(action); err != nil {
			t.Fatal(err)
		}
	}
	startIdle := func() {
		t.Helper()
		setAction(6)
		mob.PlayIdleVariation()
		if mob.action != 8 || mob.cueTime <= 0 {
			t.Fatal("expected idle variation to be in progress")
		}
	}
	startIdle()
	setAction(1)
	mob.Update(.02)
	if mob.action != 1 || mob.cueTime != 0 || mob.Time != .02 {
		t.Fatal("run did not immediately replace idle variation")
	}
	setAction(1)
	if mob.Time != .02 {
		t.Fatal("same locomotion action restarted its cycle")
	}
	startIdle()
	if !mob.PlayOnce(7) || mob.action != 7 {
		t.Fatal("the hit that drew aggro did not start a reaction")
	}
	setAction(1)
	mob.Update(.02)
	if mob.action != 1 || mob.cueTime != 0 || mob.Time != .02 {
		t.Fatal("damage reaction did not yield to the new chase")
	}
	startIdle()
	if !mob.PlayOnce(2) || !mob.Attacking() || mob.blendLeft != 0 {
		t.Fatal("attack did not immediately replace idle variation")
	}
	setAction(1)
	if !mob.Attacking() || mob.PlayOnce(7) {
		t.Fatal("locomotion or damage reaction cut a confirmed attack")
	}
}
