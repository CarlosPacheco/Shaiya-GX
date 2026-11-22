package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestAppearanceEditorCyclesAndRestrictsSex(t *testing.T) {
	e := &AppearanceEditor{}
	for i := range e.arrows {
		e.arrows[i] = Button{OffX: 0x61 + i%2*(0xbd-0x61), OffY: []int{0x13f, 0x160, 0x182}[i/2], HitW: 23, HitH: 23}
	}
	for i := range e.sexes {
		e.sexes[i] = Button{OffX: 70 + i*78, OffY: 273, HitW: 68, HitH: 22, Toggle: true}
	}
	click := func(b *Button) {
		x, y := AppearanceOrigin(800, 600)
		x += b.OffX + 1
		y += b.OffY + 1
		e.Update(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 800, 600)
		e.Update(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 800, 600)
	}
	e.Open([4]uint8{}, false)
	click(&e.arrows[0])
	if e.Values[0] != 4 {
		t.Fatal("hair did not wrap backward")
	}
	click(&e.arrows[1])
	if e.Values[0] != 0 {
		t.Fatal("hair did not wrap forward")
	}
	click(&e.arrows[5])
	if e.Values[2] != 1 {
		t.Fatal("height did not advance")
	}
	click(&e.sexes[1])
	if e.Values[3] != 0 {
		t.Fatal("sex changed on appearance-only item")
	}
	e.Open([4]uint8{}, true)
	click(&e.sexes[1])
	if e.Values[3] != 1 {
		t.Fatal("sex change item did not switch sex")
	}
}
