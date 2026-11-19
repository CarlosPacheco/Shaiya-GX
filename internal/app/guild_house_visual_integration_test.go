//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestGuildHouseWindowsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "guild house", Width: 1024, Height: 768, Hidden: true, Debug: true})
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
	a.state.MapID = 51
	g := &a.state.Guild
	g.ID, g.Name, g.Rank, g.Etin = 1, "Guardians", 1, 9000
	g.Directory = map[uint32]proto.GuildInfo{1: {ID: 1, Name: "Guardians", Master: "Alpha", Rank: 3, Points: 120}, 2: {ID: 2, Name: "Others", Master: "Beta", Rank: 31}}
	g.Npcs = map[uint8]proto.GuildNpc{0: {Type: 2, Level: 1, Slot: 0}, 1: {Type: 6, Level: 2, Slot: 1}, 2: {Type: 3, Level: 3, Slot: 2}}
	manager := &game.Entity{ID: 10, Kind: game.KindNpc, NpcType: 11, TypeID: 5}
	etin := &game.Entity{ID: 11, Kind: game.KindNpc, NpcType: 1, TypeID: 217}
	draw := func(name string, e *game.Entity, mode int) {
		kind, ok := a.houseKind(e)
		if !ok {
			t.Fatalf("%s: not a house NPC", name)
		}
		a.house.mode, a.house.selected = mode, 1
		gl.Viewport(0, 0, 1024, 768)
		gl.ClearColor(.1, .1, .1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		a.drawHouse(1024, 768, e, kind)
		a.sprites.End()
		if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
			if err := render.SavePNG(filepath.Join(dir, name+".png"), 1024, 768); err != nil {
				t.Fatal(err)
			}
		}
	}
	draw("house-manager-info", manager, ui.HouseModeInfo)
	draw("house-manager-service", manager, ui.HouseModeManager)
	draw("house-etin-list", etin, ui.HouseModeList)
	draw("house-etin-buy", etin, ui.HouseModeBuy)
	draw("house-etin-info", etin, ui.HouseModeInfo)
}
