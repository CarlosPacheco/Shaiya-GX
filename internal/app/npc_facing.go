package app

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// npcTurnSpeed is how fast an NPC swings toward the character that talks to it.
const npcTurnSpeed = 7.0 // radians per second

// turnNPCToTalker follows Game.exe 0x43bfa0 (case 4): when the conversation
// window opens, every NPC except the dead-NPC type 12 takes the direction from
// itself to the character as its heading and turns to it. The direction is
// taken once, when the window opens; a walking NPC keeps following its path.
func (a *App) turnNPCToTalker(id uint32, e *game.Entity, n *npcActor, pos render.V3) {
	if a.shop.npcID != id {
		n.talking = false
	} else if !n.talking {
		n.talking = true
		if e.NpcType != 12 {
			dir := render.V3{a.state.Self.X - pos[0], 0, a.state.Self.Z - pos[2]}
			if dir.Len() > .001 {
				n.lookAt = dir.Norm()
			}
		}
	}
	if n.lookAt.Len() == 0 {
		return
	}
	cur, want := math.Atan2(float64(n.facing[0]), float64(n.facing[2])), math.Atan2(float64(n.lookAt[0]), float64(n.lookAt[2]))
	diff := math.Remainder(want-cur, 2*math.Pi)
	step := npcTurnSpeed * a.frameDT
	if math.Abs(diff) <= step {
		n.facing, n.lookAt = n.lookAt, render.V3{}
		return
	}
	cur += math.Copysign(step, diff)
	n.facing = render.V3{float32(math.Sin(cur)), 0, float32(math.Cos(cur))}
}
