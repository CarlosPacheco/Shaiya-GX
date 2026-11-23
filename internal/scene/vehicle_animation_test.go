package scene

import (
	"fmt"
	"testing"
)

func TestVehicleBreathGestureReverseAndJump(t *testing.T) {
	assets := &Assets{clips: map[string]*Clip{}, Missing: map[string]bool{}}
	actor := &CreatureActor{Character: &Character{assets: assets}, folder: "Vehicle", action: -1}
	for index := range actor.visual.Animations {
		name := fmt.Sprintf("%d.ani", index)
		actor.visual.Animations[index] = name
		clip := poseTestClip(2)
		clip.ani.End = 60
		assets.clips["ani:vehicle/ani/"+name] = clip
	}
	mount := &Vehicle{CreatureActor: actor}
	mount.Update(1)
	if mount.action != 6 {
		t.Fatal("resting mount must breathe")
	}
	mount.Update(10.1)
	if mount.action != 8 {
		t.Fatal("occasional gesture missing")
	}
	mount.Update(2.1)
	if mount.Time >= 2 {
		t.Fatal("gesture looped instead of playing once")
	}
	mount.Update(1)
	if mount.action != 6 {
		t.Fatal("gesture did not return to breathing")
	}
	mount.SetMotion(true, false, true, false)
	mount.Update(.2)
	if mount.action != 0 || mount.Time < 1.8 {
		t.Fatal("backward walk did not start at end of clip", mount.Time)
	}
	before := mount.Time
	mount.Update(.1)
	if mount.Time >= before {
		t.Fatal("walk did not play backwards")
	}
	mount.SetMotion(true, false, false, true)
	mount.Update(.1)
	if mount.action != 2 || mount.Time != .1 || mount.blendLeft != 0 {
		t.Fatal("jump did not activate MON 2 directly")
	}
	mount.SetMotion(true, false, false, false)
	mount.Update(.1)
	if mount.action != 2 {
		t.Fatal("physical landing cut the jump")
	}
	mount.Update(2)
	if mount.action != 6 || mount.blendLeft != clipBlendDuration {
		t.Fatal("jump did not finish by blending into breathing")
	}
	mount.Update(clipBlendDuration)
	mount.Update(.1)
	if mount.action != 1 {
		t.Fatal("landing did not return to running")
	}
	mount.SetMotion(true, false, false, true)
	mount.Update(2)
	mount.Update(clipBlendDuration)
	mount.Update(.1)
	if mount.action != 1 {
		t.Fatal("a sustained airborne flag restarted the jump clip")
	}
}
