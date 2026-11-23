package app

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestPickupRejectionsReachSystemChat(t *testing.T) {
	a := inputApp()
	for code, want := range map[byte]string{
		0: "That item is reserved for another player.",
		1: "Your inventory is full.",
		2: "You do not need that quest item.",
		3: "That item is no longer available.",
		4: "That item is too far away.",
		9: "Unable to pick up that item.",
	} {
		p := proto.NewWriter(proto.AddItem)
		p.U8(0)
		p.U8(code)
		a.confirmedPickup(p.Bytes())
		if got := a.state.Chat[len(a.state.Chat)-1].Text; got != want {
			t.Errorf("reason %d: %q, want %q", code, got, want)
		}
	}
	before := len(a.state.Chat)
	a.confirmedPickup(proto.PickupPacket(42))
	if len(a.state.Chat) != before {
		t.Fatal("success-shaped packet produced a rejection")
	}
}

func TestMobPickingUsesLiveMobsAndVisibleProjection(t *testing.T) {
	a := inputApp()
	a.scene3d = &render.Scene3D{
		View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}),
		Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100),
	}
	a.state.Entities = map[uint32]*game.Entity{
		1: {ID: 1, Kind: game.KindMob},
		2: {ID: 2, Kind: game.KindMob},
		3: {ID: 3, Kind: game.KindMob, Dead: true},
		4: {ID: 4, Kind: game.KindUser},
		5: {ID: 5, Kind: game.KindMob, Z: -10},
	}
	click := platform.Input{Clicked: true, MouseX: 256, MouseY: 256}
	if target := a.pickMob(&click, 512, 512); target != 1 {
		t.Fatal("overlapping mobs did not pick deterministically", target)
	}
	delete(a.state.Entities, 1)
	delete(a.state.Entities, 2)
	if target := a.pickMob(&click, 512, 512); target != 0 {
		t.Fatal("picked a dead/non-mob/behind-camera entity", target)
	}
	a.scene3d = nil
	if target := a.pickMob(&platform.Input{}, 512, 512); target != 5 {
		t.Fatal("keyboard nearest target incorrectly requires a renderer", target)
	}
}

func TestHUDClickBlocksWorld(t *testing.T) {
	for _, point := range [][2]int{{40, 50}, {300, 30}, {717, 715}} {
		a := inputApp()
		a.state.Entities[1] = &game.Entity{ID: 1, Kind: game.KindMob}
		a.state.Target = 1
		controls := a.worldControls(&platform.Input{MouseX: point[0], MouseY: point[1], Pressed: true, MouseDown: true, Down: "W"}, 1024, 768)
		if controls.Pressed || controls.MouseDown || controls.Down != "W" {
			t.Fatal("HUD press reached world/stopped movement", point, controls)
		}
		controls = a.worldControls(&platform.Input{MouseX: 512, MouseY: 400, Clicked: true}, 1024, 768)
		if controls.Clicked {
			t.Fatal("HUD press released over world selected a mob", point)
		}
	}
}

func TestPlayerCanBeSelectedWithoutMobRequest(t *testing.T) {
	a := inputApp()
	a.scene3d = &render.Scene3D{View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}), Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100)}
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindUser, Shape: &proto.UserShape{Name: "Ally"}}
	click := &platform.Input{Clicked: true, MouseX: 256, MouseY: 256}
	a.updateCombat(click, 512, 512)
	if a.state.Target != 42 {
		t.Fatalf("player not selected: %d", a.state.Target)
	}
	if got := a.pickMob(&platform.Input{}, 512, 512); got != 0 {
		t.Fatal("tab selected a player", got)
	}
	a.state.Entities[42].Dead = true
	a.state.Target = 0
	a.updateCombat(click, 512, 512)
	if a.state.Target != 42 {
		t.Fatal("the player's corpse could not be selected with the pointer")
	}
}

