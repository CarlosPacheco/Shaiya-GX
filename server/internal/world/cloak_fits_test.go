package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
	"time"
)

func TestChestCloakFitsOnServerWire(t *testing.T) {
	_, _ = forgeRuleSession(t)
	// Exercise multiple framed chunks, including the real Anor chest identity.
	for i := 1; i <= 255; i++ {
		key := db.ItemKey{Type: 17, TypeID: uint8(i)}
		db.MustDefs().Items[key] = &db.ItemDef{Type: 17, TypeID: uint8(i), ReqVg: 8}
	}
	for i := 1; i <= 2; i++ {
		key := db.ItemKey{Type: 32, TypeID: uint8(i)}
		db.MustDefs().Items[key] = &db.ItemDef{Type: 32, TypeID: uint8(i), ReqVg: 0}
	}
	s, packets := gmWireSession(t, db.Character{ID: 990046}, true)
	expected := chestCloakFits(db.MustDefs())
	s.sendChestCloakFits()
	var got []proto.ChestCloakFit
	for len(got) < len(expected) {
		select {
		case p := <-packets:
			rows, err := proto.DecodeChestCloakFits(p)
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, rows...)
		case <-time.After(time.Second):
			t.Fatal("missing framed chest fit catalog")
		}
	}
	found := false
	for i, v := range got {
		if v != expected[i] {
			t.Fatal("fit mismatch")
		}
		if v.Type == 17 && v.TypeID == 216 {
			found = true
			if v.Fit != 7 {
				t.Fatal("actual Anor chest needs fit 8")
			}
		}
	}
	if !found {
		t.Fatal("actual chest missing")
	}
}
