package scene

import (
	"math"
	"testing"

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
