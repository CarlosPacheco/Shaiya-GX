package studio

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
	"github.com/Spelunkern/Shaiya-GX/internal/teleportrunes"
)

func TestOpenTeleportRunesPrefersCorrectedName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, catalogfiles.TeleportRunes)
	runes := []teleportrunes.Rune{{Index: 1, Light: teleportrunes.Destination{MapID: 35}}}
	must(t, os.WriteFile(filepath.Join(dir, "TeleportRUnes.ini"), catalogfiles.EncodeTeleportRunes(runes), 0600))
	before, err := OpenTeleportRunes(t.Context(), path)
	must(t, err)
	if !slices.Equal(before.Rules, runes) {
		t.Fatal("legacy file destinations were lost")
	}
	runes[0].Light.MapID = 42
	must(t, os.WriteFile(path, catalogfiles.EncodeTeleportRunes(runes), 0600))
	after, err := OpenTeleportRunes(t.Context(), path)
	must(t, err)
	if !slices.Equal(after.Rules, runes) {
		t.Fatal("corrected file did not take precedence")
	}
}

func TestSaveTeleportRunesKeepsFactionDestinationsAndVerifiedBackup(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	path := filepath.Join(root, "local.sqlite")
	db, err := openDB(ctx, path, "rwc")
	must(t, err)
	_, err = db.ExecContext(ctx, `CREATE TABLE PS_GameDefs_Items(ItemID INTEGER, Special INTEGER, ReqVg INTEGER)`)
	must(t, err)
	_, err = db.ExecContext(ctx, teleportrunes.MigrationSQL)
	must(t, err)
	must(t, db.Close())

	catalog, err := OpenTeleportRunes(ctx, path)
	must(t, err)
	draft := slices.Clone(catalog.Rules)
	draft[0].Light.X = math.Inf(1)
	if _, err := SaveTeleportRunes(ctx, root, catalog, draft); err == nil {
		t.Fatal("accepted an infinite coordinate")
	}
	draft[0].Light.X = 585.87654321
	draft = slices.Delete(draft, 1, 2)
	draft = append(draft, teleportrunes.Rune{Index: 42,
		Light: teleportrunes.Destination{MapID: 35, X: 2.125, Y: 3, Z: 4},
		Fury:  teleportrunes.Destination{MapID: 36, X: 5, Y: -6.25, Z: 7},
	})
	backup, err := SaveTeleportRunes(ctx, root, catalog, draft)
	must(t, err)
	before, err := OpenTeleportRunes(ctx, filepath.Join(backup, "rapture.sqlite"))
	must(t, err)
	after, err := OpenTeleportRunes(ctx, path)
	must(t, err)
	if !slices.Equal(before.Rules, catalog.Rules) || !slices.Equal(after.Rules, draft) {
		t.Fatal("backup or saved rune destinations changed unexpectedly")
	}
	if _, err := SaveTeleportRunes(ctx, root, catalog, draft); err == nil {
		t.Fatal("accepted a stale draft")
	}
}
