package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"strings"
	"testing"
)

func TestSpearLocomotionRetainsOtherMovementModes(t *testing.T) {
	c := &Character{folder: "Human", prefix: "humf", weaponType: 51, actionsLoaded: true, actions: make([]model.Action, 85)}
	c.actions[48].Name = "humf_048_spready.ANI"
	c.actions[54].Name = "humf_054_sprun.ANI"
	if !strings.HasSuffix(c.locomotionFile("002_run"), "054_sprun.ANI") {
		t.Fatal("spear running missing")
	}
	if c.locomotionFile("000_normal") != "" {
		t.Fatal("ready stance outside combat")
	}
	c.combatPoseLeft = 10
	if !strings.HasSuffix(c.locomotionFile("000_normal"), "048_spready.ANI") {
		t.Fatal("spear ready missing")
	}
	for _, name := range []string{"001_walk", "003_back", "006_swnormal", "007_swrun", "021_veh_br"} {
		if c.locomotionFile(name) != "" {
			t.Fatalf("weapon replaced %s", name)
		}
	}
	c.Mounted = true
	if c.locomotionFile("002_run") != "" {
		t.Fatal("weapon replaced mount")
	}
	c.Mounted = false
	c.actions[54].Name = "humf_select.ANI"
	if c.locomotionFile("002_run") != "" {
		t.Fatal("invalid weapon pose used")
	}
}
