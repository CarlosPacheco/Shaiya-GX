//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"os"
	"runtime"
	"testing"
)

func TestIdleRemoteUpdatesKeepOriginalCastAnimationGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	actor, err := scene.BuildCharacter(scene.NewAssets(arc), scene.Appearance{Race: 0, Job: 5, Helmet: -1}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	a := inputApp()
	a.state.SelfID = 1
	a.wv.self = actor
	if !actor.PlayCast(88, 3) {
		t.Fatal("missing original cast animation")
	}
	move := func(motion uint8) []byte {
		w := proto.NewWriter(proto.CharacterMove)
		w.U32(1)
		w.U8(motion)
		w.U16(400)
		w.F32(0)
		w.F32(.01)
		w.F32(0)
		return w.Bytes()
	}
	for range 100 {
		a.confirmedAnimation(move(0))
		actor.Update(.005)
		if _, _, active := actor.CastTiming(); !active {
			t.Fatal("idle/height refresh cancelled cast animation")
		}
	}
	a.confirmedAnimation(move(proto.MotionFlag80))
	if _, _, active := actor.CastTiming(); !active {
		t.Fatal("non-interrupting update cancelled cast")
	}
	a.confirmedAnimation(move(1))
	if _, _, active := actor.CastTiming(); active {
		t.Fatal("real walking kept cast animation")
	}
}

func TestRemoteJumpDoesNotReplayOnCosmeticOptionsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()

	a := inputApp()
	a.assets = scene.NewAssets(arc)
	a.arc = arc
	a.gameMenu, _ = ui.NewGameMenu(inputTextures{})
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Class: 5, Height: 2}
	a.refreshSelf(a.wv)
	a.wv.world = &scene.World{}
	const remote = 2
	shape := &proto.UserShape{ID: remote, Class: 5, Height: 2}
	shape.Equipment[7][0], shape.Equipment[7][1] = 24, 1
	a.state.Entities[remote] = &game.Entity{ID: remote, Kind: game.KindUser, Shape: shape}

	jump := func() {
		t.Helper()
		w := proto.NewWriter(proto.CharacterMotion)
		w.U32(remote)
		w.U8(proto.MotionJump)
		if _, err := a.state.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
		a.prepareRemotePlayers()
		if !a.wv.actors[remote].character.PlayingAction(8) {
			t.Fatal("new jump did not animate")
		}
	}
	jump()
	cached := a.wv.actors[remote]
	if !cached.character.HasCloak() {
		t.Fatal("fixture did not load the remote cape")
	}
	cached.character.Update(10)
	if cached.character.PlayingAction(8) {
		t.Fatal("first jump did not finish")
	}
	for _, hidden := range []bool{true, false} {
		a.gameMenu.Settings.HideCloak = hidden
		a.applyClientSettings()
		a.prepareRemotePlayers()
		actor := a.wv.actors[remote]
		if actor != cached || actor.character.HasCloak() == hidden || actor.character.PlayingAction(8) {
			t.Fatalf("cape visibility %t recreated remote state or replayed the jump", !hidden)
		}
	}
	model := cached.character
	a.gameMenu.Settings.SkillEffects = false
	a.applyClientSettings()
	a.prepareRemotePlayers()
	if a.wv.actors[remote].character != model || model.PlayingAction(8) {
		t.Fatal("effects toggle recreated remote model or replayed the jump")
	}
	a.frameDT = 1.0 / 60
	jump()
	if model.Time <= 0 {
		t.Fatal("remote jump playback did not advance on its first update")
	}
	a.state.Entities[remote].Motion = 1
	for range 60 {
		a.prepareRemotePlayers()
		if !model.PlayingAction(8) {
			break
		}
	}
	for range 10 {
		if !model.PlayingClip("000_normal") {
			t.Fatal("remote running skipped the landing blend")
		}
		a.prepareRemotePlayers()
	}
	for range 2 {
		a.prepareRemotePlayers()
	}
	if !model.PlayingClip("002_run") {
		t.Fatal("remote running did not resume after the landing blend")
	}
}
