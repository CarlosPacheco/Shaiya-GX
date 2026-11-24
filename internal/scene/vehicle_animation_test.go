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

// Game.exe 0x4192e4/0x4195b1: a hoof lands when the clip passes a step frame of
// the mount's action and rider race; frame 0 never fires.
func TestMountStepCrossed(t *testing.T) {
	// human walk: 7, 17, 2, 13
	if !MountStepCrossed(0, 6.0/30, 0, 7.5/30, 0) || MountStepCrossed(0, 7.5/30, 0, 9.0/30, 0) {
		t.Fatal("human walk frame 7")
	}
	// elf walk: 14 and 4 only
	if MountStepCrossed(0, 6.0/30, 0, 7.5/30, 1) || !MountStepCrossed(0, 3.0/30, 0, 4.5/30, 1) {
		t.Fatal("elf walk frames")
	}
	// a clip loop wrap passes frame 2 of the human walk
	if MountStepCrossed(0, 20.0/30, 0, 1.0/30, 0) || !MountStepCrossed(0, 28.0/30, 0, 3.0/30, 0) || !MountStepCrossed(0, 20.0/30, 0, 3.0/30, 0) {
		t.Fatal("wrapped loop missed a step")
	}
	// the death eater run lists frame 0, which never fires on its own
	if MountStepCrossed(1, 0, 1, 1.0/30, 3) {
		t.Fatal("frame 0 fired")
	}
	// changing action starts from the first frame
	if !MountStepCrossed(0, 25.0/30, 1, 16.0/30, 0) {
		t.Fatal("new action ignored its first frames")
	}
	if MountStepCrossed(2, 0, 2, 20.0/30, 0) || MountStepCrossed(0, 0, 0, 1, 4) {
		t.Fatal("jump or unknown race stepped")
	}
}

func TestMountStepsFollowBackwardPlayback(t *testing.T) {
	// Human walk steps at frames 7, 17, 2, 13 (30 fps). Backwards from frame 10 to 9
	// crosses none; from 14 to 12 crosses 13; wrapping back through the start from
	// 1 to the end crosses the frame-2 hoof only when the cursor passes it.
	f := func(frame float32) float32 { return frame / 30 }
	if MountStepCrossedDirected(0, f(10), 0, f(9), 0, true) {
		t.Fatal("a backward step with no hoof frame played a step")
	}
	if !MountStepCrossedDirected(0, f(14), 0, f(12), 0, true) {
		t.Fatal("a backward step over frame 13 lost its hoof")
	}
	if !MountStepCrossedDirected(0, f(3), 0, f(29.5), 0, true) {
		t.Fatal("a backward wrap over the clip start must cross the frame-2 hoof")
	}
	if MountStepCrossedDirected(0, f(1), 0, f(29.5), 0, true) {
		t.Fatal("a backward wrap with no hoof frame played a step")
	}
	if !MountStepCrossedDirected(0, f(6), 0, f(8), 0, false) || MountStepCrossedDirected(0, f(8), 0, f(9), 0, false) {
		t.Fatal("forward behaviour changed")
	}
	// The old forward-only test fired on nearly every backward frame.
	fired := 0
	for frame := float32(30); frame > 5; frame -= .4 {
		if MountStepCrossedDirected(0, f(frame), 0, f(frame-.4), 0, true) {
			fired++
		}
	}
	if fired < 2 || fired > 5 {
		t.Fatalf("backward walk fired %d steps over one cycle, want a handful", fired)
	}
}