func TestPlayerPickingAndEffectsFollowDisplayedPosition(t *testing.T) {
	a := inputApp()
	a.scene3d = &render.Scene3D{View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}), Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100)}
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindUser, X: 20, Shape: &proto.UserShape{Name: "Ally"}}
	a.wv.actors = map[uint32]*remoteActor{42: {entity: a.state.Entities[42], position: remotePosition{initialized: true, pos: render.V3{0, 0, 0}}}}
	if got := a.pickMob(&platform.Input{Clicked: true, MouseX: 256, MouseY: 256}, 512, 512); got != 42 {
		t.Fatalf("displayed player not selected: %d", got)
	}
	if got, ok := a.effectPosition(42); !ok || got != (render.V3{0, 0, 0}) {
		t.Fatalf("effect position = %v, %v", got, ok)
	}
}

func TestMonsterPickingAndEffectsFollowDisplayedPosition(t *testing.T) {
	a := inputApp()
	a.scene3d = &render.Scene3D{View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}), Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100)}
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindMob, X: 20}
	a.wv.mobs = map[uint32]*mobActor{42: {entity: a.state.Entities[42], pos: render.V3{0, 0, 0}, position: remotePosition{initialized: true}}}
	if got := a.pickMob(&platform.Input{Clicked: true, MouseX: 256, MouseY: 256}, 512, 512); got != 42 {
		t.Fatalf("displayed monster not selected: %d", got)
	}
	if got, ok := a.effectPosition(42); !ok || got != (render.V3{0, 0, 0}) {
		t.Fatalf("effect position = %v, %v", got, ok)
	}
}

func TestNPCPickingUsesTheCameraFacingRectangle(t *testing.T) {
	a := inputApp()
	a.scene3d = &render.Scene3D{View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}), Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100)}
	e := &game.Entity{ID: 42, Kind: game.KindNpc}
	a.state.Entities[42] = e
	actor := &scene.CreatureActor{Character: &scene.Character{World: render.Identity()}}
	a.wv.npcs = map[uint32]*npcActor{42: {entity: e, actor: actor, position: remotePosition{initialized: true, pos: render.V3{0, 0, 0}}}}
	vp := a.scene3d.View.Mul(a.scene3d.Proj)
	click := func(p render.V3) *platform.Input {
		x, y, _, _ := projectLabel(p, vp, 512, 512)
		return &platform.Input{Clicked: true, MouseX: x, MouseY: y}
	}
	// The rectangle is 0.6 wide and as tall as the plate height (0.3 here).
	if got := a.pickNPC(click(render.V3{0, .15, 0}), 512, 512); got != 42 {
		t.Fatalf("a click on the rectangle missed the NPC: %d", got)
	}
	if got := a.pickNPC(click(render.V3{1, .15, 0}), 512, 512); got != 0 {
		t.Fatalf("a click beside the rectangle picked the NPC: %d", got)
	}
	if got := a.pickNPC(click(render.V3{0, 1.2, 0}), 512, 512); got != 0 {
		t.Fatalf("a click above the plate height picked the NPC: %d", got)
	}
	in := click(render.V3{0, .15, 0})
	in.Clicked = false
	if got := a.pickNPC(in, 512, 512); got != 0 {
		t.Fatal("hover activated NPC")
	}
}

func TestRayBoxHandlesInsideParallelAndDepth(t *testing.T) {
	lo, hi := render.V3{-1, -1, 4}, render.V3{1, 1, 6}
	if d, ok := rayBox(render.V3{}, render.V3{0, 0, 1}, lo, hi); !ok || d != 4 {
		t.Fatalf("front hit: %v %v", d, ok)
	}
	if _, ok := rayBox(render.V3{2, 0, 0}, render.V3{0, 0, 1}, lo, hi); ok {
		t.Fatal("parallel ray hit outside box")
	}
	if d, ok := rayBox(render.V3{0, 0, 5}, render.V3{0, 0, 1}, lo, hi); !ok || d != 0 {
		t.Fatal("inside origin missed")
	}
	if _, ok := rayBox(render.V3{}, render.V3{0, 0, -1}, lo, hi); ok {
		t.Fatal("box behind ray hit")
	}
}

