package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
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

func TestRevolverKeepsOneOrderThroughLandingAndConfirmation(t *testing.T) {
	a := inputApp()
	a.state.SelfID, a.clock = 90, 10
	a.world = &session.World{}
	a.revolver.profile = a.localActionsKey()
	key := sdata.SkillKey{ID: 57, Level: 2}
	a.state.Skills[1] = proto.LearnedSkill{ID: key.ID, Level: key.Level, Number: 1}
	a.skillDefs[key] = sdata.Skill{Effect: 4, Target: 2}
	a.revolver.Slots[0][0] = revolverSlot{Skill: key.ID}
	press := func() { a.updateRevolver(&platform.Input{Keys: []platform.Key{platform.KeyRevolver}}, 1024, 768) }

	a.wv.jumping = true
	press()
	first := a.wv.landingAction.packet
	if len(first) == 0 || a.revolver.cast.deadline != 0 {
		t.Fatal("the first press should wait for landing")
	}
	for range 10 {
		press()
		if got := a.wv.landingAction.packet; len(got) == 0 || &got[0] != &first[0] {
			t.Fatal("another press replaced the pending landing order")
		}
	}

	a.wv.landingAction = landingAction{}
	a.wv.jumping = false
	if !a.prepareSkillMovement(first) || !a.revolverPending() {
		t.Fatal("the landing order should stay pending once ready to send")
	}
	press()
	if a.state == nil || a.skills.queuedSkill != nil || a.revolver.selected != 0 {
		t.Fatal("another press queued a duplicate before the result")
	}

	r := proto.SkillResult{ID: key.ID, Level: key.Level, Source: a.state.SelfID, Target: a.state.SelfID}
	a.confirmRevolver(r)
	a.confirmRevolver(r)
	if a.revolver.selected != 1 || a.revolverPending() {
		t.Fatal("the confirmation should advance one cell and release the order")
	}
}

func TestRevolverReleasesCancelledOrRejectedOrder(t *testing.T) {
	a := inputApp()
	a.state.SelfID, a.clock = 90, 12
	key := sdata.SkillKey{ID: 57, Level: 2}
	order := revolverCast{skill: key, packet: proto.UseSkillPacket(1, 90, true)}
	a.revolver.cast = order
	a.skills.queuedSkill = &queuedSkill{packet: order.packet}
	if !a.revolverPending() {
		t.Fatal("a locally queued order should reserve the slot")
	}
	a.updateQueuedSkill(&platform.Input{Down: "W"})
	if a.revolverPending() {
		t.Fatal("a cancelled queued order still reserved the slot")
	}

	a.revolver.cast = order
	a.markRevolverSent(order.packet)
	a.state.LastSkillFailure = proto.SkillFailure{Skill: key.ID + 1}
	a.confirmedSkills(proto.GXSkillFailure)
	if !a.revolverPending() {
		t.Fatal("another skill's refusal released the order")
	}
	a.state.LastSkillFailure.Skill = key.ID
	a.confirmedSkills(proto.GXSkillFailure)
	if a.revolverPending() || a.revolver.selected != 0 {
		t.Fatal("a refusal should release the order and keep the selected cell")
	}

	a.revolver.cast = order
	a.markRevolverSent(order.packet)
	a.state.LastCast = proto.SkillCast{Source: 90, ID: key.ID, Level: key.Level}
	a.confirmedSkills(proto.SkillCastStarted)
	a.clock += 4
	if !a.revolverPending() || a.revolver.selected != 0 {
		t.Fatal("an announced preparation should wait for its result")
	}
	a.clock += 15
	if a.revolverPending() {
		t.Fatal("a missing result should eventually release the order")
	}
}

func TestRevolverCompactDropsPersistAcrossReload(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	a := inputApp()
	a.state.SelfID = 42
	v := a.revolverState()
	v.Page, v.Counts[1] = 1, 6
	const w, h = 1024, 768
	for i := range 3 {
		v.Position.Vertical = i == 2
		cell := a.revolverLayout(w, h).Cell(i, false)
		slot := revolverSlot{Skill: uint16(57 + i)}
		if !a.revolverDrop(&platform.Input{MouseX: cell.X + 16, MouseY: cell.Y + 16}, w, h, slot) {
			t.Fatalf("compact cell %d rejected its drop", i)
		}
	}
	b := inputApp()
	b.state.SelfID = 42
	saved := b.revolverState()
	if saved.Expanded || saved.Page != 1 || saved.Slots[1][5].Skill != 57 ||
		saved.Slots[1][0].Skill != 58 || saved.Slots[1][1].Skill != 59 {
		t.Fatalf("compact assignments did not survive reload: %+v", saved.revolverConfig)
	}
}

func TestRevolverControlHitsMatchVerticalArtwork(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	a := inputApp()
	a.state.SelfID = 12
	v := a.revolverState()
	const w, h = 1024, 768
	center := func() [2]int {
		main := a.revolverLayout(w, h).Main
		return [2]int{main.X + main.W/2, main.Y + main.H/2}
	}
	click := func(dx, dy int) {
		t.Helper()
		main := a.revolverLayout(w, h).Main
		in := platform.Input{MouseX: main.X + dx, MouseY: main.Y + dy, Pressed: true, MouseDown: true}
		if !a.updateRevolver(&in, w, h) {
			t.Fatalf("press at %d,%d was not captured", dx, dy)
		}
		in.Pressed, in.MouseDown, in.Clicked = false, false, true
		if !a.updateRevolver(&in, w, h) {
			t.Fatalf("release at %d,%d was not captured", dx, dy)
		}
	}
	for _, expanded := range []bool{false, true} {
		v.Expanded = expanded
		panel := a.revolverLayout(w, h).Panel
		v.Position.Move(400, 240, w, h, panel.W, panel.H)
		before := center()
		click(146, 19) // Horizontal circular arrow.
		if !v.Position.Vertical || v.Expanded != expanded {
			t.Fatalf("horizontal rotate failed (editor=%t)", expanded)
		}
		if got := center(); got != before {
			t.Fatalf("horizontal rotate moved the bar center from %v to %v", before, got)
		}
		v.selected = 2
		click(47, 24) // Advance.
		if v.selected != 0 {
			t.Fatal("vertical advance did not wrap")
		}
		click(47, 132) // Go back.
		if v.selected != 2 {
			t.Fatal("vertical back did not wrap")
		}
		click(25, 146) // Vertical R button.
		if v.selected != 0 || !v.Position.Vertical || v.Expanded != expanded {
			t.Fatalf("vertical reset activated the wrong control (editor=%t)", expanded)
		}
		click(14, 146) // Vertical editor button.
		if v.Expanded == expanded || !v.Position.Vertical {
			t.Fatalf("vertical editor activated the wrong control (editor=%t)", expanded)
		}
		before = center()
		click(36, 146) // Vertical circular arrow.
		if v.Position.Vertical || v.Expanded == expanded {
			t.Fatalf("vertical rotate activated the wrong control (editor=%t)", expanded)
		}
		if got := center(); got != before {
			t.Fatalf("vertical rotate moved the bar center from %v to %v", before, got)
		}
	}
}
