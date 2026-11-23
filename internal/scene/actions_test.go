package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"testing"
)

func TestSkillActionsFollowNativeCommonBranches(t *testing.T) {
	if SkillCastAction(1, 3) != 88 || SkillCastAction(0, 1) != 85 || SkillCastAction(0, 4) != 91 {
		t.Fatal("casting ready action")
	}
	for _, tc := range []struct {
		anim                   uint16
		attack, effect, weapon uint8
		want                   int
	}{
		{100, 1, 3, 7, 100}, {110, 1, 1, 1, 110}, {0, 1, 1, 1, 35},
		{1, 3, 3, 13, 90}, {0, 3, 1, 13, 87}, {0, 0, 4, 1, 93},
	} {
		if got := SkillAction(tc.anim, tc.attack, tc.effect, tc.weapon, 0); got != tc.want {
			t.Errorf("%+v: %d", tc, got)
		}
	}
}

func TestCastCanBeInterruptedWithoutReleasingDeath(t *testing.T) {
	c := &Character{locked: true, castLeft: 2, clip: &Clip{ani: &model.ANI{Start: 0, End: 300}}}
	c.Update(.5)
	if !c.locked || c.castLeft != 1.5 {
		t.Fatal("cast released before its duration")
	}
	c.CancelCast()
	if c.locked || c.castLeft != 0 {
		t.Fatal("movement did not interrupt casting")
	}
	c.locked, c.hold = true, true
	c.CancelCast()
	if !c.locked || !c.hold {
		t.Fatal("cast interruption released death")
	}
}

func TestDeathHoldsLastFrame(t *testing.T) {
	c := &Character{locked: true, hold: true, oneShot: true, clip: &Clip{ani: &model.ANI{Start: 0, End: 30}}}
	c.Update(10)
	if c.Time != 1 || c.clip.frame(c.Time, c.oneShot) != 30 {
		t.Fatal("death looped instead of holding its final frame")
	}
	c.ReleaseAction()
	if c.locked {
		t.Fatal("resurrection did not release death pose")
	}
}

func TestSpinRepeatsThenExpiresAndCanBeReplaced(t *testing.T) {
	c := &Character{locked: true, spinLeft: 2, clip: &Clip{ani: &model.ANI{Start: 0, End: 6}}}
	for range 90 {
		c.Update(1.0 / 60)
		if !c.ActionActive() || c.Time >= .2 {
			t.Fatal("spin stopped instead of looping", c.Time, c.spinLeft)
		}
	}
	c.ReleaseAction()
	if c.ActionActive() || c.spinLeft != 0 {
		t.Fatal("spin resisted replacement")
	}
	c.locked, c.spinLeft = true, 2
	c.Update(2.01)
	if c.ActionActive() {
		t.Fatal("spin did not expire")
	}
}

func TestLateSkillResultDoesNotCutTheJumpClip(t *testing.T) {
	jump := &Character{locked: true, actionSlot: 8, clip: &Clip{ani: &model.ANI{Start: 0, End: 24}}}
	jump.ReleaseActionExceptJump()
	if !jump.PlayingAction(8) {
		t.Fatal("a rejected skill released the jump clip")
	}
	pose := &Character{locked: true, actionSlot: 20, clip: &Clip{ani: &model.ANI{Start: 0, End: 24}}}
	pose.ReleaseActionExceptJump()
	if pose.ActionActive() {
		t.Fatal("a skill pose must still be released")
	}
	var none *Character
	none.ReleaseActionExceptJump()
}

// The Wind Spin returns early from Update; the cloak clock must still run or
// the cape hangs frozen for the whole spin.
func TestCloakKeepsSimulatingDuringSpin(t *testing.T) {
	c := &Character{locked: true, spinLeft: 2, clip: &Clip{ani: &model.ANI{Start: 0, End: 6}}, cloak: &cloakInstance{}}
	c.Update(1.0 / 60)
	if c.cloak.elapsed <= 0 {
		t.Fatal("cloak physics clock stopped while spinning")
	}
}
