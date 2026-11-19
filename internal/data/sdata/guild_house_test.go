package sdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGuildHouseOriginal(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), "Desktop", "Shaiya GX", "data", "Npc", "GuildHouse.SData"))
	if err != nil {
		t.Skip("installed client data not available")
	}
	d, err := ParseGuildHouse(b)
	if err != nil {
		t.Fatal(err)
	}
	if d.BuyMoney != 50000000 || d.DefaultKeep != 2000 {
		t.Fatalf("header %+v", d)
	}
	if s := d.Service(2, 1); s == nil || s.UpgradeEtin != 1000 || s.KeepEtin != 200 || s.TextID != 8011 {
		t.Fatalf("gate level 1: %+v", s)
	}
	if d.NpcIDs[1][0] != [3]uint32{196, 198, 197} || d.NpcIDs[3][1] != [3]uint32{223, 224, 225} {
		t.Fatalf("npc ids %v", d.NpcIDs)
	}
}
