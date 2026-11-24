package db

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
)

func TestConsumableSkillAliasesAgreeWithClient(t *testing.T) {
	p := sqliteFixture(t)
	const itemIDs = `44089,44090,44091,44152,44153,44154,
		44213,44214,44217,44218,44219,44220,44221,44222,100114,
		100118,100119,100120,100121,100122,100123,100124,100125,100126,
		100134,100200,100201`
	result, err := p.Exec(`UPDATE PS_GameDefs_Items SET Range=Range-256 WHERE ItemID IN (` + itemIDs + `) AND Range>255`)
	if err != nil {
		t.Fatal(err)
	}
	if count, err := result.RowsAffected(); err != nil || count != 27 {
		t.Fatalf("expected 27 legacy aliases, got %d (%v)", count, err)
	}

	items := map[sdata.ItemKey]*sdata.Item{}
	rows, err := p.Query(`SELECT Type,TypeID,Range,AttackTime,Special FROM PS_GameDefs_Items WHERE ItemID IN (` + itemIDs + `)`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var key sdata.ItemKey
		var item sdata.Item
		if err := rows.Scan(&key.Type, &key.TypeID, &item.Range, &item.AttackTime, &item.Special); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		items[key] = &item
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	sdata.NormalizeRestoredConsumables(items)

	step := migrations[37]
	for range 2 {
		if _, err := p.Exec(step.SQL); err != nil {
			t.Fatal(err)
		}
	}
	for key, item := range items {
		var skill uint16
		var rank, special, effect uint8
		err := p.QueryRow(`SELECT i.Range,i.AttackTime,i.Special,s.TypeEffect
			FROM PS_GameDefs_Items i JOIN PS_GameDefs_Skills s ON s.SkillID=i.Range AND s.SkillLevel=i.AttackTime
			WHERE i.Type=? AND i.TypeID=?`, key.Type, key.TypeID).Scan(&skill, &rank, &special, &effect)
		if err != nil || skill <= 255 || skill != item.Range || rank != item.AttackTime || special != item.Special || effect != 4 {
			t.Errorf("item %v client=%d/%d special=%d server=%d/%d special=%d effect=%d: %v",
				key, item.Range, item.AttackTime, item.Special, skill, rank, special, effect, err)
		}
	}
}
