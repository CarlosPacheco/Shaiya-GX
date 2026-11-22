package app

import "github.com/Spelunkern/Shaiya-GX/internal/scene"

// 0x43efab, 0x44287b..0x44302e: jump adds 9 to the exponentially
// accelerated fall speed. This is visual prediction, never damage or immunity.
// startJumpAnimation is the key press: the jump clip starts at once, but the
// impulse waits for the next frame, when the clip has advanced
// (0x43bfa0: DAT_0071d010 and +0xc8 > 0 before DAT_0071d008 = 1).
func (v *worldView) startJumpAnimation() {
	if v.jumpActive {
		return // The landing frames can outlast the vertical impulse.
	}
	v.jumpPending, v.jumpActive, v.jumpMoved = true, true, false
	if v.self != nil && !v.self.Mounted {
		v.self.PlayAction(8, false)
	}
}

// releaseSkillAction drops a predicted or failed skill pose. The jump clip is
// not a skill pose: a late rejection or failure result must not cut it short
// while the character is still in the air.
func (v *worldView) releaseSkillAction() {
	v.self.ReleaseActionExceptJump()
}

// beginJump starts the impulse.
func (v *worldView) beginJump() {
	v.slope.Frames = 0
	v.jumpPending = false
	v.jumping, v.jumpActive, v.jumpFall, v.jumpFloor = true, true, -2.98, v.pos[1]
}

func (v *worldView) advanceJump(dt float32) bool {
	if !v.jumping || dt <= 0 {
		return false
	}
	v.jumpFall = max(-20, v.jumpFall*(1+2.8*dt))
	pos := v.pos
	pos[1] += (9 + v.jumpFall) * dt
	floor := v.jumpFloor
	if v.world != nil && v.world.HasTerrain() {
		floor = v.world.Height(pos[0], pos[2])
		if _, deep := v.world.WaterFloor(pos[0], pos[2]); deep {
			floor = scene.SwimEntryY
		}
	}
	if v.world != nil && v.world.HasPlayerCollision() {
		ascending := pos[1] > v.pos[1]
		resolved, hit := v.world.MovePlayer(v.pos, pos.Sub(v.pos))
		// Only a vertical move that came up short is a ceiling; terrain that
		// rises under the jumper pushes it up and keeps the impulse.
		if hit && ascending && resolved[1]-v.pos[1] >= pos[1]-v.pos[1]-1e-4 {
			hit = false
		}
		if hit {
			if ascending {
				// Any vertical contact ends the impulse; the fall starts again
				// at -2.98 (0x43bfa0).
				v.jumping, v.groundFall = false, -2.98
			} else {
				v.jumping = false // the clip runs to its end, as in the original
			}
		}
		pos = resolved
		if !v.world.HasTerrain() {
			floor = pos[1] - 1
		}
	}
	if pos[1] <= floor {
		pos[1] = floor
		// Rising terrain pushes an ascending jumper up; only a falling one lands.
		if 9+v.jumpFall <= 0 {
			v.jumping = false
		}
	}
	changed := pos != v.pos
	v.place(pos)
	return changed
}