func TestNativeTargetKeyCyclesByDistanceAndWraps(t *testing.T) {
	a := inputApp()
	a.state.Entities = map[uint32]*game.Entity{1: {ID: 1, Kind: game.KindMob, X: 2}, 2: {ID: 2, Kind: game.KindMob, X: 4}, 3: {ID: 3, Kind: game.KindMob, X: 6, Dead: true}, 4: {ID: 4, Kind: game.KindMob, X: 8}}
	for _, want := range []uint32{1, 2, 4, 1} {
		got := a.pickMob(&platform.Input{Keys: []platform.Key{platform.KeyTarget}}, 1024, 768)
		if got != want {
			t.Fatalf("target=%d want %d", got, want)
		}
		a.state.Target = got
	}
}

func TestDoubleClickKeepsApproachToMovingMonster(t *testing.T) {
	a := inputApp()
	a.scene3d = &render.Scene3D{
		View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}),
		Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100),
	}
	mob := &game.Entity{ID: 42, Kind: game.KindMob, Z: 12}
	a.state.Entities[mob.ID], a.state.Target = mob, mob.ID
	a.wv.facing = render.V3{0, 0, 1}
	for _, at := range []float64{1, 1.2} {
		a.clock = at
		in := &platform.Input{Clicked: true, MouseX: 256, MouseY: 256}
		a.updateCombat(in, 512, 512)
		a.updateActionApproach(in)
	}
	if a.wv.approachTarget != mob.ID || !a.wv.travel {
		t.Fatal("double-click did not leave the selected monster queued")
	}
	first := a.wv.destination
	mob.X, mob.Z = 5, 6
	a.updateActionApproach(&platform.Input{})
	if a.wv.destination[0] <= 3 || a.wv.destination[2] >= first[2] {
		t.Fatalf("approach kept stale destination %v after target moved to %v", a.wv.destination, mob)
	}
	a.updateActionApproach(&platform.Input{Down: "W"})
	if a.wv.approachTarget != 0 || a.wv.travel {
		t.Fatal("manual movement could not cancel the approach")
	}
}

func TestAttackApproachWaitsDuringPreviousAttackPause(t *testing.T) {
	a := inputApp()
	mob := &game.Entity{ID: 42, Kind: game.KindMob, Z: 1}
	a.state.Entities[mob.ID], a.state.Target = mob, mob.ID
	a.wv.facing = render.V3{0, 0, 1}
	a.clock, a.wv.attackMoveUntil = 1, 1.1
	if !a.attackTarget() || a.wv.approachTarget != mob.ID || a.wv.travel {
		t.Fatal("attack clicked during the movement pause was dropped")
	}
	a.updateActionApproach(&platform.Input{})
	if a.wv.approachTarget != mob.ID {
		t.Fatal("arriving during the movement pause discarded the attack")
	}
}

// Native 0x43bfa0: a click selects, a click on the selected target engages it
// with no time window, and the chase keeps following and swinging.
func TestClickOnSelectedMobEngagesWithoutTimeLimitAndChaseKeepsFollowing(t *testing.T) {
	a := inputApp()
	a.scene3d = &render.Scene3D{
		View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}),
		Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100),
	}
	mob := &game.Entity{ID: 42, Kind: game.KindMob, Z: 12}
	a.state.Entities[mob.ID] = mob
	a.wv.facing = render.V3{0, 0, 1}
	click := func(at float64) {
		a.clock = at
		in := &platform.Input{Clicked: true, MouseX: 256, MouseY: 256}
		a.updateCombat(in, 512, 512)
		a.updateActionApproach(in)
	}
	a.state.Target = mob.ID // already selected by an earlier click
	click(9)                // however long ago: there is no time window
	if a.wv.approachTarget != mob.ID || !a.wv.travel {
		t.Fatal("second click on the selected mob did not start the chase")
	}
	// Arriving and attacking leaves the chase tracking the mob.
	a.wv.pos = render.V3{0, 0, 11.5}
	mob.Z = 12
	a.wv.engagedID = mob.ID
	a.updateActionApproach(&platform.Input{})
	if a.wv.approachTarget != mob.ID || a.wv.travel {
		t.Fatal("chase must keep tracking the engaged mob without moving")
	}
	mob.Z = 30 // it runs away
	a.updateActionApproach(&platform.Input{})
	if !a.wv.travel || a.wv.engagedID != 0 {
		t.Fatal("chase did not resume after the mob left reach", a.wv.travel, a.wv.engagedID)
	}
}

