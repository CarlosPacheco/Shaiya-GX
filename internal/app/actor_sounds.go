package app

import (
	"fmt"
	"math/rand/v2"
	"regexp"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// stepFrames are the animation frames at which a character's foot lands, from
// the table at Game.exe 0x41bb1a..0x41bc55: walk 8 and 24, run 9 and 20 (also
// every weapon run), backstep 11 and 28, left/right step 11 and 25.
var stepFrames = map[string][]float32{
	"001_walk":  {8, 24},
	"002_run":   {9, 20},
	"003_bstep": {11, 28},
	"004_lstep": {11, 25},
	"005_rstep": {11, 25},
}

var stepVariant = regexp.MustCompile(`^(.*?)\d+\.\w+$`)

// footstepFile is Game.exe 0x40f700: a dungeon floor is stone, a walk below
// 0.9 units is wading water, otherwise the terrain layer under the foot picks
// the surface. The variant (001..003) is random.
func footstepFile(w *world.WLD, x, y, z float32) string {
	if w == nil {
		return ""
	}
	prefix := "ch_walks"
	switch {
	case w.Dungeon:
	case y-0.9 < 0:
		prefix = "ch_walkw"
	default:
		layer := terrainWalkSound(w, x, z)
		if layer == "" {
			return ""
		}
		prefix = layer
		if m := stepVariant.FindStringSubmatch(layer); m != nil {
			prefix = m[1]
		}
	}
	return fmt.Sprintf("%s%03d.ogg", prefix, 1+rand.IntN(3))
}

// actorSoundMemory remembers what an actor's last frame looked like so edges
// (sitting down, standing up) play their cue once.
type actorSoundMemory struct {
	known, sitting bool
}

// updateActorSounds is the character sound-event table of Game.exe 0x41b300 and
// the state handler 0x560e30, for self and every visible player:
//   - the evasion rolls (ALT 13, 14, 15) play ch_mback001 as frame 5 passes;
//   - swimming (clip 007) plays ch_mswim001 each time frame 1 passes;
//   - footsteps on the frames of stepFrames;
//   - sitting down plays ch_mdown001 and standing up ch_mstup001.
//
// Dead, mounted and not-yet-known actors stay silent.
func (a *App) updateActorSounds(mem *actorSoundMemory, c *scene.Character, x, y, z float32, sitting, dead, mounted bool) {
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
	if frames, ok := stepFrames[c.LocomotionClip()]; ok && c.ActionSlot() < 0 && !c.Concealed && a.wv != nil && a.wv.world != nil {
		for _, frame := range frames {
			if c.CrossedFrame(frame) {
				if name := footstepFile(a.wv.world.WLD, x, y, z); name != "" {
					a.playSound(name, x, z)
				}
				break
			}
		}
	}
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
