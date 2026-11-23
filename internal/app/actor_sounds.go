package app

import "github.com/Spelunkern/Shaiya-GX/internal/scene"

// actorSoundMemory remembers what an actor's last frame looked like so edges
// (sitting down, standing up) play their cue once.
type actorSoundMemory struct {
	known, sitting bool
}

// updateActorSounds is the character sound-event table of Game.exe 0x41b300 and
// the state handler 0x560e30, for self and every visible player:
//   - the evasion rolls (ALT 13, 14, 15) play ch_mback001 as frame 5 passes;
//   - swimming (clip 007) plays ch_mswim001 each time frame 1 passes;
//   - sitting down plays ch_mdown001 and standing up ch_mstup001.
//
// Dead, mounted and not-yet-known actors stay silent.
func (a *App) updateActorSounds(mem *actorSoundMemory, c *scene.Character, x, z float32, sitting, dead, mounted bool) {
	if c == nil || mem == nil || a.sound == nil {
		return
	}
	if dead || mounted {
		mem.known, mem.sitting = false, sitting
		return
	}
	if mem.known && sitting != mem.sitting {
		if sitting {
			a.playSound("ch_mdown001.ogg", x, z)
		} else {
			a.playSound("ch_mstup001.ogg", x, z)
		}
	}
	mem.known, mem.sitting = true, sitting
	switch slot := c.ActionSlot(); {
	case slot >= 13 && slot <= 15:
		if c.CrossedFrame(5) {
			a.playSound("ch_mback001.ogg", x, z)
		}
	case c.PlayingLocomotionClip("007_swim"):
		if c.CrossedFrame(1) {
			a.playSound("ch_mswim001.ogg", x, z)
		}
	}
}
