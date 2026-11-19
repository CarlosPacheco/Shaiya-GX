package world

import (
	"bytes"
	"math"
	"testing"
)

func TestDungeonSkyPreservesNativeFooter(t *testing.T) {
	// Empty layout, placements, sounds, NPCs and environment.
	base := make([]byte, 688)
	copy(base, "DUN")
	patched, err := PatchDungeonSky(base, "Sky_A1.bmp", "clouds01_A1.tga", "")
	if err != nil {
		t.Fatal(err)
	}
	w, err := ParseWLD(patched)
	if err != nil {
		t.Fatal(err)
	}
	if !w.Dungeon || w.Sky != "Sky_A1.bmp" || w.CloudA != "clouds01_A1.tga" || w.CloudB != "" ||
		!bytes.Equal(patched[:len(base)], base) {
		t.Fatalf("unexpected environment or changed base data: %+v", w)
	}
	removed, err := PatchDungeonSky(patched, "", "", "")
	if err != nil || !bytes.Equal(removed, base) {
		t.Fatal("cannot remove extension without altering legacy data", err)
	}
	if _, err := ParseWLD(patched[:len(patched)-1]); err == nil {
		t.Fatal("accepted truncated GXSK")
	}
	if _, err := PatchDungeonSky(base, "../sky.bmp", "", ""); err == nil {
		t.Fatal("accepted sky outside the Sky folder")
	}
}

func TestDungeonHasFogOnlyWithAuthoredValidValues(t *testing.T) {
	w := WLD{Dungeon: true, FogColor: [3]float32{128, 128, 128}, FogStart: 150, FogEnd: 200}
	if w.HasDungeonFog() {
		t.Fatal("stock dungeon placeholder enabled fog")
	}
	w.FogStart, w.FogEnd = 200, 300
	if !w.HasDungeonFog() {
		t.Fatal("authored dungeon range ignored")
	}
	w.FogColor[1] = float32(math.NaN())
	if w.HasDungeonFog() {
		t.Fatal("invalid fog accepted")
	}
}
