package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestRevolverPageControlsAndProfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	a := inputApp()
	a.state.SelfID = 12
	v := a.revolverState()
	const w, h = 1024, 768
	a.updateRevolver(&platform.Input{Keys: []platform.Key{platform.KeyRevolverPage4}}, w, h)
	click := func(control int) {
		box := a.revolverLayout(w, h).Controls()[control]
		in := platform.Input{MouseX: box.X + 1, MouseY: box.Y + 1, Pressed: true, MouseDown: true}
		if !a.updateRevolver(&in, w, h) {
			t.Fatal("control did not capture press", control)
		}
		in.Pressed, in.MouseDown, in.Clicked = false, false, true
		if !a.updateRevolver(&in, w, h) {
			t.Fatal("control did not capture release", control)
		}
	}
	click(2)
	for range 5 {
		click(5)
	}
	if v.Page != 3 || !v.Expanded || v.count() != 6 {
		t.Fatal("revolver page or slot limits incorrect", v.Page, v.Expanded, v.count())
	}
	cell := a.revolverLayout(w, h).Cell(5, true)
	if !a.revolverDrop(&platform.Input{MouseX: cell.X + 1, MouseY: cell.Y + 1}, w, h, revolverSlot{Skill: 57}) {
		t.Fatal("drop missed the last cell")
	}
	v.step(1)
	a.updateRevolver(&platform.Input{Keys: []platform.Key{platform.KeyRevolverReset}}, w, h)
	if v.Page != 3 || v.selected != 0 {
		t.Fatal("reset changed the revolver page or failed to select its first cell")
	}
	click(7)
	if v.Page != 0 {
		t.Fatal("clicking page four should wrap to page one")
	}
	click(7)
	click(7)
	click(7)
	click(0)
	if !v.Position.Vertical {
		t.Fatal("orientation control did not rotate")
	}
	b := inputApp()
	b.state.SelfID = 12
	got := b.revolverState()
	if got.Page != 3 || !got.Expanded || got.count() != 6 || got.Slots[3][5].Skill != 57 {
		t.Fatal("revolver layout and binding did not reload")
	}
	b.state.SelfID = 13
	if got = b.revolverState(); got.Page != 0 || got.Slots[3][5].Skill != 0 {
		t.Fatal("another character inherited the revolver configuration")
	}
}

func TestRevolverAdvancesOnlyOnceOnMatchingResult(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 90
	a.clock = 10
	v := &a.revolver
	v.Slots[0][0] = revolverSlot{Skill: 57}
	v.cast = revolverCast{skill: sdata.SkillKey{ID: 57, Level: 2}, deadline: 15}
	a.confirmRevolver(proto.SkillResult{ID: 57, Level: 2, Source: 91})
	if v.selected != 0 {
		t.Fatal("another caster advanced the revolver")
	}
	result := proto.SkillResult{ID: 57, Level: 2, Source: 90}
	a.confirmRevolver(result)
	a.confirmRevolver(result)
	if v.selected != 1 {
		t.Fatal("expected one advance after the matching result", v.selected)
	}
}
