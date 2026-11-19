//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestNPCTalkWindowGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "npc talk", Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.loadNPCs()
	a.state = game.New()
	a.state.Self.Dead = false
	e := &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 3, TypeID: 18}
	a.state.Entities[e.ID] = e
	a.shop.npcID, a.shop.talk = 42, true
	a.state.Target = 42
	_ = sdata.NPCKey{}
	gl.Viewport(0, 0, 1024, 768)
	gl.ClearColor(.1, .1, .1, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawCombat(1024, 768)
	a.drawShop(1024, 768)
	a.sprites.End()
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := render.SavePNG(filepath.Join(dir, "npc-talk.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
}
