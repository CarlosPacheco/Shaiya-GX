package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/attackspeed"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func (a *App) attackMovementAllowed() bool {
	v := a.wv
	airborne := v.jumping || v.world != nil && !v.world.PlayerSupported(v.pos)
	return !airborne || !v.moving && !v.travel && !v.autoRun
}
func (a *App) stopForAttack() bool {
	v := a.wv
	if v.moving {
		motion := uint8(0)
		if v.jumping {
			motion = 4 | proto.MotionFlag80
		}
		if err := a.world.Send(v.movePacket(motion)); err != nil {
			a.backToLogin("Unable to stop for attack.")
			return false
		}
	}
	v.moving, v.travel, v.autoRun = false, false, false
	v.attackMoveUntil = a.clock + float64(attackspeed.MovementPauseMilliseconds)/1000
	return true
}

func (a *App) confirmAttackStop() {
	v := a.wv
	v.attackMoveUntil = max(v.attackMoveUntil, a.clock+float64(attackspeed.MovementPauseMilliseconds)/1000)
	v.attackPose = true
	if v.self != nil {
		v.attackActionID = v.self.ActionID()
	}
	v.travel, v.autoRun = false, false
}

// Call immediately before sending an offensive skill, after any approach.
func (a *App) prepareSkillMovement(p []byte) bool {
	if len(p) != 7 || proto.Op(p) != proto.SkillUse && proto.Op(p) != proto.SkillUseNoTarget {
		return true
	}
	k, ok := a.state.Skills[proto.NewReader(p).U8()]
	if !ok {
		return true
	}
	d := a.skillDefs[sdata.SkillKey{ID: k.ID, Level: k.Level}]
	if d.Effect != 1 && d.Effect != 3 {
		return true
	}
	if !a.attackMovementAllowed() || a.clock < a.wv.attackMoveUntil {
		return false
	}
	return a.stopForAttack()
}
