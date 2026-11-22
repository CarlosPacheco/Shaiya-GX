package app

import "github.com/Spelunkern/Shaiya-GX/internal/scene"

// 0x43efab, 0x44287b..0x44302e: jump adds 9 to the exponentially
// accelerated fall speed. This is visual prediction, never damage or immunity.
// startJumpAnimation is the key press: the jump clip starts at once, but the
// impulse waits for the next frame, when the clip has advanced
// (0x43bfa0: DAT_0071d010 and +0xc8 > 0 before DAT_0071d008 = 1).
func (v *worldView) startJumpAnimation() {
	if v.self.PlayingAction(8) {
		return // The landing frames can outlast the vertical impulse.
	}
	v.jumpPending = true
	if v.self != nil && !v.self.Mounted {
		_, _, casting := v.self.CastTiming()
		attacking := v.attackPose && v.self.ActionActive() && v.self.ActionID() == v.attackActionID
		if casting || attacking {
			return // Vertical movement can continue under a combat action.
		}
		v.self.PlayAction(8, false)
	}
}

// beginJump starts the impulse.
func (v *worldView) beginJump() {
	v.slope.Frames = 0
	v.jumpPending = false
	v.jumping, v.jumpFall, v.jumpFloor = true, -2.98, v.pos[1]
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
		v.jumping = false
	}
	changed := pos != v.pos
	v.place(pos)
	return changed
}
