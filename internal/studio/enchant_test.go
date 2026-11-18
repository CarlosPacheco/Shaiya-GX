package studio

import (
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
)

func TestSaveEnchantBacksUpAndRejectsGaps(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	path := filepath.Join(root, "local.sqlite")
	db, err := openDB(ctx, path, "rwc")
	must(t, err)
	defer db.Close()
	_, err = db.Exec(enchantrules.CreateSQL)
	must(t, err)
	tx, err := db.BeginTx(ctx, nil)
	must(t, err)
	initial := enchantrules.Catalog{Attack: []enchantrules.Level{{Number: 0}}, Defense: []enchantrules.Level{{Number: 0}, {Number: 1, Bonus: 5}}}
	must(t, enchantrules.Replace(ctx, tx, initial))
	must(t, tx.Commit())
	catalog, err := OpenEnchant(ctx, path)
	must(t, err)
	bad := enchantrules.Catalog{Attack: []enchantrules.Level{{Number: 0}, {Number: 2}}, Defense: initial.Defense}
	if _, err := SaveEnchant(ctx, root, catalog, bad); err == nil {
		t.Fatal("gap in attack levels was accepted")
	}
	next := enchantrules.Catalog{Attack: []enchantrules.Level{{Number: 0, SuccessChance: 900000}, {Number: 1, Bonus: 7}}, Defense: []enchantrules.Level{{Number: 0}}}
	backup, err := SaveEnchant(ctx, root, catalog, next)
	must(t, err)
	old, err := OpenEnchant(ctx, filepath.Join(backup, "rapture.sqlite"))
	must(t, err)
	saved, err := OpenEnchant(ctx, path)
	must(t, err)
	attack, defense, err := saved.Rules.Caps()
	must(t, err)
	if attack != 1 || defense != 0 || !old.Rules.Equal(initial) || !saved.Rules.Equal(next) {
		t.Fatal("backup or independent caps differ from the requested catalogs")
	}
	if _, err := SaveEnchant(ctx, root, catalog, next); err == nil {
		t.Fatal("stale draft was accepted")
	}
}