func TestAttackReachFollowsWeaponAndMobSize(t *testing.T) {
	a := inputApp()
	if got := a.attackReachTo(&game.Entity{Kind: game.KindMob}); got != 1 {
		t.Fatal("bare hands reach", got)
	}
}

// Same-faction players are followed at 1.5 units without any packet
// (a nil World would panic on Send); duel and guild-duel foes are attacked.
func TestSameFactionPlayerIsFollowedWithoutPackets(t *testing.T) {
	a := inputApp()
	a.state.Faction.Faction = 1
	ally := &game.Entity{ID: 42, Kind: game.KindUser, X: 20, Shape: &proto.UserShape{Faction: 1, Guild: "A"}}
	a.state.Entities[ally.ID], a.state.Target = ally, ally.ID
	a.wv.facing = render.V3{1, 0, 0}
	a.clock = 5
	if !a.followOnly(ally) || a.attackReachTo(ally) != 1.5 {
		t.Fatal("ally must be follow-only at 1.5")
	}
	if !a.attackTarget() || a.wv.approachTarget != ally.ID || !a.wv.travel {
		t.Fatal("ally was not followed")
	}
	a.wv.pos = render.V3{19, 0, 0}
	a.updateActionApproach(&platform.Input{})
	if a.wv.travel || a.wv.approachTarget != ally.ID || a.wv.engagedID != ally.ID {
		t.Fatal("must hold beside the ally and keep following", a.wv.travel, a.wv.approachTarget, a.wv.engagedID)
	}
	ally.X = 40
	a.updateActionApproach(&platform.Input{})
	if !a.wv.travel {
		t.Fatal("follow did not resume when the ally walked away")
	}
	a.state.Duel = game.DuelState{Phase: game.DuelFighting, Opponent: ally.ID}
	if a.followOnly(ally) {
		t.Fatal("duel opponent must be attackable")
	}
	a.state.Duel = game.DuelState{}
	a.state.GuildDuel.Active = true
	a.state.Guild.Name = "B"
	if a.followOnly(ally) {
		t.Fatal("rival guild in a guild duel must be attackable")
	}
	a.state.GuildDuel.Active = false
	ally.Shape.Faction = 0
	if a.followOnly(ally) {
		t.Fatal("enemy faction must be attackable")
	}
}

// The world acts on the left press (0x43bfa0); the release is swallowed.
func TestWorldActsOnPressAndSwallowsRelease(t *testing.T) {
	a := inputApp()
	press := a.worldControls(&platform.Input{MouseX: 512, MouseY: 300, Pressed: true, MouseDown: true}, 1024, 768)
	if !press.Clicked {
		t.Fatal("an unowned press did not reach the world as a click")
	}
	release := a.worldControls(&platform.Input{MouseX: 512, MouseY: 300, Clicked: true}, 1024, 768)
	if release.Clicked {
		t.Fatal("the release repeated the action")
	}
	both := a.worldControls(&platform.Input{MouseX: 512, MouseY: 300, Pressed: true, MouseDown: true, RightDown: true}, 1024, 768)
	if both.Clicked {
		t.Fatal("mouse-run (both buttons) must not click")
	}
	a.worldControls(&platform.Input{MouseX: 512, MouseY: 300}, 1024, 768)
	hud := a.worldControls(&platform.Input{MouseX: 40, MouseY: 50, Pressed: true, MouseDown: true}, 1024, 768)
	if hud.Clicked {
		t.Fatal("a panel-owned press reached the world")
	}
}
