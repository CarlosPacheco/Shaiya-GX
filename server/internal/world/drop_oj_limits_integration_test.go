//go:build integration

package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"os"
	"path/filepath"
	"testing"
)

func TestWorldDropsRespectCatalogOJLimits(t *testing.T) {
	if _, err := os.Stat("data/game"); err != nil {
		if err = os.Chdir("../.."); err != nil {
			t.Fatal(err)
		}
	}
	config.LoadRuntimeEnv()
	if err := db.Open(db.UserData, db.GameData, db.GameDefs); err != nil {
		t.Fatal(err)
	}
	defs, err := db.Definitions()
	if err != nil {
		t.Fatal(err)
	}
	eligible := 0
	for _, d := range defs.Items {
		if d.Server == 0 || d.ReqWis <= 0 {
			continue
		}
		eligible++
		for range 10 {
			validateRecreationRoll(t, d, newDroppedItem(d).CraftName, 0)
		}
	}
	if eligible == 0 {
		t.Fatal("empty recreatable item catalog")
	}
	t.Logf("Checked ten world-drop rolls for each of %d recreatable definitions", eligible)
}

func TestClientAndServerRecreationLimitsMatch(t *testing.T) {
	directory := os.Getenv("SHAIYA_DATA")
	if directory == "" {
		t.Skip("requires matching client item data")
	}
	raw, err := os.ReadFile(filepath.Join(directory, "Item", "Item.SData"))
	if err != nil {
		t.Fatal(err)
	}
	items, err := sdata.ParseItems(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("data/game"); err != nil {
		if err = os.Chdir("../.."); err != nil {
			t.Fatal(err)
		}
	}
	config.LoadRuntimeEnv()
	if err := db.Open(db.UserData, db.GameData, db.GameDefs); err != nil {
		t.Fatal(err)
	}
	defs, err := db.Definitions()
	if err != nil {
		t.Fatal(err)
	}

	limits, err := proto.DecodeOJLimits(proto.OJLimitsPacket(equipmentOJLimits(defs)))
	if err != nil {
		t.Fatal(err)
	}
	changed, checked := 0, 0
	for _, limit := range limits {
		client := items[sdata.ItemKey{Type: limit.Type, TypeID: limit.TypeID}]
		if client == nil {
			continue
		} // Some server-only definitions have no client model.
		if client.Server != limit.Stats || client.RequiredStats[4] != uint16(limit.Value) {
			changed++
		}
		client.Server = limit.Stats
		client.RequiredStats[4] = uint16(limit.Value)
		d := defs.Item(limit.Type, limit.TypeID)
		it := newDroppedItem(d)
		if limit.Stats > 0 {
			craft, ok := ParseCraftName(it.CraftName)
			if !ok {
				t.Fatal(it)
			}
			used := 0
			pairs := craftPairs(craft)
			for _, v := range pairs[:9] {
				if v > 0 {
					used++
				}
			}
			if used > int(client.Server) {
				t.Fatal("drop exceeds authoritative tooltip", d.Type, d.TypeID)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("empty catalog")
	}
	t.Logf("Verified %d matching server-authoritative definitions; corrected %d stale client limits", checked, changed)
}
