package scene

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"path"
	"strings"
)

// Game.exe 0x4cb9a0 (running) and 0x4cb8c0 (ready stance).
// Walking, strafing, swimming, mounts and posture retain their ordinary slots.
var readySlots = [...]int{0, 34, 23, 34, 23, 41, 48, 34, 23, 64, 78, 55, 59, 30, 55, 71}

func weaponLocomotionSlot(weapon uint8, running bool) int {
	k := weaponKind(weapon)
	if k == 0 {
		return -1
	}
	ready := readySlots
	run := [...]int{2, 40, 29, 40, 29, 47, 54, 40, 29, 70, 84, 58, 63, 33, 58, 77}
	if running {
		return run[k]
	}
	return ready[k]
}
// isReadySlot reports whether an ALT slot is one of the weapon ready stances.
func isReadySlot(slot int) bool {
	for _, s := range readySlots {
		if s != 0 && s == slot {
			return true
		}
	}
	return false
}

func (c *Character) loadActions() {
	if c.actionsLoaded {
		return
	}
	c.actionsLoaded = true
	file := fmt.Sprintf("Character/%s/%s_action.alt", c.folder, c.prefix)
	b, err := c.assets.Arc.ReadFile(file)
	if err == nil {
		c.actions, err = model.ParseALT(b)
	}
	if err != nil {
		c.assets.Missing[err.Error()] = true
	}
}
func (c *Character) locomotionFile(name string) string {
	if c.Mounted || c.weaponType == 0 || name != "002_run" && (name != "000_normal" || c.combatPoseLeft <= 0) {
		return ""
	}
	c.loadActions()
	slot := weaponLocomotionSlot(c.weaponType, name == "002_run")
	if slot < 0 || slot >= len(c.actions) || c.actions[slot].Name == "" {
		return ""
	}
	file := path.Base(c.actions[slot].Name)
	want := "ready"
	if name == "002_run" {
		want = "run"
	}
	// Invalid class/weapon combinations contain select-pose placeholders.
	if !strings.Contains(strings.ToLower(file), want) {
		return ""
	}
	return fmt.Sprintf("Character/%s/ANI/%s", c.folder, file)
}
