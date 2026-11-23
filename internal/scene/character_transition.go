package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// The original character and creature controllers blend for five frames at 30 Hz.
const clipBlendDuration = float32(5.0 / 30)

func (c *Character) invalidatePose() {
	c.lastFinalsClip = nil
	c.poseReady = false
	c.boundReady = false
	c.shadowFitReady = false
}

func (c *Character) startClip(next *Clip, oneShot bool) {
	c.startClipWithBlend(next, oneShot, false)
}

func (c *Character) startClipWithBlend(next *Clip, oneShot, allowTrailingBones bool) {
	previous := c.clip
	count := len(next.ani.Bones)
	compatible := false
	if previous != nil && count > 0 {
		previousCount := len(previous.ani.Bones)
		// EP5.4's run-to-idle blend skips outgoing bones missing from idle.
		compatible = previousCount == count || (allowTrailingBones && previousCount > count)
	}
	if compatible {
		for i, bone := range next.ani.Bones {
			if bone.Parent != previous.ani.Bones[i].Parent {
				compatible = false
				break
			}
		}
	}
	if compatible {
		if cap(c.blendFrom) < count {
			c.blendFrom = make([]bonePose, count)
		}
		c.blendFrom = c.blendFrom[:count]
		frame := previous.frame(c.Time, c.oneShot)
		for i := range c.blendFrom {
			// If interrupted, continue from the visible blend rather than
			// snapping back to either of the two earlier clips.
			c.blendFrom[i] = c.localPose(i, frame)
		}
		c.blendLeft = clipBlendDuration
	} else {
		c.blendLeft = 0
	}
	c.clip, c.Time, c.oneShot = next, 0, oneShot
	c.rate.kind, c.rate.next = c.rate.next, rateNone
	c.invalidatePose()
}

func (c *Character) advanceBlend(dt float32) float32 {
	if c.blendLeft <= 0 || dt <= 0 {
		return dt
	}
	step := min(c.blendLeft, dt)
	c.blendLeft -= step
	c.invalidatePose()
	return dt - step
}

func (c *Character) localPose(i int, frame float32) bonePose {
	if c.blendLeft <= 0 {
		return c.clip.localPose(i, frame)
	}
	from, into := c.blendFrom[i], c.clip.restPose(i)
	t := 1 - c.blendLeft/clipBlendDuration
	if t == 0 {
		return from
	}
	into.rotation = render.Slerp(from.rotation, into.rotation, t)
	for axis, value := range from.position {
		into.position[axis] = value + (into.position[axis]-value)*t
	}
	return into
}
