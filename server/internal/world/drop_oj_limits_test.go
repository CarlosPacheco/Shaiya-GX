package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
)

func TestWorldDropRecreationCountsStayWithinEveryPoolAndBudget(t *testing.T) {
	for _, typ := range []uint8{1, 15, 16, 21, 22, 23, 24, 25, 40} {
		for _, budget := range []uint8{0, 1, 2, 4, 6, 9, 10, 255} {
			def := &db.ItemDef{Type: typ, TypeID: 1, Server: budget, ReqWis: 13}
			for range 50 {
				it := newDroppedItem(def)
				if budget == 0 {
					if it.CraftName != "" {
						t.Fatal("ineligible OJ")
					}
					continue
				}
				validateRecreationRoll(t, def, it.CraftName, 0)
			}
		}
	}
}
