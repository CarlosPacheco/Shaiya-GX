//go:build integration

package app

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestEmblemWindowGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "emblem", Width: 1024, Height: 768, Hidden: true, Debug: true})
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
	var cloak proto.InventoryItem
	for id := uint8(1); id < 80 && cloak.Type == 0; id++ {
		if d := a.itemsItem(24, id); d != nil && d.Model == 21 {
			cloak = proto.InventoryItem{Bag: 1, Slot: 0, Type: 24, TypeID: id, Count: 1}
		}
	}
	if d := a.itemsItem(24, cloak.TypeID); d != nil {
		t.Logf("cloak %d model %d", cloak.TypeID, d.Model)
	}
	if cloak.Type == 0 {
		t.Skip("no cloak item in the catalogue")
	}
	a.state.SelfID = 7
	a.state.Slots = map[uint8]proto.CharacterSlot{0: {ID: 7, Level: 40, Race: 0, Class: 5, Height: 2, Face: 1, Hair: 1}}
	a.state.Inventory = map[[2]uint8]proto.InventoryItem{{1, 0}: cloak}
	a.state.InventoryReady = true
	a.state.Details.Money = 900000
	e := &game.Entity{ID: 20, Kind: game.KindNpc, NpcType: 1, TypeID: 177}
	a.state.Entities = map[uint32]*game.Entity{20: e}
	if !a.emblemNPC(e) {
		t.Fatal("NPC 1/177 is not the emblem merchant")
	}
	a.shop.npcID = 20
	a.emblem.npc = 20
	a.emblemClear()
	a.emblemPlace([2]uint8{1, 0}, cloak)
	a.emblem.selection = [3]int{4, 2, 9}
	a.emblem.colors = [3]uint8{0, 6, 11}
	a.emblem.active = 2
	for _, deg := range []float64{180} {
		emblemPreviewYaw = deg * math.Pi / 180
		gl.Viewport(0, 0, 1024, 768)
		gl.ClearColor(.1, .1, .1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		a.drawEmblem(1024, 768)
		a.sprites.End()
		if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
			if err := render.SavePNG(filepath.Join(dir, fmt.Sprintf("emblem-window-%03.0f.png", deg)), 1024, 768); err != nil {
				t.Fatal(err)
			}
		}
	}
}
