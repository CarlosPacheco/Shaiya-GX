package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Movement input is shared by keyboard travel, mouse-run and autorun.
type movementControls struct {
	forward, back bool
	side, turn    int
}

func characterControls(in *platform.Input, autoRun bool) movementControls {
	if in.SuppressAutoRun {
		return movementControls{}
	}
	c := movementControls{forward: in.Held('W') || autoRun || in.MouseDown && in.RightDown, back: in.Held('S')}
	if in.Held('Q') {
		c.side--
	}
	if in.Held('E') {
		c.side++
	}
	// EP5.4 0x43f064 skips the backward step after processing a strafe.
	if c.side != 0 && !c.forward {
		c.back = false
	}
	if in.Held('A') {
		c.turn++
	}
	if in.Held('D') {
		c.turn--
	}
	return c
}

func cameraHeading(view, fallback render.V3) render.V3 {
	view[1] = 0
	if view.Len() < .0001 {
		return fallback
	}
	return view.Norm()
}

func controlVector(heading, right render.V3, c movementControls) render.V3 {
	dir := render.V3{}
	if c.forward {
		dir = dir.Add(heading)
	}
	if c.back {
		dir = dir.Sub(heading)
	}
	dir = dir.Add(right.Scale(float32(c.side)))
	return dir.Norm()
}

// Phoenix keeps combat facing separate from the heading resumed by controls.
func (v *worldView) faceCombatTarget(direction render.V3) {
	if v.combatHeading.Len() == 0 {
		v.combatHeading = v.facing
	}
	v.facing = direction
	v.visualFacing = render.V3{}
}

func (v *worldView) restoreMovementHeading() {
	if v.combatHeading.Len() > 0 {
		v.facing = v.combatHeading
		v.combatHeading = render.V3{}
	}
}
