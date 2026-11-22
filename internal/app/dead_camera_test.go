package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestDeadCharacterKeepsOnlyCameraControls(t *testing.T) {
	a := inputApp()
	in := &platform.Input{MouseX: 5, MouseY: 6, RightDown: true, MouseDown: true, Wheel: 120,
		Down: "WASDQE<>^v+-", Keys: []platform.Key{platform.KeySpace, platform.KeyF9, platform.KeyHome, platform.KeyTab}}
	got := a.deadCameraInput(in)
	for _, k := range "WASDQE" {
		if got.Held(k) {
			t.Fatalf("dead character received movement key %c", k)
		}
	}
	for _, k := range "<>^v+-" {
		if !got.Held(k) {
			t.Fatalf("camera key %c lost", k)
		}
	}
	if !got.RightDown || got.Wheel != 120 || got.MouseDown || got.HasKey(platform.KeySpace) || got.HasKey(platform.KeyTab) || !got.HasKey(platform.KeyF9) || !got.HasKey(platform.KeyHome) {
		t.Fatalf("unexpected dead input: %+v", got)
	}
}
