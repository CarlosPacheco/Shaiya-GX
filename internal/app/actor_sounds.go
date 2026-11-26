package app

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"

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

// footstepFile selects stone on dungeon floors and solid objects. Water
// requires contact with the map's water surface; dry steps use the terrain layer.
func footstepFile(w *scene.World, x, y, z float32, object bool) string {
	if w == nil || w.WLD == nil {
		return ""
	}
	prefix := "ch_walks"
	switch {
	case w.WLD.Dungeon || object:
	case w.WaterFootstepAt(x, y, z):
		prefix = "ch_walkw"
	default:
		layer := terrainWalkSound(w.WLD, x, z)
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

// objectSurfaceTolerance is the clearance (0x6bca18, 0.3) above the terrain at
// which the feet count as resting on an object.
const objectSurfaceTolerance = .3

// onObjectSurface reports feet that stand on an object (bridge, stairs, floor of
// a building) rather than the terrain. Game.exe 0x40f700 gives that contact the
// stone step instead of the terrain layer's own sound (flag +0x268, raised by
// the collision pass).
func (a *App) onObjectSurface(x, y, z float32) bool {
	return a.wv != nil && a.wv.world != nil && y-a.wv.world.Height(x, z) > objectSurfaceTolerance
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
				if name := footstepFile(a.wv.world, x, y, z, a.onObjectSurface(x, y, z)); name != "" {
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

// mountStepFile is Game.exe 0x40f7e0: the mount type follows the rider's race
// (horse, puma twice, basilisk); a dungeon is stone, wading water is "wa",
// otherwise the terrain layer's own footstep file decides: ch_walks is stone,
// ch_walkg grass and anything else earth. The variant is random.
func mountStepFile(w *scene.World, race uint8, x, y, z float32, object bool) string {
	if w == nil || w.WLD == nil || race > 3 {
		return ""
	}
	kind := [4]string{"horse", "puma", "puma", "baslk"}[race]
	surface := "st"
	switch {
	case w.WLD.Dungeon || object:
	case w.WaterFootstepAt(x, y, z):
		surface = "wa"
	default:
		layer := terrainWalkSound(w.WLD, x, z)
		switch {
		case layer == "":
			return ""
		case strings.HasPrefix(layer, "ch_walks"):
		case strings.HasPrefix(layer, "ch_walkg"):
			surface = "gr"
		default:
			surface = "ea"
		}
	}
	return fmt.Sprintf("run_%s_%s%02d.ogg", kind, surface, 1+rand.IntN(3))
}

// updateMountSounds plays the hoof cue when a mount's animation passes one of
// its step frames. prevAction and prevTime are StepCursor before the update.
func (a *App) updateMountSounds(m *scene.Vehicle, prevAction int, prevTime float32, race uint8, x, y, z float32) {
	if m == nil || a.sound == nil || a.wv == nil || a.wv.world == nil {
		return
	}
	action, now := m.StepCursor()
	if !scene.MountStepCrossedDirected(prevAction, prevTime, action, now, race, m.PlayingReverse()) {
		return
	}
	if name := mountStepFile(a.wv.world, race, x, y, z, a.onObjectSurface(x, y, z)); name != "" {
		a.playSound(name, x, z)
	}
}
