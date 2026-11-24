//go:build integration

package app

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestGuildCaptionOriginalColorsAndSpacingGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	const w, h = 1280, 800
	win, err := platform.Open(platform.Config{Width: w, Height: h, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err = a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	for i := 0; !a.SceneReady(); i++ {
		if i > 500 {
			t.Fatal("world did not load")
		}
		a.Draw(w, h)
	}

	defer func() { delete(a.wv.actors, 99) }()
	a.scene3d.Occlusion = nil
	slot, _ := a.selfSlot()
	anchor, _ := a.playerNameplateAnchor(a.state.SelfID, a.wv.self, slot.Race, false, a.state.Self.Vehicle)
	x, y, depth, ok := projectLabel(anchor, a.scene3d.View.Mul(a.scene3d.Proj), w, h)
	if !ok {
		t.Fatal("anchor off screen")
	}
	slot.Name = "GuildPlayer"
	a.state.Slots[0] = slot
	a.state.Self.IsGM = false
	a.state.Entities = make(map[uint32]*game.Entity)
	a.state.GroundItems = nil
	a.state.Guild.ID, a.state.Guild.Name = 7, "Guardians"
	clear := func() {
		gl.Viewport(0, 0, w, h)
		gl.ClearColor(0, 0, 0, 1)
		gl.DepthFunc(gl.LESS)
		gl.ClearDepth(1)
		gl.DepthMask(true)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	}
	pixels := func() []byte {
		p := make([]byte, w*h*4)
		gl.ReadPixels(0, 0, w, h, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(p))
		return p
	}
	for _, remote := range []bool{false, true} {
		a.gameMenu.Settings.OwnName = !remote
		guildID := uint32(7)
		if remote {
			guildID = 8
		}
		if remote {
			a.state.Entities[99] = &game.Entity{Kind: game.KindUser, ID: 99, GuildID: guildID, Shape: &proto.UserShape{ID: 99, Race: slot.Race, Name: slot.Name, Guild: "Guardians"}}
			a.wv.actors[99] = &remoteActor{character: a.wv.self, shape: *a.state.Entities[99].Shape}
		}
		for _, variant := range []string{"member", "leader", "without-directory"} {
			if variant == "without-directory" && !remote {
				continue
			}
			leader := variant == "leader"
			a.state.Guild.Rank = 9
			master := "AnotherPlayer"
			tint := uint32(0xE4E4E4)
			if leader {
				master = slot.Name
				tint = 0xE0AFFA
				a.state.Guild.Rank = 1
			}
			a.state.Guild.Directory = map[uint32]proto.GuildInfo{guildID: {ID: guildID, Name: "Guardians", Master: master}}
			if variant == "without-directory" {
				a.state.Guild.Directory = nil
			}
			clear()
			a.sprites.Begin(w, h)
			a.drawPlayerNames(w, h)
			a.sprites.End()
			got := pixels()
			clear()
			a.sprites.Begin(w, h)
			a.sprites.NameDepth(depth)
			a.font.DrawName(a.sprites, slot.Name, x-a.font.Width(slot.Name)/2, y, white)
			a.font.DrawName(a.sprites, "Guardians", x-a.font.Width("Guardians")/2, y-15, ui.RGBColor(tint))
			a.sprites.End()
			want := pixels()
			if !bytes.Equal(got, want) {
				t.Fatalf("guild annotation differs from native color/15px spacing/plain text: remote=%v leader=%v", remote, leader)
			}
		}
	}
	// Selection must not override either player nameplate checkbox.
	a.gameMenu.Settings.OwnName, a.gameMenu.Settings.PlayerNames = false, false
	for _, id := range []uint32{a.state.SelfID, 99} {
		a.state.Target = id
		clear()
		want := pixels()
		a.sprites.Begin(w, h)
		a.drawPlayerNames(w, h)
		a.sprites.End()
		if !bytes.Equal(pixels(), want) {
			t.Fatalf("selected player %d retained a hidden name or guild caption", id)
		}
	}
	// The remote fixture shares the self model solely for the annotation pass.
	delete(a.wv.actors, 99)
}
