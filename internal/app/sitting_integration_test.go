//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestEquipmentRefreshKeepsPostureGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and OpenGL")
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
	a.arc, a.assets = arc, scene.NewAssets(arc)
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Height: 2}
	a.wv = &worldView{world: &scene.World{}}
	a.refreshSelf(a.wv)
	press := &platform.Input{Chars: []rune{'c'}}
	a.state.Self.Sitting = true
	for _, action := range []int{10, 11} {
		actor := a.wv.self
		if !actor.PlayAction(action, false) || postureToggle(press, actor) {
			t.Fatalf("C should be ignored during action %d", action)
		}
		actor.Update(.2)
		a.state.Self.Sitting = action == 10
		if action == 10 {
			before := actor.Time
			a.updateWorld(&platform.Input{Down: "W"}, 1.0/60)
			if a.wv.sitPending || !actor.PlayingAction(10) || actor.Time <= before {
				t.Fatal("movement interrupted the active sit animation")
			}
		}
		a.refreshSelf(a.wv)
		if a.wv.self == actor || !a.wv.self.PlayingAction(action) || a.wv.self.Time != actor.Time {
			t.Fatalf("rebuilding changed posture action %d", action)
		}
	}
	a.wv.self.ReleaseAction()
	a.state.Self.Sitting = true
	a.wv.self.SetClip("012_sit")
	a.wv.self.Update(.2)
	actor := a.wv.self
	a.refreshSelf(a.wv)
	if a.wv.self == actor || !a.wv.self.PlayingClip("012_sit") || a.wv.self.Time != actor.Time || !postureToggle(press, a.wv.self) {
		t.Fatal("rebuilding reset the seated pose or blocked a fresh press")
	}
	remote := &game.Entity{ID: 2, Kind: game.KindUser, Sitting: true, Shape: &proto.UserShape{ID: 2, Height: 2}}
	a.state.Entities[2] = remote
	a.frameDT = .2
	a.prepareRemotePlayers()
	actor = a.wv.actors[2].character
	remote.Shape.Equipment[5] = [3]uint8{1, 1, 0}
	a.frameDT = 0
	a.prepareRemotePlayers()
	rebuilt := a.wv.actors[2].character
	if rebuilt == actor || !rebuilt.PlayingClip("012_sit") || rebuilt.Time != actor.Time {
		t.Fatal("remote equipment change reset the seated pose")
	}
}

func TestSittingAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_SITTING") != "1" || os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires disposable accounts and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "sitting probe", Width: 1024, Height: 734, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	st, world, _, _ := testworld.Enter(t)
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, world, screenWorld
	a.enterWorld()
	pump := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for !cond() {
			a.pumpWorld()
			if a.state == nil || time.Now().After(deadline) {
				t.Fatal("sitting confirmation timed out")
			}
			time.Sleep(time.Millisecond)
		}
	}
	a.updateWorld(&platform.Input{Chars: []rune{'c'}}, 1.0/60)
	if !a.wv.sitPending || st.Self.Sitting {
		t.Fatal("sit changed before server confirmation")
	}
	pump(func() bool { return st.Self.Sitting && !a.wv.sitPending })
	actor, posBeforeSit := a.wv.self, a.wv.pos
	a.updateWorld(&platform.Input{Down: "W"}, 1.0/60)
	if !actor.PlayingAction(10) || a.wv.sitPending || a.wv.pos != posBeforeSit {
		t.Fatal("movement interrupted the sit animation")
	}
	other, _, _, waitOther := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitOther(func() bool { e := other.Entities[st.SelfID]; return e != nil && e.Shape != nil && e.Sitting })
	for i := 0; i < 150; i++ {
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.updateWorld(&platform.Input{}, 1.0/60)
		a.Draw(1024, 734)
	}
	if !a.SceneReady() || !st.Self.Sitting {
		t.Fatal("sitting scene failed to load")
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := render.SavePNG(filepath.Join(dir, "n2-sitting.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	pos := a.wv.pos
	a.updateWorld(&platform.Input{Down: "W"}, .1)
	if !a.wv.sitPending || a.wv.pos != pos || !st.Self.Sitting {
		t.Fatal("movement before standing confirmation")
	}
	pump(func() bool { return !st.Self.Sitting && !a.wv.sitPending })
	waitOther(func() bool { e := other.Entities[st.SelfID]; return e != nil && !e.Sitting })
	if a.wv.self == nil || !a.wv.self.PlayingAction(11) {
		t.Fatal("standing transition did not start")
	}
	a.updateWorld(&platform.Input{Down: "W"}, .01)
	if a.wv.pos != pos {
		t.Fatal("movement started before the standing animation ended")
	}
	for i := 0; i < 600 && a.wv.pos == pos; i++ {
		a.updateWorld(&platform.Input{Down: "W"}, 1.0/60)
	}
	if a.wv.pos == pos {
		t.Fatal("movement did not resume after standing")
	}
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("sitting GL error %#x", err)
	}
	t.Log("sit/stand confirmed on both clients, including late observer")
}
