package scene

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestClipTransitionPreservesPoseAndDelaysPlayback(t *testing.T) {
	outgoing := poseTestClip(2)
	incoming := poseTestClip(2)
	incoming.restLocal[0][12] = 2
	incoming.restLocal[1][12] = 3
	for i := range incoming.ani.Bones {
		incoming.ani.Bones[i].Trans = nil
	}
	c := &Character{clip: outgoing, Time: .5}
	last := c.finals()[1]
	c.startClip(incoming, false)
	if c.finals()[1] != last {
		t.Fatal("clip switched before the transition")
	}
	c.Update(clipBlendDuration / 2)
	middle := c.finals()[1]
	if c.Time != 0 || math.Abs(float64(middle[12]-(last[12]+5)/2)) > .0001 {
		t.Fatal("blend did not interpolate local pose while holding playback", c.Time, middle)
	}
	c.startClip(outgoing, false)
	if c.finals()[1] != middle {
		t.Fatal("interruption discarded the pose currently shown")
	}
	c.Update(clipBlendDuration)
	if c.Time != 0 || c.blendLeft != 0 || c.finals()[1] != outgoing.Finals(0)[1] {
		t.Fatal("incoming playback did not begin at its opening pose")
	}
	c.Update(.1)
	if c.Time != .1 {
		t.Fatal("incoming playback failed to advance")
	}
}

func TestClipTransitionHoldsFinalActionPose(t *testing.T) {
	clip := poseTestClip(1)
	c := &Character{clip: clip, locked: true, oneShot: true, Time: .95}
	c.Update(.1)
	if c.ActionActive() || c.Time != 1 || c.finals()[0][13] != .5 {
		t.Fatal("completed action wrapped to its opening pose", c.Time, c.finals())
	}
	c.startClip(poseTestClip(1), false)
	if c.finals()[0][13] != .5 {
		t.Fatal("the outgoing transition did not start at the last action frame")
	}
}

func TestClipTransitionFallsBackForDifferentSkeletons(t *testing.T) {
	first := poseTestClip(2)
	second := poseTestClip(2)
	second.ani.Bones[1].Parent = -1
	c := &Character{clip: first}
	c.startClip(second, false)
	if c.blendLeft != 0 || c.finals()[1] != second.Finals(0)[1] {
		t.Fatal("blended incompatible hierarchies")
	}
}

func TestRunToNormalIdleBlendsSharedBones(t *testing.T) {
	assets := NewAssets(nil)
	run, idle := poseTestClip(3), poseTestClip(2)
	assets.clips["ani:character/human/ani/humf_040_onrun.ani"] = run
	assets.clips["ani:character/human/ani/humf_000_normal.ani"] = idle
	assets.clips["ani:character/human/ani/humf_034_onready.ani"] = idle
	assets.clips["ani:character/human/ani/humf_001_walk.ani"] = idle
	actions := make([]model.Action, 41)
	actions[40].Name = "humf_040_onrun.ANI"
	actions[34].Name = "humf_034_onready.ANI"

	for _, tc := range []struct {
		name, destination string
		combatPose        bool
		blend             bool
	}{
		{name: "normal idle", destination: "000_normal", blend: true},
		{name: "weapon ready", destination: "000_normal", combatPose: true},
		{name: "walk", destination: "001_walk"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Character{assets: assets, folder: "Human", prefix: "humf", weaponType: 45, actionsLoaded: true, actions: actions}
			c.SetClip("002_run")
			c.Update(.5)
			last := c.finals()[1]
			if tc.combatPose {
				c.combatPoseLeft = 10
			}
			c.SetClip(tc.destination)
			if !tc.blend {
				if c.blendLeft != 0 {
					t.Fatal("blended mismatched skeletons outside the normal run stop")
				}
				return
			}
			if c.blendLeft != clipBlendDuration || c.finals()[1] != last {
				t.Fatal("run stop dropped the outgoing pose")
			}
			c.Update(clipBlendDuration / 2)
			c.SetClip("000_normal")
			if c.Time != 0 || c.blendLeft != clipBlendDuration/2 || c.finals()[1] == last {
				t.Fatal("run stop failed to interpolate while holding idle playback")
			}
			c.Update(clipBlendDuration / 2)
			if c.blendLeft != 0 || c.finals()[1] != idle.Finals(0)[1] {
				t.Fatal("run stop did not reach the idle pose")
			}
		})
	}
}

func TestJumpStartsImmediatelyAndReturnsThroughNormalIdle(t *testing.T) {
	assets := NewAssets(nil)
	run, jump, idle := poseTestClip(3), poseTestClip(3), poseTestClip(2)
	assets.clips["ani:character/human/ani/humf_002_run.ani"] = run
	assets.clips["ani:character/human/ani/humf_008_jump.ani"] = jump
	assets.clips["ani:character/human/ani/humf_000_normal.ani"] = idle
	actions := make([]model.Action, 35)
	actions[8] = model.Action{Name: "humf_008_jump.ANI", Mode: 2}
	actions[34].Name = "humf_034_onready.ANI"
	c := &Character{assets: assets, folder: "Human", prefix: "humf", weaponType: 45, actionsLoaded: true, actions: actions}
	c.startClip(poseTestClip(3), false)
	c.SetClip("002_run")
	if c.blendLeft != clipBlendDuration {
		t.Fatal("fixture must start in a locomotion blend")
	}
	if !c.PlayAction(8, false) || c.blendLeft != 0 || c.finals()[1] != jump.Finals(0)[1] {
		t.Fatal("jump did not immediately replace the locomotion blend")
	}
	c.Update(.125)
	if c.Time != .125 {
		t.Fatalf("jump playback was delayed: cursor %v", c.Time)
	}
	jumpEnd := &Character{clip: jump, oneShot: true, Time: jump.Duration()}
	c.combatPoseLeft = 10
	c.Update(jump.Duration() - c.Time)
	if !c.PlayingClip("000_normal") || c.blendLeft != clipBlendDuration || c.finals()[1] != jumpEnd.finals()[1] {
		t.Fatal("jump did not blend from its last pose to normal idle")
	}
	c.SetClip("002_run")
	c.Update(clipBlendDuration / 2)
	c.SetClip("000_normal")
	if !c.PlayingClip("000_normal") || c.Time != 0 || c.finals()[1] == jumpEnd.finals()[1] {
		t.Fatal("locomotion or weapon-ready input interrupted the landing blend")
	}
	c.Update(clipBlendDuration / 2)
	if c.finals()[1] != idle.Finals(0)[1] {
		t.Fatal("landing blend did not reach the opening idle pose")
	}
	c.SetClip("002_run")
	if !c.PlayingClip("002_run") {
		t.Fatal("locomotion did not resume after the landing blend")
	}
}

func TestClipTransitionUsesLocalQuaternionInterpolation(t *testing.T) {
	first := poseTestClip(2)
	second := poseTestClip(2)
	second.ani.Bones[0].Rot, second.ani.Bones[0].Trans = nil, nil
	second.restLocal[0] = render.QuatMatrix([4]float32{0, 0, 1, 0}, [3]float32{})
	c := &Character{clip: first, Time: .5}
	c.startClip(second, false)
	c.Update(clipBlendDuration / 2)
	// A spherical half-turn blend reaches a quarter-turn matrix. Direct
	// matrix interpolation would collapse the two rotation axes here.
	m := c.finals()[0]
	if math.Abs(float64(m[0])) > .0001 || math.Abs(math.Abs(float64(m[1]))-1) > .0001 {
		t.Fatal("blend did not keep a rigid rotation", m)
	}
}
