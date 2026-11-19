package studio

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/mobaimessages"
)

func TestSaveMobAIMessagesPreservesIndexesAndBacksUpText(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	path := filepath.Join(root, "local.sqlite")
	db, err := openDB(ctx, path, "rwc")
	must(t, err)
	_, err = db.ExecContext(ctx, mobaimessages.MigrationSQL)
	must(t, err)
	must(t, db.Close())

	catalog, err := OpenMobAIMessages(ctx, path)
	must(t, err)
	draft := slices.Clone(catalog.Messages)
	draft[0].Index = 0
	if _, err := SaveMobAIMessages(ctx, root, catalog, draft); err == nil {
		t.Fatal("changed index was accepted")
	}
	draft[0].Index = catalog.Messages[0].Index
	draft[0].Text = "안녕, ¡jugador!"
	backup, err := SaveMobAIMessages(ctx, root, catalog, draft)
	must(t, err)
	before, err := OpenMobAIMessages(ctx, filepath.Join(backup, "rapture.sqlite"))
	must(t, err)
	after, err := OpenMobAIMessages(ctx, path)
	must(t, err)
	if !slices.Equal(before.Messages, catalog.Messages) || !slices.Equal(after.Messages, draft) {
		t.Fatal("backup or saved messages differ from their expected values")
	}
	if _, err := SaveMobAIMessages(ctx, root, catalog, draft); err == nil {
		t.Fatal("stale draft was accepted")
	}
}
