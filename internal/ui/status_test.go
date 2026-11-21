package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestStatusDraftLimitsCancellationAndWireOrder(t *testing.T) {
	v, err := NewStatus(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	state := StatusState{Points: 3, Base: proto.StatPoints{65535, 10, 10, 10, 10, 10}}
	click := func(x, y int) bool {
		t.Helper()
		v.Update(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768, state, 0)
		apply, _ := v.Update(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768, state, 0)
		return apply
	}
	x, y := StatusOrigin(1024, 768, false)
	click(x+1, y+157)
	if !v.Expanded {
		t.Fatal("allocation tab did not expand")
	}
	x, y = StatusOrigin(1024, 768, true)
	click(x+86, y+129) // STR saturated: no point may be wasted.
	if v.Draft.Total() != 0 {
		t.Fatal("allocation overflow")
	}
	click(x+86, y+149) // REC row -> wire index 2.
	click(x+86, y+209) // DEX row -> wire index 1.
	click(x+86, y+209)
	click(x+86, y+209)
	if v.Draft != (proto.StatPoints{0, 2, 1, 0, 0, 0}) || state.Points != 3 || state.Base[1] != 10 {
		t.Fatal("draft/order/budget changed confirmed state", v.Draft, state)
	}
	state.Busy = true
	if click(x+16, y+270) {
		t.Fatal("duplicate apply while pending")
	}
	state.Busy = false
	if !click(x+16, y+270) {
		t.Fatal("valid allocation did not apply")
	}
	click(x+81, y+270)
	if v.Expanded || v.Draft.Total() != 0 {
		t.Fatal("cancel retained unconfirmed points")
	}
}

// 0x5037f0: the side tab only opens the sheet; Apply and Cancel fold it.
func TestStatusTabOnlyOpensAndApplyFoldsSheet(t *testing.T) {
	v, err := NewStatus(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	state := StatusState{Points: 3, Base: proto.StatPoints{10, 10, 10, 10, 10, 10}}
	click := func(x, y int) bool {
		v.Update(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768, state, 0)
		apply, _ := v.Update(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768, state, 0)
		return apply
	}
	x, y := StatusOrigin(1024, 768, false)
	click(x+1, y+157)
	if !v.Expanded {
		t.Fatal("tab did not open")
	}
	x, y = StatusOrigin(1024, 768, true)
	click(x+132+1, y+157) // the tab is hidden while open
	if !v.Expanded {
		t.Fatal("tab folded the open sheet")
	}
	if !click(x+16, y+270) {
		t.Fatal("empty Apply must still be accepted")
	}
}

func TestStatusHoldRepeatsPlusAtTenPointsPerSecond(t *testing.T) {
	v, err := NewStatus(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	v.Expanded = true
	x, y := StatusOrigin(1024, 768, true)
	state := StatusState{Points: 100}
	held := platform.Input{MouseX: x + 86, MouseY: y + 129, MouseDown: true}
	press := func() {
		in := held
		in.Pressed = true
		v.Update(&in, 1024, 768, state, 0)
	}
	release := func() {
		in := held
		in.MouseDown, in.Clicked = false, true
		v.Update(&in, 1024, 768, state, 0)
	}

	press()
	v.Update(&held, 1024, 768, state, .29)
	if v.Draft[0] != 0 {
		t.Fatal("repeat started before hold delay")
	}
	release()
	if v.Draft[0] != 1 {
		t.Fatal("short tap must add exactly one point")
	}

	press()
	v.Update(&held, 1024, 768, state, .3)
	start := v.Draft[0]
	if start != 2 {
		t.Fatal("missing first automatic point", start)
	}
	for range 60 {
		v.Update(&held, 1024, 768, state, 1.0/60)
	}
	release()
	if v.Draft[0] != start+10 {
		t.Fatalf("expected ten repeats in one second and zero on release, gained %d", v.Draft[0]-start)
	}

	press()
	outside := held
	outside.MouseX -= 50
	v.Update(&outside, 1024, 768, state, .5)
	v.Update(&held, 1024, 768, state, .5)
	release()
	if v.Draft[0] != start+10 {
		t.Fatal("repeat resumed after leaving the original button")
	}

	state.Base[0] = 65534 - v.Draft[0]
	press()
	v.Update(&held, 1024, 768, state, .8)
	release()
	if v.Draft[0] != start+11 {
		t.Fatal("repeat exceeded attribute limit", v.Draft[0])
	}
	state.Base[0] = 0
	state.Points = v.Draft[0] + 2
	press()
	v.Update(&held, 1024, 768, state, .8)
	release()
	if v.Draft[0] != state.Points {
		t.Fatal("repeat exceeded available points", v.Draft[0])
	}
}
