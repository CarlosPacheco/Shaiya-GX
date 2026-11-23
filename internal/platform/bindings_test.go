package platform

import (
	"testing"

	"github.com/go-gl/glfw/v3.4/glfw"
)

func TestBindingsRemapMovementPanelAndExtraBar(t *testing.T) {
	var bindings KeyBindings
	if !bindings.Bind(2, glfw.KeyJ) || !bindings.Bind(10, glfw.KeyKP1) || !bindings.Valid() {
		t.Fatal("configured movement or panel/two-way binding rejected")
	}
	in := Input{
		physical:    true,
		Down:        "WJT",
		Chars:       []rune("jwt1"),
		charKeys:    []PhysicalKey{glfw.KeyJ, glfw.KeyW, glfw.KeyT, glfw.KeyKP1},
		KeyPresses:  []PhysicalKey{glfw.KeyW, glfw.KeyT, glfw.KeyKP1},
		keyReleases: []PhysicalKey{glfw.KeyJ},
		heldKeys:    bindingKeyBit(glfw.KeyW) | bindingKeyBit(glfw.KeyJ) | bindingKeyBit(glfw.KeyT),
		ActionHeld:  1<<10 | 1<<20,
		Actions:     []int{10, 20},
		KeypadChars: "1",
	}
	got := in.WithBindings(bindings)
	if got.Down != "W" || got.Released != "W" || string(got.Chars) != "t" || got.KeypadChars != "" ||
		got.ActionHeld != 1<<10|1<<20 || len(got.Actions) != 2 || got.Actions[0] != 20 || got.Actions[1] != 10 {
		t.Fatalf("wrong mapped gameplay input: %+v", got)
	}
	if string(in.Chars) != "jwt1" || bindings.Bind(2, glfw.KeyH) {
		t.Fatal("raw text or the debug key was changed")
	}
}

func TestPersonalLightUsesUnboundLPress(t *testing.T) {
	in := Input{physical: true, KeyPresses: []PhysicalKey{glfw.KeyL}, heldKeys: bindingKeyBit(glfw.KeyL)}
	var bindings KeyBindings
	if controls := in.WithBindings(bindings); !controls.HasKey(KeyPersonalLight) {
		t.Fatal("L press did not toggle the light")
	}
	if !bindings.Bind(2, glfw.KeyL) {
		t.Fatal("could not remap movement to L")
	}
	if controls := in.WithBindings(bindings); controls.HasKey(KeyPersonalLight) || controls.Down != "W" {
		t.Fatalf("light interfered with movement remapped to L: %+v", controls)
	}
	in.KeyPresses = nil
	if controls := in.WithBindings(KeyBindings{}); controls.HasKey(KeyPersonalLight) {
		t.Fatal("holding L toggled the light again")
	}
}
