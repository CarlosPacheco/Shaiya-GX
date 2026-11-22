package app

import "github.com/Spelunkern/Shaiya-GX/internal/scene"

// 0x43efab, 0x44287b..0x44302e: jump adds 9 to the exponentially
// accelerated fall speed. This is visual prediction, never damage or immunity.
func (v *worldView) beginJump() {
	v.slope.Frames = 0
	v.jumping, v.jumpFall, v.jumpFloor = true, -2.98, v.pos[1]
	if v.self != nil && !v.self.Mounted {
		v.self.ReleaseAction()
		v.self.PlayAction(8, false)
	}
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
				v.jumpFall = min(v.jumpFall, -9)
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
