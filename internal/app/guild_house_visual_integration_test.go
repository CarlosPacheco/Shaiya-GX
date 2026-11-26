//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
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
	manager := &game.Entity{ID: 10, Kind: game.KindNpc, NpcType: 11, TypeID: 5}
	etin := &game.Entity{ID: 11, Kind: game.KindNpc, NpcType: 1, TypeID: 217}
	a.house.mode = ui.HouseModeList
	if view := a.houseBuildView(ui.HouseEtinManager, etin); len(view.List) != 2 || view.CanBuy {
		t.Fatalf("ranking and buy controls before World reply: %+v", view)
	}
	w := proto.NewWriter(proto.GXGuildHouseServices)
	w.U32(g.ID)
	w.U32(54_321_000)
	w.U32(3000)
	w.U8(1)
	w.U8(7)
	for _, row := range []proto.GuildHouseService{
		{Type: 2, Level: 1, Upkeep: 187, NextCost: 733},
		{Type: 0x101, NextCost: 6100},
		{Type: 0x201, Level: 3},
		{Type: 6, Level: 2, Upkeep: 200, NextCost: 1000},
		{Type: 0x301, NextCost: 150},
		{Type: 3, Level: 3, Upkeep: 500, NextCost: 10000},
		{Type: 0x401, NextCost: 2000},
	} {
		w.U16(row.Type)
		w.U8(row.Level)
		w.U16(row.Upkeep)
		w.U16(row.NextCost)
	}
	if _, err := a.state.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	a.house.confirmBox = &ui.Message{}
	a.houseAskBuy()
	if !strings.Contains(a.house.confirmBox.Text, "54,321,000") {
		t.Fatalf("buy confirmation %q", a.house.confirmBox.Text)
	}
	view := a.houseBuildView(ui.HouseEtinManager, etin)
	if !view.CanBuy || view.Total != "3000" || view.Available != "6000" || !strings.Contains(view.Rows[0].FeeText, "187") {
		t.Fatalf("house upkeep view %+v", view)
	}
	view = a.houseBuildView(ui.HouseGuildManager, manager)
	if !view.Rows[0].UpgradeOn || view.Rows[1].UpgradeOn || view.Rows[2].ShowUpgrade {
		t.Fatalf("upgrade controls %+v", view.Rows[:3])
	}
	a.houseAskUpgrade(0)
	if a.house.upLevel != 2 || !strings.Contains(a.house.confirmBox.Text, "733") {
		t.Fatalf("upgrade confirmation %q for level %d", a.house.confirmBox.Text, a.house.upLevel)
	}
	a.house.confirmBox = nil
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
